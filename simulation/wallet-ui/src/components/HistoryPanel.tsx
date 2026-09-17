import { useEffect, useState } from 'react'

import { useBackend } from '../adapters'
import type { Transaction } from '../types'
import { Panel, formatAddr, formatAmt } from './ui'

interface HistoryPanelProps {
  address: string
  height: number
}

export function HistoryPanel({ address, height }: HistoryPanelProps) {
  const { adapter, mode } = useBackend()
  const [rows, setRows] = useState<Transaction[]>([])
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    void adapter
      .fetchHistory(address)
      .then((data) => {
        if (!cancelled) setRows(data.history ?? [])
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [adapter, address, height])

  const isSim = mode === 'sim'

  return (
    <Panel
      title="История"
      hint={isSim
        ? 'GET /api/account/{address}/history — обновляется с каждым блоком.'
        : 'Поиск tx через CometBFT RPC.'}
    >
      {loading && rows.length === 0 ? (
        <p className="text-sm text-[var(--muted)]">Загрузка…</p>
      ) : rows.length === 0 ? (
        <p className="text-sm text-[var(--muted)]">Пока нет транзакций для этого адреса.</p>
      ) : (
        <ul className="divide-y divide-[var(--line)]">
          {rows.map((tx) => (
            <li
              key={`${tx.tx_hash}-${tx.timestamp}`}
              className="flex flex-wrap items-start justify-between gap-3 py-3"
              data-tip={tx.details || `${tx.tx_type} · ${tx.tx_hash}`}
            >
              <div>
                <div className="font-medium capitalize text-[var(--sand)]">{tx.tx_type.replaceAll('_', ' ')}</div>
                <div className="mono mt-1 text-xs text-[var(--muted)]">
                  {formatAddr(tx.tx_hash, 12, 8)}
                  {tx.receiver ? ` → ${formatAddr(tx.receiver)}` : ''}
                </div>
                {tx.details ? <div className="mt-1 text-xs text-[var(--muted)]">{tx.details}</div> : null}
              </div>
              <div className="text-right">
                <div className="mono text-[var(--mint)]">
                  {formatAmt(tx.amount)} {tx.asset_type?.toUpperCase() ?? ''}
                </div>
                <div className="text-xs text-[var(--muted)]">
                  {tx.timestamp ? new Date(tx.timestamp * 1000).toLocaleString() : '—'}
                </div>
              </div>
            </li>
          ))}
        </ul>
      )}
    </Panel>
  )
}
