import { useParams } from 'react-router-dom'
import { useFetch } from '../hooks/useFetch'
import { useLive } from '../hooks/useLive'
import { useMempool } from '../hooks/useMempool'
import { HashLink } from '../components/HashLink'
import { Badge } from '../components/ui/Badge'
import { EventList } from '../components/ui/EventList'
import { Page } from '../components/ui/Page'
import { Panel } from '../components/ui/Panel'
import { Stat } from '../components/ui/Stat'
import type { TxDetail } from '../types/api'

export function TxDetailPage() {
  const { hash } = useParams()
  const { chainTick, mempoolTick } = useLive()
  const { data, error, loading } = useFetch<TxDetail>(hash ? `/api/v1/txs/${hash}` : null, chainTick)
  const mempool = useMempool(mempoolTick)
  const pending = data ? null : (mempool.txs.find((tx) => tx.hash === hash) ?? null)
  const messages = data
    ? ((data.tx.body as { messages?: Record<string, unknown>[] } | undefined)?.messages ?? [])
    : (pending?.messages ?? [])

  return (
    <Page
      title="Transaction"
      back={{ to: '/', label: 'Overview' }}
      error={error && !pending ? error : null}
      loading={loading && !data && !pending}
    >
      {data || pending ? (
        <>
          <div className="grid-3" style={{ marginBottom: '1rem' }}>
            <Stat label="Hash" sm>
              {data?.hash ?? pending?.hash}
            </Stat>
            {data ? (
              <Stat label="Height">
                <HashLink height={data.height} />
              </Stat>
            ) : (
              <Stat label="Status" sm>
                <Badge tone="warn">in mempool</Badge>
              </Stat>
            )}
            <Stat label="Sender" sm>
              <HashLink address={data?.sender ?? pending?.sender} />
            </Stat>
            {data ? (
              <>
                <Stat label="Result" sm>
                  {data.result ? (
                    <Badge tone={data.result.code === 0 ? 'ok' : 'bad'}>
                      {data.result.code === 0 ? 'ok' : data.result.log}
                    </Badge>
                  ) : (
                    '—'
                  )}
                </Stat>
                <Stat label="Gas" sm>
                  {data.result ? `${data.result.gas_used} / ${data.result.gas_wanted}` : '—'}
                </Stat>
              </>
            ) : null}
          </div>

          <Panel title="Messages">
            {pending ? <p className="muted">{pending.label}</p> : null}
            {messages.length ? (
              <div className="stack">
                {messages.map((m, i) => (
                  <div key={i} className="pre">
                    {JSON.stringify(m, null, 2)}
                  </div>
                ))}
              </div>
            ) : (
              <div className="empty">No messages</div>
            )}
          </Panel>

          {data?.result?.events?.length ? (
            <Panel title="Events">
              <EventList events={data.result.events} />
            </Panel>
          ) : null}
        </>
      ) : null}
    </Page>
  )
}
