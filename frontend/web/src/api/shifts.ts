// 班次管理 API
import type { ListParams, ListResponse } from '@/types/api'
import type {
  CreateShiftRequest,
  FixedAssignment,
  Shift,
  ShiftDependency,
  ShiftGroup,
  UpdateShiftRequest,
  WeeklyStaff,
} from '@/types/shift'
import client from './client'

// ==================== 班次 CRUD ====================

/** 班次列表 */
export function listShifts(params?: ListParams) {
  return client.get<ListResponse<Shift>>('/app/shifts/', { params }).then(r => r.data)
}

/** 获取班次详情 */
export function getShift(id: string) {
  return client.get<Shift>(`/app/shifts/${id}`).then(r => r.data)
}

/** 创建班次 */
export function createShift(data: CreateShiftRequest) {
  return client.post<Shift>('/app/shifts/', data).then(r => r.data)
}

/** 更新班次 */
export function updateShift(id: string, data: UpdateShiftRequest) {
  return client.put<Shift>(`/app/shifts/${id}`, data).then(r => r.data)
}

/** 删除班次 */
export function deleteShift(id: string) {
  return client.delete(`/app/shifts/${id}`)
}

/** 启用/禁用班次 */
export function toggleShiftStatus(id: string, isActive: boolean) {
  return client.put<Shift>(`/app/shifts/${id}/toggle`, { is_active: isActive }).then(r => r.data)
}

// ==================== 班次依赖 ====================

/** 获取班次依赖 */
export function getShiftDependencies() {
  return client.get<ShiftDependency[]>('/app/shifts/dependencies').then(r => r.data)
}

/** 添加班次依赖 */
export function addShiftDependency(data: { shift_id: string, depends_on_shift_id: string, type: string }) {
  return client.post<ShiftDependency>('/app/shifts/dependencies', data).then(r => r.data)
}

// ==================== 班次分组关联 ====================

/** 获取班次关联的分组 */
export function getShiftGroups(shiftId: string) {
  return client.get<ShiftGroup[]>(`/app/shifts/${shiftId}/groups`).then(r => r.data)
}

/** 批量设置班次关联分组 */
export function setShiftGroups(shiftId: string, groupIds: string[]) {
  return client.put<ShiftGroup[]>(`/app/shifts/${shiftId}/groups`, { group_ids: groupIds }).then(r => r.data)
}

/** 为班次添加关联分组 */
export function addGroupToShift(shiftId: string, groupId: string, priority = 0) {
  return client.post<ShiftGroup>(`/app/shifts/${shiftId}/groups/${groupId}`, { priority }).then(r => r.data)
}

/** 从班次移除关联分组 */
export function removeGroupFromShift(shiftId: string, groupId: string) {
  return client.delete(`/app/shifts/${shiftId}/groups/${groupId}`)
}

// ==================== 固定人员配置 ====================

/** 获取班次固定人员配置 */
export function getFixedAssignments(shiftId: string) {
  return client.get<FixedAssignment[]>(`/app/shifts/${shiftId}/fixed-assignments`).then(r => r.data)
}

/** 批量创建/更新固定人员配置 */
export function batchCreateFixedAssignments(shiftId: string, assignments: FixedAssignment[]) {
  return client.post<FixedAssignment[]>(`/app/shifts/${shiftId}/fixed-assignments`, { shift_id: shiftId, assignments }).then(r => r.data)
}

/** 删除固定人员配置 */
export function deleteFixedAssignment(shiftId: string, assignmentId: string) {
  return client.delete(`/app/shifts/${shiftId}/fixed-assignments/${assignmentId}`)
}

/** 计算固定排班 */
export function calculateFixedSchedule(shiftId: string, data: { start_date: string, end_date: string }) {
  return client.post<{ date_to_employee_ids: Record<string, string[]> }>(`/app/shifts/${shiftId}/fixed-assignments/calculate`, data).then(r => r.data)
}

// ==================== 周人数配置 ====================

/** 获取班次周配置 */
export function getShiftWeeklyStaff(shiftId: string) {
  return client.get<WeeklyStaff>(`/app/shifts/${shiftId}/weekly-staff`).then(r => r.data)
}

/** 更新班次周配置 */
export function updateShiftWeeklyStaff(shiftId: string, data: WeeklyStaff) {
  return client.put<WeeklyStaff>(`/app/shifts/${shiftId}/weekly-staff`, data).then(r => r.data)
}

/** 批量获取多个班次周配置 */
export function batchGetShiftWeeklyStaff(shiftIds: string[]) {
  return client.get<Record<string, WeeklyStaff>>('/app/shifts/weekly-staff/batch', {
    params: { shift_ids: shiftIds.join(',') },
  }).then(r => r.data)
}
