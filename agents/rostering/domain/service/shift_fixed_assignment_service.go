package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"jusha/agent/rostering/domain/model"
	"jusha/agent/rostering/domain/repository"

	"github.com/google/uuid"
)

var (
	ErrFixedAssignmentNotFound = errors.New("固定人员配置不存在")
	ErrFixedAssignmentInvalid  = errors.New("固定人员配置无效")
)

// ShiftFixedAssignmentService 班次固定人员配置服务接口
type ShiftFixedAssignmentService interface {
	// CreateFixedAssignment 创建固定人员配置
	CreateFixedAssignment(ctx context.Context, req *model.CreateShiftFixedAssignmentRequest) (*model.ShiftFixedAssignment, error)

	// BatchCreateFixedAssignments 批量创建固定人员配置（会先删除旧配置）
	BatchCreateFixedAssignments(ctx context.Context, req *model.BatchCreateShiftFixedAssignmentsRequest) error

	// UpdateFixedAssignment 更新固定人员配置
	UpdateFixedAssignment(ctx context.Context, id string, req *model.UpdateShiftFixedAssignmentRequest) (*model.ShiftFixedAssignment, error)

	// DeleteFixedAssignment 删除固定人员配置
	DeleteFixedAssignment(ctx context.Context, id string) error

	// GetFixedAssignment 获取固定人员配置详情
	GetFixedAssignment(ctx context.Context, id string) (*model.ShiftFixedAssignment, error)

	// ListFixedAssignmentsByShift 获取班次的所有固定人员配置
	ListFixedAssignmentsByShift(ctx context.Context, shiftID string) ([]*model.ShiftFixedAssignment, error)

	// ListFixedAssignmentsByStaff 获取人员的所有固定班次配置
	ListFixedAssignmentsByStaff(ctx context.Context, staffID string) ([]*model.ShiftFixedAssignment, error)

	// CalculateFixedSchedule 计算固定班次在指定周期内的实际排班
	CalculateFixedSchedule(ctx context.Context, shiftID string, startDate, endDate string) (map[string][]string, error)

	// CalculateMultipleFixedSchedules 批量计算多个班次的固定排班
	CalculateMultipleFixedSchedules(ctx context.Context, shiftIDs []string, startDate, endDate string) (map[string]map[string][]string, error)
}

// shiftFixedAssignmentServiceImpl 服务实现
type shiftFixedAssignmentServiceImpl struct {
	repo repository.ShiftFixedAssignmentRepository
}

// NewShiftFixedAssignmentService 创建固定人员配置服务实例
func NewShiftFixedAssignmentService(repo repository.ShiftFixedAssignmentRepository) ShiftFixedAssignmentService {
	return &shiftFixedAssignmentServiceImpl{
		repo: repo,
	}
}

// CreateFixedAssignment 创建固定人员配置
func (s *shiftFixedAssignmentServiceImpl) CreateFixedAssignment(ctx context.Context, req *model.CreateShiftFixedAssignmentRequest) (*model.ShiftFixedAssignment, error) {
	// 验证配置
	if err := validateFixedAssignmentRequest(req); err != nil {
		return nil, err
	}

	// 构建模型
	assignment := &model.ShiftFixedAssignment{
		ID:            uuid.New().String(),
		ShiftID:       req.ShiftID,
		StaffID:       req.StaffID,
		PatternType:   req.PatternType,
		Weekdays:      req.Weekdays,
		Monthdays:     req.Monthdays,
		SpecificDates: req.SpecificDates,
		StartDate:     req.StartDate,
		EndDate:       req.EndDate,
		IsActive:      true,
	}

	// 创建到数据库
	if err := s.repo.Create(ctx, assignment); err != nil {
		return nil, fmt.Errorf("创建固定人员配置失败: %w", err)
	}

	return assignment, nil
}

