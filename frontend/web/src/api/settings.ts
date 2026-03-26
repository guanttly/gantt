// 系统设置 API
import client from './client'

/** 获取系统设置 */
export function getSetting(key: string) {
  return client.get<{ key: string, value: string, description?: string }>(`/settings/${key}`).then(r => r.data)
}

/** 设置系统设置 */
export function setSetting(key: string, value: string, description?: string) {
  return client.put(`/settings/${key}`, { value, description })
}

/** 获取用户工作流版本偏好 */

/** 设置用户工作流版本偏好 */
