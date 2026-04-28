<script setup lang="ts">
import type { AIConversationMessage, AIConversationSummary } from '@/api/ai'
import type { Employee } from '@/types/employee'
import type { ScheduleAssignment, SchedulePlan } from '@/types/scheduling'
import type { Shift } from '@/types/shift'
import { ChatLineSquare, Close, Delete, Promotion } from '@element-plus/icons-vue'
import dayjs from 'dayjs'
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, nextTick, onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { chat, deleteChatConversation, getChatConversation, listChatConversations } from '@/api/ai'
import { listEmployees } from '@/api/employees'
import { listGroups } from '@/api/groups'
import { adjustAssignments, createSchedule, generateSchedule, getAssignments, listSchedules } from '@/api/schedules'
import { listShifts } from '@/api/shifts'

interface ChatAction {
  id: string
  type: string
  label: string
  payload?: Record<string, unknown>
}

interface MessageItem {
  id: string
  role: 'user' | 'assistant' | 'system'
  content: string
  created_at: string
  actions?: ChatAction[]
  sourcePrompt?: string
}

interface ExternalPromptOptions {
  newSession?: boolean
  autoActionType?: string
}

type AdjustmentOperation = 'replace' | 'add' | 'remove' | 'swap'

interface AdjustmentForm {
  scheduleId: string
  operation: AdjustmentOperation
  assignmentId: string
  secondAssignmentId: string
  targetEmployeeId: string
  targetShiftId: string
  targetDate: string
}

interface ResolvedDateRange {
  startDate: string
  endDate: string
}

interface AdjustmentSeed {
  range: ResolvedDateRange | null
  employeeName: string
  shiftName: string
}

const props = withDefaults(defineProps<{
  embedded?: boolean
  showClose?: boolean
}>(), {
  embedded: false,
  showClose: false,
})

const emit = defineEmits<{
  close: []
}>()

const router = useRouter()
const sessions = ref<AIConversationSummary[]>([])
const currentSessionId = ref<string | null>(null)
const messages = ref<MessageItem[]>([])
const inputText = ref('')
const sending = ref(false)
const actionPending = ref(false)
const loadingSessions = ref(false)
const loadingConversation = ref(false)
const messagesContainer = ref<HTMLElement>()

const employeeOptions = ref<Employee[]>([])
const shiftOptions = ref<Shift[]>([])
const adjustmentPanelVisible = ref(false)
const adjustmentPanelBusy = ref(false)
const adjustmentScheduleOptions = ref<SchedulePlan[]>([])
const adjustmentAssignments = ref<ScheduleAssignment[]>([])
const adjustmentSeed = ref<AdjustmentSeed | null>(null)

const adjustmentForm = reactive<AdjustmentForm>({
  scheduleId: '',
  operation: 'replace',
  assignmentId: '',
  secondAssignmentId: '',
  targetEmployeeId: '',
  targetShiftId: '',
  targetDate: '',
})

const adjustmentOperationOptions = [
  { label: '替换班次', value: 'replace' as const },
  { label: '新增班次', value: 'add' as const },
  { label: '移除班次', value: 'remove' as const },
  { label: '对调人员', value: 'swap' as const },
]

const currentAdjustmentSchedule = computed(() => adjustmentScheduleOptions.value.find(item => item.id === adjustmentForm.scheduleId) || null)
const currentAdjustmentAssignment = computed(() => adjustmentAssignments.value.find(item => item.id === adjustmentForm.assignmentId) || null)
const secondaryAssignmentOptions = computed(() => {
  if (!adjustmentForm.assignmentId) {
    return adjustmentAssignments.value
  }

  const currentAssignment = currentAdjustmentAssignment.value
  return adjustmentAssignments.value.filter((item) => {
    if (item.id === adjustmentForm.assignmentId) {
      return false
    }
    if (!currentAssignment) {
      return true
    }
    return item.date === currentAssignment.date
  })
})

const scheduleDateOptions = computed(() => {
  const schedule = currentAdjustmentSchedule.value
  if (!schedule) {
    return []
  }

  const result: Array<{ label: string, value: string }> = []
  let cursor = dayjs(schedule.start_date)
  const end = dayjs(schedule.end_date)

  while (!cursor.isAfter(end, 'day')) {
    result.push({
      label: cursor.format('MM-DD ddd'),
      value: cursor.format('YYYY-MM-DD'),
    })
    cursor = cursor.add(1, 'day')
  }

  return result
})

const adjustmentSummary = computed(() => {
  const assignment = currentAdjustmentAssignment.value
  const employeeName = getEmployeeName(adjustmentForm.targetEmployeeId)
  const shiftName = getShiftName(adjustmentForm.targetShiftId)
  const targetDate = adjustmentForm.targetDate

  if (adjustmentForm.operation === 'add') {
    if (!employeeName || !shiftName || !targetDate) {
      return '选择员工、班次和日期后，可直接在 AI 助手内提交新增。'
    }
    return `将新增 ${targetDate} 的 ${shiftName} 给 ${employeeName}。`
  }

  if (adjustmentForm.operation === 'remove') {
    if (!assignment) {
      return '选择一条现有排班后，可直接移除。'
    }
    return `将移除 ${assignment.date} ${assignment.employee_name} 的 ${assignment.shift_name}。`
  }

  if (adjustmentForm.operation === 'swap') {
    const secondAssignment = adjustmentAssignments.value.find(item => item.id === adjustmentForm.secondAssignmentId)
    if (!assignment || !secondAssignment) {
      return '选择同一天的两条排班后，可直接对调人员。'
    }
    return `将对调 ${assignment.date} 的 ${assignment.employee_name}/${assignment.shift_name} 与 ${secondAssignment.employee_name}/${secondAssignment.shift_name}。`
  }

  if (!assignment) {
    return '选择一条现有排班后，可修改人员、班次或日期。'
  }

  const summaryParts: string[] = []
  if (employeeName && employeeName !== assignment.employee_name) {
    summaryParts.push(`人员改为 ${employeeName}`)
  }
  if (shiftName && shiftName !== assignment.shift_name) {
    summaryParts.push(`班次改为 ${shiftName}`)
  }
  if (targetDate && targetDate !== assignment.date) {
    summaryParts.push(`日期改为 ${targetDate}`)
  }

  if (summaryParts.length === 0) {
    return `当前选中 ${assignment.date} ${assignment.employee_name} 的 ${assignment.shift_name}，请选择需要修改的内容。`
  }
  return `将把 ${assignment.date} ${assignment.employee_name} 的 ${assignment.shift_name} 调整为：${summaryParts.join('，')}。`
})

