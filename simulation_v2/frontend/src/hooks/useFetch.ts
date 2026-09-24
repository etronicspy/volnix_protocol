import { useCallback, useEffect, useRef, useState } from 'react'
import { api, ApiError } from '../lib/api'

interface FetchState<T> {
  data: T | null
  error: string | null
  loading: boolean
}

export function useFetch<T>(path: string | null, refreshKey = 0): FetchState<T> & { reload: () => void } {
  const [data, setData] = useState<T | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(Boolean(path))
  const [tick, setTick] = useState(0)

  const reload = useCallback(() => setTick((t) => t + 1), [])

  useEffect(() => {
    if (!path) {
      setData(null)
      setError(null)
      setLoading(false)
      return
    }
    let cancelled = false
    setLoading(true)
    api
      .get<T>(path)
      .then((d) => {
        if (!cancelled) {
          setData(d)
          setError(null)
        }
      })
      .catch((e: unknown) => {
        if (!cancelled) {
          setError(e instanceof ApiError ? e.message : 'Request failed')
          setData(null)
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [path, refreshKey, tick])

  return { data, error, loading, reload }
}

export function useLiveRefresh(intervalMs = 0): number {
  const [key, setKey] = useState(0)
  const bump = useCallback(() => setKey((k) => k + 1), [])
  const bumpRef = useRef(bump)
  bumpRef.current = bump

  useEffect(() => {
    if (intervalMs <= 0) return
    const id = window.setInterval(() => bumpRef.current(), intervalMs)
    return () => window.clearInterval(id)
  }, [intervalMs])

  return key
}
