<script setup lang="ts">
import type { Timeline as TimelineType } from 'vis-timeline/standalone'
import { ElMessage, ElMessageBox } from 'element-plus'
import { DataSet, moment, Timeline } from 'vis-timeline/standalone'
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { getEmployeeList } from '@/api/employee'
import { batchAssignSchedule, deleteScheduleAssignment, getScheduleByDateRange } from '@/api/scheduling'
import { getShiftList } from '@/api/shift'
import SvgIcon from '@/components/SvgIcon.vue'
import ShiftSelectDialog from './ShiftSelectDialog.vue'
import 'vis-timeline/styles/vis-timeline-graph2d.min.css'

const props = defineProps<{
  dateRange: [Date | string, Date | string]
  assignments?: Scheduling.Assignment[] // 可选的排班数据（如果提供，则不从 API 加载）
  readonly?: boolean // 是否只读模式（预览模式，禁用编辑）
}>()

// 设置 moment 中文 locale（使用 updateLocale 避免警告）
if (moment.localeData('zh-cn')) {
  moment.updateLocale('zh-cn', {
    weekdays: '星期日_星期一_星期二_星期三_星期四_星期五_星期六'.split('_'),
    weekdaysShort: '周日_周一_周二_周三_周四_周五_周六'.split('_'),
    months: '1月_2月_3月_4月_5月_6月_7月_8月_9月_10月_11月_12月'.split('_'),
  })
} else {
  moment.defineLocale('zh-cn', {
    weekdays: '星期日_星期一_星期二_星期三_星期四_星期五_星期六'.split('_'),
    weekdaysShort: '周日_周一_周二_周三_周四_周五_周六'.split('_'),
    months: '1月_2月_3月_4月_5月_6月_7月_8月_9月_10月_11月_12月'.split('_'),
  })
}
moment.locale('zh-cn')

defineExpose({
  refresh: loadScheduleData,
})

const orgId = ref('default-org')

// Timeline 实例
const timelineContainer = ref<HTMLElement>()
let timeline: TimelineType | null = null
let items: DataSet<any> | null = null
let groups: DataSet<any> | null = null

// 数据
const employees = ref<Employee.EmployeeInfo[]>([])
const shifts = ref<Shift.ShiftInfo[]>([])
const assignments = ref<Scheduling.Assignment[]>([])
const loading = ref(false)

// 班次选择对话框相关
const shiftDialogVisible = ref(false)
const currentAddingItem = ref<{
  employeeId: string
  employeeName: string
  date: string
  callback: (item: any) => void
} | null>(null)

// 加载基础数据(内部使用,不显示 loading)
async function loadBaseDataInternal() {
  // 加载员工列表
  const empRes = await getEmployeeList({
    orgId: orgId.value,
    page: 1,
    size: 1000,
  })
  employees.value = empRes.items || []

  // 加载班次列表
  const shiftRes = await getShiftList({
    orgId: orgId.value,
    isActive: true,
    page: 1,
    size: 100,
  })
  shifts.value = shiftRes.items || []
}

// 加载排班数据(内部使用,不显示 loading)
async function loadScheduleDataInternal() {
  if (!props.dateRange || props.dateRange.length !== 2)
    return

  // 如果提供了 assignments prop，直接使用，不调用 API
  if (props.assignments !== undefined) {
    assignments.value = props.assignments || []
    return
  }

  const [start, end] = props.dateRange
  const startDate = formatDate(start)
  const endDate = formatDate(end)

  const data = await getScheduleByDateRange({
    orgId: orgId.value,
    startDate,
    endDate,
  })
  assignments.value = data || []
}

// 加载排班数据(对外暴露,显示 loading)
async function loadScheduleData() {
  if (!props.dateRange || props.dateRange.length !== 2)
    return

  loading.value = true
  try {
    await loadScheduleDataInternal()
    // 延迟一帧再渲染,确保 loading 显示
    await new Promise(resolve => requestAnimationFrame(resolve))
    renderGantt()
    // 等待渲染完成后再关闭 loading
    await new Promise(resolve => setTimeout(resolve, 100))
  }
  catch (error: any) {
    ElMessage.error(`加载排班数据失败: ${error.message}`)
  }
  finally {
    loading.value = false
  }
}

