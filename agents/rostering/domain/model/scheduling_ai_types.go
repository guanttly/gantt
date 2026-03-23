// Package model 提供 AI 排班服务的强类型定义
// 这些类型用于替代 map[string]any，提供类型安全的 AI 接口调用
package model

import "fmt"

// ============================================================
// ShiftInfo - 班次信息（用于 AI 输入）
// 替代 map[string]any 的 shiftInfo 参数
// ============================================================

// ShiftInfo AI 排班所需的班次信息
type ShiftInfo struct {
	ShiftID   string `json:"shiftId"`   // 班次ID
	ShiftName string `json:"shiftName"` // 班次名称
	ShiftCode string `json:"shiftCode"` // 班次代码
	StartTime string `json:"startTime"` // 班次开始时间 HH:MM
	EndTime   string `json:"endTime"`   // 班次结束时间 HH:MM
	Priority  int    `json:"priority"`  // 排班优先级
	StartDate string `json:"startDate"` // 排班周期开始日期 YYYY-MM-DD
	EndDate   string `json:"endDate"`   // 排班周期结束日期 YYYY-MM-DD
}

// NewShiftInfoFromContext 从 ShiftSchedulingContext 构建 ShiftInfo
func NewShiftInfoFromContext(shiftCtx *ShiftSchedulingContext) *ShiftInfo {
	if shiftCtx == nil || shiftCtx.Shift == nil {
		return nil
	}
	return &ShiftInfo{
		ShiftID:   shiftCtx.Shift.ID,
		ShiftName: shiftCtx.Shift.Name,
		ShiftCode: shiftCtx.Shift.Code,
		StartTime: shiftCtx.Shift.StartTime,
		EndTime:   shiftCtx.Shift.EndTime,
		Priority:  shiftCtx.Shift.SchedulingPriority,
		StartDate: shiftCtx.StartDate,
		EndDate:   shiftCtx.EndDate,
	}
}

// ToMap 转换为 map 格式（用于兼容旧接口或 AI 序列化）
func (s *ShiftInfo) ToMap() map[string]any {
	if s == nil {
		return nil
	}
	return map[string]any{
		"shiftId":   s.ShiftID,
		"shiftName": s.ShiftName,
		"shiftCode": s.ShiftCode,
		"startTime": s.StartTime,
		"endTime":   s.EndTime,
		"priority":  s.Priority,
		"startDate": s.StartDate,
		"endDate":   s.EndDate,
	}
}

// ============================================================
// StaffInfoForAI - 员工信息（用于 AI 输入）
// 替代 []map[string]any 的 staffList/availableStaff 参数
// ============================================================

// StaffInfoForAI AI 排班所需的员工信息
type StaffInfoForAI struct {
	ID              string                  `json:"id"`                        // 员工ID
	Name            string                  `json:"name"`                      // 员工姓名
	Groups          []string                `json:"groups,omitempty"`          // 所属分组名称列表
	ScheduledShifts map[string][]*ShiftMark `json:"scheduledShifts,omitempty"` // 已排班次: date -> []ShiftMark
	Skills          []string                `json:"skills,omitempty"`          // 技能标签
	Preferences     map[string]any          `json:"preferences,omitempty"`     // 排班偏好
}

// NewStaffInfoListFromEmployees 从员工列表构建 AI 员工信息列表
func NewStaffInfoListFromEmployees(employees []*Employee) []*StaffInfoForAI {
	if employees == nil {
		return nil
	}
	result := make([]*StaffInfoForAI, 0, len(employees))
	for _, emp := range employees {
		if emp == nil {
			continue
		}
		info := &StaffInfoForAI{
			ID:   emp.ID,
			Name: emp.Name,
		}
		// 提取分组名称
		if len(emp.Groups) > 0 {
			info.Groups = make([]string, 0, len(emp.Groups))
			for _, g := range emp.Groups {
				if g != nil {
					info.Groups = append(info.Groups, g.Name)
				}
			}
		}
		result = append(result, info)
	}
	return result
}

// NewStaffInfoListWithScheduleMarks 从员工列表构建 AI 员工信息列表（包含已排班标记）
func NewStaffInfoListWithScheduleMarks(employees []*Employee, scheduleMarks map[string]map[string][]*ShiftMark) []*StaffInfoForAI {
	if employees == nil {
		return nil
	}
	result := make([]*StaffInfoForAI, 0, len(employees))
	for _, emp := range employees {
		if emp == nil {
			continue
		}
		info := &StaffInfoForAI{
			ID:   emp.ID,
			Name: emp.Name,
		}
		// 提取分组名称
		if len(emp.Groups) > 0 {
			info.Groups = make([]string, 0, len(emp.Groups))
			for _, g := range emp.Groups {
				if g != nil {
					info.Groups = append(info.Groups, g.Name)
				}
			}
		}
		// 添加已排班标记
		if marks, ok := scheduleMarks[emp.ID]; ok {
			info.ScheduledShifts = marks
		}
		result = append(result, info)
	}
	return result
}

