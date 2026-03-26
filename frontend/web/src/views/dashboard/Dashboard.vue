<script setup lang="ts">
import type { Leave } from '@/api/leaves'
import type { MyScheduleAssignment } from '@/api/schedules'
import { Calendar, ChatDotRound, User } from '@element-plus/icons-vue'
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { listLeaves } from '@/api/leaves'
import { getMySchedule } from '@/api/schedules'
import { useAuthStore } from '@/stores/auth'

interface QuickAction {
  label: string
  icon: typeof Calendar
  path: string
  color: string
}

interface OverviewCard {
  label: string
  value: number
  helper: string
  color: string
}

interface PermissionSummaryItem {
  label: string
  active: boolean
}

const auth = useAuthStore()
const router = useRouter()
const dashboardLoading = ref(false)
const upcomingAssignments = ref<MyScheduleAssignment[]>([])
const recentLeaves = ref<Leave[]>([])

function formatDateLocal(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function formatDisplayDate(dateText: string): string {
  const date = new Date(`${dateText}T00:00:00`)
  return `${date.getMonth() + 1}月${date.getDate()}日`
}

function formatLeavePeriod(item: Leave): string {
  if (item.start_date === item.end_date) {
    return item.start_date
  }
  return `${item.start_date} 至 ${item.end_date}`
}

function getLeaveStatusMeta(status: Leave['status']) {
  const map = {
    pending: { label: '待审批', type: 'warning' },
    approved: { label: '已通过', type: 'success' },
    rejected: { label: '已驳回', type: 'danger' },
  } as const
  return map[status] || { label: status, type: 'info' }
}

const canViewMySchedule = computed(() => auth.hasPermission('schedule:view:self'))
const canViewLeaves = computed(() => auth.hasAnyPermission(['leave:create:self', 'leave:view:node', 'leave:approve']))

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

const quickActions = computed<QuickAction[]>(() => [
  canViewMySchedule.value
    ? { label: '我的排班', icon: Calendar, path: '/scheduling/mine', color: '#0f766e' }
    : null,
  auth.hasPermission('schedule:create')
    ? { label: '创建排班', icon: Calendar, path: '/scheduling/create', color: '#7c3aed' }
    : null,
  { label: '请假管理', icon: User, path: '/leaves', color: '#2563eb' },
  auth.hasPermission('schedule:execute')
    ? { label: '排班工作台', icon: Calendar, path: '/scheduling/workspace', color: '#0f766e' }
    : null,
  { label: 'AI 助手', icon: ChatDotRound, path: '/ai/chat', color: '#059669' },
].filter((item): item is QuickAction => item !== null))

const overviewCards = computed<OverviewCard[]>(() => [
  canViewMySchedule.value
    ? { label: '未来 7 天班次', value: upcomingAssignments.value.length, helper: '仅显示已发布排班', color: '#0f766e' }
    : null,
  canViewLeaves.value
    ? { label: '待处理请假', value: recentLeaves.value.filter(item => item.status === 'pending').length, helper: '最近请假记录', color: '#d97706' }
    : null,
  { label: '当前权限数', value: auth.appPermissions.length, helper: '已同步到当前节点', color: '#2563eb' },
].filter((item): item is OverviewCard => item !== null))

const permissionSummary = computed<PermissionSummaryItem[]>(() => [
  {
    label: '排班查看',
    active: auth.hasAnyPermission(['schedule:view:self', 'schedule:view:node', 'schedule:view:all']),
  },
  {
    label: '排班执行',
    active: auth.hasAnyPermission(['schedule:create', 'schedule:execute', 'schedule:adjust', 'schedule:publish']),
  },
  {
    label: '请假处理',
    active: auth.hasAnyPermission(['leave:create:self', 'leave:view:node', 'leave:approve']),
  },
  {
    label: '个人设置',
    active: auth.hasPermission('preference:edit:self'),
  },
])

async function loadDashboard() {
  dashboardLoading.value = true
  try {
    const start = new Date()
    const end = new Date()
    end.setDate(end.getDate() + 6)

    const [scheduleData, leaveData] = await Promise.all([
      canViewMySchedule.value
        ? getMySchedule({
            start_date: formatDateLocal(start),
            end_date: formatDateLocal(end),
          })
        : Promise.resolve([]),
      canViewLeaves.value
        ? listLeaves({
            page: 1,
            page_size: 5,
            sort_by: 'created_at',
            sort_order: 'desc',
          })
        : Promise.resolve({
            items: [],
            total: 0,
            page: 1,
            page_size: 5,
            total_pages: 0,
          }),
    ])

    upcomingAssignments.value = scheduleData.slice(0, 4)
    recentLeaves.value = leaveData.items.slice(0, 5)
  }
  finally {
    dashboardLoading.value = false
  }
}

onMounted(loadDashboard)
</script>

<template>
  <div class="dashboard">
    <div class="dashboard-content">
      <section class="welcome-section">
        <div>
          <p class="eyebrow">员工工作台</p>
          <h2 class="welcome-text">
            {{ greeting }}，{{ auth.user?.username || '用户' }}
          </h2>
          <p class="welcome-desc">
            当前节点：{{ auth.currentNode?.node_name || '-' }} · {{ auth.currentRole || '未选择角色' }}
          </p>
        </div>
        <el-button plain @click="loadDashboard">
          刷新数据
        </el-button>
      </section>

      <section class="overview-grid">
        <article v-for="card in overviewCards" :key="card.label" class="overview-card">
          <span class="overview-label">{{ card.label }}</span>
          <strong class="overview-value" :style="{ color: card.color }">{{ card.value }}</strong>
          <span class="overview-helper">{{ card.helper }}</span>
        </article>
      </section>

      <section class="quick-actions">
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
      </section>

      <section class="panel-grid">
        <article v-loading="dashboardLoading" class="panel-card">
          <div class="panel-head">
            <div>
              <h3 class="section-title">未来七天排班</h3>
              <p class="panel-desc">面向个人的已发布班次预览</p>
            </div>
            <el-button v-if="canViewMySchedule" link type="primary" @click="router.push('/scheduling/mine')">
              查看全部
            </el-button>
          </div>

          <div v-if="canViewMySchedule && upcomingAssignments.length" class="schedule-list">
            <button
              v-for="assignment in upcomingAssignments"
              :key="assignment.id"
              type="button"
              class="schedule-item"
              @click="router.push('/scheduling/mine')"
            >
              <span class="schedule-date">{{ formatDisplayDate(assignment.date) }}</span>
              <div class="schedule-main">
                <strong>{{ assignment.shift_name }}</strong>
                <span>{{ assignment.start_time }} - {{ assignment.end_time }}</span>
              </div>
              <span class="schedule-plan">{{ assignment.schedule_name }}</span>
            </button>
          </div>
          <el-empty v-else :description="canViewMySchedule ? '未来七天暂无已发布排班' : '当前账号未开通个人排班查看权限'" />
        </article>

        <article v-loading="dashboardLoading" class="panel-card">
          <div class="panel-head">
            <div>
              <h3 class="section-title">最近请假记录</h3>
              <p class="panel-desc">查看个人申请或当前节点待处理记录</p>
            </div>
            <el-button v-if="canViewLeaves" link type="primary" @click="router.push('/leaves')">
              进入请假页
            </el-button>
          </div>

          <div v-if="canViewLeaves && recentLeaves.length" class="leave-list">
            <div v-for="item in recentLeaves" :key="item.id" class="leave-item">
              <div class="leave-main">
                <strong>{{ item.employee_name || '当前员工' }} · {{ item.type }}</strong>
                <span>{{ formatLeavePeriod(item) }}</span>
              </div>
              <el-tag :type="getLeaveStatusMeta(item.status).type as any" size="small">
                {{ getLeaveStatusMeta(item.status).label }}
              </el-tag>
            </div>
          </div>
          <el-empty v-else :description="canViewLeaves ? '暂无请假记录' : '当前账号未开通请假权限'" />
        </article>
      </section>

      <section class="permission-panel">
        <div class="panel-head">
          <div>
            <h3 class="section-title">权限摘要</h3>
            <p class="panel-desc">当前节点下的应用能力同步结果</p>
          </div>
        </div>

        <div class="permission-tags">
          <div v-for="item in permissionSummary" :key="item.label" class="permission-tag" :class="{ active: item.active }">
            <span>{{ item.label }}</span>
            <strong>{{ item.active ? '已开通' : '未开通' }}</strong>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.dashboard {
  height: 100%;
  overflow-y: auto;
  padding: 32px;
  background:
    radial-gradient(circle at top left, rgb(37 99 235 / 10%), transparent 28%),
    linear-gradient(180deg, #f8fafc 0%, #eef4ff 100%);
}

.dashboard-content {
  max-width: 1200px;
  margin: 0 auto;
}

.welcome-section {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  padding: 28px 32px;
  margin-bottom: 24px;
  border-radius: 24px;
  background: linear-gradient(135deg, #0f172a 0%, #1d4ed8 100%);
  color: #fff;
  box-shadow: 0 24px 60px rgb(15 23 42 / 16%);
}

.eyebrow {
  margin: 0 0 10px;
  font-size: 12px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: rgb(191 219 254 / 92%);
}

.welcome-text {
  font-size: 28px;
  font-weight: 700;
  color: #fff;
  margin: 0 0 8px;
}

.welcome-desc {
  font-size: 14px;
  color: rgb(219 234 254 / 92%);
  margin: 0;
}

.overview-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}

.overview-card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 22px 24px;
  border-radius: 20px;
  background: rgb(255 255 255 / 88%);
  border: 1px solid rgb(226 232 240 / 90%);
  backdrop-filter: blur(10px);
}

.overview-label,
.overview-helper,
.panel-desc {
  font-size: 13px;
  color: #64748b;
}

.overview-value {
  font-size: 32px;
  line-height: 1;
}

.section-title {
  font-size: 18px;
  font-weight: 600;
  color: #374151;
  margin: 0 0 6px;
}

.action-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}

