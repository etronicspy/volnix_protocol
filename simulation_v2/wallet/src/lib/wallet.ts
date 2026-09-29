import { LOCAL_KEY, SESSION_KEY } from '../config'
import { api } from './api'
import type {
  AccountDetail,
  BroadcastResult,
  ChainParams,
  OperatorAccount,
  TxMessage,
} from '../types/api'

export interface WalletSession {
  seed: string
  address: string
  remembered: boolean
}

export function readStoredSeed(): { seed: string; remembered: boolean } | null {
  const local = localStorage.getItem(LOCAL_KEY)
  if (local) return { seed: local, remembered: true }
  const session = sessionStorage.getItem(SESSION_KEY)
  if (session) return { seed: session, remembered: false }
  return null
}

export function persistSeed(seed: string, remember: boolean): void {
  sessionStorage.setItem(SESSION_KEY, seed)
  if (remember) localStorage.setItem(LOCAL_KEY, seed)
  else localStorage.removeItem(LOCAL_KEY)
}

export function clearSeed(): void {
  sessionStorage.removeItem(SESSION_KEY)
  localStorage.removeItem(LOCAL_KEY)
}

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

/** f_i = floor(α · L_i) in micro-ANT. */
export function entryBurn(lI: number, alphaNum: number, alphaDen: number): number {
  if (lI <= 0 || alphaDen <= 0) return 0
  return Math.floor((alphaNum * lI) / alphaDen)
}
