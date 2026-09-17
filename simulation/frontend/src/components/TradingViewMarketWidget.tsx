import type { EChartsOption } from 'echarts'
import ReactECharts from 'echarts-for-react'
import { useEffect, useMemo, useState } from 'react'
import { API_BASE } from '../config'
import { formatPrice } from '../lib/format'
import {
  type PriceTick,
  barsToEchartsPayload,
  ticksToOhlcBars,
} from '../lib/marketEcharts'

/**
 * Виджет в стиле TradingView: свечи через Apache ECharts.
 *
 * Ось X — category по индексу (равные слоты). Если баров мало — слева
 * паддинг пустыми слотами, чтобы свечи не растягивались на всю ширину,
 * а стояли справа с нормальной шириной, как на Binance/TradingView.
 */

interface MarketHistoryResponse {
  last_price?: number
  history?: PriceTick[]
}

const RESOLUTIONS: { label: string; seconds: number }[] = [
  { label: 'Сделка', seconds: 0 },
  { label: '1s', seconds: 1 },
  { label: '1m', seconds: 60 },
  { label: '5m', seconds: 300 },
  { label: '15m', seconds: 900 },
  { label: '1h', seconds: 3600 },
]

/** Целевое число слотов на экране — задаёт ширину одной свечи. */
const TARGET_SLOTS = 64

const CHART = {
  backgroundColor: '#131722',
  textColor: '#d1d4dc',
  grid: '#363a45',
  upColor: '#26a69a',
  downColor: '#ef5350',
}

/** ECharts: '-' в candlestick = пустой слот (не рисуем). */
type CandleValue = number[] | string

function buildCandlestickOption(
  category: string[],
  labels: string[],
  candleData: CandleValue[],
  showSecondsInLabel: boolean,
): EChartsOption {
  const n = category.length
  // Если баров больше TARGET_SLOTS — показываем хвост через zoom.
  const startPct = n <= TARGET_SLOTS ? 0 : Math.max(0, 100 - (TARGET_SLOTS / n) * 100)

  return {
    backgroundColor: CHART.backgroundColor,
    animation: false,
    legend: { show: false },
    grid: { left: 64, right: 12, top: 16, bottom: 64 },
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'cross', lineStyle: { color: '#787b86', width: 1 } },
      backgroundColor: 'rgba(30,34,45,0.96)',
      borderColor: CHART.grid,
      textStyle: { color: CHART.textColor, fontSize: 11 },
      formatter: (params: unknown) => {
        const arr = (params as Array<{
          seriesType?: string
          dataIndex?: number
          data?: number[] | number | string
        }>).filter((p) => p.seriesType === 'candlestick')
        const p = arr[0]
        const row = p?.data
        if (!Array.isArray(row) || row.length < 4) return ''
        const [open, close, low, high] = row
        const idx = p.dataIndex ?? 0
        const when = labels[idx] ?? ''
        if (!when) return ''
        return [
          when,
          `O ${formatPrice(open)}`,
          `H ${formatPrice(high)}`,
          `L ${formatPrice(low)}`,
          `C ${formatPrice(close)}`,
        ].join('<br/>')
      },
    },
    xAxis: {
      type: 'category',
      data: category,
      boundaryGap: true,
      axisLine: { onZero: false, lineStyle: { color: CHART.grid } },
      axisTick: { show: false },
      axisLabel: {
        color: '#787b86',
        fontSize: 10,
        hideOverlap: true,
        formatter: (value: string) => {
          const idx = Number(value)
          if (!Number.isFinite(idx) || idx < 0 || idx >= labels.length) return ''
          const label = labels[idx]
          if (!label) return ''
          if (!showSecondsInLabel && /:\\d{2}$/.test(label)) {
            return label.replace(/:\\d{2}$/, '')
          }
          return label
        },
      },
      splitLine: { show: false },
    },
    yAxis: {
      type: 'value',
      scale: true,
      // Паддинг от видимых данных (dataZoom filter пересчитывает extent).
      min: (value: { min: number; max: number }) => {
        const span = Math.max(value.max - value.min, Math.abs(value.max) * 1e-4, 1e-9)
        return value.min - span * 0.15
      },
      max: (value: { min: number; max: number }) => {
        const span = Math.max(value.max - value.min, Math.abs(value.max) * 1e-4, 1e-9)
        return value.max + span * 0.15
      },
      splitNumber: 4,
      splitLine: { lineStyle: { color: CHART.grid } },
      axisLabel: {
        color: '#787b86',
        fontSize: 10,
        formatter: (v: number) => formatPrice(v),
      },
    },
    dataZoom: [
      {
        type: 'inside',
        xAxisIndex: 0,
        filterMode: 'filter',
        start: startPct,
        end: 100,
        zoomOnMouseWheel: true,
        moveOnMouseMove: true,
        throttle: 50,
      },
      {
        type: 'slider',
        xAxisIndex: 0,
        height: 22,
        bottom: 6,
        borderColor: CHART.grid,
        fillerColor: 'rgba(41,98,255,0.12)',
        handleStyle: { color: '#2962FF' },
        textStyle: { color: '#787b86', fontSize: 10 },
        filterMode: 'filter',
        start: startPct,
        end: 100,
        labelFormatter: (_value: number, valueStr: string) => {
          const idx = Number(valueStr)
          if (!Number.isFinite(idx) || idx < 0 || idx >= labels.length) return ''
          return labels[idx] ?? ''
        },
      },
    ],
    series: [
      {
        type: 'line',
        name: 'Close',
        // Только по реальным барам; '-' в pad-слотах → null, линия не рвётся влево.
        data: candleData.map((v) => (Array.isArray(v) ? v[1] : null)),
        showSymbol: false,
        silent: true,
        connectNulls: false,
        z: 1,
        lineStyle: { width: 1.25, color: 'rgba(41, 98, 255, 0.7)' },
      },
      {
        type: 'candlestick',
        name: 'ANT/WRT',
        // '-' = пустой слот паддинга (поддерживается runtime ECharts, в типах нет).
        data: candleData as unknown as number[][],
        z: 2,
        barWidth: '70%',
        barMaxWidth: 14,
        barMinWidth: 4,
        itemStyle: {
          color: CHART.upColor,
          color0: CHART.downColor,
          borderColor: CHART.upColor,
          borderColor0: CHART.downColor,
          borderWidth: 1,
        },
      },
    ],
  }
}

