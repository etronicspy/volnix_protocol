import { useMemo, useState } from 'react'
import { formatMicro } from '../lib/api'
import { useFetch } from '../hooks/useFetch'
import { useLive } from '../hooks/useLive'
import { useMempool } from '../hooks/useMempool'
import { cancelIds, pendingPlaces } from '../lib/pending'
import type { PendingPlace } from '../lib/pending'
import { HashLink } from '../components/HashLink'
import { Badge } from '../components/ui/Badge'
import { DataTable } from '../components/ui/DataTable'
import { Page } from '../components/ui/Page'
import { Panel } from '../components/ui/Panel'
import type { BookLevel, OrderBook, TradeRow } from '../types/api'

export function MarketPage() {
  const { chainTick, mempoolTick } = useLive()
  const [market, setMarket] = useState('ANT/WRT')
  const book = useFetch<OrderBook>(`/api/v1/market/orderbook?market=${encodeURIComponent(market)}`, chainTick)
  const trades = useFetch<{ trades: TradeRow[] }>('/api/v1/market/trades?tail=40', chainTick)
  const mempool = useMempool(mempoolTick)
  const dropping = useMemo(() => cancelIds(mempool.txs), [mempool.txs])
  const placing = useMemo(() => pendingPlaces(mempool.txs, market), [mempool.txs, market])
  const tradeRows = trades.data?.trades ?? []

  return (
    <Page
      title="Internal market"
      lead="On-chain order books ANT/WRT and LZN/WRT — no direct MsgSend for those denoms."
    >
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
        <Panel title="Asks (sell)">
          <BookSide rows={book.data?.asks ?? []} dropping={dropping} pending={placing.filter((row) => row.side === 'SELL')} />
        </Panel>
        <Panel title="Bids (buy)">
          <BookSide rows={book.data?.bids ?? []} dropping={dropping} pending={placing.filter((row) => row.side === 'BUY')} />
        </Panel>
      </div>

      <Panel title="Recent trades">
        <DataTable head={['Height', 'Market', 'Price (WRT)', 'Amount', 'Maker / Taker']} empty="No trades yet" count={tradeRows.length}>
          {tradeRows.map((t, i) => (
            <tr key={`${t.height}-${i}`}>
              <td>
                <HashLink height={t.height} />
              </td>
              <td>{t.market ?? '—'}</td>
              <td className="mono">{t.price != null ? formatMicro(t.price) : '—'}</td>
              <td className="mono">{t.amount != null ? formatMicro(t.amount) : '—'}</td>
              <td className="mono muted">
                {t.maker ?? '—'} / {t.taker ?? '—'}
              </td>
            </tr>
          ))}
        </DataTable>
      </Panel>
    </Page>
  )
}

function BookSide({
  rows,
  dropping,
  pending,
}: {
  rows: BookLevel[]
  dropping: Set<string>
  pending: PendingPlace[]
}) {
  return (
    <DataTable head={['Price (WRT)', 'Remaining', 'Owner', '']} empty="Empty" count={rows.length + pending.length}>
      {rows.map((r) => (
        <tr key={r.order_id}>
          <td className="mono">{formatMicro(r.price)}</td>
          <td className="mono">{formatMicro(r.remaining)}</td>
          <td>
            <HashLink address={r.owner} />
          </td>
          <td>{dropping.has(r.order_id) ? <Badge tone="warn">cancelling</Badge> : null}</td>
        </tr>
      ))}
      {pending.map((row) => (
        <tr key={row.hash}>
          <td className="mono">{formatMicro(row.price)}</td>
          <td className="mono">{formatMicro(row.amount)}</td>
          <td>
            <HashLink address={row.sender} />
          </td>
          <td>
            <Badge tone="warn">order in mempool</Badge>
          </td>
        </tr>
      ))}
    </DataTable>
  )
}
