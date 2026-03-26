// 路由守卫 — 登录鉴权、节点选择、权限检查
import type { Router } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const PUBLIC_ROUTES = ['/login', '/force-reset-password']

export function setupGuards(router: Router) {
  router.beforeEach(async (to, _from, next) => {
    const auth = useAuthStore()

    // 1. 公开页面
    if (PUBLIC_ROUTES.includes(to.path)) {
      // 已登录则跳首页
      if (auth.isLoggedIn)
        return next('/')
      return next()
    }

    // 2. 未登录 → 跳转登录页
    if (!auth.isLoggedIn) {
      return next({ path: '/login', query: { redirect: to.fullPath } })
    }

    // 3. 已登录但无用户信息 → 尝试恢复
    if (!auth.user) {
      try {
        await auth.fetchUserInfo()
      }
      catch {
        auth.logout()
        return next({ path: '/login', query: { redirect: to.fullPath } })
      }
    }

    // 3.5. 强制重置密码检查
    if (auth.mustResetPwd && to.path !== '/force-reset-password') {
      return next('/force-reset-password')
    }

    // 4. 未选择节点 → 跳转节点选择页
    if (!auth.currentNodeId && to.path !== '/select-node') {
      return next('/select-node')
    }

    // 5. 权限检查
    if (to.meta.requiredRole) {
      const requiredRole = to.meta.requiredRole as string
      if (!auth.hasRole(requiredRole as any)) {
        return next('/403')
      }
    }

    if (to.meta.requiredPermission) {
      const requiredPermission = to.meta.requiredPermission as string
      if (!auth.hasPermission(requiredPermission)) {
        return next('/403')
      }
    }

    if (to.meta.requiredAnyPermissions) {
      const requiredPermissions = to.meta.requiredAnyPermissions as string[]
      if (!auth.hasAnyPermission(requiredPermissions)) {
        return next('/403')
      }
    }

    next()
  })
}
