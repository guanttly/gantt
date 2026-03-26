// 请假相关类型定义

export type LeaveType
  = 'annual'
    | 'sick'
    | 'personal'
    | 'maternity'
    | 'paternity'
    | 'marriage'
    | 'bereavement'
    | 'compensatory'
    | 'other'

export type LeaveStatus = 'pending' | 'approved' | 'rejected' | 'cancelled'

export interface Leave {
  id: string
  org_node_id: string
  employee_id: string
  employee_name?: string
  type: LeaveType
  status: LeaveStatus
  start_date: string
  end_date: string
  reason?: string
  approver_id?: string
  approver_name?: string
  approved_at?: string
  created_at: string
  updated_at: string
}

export interface CreateLeaveRequest {
  employee_id: string
  type: LeaveType
  start_date: string
  end_date: string
  reason?: string
}

export interface UpdateLeaveRequest {
  type?: LeaveType
  start_date?: string
  end_date?: string
  reason?: string
}

// ==================== 常量 ====================

export const LEAVE_TYPE_OPTIONS = [
  { label: '年假', value: 'annual' as LeaveType },
  { label: '病假', value: 'sick' as LeaveType },
  { label: '事假', value: 'personal' as LeaveType },
  { label: '产假', value: 'maternity' as LeaveType },
  { label: '陪产假', value: 'paternity' as LeaveType },
  { label: '婚假', value: 'marriage' as LeaveType },
  { label: '丧假', value: 'bereavement' as LeaveType },
  { label: '调休', value: 'compensatory' as LeaveType },
  { label: '其他', value: 'other' as LeaveType },
] as const

export function getLeaveTypeText(type: LeaveType): string {
  const opt = LEAVE_TYPE_OPTIONS.find(o => o.value === type)
  return opt?.label ?? type
}

export function getLeaveTypeTagType(type: LeaveType): string {
  const map: Record<LeaveType, string> = {
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
