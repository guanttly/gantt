// 排班工作台路由 - 核心功能区
export default {
  path: '/workspace',
  name: 'Workspace',
  component: () => import('@/layouts/modern.vue'),
  meta: {
    title: () => '排班工作台',
    icon: 'calendar',
    hideMenu: false,
    requiresAuth: true,
    order: 1, // 最高优先级
  },
  children: [
    {
      path: '',
      name: 'WorkspaceIndex',
      component: () => import('@/pages/scheduling/index.vue'),
      meta: {
        title: () => '排班工作台',
        requiresAuth: true,
      },
    },
  ],
}
