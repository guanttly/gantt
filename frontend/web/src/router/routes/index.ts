import type { RouteRecordRaw } from 'vue-router'

// 根路由
export const RootRoute: RouteRecordRaw[] = [{
  path: '/',
  name: 'Root',
  component: () => import('@/components/layout/AppLayout.vue'),
  meta: {
    title: () => '首页',
    hideMenu: true,
  },
  redirect: '/dashboard',
  children: [
    // 仪表盘
    {
      path: 'dashboard',
      name: 'Dashboard',
      component: () => import('@/views/dashboard/Dashboard.vue'),
      meta: { title: () => '仪表盘' },
    },

    // ========== 排班工作台 ==========
    {
      path: 'scheduling',
      name: 'ScheduleList',
      component: () => import('@/views/scheduling/ScheduleList.vue'),
      meta: { title: () => '排班列表', icon: 'calendar', requiredAnyPermissions: ['schedule:view:node', 'schedule:view:all'] },
    },
    {
      path: 'scheduling/mine',
      name: 'MySchedule',
      component: () => import('@/views/scheduling/MySchedule.vue'),
      meta: { title: () => '我的排班', icon: 'calendar', requiredPermission: 'schedule:view:self' },
    },
    {
      path: 'scheduling/workspace',
      name: 'SchedulingWorkspace',
      component: () => import('@/views/scheduling/SchedulingWorkspace.vue'),
      meta: { title: () => '排班工作台', requiredPermission: 'schedule:execute' },
    },
    {
      path: 'scheduling/create',
      name: 'ScheduleCreate',
      component: () => import('@/views/scheduling/ScheduleCreate.vue'),
      meta: { title: () => '创建排班', requiredPermission: 'schedule:create', hideMenu: true },
    },
    {
      path: 'scheduling/:id',
      name: 'ScheduleDetail',
      component: () => import('@/views/scheduling/ScheduleDetail.vue'),
      meta: { title: () => '排班详情', hideMenu: true, requiredAnyPermissions: ['schedule:view:node', 'schedule:view:all'] },
    },

    {
      path: 'leaves',
      name: 'LeaveList',
      component: () => import('@/views/leaves/LeaveList.vue'),
      meta: { title: () => '请假管理', icon: 'calendar' },
    },

    {
      path: 'groups',
      name: 'GroupManagement',
      component: () => import('@/views/groups/GroupList.vue'),
      meta: { title: () => '分组管理', requiredAnyPermissions: ['group:view:node', 'group:manage'] },
    },
    {
      path: 'shifts',
      name: 'ShiftManagement',
      component: () => import('@/views/shifts/ShiftList.vue'),
      meta: { title: () => '班次管理', requiredAnyPermissions: ['shift:view:node', 'shift:manage'] },
    },
    {
      path: 'rules',
      name: 'RuleManagement',
      component: () => import('@/views/rules/RuleList.vue'),
      meta: { title: () => '规则管理', requiredAnyPermissions: ['rule:view:node', 'rule:manage'] },
    },
    {
      path: 'app-permissions',
      name: 'AppPermissionManagement',
      component: () => import('@/views/permissions/AppPermissionManagement.vue'),
      meta: { title: () => '应用权限', requiredPermission: 'app-role:manage' },
    },

    {
      path: 'management-admin',
      name: 'ManagementMoved',
      component: () => import('@/views/notice/ManagementMoved.vue'),
      meta: { title: () => '配置入口已迁移', hideMenu: true },
    },

    // ========== AI ==========
    {
      path: 'ai/chat',
      name: 'AIChat',
      component: () => import('@/views/ai/ChatView.vue'),
      meta: { title: () => 'AI 助手', icon: 'chat-dot-round' },
    },
  ],
}]

// 登录路由
export const LoginRoute: RouteRecordRaw = {
  path: '/login',
  name: 'Login',
  component: () => import('@/views/auth/Login.vue'),
  meta: {
    title: () => '登录',
    public: true,
  },
}

// 节点选择路由
export const NodeSelectRoute: RouteRecordRaw = {
  path: '/select-node',
  name: 'NodeSelect',
  component: () => import('@/views/auth/NodeSelect.vue'),
  meta: {
    title: () => '选择组织节点',
  },
}

// 强制重置密码路由
export const ForceResetPasswordRoute: RouteRecordRaw = {
  path: '/force-reset-password',
  name: 'ForceResetPassword',
  component: () => import('@/views/auth/ForceResetPassword.vue'),
  meta: {
    title: () => '重置密码',
  },
}

// 错误页面
export const ErrorRoutes: RouteRecordRaw[] = [
  {
    path: '/403',
    name: 'Forbidden',
    component: () => import('@/views/errors/Forbidden.vue'),
    meta: { title: () => '无权限', public: true },
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'NotFound',
    component: () => import('@/views/errors/NotFound.vue'),
    meta: { title: () => '页面不存在', public: true },
  },
]

// 兼容旧路由 — 支持从旧路径跳转（过渡期）
const LegacyRedirects: RouteRecordRaw[] = [
  { path: '/workspace', redirect: '/scheduling' },
  { path: '/workspace/scheduling/create', redirect: '/scheduling/create' },
  { path: '/workspace/scheduling/history', redirect: '/scheduling' },
  { path: '/management/employee', redirect: '/management-admin' },
  { path: '/management/shift', redirect: '/shifts' },
  { path: '/management/group', redirect: '/groups' },
  { path: '/management/scheduling-rule', redirect: '/rules' },
  { path: '/management/leave', redirect: '/leaves' },
  { path: '/management/department', redirect: '/management-admin' },
  { path: '/management/app-permissions', redirect: '/app-permissions' },
  { path: '/employees', redirect: '/management-admin' },
  { path: '/org', redirect: '/management-admin' },
  { path: '/settings', redirect: '/management-admin' },
]

export const allRoutes: RouteRecordRaw[] = [
  LoginRoute,
  NodeSelectRoute,
  ForceResetPasswordRoute,
  ...RootRoute,
  ...LegacyRedirects,
  ...ErrorRoutes,
]
