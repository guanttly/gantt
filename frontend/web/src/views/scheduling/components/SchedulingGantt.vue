<script setup lang="ts">
import type { Timeline as TimelineType } from 'vis-timeline/standalone'
import type { ScheduleAssignment } from '@/types/scheduling'
import type { Shift } from '@/types/shift'

import { ElMessage, ElMessageBox } from 'element-plus'
import { DataSet, moment, Timeline } from 'vis-timeline/standalone'
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'

import { listEmployees } from '@/api/employees'
import { batchAssignSchedule, deleteScheduleAssignment, getScheduleByDateRange } from '@/api/schedules'
import { listShifts } from '@/api/shifts'
import SvgIcon from '@/components/SvgIcon.vue'

import ShiftSelectDialog from './ShiftSelectDialog.vue'
import 'vis-timeline/styles/vis-timeline-graph2d.min.css'

const props = defineProps<{
  dateRange: [Date | string, Date | string]
  assignments?: ScheduleAssignment[]
  readonly?: boolean
}>()

// moment 中文 locale
if (moment.localeData('zh-cn')) {
  moment.updateLocale('zh-cn', {
    weekdays: '星期日_星期一_星期二_星期三_星期四_星期五_星期六'.split('_'),
    weekdaysShort: '周日_周一_周二_周三_周四_周五_周六'.split('_'),
    months: '1月_2月_3月_4月_5月_6月_7月_8月_9月_10月_11月_12月'.split('_'),
  })
}
else {
  moment.defineLocale('zh-cn', {
    weekdays: '星期日_星期一_星期二_星期三_星期四_星期五_星期六'.split('_'),
    weekdaysShort: '周日_周一_周二_周三_周四_周五_周六'.split('_'),
    months: '1月_2月_3月_4月_5月_6月_7月_8月_9月_10月_11月_12月'.split('_'),
  })
}
moment.locale('zh-cn')

// Timeline 实例
const timelineContainer = ref<HTMLElement>()
let timeline: TimelineType | null = null
let items: DataSet<any> | null = null
let groups: DataSet<any> | null = null

// 数据
const employees = ref<any[]>([])
const shifts = ref<Shift[]>([])
const localAssignments = ref<ScheduleAssignment[]>([])
const loading = ref(false)

// 班次选择对话框
const shiftDialogVisible = ref(false)
const currentAddingItem = ref<{
  employeeId: string
  employeeName: string
  date: string
  callback: (item: any) => void
} | null>(null)

