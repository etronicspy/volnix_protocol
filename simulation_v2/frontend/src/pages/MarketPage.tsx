import { useState } from 'react'
import { useOutletContext } from 'react-router-dom'
import { formatMicro } from '../lib/api'
import { useFetch } from '../hooks/useFetch'
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

      <div className="grid-2">
        <div className="panel">
          <div className="panel-title">Asks (sell)</div>
          <BookSide rows={book.data?.asks ?? []} />
        </div>
        <div className="panel">
          <div className="panel-title">Bids (buy)</div>
          <BookSide rows={book.data?.bids ?? []} />
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
}: {
  rows: { order_id: string; owner: string; price: number; remaining: number }[]
}) {
  if (!rows.length) return <div className="empty">Empty</div>
  return (
    <div className="table-wrap">
      <table className="data">
        <thead>
          <tr>
            <th>Price</th>
            <th>Remaining</th>
            <th>Owner</th>
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
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
