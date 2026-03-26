<script setup lang="ts">
import type { DataViewDialogExpose } from '../DataViewDialog/type'
import { Close, Loading } from '@element-plus/icons-vue'
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import DataViewDialog from '../DataViewDialog/index.vue'
import ActionFormDialog from './ActionFormDialog.vue'
import ChangeDetailView from './ChangeDetailView.vue'
import ConversationHistoryDialog from './ConversationHistoryDialog.vue'
import { useChatAssistant } from './logic'
import MarkdownMessage from './MarkdownMessage.vue'
import MultiShiftScheduleDialog from './MultiShiftScheduleDialog.vue'
import PersonalNeedsDialog from './PersonalNeedsDialog.vue'
import RulesDetailsDialog from './RulesDetailsDialog.vue'
import SchedulePreviewDialog from './SchedulePreviewDialog.vue'
import ShiftScheduleDialog from './ShiftScheduleDialog.vue'
import StaffDetailsDialog from './StaffDetailsDialog.vue'
import TemporaryRulesDialog from './TemporaryRulesDialog.vue'
import ValidationResultDialog from './ValidationResultDialog.vue'
import WorkflowProgress from './WorkflowProgress.vue'

const emit = defineEmits<{
  close: []
}>()

// 各对话框状态
const dataViewDialogRef = ref<DataViewDialogExpose | null>(null)
const staffDetailsVisible = ref(false)
const staffDetailsData = ref<any>(null)
const rulesDetailsVisible = ref(false)
const rulesDetailsData = ref<any>(null)
const shiftScheduleVisible = ref(false)
const shiftScheduleData = ref<any>(null)
const multiShiftScheduleVisible = ref(false)
const multiShiftScheduleData = ref<any>(null)
const personalNeedsVisible = ref(false)
const personalNeedsData = ref<any>(null)
const temporaryRulesVisible = ref(false)
const temporaryRulesData = ref<any>(null)
const validationResultVisible = ref(false)
const validationResultData = ref<any>(null)
const schedulePreviewVisible = ref(false)
const schedulePreviewData = ref<any>(null)
const changeDetailVisible = ref(false)
const changeDetailData = ref<any>(null)
const historyDialogVisible = ref(false)

// 业务逻辑 Hook
const {
  messages,
  inputMessage,
  loading,
  isInitializing,
  messagesContainer,
  sessionStore,
  actionDialogVisible,
  currentAction,
  initSession,
  createNewSession,
  sendMessage,
  handleActionClick,
  getButtonType,
  scrollToBottom,
  syncMessagesFromStore,
  updateLastMessageActions,
  startScheduleCreationWorkflow,
  handleDialogConfirm,
  handleDialogCancel,
  loadHistoryConversation,
} = useChatAssistant(dataViewDialogRef, {
  showStaffDetails: (data: any) => {
    staffDetailsData.value = data
    staffDetailsVisible.value = true
  },
  showRulesDetails: (data: any) => {
    rulesDetailsData.value = data
    rulesDetailsVisible.value = true
  },
  showShiftSchedule: (data: any) => {
    shiftScheduleData.value = data
    shiftScheduleVisible.value = true
  },
  showMultiShiftSchedule: (data: any) => {
    multiShiftScheduleData.value = data
    multiShiftScheduleVisible.value = true
  },
  showPersonalNeeds: (data: any) => {
    personalNeedsData.value = data
    personalNeedsVisible.value = true
  },
  showTemporaryRules: (data: any) => {
    temporaryRulesData.value = data
    temporaryRulesVisible.value = true
  },
  showValidationResult: (data: any) => {
    validationResultData.value = data
    validationResultVisible.value = true
  },
  showSchedulePreview: (data: any) => {
    schedulePreviewData.value = data
    schedulePreviewVisible.value = true
  },
  showChangeDetail: (data: any) => {
    changeDetailData.value = data
    changeDetailVisible.value = true
  },
})

defineExpose({ startScheduleCreationWorkflow })

// 监听 store 消息更新
watch(
  () => sessionStore.messages,
  (newMessages) => { syncMessagesFromStore(newMessages) },
  { deep: true },
)

