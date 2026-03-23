// 工作流相关类型定义

declare namespace Workflow {
  /** 工作流触发请求 */
  interface TriggerRequest {
    sessionId: string
    event: string
    payload?: Record<string, any>
  }

  /** 工作流触发响应 */
  interface TriggerResponse {
    success: boolean
    message?: string
    workflowMeta?: WorkflowMeta
  }

  /** 工作流元数据 */
  interface WorkflowMeta {
    workflow: string
    phase: string
    description?: string
    actions?: WorkflowAction[]
    [key: string]: any
  }

  /** 工作流操作 */
  interface WorkflowAction {
    id: string
    type: 'workflow' | 'query' | 'command' | 'navigate'
    label: string
    event: string
    style?: 'primary' | 'secondary' | 'success' | 'danger' | 'warning' | 'info' | 'link'
    payload?: Record<string, any>
  }

  /** 会话状态查询响应 */
  interface SessionStatusResponse {
    sessionId: string
    state: string
    workflowMeta?: WorkflowMeta
    createdAt: string
    updatedAt: string
  }
}
