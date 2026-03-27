import type { AppRoleGrant, AppRoleName } from '@/types/auth'
import client from './client'

export interface AssignEmployeeAppRoleRequest {
  app_role: AppRoleName
  org_node_id: string
}

export function listEmployeeAppRoles(employeeId: string) {
  return client.get<AppRoleGrant[]>(`/employees/${employeeId}/app-roles`).then(r => r.data)
}

export function assignEmployeeAppRole(employeeId: string, data: AssignEmployeeAppRoleRequest) {
  return client.post<AppRoleGrant>(`/employees/${employeeId}/app-roles`, data).then(r => r.data)
}

export function removeEmployeeAppRole(employeeId: string, roleId: string) {
  return client.delete(`/employees/${employeeId}/app-roles/${roleId}`)
}