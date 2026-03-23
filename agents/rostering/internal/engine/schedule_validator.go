package engine

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"jusha/agent/rostering/domain/model"
	"jusha/mcp/pkg/logging"
)

// ScheduleValidator 确定性排班校验器（替代 LLM-5）
// 所有检查均为确定性规则计算，不需要 LLM
type ScheduleValidator struct {
	logger logging.ILogger
}

// NewScheduleValidator 创建排班校验器
func NewScheduleValidator(logger logging.ILogger) *ScheduleValidator {
	return &ScheduleValidator{logger: logger}
}

// ValidationResult 校验结果
type ValidationResult struct {
	IsValid        bool              `json:"isValid"`
	Violations     []*ValidationItem `json:"violations"`     // 硬约束违反
	Warnings       []*ValidationItem `json:"warnings"`       // 软约束/偏好未满足
	UncheckedRules []*UncheckedRule  `json:"uncheckedRules"` // 无法确定性校验的规则（需 LLM 辅助）
	Score          float64           `json:"score"`          // 排班质量评分 (0-100)
	Summary        string            `json:"summary"`        // 可读摘要
}

// UncheckedRule 无法确定性校验的规则
type UncheckedRule struct {
	RuleID      string `json:"ruleId"`
	RuleName    string `json:"ruleName"`
	RuleType    string `json:"ruleType"`
	Description string `json:"description"` // 规则原始描述
	Reason      string `json:"reason"`      // 为什么无法确定性校验
}

// ValidationItem 校验项
type ValidationItem struct {
	RuleID      string   `json:"ruleId"`
	RuleName    string   `json:"ruleName"`
	RuleType    string   `json:"ruleType"`
	Category    string   `json:"category"` // constraint/preference
	StaffIDs    []string `json:"staffIds"` // 涉及的人员
	StaffNames  []string `json:"staffNames,omitempty"`
	Date        string   `json:"date"`     // 涉及的日期
	ShiftID     string   `json:"shiftId"`  // 涉及的班次
	Message     string   `json:"message"`  // 违反描述
	Severity    string   `json:"severity"` // error/warning/info
	AutoFixable bool     `json:"autoFixable"`
}

// ValidateFullSchedule 全量校验排班结果（事后校验，遍历所有班次/日期/人员）
// 这是 V4 阶段 8 调用的主入口
func (v *ScheduleValidator) ValidateFullSchedule(
	draft *model.ScheduleDraft,
	allRules []*model.Rule,
	staffList []*model.Staff,
	shiftList []*model.Shift,
	staffRequirements map[string]map[string]int, // shiftID -> date -> count
) (*ValidationResult, error) {
	result := &ValidationResult{
		IsValid:        true,
		Score:          100.0,
		Violations:     make([]*ValidationItem, 0),
		Warnings:       make([]*ValidationItem, 0),
		UncheckedRules: make([]*UncheckedRule, 0),
	}

	if draft == nil || draft.Shifts == nil {
		result.IsValid = false
		result.Score = 0
		result.Summary = "排班结果为空"
		return result, nil
	}

	// 构建辅助映射
	staffNameMap := make(map[string]string)
	for _, s := range staffList {
		staffNameMap[s.ID] = s.Name
	}

	startDate, _ := time.Parse("2006-01-02", draft.StartDate)
	endDate, _ := time.Parse("2006-01-02", draft.EndDate)

	// ============================================================
	// 1. 人数缺口检查（基础检查）
	// ============================================================
	v.checkStaffShortages(result, draft, staffRequirements)

	// ============================================================
	// 2. 逐条规则 × 逐人 × 逐日 交叉校验
	// ============================================================

	// 记录已标记为 unchecked 的规则ID，避免重复
	uncheckedSeen := make(map[string]bool)

	for _, rule := range allRules {
		if !rule.IsActive {
			continue
		}

		// 快速判断：该规则是否可被确定性校验
		if !isRuleTypeDeterministic(rule.RuleType) {
			if !uncheckedSeen[rule.ID] {
				uncheckedSeen[rule.ID] = true
				result.UncheckedRules = append(result.UncheckedRules, &UncheckedRule{
					RuleID:      rule.ID,
					RuleName:    rule.Name,
					RuleType:    rule.RuleType,
					Description: rule.Description,
					Reason:      fmt.Sprintf("规则类型 '%s' 无法通过确定性引擎校验，需要 LLM 辅助判断", rule.RuleType),
				})
				v.logger.Info("Rule requires LLM validation",
					"ruleID", rule.ID,
					"ruleName", rule.Name,
					"ruleType", rule.RuleType,
				)
			}
			continue // 跳过遍历，该规则留给 LLM 校验
		}

		// 找到规则关联的班次
		targetShiftIDs := v.getRuleTargetShifts(rule, draft)

		// 找到规则关联的人员（空表示全员）
		targetStaffIDs := v.getRuleTargetStaff(rule)

		for _, shiftID := range targetShiftIDs {
			shiftDraft := draft.Shifts[shiftID]
			if shiftDraft == nil || shiftDraft.Days == nil {
				continue
			}

			// 遍历日期范围
			for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
				dateStr := d.Format("2006-01-02")
				dayShift := shiftDraft.Days[dateStr]
				if dayShift == nil {
					continue
				}

				// 遍历已排班人员
				for _, staffID := range dayShift.StaffIDs {
					// 如果规则关联了特定人员，只检查这些人
					if len(targetStaffIDs) > 0 && !containsString(targetStaffIDs, staffID) {
						continue
					}

					items := v.checkRuleForStaff(rule, staffID, staffNameMap[staffID], shiftID, d, draft)
					for _, item := range items {
						if item.Severity == "error" {
							result.IsValid = false
							result.Violations = append(result.Violations, item)
							result.Score -= 10.0
						} else {
							result.Warnings = append(result.Warnings, item)
							result.Score -= 2.0
						}
					}
				}
			}
		}
	}

	// ============================================================
	// 3. 负载均衡检查（全局偏好）
	// ============================================================
	v.checkWorkloadBalance(result, draft, staffNameMap, shiftList, startDate, endDate)

	// 注意：同一人同天排多个班次是否合法由 exclusive 规则决定，不做全局检查

	if result.Score < 0 {
		result.Score = 0
	}

	result.Summary = v.buildFullSummary(result)
	return result, nil
}

