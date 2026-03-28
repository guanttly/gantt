<script setup lang="ts">
import type { Group } from '@/api/groups'
import type { Employee } from '@/types/employee'
import type { FixedAssignment, Shift, ShiftType, WeeklyStaff } from '@/types/shift'
import { Delete, Edit, Plus, RefreshRight, Search } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, reactive, ref, watch } from 'vue'
import { listEmployees } from '@/api/employees'
import { listGroups } from '@/api/groups'
import {
  batchCreateFixedAssignments,
  createShift,
  deleteShift,
  getFixedAssignments,
  getShiftGroups,
  getShiftWeeklyStaff,
  listShifts,
  setShiftGroups,
  toggleShiftStatus,
  updateShift,
  updateShiftWeeklyStaff,
} from '@/api/shifts'
import { usePagination } from '@/composables/usePagination'
import { useAuthStore } from '@/stores/auth'
import {
  createDefaultWeeklyConfig,
  formatDuration,
  getShiftStatusTagType,
  getShiftStatusText,
  getShiftTypeTagType,
  getShiftTypeText,
  SHIFT_TYPE_OPTIONS,
  WEEKDAY_DISPLAY_NAMES,
  WEEKDAY_DISPLAY_ORDER,
} from '@/types/shift'

type EditableFixedAssignment = FixedAssignment & {
  monthdays_text: string
  specific_dates_text: string
}

type FixedAssignmentDraft = EditableFixedAssignment

const auth = useAuthStore()
const canManageShifts = computed(() => auth.hasPermission('shift:manage'))

const { loading, items, currentPage, currentPageSize, keyword, handlePageChange, handleSizeChange, refresh } = usePagination<Shift>({
  fetchFn: listShifts,
})

const typeFilter = ref<ShiftType | ''>('')
const statusFilter = ref<'active' | 'disabled' | ''>('')

const dialogVisible = ref(false)
const dialogTitle = ref('新增班次')
const formLoading = ref(false)
const editingId = ref<string | null>(null)
const formRef = ref()

const currentShift = ref<Shift | null>(null)

const groupDialogVisible = ref(false)
const groupDialogLoading = ref(false)
const groupDialogSaving = ref(false)
const groupOptions = ref<Group[]>([])
const selectedGroupIds = ref<string[]>([])
const selectedGroups = computed(() => {
  const groupMap = new Map(groupOptions.value.map(group => [group.id, group]))
  return selectedGroupIds.value
    .map(groupId => groupMap.get(groupId))
    .filter((group): group is Group => Boolean(group))
})

const fixedDialogVisible = ref(false)
const fixedDialogLoading = ref(false)
const fixedDialogSaving = ref(false)
const employeeOptions = ref<Employee[]>([])
const fixedAssignments = ref<EditableFixedAssignment[]>([])
const fixedAssignmentsCurrentPage = ref(1)
const fixedAssignmentsPageSize = ref(5)
const fixedAssignmentDraft = reactive<FixedAssignmentDraft>(createEmptyFixedAssignment())

const weeklyDialogVisible = ref(false)
const weeklyDialogLoading = ref(false)
const weeklyDialogSaving = ref(false)
const weeklyConfig = ref<WeeklyStaff>({ shift_id: '', weekly_config: createDefaultWeeklyConfig() })
const weeklyUniformMode = ref(true)
const weeklyUniformStaffCount = ref(0)

const form = reactive({
  name: '',
  code: '',
  color: '#409EFF',
  start_time: '',
  end_time: '',
  type: 'regular' as ShiftType,
  scheduling_priority: 1,
  is_active: true,
  description: '',
})

const rules = {
  name: [{ required: true, message: '请输入班次名称', trigger: 'blur' }],
  code: [{ required: true, message: '请输入班次编码', trigger: 'blur' }],
  start_time: [{ required: true, message: '请选择开始时间', trigger: 'change' }],
  end_time: [{ required: true, message: '请选择结束时间', trigger: 'change' }],
}

const filteredItems = computed(() => items.value.filter((item) => {
  const matchesKeyword = !keyword.value || [item.name, item.code, item.description]
    .filter(Boolean)
    .some(value => value?.toLowerCase().includes(keyword.value.toLowerCase()))
  const matchesType = !typeFilter.value || item.type === typeFilter.value
  const currentStatus = item.is_active ? 'active' : 'disabled'
  const matchesStatus = !statusFilter.value || currentStatus === statusFilter.value
  return matchesKeyword && matchesType && matchesStatus
}))

const filteredTotal = computed(() => filteredItems.value.length)

const pagedItems = computed(() => {
  const start = (currentPage.value - 1) * currentPageSize.value
  return filteredItems.value.slice(start, start + currentPageSize.value)
})

const computedDurationText = computed(() => formatDuration(calculateDurationMinutes(form.start_time, form.end_time)))
const computedCrossDay = computed(() => isCrossDay(form.start_time, form.end_time))
const pagedFixedAssignments = computed(() => {
  const start = (fixedAssignmentsCurrentPage.value - 1) * fixedAssignmentsPageSize.value
  return fixedAssignments.value.slice(start, start + fixedAssignmentsPageSize.value)
})