// ToMap 转换为 map 格式
func (s *StaffInfoForAI) ToMap() map[string]any {
	if s == nil {
		return nil
	}
	m := map[string]any{
		"id":   s.ID,
		"name": s.Name,
	}
	if len(s.Groups) > 0 {
		m["groups"] = s.Groups
	}
	if len(s.ScheduledShifts) > 0 {
		m["scheduledShifts"] = s.ScheduledShifts
	}
	if len(s.Skills) > 0 {
		m["skills"] = s.Skills
	}
	if len(s.Preferences) > 0 {
		m["preferences"] = s.Preferences
	}
	return m
}

// StaffInfoListToMaps 将员工信息列表转换为 map 列表
func StaffInfoListToMaps(list []*StaffInfoForAI) []map[string]any {
	if list == nil {
		return nil
	}
	result := make([]map[string]any, 0, len(list))
	for _, item := range list {
		if m := item.ToMap(); m != nil {
			result = append(result, m)
		}
	}
	return result
}

// ============================================================
// RuleInfo - 规则信息（用于 AI 输入）
// 替代 []map[string]any 的 rules 参数
// ============================================================

// RuleInfo AI 排班所需的规则信息
type RuleInfo struct {
	ID          string   `json:"id"`          // 规则ID
	Name        string   `json:"name"`        // 规则名称
	Type        string   `json:"type"`        // 规则类型
	Priority    int      `json:"priority"`    // 规则优先级
	Description string   `json:"description"` // 规则描述
	Condition   string   `json:"condition"`   // 规则条件（JSON或表达式）
	Action      string   `json:"action"`      // 规则动作
	IsEnabled   bool     `json:"isEnabled"`   // 是否启用
	Scope       string   `json:"scope"`       // 作用域: global/shift/group/staff
	TargetIDs   []string `json:"targetIds"`   // 目标ID列表（班次ID/分组ID/员工ID）
}

// NewRuleInfoFromRule 从 Rule 构建 RuleInfo
func NewRuleInfoFromRule(rule *Rule) *RuleInfo {
	if rule == nil {
		return nil
	}
	return &RuleInfo{
		ID:          rule.ID,
		Name:        rule.Name,
		Type:        rule.RuleType, // SDK Rule 使用 RuleType 而不是 Type
		Priority:    rule.Priority,
		Description: rule.Description,
		Condition:   rule.RuleData, // SDK Rule 使用 RuleData 而不是 Condition
		Action:      "",            // SDK Rule 没有 Action 字段
		IsEnabled:   rule.IsActive, // SDK Rule 使用 IsActive 而不是 IsEnabled
	}
}

// NewRuleInfoListFromRules 从规则列表构建 RuleInfo 列表
func NewRuleInfoListFromRules(rules []*Rule) []*RuleInfo {
	if rules == nil {
		return nil
	}
	result := make([]*RuleInfo, 0, len(rules))
	for _, rule := range rules {
		if info := NewRuleInfoFromRule(rule); info != nil {
			result = append(result, info)
		}
	}
	return result
}

// CombineRuleInfoLists 合并多个规则列表
func CombineRuleInfoLists(lists ...[]*RuleInfo) []*RuleInfo {
	total := 0
	for _, list := range lists {
		total += len(list)
	}
	result := make([]*RuleInfo, 0, total)
	for _, list := range lists {
		result = append(result, list...)
	}
	return result
}

// ToMap 转换为 map 格式
func (r *RuleInfo) ToMap() map[string]any {
	if r == nil {
		return nil
	}
	return map[string]any{
		"id":          r.ID,
		"name":        r.Name,
		"type":        r.Type,
		"priority":    r.Priority,
		"description": r.Description,
		"condition":   r.Condition,
		"action":      r.Action,
		"isEnabled":   r.IsEnabled,
		"scope":       r.Scope,
		"targetIds":   r.TargetIDs,
	}
}

// RuleInfoListToMaps 将规则信息列表转换为 map 列表
func RuleInfoListToMaps(list []*RuleInfo) []map[string]any {
	if list == nil {
		return nil
	}
	result := make([]map[string]any, 0, len(list))
	for _, item := range list {
		if m := item.ToMap(); m != nil {
			result = append(result, m)
		}
	}
	return result
}

