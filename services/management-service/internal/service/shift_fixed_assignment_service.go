package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"jusha/gantt/service/management/domain/model"
	"jusha/gantt/service/management/domain/repository"
	domain_service "jusha/gantt/service/management/domain/service"
	"jusha/mcp/pkg/logging"
)

// ShiftFixedAssignmentServiceImpl 班次固定人员配置服务实现
type ShiftFixedAssignmentServiceImpl struct {
	repo   repository.IShiftFixedAssignmentRepository
	logger logging.ILogger
}

// NewShiftFixedAssignmentService 创建班次固定人员配置服务
func NewShiftFixedAssignmentService(
	repo repository.IShiftFixedAssignmentRepository,
	logger logging.ILogger,
) domain_service.IShiftFixedAssignmentService {
	return &ShiftFixedAssignmentServiceImpl{
		repo:   repo,
		logger: logger.With("service", "ShiftFixedAssignmentService"),
	}
}

// Create 创建固定人员配置
func (s *ShiftFixedAssignmentServiceImpl) Create(ctx context.Context, assignment *model.ShiftFixedAssignment) error {
	// 验证必填字段
	if assignment.ShiftID == "" {
		return fmt.Errorf("shiftId is required")
	}
	if assignment.StaffID == "" {
		return fmt.Errorf("staffId is required")
	}
	if assignment.PatternType == "" {
		return fmt.Errorf("patternType is required")
	}

	// 生成ID
	if assignment.ID == "" {
		assignment.ID = uuid.New().String()
	}

	// 设置默认启用状态
	assignment.IsActive = true

	// 创建配置
	if err := s.repo.Create(ctx, assignment); err != nil {
		s.logger.Error("Failed to create fixed assignment", "error", err)
		return fmt.Errorf("create fixed assignment: %w", err)
	}

	s.logger.Info("Fixed assignment created successfully",
		"id", assignment.ID,
		"shiftId", assignment.ShiftID,
		"staffId", assignment.StaffID,
	)
	return nil
}

// BatchCreate 批量创建固定人员配置
func (s *ShiftFixedAssignmentServiceImpl) BatchCreate(ctx context.Context, shiftID string, assignments []*model.ShiftFixedAssignment) error {
	if shiftID == "" {
		return fmt.Errorf("shiftId is required")
	}

	// 先删除该班次的所有旧配置
	if err := s.repo.DeleteByShiftID(ctx, shiftID); err != nil {
		s.logger.Error("Failed to delete old fixed assignments", "error", err, "shiftId", shiftID)
		return fmt.Errorf("delete old fixed assignments: %w", err)
	}

	// 批量创建新配置
	if len(assignments) > 0 {
		// 生成ID并设置 shiftID
		for _, assignment := range assignments {
			if assignment.ID == "" {
				assignment.ID = uuid.New().String()
			}
			assignment.ShiftID = shiftID
			assignment.IsActive = true
		}

		if err := s.repo.BatchCreate(ctx, assignments); err != nil {
			s.logger.Error("Failed to batch create fixed assignments", "error", err)
			return fmt.Errorf("batch create fixed assignments: %w", err)
		}
	}

	s.logger.Info("Fixed assignments batch created successfully",
		"shiftId", shiftID,
		"count", len(assignments),
	)
	return nil
}

// Update 更新固定人员配置
func (s *ShiftFixedAssignmentServiceImpl) Update(ctx context.Context, id string, assignment *model.ShiftFixedAssignment) error {
	if id == "" {
		return fmt.Errorf("id is required")
	}

	// 检查配置是否存在
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get fixed assignment", "error", err, "id", id)
		return fmt.Errorf("get fixed assignment: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("fixed assignment not found: %s", id)
	}

	// 更新配置
	if err := s.repo.Update(ctx, id, assignment); err != nil {
		s.logger.Error("Failed to update fixed assignment", "error", err, "id", id)
		return fmt.Errorf("update fixed assignment: %w", err)
	}

	s.logger.Info("Fixed assignment updated successfully", "id", id)
	return nil
}

// Delete 删除固定人员配置
func (s *ShiftFixedAssignmentServiceImpl) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("id is required")
	}

	// 删除配置
	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.Error("Failed to delete fixed assignment", "error", err, "id", id)
		return fmt.Errorf("delete fixed assignment: %w", err)
	}

	s.logger.Info("Fixed assignment deleted successfully", "id", id)
	return nil
}

