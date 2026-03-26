// 请假管理 API
import type { ListParams, PaginatedResponse } from '@/types/api'
import client from './client'

export interface Leave {
  id: string
  org_node_id: string
  employee_id: string
  employee_name: string
  type: string
  start_date: string
  end_date: string
  reason?: string
  status: 'pending' | 'approved' | 'rejected'
  approved_by?: string
  created_at: string
  updated_at: string
}

export interface CreateLeaveRequest {
  employee_id: string
  type: string
  start_date: string
  end_date: string
  reason?: string
}

export interface UpdateLeaveRequest {
  type?: string
  start_date?: string
  end_date?: string
  reason?: string
}

/** 请假列表 */
export function listLeaves(params?: ListParams) {
  return client.get<PaginatedResponse<Leave>>('/leaves/', { params }).then(r => r.data)
}

/** 创建请假 */
export function createLeave(data: CreateLeaveRequest) {
  return client.post<Leave>('/leaves/', data).then(r => r.data)
}

/** 更新请假 */
export function updateLeave(id: string, data: UpdateLeaveRequest) {
  return client.put<Leave>(`/leaves/${id}`, data).then(r => r.data)
}

/** 删除请假 */
export function deleteLeave(id: string) {
  return client.delete(`/leaves/${id}`)
}

/** 审批请假 */
export function approveLeave(id: string, data: { approved: boolean, comment?: string }) {
  return client.put(`/leaves/${id}/approve`, data).then(r => r.data)
}
