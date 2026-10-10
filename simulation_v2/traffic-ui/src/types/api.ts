/** Node `GET /api/v1/operator/pace` and `POST /api/v1/operator/time-scale` response. */
export interface PaceStatus {
  attempt_window_sec: number
  pace_debt_blocks: number
  missed_budget_sec: number
  base_block_time: number
  time_scale: number
  wall_sleep_sec: number
  produce_interval_sec: number
  auto_produce: boolean
}

/** One enrichment agent row from traffic `GET /status`. */
export interface TrafficAgentStatus {
  address: string
  role: string
  genesis?: boolean
  expected_wrt: number
  growth_wrt: number
  b_i: number
  s_i: number
  enter: boolean
  adopted_from: string
}

export interface TrafficParams {
  target_enrichment_bots: number
  target_citizens: number
  citizen_transfers_per_tick: number
  horizon_blocks: number
  peer_sample_min: number
  peer_sample_max: number
}

/** Traffic `GET /status`. */
export interface TrafficStatus {
  running: boolean
  release?: boolean
  height: number
  intensity: number
  wallets: number
  roles: Record<string, number>
  pools?: Record<string, number>
  floor?: {
    min_suppliers: number
    min_validators: number
    suppliers: number
    validators: number
  }
  genesis?: { address: string; managed: boolean }
  agents?: TrafficAgentStatus[]
  horizon_blocks?: number
  params?: TrafficParams
  last_tick: {
    height?: number
    market?: number
    bots?: number
    declare?: number
    citizen_txs?: number
    spend_txs?: number
  }
  last_errors: string[]
  node_url: string
  produce_interval_sec?: number
  attempt_window_sec?: number
  pace_debt_blocks?: number
  effective_poll_sec?: number
  flags: {
    market: boolean
    bots: boolean
    declare: boolean
  }
}

/** One row from traffic `GET /wallets` (`BotWallet.to_dict()`). */
export interface TrafficWallet {
  seed: string
  address: string
  pub_hex?: string
  role: string
  desired_role?: string
  zkp_id?: string
  verified?: boolean
  funded?: boolean
  kind: string
  genesis?: boolean
}

export interface TrafficWalletsResponse {
  count: number
  wallets: TrafficWallet[]
}

export interface BroadcastResult {
  code: number
  log: string
  hash: string
}

/** `POST /api/v1/operator/mint` wraps the broadcast result. */
export interface MintResult {
  broadcast?: BroadcastResult
  from?: string
  to?: string
  amount?: number
}

export interface OperatorAccount {
  address: string
  pub_hex: string
  account: ChainAccount | null
}

export interface ChainAccount {
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
  lzn_freeze_until: number
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

export interface AccountDetail {
  account: ChainAccount
  balances_display: {
    wrt: string
    lzn: string
    lzn_activated: string
    ant: string
  }
  open_orders: OrderView[]
  last_declare: { b_i: number; s_i: number; f_i: number; w_i: number } | null
}

export interface ChainParams {
  alpha_num: number
  alpha_den: number
  lambda_num: number
  lambda_den: number
}

export interface TxMessage {
  type: string
  [key: string]: string | number
}
