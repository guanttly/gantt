// 权限检查组合函数
import type { AppPermission, RoleName } from '@/types/auth'
import { computed } from 'vue'
import { useAuthStore } from '@/stores/auth'

export function usePermission() {
  const auth = useAuthStore()

  /** 检查是否拥有指定角色（含更高级别） */
  function hasRole(role: RoleName): boolean {
    return auth.hasRole(role)
  }

  function hasPermission(permission: AppPermission | string): boolean {
    return auth.hasPermission(permission)
  }

  /** 检查是否可以管理员工 */
  const canManageEmployees = computed(() => false)

  /** 检查是否可以管理排班 */
  const canManageSchedules = computed(() => auth.hasAnyPermission(['schedule:create', 'schedule:execute', 'schedule:adjust', 'schedule:publish']))

  /** 检查是否可以创建排班 */
  const canCreateSchedules = computed(() => auth.hasPermission('schedule:create'))

  /** 检查是否可以执行排班 */
  const canExecuteSchedules = computed(() => auth.hasPermission('schedule:execute'))

  /** 检查是否可以发布排班 */
  const canPublishSchedules = computed(() => auth.hasPermission('schedule:publish'))

  /** 检查是否可以管理规则 */
  const canManageRules = computed(() => false)

  /** 检查是否可以管理组织 */
  const canManageOrg = computed(() => auth.hasRole('org_admin'))

  /** 检查是否可以审批请假 */
  const canApproveLeaves = computed(() => auth.hasPermission('leave:approve'))

  return {
    hasRole,
    hasPermission,
    canManageEmployees,
    canManageSchedules,
    canCreateSchedules,
    canExecuteSchedules,
    canPublishSchedules,
    canManageRules,
    canManageOrg,
    canApproveLeaves,
  }
}
