import { useCallback, useEffect, useState } from 'react'
import {
  Coins,
  History,
  LogOut,
  Send,
  Shield,
  TrendingUp,
  Users,
  Droplets,
} from 'lucide-react'

import { SELECTED_ADDRESS_KEY } from './config'
import { useBackend } from './adapters'
import { BalancePanel } from './components/BalancePanel'
import { ConnectScreen } from './components/ConnectScreen'
import { FaucetPanel } from './components/FaucetPanel'
import { HistoryPanel } from './components/HistoryPanel'
import { MarketPanel } from './components/MarketPanel'
import { RolePanel } from './components/RolePanel'
import { SendPanel } from './components/SendPanel'
import { StakingPanel } from './components/StakingPanel'
import type { Feedback, TabId } from './types'
import { formatAddr } from './components/ui'

const TABS: Array<{ id: TabId; label: string; icon: typeof Coins; simOnly?: boolean; tip: string }> = [
  { id: 'balance', label: 'Баланс', icon: Coins, tip: 'WRT, LZN, ANT и подтверждение ZKP (§3.1).' },
  { id: 'send', label: 'Перевод', icon: Send, tip: 'Перевод WRT/LZN. Прямой ANT узел отклонит — только рынок (§4.1).' },
  { id: 'history', label: 'История', icon: History, tip: 'Транзакции этого адреса из блоков симуляции.' },
  { id: 'role', label: 'Роль', icon: Users, tip: 'Гражданин / поставщик / валидатор. Канон проверяет узел в блоке.' },
  { id: 'staking', label: 'Стейкинг', icon: Shield, tip: 'activate_lzn и declare (§5.4). Только валидатор получит награду.' },
  { id: 'market', label: 'Рынок', icon: TrendingUp, tip: 'Внутренний рынок ANT: BUY — валидатор, SELL — поставщик (§5.2).' },
  { id: 'faucet', label: 'Faucet', icon: Droplets, tip: 'Mint из казны симуляции. ANT гражданину узел отклонит.', simOnly: true },
]

