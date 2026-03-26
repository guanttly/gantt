// 排班 WebSocket 客户端 — 与后端 AI 排班 Agent 通信
import type { ChatMessage, WorkflowAction } from '@/types/chat'
import type { ServerMessage } from '@/types/ws'
import { useAuthStore } from '@/stores/auth'

// ==================== WebSocket 消息类型常量 ====================

export const WS_MSG = {
  // 客户端 → 服务端
  GetRecentSession: 'get_recent_session',
  CreateSession: 'create_session',
  UserMessage: 'user_message',
  WorkflowCommand: 'workflow_command',
  ContextCollect: 'context_collect',
  FetchSnapshot: 'fetch_snapshot',
  FetchDecisions: 'fetch_decisions',
  LoadConversation: 'load_conversation',
  Ping: 'ping',
  // 服务端 → 客户端
  SessionSnapshot: 'session_snapshot',
  AssistantMessage: 'assistant_message',
  SessionUpdated: 'session_updated',
  DecisionLogAppend: 'decision_log_append',
  WorkflowUpdate: 'workflow_update',
  WorkflowMetaUpdate: 'workflow_meta_update',
  ContextUpdate: 'context_update',
  RuleUpdate: 'rule_update',
  ValidationResult: 'validation_result',
  FinalizeCompleted: 'finalize_completed',
  ShiftProgress: 'shift_progress',
  Error: 'error',
  Pong: 'pong',
} as const

// ==================== 消息处理器类型 ====================

export interface SessionUpdateData {
  session_id: string
  messages: ChatMessage[]
  workflow?: WorkflowState
}

export interface WorkflowState {
  workflow: string
  phase: string
  actions?: WorkflowAction[]
  extra?: Record<string, unknown>
  [key: string]: unknown
}

export interface ShiftProgressData {
  shift_id: string
  shift_name: string
  status: 'pending' | 'running' | 'success' | 'failed' | 'skipped'
  progress: number
  message?: string
  error?: string
}

export type MessageHandler = (msg: ServerMessage) => void

// ==================== SchedulingWsClient ====================

export class SchedulingWsClient {
  private ws: WebSocket | null = null
  private url: string
  private reconnectAttempts = 0
  private maxReconnectAttempts = 5
  private reconnectInterval = 3000
  private heartbeatTimer: ReturnType<typeof setInterval> | null = null
  private messageHandlers: Map<string, MessageHandler[]> = new Map()
  private pendingMessages: string[] = []

  constructor(url?: string) {
    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
    this.url = url || `${protocol}//${location.host}/api/v1/ws/scheduling`
  }

  // ==================== 连接管理 ====================

  connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      const auth = useAuthStore()
      const wsUrl = `${this.url}?token=${auth.accessToken}`

      this.ws = new WebSocket(wsUrl)

      this.ws.onopen = () => {
        this.reconnectAttempts = 0
        this.startHeartbeat()
        this.flushPendingMessages()
        resolve()
      }

      this.ws.onclose = (event) => {
        this.stopHeartbeat()
        if (!event.wasClean && this.reconnectAttempts < this.maxReconnectAttempts) {
          this.reconnectAttempts++
          setTimeout(() => this.connect(), this.reconnectInterval)
        }
      }

      this.ws.onerror = () => {
        reject(new Error('WebSocket connection failed'))
      }

      this.ws.onmessage = (event) => {
        try {
          const msg: ServerMessage = JSON.parse(event.data)
          this.dispatchMessage(msg)
        }
        catch (e) {
          console.error('[SchedulingWs] Failed to parse message:', e)
        }
      }
    })
  }

  disconnect() {
    this.stopHeartbeat()
    if (this.ws) {
      this.ws.close(1000, 'Client disconnect')
      this.ws = null
    }
  }

  get isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN
  }

  // ==================== 消息发送 ====================

  private send(type: string, sessionId: string, data?: unknown) {
    const msg = JSON.stringify({
      type,
      sessionId,
      data,
      ts: new Date().toISOString(),
    })

    if (this.isConnected) {
      this.ws!.send(msg)
    }
    else {
      this.pendingMessages.push(msg)
    }
  }

  private flushPendingMessages() {
    while (this.pendingMessages.length > 0 && this.isConnected) {
      this.ws!.send(this.pendingMessages.shift()!)
    }
  }

  // ==================== 心跳 ====================

  private startHeartbeat() {
    this.heartbeatTimer = setInterval(() => {
      if (this.isConnected) {
        this.ws!.send(JSON.stringify({ type: WS_MSG.Ping }))
      }
    }, 30000)
  }

  private stopHeartbeat() {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer)
      this.heartbeatTimer = null
    }
  }

  // ==================== 消息分发 ====================

  on(type: string, handler: MessageHandler) {
    if (!this.messageHandlers.has(type)) {
      this.messageHandlers.set(type, [])
    }
    this.messageHandlers.get(type)!.push(handler)
  }

  off(type: string, handler?: MessageHandler) {
    if (handler) {
      const handlers = this.messageHandlers.get(type) || []
      this.messageHandlers.set(type, handlers.filter(h => h !== handler))
    }
    else {
      this.messageHandlers.delete(type)
    }
  }

  private dispatchMessage(msg: ServerMessage) {
    // 按类型分发
    const handlers = this.messageHandlers.get(msg.type) || []
    handlers.forEach(h => h(msg))

    // 也发给 * 通配监听器
    const wildcardHandlers = this.messageHandlers.get('*') || []
    wildcardHandlers.forEach(h => h(msg))
  }

  // ==================== 业务方法 ====================

  getRecentSession(orgId: string, userId: string) {
    this.send(WS_MSG.GetRecentSession, '', { orgId, userId })
  }

  createSession(orgId: string, userId: string, agentType: string) {
    this.send(WS_MSG.CreateSession, '', { orgId, userId, agentType })
  }

  sendUserMessage(sessionId: string, text: string) {
    this.send(WS_MSG.UserMessage, sessionId, { text })
  }

  sendWorkflowCommand(sessionId: string, event: string, payload?: Record<string, unknown>) {
    this.send(WS_MSG.WorkflowCommand, sessionId, { event, ...payload })
  }

  fetchSnapshot(sessionId: string) {
    this.send(WS_MSG.FetchSnapshot, sessionId)
  }

  fetchDecisions(sessionId: string) {
    this.send(WS_MSG.FetchDecisions, sessionId)
  }

  loadConversation(sessionId: string, conversationId: string) {
    this.send(WS_MSG.LoadConversation, sessionId, { conversationId })
  }

  collectContext(sessionId: string) {
    this.send(WS_MSG.ContextCollect, sessionId)
  }
}
