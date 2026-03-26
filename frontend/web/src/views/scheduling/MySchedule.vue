<script setup lang="ts">
import type { MyScheduleAssignment } from '@/api/schedules'
import { Calendar, Refresh } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { computed, onMounted, ref } from 'vue'
import { getMySchedule } from '@/api/schedules'

function formatDateLocal(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

const today = new Date()
const dayOfWeek = today.getDay()
const daysFromMonday = dayOfWeek === 0 ? -6 : 1 - dayOfWeek
const startOfWeek = new Date(today)
startOfWeek.setDate(today.getDate() + daysFromMonday)
const endOfWeek = new Date(startOfWeek)
endOfWeek.setDate(startOfWeek.getDate() + 6)

const loading = ref(false)
const dateRange = ref<[string, string]>([
  formatDateLocal(startOfWeek),
  formatDateLocal(endOfWeek),
])
const assignments = ref<MyScheduleAssignment[]>([])

const groupedAssignments = computed(() => {
  const groups = new Map<string, MyScheduleAssignment[]>()
  for (const item of assignments.value) {
    const existing = groups.get(item.date) || []
    existing.push(item)
    groups.set(item.date, existing)
  }
  return Array.from(groups.entries()).map(([date, items]) => ({
    date,
    items,
  }))
})

async function loadData() {
  loading.value = true
  try {
    assignments.value = await getMySchedule({
      start_date: dateRange.value[0],
      end_date: dateRange.value[1],
    })
  }
  catch (e: any) {
    ElMessage.error(e?.response?.data?.message || '加载个人排班失败')
  }
  finally {
    loading.value = false
  }
}

function handleThisWeek() {
  dateRange.value = [formatDateLocal(startOfWeek), formatDateLocal(endOfWeek)]
  loadData()
}

onMounted(loadData)
</script>

<template>
  <div class="page-container">
    <div class="page-toolbar">
      <div class="toolbar-left">
        <el-date-picker
          v-model="dateRange"
          type="daterange"
          range-separator="至"
          start-placeholder="开始日期"
          end-placeholder="结束日期"
          value-format="YYYY-MM-DD"
          style="width: 280px"
          @change="loadData"
        />
        <el-button @click="handleThisWeek">本周</el-button>
      </div>
      <el-button :icon="Refresh" @click="loadData">刷新</el-button>
    </div>

    <div v-loading="loading" class="schedule-content">
      <div v-if="groupedAssignments.length" class="schedule-grid">
        <section v-for="group in groupedAssignments" :key="group.date" class="day-card">
          <header class="day-header">
            <div class="day-date">{{ group.date }}</div>
            <div class="day-count">{{ group.items.length }} 个班次</div>
          </header>

          <div class="shift-list">
            <article v-for="item in group.items" :key="item.id" class="shift-item">
              <div class="shift-accent" :style="{ background: item.shift_color || '#409EFF' }" />
              <div class="shift-main">
                <div class="shift-title-row">
                  <strong>{{ item.shift_name }}</strong>
                  <span class="shift-time">{{ item.start_time }} - {{ item.end_time }}</span>
                </div>
                <div class="shift-meta">
                  <span>{{ item.schedule_name }}</span>
                  <span>{{ item.status === 'published' ? '已发布' : item.status }}</span>
                </div>
              </div>
            </article>
          </div>
        </section>
      </div>

      <el-empty v-else description="当前时间范围内暂无已发布排班">
        <template #image>
          <el-icon :size="52" color="#94a3b8"><Calendar /></el-icon>
        </template>
      </el-empty>
    </div>
  </div>
</template>

<style scoped>
.page-container {
  height: 100%;
  display: flex;
  flex-direction: column;
  padding: 24px;
  overflow-y: auto;
}

.page-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  gap: 12px;
}

.toolbar-left {
  display: flex;
  gap: 8px;
  align-items: center;
}

.schedule-content {
  flex: 1;
}

.schedule-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
}

.day-card {
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 16px;
  overflow: hidden;
  box-shadow: 0 10px 30px rgb(15 23 42 / 6%);
}

.day-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 14px 16px;
  border-bottom: 1px solid #e2e8f0;
  background: linear-gradient(180deg, #f8fafc 0%, #fff 100%);
}

.day-date {
  font-weight: 700;
  color: #0f172a;
}

.day-count {
  font-size: 12px;
  color: #64748b;
}

.shift-list {
  display: flex;
  flex-direction: column;
}

.shift-item {
  display: flex;
  gap: 12px;
  padding: 14px 16px;
  border-bottom: 1px solid #f1f5f9;
}

.shift-item:last-child {
  border-bottom: none;
}

.shift-accent {
  width: 6px;
  border-radius: 999px;
  flex-shrink: 0;
}

.shift-main {
  flex: 1;
  min-width: 0;
}

.shift-title-row {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  color: #0f172a;
}

.shift-time,
.shift-meta {
  font-size: 12px;
  color: #64748b;
}

.shift-meta {
  display: flex;
  justify-content: space-between;
  margin-top: 6px;
  gap: 12px;
}

@media (max-width: 720px) {
  .page-toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .toolbar-left {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>