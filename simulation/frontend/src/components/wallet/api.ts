/**
 * REST-обёртка для /api/wallet/* (выделена из WalletPanel, чтобы независимо тестировать).
 */
export interface WalletSubmitResponse {
  accepted: boolean
  message: string
  tx_hash?: string
}

export async function submitWallet(
  apiBase: string,
  body: Record<string, unknown>,
): Promise<WalletSubmitResponse> {
  const res = await fetch(`${apiBase}/api/wallet/submit`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) {
    return { accepted: false, message: `HTTP ${res.status}` }
  }
  return res.json() as Promise<WalletSubmitResponse>
}

export async function fetchOpenOrders(
  apiBase: string,
  address: string,
): Promise<{ orders: unknown[] }> {
  const res = await fetch(
    `${apiBase}/api/wallet/open-orders?address=${encodeURIComponent(address)}`,
  )
  if (!res.ok) return { orders: [] }
  return res.json() as Promise<{ orders: unknown[] }>
}
