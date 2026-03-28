package shift

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gantt-saas/internal/tenant"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrShiftNotFound  = errors.New("班次不存在")
	ErrShiftCodeDup   = errors.New("同节点下班次编码已存在")
	ErrCyclicDep      = errors.New("班次依赖存在循环")
	ErrSelfDep        = errors.New("班次不能依赖自身")
	ErrInvalidDepType = errors.New("无效的依赖类型")
	ErrNotDeptNode    = errors.New("只有科室级（department）节点可以管理班次")
)

// CreateInput 创建班次的输入参数。
type CreateInput struct {
	Name               string          `json:"name"`
	Code               string          `json:"code"`
	Type               string          `json:"type"`
	StartTime          string          `json:"start_time"`
	EndTime            string          `json:"end_time"`
	Duration           int             `json:"duration"`
	IsCrossDay         bool            `json:"is_cross_day"`
	Color              string          `json:"color"`
	Priority           int             `json:"priority"`
	SchedulingPriority *int            `json:"scheduling_priority,omitempty"`
	Description        *string         `json:"description,omitempty"`
	Metadata           json.RawMessage `json:"metadata,omitempty"`
}

// UpdateInput 更新班次的输入参数。
type UpdateInput struct {
	Name               *string          `json:"name,omitempty"`
	Code               *string          `json:"code,omitempty"`
	Type               *string          `json:"type,omitempty"`
	StartTime          *string          `json:"start_time,omitempty"`
	EndTime            *string          `json:"end_time,omitempty"`
	Duration           *int             `json:"duration,omitempty"`
	IsCrossDay         *bool            `json:"is_cross_day,omitempty"`
	Color              *string          `json:"color,omitempty"`
	Priority           *int             `json:"priority,omitempty"`
	SchedulingPriority *int             `json:"scheduling_priority,omitempty"`
	Status             *string          `json:"status,omitempty"`
	IsActive           *bool            `json:"is_active,omitempty"`
	Description        *string          `json:"description,omitempty"`
	Metadata           *json.RawMessage `json:"metadata,omitempty"`
}

// DependencyInput 班次依赖输入。
type DependencyInput struct {
	ShiftID        string `json:"shift_id"`
	DependsOnID    string `json:"depends_on_id"`
	DependencyType string `json:"dependency_type"`
}

type ShiftGroupSetInput struct {
	GroupIDs []string `json:"group_ids"`
}

type FixedAssignmentBatchInput struct {
	Assignments []FixedAssignment `json:"assignments"`
}

type WeeklyStaffItem struct {
	Weekday     int    `json:"weekday"`
	WeekdayName string `json:"weekday_name,omitempty"`
	StaffCount  int    `json:"staff_count"`
	IsCustom    bool   `json:"is_custom"`
}

type WeeklyStaffConfig struct {
	ShiftID      string            `json:"shift_id"`
	ShiftName    string            `json:"shift_name,omitempty"`
	WeeklyConfig []WeeklyStaffItem `json:"weekly_config"`
}

type FixedSchedulePreview struct {
	DateToEmployeeIDs map[string][]string `json:"date_to_employee_ids"`
}

// Service 班次业务逻辑层。
type Service struct {
	repo            *Repository
	orgNodeResolver OrgNodeTypeChecker
}

type OrgNodeTypeChecker interface {
	GetByID(ctx context.Context, id string) (*tenant.OrgNode, error)
}

// NewService 创建班次服务。
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) SetOrgNodeResolver(resolver OrgNodeTypeChecker) {
	s.orgNodeResolver = resolver
}

func (s *Service) ensureDepartmentNode(ctx context.Context) error {
	orgNodeID := tenant.GetOrgNodeID(ctx)
	if orgNodeID == "" {
		return fmt.Errorf("缺少组织节点信息")
	}
	if s.orgNodeResolver == nil {
		return nil
	}
	node, err := s.orgNodeResolver.GetByID(ctx, orgNodeID)
	if err != nil {
		return fmt.Errorf("查询组织节点失败: %w", err)
	}
	if !tenant.IsLeafNodeType(node.NodeType) {
		return ErrNotDeptNode
	}
	return nil
}

