package http

import (
	"jusha/gantt/service/management/domain/model"
)

// ============================================================================
// V3 兼容层
// @deprecated V3: 此文件用于兼容 V3 前端旧枚举值，迁移完成后可移除
// ============================================================================

// normalizeRuleType 将 V3 前端旧枚举值转换为后端值
// @deprecated V3: 迁移完成后可移除
func normalizeRuleType(v3Value string) model.RuleType {
	// V3 前端枚举值 → V4 后端值映射
	mapping := map[string]model.RuleType{
		// V3 旧值 → V4 新值
		"max_shifts":         model.RuleTypeMaxCount,      // 最大班次数 → maxCount
		"consecutive_shifts": model.RuleTypeMaxCount,     // 连续班次 → maxCount (需要结合 ConsecutiveMax)
		"rest_days":          model.RuleTypeMaxCount,     // 休息日 → maxCount (需要结合 MinRestDays)
		"forbidden_pattern":  model.RuleTypeForbiddenDay, // 禁止模式 → forbidden_day
		"preferred_pattern":  model.RuleTypePreferred,   // 偏好模式 → preferred
		// V4 值直接返回
		"exclusive":         model.RuleTypeExclusive,
		"combinable":        model.RuleTypeCombinable,
		"required_together": model.RuleTypeRequiredTogether,
		"periodic":          model.RuleTypePeriodic,
		"maxCount":          model.RuleTypeMaxCount,
		"forbidden_day":     model.RuleTypeForbiddenDay,
		"preferred":         model.RuleTypePreferred,
	}

	if normalized, ok := mapping[v3Value]; ok {
		return normalized
	}

	// 如果找不到映射，尝试直接转换（可能是 V4 值）
	return model.RuleType(v3Value)
}

// normalizeApplyScope 将 V3 前端旧枚举值转换为后端值
// @deprecated V3: 迁移完成后可移除
func normalizeApplyScope(v3Value string) model.ApplyScope {
	// V3 前端枚举值 → V4 后端值映射
	mapping := map[string]model.ApplyScope{
		// V3 旧值 → V4 新值
		"global":   model.ApplyScopeGlobal,   // 全局 → global (不变)
		"group":    model.ApplyScopeSpecific, // 分组 → specific
		"employee": model.ApplyScopeSpecific, // 员工 → specific
		"shift":    model.ApplyScopeSpecific, // 班次 → specific
		// V4 值直接返回
		"specific": model.ApplyScopeSpecific,
	}

	if normalized, ok := mapping[v3Value]; ok {
		return normalized
	}

	// 如果找不到映射，尝试直接转换（可能是 V4 值）
	return model.ApplyScope(v3Value)
}

// normalizeTimeScope 将 V3 前端旧枚举值转换为后端值
// @deprecated V3: 迁移完成后可移除
func normalizeTimeScope(v3Value string) model.TimeScope {
	// V3 前端枚举值 → V4 后端值映射
	mapping := map[string]model.TimeScope{
		// V3 旧值 → V4 新值
		"daily":   model.TimeScopeSameDay,   // 每日 → same_day
		"weekly":  model.TimeScopeSameWeek,  // 每周 → same_week
		"monthly": model.TimeScopeSameMonth, // 每月 → same_month
		"custom":  model.TimeScopeCustom,    // 自定义 → custom (不变)
		// V4 值直接返回
		"same_day":   model.TimeScopeSameDay,
		"same_week":  model.TimeScopeSameWeek,
		"same_month": model.TimeScopeSameMonth,
	}

	if normalized, ok := mapping[v3Value]; ok {
		return normalized
	}

	// 如果找不到映射，尝试直接转换（可能是 V4 值）
	return model.TimeScope(v3Value)
}

// FillV3Defaults 为 V3 规则填充 V4 默认值
// @deprecated V3: 迁移完成后可移除
func FillV3Defaults(rule *model.SchedulingRule) {
	if rule == nil {
		return
	}

	// 如果 Version 为空或为 v3，填充默认值
	if rule.Version == "" || rule.Version == "v3" {
		// 判断是否为 V3 规则（通过检查是否有 V4 字段来判断）
		if rule.Category == "" && rule.SubCategory == "" && rule.SourceType == "" {
			// 这是一个 V3 规则，填充默认值
			rule.Version = "v3"
			rule.SourceType = "manual" // V3 规则默认为手动创建

			// 根据规则类型推断分类
			if rule.Category == "" {
				rule.Category = inferCategoryFromRuleType(rule.RuleType)
			}
			if rule.SubCategory == "" {
				rule.SubCategory = inferSubCategoryFromRuleType(rule.RuleType)
			}
		} else {
			// 已经有 V4 字段，设置为 v4
			if rule.Version == "" {
				rule.Version = "v4"
			}
		}
	}

	// 如果 SourceType 为空，默认为 manual
	if rule.SourceType == "" {
		rule.SourceType = "manual"
	}

	// 如果 Version 为空，默认为 v4（新规则）
	if rule.Version == "" {
		rule.Version = "v4"
	}
}

// inferCategoryFromRuleType 根据规则类型推断分类
func inferCategoryFromRuleType(ruleType model.RuleType) string {
	switch ruleType {
	case model.RuleTypeExclusive, model.RuleTypeForbiddenDay, model.RuleTypeMaxCount,
		model.RuleTypeRequiredTogether, model.RuleTypePeriodic:
		return model.CategoryConstraint
	case model.RuleTypePreferred, model.RuleTypeCombinable:
		return model.CategoryPreference
	default:
		return model.CategoryConstraint // 默认约束
	}
}

// inferSubCategoryFromRuleType 根据规则类型推断子分类
func inferSubCategoryFromRuleType(ruleType model.RuleType) string {
	switch ruleType {
	case model.RuleTypeExclusive, model.RuleTypeForbiddenDay:
		return model.SubCategoryForbid
	case model.RuleTypeRequiredTogether, model.RuleTypePeriodic:
		return model.SubCategoryMust
	case model.RuleTypeMaxCount:
		return model.SubCategoryLimit
	case model.RuleTypePreferred:
		return model.SubCategoryPrefer
	case model.RuleTypeCombinable:
		return model.SubCategorySuggest
	default:
		return model.SubCategoryLimit // 默认限制
	}
}
