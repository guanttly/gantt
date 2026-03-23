// 班次管理模块相关的类型定义

declare namespace Shift {
  /** 班次类型 */
  type ShiftType = 'regular' | 'normal' | 'overtime' | 'special' | 'standby' | 'fixed' | 'research' | 'fill'

  /** 班次信息 */
  interface ShiftInfo {
    id: string
    orgId: string
    name: string // 班次名称
    code: string // 班次编码
    type: string // 改为 string 以支持更多类型
    startTime: string // 开始时间 HH:mm
    endTime: string // 结束时间 HH:mm
    duration: number // 时长（分钟）
    schedulingPriority?: number // 排班优先级（用于排班排序）
    isActive: boolean
    color?: string // 用于前端展示的颜色
    description?: string
    metadata?: Record<string, any>
    createdAt: string
    updatedAt: string
    // 扩展字段（由后端附带返回）
    weeklyStaffSummary?: string // 周人数摘要，如"工作日2人/周末1人"
  }

  /** 查询班次列表参数 */
  interface ListParams {
    orgId: string
    type?: string
    isActive?: boolean
    keyword?: string
    page?: number
    size?: number
  }

  /** 班次列表数据 */
  interface ListData {
    items: ShiftInfo[]
    total: number
    page: number
    size: number
  }

  /** 创建班次请求 */
  interface CreateRequest {
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

  /** 更新班次请求 */
  interface UpdateRequest {
    orgId: string
    name?: string
    type?: string
    startTime?: string
    endTime?: string
    duration?: number
    schedulingPriority?: number // 排班优先级（用于排班排序）
    color?: string
    description?: string
    metadata?: Record<string, any>
  }

  // ==================== 班次分配 ====================

  /** 班次分配信息 */
  interface AssignmentInfo {
    id: string
    orgId: string
    employeeId: string
    shiftId: string
    date: string // YYYY-MM-DD
    employeeName?: string
    shiftName?: string
    notes?: string
    createdAt: string
    updatedAt: string
  }

  /** 查询班次分配列表参数 */
  interface AssignmentListParams {
    orgId: string
    employeeId?: string
    shiftId?: string
    startDate?: string
    endDate?: string
    page?: number
    size?: number
  }

  /** 班次分配列表数据 */
  interface AssignmentListData {
    items: AssignmentInfo[]
    total: number
    page: number
    size: number
  }

  /** 创建班次分配请求 */
  interface CreateAssignmentRequest {
    orgId: string
    employeeId: string
    shiftId: string
    date: string
    notes?: string
  }

  /** 批量删除班次分配请求 */
  interface BatchDeleteAssignmentRequest {
    orgId: string
    assignmentIds: string[]
  }

  /** 按员工查询班次分配参数 */
  interface AssignmentByEmployeeParams {
    orgId: string
    employeeId: string
    startDate: string
    endDate: string
  }

  /** 按日期范围查询班次分配参数 */
  interface AssignmentByDateRangeParams {
    orgId: string
    startDate: string
    endDate: string
  }

  // ==================== 班次-分组关联 ====================

  /** 班次-分组关联信息 */
  interface ShiftGroupInfo {
    id: number
    shiftId: string
    groupId: string
    priority: number // 优先级，数字越小优先级越高
    isActive: boolean // 是否启用
    notes?: string // 备注说明
    createdAt: string
    updatedAt: string
    // 关联信息（可选，由后端预加载）
    groupName?: string
    groupCode?: string
  }

  /** 设置班次关联分组请求 */
  interface SetShiftGroupsRequest {
    groupIds: string[]
  }

  /** 添加分组到班次请求 */
  interface AddGroupToShiftRequest {
    priority?: number
  }

  // ==================== 固定人员配置 ====================

  /** 固定人员配置模式类型 */
  type PatternType = 'weekly' | 'monthly' | 'specific'

  /** 周重复模式 */
  type WeekPattern = 'every' | 'odd' | 'even'

  /** 固定人员配置 */
  interface FixedAssignment {
    id?: string
    staffId: string
    staffName?: string
    patternType: PatternType
    weekdays?: number[]        // [1,3,5] = 周一、三、五 (1=周一, 7=周日)
    weekPattern?: WeekPattern  // every=每周, odd=奇数周, even=偶数周
    monthdays?: number[]       // [1,15,30] = 每月1号、15号、30号
    specificDates?: string[]   // ["2025-01-01"]
    startDate?: string         // YYYY-MM-DD
    endDate?: string           // YYYY-MM-DD
    isActive?: boolean
  }

  /** 批量创建固定人员配置请求 */
  interface BatchCreateFixedAssignmentsRequest {
    shiftId: string
    assignments: FixedAssignment[]
  }

  /** 固定排班计算请求 */
  interface CalculateFixedScheduleRequest {
    startDate: string
    endDate: string
  }

  /** 固定排班计算结果 */
  interface FixedScheduleResult {
    [date: string]: string[]  // 日期 -> 人员ID列表
  }
}
