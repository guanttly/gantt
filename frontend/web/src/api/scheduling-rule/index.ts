// 排班规则管理模块相关的接口请求方法
import { request } from '@/utils/request'

const prefix = '/v1'

// ==================== 排班规则管理 ====================

/**
 * 查询排班规则列表
 */
export function getSchedulingRuleList(params: SchedulingRule.ListParams) {
  return request<SchedulingRule.ListData>({
    url: `${prefix}/scheduling-rules`,
    method: 'get',
    params,
  })
}

/**
 * 获取排班规则详情
 */
export function getSchedulingRuleDetail(id: string, orgId: string) {
  return request<SchedulingRule.RuleInfo>({
    url: `${prefix}/scheduling-rules/${id}`,
    method: 'get',
    params: { orgId },
  })
}

/**
 * 创建排班规则
 */
export function createSchedulingRule(data: SchedulingRule.CreateRequest) {
  return request<SchedulingRule.RuleInfo>({
    url: `${prefix}/scheduling-rules`,
    method: 'post',
    data,
  })
}

/**
 * 更新排班规则
 */
export function updateSchedulingRule(id: string, orgId: string, data: SchedulingRule.UpdateRequest) {
  return request<SchedulingRule.RuleInfo>({
    url: `${prefix}/scheduling-rules/${id}`,
    method: 'put',
    params: { orgId },
    data,
  })
}

/**
 * 删除排班规则
 */
export function deleteSchedulingRule(id: string, orgId: string) {
  return request({
    url: `${prefix}/scheduling-rules/${id}`,
    method: 'delete',
    params: { orgId },
  })
}

/**
 * 启用/禁用排班规则
 */
export function toggleSchedulingRuleStatus(id: string, orgId: string, isActive: boolean) {
  return request({
    url: `${prefix}/scheduling-rules/${id}/status?orgId=${orgId}`,
    method: 'patch',
    data: {
      isActive,
    },
  })
}

// ==================== 规则关联管理 ====================

/**
 * 查询规则的关联关系
 */
export function getRuleAssociations(params: SchedulingRule.GetAssociationsParams) {
  return request<SchedulingRule.AssociationListData>({
    url: `${prefix}/scheduling-rules/${params.ruleId}/associations`,
    method: 'get',
    params: { orgId: params.orgId },
  })
}

/**
 * 创建规则关联
 */
export function createRuleAssociation(data: SchedulingRule.CreateAssociationRequest) {
  return request({
    url: `${prefix}/scheduling-rules/associations`,
    method: 'post',
    data,
  })
}

/**
 * 批量创建规则关联
 */
export function batchCreateRuleAssociation(data: SchedulingRule.BatchCreateAssociationRequest) {
  return request({
    url: `${prefix}/scheduling-rules/associations/batch`,
    method: 'post',
    data,
  })
}

/**
 * 删除规则关联
 */
export function deleteRuleAssociation(ruleId: string, targetType: string, targetId: string, orgId: string) {
  return request({
    url: `${prefix}/scheduling-rules/${ruleId}/associations`,
    method: 'delete',
    params: { targetType, targetId, orgId },
  })
}

/**
 * 批量删除规则关联
 */
export function batchDeleteRuleAssociation(data: SchedulingRule.BatchDeleteAssociationRequest) {
  return request({
    url: `${prefix}/scheduling-rules/associations/batch/delete`,
    method: 'post',
    data,
  })
}

// ==================== 规则查询 ====================

/**
 * 按员工查询规则
 */
export function getRulesByEmployee(params: SchedulingRule.RulesByEmployeeParams) {
  return request<SchedulingRule.ListData>({
    url: `${prefix}/scheduling-rules/by-employee`,
    method: 'get',
    params,
  })
}

/**
 * 按班次查询规则
 */
export function getRulesByShift(params: SchedulingRule.RulesByShiftParams) {
  return request<SchedulingRule.ListData>({
    url: `${prefix}/scheduling-rules/by-shift`,
    method: 'get',
    params,
  })
}

/**
 * 按分组查询规则
 */
export function getRulesByGroup(params: SchedulingRule.RulesByGroupParams) {
  return request<SchedulingRule.ListData>({
    url: `${prefix}/scheduling-rules/by-group`,
    method: 'get',
    params,
  })
}

/**
 * 验证排班是否符合规则
 */
export function validateSchedulingRules(data: SchedulingRule.ValidationRequest) {
  return request<SchedulingRule.ValidationData>({
    url: `${prefix}/scheduling-rules/validate`,
    method: 'post',
    data,
  })
}

// ==================== V4 规则组织功能 ====================

/**
 * 解析语义化规则（V4）
 * 由于涉及 LLM 调用，设置较长超时时间
 */
export function parseRule(data: SchedulingRule.ParseRuleRequest) {
  return request<SchedulingRule.ParseRuleResponse>({
    url: `${prefix}/scheduling-rules/parse`,
    method: 'post',
    data,
    timeout: 120000, // LLM 调用需要较长时间，设置 120 秒
  })
}

/**
 * 批量保存解析后的规则（V4）
 */
export function batchSaveRules(data: SchedulingRule.BatchSaveRulesRequest) {
  return request<SchedulingRule.BatchSaveRulesResponse>({
    url: `${prefix}/scheduling-rules/batch`,
    method: 'post',
    data,
  })
}

/**
 * 组织规则（V4）
 */
export function organizeRules(orgId: string) {
  return request<SchedulingRule.RuleOrganizationResult>({
    url: `${prefix}/scheduling-rules/organize`,
    method: 'post',
    params: { orgId },
  })
}
