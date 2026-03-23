// 排班管理API

import { request } from '@/utils/request'

const prefix = '/v1/scheduling'

/**
 * 批量分配排班
 */
export function batchAssignSchedule(data: Scheduling.BatchAssignRequest) {
  return request({
    url: `${prefix}/assignments/batch`,
    method: 'post',
    data,
  })
}

/**
 * 查询日期范围内的排班
 */
export function getScheduleByDateRange(params: Scheduling.QueryParams) {
  return request<Scheduling.Assignment[]>({
    url: `${prefix}/assignments`,
    method: 'get',
    params,
  })
}

/**
 * 查询员工排班
 */
export function getEmployeeSchedule(params: Scheduling.EmployeeQueryParams) {
  return request<Scheduling.Assignment[]>({
    url: `${prefix}/assignments/employee`,
    method: 'get',
    params,
  })
}

/**
 * 删除排班分配
 */
export function deleteScheduleAssignment(params: Scheduling.DeleteParams) {
  return request({
    url: `${prefix}/assignments`,
    method: 'delete',
    params,
  })
}

/**
 * 批量删除排班
 */
export function batchDeleteSchedule(data: Scheduling.BatchDeleteRequest) {
  return request({
    url: `${prefix}/assignments/batch/delete`,
    method: 'post',
    data,
  })
}

/**
 * 获取排班汇总
 */
export function getScheduleSummary(params: Scheduling.QueryParams) {
  return request<Scheduling.Summary>({
    url: `${prefix}/summary`,
    method: 'get',
    params,
  })
}
