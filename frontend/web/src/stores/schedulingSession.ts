// 排班会话状态管理 — AI 排班 Agent 工作流
import type { ChatMessage, Session, WorkflowAction } from '@/types/chat'
import type { ShiftProgressInfo } from '@/types/scheduling'
import type { ServerMessage } from '@/types/ws'
import { defineStore } from 'pinia'
import { computed, ref, shallowRef } from 'vue'
import { SchedulingWsClient, WS_MSG } from '@/services/scheduling-ws'

// 全局 WebSocket 单例
let globalWsClient: SchedulingWsClient | null = null

export const useSchedulingSessionStore = defineStore('schedulingSession', () => {
  // ==================== 状态 ====================

  const currentSessionId = ref<string | null>(null)
  const session = ref<Session | null>(null)
  const messages = ref<ChatMessage[]>([])

  // 工作流元信息
  const workflow = ref<{
    workflow: string
    phase: string
    actions?: WorkflowAction[]
    extra?: Record<string, unknown>
    [key: string]: unknown
  } | null>(null)

  const decisions = ref<unknown[]>([])
  const context = ref<unknown>(null)
  const rules = ref<unknown>(null)

  const lastError = ref<unknown>(null)
  const isLoading = ref(false)

  // 排班进度
  const shiftProgress = ref<ShiftProgressInfo | null>(null)
  const realtimeDraft = ref<unknown>(null)
  const showProgressBar = ref(false)

  // WebSocket 客户端 (shallow 避免深度代理)
  const wsClient = shallowRef<SchedulingWsClient | null>(null)

  // ==================== 计算属性 ====================

  const currentPhase = computed(() => workflow.value?.phase ?? '')
  const workflowActions = computed(() => workflow.value?.actions ?? [])
  const hasSession = computed(() => !!currentSessionId.value)

  // ==================== Session 管理 ====================

  async function loadRecentSession(orgId: string, userId: string) {
    isLoading.value = true
    try {
      // 通过 WS 请求最近会话
      const client = await getOrCreateWsClient()
      client.getRecentSession(orgId, userId)
      // 结果会通过 WS 消息推送（session_snapshot）
    }
    catch (error) {
      console.error('加载最近会话失败:', error)
      lastError.value = error
    }
    finally {
      isLoading.value = false
    }
  }

  async function createNewSession(orgId: string, userId: string, agentType: string) {
    isLoading.value = true
    try {
      const client = await getOrCreateWsClient()
      client.createSession(orgId, userId, agentType)
      // 重置本地状态
      messages.value = []
      workflow.value = null
      decisions.value = []
      context.value = null
      rules.value = null
    }
    catch (error) {
      console.error('创建新会话失败:', error)
      lastError.value = error
      throw error
    }
    finally {
      isLoading.value = false
    }
  }

  function clearSession() {
    currentSessionId.value = null
    session.value = null
    messages.value = []
    workflow.value = null
    decisions.value = []
    context.value = null
    rules.value = null
  }

  async function loadConversation(conversationId: string, sessionId: string) {
    isLoading.value = true
    try {
      const client = await getOrCreateWsClient()
      client.loadConversation(sessionId, conversationId)
      // 状态会通过 WS 消息自动更新
    }
    catch (error) {
      console.error('加载历史对话失败:', error)
      lastError.value = error
      throw error
    }
    finally {
      isLoading.value = false
    }
  }

  // ==================== WebSocket 管理 ====================

  async function getOrCreateWsClient(): Promise<SchedulingWsClient> {
    // store 中已有且已连接
    if (wsClient.value && wsClient.value.isConnected) {
      return wsClient.value
    }

    // 全局实例已连接
    if (globalWsClient && globalWsClient.isConnected) {
      wsClient.value = globalWsClient
      return globalWsClient
    }

    // 创建新的 WebSocket 客户端
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = import.meta.env.VITE_WS_HOST || window.location.host
    const subPath = import.meta.env.VITE_WS_URL || '/api/v1/ws/scheduling'
    const wsUrl = `${protocol}//${host}${subPath}`

    const client = new SchedulingWsClient(wsUrl)

    // 注册消息处理器
    registerMessageHandlers(client)

    await client.connect()

    globalWsClient = client
    wsClient.value = client

    return client
  }

  function disconnectWsClient() {
    if (wsClient.value) {
      wsClient.value.disconnect()
      wsClient.value = null
    }
    if (globalWsClient) {
      globalWsClient.disconnect()
      globalWsClient = null
    }
  }

  function registerMessageHandlers(client: SchedulingWsClient) {
    client.on(WS_MSG.SessionSnapshot, (msg: ServerMessage) => applySnapshot(msg.data))
    client.on(WS_MSG.SessionUpdated, (msg: ServerMessage) => updateSession(msg.data))
    client.on(WS_MSG.AssistantMessage, (msg: ServerMessage) => addAssistantMessage(msg.data))
    client.on(WS_MSG.WorkflowUpdate, (msg: ServerMessage) => updateWorkflow(msg.data))
    client.on(WS_MSG.WorkflowMetaUpdate, (msg: ServerMessage) => updateWorkflowMeta(msg.data))
    client.on(WS_MSG.DecisionLogAppend, (msg: ServerMessage) => appendDecisions(msg.data))
    client.on(WS_MSG.ContextUpdate, (msg: ServerMessage) => updateContext(msg.data))
    client.on(WS_MSG.RuleUpdate, (msg: ServerMessage) => updateRules(msg.data))
    client.on(WS_MSG.FinalizeCompleted, (msg: ServerMessage) => onFinalizeCompleted(msg.data))
    client.on(WS_MSG.ShiftProgress, (msg: ServerMessage) => updateShiftProgress(msg.data))
    client.on(WS_MSG.Error, (msg: ServerMessage) => onError(msg.data))
  }

  // ==================== WS 消息处理 ====================

  function filterEmptyMessages(rawMessages: any[]): ChatMessage[] {
    return rawMessages
      .filter((msg: any) => {
        const c = msg.content
        return c !== null && c !== undefined && c !== ''
      })
      .map((msg: any) => ({
        id: msg.id || `msg-${Date.now()}-${Math.random()}`,
        role: msg.role || 'assistant',
        content: msg.content,
        createdAt: msg.createdAt || msg.time || new Date().toISOString(),
        actions: msg.actions,
      }))
  }

  function applySnapshot(data: any) {
    if (!data)
      return
    if (data.dto) {
      session.value = data.dto
      if (data.dto.id)
        currentSessionId.value = data.dto.id
      if (Array.isArray(data.dto.messages))
        messages.value = filterEmptyMessages(data.dto.messages)
      if (data.dto.workflowMeta)
        workflow.value = data.dto.workflowMeta
    }
    if (Array.isArray(data.decisions))
      decisions.value = data.decisions
  }

  function updateSession(dto: any) {
    session.value = dto
    if (dto?.workflowMeta)
      workflow.value = dto.workflowMeta
    if (dto && Array.isArray(dto.messages))
      messages.value = filterEmptyMessages(dto.messages)
  }

  function updateWorkflow(data: any) {
    if (workflow.value) {
      workflow.value = { ...workflow.value, ...data }
    }
    else {
      workflow.value = data
    }
  }

  function updateWorkflowMeta(data: any) {
    if (data?.workflowMeta)
      workflow.value = data.workflowMeta
  }

  function clearWorkflowActions() {
    if (workflow.value)
      workflow.value.actions = []
  }

  function appendDecisions(data: any) {
    const list = data?.list || data?.List || []
    if (Array.isArray(list))
      decisions.value.push(...list)
  }

  function updateContext(data: any) {
    if (data?.context)
      context.value = data.context
  }

  function updateRules(data: any) {
    rules.value = data?.rules ?? data
  }

  function onFinalizeCompleted(data: any) {
    if (data?.dto)
      session.value = data.dto
  }

  function addAssistantMessage(data: any) {
    if (!data)
      return
    const content = data.content
    if (content === null || content === undefined || content === '')
      return
    messages.value.push({
      id: data.id || `msg-${Date.now()}`,
      role: data.role || 'assistant',
      content: data.content,
      createdAt: data.time || new Date().toISOString(),
      actions: data.actions,
    })
  }

  function onError(err: any) {
    lastError.value = err
    console.error('[SchedulingSession] error:', err)

    if (err?.message && Array.isArray(err.actions) && err.actions.length > 0) {
      messages.value.push({
        id: `error-${Date.now()}`,
        role: 'assistant',
        content: err.message,
        createdAt: new Date().toISOString(),
        actions: err.actions.map((a: any) => ({
          id: a.id,
          type: a.type || 'workflow',
          label: a.label,
          event: a.event,
          style: a.style || 'primary',
          payload: a.payload,
        })),
      })
    }
    else if (err?.message) {
      messages.value.push({
        id: `error-${Date.now()}`,
        role: 'assistant',
        content: `[错误] ${err.message}`,
        createdAt: new Date().toISOString(),
      })
    }
  }

  // ==================== 班次进度 ====================

  function updateShiftProgress(info: any) {
    shiftProgress.value = info
    showProgressBar.value = true

    if (info.draftPreview) {
      try {
        realtimeDraft.value = JSON.parse(info.draftPreview)
      }
      catch {
        // ignore
      }
    }

    if (info.status === 'shift_success' || info.status === 'shift_failed') {
      setTimeout(() => {
        if (info.current >= info.total) {
          showProgressBar.value = false
          shiftProgress.value = null
        }
      }, 3000)
    }
  }

  function clearShiftProgress() {
    shiftProgress.value = null
    realtimeDraft.value = null
    showProgressBar.value = false
  }

  // ==================== 暴露 ====================

  return {
    // 状态
    currentSessionId,
    session,
    messages,
    workflow,
    decisions,
    context,
    rules,
    lastError,
    isLoading,
    wsClient,
    shiftProgress,
    realtimeDraft,
    showProgressBar,

    // 计算属性
    currentPhase,
    workflowActions,
    hasSession,

    // Session
    loadRecentSession,
    createNewSession,
    clearSession,
    loadConversation,

    // WS
    getOrCreateWsClient,
    disconnectWsClient,

    // WS 消息处理（可从外部手动调用）
    applySnapshot,
    updateSession,
    updateWorkflow,
    updateWorkflowMeta,
    clearWorkflowActions,
    appendDecisions,
    updateContext,
    updateRules,
    onFinalizeCompleted,
    addAssistantMessage,
    onError,

    // 进度
    updateShiftProgress,
    clearShiftProgress,
  }
})
