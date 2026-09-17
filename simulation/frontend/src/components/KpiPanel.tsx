/**
 * Этап 4: краткий KPI-снимок (Gini, velocity, burn ratio, accepted block ratio, supply).
 * Источник: GET /api/analytics/kpi (см. simulation/backend/core/analytics.py).
 */
import { useKpi } from '../hooks/useKpi'
import { formatAmount, formatPrice } from '../lib/format'

interface RowProps {
  label: string
  value: string | number
  hint?: string
}

function Row({ label, value, hint }: RowProps) {
  return (
    <div className="flex justify-between items-baseline py-1 border-b border-gray-700 last:border-0">
      <span className="text-gray-300 text-sm">{label}</span>
      <span className="font-mono text-gray-100" title={hint} {...(hint ? { 'data-tip': hint } : {})}>
        {value}
      </span>
    </div>
  )
}

export function KpiPanel() {
  const { kpi, loading, error, reload } = useKpi(5000)

  return (
    <div className="bg-gray-800 p-4 rounded-lg shadow-md">
      <div className="flex justify-between items-center mb-3">
        <h2 className="text-lg font-semibold text-gray-100">KPI snapshot</h2>
        <button
          type="button"
          className="text-xs text-blue-400 hover:underline"
          onClick={() => void reload()}
        >
          {loading ? '…' : 'reload'}
        </button>
      </div>
      {error && <div className="text-red-400 text-sm mb-2">{error}</div>}
      {!kpi && !error && <div className="text-gray-400 text-sm">loading…</div>}
      {kpi && (
        <div className="space-y-3">
          <div>
            <h3 className="text-sm font-semibold text-gray-400 mb-1">Supply</h3>
            <Row label="WRT" value={formatAmount(kpi.supply.wrt)} hint="Суммарный Wert в состоянии (все счета)." />
            <Row label="ANT" value={formatAmount(kpi.supply.ant)} hint="Суммарный Anteil. Эмиссия эпохи — только поставщикам." />
            <Row label="LZN" value={formatAmount(kpi.supply.lzn)} hint="Lizenz: ликвид + активированный." />
          </div>
          <div>
            <h3 className="text-sm font-semibold text-gray-400 mb-1">Gini (0 = equal)</h3>
            <Row label="WRT" value={kpi.gini.wrt.toFixed(4)} hint="0 — равное распределение, 1 — всё у одного адреса." />
            <Row label="ANT" value={kpi.gini.ant.toFixed(4)} hint="Неравенство Anteil среди держателей." />
            <Row label="LZN" value={kpi.gini.lzn.toFixed(4)} hint="Неравенство Lizenz." />
          </div>
          <div>
            <h3 className="text-sm font-semibold text-gray-400 mb-1">Burn / emission (§5.4–5.5)</h3>
            <Row label="Burn (current epoch)" value={formatAmount(kpi.burn.current_epoch)} hint="Σb_i ANT, сожжённых валидаторами в текущей эпохе. Ставка s_i в ruleset v2 не сжигается и сюда не входит." />
            <Row label="Emission estimate" value={formatAmount(kpi.burn.emission_estimate)} hint="Рыночная оценка sold × coeff. Фактическая эмиссия на границе эпохи зажимается в коридор v2: от EpochBlocks×λ×L_total до EpochBlocks×L_total, поэтому при мёртвом рынке она будет выше этой оценки." />
            <Row label="Ratio Σb/Σemission" value={kpi.burn.ratio.toFixed(4)} hint="Σb_i / (sold × coeff). При sold = 0 знаменатель нулевой и показатель равен 0, а не бесконечности." />
          </div>
          <div>
            <h3 className="text-sm font-semibold text-gray-400 mb-1">Прочее</h3>
            <Row label="Accepted block ratio" value={kpi.accepted_block_ratio.toFixed(4)} hint="Высота цепи / (высота + число reject-записей §5.4 и §6.1 в канон-логе). Отклонения — min-burn и пустой ValidatorSet." />
            <Row label="Avg match spread" value={kpi.avg_match_spread.toFixed(4)} hint="Среднее |цена сделки − mid| по match-tx последних 100 блоков; mid берётся из текущей книги." />
            <Row label="Mempool" value={kpi.mempool_size} hint="Tx в ожидании следующего блока." />
            <Row label="Validators (consensus set)" value={kpi.consensus_validator_count} hint="Размер активного ValidatorSet (§6.1)." />
            <Row label="Last price" value={formatPrice(kpi.last_price)} hint="Последняя цена ANT в WRT, 6 знаков как в движке." />
          </div>
        </div>
      )}
    </div>
  )
}
