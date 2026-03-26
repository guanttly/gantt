// 组织管理 API
import client from './client'

export interface OrgTreeNode {
  id: string
  parent_id?: string
  name: string
  type: 'institution' | 'department' | 'team'
  path: string
  status: 'active' | 'suspended'
  children?: OrgTreeNode[]
  created_at: string
  updated_at: string
}

export interface CreateOrgNodeRequest {
  parent_id?: string
  name: string
  type: 'institution' | 'department' | 'team'
}

export interface UpdateOrgNodeRequest {
  name?: string
}

export interface MoveOrgNodeRequest {
  new_parent_id: string
}

/** 获取组织树 */
export function getOrgTree() {
  return client.get<OrgTreeNode[]>('/admin/org-nodes').then(r => r.data)
}

/** 创建组织节点 */
export function createOrgNode(data: CreateOrgNodeRequest) {
  return client.post<OrgTreeNode>('/admin/org-nodes', data).then(r => r.data)
}

/** 获取节点详情 */
export function getOrgNode(id: string) {
  return client.get<OrgTreeNode>(`/admin/org-nodes/${id}`).then(r => r.data)
}

/** 更新节点 */
export function updateOrgNode(id: string, data: UpdateOrgNodeRequest) {
  return client.put<OrgTreeNode>(`/admin/org-nodes/${id}`, data).then(r => r.data)
}

/** 删除节点 */
export function deleteOrgNode(id: string) {
  return client.delete(`/admin/org-nodes/${id}`)
}

/** 停用节点 */
export function suspendOrgNode(id: string) {
  return client.put(`/admin/org-nodes/${id}/suspend`).then(r => r.data)
}

/** 启用节点 */
export function activateOrgNode(id: string) {
  return client.put(`/admin/org-nodes/${id}/activate`).then(r => r.data)
}

/** 移动节点 */
export function moveOrgNode(id: string, data: MoveOrgNodeRequest) {
  return client.put(`/admin/org-nodes/${id}/move`, data).then(r => r.data)
}

/** 获取子节点 */
export function getChildNodes(id: string) {
  return client.get<OrgTreeNode[]>(`/admin/org-nodes/${id}/children`).then(r => r.data)
}
