import type {
  Schedule,
  ScheduleFilters,
  ScheduleSnapshot,
  TimeRange,
} from '@/types/schedule'
import type { WsEnvelope } from '@/types/ws'
import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

export const useScheduleStore = defineStore('schedule', () => {
  // 状态
  const viewRange = ref<TimeRange>({
    start: new Date().toISOString().split('T')[0], // 今天
    end: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString().split('T')[0], // 一周后
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
  })

  const draft = ref<Schedule | null>(null)
  const final = ref<Schedule | null>(null)
  const queryResult = ref<Schedule | null>(null)
  const active = ref<'draft' | 'final' | 'query'>('draft')

  const filters = ref<ScheduleFilters>({})

  // 计算属性
  const currentSchedule = computed(() => {
    switch (active.value) {
      case 'draft': return draft.value
      case 'final': return final.value
      case 'query': return queryResult.value
      default: return null
    }
  })

  const hasData = computed(() => {
    return draft.value !== null || final.value !== null || queryResult.value !== null
  })

  const conflictCount = computed(() => {
    const schedule = currentSchedule.value
    if (!schedule?.constraints)
      return 0
    return schedule.constraints.filter(c => c.level === 'error').length
  })

  const warningCount = computed(() => {
    const schedule = currentSchedule.value
    if (!schedule?.constraints)
      return 0
    return schedule.constraints.filter(c => c.level === 'warning').length
  })

  // 操作方法
  const setViewRange = (range: TimeRange) => {
    viewRange.value = { ...range }
  }

  const setFilters = (newFilters: Partial<ScheduleFilters>) => {
    filters.value = { ...filters.value, ...newFilters }
  }

  const setActive = (source: 'draft' | 'final' | 'query') => {
    active.value = source
  }

  const applyDraft = (schedule: Schedule) => {
    draft.value = { ...schedule, status: 'draft' }

    // 如果当前没有选中任何数据源，自动切换到草稿
    if (!currentSchedule.value) {
      active.value = 'draft'
    }
  }

  const applyFinal = (schedule: Schedule) => {
    final.value = { ...schedule, status: 'final' }

    // 定稿后自动切换显示
    active.value = 'final'
  }

  const applyQuery = (schedule: Schedule) => {
    queryResult.value = { ...schedule, status: 'query' }

    // 查询结果自动切换显示
    active.value = 'query'
  }

  const clearData = (type?: 'draft' | 'final' | 'query') => {
    if (type) {
      switch (type) {
        case 'draft':
          draft.value = null
          break
        case 'final':
          final.value = null
          break
        case 'query':
          queryResult.value = null
          break
      }
    }
    else {
      // 清空所有数据
      draft.value = null
      final.value = null
      queryResult.value = null
    }

    // 如果清空了当前选中的数据源，切换到有数据的源
    if (!currentSchedule.value) {
      if (draft.value)
        active.value = 'draft'
      else if (final.value)
        active.value = 'final'
      else if (queryResult.value)
        active.value = 'query'
    }
  }

  const loadSnapshot = (snapshot: ScheduleSnapshot) => {
    if (snapshot.draft) {
      draft.value = snapshot.draft
    }
    if (snapshot.final) {
      final.value = snapshot.final
    }
    if (snapshot.queryResult) {
      queryResult.value = snapshot.queryResult
    }
    if (snapshot.viewRange) {
      viewRange.value = snapshot.viewRange
    }

    // 智能选择活跃数据源
    if (snapshot.final) {
      active.value = 'final'
    }
    else if (snapshot.draft) {
      active.value = 'draft'
    }
    else if (snapshot.queryResult) {
      active.value = 'query'
    }
  }

  // WebSocket 事件处理
  const handleScheduleEvent = (envelope: WsEnvelope) => {
    try {
      const schedule = envelope.payload?.data
      if (!schedule) {
        console.warn('Schedule event without data:', envelope)
        return
      }

      switch (envelope.type) {
        case 'schedule_draft':
          applyDraft(schedule)
          break

        case 'schedule_final':
          applyFinal(schedule)
          break

        case 'schedule_query_result':
          applyQuery(schedule)
          break

        default:
          console.warn('Unknown schedule event type:', envelope.type)
      }
    }
    catch (error) {
      console.error('Error handling schedule event:', error, envelope)
    }
  }

  // 数据导出
  const exportSchedule = (format: 'csv' | 'excel' = 'csv') => {
    const schedule = currentSchedule.value
    if (!schedule) {
      throw new Error('没有可导出的排班数据')
    }

    // TODO: 实现导出逻辑
    console.log('Exporting schedule:', { format, schedule })
    return schedule
  }

  const calculateCoverage = (schedule: Schedule) => {
    // 简单的覆盖率计算示例
    const totalSlots = schedule.resources.length * 7 * 3 // 假设每天3班
    const assignedSlots = schedule.assignments.length
    return Math.round((assignedSlots / totalSlots) * 100)
  }

  // 数据统计
  const getStatistics = () => {
    const schedule = currentSchedule.value
    if (!schedule)
      return null

    const stats = {
      totalAssignments: schedule.assignments.length,
      resourceCount: schedule.resources.length,
      shiftTypes: schedule.shifts.length,
      conflicts: conflictCount.value,
      warnings: warningCount.value,
      coverage: calculateCoverage(schedule),
    }

    return stats
  }

  // 数据查询辅助方法
  const getAssignmentsByResource = (resourceId: string) => {
    const schedule = currentSchedule.value
    if (!schedule)
      return []
    return schedule.assignments.filter(a => a.resourceId === resourceId)
  }

  const getAssignmentsByShift = (shiftId: string) => {
    const schedule = currentSchedule.value
    if (!schedule)
      return []
    return schedule.assignments.filter(a => a.shiftId === shiftId)
  }

  const getAssignmentsByDateRange = (start: string, end: string) => {
    const schedule = currentSchedule.value
    if (!schedule)
      return []

    const startTime = new Date(start).getTime()
    const endTime = new Date(end).getTime()

    return schedule.assignments.filter((a) => {
      const assignmentStart = new Date(a.start).getTime()
      const assignmentEnd = new Date(a.end).getTime()

      return assignmentStart < endTime && assignmentEnd > startTime
    })
  }

  const getConflictsByResource = (resourceId: string) => {
    const schedule = currentSchedule.value
    if (!schedule?.constraints)
      return []

    return schedule.constraints.filter(c =>
      c.relatedIds.includes(resourceId) && c.level === 'error',
    )
  }

  return {
    // 状态
    viewRange,
    draft,
    final,
    queryResult,
    active,
    filters,

    // 计算属性
    currentSchedule,
    hasData,
    conflictCount,
    warningCount,

    // 方法
    setViewRange,
    setFilters,
    setActive,
    applyDraft,
    applyFinal,
    applyQuery,
    clearData,
    loadSnapshot,
    handleScheduleEvent,
    exportSchedule,
    getStatistics,

    // 查询方法
    getAssignmentsByResource,
    getAssignmentsByShift,
    getAssignmentsByDateRange,
    getConflictsByResource,
  }
})
