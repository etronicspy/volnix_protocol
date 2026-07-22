/**
 * Этап 5: explorer — поиск tx по hash + история по адресу (REST).
 * Источник: GET /api/tx/{hash}, GET /api/account/{addr}/history.
 */
import { useState } from 'react'

import { API_BASE } from '../config'
import { shortAddress, shortHash } from '../lib/format'

interface TxRecord {
  found: boolean
  height?: number
  tx?: Record<string, unknown>
  block_hash?: string
  block_timestamp?: number
}

interface HistoryItem {
  height: number
  tx_idx: number
  block_hash: string
  timestamp: number
}

export function TxExplorerPanel() {
  const [hashInput, setHashInput] = useState('')
  const [tx, setTx] = useState<TxRecord | null>(null)
  const [txError, setTxError] = useState<string | null>(null)

  const [addrInput, setAddrInput] = useState('')
  const [history, setHistory] = useState<HistoryItem[]>([])
  const [historyError, setHistoryError] = useState<string | null>(null)

  const lookupTx = async () => {
    setTxError(null)
    setTx(null)
    const h = hashInput.trim()
    if (!h) return
    try {
      const res = await fetch(`${API_BASE}/api/tx/${encodeURIComponent(h)}`)
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data = (await res.json()) as TxRecord
      setTx(data)
    } catch (e) {
      setTxError(e instanceof Error ? e.message : String(e))
    }
  }

  const lookupHistory = async () => {
    setHistoryError(null)
    setHistory([])
    const a = addrInput.trim()
    if (!a) return
    try {
      const res = await fetch(`${API_BASE}/api/account/${encodeURIComponent(a)}/history?limit=50`)
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data = (await res.json()) as { history: HistoryItem[] }
      setHistory(data.history ?? [])
    } catch (e) {
      setHistoryError(e instanceof Error ? e.message : String(e))
    }
  }

  return (
    <div className="bg-gray-800 p-4 rounded-lg shadow-md space-y-4">
      <h2 className="text-lg font-semibold text-gray-100">Tx explorer</h2>

      <div>
        <label className="block text-sm text-gray-400 mb-1">Lookup by tx hash</label>
        <div className="flex gap-2">
          <input
            className="flex-1 bg-gray-900 border border-gray-700 rounded px-2 py-1 text-sm text-gray-100"
            placeholder="tx_hash"
            value={hashInput}
            onChange={(e) => setHashInput(e.target.value)}
          />
          <button
            type="button"
            className="bg-blue-600 hover:bg-blue-500 text-white px-3 py-1 rounded text-sm"
            onClick={() => void lookupTx()}
          >
            find
          </button>
        </div>
        {txError && <div className="text-red-400 text-xs mt-1">{txError}</div>}
        {tx && !tx.found && <div className="text-yellow-300 text-xs mt-1">Not found</div>}
        {tx && tx.found && (
          <div className="mt-2 text-xs text-gray-200">
            <div>
              <span className="text-gray-400">height:</span> {tx.height}
            </div>
            <div>
              <span className="text-gray-400">block_hash:</span>{' '}
              {tx.block_hash ? shortHash(tx.block_hash) : '—'}
            </div>
            <pre className="bg-gray-900 p-2 rounded text-xs mt-1 overflow-auto max-h-40">
              {JSON.stringify(tx.tx, null, 2)}
            </pre>
          </div>
        )}
      </div>

      <div>
        <label className="block text-sm text-gray-400 mb-1">Account history</label>
        <div className="flex gap-2">
          <input
            className="flex-1 bg-gray-900 border border-gray-700 rounded px-2 py-1 text-sm text-gray-100"
            placeholder="address"
            value={addrInput}
            onChange={(e) => setAddrInput(e.target.value)}
          />
          <button
            type="button"
            className="bg-blue-600 hover:bg-blue-500 text-white px-3 py-1 rounded text-sm"
            onClick={() => void lookupHistory()}
          >
            find
          </button>
        </div>
        {historyError && <div className="text-red-400 text-xs mt-1">{historyError}</div>}
        {history.length > 0 && (
          <ul className="text-xs mt-2 space-y-1 max-h-40 overflow-auto">
            {history.map((h) => (
              <li key={`${h.height}-${h.tx_idx}`} className="text-gray-200">
                #{h.height} idx={h.tx_idx} block={shortHash(h.block_hash)}{' '}
                <span className="text-gray-500">
                  ts={new Date(h.timestamp * 1000).toLocaleTimeString()}
                </span>
              </li>
            ))}
          </ul>
        )}
        {addrInput && history.length === 0 && !historyError && (
          <div className="text-xs text-gray-500 mt-1">
            (нажмите «find» для {shortAddress(addrInput)})
          </div>
        )}
      </div>
    </div>
  )
}
