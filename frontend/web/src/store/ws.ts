import type { WsMessage } from 'src/ws/ws-message'
import { defineStore } from 'pinia'
import Ws from 'src/ws/websocket'
import { ref } from 'vue'

export const useWsStore = defineStore('ws', () => {
  // 实例
  const instance = ref<Ws | null>(null)
  // socket 消息
  const socketData = ref<WsMessage | null>(null)

  const wsSubscribe = (type: string) => {
    instance.value?.subscribe(type, (message: WsMessage) => {
      // console.log('接收服务端消息： ', message)
      // 每次接收到消息都会更新 socketData
      socketData.value = message
    })
  }

  const sendSocket = (data: any) => {
    instance.value?.send(JSON.stringify(data))
  }

  const destroySocket = () => {
    if (instance.value) {
      // 销毁socket
      instance.value.destroy()
      instance.value = null
    }
  }

  const initWs = async (url: string) => {
    if (!instance.value) {
      const ws = new Ws(url)
      instance.value = ws
      wsSubscribe('main')
    }
    return instance.value
  }

  return {
    socketData,
    initWs,
    sendSocket,
    destroySocket,
  }
})
