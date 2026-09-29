export interface BroadcastResult {
  code: number
  log: string
  hash: string
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
