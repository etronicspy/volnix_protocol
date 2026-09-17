import { useCallback, useEffect, useState } from 'react'

import { LZN_MAX_FROZEN_PER_ADDRESS, LZN_TOTAL_SUPPLY_REF } from '../config'
import { formatPrice } from '../lib/format'

export interface WalletAccount {
  address: string
  wrt_balance: number
  lzn_balance: number
  lzn_frozen_mining?: number
  ant_balance: number
  role: string
  zkp_verified?: boolean
}

interface OpenOrder {
  id: string
  owner: string
  order_type: string
  price: number
  amount: number
  filled: number
  timestamp: number
}

interface WalletPanelProps {
  apiBase: string
  address: string
  account: WalletAccount | undefined
  blockHeight: number
  simTreasury?: string
  genesisValidator?: string
  genesisProvider?: string
}

async function submitWallet(
  apiBase: string,
  body: Record<string, unknown>
): Promise<{ accepted: boolean; message: string; tx_hash?: string }> {
  const res = await fetch(`${apiBase}/api/wallet/submit`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  return res.json()
}

export function WalletPanel({
  apiBase,
  address,
  account,
  blockHeight,
  simTreasury,
}: WalletPanelProps) {
  const [orders, setOrders] = useState<OpenOrder[]>([])
  const [feedback, setFeedback] = useState<{ ok: boolean; text: string } | null>(null)
  const [pending, setPending] = useState(false)

  const loadOrders = useCallback(() => {
    if (!address || address === simTreasury) {
      setOrders([])
      return
    }
    fetch(`${apiBase}/api/wallet/open-orders?address=${encodeURIComponent(address)}`)
      .then((r) => r.json())
      .then((d) => setOrders(d.orders || []))
      .catch(() => setOrders([]))
  }, [address, apiBase, simTreasury])

  useEffect(() => {
    loadOrders()
  }, [loadOrders, blockHeight])

  const run = async (body: Record<string, unknown>) => {
    setPending(true)
    setFeedback(null)
    try {
      const r = await submitWallet(apiBase, body)
      setFeedback({
        ok: r.accepted,
        text: r.accepted
          ? `Accepted → mempool. tx ${(r.tx_hash || '').slice(0, 12)}… (next block)`
          : r.message || 'Rejected',
      })
      if (r.accepted) loadOrders()
    } catch {
      setFeedback({ ok: false, text: 'Network error' })
    }
    setPending(false)
  }

  if (!address) {
    return (
      <div className="bg-gray-800/60 p-6 rounded-lg border border-dashed border-gray-600 text-gray-500 text-center">
        Выберите аккаунт и нажмите «Кошелёк», чтобы подписывать транзакции в мемпул (как в реальной цепочке).
      </div>
    )
  }

  if (address === simTreasury) {
    return (
      <div className="bg-gray-800 p-6 rounded-lg border border-amber-700/50 text-amber-200/90">
        Это служебный адрес казначейства симуляции — кошелёк протокола не используется для пользовательских tx.
      </div>
    )
  }

  if (!account) {
    return (
      <div className="bg-gray-800 p-6 rounded-lg border border-gray-700 text-gray-400">
        Аккаунт не найден в состоянии цепочки (возможно, только что создан — дождитесь блока или обновите).
      </div>
    )
  }

  const role = account.role
  const Lcap = LZN_MAX_FROZEN_PER_ADDRESS
  const frozen = account.lzn_frozen_mining ?? 0
  const zkpOk = Boolean(account.zkp_verified)

  return (
    <div className="bg-gray-800 p-6 rounded-lg border border-emerald-700/40 space-y-6">
      <div className="flex flex-wrap justify-between gap-4 border-b border-gray-700 pb-4">
        <div>
          <h2 className="text-xl font-bold text-emerald-300">Кошелёк</h2>
          <p className="font-mono text-sm text-blue-300 break-all mt-1">{address}</p>
          <p className="text-xs text-gray-500 mt-2">
            Транзакции попадают в мемпул; узел включает их в следующий блок или отклоняет по правилам протокола.
          </p>
        </div>
        <div className="grid grid-cols-2 sm:grid-cols-5 gap-3 text-sm">
          <div>
            <div className="text-gray-500 text-xs uppercase">WRT</div>
            <div className="font-mono text-green-400">{account.wrt_balance.toFixed(4)}</div>
          </div>
          <div>
            <div className="text-gray-500 text-xs uppercase">LZN ликв.</div>
            <div className="font-mono text-purple-300">{account.lzn_balance.toFixed(4)}</div>
          </div>
          <div>
            <div className="text-gray-500 text-xs uppercase">LZN акт.</div>
            <div className="font-mono text-amber-300">{frozen.toFixed(4)}</div>
          </div>
          <div>
            <div className="text-gray-500 text-xs uppercase">ANT</div>
            <div className="font-mono text-orange-300">{account.ant_balance.toFixed(4)}</div>
          </div>
          <div>
            <div className="text-gray-500 text-xs uppercase">ZKP</div>
            <div className={`font-semibold ${zkpOk ? 'text-emerald-400' : 'text-gray-500'}`}>
              {zkpOk ? '✓ да' : 'нет'}
            </div>
          </div>
        </div>
      </div>

      {feedback && (
        <div
          className={`px-4 py-2 rounded text-sm ${feedback.ok ? 'bg-emerald-900/40 text-emerald-200' : 'bg-red-900/40 text-red-200'}`}
        >
          {feedback.text}
        </div>
      )}

      {/* ZKP (симуляция) */}
      <section className="bg-gray-900/50 p-4 rounded border border-gray-700">
        <h3 className="text-sm font-semibold text-gray-300 mb-2">ZKP (симуляция)</h3>
        <p className="text-xs text-gray-500 mb-3">
          §3.1: без подтверждения ZKP нельзя стать Поставщиком или Валидатором
          (кроме seed-адреса genesis §6.3).
          Здесь — одна tx <code className="text-gray-400">verify_zkp</code> в мемпул.
        </p>
        <button
          type="button"
          disabled={pending}
          data-tip="Симуляция ZKP (§3.1). Без флага узел не применит роль поставщика или валидатора."
          onClick={() => run({ op: 'verify_zkp', address })}
          className="bg-slate-700 hover:bg-slate-600 disabled:opacity-40 px-4 py-2 rounded text-sm"
        >
          {zkpOk ? 'Повторить verify_zkp' : 'Подтвердить ZKP (tx)'}
        </button>
      </section>

      {/* Роль */}
      <section className="bg-gray-900/50 p-4 rounded border border-gray-700">
        <h3 className="text-sm font-semibold text-gray-300 mb-2">Смена роли (tx set_role)</h3>
        <p className="text-xs text-gray-500 mb-3">
          Канон §4.2 (v4.20): три типа кошелька — <strong className="text-gray-400">Гражданин</strong> (тип 1),{' '}
          <strong className="text-gray-400">Поставщик</strong>, <strong className="text-gray-400">Валидатор</strong>.{' '}
          Поставщик (не genesis): ZKP. Валидатор (не genesis): ZKP и ≥1 LZN. Гражданин: WRT/LZN, баланс ANT недоступен (§4.2). Переход в
          Гражданина сжигает ANT и снимает ордера. Отдельно: <strong className="text-gray-500">граждане DAO</strong> (§4.1)
          — держатели WRT с правом голоса; не путать с типом кошелька «Гражданин» и с «Поставщиком» (§4.1–4.2).
        </p>
        <RoleChangeForm
          currentRole={role}
          disabled={pending}
          onSubmit={(newRole) => run({ op: 'set_role', address, role: newRole })}
        />
        <p className="text-[11px] text-gray-500 mt-2">
          Эта tx всегда попадает в мемпул; условия канона проверяются узлом при сборке блока. Если роль не применится — это
          будет видно в «Канон-аудит» на вкладке «Симуляция».
        </p>
      </section>

      {/* Переводы */}
      <section className="bg-gray-900/50 p-4 rounded border border-gray-700">
        <h3 className="text-sm font-semibold text-gray-300 mb-2">Перевод WRT / LZN</h3>
        <p className="text-xs text-gray-500 mb-3">
          Tx всегда попадает в мемпул; узел проверяет канон в блоке. ANT напрямую не переводится (§4.1) — только внутренний рынок.
        </p>
        <TransferForm address={address} disabled={pending} onSubmit={(f) => run({ op: 'transfer', ...f })} />
      </section>

      {/* Активация LZN */}
      <section className="bg-gray-900/50 p-4 rounded border border-gray-700">
        <h3 className="text-sm font-semibold text-gray-300 mb-2">Активация LZN под майнинг</h3>
        <p className="text-xs text-gray-500 mb-3">
          Это предложение в блок. Узел применит только для роли <strong className="text-gray-400">Валидатор</strong>, при наличии
          ликвидного LZN и в пределах потолка {Lcap} LZN (⌊{LZN_TOTAL_SUPPLY_REF}/3⌋ по §4.2). Иначе — отклонение в «Канон-аудит».
        </p>
        <ActivateForm
          maxLiquid={account.lzn_balance}
          maxMore={Math.max(0, Lcap - frozen)}
          disabled={pending}
          onSubmit={(amount) => run({ op: 'activate_lzn', address, amount })}
        />
      </section>

      {/* Рынок */}
      <section className="bg-gray-900/50 p-4 rounded border border-gray-700">
        <h3 className="text-sm font-semibold text-gray-300 mb-2">Внутренний рынок ANT</h3>
        <p className="text-xs text-gray-500 mb-3">
          Это предложение в блок. Узел проверит канон §5.2: BUY — только Валидатор (эскроу WRT), SELL — только Поставщик (эскроу ANT),
          иначе — отклонение в «Канон-аудит». Цена — WRT за ANT с точностью до 6 знаков; эталонные демоны котируют её
          от безубыточности майнера (награда за блок / λ·L_total), то есть это доли WRT, а не единицы.
        </p>
        <MarketForm
          wrtBalance={account.wrt_balance}
          disabled={pending}
          onSubmit={(payload) => {
            if (payload.mode === 'limit') {
              run({
                op: 'create_order',
                address,
                side: payload.side,
                price: payload.price,
                amount: payload.amount,
              })
            } else {
              const body: Record<string, unknown> = {
                op: 'create_order',
                address,
                side: payload.side,
                market: true,
                amount: payload.amount,
              }
              if (payload.side === 'buy' && payload.max_wrt != null && !Number.isNaN(payload.max_wrt)) {
                body.max_wrt = payload.max_wrt
              }
              run(body)
            }
          }}
        />
        <div className="mt-4">
          <h4 className="text-xs text-gray-500 uppercase mb-2">Ваши открытые ордера</h4>
          {orders.length === 0 ? (
            <p className="text-gray-600 text-sm">Нет открытых ордеров</p>
          ) : (
            <ul className="space-y-2">
              {orders.map((o) => (
                <li
                  key={o.id}
                  className="flex flex-wrap items-center justify-between gap-2 bg-gray-800/80 px-3 py-2 rounded text-sm"
                >
                  <span className="font-mono text-xs text-gray-400">{o.id.slice(0, 10)}…</span>
                  <span className={o.order_type === 'buy' ? 'text-green-400' : 'text-red-400'}>
                    {o.order_type.toUpperCase()}
                  </span>
                  <span>
                    {o.amount.toFixed(2)} ANT @ {formatPrice(o.price)} WRT
                  </span>
                  <button
                    type="button"
                    disabled={pending}
                    onClick={() => run({ op: 'cancel_order', address, order_id: o.id })}
                    className="text-xs bg-gray-700 hover:bg-red-900/50 px-2 py-1 rounded"
                  >
                    Отмена (tx)
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      </section>

      {/* Declare */}
      <section className="bg-gray-900/50 p-4 rounded border border-gray-700">
        <h3 className="text-sm font-semibold text-gray-300 mb-2">Сжигание и ставка ANT — участие в блоке (§5.4)</h3>
        <p className="text-xs text-gray-500 mb-3">
          Это предложение в блок. Узел применит только для роли <strong className="text-gray-400">Валидатор</strong> и при выполнении
          ограничений: <span className="text-amber-200/80">b + s ≤ L_i</span> (актив. LZN) и{' '}
          <span className="text-amber-200/80">b + s ≤ баланс ANT</span>. Иначе — отклонение в «Канон-аудит».
        </p>
        <p className="text-xs text-gray-500 mb-3">
          Ruleset v2: списывается <strong className="text-red-400/90">только b</strong>. Ставка{' '}
          <strong className="text-cyan-400/90">s</strong> проверяется по балансу и задаёт вес{' '}
          <span className="text-amber-200/80">w_i = s / L_i</span> для ValidatorSet, но с баланса не уходит — в отличие
          от канона v4.20, где сжигались оба числа. Сверх верхнего предела коридора{' '}
          <span className="text-amber-200/80">(1−λ)·L_total</span> по сети declare с наименьшими w_i
          отсеиваются и ждут следующей высоты. Низ коридора:{' '}
          <span className="text-amber-200/80">Σb_i ≥ λ·L_total</span> (λ ≤ 5/12).
        </p>
        <DeclareForm
          address={address}
          activatedLzn={frozen}
          antBalance={account.ant_balance}
          maxBPlusS={Math.min(frozen, account.ant_balance)}
          disabled={pending}
          onSubmit={(burn_b, stake_s) => run({ op: 'declare', address, burn_b, stake_s })}
        />
      </section>
    </div>
  )
}

const WALLET_ROLES = ['citizen', 'provider', 'validator'] as const
type WalletRole = (typeof WALLET_ROLES)[number]

function normalizeWalletRole(r: string): WalletRole {
  if (r === 'guest') return 'citizen'
  return WALLET_ROLES.includes(r as WalletRole) ? (r as WalletRole) : 'citizen'
}

function RoleChangeForm({
  currentRole,
  disabled,
  onSubmit,
}: {
  currentRole: string
  disabled: boolean
  onSubmit: (r: string) => void
}) {
  const [role, setRole] = useState<WalletRole>(() => normalizeWalletRole(currentRole))
  useEffect(() => {
    setRole(normalizeWalletRole(currentRole))
  }, [currentRole])
  return (
    <div className="flex flex-wrap gap-2 items-end">
      <select
        value={role}
        onChange={(e) => setRole(normalizeWalletRole(e.target.value))}
        className="bg-gray-700 border border-gray-600 rounded px-3 py-2 text-sm"
        disabled={disabled}
      >
        <option value="citizen">Гражданин (тип 1, §4.2)</option>
        <option value="provider">Поставщик</option>
        <option value="validator">Валидатор</option>
      </select>
      <button
        type="button"
        disabled={disabled}
        onClick={() => onSubmit(role)}
        data-tip="Tx set_role в мемпул. Условия ZKP/LZN проверяет узел в блоке."
        className="bg-emerald-700 hover:bg-emerald-600 disabled:opacity-40 px-4 py-2 rounded text-sm"
      >
        Отправить в мемпул
      </button>
    </div>
  )
}

function TransferForm({
  address,
  disabled,
  onSubmit,
}: {
  address: string
  disabled: boolean
  onSubmit: (f: { address: string; to_address: string; amount: number; asset: string }) => void
}) {
  const [to, setTo] = useState('')
  const [amount, setAmount] = useState('1')
  const [asset, setAsset] = useState('wrt')
  return (
    <form
      className="flex flex-wrap gap-2 items-end"
      onSubmit={(e) => {
        e.preventDefault()
        const a = parseFloat(amount)
        if (!to.trim() || isNaN(a)) return
        onSubmit({ address, to_address: to.trim(), amount: a, asset })
      }}
    >
      <input
        placeholder="Адрес получателя"
        value={to}
        onChange={(e) => setTo(e.target.value)}
        className="bg-gray-700 border border-gray-600 rounded px-3 py-2 text-sm flex-1 min-w-[200px]"
        disabled={disabled}
      />
      <input
        type="number"
        step="any"
        min="0"
        value={amount}
        onChange={(e) => setAmount(e.target.value)}
        className="bg-gray-700 border border-gray-600 rounded px-3 py-2 text-sm w-28"
        disabled={disabled}
      />
      <select
        value={asset}
        onChange={(e) => setAsset(e.target.value)}
        className="bg-gray-700 border border-gray-600 rounded px-3 py-2 text-sm"
        disabled={disabled}
      >
        <option value="wrt" title="Wert — основной расчётный токен">WRT</option>
        <option value="lzn" title="Lizenz — лицензия / майнинг">LZN</option>
        <option value="ant" title="Anteil — прямой перевод узел отклонит (§4.1)">ANT</option>
      </select>
      <button
        type="submit"
        disabled={disabled}
        className="bg-blue-700 hover:bg-blue-600 px-4 py-2 rounded text-sm"
        data-tip="Перевод в мемпул. ANT узел отклонит — только внутренний рынок."
      >
        transfer
      </button>
    </form>
  )
}

function ActivateForm({
  maxLiquid,
  maxMore,
  disabled,
  onSubmit,
}: {
  maxLiquid: number
  maxMore: number
  disabled: boolean
  onSubmit: (amount: number) => void
}) {
  const cap = Math.min(maxLiquid, maxMore)
  const [amt, setAmt] = useState('1')
  return (
    <div className="flex flex-wrap gap-2 items-end">
      <input
        type="number"
        step="any"
        min="0"
        value={amt}
        onChange={(e) => setAmt(e.target.value)}
        className="bg-gray-700 border border-gray-600 rounded px-3 py-2 text-sm w-32"
        disabled={disabled}
      />
      <span className="text-xs text-gray-500">ориентир узла {cap.toFixed(4)}</span>
      <button
        type="button"
        disabled={disabled}
        onClick={() => {
          const v = parseFloat(amt)
          if (!isNaN(v) && v > 0) onSubmit(v)
        }}
        className="bg-amber-800 hover:bg-amber-700 px-4 py-2 rounded text-sm"
        data-tip="Заморозить LZN в L_i. Узел применит только валидатору и в пределах потолка."
      >
        activate_lzn
      </button>
    </div>
  )
}

type MarketFormPayload =
  | { mode: 'limit'; side: 'buy' | 'sell'; price: number; amount: number }
  | { mode: 'market'; side: 'buy' | 'sell'; amount: number; max_wrt?: number }

function MarketForm({
  wrtBalance,
  disabled,
  onSubmit,
}: {
  wrtBalance: number
  disabled: boolean
  onSubmit: (payload: MarketFormPayload) => void
}) {
  const [side, setSide] = useState<'buy' | 'sell'>('buy')
  const [mode, setMode] = useState<'limit' | 'market'>('limit')
  const [price, setPrice] = useState('10')
  const [amount, setAmount] = useState('1')
  const [maxWrt, setMaxWrt] = useState('')
  return (
    <form
      className="flex flex-col gap-3"
      onSubmit={(e) => {
        e.preventDefault()
        const a = parseFloat(amount)
        if (isNaN(a) || a <= 0) return
        if (mode === 'limit') {
          const p = parseFloat(price)
          if (isNaN(p) || p <= 0) return
          onSubmit({ mode: 'limit', side, price: p, amount: a })
          return
        }
        const cap = maxWrt.trim() === '' ? undefined : parseFloat(maxWrt)
        if (cap !== undefined && (isNaN(cap) || cap <= 0)) return
        onSubmit({ mode: 'market', side, amount: a, max_wrt: cap })
      }}
    >
      <div className="flex flex-wrap gap-2 items-center">
        <select
          value={side}
          onChange={(e) => setSide(e.target.value as 'buy' | 'sell')}
          className="bg-gray-700 border border-gray-600 rounded px-3 py-2 text-sm"
          disabled={disabled}
          data-tip="BUY — эскроу WRT, узел примет только валидатора. SELL — эскроу ANT, только поставщик."
        >
          <option value="buy">Buy ANT</option>
          <option value="sell">Sell ANT</option>
        </select>
        <select
          value={mode}
          onChange={(e) => setMode(e.target.value as 'limit' | 'market')}
          className="bg-gray-700 border border-gray-600 rounded px-3 py-2 text-sm"
          disabled={disabled}
          data-tip="Limit остаётся в книге. Market исполняется сразу по лучшим ценам."
        >
          <option value="limit">Лимитная заявка</option>
          <option value="market">По рынку (сразу по книге)</option>
        </select>
      </div>
      <div className="flex flex-wrap gap-2 items-end">
        {mode === 'limit' ? (
          <input
            type="number"
            step="any"
            placeholder="Цена WRT/ANT"
            value={price}
            onChange={(e) => setPrice(e.target.value)}
            className="bg-gray-700 border border-gray-600 rounded px-3 py-2 text-sm w-28"
            disabled={disabled}
          />
        ) : (
          side === 'buy' && (
            <>
              <input
                type="number"
                step="any"
                placeholder={`Макс. WRT (пусто = весь баланс ${wrtBalance.toFixed(2)})`}
                value={maxWrt}
                onChange={(e) => setMaxWrt(e.target.value)}
                className="bg-gray-700 border border-gray-600 rounded px-3 py-2 text-sm min-w-[200px] flex-1 max-w-xs"
                disabled={disabled}
              />
            </>
          )
        )}
        <input
          type="number"
          step="any"
          placeholder={mode === 'market' && side === 'buy' ? 'Макс. ANT' : 'ANT'}
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
          className="bg-gray-700 border border-gray-600 rounded px-3 py-2 text-sm w-24"
          disabled={disabled}
        />
        <button type="submit" disabled={disabled} data-tip="Ордер в мемпул. Роль и эскроу проверяет узел." className="bg-indigo-700 hover:bg-indigo-600 px-4 py-2 rounded text-sm">
          create_order
        </button>
      </div>
      {mode === 'market' && (
        <p className="text-xs text-gray-500">
          Рыночная заявка исполняется в блоке по лучшим ценам книги и не оставляет в ней лимитного ордера (IOC).
          Покупка списывает WRT по факту каждой сделки, в пределах «Макс. WRT» и баланса; продажа возвращает
          непроданный остаток ANT на баланс.
        </p>
      )}
    </form>
  )
}

const DECLARE_STORAGE_PREFIX = 'volnix_sim_declare_'

function loadDeclarePrefs(addr: string): { b: string; s: string } | null {
  try {
    const raw = localStorage.getItem(`${DECLARE_STORAGE_PREFIX}${addr}`)
    if (!raw) return null
    const j = JSON.parse(raw) as { b?: string; s?: string }
    if (typeof j.b === 'string' && typeof j.s === 'string') return { b: j.b, s: j.s }
  } catch {
    /* ignore */
  }
  return null
}

function saveDeclarePrefs(addr: string, b: string, s: string) {
  try {
    localStorage.setItem(`${DECLARE_STORAGE_PREFIX}${addr}`, JSON.stringify({ b, s }))
  } catch {
    /* ignore */
  }
}

function DeclareForm({
  address,
  activatedLzn,
  antBalance,
  maxBPlusS,
  disabled,
  onSubmit,
}: {
  address: string
  activatedLzn: number
  antBalance: number
  maxBPlusS: number
  disabled: boolean
  onSubmit: (b: number, s: number) => void
}) {
  const [b, setB] = useState('0.1')
  const [s, setS] = useState('0.1')
  const [totalTarget, setTotalTarget] = useState('')
  const [burnPct, setBurnPct] = useState(50)

  useEffect(() => {
    const prefs = loadDeclarePrefs(address)
    if (prefs) {
      setB(prefs.b)
      setS(prefs.s)
    } else {
      setB('0.1')
      setS('0.1')
    }
    setTotalTarget('')
    setBurnPct(50)
  }, [address])

  const bv = parseFloat(b)
  const sv = parseFloat(s)
  const sum = !isNaN(bv) && !isNaN(sv) ? bv + sv : NaN
  const cap = maxBPlusS
  const overCap = !isNaN(sum) && sum > cap + 1e-9
  const noReward = !isNaN(bv) && !isNaN(sv) && bv <= 0 && sum > 0

  const applyPreset = (nb: number, ns: number) => {
    setB(String(Math.max(0, nb)))
    setS(String(Math.max(0, ns)))
  }

  const applyTotalAndSplit = () => {
    const raw = parseFloat(totalTarget)
    if (isNaN(raw) || raw <= 0) return
    const p = Math.min(100, Math.max(0, burnPct))
    const nb = (raw * p) / 100
    const ns = raw - nb
    setB(String(nb))
    setS(String(ns))
  }

  const handleDeclare = () => {
    if (isNaN(bv) || isNaN(sv) || bv < 0 || sv < 0) return
    saveDeclarePrefs(address, b, s)
    onSubmit(bv, sv)
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap gap-3 text-xs font-mono bg-gray-800/80 px-3 py-2 rounded border border-gray-600/80">
        <span className="text-gray-400">
          L_i (актив. LZN): <span className="text-amber-300">{activatedLzn.toFixed(4)}</span>
        </span>
        <span className="text-gray-500">|</span>
        <span className="text-gray-400">
          ANT: <span className="text-orange-300">{antBalance.toFixed(4)}</span>
        </span>
        <span className="text-gray-500">|</span>
        <span className="text-gray-400">
          max b+s: <span className="text-emerald-300">{cap.toFixed(4)}</span>
        </span>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <label className="text-xs text-gray-400 flex flex-col gap-1.5">
          <span>
            Сжигание <span className="text-red-400/90">b</span> (ANT, burn)
          </span>
          <input
            type="number"
            step="any"
            min="0"
            value={b}
            onChange={(e) => setB(e.target.value)}
            className="bg-gray-700 border border-gray-600 rounded px-3 py-2 text-sm w-full max-w-[200px]"
            disabled={disabled}
          />
        </label>
        <label className="text-xs text-gray-400 flex flex-col gap-1.5">
          <span>
            Ставка <span className="text-cyan-400/90">s</span> (ANT, stake)
          </span>
          <input
            type="number"
            step="any"
            min="0"
            value={s}
            onChange={(e) => setS(e.target.value)}
            className="bg-gray-700 border border-gray-600 rounded px-3 py-2 text-sm w-full max-w-[200px]"
            disabled={disabled}
          />
        </label>
      </div>

      <div className="text-xs space-y-1">
        <div className={overCap ? 'text-red-400' : 'text-gray-500'}>
          Сумма b + s: {!isNaN(sum) ? sum.toFixed(6) : '—'}
          {overCap && ` — превышает лимит ${cap.toFixed(4)}`}
        </div>
        {noReward && (
          <div className="text-amber-600/90">
            При b = 0 за высоту не начисляется ни базовая WRT (§5.1), ни доля комиссий F·(b/B) (§5.4).
          </div>
        )}
      </div>

      <div className="border border-gray-600/60 rounded p-3 space-y-3 bg-gray-800/40">
        <div className="text-xs text-gray-400 font-semibold">Быстрые пресеты (масштабируются под max b+s)</div>
        <div className="flex flex-wrap gap-2">
          <button
            type="button"
            disabled={disabled}
            onClick={() => applyPreset(cap > 0 ? cap : 1, 0)}
            className="text-xs bg-gray-700 hover:bg-gray-600 px-2 py-1.5 rounded"
          >
            Всё в сжигание (b)
          </button>
          <button
            type="button"
            disabled={disabled}
            onClick={() => applyPreset(0, cap > 0 ? cap : 1)}
            className="text-xs bg-gray-700 hover:bg-gray-600 px-2 py-1.5 rounded"
          >
            Всё в ставку (s)
          </button>
          <button
            type="button"
            disabled={disabled}
            onClick={() => {
              const t = cap > 0 ? cap : 1
              applyPreset(t / 2, t / 2)
            }}
            className="text-xs bg-gray-700 hover:bg-gray-600 px-2 py-1.5 rounded"
          >
            50% / 50%
          </button>
          <button
            type="button"
            disabled={disabled}
            onClick={() => {
              const t = cap > 0 ? cap : 1
              const minB = Math.min(0.01, t * 0.05, t)
              applyPreset(minB, Math.max(0, t - minB))
            }}
            className="text-xs bg-gray-700 hover:bg-gray-600 px-2 py-1.5 rounded"
          >
            Мин. burn + остальное в s
          </button>
        </div>

        <div className="flex flex-wrap items-end gap-3 pt-1">
          <label className="text-xs text-gray-400 flex flex-col gap-1 min-w-[140px]">
            Сумма b+s (цель)
            <input
              type="number"
              step="any"
              min="0"
              placeholder={cap > 0 ? cap.toFixed(4) : '0'}
              value={totalTarget}
              onChange={(e) => setTotalTarget(e.target.value)}
              className="bg-gray-700 border border-gray-600 rounded px-2 py-1.5 text-sm"
              disabled={disabled}
            />
          </label>
          <label className="text-xs text-gray-400 flex flex-col gap-1 flex-1 min-w-[180px]">
            Доля на сжигание b: {burnPct}%
            <input
              type="range"
              min={0}
              max={100}
              value={burnPct}
              onChange={(e) => setBurnPct(Number(e.target.value))}
              className="w-full accent-teal-500"
              disabled={disabled}
            />
          </label>
          <button
            type="button"
            disabled={disabled}
            onClick={applyTotalAndSplit}
            className="text-xs bg-teal-900/60 hover:bg-teal-800/60 px-3 py-2 rounded"
          >
            Применить сумму и долю
          </button>
        </div>
      </div>

      <button
        type="button"
        disabled={disabled}
        onClick={handleDeclare}
        data-tip="Declare §5.4: b сжигается, s — ставка. Нужны валидатор, L_i и ANT ≥ b+s."
        className="bg-teal-800 hover:bg-teal-700 disabled:opacity-40 px-4 py-2 rounded text-sm"
      >
        Отправить declare (мемпул)
      </button>
    </div>
  )
}
