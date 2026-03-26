// 排班相关类型定义（综合旧 schedule.ts + 新架构）

// ==================== 排班核心类型 ====================

export type ScheduleStatus = 'draft' | 'generating' | 'generated' | 'published' | 'final'

export interface SchedulePlan {
  id: string
  org_node_id: string
  name: string
  start_date: string
  end_date: string
  status: ScheduleStatus
  created_at: string
  updated_at: string
}

export interface ScheduleAssignment {
  id: string
  schedule_id: string
  employee_id: string
  employee_name: string
  shift_id: string
  shift_name: string
  shift_color: string
  date: string
  start_time: string
  end_time: string
  notes?: string
  status: 'normal' | 'adjusted' | 'conflict'
  meta?: {
    position?: string
    priority?: number
    rule_status?: 'pass' | 'warning' | 'error'
    remark?: string
    tags?: string[]
  }
}

export interface MyScheduleAssignment {
  id: string
  schedule_id: string
  schedule_name: string
  employee_id: string
  shift_id: string
  shift_name: string
  shift_color: string
  date: string
  start_time: string
  end_time: string
  source: string
  status: string
}

export interface CreateScheduleRequest {
  name: string
  start_date: string
  end_date: string
}

export interface AdjustAssignmentRequest {
  assignments: Array<{
    employee_id: string
    shift_id: string
    date: string
  }>
}

// ==================== 变更追踪 ====================

export type ScheduleChangeType = 'add' | 'modify' | 'remove'

export interface ScheduleChange {
  id: string
  type: ScheduleChangeType
  employee_id: string
  employee_name: string
  date: string
  old_shift_id?: string
  new_shift_id?: string
  reason?: string
  created_at: string
}

export interface DateChangePreview {
  date: string
  change_type: ScheduleChangeType
  before_ids: string[]
  after_ids: string[]
  before: string[]
  after: string[]
}

export interface ShiftChangePreview {
  shift_id: string
  shift_name: string
  changes: DateChangePreview[]
}

export interface ChangeDetailPreview {
  task_id: string
  task_title: string
  task_index: number
  timestamp: string
  shifts: ShiftChangePreview[]
}

// ==================== 统计与校验 ====================

export interface ScheduleSummary {
  total_employees: number
  total_days: number
  total_assignments: number
  conflicts: number
  coverage_rate: number
  stats_by_shift: Array<{
    shift_id: string
    shift_name: string
    count: number
  }>
}

export interface ValidationResult {
  valid: boolean
  violations: Array<{
    rule_id: string
    rule_name: string
    level: 'error' | 'warning'
    message: string
    related_employee_ids: string[]
    related_dates: string[]
  }>
}

export interface ConstraintResult {
  id: string
  type: 'conflict' | 'violation' | 'warning' | 'info'
  level: 'error' | 'warning' | 'info'
  message: string
  related_ids: string[]
  suggestion?: string
}

// ==================== 排班过滤器 ====================

export interface ScheduleFilters {
  dept_ids?: string[]
  resource_ids?: string[]
  shift_ids?: string[]
  status?: ('pass' | 'warning' | 'error')[]
  keywords?: string
  date_range?: [string, string]
}

// ==================== 排班快照 ====================

export interface TimeRange {
  start: string
  end: string
  timezone: string
}

export interface ScheduleSnapshot {
  draft?: SchedulePlan
  final?: SchedulePlan
  query_result?: SchedulePlan
  view_range?: TimeRange
}

// ==================== 班次失败与重试 ====================

export interface ShiftFailureInfo {
  shift_id: string
  shift_name: string
  failure_summary: string
  auto_retry_count: number
  manual_retry_count: number
  failure_history?: string[]
  last_error?: string
  validation_issues?: Array<{
    type: string
    description: string
    severity: string
  }>
}

export interface TaskResult {
  task_id: string
  success: boolean
  partially_succeeded?: boolean
  successful_shifts?: string[]
  failed_shifts?: Record<string, ShiftFailureInfo>
  error?: string
  execution_time?: number
  metadata?: Record<string, unknown>
}

// ==================== 班次进度 ====================

export interface ShiftProgressInfo {
  shift_id: string
  shift_name: string
  status: 'pending' | 'running' | 'success' | 'failed' | 'skipped'
  progress: number
  message?: string
  error?: string
}
