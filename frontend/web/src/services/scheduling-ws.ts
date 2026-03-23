import type { WebSocketManager } from '@/services/ws'
import type { WebSocketOptions } from '@/types/ws'
import { createWebSocketManager } from '@/services/ws'
import { useSchedulingSessionStore } from '@/store/schedulingSession'

// 后端 WS 消息类型常量（与服务端保持一致）
export const WS_MSG = {
  // client -> server
  GetRecentSession: 'get_recent_session', // 获取最近的会话
  CreateSession: 'create_session', // 创建新会话
  UserMessage: 'user_message',
  WorkflowCommand: 'workflow_command',
  ContextCollect: 'context_collect',
  FetchSnapshot: 'fetch_snapshot',
  FetchDecisions: 'fetch_decisions',
  LoadConversation: 'load_conversation', // 加载历史对话
  Ping: 'ping',

  // server -> client
  SessionSnapshot: 'session_snapshot', // 会话快照（包含最近会话数据）
  SessionCreated: 'session_created', // 新会话创建成功
  AssistantMessage: 'assistant_message',
  SessionUpdated: 'session_updated',
  WorkflowMetaUpdated: 'workflow_meta_updated', // WorkflowMeta 更新（只包含按钮等元数据）
  DecisionLogAppend: 'decision_log_append',
  WorkflowUpdate: 'workflow_update',
  ContextUpdate: 'context_update',
  RuleUpdate: 'rule_update',
  ValidationResult: 'validation_result',
  FinalizeCompleted: 'finalize_completed',
  ShiftProgress: 'shift_progress', // 班次排班进度（新增）
  Error: 'error',
  Pong: 'pong',
} as const

export type WsMessageType = typeof WS_MSG[keyof typeof WS_MSG]

export interface SchedulingWsClientOptions extends Partial<WebSocketOptions> {
  url: string // ws://host/scheduling/ws
}

export class SchedulingWsClient {
  private ws!: WebSocketManager
  private readonly url: string

  constructor(opts: SchedulingWsClientOptions) {
    this.url = opts.url
    this.ws = createWebSocketManager({
      url: this.url,
      heartbeatInterval: opts.heartbeatInterval ?? 30000,
      reconnectAttempts: opts.reconnectAttempts ?? 10,
      reconnectInterval: opts.reconnectInterval ?? 3000,
      protocols: opts.protocols ?? [],
    })

    const store = useSchedulingSessionStore()
    this.ws.onMessage = (msg) => {
      const t = msg.type as WsMessageType
      switch (t) {
        case WS_MSG.SessionSnapshot:
          store.applySnapshot(msg.data)
          break
        case WS_MSG.SessionCreated:
          // 新会话创建成功，更新 currentSessionId
          if (msg.data && msg.data.sessionId) {
            store.currentSessionId = msg.data.sessionId
            console.log('Session created:', msg.data.sessionId)
          }
          break
        case WS_MSG.SessionUpdated:
          store.updateSession(msg.data)
          break
        case WS_MSG.WorkflowMetaUpdated:
          // 只更新 WorkflowMeta,不影响 messages
          store.updateWorkflowMeta(msg.data)
          break
        case WS_MSG.WorkflowUpdate:
          store.updateWorkflow(msg.data)
          break
        case WS_MSG.DecisionLogAppend:
          store.appendDecisions(msg.data)
          break
        case WS_MSG.ContextUpdate:
          store.updateContext(msg.data)
          break
        case WS_MSG.RuleUpdate:
          store.updateRules(msg.data)
          break
        case WS_MSG.FinalizeCompleted:
          store.onFinalizeCompleted(msg.data)
          break
        case WS_MSG.AssistantMessage:
          store.addAssistantMessage(msg.data)
          break
        case WS_MSG.ShiftProgress:
          // 班次排班进度更新
          store.updateShiftProgress(msg.data)
          break
        case WS_MSG.Error:
          store.onError(msg.data)
          break
        case WS_MSG.Pong:
          // ignore
          break
        default:
          // 兼容未知类型：按需记录
          console.debug('[ws] unhandled message', msg)
      }
    }
  }

  async connect() {
    await this.ws.connect()
    return this
  }

  isConnected(): boolean {
    return this.ws.isConnected()
  }

  getState() {
    return this.ws.getState()
  }

  // Session 管理方法
  getRecentSession(orgId: string, userId: string) {
    this.ws.send({
      type: WS_MSG.GetRecentSession,
      data: { orgId, userId },
    })
  }

  createSession(orgId: string, userId: string, agentType: string) {
    this.ws.send({
      type: WS_MSG.CreateSession,
      data: { orgId, userId, agentType },
    })
  }

  // 上行消息构造
  fetchSnapshot(sessionId: string) {
    this.ws.send({ type: WS_MSG.FetchSnapshot, sessionId, data: { parts: ['session', 'decisions'] } })
  }

  fetchDecisions(sessionId: string, opts?: Record<string, any>) {
    this.ws.send({ type: WS_MSG.FetchDecisions, sessionId, data: { ...opts } })
  }

  // 工作流相关方法
  sendUserMessage(sessionId: string, message: string) {
    this.ws.send({
      type: WS_MSG.UserMessage,
      data: { sessionId, message },
    })
  }

  sendWorkflowCommand(sessionId: string, command: string, payload?: Record<string, any>) {
    console.log('[SchedulingWsClient] Sending workflow command:', {
      sessionId,
      command,
      payload,
      wsState: this.ws.getState(),
    })
    this.ws.send({
      type: WS_MSG.WorkflowCommand,
      data: { sessionId, command, payload },
    })
  }

  collectContext(sessionId: string, force?: boolean) {
    this.ws.send({ type: WS_MSG.ContextCollect, sessionId, data: { force: !!force } })
  }

  /**
   * 加载历史对话到当前 session
   * @param sessionId 会话ID
   * @param conversationId 对话ID
   */
  loadConversation(sessionId: string, conversationId: string): void {
    this.ws.send({
      type: WS_MSG.LoadConversation,
      sessionId,
      data: { sessionId, conversationId },
    })
  }

  disconnect(reason?: string) {
    this.ws.close(reason)
  }
}

export function createSchedulingWsClient(opts: SchedulingWsClientOptions) {
  return new SchedulingWsClient(opts)
}

// Types
// Note: keep opts as Record<string, any> to avoid TS config incompatibilities in some environments.
