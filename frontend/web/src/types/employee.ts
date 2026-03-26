// 员工相关类型定义

export type EmployeeStatus = 'active' | 'inactive' | 'on_leave'

export interface Employee {
  id: string
  org_node_id: string
  name: string
  employee_no?: string
  phone?: string
  email?: string
  department_id?: string
  department?: {
    id: string
    name: string
    code: string
  }
  position?: string
  status: EmployeeStatus
  hire_date?: string
  skills?: string[]
  groups?: Array<{
    id: string
    code: string
    name: string
    type: string
  }>
  metadata?: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface CreateEmployeeRequest {
  name: string
  employee_no?: string
  phone?: string
  email?: string
  department_id?: string
  position?: string
  hire_date?: string
  skills?: string[]
  metadata?: Record<string, unknown>
}

export interface UpdateEmployeeRequest {
  name?: string
  employee_no?: string
  phone?: string
  email?: string
  department_id?: string
  position?: string
  status?: EmployeeStatus
  hire_date?: string
  skills?: string[]
  metadata?: Record<string, unknown>
}

export interface BatchUpdateStatusRequest {
  employee_ids: string[]
  status: EmployeeStatus
}

/** 员工状态选项 */
export const EMPLOYEE_STATUS_OPTIONS = [
  { label: '在职', value: 'active' as EmployeeStatus },
  { label: '离职', value: 'inactive' as EmployeeStatus },
  { label: '请假中', value: 'on_leave' as EmployeeStatus },
] as const

/** 获取员工状态标签类型 */
export function getEmployeeStatusType(status: EmployeeStatus): string {
  const map: Record<EmployeeStatus, string> = {
    active: 'success',
    inactive: 'info',
    on_leave: 'warning',
  }
  return map[status] || 'info'
}

/** 获取员工状态文本 */
export function getEmployeeStatusText(status: EmployeeStatus): string {
  const map: Record<EmployeeStatus, string> = {
    active: '在职',
    inactive: '离职',
    on_leave: '请假中',
  }
  return map[status] || status
}
