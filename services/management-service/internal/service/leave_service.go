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

// LeaveServiceImpl 假期管理服务实现
type LeaveServiceImpl struct {
	leaveRepo        repository.ILeaveRepository
	leaveBalanceRepo repository.ILeaveBalanceRepository
	employeeRepo     repository.IEmployeeRepository
	holidayRepo      repository.IHolidayRepository
	logger           logging.ILogger
}

// NewLeaveService 创建假期管理服务
func NewLeaveService(
	leaveRepo repository.ILeaveRepository,
	leaveBalanceRepo repository.ILeaveBalanceRepository,
	employeeRepo repository.IEmployeeRepository,
	holidayRepo repository.IHolidayRepository,
	logger logging.ILogger,
) domain_service.ILeaveService {
	return &LeaveServiceImpl{
		leaveRepo:        leaveRepo,
		leaveBalanceRepo: leaveBalanceRepo,
		employeeRepo:     employeeRepo,
		holidayRepo:      holidayRepo,
		logger:           logger.With("service", "LeaveService"),
	}
}

// CreateLeave 创建假期记录
func (s *LeaveServiceImpl) CreateLeave(ctx context.Context, leave *model.LeaveRecord) error {
	// 验证必填字段
	if leave.OrgID == "" {
		return fmt.Errorf("orgId is required")
	}
	if leave.EmployeeID == "" {
		return fmt.Errorf("employeeId is required")
	}
	if leave.Type == "" {
		return fmt.Errorf("leave_type is required")
	}
	if leave.StartDate.IsZero() {
		return fmt.Errorf("startDate is required")
	}
	if leave.EndDate.IsZero() {
		return fmt.Errorf("endDate is required")
	}

	// 验证日期
	if leave.StartDate.After(leave.EndDate) {
		return fmt.Errorf("startDate must be before or equal to endDate")
	}

	// 生成ID
	if leave.ID == "" {
		leave.ID = uuid.New().String()
	}

	// 计算请假天数
	leave.CalculateDays()

	// 验证员工存在
	empExists, err := s.employeeRepo.Exists(ctx, leave.OrgID, leave.EmployeeID)
	if err != nil {
		return fmt.Errorf("check employee existence: %w", err)
	}
	if !empExists {
		return fmt.Errorf("employee not found: %s", leave.EmployeeID)
	}

	// 检查假期余额
	year := leave.StartDate.Year()
	balance, err := s.leaveBalanceRepo.GetByEmployeeAndType(ctx, leave.OrgID, leave.EmployeeID, leave.Type, year)
	if err == nil && balance != nil {
		if balance.Used+leave.Days > balance.Total {
			return fmt.Errorf("insufficient leave balance: available %.1f days, requested %.1f days",
				balance.Total-balance.Used, leave.Days)
		}
	}

	// 创建假期记录
	if err := s.leaveRepo.Create(ctx, leave); err != nil {
		s.logger.Error("Failed to create leave", "error", err)
		return fmt.Errorf("create leave: %w", err)
	}

	s.logger.Info("Leave created successfully",
		"leave_id", leave.ID,
		"employeeId", leave.EmployeeID,
		"type", leave.Type,
		"days", leave.Days,
	)
	return nil
}

// UpdateLeave 更新假期记录
func (s *LeaveServiceImpl) UpdateLeave(ctx context.Context, leave *model.LeaveRecord) error {
	if leave.ID == "" {
		return fmt.Errorf("leave id is required")
	}
	if leave.OrgID == "" {
		return fmt.Errorf("orgId is required")
	}

	// 检查假期记录是否存在
	_, err := s.leaveRepo.GetByID(ctx, leave.OrgID, leave.ID)
	if err != nil {
		return fmt.Errorf("get leave: %w", err)
	}

	// 重新计算天数
	if !leave.StartDate.IsZero() && !leave.EndDate.IsZero() {
		leave.CalculateDays()
	}

	if err := s.leaveRepo.Update(ctx, leave); err != nil {
		s.logger.Error("Failed to update leave", "error", err)
		return fmt.Errorf("update leave: %w", err)
	}

	s.logger.Info("Leave updated successfully", "leave_id", leave.ID)
	return nil
}