// GetByID 获取固定人员配置详情
func (s *ShiftFixedAssignmentServiceImpl) GetByID(ctx context.Context, id string) (*model.ShiftFixedAssignment, error) {
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}

	assignment, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get fixed assignment", "error", err, "id", id)
		return nil, fmt.Errorf("get fixed assignment: %w", err)
	}

	return assignment, nil
}

// ListByShiftID 获取班次的所有固定人员配置
func (s *ShiftFixedAssignmentServiceImpl) ListByShiftID(ctx context.Context, shiftID string) ([]*model.ShiftFixedAssignment, error) {
	if shiftID == "" {
		return nil, fmt.Errorf("shiftId is required")
	}

	assignments, err := s.repo.ListByShiftID(ctx, shiftID)
	if err != nil {
		s.logger.Error("Failed to list fixed assignments by shift", "error", err, "shiftId", shiftID)
		return nil, fmt.Errorf("list fixed assignments by shift: %w", err)
	}

	return assignments, nil
}

// ListByStaffID 获取人员的所有固定班次配置
func (s *ShiftFixedAssignmentServiceImpl) ListByStaffID(ctx context.Context, staffID string) ([]*model.ShiftFixedAssignment, error) {
	if staffID == "" {
		return nil, fmt.Errorf("staffId is required")
	}

	assignments, err := s.repo.ListByStaffID(ctx, staffID)
	if err != nil {
		s.logger.Error("Failed to list fixed assignments by staff", "error", err, "staffId", staffID)
		return nil, fmt.Errorf("list fixed assignments by staff: %w", err)
	}

	return assignments, nil
}

// CalculateFixedSchedule 计算固定班次在指定周期内的实际排班
func (s *ShiftFixedAssignmentServiceImpl) CalculateFixedSchedule(ctx context.Context, shiftID string, startDate, endDate string) (map[string][]string, error) {
	if shiftID == "" {
		return nil, fmt.Errorf("shiftId is required")
	}
	if startDate == "" {
		return nil, fmt.Errorf("startDate is required")
	}
	if endDate == "" {
		return nil, fmt.Errorf("endDate is required")
	}

	// 解析日期范围
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, fmt.Errorf("invalid startDate format: %w", err)
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, fmt.Errorf("invalid endDate format: %w", err)
	}

	// 获取该班次的所有固定人员配置
	assignments, err := s.repo.ListByShiftID(ctx, shiftID)
	if err != nil {
		s.logger.Error("Failed to list fixed assignments", "error", err, "shiftId", shiftID)
		return nil, fmt.Errorf("list fixed assignments: %w", err)
	}

	// 计算排班
	result := make(map[string][]string)
	// 使用 map 去重：date -> map[staffID]bool，避免同一员工在同一日期重复添加
	dateStaffMap := make(map[string]map[string]bool)
	
	for _, assign := range assignments {
		if !assign.IsActive {
			continue
		}

		if !isAssignmentEffective(assign, start, end) {
			continue
		}

		dates := calculateDatesForAssignment(assign, start, end)
		for _, date := range dates {
			dateStr := date.Format("2006-01-02")
			// 初始化该日期的员工去重 map
			if dateStaffMap[dateStr] == nil {
				dateStaffMap[dateStr] = make(map[string]bool)
			}
			// 使用 map 去重：只有当该员工在该日期尚未添加时才添加
			if !dateStaffMap[dateStr][assign.StaffID] {
				dateStaffMap[dateStr][assign.StaffID] = true
				result[dateStr] = append(result[dateStr], assign.StaffID)
			}
		}
	}

	return result, nil
}

