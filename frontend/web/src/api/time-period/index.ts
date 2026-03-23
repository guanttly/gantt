import { request } from '@/utils/request'

const prefix = '/v1/time-periods'

/** 查询时间段列表 */
export function getTimePeriodList(params: TimePeriod.ListParams) {
  return request<TimePeriod.ListResult>({
    url: prefix,
    method: 'get',
    params,
  })
}

/** 获取启用的时间段 */
export function getActiveTimePeriods(orgId: string) {
  return request<TimePeriod.Info[]>({
    url: `${prefix}/active`,
    method: 'get',
    params: { orgId },
  })
}

/** 获取时间段详情 */
export function getTimePeriod(id: string, orgId: string) {
  return request<TimePeriod.Info>({
    url: `${prefix}/${id}`,
    method: 'get',
    params: { orgId },
  })
}

/** 创建时间段 */
export function createTimePeriod(data: TimePeriod.CreateRequest) {
  return request<TimePeriod.Info>({
    url: prefix,
    method: 'post',
    data,
  })
}

/** 更新时间段 */
export function updateTimePeriod(id: string, data: TimePeriod.UpdateRequest) {
  return request<TimePeriod.Info>({
    url: `${prefix}/${id}`,
    method: 'put',
    params: { orgId: data.orgId },
    data,
  })
}

/** 删除时间段 */
export function deleteTimePeriod(id: string, orgId: string) {
  return request({
    url: `${prefix}/${id}`,
    method: 'delete',
    params: { orgId },
  })
}

/** 切换时间段状态 */
export function toggleTimePeriodStatus(id: string, orgId: string, isActive: boolean) {
  return request({
    url: `${prefix}/${id}/status`,
    method: 'patch',
    params: { orgId },
    data: { isActive },
  })
}
