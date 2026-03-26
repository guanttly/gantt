<script lang="ts" setup>
import { Calendar, ChatDotRound, Clock, Collection, Document, OfficeBuilding, Setting, User } from '@element-plus/icons-vue'
import { computed, markRaw } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import OrgNodeSelector from '@/components/common/OrgNodeSelector.vue'
import SvgIcon from '@/components/SvgIcon.vue'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

// 图标映射
const iconMap: Record<string, any> = {
  'calendar': markRaw(Calendar),
  'clock': markRaw(Clock),
  'collection': markRaw(Collection),
  'document': markRaw(Document),
  'office-building': markRaw(OfficeBuilding),
  'setting': markRaw(Setting),
  'user': markRaw(User),
  'chat-dot-round': markRaw(ChatDotRound),
}

// ======== 导航菜单定义 ========

interface NavItem {
  path: string
  label: string
  icon?: string
  children?: NavItem[]
  requiredRole?: string
  dividerAfter?: boolean
}

const navGroups: NavItem[] = [
  {
    path: '/scheduling',
    label: '排班工作台',
    icon: 'calendar',
    children: [
      { path: '/scheduling', label: '排班列表' },
      { path: '/scheduling/create', label: '创建排班' },
    ],
  },
  {
    path: '/employees',
    label: '数据管理',
    icon: 'user',
    children: [
      { path: '/employees', label: '员工管理', icon: 'user' },
      { path: '/shifts', label: '班次管理', icon: 'clock' },
      { path: '/groups', label: '分组管理', icon: 'collection' },
      { path: '/rules', label: '排班规则', icon: 'document' },
      { path: '/leaves', label: '请假管理', icon: 'calendar', dividerAfter: true },
      { path: '/org', label: '组织管理', icon: 'office-building', requiredRole: 'org_admin' },
    ],
  },
  {
    path: '/ai/chat',
    label: 'AI 助手',
    icon: 'chat-dot-round',
  },
]

// ======== 计算属性 ========

const activeTopPath = computed(() => {
  const p = route.path
  if (p.startsWith('/scheduling'))
    return '/scheduling'
  if (p.startsWith('/employees') || p.startsWith('/shifts') || p.startsWith('/groups') || p.startsWith('/rules') || p.startsWith('/leaves') || p.startsWith('/org'))
    return '/employees'
  if (p.startsWith('/ai'))
    return '/ai/chat'
  return '/dashboard'
})

const currentSubMenus = computed(() => {
  const group = navGroups.find(g => g.path === activeTopPath.value)
  if (group?.children)
    return group.children.filter(c => !c.requiredRole || auth.hasRole(c.requiredRole as any))

  return []
})

const showSubNav = computed(() => currentSubMenus.value.length > 0)

// ======== 方法 ========

function handleMenuClick(path: string) {
  router.push(path)
}

async function handleLogout() {
  auth.logout()
  await router.push('/login')
}
</script>

<template>
  <div class="modern-layout">
    <!-- 顶部导航栏 -->
    <header class="layout-header">
      <div class="header-content">
        <!-- Logo -->
        <div class="logo-section" @click="router.push('/')">
          <div class="logo-icon">
            <SvgIcon name="calendar" size="28px" />
          </div>
          <h1 class="logo-text">
            智能排班
          </h1>
        </div>

        <!-- 主导航 -->
        <nav class="nav-menu">
          <div
            v-for="item in navGroups"
            :key="item.path"
            class="nav-item"
            :class="{ active: activeTopPath === item.path }"
            @click="handleMenuClick(item.path)"
          >
            <el-icon v-if="item.icon" class="nav-icon">
              <component :is="iconMap[item.icon]" />
            </el-icon>
            <span class="nav-label">{{ item.label }}</span>
          </div>


        </nav>

        <!-- 右侧操作区 -->
        <div class="user-section">
          <!-- 组织节点选择器 -->
          <OrgNodeSelector />

          <!-- 用户头像 -->
          <el-dropdown @command="(cmd: string) => cmd === 'logout' && handleLogout()">
            <div class="user-avatar">
              <el-icon><User /></el-icon>
              <span v-if="auth.user" class="user-name">{{ auth.user.username }}</span>
            </div>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item disabled>
                  {{ auth.currentRole }}
                </el-dropdown-item>
                <el-dropdown-item divided command="logout">
                  退出登录
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>
    </header>

    <!-- 二级导航栏 -->
    <nav v-if="showSubNav" class="sub-nav">
      <div class="sub-nav-content">
        <div
          v-for="item in currentSubMenus"
          :key="item.path"
          class="sub-nav-item"
          :class="{ 'active': route.path === item.path, 'has-divider': item.dividerAfter }"
          @click="handleMenuClick(item.path)"
        >
          <el-icon v-if="item.icon" class="sub-nav-icon">
            <component :is="iconMap[item.icon]" />
          </el-icon>
          <span class="sub-nav-label">{{ item.label }}</span>
        </div>
      </div>
    </nav>

    <!-- 主内容区 -->
    <main class="layout-main">
      <router-view v-slot="{ Component }">
        <transition name="fade-slide" mode="out-in">
          <component :is="Component" />
        </transition>
      </router-view>
    </main>
  </div>
