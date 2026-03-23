package common

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"jusha/mcp/pkg/logging"
	"jusha/mcp/pkg/workflow/engine"

	d_model "jusha/agent/rostering/domain/model"
	d_service "jusha/agent/rostering/domain/service"
)

// ============================================================
// 排班基础信息上下文加载器
// 提供一键加载排班所需的基础信息（人员、规则、班次等）
// ============================================================

// ScheduleBasicContext 排班基础信息上下文
// 包含排班工作流所需的所有基础数据，不涉及交互
type ScheduleBasicContext struct {
	// 时间范围
	StartDate string `json:"startDate"` // YYYY-MM-DD
	EndDate   string `json:"endDate"`   // YYYY-MM-DD

	// 班次信息
	SelectedShifts []*d_model.Shift `json:"selectedShifts"` // 选定的班次列表

	// 人员信息
	StaffList       []*d_model.Employee               `json:"staffList"`       // 班次关联的人员列表（用于AI排班）
	AllStaffList    []*d_model.Employee               `json:"allStaffList"`    // 所有员工列表（用于信息检索）
	ShiftMembersMap map[string][]*d_model.Employee    `json:"shiftMembersMap"` // 各班次专属人员（shiftID → 成员列表，用于候选人过滤）
	StaffLeaves     map[string][]*d_model.LeaveRecord `json:"staffLeaves"`     // 人员请假信息 (staff_id -> 请假记录列表)

	// 人员配置需求
	StaffRequirements map[string]map[string]int `json:"staffRequirements"` // (shift_id -> date -> 人数)

	// 规则信息
	Rules []*d_model.Rule `json:"rules"` // 所有排班规则（已去重）
}

// LoadScheduleBasicContext 加载排班基础信息上下文
// 一键加载排班所需的所有基础数据，不涉及用户交互
//
// 参数:
//   - ctx: 上下文
//   - wctx: 工作流上下文
//   - orgID: 组织ID
//   - startDate: 开始日期 (YYYY-MM-DD)，如果为空则使用默认值
//   - endDate: 结束日期 (YYYY-MM-DD)，如果为空则使用默认值
//   - shiftIDs: 指定的班次ID列表，如果为空则加载所有激活的班次
//
// 返回:
//   - *ScheduleBasicContext: 加载的基础信息上下文
//   - error: 错误信息
func LoadScheduleBasicContext(
	ctx context.Context,
	wctx engine.Context,
	orgID string,
	startDate, endDate string,
	shiftIDs []string,
) (*ScheduleBasicContext, error) {
	logger := wctx.Logger()

	// 获取 rosteringService
	service, ok := engine.GetService[d_service.IRosteringService](wctx, engine.ServiceKeyRostering)
	if !ok {
		return nil, fmt.Errorf("rosteringService not found")
	}

	basicCtx := &ScheduleBasicContext{
		StaffRequirements: make(map[string]map[string]int),
		StaffLeaves:       make(map[string][]*d_model.LeaveRecord),
	}

	// 1. 处理时间范围
	if startDate == "" || endDate == "" {
		// 使用默认的下周范围
		var err error
		startDate, endDate, err = GetDefaultNextWeekRange()
		if err != nil {
			return nil, fmt.Errorf("failed to get default date range: %w", err)
		}
	}
	basicCtx.StartDate = startDate
	basicCtx.EndDate = endDate

	// 2. 加载班次列表
	selectedShifts, err := loadShifts(ctx, service, orgID, shiftIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to load shifts: %w", err)
	}
	basicCtx.SelectedShifts = selectedShifts
	logger.Info("Loaded shifts", "total", len(selectedShifts))

	// 3. 加载人员列表和请假记录
	// 注意：需要区分班次关联员工（用于AI排班）和所有员工（用于信息检索）
	shiftStaffList, allStaffList, shiftMembersMap, staffLeaves, err := loadStaffAndLeaves(ctx, service, orgID, selectedShifts, startDate, endDate, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to load staff and leaves: %w", err)
	}
	basicCtx.StaffList = shiftStaffList        // 班次关联的员工（用于AI排班）
	basicCtx.AllStaffList = allStaffList       // 所有员工（用于信息检索）
	basicCtx.ShiftMembersMap = shiftMembersMap // 各班次专属人员（用于候选人过滤）
	basicCtx.StaffLeaves = staffLeaves
	logger.Info("Loaded staff", "shiftStaff", len(shiftStaffList), "allStaff", len(allStaffList), "withLeaves", len(staffLeaves))

	// 4. 加载班次人数配置
	staffRequirements, err := loadStaffRequirements(ctx, service, orgID, selectedShifts, startDate, endDate, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to load staff requirements: %w", err)
	}
	basicCtx.StaffRequirements = staffRequirements
	// 汇总各班次总需求数，为 0 的班次可能是配置缺失或 GetWeeklyStaffConfig 报错
	reqSummary := make(map[string]int)
	for sid, dateMap := range staffRequirements {
		total := 0
		for _, cnt := range dateMap {
			total += cnt
		}
		reqSummary[sid] = total
	}
	logger.Info("Loaded staff requirements", "shifts", len(staffRequirements), "totalByShift", reqSummary)

	// 5. 加载规则（已去重）
	rules, err := loadRules(ctx, service, orgID, selectedShifts, shiftStaffList)
	if err != nil {
		return nil, fmt.Errorf("failed to load rules: %w", err)
	}
	basicCtx.Rules = rules
	logger.Info("Loaded rules", "total", len(rules))

	return basicCtx, nil
}

