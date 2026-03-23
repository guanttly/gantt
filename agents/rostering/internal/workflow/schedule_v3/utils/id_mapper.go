package utils

import (
	"fmt"
	"regexp"

	d_model "jusha/agent/rostering/domain/model"
)

// BuildShiftIDMappings 构建班次ID映射表
// 返回：forward映射（realID -> shortID）和reverse映射（shortID -> realID）
func BuildShiftIDMappings(shifts []*d_model.Shift) (forward, reverse map[string]string) {
	forward = make(map[string]string)
	reverse = make(map[string]string)

	for i, shift := range shifts {
		if shift == nil || shift.ID == "" {
			continue
		}
		shortID := fmt.Sprintf("shift_%d", i+1)
		forward[shift.ID] = shortID
		reverse[shortID] = shift.ID
	}

	return forward, reverse
}

// BuildRuleIDMappings 构建规则ID映射表
// 返回：forward映射（realID -> shortID）和reverse映射（shortID -> realID）
func BuildRuleIDMappings(rules []*d_model.Rule) (forward, reverse map[string]string) {
	forward = make(map[string]string)
	reverse = make(map[string]string)

	ruleIndex := 1
	for _, rule := range rules {
		if rule == nil || rule.ID == "" || !rule.IsActive {
			continue
		}
		shortID := fmt.Sprintf("rule_%d", ruleIndex)
		forward[rule.ID] = shortID
		reverse[shortID] = rule.ID
		ruleIndex++
	}

	return forward, reverse
}

// BuildStaffIDMappings 构建人员ID映射表
// 返回：forward映射（realID -> shortID）和reverse映射（shortID -> realID）
func BuildStaffIDMappings(staffList []*d_model.Employee) (forward, reverse map[string]string) {
	forward = make(map[string]string)
	reverse = make(map[string]string)

	for i, staff := range staffList {
		if staff == nil || staff.ID == "" {
			continue
		}
		shortID := fmt.Sprintf("staff_%d", i+1)
		forward[staff.ID] = shortID
		reverse[shortID] = staff.ID
	}

	return forward, reverse
}

// ReplaceIDsWithShortIDs 将文本中的真实ID替换为简短ID
// idMappings: forward映射（realID -> shortID）
func ReplaceIDsWithShortIDs(text string, idMappings map[string]string) string {
	if text == "" || len(idMappings) == 0 {
		return text
	}

	result := text

	// UUID格式：8-4-4-4-12 十六进制字符，用连字符分隔
	uuidPattern := regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)

	// 替换所有匹配的UUID
	result = uuidPattern.ReplaceAllStringFunc(result, func(match string) string {
		if shortID, ok := idMappings[match]; ok {
			return shortID
		}
		return match // 如果不在映射表中，保持原样
	})

	return result
}

// ReplaceShortIDsWithRealIDs 将文本中的简短ID替换回真实ID
// idMappings: reverse映射（shortID -> realID）
func ReplaceShortIDsWithRealIDs(text string, idMappings map[string]string) string {
	if text == "" || len(idMappings) == 0 {
		return text
	}

	result := text

	// 简短ID格式：shift_1, shift_2, rule_1, rule_2 等
	// 匹配模式：shift_数字 或 rule_数字
	shortIDPattern := regexp.MustCompile(`(shift|rule)_\d+`)

	// 替换所有匹配的简短ID
	result = shortIDPattern.ReplaceAllStringFunc(result, func(match string) string {
		if realID, ok := idMappings[match]; ok {
			return realID
		}
		return match // 如果不在映射表中，保持原样
	})

	return result
}

// ReplaceIDsInTaskPlan 替换任务计划中的ID
// 用于将LLM返回的任务计划中的简短ID替换回真实ID
func ReplaceIDsInTaskPlan(plan *d_model.ProgressiveTaskPlan, shiftReverseMappings, ruleReverseMappings map[string]string) {
	if plan == nil {
		return
	}

	for _, task := range plan.Tasks {
		if task == nil {
			continue
		}

		// 替换targetShifts中的简短ID
		if len(task.TargetShifts) > 0 {
			realShiftIDs := make([]string, 0, len(task.TargetShifts))
			for _, shiftID := range task.TargetShifts {
				if realID, ok := shiftReverseMappings[shiftID]; ok {
					realShiftIDs = append(realShiftIDs, realID)
				} else {
					// 如果不在映射表中，可能是真实ID，直接保留
					realShiftIDs = append(realShiftIDs, shiftID)
				}
			}
			task.TargetShifts = realShiftIDs
		}

		// 替换ruleIds中的简短ID
		if len(task.RuleIDs) > 0 {
			realRuleIDs := make([]string, 0, len(task.RuleIDs))
			for _, ruleID := range task.RuleIDs {
				if realID, ok := ruleReverseMappings[ruleID]; ok {
					realRuleIDs = append(realRuleIDs, realID)
				} else {
					// 如果不在映射表中，可能是真实ID，直接保留
					realRuleIDs = append(realRuleIDs, ruleID)
				}
			}
			task.RuleIDs = realRuleIDs
		}

		// 替换targetStaff中的简短ID（如果有）
		if len(task.TargetStaff) > 0 {
			// 注意：人员ID通常不需要替换，因为已经转换为姓名
			// 但如果任务计划中包含了人员ID，这里也可以处理
			// 目前先不处理，因为人员ID应该已经转换为姓名
		}

		// 【重要】不替换 Description 中的 shortID！
		// Description 是给用户看的/传给 LLM 的，应保留 shortID（如 shift_1, rule_1）
		// 真实 UUID 只在结构化字段（TargetShifts、RuleIDs）中使用，供系统内部处理
		// 这样可以避免 UUID 泄露给 LLM
	}
}

