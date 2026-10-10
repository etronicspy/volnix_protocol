import { useEffect, useRef, useState } from 'react'
import { fetchMempool, settleTx } from '../lib/pending'
import type { PendingTx } from '../lib/pending'

export function useMempool(refreshKey: number, address?: string) {
  const [txs, setTxs] = useState<PendingTx[]>([])
  const [rejects, setRejects] = useState<string[]>([])
  const watched = useRef<Set<string>>(new Set())

  useEffect(() => {
    let cancelled = false
    void (async () => {
      try {
        const live = await fetchMempool(address)
        if (cancelled) return
        const hashes = new Set(live.map((tx) => tx.hash))
        const left = [...watched.current].filter((hash) => !hashes.has(hash))
        const settled = await Promise.all(left.map((hash) => settleTx(hash).then((s) => ({ hash, s }))))
        if (cancelled) return
        const failures: string[] = []
        for (const { hash, s } of settled) {
          if (!s) continue
          watched.current.delete(hash)
          if (s.code !== 0) failures.push(s.log || `rejected (${s.code})`)
        }
        for (const tx of live) watched.current.add(tx.hash)
        setTxs(live)
        if (failures.length > 0) setRejects(failures.slice(-5))
      } catch {
        if (!cancelled) setTxs([])
      }
    })()
    return () => {
      cancelled = true
    }
  }, [refreshKey, address])

  return { txs, rejects }
}
