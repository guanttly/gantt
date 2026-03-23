// 排班规则管理模块的业务逻辑和常量

/** 默认查询参数 */
export const defaultQueryParams = {
  orgId: 'default-org',
  ruleType: undefined,
  applyScope: undefined,
  timeScope: undefined,
  isActive: undefined,
  keyword: '',
  page: 1,
  size: 20,
}

// ============================================================================
// V4 枚举值（与后端保持一致）
// ============================================================================

/** 规则类型选项（V4 后端值） */
export const ruleTypeOptions = [
  { label: '排他规则', value: 'exclusive' },
  { label: '可合并规则', value: 'combinable' },
  { label: '必须同时规则', value: 'required_together' },
  { label: '周期性规则', value: 'periodic' },
  { label: '最大次数规则', value: 'maxCount' },
  { label: '禁止日期规则', value: 'forbidden_day' },
  { label: '偏好规则', value: 'preferred' },
]

/** 应用范围选项（V4 后端值） */
export const applyScopeOptions = [
  { label: '全局', value: 'global' },
  { label: '特定对象', value: 'specific' },
]

/** 时间范围选项（V4 后端值） */
export const timeScopeOptions = [
  { label: '同一天', value: 'same_day' },
  { label: '同一周', value: 'same_week' },
  { label: '同一月', value: 'same_month' },
  { label: '自定义', value: 'custom' },
]

// ============================================================================
// V3 兼容选项（用于向后兼容显示）
// @deprecated V3: 迁移完成后可移除
// ============================================================================

import { denormalizeRuleType, denormalizeApplyScope, denormalizeTimeScope } from './v3-compat'

/** V3 规则类型选项（用于向后兼容） */
export const v3RuleTypeOptions = [
  { label: '最大班次数', value: 'max_shifts' },
  { label: '连续班次', value: 'consecutive_shifts' },
  { label: '休息日', value: 'rest_days' },
  { label: '禁止模式', value: 'forbidden_pattern' },
  { label: '偏好模式', value: 'preferred_pattern' },
]

/** V3 应用范围选项（用于向后兼容） */
export const v3ApplyScopeOptions = [
  { label: '全局', value: 'global' },
  { label: '分组', value: 'group' },
  { label: '员工', value: 'employee' },
  { label: '班次', value: 'shift' },
]

/** V3 时间范围选项（用于向后兼容） */
export const v3TimeScopeOptions = [
  { label: '每日', value: 'daily' },
  { label: '每周', value: 'weekly' },
  { label: '每月', value: 'monthly' },
  { label: '自定义', value: 'custom' },
]

/** 获取规则类型文本（支持 V4 和 V3 枚举值） */
export function getRuleTypeText(type: SchedulingRule.RuleType | string): string {
  // 先尝试 V4 枚举值
  const v4Option = ruleTypeOptions.find(o => o.value === type)
  if (v4Option) {
    return v4Option.label
  }
  // 再尝试 V3 枚举值（向后兼容）
  const v3Option = v3RuleTypeOptions.find(o => o.value === type)
  if (v3Option) {
    return v3Option.label
  }
  return type
}

/** 获取应用范围文本（支持 V4 和 V3 枚举值） */
export function getApplyScopeText(scope: SchedulingRule.ApplyScope | string): string {
  // 先尝试 V4 枚举值
  const v4Option = applyScopeOptions.find(o => o.value === scope)
  if (v4Option) {
    return v4Option.label
  }
  // 再尝试 V3 枚举值（向后兼容）
  const v3Option = v3ApplyScopeOptions.find(o => o.value === scope)
  if (v3Option) {
    return v3Option.label
  }
  return scope
}

/** 获取时间范围文本（支持 V4 和 V3 枚举值） */
export function getTimeScopeText(scope: SchedulingRule.TimeScope | string): string {
  // 先尝试 V4 枚举值
  const v4Option = timeScopeOptions.find(o => o.value === scope)
  if (v4Option) {
    return v4Option.label
  }
  // 再尝试 V3 枚举值（向后兼容）
  const v3Option = v3TimeScopeOptions.find(o => o.value === scope)
  if (v3Option) {
    return v3Option.label
  }
  return scope
}

/** 规则分类选项 */
export const categoryOptions = [
  { label: '约束', value: 'constraint' },
  { label: '偏好', value: 'preference' },
  { label: '依赖', value: 'dependency' },
]

/** 获取规则分类文本 */
export function getCategoryText(category?: SchedulingRule.Category): string {
  if (!category) return '-'
  const option = categoryOptions.find(o => o.value === category)
  return option ? option.label : category
}

/** 获取规则分类标签类型 */
export function getCategoryTagType(category?: SchedulingRule.Category): string {
  switch (category) {
    case 'constraint':
      return 'danger'
    case 'preference':
      return 'warning'
    case 'dependency':
      return 'info'
    default:
      return ''
  }
}

/** 规则子分类选项 */
export const subCategoryOptions = [
  { label: '禁止', value: 'forbid' },
  { label: '必须', value: 'must' },
  { label: '限制', value: 'limit' },
  { label: '优先', value: 'prefer' },
  { label: '建议', value: 'suggest' },
  { label: '来源', value: 'source' },
  { label: '资源', value: 'resource' },
  { label: '顺序', value: 'order' },
]

/** 获取规则子分类文本 */
export function getSubCategoryText(subCategory?: SchedulingRule.SubCategory): string {
  if (!subCategory) return '-'
  const option = subCategoryOptions.find(o => o.value === subCategory)
  return option ? option.label : subCategory
}

/** 规则来源类型选项 */
export const sourceTypeOptions = [
  { label: '手动创建', value: 'manual' },
  { label: 'LLM 解析', value: 'llm_parsed' },
  { label: '迁移', value: 'migrated' },
]

/** 获取规则来源类型文本 */
export function getSourceTypeText(sourceType?: 'manual' | 'llm_parsed' | 'migrated'): string {
  if (!sourceType) return '-'
  const option = sourceTypeOptions.find(o => o.value === sourceType)
  return option ? option.label : sourceType
}

/** 获取规则来源类型标签类型 */
export function getSourceTypeTagType(sourceType?: 'manual' | 'llm_parsed' | 'migrated'): string {
  switch (sourceType) {
    case 'manual':
      return 'success'
    case 'llm_parsed':
      return 'primary'
    case 'migrated':
      return 'warning'
    default:
      return ''
  }
}
