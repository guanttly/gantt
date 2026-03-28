import type { StatusResponse } from '@/types/api'
// 认证相关 API
import type {
  AppPermissionsResponse,
  AppRolesResponse,
  ForceResetPasswordRequest,
  LoginRequest,
  LoginResponse,
  RefreshTokenRequest,
  UserInfoResponse,
} from '@/types/auth'
import client from './client'

type AppAuthEmployeeInfo = {
  id: string
  name: string
  employee_no?: string
  phone?: string
  email?: string
  must_reset_pwd: boolean
  scheduling_role?: string
}

type AppAuthCurrentNode = {
  node_id: string
  node_name: string
  node_path: string
}

type AppAuthLoginResponse = {
  access_token: string
  refresh_token: string
  expires_in: number
  employee: AppAuthEmployeeInfo
  current_node: AppAuthCurrentNode
  must_reset_pwd: boolean
}

type AppAuthMeResponse = {
  employee: AppAuthEmployeeInfo
  current_node: AppAuthCurrentNode
}

function mapEmployeeToUser(employee: AppAuthEmployeeInfo) {
  return {
    id: employee.id,
    username: employee.name,
    email: employee.email,
    phone: employee.phone,
    must_reset_pwd: employee.must_reset_pwd,
  }
}

function mapCurrentNode(currentNode: AppAuthCurrentNode, employee: AppAuthEmployeeInfo) {
  return {
    ...currentNode,
    role_name: employee.scheduling_role || 'employee',
  }
}

function mapLoginResponse(payload: AppAuthLoginResponse): LoginResponse {
  const currentNode = mapCurrentNode(payload.current_node, payload.employee)
  return {
    access_token: payload.access_token,
    refresh_token: payload.refresh_token,
    expires_in: payload.expires_in,
    user: mapEmployeeToUser(payload.employee),
    current_node: currentNode,
    available_nodes: [currentNode],
    must_reset_pwd: payload.must_reset_pwd,
  }
}

function mapUserInfoResponse(payload: AppAuthMeResponse): UserInfoResponse {
  const currentNode = mapCurrentNode(payload.current_node, payload.employee)
  return {
    user: mapEmployeeToUser(payload.employee),
    current_node: currentNode,
    available_nodes: [currentNode],
  }
}

/** 用户登录 */
export function login(data: LoginRequest) {
  return client.post<AppAuthLoginResponse>('/app/scheduling/auth/login', data).then(r => mapLoginResponse(r.data))
}

/** 刷新 Token */
export function refreshToken(data: RefreshTokenRequest) {
  return client.post<AppAuthLoginResponse>('/app/scheduling/auth/refresh', data).then(r => mapLoginResponse(r.data))
}

/** 强制重置密码（首次登录，需认证） */
export function forceResetPassword(data: ForceResetPasswordRequest) {
  return client.post<StatusResponse>('/app/scheduling/auth/password/force-reset', data).then(r => r.data)
}

/** 获取当前用户信息（需认证） */
export function getMe() {
  return client.get<AppAuthMeResponse>('/app/scheduling/auth/me').then(r => mapUserInfoResponse(r.data))
}

/** 获取当前登录用户的应用角色 */
export function getMyAppRoles() {
  return client.get<AppRolesResponse>('/app/scheduling/auth/my-roles').then(r => r.data)
}

/** 获取当前登录用户的应用权限 */
export function getMyAppPermissions() {
  return client.get<AppPermissionsResponse>('/app/scheduling/auth/permissions').then(r => r.data)
}
