// 员工管理模块相关的接口请求方法
import { request } from '@/utils/request'

const prefix = '/v1'

/**
 * 查询员工列表
 */
export function getEmployeeList(params: Employee.ListParams) {
  return request<Employee.ListData>({
    url: `${prefix}/employees`,
    method: 'get',
    params,
  })
}

/**
 * 获取员工详情
 */
export function getEmployeeDetail(id: string, orgId: string) {
  return request<Employee.EmployeeInfo>({
    url: `${prefix}/employees/${id}`,
    method: 'get',
    params: { orgId },
  })
}

/**
 * 创建员工
 */
export function createEmployee(data: Employee.CreateRequest) {
  return request<Employee.EmployeeInfo>({
    url: `${prefix}/employees`,
    method: 'post',
    data,
  })
}

/**
 * 更新员工信息
 */
export function updateEmployee(id: string, data: Employee.UpdateRequest) {
  return request<Employee.EmployeeInfo>({
    url: `${prefix}/employees/${id}`,
    method: 'put',
    data,
  })
}

/**
 * 删除员工
 */
export function deleteEmployee(id: string, orgId: string) {
  return request({
    url: `${prefix}/employees/${id}`,
    method: 'delete',
    params: { orgId },
  })
}

/**
 * 批量更新员工状态
 */
export function batchUpdateEmployeeStatus(data: Employee.BatchUpdateStatusRequest) {
  return request({
    url: `${prefix}/employees/batch/status`,
    method: 'post',
    data,
  })
}

/**
 * 导出员工数据
 */
export function exportEmployees(orgId: string) {
  return request({
    url: `${prefix}/employees/export`,
    method: 'get',
    params: { orgId },
    responseType: 'blob',
  })
}

/**
 * 导入员工数据
 */
export function importEmployees(orgId: string, file: File) {
  const formData = new FormData()
  formData.append('file', file)
  formData.append('orgId', orgId)
  return request({
    url: `${prefix}/employees/import`,
    method: 'post',
    data: formData,
    headers: { 'Content-Type': 'multipart/form-data' },
  })
}

/**
 * 简单查询员工列表（仅返回基本信息，不关联分组）
 * 用于选择器等场景，避免复杂查询
 */
export function getSimpleEmployeeList(params: Employee.SimpleListParams) {
  return request<Employee.SimpleListData>({
    url: `${prefix}/employees/simple`,
    method: 'get',
    params,
  })
}
