import { WALLETS_KEY, WALLET_ACTIVE_KEY } from '../config'
import { normalizePhrase } from './phrase'

export interface StoredWallet {
  seed: string
  label: string
  address: string
}

function parseWallets(raw: string | null): StoredWallet[] {
  if (!raw) return []
  try {
    const parsed: unknown = JSON.parse(raw)
    if (!Array.isArray(parsed)) return []
    const out: StoredWallet[] = []
    for (const row of parsed) {
      if (!row || typeof row !== 'object') continue
      const rec = row as Record<string, unknown>
      const seed = typeof rec.seed === 'string' ? normalizePhrase(rec.seed) : ''
      const address = typeof rec.address === 'string' ? rec.address : ''
      const label = typeof rec.label === 'string' ? rec.label : ''
      if (!seed || !address) continue
      out.push({ seed, address, label })
    }
    return out
  } catch {
    return []
  }
}

export function readWallets(): StoredWallet[] {
  return parseWallets(localStorage.getItem(WALLETS_KEY))
}

function writeWallets(wallets: StoredWallet[]): void {
  localStorage.setItem(WALLETS_KEY, JSON.stringify(wallets))
}

export function addWallet(wallet: StoredWallet): StoredWallet[] {
  const seed = normalizePhrase(wallet.seed)
  const next = readWallets().filter((w) => w.seed !== seed)
  next.push({
    seed,
    address: wallet.address,
    label: wallet.label.trim() || 'локальный',
  })
  writeWallets(next)
  return next
}

export function removeWallet(seed: string): StoredWallet[] {
  const normalized = normalizePhrase(seed)
  const next = readWallets().filter((w) => w.seed !== normalized)
  writeWallets(next)
  if (readActiveSeed() === normalized) setActiveSeed(null)
  return next
}

export function readActiveSeed(): string | null {
  const raw = localStorage.getItem(WALLET_ACTIVE_KEY)
  if (!raw) return null
  const seed = normalizePhrase(raw)
  return seed || null
}

export function setActiveSeed(seed: string | null): void {
  if (!seed) {
    localStorage.removeItem(WALLET_ACTIVE_KEY)
    return
  }
  localStorage.setItem(WALLET_ACTIVE_KEY, normalizePhrase(seed))
}
