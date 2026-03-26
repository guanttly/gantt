// 排班数据状态管理 — 草稿/定稿/查询
import type {
  ConstraintResult,
  Schedule,
  ScheduleFilters,
  ScheduleSnapshot,
  TimeRange,
} from '@/types/schedule'
import type { WsEnvelope } from '@/types/ws'
import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

export const useScheduleStore = defineStore('schedule', () => {
  // ==================== 状态 ====================

  const viewRange = ref<TimeRange>({
    start: new Date().toISOString().split('T')[0],
    end: new Date(Date.now() + 7 * 86400000).toISOString().split('T')[0],
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
  })

  const draft = ref<Schedule | null>(null)
  const final = ref<Schedule | null>(null)
  const queryResult = ref<Schedule | null>(null)
  const active = ref<'draft' | 'final' | 'query'>('draft')
  const filters = ref<ScheduleFilters>({})

  // ==================== 计算属性 ====================

  const currentSchedule = computed(() => {
    switch (active.value) {
      case 'draft': return draft.value
      case 'final': return final.value
      case 'query': return queryResult.value
      default: return null
    }
  })

  const hasData = computed(() =>
    draft.value !== null || final.value !== null || queryResult.value !== null,
  )

  const conflictCount = computed(() => {
    const s = currentSchedule.value
    if (!s?.constraints)
      return 0
    return s.constraints.filter((c: ConstraintResult) => c.level === 'error').length
  })

  const warningCount = computed(() => {
    const s = currentSchedule.value
    if (!s?.constraints)
      return 0
    return s.constraints.filter((c: ConstraintResult) => c.level === 'warning').length
  })

  // ==================== 方法 ====================

  function setViewRange(range: TimeRange) {
    viewRange.value = { ...range }
  }

  function setFilters(newFilters: Partial<ScheduleFilters>) {
    filters.value = { ...filters.value, ...newFilters }
  }

  function setActive(source: 'draft' | 'final' | 'query') {
    active.value = source
  }

  function applyDraft(schedule: Schedule) {
    draft.value = { ...schedule, status: 'draft' }
    if (!currentSchedule.value)
      active.value = 'draft'
  }

  function applyFinal(schedule: Schedule) {
    final.value = { ...schedule, status: 'final' }
    active.value = 'final'
  }

  function applyQuery(schedule: Schedule) {
    queryResult.value = { ...schedule, status: 'query' }
    active.value = 'query'
  }

  function clearData(type?: 'draft' | 'final' | 'query') {
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
      draft.value = null
      final.value = null
      queryResult.value = null
    }

    // 切换到有数据的源
    if (!currentSchedule.value) {
      if (draft.value)
        active.value = 'draft'
      else if (final.value)
        active.value = 'final'
      else if (queryResult.value)
        active.value = 'query'
    }
  }

  function loadSnapshot(snapshot: ScheduleSnapshot) {
    if (snapshot.draft)
      draft.value = snapshot.draft
    if (snapshot.final)
      final.value = snapshot.final
    if (snapshot.queryResult)
      queryResult.value = snapshot.queryResult
    if (snapshot.viewRange)
      viewRange.value = snapshot.viewRange

    // 智能选择活跃数据源
    if (snapshot.final)
      active.value = 'final'
    else if (snapshot.draft)
      active.value = 'draft'
    else if (snapshot.queryResult)
      active.value = 'query'
  }

  // ==================== WebSocket 事件处理 ====================

  function handleScheduleEvent(envelope: WsEnvelope) {
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

  // ==================== 统计 ====================

  function getStatistics() {
    const schedule = currentSchedule.value
    if (!schedule)
      return null

    const totalSlots = schedule.resources.length * 7 * 3
    const assignedSlots = schedule.assignments.length
    const coverage = totalSlots > 0 ? Math.round((assignedSlots / totalSlots) * 100) : 0

    return {
      totalAssignments: schedule.assignments.length,
      resourceCount: schedule.resources.length,
      shiftTypes: schedule.shifts.length,
      conflicts: conflictCount.value,
      warnings: warningCount.value,
      coverage,
    }
  }

  // ==================== 查询辅助 ====================

  function getAssignmentsByResource(resourceId: string) {
    const s = currentSchedule.value
    if (!s)
      return []
    return s.assignments.filter(a => a.resourceId === resourceId)
  }

  function getAssignmentsByShift(shiftId: string) {
    const s = currentSchedule.value
    if (!s)
      return []
    return s.assignments.filter(a => a.shiftId === shiftId)
  }

  function getAssignmentsByDateRange(start: string, end: string) {
    const s = currentSchedule.value
    if (!s)
      return []
    const startTime = new Date(start).getTime()
    const endTime = new Date(end).getTime()
    return s.assignments.filter((a) => {
      const aStart = new Date(a.start).getTime()
      const aEnd = new Date(a.end).getTime()
      return aStart < endTime && aEnd > startTime
    })
  }

  function getConflictsByResource(resourceId: string) {
    const s = currentSchedule.value
    if (!s?.constraints)
      return []
    return s.constraints.filter(c =>
      c.relatedIds.includes(resourceId) && c.level === 'error',
    )
  }

  // ==================== 暴露 ====================

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
    getStatistics,

    // 查询
    getAssignmentsByResource,
    getAssignmentsByShift,
    getAssignmentsByDateRange,
    getConflictsByResource,
  }
})
