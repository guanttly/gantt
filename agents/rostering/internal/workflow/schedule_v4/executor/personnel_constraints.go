package executor

import (
	"encoding/json"
	"time"

	"jusha/agent/rostering/domain/model"
)

// isForbiddenOnShift 判断指定员工在某天某班次是否被"禁止排班"
// 规则来源：
// 1. PersonalNeeds 中 RequestType = "avoid" 的个人回避诉求
// 2. 规则层 RuleType = "forbidden_day"：
//   - 通过 ApplyScopes 判断该规则是否适用于该员工（支持 all/employee/group/exclude_employee/exclude_group）
//   - 通过 Associations（AssociationType=shift）判断该规则是否关联该班次
//   - 通过 RuleData 中的 forbiddenWeekdays / targetDates / forbiddenMonthDays 判断日期是否命中
//
// ※ 注意：这里只做"硬排除"，不涉及 must 逻辑
func isForbiddenOnShift(
	staffID string,
	shiftID string,
	dateStr string,
	input *SchedulingExecutionInput,
) bool {
	if input == nil {
		return false
	}

	// 1. PersonalNeeds: avoid
	for _, pn := range input.PersonalNeeds {
		if pn == nil {
			continue
		}
		if pn.StaffID != staffID || pn.RequestType != "avoid" {
			continue
		}

		// 班次限定：如果配置了 TargetShiftID，只在目标班次上生效
		if pn.TargetShiftID != "" && pn.TargetShiftID != shiftID {
			continue
		}

		// 日期限定：
		// - 如果 TargetDates 为空，表示整个排班周期内该班的该员工都应避免
		// - 否则仅在列出的日期上生效
		if len(pn.TargetDates) == 0 {
			return true
		}
		for _, d := range pn.TargetDates {
			if d == dateStr {
				return true
			}
		}
	}

	// 2. 规则层：forbidden_day
	for _, rule := range input.Rules {
		if rule == nil || !rule.IsActive || rule.RuleType != "forbidden_day" {
			continue
		}

		// 2a. 检查规则是否适用于该员工（通过 ApplyScopes）
		if !isRuleApplicableToStaff(rule, staffID, input.AllStaff) {
			continue
		}

		// 2b. 检查规则是否关联该班次（通过 Associations）
		if !isRuleTargetShift(rule, shiftID) {
			continue
		}

		// 2c. 检查日期是否命中（通过 RuleData）
		if isDateForbiddenByRule(rule, dateStr) {
			return true
		}
	}

	return false
}

// isRuleApplicableToStaff 判断规则是否适用于指定员工
// 逻辑：
//   - ApplyScope == "global" -> 适用于所有人
//   - ApplyScope == "specific" -> 通过 ApplyScopes 列表判断
//   - 兼容旧模式：如果 ApplyScopes 为空且 Associations 里有 employee 类型，用 Associations 匹配
func isRuleApplicableToStaff(rule *model.Rule, staffID string, allStaff []*model.Staff) bool {
	// 全局规则：适用于所有人
	if rule.ApplyScope == "global" {
		return true
	}

	// V4.1: 通过 ApplyScopes 判断
	if len(rule.ApplyScopes) > 0 {
		staffGroupIDs := getStaffGroupIDs(staffID, allStaff)
		return matchApplyScopes(rule.ApplyScopes, staffID, staffGroupIDs)
	}

	// 兼容旧模式：ApplyScopes 为空时，通过 Associations 中的 employee 关联判断
	for _, assoc := range rule.Associations {
		if assoc.AssociationType == model.AssociationTypeEmployee && assoc.AssociationID == staffID {
			return true
		}
	}

	return false
}

// matchApplyScopes 判断 ApplyScopes 是否匹配指定员工
func matchApplyScopes(scopes []model.RuleApplyScope, staffID string, staffGroupIDs []string) bool {
	for _, scope := range scopes {
		switch scope.ScopeType {
		case model.ScopeTypeAll:
			return true
		case model.ScopeTypeEmployee:
			if scope.ScopeID == staffID {
				return true
			}
		case model.ScopeTypeGroup:
			for _, gid := range staffGroupIDs {
				if scope.ScopeID == gid {
					return true
				}
			}
		case model.ScopeTypeExcludeEmployee:
			if scope.ScopeID == staffID {
				return false
			}
		case model.ScopeTypeExcludeGroup:
			for _, gid := range staffGroupIDs {
				if scope.ScopeID == gid {
					return false
				}
			}
		}
	}

	// 如果只有 exclude 类型且未命中排除，则视为适用
	hasPositiveScope := false
	for _, scope := range scopes {
		if scope.ScopeType == model.ScopeTypeAll ||
			scope.ScopeType == model.ScopeTypeEmployee ||
			scope.ScopeType == model.ScopeTypeGroup {
			hasPositiveScope = true
			break
		}
	}
	if !hasPositiveScope && len(scopes) > 0 {
		// 全部是 exclude 类型且都未命中 -> 该员工不在排除范围 -> 规则适用
		return true
	}

	return false
}

