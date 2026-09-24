import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { ApiError, formatMicro, trafficApi } from '../lib/api'
import type { TrafficAgentStatus, TrafficStatus } from '../types/api'
import { HashLink } from './HashLink'
import styles from './TrafficPanel.module.css'

const INTENSITY_MIN = 0
const INTENSITY_MAX = 10
const INTENSITY_DEBOUNCE_MS = 400

function clampIntensity(value: number): number {
  if (!Number.isFinite(value)) return 2
  return Math.min(INTENSITY_MAX, Math.max(INTENSITY_MIN, value))
}

function roleLabel(role: string): string {
  if (role === 'supplier') return 'поставщик'
  if (role === 'validator') return 'валидатор'
  if (role === 'citizen') return 'гражданин'
  return role
}

function sortAgents(agents: TrafficAgentStatus[]): TrafficAgentStatus[] {
  return [...agents].sort((a, b) => {
    if (Boolean(a.genesis) !== Boolean(b.genesis)) return a.genesis ? -1 : 1
    if (a.enter !== b.enter) return a.enter ? -1 : 1
    return (b.expected_wrt ?? 0) - (a.expected_wrt ?? 0)
  })
}

export function TrafficPanel() {
  const [status, setStatus] = useState<TrafficStatus | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [intensity, setIntensity] = useState(2)
  const [dragging, setDragging] = useState(false)
  const debounceRef = useRef<number | null>(null)

  const refresh = useCallback(async () => {
    try {
      const st = await trafficApi.get<TrafficStatus>('/status')
      setStatus(st)
      setError(null)
      if (!dragging) {
        setIntensity(clampIntensity(st.intensity))
      }
    } catch (e) {
      setStatus(null)
      setError(
        e instanceof ApiError
          ? e.message
          : 'Traffic process unreachable',
      )
    }
  }, [dragging])

  useEffect(() => {
    void refresh()
    const id = window.setInterval(() => void refresh(), 3000)
    return () => window.clearInterval(id)
  }, [refresh])

  useEffect(() => {
    return () => {
      if (debounceRef.current !== null) {
        window.clearTimeout(debounceRef.current)
      }
    }
  }, [])

  const online = status !== null
  const controlsDisabled = busy || !online

  async function run(fn: () => Promise<unknown>) {
    setBusy(true)
    try {
      await fn()
      await refresh()
    } catch {
      await refresh()
    } finally {
      setBusy(false)
    }
  }

  function postIntensity(value: number) {
    const next = clampIntensity(value)
    void trafficApi
      .post('/intensity', { intensity: next })
      .then(() => refresh())
      .catch(() => refresh())
  }

  function onIntensityChange(raw: string) {
    const next = clampIntensity(Number(raw))
    setIntensity(next)
    if (debounceRef.current !== null) {
      window.clearTimeout(debounceRef.current)
    }
    debounceRef.current = window.setTimeout(() => {
      debounceRef.current = null
      postIntensity(next)
    }, INTENSITY_DEBOUNCE_MS)
  }

  const params = status?.params
  const pools = status?.pools
  const floor = status?.floor
  const genesis = status?.genesis
  const tick = status?.last_tick
  const enrichment = pools?.enrichment ?? 0
  const citizens = pools?.citizen ?? 0
  const targetBots = params?.target_enrichment_bots ?? 50
  const targetCitizens = params?.target_citizens ?? 109
  const horizon = params?.horizon_blocks ?? status?.horizon_blocks ?? 12
  const peerMin = params?.peer_sample_min ?? 5
  const peerMax = params?.peer_sample_max ?? 10
  const spendTxs = tick?.spend_txs ?? tick?.citizen_txs
  const agents = useMemo(() => sortAgents(status?.agents ?? []), [status?.agents])
  const errors = status?.last_errors ?? []

  return (
    <div>
      {error || !online ? (
        <div className={styles.offline} role="status">
          Процесс трафика не запущен. Запустите <code>simulation_v2/traffic</code> (порт 8002),
          затем нажмите Refresh. Кнопки и слайдер неактивны, пока сервис недоступен.
          {error && error !== 'Traffic process unreachable' ? (
            <span className="muted"> {error}</span>
          ) : null}
        </div>
      ) : null}

      <div className={styles.grid}>
        <section className={styles.card} title="Запуск экономики и текущая высота цепи">
          <div className={styles.title}>Экономика</div>
          <div className={styles.value}>
            {online ? (status.running ? 'Запущено' : 'Пауза') : 'Нет связи'}
          </div>
          <p className={styles.meta}>
            Высота блока:{' '}
            {online && status.height >= 0 ? (
              <HashLink height={status.height} />
            ) : (
              <span className="muted">—</span>
            )}
          </p>
          <p className={styles.hint}>
            Start и Stop — цикл ботов. Force tick — один шаг без ожидания нового блока.
          </p>
          <div className={styles.actions}>
            <button
              type="button"
              className="btn"
              disabled={controlsDisabled}
              onClick={() => run(() => trafficApi.post('/start', { intensity }))}
            >
              Start
            </button>
            <button
              type="button"
              className="btn secondary"
              disabled={controlsDisabled}
              onClick={() => run(() => trafficApi.post('/stop'))}
            >
              Stop
            </button>
            <button
              type="button"
              className="btn secondary"
              disabled={controlsDisabled}
              onClick={() => run(() => trafficApi.post('/tick'))}
            >
              Force tick
            </button>
            <button type="button" className="btn secondary" disabled={busy} onClick={() => void refresh()}>
              Refresh
            </button>
          </div>
        </section>

        <section
          className={styles.card}
          title="До 50 независимых стратегий: поставщики и валидаторы"
        >
          <div className={styles.title}>Стратегические боты</div>
          <div className={styles.value}>
            {enrichment} / {targetBots}
          </div>
          <p className={styles.meta}>
            Поставщики {floor?.suppliers ?? 0}
            {floor ? ` (пол ${floor.min_suppliers})` : ''} · валидаторы {floor?.validators ?? 0}
            {floor ? ` (пол ${floor.min_validators})` : ''}
          </p>
          <p className={styles.hint}>
            Пул enrichment из настроек (до {targetBots}). Пол: хотя бы по одному поставщику и
            валидатору.
          </p>
        </section>

        <section
          className={styles.card}
          title="Все кошельки тратят WRT обычными MsgSend — имитация бытовых трат"
        >
          <div className={styles.title}>Спонтанный WRT</div>
          <div className={styles.value}>
            {spendTxs === undefined
              ? '—'
              : `${spendTxs} ${spendTxs === 1 ? 'трата' : 'трат'}`}
          </div>
          <p className={styles.meta}>
            Все кошельки: граждане {citizens}/{targetCitizens} · боты {enrichment}/{targetBots}
          </p>
          <p className={styles.hint}>
            Имитация траты денег: любой кошелёк со свободным WRT может послать перевод. Intensity
            задаёт, какая доля тратит за блок. Подтверждает узел.
          </p>
        </section>

        <section className={styles.card} title="Genesis-кошелёк сразу ведёт enrichment-бот">
          <div className={styles.title}>Genesis</div>
          <div className={styles.value}>
            {genesis?.managed ? 'Стартовый валидатор под ботом' : 'Бот ещё не принял genesis'}
          </div>
          <p className={styles.meta}>
            {genesis?.address ? <HashLink address={genesis.address} /> : <span className="muted">—</span>}
          </p>
          <p className={styles.hint}>Seed genesis закреплён за первым enrichment-агентом.</p>
        </section>

        <section
          className={styles.card}
          title="Горизонт прогноза WRT и случайный осмотр чужих стратегий"
        >
          <div className={styles.title}>Как думают боты</div>
          <div className={styles.value}>Горизонт {horizon} блоков</div>
          <p className={styles.meta}>
            Осмотр {peerMin}–{peerMax} чужих стратегий за шаг
          </p>
          <p className={styles.hint}>
            Каждый агент считает свой P&amp;L вперёд и может скопировать более удачную карточку.
          </p>
        </section>

        <section
          className={styles.card}
          title="Темп экономики. Доля кошельков, которые тратят WRT за блок"
        >
          <div className={styles.title}>Intensity</div>
          <p className={styles.hint}>
            Темп экономики: какая доля всех кошельков тратит WRT за блок (0 — никто, 10 — все со
            свободным балансом). Значение из <code>/intensity</code>.
          </p>
          <div className={styles.intensityRow}>
            <input
              id="traffic-intensity"
              className={styles.slider}
              type="range"
              min={INTENSITY_MIN}
              max={INTENSITY_MAX}
              step={0.5}
              value={intensity}
              disabled={controlsDisabled}
              aria-label="Intensity"
              onPointerDown={() => setDragging(true)}
              onPointerUp={() => setDragging(false)}
              onChange={(e) => onIntensityChange(e.target.value)}
            />
            <span className={styles.intensityValue}>{intensity.toFixed(1)}</span>
          </div>
          <div className={styles.ends}>
            <span>0</span>
            <span>10</span>
          </div>
        </section>

        <section className={`${styles.card} ${styles.cardWide}`}>
          <div className={styles.title}>Таблица агентов</div>
          <p className={styles.hint}>
            Ожидаемый WRT — прогноз агента. Ставка / сжигание — последние <code>s_i</code> /{' '}
            <code>b_i</code>. Кошелёк открывается в проводнике.
          </p>
          {agents.length === 0 ? (
            <p className="muted">Агентов пока нет — запустите трафик и дождитесь тика.</p>
          ) : (
            <div className="table-wrap">
              <table className="data">
                <thead>
                  <tr>
                    <th>Кошелёк</th>
                    <th>Роль</th>
                    <th>Ожидаемый WRT</th>
                    <th>Ставка / сжигание</th>
                    <th>Набор</th>
                    <th>Скопировал у</th>
                  </tr>
                </thead>
                <tbody>
                  {agents.map((a) => (
                    <tr key={a.address}>
                      <td>
                        <span className={styles.badgeRow}>
                          <HashLink address={a.address} />
                          {a.genesis ? <span className="badge muted">genesis</span> : null}
                        </span>
                      </td>
                      <td>{roleLabel(a.role)}</td>
                      <td className="mono">{formatMicro(a.expected_wrt)}</td>
                      <td className="mono">
                        {formatMicro(a.s_i)} / {formatMicro(a.b_i)}
                      </td>
                      <td>
                        {a.enter ? (
                          <span className="badge ok">вошёл в набор</span>
                        ) : (
                          <span className="badge muted">вне набора</span>
                        )}
                      </td>
                      <td>
                        {a.adopted_from ? <HashLink address={a.adopted_from} /> : <span className="muted">—</span>}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>

        {errors.length > 0 ? (
          <section className={`${styles.card} ${styles.cardWide}`}>
            <details className={styles.errors}>
              <summary>Ошибки последнего тика ({errors.length})</summary>
              <ul className={styles.errorList}>
                {errors.slice(-8).map((line) => (
                  <li key={line}>{line}</li>
                ))}
              </ul>
            </details>
          </section>
        ) : null}
      </div>
    </div>
  )
}
