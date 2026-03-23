import { authServer } from '@/enums/urlEnum'
import { request } from '@/utils/request'

// 获取RSA公钥
export function getRsaKey(): Promise<string> {
  return request<string>({
    url: `${authServer}/open/getKeys`,
    method: 'get',
  })
}

// 登录
export function login(data: API.LoginParams): Promise<string> {
  return request<string>({
    url: `${authServer}/open/login`,
    method: 'post',
    data,
  })
}

// 登出
export function logout() {
  return request({
    url: `${authServer}/open/logout`,
    method: 'post',
  })
}
