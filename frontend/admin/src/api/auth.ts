// 认证 API
import type { LoginRequest, LoginResponse, UserInfoResponse } from '@/types/auth'
import client from './client'

export function login(data: LoginRequest) {
  return client.post<LoginResponse>('/admin/auth/login', data).then(r => r.data)
}

export function refreshToken(data: { refresh_token: string }) {
  return client.post<LoginResponse>('/auth/refresh', data).then(r => r.data)
}

export function getMe() {
  return client.get<UserInfoResponse>('/auth/me').then(r => r.data)
}

export function forceResetPassword(data: { new_password: string }) {
  return client.post('/auth/password/force-reset', data).then(r => r.data)
}
