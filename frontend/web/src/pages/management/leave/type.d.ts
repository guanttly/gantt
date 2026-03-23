// 请假管理模块相关的类型定义

/** 假期查询表单 */
export interface LeaveQueryForm {
  orgId: string
  employeeId: string
  type: Leave.LeaveType | undefined
  startDate: string
  endDate: string
  page: number
  size: number
}

/** 假期表单 */
export interface LeaveFormData {
  orgId: string
  employeeId: string
  type: Leave.LeaveType
  startDate: string
  endDate: string
  days: number
  reason?: string
}
