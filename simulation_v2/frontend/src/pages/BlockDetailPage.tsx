import { Link, useParams } from 'react-router-dom'
import { useFetch } from '../hooks/useFetch'
import { formatMicro } from '../lib/api'
import { HashLink } from '../components/HashLink'
import { CommitVotesTable } from '../components/block/CommitVotesTable'
import { PovbCorridor } from '../components/block/PovbCorridor'
import { ValidatorSetDiff } from '../components/block/ValidatorSetDiff'
import type { BlockDetail, ChainEvent, PovbTrace } from '../types/api'

const SECTIONS = [
  { id: 'overview', label: 'Overview' },
  { id: 'commit', label: 'Commit' },
  { id: 'povb', label: 'PoVB' },
  { id: 'valset', label: 'ValidatorSet' },
  { id: 'txs', label: 'Transactions' },
  { id: 'events', label: 'Events' },
] as const

function groupEvents(events: ChainEvent[]): Record<string, ChainEvent[]> {
  const groups: Record<string, ChainEvent[]> = {}
  for (const e of events) {
    const prefix = e.type.includes('.') ? e.type.split('.')[0] : e.type
    ;(groups[prefix] ??= []).push(e)
  }
  return groups
}

function EventGroups({ title, events }: { title: string; events: ChainEvent[] }) {
  if (!events.length) return null
  const groups = groupEvents(events)
  return (
    <div className="panel">
      <div className="panel-title">{title}</div>
      <div className="stack">
        {Object.entries(groups).map(([prefix, list]) => (
          <details key={prefix} open>
            <summary className="event-group-sum">
              <span className="badge muted">{prefix}</span>
              <span className="muted"> {list.length}</span>
            </summary>
            <div className="stack" style={{ marginTop: 8 }}>
              {list.map((e, i) => (
                <div key={`${e.type}-${i}`}>
                  <span className="badge muted">{e.type}</span>
                  <div className="mono muted" style={{ marginTop: 4, fontSize: '0.8rem' }}>
                    {e.attributes.map((a) => `${a.key}=${a.value}`).join(' · ')}
                  </div>
                </div>
              ))}
            </div>
          </details>
        ))}
      </div>
    </div>
  )
}

function num(v: number | string | undefined | null): number {
  if (v === undefined || v === null || v === '') return 0
  const n = typeof v === 'string' ? Number(v) : v
  return Number.isFinite(n) ? n : 0
}

