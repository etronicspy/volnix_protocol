import { useOutletContext } from 'react-router-dom'
import { formatMicro } from '../lib/api'
import { useFetch } from '../hooks/useFetch'
import { HashLink } from '../components/HashLink'
import type { EpochRecord } from '../types/api'

interface OutletCtx {
  livePulse: number
}

export function EpochsPage() {
  const { livePulse } = useOutletContext<OutletCtx>()
  const { data, error, loading } = useFetch<{ epochs: EpochRecord[] }>('/api/v1/epochs', livePulse)
  const epochs = [...(data?.epochs ?? [])].reverse()

  return (
    <div className="page">
      <h1 className="page-title">Epochs</h1>
      <p className="page-lead">
        Boundary §5.5: cancel ANT orders → wipe all ANT → ANT_emit = L_total × EpochBlocks → LZN_emit.
      </p>
      {error ? <p className="error">{error}</p> : null}
      {loading && !data ? <p className="muted">Loading…</p> : null}

      <div className="panel">
        {epochs.length ? (
          <div className="table-wrap">
            <table className="data">
              <thead>
                <tr>
                  <th>Epoch</th>
                  <th>Height</th>
                  <th>L_total</th>
                  <th>ANT wiped</th>
                  <th>ANT emit</th>
                  <th>LZN emit</th>
                  <th>Suppliers</th>
                </tr>
              </thead>
              <tbody>
                {epochs.map((e) => (
                  <tr key={e.epoch}>
                    <td>{e.epoch}</td>
                    <td>
                      <HashLink height={e.height} />
                    </td>
                    <td className="mono">{formatMicro(e.l_total)}</td>
                    <td className="mono">{formatMicro(e.ant_wiped)}</td>
                    <td className="mono">{formatMicro(e.ant_emit)}</td>
                    <td className="mono">{e.lzn_emit.toLocaleString()}</td>
                    <td>{e.suppliers.length}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="empty">No epoch boundaries yet</div>
        )}
      </div>
    </div>
  )
}
