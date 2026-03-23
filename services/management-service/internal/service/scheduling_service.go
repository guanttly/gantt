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

// SchedulingServiceImpl 排班服务实现
type SchedulingServiceImpl struct {
	schedulingRepo repository.ISchedulingRepository
	shiftRepo      repository.IShiftRepository
	employeeRepo   repository.IEmployeeRepository
	logger         logging.ILogger
}

// NewSchedulingService 创建排班服务
func NewSchedulingService(
	schedulingRepo repository.ISchedulingRepository,
	shiftRepo repository.IShiftRepository,
	employeeRepo repository.IEmployeeRepository,
	logger logging.ILogger,
) domain_service.ISchedulingService {
	return &SchedulingServiceImpl{
		schedulingRepo: schedulingRepo,
		shiftRepo:      shiftRepo,
		employeeRepo:   employeeRepo,
		logger:         logger.With("service", "SchedulingService"),
	}
}

// BatchAssignShifts 批量分配班次
func (s *SchedulingServiceImpl) BatchAssignShifts(ctx context.Context, assignments []*model.ShiftAssignment) error {
	if len(assignments) == 0 {
		return fmt.Errorf("assignments cannot be empty")
	}

	// 验证并生成ID
	for _, assignment := range assignments {
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

		// 检查员工是否存在
		exists, err := s.employeeRepo.Exists(ctx, assignment.OrgID, assignment.EmployeeID)
		if err != nil {
			return fmt.Errorf("check employee existence: %w", err)
		}
		if !exists {
			return fmt.Errorf("employee %s not found", assignment.EmployeeID)
		}

		// 检查班次是否存在
		shiftExists, err := s.shiftRepo.Exists(ctx, assignment.OrgID, assignment.ShiftID)
		if err != nil {
			return fmt.Errorf("check shift existence: %w", err)
		}
		if !shiftExists {
			return fmt.Errorf("shift %s not found", assignment.ShiftID)
		}

		// 不再检查冲突，允许同一员工同一天有多个排班
	}

	// 批量创建排班
	if err := s.schedulingRepo.BatchCreateAssignments(ctx, assignments); err != nil {
		s.logger.Error("Failed to batch create assignments", "error", err)
		return fmt.Errorf("batch create assignments: %w", err)
	}

	s.logger.Info("Batch assignments created successfully", "count", len(assignments))
	return nil
}

// GetScheduleByDateRange 获取日期范围内的排班数据
func (s *SchedulingServiceImpl) GetScheduleByDateRange(ctx context.Context, orgID string, startDate, endDate time.Time) ([]*model.ShiftAssignment, error) {
	if orgID == "" {
		return nil, fmt.Errorf("orgId is required")
	}
	if startDate.IsZero() || endDate.IsZero() {
		return nil, fmt.Errorf("startDate and endDate are required")
	}
	if startDate.After(endDate) {
		return nil, fmt.Errorf("startDate must be before or equal to endDate")
	}

	// 限制查询范围不超过3个月
	maxDuration := 90 * 24 * time.Hour
	if endDate.Sub(startDate) > maxDuration {
		return nil, fmt.Errorf("date range cannot exceed 90 days")
	}

	assignments, err := s.schedulingRepo.ListAssignmentsByDateRange(ctx, orgID, startDate, endDate)
	if err != nil {
		s.logger.Error("Failed to get schedule by date range", "error", err)
		return nil, fmt.Errorf("get schedule by date range: %w", err)
	}

	return assignments, nil
}

// GetEmployeeSchedule 获取员工的排班数据
func (s *SchedulingServiceImpl) GetEmployeeSchedule(ctx context.Context, orgID, employeeID string, startDate, endDate time.Time) ([]*model.ShiftAssignment, error) {
	if orgID == "" {
		return nil, fmt.Errorf("orgId is required")
	}
	if employeeID == "" {
		return nil, fmt.Errorf("employeeId is required")
	}
	if startDate.IsZero() || endDate.IsZero() {
		return nil, fmt.Errorf("startDate and endDate are required")
	}
	if startDate.After(endDate) {
		return nil, fmt.Errorf("startDate must be before or equal to endDate")
	}

	assignments, err := s.schedulingRepo.ListAssignmentsByEmployee(ctx, orgID, employeeID, startDate, endDate)
	if err != nil {
		s.logger.Error("Failed to get employee schedule", "employeeId", employeeID, "error", err)
		return nil, fmt.Errorf("get employee schedule: %w", err)
	}

	return assignments, nil
}