// DeleteLeave 删除假期记录
func (s *LeaveServiceImpl) DeleteLeave(ctx context.Context, orgID, leaveID string) error {
	if orgID == "" || leaveID == "" {
		return fmt.Errorf("orgId and leave_id are required")
	}

	// 检查假期记录是否存在
	_, err := s.leaveRepo.GetByID(ctx, orgID, leaveID)
	if err != nil {
		return fmt.Errorf("get leave: %w", err)
	}

	if err := s.leaveRepo.Delete(ctx, orgID, leaveID); err != nil {
		s.logger.Error("Failed to delete leave", "error", err)
		return fmt.Errorf("delete leave: %w", err)
	}

	s.logger.Info("Leave deleted successfully", "leave_id", leaveID)
	return nil
}

// GetLeave 获取假期详情
func (s *LeaveServiceImpl) GetLeave(ctx context.Context, orgID, leaveID string) (*model.LeaveRecord, error) {
	if orgID == "" || leaveID == "" {
		return nil, fmt.Errorf("orgId and leave_id are required")
	}

	leave, err := s.leaveRepo.GetByID(ctx, orgID, leaveID)
	if err != nil {
		s.logger.Error("Failed to get leave", "error", err)
		return nil, fmt.Errorf("get leave: %w", err)
	}

	return leave, nil
}

