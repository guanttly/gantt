<script lang="ts" setup>
import { ArrowLeft } from '@element-plus/icons-vue'
import { computed, useSlots } from 'vue'

interface Props {
  title?: string
  showHeader?: boolean
  showBack?: boolean
}

withDefaults(defineProps<Props>(), {
  showHeader: true,
  showBack: false,
})

const emit = defineEmits<{
  back: []
}>()

const slots = useSlots()
const hasToolbar = computed(() => !!slots.toolbar)
const hasExtra = computed(() => !!slots.extra)
</script>

<template>
  <div class="page-container">
    <!-- 页面头部 -->
    <div v-if="showHeader && (title || hasExtra)" class="page-header">
      <div class="page-header-content">
        <div class="page-header-left">
          <el-button
            v-if="showBack"
            text
            :icon="ArrowLeft"
            @click="emit('back')"
          >
            返回
          </el-button>
          <h2 v-if="title" class="page-title">
            {{ title }}
          </h2>
        </div>
        <div v-if="hasExtra" class="page-header-extra">
          <slot name="extra" />
        </div>
      </div>
    </div>

    <!-- 工具栏区域 -->
    <div v-if="hasToolbar" class="page-toolbar">
      <slot name="toolbar" />
    </div>

    <!-- 主内容区 -->
    <div class="page-content">
      <slot />
    </div>
  </div>
</template>

<style lang="scss" scoped>
.page-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #fff;
  overflow: hidden;
}

.page-header {
  padding: 20px 24px;
  border-bottom: 1px solid #e4e7ed;
  background: #fff;

  &-content {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  &-left {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  &-extra {
    display: flex;
    align-items: center;
    gap: 12px;
  }
}

.page-title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}

.page-toolbar {
  padding: 16px 24px;
  background: #fff;
  border-bottom: 1px solid #f0f0f0;
}

.page-content {
  flex: 1;
  padding: 24px;
  overflow: auto;
  background: #f5f7fa;

  &::-webkit-scrollbar {
    width: 6px;
    height: 6px;
  }

  &::-webkit-scrollbar-thumb {
    background: #dcdfe6;
    border-radius: 3px;

    &:hover {
      background: #c0c4cc;
    }
  }

  &::-webkit-scrollbar-track {
    background: transparent;
  }
}
</style>
