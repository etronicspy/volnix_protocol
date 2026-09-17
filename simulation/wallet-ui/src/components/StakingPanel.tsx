import { useState } from 'react'

import { LZN_MAX_FROZEN_PER_ADDRESS, LZN_TOTAL_SUPPLY_REF } from '../config'
import type { Account, Feedback } from '../types'
import { Btn, FeedbackBanner, Field, Panel, TextInput, formatAmt } from './ui'

interface StakingPanelProps {
  account: Account
  pending: boolean
  feedback: Feedback | null
  onActivate: (amount: number) => void
  onDeclare: (burn_b: number, stake_s: number) => void
}

export function StakingPanel({
  account,
  pending,
  feedback,
  onActivate,
  onDeclare,
}: StakingPanelProps) {
  const frozen = account.lzn_frozen_mining ?? 0
  const maxMore = Math.max(0, LZN_MAX_FROZEN_PER_ADDRESS - frozen)
  const [amount, setAmount] = useState('1')
  const [burn, setBurn] = useState('1')
  const [stake, setStake] = useState('1')
  return (
    <Panel
      title="Стейкинг / майнинг"
      hint={`Активация LZN (потолок узла ${LZN_MAX_FROZEN_PER_ADDRESS} = ⌊${LZN_TOTAL_SUPPLY_REF}/3⌋) и declare. Отклонения — от бэкенда.`}
    >
      {feedback ? <div className="mb-4"><FeedbackBanner {...feedback} /></div> : null}

      <div className="grid gap-6 lg:grid-cols-2">
        <div className="space-y-3 rounded-2xl border border-[var(--line)] bg-[var(--panel-2)] p-4">
          <h3 className="font-semibold" data-tip="Заморозка ликвидного LZN в L_i. Узел применит только роли валидатор и в пределах потолка.">Активировать LZN</h3>
          <p className="text-sm text-[var(--muted)]">
            Ликвид: {formatAmt(account.lzn_balance)} · уже активно: {formatAmt(frozen)} · можно ещё:{' '}
            {formatAmt(maxMore)}
          </p>
          <Field label="Сумма">
            <TextInput type="number" min={0} step="any" value={amount} onChange={(e) => setAmount(e.target.value)} />
          </Field>
          <Btn
            data-tip="Tx activate_lzn. Невалидатор получит reject в блоке (§4.2)."
            disabled={pending || !(Number(amount) > 0)}
            onClick={() => onActivate(Number(amount))}
          >
            activate_lzn
          </Btn>
        </div>

        <div className="space-y-3 rounded-2xl border border-[var(--line)] bg-[var(--panel-2)] p-4">
          <h3 className="font-semibold" data-tip="Участие в блоке: b сжигается, s — ставка. Нужны роль валидатор, L_i и ANT.">Declare participation</h3>
          <p className="text-sm text-[var(--muted)]">
            ANT burn + stake для участия в наградах (§5.4). Баланс ANT: {formatAmt(account.ant_balance)}
          </p>
          <div className="grid grid-cols-2 gap-3">
            <Field label="Burn (b)" tip="Сжигание ANT. При b=0 за высоту нет ни базовой WRT (§5.1), ни доли комиссий F·(b/B) (§5.4).">
              <TextInput type="number" min={0} step="any" value={burn} onChange={(e) => setBurn(e.target.value)} />
            </Field>
            <Field label="Stake (s)" tip="Ставка ANT, не сжигается. Вес w_i = s / L_i влияет на отбор ValidatorSet.">
              <TextInput type="number" min={0} step="any" value={stake} onChange={(e) => setStake(e.target.value)} />
            </Field>
          </div>
          <Btn
            data-tip="Tx declare. Нужно b+s ≤ L_i и b+s ≤ баланс ANT. Иначе узел отклонит."
            disabled={pending || !(Number(burn) >= 0) || !(Number(stake) >= 0)}
            onClick={() => onDeclare(Number(burn), Number(stake))}
          >
            declare
          </Btn>
        </div>
      </div>
    </Panel>
  )
}
