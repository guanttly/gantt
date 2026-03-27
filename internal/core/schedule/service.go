package schedule

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"gantt-saas/internal/ai"
	"gantt-saas/internal/core/employee"
	"gantt-saas/internal/core/leave"
	"gantt-saas/internal/core/rule"
	"gantt-saas/internal/core/schedule/pipeline"
	"gantt-saas/internal/core/schedule/step"
	"gantt-saas/internal/core/shift"
	"gantt-saas/internal/infra/websocket"
	"gantt-saas/internal/tenant"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrScheduleNotFound   = errors.New("排班计划不存在")
	ErrInvalidDateRange   = errors.New("无效的日期范围")
	ErrInvalidStatus      = errors.New("无效的排班状态")
	ErrCannotGenerate     = errors.New("当前状态不允许生成排班")
	ErrCannotPublish      = errors.New("当前状态不允许发布排班")
	ErrCannotAdjust       = errors.New("当前状态不允许调整排班")
	ErrAssignmentNotFound = errors.New("排班分配不存在")
	ErrNotLeafNode        = errors.New("只有科室级（department）节点才能创建排班")
)

// CreateInput 创建排班计划的输入参数。
type CreateInput struct {
	Name         string          `json:"name"`
	GroupID      *string         `json:"group_id"`
	StartDate    string          `json:"start_date"`
	EndDate      string          `json:"end_date"`
	PipelineType string          `json:"pipeline_type"`
	Config       json.RawMessage `json:"config"`
}

// Service 排班业务逻辑层。
type Service struct {
	repo                *Repository
	ruleService         *rule.Service
	shiftService        *shift.Service
	employeeRepo        *employee.Repository
	leaveRepo           *leave.Repository
	groupMemberProvider step.GroupMemberProvider
	orgNodeResolver     OrgNodeTypeChecker
	aiProvider          ai.Provider           // 可选，为 nil 时不支持 AI 排班
	broadcaster         websocket.Broadcaster // 可选，为 nil 时不推送 WS
	logger              *zap.Logger
}

// OrgNodeTypeChecker 检查组织节点类型的接口。
type OrgNodeTypeChecker interface {
	GetByID(ctx context.Context, id string) (*tenant.OrgNode, error)
}

// NewService 创建排班服务。
func NewService(
	repo *Repository,
	ruleService *rule.Service,
	shiftService *shift.Service,
	employeeRepo *employee.Repository,
	leaveRepo *leave.Repository,
	logger *zap.Logger,
) *Service {
	return &Service{
		repo:         repo,
		ruleService:  ruleService,
		shiftService: shiftService,
		employeeRepo: employeeRepo,
		leaveRepo:    leaveRepo,
		logger:       logger,
	}
}

// SetAIProvider 设置 AI 提供者（可选）。
func (s *Service) SetAIProvider(p ai.Provider) {
	s.aiProvider = p
}

// SetBroadcaster 设置 WebSocket 广播器（可选）。
func (s *Service) SetBroadcaster(b websocket.Broadcaster) {
	s.broadcaster = b
}

// SetGroupMemberProvider 设置分组成员查询器（可选）。
func (s *Service) SetGroupMemberProvider(p step.GroupMemberProvider) {
	s.groupMemberProvider = p
}

// SetOrgNodeResolver 设置组织节点查询器（可选）。
func (s *Service) SetOrgNodeResolver(r OrgNodeTypeChecker) {
	s.orgNodeResolver = r
}

// Create 创建排班计划。
func (s *Service) Create(ctx context.Context, input CreateInput) (*Schedule, error) {
	orgNodeID := tenant.GetOrgNodeID(ctx)
	if orgNodeID == "" {
		return nil, fmt.Errorf("缺少组织节点信息")
	}

	// 检查是否为叶子节点（department）
	if s.orgNodeResolver != nil {
		node, err := s.orgNodeResolver.GetByID(ctx, orgNodeID)
		if err != nil {
			return nil, fmt.Errorf("查询组织节点失败: %w", err)
		}
		if !tenant.IsLeafNodeType(node.NodeType) {
			return nil, ErrNotLeafNode
		}
	}

	if input.Name == "" {
		return nil, fmt.Errorf("排班计划名称不能为空")
	}
	if input.StartDate == "" || input.EndDate == "" {
		return nil, ErrInvalidDateRange
	}
	if input.StartDate > input.EndDate {
		return nil, ErrInvalidDateRange
	}

	pipelineType := input.PipelineType
	if pipelineType == "" {
		pipelineType = PipelineDeterministic
	}

	sch := &Schedule{
		ID:           uuid.New().String(),
		Name:         input.Name,
		GroupID:      input.GroupID,
		StartDate:    input.StartDate,
		EndDate:      input.EndDate,
		Status:       StatusDraft,
		PipelineType: pipelineType,
		Config:       input.Config,
		CreatedBy:    "system", // TODO: 从 ctx 获取当前用户
		TenantModel: tenant.TenantModel{
			OrgNodeID: orgNodeID,
		},
	}

	if err := s.repo.CreateSchedule(ctx, sch); err != nil {
		return nil, fmt.Errorf("创建排班计划失败: %w", err)
	}

	return sch, nil
}

