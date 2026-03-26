// 认证相关类型定义

/** 用户信息 */
export interface User {
  id: string
  username: string
  email: string
  phone?: string
  must_reset_pwd: boolean
}

/** 组织节点 */
export interface OrgNode {
  node_id: string
  node_name: string
  node_path: string
  role_name: string
}

/** 当前节点信息（登录/切换后返回） */
export interface CurrentNode {
  node_id: string
  node_name: string
  node_path: string
  role_name: string
}

export interface AppRoleGrant {
  id: string
  employee_id: string
  org_node_id: string
  org_node_name: string
  app_role: string
  source: string
  source_group_id?: string
  source_group_name?: string
  granted_by: string
  granted_at: string
  expires_at?: string
}

export interface AppRolesResponse {
  employee_id: string
  org_node_id: string
  org_node_name: string
  app_roles: AppRoleGrant[]
}

export interface AppPermissionsResponse {
  employee_id: string
  org_node_id: string
  org_node_name: string
  permissions: string[]
}

/** 登录请求 */
export interface LoginRequest {
  username: string
  password: string
  org_node_id?: string
}

/** 注册请求 */
export interface RegisterRequest {
  username: string
  email: string
  phone?: string
  password: string
  org_node_id: string
  role_name: string
}

/** 登录/注册/刷新/切换节点 响应 */
export interface LoginResponse {
  access_token: string
  refresh_token: string
  expires_in: number
  user: User
  current_node: CurrentNode
  available_nodes: OrgNode[]
  must_reset_pwd: boolean
}

/** 刷新 Token 请求 */
export interface RefreshTokenRequest {
  refresh_token: string
}

/** 切换节点请求 */
export interface SwitchNodeRequest {
  org_node_id: string
}

/** 重置密码请求 */
export interface ResetPasswordRequest {
  old_password: string
  new_password: string
}

/** 强制重置密码请求（首次登录） */
export interface ForceResetPasswordRequest {
  new_password: string
}

/** 分配角色请求 */
export interface AssignRoleRequest {
  user_id: string
  org_node_id: string
  role_name: string
}

/** 用户信息响应 (GET /auth/me) */
export interface UserInfoResponse {
  user: User
  current_node: CurrentNode
  available_nodes: OrgNode[]
}

export type AppPermission =
  | 'schedule:create'
  | 'schedule:execute'
  | 'schedule:adjust'
  | 'schedule:publish'
  | 'schedule:view:all'
  | 'schedule:view:node'
  | 'schedule:view:self'
  | 'leave:approve'
  | 'leave:view:node'
  | 'leave:create:self'
  | 'preference:edit:self'

/** 角色层级（从低到高） */
export const ROLE_HIERARCHY = [
  'employee',
  'scheduler',
  'dept_admin',
  'org_admin',
  'platform_admin',
] as const

export type RoleName = (typeof ROLE_HIERARCHY)[number]
