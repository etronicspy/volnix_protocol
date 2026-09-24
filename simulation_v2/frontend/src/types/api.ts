export interface ChainSummary {
  chain_id: string
  height: number
  time: string
  app_hash: string
  epoch: number
  blocks_to_epoch: number
  era: number
  blocks_to_halving: number
  l_total: number
  l_total_display: string
  supply: {
    wrt: number
    wrt_display: string
    lzn_minted_tokens: number
    lzn_pool_remaining: number
    lzn_free: number
    lzn_activated: number
    lzn_escrow: number
    ant: number
    ant_display: string
  }
  params: {
    lambda: string
    alpha: string
    k: number
    epoch_blocks: number
    halving_interval: number
    base_block_reward: number
    current_block_reward: number
  }
  n_accounts: number
  n_validators: number
  n_suppliers: number
  mempool: number
}

export interface BlockSummary {
  height: number
  hash: string
  time: string
  proposer: string
  n_txs: number
  app_hash: string
  data_hash: string
  set_updated?: boolean
  n_declares?: number
  n_passed?: number
  sum_b?: number
  fill?: number
  l_decl?: number
  b_min?: number
  b_max?: number
}

export interface CommitSigView {
  block_id_flag: string
  validator_address: string
  timestamp: string
  signature: string
}

export interface CommitView {
  height: number
  round: number
  block_id: { hash: string; parts?: { total: number; hash: string } }
  signatures: CommitSigView[]
}

export interface ValidatorSetView {
  validators: ValidatorView[]
  proposer: string
}

export interface BlockConsensus {
  signing_set: ValidatorSetView | null
  next_set: ValidatorSetView | null
  commit: CommitView | null
  povb: PovbTrace
  voted_power: number
  total_power: number
  round: number
}

export interface BlockDetail {
  block: {
    header: Record<string, unknown>
    data: { txs: unknown[] }
    evidence: unknown[]
    last_commit: CommitView | null
  }
  hash: string
  results: {
    height: number
    txs_results: TxResult[]
    begin_block_events: ChainEvent[]
    end_block_events: ChainEvent[]
    validator_updates: ValidatorView[]
    povb: PovbTrace
  } | null
  consensus?: BlockConsensus
}

export interface TxResult {
  code: number
  log: string
  gas_wanted: number
  gas_used: number
  events: ChainEvent[]
  tx_hash: string
}

export interface ChainEvent {
  type: string
  attributes: { key: string; value: string }[]
}

export interface TxListItem {
  hash: string
  height: number
  index: number
  sender: string
}

export interface TxDetail {
  hash: string
  height: number
  index: number
  sender: string
  tx: Record<string, unknown>
  result: TxResult | null
}

export interface AccountListItem {
  address: string
  role: string
  genesis_no_zkp: boolean
  in_validator_set: boolean
  is_proposer: boolean
  wrt: number
  lzn: number
  lzn_activated: number
  ant: number
  wrt_display: string
  lzn_display: string
  lzn_activated_display: string
  ant_display: string
  sequence: number
  last_tx_height: number
  created_height: number
  lzn_freeze_until: number
  open_orders: number
}

export interface AccountListResponse {
  height: number
  count: number
  accounts: AccountListItem[]
}

export interface AccountView {
  account: {
    address: string
    pub_hex: string
    role: string
    wrt: number
    lzn: number
    lzn_activated: number
    ant: number
    sequence: number
    last_tx_height: number
    zkp_id: string
    genesis_no_zkp: boolean
  }
  balances_display: {
    wrt: string
    lzn: string
    lzn_activated: string
    ant: string
  }
  open_orders: OrderView[]
  last_declare: { b_i: number; s_i: number; f_i: number; w_i: number } | null
}

export interface OrderView {
  order_id: string
  owner: string
  market: string
  side: string
  order_type: string
  amount: number
  price: number
  remaining?: number
  status: string
}

export interface ValidatorView {
  address: string
  pub_hex: string
  l_i: number
  s_i: number
  w_i: number
  power: number
  proposer_priority: number
  role: string
  ant: number
  is_proposer: boolean
}

export interface PovbTrace {
  height?: number
  l_total?: number
  l_decl?: number
  lambda?: string
  alpha?: string
  upper?: number | string
  lower?: number | string
  b_min?: number
  b_max?: number
  sum_b?: number
  fill?: number
  k?: number
  set_updated?: boolean
  declares?: PovbDeclare[]
  passed?: string[]
}

export interface PovbDeclare {
  validator: string
  b_i: number
  s_i: number
  f_i: number
  l_i: number
  w_i: number
  valid: boolean
  reason: string
  passed: boolean
  excluded: string
}

export interface EpochRecord {
  epoch: number
  height: number
  l_total: number
  ant_wiped: number
  ant_emit: number
  lzn_emit: number
  suppliers: string[]
}

export interface OrderBook {
  market: string
  bids: { order_id: string; owner: string; price: number; remaining: number }[]
  asks: { order_id: string; owner: string; price: number; remaining: number }[]
}

export interface TradeRow {
  height: number
  market?: string
  price?: string
  amount?: string
  maker?: string
  taker?: string
}

export interface SearchResult {
  kind: 'block' | 'tx' | 'account' | 'unknown'
  height?: number
  hash?: string
  address?: string
  q?: string
}

export type WsEvent =
  | { type: 'init'; height: number; chain_id: string; app_hash: string }
  | {
      type: 'new_block'
      height: number
      hash: string
      time: string
      proposer: string
      n_txs: number
      app_hash: string
    }
  | { type: 'new_tx'; hash: string; height: number }
  | { type: 'validator_set_update'; height: number; validators: unknown[] }
  | { type: 'epoch_boundary'; epoch: number; height: number }
  | { type: 'ping' }

export interface TrafficStatus {
  running: boolean
  height: number
  intensity: number
  wallets: number
  roles: Record<string, number>
  last_tick: {
    height?: number
    market?: number
    bots?: number
    declare?: number
  }
  last_errors: string[]
  node_url: string
  produce_interval_sec?: number
  effective_poll_sec?: number
  flags: {
    market: boolean
    bots: boolean
    declare: boolean
  }
}
