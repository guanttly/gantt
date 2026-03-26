import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { RoleName } from '@/types/auth'

const router = createRouter({
  history: createWebHistory('/admin/'),
  routes: [
    {
      path: '/login',
      name: 'Login',
      component: () => import('@/views/Login.vue'),
      meta: { public: true },
    },
    {
      path: '/force-reset-password',
      name: 'ForceResetPassword',
      component: () => import('@/views/ForceResetPassword.vue'),
      meta: { public: true },
    },
    {
      path: '/',
      component: () => import('@/components/AdminLayout.vue'),
      redirect: '/dashboard',
      children: [
        {
          path: 'dashboard',
          name: 'Dashboard',
          component: () => import('@/views/Dashboard.vue'),
          meta: { title: '运营看板' },
        },
        {
          path: 'orgs',
          name: 'OrgManagement',
          component: () => import('@/views/OrgManagement.vue'),
          meta: { title: '机构管理' },
        },
        {
          path: 'subscriptions',
          name: 'Subscriptions',
          component: () => import('@/views/SubscriptionList.vue'),
          meta: { title: '订阅管理' },
        },
        {
          path: 'audit',
          name: 'AuditLogs',
          component: () => import('@/views/AuditLogs.vue'),
          meta: { title: '审计日志' },
        },
        {
          path: 'config',
          name: 'SystemConfig',
          component: () => import('@/views/SystemConfig.vue'),
          meta: { title: '系统配置' },
        },
      ],
    },
    {
      path: '/:pathMatch(.*)*',
      redirect: '/',
    },
  ],
})

// 路由守卫
router.beforeEach(async (to, _from, next) => {
  const auth = useAuthStore()

  if (to.meta.public) {
    if (auth.isLoggedIn && to.path === '/login') return next('/')
    return next()
  }

  if (!auth.isLoggedIn) {
    return next({ path: '/login', query: { redirect: to.fullPath } })
  }

  // 恢复用户信息
  if (!auth.user) {
    try {
      await auth.fetchUserInfo()
    }
    catch {
      auth.logout()
      return next({ path: '/login', query: { redirect: to.fullPath } })
    }
  }

  // 强制重置密码
  if (auth.mustResetPwd && to.path !== '/force-reset-password') {
    return next('/force-reset-password')
  }

  // 仅允许平台管理员
  if (!auth.hasRole(RoleName.PlatformAdmin)) {
    auth.logout()
    return next('/login')
  }

  next()
})

export { router }
