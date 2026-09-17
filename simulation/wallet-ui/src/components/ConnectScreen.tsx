import { useMemo, useState } from 'react'
import { Key, Plus, Wallet } from 'lucide-react'

import { useBackend, type BackendMode } from '../adapters'
import type { Account } from '../types'
import type { WalletState } from '../adapters/types'
import { Btn, Field, TextInput, TextSelect, formatAddr, formatAmt } from './ui'

interface ConnectScreenProps {
  state: WalletState | null
  error: string | null
  onConnect: (address: string) => void
  onRefresh: () => Promise<void>
}

export function ConnectScreen({ state, error, onConnect, onRefresh }: ConnectScreenProps) {
  const { mode, switchMode, connectChain } = useBackend()

  const [selected, setSelected] = useState('')
  const [mnemonic, setMnemonic] = useState('')
  const [busy, setBusy] = useState(false)
  const [localError, setLocalError] = useState<string | null>(null)
  const { adapter } = useBackend()

  const isSim = mode === 'sim'

  const accounts = useMemo(() => {
    if (!state) return [] as Account[]
    const treasury = state.sim_treasury
    return Object.values(state.accounts)
      .filter((a) => a.address !== treasury)
      .sort((a, b) => a.address.localeCompare(b.address))
  }, [state])

  const createAccount = async () => {
    if (!adapter.createAccounts) return
    setBusy(true)
    setLocalError(null)
    try {
      const addrs = await adapter.createAccounts(1)
      await onRefresh()
      const addr = addrs[0]
      if (addr) {
        setSelected(addr)
        onConnect(addr)
      }
    } catch (err) {
      setLocalError(err instanceof Error ? err.message : 'Create failed')
    } finally {
      setBusy(false)
    }
  }

  const importMnemonic = async () => {
    setBusy(true)
    setLocalError(null)
    try {
      const address = await connectChain(mnemonic.trim())
      onConnect(address)
    } catch (err) {
      setLocalError(err instanceof Error ? err.message : 'Failed to connect')
    } finally {
      setBusy(false)
    }
  }

  const handleModeToggle = (m: BackendMode) => {
    switchMode(m)
    setSelected('')
    setMnemonic('')
    setLocalError(null)
  }

  return (
    <div className="mx-auto flex min-h-screen max-w-xl flex-col justify-center px-4 py-10">
      <div className="rise space-y-6 rounded-3xl border border-[var(--line)] bg-[color-mix(in_oklab,var(--panel)_94%,black)] p-8 shadow-[0_30px_80px_rgba(0,0,0,0.35)]">
        <div>
          <p className="text-xs font-semibold uppercase tracking-[0.2em] text-[var(--mint)]">
            Volnix {isSim ? 'Simulation' : 'Blockchain'}
          </p>
          <h1 className="mt-2 text-4xl font-bold tracking-tight text-[var(--sand)]">
            {isSim ? 'Sim Wallet' : 'Wallet'}
          </h1>
          <p className="mt-3 text-[var(--muted)]">
            {isSim
              ? 'Off-chain симулятор — адрес из FastAPI :8000. Без мнемоник.'
              : 'Подключение к блокчейну Volnix через CosmJS. Нужна мнемоника.'}
          </p>
        </div>

        {/* ── Mode toggle ── */}
        <div className="flex rounded-xl border border-[var(--line)] p-1">
          {(
            [
              ['sim', 'Симуляция'],
              ['chain', 'Блокчейн'],
            ] as const
          ).map(([m, label]) => (
            <button
              key={m}
              type="button"
              data-tip={m === 'sim'
                ? 'Off-chain FastAPI :8000. Адрес выбирается из состояния, без мнемоники.'
                : 'Реальная нода CosmJS (RPC :26657). Нужна BIP-39 мнемоника.'}
              onClick={() => handleModeToggle(m)}
              className={`flex-1 rounded-lg px-4 py-2 text-sm font-semibold transition ${
                mode === m
                  ? 'bg-[var(--mint)] text-[#062019]'
                  : 'text-[var(--muted)] hover:text-[var(--sand)]'
              }`}
            >
              {label}
            </button>
          ))}
        </div>

        {(error || localError) && (
          <div className="rounded-xl bg-[color-mix(in_oklab,var(--danger)_18%,transparent)] px-3 py-2 text-sm text-[var(--danger)]">
            {localError || error}
          </div>
        )}

        {/* ── Simulation connect ── */}
        {isSim && (
          <>
            <Field label="Аккаунт симуляции" tip="Адреса из GET /api/state, кроме казначейства симуляции.">
              <TextSelect value={selected} onChange={(e) => setSelected(e.target.value)}>
                <option value="">Выберите адрес…</option>
                {accounts.map((a) => (
                  <option key={a.address} value={a.address}>
                    {formatAddr(a.address, 14, 8)} · {a.role} · WRT {formatAmt(a.wrt_balance, 2)}
                  </option>
                ))}
              </TextSelect>
            </Field>

            <div className="flex flex-wrap gap-3">
              <Btn
                disabled={!selected || busy}
                data-tip="Открыть кошелёк выбранного адреса симуляции."
                onClick={() => selected && onConnect(selected)}
                className="inline-flex items-center gap-2"
              >
                <Wallet size={16} />
                Подключить
              </Btn>
              <Btn variant="ghost" disabled={busy} data-tip="Создать bot/demo адрес через sim-operator и сразу подключить." onClick={() => void createAccount()} className="inline-flex items-center gap-2">
                <Plus size={16} />
                Создать аккаунт
              </Btn>
              <Btn variant="ghost" disabled={busy} onClick={() => void onRefresh()}>
                Обновить список
              </Btn>
            </div>
          </>
        )}

        {/* ── Chain connect ── */}
        {!isSim && (
          <>
            <Field label="Мнемоника (BIP-39, 12 или 24 слова)" tip="Только в браузере. На сервер не уходит. Деривация адреса с префиксом volnix.">
              <TextInput
                className="mono text-xs"
                placeholder="word1 word2 word3 …"
                value={mnemonic}
                onChange={(e) => setMnemonic(e.target.value)}
              />
            </Field>

            <p className="text-xs text-[var(--muted)]">
              Ключ используется только локально (в браузере) и не отправляется на сервер. Кошелёк
              подключается напрямую к ноде через RPC.
            </p>

            <div className="flex flex-wrap gap-3">
              <Btn
                disabled={!mnemonic.trim() || busy}
                onClick={() => void importMnemonic()}
                className="inline-flex items-center gap-2"
              >
                <Key size={16} />
                Импортировать
              </Btn>
            </div>
          </>
        )}

        {isSim && state && (
          <p className="mono text-xs text-[var(--muted)]">
            height {state.height} · accounts {state.accounts_count}
            {state.mempool_size != null ? ` · mempool ${state.mempool_size}` : ''}
          </p>
        )}
      </div>
    </div>
  )
}
