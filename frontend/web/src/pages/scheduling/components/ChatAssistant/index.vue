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
import SvgIcon from '@/components/SvgIcon.vue'

const emit = defineEmits<{
  close: []
}>()

// DataViewDialog 引用
const dataViewDialogRef = ref<DataViewDialogExpose | null>(null)

// StaffDetailsDialog 状态
const staffDetailsVisible = ref(false)
const staffDetailsData = ref<any>(null)

// RulesDetailsDialog 状态
const rulesDetailsVisible = ref(false)
const rulesDetailsData = ref<any>(null)

// ShiftScheduleDialog 状态
const shiftScheduleVisible = ref(false)
const shiftScheduleData = ref<any>(null)

// MultiShiftScheduleDialog 状态（用于多个班次时显示）
const multiShiftScheduleVisible = ref(false)
const multiShiftScheduleData = ref<any>(null)

// PersonalNeedsDialog 状态
const personalNeedsVisible = ref(false)
const personalNeedsData = ref<any>(null)

// TemporaryRulesDialog 状态
const temporaryRulesVisible = ref(false)
const temporaryRulesData = ref<any>(null)

// ValidationResultDialog 状态
const validationResultVisible = ref(false)
const validationResultData = ref<any>(null)

// SchedulePreviewDialog 状态
const schedulePreviewVisible = ref(false)
const schedulePreviewData = ref<any>(null)

// ChangeDetailView 状态（变更详情对话框）
const changeDetailVisible = ref(false)
const changeDetailData = ref<any>(null)

// 使用业务逻辑 Hook（传递 ref 对象和人员详情控制函数）
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
    console.log('[ChatAssistant] showChangeDetail 被调用，数据:', data)
    console.log('[ChatAssistant] data.taskId:', data?.taskId)
    console.log('[ChatAssistant] data.shifts:', data?.shifts)
    console.log('[ChatAssistant] data.shifts 长度:', data?.shifts?.length)
    changeDetailData.value = data
    changeDetailVisible.value = true
    console.log('[ChatAssistant] 对话框可见性设置为 true')
  },
})

// 暴露给父组件的方法
defineExpose({
  startScheduleCreationWorkflow,
})

// 监听 store 中的消息更新
watch(
  () => sessionStore.messages,
  (newMessages) => {
    syncMessagesFromStore(newMessages)
  },
  { deep: true },
)

// 监听 workflow 更新
watch(
  () => sessionStore.workflow,
  (newWorkflow, oldWorkflow) => {
    if (!newWorkflow)
      return

    // 状态变更通知由后端自动生成并推送，前端只需更新 actions
    if (oldWorkflow && newWorkflow.phase !== oldWorkflow.phase) {
      // 后端会推送系统消息，这里不需要手动添加
      console.log('Workflow state changed:', oldWorkflow.phase, '→', newWorkflow.phase)
    }

    // 更新当前消息的可用操作
    updateLastMessageActions(newWorkflow.actions || [])
  },
  { deep: true },
)

// 计算输入框是否禁用
const isInputDisabled = computed(() => {
  const workflow = sessionStore.workflow
  // 如果没有工作流，允许输入（用户可以开始对话）
  if (!workflow) {
    return false
  }
  // 如果工作流没有 phase（未开始），允许输入
  if (!workflow.phase) {
    return false
  }
  // 工作流已开始，根据 extra.allowUserInput 决定
  // 如果没有 extra，默认禁用（工作流进行中不允许输入）
  if (!workflow.extra) {
    return true
  }
  // 只有当 allowUserInput 明确为 true 时才启用
  return workflow.extra.allowUserInput !== true
})

// 关闭助手
function handleClose() {
  emit('close')
}

// 历史对话对话框状态
const historyDialogVisible = ref(false)

// 创建新会话
async function handleCreateNewSession() {
  await createNewSession()
}

// 打开历史对话对话框
function handleOpenHistoryDialog() {
  historyDialogVisible.value = true
}

// 加载历史对话
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
  // 确保 workflow actions 被同步（如果组件挂载时 workflow 已存在）
  if (sessionStore.workflow && sessionStore.workflow.actions && sessionStore.workflow.actions.length > 0) {
    updateLastMessageActions(sessionStore.workflow.actions)
  }
  scrollToBottom()
})

onBeforeUnmount(() => {
  // 组件销毁时不断开 WebSocket 连接，保持全局连接活跃
  // 连接会在应用关闭时自动断开
  console.log('[ChatAssistant] Component unmounted, but WebSocket connection is kept alive globally')
})
</script>

