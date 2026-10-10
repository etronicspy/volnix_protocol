import { useMemo } from 'react'
import { useParams } from 'react-router-dom'
import { formatMicro } from '../lib/api'
import { useFetch } from '../hooks/useFetch'
import { useLive } from '../hooks/useLive'
import { useMempool } from '../hooks/useMempool'
import { cancelIds, pendingPlaces } from '../lib/pending'
import { Badge } from '../components/ui/Badge'
import { DataTable } from '../components/ui/DataTable'
import { Page } from '../components/ui/Page'
import { Panel } from '../components/ui/Panel'
import { Stat } from '../components/ui/Stat'
import { TxTable } from '../components/ui/TxTable'
import type { AccountView, TxListItem } from '../types/api'

export function AccountPage() {
  const { address } = useParams()
  const { chainTick, mempoolTick } = useLive()
  const acc = useFetch<AccountView>(address ? `/api/v1/accounts/${address}` : null, chainTick)
  const txs = useFetch<{ txs: TxListItem[] }>(address ? `/api/v1/accounts/${address}/txs` : null, chainTick)
  const mempool = useMempool(mempoolTick, address)
  const dropping = useMemo(() => cancelIds(mempool.txs), [mempool.txs])
  const placing = useMemo(() => pendingPlaces(mempool.txs), [mempool.txs])

  const a = acc.data?.account
  const bal = acc.data?.balances_display
  const orders = acc.data?.open_orders ?? []

  return (
    <Page
      title="Account"
      lead={<span className="mono">{address}</span>}
      back={{ to: '/wallets', label: 'Wallets' }}
      error={acc.error}
      loading={acc.loading && !a}
    >
      {a && bal ? (
        <>
          <div className="grid-3" style={{ marginBottom: '1rem' }}>
            <Stat label="Role">{a.role}</Stat>
            <Stat label="WRT">{bal.wrt}</Stat>
            <Stat label="ANT">{bal.ant}</Stat>
            <Stat label="LZN free">{bal.lzn}</Stat>
            <Stat label="LZN activated">{bal.lzn_activated}</Stat>
            <Stat label="Sequence / last tx" sm>
              {a.sequence} / h{a.last_tx_height}
            </Stat>
          </div>

          {a.genesis_no_zkp ? <p className="muted">Genesis validator (no ZKP).</p> : null}

          {acc.data?.last_declare ? (
            <Panel title="Last declare (in-memory this height)">
              <div className="mono">
                b_i={acc.data.last_declare.b_i} · s_i={acc.data.last_declare.s_i} · f_i=
                {acc.data.last_declare.f_i} · w_i={acc.data.last_declare.w_i.toFixed(4)}
              </div>
            </Panel>
          ) : null}

          <Panel title="Open orders">
            <DataTable
              head={['ID', 'Market', 'Side', 'Amount', 'Price (WRT)', '']}
              empty="No open orders"
              count={orders.length + placing.length}
            >
              {orders.map((o) => (
                <tr key={o.order_id}>
                  <td className="mono">{o.order_id}</td>
                  <td>{o.market}</td>
                  <td>{o.side}</td>
                  <td className="mono">{formatMicro(o.amount)}</td>
                  <td className="mono">{formatMicro(o.price)}</td>
                  <td>{dropping.has(o.order_id) ? <Badge tone="warn">cancelling</Badge> : null}</td>
                </tr>
              ))}
              {placing.map((row) => (
                <tr key={row.hash}>
                  <td className="mono">—</td>
                  <td>{row.market}</td>
                  <td>{row.side}</td>
                  <td className="mono">{formatMicro(row.amount)}</td>
                  <td className="mono">{formatMicro(row.price)}</td>
                  <td>
                    <Badge tone="warn">order in mempool</Badge>
                  </td>
                </tr>
              ))}
            </DataTable>
          </Panel>

          <Panel title="Transaction history">
            <TxTable
              pending={mempool.txs}
              txs={txs.data?.txs ?? []}
              extraCol="index"
              empty="No transactions"
              rejects={mempool.rejects}
            />
          </Panel>
        </>
      ) : null}
    </Page>
  )
}
