// 分组管理模块相关的类型定义

/** 分组查询表单 */
export interface GroupQueryForm {
  orgId: string
  type: Group.GroupType | undefined
  parentId: string
  keyword: string
  page: number
  size: number
}

/** 分组表单 */
export interface GroupFormData {
  orgId: string
  code: string
  name: string
  type: Group.GroupType
  parentId?: string
  description?: string
  metadata?: Record<string, any>
}

/** 类型选项 */
export interface TypeOption {
  label: string
  value: Group.GroupType
}

/** 成员选择表单 */
export interface MemberSelectForm {
  groupId: string
  employeeIds: string[]
}
