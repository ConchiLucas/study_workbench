import { defineStore } from 'pinia'
import { ref, type Ref } from 'vue'

export type MessageHandler = (message: any) => void

/**
 * 全局 WebSocket 单例 Store
 * 保证 HomeView 和 GameView 共用同一个连接，避免重复连接导致后端状态被重置
 */
export const useWebSocketStore = defineStore('websocket', () => {
  const ws: Ref<WebSocket | null> = ref(null)
  const connected = ref(false)
  const handlers = ref<Map<string, MessageHandler>>(new Map())
  let savedToken: string = ''
  let heartbeatTimer: ReturnType<typeof setInterval> | null = null
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let reconnectPromise: Promise<void> | null = null
  let heartbeatDeadline = 0
  let manualDisconnect = false

  const HEARTBEAT_INTERVAL_MS = 25000
  const HEARTBEAT_TIMEOUT_MS = 65000

  function emitToHandlers(message: any) {
    handlers.value.forEach((handler) => {
      try {
        handler(message)
      } catch (e) {
        console.error('Handler error:', e)
      }
    })
  }

  function stopHeartbeat() {
    if (heartbeatTimer) {
      clearInterval(heartbeatTimer)
      heartbeatTimer = null
    }
  }

  function startHeartbeat() {
    stopHeartbeat()
    heartbeatDeadline = Date.now() + HEARTBEAT_TIMEOUT_MS
    heartbeatTimer = setInterval(() => {
      if (!ws.value || ws.value.readyState !== WebSocket.OPEN) return
      if (Date.now() > heartbeatDeadline) {
        console.warn('WebSocket heartbeat timeout, closing socket')
        ws.value.close()
        return
      }
      send('ping')
    }, HEARTBEAT_INTERVAL_MS)
  }

  function handleIncomingMessage(socket: WebSocket, message: any) {
    if (ws.value !== socket) return
    if (message.type === 'pong') {
      heartbeatDeadline = Date.now() + HEARTBEAT_TIMEOUT_MS
      return
    }
    if (message.type === 'duplicate_login') {
      manualDisconnect = true
      clearReconnectTimer()
      stopHeartbeat()
      emitToHandlers(message)
      socket.close()
      ws.value = null
      connected.value = false
      return
    }
    emitToHandlers(message)
  }

  function isSocketActive(socket: WebSocket) {
    return ws.value === socket
  }

  function canReuseSocket(socket: WebSocket | null) {
    return socket && (socket.readyState === WebSocket.OPEN || socket.readyState === WebSocket.CONNECTING)
  }

  function waitForOpen(socket: WebSocket): Promise<void> {
    if (socket.readyState === WebSocket.OPEN) return Promise.resolve()
    if (socket.readyState === WebSocket.CLOSED || socket.readyState === WebSocket.CLOSING) {
      return Promise.reject(new Error('WebSocket is closing'))
    }

    return new Promise((resolve, reject) => {
      let timeout: ReturnType<typeof setTimeout>
      const cleanup = () => {
        clearTimeout(timeout)
        socket.removeEventListener('open', handleOpen)
        socket.removeEventListener('close', handleClose)
        socket.removeEventListener('error', handleError)
      }
      const handleOpen = () => {
        cleanup()
        resolve()
      }
      const handleClose = () => {
        cleanup()
        reject(new Error('WebSocket closed before open'))
      }
      const handleError = () => {
        cleanup()
        reject(new Error('WebSocket failed before open'))
      }

      timeout = setTimeout(() => {
        cleanup()
        reject(new Error('WebSocket connect timeout'))
      }, 5000)

      socket.addEventListener('open', handleOpen)
      socket.addEventListener('close', handleClose)
      socket.addEventListener('error', handleError)
    })
  }

  function clearReconnectTimer() {
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
  }

  function scheduleReconnect() {
    if (manualDisconnect || reconnectTimer || !savedToken) return
    reconnectTimer = setTimeout(async () => {
      reconnectTimer = null
      try {
        await reconnect()
      } catch (e) {
        console.error('Reconnect attempt failed:', e)
        scheduleReconnect()
      }
    }, 1500)
  }

  /**
   * 注册页面级消息处理器（每个页面一个 key，如 'home' 或 'game'）
   */
  function registerHandler(key: string, handler: MessageHandler) {
    handlers.value.set(key, handler)
  }

  /**
   * 注销页面级消息处理器
   */
  function unregisterHandler(key: string) {
    handlers.value.delete(key)
  }

  /**
   * 连接 WebSocket（如果已连接则跳过）
   */
  function connect(token: string) {
    if (canReuseSocket(ws.value)) {
      console.log('WebSocket already connected or connecting, skipping')
      return
    }
    manualDisconnect = false
    savedToken = token
    // 清理旧连接
    if (ws.value) {
      ws.value.onclose = null
      ws.value.onerror = null
      ws.value.onmessage = null
      ws.value.onopen = null
      ws.value.close()
      ws.value = null
    }

    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${location.host}/ws?token=${token}`
    console.log('Connecting WebSocket:', wsUrl)
    const socket = new WebSocket(wsUrl)

    socket.onopen = () => {
      if (!isSocketActive(socket)) return
      console.log('WebSocket connected')
      connected.value = true
      clearReconnectTimer()
      startHeartbeat()
      emitToHandlers({ type: 'ws_connection', data: { status: 'connected' } })
    }

    socket.onmessage = (event) => {
      if (!isSocketActive(socket)) return
      const message = JSON.parse(event.data)
      handleIncomingMessage(socket, message)
    }

    socket.onclose = (event) => {
      if (!isSocketActive(socket)) return
      console.log('WebSocket closed:', event.code)
      connected.value = false
      ws.value = null
      stopHeartbeat()
      emitToHandlers({ type: 'ws_connection', data: { status: 'disconnected' } })
      scheduleReconnect()
    }

    socket.onerror = (error) => {
      if (!isSocketActive(socket)) return
      console.error('WebSocket error:', error)
    }

    ws.value = socket
  }

  /**
   * 发送消息
   */
  function send(type: string, data: any = {}) {
    if (!ws.value || ws.value.readyState !== WebSocket.OPEN) {
      console.error('WebSocket not connected, cannot send:', type)
      return false
    }
    ws.value.send(JSON.stringify({ type, data }))
    return true
  }

  /**
   * 断开连接（仅在登出时调用）
   */
  function disconnect() {
    manualDisconnect = true
    clearReconnectTimer()
    stopHeartbeat()
    if (ws.value) {
      ws.value.close()
      ws.value = null
    }
    connected.value = false
    savedToken = ''
    handlers.value.clear()
  }

  /**
   * 重连 WebSocket（返回 Promise，连接成功后 resolve）
   */
  function reconnect(): Promise<void> {
    if (reconnectPromise) return reconnectPromise
    if (ws.value && ws.value.readyState === WebSocket.OPEN) return Promise.resolve()
    if (ws.value && ws.value.readyState === WebSocket.CONNECTING) return waitForOpen(ws.value)

    const pendingReconnect = new Promise<void>((resolve, reject) => {
      if (!savedToken) {
        reject(new Error('No saved token for reconnect'))
        return
      }
      if (ws.value && ws.value.readyState === WebSocket.OPEN) {
        resolve()
        return
      }

      // 清理旧连接
      if (ws.value) {
        ws.value.onclose = null
        ws.value.onerror = null
        ws.value.onmessage = null
        ws.value.onopen = null
        ws.value.close()
        ws.value = null
      }

      const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
      const wsUrl = `${protocol}//${location.host}/ws?token=${savedToken}`
      console.log('Reconnecting WebSocket:', wsUrl)
      const socket = new WebSocket(wsUrl)

      const timeout = setTimeout(() => {
        if (isSocketActive(socket)) socket.close()
        reject(new Error('Reconnect timeout'))
      }, 5000)

      socket.onopen = () => {
        if (!isSocketActive(socket)) return
        clearTimeout(timeout)
        console.log('WebSocket reconnected')
        connected.value = true
        clearReconnectTimer()
        startHeartbeat()
        emitToHandlers({ type: 'ws_connection', data: { status: 'reconnected' } })
        resolve()
      }

      socket.onmessage = (event) => {
        if (!isSocketActive(socket)) return
        const message = JSON.parse(event.data)
        handleIncomingMessage(socket, message)
      }

      socket.onclose = (event) => {
        if (!isSocketActive(socket)) return
        console.log('WebSocket closed:', event.code)
        clearTimeout(timeout)
        connected.value = false
        ws.value = null
        stopHeartbeat()
        emitToHandlers({ type: 'ws_connection', data: { status: 'disconnected' } })
        scheduleReconnect()
      }

      socket.onerror = (error) => {
        if (!isSocketActive(socket)) return
        clearTimeout(timeout)
        console.error('WebSocket reconnect error:', error)
        reject(error)
      }

      ws.value = socket
    }).finally(() => {
      reconnectPromise = null
    })
    reconnectPromise = pendingReconnect

    return pendingReconnect
  }

  /**
   * 确保连接可用，断开则自动重连
   */
  async function ensureConnected(): Promise<boolean> {
    if (ws.value && ws.value.readyState === WebSocket.OPEN) return true
    try {
      await reconnect()
      return true
    } catch (e) {
      console.error('Failed to reconnect:', e)
      return false
    }
  }

  return {
    ws,
    connected,
    connect,
    send,
    disconnect,
    reconnect,
    ensureConnected,
    registerHandler,
    unregisterHandler
  }
})
