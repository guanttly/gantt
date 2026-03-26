// 管理后台认证状态
import type { CurrentNode, OrgNode, RoleName, User } from '@/types/auth'
import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { login as apiLogin, getMe } from '@/api/auth'
import { clearTokens, getAccessToken, setAccessToken, setRefreshToken } from '@/api/client'
import { ROLE_HIERARCHY } from '@/types/auth'

export const useAuthStore = defineStore('auth', () => {
  const accessToken = ref<string | null>(getAccessToken())
  const user = ref<User | null>(null)
  const currentNode = ref<CurrentNode | null>(null)
  const availableNodes = ref<OrgNode[]>([])
  const mustResetPwd = ref(false)

  const isLoggedIn = computed(() => !!accessToken.value)
  const currentRole = computed(() => (currentNode.value?.role_name ?? '') as RoleName)

  function hasRole(role: RoleName): boolean {
    return ROLE_HIERARCHY.indexOf(currentRole.value) >= ROLE_HIERARCHY.indexOf(role)
  }

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

  async function fetchUserInfo() {
    const res = await getMe()
    user.value = res.user
    currentNode.value = res.current_node
    availableNodes.value = res.available_nodes
    mustResetPwd.value = res.user.must_reset_pwd
  }

  function logout() {
    accessToken.value = null
    user.value = null
    currentNode.value = null
    availableNodes.value = []
    mustResetPwd.value = false
    clearTokens()
  }

  return {
    accessToken,
    user,
    currentNode,
    availableNodes,
    mustResetPwd,
    isLoggedIn,
    currentRole,
    hasRole,
    login,
    fetchUserInfo,
    logout,
  }
})
