import { useEffect, useMemo, useState } from 'react'
import { useNavigate, useOutletContext } from 'react-router-dom'
import { api, formatMicro, shortHash } from '../lib/api'
import { HashLink } from '../components/HashLink'
import type { BlockSummary } from '../types/api'

interface OutletCtx {
  livePulse: number
}

interface BlocksResponse {
  blocks: BlockSummary[]
  latest: number
}

const PAGE = 100

function mergeByHeight(existing: BlockSummary[], incoming: BlockSummary[]): BlockSummary[] {
  const map = new Map<number, BlockSummary>()
  for (const b of existing) map.set(b.height, b)
  for (const b of incoming) map.set(b.height, b)
  return [...map.values()].sort((a, b) => b.height - a.height)
}

export function BlocksPage() {
  const { livePulse } = useOutletContext<OutletCtx>()
  const navigate = useNavigate()
  const [blocks, setBlocks] = useState<BlockSummary[]>([])
  const [latest, setLatest] = useState(0)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [loadingOlder, setLoadingOlder] = useState(false)

  const oldest = useMemo(() => (blocks.length ? Math.min(...blocks.map((b) => b.height)) : 0), [blocks])
  const canLoadOlder = blocks.length > 0 && oldest > 0

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    api
      .get<BlocksResponse>(`/api/v1/blocks?tail=${PAGE}`)
      .then((d) => {
        if (cancelled) return
        setBlocks(d.blocks)
        setLatest(d.latest)
        setError(null)
      })
      .catch((e: unknown) => {
        if (!cancelled) setError(e instanceof Error ? e.message : 'Request failed')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [])

  // Live tip refresh: pull newest PAGE and merge without dropping older pages
  useEffect(() => {
    if (!livePulse) return
    let cancelled = false
    api
      .get<BlocksResponse>(`/api/v1/blocks?tail=${PAGE}`)
      .then((d) => {
        if (cancelled) return
        setLatest(d.latest)
        setBlocks((prev) => mergeByHeight(prev, d.blocks))
        setError(null)
      })
      .catch(() => {
        /* keep stale tape on live glitch */
      })
    return () => {
      cancelled = true
    }
  }, [livePulse])

  async function loadOlder() {
    if (!canLoadOlder || loadingOlder) return
    setLoadingOlder(true)
    try {
      const maxHeight = oldest - 1
      const d = await api.get<BlocksResponse>(
        `/api/v1/blocks?tail=${PAGE}&max_height=${maxHeight}`,
      )
      setBlocks((prev) => mergeByHeight(prev, d.blocks))
      setLatest(d.latest)
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed to load older blocks')
    } finally {
      setLoadingOlder(false)
    }
  }

  return (
    <div className="page page-wide">
      <h1 className="page-title">Blocks</h1>
      <p className="page-lead">
        Full chain tape with PoVB outcomes per height. Open a block for the consensus world
        (commit votes, declare corridor, validator set diff).
      </p>
      {error ? <p className="error">{error}</p> : null}
      {loading && !blocks.length ? <p className="muted">Loading…</p> : null}

      <div className="panel">
        <div className="panel-title row" style={{ justifyContent: 'space-between' }}>
          <span>
            Latest height {latest} · showing {blocks.length} block{blocks.length === 1 ? '' : 's'}
          </span>
        </div>
        {blocks.length ? (
          <div className="table-wrap">
            <table className="data">
              <thead>
                <tr>
                  <th>Height</th>
                  <th>Time</th>
                  <th>Proposer</th>
                  <th>Txs</th>
                  <th>PoVB</th>
                  <th>Passed</th>
                  <th>Σb / Fill</th>
                  <th>Hash</th>
                </tr>
              </thead>
              <tbody>
                {blocks.map((b) => (
                  <tr
                    key={b.height}
                    className="row-link"
                    onClick={() => navigate(`/blocks/${b.height}`)}
                  >
                    <td>
                      <HashLink height={b.height} />
                    </td>
                    <td className="mono muted">{b.time.replace('T', ' ').replace('Z', '')}</td>
                    <td onClick={(e) => e.stopPropagation()}>
                      <HashLink address={b.proposer} />
                    </td>
                    <td>{b.n_txs}</td>
                    <td>
                      {b.set_updated ? (
                        <span className="badge ok">updated</span>
                      ) : (
                        <span className="badge muted">held</span>
                      )}
                    </td>
                    <td className="mono">
                      {b.n_passed ?? 0}/{b.n_declares ?? 0}
                    </td>
                    <td className="mono muted">
                      {formatMicro(b.sum_b ?? 0, 2)} / {formatMicro(b.fill ?? 0, 2)}
                    </td>
                    <td className="mono muted" title={b.hash}>
                      {shortHash(b.hash)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="empty">No blocks</div>
        )}
        {canLoadOlder ? (
          <div className="row" style={{ marginTop: '0.85rem' }}>
            <button type="button" className="btn secondary" disabled={loadingOlder} onClick={loadOlder}>
              {loadingOlder ? 'Loading…' : `Load older (before #${oldest})`}
            </button>
          </div>
        ) : null}
      </div>
    </div>
  )
}
