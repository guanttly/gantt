package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/domain/repository"
	"jusha/mcp/pkg/logging"

	domain_service "jusha/gantt/service/management/domain/service"
)

// ShiftServiceImpl 班次管理服务实现
type ShiftServiceImpl struct {
	shiftRepo           repository.IShiftRepository
	shiftAssignmentRepo repository.IShiftAssignmentRepository
	shiftGroupRepo      repository.IShiftGroupRepository // 班次-分组关联仓储
	groupRepo           repository.IGroupRepository      // 分组仓储
	employeeRepo        repository.IEmployeeRepository
	logger              logging.ILogger
}

// NewShiftService 创建班次管理服务
func NewShiftService(
	shiftRepo repository.IShiftRepository,
	shiftAssignmentRepo repository.IShiftAssignmentRepository,
	shiftGroupRepo repository.IShiftGroupRepository,
	groupRepo repository.IGroupRepository,
	employeeRepo repository.IEmployeeRepository,
	logger logging.ILogger,
) domain_service.IShiftService {
	return &ShiftServiceImpl{
		shiftRepo:           shiftRepo,
		shiftAssignmentRepo: shiftAssignmentRepo,
		shiftGroupRepo:      shiftGroupRepo,
		groupRepo:           groupRepo,
		employeeRepo:        employeeRepo,
		logger:              logger.With("service", "ShiftService"),
	}
}

// CreateShift 创建班次
func (s *ShiftServiceImpl) CreateShift(ctx context.Context, shift *model.Shift) error {
	// 验证必填字段
	if shift.OrgID == "" {
		return fmt.Errorf("orgId is required")
	}
	if shift.Name == "" {
		return fmt.Errorf("name is required")
	}
	if shift.StartTime == "" {
		return fmt.Errorf("startTime is required")
	}
	if shift.EndTime == "" {
		return fmt.Errorf("endTime is required")
	}

	// 生成ID
	if shift.ID == "" {
		shift.ID = uuid.New().String()
	}

	// 设置默认启用状态
	shift.IsActive = true

	// 验证时间格式和逻辑
	if err := shift.ValidateShiftTime(); err != nil {
		return fmt.Errorf("invalid shift time: %w", err)
	}

	// 创建班次
	if err := s.shiftRepo.Create(ctx, shift); err != nil {
		s.logger.Error("Failed to create shift", "error", err)
		return fmt.Errorf("create shift: %w", err)
	}

	s.logger.Info("Shift created successfully", "shiftId", shift.ID, "name", shift.Name)
	return nil
}

