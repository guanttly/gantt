<script setup lang="ts">
import { ChatLineSquare, Promotion } from '@element-plus/icons-vue'
import { nextTick, onMounted, onUnmounted, ref } from 'vue'
import { chat } from '@/api/ai'

interface Message {
  role: 'user' | 'assistant' | 'system'
  content: string
}

interface Session {
  id: string
  title: string
  created_at: string
}

const sessions = ref<Session[]>([])
const currentSessionId = ref<string | null>(null)
const messages = ref<Message[]>([])
const inputText = ref('')
const sending = ref(false)
const messagesContainer = ref<HTMLElement>()

function selectSession(session: Session) {
  currentSessionId.value = session.id
  // 历史会话加载预留
  messages.value = [{ role: 'system', content: `已切换到会话：${session.title}` }]
}

function startNewSession() {
  currentSessionId.value = null
  messages.value = [{
    role: 'system',
    content: '你好！我是AI排班助手，可以帮助你进行排班规划、查看排班数据等操作。有什么可以帮你的？',
  }]
}

async function sendMessage() {
  const text = inputText.value.trim()
  if (!text || sending.value)
    return

  messages.value.push({ role: 'user', content: text })
  inputText.value = ''
  sending.value = true
  scrollToBottom()

  try {
    const res = await chat({ message: text })
    messages.value.push({
      role: 'assistant',
      content: res.reply || '（无回复）',
    })
  }
  catch {
    messages.value.push({
      role: 'assistant',
      content: '抱歉，请求出现错误，请稍后再试。',
    })
  }
  finally {
    sending.value = false
    scrollToBottom()
  }
}

function scrollToBottom() {
  nextTick(() => {
    if (messagesContainer.value) {
      messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
    }
  })
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    sendMessage()
  }
}

onMounted(() => {
  startNewSession()
})

onUnmounted(() => {
  // cleanup if needed
})
</script>

<template>
  <div class="chat-container">
    <!-- 会话列表 -->
    <aside class="session-sidebar">
      <div class="sidebar-header">
        <span>会话列表</span>
        <el-button text size="small" @click="startNewSession">
          新会话
        </el-button>
      </div>
      <div class="session-list">
        <div
          v-for="session in sessions"
          :key="session.id"
          class="session-item"
          :class="{ active: session.id === currentSessionId }"
          @click="selectSession(session)"
        >
          <el-icon><ChatLineSquare /></el-icon>
          <span class="session-title">{{ session.title || '未命名会话' }}</span>
        </div>
        <div v-if="sessions.length === 0" class="empty-tip">
          暂无历史会话
        </div>
      </div>
    </aside>

    <!-- 聊天主体 -->
    <main class="chat-main">
      <div ref="messagesContainer" class="messages">
        <div
          v-for="(msg, idx) in messages"
          :key="idx"
          class="message-row"
          :class="msg.role"
        >
          <div class="avatar">
            {{ msg.role === 'user' ? '我' : 'AI' }}
          </div>
          <div class="bubble">
            {{ msg.content }}
          </div>
        </div>

        <div v-if="sending" class="message-row assistant">
          <div class="avatar">
            AI
          </div>
          <div class="bubble typing">
            正在思考中...
          </div>
        </div>
      </div>

      <!-- 输入区域 -->
      <div class="input-area">
        <el-input
          v-model="inputText"
          type="textarea"
          :rows="2"
          placeholder="输入消息，按 Enter 发送..."
          resize="none"
          @keydown="handleKeydown"
        />
        <el-button type="primary" :icon="Promotion" :loading="sending" circle @click="sendMessage" />
      </div>
    </main>
  </div>
</template>

<style scoped>
.chat-container {
  height: 100%;
  display: flex;
  overflow: hidden;
}

.session-sidebar {
  width: 240px;
  border-right: 1px solid #e5e7eb;
  display: flex;
  flex-direction: column;
  background: #fafafa;
}

.sidebar-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  font-weight: 600;
  border-bottom: 1px solid #e5e7eb;
}

.session-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.session-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  border-radius: 8px;
  cursor: pointer;
  font-size: 14px;
  color: #374151;
  transition: background 0.15s;
}

.session-item:hover {
  background: #e5e7eb;
}

.session-item.active {
  background: #dbeafe;
  color: #1d4ed8;
}

.session-title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.empty-tip {
  text-align: center;
  padding: 32px;
  color: #9ca3af;
  font-size: 13px;
}

.chat-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.messages {
  flex: 1;
  overflow-y: auto;
  padding: 24px;
}

.message-row {
  display: flex;
  gap: 12px;
  margin-bottom: 20px;
}

.message-row.user {
  flex-direction: row-reverse;
}

.avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: #e5e7eb;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
  flex-shrink: 0;
  color: #374151;
}

.message-row.user .avatar {
  background: #3b82f6;
  color: #fff;
}

.message-row.assistant .avatar,
.message-row.system .avatar {
  background: #10b981;
  color: #fff;
}

.bubble {
  max-width: 70%;
  padding: 12px 16px;
  border-radius: 12px;
  background: #f3f4f6;
  font-size: 14px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}

.message-row.user .bubble {
  background: #3b82f6;
  color: #fff;
  border-bottom-right-radius: 4px;
}

.message-row.assistant .bubble,
.message-row.system .bubble {
  border-bottom-left-radius: 4px;
}

.bubble.typing {
  color: #9ca3af;
  font-style: italic;
}

.input-area {
  display: flex;
  gap: 12px;
  align-items: flex-end;
  padding: 16px 24px;
  border-top: 1px solid #e5e7eb;
  background: #fff;
}

.input-area .el-input {
  flex: 1;
}
</style>