// ============================================================
// TodoPlanResult - Todo 计划生成结果（用于 AI 输出）
// 替代 map[string]any 的返回值
// ============================================================

// TodoPlanResult AI 生成的 Todo 计划结果
type TodoPlanResult struct {
	Todos     []*SchedulingTodo `json:"todos"`     // 生成的 Todo 列表
	Summary   string            `json:"summary"`   // 整体计划说明
	Reasoning string            `json:"reasoning"` // AI 分析推理过程
}

// NewTodoPlanResultFromMap 从 map 解析 TodoPlanResult
func NewTodoPlanResultFromMap(m map[string]any, shift *Shift) *TodoPlanResult {
	if m == nil {
		return nil
	}

	result := &TodoPlanResult{
		Todos: make([]*SchedulingTodo, 0),
	}

	// 解析 todos 列表
	if todos, ok := m["todos"].([]any); ok {
		for i, todoAny := range todos {
			if todoMap, ok := todoAny.(map[string]any); ok {
				todo := parseTodoFromMap(todoMap, i+1)
				if todo != nil {
					result.Todos = append(result.Todos, todo)
				}
			}
		}
	}

	// 解析 summary
	if summary, ok := m["summary"].(string); ok {
		result.Summary = summary
	}

	// 解析 reasoning
	if reasoning, ok := m["reasoning"].(string); ok {
		result.Reasoning = reasoning
	}

	return result
}

// ToShiftTodoPlan 转换为 ShiftTodoPlan
func (r *TodoPlanResult) ToShiftTodoPlan(shiftID, shiftName string) *ShiftTodoPlan {
	if r == nil {
		return nil
	}
	return &ShiftTodoPlan{
		ShiftID:     shiftID,
		ShiftName:   shiftName,
		TodoList:    r.Todos,
		PlanSummary: r.Summary,
	}
}

// parseTodoFromMap 从 map 解析单个 Todo
func parseTodoFromMap(m map[string]any, order int) *SchedulingTodo {
	if m == nil {
		return nil
	}

	todo := &SchedulingTodo{
		Order:  order,
		Status: "pending",
	}

	// 解析 ID
	if id, ok := m["id"].(float64); ok {
		todo.ID = fmt.Sprintf("%d", int(id))
	} else if id, ok := m["id"].(string); ok {
		todo.ID = id
	}

	// 解析 title
	if title, ok := m["title"].(string); ok {
		todo.Title = title
	}

	// 解析 description
	if description, ok := m["description"].(string); ok {
		todo.Description = description
	}

	// 解析 priority
	if priority, ok := m["priority"].(float64); ok {
		todo.Priority = fmt.Sprintf("%d", int(priority))
	} else if priority, ok := m["priority"].(string); ok {
		todo.Priority = priority
	}

	// 解析 targetDates
	if targetDates, ok := m["targetDates"].([]any); ok {
		dates := make([]string, 0)
		for _, d := range targetDates {
			if dateStr, ok := d.(string); ok {
				dates = append(dates, dateStr)
			}
		}
		todo.TargetDates = dates
	}

	// 解析 targetStaffCount
	if targetCount, ok := m["targetStaffCount"].(float64); ok {
		todo.TargetCount = int(targetCount)
	}

	return todo
}

// ============================================================
// TodoExecutionResult - Todo 执行结果（用于 AI 输出）
// 替代 map[string]any 的返回值
// ============================================================

// TodoExecutionResult AI 执行 Todo 任务的结果
type TodoExecutionResult struct {
	Schedule        map[string][]string       `json:"schedule"`        // 日期 -> 员工ID列表
	ScheduleActions map[string]ScheduleAction `json:"scheduleActions"` // 日期 -> 操作类型（append/replace）
	Explanation     string                    `json:"explanation"`     // 执行说明
	Issues          []string                  `json:"issues"`          // 遗留问题
	Success         bool                      `json:"success"`         // 是否成功
}

// ScheduleAction 排班操作类型
type ScheduleAction string

const (
	ScheduleActionAppend  ScheduleAction = "append"  // 增量添加：追加到已有排班
	ScheduleActionReplace ScheduleAction = "replace" // 修正替换：替换已有排班（用于修正错误）
)

