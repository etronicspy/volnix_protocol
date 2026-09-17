import { ShieldCheck } from 'lucide-react'

import type { Account, Feedback } from '../types'
import { Btn, FeedbackBanner, Panel, formatAmt } from './ui'

interface BalancePanelProps {
  account: Account
  pending: boolean
  feedback: Feedback | null
  isSim: boolean
  onVerifyZkp: () => void
}

export function BalancePanel({ account, pending, feedback, isSim, onVerifyZkp }: BalancePanelProps) {
  const frozen = account.lzn_frozen_mining ?? 0
  const zkpOk = Boolean(account.zkp_verified)

  const tiles = [
    { label: 'WRT', value: formatAmt(account.wrt_balance), tone: 'text-[var(--mint)]', tip: 'Wert — расчётный токен. Переводы и эскроу BUY на рынке ANT.' },
    { label: 'LZN liquid', value: formatAmt(account.lzn_balance), tone: 'text-[var(--copper)]', tip: 'Lizenz не заморожен. Нужен валидатору, чтобы активировать под майнинг.' },
    { label: 'LZN activated', value: formatAmt(frozen), tone: 'text-[#f0c674]', tip: 'LZN в майнинге (L_i). Доля награды ∝ L_i. Потолок на адрес — ⌊10000/3⌋.' },
    { label: 'ANT', value: formatAmt(account.ant_balance), tone: 'text-[#9ecbff]', tip: 'Anteil только у поставщика и валидатора. У гражданина узел сжигает ANT каждый блок.' },
  ]

  return (
    <Panel
      title="Баланс"
      hint={isSim ? 'Данные из GET /api/state и WebSocket /ws.' : 'Данные из RPC-ноды блокчейна.'}
    >
      {feedback ? <div className="mb-4"><FeedbackBanner {...feedback} /></div> : null}

      <div className="mb-6 rounded-2xl bg-[linear-gradient(135deg,#12352d,#0c221d_55%,#1a3d30)] p-6">
        <p className="text-sm text-[var(--muted)]">Основной актив</p>
        <p className="mono mt-1 text-4xl font-semibold text-[var(--mint)]">
          {formatAmt(account.wrt_balance)} <span className="text-lg text-[var(--muted)]">WRT</span>
        </p>
        <p className="mt-3 text-sm text-[var(--sand)]">
          Роль: <strong className="capitalize">{account.role}</strong>
          {' · '}
          ZKP: {zkpOk ? 'подтверждён' : 'нет'}
        </p>
      </div>

      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        {tiles.map((t) => (
          <div
            key={t.label}
            className="rounded-xl border border-[var(--line)] bg-[var(--panel-2)] p-4"
            data-tip={t.tip}
          >
            <div className="text-xs uppercase tracking-wide text-[var(--muted)]">{t.label}</div>
            <div className={`mono mt-1 text-xl font-medium ${t.tone}`}>{t.value}</div>
          </div>
        ))}
      </div>

      <div className="mt-6 flex flex-wrap items-center gap-3">
        <Btn
          disabled={pending}
          onClick={onVerifyZkp}
          data-tip="Симуляция ZKP (§3.1). Без флага узел не применит роль поставщика или валидатора (кроме genesis)."
          className="inline-flex items-center gap-2"
        >
          <ShieldCheck size={16} />
          {zkpOk ? 'Повторить verify_zkp' : 'Подтвердить ZKP'}
        </Btn>
        <p className="text-xs text-[var(--muted)]">
          {isSim
            ? 'Tx verify_zkp → мемпул симулятора (§3.1).'
            : 'Tx verify_zkp → блокчейн Volnix (x/ident).'}
        </p>
      </div>
    </Panel>
  )
}
