// 统一 API 客户端 — Axios 实例（拦截器、Token 自动刷新）
import type { AxiosRequestConfig, InternalAxiosRequestConfig } from 'axios'
import axios from 'axios'
import { ElMessage } from 'element-plus'

type PendingRequest = {
  resolve: (value: unknown) => void
  reject: (reason?: unknown) => void
  request: AxiosRequestConfig & { _retry?: boolean }
}

type PageEnvelope = {
  data: unknown
  total: number
  page: number
  size: number
}

const AUTH_EXPIRED_MESSAGE = '登录状态已失效，请刷新页面后重试；如仍无法继续，请重新登录。'

function isPlainObject(value: unknown): value is Record<string, any> {
  return !!value && typeof value === 'object' && !Array.isArray(value)
}

function isPageEnvelope(value: unknown): value is PageEnvelope {
  return isPlainObject(value)
    && 'data' in value
    && 'total' in value
    && 'page' in value
    && 'size' in value
}

function isSuccessEnvelope(value: unknown): value is { data: unknown } {
  return isPlainObject(value) && Object.keys(value).length === 1 && 'data' in value
}

function normalizeResponseBody<T>(value: T): T {
  if (!isPlainObject(value)) {
    return value
  }

  if (isPageEnvelope(value)) {
    const pageSize = Number(value.size) || 0
    const total = Number(value.total) || 0
    return {
      items: Array.isArray(value.data) ? value.data : [],
      total,
      page: Number(value.page) || 1,
      page_size: pageSize,
      total_pages: pageSize > 0 ? Math.ceil(total / pageSize) : 0,
    } as T
  }

  if (isSuccessEnvelope(value)) {
    return value.data as T
  }

  return value
}

function normalizeErrorBody(value: unknown) {
  if (!isPlainObject(value)) {
    return value
  }

  if (isPlainObject(value.error)) {
    return {
      code: value.error.code,
      message: value.error.message || '请求失败',
    }
  }

  return value
}

function withFriendlyAuthMessage(value: unknown, status?: number) {
  if (status !== 401) {
    return value
  }

  if (isPlainObject(value)) {
    return {
      ...value,
      message: AUTH_EXPIRED_MESSAGE,
    }
  }

  return {
    message: AUTH_EXPIRED_MESSAGE,
  }
}

function getErrorMessage(value: unknown): string {
  if (isPlainObject(value) && typeof value.message === 'string' && value.message.trim()) {
    return value.message
  }
  return '请求失败'
}

function isAuthEntryRequest(url?: string): boolean {
  if (!url) {
    return false
  }
  return url.includes('/app/scheduling/auth/login')
    || url.includes('/app/scheduling/auth/refresh')
    || url.includes('/app/scheduling/auth/password/reset')
    || url.includes('/app/scheduling/auth/password/force-reset')
}

// ============ Axios 实例 ============

const client = axios.create({
  baseURL: '/api/v1',
  timeout: 15000,
})

// ============ Token 存取 ============

const TOKEN_KEY = 'access_token'
const REFRESH_TOKEN_KEY = 'refresh_token'

export function getAccessToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function setAccessToken(token: string) {
  localStorage.setItem(TOKEN_KEY, token)
}

export function getRefreshToken(): string | null {
  return localStorage.getItem(REFRESH_TOKEN_KEY)
}

export function setRefreshToken(token: string) {
  localStorage.setItem(REFRESH_TOKEN_KEY, token)
}

export function clearTokens() {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(REFRESH_TOKEN_KEY)
}

// ============ 请求拦截：自动附加 Token ============

client.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  const token = getAccessToken()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// ============ 响应拦截：401 自动刷新 Token ============

let isRefreshing = false
let pendingRequests: PendingRequest[] = []

client.interceptors.response.use(
  (response) => {
    if (response.config.responseType === 'blob' || response.config.responseType === 'arraybuffer') {
      return response
    }

    response.data = normalizeResponseBody(response.data)
    return response
  },
  async (error) => {
    const originalRequest = error.config as AxiosRequestConfig & { _retry?: boolean }

    if (error.response) {
      error.response.data = withFriendlyAuthMessage(
        normalizeErrorBody(error.response.data),
        error.response.status,
      )
    }

    if (isAuthEntryRequest(originalRequest?.url)) {
      return Promise.reject(error)
    }

    // 非 401 或已重试 → 直接抛出
    if (error.response?.status !== 401 || originalRequest._retry) {
      ElMessage.error(getErrorMessage(error.response?.data))
      return Promise.reject(error)
    }

    // 正在刷新中 → 排队等候
    if (isRefreshing) {
      return new Promise((resolve, reject) => {
        pendingRequests.push({
          resolve,
          reject,
          request: originalRequest,
        })
      })
    }

    // 发起刷新
    originalRequest._retry = true
    isRefreshing = true

    try {
      const refreshToken = getRefreshToken()
      if (!refreshToken)
        throw new Error('no_refresh_token')

      const res = await axios.post('/api/v1/app/scheduling/auth/refresh', { refresh_token: refreshToken })
      const payload = normalizeResponseBody(res.data) as { access_token: string, refresh_token?: string }
      const newToken = payload.access_token
      const newRefresh = payload.refresh_token

      setAccessToken(newToken)
      if (newRefresh)
        setRefreshToken(newRefresh)

      // 重放排队请求
      pendingRequests.forEach(({ resolve, request }) => {
        if (request.headers) {
          request.headers.Authorization = `Bearer ${newToken}`
        }
        resolve(client(request))
      })
      pendingRequests = []

      // 重放当前请求
      if (originalRequest.headers) {
        originalRequest.headers.Authorization = `Bearer ${newToken}`
      }
      return client(originalRequest)
    }
    catch (refreshError) {
      // 刷新失败 → 清空 Token → 跳转登录
      pendingRequests.forEach(({ reject }) => reject(refreshError))
      clearTokens()
      pendingRequests = []
      window.location.href = '/login'
      return Promise.reject(refreshError)
    }
    finally {
      isRefreshing = false
    }
  },
)

export default client