// Validate 校验排班结果（兼容旧接口，按已匹配规则校验）
func (v *ScheduleValidator) Validate(
	schedule *model.ScheduleDraft,
	rules *MatchedRules,
	allDraft *model.ScheduleDraft,
) (*ValidationResult, error) {
	result := &ValidationResult{IsValid: true, Score: 100.0}

	for _, rule := range rules.ConstraintRules {
		items := v.checkConstraintRule(rule, schedule, allDraft)
		for _, item := range items {
			if item.Severity == "error" {
				result.IsValid = false
				result.Violations = append(result.Violations, item)
				result.Score -= 10.0
			} else {
				result.Warnings = append(result.Warnings, item)
				result.Score -= 2.0
			}
		}
	}

	for _, rule := range rules.PreferenceRules {
		items := v.checkPreferenceRule(rule, schedule, allDraft)
		for _, item := range items {
			result.Warnings = append(result.Warnings, item)
			result.Score -= 1.0
		}
	}

	if result.Score < 0 {
		result.Score = 0
	}

	result.Summary = v.buildSummary(result)
	return result, nil
}

// ============================================================
// 核心：单规则 × 单人员 × 单日期 检查
// ============================================================

// checkRuleForStaff 检查某条规则对某个员工在某天的违反情况
func (v *ScheduleValidator) checkRuleForStaff(
	rule *model.Rule,
	staffID, staffName, shiftID string,
	date time.Time,
	draft *model.ScheduleDraft,
) []*ValidationItem {
	switch rule.RuleType {
	case "maxCount":
		return v.validateMaxCount(rule, staffID, staffName, shiftID, date, draft)
	case "consecutiveMax":
		return v.validateConsecutiveMax(rule, staffID, staffName, shiftID, date, draft)
	case "minRestDays":
		return v.validateMinRestDays(rule, staffID, staffName, shiftID, date, draft)
	case "exclusive":
		return v.validateExclusive(rule, staffID, staffName, shiftID, date, draft)
	case "forbidden_day":
		return v.validateForbiddenDay(rule, staffID, staffName, shiftID, date, draft)
	case "preferred":
		return v.validatePreferred(rule, staffID, staffName, shiftID, date, draft)
	case "combinable":
		return v.validateCombinable(rule, staffID, staffName, shiftID, date, draft)
	default:
		// 非结构化规则类型，确定性引擎无法校验
		// ValidateFullSchedule 在外层已提前收集到 UncheckedRules，不会走到这里
		// 旧接口 Validate() 调用时直接跳过
		return nil
	}
}

// validateMaxCount 校验最大次数约束
func (v *ScheduleValidator) validateMaxCount(
	rule *model.Rule,
	staffID, staffName, shiftID string,
	date time.Time,
	draft *model.ScheduleDraft,
) []*ValidationItem {
	if rule.MaxCount == nil {
		return nil
	}

	maxCount := *rule.MaxCount
	startDate, endDate := getTimeScopeRange(rule.TimeScope, date)
	currentCount := countStaffShiftInDraft(draft, staffID, shiftID, startDate, endDate)

	if currentCount > maxCount {
		return []*ValidationItem{{
			RuleID:     rule.ID,
			RuleName:   rule.Name,
			RuleType:   rule.RuleType,
			Category:   "constraint",
			StaffIDs:   []string{staffID},
			StaffNames: []string{staffName},
			Date:       date.Format("2006-01-02"),
			ShiftID:    shiftID,
			Message:    fmt.Sprintf("%s 在%s内已排%d次，超出上限%d次", staffName, describeScopeRange(rule.TimeScope), currentCount, maxCount),
			Severity:   "error",
		}}
	}
	return nil
}