</template>

<style lang="scss" scoped>
.modern-layout {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background: var(--bg-page);
}

/* 顶部导航栏 */
.layout-header {
  background: var(--bg-white);
  border-bottom: 1px solid var(--border-light);
  box-shadow: var(--shadow-sm);
  z-index: var(--z-fixed);
}

.header-content {
  display: flex;
  align-items: center;
  height: var(--header-height);
  padding: 0 var(--spacing-xl);
  max-width: 1920px;
  margin: 0 auto;
}

/* Logo */
.logo-section {
  display: flex;
  align-items: center;
  gap: var(--spacing-md);
  margin-right: var(--spacing-3xl);
  cursor: pointer;
  user-select: none;
  transition: transform var(--transition-fast);

  &:hover {
    transform: scale(1.02);
  }

  .logo-icon {
    font-size: 28px;
    line-height: 1;
  }

  .logo-text {
    font-size: var(--font-size-2xl);
    font-weight: var(--font-weight-semibold);
    color: var(--text-primary);
    margin: 0;
  }
}

/* 导航 */
.nav-menu {
  display: flex;
  align-items: center;
  gap: var(--spacing-base);
  flex: 1;
}

.nav-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 8px;
  max-height: 56px;
  border-radius: var(--radius-lg);
  cursor: pointer;
  transition: all var(--transition-fast);
  color: var(--text-regular);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  white-space: nowrap;

  .nav-icon {
    width: 22px;
    height: 22px;
    font-size: 22px;
    flex-shrink: 0;
  }

  .nav-label {
    font-size: 13px;
    line-height: 1.2;
  }

  &:hover {
    background: var(--bg-hover);
    color: var(--primary-color);
  }

  &.active {
    background: var(--primary-light);
    color: var(--primary-color);
    font-weight: var(--font-weight-semibold);
  }
}

/* 二级导航 */
.sub-nav {
  background: var(--bg-white);
  border-bottom: 1px solid var(--border-light);
}

.sub-nav-content {
  display: flex;
  align-items: center;
  gap: var(--spacing-xs);
  height: 48px;
  padding: 0 var(--spacing-xl);
  max-width: 1920px;
  margin: 0 auto;
  overflow-x: auto;
}

.sub-nav-item {
  display: flex;
  align-items: center;
  gap: var(--spacing-xs);
  padding: var(--spacing-xs) var(--spacing-lg);
  border-radius: var(--radius-base);
  cursor: pointer;
  transition: all var(--transition-fast);
  color: var(--text-regular);
  font-size: var(--font-size-base);
  white-space: nowrap;
  position: relative;

  &.has-divider::after {
    content: '';
    position: absolute;
    right: -6px;
    top: 50%;
    transform: translateY(-50%);
    width: 1px;
    height: 20px;
    background: var(--border-base);
  }

  .sub-nav-icon {
    width: 16px;
    height: 16px;
    font-size: 16px;
    flex-shrink: 0;
  }

  &:hover {
    background: var(--bg-hover);
    color: var(--primary-color);
  }

  &.active {
    color: var(--primary-color);
    font-weight: var(--font-weight-semibold);
    background: var(--primary-light);
  }
}

/* 用户区域 */
.user-section {
  display: flex;
  align-items: center;
  gap: var(--spacing-base);
}

.user-avatar {
  display: flex;
  align-items: center;
  gap: var(--spacing-xs);
  padding: 4px 12px;
  border-radius: var(--radius-round);
  background: var(--bg-hover);
  color: var(--text-regular);
  cursor: pointer;
  transition: all var(--transition-fast);

  &:hover {
    background: var(--border-light);
  }

  .el-icon {
    font-size: 18px;
  }

  .user-name {
    font-size: 13px;
    max-width: 80px;
    overflow: hidden;
    text-overflow: ellipsis;
  }
}

/* 主内容区 */
.layout-main {
  flex: 1;
  overflow: hidden;
  position: relative;
}

/* 过渡动画 */
.fade-slide-enter-active,
.fade-slide-leave-active {
  transition: all var(--transition-base) ease;
}

.fade-slide-enter-from {
  opacity: 0;
  transform: translateY(-10px);
}

.fade-slide-leave-to {
  opacity: 0;
  transform: translateY(10px);
}

/* 响应式 */
@media (max-width: 768px) {
  .header-content {
    padding: 0 var(--spacing-md);
  }

  .logo-section {
    margin-right: var(--spacing-lg);

    .logo-text {
      display: none;
    }
  }

  .user-name {
    display: none;
  }
}
</style>