// loadShifts 加载班次列表
func loadShifts(ctx context.Context, service d_service.IRosteringService, orgID string, shiftIDs []string) ([]*d_model.Shift, error) {
	// 如果指定了班次ID，只加载指定的班次
	if len(shiftIDs) > 0 {
		allShifts, err := service.ListShifts(ctx, orgID, "")
		if err != nil {
			return nil, err
		}
		shiftMap := make(map[string]*d_model.Shift)
		for _, shift := range allShifts {
			shiftMap[shift.ID] = shift
		}
		selectedShifts := make([]*d_model.Shift, 0, len(shiftIDs))
		for _, id := range shiftIDs {
			if shift, ok := shiftMap[id]; ok && shift.IsActive {
				selectedShifts = append(selectedShifts, shift)
			}
		}
		return selectedShifts, nil
	}

	// 否则加载所有激活的班次
	shifts, err := service.ListShifts(ctx, orgID, "")
	if err != nil {
		return nil, err
	}
	selectedShifts := make([]*d_model.Shift, 0)
	for _, shift := range shifts {
		if shift.IsActive {
			selectedShifts = append(selectedShifts, shift)
		}
	}
	return selectedShifts, nil
}

// loadStaffAndLeaves 加载人员列表和请假记录
// 返回：班次关联员工列表、所有员工列表、请假记录
func loadStaffAndLeaves(
	ctx context.Context,
	service d_service.IRosteringService,
	orgID string,
	shifts []*d_model.Shift,
	startDate, endDate string,
	logger interface {
		Warn(string, ...any)
		Error(string, ...any)
	},
) ([]*d_model.Employee, []*d_model.Employee, map[string][]*d_model.Employee, map[string][]*d_model.LeaveRecord, error) {
	// 1. 从班次关联的分组中获取人员（用于AI排班）
	// 使用 map 去重，但保持 SQL 返回的顺序（已按 employee_id 排序）
	shiftStaffMap := make(map[string]*d_model.Employee)
	shiftStaffOrder := make([]*d_model.Employee, 0)         // 保持顺序的切片
	shiftMembersMap := make(map[string][]*d_model.Employee) // 各班次专属人员（用于候选人过滤）
	for _, shift := range shifts {
		members, err := service.GetShiftGroupMembers(ctx, shift.ID)
		if err != nil {
			// 记录警告但继续处理其他班次
			continue
		}
		// 记录该班次的专属人员（用于候选人过滤）
		shiftMembersMap[shift.ID] = members
		// GetShiftGroupMembers 已按 employee_id 排序，直接按顺序添加
		for _, m := range members {
			if _, exists := shiftStaffMap[m.ID]; !exists {
				shiftStaffMap[m.ID] = m
				shiftStaffOrder = append(shiftStaffOrder, m)
			}
		}
	}
	shiftStaffList := shiftStaffOrder

	// 2. 获取所有员工（用于信息检索，如固定班次预览）
	allStaffMap := make(map[string]*d_model.Employee)
	allStaffOrder := make([]*d_model.Employee, 0)

	// 先添加班次关联的员工
	for _, staff := range shiftStaffList {
		allStaffMap[staff.ID] = staff
		allStaffOrder = append(allStaffOrder, staff)
	}

	// 补充获取所有员工（确保包含固定排班人员等不在班次分组中的人员）
	staffResult, err := service.ListStaff(ctx, d_model.StaffListFilter{
		OrgID: orgID,
	})
	if err != nil {
		// 记录错误并返回，确保错误被正确处理
		if logger != nil {
			logger.Error("Failed to list all staff", "error", err, "orgID", orgID)
		}
		return nil, nil, nil, nil, fmt.Errorf("failed to list all staff: %w", err)
	}
	if staffResult != nil {
		for _, staff := range staffResult.Items {
			if staff != nil {
				// 如果员工不在已添加列表中，则添加
				if _, exists := allStaffMap[staff.ID]; !exists {
					// 将 SDK 的 Employee 转换为 domain 的 Employee（保留工号字段）
					employee := &d_model.Employee{
						ID:         staff.ID,
						Name:       staff.Name,
						EmployeeID: staff.EmployeeID,
						Groups:     staff.Groups,
					}
					allStaffMap[staff.ID] = employee
					allStaffOrder = append(allStaffOrder, employee)
				}
			}
		}
	}
	// 按工号数值升序排序整个 allStaffOrder，使导出/预览顺序与系统人员列表一致
	sort.Slice(allStaffOrder, func(i, j int) bool {
		eidI := allStaffOrder[i].EmployeeID
		eidJ := allStaffOrder[j].EmployeeID
		numI, errI := strconv.Atoi(eidI)
		numJ, errJ := strconv.Atoi(eidJ)
		if errI == nil && errJ == nil {
			return numI < numJ
		}
		return eidI < eidJ
	})
	allStaffList := allStaffOrder

	// 3. 批量加载请假记录（优化：一次查询获取所有员工的请假记录）
	staffLeaveMap, err := service.BatchGetLeaveRecords(ctx, orgID, startDate, endDate)
	if err != nil {
		// 如果批量查询失败，使用空 map，不影响后续流程
		staffLeaveMap = make(map[string][]*d_model.LeaveRecord)
	} else {
		// 只保留有请假记录的员工（过滤空记录）
		for staffID, leaves := range staffLeaveMap {
			if len(leaves) == 0 {
				delete(staffLeaveMap, staffID)
			}
		}
	}

	return shiftStaffList, allStaffList, shiftMembersMap, staffLeaveMap, nil
}

