import { useEffect, useState } from 'react'

import { useBackend } from '../adapters'
import type { Account, Feedback, Market, Order } from '../types'
import { Btn, FeedbackBanner, Field, Panel, TextInput, TextSelect, formatAmt } from './ui'

interface MarketPanelProps {
  address: string
  account: Account
  market: Market
  height: number
  pending: boolean
  feedback: Feedback | null
  onCreateOrder: (payload: Record<string, unknown>) => void
  onCancelOrder: (orderId: string) => void
}

export function MarketPanel({
  address,
  account: _account,
  market,
  height,
  pending,
  feedback,
  onCreateOrder,
  onCancelOrder,
}: MarketPanelProps) {
  const { adapter } = useBackend()

  const [side, setSide] = useState<'buy' | 'sell'>('buy')
  const [mode, setMode] = useState<'limit' | 'market'>('limit')
  const [price, setPrice] = useState(String(market.last_price || 1))
  const [amount, setAmount] = useState('1')
  const [maxWrt, setMaxWrt] = useState('')
  const [orders, setOrders] = useState<Order[]>([])

  useEffect(() => {
    void adapter.fetchOpenOrders(address).then(setOrders)
  }, [adapter, address, height])

  return (
    <Panel
      title="Рынок ANT"
      hint="BUY — валидатор (эскроу WRT), SELL — поставщик (эскроу ANT)."
    >
      {feedback ? <div className="mb-4"><FeedbackBanner {...feedback} /></div> : null}

      <div className="mb-4 flex flex-wrap gap-4 text-sm text-[var(--muted)]">
        <span data-tip="Последняя цена сделки ANT/WRT (движок котирует с 6 знаками; ориентир — безубыточность майнера, доли WRT за ANT).">
          Last: <strong className="mono text-[var(--sand)]">{formatAmt(market.last_price, 6)}</strong> WRT/ANT
        </span>
        <span data-tip="Открытые заявки на покупку ANT (валидаторы, эскроу WRT).">Bids {market.bids?.length ?? 0}</span>
        <span data-tip="Открытые заявки на продажу ANT (поставщики, эскроу ANT).">Asks {market.asks?.length ?? 0}</span>
      </div>

      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <Field label="Сторона" tip="BUY узел примет только у валидатора, SELL — только у поставщика. Остальным — reject.">
          <TextSelect value={side} onChange={(e) => setSide(e.target.value as 'buy' | 'sell')}>
            <option value="buy">Buy (validator)</option>
            <option value="sell">Sell (provider)</option>
          </TextSelect>
        </Field>
        <Field label="Тип" tip="Limit остаётся в книге. Market исполняется сразу по лучшим ценам (IOC).">
          <TextSelect value={mode} onChange={(e) => setMode(e.target.value as 'limit' | 'market')}>
            <option value="limit">Limit</option>
            <option value="market">Market</option>
          </TextSelect>
        </Field>
        {mode === 'limit' ? (
          <Field label="Цена">
            <TextInput type="number" min={0} step="any" value={price} onChange={(e) => setPrice(e.target.value)} />
          </Field>
        ) : (
          <Field label="Max WRT (buy, optional)">
            <TextInput type="number" min={0} step="any" value={maxWrt} onChange={(e) => setMaxWrt(e.target.value)} />
          </Field>
        )}
        <Field label="Количество ANT">
          <TextInput type="number" min={0} step="any" value={amount} onChange={(e) => setAmount(e.target.value)} />
        </Field>
      </div>

      <div className="mt-4">
        <Btn
          data-tip="Ордер в мемпул. Узел проверит роль и эскроу в DeliverTx."
          disabled={pending || !(Number(amount) > 0)}
          onClick={() => {
            if (mode === 'limit') {
              onCreateOrder({
                op: 'create_order',
                address,
                side,
                price: Number(price),
                amount: Number(amount),
              })
            } else {
              const body: Record<string, unknown> = {
                op: 'create_order',
                address,
                side,
                amount: Number(amount),
                market: true,
              }
              if (side === 'buy' && Number(maxWrt) > 0) body.max_wrt = Number(maxWrt)
              onCreateOrder(body)
            }
          }}
        >
          Разместить ордер
        </Btn>
      </div>

      <div className="mt-8">
        <h3 className="mb-3 font-semibold">Мои открытые ордера</h3>
        {orders.length === 0 ? (
          <p className="text-sm text-[var(--muted)]">Нет открытых ордеров.</p>
        ) : (
          <ul className="space-y-2">
            {orders.map((o) => (
              <li
                key={o.id}
                className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-[var(--line)] bg-[var(--panel-2)] px-3 py-2"
              >
                <div className="mono text-sm">
                  {o.order_type.toUpperCase()} · {formatAmt(o.amount - o.filled)} @ {formatAmt(o.price)}
                </div>
                <Btn variant="danger" disabled={pending} data-tip="Снять свой ордер и вернуть эскроу. Чужой ордер узел отклонит." onClick={() => onCancelOrder(o.id)}>
                  Cancel
                </Btn>
              </li>
            ))}
          </ul>
        )}
      </div>
    </Panel>
  )
}