// UpdateShift 更新班次信息
func (s *ShiftServiceImpl) UpdateShift(ctx context.Context, shift *model.Shift) error {
	if shift.ID == "" {
		return fmt.Errorf("shift id is required")
	}
	if shift.OrgID == "" {
		return fmt.Errorf("orgId is required")
	}

	// 检查班次是否存在
	exists, err := s.shiftRepo.Exists(ctx, shift.OrgID, shift.ID)
	if err != nil {
		return fmt.Errorf("check shift existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("shift not found")
	}

	// 验证时间格式和逻辑
	if shift.StartTime != "" && shift.EndTime != "" {
		if err := shift.ValidateShiftTime(); err != nil {
			return fmt.Errorf("invalid shift time: %w", err)
		}
	}

	if err := s.shiftRepo.Update(ctx, shift); err != nil {
		s.logger.Error("Failed to update shift", "error", err)
		return fmt.Errorf("update shift: %w", err)
	}

	s.logger.Info("Shift updated successfully", "shiftId", shift.ID)
	return nil
}

// DeleteShift 删除班次
func (s *ShiftServiceImpl) DeleteShift(ctx context.Context, orgID, shiftID string) error {
	if orgID == "" || shiftID == "" {
		return fmt.Errorf("orgId and shiftId are required")
	}

	// 检查班次是否存在
	exists, err := s.shiftRepo.Exists(ctx, orgID, shiftID)
	if err != nil {
		return fmt.Errorf("check shift existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("shift not found")
	}

	// 检查是否有未来的班次分配
	// 可以在这里添加业务规则,比如不允许删除已有分配的班次

	if err := s.shiftRepo.Delete(ctx, orgID, shiftID); err != nil {
		s.logger.Error("Failed to delete shift", "error", err)
		return fmt.Errorf("delete shift: %w", err)
	}

	s.logger.Info("Shift deleted successfully", "shiftId", shiftID)
	return nil
}

// GetShift 获取班次详情
func (s *ShiftServiceImpl) GetShift(ctx context.Context, orgID, shiftID string) (*model.Shift, error) {
	if orgID == "" || shiftID == "" {
		return nil, fmt.Errorf("orgId and shiftId are required")
	}

	shift, err := s.shiftRepo.GetByID(ctx, orgID, shiftID)
	if err != nil {
		s.logger.Error("Failed to get shift", "error", err)
		return nil, fmt.Errorf("get shift: %w", err)
	}

	return shift, nil
}

// ListShifts 查询班次列表
func (s *ShiftServiceImpl) ListShifts(ctx context.Context, filter *model.ShiftFilter) (*model.ShiftListResult, error) {
	if filter == nil {
		filter = &model.ShiftFilter{}
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

	result, err := s.shiftRepo.List(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to list shifts", "error", err)
		return nil, fmt.Errorf("list shifts: %w", err)
	}

	return result, nil
}

// GetActiveShifts 获取所有启用的班次
func (s *ShiftServiceImpl) GetActiveShifts(ctx context.Context, orgID string) ([]*model.Shift, error) {
	if orgID == "" {
		return nil, fmt.Errorf("orgId is required")
	}

	shifts, err := s.shiftRepo.GetActiveShifts(ctx, orgID)
	if err != nil {
		s.logger.Error("Failed to get active shifts", "error", err)
		return nil, fmt.Errorf("get active shifts: %w", err)
	}

	return shifts, nil
}

// AssignShift 分配班次给员工
func (s *ShiftServiceImpl) AssignShift(ctx context.Context, assignment *model.ShiftAssignment) error {
	// 验证必填字段
	if assignment.OrgID == "" {
		return fmt.Errorf("orgId is required")
	}
	if assignment.EmployeeID == "" {
		return fmt.Errorf("employeeId is required")
	}
	if assignment.ShiftID == "" {
		return fmt.Errorf("shiftId is required")
	}
	if assignment.Date.IsZero() {
		return fmt.Errorf("date is required")
	}

	// 生成ID
	if assignment.ID == "" {
		assignment.ID = uuid.New().String()
	}

	// 验证员工存在
	empExists, err := s.employeeRepo.Exists(ctx, assignment.OrgID, assignment.EmployeeID)
	if err != nil {
		return fmt.Errorf("check employee existence: %w", err)
	}
	if !empExists {
		return fmt.Errorf("employee not found: %s", assignment.EmployeeID)
	}

	// 验证班次存在
	shiftExists, err := s.shiftRepo.Exists(ctx, assignment.OrgID, assignment.ShiftID)
	if err != nil {
		return fmt.Errorf("check shift existence: %w", err)
	}
	if !shiftExists {
		return fmt.Errorf("shift not found: %s", assignment.ShiftID)
	}

	// 检查是否已有该日期的班次分配
	existing, err := s.shiftAssignmentRepo.GetByEmployeeAndDate(ctx, assignment.OrgID, assignment.EmployeeID, assignment.Date)
	if err == nil && existing != nil {
		return fmt.Errorf("employee already has shift assignment on %s", assignment.Date.Format("2006-01-02"))
	}

	// 创建班次分配
	if err := s.shiftAssignmentRepo.Create(ctx, assignment); err != nil {
		s.logger.Error("Failed to assign shift", "error", err)
		return fmt.Errorf("assign shift: %w", err)
	}

	s.logger.Info("Shift assigned successfully",
		"assignment_id", assignment.ID,
		"employeeId", assignment.EmployeeID,
		"shiftId", assignment.ShiftID,
		"date", assignment.Date.Format("2006-01-02"),
	)
	return nil
}

// GetEmployeeShifts 获取员工的班次安排
func (s *ShiftServiceImpl) GetEmployeeShifts(ctx context.Context, orgID, employeeID string, startDate, endDate time.Time) ([]*model.ShiftAssignment, error) {
	if orgID == "" || employeeID == "" {
		return nil, fmt.Errorf("orgId and employeeId are required")
	}
	if startDate.IsZero() || endDate.IsZero() {
		return nil, fmt.Errorf("startDate and endDate are required")
	}
	if startDate.After(endDate) {
		return nil, fmt.Errorf("startDate must be before endDate")
	}

	assignments, err := s.shiftAssignmentRepo.ListByEmployee(ctx, orgID, employeeID, startDate, endDate)
	if err != nil {
		s.logger.Error("Failed to get employee shifts", "error", err)
		return nil, fmt.Errorf("get employee shifts: %w", err)
	}

	return assignments, nil
}

// GetShiftAssignments 获取日期范围内的班次分配
func (s *ShiftServiceImpl) GetShiftAssignments(ctx context.Context, orgID string, startDate, endDate time.Time) ([]*model.ShiftAssignment, error) {
	if orgID == "" {
		return nil, fmt.Errorf("orgId is required")
	}
	if startDate.IsZero() || endDate.IsZero() {
		return nil, fmt.Errorf("startDate and endDate are required")
	}
	if startDate.After(endDate) {
		return nil, fmt.Errorf("startDate must be before endDate")
	}

	assignments, err := s.shiftAssignmentRepo.ListByDateRange(ctx, orgID, startDate, endDate)
	if err != nil {
		s.logger.Error("Failed to get shift assignments", "error", err)
		return nil, fmt.Errorf("get shift assignments: %w", err)
	}

	return assignments, nil
}

// AddGroupToShift 为班次添加关联分组
func (s *ShiftServiceImpl) AddGroupToShift(ctx context.Context, shiftID, groupID string, priority int) error {
	if shiftID == "" {
		return fmt.Errorf("shiftId is required")
	}
	if groupID == "" {
		return fmt.Errorf("groupId is required")
	}

	// 检查关联是否已存在
	exists, err := s.shiftGroupRepo.ExistsShiftGroup(ctx, shiftID, groupID)
	if err != nil {
		s.logger.Error("Failed to check shift-group existence", "error", err)
		return fmt.Errorf("check shift-group existence: %w", err)
	}
	if exists {
		return fmt.Errorf("shift-group association already exists")
	}

	// 创建关联
	shiftGroup := &model.ShiftGroup{
		ShiftID:   shiftID,
		GroupID:   groupID,
		Priority:  priority,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.shiftGroupRepo.AddGroupToShift(ctx, shiftGroup); err != nil {
		s.logger.Error("Failed to add group to shift", "error", err)
		return fmt.Errorf("add group to shift: %w", err)
	}

	s.logger.Info("Group added to shift successfully", "shiftId", shiftID, "groupId", groupID)
	return nil
}

// RemoveGroupFromShift 从班次移除关联分组
func (s *ShiftServiceImpl) RemoveGroupFromShift(ctx context.Context, shiftID, groupID string) error {
	if shiftID == "" {
		return fmt.Errorf("shiftId is required")
	}
	if groupID == "" {
		return fmt.Errorf("groupId is required")
	}

	if err := s.shiftGroupRepo.RemoveGroupFromShift(ctx, shiftID, groupID); err != nil {
		s.logger.Error("Failed to remove group from shift", "error", err)
		return fmt.Errorf("remove group from shift: %w", err)
	}

	s.logger.Info("Group removed from shift successfully", "shiftId", shiftID, "groupId", groupID)
	return nil
}

// SetShiftGroups 批量设置班次的关联分组
func (s *ShiftServiceImpl) SetShiftGroups(ctx context.Context, shiftID string, groupIDs []string) error {
	if shiftID == "" {
		return fmt.Errorf("shiftId is required")
	}

	if err := s.shiftGroupRepo.BatchSetShiftGroups(ctx, shiftID, groupIDs); err != nil {
		s.logger.Error("Failed to set shift groups", "error", err)
		return fmt.Errorf("set shift groups: %w", err)
	}

	s.logger.Info("Shift groups set successfully", "shiftId", shiftID, "groupCount", len(groupIDs))
	return nil
}

// GetShiftGroups 获取班次关联的所有分组
func (s *ShiftServiceImpl) GetShiftGroups(ctx context.Context, shiftID string) ([]*model.ShiftGroup, error) {
	if shiftID == "" {
		return nil, fmt.Errorf("shiftId is required")
	}

	shiftGroups, err := s.shiftGroupRepo.GetShiftGroups(ctx, shiftID)
	if err != nil {
		s.logger.Error("Failed to get shift groups", "error", err)
		return nil, fmt.Errorf("get shift groups: %w", err)
	}

	return shiftGroups, nil
}

// GetGroupShifts 获取分组关联的所有班次
func (s *ShiftServiceImpl) GetGroupShifts(ctx context.Context, groupID string) ([]*model.ShiftGroup, error) {
	if groupID == "" {
		return nil, fmt.Errorf("groupId is required")
	}

	groupShifts, err := s.shiftGroupRepo.GetGroupShifts(ctx, groupID)
	if err != nil {
		s.logger.Error("Failed to get group shifts", "error", err)
		return nil, fmt.Errorf("get group shifts: %w", err)
	}

	return groupShifts, nil
}

// ToggleShiftStatus 切换班次启用/禁用状态
func (s *ShiftServiceImpl) ToggleShiftStatus(ctx context.Context, orgID, shiftID string, isActive bool) error {
	if orgID == "" {
		return fmt.Errorf("orgId is required")
	}
	if shiftID == "" {
		return fmt.Errorf("shiftId is required")
	}

	// 检查班次是否存在
	shift, err := s.shiftRepo.GetByID(ctx, orgID, shiftID)
	if err != nil {
		s.logger.Error("Failed to get shift", "error", err)
		return fmt.Errorf("get shift: %w", err)
	}
	if shift == nil {
		return fmt.Errorf("shift not found")
	}

	// 更新状态
	shift.IsActive = isActive
	shift.UpdatedAt = time.Now()

	if err := s.shiftRepo.Update(ctx, shift); err != nil {
		s.logger.Error("Failed to update shift status", "error", err)
		return fmt.Errorf("update shift status: %w", err)
	}

	s.logger.Info("Shift status toggled successfully", "shiftId", shiftID, "isActive", isActive)
	return nil
}

// GetShiftGroupMembers 获取班次关联的所有分组的成员（去重）
func (s *ShiftServiceImpl) GetShiftGroupMembers(ctx context.Context, shiftID string) ([]*model.Employee, error) {
	if shiftID == "" {
		return nil, fmt.Errorf("shiftId is required")
	}

	// 直接调用仓储层方法
	members, err := s.shiftGroupRepo.GetShiftGroupMembers(ctx, shiftID)
	if err != nil {
		s.logger.Error("Failed to get shift group members", "shiftId", shiftID, "error", err)
		return nil, fmt.Errorf("get shift group members: %w", err)
	}

	return members, nil
}
