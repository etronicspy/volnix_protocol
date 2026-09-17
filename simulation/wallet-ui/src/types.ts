export type Role = 'citizen' | 'provider' | 'validator' | string

export interface Account {
  address: string
  wrt_balance: number
  lzn_balance: number
  lzn_frozen_mining?: number
  ant_balance: number
  role: Role
  zkp_verified?: boolean
}

export interface Order {
  id: string
  owner: string
  order_type: string
  price: number
  amount: number
  filled: number
  timestamp: number
}

export interface Market {
  bids: Order[]
  asks: Order[]
  last_price: number
  history: { time: string; price: number; ts?: number }[]
}

export interface Transaction {
  tx_hash: string
  tx_type: string
  sender: string
  receiver: string
  amount: number
  asset_type?: string
  price: number
  role: string
  timestamp: number
  details?: string
  stake_amount?: number
}

export interface SimulatorState {
  height: number
  mempool_size?: number
  accounts_count: number
  accounts: Record<string, Account>
  market: Market
  genesis_validator?: string
  genesis_provider?: string
  sim_treasury?: string
  sim_speed?: number
}

export interface WalletSubmitResponse {
  accepted: boolean
  message: string
  tx_hash?: string
}

export interface Feedback {
  ok: boolean
  text: string
}

export type TabId = 'balance' | 'send' | 'history' | 'role' | 'staking' | 'market' | 'faucet'