export default function App() {
  const { mode, adapter, state, connected, error, refresh } = useBackend()
  const [address, setAddress] = useState<string | null>(null)
  const [tab, setTab] = useState<TabId>('balance')
  const [pending, setPending] = useState(false)
  const [feedback, setFeedback] = useState<Feedback | null>(null)

  useEffect(() => {
    const saved = localStorage.getItem(SELECTED_ADDRESS_KEY)
    if (saved) setAddress(saved)
  }, [])

  const connect = (addr: string) => {
    localStorage.setItem(SELECTED_ADDRESS_KEY, addr)
    setAddress(addr)
    setFeedback(null)
    setTab('balance')
  }

  const disconnect = () => {
    localStorage.removeItem(SELECTED_ADDRESS_KEY)
    setAddress(null)
    setFeedback(null)
  }

  const run = useCallback(
    async (body: Record<string, unknown>) => {
      setPending(true)
      setFeedback(null)
      try {
        const res = await adapter.submit(body)
        setFeedback({
          ok: res.accepted,
          text: res.accepted
            ? `Accepted${res.tx_hash ? ` · ${res.tx_hash.slice(0, 14)}…` : ''}`
            : res.message || 'Rejected',
        })
      } catch {
        setFeedback({ ok: false, text: 'Network error' })
      } finally {
        setPending(false)
      }
    },
    [adapter],
  )

  const isSim = mode === 'sim'
  const visibleTabs = TABS.filter((t) => !t.simOnly || isSim)

  if (!address) {
    return (
      <ConnectScreen
        state={state}
        error={error}
        onConnect={connect}
        onRefresh={refresh}
      />
    )
  }

  const account = state?.accounts[address]
  const treasury = state?.sim_treasury

  if (isSim && treasury && address === treasury) {
    return (
      <div className="mx-auto max-w-lg px-4 py-16 text-center">
        <p className="text-[var(--copper)]">
          Выбран служебный адрес казначейства симуляции — пользовательский кошелёк недоступен.
        </p>
        <button type="button" className="mt-4 text-[var(--mint)] underline" onClick={disconnect}>
          Выбрать другой адрес
        </button>
      </div>
    )
  }

  return (
    <div className="mx-auto min-h-screen max-w-5xl px-4 py-6">
      <header className="rise mb-6 flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className="text-xs font-semibold uppercase tracking-[0.2em] text-[var(--mint)]">
            Volnix {isSim ? 'Simulation' : 'Blockchain'}
          </p>
          <h1 className="mt-1 text-3xl font-bold tracking-tight">
            {isSim ? 'Sim Wallet' : 'Wallet'}
          </h1>
          <p className="mono mt-2 break-all text-sm text-[var(--muted)]">{address}</p>
        </div>
        <div className="flex flex-wrap items-center gap-3 text-sm">
          <span
            className="inline-flex items-center gap-2 rounded-full border border-[var(--line)] px-3 py-1.5"
            data-tip={isSim
              ? 'Живое состояние через WebSocket /ws. Если offline — данные только из REST.'
              : 'Опрос RPC-ноды блокчейна. Offline — нода недоступна.'}
          >
            <span
              className={`h-2 w-2 rounded-full ${connected ? 'bg-[var(--mint)] pulse-dot' : 'bg-[var(--danger)]'}`}
            />
            {isSim
              ? connected ? 'WS live' : 'WS offline'
              : connected ? 'RPC live' : 'RPC offline'}
          </span>
          <span
            className="mono rounded-full border border-[var(--line)] px-3 py-1.5 text-[var(--muted)]"
            data-tip="Высота последнего блока. Растёт, пока движок симуляции запущен."
          >
            h {state?.height ?? '—'}
          </span>
          {account && (
            <span
              className="rounded-full border border-[var(--line)] px-3 py-1.5 capitalize text-[var(--sand)]"
              data-tip="Тип кошелька §4.2: гражданин (WRT/LZN), поставщик (продаёт ANT), валидатор (покупает ANT, майнит)."
            >
              {account.role}
            </span>
          )}
          <button
            type="button"
            onClick={disconnect}
            data-tip="Сбросить выбранный адрес. Ключи и симуляция не удаляются."
            className="inline-flex items-center gap-1.5 rounded-full border border-[var(--line)] px-3 py-1.5 text-[var(--muted)] hover:text-[var(--sand)]"
          >
            <LogOut size={14} />
            Disconnect
          </button>
        </div>
      </header>

      <nav className="mb-5 flex flex-wrap gap-2">
        {visibleTabs.map(({ id, label, icon: Icon, tip }) => {
          const active = tab === id
          return (
            <button
              key={id}
              type="button"
              data-tip={tip}
              onClick={() => setTab(id)}
              className={`inline-flex items-center gap-2 rounded-xl px-3.5 py-2 text-sm font-medium transition ${
                active
                  ? 'bg-[var(--mint)] text-[#062019]'
                  : 'border border-[var(--line)] text-[var(--muted)] hover:text-[var(--sand)]'
              }`}
            >
              <Icon size={15} />
              {label}
            </button>
          )
        })}
      </nav>

      {!account ? (
        <div className="rounded-2xl border border-dashed border-[var(--line)] p-8 text-center text-[var(--muted)]">
          Аккаунт {formatAddr(address)} ещё не в состоянии цепочки.
          {isSim ? ' Создайте через faucet/operator или дождитесь блока.' : ' Пополните баланс.'}
          <div className="mt-4">
            <button type="button" className="text-[var(--mint)] underline" onClick={() => void refresh()}>
              Обновить
            </button>
          </div>
        </div>
      ) : (
        <>
          {tab === 'balance' && (
            <BalancePanel
              account={account}
              pending={pending}
              feedback={feedback}
              isSim={isSim}
              onVerifyZkp={() => void run({ op: 'verify_zkp', address })}
            />
          )}
          {tab === 'send' && (
            <SendPanel
              address={address}
              pending={pending}
              feedback={feedback}
              isSim={isSim}
              onSend={(payload) => void run({ op: 'transfer', ...payload })}
            />
          )}
          {tab === 'history' && <HistoryPanel address={address} height={state?.height ?? 0} />}
          {tab === 'role' && (
            <RolePanel
              account={account}
              pending={pending}
              feedback={feedback}
              isSim={isSim}
              genesisValidator={state?.genesis_validator}
              genesisProvider={state?.genesis_provider}
              onSetRole={(role) => void run({ op: 'set_role', address, role })}
            />
          )}
          {tab === 'staking' && (
            <StakingPanel
              account={account}
              pending={pending}
              feedback={feedback}
              onActivate={(amount) => void run({ op: 'activate_lzn', address, amount })}
              onDeclare={(burn_b, stake_s) => void run({ op: 'declare', address, burn_b, stake_s })}
            />
          )}
          {tab === 'market' && state?.market && (
            <MarketPanel
              address={address}
              account={account}
              market={state.market}
              height={state.height}
              pending={pending}
              feedback={feedback}
              onCreateOrder={(payload) => void run(payload)}
              onCancelOrder={(order_id) => void run({ op: 'cancel_order', address, order_id })}
            />
          )}
          {tab === 'faucet' && isSim && (
            <FaucetPanel
              address={address}
              onDone={(fb) => setFeedback(fb)}
            />
          )}
        </>
      )}
    </div>
  )
}