// UpdateScheduleAssignment 更新排班分配
func (s *SchedulingServiceImpl) UpdateScheduleAssignment(ctx context.Context, assignment *model.ShiftAssignment) error {
	if assignment.ID == "" {
		return fmt.Errorf("assignment id is required")
	}
	if assignment.OrgID == "" {
		return fmt.Errorf("orgId is required")
	}

	// 检查排班是否存在
	existing, err := s.schedulingRepo.GetAssignmentByID(ctx, assignment.OrgID, assignment.ID)
	if err != nil {
		return fmt.Errorf("get assignment: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("assignment not found")
	}

	// 更新排班
	if err := s.schedulingRepo.UpdateAssignment(ctx, assignment); err != nil {
		s.logger.Error("Failed to update assignment", "assignmentId", assignment.ID, "error", err)
		return fmt.Errorf("update assignment: %w", err)
	}

	s.logger.Info("Assignment updated successfully", "assignmentId", assignment.ID)
	return nil
}

// DeleteScheduleAssignmentByID 通过ID删除排班分配
func (s *SchedulingServiceImpl) DeleteScheduleAssignmentByID(ctx context.Context, orgID, assignmentID string) error {
	if orgID == "" {
		return fmt.Errorf("orgId is required")
	}
	if assignmentID == "" {
		return fmt.Errorf("assignmentId is required")
	}

	// 删除指定ID的排班分配
	if err := s.schedulingRepo.DeleteAssignment(ctx, orgID, assignmentID); err != nil {
		s.logger.Error("Failed to delete schedule assignment by ID",
			"assignmentId", assignmentID,
			"error", err)
		return fmt.Errorf("delete schedule assignment: %w", err)
	}

	s.logger.Info("Schedule assignment deleted successfully", "assignmentId", assignmentID)
	return nil
}

// DeleteScheduleAssignment 删除排班分配（删除员工在指定日期的所有排班）
func (s *SchedulingServiceImpl) DeleteScheduleAssignment(ctx context.Context, orgID, employeeID string, date time.Time) error {
	if orgID == "" {
		return fmt.Errorf("orgId is required")
	}
	if employeeID == "" {
		return fmt.Errorf("employeeId is required")
	}
	if date.IsZero() {
		return fmt.Errorf("date is required")
	}

	// 删除指定员工在指定日期的班次分配
	if err := s.schedulingRepo.DeleteByEmployeeAndDate(ctx, orgID, employeeID, date); err != nil {
		s.logger.Error("Failed to delete schedule assignment",
			"employeeId", employeeID,
			"date", date,
			"error", err)
		return fmt.Errorf("delete schedule assignment: %w", err)
	}

	s.logger.Info("Schedule assignment deleted successfully",
		"employeeId", employeeID,
		"date", date)
	return nil
}

// BatchDeleteScheduleAssignments 批量删除排班分配
func (s *SchedulingServiceImpl) BatchDeleteScheduleAssignments(ctx context.Context, orgID string, employeeIDs []string, dates []time.Time) error {
	if orgID == "" {
		return fmt.Errorf("orgId is required")
	}
	if len(employeeIDs) == 0 || len(dates) == 0 {
		return fmt.Errorf("employeeIds and dates cannot be empty")
	}

	var successCount, failCount int
	for _, employeeID := range employeeIDs {
		for _, date := range dates {
			if err := s.DeleteScheduleAssignment(ctx, orgID, employeeID, date); err != nil {
				s.logger.Warn("Failed to delete assignment",
					"employeeId", employeeID,
					"date", date,
					"error", err)
				failCount++
			} else {
				successCount++
			}
		}
	}

	s.logger.Info("Batch delete completed",
		"successCount", successCount,
		"failCount", failCount)
	return nil
}

// CheckScheduleConflict 检查排班冲突
func (s *SchedulingServiceImpl) CheckScheduleConflict(ctx context.Context, orgID, employeeID string, date time.Time) (bool, error) {
	if orgID == "" {
		return false, fmt.Errorf("orgId is required")
	}
	if employeeID == "" {
		return false, fmt.Errorf("employeeId is required")
	}
	if date.IsZero() {
		return false, fmt.Errorf("date is required")
	}

	assignment, err := s.schedulingRepo.GetAssignmentByEmployeeAndDate(ctx, orgID, employeeID, date)
	if err != nil {
		// 如果是记录不存在的错误，返回无冲突
		if err.Error() == "record not found" {
			return false, nil
		}
		return false, fmt.Errorf("check schedule conflict: %w", err)
	}

	return assignment != nil, nil
}

// GetScheduleSummary 获取排班汇总
func (s *SchedulingServiceImpl) GetScheduleSummary(ctx context.Context, orgID string, startDate, endDate time.Time) (map[string]interface{}, error) {
	if orgID == "" {
		return nil, fmt.Errorf("orgId is required")
	}
	if startDate.IsZero() || endDate.IsZero() {
		return nil, fmt.Errorf("startDate and endDate are required")
	}

	// 获取日期范围内的所有排班
	assignments, err := s.GetScheduleByDateRange(ctx, orgID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// 统计数据
	summary := map[string]interface{}{
		"totalAssignments": len(assignments),
		"dateRange": map[string]string{
			"start": startDate.Format("2006-01-02"),
			"end":   endDate.Format("2006-01-02"),
		},
	}

	// 按班次统计
	shiftStats := make(map[string]int)
	// 按日期统计
	dateStats := make(map[string]int)
	// 按员工统计
	employeeStats := make(map[string]int)

	for _, assignment := range assignments {
		shiftStats[assignment.ShiftID]++
		dateStr := assignment.Date.Format("2006-01-02")
		dateStats[dateStr]++
		employeeStats[assignment.EmployeeID]++
	}

	summary["byShift"] = shiftStats
	summary["byDate"] = dateStats
	summary["byEmployee"] = employeeStats
	summary["uniqueEmployees"] = len(employeeStats)
	summary["uniqueShifts"] = len(shiftStats)

	return summary, nil
}
