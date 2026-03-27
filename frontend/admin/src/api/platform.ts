import type { PaginatedResponse } from '@/types/api'
import client from './client'

export interface PlatformEmployee {
  id: string
  org_node_id: string
  name: string
  employee_no?: string
  phone?: string
  email?: string
  position?: string
  category?: string
  status: 'active' | 'inactive'
  hire_date?: string
  app_must_reset_pwd: boolean
  app_default_password?: string
  org_node_name?: string
  org_node_path_display?: string
  org_node_type?: string
  created_at: string
  updated_at: string
}

export interface TransferResult {
  employee_id: string
  from_org_node: { id: string, name: string, path_display: string }
  to_org_node: { id: string, name: string, path_display: string }
  roles_cleaned: number
  groups_removed: number
}

export interface PlatformEmployeePayload {
  org_node_id?: string
  name: string
  employee_no?: string
  phone?: string
  email?: string
  position?: string
  category?: string
  hire_date?: string
  status?: 'active' | 'inactive'
}

export interface PlatformUserRole {
  org_node_id: string
  org_node_name: string
  role_name: string
}

export interface PlatformUser {
  id: string
  username: string
  email: string
  phone?: string
  status: 'active' | 'disabled'
  must_reset_pwd: boolean
  bound_employee_id?: string
  roles: PlatformUserRole[]
}

export interface CreatePlatformUserPayload {
  username: string
  email: string
  phone?: string
  org_node_id: string
  role_name: 'org_admin'
}

export interface CreatePlatformUserResponse {
  user: PlatformUser
  default_password: string
}

export interface PlatformAppRoleSummary {
  org_node_id: string
  org_node_name: string
  app_role: string
  count: number
}

export interface PlatformEmployeeAppRole {
  id: string
  employee_id: string
  org_node_id: string
  org_node_name: string
  app_role: string
  source: 'manual' | 'group' | 'system'
  source_group_id?: string
  source_group_name?: string
  granted_by: string
  granted_at: string
  expires_at?: string
}

export interface PlatformExpiringRole extends PlatformEmployeeAppRole {
  employee_name: string
}

export interface AssignEmployeeAppRolePayload {
  app_role: 'app:schedule_admin' | 'app:scheduler' | 'app:leave_approver'
  org_node_id: string
  expires_at?: string | null
}

export interface BatchAssignEmployeeAppRolePayload extends AssignEmployeeAppRolePayload {
  employee_ids: string[]
}

export interface BatchAssignEmployeeAppRoleResponse {
  created: PlatformEmployeeAppRole[]
  skipped_employee_ids: string[]
}

export function listPlatformEmployees(params?: Record<string, unknown>) {
  return client.get<PaginatedResponse<PlatformEmployee>>('/platform/employees/', { params: { scope: 'tree', ...params } }).then(r => r.data)
}

export function createPlatformEmployee(data: PlatformEmployeePayload) {
  return client.post<PlatformEmployee>('/platform/employees/', data).then(r => r.data)
}

export function updatePlatformEmployee(id: string, data: PlatformEmployeePayload) {
  return client.put<PlatformEmployee>(`/platform/employees/${id}`, data).then(r => r.data)
}

export function deletePlatformEmployee(id: string) {
  return client.delete(`/platform/employees/${id}`)
}

export function resetPlatformEmployeePassword(id: string) {
  return client.put<{ default_password: string, must_reset_pwd: boolean }>(`/platform/employees/${id}/reset-pwd`).then(r => r.data)
}

export function listPlatformUsers(params?: { org_node_id?: string }) {
  return client.get<PlatformUser[]>('/admin/platform-users/', { params }).then(r => r.data)
}

export function createPlatformUser(data: CreatePlatformUserPayload) {
  return client.post<CreatePlatformUserResponse>('/admin/platform-users/', data).then(r => r.data)
}

export function resetPlatformUserPassword(id: string) {
  return client.put<{ default_password: string, must_reset_pwd: boolean }>(`/admin/platform-users/${id}/reset-pwd`).then(r => r.data)
}

export function enablePlatformUser(id: string) {
  return client.put(`/admin/platform-users/${id}/enable`).then(r => r.data)
}

export function disablePlatformUser(id: string) {
  return client.put(`/admin/platform-users/${id}/disable`).then(r => r.data)
}

export function deletePlatformUser(id: string) {
  return client.delete(`/admin/platform-users/${id}`)
}

export function listAppRoleSummary() {
  return client.get<PlatformAppRoleSummary[]>('/platform/app-roles/summary').then(r => r.data)
}

export function listExpiringAppRoles(withinDays = 7) {
  return client.get<PlatformExpiringRole[]>('/platform/app-roles/expiring', { params: { within_days: withinDays } }).then(r => r.data)
}

export function listEmployeeAppRoles(employeeId: string) {
  return client.get<PlatformEmployeeAppRole[]>(`/platform/employees/${employeeId}/app-roles`).then(r => r.data)
}

export function assignEmployeeAppRole(employeeId: string, data: AssignEmployeeAppRolePayload) {
  return client.post<PlatformEmployeeAppRole>(`/platform/employees/${employeeId}/app-roles`, data).then(r => r.data)
}

export function removeEmployeeAppRole(employeeId: string, roleId: string) {
  return client.delete(`/platform/employees/${employeeId}/app-roles/${roleId}`)
}

export function batchAssignEmployeeAppRoles(data: BatchAssignEmployeeAppRolePayload) {
  return client.post<BatchAssignEmployeeAppRoleResponse>('/platform/employees/batch-app-roles', data).then(r => r.data)
}

export function transferEmployee(id: string, data: { target_org_node_id: string, reason?: string }) {
  return client.post<TransferResult>(`/platform/employees/${id}/transfer`, data).then(r => r.data)
}

export function batchTransferEmployees(data: { employee_ids: string[], target_org_node_id: string, reason?: string }) {
  return client.post<TransferResult[]>('/platform/employees/batch-transfer', data).then(r => r.data)
}