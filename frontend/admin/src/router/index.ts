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
          meta: { title: '运营看板', requiredRole: RoleName.PlatformAdmin },
        },
        {
          path: 'orgs',
          name: 'OrgManagement',
          component: () => import('@/views/OrgManagement.vue'),
          meta: { title: '机构管理', requiredRole: RoleName.PlatformAdmin },
        },
        {
          path: 'employees',
          name: 'EmployeeManagement',
          component: () => import('@/views/EmployeeManagement.vue'),
          meta: { title: '员工管理', requiredRole: RoleName.DeptAdmin },
        },
        {
          path: 'groups',
          name: 'GroupManagement',
          component: () => import('@/views/GroupManagement.vue'),
          meta: { title: '分组管理', requiredRole: RoleName.DeptAdmin },
        },
        {
          path: 'shifts',
          name: 'ShiftManagement',
          component: () => import('@/views/ShiftManagement.vue'),
          meta: { title: '班次管理', requiredRole: RoleName.DeptAdmin },
        },
        {
          path: 'rules',
          name: 'RuleManagement',
          component: () => import('@/views/RuleManagement.vue'),
          meta: { title: '规则管理', requiredRole: RoleName.DeptAdmin },
        },
        {
          path: 'platform-users',
          name: 'PlatformUserManagement',
          component: () => import('@/views/PlatformUserManagement.vue'),
          meta: { title: '平台账号', requiredRole: RoleName.OrgAdmin },
        },
        {
          path: 'app-permissions',
          name: 'AppPermissionsManagement',
          component: () => import('@/views/AppPermissionsManagement.vue'),
          meta: { title: '应用权限', requiredRole: RoleName.DeptAdmin },
        },
        {
          path: 'subscriptions',
          name: 'Subscriptions',
          component: () => import('@/views/SubscriptionList.vue'),
          meta: { title: '订阅管理', requiredRole: RoleName.PlatformAdmin },
        },
        {
          path: 'audit',
          name: 'AuditLogs',
          component: () => import('@/views/AuditLogs.vue'),
          meta: { title: '审计日志', requiredRole: RoleName.PlatformAdmin },
        },
        {
          path: 'config',
          name: 'SystemConfig',
          component: () => import('@/views/SystemConfig.vue'),
          meta: { title: '系统配置', requiredRole: RoleName.PlatformAdmin },
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

  // 允许进入管理后台的角色：平台管理员 / 机构管理员 / 科室管理员
  if (![RoleName.PlatformAdmin, RoleName.OrgAdmin, RoleName.DeptAdmin].includes(auth.currentRole)) {
    auth.logout()
    return next('/login')
  }

  if (to.path === '/dashboard' && !auth.hasRole(RoleName.PlatformAdmin)) {
    return next('/employees')
  }

  if (to.meta.requiredRole && !auth.hasRole(to.meta.requiredRole as RoleName)) {
    if (auth.hasRole(RoleName.DeptAdmin)) {
      return next('/employees')
    }
    return next('/')
  }

  next()
})

export { router }
