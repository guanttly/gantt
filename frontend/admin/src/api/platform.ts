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

export interface PlatformGroup {
  id: string
  org_node_id: string
  name: string
  description?: string
  created_at: string
  updated_at: string
}

export interface PlatformGroupMember {
  id: string
  group_id: string
  employee_id: string
  created_at: string
}

export interface PlatformShift {
  id: string
  org_node_id: string
  name: string
  code: string
  start_time: string
  end_time: string
  duration: number
  is_cross_day: boolean
  color: string
  priority: number
  status: 'active' | 'disabled'
  created_at: string
  updated_at: string
}

export interface PlatformShiftPayload {
  name: string
  code: string
  start_time: string
  end_time: string
  duration: number
  is_cross_day: boolean
  color: string
  priority: number
}

export interface PlatformRule {
  id: string
  org_node_id: string
  name: string
  category: string
  sub_type: string
  config: Record<string, unknown>
  priority: number
  is_enabled: boolean
  disabled: boolean
  disabled_reason?: string
  override_rule_id?: string
  description?: string
  source_node: string
  is_inherited: boolean
  is_overridable: boolean
  created_at: string
  updated_at: string
}

export interface PlatformRulePayload {
  name: string
  category: string
  sub_type: string
  config: Record<string, unknown>
  priority: number
  is_enabled?: boolean
  description?: string
  override_rule_id?: string
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
  role_name: 'org_admin' | 'dept_admin'
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

export interface GroupDefaultAppRole {
  id: string
  group_id: string
  org_node_id: string
  org_node_name: string
  app_role: string
  created_by: string
  created_at: string
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

export function listPlatformGroups() {
  return client.get<PlatformGroup[]>('/platform/groups/').then(r => r.data)
}

export function createPlatformGroup(data: { name: string, description?: string }) {
  return client.post<PlatformGroup>('/platform/groups/', data).then(r => r.data)
}

export function updatePlatformGroup(id: string, data: { name?: string, description?: string }) {
  return client.put<PlatformGroup>(`/platform/groups/${id}`, data).then(r => r.data)
}

export function deletePlatformGroup(id: string) {
  return client.delete(`/platform/groups/${id}`)
}

export function listPlatformGroupMembers(id: string) {
  return client.get<PlatformGroupMember[]>(`/platform/groups/${id}/members`).then(r => r.data)
}

export function addPlatformGroupMember(groupId: string, employeeId: string) {
  return client.post(`/platform/groups/${groupId}/members`, { employee_id: employeeId }).then(r => r.data)
}

export function removePlatformGroupMember(groupId: string, employeeId: string) {
  return client.delete(`/platform/groups/${groupId}/members/${employeeId}`)
}

export function listGroupDefaultAppRoles(groupId: string) {
  return client.get<GroupDefaultAppRole[]>(`/platform/groups/${groupId}/default-app-roles`).then(r => r.data)
}

export function assignGroupDefaultAppRole(groupId: string, data: { app_role: string, org_node_id: string }) {
  return client.post<GroupDefaultAppRole>(`/platform/groups/${groupId}/default-app-roles`, data).then(r => r.data)
}

export function removeGroupDefaultAppRole(groupId: string, roleId: string) {
  return client.delete(`/platform/groups/${groupId}/default-app-roles/${roleId}`)
}

export function listPlatformShifts() {
  return client.get<PlatformShift[]>('/platform/shifts/').then(r => r.data)
}

export function createPlatformShift(data: PlatformShiftPayload) {
  return client.post<PlatformShift>('/platform/shifts/', data).then(r => r.data)
}

export function updatePlatformShift(id: string, data: Partial<PlatformShiftPayload>) {
  return client.put<PlatformShift>(`/platform/shifts/${id}`, data).then(r => r.data)
}

export function togglePlatformShift(id: string) {
  return client.put<PlatformShift>(`/platform/shifts/${id}/toggle`).then(r => r.data)
}

export function deletePlatformShift(id: string) {
  return client.delete(`/platform/shifts/${id}`)
}

export function listPlatformRules() {
  return client.get<PlatformRule[]>('/platform/rules/').then(r => r.data)
}

export function listPlatformEffectiveRules() {
  return client.get<{ rules: PlatformRule[] }>('/platform/rules/effective').then(r => r.data)
}

export function createPlatformRule(data: PlatformRulePayload) {
  return client.post<PlatformRule>('/platform/rules/', data).then(r => r.data)
}

export function updatePlatformRule(id: string, data: Partial<PlatformRulePayload>) {
  return client.put<PlatformRule>(`/platform/rules/${id}`, data).then(r => r.data)
}

export function disablePlatformRule(id: string, reason: string) {
  return client.put<PlatformRule>(`/platform/rules/${id}/disable`, { reason }).then(r => r.data)
}

export function restorePlatformRule(id: string) {
  return client.put(`/platform/rules/${id}/restore`).then(r => r.data)
}

export function deletePlatformRule(id: string) {
  return client.delete(`/platform/rules/${id}`)
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

export function disablePlatformUser(id: string) {
  return client.put(`/admin/platform-users/${id}/disable`).then(r => r.data)
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