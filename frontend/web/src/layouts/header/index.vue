<script lang="ts" setup>
import { ref } from 'vue'
import { useUserStore } from '@/store/user'
import PasswordLayer from './passwordLayer.vue'

const userStore = useUserStore()

// login out the system
function loginOut() {
  userStore.logout()
}
const passwordLayer = ref<InstanceType<typeof PasswordLayer>>()
</script>

<template>
  <header>
    <div class="left-box" />
    <div class="right-box">
      <!-- 用户信息 -->
      <div class="user-info">
        <el-dropdown>
          <span class="el-dropdown-link">
            admin
            <i class="iconfont icon-xiala" />
          </span>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item @click="passwordLayer?.showFunc">
                修改密码
              </el-dropdown-item>
              <el-dropdown-item @click="loginOut">
                登出
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
      <PasswordLayer ref="passwordLayer" />
    </div>
  </header>
</template>

<style lang="scss" scoped>
header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  height: 60px;
  background-color: var(--system-header-background);
  padding-right: 22px;
}

.left-box {
  height: 100%;
  display: flex;
  align-items: center;

  .menu-icon {
    width: 60px;
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 25px;
    font-weight: 100;
    cursor: pointer;
    margin-right: 10px;

    &:hover {
      background-color: var(--system-header-item-hover-color);
    }

    i {
      color: var(--system-header-text-color);
    }
  }
}

.right-box {
  display: flex;
  justify-content: center;
  align-items: center;

  .function-list {
    display: flex;

    .function-list-item {
      width: 30px;
      display: flex;
      justify-content: center;
      align-items: center;

      :deep(i) {
        color: var(--system-header-text-color);
      }
    }
  }

  .user-info {
    margin-left: 20px;

    .el-dropdown-link {
      color: var(--system-header-breadcrumb-text-color);
    }
  }
}

.head-fold {
  font-size: 20px;
}
</style>
