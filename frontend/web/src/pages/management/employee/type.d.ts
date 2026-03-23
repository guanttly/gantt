// 员工管理模块相关的类型定义

/** 员工查询表单 */
export interface EmployeeQueryForm {
  orgId: string
  keyword: string
  department: string
  status: Employee.EmployeeStatus | undefined
  page: number
  size: number
}

/** 员工表单 */
export interface EmployeeFormData {
  orgId: string
  employeeId: string
  name: string
  email?: string
  phone?: string
  department?: string
  position?: string
  hireDate?: string
  status?: Employee.EmployeeStatus
  metadata?: Record<string, any>
}

/** 状态选项 */
export interface StatusOption {
  label: string
  value: Employee.EmployeeStatus
}
