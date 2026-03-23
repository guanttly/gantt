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

// DepartmentServiceImpl 部门管理服务实现
type DepartmentServiceImpl struct {
	departmentRepo repository.IDepartmentRepository
	logger         logging.ILogger
}

// NewDepartmentService 创建部门管理服务
func NewDepartmentService(
	departmentRepo repository.IDepartmentRepository,
	logger logging.ILogger,
) domain_service.IDepartmentService {
	return &DepartmentServiceImpl{
		departmentRepo: departmentRepo,
		logger:         logger.With("service", "DepartmentService"),
	}
}

// CreateDepartment 创建部门
func (s *DepartmentServiceImpl) CreateDepartment(ctx context.Context, department *model.Department) error {
	// 验证必填字段
	if department.OrgID == "" {
		return fmt.Errorf("orgId is required")
	}
	if department.Name == "" {
		return fmt.Errorf("name is required")
	}
	if department.Code == "" {
		return fmt.Errorf("code is required")
	}

	// 检查编码是否已存在
	existing, err := s.departmentRepo.GetByCode(ctx, department.OrgID, department.Code)
	if err == nil && existing != nil {
		return fmt.Errorf("department code already exists")
	}

	// 生成ID
	if department.ID == "" {
		department.ID = uuid.New().String()
	}

	// 设置默认值
	if department.SortOrder == 0 {
		department.SortOrder = 0
	}
	department.IsActive = true

	// 设置层级和路径
	if department.ParentID != nil && *department.ParentID != "" {
		parent, err := s.departmentRepo.GetByID(ctx, department.OrgID, *department.ParentID)
		if err != nil {
			return fmt.Errorf("parent department not found: %w", err)
		}
		department.Level = parent.Level + 1
		department.BuildPath(parent.Path)
	} else {
		department.Level = 1
		department.BuildPath("")
	}

	// 创建部门
	if err := s.departmentRepo.Create(ctx, department); err != nil {
		s.logger.Error("Failed to create department", "error", err)
		return fmt.Errorf("create department: %w", err)
	}

	s.logger.Info("Department created successfully", "departmentId", department.ID, "name", department.Name)
	return nil
}

// UpdateDepartment 更新部门信息
func (s *DepartmentServiceImpl) UpdateDepartment(ctx context.Context, department *model.Department) error {
	if department.ID == "" {
		return fmt.Errorf("department id is required")
	}
	if department.OrgID == "" {
		return fmt.Errorf("orgId is required")
	}

	// 检查部门是否存在
	exists, err := s.departmentRepo.Exists(ctx, department.OrgID, department.ID)
	if err != nil {
		return fmt.Errorf("check department existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("department not found")
	}

	// 更新部门
	if err := s.departmentRepo.Update(ctx, department); err != nil {
		s.logger.Error("Failed to update department", "error", err)
		return fmt.Errorf("update department: %w", err)
	}

	s.logger.Info("Department updated successfully", "departmentId", department.ID)
	return nil
}

// DeleteDepartment 删除部门
func (s *DepartmentServiceImpl) DeleteDepartment(ctx context.Context, orgID, departmentID string) error {
	if orgID == "" || departmentID == "" {
		return fmt.Errorf("orgId and departmentId are required")
	}

	// 检查部门是否存在
	exists, err := s.departmentRepo.Exists(ctx, orgID, departmentID)
	if err != nil {
		return fmt.Errorf("check department existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("department not found")
	}

	// 检查是否有子部门
	children, err := s.departmentRepo.GetChildren(ctx, orgID, departmentID)
	if err != nil {
		return fmt.Errorf("check children: %w", err)
	}
	if len(children) > 0 {
		return fmt.Errorf("cannot delete department with children")
	}

	// 检查是否有员工
	empCount, err := s.departmentRepo.CountEmployees(ctx, orgID, departmentID)
	if err != nil {
		return fmt.Errorf("count employees: %w", err)
	}
	if empCount > 0 {
		return fmt.Errorf("cannot delete department with employees")
	}

	// 删除部门
	if err := s.departmentRepo.Delete(ctx, orgID, departmentID); err != nil {
		s.logger.Error("Failed to delete department", "error", err)
		return fmt.Errorf("delete department: %w", err)
	}

	s.logger.Info("Department deleted successfully", "departmentId", departmentID)
	return nil
}

// GetDepartment 获取部门详情
func (s *DepartmentServiceImpl) GetDepartment(ctx context.Context, orgID, departmentID string) (*model.Department, error) {
	if orgID == "" || departmentID == "" {
		return nil, fmt.Errorf("orgId and departmentId are required")
	}

	department, err := s.departmentRepo.GetByID(ctx, orgID, departmentID)
	if err != nil {
		return nil, fmt.Errorf("get department: %w", err)
	}

	// 获取员工数量
	empCount, err := s.departmentRepo.CountEmployees(ctx, orgID, departmentID)
	if err == nil {
		department.EmployeeCount = empCount
	}

	return department, nil
}

// ListDepartments 查询部门列表
func (s *DepartmentServiceImpl) ListDepartments(ctx context.Context, filter *model.DepartmentFilter) (*model.DepartmentListResult, error) {
	if filter.OrgID == "" {
		return nil, fmt.Errorf("orgId is required")
	}

	result, err := s.departmentRepo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list departments: %w", err)
	}

	// 为每个部门添加员工数量
	for _, dept := range result.Items {
		empCount, err := s.departmentRepo.CountEmployees(ctx, filter.OrgID, dept.ID)
		if err == nil {
			dept.EmployeeCount = empCount
		}
	}

	return result, nil
}

// GetDepartmentTree 获取部门树
func (s *DepartmentServiceImpl) GetDepartmentTree(ctx context.Context, orgID string) ([]*model.DepartmentTree, error) {
	if orgID == "" {
		return nil, fmt.Errorf("orgId is required")
	}

	// 获取所有部门
	departments, err := s.departmentRepo.GetTreeByOrgID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get department tree: %w", err)
	}

	// 为每个部门添加员工数量
	for _, dept := range departments {
		empCount, err := s.departmentRepo.CountEmployees(ctx, orgID, dept.ID)
		if err == nil {
			dept.EmployeeCount = empCount
		}
	}

	// 构建树形结构
	tree := model.BuildDepartmentTree(departments)

	return tree, nil
}

