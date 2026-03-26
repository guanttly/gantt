import type { ChatMessage, WorkflowAction } from '@/types/chat'

// 组件内部消息类型扩展
export interface ChatAssistantMessage extends ChatMessage {
  isRead?: boolean
  isSending?: boolean
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
  visible?: boolean
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
