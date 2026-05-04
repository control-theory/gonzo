import { useEffect, useRef, useCallback, useState } from 'react'
import { DashboardWebSocket } from '../api/websocket'
import type { WebSocketUpdate } from '../api/types'

export function useWebSocket() {
  const wsRef = useRef<DashboardWebSocket | null>(null)
  const [lastUpdate, setLastUpdate] = useState<WebSocketUpdate | null>(null)

  const onUpdate = useCallback((update: WebSocketUpdate) => {
    setLastUpdate(update)
  }, [])

  useEffect(() => {
    const ws = new DashboardWebSocket()
    wsRef.current = ws
    ws.subscribe(onUpdate)
    ws.connect()
    return () => ws.disconnect()
  }, [onUpdate])

  return lastUpdate
}
