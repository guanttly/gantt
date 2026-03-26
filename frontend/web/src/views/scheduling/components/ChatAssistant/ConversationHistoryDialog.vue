<script setup lang="ts">
import { Clock } from '@element-plus/icons-vue'
import { ElButton, ElDialog, ElEmpty, ElMessage, ElTable, ElTableColumn, ElTag } from 'element-plus'
import { onMounted, ref, watch } from 'vue'

import { listConversations } from '@/api/schedules'

interface SessionRecord {
  id: string
  title: string
  createdAt: string
  updatedAt: string
  messageCount: number
  status: string
}

const props = defineProps<{
  visible: boolean
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'close': []
  'load': [conversationId: string]
}>()

const sessions = ref<SessionRecord[]>([])
const isLoading = ref(false)

async function fetchSessions() {
  isLoading.value = true
  try {
    const res = await listConversations()
    const items = Array.isArray(res) ? res : (res.items || res.data || [])
    sessions.value = items.map((item: any) => ({
      id: item.id,
      title: item.title || '未命名会话',
      createdAt: item.createdAt || item.created_at || '',
      updatedAt: item.updatedAt || item.updated_at || '',
      messageCount: item.messageCount || item.message_count || 0,
      status: item.status || 'completed',
    }))
  }
  catch (err) {
    console.error('Failed to fetch sessions:', err)
    ElMessage.error('获取会话历史失败')
  }
  finally {
    isLoading.value = false
  }
}

function formatDateTime(dateStr: string): string {
  if (!dateStr)
    return '-'
  try {
    const d = new Date(dateStr)
    return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')} ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
  }
  catch {
    return dateStr
  }
}

function getStatusTagType(status: string) {
  const map: Record<string, 'success' | 'warning' | 'info' | 'danger'> = {
    completed: 'success',
    active: 'warning',
    error: 'danger',
  }
  return map[status] || 'info'
}

function getStatusLabel(status: string): string {
  const map: Record<string, string> = {
    completed: '已完成',
    active: '进行中',
    error: '出错',
  }
  return map[status] || status
}

function handleLoadSession(session: SessionRecord) {
  emit('load', session.id)
  emit('update:visible', false)
}

watch(() => props.visible, (newVal) => {
  if (newVal)
    fetchSessions()
})

onMounted(() => {
  if (props.visible)
    fetchSessions()
})

function handleClose() {
  emit('update:visible', false)
  emit('close')
}
</script>

<template>
  <ElDialog
    :before-close="handleClose"
    :model-value="visible"
    title="会话历史"
    width="750px"
    @update:model-value="$emit('update:visible', $event)"
  >
    <div v-loading="isLoading" class="history-content">
      <ElTable
        v-if="sessions.length > 0"
        :data="sessions"
        highlight-current-row
        max-height="500"
        stripe
        style="width: 100%"
      >
        <ElTableColumn label="会话标题" min-width="200" prop="title">
          <template #default="{ row }">
            <div class="session-title">
              <el-icon :size="16" class="title-icon">
                <Clock />
              </el-icon>
              <span>{{ row.title }}</span>
            </div>
          </template>
        </ElTableColumn>
        <ElTableColumn align="center" label="消息数" prop="messageCount" width="80" />
        <ElTableColumn align="center" label="状态" width="90">
          <template #default="{ row }">
            <ElTag :type="getStatusTagType(row.status)" effect="plain" size="small">
              {{ getStatusLabel(row.status) }}
            </ElTag>
          </template>
        </ElTableColumn>
        <ElTableColumn label="创建时间" width="160">
          <template #default="{ row }">
            {{ formatDateTime(row.createdAt) }}
          </template>
        </ElTableColumn>
        <ElTableColumn align="center" label="操作" width="100">
          <template #default="{ row }">
            <ElButton link size="small" type="primary" @click="handleLoadSession(row)">
              加载
            </ElButton>
          </template>
        </ElTableColumn>
      </ElTable>
      <ElEmpty v-else :image-size="80" description="暂无历史会话" />
    </div>

    <template #footer>
      <ElButton @click="handleClose">
        关闭
      </ElButton>
    </template>
  </ElDialog>
</template>

<style lang="scss" scoped>
.history-content {
  min-height: 200px;
}

.session-title {
  display: flex;
  align-items: center;
  gap: 8px;

  .title-icon {
    color: var(--el-text-color-secondary);
    flex-shrink: 0;
  }
}
</style>
