<script setup lang="ts">
import type { ConversationSummary } from '@/services/api'
import { sessionApi } from '@/services/api'
import { Refresh } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { computed, onMounted, ref, watch } from 'vue'

// 获取当前组织ID和用户ID（从 localStorage 或 store）
function getCurrentOrgId(): string {
  return localStorage.getItem('orgId') || 'default-org'
}

function getCurrentUserId(): string {
  return localStorage.getItem('userId') || 'default-user'
}

const props = defineProps<{
  visible: boolean
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'load': [conversationId: string]
}>()

const loading = ref(false)
const conversations = ref<ConversationSummary[]>([])

const visible = computed({
  get: () => props.visible,
  set: (value) => emit('update:visible', value),
})

// 加载对话历史列表
async function loadConversations() {
  loading.value = true
  try {
    const orgId = getCurrentOrgId()
    const userId = getCurrentUserId()
    const list = await sessionApi.listConversations(orgId, userId, 50)
    conversations.value = list
  }
  catch (error) {
    console.error('加载对话历史失败:', error)
    ElMessage.error('加载对话历史失败')
  }
  finally {
    loading.value = false
  }
}

// 格式化时间
function formatTime(timeStr: string) {
  if (!timeStr) return '-'
  const date = new Date(timeStr)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const days = Math.floor(diff / (1000 * 60 * 60 * 24))

  if (days === 0) {
    return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
  }
  else if (days === 1) {
    return '昨天'
  }
  else if (days < 7) {
    return `${days}天前`
  }
  else {
    return date.toLocaleDateString('zh-CN', { month: '2-digit', day: '2-digit' })
  }
}

// 获取对话标题
function getConversationTitle(conv: ConversationSummary) {
  if (conv.title && conv.title !== '新会话') {
    return conv.title
  }
  if (conv.workflowType) {
    const workflowNames: Record<string, string> = {
      'schedule.create': '创建排班',
      'schedule.adjust': '调整排班',
    }
    return workflowNames[conv.workflowType] || conv.workflowType
  }
  if (conv.scheduleStartDate && conv.scheduleEndDate) {
    return `${conv.scheduleStartDate} 至 ${conv.scheduleEndDate}`
  }
  return '新会话'
}

// 获取状态文本
function getStatusText(conv: ConversationSummary) {
  if (conv.scheduleStatus) {
    const statusMap: Record<string, string> = {
      completed: '已完成',
      failed: '失败',
      in_progress: '进行中',
    }
    return statusMap[conv.scheduleStatus] || conv.scheduleStatus
  }
  return '-'
}

// 加载对话
function handleLoad(conversationId: string) {
  emit('load', conversationId)
  visible.value = false
}

// 对话框打开时加载列表
function handleOpen() {
  loadConversations()
}

// 监听 visible 变化
watch(() => props.visible, (newVal) => {
  if (newVal) {
    handleOpen()
  }
})

onMounted(() => {
  if (props.visible) {
    handleOpen()
  }
})
</script>

<template>
  <el-dialog
    v-model="visible"
    title="历史对话"
    width="800px"
    :close-on-click-modal="false"
  >
    <div class="conversation-history-dialog">
      <!-- 工具栏 -->
      <div class="toolbar">
        <el-button :icon="Refresh" text @click="loadConversations">
          刷新
        </el-button>
      </div>

      <!-- 对话列表 -->
      <div v-loading="loading" class="conversation-list">
        <el-empty
          v-if="!loading && conversations.length === 0"
          description="暂无历史对话"
        />

        <el-table
          v-else
          :data="conversations"
          stripe
          style="width: 100%"
        >
          <el-table-column prop="title" label="标题" min-width="200">
            <template #default="{ row }">
              <div class="conversation-title">
                {{ getConversationTitle(row) }}
              </div>
            </template>
          </el-table-column>

          <el-table-column prop="lastMessageAt" label="时间" width="120">
            <template #default="{ row }">
              {{ formatTime(row.lastMessageAt) }}
            </template>
          </el-table-column>

          <el-table-column prop="messageCount" label="消息数" width="80" align="center" />

          <el-table-column label="排班周期" width="180">
            <template #default="{ row }">
              <div v-if="row.scheduleStartDate && row.scheduleEndDate" class="schedule-range">
                {{ row.scheduleStartDate }} 至 {{ row.scheduleEndDate }}
              </div>
              <span v-else class="text-placeholder">-</span>
            </template>
          </el-table-column>

          <el-table-column prop="scheduleStatus" label="状态" width="100" align="center">
            <template #default="{ row }">
              <el-tag v-if="row.scheduleStatus" :type="row.scheduleStatus === 'completed' ? 'success' : row.scheduleStatus === 'failed' ? 'danger' : 'info'" size="small">
                {{ getStatusText(row) }}
              </el-tag>
              <span v-else class="text-placeholder">-</span>
            </template>
          </el-table-column>

          <el-table-column label="操作" width="100" align="center" fixed="right">
            <template #default="{ row }">
              <el-button type="primary" size="small" @click="handleLoad(row.conversationId || row.id)">
                加载
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>

    <template #footer>
      <el-button @click="visible = false">
        关闭
      </el-button>
    </template>
  </el-dialog>
</template>

<style lang="scss" scoped>
.conversation-history-dialog {
  .toolbar {
    display: flex;
    justify-content: flex-end;
    margin-bottom: 16px;
  }

  .conversation-list {
    min-height: 300px;
    max-height: 500px;
    overflow-y: auto;

    .conversation-title {
      font-weight: 500;
      color: var(--el-text-color-primary);
    }

    .schedule-range {
      font-size: 12px;
      color: var(--el-text-color-regular);
    }

    .text-placeholder {
      color: var(--el-text-color-placeholder);
      font-size: 12px;
    }
  }
}
</style>