// validateConsecutiveMax 校验连续天数约束
func (v *ScheduleValidator) validateConsecutiveMax(
	rule *model.Rule,
	staffID, staffName, shiftID string,
	date time.Time,
	draft *model.ScheduleDraft,
) []*ValidationItem {
	if rule.ConsecutiveMax == nil {
		return nil
	}

	maxConsecutive := *rule.ConsecutiveMax

	// 从当前日期往前+往后双向统计连续天数
	consecutive := 1 // 当天算 1

	// 往前回溯
	checkDate := date.AddDate(0, 0, -1)
	for hasStaffOnDate(draft, staffID, shiftID, checkDate) {
		consecutive++
		checkDate = checkDate.AddDate(0, 0, -1)
	}

	// 往后扩展
	checkDate = date.AddDate(0, 0, 1)
	for hasStaffOnDate(draft, staffID, shiftID, checkDate) {
		consecutive++
		checkDate = checkDate.AddDate(0, 0, 1)
	}

	if consecutive > maxConsecutive {
		return []*ValidationItem{{
			RuleID:     rule.ID,
			RuleName:   rule.Name,
			RuleType:   rule.RuleType,
			Category:   "constraint",
			StaffIDs:   []string{staffID},
			StaffNames: []string{staffName},
			Date:       date.Format("2006-01-02"),
			ShiftID:    shiftID,
			Message:    fmt.Sprintf("%s 连续排班%d天（含%s），超出上限%d天", staffName, consecutive, date.Format("01-02"), maxConsecutive),
			Severity:   "error",
		}}
	}
	return nil
}

// validateMinRestDays 校验最少休息天数
func (v *ScheduleValidator) validateMinRestDays(
	rule *model.Rule,
	staffID, staffName, shiftID string,
	date time.Time,
	draft *model.ScheduleDraft,
) []*ValidationItem {
	if rule.MinRestDays == nil {
		return nil
	}

	minRest := *rule.MinRestDays
	// 检查前 minRest 天内关联班次是否有排班
	for _, assoc := range rule.Associations {
		if assoc.AssociationType == model.AssociationTypeShift {
			for i := 1; i <= minRest; i++ {
				checkDate := date.AddDate(0, 0, -i)
				if hasStaffOnDate(draft, staffID, assoc.AssociationID, checkDate) {
					return []*ValidationItem{{
						RuleID:     rule.ID,
						RuleName:   rule.Name,
						RuleType:   rule.RuleType,
						Category:   "constraint",
						StaffIDs:   []string{staffID},
						StaffNames: []string{staffName},
						Date:       date.Format("2006-01-02"),
						ShiftID:    shiftID,
						Message:    fmt.Sprintf("%s 在%s排了%s班，距%s仅%d天，未满足最少休息%d天", staffName, checkDate.Format("01-02"), assoc.AssociationID, date.Format("01-02"), i, minRest),
						Severity:   "error",
					}}
				}
			}
			// 也检查后 minRest 天
			for i := 1; i <= minRest; i++ {
				checkDate := date.AddDate(0, 0, i)
				if hasStaffOnDate(draft, staffID, assoc.AssociationID, checkDate) {
					return []*ValidationItem{{
						RuleID:     rule.ID,
						RuleName:   rule.Name,
						RuleType:   rule.RuleType,
						Category:   "constraint",
						StaffIDs:   []string{staffID},
						StaffNames: []string{staffName},
						Date:       date.Format("2006-01-02"),
						ShiftID:    shiftID,
						Message:    fmt.Sprintf("%s 在%s和%s之间休息不足%d天", staffName, date.Format("01-02"), checkDate.Format("01-02"), minRest),
						Severity:   "error",
					}}
				}
			}
		}
	}
	return nil
}

