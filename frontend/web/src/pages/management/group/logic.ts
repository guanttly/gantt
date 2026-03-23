// 分组管理模块的业务逻辑和常量

import type { TypeOption } from './type'

/** 默认查询参数 */
export const defaultQueryParams = {
  orgId: 'default-org',
  type: undefined,
  parentId: '',
  keyword: '',
  page: 1,
  size: 20,
}

/** 类型选项 */
export const typeOptions: TypeOption[] = [
  { label: '部门', value: 'department' },
  { label: '团队', value: 'team' },
  { label: '项目', value: 'project' },
  { label: '自定义', value: 'custom' },
]

/** 获取类型标签类型 */
export function getTypeTagType(type: Group.GroupType): string {
  const map: Record<Group.GroupType, string> = {
    department: '',
    team: 'success',
    project: 'warning',
    custom: 'info',
  }
  return map[type] || 'info'
}

/** 获取类型标签文本 */
export function getTypeText(type: Group.GroupType): string {
  const map: Record<Group.GroupType, string> = {
    department: '部门',
    team: '团队',
    project: '项目',
    custom: '自定义',
  }
  return map[type] || type
}

/** 构建分组树 */
export function buildGroupTree(groups: Group.GroupInfo[]): Group.TreeNode[] {
  const map = new Map<string, Group.TreeNode>()
  const roots: Group.TreeNode[] = []

  // 第一遍：创建所有节点
  groups.forEach((group) => {
    map.set(group.id, {
      id: group.id,
      name: group.name,
      type: group.type,
      parentId: group.parentId,
      memberCount: group.memberCount,
      children: [],
    })
  })

  // 第二遍：建立父子关系
  groups.forEach((group) => {
    const node = map.get(group.id)!
    if (group.parentId && map.has(group.parentId)) {
      const parent = map.get(group.parentId)!
      if (!parent.children)
        parent.children = []
      parent.children.push(node)
    }
    else {
      roots.push(node)
    }
  })

  return roots
}