function buildWelcomeMessages(): MessageItem[] {
  return [{
    id: 'welcome',
    role: 'system',
    content: '你好！我是 AI 排班助手。创建排班、查询排班、调整排班都在这里完成；排班工作台现在只负责手工查看、微调和导出。',
    created_at: new Date().toISOString(),
  }]
}

function resetDraftSession() {
  currentSessionId.value = null
  messages.value = buildWelcomeMessages()
}

function hasConversationMessages(items: MessageItem[] = messages.value) {
  return items.some(item => item.role === 'user' || item.role === 'assistant')
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === 'object' && !Array.isArray(value)
}

function normalizeCollection<T>(value: unknown): T[] {
  if (Array.isArray(value)) {
    return value as T[]
  }
  if (isRecord(value) && Array.isArray(value.items)) {
    return value.items as T[]
  }
  return []
}

function normalizeActions(actions?: Array<{
  type?: string
  label?: string
  payload?: Record<string, unknown>
  data?: Record<string, unknown>
}>) {
  return (actions || [])
    .filter(action => action && action.type && action.label)
    .map((action, index) => ({
      id: `${action.type}-${Date.now()}-${index}`,
      type: action.type!,
      label: action.label!,
      payload: action.payload || action.data || {},
    }))
}

function createLocalActions(actions: Array<{ type: string, label: string, payload?: Record<string, unknown> }>) {
  return actions.map((action, index) => ({
    id: `${action.type}-${Date.now()}-${index}`,
    type: action.type,
    label: action.label,
    payload: action.payload || {},
  }))
}

function toMessageItem(message: AIConversationMessage): MessageItem {
  return {
    id: message.id,
    role: message.role,
    content: message.content,
    created_at: message.created_at,
  }
}

function sortSessions(items: AIConversationSummary[]) {
  return [...items].sort((left, right) => {
    const leftTime = dayjs(left.updated_at || left.last_message_at || left.created_at).valueOf()
    const rightTime = dayjs(right.updated_at || right.last_message_at || right.created_at).valueOf()
    return rightTime - leftTime
  })
}

function sortPlans(items: SchedulePlan[]) {
  return [...items].sort((left, right) => dayjs(right.updated_at).valueOf() - dayjs(left.updated_at).valueOf())
}

function formatConversationTitle(session: AIConversationSummary) {
  const source = session.created_at || session.updated_at || session.last_message_at
  if (!source) {
    return session.title || '未命名会话'
  }
  return dayjs(source).format('MM-DD HH:mm')
}

function formatConversationMeta(session: AIConversationSummary) {
  if (session.message_count > 0) {
    return `共 ${session.message_count} 条消息`
  }
  return '空会话'
}