// getStaffGroupIDs 获取员工所属的分组ID列表
func getStaffGroupIDs(staffID string, allStaff []*model.Staff) []string {
	for _, staff := range allStaff {
		if staff.ID == staffID {
			groupIDs := make([]string, 0, len(staff.Groups))
			for _, g := range staff.Groups {
				if g != nil {
					groupIDs = append(groupIDs, g.ID)
				}
			}
			return groupIDs
		}
	}
	return nil
}

// isRuleTargetShift 判断规则是否关联指定班次
func isRuleTargetShift(rule *model.Rule, shiftID string) bool {
	for _, assoc := range rule.Associations {
		if assoc.AssociationType == model.AssociationTypeShift && assoc.AssociationID == shiftID {
			return true
		}
	}
	return false
}

// isDateForbiddenByRule 判断日期是否被规则禁止
// 根据 RuleData 中的 forbiddenWeekdays / targetDates / forbiddenMonthDays 判断
// 如果 RuleData 为空或无法解析出任何日期条件，则视为全周期禁止
func isDateForbiddenByRule(rule *model.Rule, dateStr string) bool {
	if rule.RuleData == "" {
		// RuleData 为空，视为全周期禁止
		return true
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(rule.RuleData), &data); err != nil {
		// 无法解析 JSON，视为全周期禁止（兼容旧的纯文本格式）
		return true
	}

	// 解析日期
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return false
	}

	hasCondition := false

	// 检查 targetDates（精确日期匹配）
	if dates, ok := data["targetDates"]; ok {
		hasCondition = true
		if datesArr, ok := dates.([]interface{}); ok {
			for _, d := range datesArr {
				if s, ok := d.(string); ok && s == dateStr {
					return true
				}
			}
		}
	}

	// 检查 forbiddenWeekdays（星期几匹配, 1=Monday...7=Sunday）
	if weekdays, ok := data["forbiddenWeekdays"]; ok {
		hasCondition = true
		if arr, ok := weekdays.([]interface{}); ok {
			goWeekday := t.Weekday() // Go: 0=Sunday...6=Saturday
			// 转换为 ISO 格式: 1=Monday...7=Sunday（与前端保存的一致）
			isoWeekday := int(goWeekday)
			if isoWeekday == 0 {
				isoWeekday = 7
			}
			for _, d := range arr {
				if f, ok := d.(float64); ok && int(f) == isoWeekday {
					return true
				}
			}
		}
	}

	// 检查 forbiddenMonthDays（每月第几天匹配, 1~31）
	if monthDays, ok := data["forbiddenMonthDays"]; ok {
		hasCondition = true
		if arr, ok := monthDays.([]interface{}); ok {
			day := t.Day()
			for _, d := range arr {
				if f, ok := d.(float64); ok && int(f) == day {
					return true
				}
			}
		}
	}

	// 如果 RuleData 中没有任何日期条件字段，视为全周期禁止
	if !hasCondition {
		return true
	}

	return false
}

// isAlreadyScheduledOnShift 判断指定员工是否已经在同一天同一班次被排班
func isAlreadyScheduledOnShift(
	staffID string,
	shiftID string,
	dateStr string,
	input *SchedulingExecutionInput,
) bool {
	if input == nil || input.CurrentDraft == nil || input.CurrentDraft.Shifts == nil {
		return false
	}
	shiftDraft, ok := input.CurrentDraft.Shifts[shiftID]
	if !ok || shiftDraft == nil || shiftDraft.Days == nil {
		return false
	}
	dayShift, ok := shiftDraft.Days[dateStr]
	if !ok || dayShift == nil {
		return false
	}
	for _, id := range dayShift.StaffIDs {
		if id == staffID {
			return true
		}
	}
	return false
}
