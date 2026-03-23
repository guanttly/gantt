import type { WsMessage } from './ws-message'
// 心跳
import heartCheck from './heart-check'
// 消息管理中心
import { emitter } from './mitt'

let flag: boolean | null = false
class Ws {
  // websocket 接口地址
  url: string

  // WebSocket 实例
  ws: WebSocket | null = null

  // 重连中
  isReconnectionLoading = false

  // 延时重连的 id
  timeId: NodeJS.Timeout | null = null

  // 用户手动关闭连接
  isCustomClose = false

  // 错误消息队列
  errorStack: string[] = []

  constructor(url: string) {
    this.url = url
    this.createWebSocket()
  }

  // 创建一个 webSocket 连接
  createWebSocket() {
    if ('WebSocket' in window) {
      if (flag)
        return
      flag = true
      // 实例化WebSocket
      this.ws = new WebSocket(this.url)
      // 监听事件
      this.onopen()
      this.onerror()
      this.onclose()
      this.onmessage()
    }
    else {
      // WebSocket not supported
      throw new Error('WebSocket is not supported in this browser')
    }
  }

  // 监听成功
  onopen() {
    if (this.ws) {
      this.ws.onopen = () => {
        // 连接成功。 检查之前发送失败的消息，如果有就直接再次发送
        this.errorStack.forEach((message) => {
          this.send(message)
        })
        this.errorStack = []
        this.isReconnectionLoading = false
        // 重置心跳时间, 开启心跳
        heartCheck.reset().start(this.ws)
      }
    }
  }

  // 监听错误
  onerror() {
    if (this.ws) {
      this.ws.onerror = (_err) => {
        this.reconnection()
        this.isReconnectionLoading = false
      }
    }
  }

  // 监听关闭
  onclose() {
    if (this.ws) {
      this.ws.onclose = () => {
        // 如果是用户手动关闭的，直接返回
        if (this.isCustomClose)
          return
        // 重新连接
        this.reconnection()
        this.isReconnectionLoading = false
      }
    }
  }

  // 接收 WebSocket 消息
  async onmessage() {
    if (this.ws) {
      this.ws.onmessage = (event) => {
        try {
          // const data = JSON.parse(event.data);
          const data = event.data
          // 接到消息重置心跳时间, 开启新的心跳
          heartCheck.reset().start(this.ws)
          if (data.data === 'pong')
            return
          // 发布消息到消息中心
          emitter.emit('main', data)
        }
        catch (error) {
          // Handle message parsing error
          if (import.meta.env.DEV) {
            console.error('WebSocket message error:', error)
          }
        }
      }
    }
  }

  // 重新链接
  reconnection() {
    // 防止重复重新链接
    if (this.isReconnectionLoading)
      return
    this.isReconnectionLoading = true
    flag = null
    if (this.timeId)
      clearTimeout(this.timeId)
    this.timeId = setTimeout(() => {
      this.createWebSocket()
    }, 3000)
  }

  // 发送消息
  send(message: string) {
    // 连接失败时的处理
    if (this.ws?.readyState !== 1) {
      this.errorStack.push(message)
      return
    }

    this.ws?.send(message)
  }

  // 手动关闭
  close() {
    flag = null
    this.isCustomClose = true
    this.ws?.close()
  }

  // 手动开启
  start() {
    this.isCustomClose = false
    this.reconnection()
  }

  // 订阅
  subscribe(eventName: string, cb: (data: WsMessage) => void) {
    emitter.on(eventName, cb)
  }

  // 取消订阅
  unsubscribe(eventName: string, cb: (data: WsMessage) => void) {
    emitter.off(eventName, cb)
  }

  // 销毁
  destroy() {
    this.close()
    this.ws = null
    this.errorStack = []
    // 移除所有事件
    emitter.all.clear()
  }
}
export default Ws
