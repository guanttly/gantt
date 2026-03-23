import { request } from '@/utils/request'

const prefix = '/v1/staffing'
const shiftPrefix = '/v1/shifts'

/** 获取规则列表 */
export function getStaffingRules(params: Staffing.RuleListParams) {
  return request<Staffing.RuleListResult>({
    url: `${prefix}/rules`,
    method: 'get',
    params,
  })
}

/** 获取规则详情 */
export function getStaffingRule(id: string, orgId: string) {
  return request<Staffing.Rule>({
    url: `${prefix}/rules/${id}`,
    method: 'get',
    params: { orgId },
  })
}

/** 创建规则 */
export function createStaffingRule(orgId: string, data: Staffing.RuleRequest) {
  return request<Staffing.Rule>({
    url: `${prefix}/rules`,
    method: 'post',
    params: { orgId },
    data,
  })
}

/** 更新规则 */
export function updateStaffingRule(id: string, orgId: string, data: Staffing.RuleRequest) {
  return request<Staffing.Rule>({
    url: `${prefix}/rules/${id}`,
    method: 'put',
    params: { orgId },
    data,
  })
}

/** 删除规则 */
export function deleteStaffingRule(id: string, orgId: string) {
  return request({
    url: `${prefix}/rules/${id}`,
    method: 'delete',
    params: { orgId },
  })
}

/** 计算排班人数（预览） */
export function calculateStaffing(orgId: string, shiftId: string) {
  return request<Staffing.CalculationPreview>({
    url: `${prefix}/calculate`,
    method: 'post',
    params: { orgId },
    data: { shiftId },
  })
}

/** 应用计算结果 */
export function applyStaffingResult(orgId: string, data: Staffing.ApplyRequest) {
  return request<Staffing.ApplyResult>({
    url: `${prefix}/apply`,
    method: 'post',
    params: { orgId },
    data,
  })
}

/** 获取班次周配置 */
export function getShiftWeeklyStaff(shiftId: string, orgId: string) {
  return request<Staffing.WeeklyStaff>({
    url: `${shiftPrefix}/${shiftId}/weekly-staff`,
    method: 'get',
    params: { orgId },
  })
}

/** 更新班次周配置 */
export function updateShiftWeeklyStaff(shiftId: string, orgId: string, data: Staffing.WeeklyStaff) {
  return request<Staffing.WeeklyStaff>({
    url: `${shiftPrefix}/${shiftId}/weekly-staff`,
    method: 'put',
    params: { orgId },
    data,
  })
}

/** 批量获取多个班次的周配置 */
export function batchGetShiftWeeklyStaff(shiftIds: string[], orgId: string) {
  return request<Staffing.BatchWeeklyStaffResult>({
    url: `${shiftPrefix}/weekly-staff/batch`,
    method: 'get',
    params: { orgId, shiftIds: shiftIds.join(',') },
  })
}
