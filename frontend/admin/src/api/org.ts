// 组织管理 API
import client from './client'

export type OrgNodeType = 'organization' | 'campus' | 'department' | 'custom'

export const PLATFORM_ROOT_CODE = 'platform-root'

export const ORG_NODE_TYPE_LABELS: Record<OrgNodeType, string> = {
  organization: '机构',
  campus: '院区',
  department: '部门',
  custom: '自定义',
}

const ALLOWED_CHILD_NODE_TYPES: Record<OrgNodeType, OrgNodeType[]> = {
  organization: ['campus', 'department', 'custom'],
  campus: ['department', 'custom'],
  custom: ['department', 'custom'],
  department: [],
}

export interface OrgTreeNode {
  id: string
  parent_id?: string
  code: string
  name: string
  node_type: OrgNodeType
  path: string
  depth: number
  is_login_point: boolean
  status: 'active' | 'suspended'
  children?: OrgTreeNode[]
  created_at: string
  updated_at: string
}

export interface CreateOrgNodeRequest {
  parent_id?: string
  name: string
  code: string
  node_type: OrgNodeType
}

export interface UpdateOrgNodeRequest {
  name?: string
}

export function isProtectedOrgNode(node: OrgTreeNode) {
  return !node.parent_id && node.code === PLATFORM_ROOT_CODE
}

export function getAllowedChildNodeTypes(parentType: OrgNodeType) {
  return ALLOWED_CHILD_NODE_TYPES[parentType] || []
}

export function canCreateChildNode(node: OrgTreeNode | OrgNodeType) {
  const parentType = typeof node === 'string' ? node : node.node_type
  return getAllowedChildNodeTypes(parentType).length > 0
}

export function getOrgTree() {
  return client.get<OrgTreeNode[]>('/admin/org-nodes').then(r => r.data)
}

export function createOrgNode(data: CreateOrgNodeRequest) {
  return client.post<OrgTreeNode>('/admin/org-nodes', data).then(r => r.data)
}

export function updateOrgNode(id: string, data: UpdateOrgNodeRequest) {
  return client.put<OrgTreeNode>(`/admin/org-nodes/${id}`, data).then(r => r.data)
}

export function deleteOrgNode(id: string) {
  return client.delete(`/admin/org-nodes/${id}`)
}
