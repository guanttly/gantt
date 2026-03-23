import { request } from '@/utils/request'

export interface SystemSetting {
  id: string
  orgId: string
  key: string
  value: string
  description?: string
  createdAt: string
  updatedAt: string
}

export interface SystemSettingResponse {
  key: string
  value: string
}

/**
 * 获取所有系统设置
 */
export function getAllSettings(orgId: string) {
  return request<SystemSetting[]>({
    url: '/v1/system-settings',
    method: 'get',
    params: { orgId },
  })
}

/**
 * 获取单个系统设置
 */
export function getSetting(orgId: string, key: string) {
  return request<SystemSettingResponse>({
    url: `/v1/system-settings/${key}`,
    method: 'get',
    params: { orgId },
  })
}

/**
 * 设置系统设置
 */
export function setSetting(orgId: string, key: string, value: string, description?: string) {
  return request<SystemSettingResponse>({
    url: `/v1/system-settings/${key}`,
    method: 'put',
    params: { orgId },
    data: {
      value,
      description,
    },
  })
}

/**
 * 删除系统设置
 */
export function deleteSetting(orgId: string, key: string) {
  return request({
    url: `/v1/system-settings/${key}`,
    method: 'delete',
    params: { orgId },
  })
}