// validateExclusive 校验排他约束（排了主体班次后，客体班次在偏移天数对应日期不可再排）
// 当 TimeOffsetDays == 0 时，表示同一天互斥；非零时表示跨天约束。
// 注意：外层循环会对规则关联的所有班次（主体+客体）都调用此函数，
// 必须通过 assoc.Role 判断当前 shiftID 的角色，再决定偏移方向，避免误报。
func (v *ScheduleValidator) validateExclusive(
	rule *model.Rule,
	staffID, staffName, shiftID string,
	date time.Time,
	draft *model.ScheduleDraft,
) []*ValidationItem {
	dateStr := date.Format("2006-01-02")

	// 读取偏移天数：正数表示客体在主体之后，负数表示之前
	offsetDays := 0
	if rule.TimeOffsetDays != nil {
		offsetDays = *rule.TimeOffsetDays
	}

	// 判断当前 shiftID 在规则中是主体（subject）还是客体（object）
	// 主体：排了它之后，客体在 +offsetDays 处不能再排
	// 客体：检查 -offsetDays 处是否存在主体
	currentIsSubject := true
	for _, assoc := range rule.Associations {
		if assoc.AssociationType == model.AssociationTypeShift && assoc.AssociationID == shiftID {
			if assoc.Role == "object" || assoc.Role == "target" {
				currentIsSubject = false
			}
			break
		}
	}

	for _, assoc := range rule.Associations {
		if assoc.AssociationType != model.AssociationTypeShift || assoc.AssociationID == shiftID {
			continue
		}
		var targetDate time.Time
		if currentIsSubject {
			// 当前是主体，找 date+offsetDays 处的客体
			targetDate = date.AddDate(0, 0, offsetDays)
		} else {
			// 当前是客体，向反方向找主体：date-offsetDays
			targetDate = date.AddDate(0, 0, -offsetDays)
		}

		if hasStaffOnDate(draft, staffID, assoc.AssociationID, targetDate) {
			var msg string
			if offsetDays == 0 {
				msg = fmt.Sprintf("%s 在%s同时被排到互斥班次（规则: %s）", staffName, dateStr, rule.Name)
			} else if currentIsSubject {
				msg = fmt.Sprintf("%s 在%s排了主体班次，但在%s又被排到禁止的客体班次（规则: %s，偏移%+d天）",
					staffName, dateStr, targetDate.Format("2006-01-02"), rule.Name, offsetDays)
			} else {
				msg = fmt.Sprintf("%s 在%s排了主体班次，但在%s又被排到禁止的客体班次（规则: %s，偏移%+d天）",
					staffName, targetDate.Format("2006-01-02"), dateStr, rule.Name, offsetDays)
			}
			return []*ValidationItem{{
				RuleID:     rule.ID,
				RuleName:   rule.Name,
				RuleType:   rule.RuleType,
				Category:   "constraint",
				StaffIDs:   []string{staffID},
				StaffNames: []string{staffName},
				Date:       dateStr,
				ShiftID:    shiftID,
				Message:    msg,
				Severity:   "error",
			}}
		}
	}
	return nil
}

// validateForbiddenDay 校验禁止日期约束
func (v *ScheduleValidator) validateForbiddenDay(
	rule *model.Rule,
	staffID, staffName, shiftID string,
	date time.Time,
	draft *model.ScheduleDraft,
) []*ValidationItem {
	dateStr := date.Format("2006-01-02")

	// 从 RuleData 解析禁止日期
	forbiddenDates := parseForbiddenDates(rule.RuleData)

	for _, fd := range forbiddenDates {
		if fd == dateStr {
			return []*ValidationItem{{
				RuleID:     rule.ID,
				RuleName:   rule.Name,
				RuleType:   rule.RuleType,
				Category:   "constraint",
				StaffIDs:   []string{staffID},
				StaffNames: []string{staffName},
				Date:       dateStr,
				ShiftID:    shiftID,
				Message:    fmt.Sprintf("%s 在禁止日期%s被排班（规则: %s）", staffName, dateStr, rule.Name),
				Severity:   "error",
			}}
		}
	}

	// 检查星期几禁止
	forbiddenWeekdays := parseForbiddenWeekdays(rule.RuleData)
	weekday := int(date.Weekday())
	for _, fw := range forbiddenWeekdays {
		if fw == weekday {
			return []*ValidationItem{{
				RuleID:     rule.ID,
				RuleName:   rule.Name,
				RuleType:   rule.RuleType,
				Category:   "constraint",
				StaffIDs:   []string{staffID},
				StaffNames: []string{staffName},
				Date:       dateStr,
				ShiftID:    shiftID,
				Message:    fmt.Sprintf("%s 在禁止的星期(%s)被排班", staffName, date.Format("Monday")),
				Severity:   "error",
			}}
		}
	}

	return nil
}

