package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/domain/repository"
	"jusha/mcp/pkg/logging"

	domain_service "jusha/gantt/service/management/domain/service"
)

// EmployeeServiceImpl 员工管理服务实现
type EmployeeServiceImpl struct {
	repo      repository.IEmployeeRepository
	groupRepo repository.IGroupRepository
	logger    logging.ILogger
}

// NewEmployeeService 创建员工管理服务
func NewEmployeeService(repo repository.IEmployeeRepository, groupRepo repository.IGroupRepository, logger logging.ILogger) domain_service.IEmployeeService {
	return &EmployeeServiceImpl{
		repo:      repo,
		groupRepo: groupRepo,
		logger:    logger.With("service", "EmployeeService"),
	}
}

// CreateEmployee 创建员工
func (s *EmployeeServiceImpl) CreateEmployee(ctx context.Context, employee *model.Employee) error {
	// 验证必填字段
	if employee.OrgID == "" {
		return fmt.Errorf("orgId is required")
	}
	if employee.EmployeeID == "" {
		return fmt.Errorf("employeeId is required")
	}
	if employee.Name == "" {
		return fmt.Errorf("name is required")
	}

	// 生成ID
	if employee.ID == "" {
		employee.ID = uuid.New().String()
	}

	// 设置默认状态
	if employee.Status == "" {
		employee.Status = model.EmployeeStatusActive
	}

	// 检查工号是否已存在
	exists, err := s.repo.Exists(ctx, employee.OrgID, employee.EmployeeID)
	if err != nil {
		s.logger.Error("Failed to check employee existence", "error", err)
		return fmt.Errorf("check employee existence: %w", err)
	}
	if exists {
		return fmt.Errorf("employee with ID %s already exists", employee.EmployeeID)
	}

	// 创建员工
	if err := s.repo.Create(ctx, employee); err != nil {
		s.logger.Error("Failed to create employee", "error", err)
		return fmt.Errorf("create employee: %w", err)
	}

	s.logger.Info("Employee created successfully", "employeeId", employee.ID)
	return nil
}