function normalizeDateToken(token: string) {
  const [year, month, day] = token.replace(/\//g, '-').split('-')
  if (!year || !month || !day) {
    return token
  }
  return `${year.padStart(4, '0')}-${month.padStart(2, '0')}-${day.padStart(2, '0')}`
}

function extractEntities(action: ChatAction) {
  const payload = action.payload
  if (!isRecord(payload)) {
    return {}
  }
  const nested = payload.entities
  if (isRecord(nested)) {
    return nested
  }
  return payload
}

function parseDateRangeFromText(text: string) {
  if (!text) {
    return null
  }

  const rangeMatch = text.match(/(\d{4}[\/-]\d{1,2}[\/-]\d{1,2})\s*(?:至|到|~|—|–)\s*(\d{4}[\/-]\d{1,2}[\/-]\d{1,2})/)
  if (rangeMatch) {
    return {
      startDate: normalizeDateToken(rangeMatch[1]),
      endDate: normalizeDateToken(rangeMatch[2]),
    }
  }

  const dateTokens = text.match(/\d{4}[\/-]\d{1,2}[\/-]\d{1,2}/g) || []
  if (dateTokens.length >= 2) {
    return {
      startDate: normalizeDateToken(dateTokens[0]),
      endDate: normalizeDateToken(dateTokens[1]),
    }
  }

  return null
}

function resolveDateRange(action: ChatAction, message: MessageItem) {
  const entities = extractEntities(action)
  const candidates = [
    typeof entities.date_range === 'string' ? entities.date_range : '',
    message.sourcePrompt || '',
    message.content || '',
  ]

  for (const candidate of candidates) {
    const range = parseDateRangeFromText(candidate)
    if (range) {
      return range
    }
  }

  return null
}

function buildScheduleName(action: ChatAction, range: ResolvedDateRange) {
  const entities = extractEntities(action)
  const department = typeof entities.department === 'string' ? entities.department.trim() : ''
  const prefix = department ? `${department} ` : ''
  return `${prefix}${range.startDate} 至 ${range.endDate} 排班`
}

function normalizeGroupItems(value: unknown) {
  if (Array.isArray(value)) {
    return value
  }
  if (isRecord(value) && Array.isArray(value.items)) {
    return value.items
  }
  return []
}

function getEmployeeName(employeeId: string) {
  return employeeOptions.value.find(item => item.id === employeeId)?.name || ''
}

function getShiftName(shiftId: string) {
  return shiftOptions.value.find(item => item.id === shiftId)?.name || ''
}

function formatScheduleStatus(status: string) {
  const statusMap: Record<string, string> = {
    draft: '草稿',
    generating: '生成中',
    generated: '已生成',
    published: '已发布',
    final: '已定稿',
  }
  return statusMap[status] || status
}

function formatScheduleOption(plan: SchedulePlan) {
  return `${plan.name} · ${plan.start_date} ~ ${plan.end_date} · ${formatScheduleStatus(plan.status)}`
}

function formatAssignmentOption(assignment: ScheduleAssignment) {
  return `${assignment.date} · ${assignment.employee_name} · ${assignment.shift_name}`
}

function overlapsRange(plan: SchedulePlan, range: ResolvedDateRange | null) {
  if (!range) {
    return true
  }
  return !(plan.end_date < range.startDate || plan.start_date > range.endDate)
}

function findBestScheduleId(plans: SchedulePlan[], range: ResolvedDateRange | null) {
  if (plans.length === 0) {
    return ''
  }
  if (!range) {
    return plans[0].id
  }
  const exactMatch = plans.find(plan => plan.start_date === range.startDate && plan.end_date === range.endDate)
  if (exactMatch) {
    return exactMatch.id
  }
  const overlapMatch = plans.find(plan => overlapsRange(plan, range))
  return overlapMatch?.id || plans[0].id
}

function matchEmployeeIdByName(name: string) {
  const normalized = name.trim()
  if (!normalized) {
    return ''
  }
  const exactMatch = employeeOptions.value.find(item => item.name.trim() === normalized)
  if (exactMatch) {
    return exactMatch.id
  }
  const fuzzyMatch = employeeOptions.value.find(item => item.name.includes(normalized) || normalized.includes(item.name))
  return fuzzyMatch?.id || ''
}

function matchShiftIdByName(name: string) {
  const normalized = name.trim()
  if (!normalized) {
    return ''
  }
  const exactMatch = shiftOptions.value.find(item => item.name.trim() === normalized)
  if (exactMatch) {
    return exactMatch.id
  }
  const fuzzyMatch = shiftOptions.value.find(item => item.name.includes(normalized) || normalized.includes(item.name))
  return fuzzyMatch?.id || ''
}

function buildScheduleResultActions(plan: SchedulePlan) {
  return [
    {
      type: 'open_schedule_detail',
      label: '查看排班详情',
      payload: { schedule_id: plan.id },
    },
    {
      type: 'open_workspace_range',
      label: '打开工作台',
      payload: {
        start_date: plan.start_date,
        end_date: plan.end_date,
      },
    },
  ]
}

function pushAssistantMessage(content: string, actions: Array<{ type: string, label: string, payload?: Record<string, unknown> }> = []) {
  messages.value.push({
    id: `local-assistant-${Date.now()}-${messages.value.length}`,
    role: 'assistant',
    content,
    created_at: new Date().toISOString(),
    actions: createLocalActions(actions),
  })
  scrollToBottom()
}

async function resolveGroupId(action: ChatAction) {
  const entities = extractEntities(action)
  const department = typeof entities.department === 'string' ? entities.department.trim() : ''
  if (!department) {
    return null
  }

  try {
    const groups = normalizeGroupItems(await listGroups({ page: 1, page_size: 200 }))
    const exactMatch = groups.find(group => isRecord(group) && typeof group.name === 'string' && group.name.trim() === department)
    if (isRecord(exactMatch) && typeof exactMatch.id === 'string') {
      return exactMatch.id
    }

    const fuzzyMatch = groups.find(group => isRecord(group)
      && typeof group.name === 'string'
      && (group.name.includes(department) || department.includes(group.name)))
    if (isRecord(fuzzyMatch) && typeof fuzzyMatch.id === 'string') {
      return fuzzyMatch.id
    }
  }
  catch {
    // ignore group lookup failure and fall back to unscoped schedule creation
  }

  return null
}

async function refreshSessions(selectedId: string | null = currentSessionId.value) {
  loadingSessions.value = true
  try {
    const items = await listChatConversations()
    sessions.value = sortSessions(items)
    if (selectedId && sessions.value.some(item => item.id === selectedId)) {
      currentSessionId.value = selectedId
      return
    }
    if (!currentSessionId.value && sessions.value.length > 0) {
      currentSessionId.value = sessions.value[0].id
    }
  }
  catch {
    ElMessage.error('加载会话列表失败')
  }
  finally {
    loadingSessions.value = false
  }
}

async function loadConversation(conversationId: string) {
  loadingConversation.value = true
  try {
    const detail = await getChatConversation(conversationId)
    currentSessionId.value = detail.id
    messages.value = detail.messages.length > 0 ? detail.messages.map(toMessageItem) : buildWelcomeMessages()
    scrollToBottom()
  }
  catch {
    ElMessage.error('加载会话失败')
  }
  finally {
    loadingConversation.value = false
  }
}

async function selectSession(session: AIConversationSummary) {
  if (session.id === currentSessionId.value && messages.value.length > 0) {
    return
  }
  await loadConversation(session.id)
}

async function startNewSession() {
  inputText.value = ''

  if (!hasConversationMessages()) {
    if (!currentSessionId.value) {
      resetDraftSession()
    }
    else {
      messages.value = buildWelcomeMessages()
    }
    scrollToBottom()
    return
  }

  const existingEmptySession = sessions.value.find(session => session.message_count === 0)
  if (existingEmptySession) {
    await loadConversation(existingEmptySession.id)
    inputText.value = ''
    return
  }

  resetDraftSession()
  scrollToBottom()
}

async function ensureAdjustmentResourcesLoaded() {
  if (employeeOptions.value.length > 0 && shiftOptions.value.length > 0) {
    return
  }

  const [employeesResponse, shiftsResponse] = await Promise.all([
    listEmployees({ page: 1, page_size: 500 }),
    listShifts({ page: 1, page_size: 200 }),
  ])

  employeeOptions.value = normalizeCollection<Employee>(employeesResponse)
  shiftOptions.value = normalizeCollection<Shift>(shiftsResponse).filter(shift => shift.is_active && shift.status !== 'disabled')
}

function resetAdjustmentForm() {
  adjustmentForm.scheduleId = ''
  adjustmentForm.operation = 'replace'
  adjustmentForm.assignmentId = ''
  adjustmentForm.secondAssignmentId = ''
  adjustmentForm.targetEmployeeId = ''
  adjustmentForm.targetShiftId = ''
  adjustmentForm.targetDate = ''
}

function closeAdjustmentPanel() {
  adjustmentPanelVisible.value = false
  adjustmentScheduleOptions.value = []
  adjustmentAssignments.value = []
  adjustmentSeed.value = null
  resetAdjustmentForm()
}

function prefillAssignmentFromSeed() {
  if (adjustmentForm.assignmentId || !adjustmentSeed.value) {
    return
  }

  const { employeeName, shiftName, range } = adjustmentSeed.value
  const matchedAssignment = adjustmentAssignments.value.find((item) => {
    const employeeMatched = !employeeName || item.employee_name.includes(employeeName) || employeeName.includes(item.employee_name)
    const shiftMatched = !shiftName || item.shift_name.includes(shiftName) || shiftName.includes(item.shift_name)
    const dateMatched = !range || (item.date >= range.startDate && item.date <= range.endDate)
    return employeeMatched && shiftMatched && dateMatched
  })

  if (matchedAssignment) {
    adjustmentForm.assignmentId = matchedAssignment.id
  }
}

async function loadAdjustmentAssignments(scheduleId: string) {
  if (!scheduleId) {
    adjustmentAssignments.value = []
    return
  }

  adjustmentPanelBusy.value = true
  try {
    adjustmentAssignments.value = await getAssignments(scheduleId)
    prefillAssignmentFromSeed()
  }
  catch {
    ElMessage.error('加载排班明细失败')
    adjustmentAssignments.value = []
  }
  finally {
    adjustmentPanelBusy.value = false
  }
}

async function runCreateScheduleAction(action: ChatAction, message: MessageItem) {
  const range = resolveDateRange(action, message)
  if (!range) {
    ElMessage.warning('缺少明确的排班日期，请在对话中补充起止日期后重试')
    return
  }

  actionPending.value = true
  try {
    const groupId = await resolveGroupId(action)
    const plan = await createSchedule({
      name: buildScheduleName(action, range),
      start_date: range.startDate,
      end_date: range.endDate,
      group_id: groupId || undefined,
    })
    await generateSchedule(plan.id)
    pushAssistantMessage(
      `已为你创建 ${range.startDate} 至 ${range.endDate} 的排班，并已开始生成。接下来你可以先查看详情，也可以在工作台继续做手工调整或导出。`,
      buildScheduleResultActions(plan),
    )
    ElMessage.success('已创建排班并开始生成')
  }
  catch (error: any) {
    ElMessage.error(error?.response?.data?.message || '排班创建失败，请重试')
  }
  finally {
    actionPending.value = false
  }
}

async function runQueryScheduleAction(action: ChatAction, message: MessageItem) {
  const range = resolveDateRange(action, message)
  if (!range) {
    ElMessage.warning('未识别到查询日期范围，请补充日期后再试')
    return
  }

  actionPending.value = true
  try {
    const response = await listSchedules({
      page: 1,
      size: 20,
      start_date: range.startDate,
      end_date: range.endDate,
    })
    const plans = sortPlans(normalizeCollection<SchedulePlan>(response)).filter(plan => overlapsRange(plan, range))

    if (plans.length === 0) {
      pushAssistantMessage(
        `没有找到 ${range.startDate} 至 ${range.endDate} 的排班。你可以直接在这里继续创建这个周期的排班。`,
        [{
          type: 'create_schedule',
          label: '创建这个周期的排班',
          payload: { date_range: `${range.startDate} 至 ${range.endDate}` },
        }],
      )
      return
    }

    const previewLines = plans.slice(0, 3).map((plan, index) => `${index + 1}. ${plan.name}（${plan.start_date} ~ ${plan.end_date}，${formatScheduleStatus(plan.status)}）`)
    const actions = plans.slice(0, 3).map(plan => ({
      type: 'open_schedule_detail',
      label: `查看 ${plan.name}`,
      payload: { schedule_id: plan.id },
    }))

    actions.push({
      type: 'open_workspace_range',
      label: '打开工作台',
      payload: { start_date: range.startDate, end_date: range.endDate },
    })

    pushAssistantMessage(`找到 ${plans.length} 份相关排班：\n${previewLines.join('\n')}`, actions)
  }
  catch (error: any) {
    ElMessage.error(error?.response?.data?.message || '查询排班失败，请重试')
  }
  finally {
    actionPending.value = false
  }
}

async function openAdjustmentPanel(action: ChatAction, message: MessageItem) {
  actionPending.value = true
  try {
    await ensureAdjustmentResourcesLoaded()

    const range = resolveDateRange(action, message)
    const entities = extractEntities(action)
    const employeeName = typeof entities.employee_name === 'string' ? entities.employee_name.trim() : ''
    const shiftName = typeof entities.shift_name === 'string' ? entities.shift_name.trim() : ''

    const response = await listSchedules({
      page: 1,
      size: 50,
      start_date: range?.startDate,
      end_date: range?.endDate,
    })
    const rawPlans = sortPlans(normalizeCollection<SchedulePlan>(response))
    const activePlans = rawPlans.filter(plan => plan.status !== 'draft' && plan.status !== 'generating')
    const candidatePlans = (activePlans.length > 0 ? activePlans : rawPlans).filter(plan => overlapsRange(plan, range))

    if (candidatePlans.length === 0) {
      pushAssistantMessage('当前没有可调整的排班。你可以先在这里创建排班，生成后再继续调整。')
      return
    }

    adjustmentScheduleOptions.value = candidatePlans
    adjustmentSeed.value = { range, employeeName, shiftName }
    resetAdjustmentForm()
    adjustmentForm.scheduleId = findBestScheduleId(candidatePlans, range)
    adjustmentForm.targetEmployeeId = matchEmployeeIdByName(employeeName)
    adjustmentForm.targetShiftId = matchShiftIdByName(shiftName)
    adjustmentForm.targetDate = range?.startDate || candidatePlans[0].start_date
    adjustmentPanelVisible.value = true
    await loadAdjustmentAssignments(adjustmentForm.scheduleId)
    ElMessage.success('已载入可调整排班，请在下方确认调整内容')
  }
  catch (error: any) {
    ElMessage.error(error?.response?.data?.message || '准备调整排班失败，请重试')
  }
  finally {
    actionPending.value = false
  }
}

function buildReplacePayload() {
  const assignment = currentAdjustmentAssignment.value
  if (!assignment) {
    throw new Error('请先选择需要调整的排班')
  }

  const nextEmployeeId = adjustmentForm.targetEmployeeId || assignment.employee_id
  const nextShiftId = adjustmentForm.targetShiftId || assignment.shift_id
  const nextDate = adjustmentForm.targetDate || assignment.date

  if (nextEmployeeId === assignment.employee_id && nextShiftId === assignment.shift_id && nextDate === assignment.date) {
    throw new Error('至少修改一项后才能提交调整')
  }

  return {
    modifies: [{
      assignment_id: assignment.id,
      employee_id: nextEmployeeId,
      shift_id: nextShiftId,
      date: nextDate,
    }],
  }
}

function buildAddPayload() {
  if (!adjustmentForm.targetEmployeeId || !adjustmentForm.targetShiftId || !adjustmentForm.targetDate) {
    throw new Error('新增排班需要员工、班次和日期')
  }

  return {
    adds: [{
      employee_id: adjustmentForm.targetEmployeeId,
      shift_id: adjustmentForm.targetShiftId,
      date: adjustmentForm.targetDate,
    }],
  }
}

function buildRemovePayload() {
  if (!adjustmentForm.assignmentId) {
    throw new Error('请选择需要移除的排班')
  }

  return {
    removes: [adjustmentForm.assignmentId],
  }
}

function buildSwapPayload() {
  const firstAssignment = currentAdjustmentAssignment.value
  const secondAssignment = adjustmentAssignments.value.find(item => item.id === adjustmentForm.secondAssignmentId)
  if (!firstAssignment || !secondAssignment) {
    throw new Error('请选择两条需要对调的排班')
  }

  return {
    modifies: [
      {
        assignment_id: firstAssignment.id,
        employee_id: secondAssignment.employee_id,
      },
      {
        assignment_id: secondAssignment.id,
        employee_id: firstAssignment.employee_id,
      },
    ],
  }
}

function buildSubmitPayload() {
  if (adjustmentForm.operation === 'add') {
    return buildAddPayload()
  }
  if (adjustmentForm.operation === 'remove') {
    return buildRemovePayload()
  }
  if (adjustmentForm.operation === 'swap') {
    return buildSwapPayload()
  }
  return buildReplacePayload()
}

function buildAdjustmentSuccessMessage() {
  const schedule = currentAdjustmentSchedule.value
  const assignment = currentAdjustmentAssignment.value

  if (adjustmentForm.operation === 'add') {
    return `已在 ${schedule?.name || '目标排班'} 中新增 ${adjustmentForm.targetDate} ${getEmployeeName(adjustmentForm.targetEmployeeId)} 的 ${getShiftName(adjustmentForm.targetShiftId)}。`
  }
  if (adjustmentForm.operation === 'remove' && assignment) {
    return `已从 ${schedule?.name || '目标排班'} 中移除 ${assignment.date} ${assignment.employee_name} 的 ${assignment.shift_name}。`
  }
  if (adjustmentForm.operation === 'swap') {
    return '已完成所选排班的人员对调。'
  }
  return `已更新 ${schedule?.name || '目标排班'} 中选中的排班安排。`
}

async function submitAdjustment() {
  if (!adjustmentForm.scheduleId) {
    ElMessage.warning('请先选择目标排班')
    return
  }

  adjustmentPanelBusy.value = true
  try {
    const payload = buildSubmitPayload()
    await adjustAssignments(adjustmentForm.scheduleId, payload)
    await loadAdjustmentAssignments(adjustmentForm.scheduleId)

    const schedule = currentAdjustmentSchedule.value
    pushAssistantMessage(
      buildAdjustmentSuccessMessage(),
      schedule ? buildScheduleResultActions(schedule) : [],
    )
    ElMessage.success('排班调整已提交')
    closeAdjustmentPanel()
  }
  catch (error: any) {
    const errorMessage = error instanceof Error
      ? error.message
      : (error?.response?.data?.message || '提交排班调整失败，请重试')
    ElMessage.error(errorMessage)
  }
  finally {
    adjustmentPanelBusy.value = false
  }
}

async function handleActionClick(action: ChatAction, message: MessageItem) {
  if (action.type === 'open_schedule_detail') {
    const scheduleId = isRecord(action.payload) && typeof action.payload.schedule_id === 'string' ? action.payload.schedule_id : ''
    if (!scheduleId) {
      ElMessage.warning('缺少排班编号，无法打开详情')
      return
    }
    await router.push({ name: 'ScheduleDetail', params: { id: scheduleId } })
    return
  }

  if (action.type === 'open_workspace_range') {
    const startDate = isRecord(action.payload) && typeof action.payload.start_date === 'string' ? action.payload.start_date : ''
    const endDate = isRecord(action.payload) && typeof action.payload.end_date === 'string' ? action.payload.end_date : ''
    await router.push({
      name: 'SchedulingWorkspace',
      query: {
        start_date: startDate,
        end_date: endDate,
      },
    })
    return
  }

  if (action.type === 'create_schedule') {
    await runCreateScheduleAction(action, message)
    return
  }

  if (action.type === 'query_schedule') {
    await runQueryScheduleAction(action, message)
    return
  }

  if (action.type === 'adjust_schedule') {
    await openAdjustmentPanel(action, message)
    return
  }

  if (action.type === 'query_rule') {
    await router.push('/rules')
    return
  }

  ElMessage.info('当前动作暂未接入，请继续在对话中补充需求')
}

async function handleDeleteSession(session: AIConversationSummary) {
  try {
    await ElMessageBox.confirm(`确认删除会话 ${formatConversationTitle(session)} 吗？`, '删除会话', {
      type: 'warning',
      confirmButtonText: '删除',
      cancelButtonText: '取消',
    })
  }
  catch {
    return
  }

  try {
    await deleteChatConversation(session.id)
    const remaining = sessions.value.filter(item => item.id !== session.id)
    sessions.value = remaining
    ElMessage.success('会话已删除')

    if (currentSessionId.value !== session.id) {
      return
    }

    if (remaining.length === 0) {
      resetDraftSession()
      return
    }

    await loadConversation(remaining[0].id)
  }
  catch {
    ElMessage.error('删除会话失败')
  }
}

async function maybeRunAutoAction(message: MessageItem, autoActionType?: string) {
  if (!autoActionType) {
    return
  }

  const matchedAction = message.actions?.find(action => action.type === autoActionType)
  if (!matchedAction) {
    return
  }

  await handleActionClick(matchedAction, message)
}

async function sendPrompt(text: string, options?: ExternalPromptOptions) {
  const trimmed = text.trim()
  if (!trimmed || sending.value) {
    return
  }

  messages.value.push({
    id: `local-user-${Date.now()}`,
    role: 'user',
    content: trimmed,
    created_at: new Date().toISOString(),
  })
  sending.value = true
  scrollToBottom()

  try {
    const res = await chat(currentSessionId.value
      ? { message: trimmed, conversation_id: currentSessionId.value }
      : { message: trimmed })
    if (res.conversation_id) {
      currentSessionId.value = res.conversation_id
    }
    const assistantMessage: MessageItem = {
      id: `local-assistant-${Date.now()}`,
      role: 'assistant',
      content: res.reply || '（无回复）',
      created_at: new Date().toISOString(),
      actions: normalizeActions(res.actions),
      sourcePrompt: trimmed,
    }
    messages.value.push(assistantMessage)
    await refreshSessions(currentSessionId.value)
    await maybeRunAutoAction(assistantMessage, options?.autoActionType)
  }
  catch {
    messages.value.push({
      id: `local-error-${Date.now()}`,
      role: 'assistant',
      content: '抱歉，请求出现错误，请稍后再试。',
      created_at: new Date().toISOString(),
    })
  }
  finally {
    sending.value = false
    scrollToBottom()
  }
}

async function sendMessage() {
  const text = inputText.value.trim()
  if (!text) {
    return
  }

  inputText.value = ''
  await sendPrompt(text)
}

async function submitExternalPrompt(prompt: string, options?: ExternalPromptOptions) {
  if (!prompt.trim()) {
    return
  }

  if (options?.newSession) {
    await startNewSession()
  }

  await sendPrompt(prompt, options)
}

function scrollToBottom() {
  nextTick(() => {
    if (messagesContainer.value) {
      messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
    }
  })
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    void sendMessage()
  }
}

