<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  /**
   * 骨架屏行数
   */
  rows?: number
  /**
   * 骨架屏列数
   */
  columns?: number
  /**
   * 是否显示表头
   */
  showHeader?: boolean
  /**
   * 动画速度（秒）
   */
  animationSpeed?: number
  /**
   * 行高度
   */
  rowHeight?: number
}

const props = withDefaults(defineProps<Props>(), {
  rows: 5,
  columns: 5,
  showHeader: true,
  animationSpeed: 1.5,
  rowHeight: 50,
})

// 生成行数组
const rowArray = computed(() => Array.from({ length: props.rows }, (_, i) => i))

// 生成列数组
const columnArray = computed(() => Array.from({ length: props.columns }, (_, i) => i))

// 动画样式
const animationStyle = computed(() => ({
  animationDuration: `${props.animationSpeed}s`,
}))

// 行样式
const rowStyle = computed(() => ({
  height: `${props.rowHeight}px`,
}))
</script>

<template>
  <div class="table-skeleton">
    <!-- 表头骨架 -->
    <div v-if="showHeader" class="skeleton-header" :style="rowStyle">
      <div
        v-for="col in columnArray"
        :key="`header-${col}`"
        class="skeleton-cell skeleton-header-cell"
        :style="animationStyle"
      />
    </div>

    <!-- 表体骨架 -->
    <div class="skeleton-body">
      <div
        v-for="row in rowArray"
        :key="`row-${row}`"
        class="skeleton-row"
        :style="rowStyle"
      >
        <div
          v-for="col in columnArray"
          :key="`cell-${row}-${col}`"
          class="skeleton-cell"
          :style="animationStyle"
        >
          <div class="skeleton-bar" />
        </div>
      </div>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.table-skeleton {
  width: 100%;
  background-color: #fff;
  border-radius: 8px;
  overflow: hidden;

  .skeleton-header {
    display: flex;
    align-items: center;
    padding: 0 16px;
    border-bottom: 2px solid var(--mgmt-border-light, #e4e7ed);
    background-color: var(--mgmt-bg-page, #f5f7fa);

    .skeleton-header-cell {
      height: 18px;
    }
  }

  .skeleton-body {
    .skeleton-row {
      display: flex;
      align-items: center;
      padding: 0 16px;
      border-bottom: 1px solid var(--mgmt-border-lighter, #ebeef5);

      &:last-child {
        border-bottom: none;
      }

      &:hover {
        background-color: var(--mgmt-bg-page, #f5f7fa);
      }
    }
  }

  .skeleton-cell {
    flex: 1;
    padding: 0 8px;
    min-width: 0;

    .skeleton-bar {
      height: 14px;
      background: linear-gradient(
        90deg,
        var(--mgmt-border-extra-light, #f2f6fc) 25%,
        var(--mgmt-border-lighter, #ebeef5) 50%,
        var(--mgmt-border-extra-light, #f2f6fc) 75%
      );
      background-size: 200% 100%;
      animation: skeleton-loading 1.5s ease-in-out infinite;
      border-radius: 4px;

      // 随机宽度效果
      &:nth-child(odd) {
        width: 80%;
      }
    }

    // 第一列稍宽一些
    &:first-child {
      flex: 1.5;
    }

    // 最后一列（操作列）窄一些
    &:last-child {
      flex: 0.8;
    }
  }
}

@keyframes skeleton-loading {
  0% {
    background-position: 200% 0;
  }
  100% {
    background-position: -200% 0;
  }
}

// 暗色模式支持
@media (prefers-color-scheme: dark) {
  .table-skeleton {
    background-color: #1a1a1a;

    .skeleton-header {
      background-color: #252525;
      border-bottom-color: #3a3a3a;
    }

    .skeleton-body .skeleton-row {
      border-bottom-color: #2a2a2a;

      &:hover {
        background-color: #252525;
      }
    }

    .skeleton-cell .skeleton-bar {
      background: linear-gradient(
        90deg,
        #2a2a2a 25%,
        #353535 50%,
        #2a2a2a 75%
      );
    }
  }
}
</style>
