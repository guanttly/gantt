// 排班相关类型定义
export interface Schedule {
  id: string
  version: number
  status: 'draft' | 'final' | 'query'
  timeRange: TimeRange
  resources: Resource[]
  shifts: Shift[]
  assignments: Assignment[]
  constraints?: ConstraintResult[]
  createdAt: string
  updatedAt: string
}

export interface TimeRange {
  start: string // ISO 日期字符串
  end: string // ISO 日期字符串
  timezone: string
}

export interface Resource {
  id: string
  name: string
  deptId: string
  deptName: string
  role: string
  capacity?: number
  skills?: string[]
  preferences?: ResourcePreference[]
}

export interface ResourcePreference {
  type: 'preferred_shifts' | 'avoid_shifts' | 'max_hours' | 'rest_days'
  value: any
}

export interface Shift {
  id: string
  name: string
  color: string
  startOffset: number // 相对于日期的小时偏移，如 8 表示 8:00
  endOffset: number // 相对于日期的小时偏移，如 18 表示 18:00
  duration: number // 班次时长（小时）
  type: 'regular' | 'overtime' | 'oncall'
}

export interface Assignment {
  id: string
  resourceId: string
  shiftId: string
  start: string // ISO 日期时间字符串
  end: string // ISO 日期时间字符串
  meta?: {
    position?: string
    priority?: number
    ruleStatus?: 'pass' | 'warning' | 'error'
    remark?: string
    tags?: string[]
  }
}

export interface ConstraintResult {
  id: string
  type: 'conflict' | 'violation' | 'warning' | 'info'
  level: 'error' | 'warning' | 'info'
  message: string
  relatedIds: string[] // 相关的 assignment 或 resource IDs
  suggestion?: string
}

export interface ScheduleFilters {
  deptIds?: string[]
  resourceIds?: string[]
  shiftIds?: string[]
  status?: ('pass' | 'warning' | 'error')[]
  keywords?: string
  dateRange?: [string, string]
}

// API 相关
export interface ScheduleSnapshot {
  draft?: Schedule
  final?: Schedule
  queryResult?: Schedule
  viewRange?: TimeRange
}

export interface ScheduleQueryOptions {
  timeRange: TimeRange
  filters?: ScheduleFilters
  includeConstraints?: boolean
}
// ============================================================
// 变更追踪系统类型定义
// ============================================================

/**
 * 排班变更类型
 */
export type ScheduleChangeType = 'add' | 'modify' | 'remove'

/**
 * 日期变更预览（单个日期的变更详情）
 */
export interface DateChangePreview {
  /** 日期 (YYYY-MM-DD) */
  date: string
  /** 变更类型 */
  changeType: ScheduleChangeType
  /** 变更前人员ID列表 */
  beforeIds: string[]
  /** 变更后人员ID列表 */
  afterIds: string[]
  /** 变更前姓名列表（用于展示） */
  before: string[]
  /** 变更后姓名列表（用于展示） */
  after: string[]
}

/**
 * 班次变更预览（单个班次的所有变更）
 */
export interface ShiftChangePreview {
  /** 班次ID */
  shiftId: string
  /** 班次名称 */
  shiftName: string
  /** 该班次下的所有日期变更 */
  changes: DateChangePreview[]
}

/**
 * 变更详情预览（整个任务的变更数据）
 */
export interface ChangeDetailPreview {
  /** 任务ID */
  taskId: string
  /** 任务标题 */
  taskTitle: string
  /** 任务序号（从1开始） */
  taskIndex: number
  /** 时间戳 */
  timestamp: string
  /** 班次变更列表（使用数组结构） */
  shifts: ShiftChangePreview[]
}

/**
 * 单条排班变更记录（旧版兼容，保留用于其他场景）
 * @deprecated 推荐使用 DateChangePreview
 */
export interface ScheduleChange {
  /** 变更类型 */
  changeType: ScheduleChangeType
  /** 班次ID */
  shiftId: string
  /** 班次名称 */
  shiftName: string
  /** 日期 (YYYY-MM-DD) */
  date: string
  /** 变更前人员ID列表 */
  beforeIds: string[]
  /** 变更后人员ID列表 */
  afterIds: string[]
  /** 变更前姓名列表（用于展示） */
  beforeNames: string[]
  /** 变更后姓名列表（用于展示） */
  afterNames: string[]
}

/**
 * 变更预览数据（旧版兼容，使用嵌套Record）
 * @deprecated 推荐使用 ChangeDetailPreview
 */
export interface ChangePreview {
  /** 任务ID */
  taskId: string
  /** 任务标题 */
  taskTitle: string
  /** 任务序号（从1开始） */
  taskIndex: number
  /** 按班次和日期组织的变更数据 */
  shifts: Record<string, Record<string, ScheduleChange>>
  /** 时间戳 */
  timestamp: string
}

/**
 * 变更批次统计信息
 */
export interface ChangeBatchStats {
  /** 新增数量 */
  addCount: number
  /** 修改数量 */
  modifyCount: number
  /** 删除数量 */
  removeCount: number
  /** 涉及的班次数 */
  affectedShiftsCount: number
  /** 涉及的日期数 */
  affectedDatesCount: number
  /** 总人次 */
  totalStaffSlots: number
}

// ============================================================
// 班次失败与重试相关类型定义
// ============================================================

/**
 * 班次失败信息
 */
export interface ShiftFailureInfo {
  /** 班次ID */
  shiftId: string
  /** 班次名称 */
  shiftName: string
  /** 失败摘要（简要描述） */
  failureSummary: string
  /** 自动重试次数 */
  autoRetryCount: number
  /** 手动重试次数 */
  manualRetryCount: number
  /** 历史失败记录（语义化描述列表） */
  failureHistory?: string[]
  /** 最后一次错误信息 */
  lastError?: string
  /** 校验问题列表 */
  validationIssues?: Array<{
    type: string
    description: string
    severity: string
  }>
}

/**
 * 任务执行结果
 */
export interface TaskResult {
  /** 任务ID */
  taskId: string
  /** 是否成功 */
  success: boolean
  /** 是否部分成功（有班次失败但有班次成功） */
  partiallySucceeded?: boolean
  /** 成功执行的班次ID列表 */
  successfulShifts?: string[]
  /** 失败的班次详细信息 */
  failedShifts?: Record<string, ShiftFailureInfo>
  /** 错误信息（如果有） */
  error?: string
  /** 执行时间（秒） */
  executionTime?: number
  /** 元数据 */
  metadata?: Record<string, any>
}

/**
 * 部分成功消息数据（结构化消息）
 */
export interface PartialSuccessMessageData {
  /** 消息类型标识 */
  type: 'partial_success'
  /** 成功班次数量 */
  successCount: number
  /** 失败班次数量 */
  failedCount: number
  /** 失败班次详情 */
  failedShifts: ShiftFailureInfo[]
  /** 可选操作列表 */
  options: Array<{
    /** 操作标识 */
    action: 'retry' | 'skip' | 'cancel'
    /** 操作显示文本 */
    label: string
    /** 对应的事件类型 */
    event: string
  }>
}
