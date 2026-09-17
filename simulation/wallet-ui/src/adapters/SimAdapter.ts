import { API_BASE, WS_URL } from '../config'
import * as api from '../api/client'
import type { Account, Order, Transaction } from '../types'
import type { BackendAdapter, SubscribeCallbacks, TxResult, WalletState } from './types'

interface BlockDelta {
  accounts_changed?: Record<string, Account>
  accounts_removed?: string[]
}

function applyDelta(
  prev: WalletState,
  delta: BlockDelta,
  height?: number,
): WalletState {
  const accounts = { ...prev.accounts }
  for (const [addr, acc] of Object.entries(delta.accounts_changed ?? {})) {
    accounts[addr] = acc
  }
  for (const addr of delta.accounts_removed ?? []) {
    delete accounts[addr]
  }
  return {
    ...prev,
    height: height ?? prev.height,
    accounts,
    accounts_count: Object.keys(accounts).length,
  }
}

export class SimAdapter implements BackendAdapter {
  readonly mode = 'sim' as const

  private lastState: WalletState | null = null

  async getState(): Promise<WalletState> {
    const s = await api.fetchState(API_BASE)
    this.lastState = { ...s, market: s.market ?? null }
    return this.lastState
  }

  subscribe({ onState, onConnected, onError }: SubscribeCallbacks): () => void {
    let closed = false
    let ws: WebSocket | null = null
    let retry: ReturnType<typeof setTimeout> | undefined

    const connect = () => {
      if (closed) return
      ws = new WebSocket(WS_URL)

      ws.onopen = () => onConnected?.(true)

      ws.onclose = () => {
        onConnected?.(false)
        if (!closed) retry = setTimeout(connect, 2000)
      }

      ws.onerror = () => ws?.close()

      ws.onmessage = (ev) => {
        try {
          const msg = JSON.parse(ev.data as string) as Record<string, unknown>
          const data = msg.data as Record<string, unknown> | undefined

          if (msg.type === 'init' && data?.state) {
            const s = data.state as WalletState
            this.lastState = s
            onState(s)
            return
          }

          if (
            msg.type === 'new_block' &&
            (data?.state || msg.state)
          ) {
            const s = (data?.state ?? msg.state) as WalletState
            this.lastState = s
            onState(s)
            return
          }

          if (msg.type === 'block_delta' && data && this.lastState) {
            const height =
              (data.block as Record<string, unknown> | undefined)?.height as
                | number
                | undefined
            const s = applyDelta(
              this.lastState,
              data as unknown as BlockDelta,
              height ?? (data.height as number | undefined),
            )
            this.lastState = s
            onState(s)
          }
        } catch {
          onError?.('Malformed WS frame')
        }
      }
    }

    connect()
    return () => {
      closed = true
      if (retry) clearTimeout(retry)
      ws?.close()
    }
  }

  async submit(body: Record<string, unknown>): Promise<TxResult> {
    const res = await api.submitWallet(body, API_BASE)
    return { accepted: res.accepted, message: res.message, tx_hash: res.tx_hash }
  }

  async fetchHistory(
    address: string,
    limit = 50,
  ): Promise<{ history: Transaction[]; open_orders: Order[] }> {
    return api.fetchAccountHistory(address, limit, API_BASE)
  }

  async fetchOpenOrders(address: string): Promise<Order[]> {
    return api.fetchOpenOrders(address, API_BASE)
  }

  async createAccounts(count: number): Promise<string[]> {
    const res = await api.createAccounts(count, API_BASE)
    return res.addresses
  }

  async mintAsset(
    address: string,
    amount: number,
    asset: 'wrt' | 'lzn' | 'ant',
  ): Promise<TxResult> {
    const res = await api.mintAsset(address, amount, asset, API_BASE)
    const ok = res.status === 'queued' || res.status === 'success'
    return {
      accepted: ok,
      message: ok
        ? `Mint queued${res.tx_hash ? ` · ${res.tx_hash.slice(0, 12)}…` : ''}`
        : res.message,
      tx_hash: res.tx_hash,
    }
  }

  disconnect(): void {
    /* no persistent connection to tear down outside of subscription */
  }
}