// 监听 workflow 更新
watch(
  () => sessionStore.workflow,
  (newWorkflow) => {
    if (!newWorkflow)
      return
    updateLastMessageActions(newWorkflow.actions || [])
  },
  { deep: true },
)

// 输入框禁用逻辑
const isInputDisabled = computed(() => {
  const workflow = sessionStore.workflow
  if (!workflow?.phase)
    return false
  if (!workflow.extra)
    return true
  return workflow.extra.allowUserInput !== true
})

function handleClose() {
  emit('close')
}

async function handleLoadConversation(conversationId: string) {
  try {
    await loadHistoryConversation(conversationId)
  }
  catch (error) {
    console.error('加载历史对话失败:', error)
  }
}

onMounted(async () => {
  await initSession()
  if (sessionStore.workflow?.actions?.length) {
    updateLastMessageActions(sessionStore.workflow.actions)
  }
  scrollToBottom()
})

onBeforeUnmount(() => {
  // 不断开全局 WS 连接
})
</script>

<template>
  <div class="chat-assistant">
    <!-- 头部 -->
    <div class="chat-header">
      <div class="header-title">
        <span class="title-text">智能排班助手</span>
      </div>
      <div class="header-actions">
        <el-button text @click="historyDialogVisible = true">
          历史对话
        </el-button>
        <el-button text @click="createNewSession">
          新建会话
        </el-button>
        <el-button :icon="Close" text @click="handleClose" />
      </div>
    </div>

    <!-- 消息区域 -->
    <div ref="messagesContainer" class="chat-messages">
      <!-- 工作流进度 -->
      <WorkflowProgress
        :workflow="sessionStore.workflow?.workflow"
        :current-phase="sessionStore.workflow?.phase"
      />

      <!-- 加载状态 -->
      <div v-if="isInitializing" class="loading-message">
        <el-icon class="is-loading">
          <Loading />
        </el-icon>
        <span>加载会话中...</span>
      </div>

      <!-- 空消息 -->
      <div v-else-if="messages.length === 0" class="empty-message">
        <p>你好！我是智能排班助手</p>
        <p class="hint">
          我可以帮你快速生成排班计划、优化排班安排
        </p>
      </div>

      <!-- 消息列表 -->
      <div
        v-for="msg in messages"
        :key="msg.id"
        class="message-item"
        :class="[msg.role, { 'system-message': msg.role === 'system' }]"
      >
        <div class="message-content">
          <MarkdownMessage v-if="(msg.role === 'assistant' || msg.role === 'system') && msg.content" :content="msg.content" />
          <div v-else-if="msg.content" class="content-text">
            {{ msg.content }}
          </div>

          <!-- 操作按钮 -->
          <div v-if="(msg.actions?.length || 0) > 0 || (msg.workflowActions?.length || 0) > 0" class="message-actions">
            <template v-if="msg.actions?.length">
              <el-button
                v-for="action in msg.actions"
                :key="action.id"
                :type="getButtonType(action.style)"
                size="small"
                @click="handleActionClick(action, msg)"
              >
                {{ action.label }}
              </el-button>
            </template>
            <template v-if="msg.workflowActions?.length">
              <el-button
                v-for="action in msg.workflowActions"
                :key="action.id"
                :type="getButtonType(action.style)"
                size="small"
                @click="handleActionClick(action, msg)"
              >
                {{ action.label }}
              </el-button>
            </template>
          </div>
        </div>
        <div class="message-time">
          {{ new Date(msg.createdAt).toLocaleTimeString() }}
        </div>
      </div>

      <!-- 思考中 -->
      <div v-if="loading" class="message-item assistant">
        <div class="message-content">
          <el-icon class="is-loading">
            <Loading />
          </el-icon>
          正在思考中...
        </div>
      </div>
    </div>

    <!-- 输入区域 -->
    <div class="chat-input">
      <el-input
        v-model="inputMessage"
        type="textarea"
        :rows="3"
        :placeholder="isInputDisabled ? '当前工作流状态不允许输入，请等待或点击操作按钮...' : '输入消息，例如：帮我生成下周的排班...'"
        :disabled="isInputDisabled || loading"
        @keydown.enter="sendMessage"
      />
      <div class="input-actions">
        <span class="input-hint">Enter 发送</span>
        <el-button
          type="primary"
          :loading="loading"
          :disabled="isInputDisabled || !inputMessage.trim()"
          @click="sendMessage"
        >
          发送
        </el-button>
      </div>
    </div>

    <!-- 弹框组件 -->
    <DataViewDialog ref="dataViewDialogRef" />
    <ActionFormDialog
      v-model:visible="actionDialogVisible"
      :action="currentAction"
      @confirm="handleDialogConfirm"
      @cancel="handleDialogCancel"
    />
    <StaffDetailsDialog v-model:visible="staffDetailsVisible" :data="staffDetailsData" />
    <RulesDetailsDialog v-model:visible="rulesDetailsVisible" :data="rulesDetailsData" />
    <ShiftScheduleDialog v-model:visible="shiftScheduleVisible" :data="shiftScheduleData" />
    <MultiShiftScheduleDialog v-model:visible="multiShiftScheduleVisible" :data="multiShiftScheduleData" />
    <PersonalNeedsDialog v-model:visible="personalNeedsVisible" :data="personalNeedsData" />
    <TemporaryRulesDialog v-model:visible="temporaryRulesVisible" :data="temporaryRulesData" />
    <ValidationResultDialog v-model:visible="validationResultVisible" :data="validationResultData" />
    <SchedulePreviewDialog v-model:visible="schedulePreviewVisible" :data="schedulePreviewData" />
    <el-dialog v-model="changeDetailVisible" title="变更详情" width="900px" :close-on-click-modal="false">
      <ChangeDetailView v-if="changeDetailData" :change-preview="changeDetailData" />
    </el-dialog>
    <ConversationHistoryDialog v-model:visible="historyDialogVisible" @load="handleLoadConversation" />
  </div>