.action-card {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 20px;
  background: #fff;
  border-radius: 16px;
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

.panel-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
  margin-bottom: 16px;
}

.panel-card,
.permission-panel {
  background: rgb(255 255 255 / 92%);
  border-radius: 20px;
  border: 1px solid rgb(226 232 240 / 92%);
  padding: 24px;
  backdrop-filter: blur(10px);
}

.panel-head {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: flex-start;
  margin-bottom: 18px;
}

.schedule-list,
.leave-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.schedule-item,
.leave-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  width: 100%;
  padding: 16px;
  border-radius: 16px;
  border: 1px solid #e2e8f0;
  background: #fff;
}

.schedule-item {
  cursor: pointer;
  text-align: left;
}

.schedule-main,
.leave-main {
  display: flex;
  flex-direction: column;
  gap: 4px;
  flex: 1;
  min-width: 0;
}

.schedule-main span,
.leave-main span,
.schedule-plan,
.schedule-date {
  font-size: 13px;
  color: #64748b;
}

.schedule-date {
  min-width: 64px;
  font-weight: 600;
  color: #0f172a;
}

.schedule-plan {
  text-align: right;
}

.permission-tags {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 12px;
}

.permission-tag {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding: 16px 18px;
  border-radius: 16px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  color: #64748b;
}

.permission-tag.active {
  background: #eff6ff;
  border-color: #bfdbfe;
  color: #1d4ed8;
}

@media (max-width: 900px) {
  .dashboard {
    padding: 20px;
  }

  .panel-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 720px) {
  .welcome-section {
    flex-direction: column;
    padding: 24px;
  }

  .schedule-item,
  .leave-item,
  .panel-head {
    flex-direction: column;
    align-items: flex-start;
  }

  .schedule-plan {
    text-align: left;
  }
}
</style>
