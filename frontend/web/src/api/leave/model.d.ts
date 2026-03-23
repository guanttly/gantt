// 假期管理模块相关的类型定义

declare namespace Leave {
  /** 假期类型 */
  type LeaveType = 'annual' | 'sick' | 'personal' | 'maternity' | 'paternity' | 'marriage' | 'bereavement' | 'compensatory' | 'other'

  /** 假期信息 */
  interface LeaveInfo {
    id: string
    orgId: string
    employeeId: string
    employeeName?: string
    type: LeaveType
    startDate: string // YYYY-MM-DD
    endDate: string // YYYY-MM-DD
    startTime?: string // HH:MM（小时级请假）
    endTime?: string // HH:MM（小时级请假）
    days: number // 请假天数
    reason?: string
    createdAt: string
    updatedAt: string
  }

  /** 查询假期列表参数 */
  interface ListParams {
    orgId: string
    employeeId?: string
    keyword?: string // 员工姓名或工号搜索
    type?: LeaveType
    startDate?: string
    endDate?: string
    page?: number
    size?: number
  }

  /** 假期列表数据 */
  interface ListData {
    items: LeaveInfo[]
    total: number
    page: number
    size: number
  }

  /** 创建假期申请请求 */
  interface CreateRequest {
    orgId: string
    employeeId: string
    type: LeaveType
    startDate: string // YYYY-MM-DD
    endDate: string // YYYY-MM-DD
    startTime?: string // HH:MM（小时级请假）
    endTime?: string // HH:MM（小时级请假）
    reason?: string
  }

  /** 更新假期申请请求 */
  interface UpdateRequest {
    orgId: string
    startDate?: string // YYYY-MM-DD
    endDate?: string // YYYY-MM-DD
    startTime?: string // HH:MM
    endTime?: string // HH:MM
    reason?: string
  }

  // ==================== 假期余额 ====================

  /** 假期余额信息 */
  interface BalanceInfo {
    orgId: string
    employeeId: string
    type: LeaveType
    total: number // 总额度
    used: number // 已使用
    remaining: number // 剩余
    year: number // 年份
  }

  /** 查询假期余额参数 */
  interface BalanceParams {
    orgId: string
    employeeId: string
    type: LeaveType
    year?: number
  }

  /** 初始化假期余额请求 */
  interface InitBalanceRequest {
    orgId: string
    employeeId: string
    type: LeaveType
    total: number
    year: number
  }
}
