// AI 服务 API
import client from './client'

export interface ChatRequest {
  message: string
  context?: Record<string, unknown>
}

export interface ChatResponse {
  reply: string
  actions?: Array<{
    type: string
    label: string
    data: Record<string, unknown>
  }>
}

export interface ParseRuleRequest {
  text: string
}

export interface ParseRuleResponse {
  rules: Array<{
    type: string
    config: Record<string, unknown>
    confidence: number
  }>
}

export interface AIQuota {
  total: number
  used: number
  remaining: number
  reset_at: string
}

export interface AIUsage {
  total_requests: number
  total_tokens: number
  by_date: Array<{
    date: string
    requests: number
    tokens: number
  }>
}

/** AI 对话 */
export function chat(data: ChatRequest) {
  return client.post<ChatResponse>('/ai/chat', data).then(r => r.data)
}

/** AI 解析规则 */
export function parseRule(data: ParseRuleRequest) {
  return client.post<ParseRuleResponse>('/ai/parse-rule', data).then(r => r.data)
}

/** 获取 AI 配额 */
export function getAIQuota() {
  return client.get<AIQuota>('/ai/quota').then(r => r.data)
}

/** 获取 AI 用量 */
export function getAIUsage() {
  return client.get<AIUsage>('/ai/usage').then(r => r.data)
}
