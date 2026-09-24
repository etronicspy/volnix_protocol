import { useEffect, useRef, useState } from 'react'
import { api } from '../lib/api'
import styles from './BlockSpeedControl.module.css'

/** Wall-clock produce interval: 1 ms … 60 s per block. */
const MIN_MS = 1
const MAX_MS = 60_000
const SLIDER_MAX = 1000

interface ProduceIntervalResponse {
  interval_sec: number
  interval_ms: number
  auto_produce: boolean
}

function msToSlider(ms: number): number {
  const clamped = Math.min(MAX_MS, Math.max(MIN_MS, ms))
  const t = Math.log(clamped / MIN_MS) / Math.log(MAX_MS / MIN_MS)
  return Math.round(t * SLIDER_MAX)
}

function sliderToMs(pos: number): number {
  const t = Math.min(SLIDER_MAX, Math.max(0, pos)) / SLIDER_MAX
  return Math.round(MIN_MS * Math.pow(MAX_MS / MIN_MS, t))
}

function formatRate(ms: number): string {
  if (ms < 1000) return `${ms} ms / block`
  if (ms < 60_000) {
    const sec = ms / 1000
    const digits = sec < 10 ? 2 : 1
    return `${sec.toFixed(digits)} s / block`
  }
  return '1 min / block'
}

export function BlockSpeedControl() {
  const [ms, setMs] = useState(1000)
  const [error, setError] = useState<string | null>(null)
  const debounceRef = useRef<number | null>(null)
  const latestMs = useRef(ms)
  latestMs.current = ms

  useEffect(() => {
    let cancelled = false
    void (async () => {
      try {
        const res = await api.get<ProduceIntervalResponse>('/api/v1/operator/produce-interval')
        if (cancelled) return
        const next = Math.round(res.interval_sec * 1000)
        latestMs.current = next
        setMs(next)
        setError(null)
      } catch {
        if (!cancelled) setError('node offline')
      }
    })()
    return () => {
      cancelled = true
      if (debounceRef.current !== null) window.clearTimeout(debounceRef.current)
    }
  }, [])

  function pushInterval(nextMs: number) {
    latestMs.current = nextMs
    setMs(nextMs)
    if (debounceRef.current !== null) window.clearTimeout(debounceRef.current)
    debounceRef.current = window.setTimeout(() => {
      const sec = latestMs.current / 1000
      void (async () => {
        try {
          const res = await api.post<ProduceIntervalResponse>('/api/v1/operator/produce-interval', {
            interval_sec: sec,
          })
          latestMs.current = Math.round(res.interval_sec * 1000)
          setMs(latestMs.current)
          setError(null)
        } catch {
          setError('set failed')
        }
      })()
    }, 120)
  }

  return (
    <div className={styles.wrap} title="Wall-clock sim speed: block production; traffic follows height">
      <label className={styles.label} htmlFor="block-speed">
        Sim speed
      </label>
      <input
        id="block-speed"
        className={styles.slider}
        type="range"
        min={0}
        max={SLIDER_MAX}
        step={1}
        value={msToSlider(ms)}
        onChange={(e) => pushInterval(sliderToMs(Number(e.target.value)))}
        aria-valuemin={MIN_MS}
        aria-valuemax={MAX_MS}
        aria-valuenow={ms}
        aria-valuetext={formatRate(ms)}
      />
      <span className={styles.value}>
        {error ? <span className={styles.err}>{error}</span> : formatRate(ms)}
      </span>
      <span className={styles.ends} aria-hidden>
        <span>1 ms</span>
        <span>1 min</span>
      </span>
    </div>
  )
}
