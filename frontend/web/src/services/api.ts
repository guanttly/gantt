import type {
  AxiosError,
  AxiosInstance,
  AxiosResponse,
} from 'axios'
import type {
  ChatMessage,
  Session,
  SessionCreateRequest,
  SessionCreateResponse,
} from '@/types/chat'
import type {
  Schedule,
  ScheduleQueryOptions,
  ScheduleSnapshot,
} from '@/types/schedule'
import axios from 'axios'
import { ElMessage } from 'element-plus'

// 创建 axios 实例
function createApiInstance(): AxiosInstance {
  const instance = axios.create({
    baseURL: import.meta.env.VITE_API_BASE_URL || '/api',
    timeout: 30000,
    headers: {
      'Content-Type': 'application/json',
    },
  })

  // 请求拦截器
  instance.interceptors.request.use(
    (config) => {
      // 添加认证 token
      const token = localStorage.getItem('auth_token')
      if (token) {
        config.headers.Authorization = `Bearer ${token}`
      }

      // 添加请求 ID 用于链路追踪
      config.headers['X-Request-ID'] = generateRequestId()

      return config
    },
    (error) => {
      return Promise.reject(error)
    },
  )

  // 响应拦截器
  instance.interceptors.response.use(
    (response: AxiosResponse) => {
      return response
    },
    (error: AxiosError) => {
      handleApiError(error)
      return Promise.reject(error)
    },
  )

  return instance
}

// 错误处理
function handleApiError(error: AxiosError) {
  const { response, request, message } = error

  if (response) {
    // 服务器返回错误响应
    const { status, data } = response
    const errorMessage = (data as any)?.message || `HTTP ${status} 错误`

    switch (status) {
      case 401:
        ElMessage.error('登录已过期，请重新登录')
        // 重定向到登录页
        window.location.href = '/login'
        break
      case 403:
        ElMessage.error('权限不足')
        break
      case 404:
        ElMessage.error('请求的资源不存在')
        break
      case 429:
        ElMessage.error('请求过于频繁，请稍后再试')
        break
      case 500:
        ElMessage.error('服务器内部错误')
        break
      default:
        ElMessage.error(errorMessage)
    }
  }
  else if (request) {
    // 请求发出但没有收到响应
    ElMessage.error('网络连接失败，请检查网络设置')
  }
  else {
    // 请求配置出错
    ElMessage.error(`请求配置错误: ${message}`)
  }
}

