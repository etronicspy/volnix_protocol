import { API_BASE, SCALE } from '../config'

export class ApiError extends Error {
  constructor(message: string) {
    super(message)
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers: {
      Accept: 'application/json',
      ...(init?.body ? { 'Content-Type': 'application/json' } : {}),
      ...init?.headers,
    },
  })
  if (!res.ok) {
    let detail = res.statusText
    try {
      const body = (await res.json()) as { detail?: string }
      if (body.detail) detail = typeof body.detail === 'string' ? body.detail : JSON.stringify(body.detail)
    } catch {
      /* ignore */
    }
    throw new ApiError(detail || `HTTP ${res.status}`)
  }
  return res.json() as Promise<T>
}

export const api = {
  get: <T>(path: string) => request<T>(path),
}

export function toNumber(v: number | string | undefined | null): number {
  if (v === undefined || v === null || v === '') return 0
  const n = typeof v === 'string' ? Number(v) : v
  return Number.isFinite(n) ? n : 0
}

/** Format micro-units for display. */
export function formatMicro(micro: number | string | undefined | null, digits = 6): string {
  if (micro === null || micro === undefined || micro === '') return '—'
  const n = typeof micro === 'string' ? Number(micro) : micro
  if (!Number.isFinite(n)) return '—'
  return (n / SCALE).toLocaleString(undefined, {
    minimumFractionDigits: 0,
    maximumFractionDigits: digits,
  })
}

export function formatTime(iso: string | undefined | null): string {
  if (!iso) return '—'
  return iso.replace('T', ' ').replace('Z', '')
}

export function shortHash(hash: string | null | undefined, head = 8, tail = 4): string {
  if (!hash) return '—'
  if (hash.length <= head + tail + 1) return hash
  return `${hash.slice(0, head)}…${hash.slice(-tail)}`
}

export function shortAddress(addr: string | null | undefined): string {
  return shortHash(addr, 10, 6)
}
