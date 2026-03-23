/** 排班人数计算相关类型定义 */
declare namespace Staffing {
  /** 班次周配置信息 */
  interface WeeklyStaff {
    shiftId: string
    shiftName?: string
    /** 周日到周六的配置 */
    weeklyConfig: DayConfig[]
  }

  /** 单日配置 */
  interface DayConfig {
    weekday: number // 0=周日, 1=周一, ..., 6=周六（与后端 Go time.Weekday 一致）
    weekdayName?: string // 周日/周一/.../周六
    staffCount: number
    isCustom?: boolean // 是否自定义配置
  }

  /** 批量获取周人数配置结果 */
  interface BatchWeeklyStaffResult {
    [shiftId: string]: WeeklyStaff
  }

  /** 计算规则 */
  interface Rule {
    id: string
    shiftId: string
    shiftName?: string
    /** 关联的机房ID列表 */
    modalityRoomIds: string[]
    /** 关联的机房详情 */
    modalityRooms?: ModalityRoom.Info[]
    /** 时间段ID */
    timePeriodId: string
    /** 时间段名称 */
    timePeriodName?: string
    /** 人均报告处理上限 */
    avgReportLimit: number
    /** 取整方式: ceil=向上取整, floor=向下取整 */
    roundingMode: 'ceil' | 'floor'
    /** 是否启用 */
    isActive: boolean
    /** 规则说明 */
    description?: string
    createdAt?: string
    updatedAt?: string
  }

  /** 规则列表查询参数 */
  interface RuleListParams {
    orgId: string
    shiftId?: string
    page?: number
    pageSize?: number
  }

  /** 规则列表结果 */
  interface RuleListResult {
    items: Rule[]
    total: number
    page?: number
    pageSize?: number
  }

  /** 创建/更新规则请求 */
  interface RuleRequest {
    shiftId: string
    modalityRoomIds: string[]
    timePeriodId: string
    avgReportLimit: number
    roundingMode: 'ceil' | 'floor'
    description?: string
  }

  /** 计算预览结果 */
  interface CalculationPreview {
    shiftId: string
    shiftName: string
    timePeriodId: string
    timePeriodName: string
    modalityRooms: ModalityRoomVolumeSummary[]
    totalVolume: number
    dataDays: number
    weeklyVolume: number
    avgReportLimit: number
    roundingMode: 'ceil' | 'floor'
    calculatedCount: number
    /** 每日计算结果 */
    dailyResults: DailyStaffingResult[]
    calculationSteps: string
  }

  /** 单日排班人数计算结果 */
  interface DailyStaffingResult {
    weekday: number
    weekdayName: string
    dailyVolume: number
    calculatedCount: number
    currentCount: number
  }

  /** 机房报告量汇总 */
  interface ModalityRoomVolumeSummary {
    modalityRoomId: string
    modalityRoomName: string
    volume: number
  }

  /** 应用人数请求 */
  interface ApplyRequest {
    shiftId: string
    staffCount: number
    applyMode: 'weekly' // 只支持周配置模式
    weekdays: number[] // 要应用的星期几
  }

  /** 应用人数结果 */
  interface ApplyResult {
    shiftId: string
    appliedCount: number
    applyMode: string
    affectedDays?: number[]
    message: string
  }
}
