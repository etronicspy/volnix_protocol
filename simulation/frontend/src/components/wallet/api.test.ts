import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { fetchOpenOrders, submitWallet } from './api'

const realFetch = global.fetch

describe('wallet api', () => {
  beforeEach(() => {
    global.fetch = vi.fn()
  })

  afterEach(() => {
    global.fetch = realFetch
  })

  it('submitWallet returns accepted=true on 200', async () => {
    ;(global.fetch as unknown as ReturnType<typeof vi.fn>).mockResolvedValue({
      ok: true,
      json: async () => ({ accepted: true, message: 'queued', tx_hash: 'h1' }),
    })
    const r = await submitWallet('http://x', { op: 'transfer' })
    expect(r.accepted).toBe(true)
    expect(r.tx_hash).toBe('h1')
  })

  it('submitWallet returns accepted=false on non-2xx', async () => {
    ;(global.fetch as unknown as ReturnType<typeof vi.fn>).mockResolvedValue({
      ok: false,
      status: 500,
      json: async () => ({}),
    })
    const r = await submitWallet('http://x', { op: 'transfer' })
    expect(r.accepted).toBe(false)
    expect(r.message).toContain('500')
  })

  it('fetchOpenOrders returns [] on non-2xx', async () => {
    ;(global.fetch as unknown as ReturnType<typeof vi.fn>).mockResolvedValue({
      ok: false,
      json: async () => ({}),
    })
    const r = await fetchOpenOrders('http://x', 'alice')
    expect(r.orders).toEqual([])
  })
})
