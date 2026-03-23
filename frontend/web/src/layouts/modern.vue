<script lang="ts" setup>
import { Calendar, Clock, Collection, DataAnalysis, Document, List, Monitor, OfficeBuilding, PriceTag, Setting, Timer, User } from '@element-plus/icons-vue'
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import SvgIcon from '@/components/SvgIcon.vue'

const router = useRouter()
const route = useRoute()

// 获取所有路由并过滤出需要显示的菜单
const allRoutes = router.options.routes
const menuItems = computed(() => {
  return allRoutes
    .filter(r => !r.meta?.hideMenu && r.meta?.requiresAuth)
    .sort((a, b) => (a.meta?.order as number || 999) - (b.meta?.order as number || 999))
})

const activeRoute = computed(() => {
  // 获取当前激活的顶级路由
  const path = route.path
  if (path.startsWith('/workspace'))
    return '/workspace'
  if (path.startsWith('/management'))
    return '/management'
  return path
})

// 获取当前活动路由的子菜单
const activeSubMenus = computed(() => {
  const currentRoute = allRoutes.find(r => r.path === activeRoute.value)
  if (currentRoute?.children) {
    return currentRoute.children.filter(child => !child.meta?.hideMenu)
  }
  return []
})

// 判断是否显示二级导航
const showSubNav = computed(() => {
  return activeSubMenus.value.length > 0
})

function handleMenuClick(path: string) {
  router.push(path)
}

function handleSubMenuClick(parentPath: string, childPath: string) {
  const fullPath = `${parentPath}/${childPath}`.replace(/\/+/g, '/')
  router.push(fullPath)
}

// 图标映射
const iconMap: Record<string, any> = {
  'calendar': Calendar,
  'setting': Setting,
  'user': User,
  'clock': Clock,
  'collection': Collection,
  'collection-tag': PriceTag,
  'document': Document,
  'office-building': OfficeBuilding,
  'timer': Timer,
  'list': List,
  'data-analysis': DataAnalysis,
  'monitor': Monitor,
}
</script>

<template>
  <div v-if="route?.meta.noLayout" class="no-layout-wrapper">
    <router-view v-slot="{ Component }">
      <component :is="Component" />
    </router-view>
  </div>
  <div v-else class="modern-layout">
    <!-- 顶部导航栏 -->
    <header class="layout-header">
      <div class="header-content">
        <!-- Logo 区域 -->
        <div class="logo-section">
          <div class="logo-icon">
            <SvgIcon name="calendar" size="28px" />
          </div>
          <h1 class="logo-text">
            智能排班
          </h1>
        </div>

        <!-- 导航菜单 -->
        <nav class="nav-menu">
          <div
            v-for="menu in menuItems"
            :key="menu.path"
            class="nav-item"
            :class="{ active: activeRoute === menu.path }"
            @click="handleMenuClick(menu.path as string)"
          >
            <el-icon class="nav-icon">
              <component :is="iconMap[menu.meta?.icon as string]" />
            </el-icon>
            <span class="nav-label">{{ (menu.meta?.title as any)?.() || menu.meta?.title }}</span>
          </div>
        </nav>

        <!-- 用户操作区 -->
        <div class="user-section">
          <el-dropdown>
            <div class="user-avatar">
              <el-icon><User /></el-icon>
            </div>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item>
                  个人设置
                </el-dropdown-item>
                <el-dropdown-item divided>
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
          v-for="subMenu in activeSubMenus"
          :key="subMenu.path"
          class="sub-nav-item"
          :class="{ active: route.path === `${activeRoute}/${subMenu.path}`, 'has-divider': subMenu.meta?.hasDivider }"
          @click="handleSubMenuClick(activeRoute, subMenu.path as string)"
        >
          <el-icon v-if="subMenu.meta?.icon" class="sub-nav-icon">
            <component :is="iconMap[subMenu.meta.icon as string]" />
          </el-icon>
          <span class="sub-nav-label">{{ (subMenu.meta?.title as any)?.() || subMenu.meta?.title }}</span>
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
.no-layout-wrapper {
  height: 100vh;
  width: 100vw;
}

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

