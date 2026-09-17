import type { Account, Market, Order, Transaction } from '../types'

export type BackendMode = 'sim' | 'chain'

export interface WalletState {
  height: number
  mempool_size?: number
  accounts_count: number
  accounts: Record<string, Account>
  market: Market | null
  genesis_validator?: string
  genesis_provider?: string
  sim_treasury?: string
  sim_speed?: number
}

export interface TxResult {
  accepted: boolean
  message: string
  tx_hash?: string
}

export interface SubscribeCallbacks {
  onState: (s: WalletState) => void
  onConnected?: (connected: boolean) => void
  onError?: (err: string) => void
}

export interface BackendAdapter {
  readonly mode: BackendMode

  getState(): Promise<WalletState>

  subscribe(cb: SubscribeCallbacks): () => void

  submit(body: Record<string, unknown>): Promise<TxResult>

  fetchHistory(
    address: string,
    limit?: number,
  ): Promise<{ history: Transaction[]; open_orders: Order[] }>

  fetchOpenOrders(address: string): Promise<Order[]>

  createAccounts?(count: number): Promise<string[]>

  mintAsset?(
    address: string,
    amount: number,
    asset: 'wrt' | 'lzn' | 'ant',
  ): Promise<TxResult>

  connectWithMnemonic?(mnemonic: string): Promise<string>

  disconnect(): void
}