function PovbPanel({ povb }: { povb: PovbTrace }) {
  const bMin = num(povb.b_min ?? povb.lower)
  const bMax = num(povb.b_max ?? povb.upper)
  const sumB = num(povb.sum_b)
  const fill = num(povb.fill)
  const lDecl = num(povb.l_decl ?? povb.l_total)

  return (
    <div className="panel" id="povb">
      <div className="panel-title">PoVB (§5.4)</div>
      <div className="grid-3" style={{ marginBottom: '0.75rem' }}>
        <div className="stat">
          <div className="stat-label">Set updated</div>
          <div className="stat-value sm">
            <span className={`badge ${povb.set_updated ? 'ok' : 'muted'}`}>
              {povb.set_updated ? 'yes' : 'no'}
            </span>
          </div>
        </div>
        <div className="stat">
          <div className="stat-label">Fill / L_decl</div>
          <div className="stat-value sm">
            {formatMicro(fill)} / {formatMicro(lDecl)}
          </div>
        </div>
        <div className="stat">
          <div className="stat-label">Σ b_i</div>
          <div className="stat-value sm">{formatMicro(sumB)}</div>
        </div>
        <div className="stat">
          <div className="stat-label">α / λ / K</div>
          <div className="stat-value sm">
            {povb.alpha ?? '—'} / {povb.lambda ?? '—'} / {povb.k ?? '—'}
          </div>
        </div>
        <div className="stat">
          <div className="stat-label">Corridor</div>
          <div className="stat-value sm">
            {formatMicro(bMin)} … {formatMicro(bMax)}
          </div>
        </div>
        <div className="stat">
          <div className="stat-label">Passed</div>
          <div className="stat-value sm">
            {(povb.passed?.length ?? 0)}/{povb.declares?.length ?? 0}
          </div>
        </div>
      </div>

      <PovbCorridor bMin={bMin} bMax={bMax} sumB={sumB} setUpdated={Boolean(povb.set_updated)} />

      {povb.declares?.length ? (
        <div className="table-wrap" style={{ marginTop: '0.85rem' }}>
          <table className="data">
            <thead>
              <tr>
                <th>Validator</th>
                <th>L_i</th>
                <th>f_i</th>
                <th>b_i</th>
                <th>s_i</th>
                <th>w_i</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {povb.declares.map((d) => (
                <tr key={d.validator}>
                  <td>
                    <HashLink address={d.validator} />
                  </td>
                  <td className="mono">{formatMicro(d.l_i)}</td>
                  <td className="mono">{formatMicro(d.f_i)}</td>
                  <td className="mono">{formatMicro(d.b_i)}</td>
                  <td className="mono">{formatMicro(d.s_i)}</td>
                  <td className="mono">{d.w_i.toFixed(4)}</td>
                  <td>
                    {!d.valid ? (
                      <span className="badge bad">{d.reason || 'invalid'}</span>
                    ) : d.passed ? (
                      <span className="badge ok">passed</span>
                    ) : (
                      <span className="badge muted">{d.excluded || 'out'}</span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <div className="empty" style={{ marginTop: '0.75rem' }}>
          No declares this height
        </div>
      )}
    </div>
  )
}

export function BlockDetailPage() {
  const { height: heightParam } = useParams()
  const height = heightParam ? Number(heightParam) : NaN
  const { data, error, loading } = useFetch<BlockDetail>(
    Number.isFinite(height) ? `/api/v1/blocks/${height}` : null,
  )
  const header = data?.block.header as Record<string, string | number> | undefined
  const txs = (data?.block.data.txs ?? []) as {
    hash?: string
    body?: { messages?: { type: string }[] }
  }[]
  const consensus = data?.consensus
  const povb = consensus?.povb ?? data?.results?.povb
  const evidence = data?.block.evidence ?? []

  return (
    <div className="page page-wide">
      <p className="muted">
        <Link to="/blocks">← Blocks</Link>
        {Number.isFinite(height) ? (
          <>
            {' · '}
            {height > 0 ? <HashLink height={height - 1} /> : <span className="muted">#0</span>}
            {' / '}
            <strong>#{height}</strong>
            {' / '}
            <HashLink height={height + 1} />
          </>
        ) : null}
      </p>
      <h1 className="page-title">Block {heightParam}</h1>
      <p className="page-lead">Consensus world for this height — CometBFT commit, PoVB EndBlocker, ValidatorSet.</p>
      {error ? <p className="error">{error}</p> : null}
      {loading && !data ? <p className="muted">Loading…</p> : null}

      {data && header ? (
        <>
          <nav className="section-nav" aria-label="Block sections">
            {SECTIONS.map((s) => (
              <a key={s.id} href={`#${s.id}`}>
                {s.label}
              </a>
            ))}
          </nav>

          <div className="panel" id="overview">
            <div className="panel-title">Overview</div>
            <div className="grid-3">
              <div className="stat">
                <div className="stat-label">Hash</div>
                <div className="stat-value sm">{data.hash}</div>
              </div>
              <div className="stat">
                <div className="stat-label">Proposer</div>
                <div className="stat-value sm">
                  <HashLink address={String(header.proposer_address)} />
                </div>
              </div>
              <div className="stat">
                <div className="stat-label">Time</div>
                <div className="stat-value sm">{String(header.time)}</div>
              </div>
              <div className="stat">
                <div className="stat-label">App hash</div>
                <div className="stat-value sm">{String(header.app_hash)}</div>
              </div>
              <div className="stat">
                <div className="stat-label">Data hash</div>
                <div className="stat-value sm">{String(header.data_hash)}</div>
              </div>
              <div className="stat">
                <div className="stat-label">Validators / next</div>
                <div className="stat-value sm">
                  {String(header.validators_hash ?? '—').slice(0, 12)}…
                  <br />
                  {String(header.next_validators_hash ?? '—').slice(0, 12)}…
                </div>
              </div>
              <div className="stat">
                <div className="stat-label">Tx count</div>
                <div className="stat-value">{txs.length}</div>
              </div>
              <div className="stat">
                <div className="stat-label">Evidence</div>
                <div className="stat-value">{evidence.length}</div>
              </div>
              <div className="stat">
                <div className="stat-label">Last results hash</div>
                <div className="stat-value sm">{String(header.last_results_hash ?? '—')}</div>
              </div>
            </div>
          </div>

          <CommitVotesTable
            commit={consensus?.commit ?? null}
            signingSet={consensus?.signing_set ?? null}
            votedPower={consensus?.voted_power ?? 0}
            totalPower={consensus?.total_power ?? 0}
            round={consensus?.round ?? 0}
          />

          {povb && Object.keys(povb).length > 0 ? (
            <PovbPanel povb={povb} />
          ) : (
            <div className="panel" id="povb">
              <div className="panel-title">PoVB (§5.4)</div>
              <div className="empty">No PoVB trace</div>
            </div>
          )}

          <ValidatorSetDiff
            signingSet={consensus?.signing_set ?? null}
            nextSet={consensus?.next_set ?? null}
            updates={data.results?.validator_updates ?? []}
          />

          <div className="panel" id="txs">
            <div className="panel-title">Transactions</div>
            {txs.length ? (
              <div className="table-wrap">
                <table className="data">
                  <thead>
                    <tr>
                      <th>#</th>
                      <th>Hash</th>
                      <th>Messages</th>
                      <th>Result</th>
                    </tr>
                  </thead>
                  <tbody>
                    {txs.map((tx, i) => {
                      const res = data.results?.txs_results[i]
                      const types = tx.body?.messages?.map((m) => m.type).join(', ') ?? '—'
                      return (
                        <tr key={tx.hash ?? i}>
                          <td>{i}</td>
                          <td>{tx.hash ? <HashLink hash={tx.hash} /> : '—'}</td>
                          <td className="mono">{types}</td>
                          <td>
                            {res ? (
                              <span className={`badge ${res.code === 0 ? 'ok' : 'bad'}`}>
                                {res.code === 0 ? 'ok' : `err ${res.code}`}
                              </span>
                            ) : (
                              '—'
                            )}
                          </td>
                        </tr>
                      )
                    })}
                  </tbody>
                </table>
              </div>
            ) : (
              <div className="empty">Empty block</div>
            )}
          </div>

          <div id="events" className="stack">
            <EventGroups title="BeginBlock events" events={data.results?.begin_block_events ?? []} />
            <EventGroups title="EndBlock events" events={data.results?.end_block_events ?? []} />
            {!data.results?.begin_block_events?.length && !data.results?.end_block_events?.length ? (
              <div className="panel">
                <div className="panel-title">Events</div>
                <div className="empty">No events</div>
              </div>
            ) : null}
          </div>
        </>
      ) : null}
    </div>
  )
}
