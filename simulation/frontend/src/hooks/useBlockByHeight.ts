import { useCallback, useEffect, useState } from 'react'

import { API_BASE } from '../config'
import type { Block } from '../types'

export interface UseBlockByHeightResult {
  block: Block | null
  found: boolean
  loading: boolean
  error: string | null
  reload: () => Promise<void>
}

interface BlockLookupResponse extends Partial<Block> {
  found?: boolean
  height?: number
}

export function useBlockByHeight(height: number | null): UseBlockByHeightResult {
  const [block, setBlock] = useState<Block | null>(null)
  const [found, setFound] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const reload = useCallback(async () => {
    if (height === null || !Number.isFinite(height) || height < 0) {
      setBlock(null)
      setFound(false)
      setError(null)
      return
    }
    setLoading(true)
    setError(null)
    try {
      const res = await fetch(`${API_BASE}/api/blocks/${height}`)
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data = (await res.json()) as BlockLookupResponse
      if (data.found === false) {
        setBlock(null)
        setFound(false)
        return
      }
      const loaded: Block = {
        height: Number(data.height ?? height),
        hash: String(data.hash ?? ''),
        tx_count: Number(data.tx_count ?? 0),
        timestamp: Number(data.timestamp ?? 0),
        proposer: typeof data.proposer === 'string' ? data.proposer : undefined,
        transactions: Array.isArray(data.transactions) ? data.transactions : [],
        competition: data.competition,
      }
      setBlock(loaded)
      setFound(true)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
      setBlock(null)
      setFound(false)
    } finally {
      setLoading(false)
    }
  }, [height])

  useEffect(() => {
    void reload()
  }, [reload])

  return { block, found, loading, error, reload }
}