// 格式化日期
function formatDate(date: Date | string): string {
  if (typeof date === 'string') {
    return date
  }
  // 使用本地时间格式化，避免时区转换导致的日期偏移
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

// 创建甘特图分组（员工）
function createGroups(): any[] {
  // 如果提供了 assignments，只包含有排班的员工
  if (props.assignments !== undefined && props.assignments.length > 0) {
    const employeeIdsInAssignments = new Set(props.assignments.map(a => a.employeeId))
    const groups = employees.value
      .filter(emp => employeeIdsInAssignments.has(emp.id))
      .map((emp, index) => ({
        id: emp.id,
        content: emp.name || emp.id,
        order: index,
      }))
    return groups
  }
  
  // 否则返回所有员工
  return employees.value.map((emp, index) => ({
    id: emp.id,
    content: emp.name || emp.id,
    order: index,
  }))
}

// 解析时间字符串（HH:MM）为分钟数，用于排序
function parseTimeToMinutes(timeStr: string): number {
  if (!timeStr) return 0
  const parts = timeStr.split(':')
  if (parts.length !== 2) return 0
  const hours = parseInt(parts[0], 10) || 0
  const minutes = parseInt(parts[1], 10) || 0
  return hours * 60 + minutes
}

// 创建甘特图数据项
function createItems(): any[] {
  const shiftMap = new Map(shifts.value.map(s => [s.id, s]))
  const employeeMap = new Map(employees.value.map(e => [e.id, e]))

  if (assignments.value.length === 0) {
    return []
  }

  // 按员工和日期分组
  const groupedAssignments = new Map<string, Scheduling.Assignment[]>()
  
  assignments.value.forEach((assignment) => {
    // 检查员工是否存在
    if (!employeeMap.has(assignment.employeeId)) {
      return
    }
    
    const key = `${assignment.employeeId}_${assignment.date}`
    if (!groupedAssignments.has(key)) {
      groupedAssignments.set(key, [])
    }
    groupedAssignments.get(key)!.push(assignment)
  })


  const items: any[] = []

  // 对每个分组内的排班按开始时间排序
  groupedAssignments.forEach((groupAssignments, key) => {
    // 按班次开始时间排序
    groupAssignments.sort((a, b) => {
      const shiftA = shiftMap.get(a.shiftId)
      const shiftB = shiftMap.get(b.shiftId)
      const timeA = shiftA?.startTime ? parseTimeToMinutes(shiftA.startTime) : 0
      const timeB = shiftB?.startTime ? parseTimeToMinutes(shiftB.startTime) : 0
      return timeA - timeB // 从早到晚排序
    })

    // 为每个排班创建数据项，使用排序后的索引作为 subgroup
    groupAssignments.forEach((assignment, index) => {
      const shift = shiftMap.get(assignment.shiftId)
      const date = new Date(assignment.date)

      // 不关注具体时间，每个班次占满整天，通过 subgroup 实现同一天多个班次堆叠显示
      const start = new Date(date)
      start.setHours(0, 0, 0, 0)
      const end = new Date(date)
      end.setHours(23, 59, 59, 999)

      items.push({
        id: assignment.id,
        group: assignment.employeeId, // 必须匹配 createGroups 中的 id
        subgroup: index, // 使用排序后的索引作为子组，确保按开始时间顺序显示
        content: shift?.name || assignment.shiftName || '未知班次',
        start,
        end,
        style: shift?.color ? `background-color: ${shift.color}; border-color: ${shift.color};` : '',
        title: `${shift?.name || '未知班次'}\n${assignment.notes || ''}`,
        className: 'scheduling-item',
        editable: {
          remove: !props.readonly,
          updateTime: !props.readonly, // 允许横向拖动改变日期
          updateGroup: !props.readonly, // 允许纵向拖动改变员工
        },
        assignmentId: assignment.id,
        employeeId: assignment.employeeId,
        shiftId: assignment.shiftId,
        date: assignment.date,
      })
    })
  })

  return items
}

// 使用函数生成 options，以响应 readonly 变化
function getOptions() {
  const baseOptions: any = {
    groupOrder: 'order',
    stack: true, // 启用堆叠，让同一天的多个班次自动堆叠
    stackSubgroups: true, // 启用子组堆叠
    editable: {
      add: !props.readonly,
      updateTime: !props.readonly, // 允许横向拖动改变日期
      updateGroup: !props.readonly, // 允许纵向拖动改变员工
      remove: !props.readonly,
    },
    margin: {
      item: {
        horizontal: 0, // 水平间距设为0，让班次填满整天
        vertical: 5,
      },
      axis: 5,
    },
    orientation: 'top',
    start: new Date(),
    end: new Date(),
    moment,
    // min: start,
    // max: end,
    // zoomMin: 1000 * 60 * 60 * 24, // 最小缩放1天
    // zoomMax: 1000 * 60 * 60 * 24 * 90, // 最大缩放90天
    zoomable: false, // 禁用滚轮缩放
    moveable: !props.readonly, // 预览模式下禁用拖动
    verticalScroll: true, // 启用垂直滚动
    horizontalScroll: true, // 启用水平滚动
    height: '100%', // 使用 100% 高度
    snap: (date: Date) => {
      // 强制对齐到天
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

  // 只在非只读模式下添加事件回调
  if (!props.readonly) {
    baseOptions.onAdd = handleAddItem
    baseOptions.onMove = handleMoveItem
    baseOptions.onRemove = handleRemoveItem
  }

  return baseOptions
}

// 渲染甘特图
function renderGantt() {
  if (!timelineContainer.value)
    return

  const newGroups = createGroups()
  const newItems = createItems()

  // 如果 Timeline 已存在,只更新数据
  if (timeline && groups && items) {
    // 更新分组
    groups.clear()
    if (newGroups.length > 0) {
      groups.add(newGroups)
    }

    // 更新数据项
    items.clear()
    if (newItems.length > 0) {
      items.add(newItems)
    }

    // 更新时间范围和边界
    const startDate = typeof props.dateRange[0] === 'string' ? new Date(props.dateRange[0]) : props.dateRange[0]
    const endDate = typeof props.dateRange[1] === 'string' ? new Date(props.dateRange[1]) : props.dateRange[1]
    timeline.setWindow(startDate, endDate)

    // 强制重绘
    timeline.redraw()

    return
  }

  // 首次创建 Timeline
  // 销毁旧实例(如果存在)
  destroyTimeline()

  // 创建数据
  groups = new DataSet(newGroups)
  items = new DataSet(newItems)

  // 配置选项
  // 确保日期范围是 Date 对象
  const startDate = typeof props.dateRange[0] === 'string' ? new Date(props.dateRange[0]) : props.dateRange[0]
  const endDate = typeof props.dateRange[1] === 'string' ? new Date(props.dateRange[1]) : props.dateRange[1]
  const timelineOptions = getOptions()
  timelineOptions.start = startDate
  timelineOptions.end = endDate
  
  // 创建Timeline
  timeline = new Timeline(timelineContainer.value, items, groups, timelineOptions)
  
  // 强制重绘一次，确保显示
  setTimeout(() => {
    if (timeline) {
      timeline.redraw()
    }
  }, 100)
}

// 添加排班项
async function handleAddItem(item: any, callback: (item: any) => void) {
  // 获取员工和日期信息
  const employeeId = item.group
  const employee = employees.value.find(emp => emp.id === employeeId)
  const date = formatDate(new Date(item.start))

  if (!employee) {
    ElMessage.warning('未找到员工信息')
    callback(null)
    return
  }

  // 显示班次选择对话框
  currentAddingItem.value = {
    employeeId,
    employeeName: employee.name,
    date,
    callback,
  }
  shiftDialogVisible.value = true
}

// 确认添加排班
async function handleConfirmAddShift(shiftId: string) {
  if (!currentAddingItem.value)
    return

  const { employeeId, date, callback } = currentAddingItem.value

  // 检查是否已经存在相同的排班（同一员工、同一班次、同一日期）
  const existingAssignment = assignments.value.find(
    assignment =>
      assignment.employeeId === employeeId &&
      assignment.shiftId === shiftId &&
      assignment.date === date,
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
    // 调用API添加排班
    await batchAssignSchedule({
      orgId: orgId.value,
      assignments: [{
        employeeId,
        shiftId,
        date,
        notes: '',
      }],
    })

    // 更新成功，刷新数据
    ElMessage.success('添加排班成功')
    await loadScheduleData()
    callback(null) // 取消原始添加，因为我们已经刷新了数据
  }
  catch (error: any) {
    ElMessage.error(`添加排班失败: ${error.message || '未知错误'}`)
    callback(null)
  }
  finally {
    currentAddingItem.value = null
  }
}

// 取消添加排班
function handleCancelAddShift() {
  if (currentAddingItem.value) {
    currentAddingItem.value.callback(null)
    currentAddingItem.value = null
  }
}

// 移动排班项（换员工或换日期）
async function handleMoveItem(item: any, callback: (item: any) => void) {
  // 先取消 vis-timeline 的自动更新，防止出现重复项目
  callback(null)

  try {
    const oldEmployeeId = item.employeeId
    const newEmployeeId = item.group
    const oldDate = item.date
    const newDate = formatDate(new Date(item.start))

    if (oldEmployeeId === newEmployeeId && oldDate === newDate) {
      // 位置没有变化，不需要操作，直接刷新数据恢复原始状态
      await loadScheduleData()
      return
    }

    // 检查目标位置是否已经存在相同的排班（排除当前正在移动的排班）
    const existingAssignment = assignments.value.find(
      assignment =>
        assignment.id !== item.assignmentId && // 排除当前排班本身
        assignment.employeeId === newEmployeeId &&
        assignment.shiftId === item.shiftId &&
        assignment.date === newDate,
    )

    if (existingAssignment) {
      const employee = employees.value.find(emp => emp.id === newEmployeeId)
      const shift = shifts.value.find(s => s.id === item.shiftId)
      ElMessage.warning(`${employee?.name || '该员工'}在${newDate}已经有${shift?.name || '该班次'}的排班，不能重复添加`)
      // 恢复原始数据
      await loadScheduleData()
      return
    }

    // 删除旧的排班（通过ID）
    await deleteScheduleAssignment({
      orgId: orgId.value,
      id: item.assignmentId,
    })

    // 添加新的排班
    await batchAssignSchedule({
      orgId: orgId.value,
      assignments: [{
        employeeId: newEmployeeId,
        shiftId: item.shiftId,
        date: newDate,
        notes: item.title,
      }],
    })

    ElMessage.success('排班已更新')
    // 刷新数据，确保界面与数据一致
    await loadScheduleData()
  }
  catch (error: any) {
    ElMessage.error(`更新排班失败: ${error.message}`)
    // 出错时恢复原始数据
    await loadScheduleData()
  }
}

// 删除排班项
async function handleRemoveItem(item: any, callback: (item: any) => void) {
  try {
    await ElMessageBox.confirm('确认删除这个排班吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })

    await deleteScheduleAssignment({
      orgId: orgId.value,
      id: item.assignmentId,
    })

    ElMessage.success('排班已删除')
    await loadScheduleData()
    callback(item)
  }
  catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(`删除排班失败: ${error.message}`)
    }
    callback(null)
  }
}

// 销毁Timeline
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

// 监听日期范围变化
let dateChangeTimer: any = null
watch(() => props.dateRange, () => {
  // 防抖处理,避免快速切换导致多次渲染
  if (dateChangeTimer) {
    clearTimeout(dateChangeTimer)
  }
  dateChangeTimer = setTimeout(() => {
    loadScheduleData()
  }, 300)
}, { deep: true })

// 监听 assignments prop 变化（预览模式）
watch(() => props.assignments, async (newAssignments) => {
  if (props.assignments !== undefined && newAssignments) {
    // 如果提供了 assignments prop，直接使用，不调用 API
    assignments.value = newAssignments
    
    // 确保基础数据已加载（预览模式必须加载员工和班次数据）
    if (employees.value.length === 0 || shifts.value.length === 0) {
      await loadBaseDataInternal()
    }
    
    // 延迟一帧确保 DOM 已更新
    await nextTick()
    renderGantt()
  }
}, { deep: true, immediate: true })

// 处理键盘删除事件
function handleKeyDown(event: KeyboardEvent) {
  // 只读模式下禁用删除
  if (props.readonly) {
    return
  }

  // 只处理 Delete 或 Backspace 键
  if (event.key !== 'Delete' && event.key !== 'Backspace') {
    return
  }

  // 如果焦点在输入框、文本域或其他可输入元素上，不处理删除
  const target = event.target as HTMLElement
  if (
    target.tagName === 'INPUT' ||
    target.tagName === 'TEXTAREA' ||
    target.isContentEditable
  ) {
    return
  }

  // 检查是否有选中的项目
  if (!timeline || !items) {
    return
  }

  // 获取选中的项目 ID
  const selectedIds = timeline.getSelection()
  if (selectedIds.length === 0) {
    return
  }

  // 阻止默认行为（防止浏览器后退）
  event.preventDefault()
  event.stopPropagation()

  // 获取第一个选中的项目
  const selectedId = selectedIds[0]
  const selectedItem = items.get(selectedId)
  if (!selectedItem) {
    return
  }

  // 调用删除处理函数
  handleRemoveItem(selectedItem, (item) => {
    // 删除成功后，从 timeline 中移除选中状态
    if (item && timeline) {
      timeline.setSelection([])
    }
  })
}

// 处理容器大小变化（助手展开/收起时）
let resizeObserver: ResizeObserver | null = null
function setupResizeObserver() {
  if (!timelineContainer.value || !window.ResizeObserver) {
    return
  }

  resizeObserver = new ResizeObserver(() => {
    // 容器大小变化时，重新计算 Timeline 的位置
    if (timeline) {
      // 使用 requestAnimationFrame 确保在下一帧执行，避免频繁触发
      requestAnimationFrame(() => {
        if (timeline) {
          timeline.redraw()
        }
      })
    }
  })

  resizeObserver.observe(timelineContainer.value)
}

// 生命周期
onMounted(async () => {
  loading.value = true
  try {
    await loadBaseDataInternal()
    // 如果提供了 assignments prop，不调用 API 加载
    if (props.assignments === undefined) {
      await loadScheduleDataInternal()
    } else {
      // 预览模式：直接使用传入的 assignments
      assignments.value = props.assignments || []
    }
    renderGantt()
    // 设置 resize 监听，处理助手展开/收起时的布局变化
    setupResizeObserver()
    // 添加键盘事件监听，支持 Delete 键删除（只读模式下不添加）
    if (!props.readonly) {
      window.addEventListener('keydown', handleKeyDown)
    }
  }
  catch (error: any) {
    ElMessage.error(`初始化失败: ${error.message}`)
  }
  finally {
    loading.value = false
  }
})

onBeforeUnmount(() => {
  // 清理定时器
  if (dateChangeTimer) {
    clearTimeout(dateChangeTimer)
  }
  // 清理 resize observer
  if (resizeObserver) {
    resizeObserver.disconnect()
    resizeObserver = null
  }
  // 移除键盘事件监听
  window.removeEventListener('keydown', handleKeyDown)
  destroyTimeline()
})
</script>

<template>
  <div
    v-loading="loading"
    class="scheduling-gantt"
    element-loading-text="加载中..."
    element-loading-background="rgba(255, 255, 255, 0.8)"
  >
    <div v-if="!loading && employees.length === 0 && !props.assignments?.length" class="empty-state">
      <el-empty description="暂无员工数据，请先添加员工" />
    </div>
    <template v-else>
      <!-- 操作提示（仅在非只读模式显示） -->
      <div v-if="!readonly" class="gantt-tips">
        <span class="tip-item"><SvgIcon name="lightbulb" size="1em" /> 双击空白处添加排班</span>
        <span class="tip-item"><SvgIcon name="pushpin" size="1em" /> 拖动排班可调整日期或员工</span>
        <span class="tip-item"><SvgIcon name="trash" size="1em" /> 选中后按 Delete 删除</span>
      </div>
      <div ref="timelineContainer" class="gantt-container" />
    </template>

    <!-- 班次选择对话框 -->
    <ShiftSelectDialog
      v-model:visible="shiftDialogVisible"
      :employee-name="currentAddingItem?.employeeName || ''"
      :date="currentAddingItem?.date || ''"
      :shifts="shifts"
      @confirm="handleConfirmAddShift"
      @cancel="handleCancelAddShift"
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
  min-height: 400px; // 确保预览模式下有最小高度
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
    min-height: 400px; // 确保有最小高度
    overflow: auto; // 使用原生滚动，避免自定义 wheel 监听器
    transition: opacity 0.3s ease;
  }
}

// Timeline 样式覆盖
:deep(.vis-timeline) {
  border: none;
  font-family: var(--el-font-family);
  visibility: visible !important; // 强制显示，防止 vis-timeline 自动隐藏
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
  // 选中时不改变背景色和文字颜色，只通过简单的边框提示
  outline: 2px solid rgba(64, 158, 255, 0.6) !important;
  outline-offset: -2px !important;
  z-index: 999;
  background-color: inherit !important;
  color: inherit !important; // 保持原有文字颜色
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

// 删除按钮样式 - 红色圆形，×居中，位置在右侧向左偏移
:deep(.vis-item.vis-delete),
:deep(.vis-item.vis-delete-button),
:deep(.vis-item .vis-delete),
:deep(.vis-item .vis-close),
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
  right: 4px !important; // 在右侧，向左偏移4px
  top: 50% !important;
  transform: translateY(-50%) !important;
  cursor: pointer !important;
  display: inline-flex !important;
  align-items: center !important;
  justify-content: center !important;
  font-size: 14px !important;
  font-weight: bold !important;
  color: #fff !important;
  line-height: 1 !important;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.3) !important;
  transition: all 0.2s ease !important;
  z-index: 1000 !important;
  // 隐藏vis-timeline自带的文本内容
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
