import { useEffect, useRef, useState } from 'react'
import { api, ApiError } from '../lib/api'

interface FetchState<T> {
  data: T | null
  error: string | null
  loading: boolean
}

export function useFetch<T>(path: string | null, refreshKey = 0): FetchState<T> {
  const [data, setData] = useState<T | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [readyPath, setReadyPath] = useState<string | null>(null)
  const pathRef = useRef(path)

  useEffect(() => {
    pathRef.current = path
    if (!path) return
    const requested = path
    api
      .get<T>(requested)
      .then((d) => {
        if (pathRef.current !== requested) return
        setReadyPath(requested)
        setData(d)
        setError(null)
      })
      .catch((e: unknown) => {
        if (pathRef.current !== requested) return
        setReadyPath(requested)
        setError(e instanceof ApiError ? e.message : 'Request failed')
      })
  }, [path, refreshKey])

  if (!path) return { data: null, error: null, loading: false }
  if (readyPath !== path) return { data: null, error: null, loading: true }
  return { data, error, loading: false }
}

export function useLiveRefresh(intervalMs = 0): number {
  const [key, setKey] = useState(0)

  useEffect(() => {
    if (intervalMs <= 0) return
    const id = window.setInterval(() => setKey((k) => k + 1), intervalMs)
    return () => window.clearInterval(id)
  }, [intervalMs])

  return key
}
