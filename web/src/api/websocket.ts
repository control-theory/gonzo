import type { WebSocketUpdate } from './types'

type UpdateHandler = (update: WebSocketUpdate) => void

export class DashboardWebSocket {
  private ws: WebSocket | null = null
  private handlers: Set<UpdateHandler> = new Set()
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private url: string

  constructor() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    this.url = `${protocol}//${window.location.host}/ws`
  }

  connect() {
    if (this.ws?.readyState === WebSocket.OPEN) return

    this.ws = new WebSocket(this.url)
    this.ws.onmessage = (event) => {
      try {
        const update: WebSocketUpdate = JSON.parse(event.data)
        this.handlers.forEach((h) => h(update))
      } catch {
        // ignore malformed messages
      }
    }
    this.ws.onclose = () => {
      this.scheduleReconnect()
    }
    this.ws.onerror = () => {
      this.ws?.close()
    }
  }

  disconnect() {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
    this.ws?.close()
    this.ws = null
  }

  subscribe(handler: UpdateHandler): () => void {
    this.handlers.add(handler)
    return () => this.handlers.delete(handler)
  }

  private scheduleReconnect() {
    if (this.reconnectTimer) return
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null
      this.connect()
    }, 2000)
  }
}
