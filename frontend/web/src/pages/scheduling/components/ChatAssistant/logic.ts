import type { Ref } from 'vue'
import type { DataViewDialogExpose } from '../DataViewDialog/type'
import type { ActionHandlers, ButtonType, ChatAssistantMessage, StyleMapping } from './type'
import type { SchedulingWsClient } from '@/services/scheduling-ws'
import type { WorkflowAction } from '@/types/chat'
import { ElMessage, ElMessageBox } from 'element-plus'
import { nextTick, ref } from 'vue'
import { useSchedulingSessionStore } from '@/store/schedulingSession'
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

  /**
   * 获取 WebSocket 客户端（从全局 store）
   */
  async function getWsClient(): Promise<SchedulingWsClient> {
    return await sessionStore.getOrCreateWsClient()
  }

  /**
   * 初始化会话
   * 在组件挂载时调用，自动加载最近会话或创建新会话
   */
  async function initSession() {
    isInitializing.value = true
    try {
      // 1. 获取或创建全局 WebSocket 客户端
      const wsClient = await getWsClient()

      // 2. 获取组织和用户信息
      const orgId = getCurrentOrgId()
      const userId = getCurrentUserId()

      // 3. 通过 WebSocket 请求最近会话（这会自动绑定客户端到会话）
      wsClient.getRecentSession(orgId, userId)
      // 响应将通过 Store 的 WebSocket 消息处理器自动更新
      // 如果有会话，Store 会自动更新 messages
      // 这里等待一段时间让响应返回
      await new Promise(resolve => setTimeout(resolve, 500))

      if (sessionStore.currentSessionId) {
        console.log('[ChatAssistant] Loaded recent session:', sessionStore.currentSessionId)
        syncMessagesFromStore(sessionStore.messages)
        // 确保 workflow actions 也被同步（如果存在）
        if (sessionStore.workflow && sessionStore.workflow.actions && sessionStore.workflow.actions.length > 0) {
          updateLastMessageActions(sessionStore.workflow.actions)
        }
      }
      else {
        // 没有历史会话，创建新会话
        console.log('[ChatAssistant] No recent session found, creating new session...')
        wsClient.createSession(orgId, userId, 'rostering')
        // 等待会话创建完成
        await new Promise(resolve => setTimeout(resolve, 500))

        if (sessionStore.currentSessionId) {
          console.log('[ChatAssistant] New session created:', sessionStore.currentSessionId)
        }
        else {
          console.warn('[ChatAssistant] Failed to create session, sessionId is still null')
        }
        // 欢迎消息由后端发送，不需要前端显示
      }
    }
    catch (error) {
      console.error('初始化会话失败:', error)
      ElMessage.error('加载会话失败')
      // 降级：尝试创建新会话
      try {
        const orgId = getCurrentOrgId()
        const userId = getCurrentUserId()
        const wsClient = await getWsClient()
        wsClient.createSession(orgId, userId, 'rostering')
        await new Promise(resolve => setTimeout(resolve, 500))
      }
      catch (e) {
        console.error('创建会话也失败:', e)
      }
      // 错误消息由后端发送，不需要前端显示欢迎消息
    }
    finally {
      isInitializing.value = false
    }
  }

  /**
   * 创建新会话
   * 用户点击"新建会话"按钮时调用
   */
  async function createNewSession() {
    try {
      loading.value = true
      console.log('[ChatAssistant] Creating new session...')

      // 清空当前会话
      sessionStore.clearSession()
      messages.value = []

      // 获取全局 WebSocket 客户端（不断开连接）
      const wsClient = await getWsClient()

      // 通过 WebSocket 创建新会话
      const orgId = getCurrentOrgId()
      const userId = getCurrentUserId()
      console.log('[ChatAssistant] Creating session with:', { orgId, userId })

      wsClient.createSession(orgId, userId, 'rostering')
      // 响应将通过 Store 的 WebSocket 消息处理器自动更新 currentSessionId
      await new Promise(resolve => setTimeout(resolve, 500))

      console.log('[ChatAssistant] Session created, ID:', sessionStore.currentSessionId)
      ElMessage.success('已创建新会话')
      // 欢迎消息由后端发送，不需要前端显示
    }
    catch (error) {
      console.error('创建新会话失败:', error)
      ElMessage.error('创建会话失败，请重试')
    }
    finally {
      loading.value = false
    }
  }

  // 移除 showWelcomeMessage 函数，欢迎消息由后端发送

  /**
   * 处理重连会话
   */
  async function handleReconnectSession() {
    try {
      loading.value = true
      console.log('[ChatAssistant] Reconnecting session...')

      // 获取全局 WebSocket 客户端（会自动重连）
      const wsClient = await getWsClient()

      // 获取最近会话
      const orgId = getCurrentOrgId()
      const userId = getCurrentUserId()
      wsClient.getRecentSession(orgId, userId)
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

  /**
   * 处理创建新会话并断开连接
   */
  async function handleCreateNewSessionAndDisconnect() {
    try {
      loading.value = true
      console.log('[ChatAssistant] Creating new session and disconnecting...')

      // 清空当前会话
      sessionStore.clearSession()
      messages.value = []

      // 获取全局 WebSocket 客户端
      const wsClient = await getWsClient()

      // 创建新会话
      const orgId = getCurrentOrgId()
      const userId = getCurrentUserId()
      wsClient.createSession(orgId, userId, 'rostering')
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

  /**
   * 获取当前组织 ID
   * TODO: 从实际的用户信息或路由中获取
   */
  function getCurrentOrgId(): string {
    // 从 localStorage 或环境变量默认值
    return localStorage.getItem('current_org_id') || import.meta.env.VITE_DEFAULT_ORG_ID
  }

  /**
   * 获取当前用户 ID
   * TODO: 从实际的用户信息中获取
   */
  function getCurrentUserId(): string {
    // 从 localStorage 或环境变量默认值
    return localStorage.getItem('current_user_id') || import.meta.env.VITE_DEFAULT_USER_ID
  }

  /**
   * 断开 WebSocket 连接（仅在应用关闭时调用，组件卸载时不断开）
   * 注意：这个方法现在基本不再使用，因为连接是全局的
   */
  function disconnectWebSocket() {
    // 不再主动断开连接，让全局连接保持活跃
    console.log('[ChatAssistant] disconnectWebSocket called, but connection is kept alive globally')
  }

  // ==================== 原有功能 ====================

  /**
   * 更新最后一条助手消息的操作按钮
   */
  function updateLastMessageActions(actions: WorkflowAction[]) {
    for (let i = messages.value.length - 1; i >= 0; i--) {
      if (messages.value[i].role === 'assistant') {
        messages.value[i].workflowActions = actions
        break
      }
    }
  }

  /**
   * 发送用户消息
   */
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
      if (!sessionStore.currentSessionId) {
        throw new Error('会话未初始化')
      }

      // 获取全局 WebSocket 客户端
      const wsClient = await getWsClient()

      // 检查当前工作流状态，如果处于等待调整状态，发送 workflow_command
      const workflow = sessionStore.workflow
      const isV2WaitingAdjustment = workflow?.phase === '_schedule_v2_create_waiting_adjustment_'
      const isV3WaitingAdjustment = workflow?.phase === '_schedule_v3_create_waiting_adjustment_'

      if (isV2WaitingAdjustment || isV3WaitingAdjustment) {
        // 在等待调整状态时，发送 workflow_command 事件
        console.log('Sending adjustment message as workflow command')
        wsClient.sendWorkflowCommand(
          sessionStore.currentSessionId,
          isV3WaitingAdjustment ? '_schedule_v3_create_user_adjustment_message_' : '_schedule_v2_create_user_adjustment_message_',
          { message: msgContent },
        )
        // 消息发送成功，标记为已发送
        userMessage.isSending = false
      }
      else {
        // 普通状态，发送普通用户消息
        wsClient.sendUserMessage(sessionStore.currentSessionId, msgContent)
        // 消息发送成功，标记为已发送
        userMessage.isSending = false
      }
    }
    catch (error) {
      console.error('发送消息失败:', error)
      ElMessage.error('发送消息失败，请重试')
      userMessage.isSending = false
    }
    finally {
      loading.value = false
      if (userMessage.isSending) {
        userMessage.isSending = false
      }
    }
  }

  /**
   * 模拟助手响应（开发阶段使用）
   * 保留作为示例代码
   */
  async function _mockAssistantResponse() {
    return new Promise<void>((resolve) => {
      setTimeout(() => {
        const assistantMsg: ChatAssistantMessage = {
          id: `assistant-${Date.now()}`,
          role: 'assistant',
          content: '请选择排班周期:',
          createdAt: new Date().toISOString(),
          actions: [
            {
              id: 'confirm-period',
              type: 'command',
              label: '确认周期',
              event: '_schedule_create_period_confirmed_',
              style: 'primary',
            },
            {
              id: 'modify-period',
              type: 'command',
              label: '修改周期',
              event: '_schedule_create_period_modified_',
              style: 'secondary',
            },
            {
              id: 'view-period',
              type: 'query',
              label: '查看周期详情',
              event: 'view_period_details',
              style: 'info',
              payload: {
                start: '2025-01-01',
                end: '2025-01-07',
              },
            },
          ],
        }
        messages.value.push(assistantMsg)
        sessionStore.messages.push({
          role: 'assistant',
          content: assistantMsg.content,
          time: assistantMsg.createdAt,
          actions: assistantMsg.actions,
        })
        scrollToBottom()
        resolve()
      }, 1000)
    })
  }

  /**
   * 处理 workflow 类型操作
   * 工作流状态变更 - 触发后端事件（后端会自动生成系统消息）
   */
  async function handleWorkflowAction(action: WorkflowAction) {
    try {
      // 确保会话已初始化
      if (!sessionStore.currentSessionId) {
        console.log('No session, creating new one...')
        await createNewSession()
      }

      if (!sessionStore.currentSessionId) {
        ElMessage.error('会话初始化失败，请刷新页面重试')
        return
      }

      // 如果操作定义了字段，弹出动态表单对话框
      if (action.fields && action.fields.length > 0) {
        await showActionDialog(action)
        return
      }

      // 特殊处理：如果是 _start_ 事件，需要提供默认参数
      let payload = action.payload
      if (action.event === '_start_' && !payload) {
        // 计算默认日期（下周一到下周日）
        const defaultRange = getNextWeekRange()
        payload = {
          startDate: defaultRange.start,
          endDate: defaultRange.end,
          orgId: getCurrentOrgId(),
        }
      }

      // 通过 WebSocket 发送工作流命令（内部会获取 WebSocket 客户端）
      await sendWorkflowCommand(action.event, payload)
    }
    catch (error) {
      console.error('工作流操作失败:', error)
      ElMessage.error('操作失败，请重试')
    }
  }

  /**
   * 发送工作流命令
   */
  async function sendWorkflowCommand(event: string, payload?: Record<string, any>) {
    if (!sessionStore.currentSessionId) {
      ElMessage.error('会话未初始化')
      return
    }

    // 获取全局 WebSocket 客户端
    const wsClient = await getWsClient()
    wsClient.sendWorkflowCommand(
      sessionStore.currentSessionId,
      event,
      payload,
    )

    console.log('Workflow command sent:', event, payload)
  }

  /**
   * 显示动态表单对话框
   */
  async function showActionDialog(action: WorkflowAction) {
    currentAction.value = action
    actionDialogVisible.value = true
  }

  /**
   * 处理对话框确认
   */
  function handleDialogConfirm(formData: Record<string, any>) {
    if (currentAction.value) {
      sendWorkflowCommand(currentAction.value.event, formData)
    }
  }

  /**
   * 处理对话框取消
   */
  function handleDialogCancel() {
    console.log('User cancelled action dialog')
  }

  /**
   * 一键启动排班创建工作流
   * 此方法可被父组件调用
   */
  async function startScheduleCreationWorkflow(options?: {
    startDate?: string
    endDate?: string
    orgId?: string
  }) {
    try {
      // 确保会话已初始化
      if (!sessionStore.currentSessionId) {
        await initSession()
        // 如果仍然没有会话，创建新会话
        if (!sessionStore.currentSessionId) {
          await createNewSession()
        }
      }

      if (!sessionStore.currentSessionId) {
        ElMessage.error('无法启动工作流，请刷新页面重试')
        return
      }

      // 获取全局 WebSocket 客户端
      const wsClient = await getWsClient()

      // 计算默认日期（下周一到下周日）
      const defaultRange = getNextWeekRange()
      const payload = {
        startDate: options?.startDate || defaultRange.start,
        endDate: options?.endDate || defaultRange.end,
        orgId: options?.orgId || getCurrentOrgId(),
      }

      // 发送启动事件
      wsClient.sendWorkflowCommand(
        sessionStore.currentSessionId,
        '_start_', // EventStart
        payload,
      )

      ElMessage.success('正在启动排班创建流程...')
      console.log('Schedule creation workflow started with payload:', payload)
    }
    catch (error) {
      console.error('启动排班创建失败:', error)
      ElMessage.error('启动失败，请重试')
    }
  }

  /**
   * 处理 query 类型操作
   * 查询操作 - 弹框显示数据
   */
  function handleQueryAction(action: WorkflowAction) {
    // 检查是否是班次分组人员预览
    if (action.id === 'preview_shift_groups' && callbacks?.showStaffDetails) {
      // 使用人员详情对话框显示班次分组人员
      callbacks.showStaffDetails(action.payload)
      return
    }

    // 检查是否是人员详情查询
    if (action.id === 'view_staff_details' && callbacks?.showStaffDetails) {
      // 使用专门的人员详情对话框
      callbacks.showStaffDetails(action.payload)
      return
    }

    // 检查是否是规则详情查询
    if (action.id === 'view_rules_details' && callbacks?.showRulesDetails) {
      // 使用专门的规则详情对话框
      callbacks.showRulesDetails(action.payload)
      return
    }

    // 检查是否是临时规则详情查询
    if (action.id === 'view_temporary_rules' && callbacks?.showTemporaryRules) {
      // 使用专门的临时规则详情对话框
      callbacks.showTemporaryRules(action.payload)
      return
    }

    // 检查是否是变更详情查询
    if (action.id === 'view_changes_detail' && callbacks?.showChangeDetail) {
      // 调试日志：检查payload数据
      console.log('[ChatAssistant] 变更详情 payload:', action.payload)
      console.log('[ChatAssistant] payload.shifts:', action.payload?.shifts)
      // 使用专门的变更详情对话框
      callbacks.showChangeDetail(action.payload)
      return
    }

    // 检查是否是排班详情查询（任务执行后查看当前班次安排）
    // 包括 view_schedule_detail（旧）和 view_task_schedule_detail（新）
    if ((action.id === 'view_schedule_detail' || action.id === 'view_task_schedule_detail' || action.id === 'preview_full_schedule') && callbacks?.showMultiShiftSchedule) {
      console.log('[ChatAssistant] 排班详情 payload:', action.payload)
      // 直接使用多班次对话框显示（payload已经是正确的格式）
      const payload = action.payload as any
      if (payload?.shifts) {
        callbacks.showMultiShiftSchedule(payload)
        return
      }
      // 旧格式兼容：将 draftSchedule 转换为 多班次格式
      if (payload?.draftSchedule && callbacks?.showMultiShiftSchedule) {
        const draftSchedule = payload.draftSchedule
        const shifts: Record<string, any> = {}

        // 将 draftSchedule 中的班次数据转换为前端期望的格式
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

        // 构建前端需要的多班次数据格式
        const multiShiftData = {
          shifts,
          startDate: payload.startDate,
          endDate: payload.endDate,
          shiftInfoList: Object.keys(shifts).map(shiftId => ({
            id: shiftId,
            name: shifts[shiftId].shiftName,
          })),
        }

        callbacks.showMultiShiftSchedule(multiShiftData)
        return
      }
    }

    // 检查是否是班次排班详情查询
    if (action.id.startsWith('view_shift_schedule_') && callbacks?.showShiftSchedule) {
      callbacks.showShiftSchedule(action.payload)
      return
    }

    // 检查是否是固定班次排班详情查询
    if (action.id === 'view_fixed_shifts_detail') {
      // 固定班次数据格式: { shifts: { shiftId: { shiftId, priority, days } }, startDate, endDate, shiftInfoList }
      const fixedData = action.payload as any
      if (fixedData?.shifts) {
        const shifts = fixedData.shifts
        const shiftIds = Object.keys(shifts)
        const shiftInfoMap = new Map<string, { id: string, name: string }>()
        if (Array.isArray(fixedData.shiftInfoList)) {
          fixedData.shiftInfoList.forEach((info: any) => {
            if (info?.id && info?.name) {
              shiftInfoMap.set(info.id, { id: info.id, name: info.name })
            }
          })
        }

        // 如果有多个班次，使用多班次标签页对话框显示
        if (shiftIds.length > 1) {
          if (callbacks?.showMultiShiftSchedule) {
            callbacks.showMultiShiftSchedule(fixedData)
          }
          else {
            // 降级：如果没有回调，使用 DataViewDialog 显示
            if (dataViewDialogRef?.value) {
              dataViewDialogRef.value.open({
                title: action.label || '固定班次排班详情',
                data: fixedData,
                mode: 'json',
                showCopy: true,
                showExport: false,
              })
            }
          }
          return
        }

        // 如果只有一个班次，转换为 ShiftScheduleDialog 期望的格式
        if (shiftIds.length === 1 && callbacks?.showShiftSchedule) {
          const shiftId = shiftIds[0]
          const shiftSchedule = shifts[shiftId]
          const shiftInfo = shiftInfoMap.get(shiftId)
          const convertedData = {
            shiftId: shiftSchedule.shiftId || shiftId,
            shiftName: shiftInfo?.name || `固定班次-${shiftId}`,
            startDate: fixedData.startDate,
            endDate: fixedData.endDate,
            schedule: shiftSchedule,
          }
          callbacks.showShiftSchedule(convertedData)
          return
        }
      }
      // 降级：如果没有班次数据，使用默认处理
    }

    // 检查是否是个人需求详情查询
    if (action.id === 'view_personal_needs' && callbacks?.showPersonalNeeds) {
      callbacks.showPersonalNeeds(action.payload)
      return
    }

    // 检查是否是校验结果查询（V4检查结果）
    if (action.id === 'view_validation_result' && callbacks?.showValidationResult) {
      callbacks.showValidationResult(action.payload)
      return
    }

    // 检查是否是完整排班预览
    if (action.id === 'preview_full_schedule' && callbacks?.showSchedulePreview) {
      callbacks.showSchedulePreview(action.payload)
      return
    }

    // 其他查询使用 DataViewDialog 显示查询数据
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
      // 降级方案：使用 ElMessageBox
      ElMessageBox.alert(
        `<pre style="max-height: 400px; overflow: auto;">${JSON.stringify(action.payload || {}, null, 2)}</pre>`,
        action.label,
        {
          dangerouslyUseHTMLString: true,
          confirmButtonText: '关闭',
        },
      )
    }
  }

  /**
   * 处理 navigate 类型操作
   * 导航操作 - 跳转到其他页面
   */
  function handleNavigateAction(action: WorkflowAction) {
    const targetUrl = action.payload?.url
    if (targetUrl) {
      // 使用 vue-router 或直接跳转
      window.location.href = targetUrl
    }
    else {
      ElMessage.warning('导航地址未配置')
    }
  }

  /**
   * Action 处理器映射
   */
  const actionHandlers: ActionHandlers = {
    workflow: handleWorkflowAction,
    query: handleQueryAction,
    command: handleWorkflowAction, // command 类型暂时映射到 workflow（兼容旧代码）
    navigate: handleNavigateAction,
  }

  /**
   * 处理操作按钮点击
   */
  async function handleActionClick(action: WorkflowAction, message: ChatAssistantMessage) {
    // 处理错误操作按钮
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

  /**
   * 滚动到底部
   */
  function scrollToBottom() {
    nextTick(() => {
      if (messagesContainer.value) {
        messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
      }
    })
  }

  /**
   * 获取按钮类型映射
   */
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

  /**
   * 从 store 同步消息
   */
  function syncMessagesFromStore(storeMessages: any[]) {
    // 过滤掉空消息（content 为空、null 或 undefined）
    messages.value = storeMessages
      .filter((msg) => {
        const content = msg.content
        return content !== null && content !== undefined && content !== ''
      })
      .map(msg => ({
        id: msg.id || `${Date.now()}-${Math.random()}`,
        role: msg.role || 'assistant', // 保留 role（包括 system）
        content: msg.content,
        createdAt: msg.time || msg.timestamp || new Date().toISOString(),
        actions: msg.actions || [],
        workflowActions: msg.workflowActions || [], // 保留 workflowActions
      }))

    // 如果 store 中有 workflow.actions，同步到最后一条助手消息
    if (sessionStore.workflow && sessionStore.workflow.actions && sessionStore.workflow.actions.length > 0) {
      updateLastMessageActions(sessionStore.workflow.actions)
    }

    scrollToBottom()
  }

  /**
   * 加载历史对话
   * @param conversationId 对话ID
   */
  async function loadHistoryConversation(conversationId: string) {
    try {
      loading.value = true
      console.log('[ChatAssistant] Loading history conversation:', conversationId)

      // 确保有当前 session
      if (!sessionStore.currentSessionId) {
        // 如果没有 session，先创建一个
        const orgId = getCurrentOrgId()
        const userId = getCurrentUserId()
        const wsClient = await getWsClient()
        wsClient.createSession(orgId, userId, 'rostering')
        await new Promise(resolve => setTimeout(resolve, 500))
      }

      if (!sessionStore.currentSessionId) {
        throw new Error('无法创建或获取当前会话')
      }

      // 通过 WebSocket 发送 load_conversation 消息
      const wsClient = await getWsClient()
      wsClient.loadConversation(sessionStore.currentSessionId, conversationId)

      // 等待后端处理并推送更新（通过 WebSocket 的 session_updated 消息）
      await new Promise(resolve => setTimeout(resolve, 1500))

      // 同步消息列表
      syncMessagesFromStore(sessionStore.messages)

      // 同步工作流状态
      if (sessionStore.workflow && sessionStore.workflow.actions && sessionStore.workflow.actions.length > 0) {
        updateLastMessageActions(sessionStore.workflow.actions)
      }

      // 滚动到底部
      await nextTick()
      scrollToBottom()

      ElMessage.success('历史对话加载成功')
      console.log('[ChatAssistant] History conversation loaded successfully')
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

    // Session 管理方法
    initSession,
    createNewSession,
    disconnectWebSocket,

    // 原有方法
    sendMessage,
    handleActionClick,
    getButtonType,
    scrollToBottom,
    updateLastMessageActions,
    syncMessagesFromStore,

    // 工作流启动方法
    startScheduleCreationWorkflow,

    // 对话框处理
    handleDialogConfirm,
    handleDialogCancel,

    // 对话历史管理
    loadHistoryConversation,
  }
}
