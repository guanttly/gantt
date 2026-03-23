package engine

import (
	"jusha/agent/rostering/domain/model"
	"jusha/mcp/pkg/logging"
	"time"
)

// RuleMatcher 规则匹配器（替代 LLM-2）
type RuleMatcher struct {
	logger logging.ILogger
}

// NewRuleMatcher 创建规则匹配器
func NewRuleMatcher(logger logging.ILogger) *RuleMatcher {
	return &RuleMatcher{logger: logger}
}

// MatchRules 匹配规则
func (m *RuleMatcher) MatchRules(
	allRules []*model.Rule,
	shiftID string,
	date time.Time,
) *MatchedRules {
	result := &MatchedRules{
		ConstraintRules: make([]*ClassifiedRule, 0),
		PreferenceRules: make([]*ClassifiedRule, 0),
		DependencyRules: make([]*ClassifiedRule, 0),
	}

	for _, rule := range allRules {
		// 检查规则是否在指定日期有效
		if !m.isRuleValid(rule, date) {
			continue
		}

		// 检查规则是否适用于当前班次
		if !m.isRuleApplicable(rule, shiftID) {
			continue
		}

		classified := &ClassifiedRule{
			Rule:           rule,
			Category:       m.classifyCategory(rule),
			SubCategory:    m.classifySubCategory(rule),
			Dependencies:   make([]string, 0),
			Conflicts:      make([]string, 0),
			ExecutionOrder: rule.Priority,
		}

		// 根据分类添加到对应列表
		switch classified.Category {
		case "constraint":
			result.ConstraintRules = append(result.ConstraintRules, classified)
		case "preference":
			result.PreferenceRules = append(result.PreferenceRules, classified)
		case "dependency":
			result.DependencyRules = append(result.DependencyRules, classified)
		}
	}

	return result
}

// isRuleValid 检查规则在指定日期是否有效
func (m *RuleMatcher) isRuleValid(rule *model.Rule, date time.Time) bool {
	if !rule.IsActive {
		return false
	}
	if rule.ValidFrom != nil && date.Before(*rule.ValidFrom) {
		return false
	}
	if rule.ValidTo != nil && date.After(*rule.ValidTo) {
		return false
	}
	return true
}

// isRuleApplicable 检查规则是否适用于当前班次
//
// 版本语义：
//   - V3（rule.Version == "" 或 "v3"）：Associations 中 AssociationID 匹配即适用，不检查 Role。
//   - V4（rule.Version == "v4"）：Associations 中 AssociationType=="shift" 且 Role=="target" 且 AssociationID==shiftID 才算约束目标。
func (m *RuleMatcher) isRuleApplicable(rule *model.Rule, shiftID string) bool {
	if rule.Version == "v4" {
		// ── V4 路径：Associations 中班次关联且 Role 为 target/subject/object ──
		for _, assoc := range rule.Associations {
			if assoc.AssociationType == model.AssociationTypeShift && assoc.AssociationID == shiftID {
				if assoc.Role == model.RelationRoleTarget ||
					assoc.Role == model.RelationRoleSubject ||
					assoc.Role == model.RelationRoleObject {
					return true
				}
			}
		}
		return false
	}

	// ── V3 路径（rule.Version == "" 或 "v3"）──
	// V3 的 Associations 没有 Role 语义，只要 AssociationID 匹配即视为该规则作用于此班次
	for _, assoc := range rule.Associations {
		if assoc.AssociationType == "shift" {
			if assoc.AssociationID == shiftID {
				return true
			}
		}
	}
	return false
}

// classifyCategory 分类规则
func (m *RuleMatcher) classifyCategory(rule *model.Rule) string {
	// SDK model 可能没有 Category 字段，直接根据规则类型推断
	// 如果将来 SDK model 添加了 Category 字段，可以在这里添加检查
	// if rule.Category != "" {
	//     return rule.Category
	// }

	// 根据规则类型推断分类
	switch rule.RuleType {
	case "exclusive", "forbidden_day", "maxCount", "consecutiveMax", "minRestDays", "required_together", "periodic":
		return "constraint"
	case "preferred", "combinable":
		return "preference"
	default:
		return "constraint" // 默认
	}
}

// classifySubCategory 分类子类型
func (m *RuleMatcher) classifySubCategory(rule *model.Rule) string {
	// SDK model 可能没有 SubCategory 字段，直接根据规则类型推断
	// 如果将来 SDK model 添加了 SubCategory 字段，可以在这里添加检查
	// if rule.SubCategory != "" {
	//     return rule.SubCategory
	// }

	// 根据规则类型推断子分类
	switch rule.RuleType {
	case "exclusive", "forbidden_day":
		return "forbid"
	case "required_together", "periodic":
		return "must"
	case "maxCount", "consecutiveMax", "minRestDays":
		return "limit"
	case "preferred":
		return "prefer"
	case "combinable":
		return "suggest"
	default:
		return "limit"
	}
}
