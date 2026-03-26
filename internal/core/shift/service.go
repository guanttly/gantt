package shift

import (
	"context"
	"errors"
	"fmt"

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
)

// CreateInput 创建班次的输入参数。
type CreateInput struct {
	Name       string `json:"name"`
	Code       string `json:"code"`
	StartTime  string `json:"start_time"`
	EndTime    string `json:"end_time"`
	Duration   int    `json:"duration"`
	IsCrossDay bool   `json:"is_cross_day"`
	Color      string `json:"color"`
	Priority   int    `json:"priority"`
}

// UpdateInput 更新班次的输入参数。
type UpdateInput struct {
	Name       *string `json:"name,omitempty"`
	StartTime  *string `json:"start_time,omitempty"`
	EndTime    *string `json:"end_time,omitempty"`
	Duration   *int    `json:"duration,omitempty"`
	IsCrossDay *bool   `json:"is_cross_day,omitempty"`
	Color      *string `json:"color,omitempty"`
	Priority   *int    `json:"priority,omitempty"`
	Status     *string `json:"status,omitempty"`
}

// DependencyInput 班次依赖输入。
type DependencyInput struct {
	ShiftID        string `json:"shift_id"`
	DependsOnID    string `json:"depends_on_id"`
	DependencyType string `json:"dependency_type"`
}

// Service 班次业务逻辑层。
type Service struct {
	repo *Repository
}

// NewService 创建班次服务。
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Create 创建班次。
func (s *Service) Create(ctx context.Context, input CreateInput) (*Shift, error) {
	orgNodeID := tenant.GetOrgNodeID(ctx)
	if orgNodeID == "" {
		return nil, fmt.Errorf("缺少组织节点信息")
	}

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
		ID:         uuid.New().String(),
		Name:       input.Name,
		Code:       input.Code,
		StartTime:  input.StartTime,
		EndTime:    input.EndTime,
		Duration:   input.Duration,
		IsCrossDay: input.IsCrossDay,
		Color:      color,
		Priority:   input.Priority,
		Status:     StatusActive,
		TenantModel: tenant.TenantModel{
			OrgNodeID: orgNodeID,
		},
	}

	if err := s.repo.Create(ctx, sh); err != nil {
		return nil, fmt.Errorf("创建班次失败: %w", err)
	}

	return sh, nil
}

// GetByID 获取班次详情。
func (s *Service) GetByID(ctx context.Context, id string) (*Shift, error) {
	sh, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrShiftNotFound
		}
		return nil, err
	}
	return sh, nil
}

// Update 更新班次信息。
func (s *Service) Update(ctx context.Context, id string, input UpdateInput) (*Shift, error) {
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
	if input.Priority != nil {
		sh.Priority = *input.Priority
	}
	if input.Status != nil {
		sh.Status = *input.Status
	}

	if err := s.repo.Update(ctx, sh); err != nil {
		return nil, fmt.Errorf("更新班次失败: %w", err)
	}

	return sh, nil
}

// ToggleStatus 切换班次启用状态。
func (s *Service) ToggleStatus(ctx context.Context, id string) (*Shift, error) {
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

	return sh, nil
}

// Delete 删除班次。
func (s *Service) Delete(ctx context.Context, id string) error {
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
	return s.repo.List(ctx)
}

// ListAvailable 查询排班应用可用班次列表。
func (s *Service) ListAvailable(ctx context.Context) ([]Shift, error) {
	return s.repo.ListActive(ctx)
}

// GetDependencies 查询依赖关系列表。
func (s *Service) GetDependencies(ctx context.Context) ([]ShiftDependency, error) {
	return s.repo.GetDependencies(ctx)
}

// AddDependency 添加班次依赖。
func (s *Service) AddDependency(ctx context.Context, input DependencyInput) (*ShiftDependency, error) {
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

// GetTopologicalOrder 获取班次的拓扑排序（供排班引擎使用）。
func (s *Service) GetTopologicalOrder(ctx context.Context) ([]Shift, error) {
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
