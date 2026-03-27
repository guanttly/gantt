// 统一 API 客户端 — Axios 实例
import type { InternalAxiosRequestConfig } from 'axios'
import axios from 'axios'
import { createDiscreteApi } from 'naive-ui'

const { message } = createDiscreteApi(['message'])

type WrappedSuccessResponse<T> = {
  data: T
}

const client = axios.create({
  baseURL: '/api/v1',
  timeout: 15000,
})

function unwrapSuccessResponse<T>(payload: T | WrappedSuccessResponse<T>): T {
  if (
    payload
    && typeof payload === 'object'
    && 'data' in payload
    && Object.keys(payload as Record<string, unknown>).length === 1
  ) {
    return (payload as WrappedSuccessResponse<T>).data
  }
  return payload as T
}

function isAuthEntryRequest(url?: string): boolean {
  if (!url) {
    return false
  }
  return url.includes('/admin/auth/login') || url.includes('/auth/refresh')
}

function extractErrorMessage(payload: unknown): string | null {
  if (!payload || typeof payload !== 'object') {
    return null
  }

  if ('message' in payload && typeof payload.message === 'string') {
    return payload.message
  }

  if (
    'error' in payload
    && payload.error
    && typeof payload.error === 'object'
    && 'message' in payload.error
    && typeof payload.error.message === 'string'
  ) {
    return payload.error.message
  }

  return null
}

// Token 存取
const TOKEN_KEY = 'admin_access_token'
const REFRESH_TOKEN_KEY = 'admin_refresh_token'

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

// 请求拦截
client.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  const token = getAccessToken()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// 响应拦截
let isRefreshing = false
let pendingRequests: Array<(token: string) => void> = []

client.interceptors.response.use(
  (response) => {
    response.data = unwrapSuccessResponse(response.data)
    return response
  },
  async (error) => {
    const originalRequest = error.config as InternalAxiosRequestConfig & { _retry?: boolean }

    if (isAuthEntryRequest(originalRequest?.url)) {
      return Promise.reject(error)
    }

    if (error.response?.status !== 401 || originalRequest._retry) {
      const msg = extractErrorMessage(error.response?.data) || '请求失败'
      message.error(msg)
      return Promise.reject(error)
    }

    if (isRefreshing) {
      return new Promise((resolve) => {
        pendingRequests.push((token: string) => {
          if (originalRequest.headers) {
            originalRequest.headers.Authorization = `Bearer ${token}`
          }
          resolve(client(originalRequest))
        })
      })
    }

    originalRequest._retry = true
    isRefreshing = true

    try {
      const refreshToken = getRefreshToken()
      if (!refreshToken) throw new Error('no_refresh_token')

      const res = await axios.post('/api/v1/auth/refresh', { refresh_token: refreshToken })
      const refreshPayload = unwrapSuccessResponse(res.data)
      const newToken: string = refreshPayload.access_token
      const newRefresh: string = refreshPayload.refresh_token

      setAccessToken(newToken)
      if (newRefresh) setRefreshToken(newRefresh)

      pendingRequests.forEach(cb => cb(newToken))
      pendingRequests = []

      if (originalRequest.headers) {
        originalRequest.headers.Authorization = `Bearer ${newToken}`
      }
      return client(originalRequest)
    }
    catch {
      clearTokens()
      window.location.href = '/admin/login'
      return Promise.reject(error)
    }
    finally {
      isRefreshing = false
    }
  },
)

export default client