watch([typeFilter, statusFilter], () => {
  currentPage.value = 1
})

function normalizeListResponse<T>(result: T[] | { items?: T[] } | undefined | null): T[] {
  if (!result) {
    return []
  }
  return Array.isArray(result) ? result : (result.items || [])
}

function calculateDurationMinutes(startTime: string, endTime: string) {
  if (!startTime || !endTime) {
    return 0
  }
  const [sh, sm] = startTime.split(':').map(Number)
  const [eh, em] = endTime.split(':').map(Number)
  let diff = (eh * 60 + em) - (sh * 60 + sm)
  if (diff <= 0) {
    diff += 24 * 60
  }
  return diff
}

function isCrossDay(startTime: string, endTime: string) {
  if (!startTime || !endTime) {
    return false
  }
  const [sh, sm] = startTime.split(':').map(Number)
  const [eh, em] = endTime.split(':').map(Number)
  return (eh * 60 + em) < (sh * 60 + sm)
}

function resetForm() {
  Object.assign(form, {
    name: '',
    code: '',
    color: '#409EFF',
    start_time: '',
    end_time: '',
    type: 'regular',
    scheduling_priority: 1,
    is_active: true,
    description: '',
  })
}

function handleAdd() {
  editingId.value = null
  dialogTitle.value = '新增班次'
  resetForm()
  dialogVisible.value = true
}

function handleEdit(row: Shift) {
  editingId.value = row.id
  dialogTitle.value = '编辑班次'
  Object.assign(form, {
    name: row.name,
    code: row.code || '',
    color: row.color,
    start_time: row.start_time,
    end_time: row.end_time,
    type: row.type,
    scheduling_priority: row.scheduling_priority || 1,
    is_active: row.is_active,
    description: row.description || '',
  })
  dialogVisible.value = true
}

async function handleSubmit() {
  try {
    await formRef.value?.validate()
  }
  catch {
    return
  }

  formLoading.value = true
  try {
    const payload = {
      name: form.name,
      code: form.code,
      color: form.color,
      start_time: form.start_time,
      end_time: form.end_time,
      type: form.type,
      scheduling_priority: form.scheduling_priority,
      is_active: form.is_active,
      description: form.description || undefined,
    }
    if (editingId.value) {
      await updateShift(editingId.value, payload)
      ElMessage.success('班次已更新')
    }
    else {
      await createShift(payload)
      ElMessage.success('班次已创建')
    }
    dialogVisible.value = false
    refresh()
  }
  catch (e: any) {
    ElMessage.error(e?.response?.data?.message || '操作失败')
  }
  finally {
    formLoading.value = false
  }
}

async function handleDelete(row: Shift) {
  await ElMessageBox.confirm(`确定删除班次「${row.name}」吗？`, '确认删除', { type: 'warning' })
  try {
    await deleteShift(row.id)
    ElMessage.success('删除成功')
    refresh()
  }
  catch (e: any) {
    ElMessage.error(e?.response?.data?.message || '删除失败')
  }
}

async function handleToggleStatus(row: Shift) {
  const actionText = row.is_active ? '禁用' : '启用'
  await ElMessageBox.confirm(`确定${actionText}班次「${row.name}」吗？`, `${actionText}班次`, { type: 'warning' })
  try {
    await toggleShiftStatus(row.id, !row.is_active)
    ElMessage.success(`${actionText}成功`)
    refresh()
  }
  catch (e: any) {
    ElMessage.error(e?.response?.data?.message || `${actionText}失败`)
  }
}

async function handleManageGroups(row: Shift) {
  currentShift.value = row
  groupDialogVisible.value = true
  groupDialogLoading.value = true
  try {
    const [groups, relatedGroups] = await Promise.all([
      listGroups({ page: 1, page_size: 500 }),
      getShiftGroups(row.id),
    ])
    groupOptions.value = normalizeListResponse(groups)
    selectedGroupIds.value = relatedGroups.map(item => item.group_id)
  }
  catch (e: any) {
    ElMessage.error(e?.response?.data?.message || '加载关联分组失败')
    groupDialogVisible.value = false
  }
  finally {
    groupDialogLoading.value = false
  }
}

async function saveShiftGroups() {
  if (!currentShift.value) {
    return
  }
  groupDialogSaving.value = true
  try {
    await setShiftGroups(currentShift.value.id, selectedGroupIds.value)
    ElMessage.success('关联分组已更新')
    groupDialogVisible.value = false
    refresh()
  }
  catch (e: any) {
    ElMessage.error(e?.response?.data?.message || '保存关联分组失败')
  }
  finally {
    groupDialogSaving.value = false
  }
}

