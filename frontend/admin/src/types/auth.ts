// 认证相关类型定义

export interface User {
  id: string
  username: string
  email: string
  phone?: string
  must_reset_pwd: boolean
}

export interface OrgNode {
  node_id: string
  node_name: string
  node_path: string
  role_name: RoleName
}

export interface CurrentNode {
  node_id: string
  node_name: string
  node_path: string
  role_name: RoleName
}

export interface LoginRequest {
  username: string
  password: string
  org_node_id?: string
}

export interface LoginResponse {
  access_token: string
  refresh_token: string
  expires_in: number
  user: User
  current_node: CurrentNode
  available_nodes: OrgNode[]
  must_reset_pwd: boolean
}

export interface UserInfoResponse {
  user: User
  current_node: CurrentNode
  available_nodes: OrgNode[]
}

export enum RoleName {
  Employee = 'employee',
  Scheduler = 'scheduler',
  DeptAdmin = 'dept_admin',
  OrgAdmin = 'org_admin',
  PlatformAdmin = 'platform_admin',
}

export const ROLE_HIERARCHY = [
  RoleName.Employee,
  RoleName.Scheduler,
  RoleName.DeptAdmin,
  RoleName.OrgAdmin,
  RoleName.PlatformAdmin,
] as const
