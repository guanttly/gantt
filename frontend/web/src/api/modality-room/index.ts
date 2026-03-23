import { request } from '@/utils/request'

const prefix = '/v1/modality-rooms'

/** 查询机房列表 */
export function getModalityRoomList(params: ModalityRoom.ListParams) {
  return request<ModalityRoom.ListResult>({
    url: prefix,
    method: 'get',
    params,
  })
}

/** 获取启用的机房 */
export function getActiveModalityRooms(orgId: string) {
  return request<ModalityRoom.Info[]>({
    url: `${prefix}/active`,
    method: 'get',
    params: { orgId },
  })
}

/** 获取机房详情 */
export function getModalityRoom(id: string, orgId: string) {
  return request<ModalityRoom.Info>({
    url: `${prefix}/${id}`,
    method: 'get',
    params: { orgId },
  })
}

/** 创建机房 */
export function createModalityRoom(data: ModalityRoom.CreateRequest) {
  return request<ModalityRoom.Info>({
    url: prefix,
    method: 'post',
    data,
  })
}

/** 更新机房 */
export function updateModalityRoom(id: string, data: ModalityRoom.UpdateRequest) {
  return request<ModalityRoom.Info>({
    url: `${prefix}/${id}`,
    method: 'put',
    params: { orgId: data.orgId },
    data,
  })
}

/** 删除机房 */
export function deleteModalityRoom(id: string, orgId: string) {
  return request({
    url: `${prefix}/${id}`,
    method: 'delete',
    params: { orgId },
  })
}

/** 切换机房状态 */
export function toggleModalityRoomStatus(id: string, orgId: string, isActive: boolean) {
  return request({
    url: `${prefix}/${id}/status`,
    method: 'patch',
    params: { orgId },
    data: { isActive },
  })
}

/** 获取机房周检查量配置 */
export function getWeeklyVolumes(id: string, orgId: string) {
  return request<ModalityRoom.WeeklyVolumeListResult>({
    url: `${prefix}/${id}/weekly-volumes`,
    method: 'get',
    params: { orgId },
  })
}

/** 保存机房周检查量配置 */
export function saveWeeklyVolumes(id: string, orgId: string, items: ModalityRoom.WeeklyVolumeItem[]) {
  return request({
    url: `${prefix}/${id}/weekly-volumes`,
    method: 'put',
    params: { orgId },
    data: items,
  })
}

/** 删除机房周检查量配置 */
export function deleteWeeklyVolumes(id: string, orgId: string) {
  return request({
    url: `${prefix}/${id}/weekly-volumes`,
    method: 'delete',
    params: { orgId },
  })
}
