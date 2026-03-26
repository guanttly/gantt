// 统一 API 客户端 — Axios 实例（拦截器、Token 自动刷新）
import type { AxiosRequestConfig, InternalAxiosRequestConfig } from 'axios'
import axios from 'axios'
import { ElMessage } from 'element-plus'

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
let pendingRequests: Array<(token: string) => void> = []

client.interceptors.response.use(
  response => response,
  async (error) => {
    const originalRequest = error.config as AxiosRequestConfig & { _retry?: boolean }

    // 非 401 或已重试 → 直接抛出
    if (error.response?.status !== 401 || originalRequest._retry) {
      const msg = error.response?.data?.message || '请求失败'
      ElMessage.error(msg)
      return Promise.reject(error)
    }

    // 正在刷新中 → 排队等候
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

    // 发起刷新
    originalRequest._retry = true
    isRefreshing = true

    try {
      const refreshToken = getRefreshToken()
      if (!refreshToken)
        throw new Error('no_refresh_token')

      const res = await axios.post('/api/v1/auth/refresh', { refresh_token: refreshToken })
      const newToken: string = res.data.access_token
      const newRefresh: string = res.data.refresh_token

      setAccessToken(newToken)
      if (newRefresh)
        setRefreshToken(newRefresh)

      // 重放排队请求
      pendingRequests.forEach(cb => cb(newToken))
      pendingRequests = []

      // 重放当前请求
      if (originalRequest.headers) {
        originalRequest.headers.Authorization = `Bearer ${newToken}`
      }
      return client(originalRequest)
    }
    catch {
      // 刷新失败 → 清空 Token → 跳转登录
      clearTokens()
      pendingRequests = []
      window.location.href = '/login'
      return Promise.reject(error)
    }
    finally {
      isRefreshing = false
    }
  },
)

export default client
