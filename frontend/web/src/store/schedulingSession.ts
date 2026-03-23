import { defineStore } from 'pinia'
import { sessionApi, type ConversationSummary } from '@/services/api'
import type { SchedulingWsClient } from '@/services/scheduling-ws'
import { createSchedulingWsClient } from '@/services/scheduling-ws'

// 全局 WebSocket 客户端实例（单例）
let globalWsClient: SchedulingWsClient | null = null

// 班次进度信息类型
export interface ShiftProgressInfo {
  shiftId: string
  shiftName: string
  current: number // 当前第几个班次
  total: number // 总班次数
  status: string // day_generating, day_completed, shift_validating, shift_retrying, shift_success, shift_failed
  message: string
  reasoning?: string
  previewData?: string
  // 天级别进度
  currentDay?: number
  totalDays?: number
  currentDate?: string
  completedDates?: string[]
  draftPreview?: string
}

// 该 Store 管理排班会话状态和 WebSocket 消息
export const useSchedulingSessionStore = defineStore('schedulingSession', {
  state: () => ({
    currentSessionId: null as string | null,
    session: null as any,
    workflow: null as any,
    decisions: [] as Array<any>,
    context: null as any,
    rules: null as any,
    messages: [] as any[],
    lastError: null as any,
    isLoading: false,
    wsClient: null as SchedulingWsClient | null, // WebSocket 客户端实例
    
    // 班次排班进度（新增）
    shiftProgress: null as ShiftProgressInfo | null,
    // 实时草案预览（新增）
    realtimeDraft: null as any,
    // 是否显示进度条（新增）
    showProgressBar: false,
  }),
  actions: {
    // ==================== Session 管理 ====================

    /**
     * 加载最近会话
     */
    async loadRecentSession(orgId: string, userId: string) {
      this.isLoading = true
      try {
        const session = await sessionApi.getRecentSession(orgId, userId)
        if (session) {
          this.currentSessionId = session.id
          this.session = session
          this.messages = session.messages || []
          this.workflow = session.workflowMeta || null
          return session
        }
        return null
      }
      catch (error) {
        console.error('加载最近会话失败:', error)
        this.lastError = error
        return null
      }
      finally {
        this.isLoading = false
      }
    },

    /**
     * 创建新会话
     */
    async createNewSession(orgId: string, userId: string, agentType: string) {
      this.isLoading = true
      try {
        const response = await sessionApi.createSession({
          orgId,
          userId,
          agentType,
        })
        this.currentSessionId = response.sessionId
        this.messages = []
        this.workflow = null
        this.session = null
        return response
      }
      catch (error) {
        console.error('创建新会话失败:', error)
        this.lastError = error
        throw error
      }
      finally {
        this.isLoading = false
      }
    },

    /**
     * 清空当前会话
     */
    clearSession() {
      this.currentSessionId = null
      this.session = null
      this.messages = []
      this.workflow = null
      this.decisions = []
      this.context = null
      this.rules = null
    },

    /**
     * 加载历史对话
     * @param conversationId 对话ID
     * @param sessionId 当前会话ID
     */
    async loadConversation(conversationId: string, sessionId: string) {
      this.isLoading = true
      try {
        await sessionApi.loadConversation(conversationId, sessionId)
        // 等待后端推送更新，这里不直接更新状态
        // 状态会通过 WebSocket 消息自动更新
      }
      catch (error) {
        console.error('加载历史对话失败:', error)
        this.lastError = error
        throw error
      }
      finally {
        this.isLoading = false
      }
    },

    // ==================== WebSocket 连接管理 ====================

    /**
     * 获取或创建全局 WebSocket 客户端
     */
    async getOrCreateWsClient(): Promise<SchedulingWsClient> {
      // 如果 store 中已有客户端且已连接，直接返回
      if (this.wsClient && this.wsClient.isConnected()) {
        return this.wsClient
      }

      // 如果全局实例存在且已连接，使用全局实例
      if (globalWsClient && globalWsClient.isConnected()) {
        this.wsClient = globalWsClient
        return globalWsClient
      }

      // 创建新的 WebSocket 客户端
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      const host = import.meta.env.VITE_WS_HOST || window.location.host
      const subPath = import.meta.env.VITE_WS_URL || '/api/scheduling/ws'
      const wsUrl = `${protocol}//${host}${subPath}`

      const client = createSchedulingWsClient({
        url: wsUrl,
        reconnectAttempts: 999, // 无限重连（直到手动关闭）
        reconnectInterval: 3000,
        heartbeatInterval: 30000,
      })

      // 连接到服务器
      await client.connect()

      // 保存到全局和 store
      globalWsClient = client
      this.wsClient = client

      console.log('[SchedulingSession] WebSocket 客户端已创建并连接')
      return client
    },

    /**
     * 断开 WebSocket 连接（仅在应用关闭时调用）
     */
    disconnectWsClient() {
      if (this.wsClient) {
        this.wsClient.disconnect('应用关闭')
        this.wsClient = null
      }
      if (globalWsClient) {
        globalWsClient.disconnect('应用关闭')
        globalWsClient = null
      }
      console.log('[SchedulingSession] WebSocket 客户端已断开')
    },

    // ==================== WebSocket 消息处理 ====================

    applySnapshot(data: any) {
      if (!data) {
        return
      }
      if (data.dto) {
        this.session = data.dto
        // 更新 currentSessionId
        if (data.dto.id) {
          this.currentSessionId = data.dto.id
        }
        // 同步消息（过滤空消息）
        if (Array.isArray(data.dto.messages)) {
          this.messages = data.dto.messages
            .filter((msg: any) => {
              const content = msg.content
              return content !== null && content !== undefined && content !== ''
            })
            .map((msg: any) => ({
              role: msg.role || 'assistant',
              content: msg.content,
              time: msg.createdAt || msg.time,
              actions: msg.actions,
            }))
        }
        // 同步 workflowMeta
        if (data.dto.workflowMeta) {
          this.workflow = data.dto.workflowMeta
        }
      }
      if (Array.isArray(data.decisions)) {
        this.decisions = data.decisions
      }
    },
    updateSession(dto: any) {
      this.session = dto

      // 同步 workflowMeta（交互按钮）
      if (dto && dto.workflowMeta) {
        this.workflow = dto.workflowMeta
      }

      // 同步 messages（后端推送的 session 包含完整的 Messages 数组，包括系统消息）
      // 过滤掉空消息（content 为空、null 或 undefined）
      if (dto && Array.isArray(dto.messages)) {
        this.messages = dto.messages
          .filter((msg: any) => {
            const content = msg.content
            return content !== null && content !== undefined && content !== ''
          })
          .map((msg: any) => ({
            role: msg.role || 'assistant',
            content: msg.content,
            time: msg.createdAt || msg.time,
            actions: msg.actions,
          }))
      }
    },
    updateWorkflow(data: any) {
      // workflow_update 事件只包含状态变更信息 {phase, from, event}
      // 需要合并更新,保留现有的 actions 等字段
      console.log('[schedulingSession] updateWorkflow called', {
        data,
        currentActions: this.workflow?.actions?.length || 0,
      })

      if (this.workflow) {
        this.workflow = {
          ...this.workflow,
          ...data,
        }
        console.log('[schedulingSession] workflow merged', {
          phase: this.workflow.phase,
          actionsPreserved: this.workflow.actions?.length || 0,
        })
      }
      else {
        this.workflow = data
        console.log('[schedulingSession] workflow initialized', data)
      }
    },
    updateWorkflowMeta(data: any) {
      // 只更新 WorkflowMeta,不影响 messages (data 格式: {sessionId, workflowMeta})
      if (data && data.workflowMeta) {
        this.workflow = data.workflowMeta
        console.log('[schedulingSession] workflowMeta updated independently', {
          actionsCount: data.workflowMeta.actions?.length || 0,
        })
      }
    },
    clearWorkflowActions() {
      // 清空 WorkflowMeta 中的 actions
      if (this.workflow) {
        this.workflow.actions = []
        console.log('[schedulingSession] workflow actions cleared')
      }
    },
    appendDecisions(data: any) {
      const list = (data && (data.list || data.List)) || []
      if (Array.isArray(list)) {
        this.decisions.push(...list)
      }
    },
    updateContext(data: any) {
      if (data && data.context) {
        this.context = data.context
      }
    },
    updateRules(data: any) {
      this.rules = data?.rules ?? data
    },
    onFinalizeCompleted(data: any) {
      if (data && data.dto) {
        this.session = data.dto
      }
    },
    addAssistantMessage(data: any) {
      if (!data) {
        return
      }
      // 过滤空消息
      const content = data.content
      if (content === null || content === undefined || content === '') {
        return
      }
      this.messages.push({ role: (data.role || 'assistant'), content: data.content, time: data.time })
    },
    onError(err: any) {
      this.lastError = err
      console.error('[SchedulingSession] WebSocket error:', err)

      // 错误消息通常通过 AddAssistantMessageWithActions 添加到 Session.Messages 中
      // 并通过 session_updated 广播到前端，这里只处理特殊情况
      // 如果错误消息包含 actions，将其添加到消息中
      if (err && err.message && Array.isArray(err.actions) && err.actions.length > 0) {
        const errorMessage = {
          id: `error-${Date.now()}`,
          role: 'assistant',
          content: err.message,
          createdAt: new Date().toISOString(),
          actions: err.actions.map((action: any) => ({
            id: action.id,
            type: action.type || 'workflow',
            label: action.label,
            event: action.event,
            style: action.style || 'primary',
            payload: action.payload,
          })),
        }
        this.messages.push(errorMessage)
      } else if (err && err.message) {
        // 如果没有 actions，也显示错误消息（作为系统消息或助手消息）
        const errorMessage = {
          id: `error-${Date.now()}`,
          role: 'assistant',
          content: `[错误] ${err.message}`,
          createdAt: new Date().toISOString(),
        }
        this.messages.push(errorMessage)
      }
    },
    
    // ==================== 班次进度管理（新增） ====================
    
    /**
     * 更新班次进度
     */
    updateShiftProgress(info: ShiftProgressInfo) {
      this.shiftProgress = info
      this.showProgressBar = true
      
      // 解析草案预览
      if (info.draftPreview) {
        try {
          this.realtimeDraft = JSON.parse(info.draftPreview)
        } catch (e) {
          console.warn('Failed to parse draft preview:', e)
        }
      }
      
      // 如果班次完成或失败，延迟隐藏进度条
      if (info.status === 'shift_success' || info.status === 'shift_failed') {
        setTimeout(() => {
          // 如果是最后一个班次，隐藏进度条
          if (info.current >= info.total) {
            this.showProgressBar = false
            this.shiftProgress = null
          }
        }, 3000)
      }
    },
    
    /**
     * 清除班次进度
     */
    clearShiftProgress() {
      this.shiftProgress = null
      this.realtimeDraft = null
      this.showProgressBar = false
    },
  },
})