// GetActiveDepartments 获取所有启用的部门
func (s *DepartmentServiceImpl) GetActiveDepartments(ctx context.Context, orgID string) ([]*model.Department, error) {
	if orgID == "" {
		return nil, fmt.Errorf("orgId is required")
	}

	departments, err := s.departmentRepo.GetActiveDepartments(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get active departments: %w", err)
	}

	return departments, nil
}

// UpdateSortOrder 更新排序
func (s *DepartmentServiceImpl) UpdateSortOrder(ctx context.Context, orgID, departmentID string, sortOrder int) error {
	if orgID == "" || departmentID == "" {
		return fmt.Errorf("orgId and departmentId are required")
	}

	if err := s.departmentRepo.UpdateSortOrder(ctx, orgID, departmentID, sortOrder); err != nil {
		s.logger.Error("Failed to update sort order", "error", err)
		return fmt.Errorf("update sort order: %w", err)
	}

	s.logger.Info("Sort order updated successfully", "departmentId", departmentID, "sortOrder", sortOrder)
	return nil
}

// MoveDepartment 移动部门到新的父部门下
func (s *DepartmentServiceImpl) MoveDepartment(ctx context.Context, orgID, departmentID string, newParentID *string) error {
	if orgID == "" || departmentID == "" {
		return fmt.Errorf("orgId and departmentID are required")
	}

	// 获取当前部门
	department, err := s.departmentRepo.GetByID(ctx, orgID, departmentID)
	if err != nil {
		return fmt.Errorf("get department: %w", err)
	}

	// 检查是否移动到自己的子部门下（防止循环）
	if newParentID != nil && *newParentID != "" {
		newParent, err := s.departmentRepo.GetByID(ctx, orgID, *newParentID)
		if err != nil {
			return fmt.Errorf("new parent department not found: %w", err)
		}

		// 检查新父部门是否是当前部门的后代
		if len(newParent.Path) > len(department.Path) &&
			newParent.Path[:len(department.Path)] == department.Path {
			return fmt.Errorf("cannot move department to its descendant")
		}

		department.ParentID = newParentID
		department.Level = newParent.Level + 1
		department.BuildPath(newParent.Path)
	} else {
		// 移动到顶级
		department.ParentID = nil
		department.Level = 1
		department.BuildPath("")
	}

	// 更新部门
	if err := s.departmentRepo.Update(ctx, department); err != nil {
		s.logger.Error("Failed to move department", "error", err)
		return fmt.Errorf("move department: %w", err)
	}

	// TODO: 更新所有子部门的level和path

	s.logger.Info("Department moved successfully", "departmentId", departmentID)
	return nil
}
