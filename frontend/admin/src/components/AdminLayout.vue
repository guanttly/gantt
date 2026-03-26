<script setup lang="ts">
import { BarChartOutline, BusinessOutline, DocumentTextOutline, ListOutline, LogOutOutline, PeopleOutline, PersonCircleOutline, PersonOutline, ReceiptOutline, SettingsOutline, ShieldCheckmarkOutline, TimeOutline } from '@vicons/ionicons5'
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
  { path: '/orgs', label: '机构管理', icon: BusinessOutline, requiredRole: RoleName.PlatformAdmin },
  { path: '/employees', label: '员工管理', icon: PeopleOutline, requiredRole: RoleName.DeptAdmin },
  { path: '/groups', label: '分组管理', icon: ListOutline, requiredRole: RoleName.DeptAdmin },
  { path: '/shifts', label: '班次管理', icon: TimeOutline, requiredRole: RoleName.DeptAdmin },
  { path: '/rules', label: '规则管理', icon: DocumentTextOutline, requiredRole: RoleName.DeptAdmin },
  { path: '/platform-users', label: '平台账号', icon: PersonOutline, requiredRole: RoleName.OrgAdmin },
  { path: '/app-permissions', label: '应用权限', icon: ShieldCheckmarkOutline, requiredRole: RoleName.DeptAdmin },
  { path: '/subscriptions', label: '订阅管理', icon: ReceiptOutline, requiredRole: RoleName.PlatformAdmin },
  { path: '/audit', label: '审计日志', icon: DocumentTextOutline, requiredRole: RoleName.PlatformAdmin },
  { path: '/config', label: '系统配置', icon: SettingsOutline, requiredRole: RoleName.PlatformAdmin },
]

const visibleMenuItems = computed(() => menuItems.filter(item => auth.hasRole(item.requiredRole)))

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
        <div class="sidebar-badge">平台控制台</div>
        <h2 class="sidebar-title">平台管理后台</h2>
        <p class="sidebar-subtitle">多租户管理、订阅运营与系统配置统一收口。</p>
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
            <span class="user-role">平台管理员</span>
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
        <h1 class="content-heading">{{ visibleMenuItems.find(item => item.path === activePath)?.label || '平台管理' }}</h1>
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
