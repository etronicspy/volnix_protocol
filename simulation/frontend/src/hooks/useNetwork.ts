/**
 * Multi-node sim (Phase C): чтение состояния NetworkSim + горячая конфигурация
 * gossip_latency_ms / gossip_loss_pct / quorum_pct.
 *
 * Источники:
 *   - GET  /api/network/nodes     (Узлы + последний голос за блок)
 *   - GET  /api/network/topology  (peers + config)
 *   - POST /api/network/config    (runtime изменение латентности/loss/кворума)
 */
import { useEffect, useRef, useState } from 'react'

import { API_BASE } from '../config'

export interface NodeVote {
  pre_vote?: string
  pre_commit?: string
}

export interface NodeSummary {
  id: string
  addresses: string[]
  validators: string[]
  mempool_size: number
  last_gossip_lag_ms: number
  is_proposer: boolean
  votes: Record<string, NodeVote>
}

export interface NetworkConfig {
  num_nodes: number
  gossip_latency_ms: number
  gossip_loss_pct: number
  quorum_pct: number
}

export interface NetworkTopology {
  enabled: boolean
  config: NetworkConfig | null
  peers: Record<string, string[]>
}

export interface UseNetworkResult {
  enabled: boolean
  nodes: NodeSummary[]
  topology: NetworkTopology | null
  loading: boolean
  error: string | null
  reload: () => Promise<void>
  setConfig: (patch: Partial<NetworkConfig>) => Promise<NetworkConfig | null>
}

export function useNetwork(intervalMs = 5000): UseNetworkResult {
  const [enabled, setEnabled] = useState(false)
  const [nodes, setNodes] = useState<NodeSummary[]>([])
  const [topology, setTopology] = useState<NetworkTopology | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const aborted = useRef(false)

  const reload = async () => {
    setLoading(true)
    setError(null)
    try {
      const [n, t] = await Promise.all([
        fetch(`${API_BASE}/api/network/nodes`).then((r) => r.json()),
        fetch(`${API_BASE}/api/network/topology`).then((r) => r.json()),
      ])
      if (aborted.current) return
      setEnabled(Boolean(n?.enabled))
      setNodes(Array.isArray(n?.nodes) ? (n.nodes as NodeSummary[]) : [])
      setTopology(t as NetworkTopology)
    } catch (e) {
      if (!aborted.current) setError(e instanceof Error ? e.message : String(e))
    } finally {
      if (!aborted.current) setLoading(false)
    }
  }

  const setConfig = async (patch: Partial<NetworkConfig>): Promise<NetworkConfig | null> => {
    try {
      const res = await fetch(`${API_BASE}/api/network/config`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(patch),
      })
      const data = await res.json()
      if (!data?.ok) {
        setError(data?.message ?? `HTTP ${res.status}`)
        return null
      }
      void reload()
      return (data.config as NetworkConfig) ?? null
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
      return null
    }
  }

  useEffect(() => {
    aborted.current = false
    void reload()
    if (intervalMs <= 0) return undefined
    const id = setInterval(() => void reload(), intervalMs)
    return () => {
      aborted.current = true
      clearInterval(id)
    }
  }, [intervalMs])

  return { enabled, nodes, topology, loading, error, reload, setConfig }
}
