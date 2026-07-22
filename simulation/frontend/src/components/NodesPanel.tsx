/**
 * Phase C: визуализация NetworkSim — узлы, кворум последнего блока, gossip lag,
 * runtime-конфиг (latency/loss/quorum).
 *
 * REST:
 *   - GET  /api/network/nodes
 *   - GET  /api/network/topology
 *   - POST /api/network/config
 */
import { useState } from 'react'

import { useNetwork, type NetworkConfig } from '../hooks/useNetwork'

function voteBadge(vote?: string): { label: string; cls: string } {
  if (vote === 'commit') return { label: 'commit', cls: 'bg-emerald-700 text-emerald-100' }
  if (vote === 'nil') return { label: 'nil', cls: 'bg-amber-700 text-amber-100' }
  if (vote === 'absent') return { label: 'absent', cls: 'bg-gray-700 text-gray-200' }
  if (vote === 'double_sign')
    return { label: 'double_sign', cls: 'bg-rose-700 text-rose-100' }
  if (!vote) return { label: '—', cls: 'bg-gray-800 text-gray-500' }
  return { label: vote, cls: 'bg-slate-700 text-slate-100' }
}

export function NodesPanel() {
  const { enabled, nodes, topology, loading, error, reload, setConfig } = useNetwork(5000)
  const cfg: NetworkConfig | null = topology?.config ?? null

  const [draftLatency, setDraftLatency] = useState<string>('')
  const [draftLoss, setDraftLoss] = useState<string>('')
  const [draftQuorum, setDraftQuorum] = useState<string>('')

  const apply = async () => {
    const patch: Partial<NetworkConfig> = {}
    if (draftLatency.trim() !== '') patch.gossip_latency_ms = Number(draftLatency)
    if (draftLoss.trim() !== '') patch.gossip_loss_pct = Number(draftLoss)
    if (draftQuorum.trim() !== '') patch.quorum_pct = Number(draftQuorum)
    if (Object.keys(patch).length === 0) return
    await setConfig(patch)
    setDraftLatency('')
    setDraftLoss('')
    setDraftQuorum('')
  }

  if (!enabled) {
    return (
      <div className="bg-gray-800 p-4 rounded-lg shadow-md">
        <h2 className="text-lg font-semibold text-gray-100 mb-2">Сеть (NetworkSim)</h2>
        <p className="text-gray-400 text-sm">
          NetworkSim выключен. Перезапустите бэкенд с переменными окружения{' '}
          <code className="text-amber-300">VOLNIX_SIM_NUM_NODES&gt;=2</code>
          {' '}(по умолчанию 1), либо вызовите шаг сценария <code>network_set</code>.
        </p>
      </div>
    )
  }

  return (
    <div className="bg-gray-800 p-4 rounded-lg shadow-md space-y-4">
      <div className="flex justify-between items-center">
        <h2 className="text-lg font-semibold text-gray-100">Сеть (NetworkSim)</h2>
        <button
          type="button"
          className="text-xs text-blue-400 hover:underline"
          onClick={() => void reload()}
        >
          {loading ? '…' : 'reload'}
        </button>
      </div>

      {error && <div className="text-red-400 text-sm">{error}</div>}

      {cfg && (
        <div className="text-xs text-gray-300 grid grid-cols-2 sm:grid-cols-4 gap-2">
          <div>
            Узлов: <span className="font-mono">{cfg.num_nodes}</span>
          </div>
          <div>
            Latency: <span className="font-mono">{cfg.gossip_latency_ms} мс</span>
          </div>
          <div>
            Loss: <span className="font-mono">{cfg.gossip_loss_pct}%</span>
          </div>
          <div>
            Quorum: <span className="font-mono">{(cfg.quorum_pct * 100).toFixed(1)}%</span>
          </div>
        </div>
      )}

      <div>
        <h3 className="text-sm font-semibold text-gray-300 mb-2">Узлы</h3>
        <div className="overflow-x-auto">
          <table className="w-full text-xs text-left text-gray-300">
            <thead className="text-gray-400 uppercase">
              <tr>
                <th className="py-1 pr-2">ID</th>
                <th className="py-1 pr-2">Адресов</th>
                <th className="py-1 pr-2">Mempool</th>
                <th className="py-1 pr-2">Lag, мс</th>
                <th className="py-1 pr-2">Роль</th>
                <th className="py-1 pr-2">Голос (последний блок)</th>
              </tr>
            </thead>
            <tbody>
              {nodes.map((n) => {
                const validatorAddrs = n.validators
                return (
                  <tr key={n.id} className="border-t border-gray-700">
                    <td className="py-1 pr-2 font-mono text-gray-100">
                      {n.id} {n.is_proposer && <span className="text-amber-400">★</span>}
                    </td>
                    <td className="py-1 pr-2 font-mono">{n.addresses.length}</td>
                    <td className="py-1 pr-2 font-mono">{n.mempool_size}</td>
                    <td className="py-1 pr-2 font-mono">{n.last_gossip_lag_ms}</td>
                    <td className="py-1 pr-2">
                      {n.is_proposer ? 'proposer' : validatorAddrs.length ? 'validator' : '—'}
                    </td>
                    <td className="py-1 pr-2">
                      {validatorAddrs.length === 0 && <span className="text-gray-500">—</span>}
                      <div className="flex flex-wrap gap-1">
                        {validatorAddrs.map((addr) => {
                          const v = n.votes[addr] ?? {}
                          const pv = voteBadge(v.pre_vote)
                          const pc = voteBadge(v.pre_commit)
                          return (
                            <span
                              key={addr}
                              className="flex items-center gap-1"
                              title={addr}
                            >
                              <span className={`px-1 rounded ${pv.cls}`}>PV:{pv.label}</span>
                              <span className={`px-1 rounded ${pc.cls}`}>PC:{pc.label}</span>
                            </span>
                          )
                        })}
                      </div>
                    </td>
                  </tr>
                )
              })}
              {nodes.length === 0 && (
                <tr>
                  <td colSpan={6} className="py-2 text-gray-500">
                    нет узлов
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>

      <div>
        <h3 className="text-sm font-semibold text-gray-300 mb-2">Runtime config</h3>
        <div className="flex flex-wrap gap-2 items-end">
          <label className="text-xs text-gray-400 flex flex-col">
            Latency, мс
            <input
              className="bg-gray-900 text-gray-100 font-mono p-1 rounded w-24"
              placeholder={cfg ? String(cfg.gossip_latency_ms) : ''}
              value={draftLatency}
              onChange={(e) => setDraftLatency(e.target.value)}
            />
          </label>
          <label className="text-xs text-gray-400 flex flex-col">
            Loss, %
            <input
              className="bg-gray-900 text-gray-100 font-mono p-1 rounded w-24"
              placeholder={cfg ? String(cfg.gossip_loss_pct) : ''}
              value={draftLoss}
              onChange={(e) => setDraftLoss(e.target.value)}
            />
          </label>
          <label className="text-xs text-gray-400 flex flex-col">
            Quorum (0..1)
            <input
              className="bg-gray-900 text-gray-100 font-mono p-1 rounded w-24"
              placeholder={cfg ? String(cfg.quorum_pct) : ''}
              value={draftQuorum}
              onChange={(e) => setDraftQuorum(e.target.value)}
            />
          </label>
          <button
            type="button"
            onClick={() => void apply()}
            className="px-3 py-1 rounded bg-blue-600 hover:bg-blue-700 text-white text-xs"
          >
            Apply
          </button>
        </div>
      </div>
    </div>
  )
}
