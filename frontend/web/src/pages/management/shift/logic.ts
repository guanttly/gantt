// 班次管理模块的业务逻辑和常量

import type { TypeOption } from './type'

/** 默认查询参数 */
export const defaultQueryParams = {
  orgId: 'default-org',
  type: undefined,
  isActive: undefined,
  keyword: '',
  page: 1,
  size: 20,
}

/** 类型选项 - 用户可创建的班次类型 */
export const typeOptions: TypeOption[] = [
  { label: '常规班次', value: 'regular' },
  { label: '加班班次', value: 'overtime' },
  { label: '备班班次', value: 'standby' },
]

/** 所有类型选项 - 用于筛选和显示（包含工作流内部类型） */
export const allTypeOptions: TypeOption[] = [
  { label: '常规班次', value: 'regular' },
  { label: '普通班次', value: 'normal' },
  { label: '加班班次', value: 'overtime' },
  { label: '特殊班次', value: 'special' },
  { label: '备班班次', value: 'standby' },
  { label: '固定班次', value: 'fixed' },
  { label: '科研班次', value: 'research' },
  { label: '填充班次', value: 'fill' },
]

/** 预设颜色 */
export const colorPresets = [
  '#409EFF', // 蓝色
  '#67C23A', // 绿色
  '#E6A23C', // 橙色
  '#F56C6C', // 红色
  '#909399', // 灰色
  '#C71585', // 紫红色
  '#20B2AA', // 青色
  '#FF69B4', // 粉色
]

// ==================== Weekday 枚举和常量 ====================

/** 星期枚举（与 Go time.Weekday 一致：0=周日, 1=周一, ..., 6=周六） */
export enum Weekday {
  Sunday = 0,
  Monday = 1,
  Tuesday = 2,
  Wednesday = 3,
  Thursday = 4,
  Friday = 5,
  Saturday = 6,
}

/** 星期中文名称数组（索引对应 Weekday 枚举值：0=周日, 1=周一, ..., 6=周六） */
export const WEEKDAY_NAMES = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']

/** 星期展示顺序数组（从周一开始：周一, 周二, ..., 周日） */
export const WEEKDAY_DISPLAY_ORDER = [1, 2, 3, 4, 5, 6, 0] // 对应 Weekday.Monday 到 Weekday.Sunday

/** 星期展示名称数组（从周一开始） */
export const WEEKDAY_DISPLAY_NAMES = ['周一', '周二', '周三', '周四', '周五', '周六', '周日']

/** 获取星期中文名称 */
export function getWeekdayName(weekday: number): string {
  return WEEKDAY_NAMES[weekday] || ''
}

/** 获取星期展示名称（从周一开始的顺序） */
export function getWeekdayDisplayName(displayIndex: number): string {
  if (displayIndex >= 0 && displayIndex < WEEKDAY_DISPLAY_ORDER.length) {
    const weekday = WEEKDAY_DISPLAY_ORDER[displayIndex]
    return WEEKDAY_NAMES[weekday] || ''
  }
  return ''
}

/** 将展示索引转换为 weekday 值 */
export function displayIndexToWeekday(displayIndex: number): number {
  if (displayIndex >= 0 && displayIndex < WEEKDAY_DISPLAY_ORDER.length) {
    return WEEKDAY_DISPLAY_ORDER[displayIndex]
  }
  return displayIndex
}

/** 将 weekday 值转换为展示索引 */
export function weekdayToDisplayIndex(weekday: number): number {
  return WEEKDAY_DISPLAY_ORDER.indexOf(weekday)
}

/** 判断是否为周末（周六或周日） */
export function isWeekend(weekday: number): boolean {
  return weekday === Weekday.Sunday || weekday === Weekday.Saturday
}

/** 获取工作日列表（周一到周五） */
export function getWorkdays(): number[] {
  return [Weekday.Monday, Weekday.Tuesday, Weekday.Wednesday, Weekday.Thursday, Weekday.Friday]
}

/** 获取周末列表（周六和周日） */
export function getWeekends(): number[] {
  return [Weekday.Sunday, Weekday.Saturday]
}

/**
 * 创建默认的7天周配置（全部为0）
 * 注意：数组索引对应 weekday 值（0=周日, 1=周一, ..., 6=周六），与后端数据对应
 * 展示时需要使用 WEEKDAY_DISPLAY_ORDER 重新排序
 */
export function createDefaultWeeklyConfig(): Staffing.DayConfig[] {
  return Array.from({ length: 7 }, (_, i) => ({
    weekday: i,
    weekdayName: WEEKDAY_NAMES[i],
    staffCount: 0,
    isCustom: false,
  }))
}