function formatDate(date: Date | string): string {
  if (typeof date === 'string')
    return date
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function parseTimeToMinutes(timeStr: string): number {
  if (!timeStr)
    return 0
  const parts = timeStr.split(':')
  if (parts.length !== 2)
    return 0
  const hours = Number.parseInt(parts[0], 10) || 0
  const minutes = Number.parseInt(parts[1], 10) || 0
  return hours * 60 + minutes
}

async function loadBaseDataInternal() {
  const [empRes, shiftRes] = await Promise.all([
    listEmployees({ page: 1, size: 1000 }),
    listShifts({ page: 1, size: 100 }),
  ])
  employees.value = empRes.items || []
  shifts.value = Array.isArray(shiftRes) ? shiftRes : (shiftRes.items || [])
}

async function loadScheduleDataInternal() {
  if (!props.dateRange || props.dateRange.length !== 2)
    return

  if (props.assignments !== undefined) {
    localAssignments.value = props.assignments || []
    return
  }

  const [start, end] = props.dateRange
  const startDate = formatDate(start)
  const endDate = formatDate(end)

  const data = await getScheduleByDateRange({
    start_date: startDate,
    end_date: endDate,
  })
  localAssignments.value = data || []
}

async function loadScheduleData() {
  if (!props.dateRange || props.dateRange.length !== 2)
    return

  loading.value = true
  try {
    await loadScheduleDataInternal()
    await new Promise(resolve => requestAnimationFrame(resolve))
    renderGantt()
    await new Promise(resolve => setTimeout(resolve, 100))
  }
  catch (error: any) {
    ElMessage.error(`加载排班数据失败: ${error.message}`)
  }
  finally {
    loading.value = false
  }
}

function createGroups(): any[] {
  if (props.assignments !== undefined && props.assignments.length > 0) {
    const employeeIdsInAssignments = new Set(props.assignments.map(a => a.employee_id))
    return employees.value
      .filter(emp => employeeIdsInAssignments.has(emp.id))
      .map((emp, index) => ({
        id: emp.id,
        content: emp.name || emp.id,
        order: index,
      }))
  }
  return employees.value.map((emp, index) => ({
    id: emp.id,
    content: emp.name || emp.id,
    order: index,
  }))
}

function createItems(): any[] {
  const shiftMap = new Map(shifts.value.map(s => [s.id, s]))
  const employeeMap = new Map(employees.value.map(e => [e.id, e]))

  if (localAssignments.value.length === 0)
    return []

  const groupedAssignments = new Map<string, ScheduleAssignment[]>()
  for (const assignment of localAssignments.value) {
    if (!employeeMap.has(assignment.employee_id))
      continue
    const key = `${assignment.employee_id}_${assignment.date}`
    if (!groupedAssignments.has(key))
      groupedAssignments.set(key, [])
    groupedAssignments.get(key)!.push(assignment)
  }

  const resultItems: any[] = []

  groupedAssignments.forEach((groupAssignments) => {
    groupAssignments.sort((a, b) => {
      const shiftA = shiftMap.get(a.shift_id)
      const shiftB = shiftMap.get(b.shift_id)
      const timeA = shiftA?.start_time ? parseTimeToMinutes(shiftA.start_time) : 0
      const timeB = shiftB?.start_time ? parseTimeToMinutes(shiftB.start_time) : 0
      return timeA - timeB
    })

    groupAssignments.forEach((assignment, index) => {
      const shift = shiftMap.get(assignment.shift_id)
      const date = new Date(assignment.date)

      const start = new Date(date)
      start.setHours(0, 0, 0, 0)
      const end = new Date(date)
      end.setHours(23, 59, 59, 999)

      resultItems.push({
        id: assignment.id,
        group: assignment.employee_id,
        subgroup: index,
        content: shift?.name || assignment.shift_name || '未知班次',
        start,
        end,
        style: shift?.color
          ? `background-color: ${shift.color}; border-color: ${shift.color};`
          : '',
        title: `${shift?.name || '未知班次'}\n${assignment.notes || ''}`,
        className: 'scheduling-item',
        editable: {
          remove: !props.readonly,
          updateTime: !props.readonly,
          updateGroup: !props.readonly,
        },
        assignmentId: assignment.id,
        employeeId: assignment.employee_id,
        shiftId: assignment.shift_id,
        date: assignment.date,
      })
    })
  })

  return resultItems
}

function getTimelineOptions() {
  const baseOptions: any = {
    groupOrder: 'order',
    stack: true,
    stackSubgroups: true,
    editable: {
      add: !props.readonly,
      updateTime: !props.readonly,
      updateGroup: !props.readonly,
      remove: !props.readonly,
    },
    margin: {
      item: { horizontal: 0, vertical: 5 },
      axis: 5,
    },
    orientation: 'top',
    start: new Date(),
    end: new Date(),
    moment,
    zoomable: false,
    moveable: !props.readonly,
    verticalScroll: true,
    horizontalScroll: true,
    height: '100%',
    snap: (date: Date) => {
      const d = new Date(date)
      d.setHours(0, 0, 0, 0)
      return d
    },
    format: {
      minorLabels: {
        millisecond: 'SSS',
        second: 's',
        minute: 'HH:mm',
        hour: 'HH:mm',
        weekday: 'M月D日 ddd',
        day: 'D日 ddd',
        week: 'w周',
        month: 'M月',
        year: 'YYYY',
      },
      majorLabels: {
        millisecond: 'HH:mm:ss',
        second: 'D日 HH:mm',
        minute: 'ddd D日',
        hour: 'ddd D日',
        weekday: 'YYYY年M月',
        day: 'YYYY年M月',
        week: 'YYYY年M月',
        month: 'YYYY年',
        year: '',
      },
    },
  }

  if (!props.readonly) {
    baseOptions.onAdd = handleAddItem
    baseOptions.onMove = handleMoveItem
    baseOptions.onRemove = handleRemoveItem
  }

  return baseOptions
}

function renderGantt() {
  if (!timelineContainer.value)
    return

  const newGroups = createGroups()
  const newItems = createItems()

  if (timeline && groups && items) {
    groups.clear()
    if (newGroups.length > 0)
      groups.add(newGroups)

    items.clear()
    if (newItems.length > 0)
      items.add(newItems)

    const startDate = typeof props.dateRange[0] === 'string'
      ? new Date(props.dateRange[0])
      : props.dateRange[0]
    const endDate = typeof props.dateRange[1] === 'string'
      ? new Date(props.dateRange[1])
      : props.dateRange[1]
    timeline.setWindow(startDate, endDate)
    timeline.redraw()
    return
  }

  destroyTimeline()

  groups = new DataSet(newGroups)
  items = new DataSet(newItems)

  const startDate = typeof props.dateRange[0] === 'string'
    ? new Date(props.dateRange[0])
    : props.dateRange[0]
  const endDate = typeof props.dateRange[1] === 'string'
    ? new Date(props.dateRange[1])
    : props.dateRange[1]

  const timelineOptions = getTimelineOptions()
  timelineOptions.start = startDate
  timelineOptions.end = endDate

  timeline = new Timeline(timelineContainer.value, items, groups, timelineOptions)

  setTimeout(() => {
    if (timeline)
      timeline.redraw()
  }, 100)
}

async function handleAddItem(item: any, callback: (item: any) => void) {
  const employeeId = item.group
  const employee = employees.value.find(emp => emp.id === employeeId)
  const date = formatDate(new Date(item.start))

  if (!employee) {
    ElMessage.warning('未找到员工信息')
    callback(null)
    return
  }

  currentAddingItem.value = {
    employeeId,
    employeeName: employee.name,
    date,
    callback,
  }
  shiftDialogVisible.value = true
}

async function handleConfirmAddShift(shiftId: string) {
  if (!currentAddingItem.value)
    return

  const { employeeId, date, callback } = currentAddingItem.value

  const existingAssignment = localAssignments.value.find(
    assignment =>
      assignment.employee_id === employeeId
      && assignment.shift_id === shiftId
      && assignment.date === date,
  )

  if (existingAssignment) {
    const employee = employees.value.find(emp => emp.id === employeeId)
    const shift = shifts.value.find(s => s.id === shiftId)
    ElMessage.warning(`${employee?.name || '该员工'}在${date}已经有${shift?.name || '该班次'}的排班，不能重复添加`)
    callback(null)
    currentAddingItem.value = null
    return
  }

  try {
    await batchAssignSchedule({
      assignments: [{
        employee_id: employeeId,
        shift_id: shiftId,
        date,
        notes: '',
      }],
    })

    ElMessage.success('添加排班成功')
    await loadScheduleData()
    callback(null)
  }
  catch (error: any) {
    ElMessage.error(`添加排班失败: ${error.message || '未知错误'}`)
    callback(null)
  }
  finally {
    currentAddingItem.value = null
  }
}

function handleCancelAddShift() {
  if (currentAddingItem.value) {
    currentAddingItem.value.callback(null)
    currentAddingItem.value = null
  }
}

async function handleMoveItem(item: any, callback: (item: any) => void) {
  callback(null)

  try {
    const oldEmployeeId = item.employeeId
    const newEmployeeId = item.group
    const oldDate = item.date
    const newDate = formatDate(new Date(item.start))

    if (oldEmployeeId === newEmployeeId && oldDate === newDate) {
      await loadScheduleData()
      return
    }

    const existingAssignment = localAssignments.value.find(
      assignment =>
        assignment.id !== item.assignmentId
        && assignment.employee_id === newEmployeeId
        && assignment.shift_id === item.shiftId
        && assignment.date === newDate,
    )

    if (existingAssignment) {
      const employee = employees.value.find(emp => emp.id === newEmployeeId)
      const shift = shifts.value.find(s => s.id === item.shiftId)
      ElMessage.warning(`${employee?.name || '该员工'}在${newDate}已经有${shift?.name || '该班次'}的排班，不能重复添加`)
      await loadScheduleData()
      return
    }

    await deleteScheduleAssignment(item.assignmentId)

    await batchAssignSchedule({
      assignments: [{
        employee_id: newEmployeeId,
        shift_id: item.shiftId,
        date: newDate,
        notes: item.title,
      }],
    })

    ElMessage.success('排班已更新')
    await loadScheduleData()
  }
  catch (error: any) {
    ElMessage.error(`更新排班失败: ${error.message}`)
    await loadScheduleData()
  }
}

async function handleRemoveItem(item: any, callback: (item: any) => void) {
  try {
    await ElMessageBox.confirm('确认删除这个排班吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })

    await deleteScheduleAssignment(item.assignmentId)
    ElMessage.success('排班已删除')
    await loadScheduleData()
    callback(item)
  }
  catch (error: any) {
    if (error !== 'cancel')
      ElMessage.error(`删除排班失败: ${error.message}`)
    callback(null)
  }
}

function destroyTimeline() {
  if (timeline) {
    timeline.destroy()
    timeline = null
  }
  if (items) {
    items.clear()
    items = null
  }
  if (groups) {
    groups.clear()
    groups = null
  }
}

function handleKeyDown(event: KeyboardEvent) {
  if (props.readonly)
    return
  if (event.key !== 'Delete' && event.key !== 'Backspace')
    return

  const target = event.target as HTMLElement
  if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable)
    return
  if (!timeline || !items)
    return

  const selectedIds = timeline.getSelection()
  if (selectedIds.length === 0)
    return

  event.preventDefault()
  event.stopPropagation()

  const selectedId = selectedIds[0]
  const selectedItem = items.get(selectedId)
  if (!selectedItem)
    return

  handleRemoveItem(selectedItem, (item) => {
    if (item && timeline)
      timeline.setSelection([])
  })
}

// 容器尺寸变化监听
let resizeObserver: ResizeObserver | null = null
function setupResizeObserver() {
  if (!timelineContainer.value || !window.ResizeObserver)
    return

  resizeObserver = new ResizeObserver(() => {
    if (timeline) {
      requestAnimationFrame(() => {
        if (timeline)
          timeline.redraw()
      })
    }
  })
  resizeObserver.observe(timelineContainer.value)
}

// Watch dateRange
let dateChangeTimer: ReturnType<typeof setTimeout> | null = null
watch(() => props.dateRange, () => {
  if (dateChangeTimer)
    clearTimeout(dateChangeTimer)
  dateChangeTimer = setTimeout(() => {
    loadScheduleData()
  }, 300)
}, { deep: true })

// Watch assignments prop (preview mode)
watch(() => props.assignments, async (newAssignments) => {
  if (props.assignments !== undefined && newAssignments) {
    localAssignments.value = newAssignments
    if (employees.value.length === 0 || shifts.value.length === 0)
      await loadBaseDataInternal()
    await nextTick()
    renderGantt()
  }
}, { deep: true, immediate: true })

onMounted(async () => {
  loading.value = true
  try {
    await loadBaseDataInternal()
    if (props.assignments === undefined) {
      await loadScheduleDataInternal()
    }
    else {
      localAssignments.value = props.assignments || []
    }
    renderGantt()
    setupResizeObserver()
    if (!props.readonly)
      window.addEventListener('keydown', handleKeyDown)
  }
  catch (error: any) {
    ElMessage.error(`初始化失败: ${error.message}`)
  }
  finally {
    loading.value = false
  }
})

onBeforeUnmount(() => {
  if (dateChangeTimer)
    clearTimeout(dateChangeTimer)
  if (resizeObserver) {
    resizeObserver.disconnect()
    resizeObserver = null
  }
  window.removeEventListener('keydown', handleKeyDown)
  destroyTimeline()
})

defineExpose({ refresh: loadScheduleData })
</script>

<template>
  <div
    v-loading="loading"
    class="scheduling-gantt"
    element-loading-background="rgba(255, 255, 255, 0.8)"
    element-loading-text="加载中..."
  >
    <div v-if="!loading && employees.length === 0 && !props.assignments?.length" class="empty-state">
      <el-empty description="暂无员工数据，请先添加员工" />
    </div>
    <template v-else>
      <div v-if="!readonly" class="gantt-tips">
        <span class="tip-item"><SvgIcon name="lightbulb" size="1em" /> 双击空白处添加排班</span>
        <span class="tip-item"><SvgIcon name="pushpin" size="1em" /> 拖动排班可调整日期或员工</span>
        <span class="tip-item"><SvgIcon name="trash" size="1em" /> 选中后按 Delete 删除</span>
      </div>
      <div ref="timelineContainer" class="gantt-container" />
    </template>

    <ShiftSelectDialog
      v-model:visible="shiftDialogVisible"
      :date="currentAddingItem?.date || ''"
      :employee-name="currentAddingItem?.employeeName || ''"
      :shifts="shifts"
      @cancel="handleCancelAddShift"
      @confirm="handleConfirmAddShift"
    />
  </div>
</template>

<style lang="scss" scoped>
.scheduling-gantt {
  flex: 1;
  overflow: hidden;
  position: relative;
  background-color: #fff;
  transition: opacity 0.3s ease;
  display: flex;
  flex-direction: column;
  min-height: 400px;
  height: 100%;

  .empty-state {
    height: 100%;
    min-height: 400px;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .gantt-tips {
    padding: 8px 16px;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    color: #fff;
    display: flex;
    gap: 24px;
    font-size: 13px;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
    z-index: 10;

    .tip-item {
      display: flex;
      align-items: center;
      gap: 4px;
      opacity: 0.95;

      &:hover {
        opacity: 1;
      }
    }
  }

  .gantt-container {
    flex: 1;
    width: 100%;
    min-height: 400px;
    overflow: auto;
    transition: opacity 0.3s ease;
  }
}

:deep(.vis-timeline) {
  border: none;
  font-family: var(--el-font-family);
  visibility: visible !important;
}

:deep(.vis-item) {
  border-radius: 4px;
  border: none;
  font-size: 12px;
  color: #fff;
  cursor: pointer;
  transition: all 0.2s;
  padding: 4px 8px;
  box-sizing: border-box;

  &:hover {
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
  }
}

:deep(.vis-item.vis-selected) {
  outline: 2px solid rgba(64, 158, 255, 0.6) !important;
  outline-offset: -2px !important;
  z-index: 999;
  background-color: inherit !important;
  color: inherit !important;
}

:deep(.vis-labelset .vis-label) {
  background-color: #f5f7fa;
  border-bottom: 1px solid #e4e7ed;
  color: var(--el-text-color-primary);
  font-weight: 500;
}

:deep(.vis-time-axis .vis-text) {
  color: var(--el-text-color-regular);
}

:deep(.vis-panel.vis-center),
:deep(.vis-panel.vis-top) {
  border-color: #e4e7ed;
}

:deep(.vis-current-time) {
  background-color: #f56c6c;
  width: 2px;
}

:deep(.vis-delete) {
  background: #f56c6c !important;
  border: none !important;
  border-radius: 50% !important;
  width: 18px !important;
  height: 18px !important;
  min-width: 18px !important;
  min-height: 18px !important;
  padding: 0 !important;
  margin: 0 !important;
  position: absolute !important;
  right: 4px !important;
  top: 50% !important;
  transform: translateY(-50%) !important;
  cursor: pointer !important;
  display: inline-flex !important;
  align-items: center !important;
  justify-content: center !important;
  color: #fff !important;
  line-height: 1 !important;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.3) !important;
  transition: all 0.2s ease !important;
  z-index: 1000 !important;
  text-indent: -9999px !important;
  overflow: hidden !important;
  font-size: 0 !important;

  &::before {
    content: '×' !important;
    display: inline-block !important;
    line-height: 1 !important;
    margin: 0 !important;
    padding: 0 !important;
    font-size: 14px !important;
    text-indent: 0 !important;
    color: #fff !important;
  }

  &:hover {
    background: #f78989 !important;
    box-shadow: 0 2px 6px rgba(245, 108, 108, 0.6) !important;
    transform: translateY(-50%) scale(1.1) !important;
  }
}
</style>