function handleClose() {
  emit('close')
}

watch(() => adjustmentForm.scheduleId, async (scheduleId, previousValue) => {
  if (!scheduleId || scheduleId === previousValue) {
    return
  }

  adjustmentForm.assignmentId = ''
  adjustmentForm.secondAssignmentId = ''
  const schedule = adjustmentScheduleOptions.value.find(item => item.id === scheduleId)
  if (schedule && (!adjustmentForm.targetDate || !scheduleDateOptions.value.some(item => item.value === adjustmentForm.targetDate))) {
    adjustmentForm.targetDate = schedule.start_date
  }
  await loadAdjustmentAssignments(scheduleId)
})

watch(() => adjustmentForm.operation, (operation) => {
  adjustmentForm.secondAssignmentId = ''
  if (operation === 'remove' || operation === 'swap') {
    adjustmentForm.targetEmployeeId = ''
    adjustmentForm.targetShiftId = ''
  }
})

watch(() => adjustmentForm.assignmentId, (assignmentId) => {
  if (!assignmentId) {
    return
  }

  const assignment = adjustmentAssignments.value.find(item => item.id === assignmentId)
  if (!assignment) {
    return
  }

  if (!adjustmentForm.targetDate) {
    adjustmentForm.targetDate = assignment.date
  }
  if (adjustmentForm.operation === 'replace' && !adjustmentForm.targetShiftId) {
    adjustmentForm.targetShiftId = assignment.shift_id
  }
})

