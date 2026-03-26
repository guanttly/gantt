package employee

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gantt-saas/internal/tenant"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrEmployeeNotFound = errors.New("员工不存在")
	ErrEmployeeNoDup    = errors.New("同节点下工号已存在")
)

// CreateInput 创建员工的输入参数。
type CreateInput struct {
	Name       string  `json:"name"`
	EmployeeNo *string `json:"employee_no"`
	Phone      *string `json:"phone"`
	Email      *string `json:"email"`
	Position   *string `json:"position"`
	Category   *string `json:"category"`
	HireDate   *string `json:"hire_date"`
}

// UpdateInput 更新员工的输入参数。
type UpdateInput struct {
	Name       *string `json:"name,omitempty"`
	EmployeeNo *string `json:"employee_no,omitempty"`
	Phone      *string `json:"phone,omitempty"`
	Email      *string `json:"email,omitempty"`
	Position   *string `json:"position,omitempty"`
	Category   *string `json:"category,omitempty"`
	Status     *string `json:"status,omitempty"`
	HireDate   *string `json:"hire_date,omitempty"`
}

// Service 员工业务逻辑层。
type Service struct {
	repo           *Repository
	appRoleCleaner AppRoleCleaner
}

type AppRoleCleaner interface {
	CleanupEmployeeRoles(ctx context.Context, employeeID string) error
}

// NewService 创建员工服务。
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) SetAppRoleCleaner(cleaner AppRoleCleaner) {
	s.appRoleCleaner = cleaner
}

// Create 创建员工。
func (s *Service) Create(ctx context.Context, input CreateInput) (*Employee, error) {
	orgNodeID := tenant.GetOrgNodeID(ctx)
	if orgNodeID == "" {
		return nil, fmt.Errorf("缺少组织节点信息")
	}

	// 检查工号唯一性
	if input.EmployeeNo != nil && *input.EmployeeNo != "" {
		existing, err := s.repo.GetByOrgNodeAndNo(ctx, orgNodeID, *input.EmployeeNo)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("检查工号唯一性失败: %w", err)
		}
		if existing != nil {
			return nil, ErrEmployeeNoDup
		}
	}

	emp := &Employee{
		ID:              uuid.New().String(),
		Name:            input.Name,
		EmployeeNo:      input.EmployeeNo,
		Phone:           input.Phone,
		Email:           input.Email,
		Position:        input.Position,
		Category:        input.Category,
		SchedulingRole:  SchedulingRoleEmployee,
		AppMustResetPwd: true,
		HireDate:        input.HireDate,
		Status:          StatusActive,
		TenantModel: tenant.TenantModel{
			OrgNodeID: orgNodeID,
		},
	}

	defaultPassword := buildDefaultAppPassword(emp)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("生成员工应用密码失败: %w", err)
	}
	emp.AppPasswordHash = &[]string{string(hashedPassword)}[0]
	emp.AppDefaultPassword = &defaultPassword

	if err := s.repo.Create(ctx, emp); err != nil {
		return nil, fmt.Errorf("创建员工失败: %w", err)
	}

	return emp, nil
}

// GetByID 获取员工详情。
func (s *Service) GetByID(ctx context.Context, id string) (*Employee, error) {
	emp, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEmployeeNotFound
		}
		return nil, err
	}
	return emp, nil
}

func buildDefaultAppPassword(emp *Employee) string {
	if emp.EmployeeNo != nil && strings.TrimSpace(*emp.EmployeeNo) != "" {
		return strings.TrimSpace(*emp.EmployeeNo) + "@App1"
	}
	if emp.Phone != nil && strings.TrimSpace(*emp.Phone) != "" {
		phone := strings.TrimSpace(*emp.Phone)
		if len(phone) > 4 {
			phone = phone[len(phone)-4:]
		}
		return "Emp" + phone + "@App1"
	}
	identifier := emp.ID
	if len(identifier) > 6 {
		identifier = identifier[len(identifier)-6:]
	}
	return "Emp" + identifier + "@App1"
}

// Update 更新员工信息。
func (s *Service) Update(ctx context.Context, id string, input UpdateInput) (*Employee, error) {
	emp, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEmployeeNotFound
		}
		return nil, err
	}

	if input.Name != nil {
		emp.Name = *input.Name
	}
	if input.EmployeeNo != nil {
		// 检查新工号唯一性
		if *input.EmployeeNo != "" {
			existing, err := s.repo.GetByOrgNodeAndNo(ctx, emp.OrgNodeID, *input.EmployeeNo)
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("检查工号唯一性失败: %w", err)
			}
			if existing != nil && existing.ID != emp.ID {
				return nil, ErrEmployeeNoDup
			}
		}
		emp.EmployeeNo = input.EmployeeNo
	}
	if input.Phone != nil {
		emp.Phone = input.Phone
	}
	if input.Email != nil {
		emp.Email = input.Email
	}
	if input.Position != nil {
		emp.Position = input.Position
	}
	if input.Category != nil {
		emp.Category = input.Category
	}
	if input.Status != nil {
		emp.Status = *input.Status
	}
	if input.HireDate != nil {
		emp.HireDate = input.HireDate
	}

	if err := s.repo.Update(ctx, emp); err != nil {
		return nil, fmt.Errorf("更新员工失败: %w", err)
	}

	if input.Status != nil && *input.Status == StatusInactive && s.appRoleCleaner != nil {
		if err := s.appRoleCleaner.CleanupEmployeeRoles(ctx, emp.ID); err != nil {
			return nil, err
		}
	}

	return emp, nil
}

// Delete 删除员工。
func (s *Service) Delete(ctx context.Context, id string) error {
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrEmployeeNotFound
		}
		return err
	}
	if s.appRoleCleaner != nil {
		if err := s.appRoleCleaner.CleanupEmployeeRoles(ctx, id); err != nil {
			return err
		}
	}
	return s.repo.Delete(ctx, id)
}

// List 分页查询员工列表。
func (s *Service) List(ctx context.Context, opts ListOptions) ([]Employee, int64, error) {
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