// Create 创建班次。
func (s *Service) Create(ctx context.Context, input CreateInput) (*Shift, error) {
	if err := s.ensureDepartmentNode(ctx); err != nil {
		return nil, err
	}

	orgNodeID := tenant.GetOrgNodeID(ctx)
	if orgNodeID == "" {
		return nil, fmt.Errorf("缺少组织节点信息")
	}

	if input.SchedulingPriority != nil {
		input.Priority = *input.SchedulingPriority
	}
	input.Type = normalizeShiftType(input.Type)
	input.Duration, input.IsCrossDay = normalizeShiftDuration(input.StartTime, input.EndTime, input.Duration, input.IsCrossDay)
	input.Metadata = normalizeJSON(input.Metadata)

	// 检查编码唯一性
	existing, err := s.repo.GetByOrgNodeAndCode(ctx, orgNodeID, input.Code)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("检查编码唯一性失败: %w", err)
	}
	if existing != nil {
		return nil, ErrShiftCodeDup
	}

	color := input.Color
	if color == "" {
		color = "#409EFF"
	}

	sh := &Shift{
		ID:          uuid.New().String(),
		Name:        input.Name,
		Code:        input.Code,
		Type:        input.Type,
		StartTime:   input.StartTime,
		EndTime:     input.EndTime,
		Duration:    input.Duration,
		IsCrossDay:  input.IsCrossDay,
		Color:       color,
		Priority:    input.Priority,
		Status:      StatusActive,
		Description: input.Description,
		Metadata:    input.Metadata,
		TenantModel: tenant.TenantModel{
			OrgNodeID: orgNodeID,
		},
	}

	if err := s.repo.Create(ctx, sh); err != nil {
		return nil, fmt.Errorf("创建班次失败: %w", err)
	}

	return s.enrichShift(ctx, sh)
}

// GetByID 获取班次详情。
func (s *Service) GetByID(ctx context.Context, id string) (*Shift, error) {
	if err := s.ensureDepartmentNode(ctx); err != nil {
		return nil, err
	}

	sh, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrShiftNotFound
		}
		return nil, err
	}
	return s.enrichShift(ctx, sh)
}

// Update 更新班次信息。
func (s *Service) Update(ctx context.Context, id string, input UpdateInput) (*Shift, error) {
	if err := s.ensureDepartmentNode(ctx); err != nil {
		return nil, err
	}

	sh, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrShiftNotFound
		}
		return nil, err
	}

	if input.Name != nil {
		sh.Name = *input.Name
	}
	if input.Code != nil {
		existing, err := s.repo.GetByOrgNodeAndCode(ctx, sh.OrgNodeID, *input.Code)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("检查编码唯一性失败: %w", err)
		}
		if existing != nil && existing.ID != sh.ID {
			return nil, ErrShiftCodeDup
		}
		sh.Code = *input.Code
	}
	if input.Type != nil {
		sh.Type = normalizeShiftType(*input.Type)
	}
	if input.StartTime != nil {
		sh.StartTime = *input.StartTime
	}
	if input.EndTime != nil {
		sh.EndTime = *input.EndTime
	}
	if input.Duration != nil {
		sh.Duration = *input.Duration
	}
	if input.IsCrossDay != nil {
		sh.IsCrossDay = *input.IsCrossDay
	}
	if input.Color != nil {
		sh.Color = *input.Color
	}
	if input.SchedulingPriority != nil {
		sh.Priority = *input.SchedulingPriority
	} else if input.Priority != nil {
		sh.Priority = *input.Priority
	}
	if input.Priority != nil {
		sh.Priority = *input.Priority
	}
	if input.IsActive != nil {
		if *input.IsActive {
			sh.Status = StatusActive
		} else {
			sh.Status = StatusDisabled
		}
	}
	if input.Status != nil {
		sh.Status = *input.Status
	}
	if input.Description != nil {
		sh.Description = input.Description
	}
	if input.Metadata != nil {
		sh.Metadata = normalizeJSON(*input.Metadata)
	}
	sh.Duration, sh.IsCrossDay = normalizeShiftDuration(sh.StartTime, sh.EndTime, sh.Duration, sh.IsCrossDay)

	if err := s.repo.Update(ctx, sh); err != nil {
		return nil, fmt.Errorf("更新班次失败: %w", err)
	}

	return s.enrichShift(ctx, sh)
}

