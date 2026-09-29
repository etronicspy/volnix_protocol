import { useMemo } from 'react'
import { Link, useOutletContext, useParams } from 'react-router-dom'
import { useFetch } from '../hooks/useFetch'
import { useMempool } from '../hooks/useMempool'
import { cancelIds, pendingPlaces } from '../lib/pending'
import { HashLink } from '../components/HashLink'
import type { AccountView, TxListItem } from '../types/api'

interface OutletCtx {
  livePulse: number
}

export function AccountPage() {
  const { address } = useParams()
  const { livePulse } = useOutletContext<OutletCtx>()
  const acc = useFetch<AccountView>(address ? `/api/v1/accounts/${address}` : null, livePulse)
  const txs = useFetch<{ txs: TxListItem[] }>(address ? `/api/v1/accounts/${address}/txs` : null, livePulse)
  const mempool = useMempool(livePulse, address)
  const dropping = useMemo(() => cancelIds(mempool.txs), [mempool.txs])
  const placing = useMemo(() => pendingPlaces(mempool.txs), [mempool.txs])

  const a = acc.data?.account
  const bal = acc.data?.balances_display

  return (
    <div className="page">
      <p className="muted">
        <Link to="/wallets">← Wallets</Link>
      </p>
      <h1 className="page-title">Account</h1>
      <p className="page-lead mono">{address}</p>
      {acc.error ? <p className="error">{acc.error}</p> : null}
      {acc.loading && !a ? <p className="muted">Loading…</p> : null}

      {a && bal ? (
        <>
          <div className="grid-3" style={{ marginBottom: '1rem' }}>
            <div className="stat">
              <div className="stat-label">Role</div>
              <div className="stat-value">{a.role}</div>
            </div>
            <div className="stat">
              <div className="stat-label">WRT</div>
              <div className="stat-value">{bal.wrt}</div>
            </div>
            <div className="stat">
              <div className="stat-label">ANT</div>
              <div className="stat-value">{bal.ant}</div>
            </div>
            <div className="stat">
              <div className="stat-label">LZN free</div>
              <div className="stat-value">{bal.lzn}</div>
            </div>
            <div className="stat">
              <div className="stat-label">LZN activated</div>
              <div className="stat-value">{bal.lzn_activated}</div>
            </div>
            <div className="stat">
              <div className="stat-label">Sequence / last tx</div>
              <div className="stat-value sm">
                {a.sequence} / h{a.last_tx_height}
              </div>
            </div>
          </div>

          {a.genesis_no_zkp ? <p className="muted">Genesis validator (no ZKP).</p> : null}

          {acc.data?.last_declare ? (
            <div className="panel">
              <div className="panel-title">Last declare (in-memory this height)</div>
              <div className="mono">
                b_i={acc.data.last_declare.b_i} · s_i={acc.data.last_declare.s_i} · f_i=
                {acc.data.last_declare.f_i} · w_i={acc.data.last_declare.w_i.toFixed(4)}
              </div>
            </div>
          ) : null}

          <div className="panel">
            <div className="panel-title">Open orders</div>
            {acc.data?.open_orders?.length || placing.length ? (
              <div className="table-wrap">
                <table className="data">
                  <thead>
                    <tr>
                      <th>ID</th>
                      <th>Market</th>
                      <th>Side</th>
                      <th>Amount</th>
                      <th>Price</th>
                      <th></th>
                    </tr>
                  </thead>
                  <tbody>
                    {acc.data?.open_orders.map((o) => (
                      <tr key={o.order_id}>
                        <td className="mono">{o.order_id}</td>
                        <td>{o.market}</td>
                        <td>{o.side}</td>
                        <td className="mono">{o.amount}</td>
                        <td className="mono">{o.price}</td>
                        <td>{dropping.has(o.order_id) ? <span className="badge warn">cancelling</span> : null}</td>
                      </tr>
                    ))}
                    {placing.map((row) => (
                      <tr key={row.hash}>
                        <td className="mono">—</td>
                        <td>{row.market}</td>
                        <td>{row.side}</td>
                        <td className="mono">{row.amount}</td>
                        <td className="mono">{row.price}</td>
                        <td>
                          <span className="badge warn">order in mempool</span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <div className="empty">No open orders</div>
            )}
          </div>

          <div className="panel">
            <div className="panel-title">Transaction history</div>
            {mempool.rejects.length ? <p className="error">{mempool.rejects.join(' · ')}</p> : null}
            {mempool.txs.length || txs.data?.txs?.length ? (
              <div className="table-wrap">
                <table className="data">
                  <thead>
                    <tr>
                      <th>Hash</th>
                      <th>Height</th>
                      <th>Index</th>
                    </tr>
                  </thead>
                  <tbody>
                    {mempool.txs.map((t) => (
                      <tr key={t.hash}>
                        <td>
                          <HashLink hash={t.hash} />
                        </td>
                        <td>
                          <span className="badge warn">in mempool</span>
                          <div className="muted" style={{ marginTop: 4, fontSize: '0.75rem' }}>
                            {t.label}
                          </div>
                        </td>
                        <td>—</td>
                      </tr>
                    ))}
                    {txs.data?.txs.map((t) => (
                      <tr key={t.hash}>
                        <td>
                          <HashLink hash={t.hash} />
                        </td>
                        <td>
                          <HashLink height={t.height} />
                        </td>
                        <td>{t.index}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <div className="empty">No transactions</div>
            )}
          </div>
        </>
      ) : null}
    </div>
  )
}
