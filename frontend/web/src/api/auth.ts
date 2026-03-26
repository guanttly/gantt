import type { StatusResponse } from '@/types/api'
// 认证相关 API
import type {
  AssignRoleRequest,
  ForceResetPasswordRequest,
  LoginRequest,
  LoginResponse,
  RefreshTokenRequest,
  RegisterRequest,
  ResetPasswordRequest,
  UserInfoResponse,
} from '@/types/auth'
import client from './client'

/** 用户注册 */
export function register(data: RegisterRequest) {
  return client.post<LoginResponse>('/auth/register', data).then(r => r.data)
}

/** 用户登录 */
export function login(data: LoginRequest) {
  return client.post<LoginResponse>('/auth/login', data).then(r => r.data)
}

/** 刷新 Token */
export function refreshToken(data: RefreshTokenRequest) {
  return client.post<LoginResponse>('/auth/refresh', data).then(r => r.data)
}

/** 切换组织节点（需认证） */
export function switchNode(data: { org_node_id: string }) {
  return client.post<LoginResponse>('/auth/switch-node', data).then(r => r.data)
}

/** 重置密码（需认证） */
export function resetPassword(data: ResetPasswordRequest) {
  return client.post<StatusResponse>('/auth/password/reset', data).then(r => r.data)
}

/** 强制重置密码（首次登录，需认证） */
export function forceResetPassword(data: ForceResetPasswordRequest) {
  return client.post<StatusResponse>('/auth/password/force-reset', data).then(r => r.data)
}

/** 获取当前用户信息（需认证） */
export function getMe() {
  return client.get<UserInfoResponse>('/auth/me').then(r => r.data)
}

/** 分配角色（需管理员权限） */
export function assignRole(data: AssignRoleRequest) {
  return client.post('/auth/roles', data).then(r => r.data)
}
