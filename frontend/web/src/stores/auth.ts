// 认证状态管理 — 多租户组织节点支持
import type { AppPermission, AppPermissionsResponse, AppRoleGrant, AppRolesResponse, CurrentNode, OrgNode, RoleName, User, UserInfoResponse } from '@/types/auth'
import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { getMe, getMyAppPermissions, getMyAppRoles, login as apiLogin, refreshToken as apiRefresh, switchNode as apiSwitch } from '@/api/auth'
import { clearTokens, getAccessToken, getRefreshToken, setAccessToken, setRefreshToken } from '@/api/client'
import { ROLE_HIERARCHY } from '@/types/auth'

export const useAuthStore = defineStore('auth', () => {
  // ======== 状态 ========
  const accessToken = ref<string | null>(getAccessToken())
  const user = ref<User | null>(null)
  const currentNode = ref<CurrentNode | null>(null)
  const availableNodes = ref<OrgNode[]>([])
  const mustResetPwd = ref(false)
  const appRoles = ref<AppRoleGrant[]>([])
  const appPermissions = ref<AppPermission[]>([])
  const boundEmployeeId = ref<string | null>(null)

  // ======== 计算属性 ========
  const isLoggedIn = computed(() => !!accessToken.value)
  const currentNodeId = computed(() => currentNode.value?.node_id ?? null)
  const currentNodePath = computed(() => currentNode.value?.node_path ?? '')
  const currentRole = computed(() => (currentNode.value?.role_name ?? '') as RoleName)

  // ======== 方法 ========

  /** 角色层级检查：当前角色 >= 目标角色 */
  function hasRole(role: RoleName): boolean {
    return ROLE_HIERARCHY.indexOf(currentRole.value) >= ROLE_HIERARCHY.indexOf(role)
  }

  function hasPermission(permission: AppPermission | string): boolean {
    return appPermissions.value.includes(permission as AppPermission)
  }

  function hasAnyPermission(permissions: Array<AppPermission | string>): boolean {
    return permissions.some(permission => hasPermission(permission))
  }

  async function syncAppAccess() {
    if (!accessToken.value) {
      appRoles.value = []
      appPermissions.value = []
      boundEmployeeId.value = null
      return
    }

    try {
      const [rolesPayload, permissionsPayload] = await Promise.all([
        getMyAppRoles(),
        getMyAppPermissions(),
      ])
      const roles = rolesPayload as AppRolesResponse
      const permissions = permissionsPayload as AppPermissionsResponse
      appRoles.value = roles.app_roles || []
      appPermissions.value = (permissions.permissions || []) as AppPermission[]
      boundEmployeeId.value = permissions.employee_id || roles.employee_id || null
    }
    catch (error: any) {
      if (error?.response?.status === 403 || error?.response?.status === 404) {
        appRoles.value = []
        appPermissions.value = []
        boundEmployeeId.value = null
        return
      }
      throw error
    }
  }

  /** 登录 */
  async function login(username: string, password: string) {
    const res = await apiLogin({ username, password })
    accessToken.value = res.access_token
    setAccessToken(res.access_token)
    setRefreshToken(res.refresh_token)
    user.value = res.user
    currentNode.value = res.current_node
    availableNodes.value = res.available_nodes
    mustResetPwd.value = res.must_reset_pwd
    await syncAppAccess()
    return res
  }

  /** 选择/切换组织节点 */
  async function selectNode(nodeId: string) {
    const res = await apiSwitch({ org_node_id: nodeId })
    accessToken.value = res.access_token
    setAccessToken(res.access_token)
    if (res.refresh_token)
      setRefreshToken(res.refresh_token)
    currentNode.value = res.current_node
    availableNodes.value = res.available_nodes
    await syncAppAccess()
    return res
  }

  /** 刷新 Token */
  async function doRefreshToken(): Promise<string> {
    const rt = getRefreshToken()
    if (!rt)
      throw new Error('no_refresh_token')

    const res = await apiRefresh({ refresh_token: rt })
    accessToken.value = res.access_token
    setAccessToken(res.access_token)
    if (res.refresh_token)
      setRefreshToken(res.refresh_token)
    await syncAppAccess()
    return res.access_token
  }

  /** 获取用户信息（页面刷新时恢复） */
  async function fetchUserInfo() {
    const res = await getMe() as UserInfoResponse
    user.value = res.user
    currentNode.value = res.current_node
    availableNodes.value = res.available_nodes
    mustResetPwd.value = res.user.must_reset_pwd
    await syncAppAccess()
  }

  /** 登出 */
  function logout() {
    accessToken.value = null
    user.value = null
    currentNode.value = null
    availableNodes.value = []
    mustResetPwd.value = false
    appRoles.value = []
    appPermissions.value = []
    boundEmployeeId.value = null
    clearTokens()
  }

  return {
    // 状态
    accessToken,
    user,
    currentNode,
    availableNodes,
    mustResetPwd,
    appRoles,
    appPermissions,
    boundEmployeeId,
    // 计算属性
    isLoggedIn,
    currentNodeId,
    currentNodePath,
    currentRole,
    // 方法
    hasRole,
    hasPermission,
    hasAnyPermission,
    login,
    selectNode,
    doRefreshToken,
    fetchUserInfo,
    syncAppAccess,
    logout,
  }
})