// UpdateEmployee 更新员工信息
func (s *EmployeeServiceImpl) UpdateEmployee(ctx context.Context, employee *model.Employee) error {
	if employee.ID == "" {
		return fmt.Errorf("employee id is required")
	}
	if employee.OrgID == "" {
		return fmt.Errorf("orgId is required")
	}

	// 检查员工是否存在
	exists, err := s.repo.Exists(ctx, employee.OrgID, employee.ID)
	if err != nil {
		return fmt.Errorf("check employee existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("employee not found")
	}

	if err := s.repo.Update(ctx, employee); err != nil {
		s.logger.Error("Failed to update employee", "error", err)
		return fmt.Errorf("update employee: %w", err)
	}

	s.logger.Info("Employee updated successfully", "employeeId", employee.ID)
	return nil
}

// DeleteEmployee 删除员工
func (s *EmployeeServiceImpl) DeleteEmployee(ctx context.Context, orgID, employeeID string) error {
	if orgID == "" || employeeID == "" {
		return fmt.Errorf("orgId and employeeId are required")
	}

	// 检查员工是否存在
	exists, err := s.repo.Exists(ctx, orgID, employeeID)
	if err != nil {
		return fmt.Errorf("check employee existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("employee not found")
	}

	if err := s.repo.Delete(ctx, orgID, employeeID); err != nil {
		s.logger.Error("Failed to delete employee", "error", err)
		return fmt.Errorf("delete employee: %w", err)
	}

	s.logger.Info("Employee deleted successfully", "employeeId", employeeID)
	return nil
}

// GetEmployee 获取员工详情
func (s *EmployeeServiceImpl) GetEmployee(ctx context.Context, orgID, employeeID string) (*model.Employee, error) {
	if orgID == "" || employeeID == "" {
		return nil, fmt.Errorf("orgId and employeeId are required")
	}

	employee, err := s.repo.GetByID(ctx, orgID, employeeID)
	if err != nil {
		s.logger.Error("Failed to get employee", "error", err)
		return nil, fmt.Errorf("get employee: %w", err)
	}

	// 注意：GetEmployee 不查询分组信息，因为员工详情页面通常不需要分组
	// 如果将来需要，可以添加一个 includeGroups 参数
	employee.Groups = []*model.Group{}

	return employee, nil
}

// ListEmployees 查询员工列表
func (s *EmployeeServiceImpl) ListEmployees(ctx context.Context, filter *model.EmployeeFilter) (*model.EmployeeListResult, error) {
	if filter == nil {
		filter = &model.EmployeeFilter{}
	}
	if filter.OrgID == "" {
		return nil, fmt.Errorf("orgId is required")
	}

	// 设置默认分页
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 20
	}

	result, err := s.repo.List(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to list employees", "error", err)
		return nil, fmt.Errorf("list employees: %w", err)
	}

	// 只在需要时才批量获取所有员工的分组信息
	// 分组信息仅在员工管理页面需要，排班等场景不需要，避免不必要的数据库查询
	if filter.IncludeGroups && len(result.Items) > 0 {
		// 收集所有员工ID
		employeeIDs := make([]string, len(result.Items))
		for i, emp := range result.Items {
			employeeIDs[i] = emp.ID
		}

		// 批量查询分组信息
		groupsMap, err := s.groupRepo.BatchGetMemberGroups(ctx, filter.OrgID, employeeIDs)
		if err != nil {
			s.logger.Warn("Failed to batch get employee groups", "error", err)
			// 即使获取分组失败，也不影响员工列表的返回
			for _, emp := range result.Items {
				emp.Groups = []*model.Group{}
			}
		} else {
			// 将分组信息分配给对应的员工
			for _, emp := range result.Items {
				if groups, exists := groupsMap[emp.ID]; exists {
					emp.Groups = groups
				} else {
					emp.Groups = []*model.Group{}
				}
			}
		}
	} else {
		// 不需要分组信息时，确保 Groups 字段为空数组
		for _, emp := range result.Items {
			emp.Groups = []*model.Group{}
		}
	}

	return result, nil
}

// ListSimpleEmployees 查询简单员工列表（不加载分组信息，性能更好）
func (s *EmployeeServiceImpl) ListSimpleEmployees(ctx context.Context, filter *model.EmployeeFilter) (*model.EmployeeListResult, error) {
	if filter == nil {
		filter = &model.EmployeeFilter{}
	}
	if filter.OrgID == "" {
		return nil, fmt.Errorf("orgId is required")
	}

	// 设置默认分页（简单查询使用更大的默认页大小）
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 50
	}

	result, err := s.repo.List(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to list simple employees", "error", err)
		return nil, fmt.Errorf("list simple employees: %w", err)
	}

	// 简单查询不加载分组信息，直接返回
	return result, nil
}

// UpdateEmployeeStatus 更新员工状态
func (s *EmployeeServiceImpl) UpdateEmployeeStatus(ctx context.Context, orgID, employeeID string, status model.EmployeeStatus) error {
	if orgID == "" || employeeID == "" {
		return fmt.Errorf("orgId and employeeId are required")
	}

	// 验证状态值
	validStatuses := map[model.EmployeeStatus]bool{
		model.EmployeeStatusActive:   true,
		model.EmployeeStatusInactive: true,
		model.EmployeeStatusLeave:    true,
		model.EmployeeStatusSuspend:  true,
	}
	if !validStatuses[status] {
		return fmt.Errorf("invalid status: %s", status)
	}

	if err := s.repo.UpdateStatus(ctx, orgID, employeeID, status); err != nil {
		s.logger.Error("Failed to update employee status", "error", err)
		return fmt.Errorf("update employee status: %w", err)
	}

	s.logger.Info("Employee status updated", "employeeId", employeeID, "status", status)
	return nil
}

// BatchGetEmployees 批量获取员工
func (s *EmployeeServiceImpl) BatchGetEmployees(ctx context.Context, orgID string, employeeIDs []string) ([]*model.Employee, error) {
	if orgID == "" {
		return nil, fmt.Errorf("orgId is required")
	}
	if len(employeeIDs) == 0 {
		return []*model.Employee{}, nil
	}

	employees, err := s.repo.BatchGet(ctx, orgID, employeeIDs)
	if err != nil {
		s.logger.Error("Failed to batch get employees", "error", err)
		return nil, fmt.Errorf("batch get employees: %w", err)
	}

	return employees, nil
}
