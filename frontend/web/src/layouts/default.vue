<script lang="ts" setup>
import { useRoute } from 'vue-router'
import Header from './header/index.vue'
import Logo from './logo/index.vue'
import Menu from './menu/index.vue'

const myRoute = useRoute()
</script>

<template>
  <div v-if="myRoute?.meta.noLayout" style="height: 100vh">
    <router-view v-slot="{ Component }">
      <component :is="Component" />
    </router-view>
  </div>
  <el-container v-else style="height: 100vh">
    <el-aside
      width="250px"
      class="show-side"
    >
      <Logo />
      <Menu />
    </el-aside>
    <el-container>
      <el-header>
        <Header />
      </el-header>
      <el-main>
        <router-view v-slot="{ Component }">
          <!-- 去除自定义过度动画 -->
          <!-- :name="route.meta.transition || 'fade-transform'" -->
          <component :is="Component" />
        </router-view>
      </el-main>
    </el-container>
  </el-container>
</template>

<style lang="scss" scoped>
.el-header {
  padding-left: 0;
  padding-right: 0;
}

.el-aside {
  display: flex;
  flex-direction: column;
  transition: 0.2s;
  overflow-x: hidden;
  transition: 0.3s;

  &::-webkit-scrollbar {
    width: 0 !important;
  }
}

.el-main {
  background-color: var(--system-container-background);
  height: 100%;
  padding: 0;
  overflow-x: hidden;
}

.el-main-box {
  width: 100%;
  height: 100%;
  overflow-y: auto;
  box-sizing: border-box;
}

@media screen and (max-width: 1000px) {
  .el-aside {
    position: fixed;
    top: 0;
    left: 0;
    height: 100vh;
    z-index: 1000;

    &.hide-aside {
      left: -250px;
    }
  }

  .mask {
    position: fixed;
    top: 0;
    left: 0;
    width: 100vw;
    height: 100vh;
    z-index: 999;
    background: rgba(0, 0, 0, 0.5);
  }
}
</style>
