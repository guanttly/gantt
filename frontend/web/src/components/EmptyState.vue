<script setup lang="ts">
import { Document } from '@element-plus/icons-vue'
import { computed } from 'vue'

interface Props {
  /**
   * 图标组件或图标名称
   */
  icon?: object | string
  /**
   * 图标大小
   */
  iconSize?: number
  /**
   * 主要文案
   */
  title?: string
  /**
   * 描述文案
   */
  description?: string
  /**
   * 按钮文字
   */
  buttonText?: string
  /**
   * 按钮类型
   */
  buttonType?: 'primary' | 'success' | 'warning' | 'danger' | 'info' | 'text'
  /**
   * 是否显示按钮
   */
  showButton?: boolean
  /**
   * 图片地址（可选，优先级高于图标）
   */
  image?: string
  /**
   * 图片宽度
   */
  imageWidth?: number
}

const props = withDefaults(defineProps<Props>(), {
  icon: () => Document,
  iconSize: 80,
  title: '暂无数据',
  description: '',
  buttonText: '新增数据',
  buttonType: 'primary',
  showButton: false,
  image: '',
  imageWidth: 200,
})

const emit = defineEmits<{
  (e: 'action'): void
}>()

// 图标样式
const iconStyle = computed(() => ({
  fontSize: `${props.iconSize}px`,
}))

// 图片样式
const imageStyle = computed(() => ({
  width: `${props.imageWidth}px`,
}))

// 处理按钮点击
function handleAction() {
  emit('action')
}
</script>

<template>
  <div class="empty-state">
    <!-- 图片模式 -->
    <div v-if="image" class="empty-image">
      <img :src="image" :style="imageStyle" alt="Empty">
    </div>

    <!-- 图标模式 -->
    <div v-else class="empty-icon" :style="iconStyle">
      <component :is="icon" v-if="typeof icon === 'object'" />
      <el-icon v-else>
        <component :is="icon" />
      </el-icon>
    </div>

    <!-- 文案 -->
    <div v-if="title" class="empty-title">
      {{ title }}
    </div>
    <div v-if="description" class="empty-description">
      {{ description }}
    </div>

    <!-- 操作按钮 -->
    <div v-if="showButton || $slots.action" class="empty-action">
      <slot name="action">
        <el-button :type="buttonType" @click="handleAction">
          {{ buttonText }}
        </el-button>
      </slot>
    </div>

    <!-- 自定义内容插槽 -->
    <div v-if="$slots.default" class="empty-extra">
      <slot />
    </div>
  </div>
</template>

<style lang="scss" scoped>
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  color: var(--mgmt-text-secondary, #909399);
  user-select: none;

  .empty-image {
    margin-bottom: 20px;
    opacity: 0.6;

    img {
      display: block;
      max-width: 100%;
      height: auto;
    }
  }

  .empty-icon {
    margin-bottom: 20px;
    color: var(--mgmt-text-placeholder, #c0c4cc);
    opacity: 0.6;
    transition: all 0.3s ease;

    &:hover {
      opacity: 0.8;
    }

    .el-icon {
      font-size: inherit;
    }
  }

  .empty-title {
    font-size: 16px;
    font-weight: 500;
    color: var(--mgmt-text-regular, #606266);
    margin-bottom: 8px;
    line-height: 1.5;
  }

  .empty-description {
    font-size: 13px;
    color: var(--mgmt-text-secondary, #909399);
    margin-bottom: 24px;
    line-height: 1.6;
    max-width: 400px;
    text-align: center;
  }

  .empty-action {
    margin-top: 8px;
  }

  .empty-extra {
    margin-top: 16px;
    width: 100%;
  }
}

// 动画效果
@keyframes float {
  0%, 100% {
    transform: translateY(0);
  }
  50% {
    transform: translateY(-10px);
  }
}

.empty-icon,
.empty-image {
  animation: float 3s ease-in-out infinite;
}
</style>