// validatePreferred 校验偏好规则是否被满足
// preferred 规则是软约束：违反不会导致校验失败，但会产生 warning
func (v *ScheduleValidator) validatePreferred(
	rule *model.Rule,
	staffID, staffName, shiftID string,
	date time.Time,
	draft *model.ScheduleDraft,
) []*ValidationItem {
	// 检查规则的 SubCategory / RuleData 确定偏好方向
	subCat := rule.SubCategory
	if subCat == "" {
		subCat = inferPreferenceDirection(rule)
	}

	switch subCat {
	case "prefer":
		// "prefer" 型：该员工被排到了关联班次 → 符合偏好，无需警告
		// 检查的是：该员工关联了偏好但没有被排到偏好班次的情况
		// 但这里我们是在已排班的 (staffID, shiftID, date) 组合上被调用的
		// 所以反过来检查：如果规则说 "某人偏好某班次"，而当前排的不是偏好班次
		// → 只在规则关联了该员工时才检查
		isTargetStaff := false
		preferredShiftID := ""
		for _, assoc := range rule.Associations {
			if assoc.AssociationType == model.AssociationTypeEmployee && assoc.AssociationID == staffID {
				isTargetStaff = true
			}
			if assoc.AssociationType == model.AssociationTypeShift {
				preferredShiftID = assoc.AssociationID
			}
		}
		// 如果该员工是规则关联人员，且当前班次不是偏好班次 → warning
		if isTargetStaff && preferredShiftID != "" && shiftID != preferredShiftID {
			return []*ValidationItem{{
				RuleID:     rule.ID,
				RuleName:   rule.Name,
				RuleType:   rule.RuleType,
				Category:   "preference",
				StaffIDs:   []string{staffID},
				StaffNames: []string{staffName},
				Date:       date.Format("2006-01-02"),
				ShiftID:    shiftID,
				Message:    fmt.Sprintf("%s 偏好排%s班，但被排到了其他班次（规则: %s）", staffName, preferredShiftID, rule.Name),
				Severity:   "warning",
			}}
		}

	case "avoid", "forbid":
		// "avoid" 型偏好：该员工应尽量避免某班次/日期
		isTargetStaff := false
		avoidShiftID := ""
		for _, assoc := range rule.Associations {
			if assoc.AssociationType == model.AssociationTypeEmployee && assoc.AssociationID == staffID {
				isTargetStaff = true
			}
			if assoc.AssociationType == model.AssociationTypeShift {
				avoidShiftID = assoc.AssociationID
			}
		}
		if isTargetStaff && avoidShiftID != "" && shiftID == avoidShiftID {
			return []*ValidationItem{{
				RuleID:     rule.ID,
				RuleName:   rule.Name,
				RuleType:   rule.RuleType,
				Category:   "preference",
				StaffIDs:   []string{staffID},
				StaffNames: []string{staffName},
				Date:       date.Format("2006-01-02"),
				ShiftID:    shiftID,
				Message:    fmt.Sprintf("%s 希望避免排%s班，但仍被排入（规则: %s）", staffName, avoidShiftID, rule.Name),
				Severity:   "warning",
			}}
		}
	}

	return nil
}

// validateCombinable 校验组合偏好规则
// combinable 规则：建议两个班次尽量由同一人排（或尽量在同一天组合）
func (v *ScheduleValidator) validateCombinable(
	rule *model.Rule,
	staffID, staffName, shiftID string,
	date time.Time,
	draft *model.ScheduleDraft,
) []*ValidationItem {
	// 找到规则关联的其他班次
	pairedShiftIDs := make([]string, 0)
	for _, assoc := range rule.Associations {
		if assoc.AssociationType == model.AssociationTypeShift && assoc.AssociationID != shiftID {
			pairedShiftIDs = append(pairedShiftIDs, assoc.AssociationID)
		}
	}

	if len(pairedShiftIDs) == 0 {
		return nil
	}

	// 检查同一天该员工是否也被排到了配对班次
	for _, pairedID := range pairedShiftIDs {
		if !hasStaffOnDate(draft, staffID, pairedID, date) {
			// 建议组合但未组合 → 轻微 warning
			return []*ValidationItem{{
				RuleID:     rule.ID,
				RuleName:   rule.Name,
				RuleType:   rule.RuleType,
				Category:   "preference",
				StaffIDs:   []string{staffID},
				StaffNames: []string{staffName},
				Date:       date.Format("2006-01-02"),
				ShiftID:    shiftID,
				Message:    fmt.Sprintf("%s 在%s排了%s班，建议同时排%s班（规则: %s）", staffName, date.Format("01-02"), shiftID, pairedID, rule.Name),
				Severity:   "info",
			}}
		}
	}

	return nil
}

// inferPreferenceDirection 从规则描述/RuleData 推断偏好方向
func inferPreferenceDirection(rule *model.Rule) string {
	desc := strings.ToLower(rule.Description + " " + rule.RuleData)
	if strings.Contains(desc, "避免") || strings.Contains(desc, "不要") || strings.Contains(desc, "avoid") {
		return "avoid"
	}
	if strings.Contains(desc, "偏好") || strings.Contains(desc, "喜欢") || strings.Contains(desc, "prefer") {
		return "prefer"
	}
	return "prefer" // 默认
}

// ============================================================
// 全局检查
// ============================================================

// checkStaffShortages 检查人数缺口
func (v *ScheduleValidator) checkStaffShortages(
	result *ValidationResult,
	draft *model.ScheduleDraft,
	staffRequirements map[string]map[string]int,
) {
	for shiftID, dateReqs := range staffRequirements {
		shiftDraft := draft.Shifts[shiftID]
		for date, requiredCount := range dateReqs {
			// 如果需求人数为0或负数，跳过检查（不需要排班）
			if requiredCount <= 0 {
				continue
			}

			actualCount := 0
			if shiftDraft != nil && shiftDraft.Days != nil {
				if dayShift := shiftDraft.Days[date]; dayShift != nil {
					// 修复：直接使用 StaffIDs 的长度，而不是 ActualCount
					// 因为 ActualCount 可能在某些情况下没有正确更新
					// StaffIDs 是实际的数据源，更可靠
					if dayShift.StaffIDs != nil {
						actualCount = len(dayShift.StaffIDs)
					} else {
						// 如果 StaffIDs 为 nil，回退到 ActualCount
						actualCount = dayShift.ActualCount
					}
				}
			}
			if actualCount < requiredCount {
				result.Warnings = append(result.Warnings, &ValidationItem{
					RuleType: "staffCount",
					Category: "requirement",
					Date:     date,
					ShiftID:  shiftID,
					Message:  fmt.Sprintf("%s 需要%d人，实际%d人，缺%d人", date, requiredCount, actualCount, requiredCount-actualCount),
					Severity: "warning",
				})
				result.Score -= 3.0
			}
		}
	}
}

