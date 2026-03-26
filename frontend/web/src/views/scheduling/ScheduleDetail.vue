<script setup lang="ts">
import type { ScheduleAssignment, SchedulePlan, ScheduleSummary } from '@/api/schedules'
import { ElMessage } from 'element-plus'
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { generateSchedule, getAssignments, getSchedule, getScheduleSummary, publishSchedule, validateSchedule } from '@/api/schedules'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const scheduleId = route.params.id as string

const loading = ref(true)
const schedule = ref<SchedulePlan | null>(null)
const assignments = ref<ScheduleAssignment[]>([])
const summary = ref<ScheduleSummary | null>(null)

async function loadData() {
  loading.value = true
  try {
    const [s, a] = await Promise.all([
      getSchedule(scheduleId),
      getAssignments(scheduleId),
    ])
    schedule.value = s
    assignments.value = a

    // 如果已生成，加载统计
    if (s.status !== 'draft') {
      summary.value = await getScheduleSummary(scheduleId)
    }
  }
  catch {
    ElMessage.error('加载排班数据失败')
  }
  finally {
    loading.value = false
  }
}

async function handleGenerate() {
  try {
    await generateSchedule(scheduleId)
    ElMessage.success('排班生成请求已发送')
    await loadData()
  }
  catch (e: any) {
    ElMessage.error(e?.response?.data?.message || '生成失败')
  }
}

async function handleValidate() {
  try {
    const result = await validateSchedule(scheduleId)
    if (result.valid) {
      ElMessage.success('排班验证通过，无违规')
    }
    else {
      ElMessage.warning(`发现 ${result.violations.length} 条违规`)
    }
  }
  catch (e: any) {
    ElMessage.error(e?.response?.data?.message || '验证失败')
  }
}

async function handlePublish() {
  try {
    await publishSchedule(scheduleId)
    ElMessage.success('排班已发布')
    await loadData()
  }
  catch (e: any) {
    ElMessage.error(e?.response?.data?.message || '发布失败')
  }
}

onMounted(loadData)
</script>

<template>
  <div v-loading="loading" class="detail-page">
    <template v-if="schedule">
      <!-- 头部 -->
      <div class="detail-header">
        <div class="header-left">
          <el-button text @click="router.back()">
            ← 返回
          </el-button>
          <h2 class="schedule-name">
            {{ schedule.name }}
          </h2>
          <el-tag :type="schedule.status === 'published' ? 'success' : schedule.status === 'generated' ? '' : 'info'" size="small">
            {{ schedule.status }}
          </el-tag>
        </div>
        <div class="header-actions">
          <el-button v-if="schedule.status === 'draft' && auth.hasPermission('schedule:execute')" type="primary" @click="handleGenerate">
            生成排班
          </el-button>
          <el-button v-if="schedule.status === 'generated' && auth.hasPermission('schedule:adjust')" @click="handleValidate">
            验证
          </el-button>
          <el-button v-if="schedule.status === 'generated' && auth.hasPermission('schedule:publish')" type="success" @click="handlePublish">
            发布
          </el-button>
        </div>
      </div>

      <!-- 基本信息 -->
      <div class="info-bar">
        <span>日期范围：{{ schedule.start_date }} ~ {{ schedule.end_date }}</span>
        <span v-if="summary">共 {{ summary.total_employees }} 人 · {{ summary.total_assignments }} 班次</span>
      </div>

      <!-- 排班表格 -->
      <div class="assignment-table">
        <el-table :data="assignments" border stripe style="width: 100%">
          <el-table-column prop="employee_name" label="员工" width="120" />
          <el-table-column prop="date" label="日期" width="120" />
          <el-table-column prop="shift_name" label="班次" width="120">
            <template #default="{ row }">
              <div style="display: flex; align-items: center; gap: 6px">
                <span class="color-dot" :style="{ background: row.shift_color }" />
                {{ row.shift_name }}
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="start_time" label="开始" width="100" />
          <el-table-column prop="end_time" label="结束" width="100" />
          <el-table-column prop="status" label="状态" width="100">
            <template #default="{ row }">
              <el-tag
                :type="row.status === 'conflict' ? 'danger' : row.status === 'adjusted' ? 'warning' : 'success'"
                size="small"
              >
                {{ row.status }}
              </el-tag>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </template>
  </div>
</template>

<style scoped>
.detail-page {
  height: 100%;
  display: flex;
  flex-direction: column;
  padding: 24px;
  overflow-y: auto;
}

.detail-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.schedule-name {
  font-size: 20px;
  font-weight: 600;
  margin: 0;
}

.info-bar {
  display: flex;
  gap: 24px;
  padding: 12px 16px;
  background: #f5f7fa;
  border-radius: 8px;
  font-size: 14px;
  color: #6b7280;
  margin-bottom: 16px;
}

.assignment-table {
  flex: 1;
  overflow: hidden;
}

.color-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  flex-shrink: 0;
}
</style>
