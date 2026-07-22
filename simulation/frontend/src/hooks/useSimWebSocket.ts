/**
 * Единый WS-хук симуляции (Этап 5): инкапсулирует подключение к /ws,
 * парсинг сообщений init / new_block / block_delta и обновление zustand-стора.
 */
import { useEffect } from 'react'

import { WS_URL } from '../config'
import { useSimStore } from '../store/sim'
import type { BlockDeltaPayload } from '../store/sim'
import type { Market, SimulatorApiState } from '../types'

interface WsMessage {
  type: 'init' | 'new_block' | 'block_delta' | string
  data: {
    state?: SimulatorApiState
    block_time?: number
    block?: BlockDeltaPayload['block']
    delta?: BlockDeltaPayload['delta']
    market?: Market
    consensus_validators?: BlockDeltaPayload['consensus_validators']
    next_proposer?: string
    epoch_ant_sold_volume?: number
    epoch_emission_coefficient?: number
    mempool_size?: number
    tps_history_tail?: BlockDeltaPayload['tps_history_tail']
  }
}

export interface UseSimWebSocketOpts {
  /** Колбек на init — например, синхронизировать UI-инпут блок-тайма. */
  onInitBlockTime?: (sec: number) => void
}

export function useSimWebSocket(opts: UseSimWebSocketOpts = {}) {
  const { onInitBlockTime } = opts
  const setStatus = useSimStore((s) => s.setStatus)
  const applySnapshot = useSimStore((s) => s.applySnapshot)
  const applyDelta = useSimStore((s) => s.applyDelta)

  useEffect(() => {
    const ws = new WebSocket(WS_URL)
    ws.onopen = () => setStatus('Connected (Live)')
    ws.onclose = () => setStatus('Disconnected')
    ws.onerror = () => setStatus('Error')

    ws.onmessage = (event: MessageEvent<string>) => {
      let msg: WsMessage
      try {
        msg = JSON.parse(event.data)
      } catch {
        return
      }

      if (msg.type === 'init' && msg.data.state) {
        const block = msg.data.block
        const txs = block?.transactions ?? []
        applySnapshot(msg.data.state, msg.data.block_time ?? 0, {
          recent: 'keep',
          blockTxs: txs,
        })
        if (onInitBlockTime && typeof msg.data.block_time === 'number') {
          onInitBlockTime(msg.data.block_time)
        }
        return
      }

      if (msg.type === 'new_block' && msg.data.state) {
        const block = msg.data.block
        const txs = block?.transactions ?? []
        const isReset = msg.data.state.height === 0 && block?.height === 0
        applySnapshot(msg.data.state, msg.data.block_time ?? 0, {
          recent: isReset ? 'clear' : 'append',
          blockTxs: txs,
        })
        return
      }

      if (msg.type === 'block_delta' && msg.data.block) {
        applyDelta({
          block_time: msg.data.block_time,
          block: msg.data.block,
          delta: msg.data.delta,
          market: msg.data.market,
          consensus_validators: msg.data.consensus_validators,
          next_proposer: msg.data.next_proposer,
          epoch_ant_sold_volume: msg.data.epoch_ant_sold_volume,
          epoch_emission_coefficient: msg.data.epoch_emission_coefficient,
          mempool_size: msg.data.mempool_size,
          tps_history_tail: msg.data.tps_history_tail,
        })
        return
      }
    }

    return () => {
      ws.close()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])
}
