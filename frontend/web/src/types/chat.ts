// 聊天相关类型定义
export interface ChatMessage {
  id: string
  role: 'user' | 'assistant' | 'system'
  content: string
  createdAt: string
  payload?: MessagePayload
  meta?: MessageMeta
  isLoading?: boolean
  actions?: WorkflowAction[]
}

// 操作类型枚举（与后端同步）
export type WorkflowActionType = 'workflow' | 'query' | 'command' | 'navigate'

// 操作样式枚举（与后端同步）
export type WorkflowActionStyle = 'primary' | 'secondary' | 'success' | 'danger' | 'warning' | 'info' | 'link'

// 字段类型枚举（与后端同步）
export type WorkflowActionFieldType = 'text' | 'number' | 'date' | 'datetime' | 'select' | 'checkbox' | 'textarea' | 'multi-select' | 'checked-table' | 'daily-grid' | 'repeat' | 'table-form'

// 字段选项
export interface FieldOption {
  label: string
  value: any
  description?: string // 描述信息
  disabled?: boolean // 是否禁用
  icon?: string // 图标
  extra?: Record<string, any> // 额外数据
}

// 字段验证规则
export interface FieldValidation {
  min?: number
  max?: number
  pattern?: string
  message?: string
}

// 操作字段定义（用于动态表单生成）
export interface WorkflowActionField {
  name: string // 字段名称（用于 payload key）
  label: string // 显示标签
  type: WorkflowActionFieldType // 字段类型
  required?: boolean // 是否必填
  placeholder?: string // 占位符
  defaultValue?: any // 默认值
  options?: FieldOption[] // 选项（用于 select/checkbox）
  validation?: FieldValidation // 验证规则
  extra?: Record<string, any> // 额外数据（用于扩展配置，如多选日期）
}

// 工作流操作（与后端 WorkflowAction 结构同步）
export interface WorkflowAction {
  id: string
  type: WorkflowActionType // 操作类型
  label: string // 显示文本
  event: string // 触发事件
  style?: WorkflowActionStyle // 按钮样式
  payload?: Record<string, any> // 携带数据
  fields?: WorkflowActionField[] // 字段定义（用于动态表单）
}

export interface MessagePayload {
  // 扩展一个通用的状态类消息，便于在仅有状态推送时也能在对话框中显示
  type: 'schedule' | 'progress' | 'error' | 'card' | 'action' | 'status'
  data: any
}

export interface MessageMeta {
  timeRange?: {
    start: string
    end: string
    timezone: string
  }
  filters?: {
    deptIds?: string[]
    resourceIds?: string[]
    keywords?: string
  }
}

export interface ChatSession {
  id: string
  title?: string
  createdAt: string
  updatedAt: string
  meta?: {
    locale?: string
    creatorId?: string
  }
}

export interface SessionCreateRequest {
  orgId: string
  userId: string
  agentType: string
  title?: string
  locale?: string
}

export interface SessionCreateResponse {
  sessionId: string
  createdAt: string
}

/**
 * Session 完整数据结构
 */
export interface Session {
  id: string
  orgId: string
  userId: string
  agentType: string
  state: string
  messages: ChatMessage[]
  workflowMeta?: {
    workflow: string
    phase: string
    actions?: WorkflowAction[]
    [key: string]: any
  }
  createdAt: string
  updatedAt: string
  [key: string]: any
}
