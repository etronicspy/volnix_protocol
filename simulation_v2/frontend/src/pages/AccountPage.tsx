import { Link, useParams } from 'react-router-dom'
import { useFetch } from '../hooks/useFetch'
import { HashLink } from '../components/HashLink'
import type { AccountView, TxListItem } from '../types/api'

export function AccountPage() {
  const { address } = useParams()
  const acc = useFetch<AccountView>(address ? `/api/v1/accounts/${address}` : null)
  const txs = useFetch<{ txs: TxListItem[] }>(address ? `/api/v1/accounts/${address}/txs` : null)

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
            {acc.data?.open_orders?.length ? (
              <div className="table-wrap">
                <table className="data">
                  <thead>
                    <tr>
                      <th>ID</th>
                      <th>Market</th>
                      <th>Side</th>
                      <th>Amount</th>
                      <th>Price</th>
                    </tr>
                  </thead>
                  <tbody>
                    {acc.data.open_orders.map((o) => (
                      <tr key={o.order_id}>
                        <td className="mono">{o.order_id}</td>
                        <td>{o.market}</td>
                        <td>{o.side}</td>
                        <td className="mono">{o.amount}</td>
                        <td className="mono">{o.price}</td>
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
            {txs.data?.txs?.length ? (
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
                    {txs.data.txs.map((t) => (
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
