// 员工管理 API
import type { ListParams, PaginatedResponse } from '@/types/api'
import type { BatchUpdateStatusRequest, CreateEmployeeRequest, Employee, EmployeeStatus, UpdateEmployeeRequest } from '@/types/employee'
import client from './client'

/** 员工列表 */
export function listEmployees(params?: ListParams & { include_groups?: boolean }) {
  const query = {
    ...params,
    size: params?.page_size,
  }
  return client.get<PaginatedResponse<Employee>>('/app/scheduling/ref/employees', { params: query }).then(r => r.data)
}

/** 创建员工 */
export function createEmployee(data: CreateEmployeeRequest) {
  return client.post<Employee>('/employees/', data).then(r => r.data)
}

/** 获取员工详情 */
export function getEmployee(id: string) {
  return client.get<Employee>(`/employees/${id}`).then(r => r.data)
}

/** 更新员工 */
export function updateEmployee(id: string, data: UpdateEmployeeRequest) {
  return client.put<Employee>(`/employees/${id}`, data).then(r => r.data)
}

/** 删除员工 */
export function deleteEmployee(id: string) {
  return client.delete(`/employees/${id}`)
}

/** 批量更新员工状态 */
export function batchUpdateEmployeeStatus(data: BatchUpdateStatusRequest) {
  return client.post('/employees/batch-status', data)
}

/** 更新员工状态 */
export function updateEmployeeStatus(id: string, status: EmployeeStatus) {
  return client.patch(`/employees/${id}/status`, { status })
}