// 生成请求 ID
function generateRequestId() {
  return `req_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
}

// API 实例
const api = createApiInstance()

// 会话 API
export const sessionApi = {
  // 获取最近会话
  getRecentSession: async (orgId: string, userId: string): Promise<Session | null> => {
    try {
      const response = await api.get('/scheduling/session/recent', {
        params: { orgId, userId },
      })
      const payload = response.data as any
      if (payload && payload.success) {
        return payload.data
      }
      return null
    }
    catch (error) {
      console.error('获取最近会话失败:', error)
      return null
    }
  },

  // 创建新会话
  createSession: async (request: SessionCreateRequest): Promise<SessionCreateResponse> => {
    const response = await api.post('/scheduling/session/start', request)
    const payload = response.data as any
    // 后端返回通用包裹 { success, data, error }
    if (payload && typeof payload === 'object') {
      if (payload.success && payload.data) {
        const dto = payload.data
        return {
          sessionId: dto.id,
          createdAt: dto.startedAt,
        }
      }
      // 非成功时抛错，交由上层处理
      throw new Error(payload.error || 'failed to create session')
    }
    // 兜底：兼容旧格式（直接返回）
    return payload
  },

  // 获取会话历史消息
  getSessionMessages: async (sessionId: string): Promise<ChatMessage[]> => {
    const response = await api.get(`/scheduling/session/${sessionId}`)
    const payload = response.data as any
    const dto = payload?.data ?? payload
    // 后端返回的是完整的会话DTO（包裹在 data 内），这里仅提取 messages 字段
    return dto?.messages || []
  },

  // 获取会话的排班快照
  getSessionSnapshot: async (sessionId: string): Promise<ScheduleSnapshot> => {
    const response = await api.get(`/scheduling/session/${sessionId}`)
    // 从会话DTO中提取排班相关数据（DTO 可能包裹在 data 内）
    const raw = response.data as any
    const dto = raw?.data ?? raw
    return {
      draft: dto?.draft,
      final: dto?.scheduleResult,
      queryResult: undefined, // 暂时没有对应字段
      viewRange: dto?.bizDateRange
        ? {
            start: dto.bizDateRange.split(' - ')[0],
            end: dto.bizDateRange.split(' - ')[1],
            timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
          }
        : undefined,
    }
  },

  // 获取排班预览数据
  getSchedulePreview: async (sessionId: string): Promise<Schedule> => {
    const response = await api.get(`/scheduling/session/${sessionId}/schedule/preview`)
    const raw = response.data as any
    return raw?.data ?? raw
  },

  // 发送消息（通过REST，虽然主要用WS）
  sendMessage: async (sessionId: string, message: string): Promise<void> => {
    await api.post(`/scheduling/session/${sessionId}/message`, { message })
  },

  // 定稿排班
  finalizeSession: async (sessionId: string): Promise<void> => {
    await api.post(`/scheduling/session/${sessionId}/finalize`)
  },

  // 工作流相关操作
  sendWorkflowCommand: async (sessionId: string, command: string, payload?: any): Promise<void> => {
    await api.post(`/scheduling/session/${sessionId}/workflow/command`, { command, payload })
  },

  collectContext: async (sessionId: string, force = false): Promise<void> => {
    await api.post(`/scheduling/session/${sessionId}/context/collect`, { force })
  },

  validateRules: async (sessionId: string): Promise<{ valid: boolean, errors: string[] }> => {
    const response = await api.get(`/scheduling/session/${sessionId}/rules/validate`)
    const payload = response.data as any
    return payload?.data ?? payload
  },

  confirmRule: async (sessionId: string, ruleKey: string): Promise<void> => {
    await api.post(`/scheduling/session/${sessionId}/rules/confirm/${ruleKey}`)
  },

  confirmAllRules: async (sessionId: string): Promise<void> => {
    await api.post(`/scheduling/session/${sessionId}/rules/confirm-all`)
  },

  // 获取工作流决策日志
  getWorkflowDecisions: async (sessionId: string, options?: {
    limit?: number
    event?: string
    from?: string
    to?: string
    reverse?: boolean
  }): Promise<any[]> => {
    const response = await api.get(`/scheduling/session/${sessionId}/workflow/decisions`, { params: options })
    const payload = response.data as any
    return payload?.data ?? payload
  },

  // 列出对话历史（调用管理服务）
  listConversations: async (orgId: string, userId: string, limit?: number): Promise<ConversationSummary[]> => {
    const response = await api.get('/v1/conversations/schedules', {
      params: { orgId, userId, limit },
    })
    const payload = response.data as any
    // 兼容管理服务的响应格式：{code: 0, message: "success", data: [...]}
    if (payload && payload.code === 0 && Array.isArray(payload.data)) {
      return payload.data
    }
    // 兼容其他格式：{success: true, data: [...]}
    if (payload && payload.success && Array.isArray(payload.data)) {
      return payload.data
    }
    // 兼容直接返回数组的情况
    if (Array.isArray(payload)) {
      return payload
    }
    return []
  },

  // 加载历史对话到 session
  // 流程：1) 从管理服务获取 workflow context，2) 通过 WebSocket 发送给排班 agent 加载
  loadConversation: async (conversationId: string, sessionId: string): Promise<any> => {
    // 第一步：从管理服务获取 workflow context
    const response = await api.post(`/v1/conversations/schedules/${conversationId}/load`, { sessionId })
    const payload = response.data as any
    if (payload && !payload.success) {
      throw new Error(payload.error || 'Failed to load conversation')
    }
    // 返回 workflow context，由前端通过 WebSocket 发送给排班 agent
    return payload?.data || payload
  },
}

// 用户偏好 API
export const userPreferenceApi = {
  // 获取用户工作流版本偏好
  getUserWorkflowVersion: async (userId: string, orgId: string): Promise<'v2' | 'v3' | 'v4'> => {
    try {
      const response = await api.get(`/v1/users/${userId}/preferences/workflow-version`, {
        params: { orgId },
      })
      const payload = response.data as any
      // 管理服务返回格式: { code, message, data: { version: "v2" } }
      const version = payload?.data?.version || payload?.version || 'v2'
      // 验证版本值
      if (version === 'v2' || version === 'v3' || version === 'v4') {
        return version
      }
      return 'v2' // 默认值
    }
    catch (error) {
      console.error('获取用户工作流版本偏好失败:', error)
      return 'v2' // 默认值
    }
  },

  // 设置用户工作流版本偏好
  setUserWorkflowVersion: async (userId: string, orgId: string, version: 'v2' | 'v3' | 'v4'): Promise<void> => {
    try {
      await api.put(`/v1/users/${userId}/preferences/workflow-version`, {
        orgId,
        version,
      })
    }
    catch (error) {
      console.error('设置用户工作流版本偏好失败:', error)
      throw error
    }
  },
}

// 对话历史相关类型
export interface ConversationSummary {
  id: string // 管理服务的内部ID
  conversationId: string // context-server 的 conversation ID（用于加载历史对话）
  title: string
  lastMessageAt: string
  messageCount: number
  orgId: string
  userId: string
  workflowType?: string
  scheduleStartDate?: string
  scheduleEndDate?: string
  scheduleShiftIds?: string[]
  scheduleId?: string
  scheduleStatus?: string
}

// 排班 API
export const scheduleApi = {
  // 查询排班数据
  querySchedule: async (options: ScheduleQueryOptions): Promise<Schedule> => {
    const response = await api.post('/schedule/query', options)
    return response.data
  },

  // 保存排班草稿
  saveDraft: async (schedule: Schedule): Promise<Schedule> => {
    const response = await api.post('/schedule/draft', schedule)
    return response.data
  },

  // 定稿排班
  finalize: async (scheduleId: string): Promise<Schedule> => {
    const response = await api.post(`/schedule/${scheduleId}/finalize`)
    return response.data
  },

  // 获取排班详情
  getSchedule: async (scheduleId: string): Promise<Schedule> => {
    const response = await api.get(`/schedule/${scheduleId}`)
    return response.data
  },

  // 获取排班列表
  getSchedules: async (params?: {
    status?: string[]
    dateRange?: [string, string]
    limit?: number
    offset?: number
  }): Promise<{
    schedules: Schedule[]
    total: number
  }> => {
    const response = await api.get('/schedule', { params })
    return response.data
  },

  // 导出排班数据
  exportSchedule: async (
    scheduleId: string,
    format: 'csv' | 'excel' = 'csv',
  ): Promise<Blob> => {
    const response = await api.get(`/schedule/${scheduleId}/export`, {
      params: { format },
      responseType: 'blob',
    })
    return response.data
  },

  // 批量操作
  batchUpdate: async (operations: Array<{
    type: 'assign' | 'unassign' | 'move'
    assignmentId?: string
    resourceId?: string
    shiftId?: string
    start?: string
    end?: string
  }>): Promise<Schedule> => {
    const response = await api.post('/schedule/batch', { operations })
    return response.data
  },
}

// 资源 API
export const resourceApi = {
  // 获取资源列表
  getResources: async (params?: {
    deptIds?: string[]
    roles?: string[]
    keywords?: string
    limit?: number
    offset?: number
  }): Promise<{
    resources: Array<{
      id: string
      name: string
      deptId: string
      deptName: string
      role: string
      skills: string[]
      capacity: number
    }>
    total: number
  }> => {
    const response = await api.get('/resources', { params })
    return response.data
  },

  // 获取科室列表
  getDepartments: async (): Promise<Array<{
    id: string
    name: string
    parentId?: string
    level: number
  }>> => {
    const response = await api.get('/departments')
    return response.data
  },

  // 获取班次模板
  getShiftTemplates: async (): Promise<Array<{
    id: string
    name: string
    color: string
    startOffset: number
    endOffset: number
    duration: number
    type: string
  }>> => {
    const response = await api.get('/shift-templates')
    return response.data
  },
}

// 工具 API
export const utilApi = {
  // 健康检查
  healthCheck: async (): Promise<{
    status: 'ok' | 'error'
    timestamp: string
    version: string
    services: Record<string, boolean>
  }> => {
    const response = await api.get('/health')
    return response.data
  },

  // 获取系统配置
  getConfig: async (): Promise<{
    wsUrl: string
    features: string[]
    limits: {
      maxSessionDuration: number
      maxMessageLength: number
      maxScheduleSize: number
    }
  }> => {
    const response = await api.get('/config')
    return response.data
  },

  // 上传文件
  uploadFile: async (file: File, type: 'template' | 'data' = 'data'): Promise<{
    fileId: string
    url: string
    originalName: string
    size: number
  }> => {
    const formData = new FormData()
    formData.append('file', file)
    formData.append('type', type)

    const response = await api.post('/upload', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    })
    return response.data
  },
}

// 统计 API
export const statsApi = {
  // 获取排班统计数据
  getScheduleStats: async (params?: {
    dateRange?: [string, string]
    deptIds?: string[]
  }): Promise<{
    totalAssignments: number
    resourceUtilization: number
    conflictRate: number
    coverageRate: number
    departmentStats: Array<{
      deptId: string
      deptName: string
      assignmentCount: number
      conflictCount: number
    }>
    dailyStats: Array<{
      date: string
      assignmentCount: number
      conflictCount: number
    }>
  }> => {
    const response = await api.get('/stats/schedule', { params })
    return response.data
  },

  // 获取对话统计
  getChatStats: async (): Promise<{
    totalSessions: number
    totalMessages: number
    avgSessionDuration: number
    topIntents: Array<{
      intent: string
      count: number
    }>
  }> => {
    const response = await api.get('/stats/chat')
    return response.data
  },
}

// 导出默认 API 实例
export default api

// 导出工具函数
export {
  generateRequestId,
  handleApiError,
}
