<script lang="ts" setup>
import { computed } from 'vue'
import { useUserStore } from '@/store/user'
import AppLink from './Link.vue'
import MenuTitle from './MenuTitle.vue'

const props = defineProps({
  menu: {
    type: Object,
    required: true,
  },
  basePath: {
    type: String,
    default: '',
  },
})
const userStore = useUserStore()
const menu = props.menu
const icon = computed(() => {
  return menu.meta.icon || ''
})
const title = computed(() => {
  return menu.meta.title || menu.name
})
const isHidden = computed(() => {
  const isHide = menu.meta?.hideMenu || false
  const fullPath = menu.meta?.fullPath || ''
  const authed = userStore.menuList.some(menuItem => menuItem.path === fullPath) // 判断to.path是否在用户拥有的菜单权限列表中
  return isHide || !authed
})
// todo: 优化if结构
const showMenuType = computed(() => { // 0: 无子菜单， 1：有1个子菜单， 2：显示上下级子菜单
  if (menu.children && (menu.children.length > 1 || (menu.children.length === 1 && menu.alwayShow)))
    return 2
  else if (menu.children && menu.children.length === 1 && !menu.alwayShow)
    return 1
  else
    return 0
})
const pathResolve = computed(() => {
  return menu.meta.fullPath
})
</script>

<template>
  <template v-if="!isHidden && menu.path !== '/:all(.*)*'">
    <el-sub-menu v-if="showMenuType === 2" :index="pathResolve" :show-timeout="0" :hide-timeout="0">
      <template #title>
        <MenuTitle :title="title" :icon="icon" />
      </template>
      <menu-item v-for="(item, key) in menu.children" :key="key" :menu="item" :base-path="pathResolve" />
    </el-sub-menu>
    <AppLink v-else-if="showMenuType === 1" :to="pathResolve">
      <el-menu-item v-if="!menu.children[0].children || menu.children[0].children.length === 0" :index="menu.children[0].meta.fullPath">
        <template #title>
          <MenuTitle :title="title" :icon="icon" />
        </template>
      </el-menu-item>
      <el-sub-menu v-else :index="menu.children[0].meta.fullPath" :show-timeout="0" :hide-timeout="0">
        <template #title>
          <MenuTitle :title="title" :icon="icon" />
        </template>
        <menu-item v-for="(item, key) in menu.children[0].children" :key="key" :menu="item" :base-path="pathResolve" />
      </el-sub-menu>
    </AppLink>
    <AppLink v-else :to="pathResolve">
      <el-menu-item :index="pathResolve">
        <template #title>
          <MenuTitle :title="title" :icon="icon" />
        </template>
      </el-menu-item>
    </AppLink>
  </template>
</template>

<style lang="scss" scoped>
.el-menu-item i,
.el-sub-menu__title i {
  padding-right: 8px;
}

.custom-icon-class{
  font-size: 20px;
  padding-right: 10px;
}

// 覆盖激活菜单项的文字颜色
:deep(.el-menu-item.is-active) {
  color: #409eff; // 您可以替换为您想要的颜色值
  background-color: var(--el-menu-hover-bg-color); // 可选：同时修改背景色
}

// 覆盖激活子菜单标题的文字颜色
:deep(.el-sub-menu.is-active > .el-sub-menu__title) {
  color: var(--el-menu-active-color); // Element Plus 默认激活颜色变量，如果需要不同颜色请替换
}

// 鼠标悬停时也使用稍微柔和的颜色（可选）
:deep(.el-menu-item:hover) {
  color: #66b1ff; // 稍微柔和的悬停颜色
}
:deep(.el-sub-menu__title:hover) {
  color: #66b1ff; // 稍微柔和的悬停颜色
}
</style>
