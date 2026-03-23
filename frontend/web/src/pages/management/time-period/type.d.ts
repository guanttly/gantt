/** 时间段页面相关类型定义 */
export interface TimePeriodFormData {
  orgId: string
  name: string
  code: string
  startTime: string
  endTime: string
  isCrossDay: boolean
  description?: string
}
