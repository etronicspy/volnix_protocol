import { api } from './api'

export interface TxMessage {
  type: string
  [key: string]: string | number | undefined
}

export interface PendingTx {
  hash: string
  sender: string
  label: string
  messages: TxMessage[]
}

interface UnconfirmedTx {
  hash?: string
  body?: { messages?: TxMessage[] }
  auth_info?: { signer_infos?: { address?: string }[] }
}

interface UnconfirmedResponse {
  result?: { txs?: UnconfirmedTx[] }
}

interface TxLookup {
  result?: { code?: number; log?: string } | null
}

export function describeMessage(msg: TxMessage): string {
  switch (msg.type) {
    case 'bank/MsgSend':
      return 'WRT transfer'
    case 'ident/MsgVerifyIdentity':
      return `role → ${String(msg.desired_role ?? '')}`
    case 'ident/MsgMigrateRole':
      return 'role migrate'
    case 'povb/MsgDeclareParticipation':
      return 'declare bᵢ/sᵢ'
    case 'lizenz/MsgActivateLZN':
      return 'activate LZN'
    case 'lizenz/MsgDeactivateLZN':
      return 'deactivate LZN'
    case 'anteil/MsgPlaceOrder':
      return `order ${String(msg.market ?? '')} ${String(msg.side ?? '')}`
    case 'anteil/MsgCancelOrder':
      return `cancel ${String(msg.order_id ?? '')}`
    default:
      return msg.type
  }
}

export function cancelIds(txs: PendingTx[]): Set<string> {
  const ids = new Set<string>()
  for (const tx of txs) {
    for (const msg of tx.messages) {
      if (msg.type === 'anteil/MsgCancelOrder' && msg.order_id) ids.add(String(msg.order_id))
    }
  }
  return ids
}

export interface PendingPlace {
  hash: string
  sender: string
  market: string
  side: string
  amount: number
  price: number
}

export function pendingPlaces(txs: PendingTx[], market?: string): PendingPlace[] {
  const rows: PendingPlace[] = []
  for (const tx of txs) {
    for (const msg of tx.messages) {
      if (msg.type !== 'anteil/MsgPlaceOrder') continue
      const rowMarket = String(msg.market ?? '')
      if (market && rowMarket !== market) continue
      rows.push({
        hash: tx.hash,
        sender: tx.sender,
        market: rowMarket,
        side: String(msg.side ?? ''),
        amount: Number(msg.amount ?? 0),
        price: Number(msg.price ?? 0),
      })
    }
  }
  return rows
}

export async function fetchMempool(address?: string): Promise<PendingTx[]> {
  const body = await api.get<UnconfirmedResponse>('/unconfirmed_txs')
  const txs = body.result?.txs ?? []
  const out: PendingTx[] = []
  for (const tx of txs) {
    const sender = tx.auth_info?.signer_infos?.[0]?.address ?? ''
    if (!tx.hash) continue
    if (address && sender !== address) continue
    const messages = tx.body?.messages ?? []
    out.push({
      hash: tx.hash,
      sender,
      label: messages.map(describeMessage).join(', ') || 'transaction',
      messages,
    })
  }
  return out
}

export async function findPending(hash: string): Promise<PendingTx | null> {
  const txs = await fetchMempool()
  return txs.find((tx) => tx.hash === hash) ?? null
}

/** code 0 settled, non-zero rejected, null still missing from the index. */
export async function settleTx(hash: string): Promise<{ code: number; log: string } | null> {
  try {
    const body = await api.get<TxLookup>(`/api/v1/txs/${hash}`)
    if (!body.result || typeof body.result.code !== 'number') return null
    return { code: body.result.code, log: body.result.log ?? '' }
  } catch {
    return null
  }
}