function toEditableAssignment(item?: FixedAssignment): EditableFixedAssignment {
  return {
    id: item?.id,
    shift_id: item?.shift_id,
    staff_id: item?.staff_id || '',
    staff_name: item?.staff_name,
    pattern_type: item?.pattern_type || 'weekly',
    weekdays: item?.weekdays || [1, 2, 3, 4, 5],
    week_pattern: item?.week_pattern || 'every',
    monthdays: item?.monthdays || [],
    specific_dates: item?.specific_dates || [],
    start_date: item?.start_date,
    end_date: item?.end_date,
    is_active: item?.is_active ?? true,
    monthdays_text: item?.monthdays?.join(',') || '',
    specific_dates_text: item?.specific_dates?.join(',') || '',
  }
}

function createEmptyFixedAssignment(): FixedAssignmentDraft {
  return {
    staff_id: '',
    pattern_type: 'weekly',
    weekdays: [1, 2, 3, 4, 5],
    week_pattern: 'every',
    monthdays: [],
    specific_dates: [],
    start_date: undefined,
    end_date: undefined,
    is_active: true,
    monthdays_text: '',
    specific_dates_text: '',
  }
}

function resetFixedAssignmentDraft() {
  Object.assign(fixedAssignmentDraft, createEmptyFixedAssignment())
}

function getEmployeeDisplayName(staffId?: string) {
  if (!staffId) {
    return '-'
  }
  const employee = employeeOptions.value.find(item => item.id === staffId)
  if (!employee) {
    return staffId
  }
  return `${employee.name}${employee.employee_no ? `（${employee.employee_no}）` : ''}`
}

function getPatternTypeLabel(patternType: EditableFixedAssignment['pattern_type']) {
  switch (patternType) {
    case 'monthly':
      return '按月重复'
    case 'specific':
      return '指定日期'
    default:
      return '按周重复'
  }
}

function getWeekPatternLabel(value?: EditableFixedAssignment['week_pattern']) {
  switch (value) {
    case 'odd':
      return '奇数周'
    case 'even':
      return '偶数周'
    default:
      return '每周'
  }
}

function getWeekdayLabels(weekdays?: number[]) {
  return (weekdays || [])
    .map((weekday) => {
      const index = WEEKDAY_DISPLAY_ORDER.indexOf(weekday)
      return index >= 0 ? WEEKDAY_DISPLAY_NAMES[index] : ''
    })
    .filter(Boolean)
}

function formatFixedRule(item: EditableFixedAssignment) {
  if (item.pattern_type === 'monthly') {
    const monthdays = parseNumberList(item.monthdays_text)
    return monthdays.length ? `每月 ${monthdays.join('、')} 日` : '-'
  }
  if (item.pattern_type === 'specific') {
    const dates = parseStringList(item.specific_dates_text)
    return dates.length ? dates.join('、') : '-'
  }
  const weekdays = getWeekdayLabels(item.weekdays)
  if (!weekdays.length) {
    return getWeekPatternLabel(item.week_pattern)
  }
  return `${getWeekPatternLabel(item.week_pattern)} · ${weekdays.join('、')}`
}

function formatEffectiveRange(item: EditableFixedAssignment) {
  const start = item.start_date || ''
  const end = item.end_date || ''
  if (!start && !end) {
    return '永久生效'
  }
  if (start && end) {
    return `${start} 至 ${end}`
  }
  if (start) {
    return `${start} 起生效`
  }
  return `截止 ${end}`
}

function validateFixedAssignmentDraft(item: FixedAssignmentDraft) {
  if (!item.staff_id) {
    return '请选择人员'
  }
  if (item.pattern_type === 'weekly' && !(item.weekdays || []).length) {
    return '请选择重复周几'
  }
  if (item.pattern_type === 'monthly' && !parseNumberList(item.monthdays_text).length) {
    return '请输入每月执行日期'
  }
  if (item.pattern_type === 'specific' && !parseStringList(item.specific_dates_text).length) {
    return '请输入指定日期'
  }
  if (item.start_date && item.end_date && item.start_date > item.end_date) {
    return '开始日期不能晚于结束日期'
  }
  return ''
}

function cloneDraftToAssignment(item: FixedAssignmentDraft): EditableFixedAssignment {
  return toEditableAssignment({
    staff_id: item.staff_id,
    pattern_type: item.pattern_type,
    weekdays: [...(item.weekdays || [])],
    week_pattern: item.week_pattern,
    monthdays: parseNumberList(item.monthdays_text),
    specific_dates: parseStringList(item.specific_dates_text),
    start_date: item.start_date,
    end_date: item.end_date,
    is_active: item.is_active,
  })
}

function parseNumberList(value: string) {
  return value
    .split(',')
    .map(item => Number(item.trim()))
    .filter(item => !Number.isNaN(item))
}

function parseStringList(value: string) {
  return value
    .split(',')
    .map(item => item.trim())
    .filter(Boolean)
}