// ToggleStatus 切换班次启用状态。
func (s *Service) ToggleStatus(ctx context.Context, id string) (*Shift, error) {
	if err := s.ensureDepartmentNode(ctx); err != nil {
		return nil, err
	}

	sh, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrShiftNotFound
		}
		return nil, err
	}

	if sh.Status == StatusDisabled {
		sh.Status = StatusActive
	} else {
		sh.Status = StatusDisabled
	}

	if err := s.repo.Update(ctx, sh); err != nil {
		return nil, fmt.Errorf("切换班次状态失败: %w", err)
	}

	return s.enrichShift(ctx, sh)
}

// Delete 删除班次。
func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.ensureDepartmentNode(ctx); err != nil {
		return err
	}

	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrShiftNotFound
		}
		return err
	}
	// 删除关联的依赖关系
	if err := s.repo.DeleteDependenciesByShift(ctx, id); err != nil {
		return fmt.Errorf("删除班次依赖失败: %w", err)
	}
	return s.repo.Delete(ctx, id)
}

// List 查询班次列表。
func (s *Service) List(ctx context.Context) ([]Shift, error) {
	if err := s.ensureDepartmentNode(ctx); err != nil {
		return nil, err
	}

	shifts, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	return s.enrichShiftList(ctx, shifts)
}

// GetShiftTimeRange 获取班次的开始和结束时间（满足 step.ShiftTimeResolver 接口）。
func (s *Service) GetShiftTimeRange(ctx context.Context, shiftID string) (string, string, error) {
	sh, err := s.GetByID(ctx, shiftID)
	if err != nil {
		return "", "", err
	}
	return sh.StartTime, sh.EndTime, nil
}

// ListAvailable 查询排班应用可用班次列表。
func (s *Service) ListAvailable(ctx context.Context) ([]Shift, error) {
	shifts, err := s.repo.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	return s.enrichShiftList(ctx, shifts)
}

// GetDependencies 查询依赖关系列表。
func (s *Service) GetDependencies(ctx context.Context) ([]ShiftDependency, error) {
	if err := s.ensureDepartmentNode(ctx); err != nil {
		return nil, err
	}

	return s.repo.GetDependencies(ctx)
}

// AddDependency 添加班次依赖。
func (s *Service) AddDependency(ctx context.Context, input DependencyInput) (*ShiftDependency, error) {
	if err := s.ensureDepartmentNode(ctx); err != nil {
		return nil, err
	}

	if input.ShiftID == input.DependsOnID {
		return nil, ErrSelfDep
	}
	if input.DependencyType != DepTypeSource && input.DependencyType != DepTypeOrder {
		return nil, ErrInvalidDepType
	}

	orgNodeID := tenant.GetOrgNodeID(ctx)
	if orgNodeID == "" {
		return nil, fmt.Errorf("缺少组织节点信息")
	}

	// 验证两个班次都存在
	if _, err := s.repo.GetByID(ctx, input.ShiftID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrShiftNotFound
		}
		return nil, err
	}
	if _, err := s.repo.GetByID(ctx, input.DependsOnID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrShiftNotFound
		}
		return nil, err
	}

	// TODO: 循环依赖检测（拓扑排序）

	dep := &ShiftDependency{
		ID:             uuid.New().String(),
		ShiftID:        input.ShiftID,
		DependsOnID:    input.DependsOnID,
		DependencyType: input.DependencyType,
		TenantModel: tenant.TenantModel{
			OrgNodeID: orgNodeID,
		},
	}

	if err := s.repo.CreateDependency(ctx, dep); err != nil {
		return nil, fmt.Errorf("创建班次依赖失败: %w", err)
	}

	return dep, nil
}

func (s *Service) GetShiftGroups(ctx context.Context, shiftID string) ([]ShiftGroup, error) {
	if err := s.ensureDepartmentNode(ctx); err != nil {
		return nil, err
	}
	if _, err := s.repo.GetByID(ctx, shiftID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrShiftNotFound
		}
		return nil, err
	}
	return s.repo.GetShiftGroups(ctx, shiftID)
}

