import { useMemo, useState } from 'react'
import { useOutletContext } from 'react-router-dom'
import { formatMicro } from '../lib/api'
import { useFetch } from '../hooks/useFetch'
import { useMempool } from '../hooks/useMempool'
import { cancelIds, pendingPlaces } from '../lib/pending'
import { HashLink } from '../components/HashLink'
import type { OrderBook, TradeRow } from '../types/api'

interface OutletCtx {
  livePulse: number
}

export function MarketPage() {
  const { livePulse } = useOutletContext<OutletCtx>()
  const [market, setMarket] = useState('ANT/WRT')
  const book = useFetch<OrderBook>(`/api/v1/market/orderbook?market=${encodeURIComponent(market)}`, livePulse)
  const trades = useFetch<{ trades: TradeRow[] }>('/api/v1/market/trades?tail=40', livePulse)
  const mempool = useMempool(livePulse)
  const dropping = useMemo(() => cancelIds(mempool.txs), [mempool.txs])
  const placing = useMemo(() => pendingPlaces(mempool.txs, market), [mempool.txs, market])

  return (
    <div className="page">
      <h1 className="page-title">Internal market</h1>
      <p className="page-lead">On-chain order books ANT/WRT and LZN/WRT — no direct MsgSend for those denoms.</p>

      <div className="row" style={{ marginBottom: '1rem' }}>
        {(['ANT/WRT', 'LZN/WRT'] as const).map((m) => (
          <button
            key={m}
            type="button"
            className={`btn ${market === m ? '' : 'secondary'}`}
            onClick={() => setMarket(m)}
          >
            {m}
          </button>
        ))}
      </div>

      {book.error ? <p className="error">{book.error}</p> : null}
      {mempool.rejects.length ? <p className="error">{mempool.rejects.join(' · ')}</p> : null}

      <div className="grid-2">
        <div className="panel">
          <div className="panel-title">Asks (sell)</div>
          <BookSide
            rows={book.data?.asks ?? []}
            dropping={dropping}
            pending={placing.filter((row) => row.side === 'SELL')}
          />
        </div>
        <div className="panel">
          <div className="panel-title">Bids (buy)</div>
          <BookSide
            rows={book.data?.bids ?? []}
            dropping={dropping}
            pending={placing.filter((row) => row.side === 'BUY')}
          />
        </div>
      </div>

      <div className="panel">
        <div className="panel-title">Recent trades</div>
        {trades.data?.trades?.length ? (
          <div className="table-wrap">
            <table className="data">
              <thead>
                <tr>
                  <th>Height</th>
                  <th>Market</th>
                  <th>Price</th>
                  <th>Amount</th>
                  <th>Maker / Taker</th>
                </tr>
              </thead>
              <tbody>
                {trades.data.trades.map((t, i) => (
                  <tr key={`${t.height}-${i}`}>
                    <td>
                      <HashLink height={t.height} />
                    </td>
                    <td>{t.market ?? '—'}</td>
                    <td className="mono">{t.price ?? '—'}</td>
                    <td className="mono">{t.amount ?? '—'}</td>
                    <td className="mono muted">
                      {t.maker ?? '—'} / {t.taker ?? '—'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="empty">No trades yet</div>
        )}
      </div>
    </div>
  )
}

function BookSide({
  rows,
  dropping,
  pending,
}: {
  rows: { order_id: string; owner: string; price: number; remaining: number }[]
  dropping: Set<string>
  pending: { hash: string; sender: string; amount: number; price: number }[]
}) {
  if (!rows.length && !pending.length) return <div className="empty">Empty</div>
  return (
    <div className="table-wrap">
      <table className="data">
        <thead>
          <tr>
            <th>Price</th>
            <th>Remaining</th>
            <th>Owner</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {rows.map((r) => (
            <tr key={r.order_id}>
              <td className="mono">{r.price}</td>
              <td className="mono">{formatMicro(r.remaining)}</td>
              <td>
                <HashLink address={r.owner} />
              </td>
              <td>{dropping.has(r.order_id) ? <span className="badge warn">cancelling</span> : null}</td>
            </tr>
          ))}
          {pending.map((row) => (
            <tr key={row.hash}>
              <td className="mono">{row.price}</td>
              <td className="mono">{formatMicro(row.amount)}</td>
              <td>
                <HashLink address={row.sender} />
              </td>
              <td>
                <span className="badge warn">order in mempool</span>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