function toAssignmentPayload(item: EditableFixedAssignment): FixedAssignment {
  return {
    id: item.id,
    shift_id: item.shift_id,
    staff_id: item.staff_id,
    pattern_type: item.pattern_type,
    weekdays: item.pattern_type === 'weekly' ? (item.weekdays || []) : undefined,
    week_pattern: item.pattern_type === 'weekly' ? item.week_pattern : undefined,
    monthdays: item.pattern_type === 'monthly' ? parseNumberList(item.monthdays_text) : undefined,
    specific_dates: item.pattern_type === 'specific' ? parseStringList(item.specific_dates_text) : undefined,
    start_date: item.start_date || undefined,
    end_date: item.end_date || undefined,
    is_active: item.is_active,
  }
}

async function handleManageFixedAssignments(row: Shift) {
  currentShift.value = row
  fixedDialogVisible.value = true
  fixedDialogLoading.value = true
  try {
    const [employees, assignments] = await Promise.all([
      listEmployees({ page: 1, page_size: 500 }),
      getFixedAssignments(row.id),
    ])
    employeeOptions.value = employees.items || []
    fixedAssignments.value = assignments.map(item => toEditableAssignment(item))
    fixedAssignmentsCurrentPage.value = 1
    resetFixedAssignmentDraft()
  }
  catch (e: any) {
    ElMessage.error(e?.response?.data?.message || '加载固定人员失败')
    fixedDialogVisible.value = false
  }
  finally {
    fixedDialogLoading.value = false
  }
}

function addFixedAssignmentRow() {
  const message = validateFixedAssignmentDraft(fixedAssignmentDraft)
  if (message) {
    ElMessage.warning(message)
    return
  }
  fixedAssignments.value.push(cloneDraftToAssignment(fixedAssignmentDraft))
  fixedAssignmentsCurrentPage.value = Math.max(1, Math.ceil(fixedAssignments.value.length / fixedAssignmentsPageSize.value))
  resetFixedAssignmentDraft()
  ElMessage.success('已加入待保存列表')
}

function removeFixedAssignmentRow(item: EditableFixedAssignment) {
  fixedAssignments.value = fixedAssignments.value.filter(current => current !== item)
  const maxPage = Math.max(1, Math.ceil(fixedAssignments.value.length / fixedAssignmentsPageSize.value))
  if (fixedAssignmentsCurrentPage.value > maxPage) {
    fixedAssignmentsCurrentPage.value = maxPage
  }
}

async function refreshFixedAssignments() {
  if (!currentShift.value) {
    return
  }
  fixedDialogLoading.value = true
  try {
    fixedAssignments.value = (await getFixedAssignments(currentShift.value.id)).map(item => toEditableAssignment(item))
    fixedAssignmentsCurrentPage.value = 1
  }
  catch (e: any) {
    ElMessage.error(e?.response?.data?.message || '刷新固定人员失败')
  }
  finally {
    fixedDialogLoading.value = false
  }
}

async function saveFixedAssignments() {
  if (!currentShift.value) {
    return
  }
  fixedDialogSaving.value = true
  try {
    await batchCreateFixedAssignments(currentShift.value.id, fixedAssignments.value.map(toAssignmentPayload))
    ElMessage.success('固定人员已更新')
    fixedDialogVisible.value = false
    refresh()
  }
  catch (e: any) {
    ElMessage.error(e?.response?.data?.message || '保存固定人员失败')
  }
  finally {
    fixedDialogSaving.value = false
  }
}

async function handleManageWeeklyStaff(row: Shift) {
  currentShift.value = row
  weeklyDialogVisible.value = true
  weeklyDialogLoading.value = true
  try {
    const config = await getShiftWeeklyStaff(row.id)
    weeklyConfig.value = config?.weekly_config?.length
      ? config
      : { shift_id: row.id, shift_name: row.name, weekly_config: createDefaultWeeklyConfig() }
    syncWeeklyModeState()
  }
  catch (e: any) {
    ElMessage.error(e?.response?.data?.message || '加载周人数配置失败')
    weeklyDialogVisible.value = false
  }
  finally {
    weeklyDialogLoading.value = false
  }
}

async function saveWeeklyStaff() {
  if (!currentShift.value) {
    return
  }
  weeklyDialogSaving.value = true
  try {
    await updateShiftWeeklyStaff(currentShift.value.id, {
      ...weeklyConfig.value,
      weekly_config: weeklyConfig.value.weekly_config.map(item => ({
        ...item,
        is_custom: !weeklyUniformMode.value,
      })),
    })
    ElMessage.success('周人数配置已更新')
    weeklyDialogVisible.value = false
    refresh()
  }
  catch (e: any) {
    ElMessage.error(e?.response?.data?.message || '保存周人数配置失败')
  }
  finally {
    weeklyDialogSaving.value = false
  }
}

function isUniformWeeklyConfig(config: WeeklyStaff['weekly_config']) {
  if (!config.length) {
    return true
  }
  const firstCount = config[0]?.staff_count || 0
  return config.every(item => !item.is_custom && item.staff_count === firstCount)
}