defineExpose({ submitExternalPrompt })

onMounted(async () => {
  await refreshSessions()
  if (currentSessionId.value) {
    await loadConversation(currentSessionId.value)
    return
  }
  resetDraftSession()
})
</script>

<template>
  <div class="conversation-panel" :class="{ embedded }">
    <div v-if="embedded" class="panel-header">
      <span class="panel-title">智能排班助手</span>
      <el-button v-if="showClose" :icon="Close" text @click="handleClose" />
    </div>

    <div class="chat-container">
      <aside class="session-sidebar">
        <div class="sidebar-header">
          <span>会话列表</span>
          <el-button text size="small" @click="startNewSession">
            新会话
          </el-button>
        </div>
        <div class="session-list">
          <div v-if="loadingSessions" class="empty-tip">
            正在加载会话...
          </div>
          <div
            v-for="session in sessions"
            :key="session.id"
            class="session-item"
            :class="{ active: session.id === currentSessionId }"
            @click="selectSession(session)"
          >
            <el-icon class="session-icon">
              <ChatLineSquare />
            </el-icon>
            <div class="session-body">
              <span class="session-title">{{ formatConversationTitle(session) }}</span>
              <span class="session-meta">{{ formatConversationMeta(session) }}</span>
            </div>
            <el-button
              class="session-delete"
              text
              circle
              @click.stop="handleDeleteSession(session)"
            >
              <el-icon><Delete /></el-icon>
            </el-button>
          </div>
          <div v-if="!loadingSessions && sessions.length === 0" class="empty-tip">
            暂无历史会话
          </div>
        </div>
      </aside>

      <main class="chat-main">
        <div ref="messagesContainer" class="messages">
          <div v-if="loadingConversation" class="empty-tip">
            正在加载会话内容...
          </div>
          <div
            v-for="(msg, idx) in messages"
            :key="msg.id || idx"
            class="message-row"
            :class="msg.role"
          >
            <div class="avatar">
              {{ msg.role === 'user' ? '我' : 'AI' }}
            </div>
            <div class="bubble">
              <div>{{ msg.content }}</div>
              <div v-if="msg.actions?.length" class="bubble-actions">
                <el-button
                  v-for="action in msg.actions"
                  :key="action.id"
                  size="small"
                  plain
                  type="primary"
                  :loading="actionPending"
                  @click="handleActionClick(action, msg)"
                >
                  {{ action.label }}
                </el-button>
              </div>
            </div>
          </div>

          <div v-if="sending" class="message-row assistant">
            <div class="avatar">
              AI
            </div>
            <div class="bubble typing">
              正在思考中...
            </div>
          </div>
        </div>

        <section v-if="adjustmentPanelVisible" class="adjustment-panel">
          <div class="adjustment-header">
            <div>
              <div class="adjustment-eyebrow">调整排班</div>
              <h3 class="adjustment-title">在 AI 助手内完成调整</h3>
              <p class="adjustment-desc">选择目标排班和调整动作后，直接提交到排班调整接口。</p>
            </div>
            <el-button :icon="Close" text @click="closeAdjustmentPanel" />
          </div>

          <div class="adjustment-grid">
            <div class="field-block field-block--wide">
              <label>目标排班</label>
              <el-select v-model="adjustmentForm.scheduleId" filterable placeholder="请选择要调整的排班" :loading="adjustmentPanelBusy">
                <el-option
                  v-for="plan in adjustmentScheduleOptions"
                  :key="plan.id"
                  :label="formatScheduleOption(plan)"
                  :value="plan.id"
                />
              </el-select>
            </div>

            <div class="field-block field-block--wide">
              <label>调整动作</label>
              <el-radio-group v-model="adjustmentForm.operation" class="adjustment-operation-group">
                <el-radio-button
                  v-for="option in adjustmentOperationOptions"
                  :key="option.value"
                  :label="option.value"
                >
                  {{ option.label }}
                </el-radio-button>
              </el-radio-group>
            </div>

            <template v-if="adjustmentForm.operation !== 'add'">
              <div class="field-block field-block--wide">
                <label>{{ adjustmentForm.operation === 'remove' ? '待移除排班' : adjustmentForm.operation === 'swap' ? '第一条排班' : '待调整排班' }}</label>
                <el-select v-model="adjustmentForm.assignmentId" filterable placeholder="请选择现有排班" :loading="adjustmentPanelBusy">
                  <el-option
                    v-for="assignment in adjustmentAssignments"
                    :key="assignment.id"
                    :label="formatAssignmentOption(assignment)"
                    :value="assignment.id"
                  />
                </el-select>
              </div>
            </template>

            <template v-if="adjustmentForm.operation === 'swap'">
              <div class="field-block field-block--wide">
                <label>第二条排班</label>
                <el-select v-model="adjustmentForm.secondAssignmentId" filterable placeholder="请选择需要对调的排班" :loading="adjustmentPanelBusy">
                  <el-option
                    v-for="assignment in secondaryAssignmentOptions"
                    :key="assignment.id"
                    :label="formatAssignmentOption(assignment)"
                    :value="assignment.id"
                  />
                </el-select>
              </div>
            </template>

            <template v-if="adjustmentForm.operation === 'add' || adjustmentForm.operation === 'replace'">
              <div class="field-block">
                <label>{{ adjustmentForm.operation === 'add' ? '员工' : '目标员工' }}</label>
                <el-select v-model="adjustmentForm.targetEmployeeId" filterable clearable placeholder="请选择员工">
                  <el-option
                    v-for="employee in employeeOptions"
                    :key="employee.id"
                    :label="employee.name"
                    :value="employee.id"
                  />
                </el-select>
              </div>

              <div class="field-block">
                <label>{{ adjustmentForm.operation === 'add' ? '班次' : '目标班次' }}</label>
                <el-select v-model="adjustmentForm.targetShiftId" filterable clearable placeholder="请选择班次">
                  <el-option
                    v-for="shift in shiftOptions"
                    :key="shift.id"
                    :label="shift.name"
                    :value="shift.id"
                  />
                </el-select>
              </div>

              <div class="field-block">
                <label>{{ adjustmentForm.operation === 'add' ? '排班日期' : '目标日期' }}</label>
                <el-select v-model="adjustmentForm.targetDate" filterable clearable placeholder="请选择日期">
                  <el-option
                    v-for="dateOption in scheduleDateOptions"
                    :key="dateOption.value"
                    :label="dateOption.label"
                    :value="dateOption.value"
                  />
                </el-select>
              </div>
            </template>
          </div>

          <div class="adjustment-summary">
            {{ adjustmentSummary }}
          </div>

          <div class="adjustment-actions">
            <el-button @click="closeAdjustmentPanel">
              取消
            </el-button>
            <el-button type="primary" :loading="adjustmentPanelBusy" @click="submitAdjustment">
              确认调整
            </el-button>
          </div>
        </section>

        <div class="input-area">
          <el-input
            v-model="inputText"
            type="textarea"
            :rows="2"
            placeholder="输入消息，按 Enter 发送..."
            resize="none"
            @keydown="handleKeydown"
          />
          <el-button type="primary" :icon="Promotion" :loading="sending" circle @click="sendMessage" />
        </div>
      </main>
    </div>
  </div>