/* Logo 区域 */
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

/* 导航菜单 */
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
  height: 100%;
  max-height: 56px;
  border-radius: var(--radius-lg);
  cursor: pointer;
  transition: all var(--transition-fast);
  color: var(--text-regular);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  position: relative;
  white-space: nowrap;

  .nav-icon {
    width: 22px;
    height: 22px;
    font-size: 22px;
    flex-shrink: 0;
    line-height: 1;
    display: flex;
    align-items: center;
    justify-content: center;

    svg {
      width: 22px;
      height: 22px;
    }
  }

  .nav-label {
    font-size: 13px;
    line-height: 1.2;
    font-weight: var(--font-weight-medium);
  }

  &:hover {
    background: var(--bg-hover);
    color: var(--primary-color);

    .nav-icon {
      transform: translateY(-1px);
    }
  }

  &.active {
    background: var(--primary-light);
    color: var(--primary-color);
    font-weight: var(--font-weight-semibold);

    .nav-icon {
      color: var(--primary-color);
    }

    // &::after {
    //   content: '';
    //   position: absolute;
    //   bottom: 0;
    //   left: 50%;
    //   transform: translateX(-50%);
    //   width: 40px;
    //   height: 3px;
    //   background: var(--primary-color);
    //   border-radius: var(--radius-base) var(--radius-base) 0 0;
    // }
  }
}

/* 二级导航栏 */
.sub-nav {
  background: var(--bg-white);
  border-bottom: 1px solid var(--border-light);
  box-shadow: inset 0 -1px 0 0 var(--border-lighter);
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
  overflow-y: hidden;

  &::-webkit-scrollbar {
    height: 4px;
  }

  &::-webkit-scrollbar-thumb {
    background: var(--border-base);
    border-radius: var(--radius-base);
  }
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
  font-weight: var(--font-weight-regular);
  white-space: nowrap;
  position: relative;

  // 分隔线样式
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

    svg {
      width: 16px;
      height: 16px;
    }
  }

  .sub-nav-label {
    line-height: 1.5;
  }

  &:hover {
    background: var(--bg-hover);
    color: var(--primary-color);
  }

  &.active {
    color: var(--primary-color);
    font-weight: var(--font-weight-semibold);
    background: var(--primary-light);

    &::after {
      content: '';
      position: absolute;
      bottom: 0;
      left: 0;
      right: 0;
      height: 2px;
      background: var(--primary-color);
    }
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
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: var(--radius-round);
  background: var(--bg-hover);
  color: var(--text-regular);
  cursor: pointer;
  transition: all var(--transition-fast);

  &:hover {
    background: var(--border-light);
    transform: scale(1.05);
  }

  .el-icon {
    font-size: 20px;
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

/* 响应式设计 */
@media (max-width: 768px) {
  .header-content {
    padding: 0 var(--spacing-md);
  }

  .logo-section {
    margin-right: var(--spacing-lg);

    .logo-text {
      font-size: var(--font-size-base);
    }
  }

  .nav-menu {
    gap: var(--spacing-xs);
  }

  .nav-item {
    padding: 4px var(--spacing-sm);
    min-width: 60px;
    max-height: 48px;
    gap: 2px;

    .nav-icon {
      width: 16px;
      height: 16px;
      font-size: 16px;

      svg {
        width: 16px;
        height: 16px;
      }
    }

    .nav-label {
      font-size: 11px;
    }
  }
}

@media (max-width: 480px) {
  .logo-text {
    display: none;
  }

  .nav-item {
    padding: 4px 8px;
    min-width: 50px;

    .nav-icon {
      width: 14px;
      height: 14px;
      font-size: 14px;

      svg {
        width: 14px;
        height: 14px;
      }
    }

    .nav-label {
      font-size: 10px;
    }
  }
}
</style>
