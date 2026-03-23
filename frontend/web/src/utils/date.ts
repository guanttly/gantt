/**
 * 日期工具函数
 */

/**
 * 获取本周的开始和结束日期（周一到周日）
 * @returns { start: string, end: string } 格式为 YYYY-MM-DD
 */
export function getThisWeekRange() {
  const now = new Date()
  const day = now.getDay() // 0(周日) 到 6(周六)

  // 计算本周一（如果今天是周日，则为上周一）
  const monday = new Date(now)
  const diff = day === 0 ? -6 : 1 - day // 周日特殊处理
  monday.setDate(now.getDate() + diff)
  monday.setHours(0, 0, 0, 0)

  // 计算本周日
  const sunday = new Date(monday)
  sunday.setDate(monday.getDate() + 6)
  sunday.setHours(23, 59, 59, 999)

  return {
    start: formatDate(monday),
    end: formatDate(sunday),
  }
}

/**
 * 获取下周的开始和结束日期（下周一到下周日）
 * @returns { start: string, end: string } 格式为 YYYY-MM-DD
 */
export function getNextWeekRange() {
  const today = new Date()
  const day = today.getDay() // 0(周日) 到 6(周六)

  // 计算距离下周一的天数
  // 周一=1, 周二=2, ..., 周六=6, 周日=0
  // 如果今天是周一(1)，下周一是7天后
  // 如果今天是周日(0)，下周一是1天后
  const daysUntilNextMonday = day === 0 ? 1 : (8 - day)

  const nextMonday = new Date(today)
  nextMonday.setDate(today.getDate() + daysUntilNextMonday)
  nextMonday.setHours(0, 0, 0, 0)

  const nextSunday = new Date(nextMonday)
  nextSunday.setDate(nextMonday.getDate() + 6)
  nextSunday.setHours(23, 59, 59, 999)

  return {
    start: formatDate(nextMonday),
    end: formatDate(nextSunday),
  }
}

/**
 * 获取本月的开始和结束日期
 * @returns { start: string, end: string } 格式为 YYYY-MM-DD
 */
export function getThisMonthRange() {
  const now = new Date()
  const start = new Date(now.getFullYear(), now.getMonth(), 1)
  const end = new Date(now.getFullYear(), now.getMonth() + 1, 0)

  return {
    start: formatDate(start),
    end: formatDate(end),
  }
}

/**
 * 格式化日期为 YYYY-MM-DD
 * @param date Date 对象
 * @returns 格式化后的日期字符串
 */
export function formatDate(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

/**
 * 格式化日期为中文描述 (如: 2025年01月20日)
 * @param date Date 对象或日期字符串
 * @returns 中文格式的日期字符串
 */
export function formatDateChinese(date: Date | string): string {
  const d = typeof date === 'string' ? new Date(date) : date
  const year = d.getFullYear()
  const month = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${year}年${month}月${day}日`
}

/**
 * 生成本周排班的提示词模板
 * @param department 部门（可选）
 * @param modality 检查类型（可选）
 * @returns 完整的提示词
 */
export function generateThisWeekSchedulePrompt(department?: string, modality?: string): string {
  const { start, end } = getThisWeekRange()
  const startChinese = formatDateChinese(start)
  const endChinese = formatDateChinese(end)

  let prompt = `请帮我安排本周（${startChinese}至${endChinese}）的排班`

  if (department) {
    prompt += `，部门：${department}`
  }

  if (modality) {
    prompt += `，检查类型：${modality}`
  }

  return prompt
}

/**
 * 生成下周排班的提示词模板
 * @param department 部门（可选）
 * @param modality 检查类型（可选）
 * @returns 完整的提示词
 */
export function generateNextWeekSchedulePrompt(department?: string, modality?: string): string {
  const { start, end } = getNextWeekRange()
  const startChinese = formatDateChinese(start)
  const endChinese = formatDateChinese(end)

  let prompt = `请帮我安排下周（${startChinese}至${endChinese}）的排班`

  if (department) {
    prompt += `，部门：${department}`
  }

  if (modality) {
    prompt += `，检查类型：${modality}`
  }

  return prompt
}

/**
 * 生成本月排班的提示词模板
 * @param department 部门（可选）
 * @param modality 检查类型（可选）
 * @returns 完整的提示词
 */
export function generateThisMonthSchedulePrompt(department?: string, modality?: string): string {
  const { start, end } = getThisMonthRange()
  const startChinese = formatDateChinese(start)
  const endChinese = formatDateChinese(end)

  let prompt = `请帮我安排本月（${startChinese}至${endChinese}）的排班`

  if (department) {
    prompt += `，部门：${department}`
  }

  if (modality) {
    prompt += `，检查类型：${modality}`
  }

  return prompt
}
