// WebSocket 相关类型定义

// 后端兼容的消息格式
export interface ClientMessage {
  type: string
  sessionId: string
  data?: any
  ts?: string
  seq?: number
}

export interface ServerMessage {
  type: string
  sessionId?: string
  data?: any
  ts?: string
  eventId?: number
}

// 原有的通用封装格式（保持兼容）
export interface WsEnvelope {
  id?: string
  type: WsEventType
  payload?: any
  timestamp?: string
}

export type WsEventType
  // 上行事件（客户端 -> 服务端）
  = | 'user_message'
    | 'workflow_command'
    | 'context_collect'
    | 'start_generate'
    | 'finalize'
    | 'ping'
    | 'fetch_decisions'
    | 'fetch_snapshot'

  // 下行事件（服务端 -> 客户端）
    | 'session_snapshot'
    | 'assistant_message'
    | 'session_updated'
    | 'decision_log_append'
    | 'workflow_update'
    | 'context_update'
    | 'rule_update'
    | 'validation_result'
    | 'finalize_completed'
    | 'error'
    | 'pong'

  // 保持兼容的前端原有类型
    | 'command'
    | 'schedule_draft'
    | 'schedule_final'
    | 'schedule_query_result'
    | 'progress'
    | 'session_info'

export interface UserMessageEvent {
  type: 'user_message'
  text: string
  meta?: {
    timeRange?: {
      start: string
      end: string
      timezone: string
    }
    filters?: any
  }
}

export interface CommandEvent {
  type: 'command'
  name: 'finalize' | 'query' | 'apply_template' | 'undo' | 'redo' | 'optimize'
  payload?: any
}

export interface AssistantMessageEvent {
  type: 'assistant_message'
  text: string
  partial?: boolean // 是否为流式消息的一部分
}

export interface ScheduleEvent {
  type: 'schedule_draft' | 'schedule_final' | 'schedule_query_result'
  data: any // Schedule 对象
}

export interface ProgressEvent {
  type: 'progress'
  stage: 'planning' | 'optimizing' | 'validating' | 'finalizing'
  percent: number
  message?: string
}

export interface ErrorEvent {
  type: 'error'
  code: string
  message: string
  details?: any
}

export type SocketState = 'idle' | 'connecting' | 'open' | 'closed' | 'error'

export interface WebSocketOptions {
  url: string
  protocols?: string[]
  heartbeatInterval?: number // 心跳间隔（毫秒）
  reconnectAttempts?: number // 重连次数
  reconnectInterval?: number // 重连间隔（毫秒）
}