// checkWorkloadBalance 检查负载均衡（两个维度）
// 1. 每周总班次数（环比面）
// 2. 加班班次数：属于夜班（IsOvernight）或服务日期为周末的班次
func (v *ScheduleValidator) checkWorkloadBalance(
	result *ValidationResult,
	draft *model.ScheduleDraft,
	staffNameMap map[string]string,
	shiftList []*model.Shift,
	startDate, endDate time.Time,
) {
	// 构建小工具映射：shiftID -> 是否为夜班
	nightShiftIDs := make(map[string]bool)
	for _, s := range shiftList {
		if s.IsOvernight {
			nightShiftIDs[s.ID] = true
		}
	}

	// 按人统计：总班次数、加班班次数
	staffTotalDays := make(map[string]int)    // 总班次
	staffOvertimeDays := make(map[string]int) // 加班班次（夜班 + 周末班）

	for shiftID, shiftDraft := range draft.Shifts {
		if shiftDraft == nil || shiftDraft.Days == nil {
			continue
		}
		isNight := nightShiftIDs[shiftID]
		for dateStr, dayShift := range shiftDraft.Days {
			if dayShift == nil {
				continue
			}
			// 判断是否周末
			isWeekend := false
			if d, err := time.Parse("2006-01-02", dateStr); err == nil {
				w := d.Weekday()
				isWeekend = w == time.Saturday || w == time.Sunday
			}
			isOvertime := isNight || isWeekend

			for _, staffID := range dayShift.StaffIDs {
				staffTotalDays[staffID]++
				if isOvertime {
					staffOvertimeDays[staffID]++
				}
			}
		}
	}

	if len(staffTotalDays) < 2 {
		return
	}

	// ---- 维度1：加班班次均衡（优先检查，只有存在加班班次时才检查）----
	totalOvertime := 0
	for _, d := range staffOvertimeDays {
		totalOvertime += d
	}
	if totalOvertime > 0 && len(staffOvertimeDays) >= 2 {
		// 仅对有加班记录的人员计算平均（负载0不参与均摊）
		avgOvertime := float64(totalOvertime) / float64(len(staffOvertimeDays))

		for staffID, days := range staffOvertimeDays {
			deviation := float64(days) - avgOvertime
			// 同时满足相对偏差>50% 且 绝对偏差>=2次，避免均值小时误报
			if avgOvertime > 0 && deviation > avgOvertime*0.5 && deviation >= 2 {
				result.Warnings = append(result.Warnings, &ValidationItem{
					RuleType:   "workloadOvertime",
					Category:   "preference",
					StaffIDs:   []string{staffID},
					StaffNames: []string{staffNameMap[staffID]},
					Message:    fmt.Sprintf("%s 加班%d次（夜/周末），平均%.1f次，加班偏高", staffNameMap[staffID], days, avgOvertime),
					Severity:   "warning",
				})
				result.Score -= 1.0
			} else if avgOvertime > 0 && deviation < -avgOvertime*0.5 && deviation <= -2 {
				result.Warnings = append(result.Warnings, &ValidationItem{
					RuleType:   "workloadOvertime",
					Category:   "preference",
					StaffIDs:   []string{staffID},
					StaffNames: []string{staffNameMap[staffID]},
					Message:    fmt.Sprintf("%s 加班%d次（夜/周末），平均%.1f次，加班偏低", staffNameMap[staffID], days, avgOvertime),
					Severity:   "info",
				})
			}
		}
	}

	// ---- 维度2：总班次均衡 ----
	totalDays := 0
	for _, d := range staffTotalDays {
		totalDays += d
	}
	avgTotal := float64(totalDays) / float64(len(staffTotalDays))

	for staffID, days := range staffTotalDays {
		deviation := float64(days) - avgTotal
		// 同时满足相对偏差>50% 且 绝对偏差>=3次，避免均值小时误报
		if avgTotal > 0 && deviation > avgTotal*0.5 && deviation >= 3 {
			result.Warnings = append(result.Warnings, &ValidationItem{
				RuleType:   "workloadTotal",
				Category:   "preference",
				StaffIDs:   []string{staffID},
				StaffNames: []string{staffNameMap[staffID]},
				Message:    fmt.Sprintf("%s 总班次%d次，平均%.1f次，负载偏高", staffNameMap[staffID], days, avgTotal),
				Severity:   "warning",
			})
			result.Score -= 1.0
		} else if avgTotal > 0 && deviation < -avgTotal*0.5 && deviation <= -3 && days > 0 {
			result.Warnings = append(result.Warnings, &ValidationItem{
				RuleType:   "workloadTotal",
				Category:   "preference",
				StaffIDs:   []string{staffID},
				StaffNames: []string{staffNameMap[staffID]},
				Message:    fmt.Sprintf("%s 总班次%d次，平均%.1f次，负载偏低", staffNameMap[staffID], days, avgTotal),
				Severity:   "info",
			})
		}
	}
}

