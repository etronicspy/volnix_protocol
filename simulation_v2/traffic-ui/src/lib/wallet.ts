import { SCALE } from '../config'
import { api } from './api'
import type {
  AccountDetail,
  BroadcastResult,
  ChainParams,
  MintResult,
  OperatorAccount,
  TxMessage,
} from '../types/api'

export async function unlockAccount(seed: string): Promise<OperatorAccount> {
  const out = await api.post<OperatorAccount>('/api/v1/operator/account', { seed })
  if (!out.address) throw new Error('нода не вернула адрес')
  return out
}

export async function fetchAccount(address: string): Promise<AccountDetail> {
  return api.get<AccountDetail>(`/api/v1/accounts/${encodeURIComponent(address)}`)
}

export async function fetchParams(): Promise<ChainParams> {
  return api.get<ChainParams>('/api/v1/params')
}

function assertBroadcast(result: BroadcastResult): string {
  if (result.code !== 0) {
    throw new Error(result.log || `отклонено (code ${result.code})`)
  }
  return result.hash
}

export async function verifyRole(seed: string, desiredRole: string, zkpProof: string): Promise<string> {
  const out = await api.post<BroadcastResult>('/api/v1/operator/verify', {
    seed,
    desired_role: desiredRole,
    zkp_proof: zkpProof,
  })
  return assertBroadcast(out)
}

export async function declareBurn(seed: string, bI: number, sI: number): Promise<string> {
  const out = await api.post<BroadcastResult>('/api/v1/operator/declare', {
    seed,
    b_i: bI,
    s_i: sI,
  })
  return assertBroadcast(out)
}

export async function signedTx(seed: string, messages: TxMessage[]): Promise<string> {
  const q = encodeURIComponent(seed)
  const out = await api.post<BroadcastResult>(`/api/v1/operator/tx?seed=${q}`, messages)
  return assertBroadcast(out)
}

export async function mintWrt(address: string, amountMicro: number): Promise<string> {
  const out = await api.post<MintResult>('/api/v1/operator/mint', {
    address,
    denom: 'uwrt',
    amount: amountMicro,
  })
  const broadcast = out.broadcast
  if (!broadcast) throw new Error('нода не вернула broadcast')
  return assertBroadcast(broadcast)
}

/** f_i = floor(α · L_i) in micro-ANT. */
export function entryBurn(lI: number, alphaNum: number, alphaDen: number): number {
  if (lI <= 0 || alphaDen <= 0) return 0
  return Math.floor((alphaNum * lI) / alphaDen)
}

/** Whole tokens → micro-units. Empty or invalid returns null. */
export function tokensToMicro(input: string): number | null {
  const t = input.trim().replace(',', '.')
  if (!t) return null
  const n = Number(t)
  if (!Number.isFinite(n) || n < 0) return null
  return Math.round(n * SCALE)
}