// BatchCreateFixedAssignments 批量创建固定人员配置
func (s *shiftFixedAssignmentServiceImpl) BatchCreateFixedAssignments(ctx context.Context, req *model.BatchCreateShiftFixedAssignmentsRequest) error {
	if req.ShiftID == "" {
		return ErrFixedAssignmentInvalid
	}

	// 先删除该班次的所有旧配置
	if err := s.repo.DeleteByShiftID(ctx, req.ShiftID); err != nil {
		return fmt.Errorf("删除旧配置失败: %w", err)
	}

	// 批量创建新配置
	assignments := make([]*model.ShiftFixedAssignment, 0, len(req.Assignments))
	for _, assignReq := range req.Assignments {
		// 验证配置
		if err := validateFixedAssignmentRequest(&assignReq); err != nil {
			return err
		}

		assignment := &model.ShiftFixedAssignment{
			ID:            uuid.New().String(),
			ShiftID:       req.ShiftID,
			StaffID:       assignReq.StaffID,
			PatternType:   assignReq.PatternType,
			Weekdays:      assignReq.Weekdays,
			SpecificDates: assignReq.SpecificDates,
			StartDate:     assignReq.StartDate,
			EndDate:       assignReq.EndDate,
			IsActive:      true,
		}

		assignments = append(assignments, assignment)
	}

	if len(assignments) > 0 {
		if err := s.repo.BatchCreate(ctx, assignments); err != nil {
			return fmt.Errorf("批量创建固定人员配置失败: %w", err)
		}
	}

	return nil
}

// UpdateFixedAssignment 更新固定人员配置
func (s *shiftFixedAssignmentServiceImpl) UpdateFixedAssignment(ctx context.Context, id string, req *model.UpdateShiftFixedAssignmentRequest) (*model.ShiftFixedAssignment, error) {
	// 获取现有配置
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取固定人员配置失败: %w", err)
	}
	if existing == nil {
		return nil, ErrFixedAssignmentNotFound
	}

	// 更新字段
	existing.PatternType = req.PatternType
	existing.Weekdays = req.Weekdays
	existing.SpecificDates = req.SpecificDates
	existing.StartDate = req.StartDate
	existing.EndDate = req.EndDate
	existing.IsActive = req.IsActive

	// 保存更新
	if err := s.repo.Update(ctx, id, existing); err != nil {
		return nil, fmt.Errorf("更新固定人员配置失败: %w", err)
	}

	return existing, nil
}

// DeleteFixedAssignment 删除固定人员配置
func (s *shiftFixedAssignmentServiceImpl) DeleteFixedAssignment(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("删除固定人员配置失败: %w", err)
	}
	return nil
}

// GetFixedAssignment 获取固定人员配置详情
func (s *shiftFixedAssignmentServiceImpl) GetFixedAssignment(ctx context.Context, id string) (*model.ShiftFixedAssignment, error) {
	assignment, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取固定人员配置失败: %w", err)
	}
	if assignment == nil {
		return nil, ErrFixedAssignmentNotFound
	}
	return assignment, nil
}

// ListFixedAssignmentsByShift 获取班次的所有固定人员配置
func (s *shiftFixedAssignmentServiceImpl) ListFixedAssignmentsByShift(ctx context.Context, shiftID string) ([]*model.ShiftFixedAssignment, error) {
	return s.repo.ListByShiftID(ctx, shiftID)
}

// ListFixedAssignmentsByStaff 获取人员的所有固定班次配置
func (s *shiftFixedAssignmentServiceImpl) ListFixedAssignmentsByStaff(ctx context.Context, staffID string) ([]*model.ShiftFixedAssignment, error) {
	return s.repo.ListByStaffID(ctx, staffID)
}

