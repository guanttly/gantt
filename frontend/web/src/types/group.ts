// 分组相关类型定义

export type GroupType = 'department' | 'team' | 'project' | 'custom'

export interface Group {
  id: string
  org_node_id: string
  name: string
  code: string
  type: GroupType
  parent_id?: string
  member_count: number
  description?: string
  created_at: string
  updated_at: string
}

export interface GroupTreeNode {
  id: string
  name: string
  type: GroupType
  parent_id?: string
  member_count: number
  children: GroupTreeNode[]
}

export interface CreateGroupRequest {
  name: string
  code: string
  type: GroupType
  parent_id?: string
  description?: string
}

export interface UpdateGroupRequest {
  name?: string
  code?: string
  type?: GroupType
  parent_id?: string
  description?: string
}

export interface GroupMember {
  id: string
  employee_id: string
  employee_name: string
  employee_no?: string
  role?: string
  joined_at: string
}

// ==================== 常量 ====================

export const GROUP_TYPE_OPTIONS = [
  { label: '部门', value: 'department' as GroupType },
  { label: '团队', value: 'team' as GroupType },
  { label: '项目', value: 'project' as GroupType },
  { label: '自定义', value: 'custom' as GroupType },
] as const

export function getGroupTypeText(type: GroupType): string {
  const map: Record<GroupType, string> = {
    department: '部门',
    team: '团队',
    project: '项目',
    custom: '自定义',
  }
  return map[type] || type
}

export function getGroupTypeTagType(type: GroupType): string {
  const map: Record<GroupType, string> = {
    department: '',
    team: 'success',
    project: 'warning',
    custom: 'info',
  }
  return map[type] || 'info'
}

/** 构建分组树 */
export function buildGroupTree(groups: Group[]): GroupTreeNode[] {
  const map = new Map<string, GroupTreeNode>()
  const roots: GroupTreeNode[] = []

  groups.forEach((g) => {
    map.set(g.id, {
      id: g.id,
      name: g.name,
      type: g.type,
      parent_id: g.parent_id,
      member_count: g.member_count,
      children: [],
    })
  })

  groups.forEach((g) => {
    const node = map.get(g.id)!
    if (g.parent_id && map.has(g.parent_id)) {
      map.get(g.parent_id)!.children.push(node)
    }
    else {
      roots.push(node)
    }
  })

  return roots
}