<template>
  <div class="chat-assistant">
    <!-- 头部 -->
    <div class="chat-header">
      <div class="header-title">
        <span class="title-icon"><SvgIcon name="robot" size="1.2em" /></span>
        <span class="title-text">智能排班助手</span>
      </div>
      <div class="header-actions">
        <el-button text @click="handleOpenHistoryDialog">
          历史对话
        </el-button>
        <el-button text @click="handleCreateNewSession">
          新建会话
        </el-button>
        <el-button :icon="Close" text @click="handleClose" />
      </div>
    </div>

    <!-- 消息区域 -->
    <div ref="messagesContainer" class="chat-messages">
      <!-- 工作流进度显示 -->
      <WorkflowProgress
        :workflow="sessionStore.workflow?.workflow"
        :current-phase="sessionStore.workflow?.phase"
      />

      <!-- 初始化加载状态 -->
      <div v-if="isInitializing" class="loading-message">
        <el-icon class="is-loading">
          <Loading />
        </el-icon>
        <span>加载会话中...</span>
      </div>

      <!-- 空消息状态 -->
      <div v-else-if="messages.length === 0" class="empty-message">
        <div class="empty-icon">
          <SvgIcon name="message-circle" size="48px" />
        </div>
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
          <!-- 使用Markdown渲染 -->
          <MarkdownMessage v-if="msg.role === 'assistant' && msg.content" :content="msg.content || ''" />
          <MarkdownMessage v-else-if="msg.role === 'system' && msg.content" :content="msg.content || ''" />
          <div v-else-if="msg.content" class="content-text">
            {{ msg.content }}
          </div>
          <div v-else class="content-text empty-content">
            （无内容）
          </div>

          <!-- 操作按钮区域 -->
          <div v-if="(msg.actions && msg.actions.length > 0) || (msg.workflowActions && msg.workflowActions.length > 0)" class="message-actions">
            <!-- 持久化按钮 -->
            <template v-if="msg.actions && msg.actions.length > 0">
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

            <!-- 临时工作流按钮 -->
            <template v-if="msg.workflowActions && msg.workflowActions.length > 0">
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

      <!-- 加载状态 -->
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

    <!-- 数据查看弹框 -->
    <DataViewDialog ref="dataViewDialogRef" />

    <!-- 动作表单对话框 -->
    <ActionFormDialog
      v-model:visible="actionDialogVisible"
      :action="currentAction"
      @confirm="handleDialogConfirm"
      @cancel="handleDialogCancel"
    />

    <!-- 人员详情对话框 -->
    <StaffDetailsDialog
      v-model:visible="staffDetailsVisible"
      :data="staffDetailsData"
    />

    <!-- 规则详情对话框 -->
    <RulesDetailsDialog
      v-model:visible="rulesDetailsVisible"
      :data="rulesDetailsData"
    />

    <!-- 班次排班详情对话框（单个班次） -->
    <ShiftScheduleDialog
      v-model:visible="shiftScheduleVisible"
      :data="shiftScheduleData"
    />

    <!-- 多班次排班详情对话框（多个班次，使用标签页） -->
    <MultiShiftScheduleDialog
      v-model:visible="multiShiftScheduleVisible"
      :data="multiShiftScheduleData"
      :title="multiShiftScheduleData?.title || '排班详情'"
    />

    <!-- 个人需求详情对话框 -->
    <PersonalNeedsDialog
      v-model:visible="personalNeedsVisible"
      :data="personalNeedsData"
    />

    <!-- 临时规则详情对话框 -->
    <TemporaryRulesDialog
      v-model:visible="temporaryRulesVisible"
      :data="temporaryRulesData"
    />

    <!-- 校验结果详情对话框 -->
    <ValidationResultDialog
      v-model:visible="validationResultVisible"
      :data="validationResultData"
    />

    <!-- 完整排班预览对话框 -->
    <SchedulePreviewDialog
      v-model:visible="schedulePreviewVisible"
      :draft-schedule="schedulePreviewData?.draftSchedule"
      :start-date="schedulePreviewData?.startDate || ''"
      :end-date="schedulePreviewData?.endDate || ''"
    />

    <!-- 变更详情对话框 -->
    <el-dialog
      v-model="changeDetailVisible"
      title="变更详情"
      width="900px"
      :close-on-click-modal="false"
    >
      <ChangeDetailView
        v-if="changeDetailData"
        :change-preview="changeDetailData"
      />
    </el-dialog>

    <!-- 历史对话对话框 -->
    <ConversationHistoryDialog
      v-model:visible="historyDialogVisible"
      @load="handleLoadConversation"
    />
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

    .title-icon {
      font-size: 20px;
    }
  }

  .header-actions {
    display: flex;
    align-items: center;
    gap: 8px;

    .version-selector {
      margin-right: 4px;
    }
  }
}