// ============================================================
// 兼容旧接口的约束/偏好检查
// ============================================================

// checkConstraintRule 检查约束型规则（按 ClassifiedRule 接口）
func (v *ScheduleValidator) checkConstraintRule(
	rule *ClassifiedRule,
	schedule *model.ScheduleDraft,
	allDraft *model.ScheduleDraft,
) []*ValidationItem {
	if schedule == nil || schedule.Shifts == nil {
		return nil
	}

	draft := allDraft
	if draft == nil {
		draft = schedule
	}

	var items []*ValidationItem
	startDate, _ := time.Parse("2006-01-02", draft.StartDate)
	endDate, _ := time.Parse("2006-01-02", draft.EndDate)

	for shiftID, shiftDraft := range schedule.Shifts {
		if shiftDraft == nil || shiftDraft.Days == nil {
			continue
		}
		for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
			dateStr := d.Format("2006-01-02")
			dayShift := shiftDraft.Days[dateStr]
			if dayShift == nil {
				continue
			}
			for _, staffID := range dayShift.StaffIDs {
				violations := v.checkRuleForStaff(rule.Rule, staffID, "", shiftID, d, draft)
				items = append(items, violations...)
			}
		}
	}
	return items
}

// checkPreferenceRule 检查偏好型规则
func (v *ScheduleValidator) checkPreferenceRule(
	rule *ClassifiedRule,
	schedule *model.ScheduleDraft,
	allDraft *model.ScheduleDraft,
) []*ValidationItem {
	if schedule == nil || schedule.Shifts == nil {
		return nil
	}

	draft := allDraft
	if draft == nil {
		draft = schedule
	}

	var items []*ValidationItem
	startDate, _ := time.Parse("2006-01-02", draft.StartDate)
	endDate, _ := time.Parse("2006-01-02", draft.EndDate)

	for shiftID, shiftDraft := range schedule.Shifts {
		if shiftDraft == nil || shiftDraft.Days == nil {
			continue
		}
		for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
			dateStr := d.Format("2006-01-02")
			dayShift := shiftDraft.Days[dateStr]
			if dayShift == nil {
				continue
			}
			for _, staffID := range dayShift.StaffIDs {
				violations := v.checkRuleForStaff(rule.Rule, staffID, "", shiftID, d, draft)
				items = append(items, violations...)
			}
		}
	}
	return items
}

// ============================================================
// 辅助方法
// ============================================================

// getRuleTargetShifts 获取规则关联的班次 ID 列表
func (v *ScheduleValidator) getRuleTargetShifts(rule *model.Rule, draft *model.ScheduleDraft) []string {
	shiftIDs := make([]string, 0)
	for _, assoc := range rule.Associations {
		if assoc.AssociationType == model.AssociationTypeShift {
			shiftIDs = append(shiftIDs, assoc.AssociationID)
		}
	}
	// 全局规则适用所有班次
	if len(shiftIDs) == 0 && (rule.ApplyScope == "global" || rule.ApplyScope == "") {
		for shiftID := range draft.Shifts {
			shiftIDs = append(shiftIDs, shiftID)
		}
	}
	return shiftIDs
}

// getRuleTargetStaff 获取规则关联的员工 ID 列表
func (v *ScheduleValidator) getRuleTargetStaff(rule *model.Rule) []string {
	staffIDs := make([]string, 0)
	for _, assoc := range rule.Associations {
		// AssociationType 实际值为 "employee"（与 model.AssociationTypeEmployee 一致），
		// 不能用 "staff"，否则永远匹配不到，导致规则被误当成全员适用
		if assoc.AssociationType == model.AssociationTypeEmployee {
			staffIDs = append(staffIDs, assoc.AssociationID)
		}
	}
	return staffIDs // 空表示全员
}

// buildSummary 构建摘要（兼容旧接口）
func (v *ScheduleValidator) buildSummary(result *ValidationResult) string {
	if result.IsValid {
		return fmt.Sprintf("排班校验通过，质量评分：%.1f分", result.Score)
	}
	return fmt.Sprintf("排班校验失败，发现%d个违反项，质量评分：%.1f分", len(result.Violations), result.Score)
}

// isRuleTypeDeterministic 判断规则类型是否可通过确定性引擎校验
func isRuleTypeDeterministic(ruleType string) bool {
	switch ruleType {
	case "maxCount", "consecutiveMax", "minRestDays", "exclusive",
		"forbidden_day", "preferred", "combinable":
		return true
	default:
		return false
	}
}