/** 获取按展示顺序排序的周配置（从周一开始） */
export function getWeeklyConfigInDisplayOrder(config: Staffing.DayConfig[]): Staffing.DayConfig[] {
  return WEEKDAY_DISPLAY_ORDER.map((weekday) => {
    const item = config.find(c => c.weekday === weekday)
    return item || {
      weekday,
      weekdayName: WEEKDAY_NAMES[weekday],
      staffCount: 0,
      isCustom: false,
    }
  })
}

// ==================== 类型说明 ====================
//
// 班次类型分为两类：
// 1. 用户可创建的类型（typeOptions）：regular, overtime, standby
//    - 这些是用户在创建/编辑班次时可以选择的类型
//
// 2. 所有类型（allTypeOptions）：包含上述类型 + 工作流内部类型
//    - normal, special：由后端映射生成（regular→normal, overtime/standby→special）
//    - fixed, research, fill：工作流内部使用的类型
//    - 用于列表显示和筛选，但不可直接创建

// ==================== 辅助函数 ====================

/** 获取类型标签类型 */
export function getTypeTagType(type: string): 'primary' | 'success' | 'info' | 'warning' | 'danger' {
  const map: Record<string, 'primary' | 'success' | 'info' | 'warning' | 'danger'> = {
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

/** 获取类型标签文本 */
export function getTypeText(type: string): string {
  const map: Record<string, string> = {
    regular: '常规',
    normal: '普通',
    overtime: '加班',
    special: '特殊',
    standby: '备班',
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

  const [startHour, startMinute] = startTime.split(':').map(Number)
  const [endHour, endMinute] = endTime.split(':').map(Number)

  const startMinutes = startHour * 60 + startMinute
  let endMinutes = endHour * 60 + endMinute

  // 如果结束时间小于开始时间，说明跨天了
  if (endMinutes < startMinutes) {
    endMinutes += 24 * 60
  }

  return endMinutes - startMinutes
}

/** 格式化时长显示 */
export function formatDuration(minutes: number): string {
  const hours = Math.floor(minutes / 60)
  const mins = minutes % 60
  return `${hours}小时${mins > 0 ? `${mins}分钟` : ''}`
}

// ==================== 固定人员配置相关 ====================

/** 星期选项（用于固定人员配置，1=周一, 7=周日） */
export const fixedAssignmentWeekdayOptions = [
  { label: '周一', value: 1 },
  { label: '周二', value: 2 },
  { label: '周三', value: 3 },
  { label: '周四', value: 4 },
  { label: '周五', value: 5 },
  { label: '周六', value: 6 },
  { label: '周日', value: 7 },
]

/** 周重复模式选项 */
export const weekPatternOptions = [
  { label: '每周', value: 'every' as Shift.WeekPattern, desc: '每周都执行' },
  { label: '奇数周', value: 'odd' as Shift.WeekPattern, desc: '第1、3、5、7...周' },
  { label: '偶数周', value: 'even' as Shift.WeekPattern, desc: '第2、4、6、8...周' },
]

/** 获取每月日期选项（1-31号） */
export function getMonthdayOptions() {
  return Array.from({ length: 31 }, (_, i) => ({
    label: `${i + 1}号`,
    value: i + 1,
  }))
}

/** 格式化模式类型文本 */
export function formatPatternTypeText(patternType: Shift.PatternType): string {
  const map: Record<Shift.PatternType, string> = {
    weekly: '按周重复',
    monthly: '按月重复',
    specific: '指定日期',
  }
  return map[patternType] || patternType
}

/** 格式化固定人员配置规则文本 */
export function formatFixedAssignmentPattern(assignment: Shift.FixedAssignment): string {
  if (assignment.patternType === 'weekly' && assignment.weekdays) {
    const days = assignment.weekdays.map((d) => {
      const day = fixedAssignmentWeekdayOptions.find(opt => opt.value === d)
      return day ? day.label : `周${d}`
    }).join('、')

    // 添加周期信息
    let pattern = ''
    if (assignment.weekPattern === 'odd') {
      pattern = '（奇数周）'
    }
    else if (assignment.weekPattern === 'even') {
      pattern = '（偶数周）'
    }

    return days + pattern
  }
  else if (assignment.patternType === 'monthly' && assignment.monthdays) {
    const days = assignment.monthdays.sort((a, b) => a - b).map(d => `${d}号`).join('、')
    return days
  }
  else if (assignment.patternType === 'specific' && assignment.specificDates) {
    return `${assignment.specificDates.length}个日期`
  }
  return '-'
}