function syncWeeklyModeState() {
  const config = weeklyConfig.value.weekly_config
  const isUniform = isUniformWeeklyConfig(config)
  weeklyUniformMode.value = isUniform
  weeklyUniformStaffCount.value = config[0]?.staff_count || 0
  if (isUniform) {
    applyUniformWeeklyStaffCount(weeklyUniformStaffCount.value)
  }
}

function applyUniformWeeklyStaffCount(staffCount: number) {
  weeklyConfig.value.weekly_config = weeklyConfig.value.weekly_config.map(item => ({
    ...item,
    staff_count: staffCount,
    is_custom: false,
  }))
}

function handleWeeklyUniformModeChange(enabled: boolean) {
  if (!weeklyConfig.value.weekly_config.length) {
    weeklyConfig.value.weekly_config = createDefaultWeeklyConfig()
  }
  if (enabled) {
    applyUniformWeeklyStaffCount(weeklyUniformStaffCount.value)
    return
  }
  weeklyConfig.value.weekly_config = weeklyConfig.value.weekly_config.map(item => ({
    ...item,
    is_custom: true,
  }))
}

function handleWeeklyUniformStaffCountChange(value: number | undefined) {
  weeklyUniformStaffCount.value = value ?? 0
  if (weeklyUniformMode.value) {
    applyUniformWeeklyStaffCount(weeklyUniformStaffCount.value)
  }
}
</script>