// GetByID 获取排班计划详情。
func (s *Service) GetByID(ctx context.Context, id string) (*Schedule, error) {
	sch, err := s.repo.GetScheduleByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrScheduleNotFound
		}
		return nil, err
	}
	return sch, nil
}

// List 查询排班计划列表。
func (s *Service) List(ctx context.Context, opts ScheduleListOptions) ([]Schedule, int64, error) {
	if opts.Page <= 0 {
		opts.Page = 1
	}
	if opts.Size <= 0 {
		opts.Size = 20
	}
	return s.repo.ListSchedules(ctx, opts)
}

// Generate 触发排班生成（执行 Pipeline）。
func (s *Service) Generate(ctx context.Context, id string) (*GenerateResult, error) {
	sch, err := s.repo.GetScheduleByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrScheduleNotFound
		}
		return nil, err
	}

	// 只有 draft 或 review 状态可以重新生成
	if sch.Status != StatusDraft && sch.Status != StatusReview {
		return nil, ErrCannotGenerate
	}

	// 更新状态为 generating
	if err := s.repo.UpdateScheduleStatus(ctx, id, StatusGenerating); err != nil {
		return nil, fmt.Errorf("更新状态失败: %w", err)
	}

	// 解析排班配置
	var config step.ScheduleConfig
	if len(sch.Config) > 0 {
		if err := json.Unmarshal(sch.Config, &config); err != nil {
			return nil, fmt.Errorf("解析排班配置失败: %w", err)
		}
	}

	// 创建 Pipeline 共享状态
	groupID := ""
	if sch.GroupID != nil {
		groupID = *sch.GroupID
	}
	state := step.NewScheduleState(
		sch.ID,
		sch.TenantModel.OrgNodeID,
		groupID,
		sch.StartDate,
		sch.EndDate,
		sch.CreatedBy,
		&config,
	)

	// 设置 WebSocket 进度回调
	if s.broadcaster != nil {
		state.OnProgress = func(stepName string, progress float64, message string) {
			msg := websocket.NewProgressMessage(sch.ID, stepName, progress, message)
			_ = s.broadcaster.BroadcastToGroup(sch.ID, msg)
		}
	}

	// 创建并执行排班 Pipeline
	var p interface {
		Run(context.Context, *step.ScheduleState) error
	}

	if sch.PipelineType == PipelineAIAssisted && s.aiProvider != nil {
		p = pipeline.NewAIAssistedPipeline(&pipeline.AIAssistedDeps{
			RuleService:  s.ruleService,
			ShiftService: s.shiftService,
			EmployeeRepo: s.employeeRepo,
			LeaveRepo:    s.leaveRepo,
			DraftSaver:   s.repo,
			AIProvider:   s.aiProvider,
			Broadcaster:  s.broadcaster,
			Logger:       s.logger,
		})
	} else {
		p = pipeline.NewDeterministicPipeline(&pipeline.DeterministicDeps{
			RuleService:         s.ruleService,
			ShiftService:        s.shiftService,
			EmployeeRepo:        s.employeeRepo,
			LeaveRepo:           s.leaveRepo,
			GroupMemberProvider: s.groupMemberProvider,
			ConflictChecker:     s.repo,
			ShiftResolver:       s.shiftService,
			DraftSaver:          s.repo,
			Broadcaster:         s.broadcaster,
			Logger:              s.logger,
		})
	}

	if err := p.Run(ctx, state); err != nil {
		// 排班失败，回滚状态
		_ = s.repo.UpdateScheduleStatus(ctx, id, StatusDraft)
		return nil, fmt.Errorf("排班生成失败: %w", err)
	}

	return &GenerateResult{
		ScheduleID:       id,
		Status:           StatusReview,
		AssignmentsCount: len(state.Assignments),
		ViolationsCount:  len(state.Violations),
		Violations:       state.Violations,
	}, nil
}

