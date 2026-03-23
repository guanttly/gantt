import type { ChatMessage, WorkflowAction } from '@/types/chat'

// 组件内部消息类型扩展
export interface ChatAssistantMessage extends ChatMessage {
  // 消息是否已读
  isRead?: boolean
  // 消息是否正在发送
  isSending?: boolean
  // 临时工作流操作（不持久化）
  workflowActions?: WorkflowAction[]
}

// Action 处理器类型
export type ActionHandler = (action: WorkflowAction, message: ChatAssistantMessage) => Promise<void> | void

// Action 类型处理器映射
export interface ActionHandlers {
  workflow: ActionHandler
  query: ActionHandler
  command: ActionHandler
  navigate: ActionHandler
}

// 组件 Props
export interface ChatAssistantProps {
  // 是否显示
  visible?: boolean
  // 初始消息列表
  initialMessages?: ChatMessage[]
}

// 组件 Emits
export interface ChatAssistantEmits {
  close: []
  messagesSent: [message: ChatMessage]
  actionTriggered: [action: WorkflowAction, message: ChatMessage]
}

// 按钮样式映射
export type ButtonType = 'primary' | 'success' | 'warning' | 'danger' | 'info' | 'default'

export interface StyleMapping {
  [key: string]: ButtonType
}