</template>

<style scoped>
.conversation-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: #fff;
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid #e5e7eb;
  background: #fff;
}

.panel-title {
  font-size: 15px;
  font-weight: 600;
  color: #111827;
}

.chat-container {
  height: 100%;
  display: flex;
  overflow: hidden;
  flex: 1;
  min-height: 0;
}

.session-sidebar {
  width: 240px;
  border-right: 1px solid #e5e7eb;
  display: flex;
  flex-direction: column;
  background: #fafafa;
}

.sidebar-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  font-weight: 600;
  border-bottom: 1px solid #e5e7eb;
}

.session-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.session-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border-radius: 8px;
  cursor: pointer;
  font-size: 14px;
  color: #374151;
  transition: background 0.15s;
}

.session-item:hover {
  background: #e5e7eb;
}

.session-item.active {
  background: #dbeafe;
  color: #1d4ed8;
}

.session-icon {
  flex-shrink: 0;
}

.session-body {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  gap: 2px;
}

.session-title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-weight: 600;
}

.session-meta {
  color: #94a3b8;
  font-size: 12px;
}

.session-delete {
  opacity: 0;
  transition: opacity 0.15s;
}

.session-item:hover .session-delete,
.session-item.active .session-delete {
  opacity: 1;
}

.empty-tip {
  text-align: center;
  padding: 32px;
  color: #9ca3af;
  font-size: 13px;
}

