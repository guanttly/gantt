// 班次管理模块相关的类型定义

/** 班次查询表单 */
export interface ShiftQueryForm {
  orgId: string
  type: string | undefined
  isActive: boolean | undefined
  keyword: string
  page: number
  size: number
}

/** 班次表单 */
export interface ShiftFormData {
  orgId: string
  name: string
  code: string
  type: string
  startTime: string
  endTime: string
  duration: number
  schedulingPriority?: number // 排班优先级（用于排班排序）
  color?: string
  description?: string
  metadata?: Record<string, any>
}

/** 类型选项 */
export interface TypeOption {
  label: string
  value: string
}

/** 班次分配表单 */
export interface ShiftAssignmentForm {
  orgId: string
  employeeId: string
  shiftId: string
  date: string
  notes?: string
}

/** 固定人员配置表单 */
export interface FixedAssignmentForm {
  staffId: string
  patternType: Shift.PatternType
  weekdays: number[]
  weekPattern: Shift.WeekPattern
  monthdays: number[]
  specificDates: string[]
  dateRange: [string, string] | null
}