// NewTodoExecutionResultFromMap 从 map 解析 TodoExecutionResult
func NewTodoExecutionResultFromMap(m map[string]any) *TodoExecutionResult {
	if m == nil {
		return nil
	}

	result := &TodoExecutionResult{
		Schedule:        make(map[string][]string),
		ScheduleActions: make(map[string]ScheduleAction),
		Issues:          make([]string, 0),
		Success:         true,
	}

	// 解析 schedule
	if schedule, ok := m["schedule"].(map[string]any); ok {
		for date, staffAny := range schedule {
			if staffList, ok := staffAny.([]any); ok {
				staffIDs := make([]string, 0)
				for _, sid := range staffList {
					if sidStr, ok := sid.(string); ok {
						staffIDs = append(staffIDs, sidStr)
					}
				}
				result.Schedule[date] = staffIDs
			}
		}
	}

	// 解析 scheduleActions
	if actions, ok := m["scheduleActions"].(map[string]any); ok {
		for date, actionAny := range actions {
			if actionStr, ok := actionAny.(string); ok {
				result.ScheduleActions[date] = ScheduleAction(actionStr)
			}
		}
	}

	// 如果没有提供 scheduleActions，默认为 append（向后兼容）
	for date := range result.Schedule {
		if _, exists := result.ScheduleActions[date]; !exists {
			result.ScheduleActions[date] = ScheduleActionAppend
		}
	}

	// 解析 explanation
	if explanation, ok := m["explanation"].(string); ok {
		result.Explanation = explanation
	}

	// 解析 issues
	if issues, ok := m["issues"].([]any); ok {
		for _, issue := range issues {
			if issueStr, ok := issue.(string); ok {
				result.Issues = append(result.Issues, issueStr)
			}
		}
	}

	// 解析 success
	if success, ok := m["success"].(bool); ok {
		result.Success = success
	}

	return result
}

// ============================================================
// ValidationResult - 校验结果（用于 AI 输出）
// 替代 map[string]any 的返回值
// ============================================================

// ValidationResult AI 校验排班结果
type ValidationResult struct {
	Passed           bool                `json:"passed"`           // 是否通过校验（AI 返回字段名）
	Issues           []*ValidationIssue  `json:"issues"`           // 发现的问题列表
	AdjustedSchedule map[string][]string `json:"adjustedSchedule"` // 建议的调整: date -> 调整后的员工ID列表
	Summary          string              `json:"summary"`          // 校验总结
}

// NewValidationResultFromMap 从 map 解析 ValidationResult
func NewValidationResultFromMap(m map[string]any) *ValidationResult {
	if m == nil {
		return nil
	}

	result := &ValidationResult{
		AdjustedSchedule: make(map[string][]string),
		Issues:           make([]*ValidationIssue, 0),
		Passed:           true,
	}

	// 解析 passed
	if passed, ok := m["passed"].(bool); ok {
		result.Passed = passed
	}

	// 解析 adjustedSchedule
	if adjustedSchedule, ok := m["adjustedSchedule"].(map[string]any); ok {
		for date, staffAny := range adjustedSchedule {
			if staffList, ok := staffAny.([]any); ok {
				staffIDs := make([]string, 0)
				for _, sid := range staffList {
					if sidStr, ok := sid.(string); ok {
						staffIDs = append(staffIDs, sidStr)
					}
				}
				result.AdjustedSchedule[date] = staffIDs
			}
		}
	}

	// 解析 issues（简化版，只取 description）
	if issues, ok := m["issues"].([]any); ok {
		for _, issue := range issues {
			if issueMap, ok := issue.(map[string]any); ok {
				vi := &ValidationIssue{}
				if t, ok := issueMap["type"].(string); ok {
					vi.Type = t
				}
				if s, ok := issueMap["severity"].(string); ok {
					vi.Severity = s
				}
				if d, ok := issueMap["description"].(string); ok {
					vi.Description = d
				}
				if dates, ok := issueMap["affectedDates"].([]any); ok {
					vi.AffectedDates = make([]string, 0, len(dates))
					for _, d := range dates {
						if ds, ok := d.(string); ok {
							vi.AffectedDates = append(vi.AffectedDates, ds)
						}
					}
				}
				if staff, ok := issueMap["affectedStaff"].([]any); ok {
					vi.AffectedStaff = make([]string, 0, len(staff))
					for _, s := range staff {
						if ss, ok := s.(string); ok {
							vi.AffectedStaff = append(vi.AffectedStaff, ss)
						}
					}
				}
				result.Issues = append(result.Issues, vi)
			}
		}
	}

	// 解析 summary
	if summary, ok := m["summary"].(string); ok {
		result.Summary = summary
	}

	return result
}
