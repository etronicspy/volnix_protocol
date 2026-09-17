/**
 * Соревнование λ/K (§5.4): ранжирование declare по w_i = s_i/L_i на высоту.
 * Источник — поле `competition` committed-блока (GET /api/blocks/{height}).
 */
import { useEffect, useMemo, useState } from 'react'

import { useBlockByHeight } from '../hooks/useBlockByHeight'
import {
  competitionStatusLabel,
  corridorFill,
  isGenesisCompetition,
} from '../lib/competition'
import { formatAmount, shortAddress, shortHash } from '../lib/format'
import type { PovbCompetition, PovbCompetitionEntry } from '../types'

interface TapeBlock {
  height: number
  competition?: {
    selected_count?: number
    K?: number
    B_selected?: number
    candidates_count?: number
  }
}

interface CompetitionPanelProps {
  currentHeight: number
  tapeBlocks: TapeBlock[]
}

function statusClass(status: string): string {
  switch (status) {
    case 'selected':
      return 'bg-emerald-900/50 text-emerald-300'
    case 'culled_lambda':
      return 'bg-amber-900/40 text-amber-200'
    case 'culled_k':
      return 'bg-rose-900/40 text-rose-300'
    case 'deferred':
      return 'bg-slate-700/70 text-slate-300'
    default:
      return 'bg-gray-700 text-gray-300'
  }
}

function SummaryCard({
  label,
  value,
  hint,
}: {
  label: string
  value: string
  hint?: string
}) {
  return (
    <div className="bg-gray-900/70 border border-gray-700 rounded p-3" data-tip={hint}>
      <div className="text-[10px] uppercase tracking-wider text-gray-500">{label}</div>
      <div className="font-mono text-sm text-gray-100 mt-1 break-all">{value}</div>
    </div>
  )
}

