import { useParams } from 'react-router-dom'
import { useFetch } from '../hooks/useFetch'
import { HashLink } from '../components/HashLink'
import { CommitVotesTable } from '../components/block/CommitVotesTable'
import { EventGroups } from '../components/block/EventGroups'
import { PovbPanel } from '../components/block/PovbPanel'
import { ValidatorSetDiff } from '../components/block/ValidatorSetDiff'
import { Badge } from '../components/ui/Badge'
import { DataTable } from '../components/ui/DataTable'
import { Page } from '../components/ui/Page'
import { Panel } from '../components/ui/Panel'
import { Stat } from '../components/ui/Stat'
import type { BlockDetail } from '../types/api'

const SECTIONS = [
  { id: 'overview', label: 'Overview' },
  { id: 'commit', label: 'Commit' },
  { id: 'povb', label: 'PoVB' },
  { id: 'valset', label: 'ValidatorSet' },
  { id: 'txs', label: 'Transactions' },
  { id: 'events', label: 'Events' },
] as const

export function BlockDetailPage() {
  const { height: heightParam } = useParams()
  const height = heightParam ? Number(heightParam) : NaN
  const { data, error, loading } = useFetch<BlockDetail>(
    Number.isFinite(height) ? `/api/v1/blocks/${height}` : null,
  )
  const header = data?.block.header
  const txs = data?.block.data.txs ?? []
  const consensus = data?.consensus
  const povb = consensus?.povb ?? data?.results?.povb
  const evidence = data?.block.evidence ?? []

  return (
    <Page
      title={`Block ${heightParam}`}
      lead="Consensus world for this height — CometBFT commit, PoVB EndBlocker, ValidatorSet."
      back={{ to: '/blocks', label: 'Blocks' }}
      afterBack={
        Number.isFinite(height) ? (
          <>
            {' · '}
            {height > 0 ? <HashLink height={height - 1} /> : <span className="muted">#0</span>}
            {' / '}
            <strong>#{height}</strong>
            {' / '}
            <HashLink height={height + 1} />
          </>
        ) : null
      }
      error={error}
      loading={loading && !data}
      wide
    >
      {data && header ? (
        <>
          <nav className="section-nav" aria-label="Block sections">
            {SECTIONS.map((s) => (
              <a key={s.id} href={`#${s.id}`}>
                {s.label}
              </a>
            ))}
          </nav>

          <Panel id="overview" title="Overview">
            <div className="grid-3">
              <Stat label="Hash" sm>
                {data.hash}
              </Stat>
              <Stat label="Proposer" sm>
                <HashLink address={String(header.proposer_address)} />
              </Stat>
              <Stat label="Time" sm>
                {String(header.time)}
              </Stat>
              <Stat label="App hash" sm>
                {String(header.app_hash)}
              </Stat>
              <Stat label="Data hash" sm>
                {String(header.data_hash)}
              </Stat>
              <Stat label="Validators / next" sm>
                {String(header.validators_hash ?? '—').slice(0, 12)}…
                <br />
                {String(header.next_validators_hash ?? '—').slice(0, 12)}…
              </Stat>
              <Stat label="Tx count">{txs.length}</Stat>
              <Stat label="Evidence">{evidence.length}</Stat>
              <Stat label="Last results hash" sm>
                {String(header.last_results_hash ?? '—')}
              </Stat>
            </div>
          </Panel>

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
            <Panel id="povb" title="PoVB (§5.4)">
              <div className="empty">No PoVB trace</div>
            </Panel>
          )}

          <ValidatorSetDiff
            signingSet={consensus?.signing_set ?? null}
            nextSet={consensus?.next_set ?? null}
            updates={data.results?.validator_updates ?? []}
          />

          <Panel id="txs" title="Transactions">
            <DataTable head={['#', 'Hash', 'Messages', 'Result']} empty="Empty block" count={txs.length}>
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
                        <Badge tone={res.code === 0 ? 'ok' : 'bad'}>
                          {res.code === 0 ? 'ok' : `err ${res.code}`}
                        </Badge>
                      ) : (
                        '—'
                      )}
                    </td>
                  </tr>
                )
              })}
            </DataTable>
          </Panel>

          <div id="events" className="stack">
            <EventGroups title="BeginBlock events" events={data.results?.begin_block_events ?? []} />
            <EventGroups title="EndBlock events" events={data.results?.end_block_events ?? []} />
            {!data.results?.begin_block_events?.length && !data.results?.end_block_events?.length ? (
              <Panel title="Events">
                <div className="empty">No events</div>
              </Panel>
            ) : null}
          </div>
        </>
      ) : null}
    </Page>
  )
}
