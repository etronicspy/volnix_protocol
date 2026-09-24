import { API_BASE, SCALE, TRAFFIC_BASE } from '../config'

export class ApiError extends Error {
  status: number
  constructor(message: string, status: number) {
    super(message)
    this.status = status
  }
}

async function request<T>(base: string, path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${base}${path}`, {
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
    throw new ApiError(detail || `HTTP ${res.status}`, res.status)
  }
  return res.json() as Promise<T>
}

export const api = {
  get: <T>(path: string) => request<T>(API_BASE, path),
  post: <T>(path: string, body?: unknown) =>
    request<T>(API_BASE, path, {
      method: 'POST',
      body: body === undefined ? undefined : JSON.stringify(body),
    }),
}

/** Control plane for simulation_v2/traffic (default :8002). */
export const trafficApi = {
  get: <T>(path: string) => request<T>(TRAFFIC_BASE, path),
  post: <T>(path: string, body?: unknown) =>
    request<T>(TRAFFIC_BASE, path, {
      method: 'POST',
      body: body === undefined ? undefined : JSON.stringify(body),
    }),
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

export function shortHash(hash: string | null | undefined, head = 8, tail = 4): string {
  if (!hash) return '—'
  if (hash.length <= head + tail + 1) return hash
  return `${hash.slice(0, head)}…${hash.slice(-tail)}`
}

export function shortAddress(addr: string | null | undefined): string {
  return shortHash(addr, 10, 6)
}
