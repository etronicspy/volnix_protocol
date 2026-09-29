import { useOutletContext } from 'react-router-dom'
import { formatMicro } from '../lib/api'
import { useFetch } from '../hooks/useFetch'
import { useMempool } from '../hooks/useMempool'
import { HashLink } from '../components/HashLink'
import type { BlockSummary, ChainSummary, TxListItem } from '../types/api'

interface OutletCtx {
  livePulse: number
  liveHeight: number
}

export function OverviewPage() {
  const { livePulse } = useOutletContext<OutletCtx>()
  const summary = useFetch<ChainSummary>('/api/v1/chain/summary', livePulse)
  const blocks = useFetch<{ blocks: BlockSummary[] }>('/api/v1/blocks?tail=12', livePulse)
  const txs = useFetch<{ txs: TxListItem[] }>('/api/v1/txs?limit=12', livePulse)
  const mempool = useMempool(livePulse)

  const s = summary.data

  return (
    <div className="page">
      <h1 className="page-title">Chain overview</h1>
      <p className="page-lead">Live view of Volnix sim-2 — height, supply, PoVB parameters, recent blocks.</p>

      {summary.error ? <p className="error">{summary.error}</p> : null}
      {summary.loading && !s ? <p className="muted">Loading…</p> : null}

      {s ? (
        <>
          <div className="grid-3" style={{ marginBottom: '1rem' }}>
            <div className="stat">
              <div className="stat-label">Height</div>
              <div className="stat-value">{s.height}</div>
            </div>
            <div className="stat">
              <div className="stat-label">Epoch</div>
              <div className="stat-value">
                {s.epoch} <span className="muted" style={{ fontSize: '0.85rem' }}>({s.blocks_to_epoch} left)</span>
              </div>
            </div>
            <div className="stat">
              <div className="stat-label">L_total</div>
              <div className="stat-value">{s.l_total_display}</div>
            </div>
            <div className="stat">
              <div className="stat-label">WRT supply</div>
              <div className="stat-value">{s.supply.wrt_display}</div>
            </div>
            <div className="stat">
              <div className="stat-label">ANT supply</div>
              <div className="stat-value">{s.supply.ant_display}</div>
            </div>
            <div className="stat">
              <div className="stat-label">LZN minted</div>
              <div className="stat-value">{s.supply.lzn_minted_tokens.toLocaleString()}</div>
            </div>
          </div>

          <div className="panel">
            <div className="panel-title">Canon parameters</div>
            <div className="grid-3">
              <div className="stat">
                <div className="stat-label as-written">λ / α / K</div>
                <div className="stat-value sm">
                  {s.params.lambda} · {s.params.alpha} · {s.params.k}
                </div>
              </div>
              <div className="stat">
                <div className="stat-label">Block reward</div>
                <div className="stat-value sm">{formatMicro(s.params.current_block_reward)} WRT</div>
              </div>
              <div className="stat">
                <div className="stat-label">Validators / suppliers</div>
                <div className="stat-value sm">
                  {s.n_validators} / {s.n_suppliers}
                </div>
              </div>
              <div className="stat">
                <div className="stat-label">Mempool</div>
                <div className="stat-value sm">{s.mempool} txs</div>
              </div>
              <div className="stat">
                <div className="stat-label">App hash</div>
                <div className="stat-value sm">{s.app_hash.slice(0, 16)}…</div>
              </div>
              <div className="stat">
                <div className="stat-label">Chain time</div>
                <div className="stat-value sm">{s.time}</div>
              </div>
            </div>
          </div>
        </>
      ) : null}

      <div className="grid-2" style={{ marginTop: '1rem' }}>
        <div className="panel">
          <div className="panel-title">Recent blocks</div>
          {blocks.data?.blocks?.length ? (
            <div className="table-wrap">
              <table className="data">
                <thead>
                  <tr>
                    <th>Height</th>
                    <th>Txs</th>
                    <th>Proposer</th>
                    <th>Time</th>
                  </tr>
                </thead>
                <tbody>
                  {blocks.data.blocks.map((b) => (
                    <tr key={b.height}>
                      <td>
                        <HashLink height={b.height} />
                      </td>
                      <td>{b.n_txs}</td>
                      <td>
                        <HashLink address={b.proposer} />
                      </td>
                      <td className="mono muted">{b.time.replace('T', ' ').replace('Z', '')}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <div className="empty">No blocks yet</div>
          )}
        </div>

        <div className="panel">
          <div className="panel-title">Recent transactions</div>
          {mempool.rejects.length ? <p className="error">{mempool.rejects.join(' · ')}</p> : null}
          {mempool.txs.length || txs.data?.txs?.length ? (
            <div className="table-wrap">
              <table className="data">
                <thead>
                  <tr>
                    <th>Hash</th>
                    <th>Height</th>
                    <th>Sender</th>
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
                      <td>
                        <HashLink address={t.sender} />
                      </td>
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
                      <td>
                        <HashLink address={t.sender} />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <div className="empty">No transactions yet</div>
          )}
        </div>
      </div>
    </div>
  )
}