<template>
  <div class="page-container">
    <div class="page-toolbar">
      <div class="toolbar-filters">
        <el-input
          v-model="keyword"
          placeholder="搜索班次名称、编码"
          clearable
          style="width: 260px"
          :prefix-icon="Search"
        />
        <el-select v-model="typeFilter" clearable placeholder="类型" style="width: 150px">
          <el-option v-for="item in SHIFT_TYPE_OPTIONS" :key="item.value" :label="item.label" :value="item.value" />
        </el-select>
        <el-select v-model="statusFilter" clearable placeholder="状态" style="width: 150px">
          <el-option label="启用" value="active" />
          <el-option label="禁用" value="disabled" />
        </el-select>
      </div>
      <el-button v-if="canManageShifts" type="primary" :icon="Plus" @click="handleAdd">
        新增班次
      </el-button>
    </div>

    <el-table v-loading="loading" :data="pagedItems" border stripe style="width: 100%">
      <el-table-column prop="code" label="编码" width="100" />
      <el-table-column prop="name" label="名称" min-width="160">
        <template #default="{ row }">
          <div class="name-cell">
            <span class="color-dot" :style="{ background: row.color }" />
            <span>{{ row.name }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="type" label="类型" width="100">
        <template #default="{ row }">
          <el-tag :type="getShiftTypeTagType(row.type) as any" size="small">
            {{ getShiftTypeText(row.type) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="start_time" label="开始时间" width="100" />
      <el-table-column prop="end_time" label="结束时间" width="100" />
      <el-table-column label="时长" width="100">
        <template #default="{ row }">
          <el-tag size="small" effect="plain">
            {{ formatDuration(row.duration) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="排班优先级" width="110">
        <template #default="{ row }">
          <el-tag size="small" effect="plain" type="warning">
            {{ row.scheduling_priority || 0 }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="固定人员" width="120">
        <template #default="{ row }">
          {{ row.fixed_staff_summary || '-' }}
        </template>
      </el-table-column>
      <el-table-column label="周人数配置" width="180">
        <template #default="{ row }">
          <el-tag size="small" type="success" effect="plain">
            {{ row.weekly_staff_summary || '-' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="颜色" width="80">
        <template #default="{ row }">
          <span class="color-block" :style="{ background: row.color }" />
        </template>
      </el-table-column>
      <el-table-column label="状态" width="90">
        <template #default="{ row }">
          <el-tag :type="getShiftStatusTagType(row.is_active) as any" size="small">
            {{ getShiftStatusText(row.is_active) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="关联分组" min-width="220" show-overflow-tooltip>
        <template #default="{ row }">
          {{ row.group_names?.length ? row.group_names.join('、') : (row.group_summary || '-') }}
        </template>
      </el-table-column>
      <el-table-column prop="description" label="描述" min-width="180" show-overflow-tooltip />
      <el-table-column v-if="canManageShifts" label="操作" width="360" fixed="right">
        <template #default="{ row }">
          <el-button :icon="Edit" link type="primary" @click="handleEdit(row)">
            编辑
          </el-button>
          <el-button link type="success" @click="handleManageFixedAssignments(row)">
            固定人员
          </el-button>
          <el-button link type="primary" @click="handleManageWeeklyStaff(row)">
            人数
          </el-button>
          <el-button link type="primary" @click="handleManageGroups(row)">
            分组
          </el-button>
          <el-button link :type="row.is_active ? 'warning' : 'success'" @click="handleToggleStatus(row)">
            {{ row.is_active ? '禁用' : '启用' }}
          </el-button>
          <el-button :icon="Delete" link type="danger" @click="handleDelete(row)">
            删除
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="page-pagination">
      <el-pagination
        v-model:current-page="currentPage"
        v-model:page-size="currentPageSize"
        :total="filteredTotal"
        :page-sizes="[10, 20, 50]"
        layout="total, sizes, prev, pager, next"
        @current-change="handlePageChange"
        @size-change="handleSizeChange"
      />
    </div>

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="620px">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <div class="form-grid">
          <el-form-item label="编码" prop="code">
            <el-input v-model="form.code" placeholder="如：302" />
          </el-form-item>
          <el-form-item label="名称" prop="name">
            <el-input v-model="form.name" placeholder="如：江北穿刺" />
          </el-form-item>
          <el-form-item label="类型" prop="type">
            <el-select v-model="form.type" placeholder="选择类型" style="width: 100%">
              <el-option v-for="item in SHIFT_TYPE_OPTIONS" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
          </el-form-item>
          <el-form-item label="排班优先级">
            <el-input-number v-model="form.scheduling_priority" :min="0" :max="999" style="width: 100%" />
          </el-form-item>
          <el-form-item label="开始时间" prop="start_time">
            <el-time-select v-model="form.start_time" start="00:00" step="00:30" end="23:30" placeholder="选择时间" style="width: 100%" />
          </el-form-item>
          <el-form-item label="结束时间" prop="end_time">
            <el-time-select v-model="form.end_time" start="00:00" step="00:30" end="23:30" placeholder="选择时间" style="width: 100%" />
          </el-form-item>
          <el-form-item label="时长">
            <el-input :model-value="computedDurationText" disabled />
          </el-form-item>
          <el-form-item label="是否跨天">
            <el-switch :model-value="computedCrossDay" disabled />
          </el-form-item>
          <el-form-item label="颜色">
            <el-color-picker v-model="form.color" />
          </el-form-item>
          <el-form-item label="是否启用">
            <el-switch v-model="form.is_active" />
          </el-form-item>
        </div>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="3" placeholder="可选" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">
          取消
        </el-button>
        <el-button type="primary" :loading="formLoading" @click="handleSubmit">
          确定
        </el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="groupDialogVisible" :title="currentShift ? `${currentShift.name} · 关联分组` : '关联分组'" width="560px">
      <div v-loading="groupDialogLoading" class="group-dialog-body">
        <el-select v-model="selectedGroupIds" multiple filterable placeholder="选择关联分组" style="width: 100%">
          <el-option v-for="group in groupOptions" :key="group.id" :label="group.name" :value="group.id" />
        </el-select>
        <div v-if="selectedGroups.length" class="selected-group-panel">
          <div class="selected-group-label">
            已选分组
          </div>
          <div class="selected-group-list">
            <el-tag v-for="group in selectedGroups" :key="group.id" effect="plain" class="selected-group-tag">
              {{ group.name }}
            </el-tag>
          </div>
        </div>
        <div v-else class="selected-group-empty">
          当前未选择分组
        </div>
      </div>
      <template #footer>
        <el-button @click="groupDialogVisible = false">
          取消
        </el-button>
        <el-button type="primary" :loading="groupDialogSaving" @click="saveShiftGroups">
          保存
        </el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="fixedDialogVisible" class="fixed-assignment-dialog" :title="currentShift ? `配置固定人员 - ${currentShift.name}` : '配置固定人员'" width="1120px" top="2vh">
      <div v-loading="fixedDialogLoading" class="fixed-dialog-body">
        <div class="fixed-dialog-section">
          <div class="fixed-dialog-header">
            <div class="fixed-dialog-title">
              已配置人员（{{ fixedAssignments.length }}）
            </div>
            <el-button :icon="RefreshRight" text @click="refreshFixedAssignments">
              刷新
            </el-button>
          </div>
          <el-table v-if="fixedAssignments.length" :data="pagedFixedAssignments" border stripe height="180" style="width: 100%">
            <el-table-column label="人员" min-width="180">
              <template #default="{ row }">
                {{ row.staff_name || getEmployeeDisplayName(row.staff_id) }}
              </template>
            </el-table-column>
            <el-table-column label="模式" width="140">
              <template #default="{ row }">
                {{ getPatternTypeLabel(row.pattern_type) }}
              </template>
            </el-table-column>
            <el-table-column label="规则" min-width="320">
              <template #default="{ row }">
                {{ formatFixedRule(row) }}
              </template>
            </el-table-column>
            <el-table-column label="生效时间" min-width="220">
              <template #default="{ row }">
                {{ formatEffectiveRange(row) }}
              </template>
            </el-table-column>
            <el-table-column label="状态" width="90">
              <template #default="{ row }">
                <el-tag :type="row.is_active ? 'success' : 'info'" size="small">
                  {{ row.is_active ? '启用' : '停用' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="90">
              <template #default="{ row }">
                <el-button link type="danger" @click="removeFixedAssignmentRow(row)">
                  删除
                </el-button>
              </template>
            </el-table-column>
          </el-table>
          <div v-else class="fixed-assignment-empty">
            <el-empty :image-size="68" description="暂无固定人员配置" />
          </div>
          <div v-if="fixedAssignments.length" class="fixed-assignment-pagination">
            <el-pagination
              v-model:current-page="fixedAssignmentsCurrentPage"
              v-model:page-size="fixedAssignmentsPageSize"
              small
              background
              layout="total, prev, pager, next"
              :total="fixedAssignments.length"
            />
          </div>
        </div>

        <div class="fixed-dialog-section fixed-dialog-form-section">
          <div class="fixed-dialog-title">
            添加固定人员
          </div>
          <div class="fixed-form-grid">
            <div class="fixed-form-row">
              <label class="fixed-form-label fixed-required">选择人员</label>
              <el-select v-model="fixedAssignmentDraft.staff_id" filterable placeholder="请选择人员" style="width: 100%">
                <el-option v-for="employee in employeeOptions" :key="employee.id" :label="`${employee.name}${employee.employee_no ? `（${employee.employee_no}）` : ''}`" :value="employee.id" />
              </el-select>
            </div>

            <div class="fixed-form-row">
              <label class="fixed-form-label fixed-required">配置模式</label>
              <el-radio-group v-model="fixedAssignmentDraft.pattern_type">
                <el-radio value="weekly">
                  按周重复
                </el-radio>
                <el-radio value="monthly">
                  按月重复
                </el-radio>
                <el-radio value="specific">
                  指定日期
                </el-radio>
              </el-radio-group>
            </div>

            <template v-if="fixedAssignmentDraft.pattern_type === 'weekly'">
              <div class="fixed-form-row">
                <label class="fixed-form-label">周期</label>
                <el-radio-group v-model="fixedAssignmentDraft.week_pattern">
                  <el-radio value="every">
                    每周
                  </el-radio>
                  <el-radio value="odd">
                    奇数周
                  </el-radio>
                  <el-radio value="even">
                    偶数周
                  </el-radio>
                </el-radio-group>
              </div>
              <div class="fixed-form-row">
                <label class="fixed-form-label fixed-required">选择周几</label>
                <el-checkbox-group v-model="fixedAssignmentDraft.weekdays">
                  <el-checkbox v-for="(weekday, index) in WEEKDAY_DISPLAY_ORDER" :key="weekday" :label="weekday">
                    {{ WEEKDAY_DISPLAY_NAMES[index] }}
                  </el-checkbox>
                </el-checkbox-group>
              </div>
            </template>

            <div v-else-if="fixedAssignmentDraft.pattern_type === 'monthly'" class="fixed-form-row">
              <label class="fixed-form-label fixed-required">执行日期</label>
              <el-input v-model="fixedAssignmentDraft.monthdays_text" placeholder="请输入每月日期，如 1,15,28" />
            </div>

            <div v-else class="fixed-form-row">
              <label class="fixed-form-label fixed-required">指定日期</label>
              <el-input v-model="fixedAssignmentDraft.specific_dates_text" placeholder="请输入日期，如 2026-04-01,2026-04-15" />
            </div>

            <div class="fixed-form-row">
              <label class="fixed-form-label">生效时间</label>
              <div class="fixed-date-range">
                <el-date-picker v-model="fixedAssignmentDraft.start_date" type="date" value-format="YYYY-MM-DD" placeholder="开始日期" />
                <span class="fixed-date-separator">至</span>
                <el-date-picker v-model="fixedAssignmentDraft.end_date" type="date" value-format="YYYY-MM-DD" placeholder="结束日期" />
              </div>
            </div>

            <div class="fixed-form-tip">
              可选，不设置则永久生效
            </div>

            <div class="fixed-form-row">
              <label class="fixed-form-label">启用状态</label>
              <el-switch v-model="fixedAssignmentDraft.is_active" />
            </div>

            <div class="fixed-form-actions">
              <el-button type="primary" @click="addFixedAssignmentRow">
                + 添加到列表
              </el-button>
              <el-button @click="resetFixedAssignmentDraft">
                重置
              </el-button>
            </div>
          </div>
        </div>
      </div>
      <template #footer>
        <el-button @click="fixedDialogVisible = false">
          取消
        </el-button>
        <el-button type="primary" :loading="fixedDialogSaving" @click="saveFixedAssignments">
          保存配置
        </el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="weeklyDialogVisible" :title="currentShift ? `${currentShift.name} · 周人数配置` : '周人数配置'" width="700px">
      <div v-loading="weeklyDialogLoading">
        <div class="weekly-config-toolbar">
          <div class="weekly-config-mode">
            <span class="weekly-config-mode-label">一键配置</span>
            <el-switch
              v-model="weeklyUniformMode"
              active-text="所有天数一致"
              inactive-text="按天单独配置"
              @change="handleWeeklyUniformModeChange"
            />
          </div>
          <div v-if="weeklyUniformMode" class="weekly-config-bulk">
            <span class="weekly-config-bulk-label">统一人数</span>
            <el-input-number
              v-model="weeklyUniformStaffCount"
              :min="0"
              :max="999"
              @change="handleWeeklyUniformStaffCountChange"
            />
          </div>
        </div>
        <el-table :data="weeklyConfig.weekly_config" border stripe>
          <el-table-column label="星期" width="120">
            <template #default="{ row }">
              {{ row.weekday_name || `周${row.weekday}` }}
            </template>
          </el-table-column>
          <el-table-column label="人数配置" width="180">
            <template #default="{ row }">
              <el-input-number v-model="row.staff_count" :min="0" :max="999" :disabled="weeklyUniformMode" />
            </template>
          </el-table-column>
        </el-table>
      </div>
      <template #footer>
        <el-button @click="weeklyDialogVisible = false">
          取消
        </el-button>
        <el-button type="primary" :loading="weeklyDialogSaving" @click="saveWeeklyStaff">
          保存
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.page-container {
  height: 100%;
  display: flex;
  flex-direction: column;
  padding: 24px;
  overflow: hidden;
}

.page-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  margin-bottom: 16px;
}

.toolbar-filters {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.page-pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

.name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.color-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  flex-shrink: 0;
}

.color-block {
  display: inline-block;
  width: 28px;
  height: 18px;
  border-radius: 4px;
  border: 1px solid var(--el-border-color);
}

.group-dialog-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.selected-group-panel {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-fill-color-light);
  padding: 12px;
}

.selected-group-label {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  margin-bottom: 8px;
}

.selected-group-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.selected-group-tag {
  max-width: 100%;
}

.selected-group-empty {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 16px;
}

.fixed-config-cell {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.fixed-dialog-section {
  padding-bottom: 12px;
}

.fixed-dialog-section + .fixed-dialog-section {
  margin-top: 12px;
  padding-top: 16px;
  border-top: 1px solid var(--el-border-color-lighter);
}

.fixed-dialog-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.fixed-dialog-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.fixed-dialog-form-section {
  padding-bottom: 0;
}

.fixed-assignment-empty {
  height: 120px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 4px;
  background: var(--el-fill-color-blank);
}

.fixed-assignment-pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 8px;
}

.fixed-form-grid {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.fixed-form-row {
  display: flex;
  align-items: center;
  gap: 16px;
}

.fixed-form-label {
  width: 88px;
  flex-shrink: 0;
  color: var(--el-text-color-regular);
}

.fixed-required::before {
  content: '*';
  color: var(--el-color-danger);
  margin-right: 4px;
}

.fixed-date-range {
  display: flex;
  align-items: center;
  gap: 12px;
}

.fixed-date-separator {
  color: var(--el-text-color-secondary);
}

.fixed-form-tip {
  margin-left: 104px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.fixed-form-actions {
  margin-left: 104px;
  display: flex;
  gap: 12px;
}

.fixed-dialog-body {
  overflow-y: auto;
  overflow-x: hidden;
  max-height: calc(100vh - 300px);
  padding-right: 4px;
}

.weekly-config-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}

.weekly-config-mode {
  display: flex;
  align-items: center;
  gap: 12px;
}

.weekly-config-mode-label,
.weekly-config-bulk-label {
  color: var(--el-text-color-regular);
  font-size: 14px;
}

.weekly-config-bulk {
  display: flex;
  align-items: center;
  gap: 12px;
}

.fixed-config-row {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.fixed-config-row-inline {
  align-items: center;
}

.fixed-config-label {
  min-width: 36px;
  color: var(--el-text-color-secondary);
  line-height: 32px;
  flex-shrink: 0;
}

:deep(.fixed-assignment-dialog .el-dialog) {
  max-width: calc(100vw - 48px);
  max-height: calc(100vh - 32px);
  display: flex;
  flex-direction: column;
}

:deep(.fixed-assignment-dialog .el-dialog__body) {
  padding-top: 12px;
  padding-bottom: 8px;
  flex: 1;
  min-height: 0;
}

:deep(.fixed-assignment-dialog .el-dialog__header) {
  padding-bottom: 12px;
}

:deep(.fixed-assignment-dialog .el-dialog__footer) {
  padding-top: 12px;
}

@media (max-width: 900px) {
  .page-toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .weekly-config-toolbar {
    flex-direction: column;
    align-items: flex-start;
  }

  .form-grid {
    grid-template-columns: 1fr;
  }

  .fixed-form-row {
    flex-wrap: wrap;
    align-items: flex-start;
  }

  .fixed-form-label,
  .fixed-form-tip,
  .fixed-form-actions {
    width: 100%;
    margin-left: 0;
  }

  .fixed-date-range {
    flex-wrap: wrap;
  }

  .fixed-dialog-body {
    max-height: calc(100vh - 240px);
  }
}
</style>
