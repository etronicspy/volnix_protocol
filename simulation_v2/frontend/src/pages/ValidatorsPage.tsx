import { useOutletContext } from 'react-router-dom'
import { formatMicro } from '../lib/api'
import { useFetch } from '../hooks/useFetch'
import { HashLink } from '../components/HashLink'
import type { ValidatorView } from '../types/api'

interface OutletCtx {
  livePulse: number
}

export function ValidatorsPage() {
  const { livePulse } = useOutletContext<OutletCtx>()
  const { data, error, loading } = useFetch<{
    proposer: string
    total_power: number
    validators: ValidatorView[]
  }>('/api/v1/validators', livePulse)

  return (
    <div className="page">
      <h1 className="page-title">Validators</h1>
      <p className="page-lead">Current ValidatorSet — weight w_i = s_i / L_i, integer power for proposer priority.</p>
      {error ? <p className="error">{error}</p> : null}
      {loading && !data ? <p className="muted">Loading…</p> : null}

      {data ? (
        <div className="panel">
          <div className="panel-title">
            Proposer <HashLink address={data.proposer} /> · total power {data.total_power.toLocaleString()}
          </div>
          {data.validators.length ? (
            <div className="table-wrap">
              <table className="data">
                <thead>
                  <tr>
                    <th>Address</th>
                    <th>L_i</th>
                    <th>s_i</th>
                    <th>w_i</th>
                    <th>Power</th>
                    <th>ANT</th>
                    <th></th>
                  </tr>
                </thead>
                <tbody>
                  {data.validators.map((v) => (
                    <tr key={v.address}>
                      <td>
                        <HashLink address={v.address} />
                      </td>
                      <td className="mono">{formatMicro(v.l_i)}</td>
                      <td className="mono">{formatMicro(v.s_i)}</td>
                      <td className="mono">{v.w_i.toFixed(4)}</td>
                      <td className="mono">{v.power.toLocaleString()}</td>
                      <td className="mono">{formatMicro(v.ant)}</td>
                      <td>{v.is_proposer ? <span className="badge ok">proposer</span> : null}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <div className="empty">Empty set</div>
          )}
        </div>
      ) : null}
    </div>
  )
}
