import { useMemo, useState } from 'react'
import type { FormEvent } from 'react'
import { formatMicro } from '../../lib/api'
import { generatePhrase, normalizePhrase } from '../../lib/phrase'
import { entryBurn, tokensToMicro, unlockAccount } from '../../lib/wallet'
import type { TxMessage } from '../../types/api'

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="stat">
      <div className="stat-label">{label}</div>
      <div className="stat-value sm">{value}</div>
    </div>
  )
}

export function WalletStats({
  wrt,
  lzn,
  lznActivated,
  ant,
  sequence,
}: {
  wrt?: number
  lzn?: number
  lznActivated?: number
  ant?: number
  sequence?: number
}) {
  return (
    <div className="grid-3">
      <Stat label="WRT" value={formatMicro(wrt)} />
      <Stat label="LZN" value={formatMicro(lzn)} />
      <Stat label="LZN акт." value={formatMicro(lznActivated)} />
      <Stat label="ANT" value={formatMicro(ant)} />
      <Stat label="seq" value={sequence === undefined ? '—' : String(sequence)} />
    </div>
  )
}

export function SendForm({
  pending,
  from,
  onSubmit,
}: {
  pending: boolean
  from: string
  onSubmit: (to: string, amount: number) => void
}) {
  const [to, setTo] = useState('')
  const [amount, setAmount] = useState('')
  return (
    <form
      className="panel"
      onSubmit={(e: FormEvent) => {
        e.preventDefault()
        const micro = tokensToMicro(amount)
        if (!to.trim() || micro === null || micro <= 0 || to.trim() === from) return
        onSubmit(to.trim(), micro)
      }}
    >
      <div className="panel-title">Перевод WRT</div>
      <div className="field">
        <label htmlFor="to">Адрес</label>
        <input id="to" value={to} onChange={(e) => setTo(e.target.value)} />
      </div>
      <div className="field">
        <label htmlFor="send-amt">Сумма</label>
        <input id="send-amt" value={amount} onChange={(e) => setAmount(e.target.value)} />
      </div>
      <button className="btn" type="submit" disabled={pending}>
        Отправить
      </button>
    </form>
  )
}

export function MintForm({
  pending,
  onSubmit,
}: {
  pending: boolean
  onSubmit: (amount: number) => void
}) {
  const [amount, setAmount] = useState('1')
  return (
    <form
      className="panel"
      onSubmit={(e: FormEvent) => {
        e.preventDefault()
        const micro = tokensToMicro(amount)
        if (micro === null || micro <= 0) return
        onSubmit(micro)
      }}
    >
      <div className="panel-title">Пополнить WRT из genesis</div>
      <p>
        Стендовый <code>/operator/mint</code>: перевод с genesis (нужна накопленная
        субсидия, преминта нет).
      </p>
      <div className="field">
        <label htmlFor="mint-amt">Сумма</label>
        <input id="mint-amt" value={amount} onChange={(e) => setAmount(e.target.value)} />
      </div>
      <button className="btn" type="submit" disabled={pending}>
        Пополнить
      </button>
    </form>
  )
}

export function VerifyForm({
  pending,
  onSubmit,
}: {
  pending: boolean
  onSubmit: (role: string) => void
}) {
  const [role, setRole] = useState('validator')
  return (
    <form
      className="panel"
      onSubmit={(e: FormEvent) => {
        e.preventDefault()
        onSubmit(role)
      }}
    >
      <div className="panel-title">Смена роли</div>
      <p>Citizen может стать supplier или validator. Обратно роль снимает только MOA.</p>
      <div className="field">
        <label htmlFor="role">Роль</label>
        <select id="role" value={role} onChange={(e) => setRole(e.target.value)}>
          <option value="validator">validator</option>
          <option value="supplier">supplier</option>
        </select>
      </div>
      <button className="btn" type="submit" disabled={pending}>
        Подтвердить роль
      </button>
    </form>
  )
}

export function MigrateForm({
  pending,
  sourceSeed,
  from,
  onSubmit,
}: {
  pending: boolean
  sourceSeed: string
  from: string
  onSubmit: (to: string, proof: string) => void
}) {
  const [phrase, setPhrase] = useState('')
  const [dest, setDest] = useState('')
  const [localError, setLocalError] = useState<string | null>(null)

  async function resolve(e: FormEvent) {
    e.preventDefault()
    const seed = normalizePhrase(phrase)
    if (!seed || seed === sourceSeed) {
      setLocalError('Нужна новая фраза получателя')
      return
    }
    setLocalError(null)
    try {
      const account = await unlockAccount(seed)
      if (account.address === from) {
        setLocalError('Адрес совпадает с текущим')
        return
      }
      setDest(account.address)
      onSubmit(account.address, `stub-zkp-${crypto.randomUUID()}`)
    } catch (err) {
      setLocalError(err instanceof Error ? err.message : 'Не удалось получить адрес')
    }
  }

  return (
    <form className="panel" onSubmit={(e) => void resolve(e)}>
      <div className="panel-title">Перенос роли</div>
      <p>Новая сид-фраза citizen-кошелька. ANT и LZN уйдут на него, WRT останется здесь. Запишите фразу.</p>
      <div className="row" style={{ marginBottom: '0.75rem' }}>
        <button type="button" className="btn secondary" onClick={() => setPhrase(generatePhrase())}>
          Сгенерировать
        </button>
      </div>
      <div className="field">
        <label htmlFor="dest-phrase">Фраза получателя</label>
        <textarea id="dest-phrase" value={phrase} onChange={(e) => setPhrase(e.target.value)} />
      </div>
      {dest ? <p className="mono">{dest}</p> : null}
      {localError ? <p className="error">{localError}</p> : null}
      <button className="btn" type="submit" disabled={pending}>
        Перенести роль
      </button>
    </form>
  )
}

