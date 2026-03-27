<script setup lang="ts">
import { BarChartOutline, BusinessOutline, DocumentTextOutline, LogOutOutline, PeopleOutline, PersonCircleOutline, PersonOutline, ReceiptOutline, SettingsOutline } from '@vicons/ionicons5'
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NButton, NIcon } from 'naive-ui'
import { useAuthStore } from '@/stores/auth'
import { RoleName } from '@/types/auth'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

const menuItems = [
  { path: '/dashboard', label: '运营看板', icon: BarChartOutline, requiredRole: RoleName.PlatformAdmin },
  { path: '/orgs', label: '组织管理', icon: BusinessOutline, requiredRole: RoleName.OrgAdmin },
  { path: '/employees', label: '员工管理', icon: PeopleOutline, requiredRole: RoleName.OrgAdmin },
  { path: '/platform-users', label: '平台账号', icon: PersonOutline, requiredRole: RoleName.OrgAdmin },
  { path: '/subscriptions', label: '订阅管理', icon: ReceiptOutline, requiredRole: RoleName.PlatformAdmin },
  { path: '/audit', label: '审计日志', icon: DocumentTextOutline, requiredRole: RoleName.PlatformAdmin },
  { path: '/config', label: '系统配置', icon: SettingsOutline, requiredRole: RoleName.PlatformAdmin },
]

const visibleMenuItems = computed(() => menuItems.filter(item => (item.requiredRole ? auth.hasRole(item.requiredRole) : true)))

const roleLabelMap: Record<RoleName, string> = {
	[RoleName.Employee]: '普通员工',
	[RoleName.Scheduler]: '排班负责人',
	[RoleName.DeptAdmin]: '科室管理员',
	[RoleName.OrgAdmin]: '机构管理员',
	[RoleName.PlatformAdmin]: '平台管理员',
}

const layoutMetaMap: Record<RoleName, { badge: string, title: string, subtitle: string }> = {
  [RoleName.Employee]: {
    badge: '员工视图',
    title: '员工工作台',
    subtitle: '查看个人排班、请假与基础信息。',
  },
  [RoleName.Scheduler]: {
    badge: '排班控制台',
    title: '排班业务后台',
    subtitle: '聚焦分组排班、班次配置与业务规则执行。',
  },
  [RoleName.DeptAdmin]: {
    badge: '科室控制台',
    title: '科室业务后台',
    subtitle: '维护科室员工、排班分组、班次与排班规则。',
  },
  [RoleName.OrgAdmin]: {
    badge: '机构控制台',
    title: '机构管理后台',
    subtitle: '负责人事、组织节点、后台账号与管理能力下放，不直接处理科室排班业务。',
  },
  [RoleName.PlatformAdmin]: {
    badge: '平台控制台',
    title: '平台管理后台',
    subtitle: '多租户管理、订阅运营与系统配置统一收口。',
  },
}

const currentRoleLabel = computed(() => roleLabelMap[auth.currentRole] || '未设置角色')
const currentLayoutMeta = computed(() => {
  const base = layoutMetaMap[auth.currentRole] || layoutMetaMap[RoleName.PlatformAdmin]
  if (auth.currentRole === RoleName.OrgAdmin && auth.currentNode?.node_name) {
    return {
      ...base,
      title: auth.currentNode.node_name,
      subtitle: `负责人事、组织节点、后台账号与管理能力下放，当前管理范围为 ${auth.currentNode.node_name}。`,
    }
  }
  return base
})

const activePath = computed(() => {
  const p = route.path
  const match = visibleMenuItems.value.find(m => p.startsWith(m.path))
  return match?.path ?? visibleMenuItems.value[0]?.path ?? '/employees'
})

function handleMenuClick(path: string) {
  router.push(path)
}

async function handleLogout() {
  auth.logout()
  await router.push('/login')
}
</script>

<template>
  <div class="admin-layout">
    <aside class="sidebar">
      <div class="sidebar-header">
        <div class="sidebar-badge">{{ currentLayoutMeta.badge }}</div>
        <h2 class="sidebar-title">{{ currentLayoutMeta.title }}</h2>
        <p class="sidebar-subtitle">{{ currentLayoutMeta.subtitle }}</p>
      </div>
      <nav class="sidebar-menu">
        <div
          v-for="item in visibleMenuItems"
          :key="item.path"
          class="menu-item"
          :class="{ active: activePath === item.path }"
          role="button"
          tabindex="0"
          @click="handleMenuClick(item.path)"
          @keyup.enter="handleMenuClick(item.path)"
        >
          <n-icon class="menu-icon" :size="18"><component :is="item.icon" /></n-icon>
          <span class="menu-label">{{ item.label }}</span>
        </div>
      </nav>
      <div class="sidebar-footer">
        <div class="user-info">
          <n-icon :size="24" color="rgba(226, 232, 240, 0.86)"><person-circle-outline /></n-icon>
          <div class="user-meta">
            <span v-if="auth.user" class="user-name">{{ auth.user.username }}</span>
            <span class="user-role">{{ currentRoleLabel }}</span>
          </div>
        </div>
        <n-button text class="logout-button" @click="handleLogout">
          <template #icon>
            <n-icon><log-out-outline /></n-icon>
          </template>
          退出
        </n-button>
      </div>
    </aside>

    <div class="content-area">
      <header class="content-header">
        <h1 class="content-heading">{{ visibleMenuItems.find(item => item.path === activePath)?.label || '管理后台' }}</h1>
      </header>
      <main class="main-content">
        <router-view />
      </main>
    </div>
  </div>