.chat-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.messages {
  flex: 1;
  overflow-y: auto;
  padding: 24px;
}

.message-row {
  display: flex;
  gap: 12px;
  margin-bottom: 20px;
}

.message-row.user {
  flex-direction: row-reverse;
}

.avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: #e5e7eb;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
  flex-shrink: 0;
  color: #374151;
}

.message-row.user .avatar {
  background: #3b82f6;
  color: #fff;
}

.message-row.assistant .avatar,
.message-row.system .avatar {
  background: #10b981;
  color: #fff;
}

.bubble {
  max-width: 70%;
  padding: 12px 16px;
  border-radius: 12px;
  background: #f3f4f6;
  display: flex;
  flex-direction: column;
  gap: 12px;
  font-size: 14px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}

.bubble-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.message-row.user .bubble {
  background: #3b82f6;
  color: #fff;
  border-bottom-right-radius: 4px;
}

.message-row.assistant .bubble,
.message-row.system .bubble {
  border-bottom-left-radius: 4px;
}

.bubble.typing {
  color: #9ca3af;
  font-style: italic;
}

.adjustment-panel {
  border-top: 1px solid #e5e7eb;
  background: linear-gradient(180deg, #f8fbff 0%, #ffffff 100%);
  padding: 18px 24px;
}

.adjustment-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.adjustment-eyebrow {
  color: #0f766e;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.adjustment-title {
  margin: 4px 0 0;
  color: #111827;
  font-size: 18px;
}

.adjustment-desc {
  margin: 6px 0 0;
  color: #6b7280;
  font-size: 13px;
  line-height: 1.6;
}

.adjustment-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
  margin-top: 16px;
}