func (s *Service) SetShiftGroups(ctx context.Context, shiftID string, groupIDs []string) ([]ShiftGroup, error) {
	if err := s.ensureDepartmentNode(ctx); err != nil {
		return nil, err
	}
	if _, err := s.repo.GetByID(ctx, shiftID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrShiftNotFound
		}
		return nil, err
	}
	if err := s.repo.ReplaceShiftGroups(ctx, shiftID, uniqueStrings(groupIDs)); err != nil {
		return nil, err
	}
	return s.repo.GetShiftGroups(ctx, shiftID)
}

func (s *Service) AddGroupToShift(ctx context.Context, shiftID, groupID string, priority int) (*ShiftGroup, error) {
	if err := s.ensureDepartmentNode(ctx); err != nil {
		return nil, err
	}
	orgNodeID := tenant.GetOrgNodeID(ctx)
	item := &ShiftGroup{
		ID:          uuid.New().String(),
		ShiftID:     shiftID,
		GroupID:     groupID,
		Priority:    priority,
		IsActive:    true,
		TenantModel: tenant.TenantModel{OrgNodeID: orgNodeID},
	}
	if err := s.repo.UpsertShiftGroup(ctx, item); err != nil {
		return nil, err
	}
	items, err := s.repo.GetShiftGroups(ctx, shiftID)
	if err != nil {
		return nil, err
	}
	for _, current := range items {
		if current.GroupID == groupID {
			return &current, nil
		}
	}
	return item, nil
}

func (s *Service) RemoveGroupFromShift(ctx context.Context, shiftID, groupID string) error {
	if err := s.ensureDepartmentNode(ctx); err != nil {
		return err
	}
	return s.repo.RemoveShiftGroup(ctx, shiftID, groupID)
}

func (s *Service) GetFixedAssignments(ctx context.Context, shiftID string) ([]FixedAssignment, error) {
	if err := s.ensureDepartmentNode(ctx); err != nil {
		return nil, err
	}
	return s.repo.GetFixedAssignments(ctx, shiftID)
}

func (s *Service) SaveFixedAssignments(ctx context.Context, shiftID string, assignments []FixedAssignment) ([]FixedAssignment, error) {
	if err := s.ensureDepartmentNode(ctx); err != nil {
		return nil, err
	}
	orgNodeID := tenant.GetOrgNodeID(ctx)
	normalized := make([]FixedAssignment, 0, len(assignments))
	for _, assignment := range assignments {
		assignment.ID = strings.TrimSpace(assignment.ID)
		if assignment.ID == "" {
			assignment.ID = uuid.New().String()
		}
		assignment.ShiftID = shiftID
		assignment.TenantModel = tenant.TenantModel{OrgNodeID: orgNodeID}
		assignment.PatternType = normalizePatternType(assignment.PatternType)
		assignment.Weekdays = normalizeJSON(assignment.Weekdays)
		assignment.Monthdays = normalizeJSON(assignment.Monthdays)
		assignment.SpecificDates = normalizeJSON(assignment.SpecificDates)
		normalized = append(normalized, assignment)
	}
	if err := s.repo.ReplaceFixedAssignments(ctx, shiftID, normalized); err != nil {
		return nil, err
	}
	return s.repo.GetFixedAssignments(ctx, shiftID)
}

func (s *Service) DeleteFixedAssignment(ctx context.Context, shiftID, assignmentID string) error {
	if err := s.ensureDepartmentNode(ctx); err != nil {
		return err
	}
	return s.repo.DeleteFixedAssignment(ctx, shiftID, assignmentID)
}

func (s *Service) CalculateFixedSchedule(ctx context.Context, shiftID, startDate, endDate string) (*FixedSchedulePreview, error) {
	calendar, err := s.GetFixedAssignmentsForRange(ctx, []string{shiftID}, startDate, endDate)
	if err != nil {
		return nil, err
	}
	preview := &FixedSchedulePreview{DateToEmployeeIDs: map[string][]string{}}
	if startDate == "" || endDate == "" {
		return preview, nil
	}
	for date, employeeIDs := range calendar[shiftID] {
		preview.DateToEmployeeIDs[date] = append(preview.DateToEmployeeIDs[date], employeeIDs...)
	}
	return preview, nil
}

