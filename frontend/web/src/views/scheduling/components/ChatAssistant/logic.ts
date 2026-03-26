import type { Ref } from 'vue'
import type { DataViewDialogExpose } from '../DataViewDialog/type'
import type { ActionHandlers, ButtonType, ChatAssistantMessage, StyleMapping } from './type'
import type { WorkflowAction } from '@/types/chat'
import { ElMessage, ElMessageBox } from 'element-plus'
import { nextTick, ref } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useSchedulingSessionStore } from '@/stores/schedulingSession'
import { getNextWeekRange } from '@/utils/date'

/**
 * ChatAssistant 业务逻辑 Hook
 */
export function useChatAssistant(
  dataViewDialogRef?: Ref<DataViewDialogExpose | null>,
  callbacks?: {
    showStaffDetails?: (data: any) => void
    showRulesDetails?: (data: any) => void
    showShiftSchedule?: (data: any) => void
    showPersonalNeeds?: (data: any) => void
    showTemporaryRules?: (data: any) => void
    showMultiShiftSchedule?: (data: any) => void
    showSchedulePreview?: (data: any) => void
    showChangeDetail?: (data: any) => void
    showValidationResult?: (data: any) => void
  },
) {
  const sessionStore = useSchedulingSessionStore()
  const authStore = useAuthStore()

  // 状态
  const messages = ref<ChatAssistantMessage[]>([])
  const inputMessage = ref('')
  const loading = ref(false)
  const messagesContainer = ref<HTMLElement>()
  const isInitializing = ref(false)

  // 表单对话框状态
  const actionDialogVisible = ref(false)
  const currentAction = ref<WorkflowAction | null>(null)

  // ==================== Session 管理 ====================

  function getCurrentOrgId(): string {
    return authStore.currentNodeId || localStorage.getItem('current_org_id') || ''
  }

  function getCurrentUserId(): string {
    return authStore.user?.id || localStorage.getItem('current_user_id') || ''
  }

  async function getWsClient() {
    return await sessionStore.getOrCreateWsClient()
  }

  /** 初始化会话 */
  async function initSession() {
    isInitializing.value = true
    try {
      const wsClient = await getWsClient()
      const orgId = getCurrentOrgId()
      const userId = getCurrentUserId()

      wsClient.getRecentSession(orgId, userId)
      await new Promise(resolve => setTimeout(resolve, 500))

      if (sessionStore.currentSessionId) {
        syncMessagesFromStore(sessionStore.messages)
        if (sessionStore.workflow?.actions?.length) {
          updateLastMessageActions(sessionStore.workflow.actions)
        }
      }
      else {
        wsClient.createSession(orgId, userId, 'rostering')
        await new Promise(resolve => setTimeout(resolve, 500))
      }
    }
    catch (error) {
      console.error('初始化会话失败:', error)
      ElMessage.error('加载会话失败')
      try {
        const wsClient = await getWsClient()
        wsClient.createSession(getCurrentOrgId(), getCurrentUserId(), 'rostering')
        await new Promise(resolve => setTimeout(resolve, 500))
      }
      catch {
        // ignore
      }
    }
    finally {
      isInitializing.value = false
    }
  }

  /** 创建新会话 */
  async function createNewSession() {
    try {
      loading.value = true
      sessionStore.clearSession()
      messages.value = []

      const wsClient = await getWsClient()
      wsClient.createSession(getCurrentOrgId(), getCurrentUserId(), 'rostering')
      await new Promise(resolve => setTimeout(resolve, 500))

      ElMessage.success('已创建新会话')
    }
    catch (error) {
      console.error('创建新会话失败:', error)
      ElMessage.error('创建会话失败，请重试')
    }
    finally {
      loading.value = false
    }
  }

  /** 重连会话 */
  async function handleReconnectSession() {
    try {
      loading.value = true
      const wsClient = await getWsClient()
      wsClient.getRecentSession(getCurrentOrgId(), getCurrentUserId())
      await new Promise(resolve => setTimeout(resolve, 500))

      if (sessionStore.currentSessionId) {
        ElMessage.success('已重新连接会话')
        syncMessagesFromStore(sessionStore.messages)
      }
      else {
        ElMessage.warning('未找到最近会话，请创建新会话')
      }
    }
    catch (error) {
      console.error('重连会话失败:', error)
      ElMessage.error('重连失败，请重试')
    }
    finally {
      loading.value = false
    }
  }

  /** 创建新会话并断开旧连接 */
  async function handleCreateNewSessionAndDisconnect() {
    try {
      loading.value = true
      sessionStore.clearSession()
      messages.value = []

      const wsClient = await getWsClient()
      wsClient.createSession(getCurrentOrgId(), getCurrentUserId(), 'rostering')
      await new Promise(resolve => setTimeout(resolve, 500))

      if (sessionStore.currentSessionId) {
        ElMessage.success('已创建新会话')
      }
    }
    catch (error) {
      console.error('创建新会话失败:', error)
      ElMessage.error('创建会话失败，请重试')
    }
    finally {
      loading.value = false
    }
  }

  function disconnectWebSocket() {
    // 保持全局连接活跃，不主动断开
  }

  // ==================== 消息处理 ====================

  function updateLastMessageActions(actions: WorkflowAction[]) {
    for (let i = messages.value.length - 1; i >= 0; i--) {
      if (messages.value[i].role === 'assistant') {
        messages.value[i].workflowActions = actions
        break
      }
    }
  }

  /** 发送用户消息 */
  async function sendMessage() {
    if (!inputMessage.value.trim())
      return

    const msgContent = inputMessage.value.trim()
    const userMessage: ChatAssistantMessage = {
      id: `user-${Date.now()}`,
      role: 'user',
      content: msgContent,
      createdAt: new Date().toISOString(),
      isSending: true,
    }
    messages.value.push(userMessage)
    inputMessage.value = ''
    loading.value = true

    try {
      if (!sessionStore.currentSessionId)
        throw new Error('会话未初始化')

      const wsClient = await getWsClient()
      const workflow = sessionStore.workflow
      const isWaitingAdjustment = workflow?.phase?.includes('waiting_adjustment')

      if (isWaitingAdjustment) {
        wsClient.sendWorkflowCommand(sessionStore.currentSessionId, '_modify_', { message: msgContent })
      }
      else {
        wsClient.sendUserMessage(sessionStore.currentSessionId, msgContent)
      }
      userMessage.isSending = false
    }
    catch (error) {
      console.error('发送消息失败:', error)
      ElMessage.error('发送消息失败，请重试')
      userMessage.isSending = false
    }
    finally {
      loading.value = false
    }
  }

  // ==================== 操作处理 ====================

  /** 处理 workflow 类型操作 */
  async function handleWorkflowAction(action: WorkflowAction) {
    try {
      if (!sessionStore.currentSessionId) {
        await createNewSession()
      }
      if (!sessionStore.currentSessionId) {
        ElMessage.error('会话初始化失败，请刷新页面重试')
        return
      }

      // 有字段定义时弹出表单对话框
      if (action.fields?.length) {
        currentAction.value = action
        actionDialogVisible.value = true
        return
      }

      // 特殊处理 _start_ 事件
      let payload = action.payload
      if (action.event === '_start_' && !payload) {
        const defaultRange = getNextWeekRange()
        payload = {
          startDate: defaultRange.start,
          endDate: defaultRange.end,
          orgId: getCurrentOrgId(),
        }
      }

      await sendWorkflowCommand(action.event, payload)
    }
    catch (error) {
      console.error('工作流操作失败:', error)
      ElMessage.error('操作失败，请重试')
    }
  }

  /** 发送工作流命令 */
  async function sendWorkflowCommand(event: string, payload?: Record<string, any>) {
    if (!sessionStore.currentSessionId) {
      ElMessage.error('会话未初始化')
      return
    }
    const wsClient = await getWsClient()
    wsClient.sendWorkflowCommand(sessionStore.currentSessionId, event, payload)
  }

  /** 处理 query 类型操作 */
  function handleQueryAction(action: WorkflowAction) {
    // 人员详情
    if ((action.id === 'preview_shift_groups' || action.id === 'view_staff_details') && callbacks?.showStaffDetails) {
      callbacks.showStaffDetails(action.payload)
      return
    }
    // 规则详情
    if (action.id === 'view_rules_details' && callbacks?.showRulesDetails) {
      callbacks.showRulesDetails(action.payload)
      return
    }
    // 临时规则
    if (action.id === 'view_temporary_rules' && callbacks?.showTemporaryRules) {
      callbacks.showTemporaryRules(action.payload)
      return
    }
    // 变更详情
    if (action.id === 'view_changes_detail' && callbacks?.showChangeDetail) {
      callbacks.showChangeDetail(action.payload)
      return
    }
    // 排班详情（多班次）
    if ((action.id === 'view_schedule_detail' || action.id === 'view_task_schedule_detail' || action.id === 'preview_full_schedule') && callbacks?.showMultiShiftSchedule) {
      const payload = action.payload as any
      if (payload?.shifts) {
        callbacks.showMultiShiftSchedule(payload)
        return
      }
      // 旧格式兼容
      if (payload?.draftSchedule) {
        const draftSchedule = payload.draftSchedule
        const shifts: Record<string, any> = {}
        if (draftSchedule.Shifts) {
          Object.keys(draftSchedule.Shifts).forEach((shiftId) => {
            const shiftData = draftSchedule.Shifts[shiftId]
            shifts[shiftId] = {
              shiftId,
              shiftName: shiftData.ShiftName || shiftId,
              days: shiftData.Days || {},
            }
          })
        }
        callbacks.showMultiShiftSchedule({
          shifts,
          startDate: payload.startDate,
          endDate: payload.endDate,
          shiftInfoList: Object.keys(shifts).map(shiftId => ({
            id: shiftId,
            name: shifts[shiftId].shiftName,
          })),
        })
        return
      }
    }
    // 单班次排班
    if (action.id.startsWith('view_shift_schedule_') && callbacks?.showShiftSchedule) {
      callbacks.showShiftSchedule(action.payload)
      return
    }
    // 固定班次
    if (action.id === 'view_fixed_shifts_detail') {
      const fixedData = action.payload as any
      if (fixedData?.shifts) {
        const shiftIds = Object.keys(fixedData.shifts)
        if (shiftIds.length > 1 && callbacks?.showMultiShiftSchedule) {
          callbacks.showMultiShiftSchedule(fixedData)
          return
        }
        if (shiftIds.length === 1 && callbacks?.showShiftSchedule) {
          const shiftId = shiftIds[0]
          const shiftSchedule = fixedData.shifts[shiftId]
          callbacks.showShiftSchedule({
            shiftId: shiftSchedule.shiftId || shiftId,
            shiftName: `固定班次-${shiftId}`,
            startDate: fixedData.startDate,
            endDate: fixedData.endDate,
            schedule: shiftSchedule,
          })
          return
        }
      }
    }
    // 个人需求
    if (action.id === 'view_personal_needs' && callbacks?.showPersonalNeeds) {
      callbacks.showPersonalNeeds(action.payload)
      return
    }
    // 校验结果
    if (action.id === 'view_validation_result' && callbacks?.showValidationResult) {
      callbacks.showValidationResult(action.payload)
      return
    }
    // 排班预览
    if (action.id === 'preview_full_schedule' && callbacks?.showSchedulePreview) {
      callbacks.showSchedulePreview(action.payload)
      return
    }

    // 通用：使用 DataViewDialog
    if (dataViewDialogRef?.value) {
      dataViewDialogRef.value.open({
        title: action.label,
        data: action.payload || {},
        mode: 'auto',
        showCopy: true,
        showExport: false,
      })
    }
    else {
      ElMessageBox.alert(
        `<pre style="max-height: 400px; overflow: auto;">${JSON.stringify(action.payload || {}, null, 2)}</pre>`,
        action.label,
        { dangerouslyUseHTMLString: true, confirmButtonText: '关闭' },
      )
    }
  }

  /** 处理 navigate 类型操作 */
  function handleNavigateAction(action: WorkflowAction) {
    const targetUrl = action.payload?.url
    if (targetUrl) {
      window.location.href = targetUrl
    }
    else {
      ElMessage.warning('导航地址未配置')
    }
  }

  const actionHandlers: ActionHandlers = {
    workflow: handleWorkflowAction,
    query: handleQueryAction,
    command: handleWorkflowAction,
    navigate: handleNavigateAction,
  }

  /** 处理操作按钮点击 */
  async function handleActionClick(action: WorkflowAction, message: ChatAssistantMessage) {
    if (action.id === 'reconnect' && action.event === 'reconnect_session') {
      await handleReconnectSession()
      return
    }
    if (action.id === 'create_new_and_disconnect' && action.event === 'create_new_session_and_disconnect') {
      await handleCreateNewSessionAndDisconnect()
      return
    }

    const handler = actionHandlers[action.type]
    if (handler) {
      await handler(action, message)
    }
    else {
      console.warn('未知的操作类型:', action.type)
    }
  }

  // ==================== 辅助方法 ====================

  function scrollToBottom() {
    nextTick(() => {
      if (messagesContainer.value) {
        messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
      }
    })
  }

  function getButtonType(style?: string): ButtonType {
    const mapping: StyleMapping = {
      primary: 'primary',
      secondary: 'default',
      success: 'success',
      danger: 'danger',
      warning: 'warning',
      info: 'info',
      link: 'primary',
    }
    return mapping[style || 'primary'] || 'default'
  }

  function syncMessagesFromStore(storeMessages: any[]) {
    messages.value = storeMessages
      .filter((msg) => {
        const content = msg.content
        return content !== null && content !== undefined && content !== ''
      })
      .map(msg => ({
        id: msg.id || `${Date.now()}-${Math.random()}`,
        role: msg.role || 'assistant',
        content: msg.content,
        createdAt: msg.time || msg.timestamp || new Date().toISOString(),
        actions: msg.actions || [],
        workflowActions: msg.workflowActions || [],
      }))

    if (sessionStore.workflow?.actions?.length) {
      updateLastMessageActions(sessionStore.workflow.actions)
    }

    scrollToBottom()
  }

  function handleDialogConfirm(formData: Record<string, any>) {
    if (currentAction.value) {
      sendWorkflowCommand(currentAction.value.event, formData)
    }
  }

  function handleDialogCancel() {
    // no-op
  }

  /** 一键启动排班创建工作流 */
  async function startScheduleCreationWorkflow(options?: {
    startDate?: string
    endDate?: string
    orgId?: string
  }) {
    try {
      if (!sessionStore.currentSessionId) {
        await initSession()
        if (!sessionStore.currentSessionId) {
          await createNewSession()
        }
      }
      if (!sessionStore.currentSessionId) {
        ElMessage.error('无法启动工作流，请刷新页面重试')
        return
      }

      const wsClient = await getWsClient()
      const defaultRange = getNextWeekRange()
      const payload = {
        startDate: options?.startDate || defaultRange.start,
        endDate: options?.endDate || defaultRange.end,
        orgId: options?.orgId || getCurrentOrgId(),
      }

      wsClient.sendWorkflowCommand(sessionStore.currentSessionId, '_start_', payload)
      ElMessage.success('正在启动排班创建流程...')
    }
    catch (error) {
      console.error('启动排班创建失败:', error)
      ElMessage.error('启动失败，请重试')
    }
  }

  /** 加载历史对话 */
  async function loadHistoryConversation(conversationId: string) {
    try {
      loading.value = true
      if (!sessionStore.currentSessionId) {
        const wsClient = await getWsClient()
        wsClient.createSession(getCurrentOrgId(), getCurrentUserId(), 'rostering')
        await new Promise(resolve => setTimeout(resolve, 500))
      }
      if (!sessionStore.currentSessionId)
        throw new Error('无法创建或获取当前会话')

      const wsClient = await getWsClient()
      wsClient.loadConversation(sessionStore.currentSessionId, conversationId)
      await new Promise(resolve => setTimeout(resolve, 1500))

      syncMessagesFromStore(sessionStore.messages)
      if (sessionStore.workflow?.actions?.length) {
        updateLastMessageActions(sessionStore.workflow.actions)
      }
      await nextTick()
      scrollToBottom()

      ElMessage.success('历史对话加载成功')
    }
    catch (error) {
      console.error('加载历史对话失败:', error)
      ElMessage.error('加载历史对话失败，请重试')
      throw error
    }
    finally {
      loading.value = false
    }
  }

  return {
    // 状态
    messages,
    inputMessage,
    loading,
    isInitializing,
    messagesContainer,
    sessionStore,
    actionDialogVisible,
    currentAction,

    // Session 管理
    initSession,
    createNewSession,
    disconnectWebSocket,

    // 消息方法
    sendMessage,
    handleActionClick,
    getButtonType,
    scrollToBottom,
    updateLastMessageActions,
    syncMessagesFromStore,

    // 工作流
    startScheduleCreationWorkflow,

    // 对话框
    handleDialogConfirm,
    handleDialogCancel,

    // 历史对话
    loadHistoryConversation,
  }
}
