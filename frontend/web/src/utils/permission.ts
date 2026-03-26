// 权限相关工具方法
import type { RoleName } from '@/types/auth'
import { useAuthStore } from '@/stores/auth'
import { ROLE_HIERARCHY } from '@/types/auth'

/**
 * 检查当前用户是否具有指定角色或更高权限
 */
export function hasRole(requiredRole: RoleName): boolean {
  const authStore = useAuthStore()
  return authStore.hasRole(requiredRole)
}

/**
 * 获取当前用户的角色等级
 */
export function getRoleLevel(role: RoleName): number {
  return ROLE_HIERARCHY.indexOf(role)
}
