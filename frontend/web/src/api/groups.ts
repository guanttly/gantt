// 分组管理 API
import type { ListParams, PaginatedResponse } from '@/types/api'
import client from './client'

export interface Group {
  id: string
  org_node_id: string
  name: string
  description?: string
  member_count: number
  created_at: string
  updated_at: string
}

export interface GroupMember {
  id: string
  employee_id: string
  employee_name: string
  joined_at: string
}

export interface CreateGroupRequest {
  name: string
  description?: string
}

export interface UpdateGroupRequest {
  name?: string
  description?: string
}

/** 分组列表 */
export function listGroups(params?: ListParams) {
  return client.get<PaginatedResponse<Group>>('/groups/', { params }).then(r => r.data)
}

/** 创建分组 */
export function createGroup(data: CreateGroupRequest) {
  return client.post<Group>('/groups/', data).then(r => r.data)
}

/** 更新分组 */
export function updateGroup(id: string, data: UpdateGroupRequest) {
  return client.put<Group>(`/groups/${id}`, data).then(r => r.data)
}

/** 删除分组 */
export function deleteGroup(id: string) {
  return client.delete(`/groups/${id}`)
}

/** 获取分组成员 */
export function getGroupMembers(id: string) {
  return client.get<GroupMember[]>(`/groups/${id}/members`).then(r => r.data)
}

/** 添加分组成员 */
export function addGroupMember(groupId: string, data: { employee_id: string }) {
  return client.post(`/groups/${groupId}/members`, data).then(r => r.data)
}

/** 移除分组成员 */
export function removeGroupMember(groupId: string, employeeId: string) {
  return client.delete(`/groups/${groupId}/members/${employeeId}`)
}
