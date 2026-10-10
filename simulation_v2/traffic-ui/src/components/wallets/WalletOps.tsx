import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { WS_URL } from '../../config'
import { ApiError, formatMicro } from '../../lib/api'
import { cancelIds, fetchMempool, pendingPlaces, settleTx } from '../../lib/pending'
import type { PendingTx } from '../../lib/pending'
import {
  declareBurn,
  fetchAccount,
  fetchParams,
  mintWrt,
  signedTx,
  verifyRole,
} from '../../lib/wallet'
import type { AccountDetail, ChainParams, TxMessage } from '../../types/api'
import {
  DeclareForm,
  LznForm,
  MigrateForm,
  MintForm,
  OrderForm,
  SendForm,
  VerifyForm,
  WalletStats,
} from './WalletForms'

interface WalletOpsProps {
  seed: string
  address: string
}

export function WalletOps({ seed, address }: WalletOpsProps) {
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
    const txs = await fetchMempool(address)
    const liveHashes = new Set(txs.map((tx) => tx.hash))
    const left = [...watched.current].filter((hash) => !liveHashes.has(hash))
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
  }, [address])

  const load = useCallback(async () => {
    try {
      const chain = await fetchParams()
      let account: AccountDetail | null = null
      try {
        account = await fetchAccount(address)
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
  }, [address, refreshMempool])

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

  return (
    <div className="stack">
      <div className="topbar">
        <div>
          <div className="panel-title">Операции</div>
          <div className="addr">{address}</div>
          <p className="muted" style={{ marginTop: '0.35rem' }}>
            {live ? 'live' : 'offline'}
            {height ? ` · h${height}` : ''} · {detail ? role : 'ещё нет в цепочке'}
          </p>
        </div>
      </div>

      {error ? <p className="error">{error}</p> : null}
      {reject ? <p className="error">{reject}</p> : null}

      <WalletStats
        wrt={account?.wrt}
        lzn={account?.lzn}
        lznActivated={account?.lzn_activated}
        ant={account?.ant}
        sequence={account?.sequence}
      />

      <div className="grid-2">
        <SendForm
          pending={busy}
          from={address}
          onSubmit={(to, amount) =>
            run(() =>
              signedTx(seed, [
                {
                  type: 'bank/MsgSend',
                  from_address: address,
                  to_address: to,
                  denom: 'uwrt',
                  amount,
                },
              ]),
            )
          }
        />
        <MintForm pending={busy} onSubmit={(amount) => run(() => mintWrt(address, amount))} />
      </div>

      <div className="grid-2">
        {role === 'citizen' ? (
          <VerifyForm
            pending={busy}
            onSubmit={(desired) =>
              run(() => verifyRole(seed, desired, `stub-zkp-${crypto.randomUUID()}`))
            }
          />
        ) : (
          <MigrateForm
            pending={busy}
            sourceSeed={seed}
            from={address}
            onSubmit={(to, proof) =>
              run(() =>
                signedTx(seed, [
                  {
                    type: 'ident/MsgMigrateRole',
                    from_address: address,
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
            onSubmit={(bI, sI) => run(() => declareBurn(seed, bI, sI))}
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
              run(() => signedTx(seed, [{ type: 'lizenz/MsgActivateLZN', validator: address, amount }]))
            }
          />
          <LznForm
            kind="deactivate"
            pending={busy}
            onSubmit={(amount) =>
              run(() =>
                signedTx(seed, [
                  {
                    type: 'lizenz/MsgDeactivateLZN',
                    validator: address,
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
        owner={address}
        onSubmit={(msg: TxMessage) => run(() => signedTx(seed, [msg]))}
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
                <th>price (WRT)</th>
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
                    <td>{formatMicro(order.price)}</td>
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
                              signedTx(seed, [
                                {
                                  type: 'anteil/MsgCancelOrder',
                                  owner: address,
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
                  <td>{formatMicro(row.price)}</td>
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