// GetAssignments 查看排班结果。
func (s *Service) GetAssignments(ctx context.Context, scheduleID string) ([]Assignment, error) {
	// 先验证排班计划存在
	_, err := s.repo.GetScheduleByID(ctx, scheduleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrScheduleNotFound
		}
		return nil, err
	}
	return s.repo.ListAssignmentsBySchedule(ctx, scheduleID)
}

// GetSelfAssignments 查询当前员工在日期范围内的已发布排班。
func (s *Service) GetSelfAssignments(ctx context.Context, employeeID, startDate, endDate string) ([]SelfAssignmentView, error) {
	if employeeID == "" {
		return nil, ErrAssignmentNotFound
	}
	if startDate == "" || endDate == "" || startDate > endDate {
		return nil, ErrInvalidDateRange
	}
	return s.repo.ListSelfAssignments(ctx, employeeID, startDate, endDate)
}

// AdjustAssignments 手动调整排班（触发调整 Pipeline）。
func (s *Service) AdjustAssignments(ctx context.Context, scheduleID string, input step.EditInput) (*GenerateResult, error) {
	sch, err := s.repo.GetScheduleByID(ctx, scheduleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrScheduleNotFound
		}
		return nil, err
	}

	// 只有 review 或 draft 状态可以调整
	if sch.Status != StatusReview && sch.Status != StatusDraft {
		return nil, ErrCannotAdjust
	}

	// 解析配置
	var config step.ScheduleConfig
	if len(sch.Config) > 0 {
		if err := json.Unmarshal(sch.Config, &config); err != nil {
			return nil, fmt.Errorf("解析排班配置失败: %w", err)
		}
	}

	// 加载现有排班
	existing, err := s.repo.ListAssignmentsBySchedule(ctx, scheduleID)
	if err != nil {
		return nil, fmt.Errorf("查询现有排班失败: %w", err)
	}

	// 创建 Pipeline 共享状态
	adjGroupID := ""
	if sch.GroupID != nil {
		adjGroupID = *sch.GroupID
	}
	state := step.NewScheduleState(
		sch.ID,
		sch.TenantModel.OrgNodeID,
		adjGroupID,
		sch.StartDate,
		sch.EndDate,
		sch.CreatedBy,
		&config,
	)
	// 将 DB Assignment 转换为 step.Assignment
	state.Assignments = dbAssignmentsToStep(existing)
	state.EditInput = &input

	// 创建并执行调整 Pipeline
	p := pipeline.NewAdjustPipeline(&pipeline.AdjustDeps{
		RuleService:  s.ruleService,
		ShiftService: s.shiftService,
		EditRepo:     s.repo,
		DraftSaver:   s.repo,
		Broadcaster:  s.broadcaster,
		Logger:       s.logger,
	})

	if err := p.Run(ctx, state); err != nil {
		return nil, fmt.Errorf("排班调整失败: %w", err)
	}

	return &GenerateResult{
		ScheduleID:       scheduleID,
		Status:           StatusReview,
		AssignmentsCount: len(state.Assignments),
		ViolationsCount:  len(state.Violations),
		Violations:       state.Violations,
	}, nil
}

// Validate 手动触发全规则校验。
func (s *Service) Validate(ctx context.Context, scheduleID string) (*GenerateResult, error) {
	sch, err := s.repo.GetScheduleByID(ctx, scheduleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrScheduleNotFound
		}
		return nil, err
	}

	// 解析配置
	var config step.ScheduleConfig
	if len(sch.Config) > 0 {
		json.Unmarshal(sch.Config, &config)
	}

	// 加载现有排班
	existing, err := s.repo.ListAssignmentsBySchedule(ctx, scheduleID)
	if err != nil {
		return nil, fmt.Errorf("查询现有排班失败: %w", err)
	}

	// 创建状态并只运行校验
	valGroupID := ""
	if sch.GroupID != nil {
		valGroupID = *sch.GroupID
	}
	state := step.NewScheduleState(
		sch.ID,
		sch.TenantModel.OrgNodeID,
		valGroupID,
		sch.StartDate,
		sch.EndDate,
		sch.CreatedBy,
		&config,
	)
	state.Assignments = dbAssignmentsToStep(existing)

	// 加载生效规则
	nodeID := tenant.GetOrgNodeID(ctx)
	effectiveRules, err := s.ruleService.ComputeEffectiveRules(ctx, nodeID)
	if err != nil {
		return nil, fmt.Errorf("计算生效规则失败: %w", err)
	}
	state.EffectiveRules = effectiveRules.Rules

	// 使用 ValidationRunner 执行校验
	validationStep := &pipeline.ValidationRunner{}
	if err := validationStep.RunValidation(ctx, state); err != nil {
		return nil, fmt.Errorf("校验失败: %w", err)
	}

	return &GenerateResult{
		ScheduleID:       scheduleID,
		Status:           sch.Status,
		AssignmentsCount: len(state.Assignments),
		ViolationsCount:  len(state.Violations),
		Violations:       state.Violations,
	}, nil
}