func (s *Service) GetFixedAssignmentsForRange(ctx context.Context, shiftIDs []string, startDate, endDate string) (map[string]map[string][]string, error) {
	result := make(map[string]map[string][]string, len(shiftIDs))
	shiftIDs = uniqueStrings(shiftIDs)
	if len(shiftIDs) == 0 || startDate == "" || endDate == "" {
		return result, nil
	}
	dates, err := enumerateDates(startDate, endDate)
	if err != nil {
		return nil, err
	}
	for _, shiftID := range shiftIDs {
		assignments, err := s.GetFixedAssignments(ctx, shiftID)
		if err != nil {
			return nil, err
		}
		calendar := make(map[string][]string)
		for _, assignment := range assignments {
			for _, date := range dates {
				if !matchesFixedAssignmentDate(assignment, date) {
					continue
				}
				calendar[date] = appendUniqueString(calendar[date], assignment.EmployeeID)
			}
		}
		result[shiftID] = calendar
	}
	return result, nil
}

func (s *Service) GetWeeklyStaffConfig(ctx context.Context, shiftID string) (*WeeklyStaffConfig, error) {
	if err := s.ensureDepartmentNode(ctx); err != nil {
		return nil, err
	}
	shift, err := s.repo.GetByID(ctx, shiftID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrShiftNotFound
		}
		return nil, err
	}
	items, err := s.repo.GetWeeklyStaffConfig(ctx, shiftID)
	if err != nil {
		return nil, err
	}
	return buildWeeklyStaffConfig(shift, items), nil
}

func (s *Service) UpdateWeeklyStaffConfig(ctx context.Context, shiftID string, config WeeklyStaffConfig) (*WeeklyStaffConfig, error) {
	if err := s.ensureDepartmentNode(ctx); err != nil {
		return nil, err
	}
	orgNodeID := tenant.GetOrgNodeID(ctx)
	rows := make([]ShiftWeeklyStaff, 0, len(config.WeeklyConfig))
	for _, item := range config.WeeklyConfig {
		rows = append(rows, ShiftWeeklyStaff{
			ID:          uuid.New().String(),
			ShiftID:     shiftID,
			Weekday:     item.Weekday,
			StaffCount:  item.StaffCount,
			IsCustom:    item.IsCustom,
			TenantModel: tenant.TenantModel{OrgNodeID: orgNodeID},
		})
	}
	if err := s.repo.ReplaceWeeklyStaffConfig(ctx, shiftID, rows); err != nil {
		return nil, err
	}
	return s.GetWeeklyStaffConfig(ctx, shiftID)
}

func (s *Service) BatchGetWeeklyStaffConfig(ctx context.Context, shiftIDs []string) (map[string]WeeklyStaffConfig, error) {
	if err := s.ensureDepartmentNode(ctx); err != nil {
		return nil, err
	}
	shifts, err := s.repo.ListByIDs(ctx, shiftIDs)
	if err != nil {
		return nil, err
	}
	configs, err := s.repo.BatchGetWeeklyStaffConfig(ctx, shiftIDs)
	if err != nil {
		return nil, err
	}
	result := make(map[string]WeeklyStaffConfig, len(shiftIDs))
	for _, shift := range shifts {
		result[shift.ID] = *buildWeeklyStaffConfig(&shift, configs[shift.ID])
	}
	return result, nil
}

// GetTopologicalOrder 获取班次的拓扑排序（供排班引擎使用）。
func (s *Service) GetTopologicalOrder(ctx context.Context) ([]Shift, error) {
	if err := s.ensureDepartmentNode(ctx); err != nil {
		return nil, err
	}

	shifts, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	deps, err := s.repo.GetDependencies(ctx)
	if err != nil {
		return nil, err
	}

	// 构建 DAG
	inDegree := make(map[string]int)
	graph := make(map[string][]string)
	shiftMap := make(map[string]Shift)

	for _, sh := range shifts {
		inDegree[sh.ID] = 0
		shiftMap[sh.ID] = sh
	}

	for _, dep := range deps {
		if dep.DependencyType == DepTypeOrder {
			graph[dep.DependsOnID] = append(graph[dep.DependsOnID], dep.ShiftID)
			inDegree[dep.ShiftID]++
		}
	}

	// Kahn 算法
	var queue []string
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	var result []Shift
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		result = append(result, shiftMap[id])

		for _, next := range graph[id] {
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}

	if len(result) != len(shifts) {
		return nil, ErrCyclicDep
	}

	return result, nil
}

