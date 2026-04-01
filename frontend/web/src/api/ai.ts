// AI 服务 API
import client from './client'
import { getAccessToken } from './client'

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

export interface ShiftCatalogItem {
  code: string
  name: string
  aliases?: string[]
}

/** 批量规则解析 — 请求 */
export interface ParseRulesBatchRequest {
  description: string
  shift_catalog?: ShiftCatalogItem[]
}

/** 批量规则解析 — 单条解析结果 */
export interface ParsedRuleConfig {
  name: string
  type?: string
  rule_type?: string
  category: string
  sub_type: string
  apply_scope?: string
  time_scope?: string
  time_offset_days?: number
  rule_data?: string
  priority?: number
  source_type?: string
  parse_confidence?: number
  version?: string
  config: Record<string, unknown>
  description: string
  subject_shifts?: string[]
  object_shifts?: string[]
  target_shifts?: string[]
  scope_type?: string
  scope_employees?: string[]
  scope_groups?: string[]
}

export interface ParsedRuleDependencyInfo {
  dependent_rule_name: string
  dependent_on_rule_name: string
  dependency_type: 'time' | 'source' | 'resource' | 'order'
  description?: string
}

export interface ParsedRuleConflictInfo {
  rule_name_1: string
  rule_name_2: string
  conflict_type: 'exclusive' | 'resource' | 'time' | 'frequency' | 'duplicate'
  description?: string
}

/** 批量规则解析 — 响应 */
export interface ParseRulesBatchResponse {
  rules: ParsedRuleConfig[]
  parsed_rules?: ParsedRuleConfig[]
  dependencies?: ParsedRuleDependencyInfo[]
  conflicts?: ParsedRuleConflictInfo[]
  reasoning: string
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

/** AI 批量解析规则（自然语言 → 结构化规则列表） */
export function parseRulesBatch(data: ParseRulesBatchRequest) {
  return client.post<ParseRulesBatchResponse>('/ai/parse-rules', data, { timeout: 120000 }).then(r => r.data)
}

/** 获取 AI 配额 */
export function getAIQuota() {
  return client.get<AIQuota>('/ai/quota').then(r => r.data)
}

/** 获取 AI 用量 */
export function getAIUsage() {
  return client.get<AIUsage>('/ai/usage').then(r => r.data)
}

/** SSE 流式批量解析规则 */
export interface ParseRulesBatchStreamCallbacks {
  onReasoning: (text: string) => void
  onChunk: (text: string) => void
  onDone: (result: ParseRulesBatchResponse) => void
  onError: (message: string) => void
}

export function parseRulesBatchStream(
  data: ParseRulesBatchRequest,
  callbacks: ParseRulesBatchStreamCallbacks,
): AbortController {
  const controller = new AbortController()

  const token = getAccessToken()
  fetch('/api/v1/ai/parse-rules-stream', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify(data),
    signal: controller.signal,
  })
    .then(async (resp) => {
      if (!resp.ok) {
        let msg = '规则批量解析失败'
        try {
          const errBody = await resp.json()
          msg = errBody?.error?.message || errBody?.message || msg
        }
        catch {}
        callbacks.onError(msg)
        return
      }

      const reader = resp.body?.getReader()
      if (!reader) {
        callbacks.onError('浏览器不支持流式读取')
        return
      }

      const decoder = new TextDecoder()
      let buffer = ''

      while (true) {
        const { done, value } = await reader.read()
        if (done)
          break

        buffer += decoder.decode(value, { stream: true })

        // 按双换行切分 SSE 事件
        const parts = buffer.split('\n\n')
        buffer = parts.pop() || ''

        for (const part of parts) {
          const lines = part.split('\n')
          let eventType = ''
          let eventData = ''

          for (const line of lines) {
            if (line.startsWith('event: '))
              eventType = line.slice(7)
            else if (line.startsWith('data: '))
              eventData = line.slice(6)
          }

          if (!eventType || !eventData)
            continue

          try {
            const parsed = JSON.parse(eventData)
            if (eventType === 'reasoning') {
              callbacks.onReasoning(parsed.reasoning || '')
            }
            else if (eventType === 'chunk') {
              callbacks.onChunk(parsed.content || '')
            }
            else if (eventType === 'done') {
              callbacks.onDone(parsed as ParseRulesBatchResponse)
            }
            else if (eventType === 'error') {
              callbacks.onError(parsed.message || 'AI 解析失败')
            }
          }
          catch {
            // ignore malformed SSE data
          }
        }
      }
    })
    .catch((err) => {
      if (err.name !== 'AbortError') {
        callbacks.onError(err.message || '网络错误')
      }
    })

  return controller
}
