// 工作流管理API

import { request } from '@/utils/request'

const prefix = '/v1/workflow'

/**
 * 触发工作流事件
 */
export function triggerWorkflowEvent(data: Workflow.TriggerRequest) {
  return request<Workflow.TriggerResponse>({
    url: `${prefix}/trigger`,
    method: 'post',
    data,
  })
}

/**
 * 查询会话状态
 */
export function getSessionStatus(sessionId: string) {
  return request<Workflow.SessionStatusResponse>({
    url: `${prefix}/session/${sessionId}/status`,
    method: 'get',
  })
}

/**
 * 取消当前工作流
 */
export function cancelWorkflow(sessionId: string, reason?: string) {
  return request<Workflow.TriggerResponse>({
    url: `${prefix}/session/${sessionId}/cancel`,
    method: 'post',
    data: { reason },
  })
}
