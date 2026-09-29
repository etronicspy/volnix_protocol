import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import type { FormEvent } from 'react'
import { WS_URL } from './config'
import { ApiError, formatMicro, tokensToMicro } from './lib/api'
import { generatePhrase, normalizePhrase } from './lib/phrase'
import { cancelIds, fetchMempool, pendingPlaces, settleTx } from './lib/pending'
import type { PendingTx } from './lib/pending'
import {
  clearSeed,
  declareBurn,
  entryBurn,
  fetchAccount,
  fetchParams,
  signedTx,
  unlockAccount,
  verifyRole,
} from './lib/wallet'
import type { WalletSession } from './lib/wallet'
import type { AccountDetail, ChainParams } from './types/api'

interface WalletHomeProps {
  session: WalletSession
  onLogout: () => void
}

export function WalletHome({ session, onLogout }: WalletHomeProps) {
  const [detail, setDetail] = useState<AccountDetail | null>(null)
  const [params, setParams] = useState<ChainParams | null>(null)
  const [height, setHeight] = useState(0)
  const [live, setLive] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [reject, setReject] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [mempool, setMempool] = useState<PendingTx[]>([])
  const watched = useRef<Set<string>>(new Set())

  const refreshMempool = useCallback(async () => {
    const txs = await fetchMempool(session.address)
    const live = new Set(txs.map((tx) => tx.hash))
    const left = [...watched.current].filter((hash) => !live.has(hash))
    const failures: string[] = []
    for (const hash of left) {
      const settled = await settleTx(hash)
      if (!settled) continue
      watched.current.delete(hash)
      if (settled.code !== 0) failures.push(settled.log || `отклонено (${settled.code})`)
    }
    for (const tx of txs) watched.current.add(tx.hash)
    setMempool(txs)
    if (failures.length > 0) setReject(failures.join(' · '))
  }, [session.address])

  const load = useCallback(async () => {
    try {
      const chain = await fetchParams()
      let account: AccountDetail | null = null
      try {
        account = await fetchAccount(session.address)
      } catch (err) {
        if (!(err instanceof ApiError) || err.status !== 404) throw err
      }
      setDetail(account)
      setParams(chain)
      setError(null)
      await refreshMempool()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось прочитать счёт')
    }
  }, [session.address, refreshMempool])

  useEffect(() => {
    void load()
  }, [load])

  useEffect(() => {
    let ws: WebSocket | null = null
    let closed = false
    let timer: number | undefined
    let retry = 0

    const connect = () => {
      if (closed) return
      ws = new WebSocket(WS_URL)
      ws.onopen = () => {
        retry = 0
        setLive(true)
      }
      ws.onclose = () => {
        setLive(false)
        if (closed) return
        timer = window.setTimeout(connect, Math.min(10_000, 500 * 2 ** retry))
        retry += 1
      }
      ws.onmessage = (ev) => {
        try {
          const msg = JSON.parse(String(ev.data)) as { type?: string; height?: number }
          if (msg.type === 'new_block' || msg.type === 'init') {
            if (typeof msg.height === 'number') setHeight(msg.height)
            if (msg.type === 'new_block') void load()
          }
        } catch {
          /* ignore */
        }
      }
    }
    connect()
    return () => {
      closed = true
      if (timer) window.clearTimeout(timer)
      ws?.close()
    }
  }, [load])

  const account = detail?.account
  const role = account?.role ?? 'citizen'
  const dropping = useMemo(() => cancelIds(mempool), [mempool])
  const placing = useMemo(() => pendingPlaces(mempool), [mempool])

  async function run(action: () => Promise<string>) {
    setBusy(true)
    setError(null)
    setReject(null)
    try {
      const hash = await action()
      watched.current.add(hash)
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Отклонено')
    } finally {
      setBusy(false)
    }
  }

  function logout() {
    clearSeed()
    onLogout()
  }

  return (
    <div className="page">
      <div className="topbar">
        <div>
          <div className="brand">
            Volnix <span>Wallet</span>
          </div>
          <div className="addr">{session.address}</div>
          <p className="muted" style={{ marginTop: '0.35rem' }}>
            {live ? 'live' : 'offline'}
            {height ? ` · h${height}` : ''} · {detail ? role : 'ещё нет в цепочке'}
          </p>
        </div>
        <button type="button" className="btn secondary" onClick={logout}>
          Выйти
        </button>
      </div>

      {error ? <p className="error">{error}</p> : null}
      {reject ? <p className="error">{reject}</p> : null}

      <div className="grid-stats">
        <Stat label="WRT" value={formatMicro(account?.wrt)} />
        <Stat label="LZN" value={formatMicro(account?.lzn)} />
        <Stat label="LZN акт." value={formatMicro(account?.lzn_activated)} />
        <Stat label="ANT" value={formatMicro(account?.ant)} />
        <Stat label="seq" value={String(account?.sequence ?? '—')} />
      </div>

      <div className="grid-2">
        <SendForm
          pending={busy}
          from={session.address}
          onSubmit={(to, amount) =>
            run(() =>
              signedTx(session.seed, [
                {
                  type: 'bank/MsgSend',
                  from_address: session.address,
                  to_address: to,
                  denom: 'uwrt',
                  amount,
                },
              ]),
            )
          }
        />
      </div>

      <div className="grid-2">
        {role === 'citizen' ? (
          <VerifyForm
            pending={busy}
            onSubmit={(desired) =>
              run(() => verifyRole(session.seed, desired, `stub-zkp-${crypto.randomUUID()}`))
            }
          />
        ) : (
          <MigrateForm
            pending={busy}
            sourceSeed={session.seed}
            from={session.address}
            onSubmit={(to, proof) =>
              run(() =>
                signedTx(session.seed, [
                  {
                    type: 'ident/MsgMigrateRole',
                    from_address: session.address,
                    to_address: to,
                    zkp_proof: proof,
                  },
                ]),
              )
            }
          />
        )}
        {role === 'validator' ? (
          <DeclareForm
            pending={busy}
            ant={account?.ant ?? 0}
            lznActivated={account?.lzn_activated ?? 0}
            alphaNum={params?.alpha_num ?? 1}
            alphaDen={params?.alpha_den ?? 50}
            onSubmit={(bI, sI) => run(() => declareBurn(session.seed, bI, sI))}
          />
        ) : (
          <section className="panel">
            <div className="panel-title">Сжигание</div>
            <p>Объявление bᵢ / sᵢ доступно роли validator. Входное fᵢ списывается из ANT в конце блока.</p>
          </section>
        )}
      </div>

      {role === 'validator' ? (
        <div className="grid-2">
          <LznForm
            kind="activate"
            pending={busy}
            onSubmit={(amount) =>
              run(() =>
                signedTx(session.seed, [
                  { type: 'lizenz/MsgActivateLZN', validator: session.address, amount },
                ]),
              )
            }
          />
          <LznForm
            kind="deactivate"
            pending={busy}
            onSubmit={(amount) =>
              run(() =>
                signedTx(session.seed, [
                  {
                    type: 'lizenz/MsgDeactivateLZN',
                    validator: session.address,
                    amount,
                    reason: 'wallet',
                  },
                ]),
              )
            }
          />
        </div>
      ) : null}

      <OrderForm
        pending={busy}
        owner={session.address}
        onSubmit={(msg) => run(() => signedTx(session.seed, [msg]))}
      />

      <section className="panel">
        <div className="panel-title">В мемпуле</div>
        {mempool.length > 0 ? (
          <ul className="pending-list">
            {mempool.map((tx) => (
              <li key={tx.hash}>
                <span className="tag">в мемпуле</span> {tx.label}
                <span className="mono muted"> {tx.hash.slice(0, 12)}…</span>
              </li>
            ))}
          </ul>
        ) : (
          <p className="muted">Нет ожидающих транзакций.</p>
        )}
      </section>

      <section className="panel">
        <div className="panel-title">Открытые ордера</div>
        {(detail && detail.open_orders.length > 0) || placing.length > 0 ? (
          <table className="data">
            <thead>
              <tr>
                <th>id</th>
                <th>market</th>
                <th>side</th>
                <th>amount</th>
                <th>price</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {detail?.open_orders.map((order) => {
                const lifting = dropping.has(order.order_id)
                return (
                  <tr key={order.order_id}>
                    <td className="mono">{order.order_id.slice(0, 10)}</td>
                    <td>{order.market}</td>
                    <td>{order.side}</td>
                    <td>{formatMicro(order.amount)}</td>
                    <td>{order.price}</td>
                    <td>
                      {lifting ? (
                        <span className="tag">снимается</span>
                      ) : (
                        <button
                          type="button"
                          className="btn secondary"
                          disabled={busy}
                          onClick={() =>
                            void run(() =>
                              signedTx(session.seed, [
                                {
                                  type: 'anteil/MsgCancelOrder',
                                  owner: session.address,
                                  order_id: order.order_id,
                                },
                              ]),
                            )
                          }
                        >
                          Отмена
                        </button>
                      )}
                    </td>
                  </tr>
                )
              })}
              {placing.map((row) => (
                <tr key={row.hash}>
                  <td className="mono">—</td>
                  <td>{row.market}</td>
                  <td>{row.side}</td>
                  <td>{formatMicro(row.amount)}</td>
                  <td>{row.price}</td>
                  <td>
                    <span className="tag">ордер в мемпуле</span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <p className="muted">Нет открытых ордеров.</p>
        )}
      </section>
    </div>
  )
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="stat">
      <div className="stat-label">{label}</div>
      <div className="stat-value sm">{value}</div>
    </div>
  )
}

function SendForm({
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

function VerifyForm({
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

function MigrateForm({
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

function DeclareForm({
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

function LznForm({
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

function OrderForm({
  pending,
  owner,
  onSubmit,
}: {
  pending: boolean
  owner: string
  onSubmit: (msg: {
    type: string
    owner: string
    market: string
    side: string
    order_type: string
    amount: number
    price: number
  }) => void
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
        const priceN = Number(price)
        if (micro === null || micro <= 0 || !Number.isInteger(priceN) || priceN <= 0) return
        onSubmit({
          type: 'anteil/MsgPlaceOrder',
          owner,
          market,
          side,
          order_type: 'LIMIT',
          amount: micro,
          price: priceN,
        })
      }}
    >
      <div className="panel-title">Лимитный ордер</div>
      <p>Цена — целое число micro-WRT за micro базового актива (1 = один к одному).</p>
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
