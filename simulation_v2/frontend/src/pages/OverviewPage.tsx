import { formatMicro, formatTime } from '../lib/api'
import { useFetch } from '../hooks/useFetch'
import { useLive } from '../hooks/useLive'
import { useMempool } from '../hooks/useMempool'
import { HashLink } from '../components/HashLink'
import { DataTable } from '../components/ui/DataTable'
import { Page } from '../components/ui/Page'
import { Panel } from '../components/ui/Panel'
import { Stat } from '../components/ui/Stat'
import { TxTable } from '../components/ui/TxTable'
import type { BlockSummary, ChainSummary, TxListItem } from '../types/api'

export function OverviewPage() {
  const { chainTick, mempoolTick } = useLive()
  const summary = useFetch<ChainSummary>('/api/v1/chain/summary', chainTick)
  const blocks = useFetch<{ blocks: BlockSummary[] }>('/api/v1/blocks?tail=12', chainTick)
  const txs = useFetch<{ txs: TxListItem[] }>('/api/v1/txs?limit=12', chainTick)
  const mempool = useMempool(mempoolTick)

  const s = summary.data
  const recent = blocks.data?.blocks ?? []

  return (
    <Page
      title="Chain overview"
      lead="Live view of Volnix sim-2 — height, supply, PoVB parameters, recent blocks."
      error={summary.error}
      loading={summary.loading && !s}
    >
      {s ? (
        <>
          <div className="grid-3" style={{ marginBottom: '1rem' }}>
            <Stat label="Height">{s.height}</Stat>
            <Stat label="Epoch">
              {s.epoch} <span className="muted" style={{ fontSize: '0.85rem' }}>({s.blocks_to_epoch} left)</span>
            </Stat>
            <Stat label="L_total">{s.l_total_display}</Stat>
            <Stat label="WRT supply">{s.supply.wrt_display}</Stat>
            <Stat label="ANT supply">{s.supply.ant_display}</Stat>
            <Stat label="LZN minted">{s.supply.lzn_minted_tokens.toLocaleString()}</Stat>
          </div>

          <Panel title="Canon parameters">
            <div className="grid-3">
              <Stat label="λ / α / K" sm asWritten>
                {s.params.lambda} · {s.params.alpha} · {s.params.k}
              </Stat>
              <Stat label="Block reward" sm>
                {formatMicro(s.params.current_block_reward)} WRT
              </Stat>
              <Stat label="Validators / suppliers" sm>
                {s.n_validators} / {s.n_suppliers}
              </Stat>
              <Stat label="Mempool" sm>
                {s.mempool} txs
              </Stat>
              <Stat label="App hash" sm>
                {s.app_hash.slice(0, 16)}…
              </Stat>
              <Stat label="Chain time" sm>
                {s.time}
              </Stat>
            </div>
          </Panel>
        </>
      ) : null}

      <div className="grid-2" style={{ marginTop: '1rem' }}>
        <Panel title="Recent blocks">
          <DataTable head={['Height', 'Txs', 'Proposer', 'Time']} empty="No blocks yet" count={recent.length}>
            {recent.map((b) => (
              <tr key={b.height}>
                <td>
                  <HashLink height={b.height} />
                </td>
                <td>{b.n_txs}</td>
                <td>
                  <HashLink address={b.proposer} />
                </td>
                <td className="mono muted">{formatTime(b.time)}</td>
              </tr>
            ))}
          </DataTable>
        </Panel>

        <Panel title="Recent transactions">
          <TxTable
            pending={mempool.txs}
            txs={txs.data?.txs ?? []}
            empty="No transactions yet"
            rejects={mempool.rejects}
          />
        </Panel>
      </div>
    </Page>
  )
}
