// 班次管理模块相关的接口请求方法
import { request } from '@/utils/request'

const prefix = '/v1'

// ==================== 班次管理 ====================

/**
 * 查询班次列表
 */
export function getShiftList(params: Shift.ListParams) {
  return request<Shift.ListData>({
    url: `${prefix}/shifts`,
    method: 'get',
    params,
  })
}

/**
 * 获取班次详情
 */
export function getShiftDetail(id: string, orgId: string) {
  return request<Shift.ShiftInfo>({
    url: `${prefix}/shifts/${id}`,
    method: 'get',
    params: { orgId },
  })
}

/**
 * 创建班次
 */
export function createShift(data: Shift.CreateRequest) {
  return request<Shift.ShiftInfo>({
    url: `${prefix}/shifts`,
    method: 'post',
    data,
  })
}

/**
 * 更新班次信息
 */
export function updateShift(id: string, data: Shift.UpdateRequest) {
  return request<Shift.ShiftInfo>({
    url: `${prefix}/shifts/${id}`,
    method: 'put',
    params: { orgId: data.orgId },
    data,
  })
}

/**
 * 删除班次
 */
export function deleteShift(id: string, orgId: string) {
  return request({
    url: `${prefix}/shifts/${id}`,
    method: 'delete',
    params: { orgId },
  })
}

/**
 * 启用/禁用班次
 */
export function toggleShiftStatus(id: string, orgId: string, isActive: boolean) {
  return request({
    url: `${prefix}/shifts/${id}/status?orgId=${orgId}`,
    method: 'patch',
    data: {
      isActive,
    },
  })
}

// ==================== 班次分配 ====================

/**
 * 查询班次分配列表
 */
export function getShiftAssignmentList(params: Shift.AssignmentListParams) {
  return request<Shift.AssignmentListData>({
    url: `${prefix}/shift-assignments`,
    method: 'get',
    params,
  })
}

/**
 * 创建班次分配
 */
export function createShiftAssignment(data: Shift.CreateAssignmentRequest) {
  return request({
    url: `${prefix}/shift-assignments`,
    method: 'post',
    data,
  })
}

/**
 * 删除班次分配
 */
export function deleteShiftAssignment(id: string, orgId: string) {
  return request({
    url: `${prefix}/shift-assignments/${id}`,
    method: 'delete',
    params: { orgId },
  })
}

/**
 * 批量删除班次分配
 */
export function batchDeleteShiftAssignment(data: Shift.BatchDeleteAssignmentRequest) {
  return request({
    url: `${prefix}/shift-assignments/batch/delete`,
    method: 'post',
    data,
  })
}

/**
 * 按员工查询班次分配
 */
export function getShiftAssignmentByEmployee(params: Shift.AssignmentByEmployeeParams) {
  return request<Shift.AssignmentListData>({
    url: `${prefix}/shift-assignments/by-employee`,
    method: 'get',
    params,
  })
}

/**
 * 按日期范围查询班次分配
 */
export function getShiftAssignmentByDateRange(params: Shift.AssignmentByDateRangeParams) {
  return request<Shift.AssignmentListData>({
    url: `${prefix}/shift-assignments/by-date-range`,
    method: 'get',
    params,
  })
}

// ==================== 班次-分组关联 ====================

/**
 * 获取班次关联的所有分组
 */
export function getShiftGroups(shiftId: string) {
  return request<Shift.ShiftGroupInfo[]>({
    url: `${prefix}/shifts/${shiftId}/groups`,
    method: 'get',
  })
}

/**
 * 批量设置班次的关联分组
 */
export function setShiftGroups(shiftId: string, data: Shift.SetShiftGroupsRequest) {
  return request({
    url: `${prefix}/shifts/${shiftId}/groups`,
    method: 'put',
    data,
  })
}

/**
 * 为班次添加关联分组
 */
export function addGroupToShift(shiftId: string, groupId: string, priority: number = 0) {
  return request({
    url: `${prefix}/shifts/${shiftId}/groups/${groupId}`,
    method: 'post',
    data: { priority },
  })
}

/**
 * 从班次移除关联分组
 */
export function removeGroupFromShift(shiftId: string, groupId: string) {
  return request({
    url: `${prefix}/shifts/${shiftId}/groups/${groupId}`,
    method: 'delete',
  })
}

// ==================== 固定人员配置 ====================

/**
 * 批量创建/更新班次的固定人员配置
 */
export function batchCreateFixedAssignments(data: Shift.BatchCreateFixedAssignmentsRequest) {
  return request({
    url: `${prefix}/shifts/${data.shiftId}/fixed-assignments`,
    method: 'post',
    data,
  })
}

/**
 * 获取班次的固定人员配置列表
 */
export function getFixedAssignments(shiftId: string) {
  return request<Shift.FixedAssignment[]>({
    url: `${prefix}/shifts/${shiftId}/fixed-assignments`,
    method: 'get',
  })
}

/**
 * 删除固定人员配置
 */
export function deleteFixedAssignment(shiftId: string, assignmentId: string) {
  return request({
    url: `${prefix}/shifts/${shiftId}/fixed-assignments/${assignmentId}`,
    method: 'delete',
  })
}

/**
 * 计算固定排班
 */
export function calculateFixedSchedule(shiftId: string, data: Shift.CalculateFixedScheduleRequest) {
  return request<Shift.FixedScheduleResult>({
    url: `${prefix}/shifts/${shiftId}/fixed-assignments/calculate`,
    method: 'post',
    data,
  })
}
