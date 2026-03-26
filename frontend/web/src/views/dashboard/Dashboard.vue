<script setup lang="ts">
import { Calendar, Clock, Collection, Document, User } from '@element-plus/icons-vue'
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const router = useRouter()

const greeting = computed(() => {
  const hour = new Date().getHours()
  if (hour < 6)
    return '夜深了'
  if (hour < 12)
    return '早上好'
  if (hour < 18)
    return '下午好'
  return '晚上好'
})

const quickActions = [
  { label: '创建排班', icon: Calendar, path: '/scheduling/create', color: '#7c3aed' },
  { label: '员工管理', icon: User, path: '/employees', color: '#2563eb' },
  { label: '班次管理', icon: Clock, path: '/shifts', color: '#059669' },
  { label: '排班规则', icon: Document, path: '/rules', color: '#d97706' },
  { label: '分组管理', icon: Collection, path: '/groups', color: '#dc2626' },
]
</script>

<template>
  <div class="dashboard">
    <div class="dashboard-content">
      <!-- 欢迎区 -->
      <div class="welcome-section">
        <h2 class="welcome-text">
          {{ greeting }}，{{ auth.user?.username || '用户' }}
        </h2>
        <p class="welcome-desc">
          当前节点：{{ auth.currentNode?.node_name || '-' }} · {{ auth.currentRole }}
        </p>
      </div>

      <!-- 快捷操作 -->
      <div class="quick-actions">
        <h3 class="section-title">
          快捷操作
        </h3>
        <div class="action-grid">
          <div
            v-for="action in quickActions"
            :key="action.path"
            class="action-card"
            @click="router.push(action.path)"
          >
            <div class="action-icon" :style="{ background: `${action.color}15`, color: action.color }">
              <el-icon :size="24">
                <component :is="action.icon" />
              </el-icon>
            </div>
            <span class="action-label">{{ action.label }}</span>
          </div>
        </div>
      </div>

      <!-- 占位 — 后续可添加排班统计、待办事项等 -->
      <div class="stats-section">
        <el-empty description="仪表盘统计数据即将上线" />
      </div>
    </div>
  </div>
</template>

<style scoped>
.dashboard {
  height: 100%;
  overflow-y: auto;
  padding: 32px;
}

.dashboard-content {
  max-width: 1200px;
  margin: 0 auto;
}

.welcome-section {
  margin-bottom: 40px;
}

.welcome-text {
  font-size: 28px;
  font-weight: 700;
  color: #1a1a2e;
  margin: 0 0 8px;
}

.welcome-desc {
  font-size: 14px;
  color: #6b7280;
  margin: 0;
}

.section-title {
  font-size: 18px;
  font-weight: 600;
  color: #374151;
  margin: 0 0 20px;
}

.action-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 16px;
  margin-bottom: 40px;
}

.action-card {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 20px;
  background: #fff;
  border-radius: 12px;
  border: 1px solid #e5e7eb;
  cursor: pointer;
  transition: all 0.2s;
}

.action-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 24px rgb(0 0 0 / 8%);
  border-color: transparent;
}

.action-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  border-radius: 12px;
  flex-shrink: 0;
}

.action-label {
  font-size: 15px;
  font-weight: 500;
  color: #374151;
}

.stats-section {
  background: #fff;
  border-radius: 12px;
  border: 1px solid #e5e7eb;
  padding: 48px 24px;
}
</style>