/**
 * Паддинг слева пустыми слотами, чтобы N свечей не растягивались на всю ширину.
 * Итог: как на TradingView — плотный ряд свечей нормальной ширины справа.
 */
export function padBarsToSlots(
  category: string[],
  labels: string[],
  values: number[][],
  targetSlots: number = TARGET_SLOTS,
): { category: string[]; labels: string[]; candleData: CandleValue[] } {
  const n = values.length
  if (n >= targetSlots) {
    return {
      category,
      labels,
      candleData: values,
    }
  }
  const pad = targetSlots - n
  const padCat = Array.from({ length: pad }, (_, i) => `pad-${i}`)
  const padLabels = Array.from({ length: pad }, () => '')
  const padVals: CandleValue[] = Array.from({ length: pad }, () => '-')
  // Реальные бары получают индексы pad..pad+n-1 как category keys.
  const realCat = Array.from({ length: n }, (_, i) => String(pad + i))
  return {
    category: [...padCat, ...realCat],
    labels: [...padLabels, ...labels],
    candleData: [...padVals, ...values],
  }
}

function mergeHistory(a: PriceTick[], b: PriceTick[]): PriceTick[] {
  if (a.length === 0) return b
  if (b.length === 0) return a
  const byTs = new Map<number, PriceTick>()
  for (const t of [...a, ...b]) {
    let ts: number
    if (typeof t.ts === 'number' && Number.isFinite(t.ts)) ts = t.ts
    else if (typeof t.ts === 'string') {
      const n = Number(t.ts)
      ts = Number.isFinite(n) ? n : NaN
    } else ts = NaN
    if (!Number.isFinite(ts)) continue
    byTs.set(ts, t)
  }
  return [...byTs.entries()]
    .sort((x, y) => x[0] - y[0])
    .map(([, t]) => t)
}

export type { PriceTick }

export interface TradingViewMarketWidgetProps {
  history: PriceTick[]
  lastPrice: number
  height?: number
}

