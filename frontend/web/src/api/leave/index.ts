// 假期管理模块相关的接口请求方法
import { request } from '@/utils/request'

const prefix = '/v1'

// ==================== 请假管理 ====================

/**
 * 查询假期列表
 */
export function getLeaveList(params: Leave.ListParams) {
  return request<Leave.ListData>({
    url: `${prefix}/leaves`,
    method: 'get',
    params,
  })
}

/**
 * 获取假期详情
 */
export function getLeaveDetail(id: string, orgId: string) {
  return request<Leave.LeaveInfo>({
    url: `${prefix}/leaves/${id}`,
    method: 'get',
    params: { orgId },
  })
}

/**
 * 创建假期申请
 */
export function createLeave(data: Leave.CreateRequest) {
  return request<Leave.LeaveInfo>({
    url: `${prefix}/leaves`,
    method: 'post',
    data,
  })
}

/**
 * 更新假期申请
 */
export function updateLeave(id: string, data: Leave.UpdateRequest) {
  return request<Leave.LeaveInfo>({
    url: `${prefix}/leaves/${id}`,
    method: 'put',
    data,
  })
}

/**
 * 删除假期申请
 */
export function deleteLeave(id: string, orgId: string) {
  return request({
    url: `${prefix}/leaves/${id}`,
    method: 'delete',
    params: { orgId },
  })
}

// ==================== 假期余额管理 ====================

/**
 * 查询假期余额
 */
export function getLeaveBalance(params: Leave.BalanceParams) {
  return request<Leave.BalanceInfo>({
    url: `${prefix}/leave-balance`,
    method: 'get',
    params,
  })
}

/**
 * 初始化假期余额
 */
export function initLeaveBalance(data: Leave.InitBalanceRequest) {
  return request({
    url: `${prefix}/leave-balance/init`,
    method: 'post',
    data,
  })
}

/**
 * 查询员工所有类型假期余额
 */
export function getAllLeaveBalances(orgId: string, employeeId: string) {
  return request<{ code: number, message: string, data: Leave.BalanceInfo[] }>({
    url: `${prefix}/leave-balance/all`,
    method: 'get',
    params: { orgId, employeeId },
  })
}