func normalizeShiftType(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ShiftTypeRegular
	}
	switch value {
	case ShiftTypeRegular, ShiftTypeOvertime, ShiftTypeOnCall, ShiftTypeStandby:
		return value
	default:
		return ShiftTypeRegular
	}
}

func normalizeShiftDuration(startTime, endTime string, duration int, isCrossDay bool) (int, bool) {
	if startTime == "" || endTime == "" {
		return duration, isCrossDay
	}
	var startHour, startMinute, endHour, endMinute int
	if _, err := fmt.Sscanf(startTime, "%d:%d", &startHour, &startMinute); err != nil {
		return duration, isCrossDay
	}
	if _, err := fmt.Sscanf(endTime, "%d:%d", &endHour, &endMinute); err != nil {
		return duration, isCrossDay
	}
	startTotal := startHour*60 + startMinute
	endTotal := endHour*60 + endMinute
	computedCrossDay := endTotal < startTotal
	computedDuration := endTotal - startTotal
	if computedDuration <= 0 {
		computedDuration += 24 * 60
	}
	return computedDuration, computedCrossDay
}

func normalizeJSON(value json.RawMessage) json.RawMessage {
	trimmed := strings.TrimSpace(string(value))
	if trimmed == "" || trimmed == "null" {
		return nil
	}
	return json.RawMessage(trimmed)
}

func normalizePatternType(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "weekly"
	}
	return value
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func appendUniqueString(values []string, value string) []string {
	for _, current := range values {
		if current == value {
			return values
		}
	}
	return append(values, value)
}

func enumerateDates(startDate, endDate string) ([]string, error) {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, fmt.Errorf("解析开始日期失败: %w", err)
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, fmt.Errorf("解析结束日期失败: %w", err)
	}
	if end.Before(start) {
		return nil, fmt.Errorf("结束日期不能早于开始日期")
	}
	result := make([]string, 0, int(end.Sub(start).Hours()/24)+1)
	for current := start; !current.After(end); current = current.AddDate(0, 0, 1) {
		result = append(result, current.Format("2006-01-02"))
	}
	return result, nil
}

