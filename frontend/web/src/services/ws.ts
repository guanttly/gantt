import type {
  SocketState,
  WebSocketOptions,
} from '@/types/ws'

/**
 * 规范化 WebSocket URL
 * 如果是相对路径,则转换为完整的 WebSocket URL
 */
function normalizeWebSocketUrl(url: string): string {
  // 如果已经是完整的 WebSocket URL,直接返回
  if (url.startsWith('ws://') || url.startsWith('wss://')) {
    return url
  }

  // 处理相对路径
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const host = window.location.host

  // 确保路径以 / 开头
  const path = url.startsWith('/') ? url : `/${url}`

  return `${protocol}//${host}${path}`
}

export class WebSocketManager {
  private ws: WebSocket | null = null
  private options: Required<WebSocketOptions>
  private reconnectCount = 0
  private heartbeatTimer: ReturnType<typeof setInterval> | null = null
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private messageQueue: Record<string, any>[] = []
  private connecting = false

  // 事件回调
  onStateChange: ((state: SocketState) => void) | null = null
  onMessage: ((message: Record<string, any>) => void) | null = null
  onError: ((error: Event | Error) => void) | null = null
  onOpen: (() => void) | null = null
  onClose: ((reason: string) => void) | null = null

  constructor(options: WebSocketOptions) {
    this.options = {
      protocols: [],
      heartbeatInterval: 30000,
      reconnectAttempts: 3,
      reconnectInterval: 5000,
      ...options,
      // 规范化 URL,将相对路径转换为完整的 WebSocket URL
      url: normalizeWebSocketUrl(options.url),
    }
  }

  // 建立连接
  async connect(): Promise<void> {
    if (this.connecting || this.isConnected()) {
      return Promise.resolve()
    }

    this.connecting = true
    this.updateState('connecting')

    return new Promise((resolve, reject) => {
      try {
        this.ws = new WebSocket(this.options.url, this.options.protocols)

        this.ws.onopen = (_event) => {
          this.connecting = false
          this.reconnectCount = 0
          this.updateState('open')
          this.startHeartbeat()
          this.flushMessageQueue()

          this.onOpen?.()
          resolve()
        }

        this.ws.onclose = (event) => {
          this.connecting = false
          this.stopHeartbeat()

          const reason = event.reason || 'Connection closed'
          this.updateState('closed')

          // 自动重连（除非是主动关闭）
          if (!event.wasClean && this.reconnectCount < this.options.reconnectAttempts) {
            this.scheduleReconnect()
          }

          this.onClose?.(reason)
        }

        this.ws.onerror = (event) => {
          this.connecting = false
          this.updateState('error')
          this.onError?.(event)

          if (this.reconnectCount === 0) {
            // 初次连接失败
            reject(new Error('WebSocket connection failed'))
          }
        }

        this.ws.onmessage = (event) => {
          this.handleMessage(event.data)
        }

        // 连接超时处理
        setTimeout(() => {
          if (this.connecting) {
            this.ws?.close()
            reject(new Error('WebSocket connection timeout'))
          }
        }, 10000) // 10s 超时
      }
      catch (error) {
        this.connecting = false
        this.updateState('error')
        reject(error)
      }
    })
  }

  // 关闭连接
  close(reason = 'Manual close'): void {
    if (this.ws) {
      this.stopHeartbeat()
      this.clearReconnectTimer()

      // 发送关闭帧
      this.ws.close(1000, reason)
      this.ws = null
    }

    this.updateState('closed')
    this.messageQueue = []
  }

  // 发送消息
  send(message: Record<string, any>): void {
    if (!message.timestamp && !message.ts) {
      message.ts = new Date().toISOString()
    }

    if (this.isConnected()) {
      try {
        const messageStr = JSON.stringify(message)
        this.ws!.send(messageStr)
      }
      catch (error) {
        console.error('Failed to send WebSocket message:', error, message)
        throw error
      }
    }
    else {
      // 连接断开时将消息加入队列
      this.messageQueue.push(message as any)

      // 队列过长时清理旧消息
      if (this.messageQueue.length > 100) {
        this.messageQueue = this.messageQueue.slice(-50)
      }
    }
  }

  // 发送 ping
  ping(): void {
    const nonce = Math.random().toString(36).substr(2, 9)
    this.send({
      type: 'ping',
      data: { nonce },
      ts: new Date().toISOString(),
    })
  }

  // 检查连接状态
  isConnected(): boolean {
    return this.ws !== null && this.ws.readyState === WebSocket.OPEN
  }

  isConnecting(): boolean {
    return this.connecting
  }

  getState(): SocketState {
    if (!this.ws)
      return 'idle'

    switch (this.ws.readyState) {
      case WebSocket.CONNECTING:
        return 'connecting'
      case WebSocket.OPEN:
        return 'open'
      case WebSocket.CLOSING:
      case WebSocket.CLOSED:
        return 'closed'
      default:
        return 'error'
    }
  }

  // 私有方法
  private handleMessage(data: string): void {
    try {
      const message: Record<string, any> = JSON.parse(data)

      // 处理内置消息类型
      if (message.type === 'pong') {
        // 心跳响应，更新最后活跃时间
        return
      }

      // 转发到应用层 - 直接传递解析后的消息对象
      this.onMessage?.(message as any)
    }
    catch (error) {
      console.error('Failed to parse WebSocket message:', error, data)
    }
  }

  private updateState(state: SocketState): void {
    this.onStateChange?.(state)
  }

  private startHeartbeat(): void {
    if (this.options.heartbeatInterval <= 0)
      return

    this.stopHeartbeat()
    this.heartbeatTimer = setInterval(() => {
      if (this.isConnected()) {
        this.ping()
      }
    }, this.options.heartbeatInterval)
  }

  private stopHeartbeat(): void {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer)
      this.heartbeatTimer = null
    }
  }

  private scheduleReconnect(): void {
    if (this.reconnectCount >= this.options.reconnectAttempts) {
      return
    }

    this.clearReconnectTimer()
    this.reconnectTimer = setTimeout(() => {
      this.reconnectCount++
      console.warn(`Attempting to reconnect... (${this.reconnectCount}/${this.options.reconnectAttempts})`)
      this.connect().catch((error) => {
        console.error('Reconnect failed:', error)
      })
    }, this.options.reconnectInterval)
  }

  private clearReconnectTimer(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
  }

  private flushMessageQueue(): void {
    if (this.messageQueue.length > 0) {
      const messages = [...this.messageQueue]
      this.messageQueue = []

      // 重发队列中的消息
      messages.forEach((message) => {
        try {
          this.send(message as any)
        }
        catch (error) {
          console.error('Failed to flush queued message:', error, message)
        }
      })
    }
  }

  // 销毁实例
  destroy(): void {
    this.close('Manager destroyed')
    this.onStateChange = null
    this.onMessage = null
    this.onError = null
    this.onOpen = null
    this.onClose = null
  }
}

// 工具函数：创建 WebSocket 管理器
export function createWebSocketManager(options: WebSocketOptions) {
  return new WebSocketManager(options)
}

// 简化的 WebSocket 连接函数
export async function connectWebSocket(url: string, options?: Partial<WebSocketOptions>): Promise<WebSocketManager> {
  const manager = new WebSocketManager({
    url,
    ...options,
  })

  await manager.connect()
  return manager
}
