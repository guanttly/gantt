// 班次相关类型定义

/** 班次类型（用户可创建） */
export type ShiftType = 'regular' | 'overtime' | 'oncall' | 'standby'

/** 扩展班次类型（含工作流内部） */
export type ShiftTypeExtended = ShiftType | 'normal' | 'special' | 'fixed' | 'research' | 'fill'

export interface Shift {
  id: string
  org_node_id: string
  name: string
  code?: string
  color: string
  start_time: string // HH:mm 格式
  end_time: string // HH:mm 格式
  duration: number // 分钟
  type: ShiftTypeExtended
  scheduling_priority?: number
  is_active: boolean
  is_cross_day: boolean
  description?: string
  metadata?: Record<string, unknown>
  weekly_staff_summary?: string
  created_at: string
  updated_at: string
}

export interface CreateShiftRequest {
  name: string
  code?: string
  color: string
  start_time: string
  end_time: string
  type?: ShiftType
  scheduling_priority?: number
  description?: string
  metadata?: Record<string, unknown>
}

export interface UpdateShiftRequest {
  name?: string
  code?: string
  color?: string
  start_time?: string
  end_time?: string
  type?: ShiftType
  scheduling_priority?: number
  description?: string
  metadata?: Record<string, unknown>
}

export interface ShiftDependency {
  id: string
  shift_id: string
  depends_on_shift_id: string
  type: string
}

// ==================== 班次分组关联 ====================

export interface ShiftGroup {
  id: number
  shift_id: string
  group_id: string
  priority: number
  is_active: boolean
  notes?: string
  group_name?: string
  group_code?: string
  created_at: string
  updated_at: string
}

// ==================== 固定人员配置 ====================

export type PatternType = 'weekly' | 'monthly' | 'specific'
export type WeekPattern = 'every' | 'odd' | 'even'

export interface FixedAssignment {
  id?: string
  staff_id: string
  staff_name?: string
  pattern_type: PatternType
  weekdays?: number[]
  week_pattern?: WeekPattern
  monthdays?: number[]
  specific_dates?: string[]
  start_date?: string
  end_date?: string
  is_active?: boolean
}

// ==================== 周人数配置 ====================

export interface DayConfig {
  weekday: number
  weekday_name?: string
  staff_count: number
  is_custom?: boolean
}

export interface WeeklyStaff {
  shift_id: string
  shift_name?: string
  weekly_config: DayConfig[]
}

// ==================== 常量和辅助函数 ====================

/** 预设颜色 */
export const SHIFT_COLOR_PRESETS = [
  '#409EFF',
  '#67C23A',
  '#E6A23C',
  '#F56C6C',
  '#909399',
  '#C71585',
  '#20B2AA',
  '#FF69B4',
]

/** 班次类型选项 */
export const SHIFT_TYPE_OPTIONS = [
  { label: '常规班次', value: 'regular' },
  { label: '加班班次', value: 'overtime' },
  { label: '备班班次', value: 'standby' },
] as const

/** 所有类型选项（含工作流内部） */
export const ALL_SHIFT_TYPE_OPTIONS = [
  { label: '常规班次', value: 'regular' },
  { label: '普通班次', value: 'normal' },
  { label: '加班班次', value: 'overtime' },
  { label: '特殊班次', value: 'special' },
  { label: '备班班次', value: 'standby' },
  { label: '固定班次', value: 'fixed' },
  { label: '科研班次', value: 'research' },
  { label: '填充班次', value: 'fill' },
] as const

/** 获取班次类型标签类型 */
export function getShiftTypeTagType(type: string): string {
  const map: Record<string, string> = {
    regular: 'primary',
    normal: 'primary',
    overtime: 'warning',
    special: 'warning',
    standby: 'info',
    fixed: 'success',
    research: 'info',
    fill: 'info',
  }
  return map[type] || 'info'
}

/** 获取班次类型文本 */
export function getShiftTypeText(type: string): string {
  const map: Record<string, string> = {
    regular: '常规',
    normal: '普通',
    overtime: '加班',
    special: '特殊',
    standby: '备班',
    oncall: '值班',
    fixed: '固定',
    research: '科研',
    fill: '填充',
  }
  return map[type] || type
}

/** 计算时长（分钟） */
export function calculateDuration(startTime: string, endTime: string): number {
  if (!startTime || !endTime)
    return 0
  const [sh, sm] = startTime.split(':').map(Number)
  const [eh, em] = endTime.split(':').map(Number)
  let diff = (eh * 60 + em) - (sh * 60 + sm)
  if (diff < 0)
    diff += 24 * 60
  return diff
}

/** 格式化时长 */
export function formatDuration(minutes: number): string {
  const h = Math.floor(minutes / 60)
  const m = minutes % 60
  return `${h}小时${m > 0 ? `${m}分钟` : ''}`
}

// ==================== 星期常量 ====================

export const WEEKDAY_NAMES = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']
export const WEEKDAY_DISPLAY_ORDER = [1, 2, 3, 4, 5, 6, 0]
export const WEEKDAY_DISPLAY_NAMES = ['周一', '周二', '周三', '周四', '周五', '周六', '周日']

export function getWeekdayName(weekday: number): string {
  return WEEKDAY_NAMES[weekday] || ''
}

export function createDefaultWeeklyConfig(): DayConfig[] {
  return Array.from({ length: 7 }, (_, i) => ({
    weekday: i,
    weekday_name: WEEKDAY_NAMES[i],
    staff_count: 0,
    is_custom: false,
  }))
}
