/**
 * Этап 4: запуск YAML-сценариев + краткий отчёт.
 * Источник: GET /api/scenarios, POST /api/scenarios/run (core/scenarios.py).
 */
import { useState } from 'react'

import { useScenarios } from '../hooks/useKpi'

export function ScenariosPanel() {
  const { scenarios, loading, error, reload, run, lastReport } = useScenarios()
  const [selected, setSelected] = useState<string>('')
  const [resetState, setResetState] = useState(true)

  const onRun = async () => {
    const target = selected || (scenarios[0] ?? '')
    if (!target) return
    await run(target, resetState)
  }

  return (
    <div className="bg-gray-800 p-4 rounded-lg shadow-md">
      <div className="flex justify-between items-center mb-3">
        <h2 className="text-lg font-semibold text-gray-100">YAML scenarios</h2>
        <button
          type="button"
          className="text-xs text-blue-400 hover:underline"
          onClick={() => void reload()}
        >
          {loading ? '…' : 'reload list'}
        </button>
      </div>
      {error && <div className="text-red-400 text-sm mb-2">{error}</div>}
      <div className="flex flex-col gap-2 mb-3">
        <select
          aria-label="scenario"
          className="bg-gray-900 border border-gray-700 rounded px-2 py-1 text-sm text-gray-100"
          value={selected}
          onChange={(e) => setSelected(e.target.value)}
        >
          <option value="">— выберите сценарий —</option>
          {scenarios.map((s) => (
            <option key={s} value={s}>
              {s}
            </option>
          ))}
        </select>
        <label className="flex items-center gap-2 text-xs text-gray-300">
          <input
            type="checkbox"
            checked={resetState}
            onChange={(e) => setResetState(e.target.checked)}
          />
          reset_state перед запуском
        </label>
        <button
          type="button"
          className="bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white px-3 py-1 rounded text-sm"
          onClick={() => void onRun()}
          disabled={loading || !(selected || scenarios.length)}
        >
          run
        </button>
      </div>
      {lastReport && (
        <div className="border-t border-gray-700 pt-3">
          <h3 className="text-sm font-semibold text-gray-300 mb-1">
            {lastReport.name}{' '}
            <span className={lastReport.passed ? 'text-green-400' : 'text-red-400'}>
              [{lastReport.passed ? 'passed' : 'failed'}]
            </span>
          </h3>
          {lastReport.error && (
            <div className="text-xs text-red-400 mb-1">{lastReport.error}</div>
          )}
          <div className="text-xs text-gray-400 mb-2">
            steps={lastReport.steps_executed}, blocks={lastReport.blocks_produced},{' '}
            duration={lastReport.duration_sec.toFixed(2)}s
          </div>
          <ul className="text-xs space-y-1">
            {lastReport.asserts.map((a, idx) => (
              <li key={idx} className={a.passed ? 'text-green-300' : 'text-red-300'}>
                {a.passed ? '✓' : '✗'} {a.description}{' '}
                {!a.passed && (
                  <span className="text-gray-400">
                    (actual={String(a.actual)}, expected={String(a.expected)})
                  </span>
                )}
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  )
}
