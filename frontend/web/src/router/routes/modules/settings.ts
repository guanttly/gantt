// 系统设置模块路由
export default {
  path: '/settings',
  name: 'Settings',
  redirect: '/settings/index',
  component: () => import('@/layouts/modern.vue'),
  meta: {
    title: () => '系统设置',
    icon: 'setting',
    hideMenu: false,
    requiresAuth: true,
    order: 3, // 第三优先级，在排班工作台和数据管理之后
  },
  children: [
    {
      path: 'index',
      name: 'SettingsIndex',
      component: () => import('@/pages/settings/index.vue'),
      meta: {
        title: () => '系统设置',
        requiresAuth: true,
      },
    },
  ],
}

