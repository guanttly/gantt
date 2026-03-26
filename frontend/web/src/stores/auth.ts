// 认证状态管理 — 多租户组织节点支持
import type { CurrentNode, OrgNode, RoleName, User } from '@/types/auth'
import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { login as apiLogin, refreshToken as apiRefresh, switchNode as apiSwitch, getMe } from '@/api/auth'
import { clearTokens, getAccessToken, getRefreshToken, setAccessToken, setRefreshToken } from '@/api/client'
import { ROLE_HIERARCHY } from '@/types/auth'

export const useAuthStore = defineStore('auth', () => {
  // ======== 状态 ========
  const accessToken = ref<string | null>(getAccessToken())
  const user = ref<User | null>(null)
  const currentNode = ref<CurrentNode | null>(null)
  const availableNodes = ref<OrgNode[]>([])
  const mustResetPwd = ref(false)

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
    return res.access_token
  }

  /** 获取用户信息（页面刷新时恢复） */
  async function fetchUserInfo() {
    const res = await getMe()
    user.value = res.user
    currentNode.value = res.current_node
    availableNodes.value = res.available_nodes
    mustResetPwd.value = res.user.must_reset_pwd
  }

  /** 登出 */
  function logout() {
    accessToken.value = null
    user.value = null
    currentNode.value = null
    availableNodes.value = []
    mustResetPwd.value = false
    clearTokens()
  }

  return {
    // 状态
    accessToken,
    user,
    currentNode,
    availableNodes,
    mustResetPwd,
    // 计算属性
    isLoggedIn,
    currentNodeId,
    currentNodePath,
    currentRole,
    // 方法
    hasRole,
    login,
    selectNode,
    doRefreshToken,
    fetchUserInfo,
    logout,
  }
})
