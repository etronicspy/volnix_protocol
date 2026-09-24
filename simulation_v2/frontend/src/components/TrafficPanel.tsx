import { useEffect, useState } from 'react'
import { ApiError, trafficApi } from '../lib/api'
import type { TrafficStatus } from '../types/api'

interface TrafficPanelProps {
  onLog: (label: string, payload: unknown) => void
}

function formatPoll(sec: number): string {
  if (sec < 1) return `${Math.round(sec * 1000)} ms`
  const digits = sec < 10 ? 2 : 1
  return `${sec.toFixed(digits)} s`
}

export function TrafficPanel({ onLog }: TrafficPanelProps) {
  const [status, setStatus] = useState<TrafficStatus | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [intensity, setIntensity] = useState('2')

  async function refresh() {
    try {
      const st = await trafficApi.get<TrafficStatus>('/status')
      setStatus(st)
      setIntensity(String(st.intensity))
      setError(null)
    } catch (e) {
      setStatus(null)
      setError(
        e instanceof ApiError
          ? e.message
          : 'Traffic process unreachable — start simulation_v2/traffic (port 8002)',
      )
    }
  }

  useEffect(() => {
    void refresh()
    const id = window.setInterval(() => void refresh(), 3000)
    return () => window.clearInterval(id)
  }, [])

  async function run(label: string, fn: () => Promise<unknown>) {
    setBusy(true)
    try {
      const res = await fn()
      onLog(label, res)
      await refresh()
    } catch (e) {
      onLog(label, e instanceof ApiError ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }

  const roles = status?.roles
  const tick = status?.last_tick

  return (
    <div className="panel" style={{ gridColumn: '1 / -1' }}>
      <div className="panel-title">Traffic / economy engine</div>
      <p className="muted" style={{ fontSize: '0.85rem', marginBottom: '0.75rem' }}>
        Separate process (<code>simulation_v2/traffic</code>) — market makers, bot wallets, stub ZKP,
        declares. Default control <code>:8002</code>.
      </p>

      {error ? (
        <p className="error" style={{ marginBottom: '0.75rem' }}>
          {error}
        </p>
      ) : null}

      {status ? (
        <div className="row" style={{ flexWrap: 'wrap', gap: '1rem', marginBottom: '0.75rem' }}>
          <span>
            <strong>{status.running ? 'running' : 'stopped'}</strong>
            {` · h${status.height}`}
          </span>
          <span>
            wallets <strong>{status.wallets}</strong>
            {roles
              ? ` (c${roles.citizen ?? 0} / s${roles.supplier ?? 0} / v${roles.validator ?? 0})`
              : ''}
          </span>
          <span>intensity {status.intensity}</span>
          {typeof status.effective_poll_sec === 'number' ? (
            <span className="muted" title="Poll adapts to node produce_interval so traffic keeps up with Sim speed">
              poll {formatPoll(status.effective_poll_sec)}
              {typeof status.produce_interval_sec === 'number'
                ? ` · synced to chain ${formatPoll(status.produce_interval_sec)}/block`
                : ' · synced to chain'}
            </span>
          ) : null}
          {tick ? (
            <span className="muted">
              last tick: market {tick.market ?? 0} · bots {tick.bots ?? 0} · declare {tick.declare ?? 0}
            </span>
          ) : null}
        </div>
      ) : null}

      <div className="field" style={{ maxWidth: 160 }}>
        <label htmlFor="intensity">Intensity</label>
        <input
          id="intensity"
          value={intensity}
          onChange={(e) => setIntensity(e.target.value)}
          disabled={busy || !status}
        />
      </div>

      <div className="row" style={{ gap: '0.5rem', flexWrap: 'wrap' }}>
        <button
          type="button"
          className="btn"
          disabled={busy || !status}
          onClick={() =>
            run('traffic start', () =>
              trafficApi.post('/start', { intensity: Number(intensity) || 2 }),
            )
          }
        >
          Start
        </button>
        <button
          type="button"
          className="btn secondary"
          disabled={busy || !status}
          onClick={() => run('traffic stop', () => trafficApi.post('/stop'))}
        >
          Stop
        </button>
        <button
          type="button"
          className="btn secondary"
          disabled={busy || !status}
          onClick={() =>
            run('traffic intensity', () =>
              trafficApi.post('/intensity', { intensity: Number(intensity) || 1 }),
            )
          }
        >
          Set intensity
        </button>
        <button
          type="button"
          className="btn secondary"
          disabled={busy || !status}
          onClick={() => run('traffic tick', () => trafficApi.post('/tick'))}
        >
          Force tick
        </button>
        <button
          type="button"
          className="btn secondary"
          disabled={busy}
          onClick={() => void refresh()}
        >
          Refresh
        </button>
      </div>

      {status?.last_errors && status.last_errors.length > 0 ? (
        <pre className="pre" style={{ marginTop: '0.75rem', maxHeight: 120 }}>
          {status.last_errors.slice(-5).join('\n')}
        </pre>
      ) : null}
    </div>
  )
}
