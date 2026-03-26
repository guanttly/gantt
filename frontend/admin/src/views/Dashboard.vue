<script setup lang="ts">
import { computed } from 'vue'
import { onMounted, ref } from 'vue'
import { NEmpty, NSpin, useMessage } from 'naive-ui'
import { getAdminDashboard } from '@/api/admin'
import type { AdminDashboard } from '@/api/admin'

const loading = ref(true)
const data = ref<AdminDashboard | null>(null)
const message = useMessage()

const statCards = computed(() => {
  if (!data.value) return []

  return [
    { label: '机构总数', value: data.value.total_orgs, tone: 'primary' },
    { label: '活跃机构', value: data.value.active_orgs, tone: 'success' },
    { label: '用户总数', value: data.value.total_users, tone: 'neutral' },
    { label: '近 30 天活跃用户', value: data.value.active_users_30d, tone: 'accent' },
    { label: '近 30 天生成排班', value: data.value.schedules_generated_30d, tone: 'warning' },
  ]
})

const subscriptionItems = computed(() => {
  if (!data.value) return []

  const labels: Record<string, string> = {
    free: '免费版',
    standard: '标准版',
    premium: '高级版',
  }

  return Object.entries(data.value.subscription_breakdown || {}).map(([plan, count]) => ({
    plan,
    label: labels[plan] || plan,
    count,
  }))
})

async function loadData() {
  loading.value = true
  try {
    data.value = await getAdminDashboard()
  }
  catch {
    message.error('加载看板数据失败')
  }
  finally {
    loading.value = false
  }
}

onMounted(loadData)
</script>

<template>
  <div class="page-shell">
    <div class="page-container">
      <section class="page-header">
        <div>
          <h2 class="page-title">运营看板</h2>
          <p class="page-subtitle">查看平台租户活跃度、用户规模与近期排班产出。</p>
        </div>
      </section>

      <n-spin :show="loading">
        <template #description>
          正在加载运营数据
        </template>

        <template v-if="data">
          <div class="dashboard-stack">
            <section class="stats-grid">
              <article v-for="item in statCards" :key="item.label" class="stat-card" :data-tone="item.tone">
                <div class="stat-label">{{ item.label }}</div>
                <div class="stat-value">{{ item.value }}</div>
              </article>
            </section>

            <section class="page-card subscription-card">
              <div class="page-card-inner subscription-panel">
                <div>
                  <h3 class="panel-title">订阅分布</h3>
                  <p class="panel-copy">以后端真实统计为准，帮助判断当前平台套餐结构。</p>
                </div>
                <div v-if="subscriptionItems.length" class="subscription-list">
                  <div v-for="item in subscriptionItems" :key="item.plan" class="subscription-item">
                    <div>
                      <div class="subscription-label">{{ item.label }}</div>
                      <div class="subscription-key">{{ item.plan }}</div>
                    </div>
                    <div class="subscription-count">{{ item.count }}</div>
                  </div>
                </div>
                <n-empty v-else description="暂无订阅统计数据" />
              </div>
            </section>
          </div>
        </template>
      </n-spin>
    </div>
  </div>
</template>

<style scoped>
.dashboard-stack {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 20px;
}

.stat-card {
  min-height: 116px;
  border: 1px solid var(--admin-border);
  border-radius: 18px;
  padding: 22px 22px 24px;
  background: rgba(255, 255, 255, 0.92);
  box-shadow: 0 14px 30px rgba(15, 23, 42, 0.04);
}
.stat-card[data-tone='primary'] {
  background: linear-gradient(135deg, rgba(15, 118, 110, 0.12), rgba(255, 255, 255, 0.96));
}
.stat-card[data-tone='success'] {
  background: linear-gradient(135deg, rgba(22, 163, 74, 0.12), rgba(255, 255, 255, 0.96));
}
.stat-card[data-tone='accent'] {
  background: linear-gradient(135deg, rgba(37, 99, 235, 0.1), rgba(255, 255, 255, 0.96));
}
.stat-card[data-tone='warning'] {
  background: linear-gradient(135deg, rgba(245, 158, 11, 0.15), rgba(255, 255, 255, 0.96));
}
.stat-value {
  margin-top: 10px;
  font-size: 34px;
  font-weight: 700;
  color: #0f172a;
}
.stat-label {
  font-size: 14px;
  line-height: 1.4;
  color: var(--admin-text-muted);
}

.subscription-card {
  overflow: hidden;
}

.subscription-panel {
  display: flex;
  flex-direction: column;
  gap: 20px;
}
.panel-title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}
.panel-copy {
  margin: 8px 0 0;
  color: var(--admin-text-muted);
  font-size: 14px;
}
.subscription-list {
  display: grid;
  gap: 14px;
}

.subscription-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 68px;
  padding: 16px 18px;
  border: 1px solid var(--admin-border);
  border-radius: 14px;
  background: var(--admin-surface-soft);
}
.subscription-label {
  font-weight: 600;
}
.subscription-key {
  margin-top: 4px;
  color: var(--admin-text-muted);
  font-size: 12px;
  text-transform: uppercase;
}
.subscription-count {
  color: var(--admin-primary);
  font-size: 28px;
  font-weight: 700;
}

@media (max-width: 768px) {
  .dashboard-stack {
    gap: 18px;
  }

  .stats-grid {
    gap: 16px;
  }

  .stat-card {
    min-height: 104px;
  }
}
</style>
