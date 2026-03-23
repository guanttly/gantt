/**
 * API 路径配置
 * 使用环境变量统一管理 API 基础路径
 */

// 从环境变量读取 API 基础路径，如果未配置则使用默认值
export const baseApiUrl = import.meta.env.VITE_APP_API_BASE_URL || '/api/'

// 微服务名-权限管理部分，包括分组管理、用户管理、角色管理、菜单路由管理、登录登出等
export const authServer = '/auth-server'
