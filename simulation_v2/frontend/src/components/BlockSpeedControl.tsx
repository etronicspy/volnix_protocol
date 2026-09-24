import { useEffect, useRef, useState } from 'react'
import { api } from '../lib/api'
import styles from './BlockSpeedControl.module.css'

const SCALE_MIN = 1
const SCALE_MAX = 3600
const SLIDER_MAX = 1000

interface PaceStatus {
  attempt_window_sec: number
  pace_debt_blocks: number
  missed_budget_sec: number
  base_block_time: number
  time_scale: number
  wall_sleep_sec: number
  produce_interval_sec: number
  auto_produce: boolean
}

function scaleToSlider(scale: number): number {
  const clamped = Math.min(SCALE_MAX, Math.max(SCALE_MIN, scale))
  const t = Math.log(clamped / SCALE_MIN) / Math.log(SCALE_MAX / SCALE_MIN)
  return Math.round(t * SLIDER_MAX)
}

function sliderToScale(pos: number): number {
  const t = Math.min(SLIDER_MAX, Math.max(0, pos)) / SLIDER_MAX
  return Math.round(SCALE_MIN * Math.pow(SCALE_MAX / SCALE_MIN, t))
}

function formatScale(scale: number): string {
  if (scale >= 100) return `${Math.round(scale)}×`
  if (scale >= 10) return `${scale.toFixed(0)}×`
  return `${scale.toFixed(1)}×`
}

function formatWindow(sec: number): string {
  if (sec < 1) return `${Math.round(sec * 1000)} ms`
  if (sec < 60) {
    const digits = sec < 10 ? 1 : 0
    return `${sec.toFixed(digits)} s`
  }
  return '60 s'
}

function formatWall(sec: number): string {
  if (sec < 0.01) return `${Math.round(sec * 1000)} ms`
  if (sec < 1) return `${(sec * 1000).toFixed(0)} ms`
  if (sec < 10) return `${sec.toFixed(2)} s`
  return `${sec.toFixed(1)} s`
}

export function BlockSpeedControl() {
  const [pace, setPace] = useState<PaceStatus | null>(null)
  const [scale, setScale] = useState(60)
  const [error, setError] = useState<string | null>(null)
  const debounceRef = useRef<number | null>(null)
  const latestScale = useRef(scale)
  const loadedRef = useRef(false)
  latestScale.current = scale

  useEffect(() => {
    let cancelled = false
    async function poll() {
      try {
        const res = await api.get<PaceStatus>('/api/v1/operator/pace')
        if (cancelled) return
        setPace(res)
        if (!loadedRef.current) {
          const next = Math.max(SCALE_MIN, Math.min(SCALE_MAX, Number(res.time_scale) || 60))
          latestScale.current = next
          setScale(next)
          loadedRef.current = true
        }
        setError(null)
      } catch {
        if (!cancelled) setError('node offline')
      }
    }
    void poll()
    const id = window.setInterval(() => void poll(), 1000)
    return () => {
      cancelled = true
      window.clearInterval(id)
      if (debounceRef.current !== null) window.clearTimeout(debounceRef.current)
    }
  }, [])

  function pushScale(next: number) {
    latestScale.current = next
    setScale(next)
    if (debounceRef.current !== null) window.clearTimeout(debounceRef.current)
    debounceRef.current = window.setTimeout(() => {
      void (async () => {
        try {
          const res = await api.post<PaceStatus>('/api/v1/operator/time-scale', {
            time_scale: latestScale.current,
          })
          latestScale.current = Number(res.time_scale)
          setScale(latestScale.current)
          setPace(res)
          setError(null)
        } catch {
          setError('set failed')
        }
      })()
    }, 120)
  }

  const windowLabel = pace ? formatWindow(pace.attempt_window_sec) : '—'
  const wallLabel = pace ? formatWall(pace.wall_sleep_sec ?? pace.produce_interval_sec) : '—'
  const debt = pace?.pace_debt_blocks ?? 0
  const title =
    'Stand time_scale: wall_sleep = canon T / scale. Traffic follows wall sleep. Canon T stays 1–60 s.'

  return (
    <div className={styles.wrap} title={title}>
      <label className={styles.label} htmlFor="time-scale">
        Time scale
      </label>
      <input
        id="time-scale"
        className={styles.slider}
        type="range"
        min={0}
        max={SLIDER_MAX}
        step={1}
        value={scaleToSlider(scale)}
        onChange={(e) => pushScale(sliderToScale(Number(e.target.value)))}
        aria-valuemin={SCALE_MIN}
        aria-valuemax={SCALE_MAX}
        aria-valuenow={scale}
        aria-valuetext={formatScale(scale)}
      />
      <span className={styles.value}>
        {error ? (
          <span className={styles.err}>{error}</span>
        ) : (
          <>
            {formatScale(scale)}
            <span className={styles.wall}> · {wallLabel}</span>
          </>
        )}
      </span>
      <span className={styles.meta} aria-hidden>
        canon {windowLabel}
        {debt > 0 ? ` · debt ${debt}` : ''}
        <span className={styles.ends}>
          <span>1×</span>
          <span>3600×</span>
        </span>
      </span>
    </div>
  )
}
