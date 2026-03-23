// 排班管理类型定义

declare namespace Scheduling {
  /** 排班分配项 */
  interface AssignmentItem {
    employeeId: string
    shiftId: string
    date: string // YYYY-MM-DD
    notes?: string
  }

  /** 排班分配信息（完整） */
  interface Assignment {
    id: string
    orgId: string
    employeeId: string
    shiftId: string
    date: string // YYYY-MM-DD
    notes?: string
    employeeName?: string
    employeeCode?: string
    shiftName?: string
    shiftCode?: string
    shiftColor?: string
    createdAt: string
    updatedAt: string
  }

  /** 批量分配请求 */
  interface BatchAssignRequest {
    orgId: string
    assignments: AssignmentItem[]
  }

  /** 查询参数 */
  interface QueryParams {
    orgId: string
    startDate: string // YYYY-MM-DD
    endDate: string // YYYY-MM-DD
  }

  /** 员工查询参数 */
  interface EmployeeQueryParams extends QueryParams {
    employeeId: string
  }

  /** 删除参数 */
  interface DeleteParams {
    orgId: string
    id: string // assignment ID
  }

  /** 批量删除请求 */
  interface BatchDeleteRequest {
    orgId: string
    employeeIds: string[]
    dates: string[] // YYYY-MM-DD
  }

  /** 排班汇总 */
  interface Summary {
    totalAssignments: number
    dateRange: {
      start: string
      end: string
    }
    byShift: Record<string, number> // shiftId -> count
    byDate: Record<string, number> // date -> count
    byEmployee: Record<string, number> // employeeId -> count
    uniqueEmployees: number
    uniqueShifts: number
  }

  /** 甘特图数据项 */
  interface GanttItem {
    id: string
    group: string // employeeId
    content: string // shift name
    start: Date
    end: Date
    className?: string
    style?: string
    title?: string
    editable?: boolean
  }

  /** 甘特图分组 */
  interface GanttGroup {
    id: string
    content: string
    order?: number
  }

  /** 日期范围 */
  interface DateRange {
    start: Date
    end: Date
  }
}