func matchesFixedAssignmentDate(assignment FixedAssignment, date string) bool {
	if !assignment.IsActive {
		return false
	}
	if assignment.StartDate != nil && *assignment.StartDate != "" && date < *assignment.StartDate {
		return false
	}
	if assignment.EndDate != nil && *assignment.EndDate != "" && date > *assignment.EndDate {
		return false
	}
	current, err := time.Parse("2006-01-02", date)
	if err != nil {
		return false
	}
	switch assignment.PatternType {
	case "monthly":
		var monthdays []int
		if len(assignment.Monthdays) > 0 {
			if err := json.Unmarshal(assignment.Monthdays, &monthdays); err != nil {
				return false
			}
		}
		day := current.Day()
		for _, item := range monthdays {
			if item == day {
				return true
			}
		}
		return false
	case "specific":
		var specificDates []string
		if len(assignment.SpecificDates) > 0 {
			if err := json.Unmarshal(assignment.SpecificDates, &specificDates); err != nil {
				return false
			}
		}
		for _, item := range specificDates {
			if item == date {
				return true
			}
		}
		return false
	default:
		var weekdays []int
		if len(assignment.Weekdays) > 0 {
			if err := json.Unmarshal(assignment.Weekdays, &weekdays); err != nil {
				return false
			}
		}
		weekday := int(current.Weekday())
		matched := false
		for _, item := range weekdays {
			if item == weekday {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
		if assignment.WeekPattern == nil || *assignment.WeekPattern == "" || *assignment.WeekPattern == "every" {
			return true
		}
		_, week := current.ISOWeek()
		if *assignment.WeekPattern == "odd" {
			return week%2 == 1
		}
		if *assignment.WeekPattern == "even" {
			return week%2 == 0
		}
		return true
	}
}

func (s *Service) enrichShift(ctx context.Context, shift *Shift) (*Shift, error) {
	if shift == nil {
		return nil, nil
	}
	items, err := s.enrichShiftList(ctx, []Shift{*shift})
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return shift, nil
	}
	return &items[0], nil
}

func (s *Service) enrichShiftList(ctx context.Context, shifts []Shift) ([]Shift, error) {
	if len(shifts) == 0 {
		return shifts, nil
	}
	shiftIDs := make([]string, 0, len(shifts))
	for _, shift := range shifts {
		shiftIDs = append(shiftIDs, shift.ID)
	}
	groupCounts, err := s.repo.GetShiftGroupCounts(ctx, shiftIDs)
	if err != nil {
		return nil, err
	}
	groupNames, err := s.repo.GetShiftGroupNames(ctx, shiftIDs)
	if err != nil {
		return nil, err
	}
	fixedCounts, err := s.repo.GetFixedAssignmentCounts(ctx, shiftIDs)
	if err != nil {
		return nil, err
	}
	weeklyConfigs, err := s.repo.BatchGetWeeklyStaffConfig(ctx, shiftIDs)
	if err != nil {
		return nil, err
	}
	result := make([]Shift, 0, len(shifts))
	for _, shift := range shifts {
		shift.IsActive = shift.Status == StatusActive
		shift.GroupCount = groupCounts[shift.ID]
		shift.GroupNames = groupNames[shift.ID]
		shift.FixedAssignmentCount = fixedCounts[shift.ID]
		shift.GroupSummary = buildGroupSummary(shift.GroupNames, shift.GroupCount)
		shift.FixedStaffSummary = buildFixedStaffSummary(shift.FixedAssignmentCount)
		shift.WeeklyStaffSummary = summarizeWeeklyStaff(weeklyConfigs[shift.ID])
		result = append(result, shift)
	}
	return result, nil
}

func buildGroupSummary(groupNames []string, count int64) string {
	if len(groupNames) > 0 {
		return strings.Join(groupNames, "、")
	}
	if count <= 0 {
		return "-"
	}
	return fmt.Sprintf("%d个分组", count)
}

func buildFixedStaffSummary(count int64) string {
	if count <= 0 {
		return "-"
	}
	return fmt.Sprintf("%d人", count)
}

func summarizeWeeklyStaff(items []ShiftWeeklyStaff) string {
	if len(items) == 0 {
		return "-"
	}
	counts := make(map[int]int, len(items))
	for _, item := range items {
		counts[item.Weekday] = item.StaffCount
	}
	weekday := counts[1]
	weekend := counts[0]
	allSame := true
	for day := 0; day < 7; day++ {
		if counts[day] != counts[0] {
			allSame = false
			break
		}
	}
	if allSame {
		return fmt.Sprintf("统一%d人", counts[0])
	}
	weekdaySame := true
	for _, day := range []int{1, 2, 3, 4, 5} {
		if counts[day] != weekday {
			weekdaySame = false
			break
		}
	}
	weekendSame := counts[6] == weekend
	if weekdaySame && weekendSame {
		return fmt.Sprintf("工作日%d人/周末%d人", weekday, weekend)
	}
	configured := 0
	for _, count := range counts {
		if count > 0 {
			configured++
		}
	}
	if configured == 0 {
		return "统一0人"
	}
	return fmt.Sprintf("已配置%d天", configured)
}

func buildWeeklyStaffConfig(shift *Shift, items []ShiftWeeklyStaff) *WeeklyStaffConfig {
	lookup := make(map[int]ShiftWeeklyStaff, len(items))
	for _, item := range items {
		lookup[item.Weekday] = item
	}
	config := &WeeklyStaffConfig{ShiftID: shift.ID, ShiftName: shift.Name}
	for _, day := range []int{1, 2, 3, 4, 5, 6, 0} {
		item := lookup[day]
		config.WeeklyConfig = append(config.WeeklyConfig, WeeklyStaffItem{
			Weekday:     day,
			WeekdayName: weekdayName(day),
			StaffCount:  item.StaffCount,
			IsCustom:    item.IsCustom,
		})
	}
	return config
}

func weekdayName(weekday int) string {
	switch weekday {
	case 1:
		return "周一"
	case 2:
		return "周二"
	case 3:
		return "周三"
	case 4:
		return "周四"
	case 5:
		return "周五"
	case 6:
		return "周六"
	default:
		return "周日"
	}
}
