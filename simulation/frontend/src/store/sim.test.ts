import { describe, expect, it } from 'vitest'

import type { SimulatorApiState } from '../types'
import { INITIAL_STATE, applyBlockDelta, mergeFromSimulatorApi } from './sim'

const emptyMarket = { bids: [], asks: [], last_price: 0, history: [] }

const baseSnapshot: SimulatorApiState = {
  height: 5,
  accounts_count: 2,
  accounts: {
    alice: {
      address: 'alice',
      wrt_balance: 100,
      lzn_balance: 0,
      ant_balance: 0,
      role: 'citizen',
    },
    bob: {
      address: 'bob',
      wrt_balance: 50,
      lzn_balance: 0,
      ant_balance: 0,
      role: 'provider',
    },
  },
  market: emptyMarket,
  blocks: [],
  tps_history: [],
  mempool_size: 3,
  blocks_per_epoch: 100,
}

describe('mergeFromSimulatorApi', () => {
  it('replaces accounts and updates height + mempool', () => {
    const next = mergeFromSimulatorApi(INITIAL_STATE, baseSnapshot, 1.5, { recent: 'keep' })
    expect(next.block_height).toBe(5)
    expect(next.block_time).toBe(1.5)
    expect(next.accounts.alice.wrt_balance).toBe(100)
    expect(next.mempool_size).toBe(3)
    expect(next.blocks_per_epoch).toBe(100)
  })

  it('clears recent_txs on full reset snapshot', () => {
    const prev = {
      ...INITIAL_STATE,
      recent_txs: [{
        tx_hash: 'h1', tx_type: 'transfer', sender: 'x', receiver: 'y',
        amount: 1, price: 0, role: 'citizen', timestamp: 0,
      }],
    }
    const next = mergeFromSimulatorApi(prev, baseSnapshot, 1, { recent: 'clear' })
    expect(next.recent_txs).toEqual([])
  })

  it('prepends new block txs when recent=append', () => {
    const prev = { ...INITIAL_STATE, recent_txs: [] }
    const block_tx = {
      tx_hash: 'new', tx_type: 'mint', sender: 'sim_treasury',
      receiver: 'alice', amount: 100, price: 0, role: 'system', timestamp: 1,
    }
    const next = mergeFromSimulatorApi(prev, baseSnapshot, 1, {
      recent: 'append',
      blockTxs: [block_tx as unknown as Record<string, unknown>],
    })
    expect(next.recent_txs[0].tx_hash).toBe('new')
  })
})

describe('applyBlockDelta', () => {
  it('merges changed accounts and prepends txs', () => {
    const prev = {
      ...INITIAL_STATE,
      accounts: {
        alice: {
          address: 'alice', wrt_balance: 100, lzn_balance: 0, ant_balance: 0, role: 'citizen',
        },
      },
      accounts_count: 1,
    }
    const next = applyBlockDelta(prev, {
      block_time: 0.5,
      block: {
        height: 7,
        hash: 'h7',
        tx_count: 1,
        timestamp: Date.now(),
        transactions: [{
          tx_hash: 'tx1', tx_type: 'transfer', sender: 'alice', receiver: 'bob',
          amount: 5, price: 0, role: 'citizen', timestamp: 0,
        }],
      },
      delta: {
        accounts_changed: {
          alice: { address: 'alice', wrt_balance: 95, lzn_balance: 0, ant_balance: 0, role: 'citizen' },
          bob: { address: 'bob', wrt_balance: 5, lzn_balance: 0, ant_balance: 0, role: 'citizen' },
        },
      },
    })
    expect(next.block_height).toBe(7)
    expect(next.accounts.alice.wrt_balance).toBe(95)
    expect(next.accounts.bob.wrt_balance).toBe(5)
    expect(next.recent_txs[0].tx_hash).toBe('tx1')
    expect(next.blocks).toHaveLength(1)
  })

  it('removes accounts via accounts_removed', () => {
    const prev = {
      ...INITIAL_STATE,
      accounts: {
        alice: { address: 'alice', wrt_balance: 1, lzn_balance: 0, ant_balance: 0, role: 'citizen' },
        bob: { address: 'bob', wrt_balance: 1, lzn_balance: 0, ant_balance: 0, role: 'citizen' },
      },
    }
    const next = applyBlockDelta(prev, {
      block: { height: 1 },
      delta: { accounts_removed: ['bob'] },
    })
    expect(next.accounts).toHaveProperty('alice')
    expect(next.accounts).not.toHaveProperty('bob')
  })

  it('resets blocks tape on block.height === 0', () => {
    const prev = {
      ...INITIAL_STATE,
      blocks: [
        { height: 1, hash: 'h1', tx_count: 0, timestamp: 0, transactions: [] },
        { height: 2, hash: 'h2', tx_count: 0, timestamp: 0, transactions: [] },
      ],
      recent_txs: [{
        tx_hash: 'old', tx_type: 'transfer', sender: 'x', receiver: 'y',
        amount: 1, price: 0, role: 'citizen', timestamp: 0,
      }],
    }
    const next = applyBlockDelta(prev, {
      block: { height: 0, hash: 'genesis', tx_count: 1, timestamp: 0, transactions: [] },
    })
    expect(next.blocks).toHaveLength(1)
    expect(next.blocks[0].height).toBe(0)
    expect(next.recent_txs).toEqual([])
  })
})
