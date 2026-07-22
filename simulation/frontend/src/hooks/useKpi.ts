/**
 * KPI-снимок и список сценариев через REST (poll).
 *
 * Простой кеш в памяти; без react-query чтобы не тянуть лишнюю зависимость в R&D-стенд.
 */
import { useEffect, useRef, useState } from 'react'

import { API_BASE } from '../config'
import type { KpiSnapshot, ScenarioReport } from '../types'

export interface UseKpiResult {
  kpi: KpiSnapshot | null
  loading: boolean
  error: string | null
  reload: () => Promise<void>
}

export function useKpi(intervalMs = 5000): UseKpiResult {
  const [kpi, setKpi] = useState<KpiSnapshot | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const aborted = useRef(false)

  const reload = async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await fetch(`${API_BASE}/api/analytics/kpi`)
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data = (await res.json()) as KpiSnapshot
      if (!aborted.current) setKpi(data)
    } catch (e) {
      if (!aborted.current) setError(e instanceof Error ? e.message : String(e))
    } finally {
      if (!aborted.current) setLoading(false)
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

  return { kpi, loading, error, reload }
}

export interface UseScenariosResult {
  scenarios: string[]
  loading: boolean
  error: string | null
  reload: () => Promise<void>
  run: (path: string, resetState?: boolean) => Promise<ScenarioReport | null>
  lastReport: ScenarioReport | null
}

export function useScenarios(): UseScenariosResult {
  const [scenarios, setScenarios] = useState<string[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [lastReport, setLastReport] = useState<ScenarioReport | null>(null)

  const reload = async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await fetch(`${API_BASE}/api/scenarios`)
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data = (await res.json()) as { scenarios: string[] }
      setScenarios(data.scenarios)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setLoading(false)
    }
  }

  const run = async (path: string, resetState = true): Promise<ScenarioReport | null> => {
    setLoading(true)
    setError(null)
    try {
      const res = await fetch(`${API_BASE}/api/scenarios/run`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path, reset_state: resetState }),
      })
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data = (await res.json()) as { ok: boolean; report: ScenarioReport }
      setLastReport(data.report)
      return data.report
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
      return null
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void reload()
  }, [])

  return { scenarios, loading, error, reload, run, lastReport }
}
