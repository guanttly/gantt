class HeartCheck {
  timeout = 1000 * 6
  timeoutObj: NodeJS.Timeout | null = null
  serverTimeoutObj: NodeJS.Timeout | null = null
  reset() {
    if (this.timeoutObj)
      clearTimeout(this.timeoutObj)
    if (this.serverTimeoutObj)
      clearTimeout(this.serverTimeoutObj)
    return this
  }

  start(socket: WebSocket | null) {
    if (socket) {
      this.timeoutObj = setTimeout(() => {
        // 这里发送一个心跳，后端收到后，返回一个心跳消息，
        // onmessage拿到返回的心跳就说明连接正常
        socket.send('ping')
        // 如果超过一定时间还没重置，说明后端主动断开了
        this.serverTimeoutObj = setTimeout(() => {
          // 如果onclose会执行reconnect，我们执行ws.close()就行了.
          // 如果直接执行reconnect 会触发onclose导致重连两次
          socket.close()
        }, this.timeout)
      }, this.timeout)
    }
  }
}

export default new HeartCheck()
