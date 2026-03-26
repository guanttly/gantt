// 权限检查组合函数
import type { RoleName } from '@/types/auth'
import { computed } from 'vue'
import { useAuthStore } from '@/stores/auth'

export function usePermission() {
  const auth = useAuthStore()

  /** 检查是否拥有指定角色（含更高级别） */
  function hasRole(role: RoleName): boolean {
    return auth.hasRole(role)
  }

  /** 检查是否可以管理员工 */
  const canManageEmployees = computed(() => auth.hasRole('dept_admin'))

  /** 检查是否可以管理排班 */
  const canManageSchedules = computed(() => auth.hasRole('scheduler'))

  /** 检查是否可以管理规则 */
  const canManageRules = computed(() => auth.hasRole('dept_admin'))

  /** 检查是否可以管理组织 */
  const canManageOrg = computed(() => auth.hasRole('org_admin'))

  return {
    hasRole,
    canManageEmployees,
    canManageSchedules,
    canManageRules,
    canManageOrg,
  }
}
