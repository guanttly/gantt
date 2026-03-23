// 员工管理模块的业务逻辑和常量

import type { StatusOption } from './type'

/** 默认查询参数 */
export const defaultQueryParams = {
  orgId: 'default-org',
  keyword: '',
  department: '',
  status: undefined,
  page: 1,
  size: 20,
}

/** 状态选项 */
export const statusOptions: StatusOption[] = [
  { label: '在职', value: 'active' },
  { label: '离职', value: 'inactive' },
  { label: '请假中', value: 'on_leave' },
]

/** 获取状态标签类型 */
export function getStatusType(status: Employee.EmployeeStatus): string {
  const map: Record<Employee.EmployeeStatus, string> = {
    active: 'success',
    inactive: 'info',
    on_leave: 'warning',
  }
  return map[status] || 'info'
}

/** 获取状态标签文本 */
export function getStatusText(status: Employee.EmployeeStatus): string {
  const map: Record<Employee.EmployeeStatus, string> = {
    active: '在职',
    inactive: '离职',
    on_leave: '请假中',
  }
  return map[status] || status
}