// buildFullSummary 构建完整摘要
func (v *ScheduleValidator) buildFullSummary(result *ValidationResult) string {
	parts := make([]string, 0)

	if result.IsValid {
		parts = append(parts, "✅ 校验通过")
	} else {
		parts = append(parts, fmt.Sprintf("❌ 发现 %d 个违反项", len(result.Violations)))
	}

	if len(result.Warnings) > 0 {
		parts = append(parts, fmt.Sprintf("⚠️ %d 个警告", len(result.Warnings)))
	}

	if len(result.UncheckedRules) > 0 {
		parts = append(parts, fmt.Sprintf("🔍 %d 条规则需 LLM 辅助校验", len(result.UncheckedRules)))
	}

	parts = append(parts, fmt.Sprintf("质量评分：%.1f/100", result.Score))

	return strings.Join(parts, "，")
}

// ============================================================
// 公共辅助函数（供 ConstraintChecker 和 ScheduleValidator 共用）
// ============================================================

func getTimeScopeRange(timeScope string, date time.Time) (time.Time, time.Time) {
	switch timeScope {
	case "same_day":
		return date, date
	case "same_week":
		weekday := int(date.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start := date.AddDate(0, 0, -(weekday - 1))
		end := start.AddDate(0, 0, 6)
		return start, end
	case "same_month":
		start := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Location())
		end := start.AddDate(0, 1, -1)
		return start, end
	default:
		// 默认：整个排班周期
		return date, date
	}
}

func countStaffShiftInDraft(draft *model.ScheduleDraft, staffID, shiftID string, start, end time.Time) int {
	if draft == nil {
		return 0
	}
	count := 0
	shiftDraft, ok := draft.Shifts[shiftID]
	if !ok {
		return 0
	}
	for dateStr, dayShift := range shiftDraft.Days {
		d, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}
		if (d.Equal(start) || d.After(start)) && (d.Equal(end) || d.Before(end)) {
			for _, sid := range dayShift.StaffIDs {
				if sid == staffID {
					count++
					break
				}
			}
		}
	}
	return count
}

func hasStaffOnDate(draft *model.ScheduleDraft, staffID, shiftID string, date time.Time) bool {
	if draft == nil {
		return false
	}
	dateStr := date.Format("2006-01-02")
	shiftDraft, ok := draft.Shifts[shiftID]
	if !ok {
		return false
	}
	dayShift, ok := shiftDraft.Days[dateStr]
	if !ok {
		return false
	}
	for _, sid := range dayShift.StaffIDs {
		if sid == staffID {
			return true
		}
	}
	return false
}

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func describeScopeRange(timeScope string) string {
	switch timeScope {
	case "same_day":
		return "当天"
	case "same_week":
		return "本周"
	case "same_month":
		return "本月"
	default:
		return "该周期"
	}
}

// parseForbiddenDates 从 RuleData 解析禁止日期列表
func parseForbiddenDates(ruleData string) []string {
	if ruleData == "" {
		return nil
	}

	// 尝试 JSON 格式: {"targetDates": ["2026-01-01", ...]}
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(ruleData), &data); err == nil {
		if dates, ok := data["targetDates"]; ok {
			if datesArr, ok := dates.([]interface{}); ok {
				result := make([]string, 0, len(datesArr))
				for _, d := range datesArr {
					if s, ok := d.(string); ok {
						result = append(result, s)
					}
				}
				return result
			}
		}
	}

	// 尝试从描述中提取 "dates:{...}" 后缀格式
	if idx := strings.Index(ruleData, "dates:"); idx >= 0 {
		jsonPart := ruleData[idx+6:]
		var dateData map[string][]string
		if err := json.Unmarshal([]byte(jsonPart), &dateData); err == nil {
			return dateData["targetDates"]
		}
	}

	return nil
}

// parseForbiddenMonthDays 从 RuleData 解析每月内禁止排班的日期（1~31）
func parseForbiddenMonthDays(ruleData string) []int {
	if ruleData == "" {
		return nil
	}
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(ruleData), &data); err == nil {
		if v, ok := data["forbiddenMonthDays"]; ok {
			if arr, ok := v.([]interface{}); ok {
				result := make([]int, 0, len(arr))
				for _, d := range arr {
					if f, ok := d.(float64); ok {
						result = append(result, int(f))
					}
				}
				return result
			}
		}
	}
	return nil
}

// parseForbiddenWeekdays 从 RuleData 解析禁止的星期几 (0=Sunday ... 6=Saturday)
func parseForbiddenWeekdays(ruleData string) []int {
	if ruleData == "" {
		return nil
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(ruleData), &data); err == nil {
		if weekdays, ok := data["forbiddenWeekdays"]; ok {
			if arr, ok := weekdays.([]interface{}); ok {
				result := make([]int, 0, len(arr))
				for _, d := range arr {
					if f, ok := d.(float64); ok {
						result = append(result, int(f))
					}
				}
				return result
			}
		}
	}

	return nil
}
