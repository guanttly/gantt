// 该模块（包括页面data和interface）相关的接口请求方法
// 若该模块下的多个页面，统一到这一个文件中后代码行数超过500行，难以维护
// 则建议按照页面粒度来拆分
// （/permissions/data/index.ts、/permissions/interface/index.ts）
import { request } from '@/utils/request'

const prefix = '/auth-server'

// 获取当前用户权限系统上配置的功能权限列表
export function getFuncList() {
  return request({
    url: `${prefix}/menu/getPermission`,
    method: 'get',
  })
}
// 获取当前用户权限系统上配置的菜单权限列表
export function getMenuList() {
  return request({
    url: `${prefix}/menu/getMenuRouters`,
    method: 'get',
  })
}
/**
 * 获取当前用户权限系统上配置的角色列表(一个用户可配多个角色)
 * @param userId 用户ID
 * @returns 角色列表
 */
export function getRoleList(userId: number) {
  return request({
    url: `${prefix}/user/authRole/query?userId=${userId}`,
    method: 'get',
  })
}
// 获取权限系统上配置的所有角色列表
export function getAllRoleList() {
  return request({
    url: `${prefix}/role/list`,
    method: 'get',
  })
}
