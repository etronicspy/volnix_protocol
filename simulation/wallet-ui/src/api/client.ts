import { API_BASE } from '../config'
import type { Order, SimulatorState, Transaction, WalletSubmitResponse } from '../types'

export async function fetchState(apiBase = API_BASE): Promise<SimulatorState> {
  const res = await fetch(`${apiBase}/api/state`)
  if (!res.ok) throw new Error(`GET /api/state → HTTP ${res.status}`)
  return res.json() as Promise<SimulatorState>
}

export async function submitWallet(
  body: Record<string, unknown>,
  apiBase = API_BASE,
): Promise<WalletSubmitResponse> {
  const res = await fetch(`${apiBase}/api/wallet/submit`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) return { accepted: false, message: `HTTP ${res.status}` }
  return res.json() as Promise<WalletSubmitResponse>
}

export async function fetchOpenOrders(
  address: string,
  apiBase = API_BASE,
): Promise<Order[]> {
  const res = await fetch(
    `${apiBase}/api/wallet/open-orders?address=${encodeURIComponent(address)}`,
  )
  if (!res.ok) return []
  const data = (await res.json()) as { orders?: Order[] }
  return data.orders ?? []
}

export async function fetchAccountHistory(
  address: string,
  limit = 50,
  apiBase = API_BASE,
): Promise<{ history: Transaction[]; open_orders: Order[] }> {
  const res = await fetch(
    `${apiBase}/api/account/${encodeURIComponent(address)}/history?limit=${limit}`,
  )
  if (!res.ok) return { history: [], open_orders: [] }
  return res.json() as Promise<{ history: Transaction[]; open_orders: Order[] }>
}

export async function createAccounts(
  count: number,
  apiBase = API_BASE,
): Promise<{ status: string; addresses: string[] }> {
  const res = await fetch(`${apiBase}/api/sim-operator/accounts`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ count }),
  })
  if (!res.ok) throw new Error(`create accounts → HTTP ${res.status}`)
  return res.json() as Promise<{ status: string; addresses: string[] }>
}

export async function mintAsset(
  address: string,
  amount: number,
  asset_type: 'wrt' | 'lzn' | 'ant',
  apiBase = API_BASE,
): Promise<{ status: string; message: string; tx_hash?: string }> {
  const res = await fetch(`${apiBase}/api/sim-operator/mint`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ address, amount, asset_type }),
  })
  if (!res.ok) return { status: 'error', message: `HTTP ${res.status}` }
  return res.json() as Promise<{ status: string; message: string; tx_hash?: string }>
}