// loadStaffRequirements 加载班次人数配置
func loadStaffRequirements(
	ctx context.Context,
	service d_service.IRosteringService,
	orgID string,
	shifts []*d_model.Shift,
	startDate, endDate string,
	logger logging.ILogger,
) (map[string]map[string]int, error) {
	requirements := make(map[string]map[string]int)
	weeklyConfigs := make(map[string]map[int]int)

	// 加载周配置
	for _, shift := range shifts {
		weeklyConfig, err := service.GetWeeklyStaffConfig(ctx, orgID, shift.ID)
		if err != nil {
			// GetWeeklyStaffConfig 失败时该班次的 requiredCount 将全为 0，Phase1/2 会跳过它
			logger.Warn("loadStaffRequirements: GetWeeklyStaffConfig failed, shift will be skipped by scheduler",
				"shiftID", shift.ID, "shiftName", shift.Name, "error", err)
			continue
		}
		if weeklyConfig != nil {
			dayConfig := make(map[int]int)
			for _, dc := range weeklyConfig.WeeklyConfig {
				dayConfig[dc.Weekday] = dc.StaffCount
			}
			weeklyConfigs[shift.ID] = dayConfig
		}
	}

	// 解析日期范围
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start date: %w", err)
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, fmt.Errorf("invalid end date: %w", err)
	}

	// 根据周配置初始化人数需求
	for _, shift := range shifts {
		if _, ok := requirements[shift.ID]; !ok {
			requirements[shift.ID] = make(map[string]int)
		}

		shiftWeeklyConfig := weeklyConfigs[shift.ID]
		for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
			dateStr := d.Format("2006-01-02")
			weekday := int(d.Weekday()) // 0=Sunday, 1=Monday, ..., 6=Saturday

			staffCount := 0
			if shiftWeeklyConfig != nil {
				if count, ok := shiftWeeklyConfig[weekday]; ok {
					staffCount = count
				}
			}

			if staffCount > 0 {
				requirements[shift.ID][dateStr] = staffCount
			}
		}
	}

	// 输出各班次总需求汇总；总需求为 0 的班次在 Phase1/2 中会被完全跳过
	for _, shift := range shifts {
		total := 0
		for _, cnt := range requirements[shift.ID] {
			total += cnt
		}
		if total == 0 {
			logger.Warn("loadStaffRequirements: shift has zero total requirements, will be skipped by scheduler",
				"shiftID", shift.ID, "shiftName", shift.Name,
				"hint", "请检查该班次的周配置（GetWeeklyStaffConfig）是否正确配置了每周排班人数")
		}
	}

	return requirements, nil
}