export function DeclareForm({
  pending,
  ant,
  lznActivated,
  alphaNum,
  alphaDen,
  onSubmit,
}: {
  pending: boolean
  ant: number
  lznActivated: number
  alphaNum: number
  alphaDen: number
  onSubmit: (bI: number, sI: number) => void
}) {
  const [b, setB] = useState('0.5')
  const [s, setS] = useState('0.4')
  const preview = useMemo(() => {
    const f = entryBurn(lznActivated, alphaNum, alphaDen)
    const bMicro = tokensToMicro(b) ?? 0
    const sMicro = tokensToMicro(s) ?? 0
    return { f, bMicro, sMicro, sum: f + bMicro + sMicro }
  }, [alphaDen, alphaNum, b, lznActivated, s])

  return (
    <form
      className="panel"
      onSubmit={(e: FormEvent) => {
        e.preventDefault()
        if (preview.sMicro <= 0) return
        onSubmit(preview.bMicro, preview.sMicro)
      }}
    >
      <div className="panel-title">Сжигание ANT</div>
      <p>
        fᵢ = ⌊α · Lᵢ⌋ считается из активированного LZN (α = {alphaNum}/{alphaDen}). sᵢ должен быть больше нуля.
        Суммы в токенах.
      </p>
      <div className="field">
        <label htmlFor="bi">bᵢ</label>
        <input id="bi" value={b} onChange={(e) => setB(e.target.value)} />
      </div>
      <div className="field">
        <label htmlFor="si">sᵢ</label>
        <input id="si" value={s} onChange={(e) => setS(e.target.value)} />
      </div>
      <p className="mono">
        fᵢ {formatMicro(preview.f)} · bᵢ {formatMicro(preview.bMicro)} · sᵢ {formatMicro(preview.sMicro)}
        <br />
        сумма {formatMicro(preview.sum)} · ANT {formatMicro(ant)}
      </p>
      <button className="btn" type="submit" disabled={pending || preview.sMicro <= 0}>
        Объявить
      </button>
    </form>
  )
}

export function LznForm({
  kind,
  pending,
  onSubmit,
}: {
  kind: 'activate' | 'deactivate'
  pending: boolean
  onSubmit: (amount: number) => void
}) {
  const [amount, setAmount] = useState('1')
  const title = kind === 'activate' ? 'Активировать LZN' : 'Деактивировать LZN'
  return (
    <form
      className="panel"
      onSubmit={(e: FormEvent) => {
        e.preventDefault()
        const micro = tokensToMicro(amount)
        if (micro === null || micro <= 0) return
        onSubmit(micro)
      }}
    >
      <div className="panel-title">{title}</div>
      <div className="field">
        <label htmlFor={kind}>Сумма</label>
        <input id={kind} value={amount} onChange={(e) => setAmount(e.target.value)} />
      </div>
      <button className="btn" type="submit" disabled={pending}>
        {title}
      </button>
    </form>
  )
}

export function OrderForm({
  pending,
  owner,
  onSubmit,
}: {
  pending: boolean
  owner: string
  onSubmit: (msg: TxMessage) => void
}) {
  const [market, setMarket] = useState('ANT/WRT')
  const [side, setSide] = useState('BUY')
  const [amount, setAmount] = useState('1')
  const [price, setPrice] = useState('1')

  return (
    <form
      className="panel"
      onSubmit={(e: FormEvent) => {
        e.preventDefault()
        const micro = tokensToMicro(amount)
        const priceMicro = tokensToMicro(price)
        if (micro === null || micro <= 0 || priceMicro === null || priceMicro <= 0) return
        onSubmit({
          type: 'anteil/MsgPlaceOrder',
          owner,
          market,
          side,
          order_type: 'LIMIT',
          amount: micro,
          price: priceMicro,
        })
      }}
    >
      <div className="panel-title">Лимитный ордер</div>
      <p>Цена — WRT за 1 целый ANT/LZN (можно меньше 1, например 0.25).</p>
      <div className="grid-2">
        <div className="field">
          <label htmlFor="market">Рынок</label>
          <select id="market" value={market} onChange={(e) => setMarket(e.target.value)}>
            <option value="ANT/WRT">ANT/WRT</option>
            <option value="LZN/WRT">LZN/WRT</option>
          </select>
        </div>
        <div className="field">
          <label htmlFor="side">Сторона</label>
          <select id="side" value={side} onChange={(e) => setSide(e.target.value)}>
            <option value="BUY">BUY</option>
            <option value="SELL">SELL</option>
          </select>
        </div>
      </div>
      <div className="grid-2">
        <div className="field">
          <label htmlFor="ord-amt">Количество</label>
          <input id="ord-amt" value={amount} onChange={(e) => setAmount(e.target.value)} />
        </div>
        <div className="field">
          <label htmlFor="ord-price">Цена</label>
          <input id="ord-price" value={price} onChange={(e) => setPrice(e.target.value)} />
        </div>
      </div>
      <button className="btn" type="submit" disabled={pending}>
        Выставить
      </button>
    </form>
  )
}