// Publish 发布排班。
func (s *Service) Publish(ctx context.Context, id string) error {
	sch, err := s.repo.GetScheduleByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrScheduleNotFound
		}
		return err
	}

	if sch.Status != StatusReview {
		return ErrCannotPublish
	}

	return s.repo.UpdateScheduleStatus(ctx, id, StatusPublished)
}

// GetChanges 查看变更记录。
func (s *Service) GetChanges(ctx context.Context, scheduleID string, opts ChangeListOptions) ([]Change, int64, error) {
	if opts.Page <= 0 {
		opts.Page = 1
	}
	if opts.Size <= 0 {
		opts.Size = 20
	}
	return s.repo.ListChangesBySchedule(ctx, scheduleID, opts)
}

// Delete 删除排班计划。
func (s *Service) Delete(ctx context.Context, id string) error {
	_, err := s.repo.GetScheduleByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrScheduleNotFound
		}
		return err
	}
	return s.repo.DeleteSchedule(ctx, id)
}

// GenerateResult 排班生成结果。
type GenerateResult struct {
	ScheduleID       string           `json:"schedule_id"`
	Status           string           `json:"status"`
	AssignmentsCount int              `json:"assignments_count"`
	ViolationsCount  int              `json:"violations_count"`
	Violations       []step.Violation `json:"violations,omitempty"`
}

// ScheduleSummary 排班统计汇总。
type ScheduleSummary struct {
	TotalAssignments int            `json:"total_assignments"`
	ByShift          map[string]int `json:"by_shift"`
	ByEmployee       map[string]int `json:"by_employee"`
	ByDate           map[string]int `json:"by_date"`
	CoverageRate     float64        `json:"coverage_rate"`
	ComplianceRate   float64        `json:"compliance_rate"`
	ViolationsCount  int            `json:"violations_count"`
}

// GetSummary 获取排班统计汇总。
func (s *Service) GetSummary(ctx context.Context, scheduleID string) (*ScheduleSummary, error) {
	sch, err := s.repo.GetScheduleByID(ctx, scheduleID)
	if err != nil {
		return nil, ErrScheduleNotFound
	}

	assignments, err := s.repo.ListAssignmentsBySchedule(ctx, scheduleID)
	if err != nil {
		return nil, fmt.Errorf("查询排班分配失败: %w", err)
	}

	summary := &ScheduleSummary{
		TotalAssignments: len(assignments),
		ByShift:          make(map[string]int),
		ByEmployee:       make(map[string]int),
		ByDate:           make(map[string]int),
		CoverageRate:     100,
		ComplianceRate:   100,
	}

	for _, a := range assignments {
		summary.ByShift[a.ShiftID]++
		summary.ByEmployee[a.EmployeeID]++
		summary.ByDate[a.Date]++
	}

	// 计算覆盖率（如果有配置）
	if len(sch.Config) > 0 {
		var config step.ScheduleConfig
		if err := json.Unmarshal(sch.Config, &config); err == nil && config.Requirements != nil {
			totalRequired := 0
			for _, dateCounts := range config.Requirements {
				for _, count := range dateCounts {
					totalRequired += count
				}
			}
			if totalRequired > 0 {
				rate := float64(len(assignments)) / float64(totalRequired) * 100
				if rate > 100 {
					rate = 100
				}
				summary.CoverageRate = rate
			}
		}
	}

	return summary, nil
}

// ── 辅助函数 ──

// dbAssignmentsToStep 将 DB 层的 Assignment 转换为 step.Assignment。
func dbAssignmentsToStep(dbAssignments []Assignment) []step.Assignment {
	result := make([]step.Assignment, 0, len(dbAssignments))
	for _, a := range dbAssignments {
		result = append(result, step.Assignment{
			ID:         a.ID,
			ScheduleID: a.ScheduleID,
			EmployeeID: a.EmployeeID,
			ShiftID:    a.ShiftID,
			Date:       a.Date,
			Source:     a.Source,
			OrgNodeID:  a.TenantModel.OrgNodeID,
		})
	}
	return result
}
