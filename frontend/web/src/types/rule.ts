// 排班规则类型定义

export interface Rule {
  id: string
  org_node_id: string
  name: string
  type: RuleType
  category: RuleCategory
  sub_category?: RuleSubCategory
  apply_scope: RuleApplyScope
  time_scope: RuleTimeScope
  time_offset_days?: number
  priority: number
  enabled: boolean
  rule_data: string
  config: Record<string, unknown>
  inherited?: boolean
  source_node_id?: string
  source_node_name?: string
  description?: string
  /** 关联统计 */
  association_count?: number
  employee_count?: number
  shift_count?: number
  group_count?: number
  /** V4 字段 */
  source_type?: RuleSourceType
  parse_confidence?: number
  version?: 'v4'
  /** 关联列表 */
  associations?: RuleAssociation[]
  apply_scopes?: RuleApplyScopeInfo[]
  created_at: string
  updated_at: string
}

export type RuleType
  = 'exclusive'
    | 'combinable'
    | 'required_together'
    | 'periodic'
    | 'maxCount'
    | 'forbidden_day'
    | 'preferred'

export type RuleCategory = 'hard' | 'soft' | 'preference' | 'constraint' | 'dependency'
export type RuleSubCategory = 'forbid' | 'must' | 'limit' | 'prefer' | 'suggest' | 'source' | 'resource' | 'order'
export type RuleApplyScope = 'global' | 'specific'
export type RuleTimeScope = 'same_day' | 'same_week' | 'same_month' | 'custom'
export type RuleSourceType = 'manual' | 'llm_parsed' | 'migrated'
export type RuleRelationRole = 'subject' | 'object' | 'target'
export type RuleScopeType = 'all' | 'employee' | 'group' | 'exclude_employee' | 'exclude_group'

export interface RuleAssociation {
  id: string
  rule_id: string
  association_type: 'employee' | 'shift' | 'group'
  association_id: string
  role?: RuleRelationRole
  created_at?: string
}

export interface RuleApplyScopeInfo {
  id: string
  rule_id: string
  scope_type: RuleScopeType
  scope_id?: string
  scope_name?: string
  created_at?: string
}

export interface CreateRuleRequest {
  name: string
  type: RuleType
  category: RuleCategory
  sub_category?: RuleSubCategory
  apply_scope?: RuleApplyScope
  time_scope?: RuleTimeScope
  time_offset_days?: number
  priority?: number
  enabled?: boolean
  rule_data?: string
  config: Record<string, unknown>
  description?: string
  source_type?: RuleSourceType
  associations?: RuleAssociationInput[]
  apply_scopes?: RuleApplyScopeInput[]
}

export interface RuleAssociationInput {
  association_type: 'employee' | 'shift' | 'group'
  association_id: string
  role?: RuleRelationRole
}

export interface RuleApplyScopeInput {
  scope_type: RuleScopeType
  scope_id?: string
  scope_name?: string
}

export interface UpdateRuleRequest {
  name?: string
  category?: RuleCategory
  sub_category?: RuleSubCategory
  priority?: number
  enabled?: boolean
  rule_data?: string
  config?: Record<string, unknown>
  description?: string
  source_type?: RuleSourceType
  associations?: RuleAssociationInput[]
  apply_scopes?: RuleApplyScopeInput[]
}

export interface ValidateRuleRequest {
  rules: Array<{
    type: RuleType
    config: Record<string, unknown>
  }>
}

export interface ValidateRuleResponse {
  valid: boolean
  errors: Array<{
    index: number
    message: string
  }>
}

/** 生效规则查询参数 */
export interface EffectiveRulesParams {
}

// ==================== AI 规则解析 ====================

export interface ParseRuleRequest {
  name: string
  rule_description: string
  apply_scope?: string
  priority: number
}

export interface ParsedRule {
  name: string
  category: RuleCategory
  sub_category: RuleSubCategory
  rule_type: RuleType
  apply_scope: RuleApplyScope
  time_scope: RuleTimeScope
  time_offset_days?: number
  description: string
  rule_data: string
  priority: number
  associations?: RuleAssociationInput[]
  subject_shifts?: string[]
  object_shifts?: string[]
  target_shifts?: string[]
  scope_type?: RuleScopeType
  scope_employees?: string[]
  scope_groups?: string[]
}

export interface RuleDependencyInfo {
  dependent_rule_name: string
  dependent_on_rule_name: string
  dependency_type: 'time' | 'source' | 'resource' | 'order'
  description: string
}

export interface RuleConflictInfo {
  rule_name_1: string
  rule_name_2: string
  conflict_type: 'exclusive' | 'resource' | 'time' | 'frequency'
  description: string
}

export interface ParseRuleResponse {
  original_rule: string
  parsed_rules: ParsedRule[]
  dependencies: RuleDependencyInfo[]
  conflicts: RuleConflictInfo[]
  reasoning: string
}

// ==================== 常量 ====================

export const RULE_TYPE_OPTIONS = [
  { label: '排他规则', value: 'exclusive' },
  { label: '可合并规则', value: 'combinable' },
  { label: '必须同时规则', value: 'required_together' },
  { label: '周期性规则', value: 'periodic' },
  { label: '最大次数规则', value: 'maxCount' },
  { label: '禁止日期规则', value: 'forbidden_day' },
  { label: '偏好规则', value: 'preferred' },
] as const

export const RULE_CATEGORY_OPTIONS = [
  { label: '约束', value: 'constraint' },
  { label: '偏好', value: 'preference' },
  { label: '依赖', value: 'dependency' },
] as const

export const RULE_APPLY_SCOPE_OPTIONS = [
  { label: '全局', value: 'global' },
  { label: '特定对象', value: 'specific' },
] as const

export const RULE_TIME_SCOPE_OPTIONS = [
  { label: '同一天', value: 'same_day' },
  { label: '同一周', value: 'same_week' },
  { label: '同一月', value: 'same_month' },
  { label: '自定义', value: 'custom' },
] as const

export const RULE_SOURCE_TYPE_OPTIONS = [
  { label: '手动创建', value: 'manual' },
  { label: 'LLM 解析', value: 'llm_parsed' },
  { label: '迁移', value: 'migrated' },
] as const

/** 获取规则类型文本 */
export function getRuleTypeText(type: string): string {
  const opt = RULE_TYPE_OPTIONS.find(o => o.value === type)
  return opt?.label ?? type
}

/** 获取规则分类标签类型 */
export function getRuleCategoryTagType(category?: string): string {
  switch (category) {
    case 'constraint': case 'hard': return 'danger'
    case 'preference': case 'soft': return 'warning'
    case 'dependency': return 'info'
    default: return ''
  }
}

/** 获取规则来源类型标签类型 */
export function getRuleSourceTypeTagType(sourceType?: string): string {
  switch (sourceType) {
    case 'manual': return 'success'
    case 'llm_parsed': return 'primary'
    case 'migrated': return 'warning'
    default: return ''
  }
}