// CalculateMultipleFixedSchedules 批量计算多个班次的固定排班
func (s *ShiftFixedAssignmentServiceImpl) CalculateMultipleFixedSchedules(ctx context.Context, shiftIDs []string, startDate, endDate string) (map[string]map[string][]string, error) {
	if len(shiftIDs) == 0 {
		return make(map[string]map[string][]string), nil
	}
	if startDate == "" {
		return nil, fmt.Errorf("startDate is required")
	}
	if endDate == "" {
		return nil, fmt.Errorf("endDate is required")
	}

	// 解析日期范围
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, fmt.Errorf("invalid startDate format: %w", err)
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, fmt.Errorf("invalid endDate format: %w", err)
	}

	// 批量获取所有班次的配置
	assignmentsMap, err := s.repo.ListByShiftIDs(ctx, shiftIDs)
	if err != nil {
		s.logger.Error("Failed to list fixed assignments by shift IDs", "error", err)
		return nil, fmt.Errorf("list fixed assignments: %w", err)
	}

	// 为每个班次计算排班
	result := make(map[string]map[string][]string)
	for shiftID, assignments := range assignmentsMap {
		shiftSchedule := make(map[string][]string)
		// 使用 map 去重：date -> map[staffID]bool，避免同一员工在同一日期重复添加
		dateStaffMap := make(map[string]map[string]bool)

		for _, assign := range assignments {
			if !assign.IsActive {
				continue
			}

			if !isAssignmentEffective(assign, start, end) {
				continue
			}

			dates := calculateDatesForAssignment(assign, start, end)
			for _, date := range dates {
				dateStr := date.Format("2006-01-02")
				// 初始化该日期的员工去重 map
				if dateStaffMap[dateStr] == nil {
					dateStaffMap[dateStr] = make(map[string]bool)
				}
				// 使用 map 去重：只有当该员工在该日期尚未添加时才添加
				if !dateStaffMap[dateStr][assign.StaffID] {
					dateStaffMap[dateStr][assign.StaffID] = true
					shiftSchedule[dateStr] = append(shiftSchedule[dateStr], assign.StaffID)
				}
			}
		}

		if len(shiftSchedule) > 0 {
			result[shiftID] = shiftSchedule
		}
	}

	return result, nil
}

// ============================================================
// 辅助函数
// ============================================================

// isAssignmentEffective 检查配置在指定日期范围内是否生效
func isAssignmentEffective(assign *model.ShiftFixedAssignment, start, end time.Time) bool {
	// 检查开始日期
	if assign.StartDate != nil && assign.StartDate.After(end) {
		return false
	}

	// 检查结束日期
	if assign.EndDate != nil && assign.EndDate.Before(start) {
		return false
	}

	return true
}

// calculateDatesForAssignment 计算配置在指定日期范围内的所有生效日期
func calculateDatesForAssignment(assign *model.ShiftFixedAssignment, start, end time.Time) []time.Time {
	var dates []time.Time

	switch assign.PatternType {
	case model.PatternTypeWeekly:
		// 按周重复模式
		weekdaysSet := make(map[int]bool)
		for _, day := range assign.Weekdays {
			// 前端传入的是 1=周一, 7=周日，转换为Go的 Weekday (0=Sunday, 1=Monday, ..., 6=Saturday)
			goWeekday := day % 7 // 7 -> 0 (Sunday), 1 -> 1 (Monday), ..., 6 -> 6 (Saturday)
			weekdaysSet[goWeekday] = true
		}

		// 遍历日期范围
		for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
			// 检查是否在配置的生效时间范围内
			if assign.StartDate != nil && d.Before(*assign.StartDate) {
				continue
			}
			if assign.EndDate != nil && d.After(*assign.EndDate) {
				continue
			}

			// 检查星期是否匹配
			weekday := int(d.Weekday()) // 0=Sunday, 1=Monday, ..., 6=Saturday
			if !weekdaysSet[weekday] {
				continue
			}

			// 检查周模式（奇数周/偶数周）
			if assign.WeekPattern != "" && assign.WeekPattern != model.WeekPatternEvery {
				_, weekNum := d.ISOWeek() // 获取 ISO 周编号（1-53）
				isOddWeek := weekNum%2 == 1

				if assign.WeekPattern == model.WeekPatternOdd && !isOddWeek {
					continue // 配置为奇数周，但当前是偶数周
				}
				if assign.WeekPattern == model.WeekPatternEven && isOddWeek {
					continue // 配置为偶数周，但当前是奇数周
				}
			}

			dates = append(dates, d)
		}

	case model.PatternTypeMonthly:
		// 按月重复模式
		monthdaysSet := make(map[int]bool)
		for _, day := range assign.Monthdays {
			monthdaysSet[day] = true
		}

		// 遍历日期范围
		for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
			// 检查是否在配置的生效时间范围内
			if assign.StartDate != nil && d.Before(*assign.StartDate) {
				continue
			}
			if assign.EndDate != nil && d.After(*assign.EndDate) {
				continue
			}

			// 检查月内日期是否匹配
			monthday := d.Day() // 1-31
			if monthdaysSet[monthday] {
				dates = append(dates, d)
			}
		}

	case model.PatternTypeSpecific:
		// 指定日期模式
		specificDatesSet := make(map[string]bool)
		for _, dateStr := range assign.SpecificDates {
			specificDatesSet[dateStr] = true
		}

		// 遍历日期范围
		for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
			dateStr := d.Format("2006-01-02")
			if specificDatesSet[dateStr] {
				dates = append(dates, d)
			}
		}
	}

	return dates
}