</template>

<style lang="scss" scoped>
.chat-assistant {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: #fff;
}

.chat-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  border-bottom: 1px solid #e4e7ed;

  .header-title {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 16px;
    font-weight: 600;
  }

  .header-actions {
    display: flex;
    align-items: center;
    gap: 8px;
  }
}

.chat-messages {
  flex: 1;
  overflow-y: auto;
  padding: 16px;

  .loading-message,
  .empty-message {
    text-align: center;
    padding: 60px 20px;
    color: var(--el-text-color-secondary);
  }

  .loading-message {
    display: flex;
    flex-direction: column;
    align-items: center;

    .el-icon {
      font-size: 32px;
      margin-bottom: 12px;
    }
  }

  .empty-message .hint {
    font-size: 12px;
    color: var(--el-text-color-placeholder);
  }

  .message-item {
    margin-bottom: 16px;
    display: flex;
    flex-direction: column;

    &.user {
      align-items: flex-end;

      .message-content {
        background: var(--el-color-primary);
        color: #fff;
      }
    }

    &.assistant {
      align-items: flex-start;

      .message-content {
        background: #f5f7fa;
        color: var(--el-text-color-primary);
      }
    }

    &.system {
      align-items: center;

      .message-content {
        background-color: #d1d5db;
        color: #4b5563;
        font-size: 11px;
        padding: 5px 14px;
        border-radius: 14px;
      }

      .message-time {
        display: none;
      }
    }

    .message-content {
      max-width: 80%;
      padding: 10px 14px;
      border-radius: 8px;
      word-break: break-word;
      line-height: 1.5;
    }

    .message-actions {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
      margin-top: 12px;
      padding-top: 12px;
      border-top: 1px solid rgba(0, 0, 0, 0.1);
    }

    .message-time {
      font-size: 12px;
      color: var(--el-text-color-placeholder);
      margin-top: 4px;
      padding: 0 4px;
    }
  }
}

.chat-input {
  padding: 16px;
  border-top: 1px solid #e4e7ed;

  .input-actions {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-top: 8px;

    .input-hint {
      font-size: 12px;
      color: var(--el-text-color-placeholder);
    }
  }
}
</style>
