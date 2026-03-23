// 分组管理模块相关的类型定义

declare namespace Group {
  /** 分组类型 */
  type GroupType = 'department' | 'team' | 'project' | 'custom'

  /** 分组信息 */
  interface GroupInfo {
    id: string
    orgId: string
    code: string // 分组编码
    name: string // 分组名称
    type: GroupType
    parentId?: string // 父分组ID
    description?: string
    memberCount?: number // 成员数量
    metadata?: Record<string, any>
    createdAt: string
    updatedAt: string
  }

  /** 查询分组列表参数 */
  interface ListParams {
    orgId: string
    type?: GroupType
    parentId?: string
    keyword?: string
    page?: number
    size?: number
  }

  /** 分组列表数据 */
  interface ListData {
    items: GroupInfo[]
    total: number
    page: number
    size: number
  }

  /** 创建分组请求 */
  interface CreateRequest {
    orgId: string
    code: string
    name: string
    type: GroupType
    parentId?: string
    description?: string
    metadata?: Record<string, any>
  }

  /** 更新分组请求 */
  interface UpdateRequest {
    orgId: string
    name?: string
    type?: GroupType
    parentId?: string
    description?: string
    metadata?: Record<string, any>
  }

  /** 分组树节点 */
  interface TreeNode {
    id: string
    name: string
    type: GroupType
    parentId?: string
    children?: TreeNode[]
    memberCount?: number
  }

  // ==================== 分组成员 ====================

  /** 分组成员信息（实际返回的是 Employee 对象） */
  type MemberInfo = Employee.EmployeeInfo

  /** 查询分组成员列表参数 */
  interface MemberListParams {
    orgId: string
    groupId: string
    page?: number
    size?: number
  }

  /** 分组成员列表数据 */
  interface MemberListData {
    items: MemberInfo[]
    total: number
    page: number
    size: number
  }

  /** 添加分组成员请求 */
  interface AddMemberRequest {
    orgId: string
    groupId: string
    employeeId: string
  }

  /** 移除分组成员请求 */
  interface RemoveMemberRequest {
    orgId: string
    groupId: string
    employeeId: string
  }

  /** 批量添加分组成员请求 */
  interface BatchAddMembersRequest {
    orgId: string
    groupId: string
    employeeIds: string[]
  }
}
