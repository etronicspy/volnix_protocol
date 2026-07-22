/**
 * Zustand store: единое состояние симуляции (вместо локального useState в App.tsx).
 *
 * Хранит результаты REST `init` и инкрементальные обновления из WebSocket
 * (Этап 2 — `block_delta`, legacy `new_block`). Чистые редьюсеры можно тестировать
 * без React (см. `src/store/sim.test.ts`).
 */
import { create } from 'zustand'

import type {
  Account,
  Block,
  BlockDelta,
  Market,
  NetworkState,
  SimulatorApiState,
  Transaction,
} from '../types'

export const INITIAL_STATE: NetworkState = {
  status: 'Connecting...',
  block_height: 0,
  block_time: 0,
  accounts_count: 0,
  accounts: {},
  recent_txs: [],
  market: { bids: [], asks: [], last_price: 0, history: [] },
  blocks: [],
  tps_history: [],
  blocks_per_epoch: 10080,
  canon_log: [],
}

export interface SnapshotMergeOpts {
  recent: 'append' | 'keep' | 'clear'
  blockTxs?: Record<string, unknown>[]
}

/** Чистая функция мерджа полного snapshot (init / legacy new_block). */
export function mergeFromSimulatorApi(
  prev: NetworkState,
  snapshot: SimulatorApiState,
  blockTime: number,
  opts: SnapshotMergeOpts,
): NetworkState {
  const { recent, blockTxs = [] } = opts
  let recent_txs = prev.recent_txs
  if (recent === 'clear') {
    recent_txs = []
  } else if (recent === 'append') {
    recent_txs = [...(blockTxs as unknown as Transaction[]), ...prev.recent_txs].slice(0, 10)
  }
  return {
    ...prev,
    block_height: snapshot.height,
    block_time: blockTime,
    accounts_count: snapshot.accounts_count,
    mempool_size: snapshot.mempool_size,
    accounts: snapshot.accounts,
    market: snapshot.market,
    blocks: snapshot.blocks,
    tps_history: snapshot.tps_history,
    blocks_per_epoch: snapshot.blocks_per_epoch ?? prev.blocks_per_epoch,
    epoch_ant_sold_volume: snapshot.epoch_ant_sold_volume,
    epoch_ant_sold_last: snapshot.epoch_ant_sold_last,
    epoch_emission_coefficient: snapshot.epoch_emission_coefficient,
    genesis_validator: snapshot.genesis_validator,
    consensus_validators: snapshot.consensus_validators ?? prev.consensus_validators,
    next_proposer: snapshot.next_proposer ?? prev.next_proposer,
    genesis_provider: snapshot.genesis_provider,
    sim_treasury: snapshot.sim_treasury,
    canonical_block_interval_sec:
      snapshot.canonical_block_interval_sec ?? prev.canonical_block_interval_sec,
    sim_block_interval_sec: snapshot.sim_block_interval_sec ?? prev.sim_block_interval_sec,
    canon_log: snapshot.canon_log ?? prev.canon_log ?? [],
    recent_txs,
  }
}

export interface BlockDeltaPayload {
  block_time?: number
  block: Partial<Block> & { height?: number; transactions?: Record<string, unknown>[] }
  delta?: BlockDelta
  market?: Market
  consensus_validators?: NetworkState['consensus_validators']
  next_proposer?: string
  epoch_ant_sold_volume?: number
  epoch_emission_coefficient?: number
  mempool_size?: number
  tps_history_tail?: NetworkState['tps_history']
}

/** Чистая функция: применить incremental block_delta. */
export function applyBlockDelta(prev: NetworkState, data: BlockDeltaPayload): NetworkState {
  const block = data.block
  const txs = (block.transactions as Transaction[] | undefined) ?? []
  const delta = data.delta ?? {}
  const accounts: Record<string, Account> = { ...prev.accounts }
  for (const [addr, acc] of Object.entries(delta.accounts_changed ?? {})) {
    accounts[addr] = acc
  }
  for (const addr of delta.accounts_removed ?? []) {
    delete accounts[addr]
  }
  const isReset = (block.height ?? -1) === 0
  const blocks = isReset
    ? [block as unknown as Block]
    : [...prev.blocks, block as unknown as Block].slice(-10)
  const recent_txs = isReset ? [] : [...txs, ...prev.recent_txs].slice(0, 10)
  return {
    ...prev,
    block_height: block.height ?? prev.block_height,
    block_time: data.block_time ?? prev.block_time,
    accounts,
    accounts_count: Object.keys(accounts).length,
    market: data.market ?? prev.market,
    blocks,
    recent_txs,
    consensus_validators: data.consensus_validators ?? prev.consensus_validators,
    next_proposer: data.next_proposer ?? prev.next_proposer,
    epoch_ant_sold_volume: data.epoch_ant_sold_volume ?? prev.epoch_ant_sold_volume,
    epoch_emission_coefficient:
      data.epoch_emission_coefficient ?? prev.epoch_emission_coefficient,
    mempool_size: data.mempool_size ?? prev.mempool_size,
    tps_history: data.tps_history_tail ?? prev.tps_history,
  }
}

export interface SimStore {
  state: NetworkState
  setStatus: (s: string) => void
  applySnapshot: (snapshot: SimulatorApiState, blockTime: number, opts: SnapshotMergeOpts) => void
  applyDelta: (payload: BlockDeltaPayload) => void
  reset: () => void
}

export const useSimStore = create<SimStore>((set) => ({
  state: INITIAL_STATE,
  setStatus: (status) => set((s) => ({ state: { ...s.state, status } })),
  applySnapshot: (snapshot, blockTime, opts) =>
    set((s) => ({ state: mergeFromSimulatorApi(s.state, snapshot, blockTime, opts) })),
  applyDelta: (payload) => set((s) => ({ state: applyBlockDelta(s.state, payload) })),
  reset: () => set({ state: INITIAL_STATE }),
}))