.chat-messages {
  flex: 1;
  overflow-y: auto;
  padding: 16px;

  .loading-message {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 60px 20px;
    color: var(--el-text-color-secondary);

    .el-icon {
      font-size: 32px;
      margin-bottom: 12px;
    }

    span {
      font-size: 14px;
    }
  }

  .empty-message {
    text-align: center;
    padding: 60px 20px;
    color: var(--el-text-color-secondary);

    .empty-icon {
      font-size: 48px;
      margin-bottom: 16px;
    }

    p {
      margin: 8px 0;
    }

    .hint {
      font-size: 12px;
      color: var(--el-text-color-placeholder);
    }
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

    &.system,
    &.system-message {
      align-items: center !important;
      text-align: center !important;
      margin: 8px 0 !important;
      justify-content: center !important;

      .message-content {
        display: inline-block !important;
        background-color: #d1d5db !important; // 更明显的灰色背景
        color: #4b5563 !important; // 更深的灰色文字，与背景形成对比
        font-size: 11px !important; // 更小的字体，与普通消息区分
        padding: 5px 14px !important;
        border-radius: 14px !important;
        max-width: auto !important;
        margin: 0 auto !important;
        border: none !important;
        box-shadow: none !important;
        font-weight: 400 !important;
        line-height: 1.4 !important;
        letter-spacing: 0.2px !important;

        // 确保 Markdown 内容也使用系统消息样式
        :deep(*) {
          color: #4b5563 !important;
          font-size: 11px !important;
          line-height: 1.4 !important;
        }

        // Markdown 标题样式 - 系统消息中标题应该更小
        :deep(h1),
        :deep(h2),
        :deep(h3),
        :deep(h4),
        :deep(h5),
        :deep(h6) {
          color: #4b5563 !important;
          font-size: 11px !important;
          font-weight: 500 !important;
          margin: 0 !important;
          padding: 0 !important;
          border: none !important;
          line-height: 1.4 !important;
        }

        // Markdown 文本样式
        :deep(p) {
          color: #4b5563 !important;
          font-size: 11px !important;
          margin: 0 !important;
          line-height: 1.4 !important;
        }

        // Markdown 列表样式
        :deep(ul),
        :deep(ol) {
          margin: 0 !important;
          padding-left: 16px !important;
          color: #4b5563 !important;
          font-size: 11px !important;
        }

        :deep(li) {
          margin: 0 !important;
          color: #4b5563 !important;
          font-size: 11px !important;
        }

        // Markdown 粗体和斜体
        :deep(strong),
        :deep(b) {
          color: #4b5563 !important;
          font-weight: 500 !important;
          font-size: 11px !important;
        }

        :deep(em),
        :deep(i) {
          color: #4b5563 !important;
          font-size: 11px !important;
        }

        // Markdown 链接
        :deep(a) {
          color: #4b5563 !important;
          font-size: 11px !important;
          text-decoration: underline !important;
        }

        // Markdown 代码
        :deep(code) {
          background-color: rgba(0, 0, 0, 0.1) !important;
          color: #4b5563 !important;
          font-size: 10px !important;
          padding: 1px 4px !important;
        }

        // Markdown 代码块
        :deep(pre) {
          background-color: rgba(0, 0, 0, 0.1) !important;
          padding: 4px 8px !important;
          margin: 4px 0 !important;
          font-size: 10px !important;

          code {
            background-color: transparent !important;
            padding: 0 !important;
            color: #4b5563 !important;
            font-size: 10px !important;
          }
        }
      }

      .message-time {
        display: none !important;
      }
    }

    .message-content {
      max-width: 80%;
      padding: 10px 14px;
      border-radius: 8px;
      word-break: break-word;
      line-height: 1.5;

      .content-text {
        margin-bottom: 8px;

        &:last-child {
          margin-bottom: 0;
        }
      }

      .message-actions {
        display: flex;
        flex-wrap: wrap;
        gap: 8px;
        margin-top: 12px;
        padding-top: 12px;
        border-top: 1px solid rgba(0, 0, 0, 0.1);

        // 让按钮在小屏上也能很好地排列
        .el-button {
          margin: 0; // 移除 el-button 默认 margin
        }
      }
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
