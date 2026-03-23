// 数据管理模块路由 - 辅助配置功能
export default [
  {
    path: '/management',
    name: 'Management',
    redirect: '/management/employee',
    component: () => import('@/layouts/modern.vue'),
    meta: {
      title: () => '数据管理',
      icon: 'data-analysis',
      hideMenu: false,
      requiresAuth: true,
      order: 2, // 次要优先级
    },
    children: [
      // ========== 常用功能 ==========
      {
        path: 'employee',
        name: 'EmployeeManagement',
        component: () => import('@/pages/management/employee/index.vue'),
        meta: {
          title: () => '员工管理',
          icon: 'user',
          requiresAuth: true,
        },
      },
      {
        path: 'shift',
        name: 'ShiftManagement',
        component: () => import('@/pages/management/shift/index.vue'),
        meta: {
          title: () => '班次管理',
          icon: 'clock',
          requiresAuth: true,
        },
      },
      {
        path: 'group',
        name: 'GroupManagement',
        component: () => import('@/pages/management/group/index.vue'),
        meta: {
          title: () => '分组管理',
          icon: 'collection',
          requiresAuth: true,
        },
      },
      {
        path: 'scheduling-rule',
        name: 'SchedulingRuleManagement',
        component: () => import('@/pages/management/scheduling-rule/index.vue'),
        meta: {
          title: () => '排班规则',
          icon: 'document',
          requiresAuth: true,
        },
      },
      {
        path: 'leave',
        name: 'LeaveManagement',
        component: () => import('@/pages/management/leave/index.vue'),
        meta: {
          title: () => '请假管理',
          icon: 'calendar',
          requiresAuth: true,
          hasDivider: true, // 分隔线标记
        },
      },
      // ========== 配置功能 ==========
      {
        path: 'department',
        name: 'DepartmentManagement',
        component: () => import('@/pages/management/department/index.vue'),
        meta: {
          title: () => '部门管理',
          icon: 'office-building',
          requiresAuth: true,
        },
      },
      {
        path: 'shift-type',
        name: 'ShiftTypeManagement',
        component: () => import('@/pages/management/shift-type/index.vue'),
        meta: {
          title: () => '班次类型管理',
          icon: 'collection-tag',
          requiresAuth: true,
        },
      },
      {
        path: 'time-period',
        name: 'TimePeriodManagement',
        component: () => import('@/pages/management/time-period/index.vue'),
        meta: {
          title: () => '时间段管理',
          icon: 'timer',
          requiresAuth: true,
        },
      },
      {
        path: 'modality-room',
        name: 'ModalityRoomManagement',
        component: () => import('@/pages/management/modality-room/index.vue'),
        meta: {
          title: () => '机房管理',
          icon: 'monitor',
          requiresAuth: true,
        },
      },
      {
        path: 'scan-type',
        name: 'ScanTypeManagement',
        component: () => import('@/pages/management/scan-type/index.vue'),
        meta: {
          title: () => '检查类型管理',
          icon: 'list',
          requiresAuth: true,
        },
      },
      {
        path: 'staffing',
        name: 'StaffingManagement',
        component: () => import('@/pages/management/staffing/index.vue'),
        meta: {
          title: () => '排班人数计算',
          icon: 'data-analysis',
          requiresAuth: true,
        },
      },
    ],
  },
]
