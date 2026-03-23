<template>
  <div class="rule-statistics-card">
    <el-card shadow="hover">
      <template #header>
        <div class="card-header">
          <span>规则统计</span>
          <el-button
            text
            :icon="Refresh"
            @click="handleRefresh"
          >
            刷新
          </el-button>
        </div>
      </template>

      <div class="statistics-grid">
        <div class="stat-item">
          <div class="stat-label">总规则数</div>
          <div class="stat-value">{{ statistics.total || 0 }}</div>
        </div>

        <div class="stat-item constraint">
          <div class="stat-label">约束规则</div>
          <div class="stat-value">{{ statistics.constraint || 0 }}</div>
        </div>

        <div class="stat-item preference">
          <div class="stat-label">偏好规则</div>
          <div class="stat-value">{{ statistics.preference || 0 }}</div>
        </div>

        <div class="stat-item dependency">
          <div class="stat-label">依赖规则</div>
          <div class="stat-value">{{ statistics.dependency || 0 }}</div>
        </div>

        <div class="stat-item v3">
          <div class="stat-label">V3 规则</div>
          <div class="stat-value">{{ statistics.v3 || 0 }}</div>
        </div>

        <div class="stat-item v4">
          <div class="stat-label">V4 规则</div>
          <div class="stat-value">{{ statistics.v4 || 0 }}</div>
        </div>

        <div class="stat-item active">
          <div class="stat-label">启用规则</div>
          <div class="stat-value">{{ statistics.active || 0 }}</div>
        </div>

        <div class="stat-item inactive">
          <div class="stat-label">禁用规则</div>
          <div class="stat-value">{{ statistics.inactive || 0 }}</div>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { Refresh } from '@element-plus/icons-vue'
import { computed } from 'vue'

interface Props {
  statistics: {
    total?: number
    constraint?: number
    preference?: number
    dependency?: number
    v3?: number
    v4?: number
    active?: number
    inactive?: number
  }
}

const props = defineProps<Props>()

const emit = defineEmits<{
  refresh: []
}>()

function handleRefresh() {
  emit('refresh')
}
</script>

<style scoped lang="scss">
.rule-statistics-card {
  margin-bottom: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.statistics-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
}

.stat-item {
  padding: 16px;
  border-radius: 8px;
  background: #f5f7fa;
  text-align: center;
  transition: all 0.3s;

  &:hover {
    transform: translateY(-2px);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  }

  &.constraint {
    background: #fef0f0;
    border-left: 4px solid #f56c6c;
  }

  &.preference {
    background: #f0f9ff;
    border-left: 4px solid #409eff;
  }

  &.dependency {
    background: #fef9e7;
    border-left: 4px solid #e6a23c;
  }

  &.v3 {
    background: #f4f4f5;
    border-left: 4px solid #909399;
  }

  &.v4 {
    background: #f0f9ff;
    border-left: 4px solid #67c23a;
  }

  &.active {
    background: #f0f9ff;
    border-left: 4px solid #67c23a;
  }

  &.inactive {
    background: #fef0f0;
    border-left: 4px solid #f56c6c;
  }
}

.stat-label {
  font-size: 14px;
  color: #606266;
  margin-bottom: 8px;
}

.stat-value {
  font-size: 24px;
  font-weight: bold;
  color: #303133;
}
</style>
