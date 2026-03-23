// 员工管理模块相关的类型定义

declare namespace Employee {
  /** 员工状态 */
  type EmployeeStatus = 'active' | 'inactive' | 'on_leave'

  /** 员工信息 */
  interface EmployeeInfo {
    id: string
    orgId: string
    employeeId: string // 工号
    name: string // 姓名
    email?: string
    phone?: string
    departmentId?: string // 部门ID
    department?: {
      id: string
      name: string
      code: string
    } // 部门详情（从后端查询时返回）
    position?: string // 职位
    status: EmployeeStatus
    hireDate?: string // 入职日期 YYYY-MM-DD
    groups?: Array<{
      id: string
      code: string
      name: string
      type: string
    }> // 所属分组列表
    metadata?: Record<string, any> // 额外元数据
    createdAt: string
    updatedAt: string
  }

  /** 查询员工列表参数 */
  interface ListParams {
    orgId: string
    keyword?: string
    department?: string
    status?: EmployeeStatus
    page?: number
    size?: number
    includeGroups?: boolean // 是否加载分组信息（默认 false，仅在员工管理页面需要时设为 true）
  }

  /** 员工列表数据 */
  interface ListData {
    items: EmployeeInfo[]
    total: number
    page: number
    size: number
  }

  /** 简单员工信息（不包含分组和部门详情） */
  interface SimpleEmployeeInfo {
    id: string
    orgId: string
    employeeId: string // 工号
    name: string // 姓名
    email?: string
    phone?: string
    departmentId?: string // 部门ID（仅ID，不包含详情）
    position?: string // 职位
    status: EmployeeStatus
  }

  /** 简单查询员工列表参数 */
  interface SimpleListParams {
    orgId: string
    keyword?: string
    status?: EmployeeStatus
    page?: number
    size?: number
  }

  /** 简单员工列表数据 */
  interface SimpleListData {
    items: SimpleEmployeeInfo[]
    total: number
    page: number
    size: number
  }

  /** 创建员工请求 */
  interface CreateRequest {
    orgId: string
    employeeId: string // 工号
    name: string
    email?: string
    phone?: string
    department?: string
    position?: string
    hireDate?: string
    metadata?: Record<string, any>
  }

  /** 更新员工请求 */
  interface UpdateRequest {
    orgId: string
    name?: string
    email?: string
    phone?: string
    department?: string
    position?: string
    status?: EmployeeStatus
    metadata?: Record<string, any>
  }

  /** 批量更新员工状态请求 */
  interface BatchUpdateStatusRequest {
    orgId: string
    employeeIds: string[]
    status: EmployeeStatus
  }
}
