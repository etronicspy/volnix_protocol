import { useMemo, useState } from 'react'
import { useOutletContext } from 'react-router-dom'
import { useFetch } from '../hooks/useFetch'
import { HashLink } from '../components/HashLink'
import type { AccountListResponse } from '../types/api'

interface OutletCtx {
  livePulse: number
}

type RoleFilter = 'all' | 'citizen' | 'supplier' | 'validator'

export function WalletsPage() {
  const { livePulse } = useOutletContext<OutletCtx>()
  const [role, setRole] = useState<RoleFilter>('all')
  const { data, error, loading } = useFetch<AccountListResponse>('/api/v1/accounts', livePulse)

  const filtered = useMemo(() => {
    const all = data?.accounts ?? []
    if (role === 'all') return all
    return all.filter((a) => a.role === role)
  }, [data, role])

  const counts = useMemo(() => {
    const all = data?.accounts ?? []
    return {
      total: all.length,
      citizen: all.filter((a) => a.role === 'citizen').length,
      supplier: all.filter((a) => a.role === 'supplier').length,
      validator: all.filter((a) => a.role === 'validator').length,
    }
  }, [data])

  return (
    <div className="page">
      <h1 className="page-title">Wallets</h1>
      <p className="page-lead">
        All simulation accounts — role/status and WRT / LZN / ANT balances.
      </p>
      {error ? <p className="error">{error}</p> : null}
      {loading && !data ? <p className="muted">Loading…</p> : null}

      {data ? (
        <>
          <div className="grid-3" style={{ marginBottom: '1rem' }}>
            <div className="stat">
              <div className="stat-label">Accounts</div>
              <div className="stat-value">{counts.total}</div>
            </div>
            <div className="stat">
              <div className="stat-label">Height</div>
              <div className="stat-value">{data.height}</div>
            </div>
            <div className="stat">
              <div className="stat-label">By role</div>
              <div className="stat-value sm">
                V {counts.validator} · S {counts.supplier} · C {counts.citizen}
              </div>
            </div>
          </div>

          <div className="row" style={{ marginBottom: '0.75rem', flexWrap: 'wrap', gap: '0.5rem' }}>
            {(
              [
                ['all', 'All'],
                ['validator', 'Validators'],
                ['supplier', 'Suppliers'],
                ['citizen', 'Citizens'],
              ] as const
            ).map(([value, label]) => (
              <button
                key={value}
                type="button"
                className={`btn ${role === value ? '' : 'secondary'}`}
                onClick={() => setRole(value)}
              >
                {label}
              </button>
            ))}
          </div>

          <div className="panel">
            <div className="panel-title">
              {filtered.length} wallet{filtered.length === 1 ? '' : 's'}
              {role !== 'all' ? ` · ${role}` : ''}
            </div>
            {filtered.length ? (
              <div className="table-wrap">
                <table className="data">
                  <thead>
                    <tr>
                      <th>Address</th>
                      <th>Role</th>
                      <th>Status</th>
                      <th>WRT</th>
                      <th>ANT</th>
                      <th>LZN</th>
                      <th>LZN act.</th>
                      <th>Last tx</th>
                    </tr>
                  </thead>
                  <tbody>
                    {filtered.map((a) => (
                      <tr key={a.address}>
                        <td>
                          <HashLink address={a.address} />
                        </td>
                        <td>
                          <span
                            className={`badge ${
                              a.role === 'validator'
                                ? 'ok'
                                : a.role === 'supplier'
                                  ? 'warn'
                                  : 'muted'
                            }`}
                          >
                            {a.role}
                          </span>
                        </td>
                        <td>
                          <span className="row" style={{ gap: '0.35rem', flexWrap: 'wrap' }}>
                            {a.genesis_no_zkp ? (
                              <span className="badge muted">genesis</span>
                            ) : null}
                            {a.in_validator_set ? (
                              <span className="badge ok">
                                {a.is_proposer ? 'proposer' : 'in set'}
                              </span>
                            ) : a.role === 'validator' ? (
                              <span className="badge muted">not in set</span>
                            ) : null}
                            {a.open_orders > 0 ? (
                              <span className="badge warn">{a.open_orders} orders</span>
                            ) : null}
                          </span>
                        </td>
                        <td className="mono">{a.wrt_display}</td>
                        <td className="mono">{a.ant_display}</td>
                        <td className="mono">{a.lzn_display}</td>
                        <td className="mono">{a.lzn_activated_display}</td>
                        <td className="mono muted">
                          {a.last_tx_height > 0 ? `h${a.last_tx_height}` : '—'}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <div className="empty">No wallets match this filter</div>
            )}
          </div>
        </>
      ) : null}
    </div>
  )
}