</template>

<style scoped>
.admin-layout {
  display: flex;
  height: 100vh;
  background: transparent;
}

.sidebar {
  width: 272px;
  padding: 20px 18px 18px;
  background: linear-gradient(180deg, #102033 0%, #16263a 100%);
  color: #f8fafc;
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  gap: 18px;
}

.sidebar-header {
  padding: 12px 12px 0;
}

.sidebar-badge {
  display: inline-flex;
  width: fit-content;
  padding: 6px 10px;
  border: 1px solid rgba(226, 232, 240, 0.18);
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.06);
  color: rgba(226, 232, 240, 0.72);
  font-size: 11px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.sidebar-title {
  margin: 14px 0 0;
  font-size: 24px;
  line-height: 1.15;
  font-weight: 700;
}

.sidebar-subtitle {
  margin: 12px 0 0;
  color: rgba(226, 232, 240, 0.7);
  font-size: 13px;
  line-height: 1.7;
}

.sidebar-menu {
  flex: 1;
  padding: 10px 6px;
  overflow-y: auto;
  scrollbar-gutter: stable;
  scrollbar-width: thin;
  scrollbar-color: rgba(148, 163, 184, 0.42) transparent;
}

.sidebar-menu::-webkit-scrollbar {
  width: 10px;
}

.sidebar-menu::-webkit-scrollbar-track {
  background: transparent;
}

.sidebar-menu::-webkit-scrollbar-thumb {
  border: 2px solid transparent;
  border-radius: 999px;
  background: linear-gradient(180deg, rgba(148, 163, 184, 0.5), rgba(94, 234, 212, 0.28));
  background-clip: content-box;
}

.sidebar-menu::-webkit-scrollbar-thumb:hover {
  background: linear-gradient(180deg, rgba(226, 232, 240, 0.62), rgba(94, 234, 212, 0.42));
  background-clip: content-box;
}

.menu-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 14px 16px;
  border: 1px solid transparent;
  border-radius: 14px;
  cursor: pointer;
  color: rgba(255, 255, 255, 0.76);
  transition: all 0.2s;
  margin-bottom: 8px;
}

.menu-item:hover {
  background: rgba(255, 255, 255, 0.08);
  border-color: rgba(255, 255, 255, 0.08);
  color: #fff;
}

.menu-item.active {
  background: linear-gradient(135deg, rgba(15, 118, 110, 0.3), rgba(255, 255, 255, 0.08));
  border-color: rgba(94, 234, 212, 0.2);
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.04);
  color: #ffffff;
}

.menu-icon {
  font-size: 18px;
}

.menu-label {
  font-size: 14px;
  font-weight: 500;
}

.sidebar-footer {
  padding: 16px 14px;
  border: 1px solid rgba(226, 232, 240, 0.1);
  border-radius: 18px;
  background: rgba(255, 255, 255, 0.06);
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.user-meta {
  display: flex;
  min-width: 0;
  flex-direction: column;
}

.user-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 14px;
  font-weight: 600;
}

.user-role {
  color: rgba(226, 232, 240, 0.66);
  font-size: 12px;
}

.logout-button {
  color: rgba(226, 232, 240, 0.84);
}

.content-area {
  flex: 1;
  display: flex;
  min-width: 0;
  flex-direction: column;
}

.content-header {
  display: flex;
  align-items: center;
  min-height: 88px;
  padding: 22px 28px 0;
}

.content-eyebrow {
  margin: 0 0 6px;
  color: #0f766e;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.content-heading {
  margin: 0;
  color: #0f172a;
  font-size: 32px;
  line-height: 1.1;
}

.main-content {
  flex: 1;
  min-height: 0;
  overflow: auto;
}

@media (max-width: 960px) {
  .admin-layout {
    flex-direction: column;
    height: auto;
    min-height: 100vh;
  }

  .sidebar {
    width: 100%;
  }

  .content-header {
    min-height: auto;
    padding: 18px 18px 0;
  }

  .content-heading {
    font-size: 26px;
  }
}
</style>
