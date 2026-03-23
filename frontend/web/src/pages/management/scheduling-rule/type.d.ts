// 排班规则管理模块相关的类型定义

/** 规则查询表单 */
export interface RuleQueryForm {
  orgId: string
  ruleType: SchedulingRule.RuleType | undefined
  applyScope: SchedulingRule.ApplyScope | undefined
  timeScope: SchedulingRule.TimeScope | undefined
  isActive: boolean | undefined
  keyword: string
  page: number
  size: number
}

/** 规则表单 */
export interface RuleFormData {
  orgId: string
  name: string
  ruleType: SchedulingRule.RuleType
  applyScope: SchedulingRule.ApplyScope
  timeScope: SchedulingRule.TimeScope
  priority: number
  config: Record<string, any>
  description?: string
}
