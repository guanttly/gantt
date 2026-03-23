import { request } from '@/utils/request'

const prefix = '/v1/scan-types'

/** 获取检查类型列表 */
export function getScanTypeList(params: ScanType.ListParams) {
  return request<ScanType.ListResult>({
    url: prefix,
    method: 'get',
    params,
  })
}

/** 获取检查类型详情 */
export function getScanType(id: string, orgId: string) {
  return request<ScanType.Info>({
    url: `${prefix}/${id}`,
    method: 'get',
    params: { orgId },
  })
}

/** 创建检查类型 */
export function createScanType(data: ScanType.CreateRequest) {
  return request<ScanType.Info>({
    url: prefix,
    method: 'post',
    data,
  })
}

/** 更新检查类型 */
export function updateScanType(id: string, data: ScanType.UpdateRequest) {
  const { orgId, ...rest } = data
  return request<ScanType.Info>({
    url: `${prefix}/${id}`,
    method: 'put',
    params: { orgId },
    data: rest,
  })
}

/** 删除检查类型 */
export function deleteScanType(id: string, orgId: string) {
  return request({
    url: `${prefix}/${id}`,
    method: 'delete',
    params: { orgId },
  })
}

/** 切换检查类型状态 */
export function toggleScanTypeStatus(id: string, orgId: string, isActive: boolean) {
  return request({
    url: `${prefix}/${id}/status`,
    method: 'patch',
    params: { orgId },
    data: { isActive },
  })
}

/** 获取所有启用的检查类型 */
export function getActiveScanTypes(orgId: string) {
  return request<ScanType.Info[]>({
    url: `${prefix}/active`,
    method: 'get',
    params: { orgId },
  })
}