// ListLeaves 查询假期记录列表
func (s *LeaveServiceImpl) ListLeaves(ctx context.Context, filter *model.LeaveFilter) (*model.LeaveListResult, error) {
	if filter == nil {
		filter = &model.LeaveFilter{}
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

	result, err := s.leaveRepo.List(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to list leaves", "error", err)
		return nil, fmt.Errorf("list leaves: %w", err)
	}

	// 批量获取员工姓名
	if len(result.Items) > 0 {
		employeeIDs := make([]string, 0, len(result.Items))
		for _, leave := range result.Items {
			employeeIDs = append(employeeIDs, leave.EmployeeID)
		}

		// 批量查询员工信息
		employees, err := s.employeeRepo.BatchGet(ctx, filter.OrgID, employeeIDs)
		if err != nil {
			s.logger.Warn("Failed to batch get employees", "error", err)
			// 即使获取失败，也不影响假期记录的返回
		} else {
			// 创建员工ID到姓名的映射
			employeeMap := make(map[string]string)
			for _, emp := range employees {
				employeeMap[emp.ID] = emp.Name
			}

			// 填充员工姓名
			for _, leave := range result.Items {
				if name, exists := employeeMap[leave.EmployeeID]; exists {
					leave.EmployeeName = name
				}
			}
		}
	}

	return result, nil
}

// GetEmployeeLeaves 获取员工的假期记录
func (s *LeaveServiceImpl) GetEmployeeLeaves(ctx context.Context, orgID, employeeID string, startDate, endDate *time.Time) ([]*model.LeaveRecord, error) {
	if orgID == "" || employeeID == "" {
		return nil, fmt.Errorf("orgId and employeeId are required")
	}

	filter := &model.LeaveFilter{
		OrgID:      orgID,
		EmployeeID: &employeeID,
		StartDate:  startDate,
		EndDate:    endDate,
	}

	result, err := s.leaveRepo.List(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to get employee leaves", "error", err)
		return nil, fmt.Errorf("get employee leaves: %w", err)
	}

	return result.Items, nil
}

// GetLeaveBalance 获取假期余额
func (s *LeaveServiceImpl) GetLeaveBalance(ctx context.Context, orgID, employeeID string, leaveType model.LeaveType, year int) (*model.LeaveBalance, error) {
	if orgID == "" || employeeID == "" {
		return nil, fmt.Errorf("orgId and employeeId are required")
	}
	if year == 0 {
		year = time.Now().Year()
	}

	balance, err := s.leaveBalanceRepo.GetByEmployeeAndType(ctx, orgID, employeeID, leaveType, year)
	if err != nil {
		s.logger.Error("Failed to get leave balance", "error", err)
		return nil, fmt.Errorf("get leave balance: %w", err)
	}

	return balance, nil
}

// InitializeLeaveBalance 初始化假期余额
func (s *LeaveServiceImpl) InitializeLeaveBalance(ctx context.Context, orgID, employeeID string, leaveType model.LeaveType, year int, totalDays float64) error {
	if orgID == "" || employeeID == "" {
		return fmt.Errorf("orgId and employeeId are required")
	}
	if year == 0 {
		year = time.Now().Year()
	}
	if totalDays <= 0 {
		return fmt.Errorf("totalDays must be greater than 0")
	}

	// 验证员工存在
	empExists, err := s.employeeRepo.Exists(ctx, orgID, employeeID)
	if err != nil {
		return fmt.Errorf("check employee existence: %w", err)
	}
	if !empExists {
		return fmt.Errorf("employee not found: %s", employeeID)
	}

	// 检查是否已存在
	existing, err := s.leaveBalanceRepo.GetByEmployeeAndType(ctx, orgID, employeeID, leaveType, year)
	if err == nil && existing != nil {
		return fmt.Errorf("leave balance already exists for employee %s, type %s, year %d", employeeID, leaveType, year)
	}

	// 创建假期余额记录
	balance := &model.LeaveBalance{
		OrgID:      orgID,
		EmployeeID: employeeID,
		Type:       leaveType,
		Year:       year,
		Total:      totalDays,
		Used:       0,
		Remaining:  totalDays,
	}

	if err := s.leaveBalanceRepo.Create(ctx, balance); err != nil {
		s.logger.Error("Failed to initialize leave balance", "error", err)
		return fmt.Errorf("initialize leave balance: %w", err)
	}

	s.logger.Info("Leave balance initialized successfully",
		"employeeId", employeeID,
		"leave_type", leaveType,
		"year", year,
		"totalDays", totalDays,
	)
	return nil
}

// GetEmployeeLeaveBalance 获取员工所有类型的假期余额
func (s *LeaveServiceImpl) GetEmployeeLeaveBalance(ctx context.Context, orgID, employeeID string, year int) ([]*model.LeaveBalance, error) {
	if orgID == "" || employeeID == "" {
		return nil, fmt.Errorf("orgId and employeeId are required")
	}
	if year == 0 {
		year = time.Now().Year()
	}

	balances, err := s.leaveBalanceRepo.ListByEmployee(ctx, orgID, employeeID, year)
	if err != nil {
		s.logger.Error("Failed to get employee leave balance", "error", err)
		return nil, fmt.Errorf("get employee leave balance: %w", err)
	}

	return balances, nil
}

// CalculateLeaveDays 计算实际请假天数（考虑工作日、节假日、小时级请假）
func (s *LeaveServiceImpl) CalculateLeaveDays(ctx context.Context, orgID string, startDate, endDate time.Time, startTime, endTime *string) (float64, error) {
	if startDate.After(endDate) {
		return 0, fmt.Errorf("start date cannot be after end date")
	}

	// 如果提供了具体时间，按小时计算
	if startTime != nil && endTime != nil {
		return s.calculateHourlyLeave(startDate, endDate, *startTime, *endTime)
	}

	// 否则按天计算（排除周末和节假日）
	return s.calculateDailyLeave(ctx, orgID, startDate, endDate)
}

// calculateHourlyLeave 计算小时级请假天数
func (s *LeaveServiceImpl) calculateHourlyLeave(startDate, endDate time.Time, startTime, endTime string) (float64, error) {
	const workHoursPerDay = 8.0 // 每天标准工作时长

	// 解析开始时间
	startHour, startMin, err := parseTime(startTime)
	if err != nil {
		return 0, fmt.Errorf("invalid start time: %w", err)
	}

	// 解析结束时间
	endHour, endMin, err := parseTime(endTime)
	if err != nil {
		return 0, fmt.Errorf("invalid end time: %w", err)
	}

	// 如果是同一天
	if startDate.Format("2006-01-02") == endDate.Format("2006-01-02") {
		startMinutes := startHour*60 + startMin
		endMinutes := endHour*60 + endMin

		if endMinutes <= startMinutes {
			return 0, fmt.Errorf("end time must be after start time")
		}

		hours := float64(endMinutes-startMinutes) / 60.0
		days := hours / workHoursPerDay
		return roundToHalf(days), nil
	}

	// 跨天请假：第一天 + 中间完整天 + 最后一天
	// 第一天：从开始时间到下班时间（假设18:00下班）
	firstDayMinutes := (18 * 60) - (startHour*60 + startMin)
	firstDayHours := float64(firstDayMinutes) / 60.0

	// 最后一天：从上班时间（假设9:00上班）到结束时间
	lastDayMinutes := (endHour*60 + endMin) - (9 * 60)
	lastDayHours := float64(lastDayMinutes) / 60.0

	// 中间完整天数
	middleDays := 0
	current := startDate.AddDate(0, 0, 1)
	for current.Before(endDate) {
		if !isWeekend(current) {
			middleDays++
		}
		current = current.AddDate(0, 0, 1)
	}

	totalHours := firstDayHours + lastDayHours + float64(middleDays)*workHoursPerDay
	days := totalHours / workHoursPerDay

	return roundToHalf(days), nil
}

// calculateDailyLeave 计算按天请假天数（排除周末和节假日）
func (s *LeaveServiceImpl) calculateDailyLeave(ctx context.Context, orgID string, startDate, endDate time.Time) (float64, error) {
	// 获取日期范围内的节假日配置
	holidays, err := s.holidayRepo.ListByDateRange(ctx, orgID, startDate, endDate)
	if err != nil {
		s.logger.Warn("Failed to get holidays, fallback to simple calculation", "error", err)
		// 如果获取节假日失败，降级为简单计算（只排除周末）
		return s.calculateDailyLeaveSimple(startDate, endDate), nil
	}

	// 构建节假日映射表（日期 -> 节假日类型）
	holidayMap := make(map[string]model.HolidayType)
	for _, h := range holidays {
		dateKey := h.Date.Format("2006-01-02")
		// 如果同一天有多个配置，优先使用非空orgID的（组织特定配置）
		if existing, ok := holidayMap[dateKey]; !ok || (existing != h.Type && h.OrgID != "") {
			holidayMap[dateKey] = h.Type
		}
	}

	// 遍历日期范围，计算工作日天数
	days := 0.0
	current := startDate

	for !current.After(endDate) {
		dateKey := current.Format("2006-01-02")

		// 检查是否有节假日配置
		if holidayType, exists := holidayMap[dateKey]; exists {
			if holidayType == model.HolidayTypeWorkday {
				// 调休补班日（周末上班），算工作日
				days += 1.0
			}
			// 如果是 HolidayTypeHoliday 或 HolidayTypeCustom，不计入工作日
		} else {
			// 没有特殊配置，按正常规则判断
			if !isWeekend(current) {
				// 工作日，计入
				days += 1.0
			}
			// 周末不计入
		}

		current = current.AddDate(0, 0, 1)
	}

	s.logger.Debug("Calculated daily leave",
		"orgId", orgID,
		"startDate", startDate.Format("2006-01-02"),
		"endDate", endDate.Format("2006-01-02"),
		"days", days,
	)

	return days, nil
}

// calculateDailyLeaveSimple 简单计算按天请假天数（仅排除周末，不考虑节假日）
// 用于节假日数据不可用时的降级方案
func (s *LeaveServiceImpl) calculateDailyLeaveSimple(startDate, endDate time.Time) float64 {
	days := 0.0
	current := startDate

	for !current.After(endDate) {
		if !isWeekend(current) {
			days += 1.0
		}
		current = current.AddDate(0, 0, 1)
	}

	return days
}

// isWeekend 判断是否为周末
func isWeekend(date time.Time) bool {
	weekday := date.Weekday()
	return weekday == time.Saturday || weekday == time.Sunday
}

// parseTime 解析 HH:MM 格式的时间
func parseTime(timeStr string) (hour, minute int, err error) {
	if len(timeStr) != 5 || timeStr[2] != ':' {
		return 0, 0, fmt.Errorf("time format must be HH:MM")
	}

	_, err = fmt.Sscanf(timeStr, "%d:%d", &hour, &minute)
	if err != nil {
		return 0, 0, err
	}

	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, 0, fmt.Errorf("invalid time values")
	}

	return hour, minute, nil
}

// roundToHalf 四舍五入到最近的0.5
func roundToHalf(value float64) float64 {
	return float64(int(value*2+0.5)) / 2
}