// ReplaceIDsInToolArguments 替换工具调用参数中的简短ID为真实ID
// 用于在工具执行前，将LLM返回的工具调用参数中的简短ID替换回真实ID
func ReplaceIDsInToolArguments(arguments map[string]any, shiftReverseMappings, ruleReverseMappings map[string]string) {
	if len(arguments) == 0 {
		return
	}

	// 合并所有反向映射表
	allReverseMappings := make(map[string]string)
	for k, v := range shiftReverseMappings {
		allReverseMappings[k] = v
	}
	for k, v := range ruleReverseMappings {
		allReverseMappings[k] = v
	}

	// 处理各种可能的参数类型
	for key, value := range arguments {
		switch v := value.(type) {
		case string:
			// 字符串参数：可能是单个ID
			arguments[key] = ReplaceShortIDsWithRealIDs(v, allReverseMappings)
		case []any:
			// 数组参数：可能是ID列表（如shiftIds, ruleIds等）
			replaced := make([]any, 0, len(v))
			for _, item := range v {
				if str, ok := item.(string); ok {
					// 如果是字符串，尝试替换
					replaced = append(replaced, ReplaceShortIDsWithRealIDs(str, allReverseMappings))
				} else {
					replaced = append(replaced, item)
				}
			}
			arguments[key] = replaced
		case []string:
			// 字符串切片：可能是ID列表
			replaced := make([]string, 0, len(v))
			for _, str := range v {
				replaced = append(replaced, ReplaceShortIDsWithRealIDs(str, allReverseMappings))
			}
			arguments[key] = replaced
		}
	}
}

// BuildStaffNameToIDMapping 构建人员姓名到ID的映射（用于LLM返回中文名时的兜底查找）
func BuildStaffNameToIDMapping(allStaffList []*d_model.Employee) map[string]string {
	nameToID := make(map[string]string)
	for _, staff := range allStaffList {
		if staff != nil && staff.ID != "" && staff.Name != "" {
			nameToID[staff.Name] = staff.ID
		}
	}
	return nameToID
}

// BuildStaffIDToNameMapping 构建人员ID到姓名的映射
func BuildStaffIDToNameMapping(allStaffList []*d_model.Employee) map[string]string {
	idToName := make(map[string]string)
	for _, staff := range allStaffList {
		if staff != nil && staff.ID != "" && staff.Name != "" {
			idToName[staff.ID] = staff.Name
		}
	}
	return idToName
}

// ReplaceStaffIDsWithNames 将文本中的人员ID替换为姓名
func ReplaceStaffIDsWithNames(text string, idToName map[string]string) string {
	if text == "" || len(idToName) == 0 {
		return text
	}

	result := text

	// UUID格式：8-4-4-4-12 十六进制字符，用连字符分隔
	uuidPattern := regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)

	// 替换所有匹配的UUID
	result = uuidPattern.ReplaceAllStringFunc(result, func(match string) string {
		if name, ok := idToName[match]; ok {
			return name
		}
		return match // 如果不在映射表中，保持原样
	})

	return result
}

// ReplaceStaffIDsInList 将人员ID列表替换为姓名列表
func ReplaceStaffIDsInList(staffIDs []string, idToName map[string]string) []string {
	if len(staffIDs) == 0 || len(idToName) == 0 {
		return staffIDs
	}

	names := make([]string, 0, len(staffIDs))
	for _, id := range staffIDs {
		if name, ok := idToName[id]; ok {
			names = append(names, name)
		} else {
			// 如果不在映射表中，保留原ID（可能是其他类型的ID）
			names = append(names, id)
		}
	}
	return names
}
