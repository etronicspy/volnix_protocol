import { useEffect, useRef, useState } from 'react'
import { WS_URL } from '../config'
import type { WsEvent } from '../types/api'

interface LiveState {
  connected: boolean
  height: number
  chainId: string
  lastEvent: WsEvent | null
  pulse: number
}

export function useChainWebSocket(): LiveState {
  const [state, setState] = useState<LiveState>({
    connected: false,
    height: 0,
    chainId: '',
    lastEvent: null,
    pulse: 0,
  })
  const retryRef = useRef(0)

  useEffect(() => {
    let ws: WebSocket | null = null
    let closed = false
    let timer: number | undefined

    const connect = () => {
      if (closed) return
      ws = new WebSocket(WS_URL)
      ws.onopen = () => {
        retryRef.current = 0
        setState((s) => ({ ...s, connected: true }))
      }
      ws.onclose = () => {
        setState((s) => ({ ...s, connected: false }))
        if (closed) return
        const delay = Math.min(10_000, 500 * 2 ** retryRef.current)
        retryRef.current += 1
        timer = window.setTimeout(connect, delay)
      }
      ws.onerror = () => {
        ws?.close()
      }
      ws.onmessage = (ev) => {
        try {
          const msg = JSON.parse(String(ev.data)) as WsEvent
          if (msg.type === 'ping') return
          setState((s) => {
            const next = { ...s, lastEvent: msg, pulse: s.pulse + 1 }
            if (msg.type === 'init') {
              next.height = msg.height
              next.chainId = msg.chain_id
            }
            if (msg.type === 'new_block' || msg.type === 'chain_reset') {
              next.height = msg.height
              if (msg.type === 'chain_reset') next.chainId = msg.chain_id
            }
            return next
          })
        } catch {
          /* ignore */
        }
      }
    }

    connect()
    return () => {
      closed = true
      if (timer) window.clearTimeout(timer)
      ws?.close()
    }
  }, [])

  return state
}
