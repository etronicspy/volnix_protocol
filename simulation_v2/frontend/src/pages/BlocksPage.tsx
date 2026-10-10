import { useEffect, useMemo, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api, formatMicro, formatTime, shortHash } from '../lib/api'
import { HashLink } from '../components/HashLink'
import { useLive } from '../hooks/useLive'
import { Badge } from '../components/ui/Badge'
import { DataTable } from '../components/ui/DataTable'
import { Page } from '../components/ui/Page'
import { Panel } from '../components/ui/Panel'
import type { BlockSummary } from '../types/api'

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
  const { chainTick } = useLive()
  const navigate = useNavigate()
  const [blocks, setBlocks] = useState<BlockSummary[]>([])
  const [latest, setLatest] = useState(0)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [loadingOlder, setLoadingOlder] = useState(false)
  const loaded = useRef(false)

  const oldest = useMemo(() => (blocks.length ? Math.min(...blocks.map((b) => b.height)) : 0), [blocks])
  const canLoadOlder = blocks.length > 0 && oldest > 0

  useEffect(() => {
    let cancelled = false
    const first = !loaded.current
    if (first) setLoading(true)
    api
      .get<BlocksResponse>(`/api/v1/blocks?tail=${PAGE}`)
      .then((d) => {
        if (cancelled) return
        loaded.current = true
        setLatest(d.latest)
        setBlocks((prev) => (prev.length ? mergeByHeight(prev, d.blocks) : d.blocks))
        setError(null)
      })
      .catch((e: unknown) => {
        if (cancelled) return
        if (first) setError(e instanceof Error ? e.message : 'Request failed')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [chainTick])

  async function loadOlder() {
    if (!canLoadOlder || loadingOlder) return
    setLoadingOlder(true)
    try {
      const maxHeight = oldest - 1
      const d = await api.get<BlocksResponse>(`/api/v1/blocks?tail=${PAGE}&max_height=${maxHeight}`)
      setBlocks((prev) => mergeByHeight(prev, d.blocks))
      setLatest(d.latest)
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed to load older blocks')
    } finally {
      setLoadingOlder(false)
    }
  }

  return (
    <Page
      title="Blocks"
      lead="Full chain tape with PoVB outcomes per height. Open a block for the consensus world (commit votes, declare corridor, validator set diff)."
      error={error}
      loading={loading && !blocks.length}
      wide
    >
      <Panel
        title={
          <span>
            Latest height {latest} · showing {blocks.length} block{blocks.length === 1 ? '' : 's'}
          </span>
        }
      >
        <DataTable
          head={['Height', 'Time', 'Proposer', 'Txs', 'PoVB', 'Passed', 'Σb / Fill', 'Hash']}
          empty="No blocks"
          count={blocks.length}
        >
          {blocks.map((b) => (
            <tr key={b.height} className="row-link" onClick={() => navigate(`/blocks/${b.height}`)}>
              <td>
                <HashLink height={b.height} />
              </td>
              <td className="mono muted">{formatTime(b.time)}</td>
              <td onClick={(e) => e.stopPropagation()}>
                <HashLink address={b.proposer} />
              </td>
              <td>{b.n_txs}</td>
              <td>
                {b.set_updated ? <Badge tone="ok">updated</Badge> : <Badge tone="muted">held</Badge>}
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
        </DataTable>
        {canLoadOlder ? (
          <div className="row" style={{ marginTop: '0.85rem' }}>
            <button type="button" className="btn secondary" disabled={loadingOlder} onClick={loadOlder}>
              {loadingOlder ? 'Loading…' : `Load older (before #${oldest})`}
            </button>
          </div>
        ) : null}
      </Panel>
    </Page>
  )
}
