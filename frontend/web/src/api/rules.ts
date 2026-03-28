// 排班规则 API
import type { ListParams, PaginatedResponse } from '@/types/api'
import type {
  CreateRuleRequest,
  EffectiveRulesParams,
  ParseRuleRequest,
  ParseRuleResponse,
  Rule,
  RuleAssociationInput,
  UpdateRuleRequest,
  ValidateRuleRequest,
  ValidateRuleResponse,
} from '@/types/rule'
import client from './client'

// ==================== 规则 CRUD ====================

/** 规则列表 */
export function listRules(params?: ListParams) {
  return client.get<PaginatedResponse<Rule>>('/app/rules/', { params }).then(r => r.data)
}

/** 获取当前科室生效规则 */
export function getEffectiveRules(params?: EffectiveRulesParams) {
  return client.get<Rule[]>('/app/rules/effective', { params }).then(r => r.data)
}

/** 创建规则 */
export function createRule(data: CreateRuleRequest) {
  return client.post<Rule>('/app/rules/', data).then(r => r.data)
}

/** 获取规则详情 */
export function getRule(id: string) {
  return client.get<Rule>(`/app/rules/${id}`).then(r => r.data)
}

/** 更新规则 */
export function updateRule(id: string, data: UpdateRuleRequest) {
  return client.put<Rule>(`/app/rules/${id}`, data).then(r => r.data)
}

/** 删除规则 */
export function deleteRule(id: string) {
  return client.delete(`/app/rules/${id}`)
}

/** 启用/禁用规则 */
export function toggleRuleStatus(id: string, enabled: boolean) {
  return client.patch(`/rules/${id}/status`, { enabled })
}

/** 验证规则 */
export function validateRules(data: ValidateRuleRequest) {
  return client.post<ValidateRuleResponse>('/app/rules/validate', data).then(r => r.data)
}

// ==================== 规则关联 ====================

/** 查询规则关联 */
export function getRuleAssociations(ruleId: string) {
  return client.get(`/rules/${ruleId}/associations`).then(r => r.data)
}

/** 批量设置规则关联 */
export function setRuleAssociations(ruleId: string, associations: RuleAssociationInput[]) {
  return client.put(`/rules/${ruleId}/associations`, { associations })
}

/** 按员工查询规则 */
export function getRulesByEmployee(employeeId: string) {
  return client.get<Rule[]>('/rules/by-employee', { params: { employee_id: employeeId } }).then(r => r.data)
}

/** 按班次查询规则 */
export function getRulesByShift(shiftId: string) {
  return client.get<Rule[]>('/rules/by-shift', { params: { shift_id: shiftId } }).then(r => r.data)
}

/** 按分组查询规则 */
export function getRulesByGroup(groupId: string) {
  return client.get<Rule[]>('/rules/by-group', { params: { group_id: groupId } }).then(r => r.data)
}

// ==================== AI 规则解析 ====================

/** 解析语义化规则（LLM） */
export function parseRule(data: ParseRuleRequest) {
  return client.post<ParseRuleResponse>('/rules/parse', data, { timeout: 120000 }).then(r => r.data)
}

/** 批量保存解析后的规则 */
export function batchSaveRules(data: { parsed_rules: unknown[], dependencies: unknown[], conflicts: unknown[] }) {
  return client.post('/rules/batch', data).then(r => r.data)
}

/** 组织规则（排序依赖） */
export function organizeRules() {
  return client.post('/rules/organize').then(r => r.data)
}
