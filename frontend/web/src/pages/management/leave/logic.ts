// 假期管理模块的业务逻辑和常量

/** 默认查询参数 */
export const defaultQueryParams = {
  orgId: 'default-org',
  employeeId: '',
  type: undefined,
  startDate: '',
  endDate: '',
  page: 1,
  size: 20,
}

/** 假期类型选项 */
export const leaveTypeOptions = [
  { label: '年假', value: 'annual' },
  { label: '病假', value: 'sick' },
  { label: '事假', value: 'personal' },
  { label: '产假', value: 'maternity' },
  { label: '陪产假', value: 'paternity' },
  { label: '婚假', value: 'marriage' },
  { label: '丧假', value: 'bereavement' },
  { label: '调休', value: 'compensatory' },
  { label: '其他', value: 'other' },
]

/** 获取假期类型文本 */
export function getLeaveTypeText(type: Leave.LeaveType): string {
  const option = leaveTypeOptions.find(o => o.value === type)
  return option ? option.label : type
}

/** 获取假期类型标签类型 */
export function getLeaveTypeTagType(type: Leave.LeaveType): string {
  const map: Record<Leave.LeaveType, string> = {
    annual: 'success',
    sick: 'warning',
    personal: 'info',
    maternity: 'danger',
    paternity: 'primary',
    marriage: '',
    bereavement: '',
    compensatory: 'success',
    other: 'info',
  }
  return map[type] || 'info'
}
