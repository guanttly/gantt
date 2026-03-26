// 认证相关组合函数
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

export function useAuth() {
  const auth = useAuthStore()
  const router = useRouter()

  /** 登录并跳转 */
  async function login(username: string, password: string, redirect?: string) {
    const res = await auth.login(username, password)

    // 如果有多个可用节点且未自动选中，跳节点选择页
    if (res.available_nodes.length > 1 && !res.current_node) {
      await router.push('/select-node')
    }
    else {
      await router.push(redirect || '/')
    }
  }

  /** 登出并跳转登录页 */
  async function logout() {
    auth.logout()
    await router.push('/login')
  }

  /** 选择节点并进入首页 */
  async function selectNode(nodeId: string) {
    await auth.selectNode(nodeId)
    await router.push('/')
  }

  return {
    ...auth,
    login,
    logout,
    selectNode,
  }
}
