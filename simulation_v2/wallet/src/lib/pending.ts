import { api } from './api'
import type { TxMessage } from '../types/api'

export interface PendingTx {
  hash: string
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
      return 'перевод WRT'
    case 'ident/MsgVerifyIdentity':
      return `роль → ${String(msg.desired_role ?? '')}`
    case 'ident/MsgMigrateRole':
      return 'перенос роли'
    case 'povb/MsgDeclareParticipation':
      return 'объявление bᵢ/sᵢ'
    case 'lizenz/MsgActivateLZN':
      return 'активация LZN'
    case 'lizenz/MsgDeactivateLZN':
      return 'деактивация LZN'
    case 'anteil/MsgPlaceOrder':
      return `ордер ${String(msg.market ?? '')} ${String(msg.side ?? '')}`
    case 'anteil/MsgCancelOrder':
      return `отмена ${String(msg.order_id ?? '')}`
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
  market: string
  side: string
  amount: number
  price: number
}

export function pendingPlaces(txs: PendingTx[]): PendingPlace[] {
  const rows: PendingPlace[] = []
  for (const tx of txs) {
    for (const msg of tx.messages) {
      if (msg.type !== 'anteil/MsgPlaceOrder') continue
      rows.push({
        hash: tx.hash,
        market: String(msg.market ?? ''),
        side: String(msg.side ?? ''),
        amount: Number(msg.amount ?? 0),
        price: Number(msg.price ?? 0),
      })
    }
  }
  return rows
}

export async function fetchMempool(address: string): Promise<PendingTx[]> {
  const body = await api.get<UnconfirmedResponse>('/unconfirmed_txs')
  const txs = body.result?.txs ?? []
  const mine: PendingTx[] = []
  for (const tx of txs) {
    const signer = tx.auth_info?.signer_infos?.[0]?.address
    if (signer !== address || !tx.hash) continue
    const messages = tx.body?.messages ?? []
    mine.push({
      hash: tx.hash,
      label: messages.map(describeMessage).join(', ') || 'транзакция',
      messages,
    })
  }
  return mine
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
