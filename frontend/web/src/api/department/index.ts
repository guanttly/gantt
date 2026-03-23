/**
 * 部门管理 - API 接口
 */

import type {
  CreateDepartmentRequest,
  DepartmentInfo,
  DepartmentListParams,
  DepartmentListResult,
  DepartmentTree,
  UpdateDepartmentRequest,
} from './model'
import { request } from '@/utils/request'

/**
 * 获取部门列表
 */
export function getDepartmentList(params: DepartmentListParams) {
  return request<DepartmentListResult>({
    url: '/v1/departments',
    method: 'get',
    params,
  })
}

/**
 * 获取部门树
 */
export function getDepartmentTree(orgId: string) {
  return request<DepartmentTree[]>({
    url: '/v1/departments/tree',
    method: 'get',
    params: { orgId },
  })
}

/**
 * 获取启用的部门列表
 */
export function getActiveDepartments(orgId: string) {
  return request<DepartmentInfo[]>({
    url: '/v1/departments/active',
    method: 'get',
    params: { orgId },
  })
}

/**
 * 获取部门详情
 */
export function getDepartmentDetail(id: string, orgId: string) {
  return request<DepartmentInfo>({
    url: `/v1/departments/${id}`,
    method: 'get',
    params: { orgId },
  })
}

/**
 * 创建部门
 */
export function createDepartment(data: CreateDepartmentRequest) {
  return request<DepartmentInfo>({
    url: '/v1/departments',
    method: 'post',
    data,
  })
}

/**
 * 更新部门信息
 */
export function updateDepartment(id: string, data: UpdateDepartmentRequest) {
  return request<DepartmentInfo>({
    url: `/v1/departments/${id}`,
    method: 'put',
    data,
  })
}

/**
 * 删除部门
 */
export function deleteDepartment(id: string, orgId: string) {
  return request<void>({
    url: `/v1/departments/${id}`,
    method: 'delete',
    params: { orgId },
  })
}
