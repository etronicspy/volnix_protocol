import { useEffect, useState } from 'react'
import { Link, useOutletContext, useParams } from 'react-router-dom'
import { useFetch } from '../hooks/useFetch'
import { findPending } from '../lib/pending'
import type { PendingTx } from '../lib/pending'
import { HashLink } from '../components/HashLink'
import type { TxDetail } from '../types/api'

interface OutletCtx {
  livePulse: number
}

export function TxDetailPage() {
  const { hash } = useParams()
  const { livePulse } = useOutletContext<OutletCtx>()
  const { data, error, loading } = useFetch<TxDetail>(hash ? `/api/v1/txs/${hash}` : null, livePulse)
  const [pending, setPending] = useState<PendingTx | null>(null)

  useEffect(() => {
    if (!hash || data) {
      setPending(null)
      return
    }
    let cancelled = false
    void findPending(hash).then((tx) => {
      if (!cancelled) setPending(tx)
    })
    return () => {
      cancelled = true
    }
  }, [hash, data, livePulse, error])
  const messages =
    (data?.tx?.body as { messages?: Record<string, unknown>[] } | undefined)?.messages ?? []

  return (
    <div className="page">
      <p className="muted">
        <Link to="/">← Overview</Link>
      </p>
      <h1 className="page-title">Transaction</h1>
      {error && !pending ? <p className="error">{error}</p> : null}
      {loading && !data && !pending ? <p className="muted">Loading…</p> : null}

      {pending && !data ? (
        <>
          <div className="grid-3" style={{ marginBottom: '1rem' }}>
            <div className="stat">
              <div className="stat-label">Hash</div>
              <div className="stat-value sm">{pending.hash}</div>
            </div>
            <div className="stat">
              <div className="stat-label">Status</div>
              <div className="stat-value sm">
                <span className="badge warn">in mempool</span>
              </div>
            </div>
            <div className="stat">
              <div className="stat-label">Sender</div>
              <div className="stat-value sm">
                <HashLink address={pending.sender} />
              </div>
            </div>
          </div>
          <div className="panel">
            <div className="panel-title">Messages</div>
            <p className="muted">{pending.label}</p>
            <div className="stack">
              {pending.messages.map((m, i) => (
                <div key={i} className="pre">
                  {JSON.stringify(m, null, 2)}
                </div>
              ))}
            </div>
          </div>
        </>
      ) : null}

      {data ? (
        <>
          <div className="grid-3" style={{ marginBottom: '1rem' }}>
            <div className="stat">
              <div className="stat-label">Hash</div>
              <div className="stat-value sm">{data.hash}</div>
            </div>
            <div className="stat">
              <div className="stat-label">Height</div>
              <div className="stat-value">
                <HashLink height={data.height} />
              </div>
            </div>
            <div className="stat">
              <div className="stat-label">Sender</div>
              <div className="stat-value sm">
                <HashLink address={data.sender} />
              </div>
            </div>
            <div className="stat">
              <div className="stat-label">Result</div>
              <div className="stat-value sm">
                {data.result ? (
                  <span className={`badge ${data.result.code === 0 ? 'ok' : 'bad'}`}>
                    {data.result.code === 0 ? 'ok' : data.result.log}
                  </span>
                ) : (
                  '—'
                )}
              </div>
            </div>
            <div className="stat">
              <div className="stat-label">Gas</div>
              <div className="stat-value sm">
                {data.result ? `${data.result.gas_used} / ${data.result.gas_wanted}` : '—'}
              </div>
            </div>
          </div>

          <div className="panel">
            <div className="panel-title">Messages</div>
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
          </div>

          {data.result?.events?.length ? (
            <div className="panel">
              <div className="panel-title">Events</div>
              <div className="stack">
                {data.result.events.map((e, i) => (
                  <div key={i}>
                    <span className="badge muted">{e.type}</span>
                    <div className="mono muted" style={{ marginTop: 4, fontSize: '0.8rem' }}>
                      {e.attributes.map((a) => `${a.key}=${a.value}`).join(' · ')}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          ) : null}
        </>
      ) : null}
    </div>
  )
}
