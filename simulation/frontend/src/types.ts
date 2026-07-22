/**
 * Общие типы для симуляции (выделены из App.tsx).
 * Источник правды для бэка: `simulation/backend/core/models.py` и
 * REST/WS API в `simulation/backend/main.py`.
 */

export interface Account {
  address: string
  wrt_balance: number
  lzn_balance: number
  lzn_frozen_mining?: number
  ant_balance: number
  role: string
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
  buyer?: string
  seller?: string
  details?: string
  stake_amount?: number
}

export interface Block {
  height: number
  hash: string
  tx_count: number
  timestamp: number
  proposer?: string
  transactions: Record<string, unknown>[]
}

export interface CanonLogEntry {
  id: number
  ts: number
  source: string
  status: string
  category: string
  canon: string
  title: string
  detail: string
  tx_hash: string
  block_height: number | null
  meta?: Record<string, unknown>
}

export interface ConsensusValidator {
  address: string
  power: number
}

export interface NetworkState {
  status: string
  block_height: number
  block_time: number
  accounts_count: number
  mempool_size?: number
  accounts: Record<string, Account>
  recent_txs: Transaction[]
  market: Market
  blocks: Block[]
  tps_history: { time: string; tps: number }[]
  blocks_per_epoch?: number
  epoch_ant_sold_volume?: number
  epoch_ant_sold_last?: number
  epoch_emission_coefficient?: number
  genesis_validator?: string
  consensus_validators?: ConsensusValidator[]
  next_proposer?: string
  genesis_provider?: string
  sim_treasury?: string
  canon_log?: CanonLogEntry[]
  canonical_block_interval_sec?: number
  sim_block_interval_sec?: number
}

/** Полный snapshot из GET /api/state и WS-сообщения init. */
export interface SimulatorApiState {
  height: number
  mempool_size?: number
  accounts_count: number
  accounts: Record<string, Account>
  market: Market
  blocks: Block[]
  tps_history: { time: string; tps: number }[]
  blocks_per_epoch?: number
  epoch_ant_sold_volume?: number
  epoch_ant_sold_last?: number
  epoch_emission_coefficient?: number
  genesis_validator?: string
  consensus_validators?: ConsensusValidator[]
  next_proposer?: string
  genesis_provider?: string
  sim_treasury?: string
  canon_log?: CanonLogEntry[]
  canonical_block_interval_sec?: number
  sim_block_interval_sec?: number
}

/** Этап 2 — дельта в WS сообщении `block_delta`. */
export interface BlockDelta {
  accounts_changed?: Record<string, Account>
  accounts_removed?: string[]
  orders_changed?: Record<string, unknown>
  orders_removed?: string[]
}

/** Этап 4 — KPI snapshot (GET /api/analytics/kpi). */
export interface KpiSnapshot {
  height: number
  blocks_per_epoch: number
  epoch_remaining_blocks: number
  supply: { wrt: number; lzn: number; ant: number }
  gini: { wrt: number; lzn: number; ant: number }
  velocity: { wrt: number; lzn: number; ant: number }
  burn: { current_epoch: number; emission_estimate: number; ratio: number }
  accepted_block_ratio: number
  avg_match_spread: number
  fee_distribution_top: { address: string; value: number }[]
  role_counts: Record<string, number>
  consensus_validator_count: number
  mempool_size: number
  last_price: number
}

export interface ScenarioReport {
  name: string
  passed: boolean
  error: string | null
  steps_executed: number
  blocks_produced: number
  duration_sec: number
  asserts: {
    description: string
    passed: boolean
    expected: unknown
    actual: unknown
    detail: string
  }[]
}