export function CompetitionPanel({ currentHeight, tapeBlocks }: CompetitionPanelProps) {
  const [followLatest, setFollowLatest] = useState(true)
  const [heightInput, setHeightInput] = useState(String(currentHeight))
  const [selectedHeight, setSelectedHeight] = useState(currentHeight)

  useEffect(() => {
    if (!followLatest) return
    setSelectedHeight(currentHeight)
    setHeightInput(String(currentHeight))
  }, [currentHeight, followLatest])

  const { block, found, loading, error, reload } = useBlockByHeight(selectedHeight)
  const competition: PovbCompetition | undefined = block?.competition

  const tapeWithContest = useMemo(
    () =>
      tapeBlocks
        .slice()
        .reverse()
        .filter((b) => Boolean(b.competition) && ((b.competition?.candidates_count ?? 0) > 0 || b.height === 0)),
    [tapeBlocks],
  )

  const goTo = (h: number) => {
    if (!Number.isFinite(h) || h < 0) return
    setFollowLatest(false)
    setSelectedHeight(Math.floor(h))
    setHeightInput(String(Math.floor(h)))
  }

  const fill = competition ? corridorFill(competition) : 0

  return (
    <div className="bg-gray-800 p-6 rounded-lg border border-gray-700 mb-8">
      <div className="flex flex-wrap justify-between items-start gap-3 mb-4">
        <div>
          <h2 className="text-xl font-bold text-amber-300">Соревнование λ/K</h2>
          <p className="text-xs text-gray-500 mt-1 max-w-2xl leading-snug">
            EndBlocker блока N ранжирует declare по <span className="font-mono text-gray-400">w_i = s_i / L_i</span>
            : сверху коридора выбывают наименьшие веса, затем в набор входят не более K с наибольшим{' '}
            <span className="font-mono text-gray-400">w_i</span>. Снимок пишется в блок как{' '}
            <span className="font-mono text-gray-400">competition</span> и читается с цепи.
          </p>
        </div>
        <label className="text-sm text-gray-300 flex items-center gap-2">
          <input
            type="checkbox"
            checked={followLatest}
            onChange={(e) => setFollowLatest(e.target.checked)}
          />
          следить за головой
        </label>
      </div>

      <div className="flex flex-wrap items-end gap-2 mb-4">
        <div className="flex flex-col gap-1">
          <label className="text-xs text-gray-400">Высота блока</label>
          <input
            type="number"
            min={0}
            step={1}
            value={heightInput}
            onChange={(e) => setHeightInput(e.target.value)}
            className="bg-gray-700 text-white px-3 py-1.5 rounded w-32 border border-gray-600 focus:outline-none focus:border-amber-500"
          />
        </div>
        <button
          type="button"
          onClick={() => goTo(parseInt(heightInput, 10))}
          className="bg-amber-700 hover:bg-amber-600 text-white px-4 py-1.5 rounded text-sm"
        >
          Открыть
        </button>
        <button
          type="button"
          disabled={selectedHeight <= 0}
          onClick={() => goTo(selectedHeight - 1)}
          className="bg-gray-700 hover:bg-gray-600 disabled:opacity-40 text-white px-3 py-1.5 rounded text-sm"
        >
          ← пред.
        </button>
        <button
          type="button"
          disabled={selectedHeight >= currentHeight}
          onClick={() => goTo(selectedHeight + 1)}
          className="bg-gray-700 hover:bg-gray-600 disabled:opacity-40 text-white px-3 py-1.5 rounded text-sm"
        >
          след. →
        </button>
        <button
          type="button"
          onClick={() => void reload()}
          className="text-xs text-blue-400 hover:underline px-2 py-1.5"
        >
          {loading ? '…' : 'обновить'}
        </button>
      </div>

      {tapeWithContest.length > 0 ? (
        <div className="flex gap-2 overflow-x-auto pb-3 mb-4">
          {tapeWithContest.map((b) => (
            <button
              key={b.height}
              type="button"
              onClick={() => goTo(b.height)}
              className={`min-w-[140px] text-left bg-gray-900 p-3 rounded border shrink-0 ${
                b.height === selectedHeight ? 'border-amber-500' : 'border-gray-700 hover:border-gray-500'
              }`}
            >
              <div className="text-amber-300 font-bold">#{b.height}</div>
              <div className="text-[11px] text-gray-400 mt-1">
                {b.competition?.selected_count ?? 0}/{b.competition?.K ?? '—'} в наборе
              </div>
              <div className="text-[11px] font-mono text-gray-500">
                Σb={formatAmount(b.competition?.B_selected)}
              </div>
            </button>
          ))}
        </div>
      ) : null}

      {error ? <p className="text-red-400 text-sm mb-3">{error}</p> : null}
      {loading && !block ? <p className="text-gray-500 text-sm">Загрузка блока…</p> : null}
      {!loading && !found ? (
        <p className="text-gray-500 text-sm">Блок #{selectedHeight} не найден в ledger.</p>
      ) : null}

      {block && competition ? (
        <>
          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3 mb-4">
            <SummaryCard
              label="Высота"
              value={`#${block.height}`}
              hint="Техническая высота committed-блока."
            />
            <SummaryCard
              label="L_total"
              value={formatAmount(competition.L_total, 4)}
              hint="Σ активированного LZN на момент отбора."
            />
            <SummaryCard
              label="Коридор Σb"
              value={`${formatAmount(competition.floor, 4)} … ${formatAmount(competition.cap, 4)}`}
              hint="λ·L_total ≤ Σb_i ≤ (1−λ)·L_total."
            />
            <SummaryCard
              label="Σb выбранных"
              value={formatAmount(competition.B_selected, 4)}
              hint="Сумма b_i прошедших λ/K и исполненных в блоке."
            />
            <SummaryCard
              label="Набор / K"
              value={`${competition.selected_count} / ${competition.K}`}
              hint="Сколько подписантов вошло в ValidatorSet vs потолок K."
            />
            <SummaryCard
              label="Отсев λ / K"
              value={`${competition.culled_lambda_count} / ${competition.culled_k_count}`}
              hint="Сколько declare сняли верх коридора и сколько не вошли в top-K."
            />
          </div>

          <div className="mb-5">
            <div className="flex justify-between text-[11px] text-gray-500 mb-1">
              <span>пол λ·L</span>
              <span>
                λ={competition.lambda.toFixed(4)}
                {isGenesisCompetition(competition) ? ' · genesis (без declare)' : ''}
              </span>
              <span>потолок (1−λ)·L</span>
            </div>
            <div className="h-2 bg-gray-900 rounded overflow-hidden border border-gray-700">
              <div
                className="h-full bg-amber-500/80"
                style={{ width: `${Math.round(fill * 100)}%` }}
              />
            </div>
          </div>

          {isGenesisCompetition(competition) || competition.entries.length === 0 ? (
            <p className="text-gray-500 text-sm">
              На этой высоте нет участников соревнования
              {isGenesisCompetition(competition) ? ' (genesis: ValidatorSet задан без λ/K).' : '.'}
            </p>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead>
                  <tr className="border-b border-gray-700 text-gray-400">
                    <th className="pb-2 font-medium">#</th>
                    <th className="pb-2 font-medium">Адрес</th>
                    <th className="pb-2 font-medium">Статус</th>
                    <th className="pb-2 font-medium">w_i</th>
                    <th className="pb-2 font-medium">b_i</th>
                    <th className="pb-2 font-medium">s_i</th>
                    <th className="pb-2 font-medium">L_i</th>
                    <th className="pb-2 font-medium">tx</th>
                  </tr>
                </thead>
                <tbody>
                  {competition.entries.map((e: PovbCompetitionEntry) => (
                    <tr key={`${e.rank}-${e.tx_hash}`} className="border-b border-gray-700/40">
                      <td className="py-2 font-mono text-gray-300">{e.rank}</td>
                      <td className="py-2 font-mono text-xs text-blue-300" title={e.address}>
                        {shortAddress(e.address, 10, 6)}
                      </td>
                      <td className="py-2">
                        <span className={`px-2 py-0.5 rounded text-[11px] uppercase ${statusClass(e.status)}`}>
                          {competitionStatusLabel(e.status)}
                        </span>
                      </td>
                      <td className="py-2 font-mono text-amber-200">{e.w_i.toFixed(4)}</td>
                      <td className="py-2 font-mono">{formatAmount(e.b, 4)}</td>
                      <td className="py-2 font-mono">{formatAmount(e.s, 4)}</td>
                      <td className="py-2 font-mono text-gray-400">{formatAmount(e.L_i, 4)}</td>
                      <td className="py-2 font-mono text-[11px] text-gray-500" title={e.tx_hash}>
                        {shortHash(e.tx_hash)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      ) : null}
    </div>
  )
}