// CalculateFixedSchedule 计算固定班次在指定周期内的实际排班
func (s *shiftFixedAssignmentServiceImpl) CalculateFixedSchedule(ctx context.Context, shiftID string, startDate, endDate string) (map[string][]string, error) {
	// 1. 获取该班次的所有固定人员配置
	assignments, err := s.repo.ListByShiftID(ctx, shiftID)
	if err != nil {
		return nil, fmt.Errorf("获取固定人员配置失败: %w", err)
	}

	// 2. 解析日期范围
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, fmt.Errorf("开始日期格式错误: %w", err)
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, fmt.Errorf("结束日期格式错误: %w", err)
	}

	// 3. 遍历每个配置，根据模式计算实际日期
	result := make(map[string][]string) // map[date][]staffID
	// 使用 map 去重：date -> map[staffID]bool，避免同一员工在同一日期重复添加
	dateStaffMap := make(map[string]map[string]bool)

	for _, assign := range assignments {
		if !assign.IsActive {
			continue
		}

		// 检查配置的生效时间范围
		if !isAssignmentEffective(assign, start, end) {
			continue
		}

		// 根据模式计算日期
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
func (s *shiftFixedAssignmentServiceImpl) CalculateMultipleFixedSchedules(ctx context.Context, shiftIDs []string, startDate, endDate string) (map[string]map[string][]string, error) {
	// 批量获取所有班次的配置
	assignmentsMap, err := s.repo.ListByShiftIDs(ctx, shiftIDs)
	if err != nil {
		return nil, fmt.Errorf("批量获取固定人员配置失败: %w", err)
	}

	// 解析日期范围
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, fmt.Errorf("开始日期格式错误: %w", err)
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, fmt.Errorf("结束日期格式错误: %w", err)
	}

	// 为每个班次计算排班
	result := make(map[string]map[string][]string) // map[shiftID]map[date][]staffID

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

// validateFixedAssignmentRequest 验证固定人员配置请求
func validateFixedAssignmentRequest(req *model.CreateShiftFixedAssignmentRequest) error {
	if req.ShiftID == "" {
		return fmt.Errorf("%w: 班次ID不能为空", ErrFixedAssignmentInvalid)
	}
	if req.StaffID == "" {
		return fmt.Errorf("%w: 人员ID不能为空", ErrFixedAssignmentInvalid)
	}

	switch req.PatternType {
	case model.PatternTypeWeekly:
		if len(req.Weekdays) == 0 {
			return fmt.Errorf("%w: 按周重复模式必须指定星期", ErrFixedAssignmentInvalid)
		}
		// 验证星期值范围 (1-7: 1=周一, 7=周日)
		for _, day := range req.Weekdays {
			if day < 1 || day > 7 {
				return fmt.Errorf("%w: 星期值必须在1-7之间", ErrFixedAssignmentInvalid)
			}
		}
		// 设置周模式默认值
		if req.WeekPattern == "" {
			req.WeekPattern = model.WeekPatternEvery
		}
		// 验证周模式
		if req.WeekPattern != model.WeekPatternEvery &&
			req.WeekPattern != model.WeekPatternOdd &&
			req.WeekPattern != model.WeekPatternEven {
			return fmt.Errorf("%w: 周模式必须是 every, odd 或 even", ErrFixedAssignmentInvalid)
		}
	case model.PatternTypeMonthly:
		if len(req.Monthdays) == 0 {
			return fmt.Errorf("%w: 按月重复模式必须指定日期", ErrFixedAssignmentInvalid)
		}
		// 验证日期值范围 (1-31)
		for _, day := range req.Monthdays {
			if day < 1 || day > 31 {
				return fmt.Errorf("%w: 月内日期必须在1-31之间", ErrFixedAssignmentInvalid)
			}
		}
	case model.PatternTypeSpecific:
		if len(req.SpecificDates) == 0 {
			return fmt.Errorf("%w: 指定日期模式必须提供日期列表", ErrFixedAssignmentInvalid)
		}
		// 验证日期格式
		for _, dateStr := range req.SpecificDates {
			if _, err := time.Parse("2006-01-02", dateStr); err != nil {
				return fmt.Errorf("%w: 日期格式错误: %s", ErrFixedAssignmentInvalid, dateStr)
			}
		}
	default:
		return fmt.Errorf("%w: 不支持的模式类型: %s", ErrFixedAssignmentInvalid, req.PatternType)
	}

	return nil
}

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
