package subscription

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

var (
	ErrSubscriptionNotFound      = errors.New("订阅不存在")
	ErrQuotaExceeded             = errors.New("超出订阅配额限制")
	ErrInvalidPlan               = errors.New("无效的套餐类型")
	ErrSubscriptionAlreadyExists = errors.New("该组织已存在订阅")
)

// CreateInput 创建订阅的输入参数。
type CreateInput struct {
	OrgNodeID    string  `json:"org_node_id"`
	Plan         string  `json:"plan"`
	MaxEmployees *int    `json:"max_employees,omitempty"`
	MaxAITokens  *int    `json:"max_ai_tokens,omitempty"`
	EndDate      *string `json:"end_date,omitempty"` // YYYY-MM-DD
}

// UpdateInput 更新订阅的输入参数。
type UpdateInput struct {
	Plan         *string `json:"plan,omitempty"`
	Status       *string `json:"status,omitempty"`
	MaxEmployees *int    `json:"max_employees,omitempty"`
	MaxAITokens  *int    `json:"max_ai_tokens,omitempty"`
	EndDate      *string `json:"end_date,omitempty"`
}

// Service 订阅业务逻辑层。
type Service struct {
	repo *Repository
}

// NewService 创建订阅服务。
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Create 创建订阅。
func (s *Service) Create(ctx context.Context, input CreateInput) (*Subscription, error) {
	if input.Plan == "" {
		input.Plan = PlanFree
	}
	defaults, ok := PlanDefaults[input.Plan]
	if !ok {
		return nil, ErrInvalidPlan
	}

	// 检查是否已存在订阅
	existing, err := s.repo.GetByOrgNode(ctx, input.OrgNodeID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("检查订阅失败: %w", err)
	}
	if existing != nil {
		return nil, ErrSubscriptionAlreadyExists
	}

	maxEmp := defaults.MaxEmployees
	if input.MaxEmployees != nil {
		maxEmp = *input.MaxEmployees
	}
	maxTokens := defaults.MaxAITokens
	if input.MaxAITokens != nil {
		maxTokens = *input.MaxAITokens
	}

	sub := &Subscription{
		OrgNodeID:    input.OrgNodeID,
		Plan:         input.Plan,
		Status:       StatusActive,
		MaxEmployees: maxEmp,
		MaxAITokens:  maxTokens,
		StartDate:    time.Now(),
	}

	if input.EndDate != nil {
		endDate, err := time.Parse("2006-01-02", *input.EndDate)
		if err != nil {
			return nil, fmt.Errorf("end_date 格式错误: %w", err)
		}
		sub.EndDate = &endDate
	}

	if err := s.repo.Create(ctx, sub); err != nil {
		return nil, fmt.Errorf("创建订阅失败: %w", err)
	}
	return sub, nil
}

// GetByID 获取订阅详情。
func (s *Service) GetByID(ctx context.Context, id string) (*Subscription, error) {
	sub, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSubscriptionNotFound
		}
		return nil, err
	}
	return sub, nil
}

// GetByOrgNode 根据组织节点获取订阅。
func (s *Service) GetByOrgNode(ctx context.Context, orgNodeID string) (*Subscription, error) {
	sub, err := s.repo.GetByOrgNode(ctx, orgNodeID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSubscriptionNotFound
		}
		return nil, err
	}
	return sub, nil
}

// Update 更新订阅。
func (s *Service) Update(ctx context.Context, id string, input UpdateInput) (*Subscription, error) {
	sub, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSubscriptionNotFound
		}
		return nil, err
	}

	if input.Plan != nil {
		if _, ok := PlanDefaults[*input.Plan]; !ok {
			return nil, ErrInvalidPlan
		}
		sub.Plan = *input.Plan
	}
	if input.Status != nil {
		sub.Status = *input.Status
	}
	if input.MaxEmployees != nil {
		sub.MaxEmployees = *input.MaxEmployees
	}
	if input.MaxAITokens != nil {
		sub.MaxAITokens = *input.MaxAITokens
	}
	if input.EndDate != nil {
		endDate, err := time.Parse("2006-01-02", *input.EndDate)
		if err != nil {
			return nil, fmt.Errorf("end_date 格式错误: %w", err)
		}
		sub.EndDate = &endDate
	}

	if err := s.repo.Update(ctx, sub); err != nil {
		return nil, fmt.Errorf("更新订阅失败: %w", err)
	}
	return sub, nil
}

// List 分页查询订阅列表。
func (s *Service) List(ctx context.Context, opts ListOptions) ([]Subscription, int64, error) {
	if opts.Page <= 0 {
		opts.Page = 1
	}
	if opts.Size <= 0 {
		opts.Size = 20
	}
	if opts.Size > 100 {
		opts.Size = 100
	}
	return s.repo.List(ctx, opts)
}

// CheckEmployeeQuota 检查员工配额。
func (s *Service) CheckEmployeeQuota(ctx context.Context, orgNodeID string) error {
	sub, err := s.repo.GetByOrgNode(ctx, orgNodeID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 无订阅，使用 free 默认值
			sub = &Subscription{
				MaxEmployees: PlanDefaults[PlanFree].MaxEmployees,
			}
		} else {
			return fmt.Errorf("获取订阅失败: %w", err)
		}
	}

	if sub.IsUnlimitedEmployees() {
		return nil
	}

	currentCount, err := s.repo.CountEmployees(ctx, orgNodeID)
	if err != nil {
		return fmt.Errorf("统计员工数量失败: %w", err)
	}

	if int(currentCount) >= sub.MaxEmployees {
		return fmt.Errorf("%w: 当前 %d / 上限 %d", ErrQuotaExceeded, currentCount, sub.MaxEmployees)
	}
	return nil
}
