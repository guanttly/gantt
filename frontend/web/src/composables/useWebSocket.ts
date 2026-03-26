// WebSocket 连接管理
import { onMounted, onUnmounted, ref } from 'vue'
import { getAccessToken } from '@/api/client'

export interface WSMessage {
  type: string
  payload: Record<string, unknown>
  timestamp: string
}

interface UseWebSocketOptions {
  /** 自动重连间隔（毫秒） */
  reconnectInterval?: number
  /** 最大重连次数 */
  maxRetries?: number
  /** 消息处理回调 */
  onMessage?: (msg: WSMessage) => void
}

export function useWebSocket(options: UseWebSocketOptions = {}) {
  const { reconnectInterval = 3000, maxRetries = 10, onMessage } = options

  const ws = ref<WebSocket | null>(null)
  const connected = ref(false)
  const messages = ref<WSMessage[]>([])
  let retryCount = 0
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null

  function connect() {
    const token = getAccessToken()
    if (!token)
      return

    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
    const socket = new WebSocket(`${protocol}//${location.host}/api/v1/ws?token=${token}`)

    socket.onopen = () => {
      connected.value = true
      retryCount = 0
    }

    socket.onclose = () => {
      connected.value = false
      ws.value = null
      // 自动重连
      if (retryCount < maxRetries) {
        retryCount++
        reconnectTimer = setTimeout(connect, reconnectInterval)
      }
    }

    socket.onerror = () => {
      socket.close()
    }

    socket.onmessage = (event) => {
      try {
        const msg: WSMessage = JSON.parse(event.data)
        messages.value.push(msg)
        onMessage?.(msg)
      }
      catch (e) {
        console.error('[WebSocket] Failed to parse message:', e)
      }
    }

    ws.value = socket
  }

  function disconnect() {
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
    retryCount = maxRetries // 防止自动重连
    ws.value?.close()
    ws.value = null
    connected.value = false
  }

  function send(data: Record<string, unknown>) {
    if (ws.value?.readyState === WebSocket.OPEN) {
      ws.value.send(JSON.stringify(data))
    }
  }

  function clearMessages() {
    messages.value = []
  }

  onMounted(connect)
  onUnmounted(disconnect)

  return {
    connected,
    messages,
    connect,
    disconnect,
    send,
    clearMessages,
  }
}