.field-block {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.field-block label {
  color: #374151;
  font-size: 13px;
  font-weight: 600;
}

.field-block--wide {
  grid-column: span 3;
}

.adjustment-operation-group {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.adjustment-summary {
  margin-top: 14px;
  padding: 12px 14px;
  border-radius: 10px;
  background: rgba(219, 234, 254, 0.55);
  color: #1e3a8a;
  font-size: 13px;
  line-height: 1.7;
}

.adjustment-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 14px;
}

.input-area {
  display: flex;
  gap: 12px;
  align-items: flex-end;
  padding: 16px 24px;
  border-top: 1px solid #e5e7eb;
  background: #fff;
}

.input-area .el-input {
  flex: 1;
}

@media (max-width: 1080px) {
  .adjustment-grid {
    grid-template-columns: 1fr;
  }

  .field-block--wide {
    grid-column: span 1;
  }
}

@media (max-width: 768px) {
  .chat-container {
    flex-direction: column;
  }

  .session-sidebar {
    width: 100%;
    max-height: 220px;
    border-right: 0;
    border-bottom: 1px solid #e5e7eb;
  }

  .messages {
    padding: 16px;
  }

  .adjustment-panel {
    padding: 14px 16px;
  }

  .adjustment-header,
  .adjustment-actions {
    flex-direction: column;
    align-items: stretch;
  }

  .bubble {
    max-width: 100%;
  }

  .input-area {
    padding: 12px 16px;
  }
}
</style>