export function TradingViewMarketWidget({
  history: historyProp,
  lastPrice,
  height = 360,
}: TradingViewMarketWidgetProps) {
  const [intervalSec, setIntervalSec] = useState(60)
  const [fetchedHistory, setFetchedHistory] = useState<PriceTick[]>([])
  const [fetchedLast, setFetchedLast] = useState<number | null>(null)

  useEffect(() => {
    let cancelled = false
    const load = () => {
      fetch(`${API_BASE}/api/market/history?limit=20000`)
        .then((r) => (r.ok ? r.json() : Promise.reject(new Error(String(r.status)))))
        .then((d: MarketHistoryResponse) => {
          if (cancelled) return
          setFetchedHistory(Array.isArray(d.history) ? d.history : [])
          if (typeof d.last_price === 'number') setFetchedLast(d.last_price)
        })
        .catch(() => {
          /* keep previous */
        })
    }
    load()
    // Периодический догруз: WS-хвост в props может быть короче полной истории.
    const id = window.setInterval(load, 15_000)
    return () => {
      cancelled = true
      window.clearInterval(id)
    }
  }, [])

  const history = useMemo(
    () => mergeHistory(fetchedHistory, historyProp),
    [fetchedHistory, historyProp],
  )
  const displayLast =
    historyProp.length > 0 && Number.isFinite(lastPrice) && lastPrice > 0
      ? lastPrice
      : fetchedLast !== null && fetchedLast > 0
        ? fetchedLast
        : lastPrice

  const chartOption = useMemo(() => {
    const tradeMode = intervalSec <= 0
    const bars = ticksToOhlcBars(history, intervalSec)
    const payload = barsToEchartsPayload(bars, tradeMode)
    if (payload.values.length === 0) return null
    const padded = padBarsToSlots(payload.category, payload.labels, payload.values)
    return buildCandlestickOption(
      padded.category,
      padded.labels,
      padded.candleData,
      tradeMode || intervalSec <= 1,
    )
  }, [history, intervalSec])

  const empty = history.length === 0

  return (
    <div className="tradingview-widget-container w-full min-w-0 overflow-hidden rounded-md border border-[#363a45] shadow-lg">
      <div className="flex flex-wrap items-center justify-between gap-2 px-3 py-2 bg-[#1e222d] border-b border-[#363a45]">
        <div className="flex flex-wrap items-center gap-2 min-w-0">
          <span className="text-[#2962FF] font-bold text-sm shrink-0">VOLNIX</span>
          <span className="text-[#d1d4dc] text-sm font-semibold truncate">ANT / WRT</span>
          <span className="text-[#787b86] text-xs shrink-0">симуляция</span>
        </div>
        <div className="flex items-baseline gap-1 shrink-0">
          <span className="text-[#787b86] text-xs">Last</span>
          <span className="font-mono text-[#d1d4dc] text-base tabular-nums">
            {formatPrice(displayLast)}
          </span>
          <span className="text-[#787b86] text-xs">WRT</span>
        </div>
      </div>

      <div className="flex flex-wrap gap-1 px-2 py-1.5 bg-[#1e222d] border-b border-[#363a45]">
        {RESOLUTIONS.map((r) => (
          <button
            key={r.seconds}
            type="button"
            onClick={() => setIntervalSec(r.seconds)}
            className={`px-2 py-0.5 rounded text-xs font-medium transition-colors ${
              intervalSec === r.seconds
                ? 'bg-[#2962FF] text-white'
                : 'text-[#787b86] hover:text-[#d1d4dc] hover:bg-[#2a2e39]'
            }`}
          >
            {r.label}
          </button>
        ))}
      </div>

      <div className="relative w-full bg-[#131722]" style={{ height }}>
        {!empty && chartOption ? (
          <ReactECharts
            option={chartOption}
            style={{ height: '100%', width: '100%' }}
            opts={{ renderer: 'canvas' }}
            notMerge
            lazyUpdate={false}
          />
        ) : null}
        {empty ? (
          <div className="absolute inset-0 flex items-center justify-center text-[#787b86] text-sm px-4 text-center pointer-events-none">
            Нет тиков — сделка или бот. REST{' '}
            <code className="mx-1 text-[#9CA3AF]">/api/market/history</code>, лента — WS.
          </div>
        ) : null}
      </div>

      <div className="px-2 py-1 bg-[#1e222d] border-t border-[#363a45] text-[10px] text-[#787b86] flex flex-wrap gap-x-3 gap-y-0.5 justify-between">
        <span>
          Volnix Simulation · <code className="text-[#9CA3AF]">/api/market/history</code> ·{' '}
          <code className="text-[#9CA3AF]">/api/market/bars</code> · <code className="text-[#9CA3AF]">/ws</code>
        </span>
        <span>Apache ECharts</span>
      </div>
    </div>
  )
}