// loadRules 加载规则（已去重）
func loadRules(
	ctx context.Context,
	service d_service.IRosteringService,
	orgID string,
	shifts []*d_model.Shift,
	staffList []*d_model.Employee,
) ([]*d_model.Rule, error) {
	// 使用 map 去重，避免同一条规则因关联多个对象而重复
	ruleMap := make(map[string]*d_model.Rule)

	// 获取全局规则
	globalRules, err := service.ListRules(ctx, d_model.ListRulesRequest{
		OrgID:      orgID,
		ApplyScope: "global",
		IsActive:   BoolPtr(true),
		Page:       1,
		PageSize:   100,
	})
	if err != nil {
		// 记录警告但继续处理
	} else {
		for _, rule := range globalRules {
			if rule != nil && rule.ID != "" {
				ruleMap[rule.ID] = rule
			}
		}
	}

	// 批量获取班次规则
	if len(shifts) > 0 {
		shiftIDs := make([]string, 0, len(shifts))
		for _, shift := range shifts {
			shiftIDs = append(shiftIDs, shift.ID)
		}
		shiftRulesMap, err := service.GetRulesForShifts(ctx, orgID, shiftIDs)
		if err != nil {
			// 记录警告但继续处理
		} else {
			for _, shiftRules := range shiftRulesMap {
				for _, rule := range shiftRules {
					if rule != nil && rule.ID != "" {
						ruleMap[rule.ID] = rule
					}
				}
			}
		}
	}

	// 批量获取分组规则
	groupIDsMap := make(map[string]bool)
	for _, staff := range staffList {
		if staff.Groups != nil {
			for _, group := range staff.Groups {
				if group.ID != "" {
					groupIDsMap[group.ID] = true
				}
			}
		}
	}
	if len(groupIDsMap) > 0 {
		groupIDs := make([]string, 0, len(groupIDsMap))
		for groupID := range groupIDsMap {
			groupIDs = append(groupIDs, groupID)
		}
		groupRulesMap, err := service.GetRulesForGroups(ctx, orgID, groupIDs)
		if err != nil {
			// 记录警告但继续处理
		} else {
			for _, groupRules := range groupRulesMap {
				for _, rule := range groupRules {
					if rule != nil && rule.ID != "" {
						ruleMap[rule.ID] = rule
					}
				}
			}
		}
	}

	// 批量获取人员规则
	if len(staffList) > 0 {
		employeeIDs := make([]string, 0, len(staffList))
		for _, staff := range staffList {
			employeeIDs = append(employeeIDs, staff.ID)
		}
		employeeRulesMap, err := service.GetRulesForEmployees(ctx, orgID, employeeIDs)
		if err != nil {
			// 记录警告但继续处理
		} else {
			for _, employeeRules := range employeeRulesMap {
				for _, rule := range employeeRules {
					if rule != nil && rule.ID != "" {
						ruleMap[rule.ID] = rule
					}
				}
			}
		}
	}

	// 转换为切片
	allRules := make([]*d_model.Rule, 0, len(ruleMap))
	for _, rule := range ruleMap {
		allRules = append(allRules, rule)
	}

	return allRules, nil
}

// ============================================================
// 辅助函数
// ============================================================
