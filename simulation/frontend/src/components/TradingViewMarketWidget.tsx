import type { EChartsOption } from 'echarts'
import ReactECharts from 'echarts-for-react'
import { useEffect, useMemo, useState } from 'react'
import { API_BASE } from '../config'
import {
  type PriceTick,
  barsToEchartsPayload,
  ticksToOhlcBars,
} from '../lib/marketEcharts'

/**
 * Виджет в стиле TradingView: свечи через Apache ECharts (echarts-for-react).
 * Данные: проп history + при монте — REST /api/market/history; агрегация совпадает с GET /api/market/bars.
 */

interface MarketHistoryResponse {
  last_price?: number
  history?: PriceTick[]
}

/** seconds === 0: одна свеча на сделку. */
const RESOLUTIONS: { label: string; seconds: number }[] = [
  { label: 'Сделка', seconds: 0 },
  { label: '1s', seconds: 1 },
  { label: '1m', seconds: 60 },
  { label: '5m', seconds: 300 },
  { label: '15m', seconds: 900 },
  { label: '1h', seconds: 3600 },
]

const CHART = {
  backgroundColor: '#131722',
  textColor: '#d1d4dc',
  grid: '#363a45',
  upColor: '#26a69a',
  downColor: '#ef5350',
}

function buildCandlestickOption(
  category: string[],
  values: number[][],
  times: number[],
  tradeMode: boolean,
): EChartsOption {
  const rotate = category.length > 48 ? 40 : 0
  return {
    backgroundColor: CHART.backgroundColor,
    animation: false,
    grid: { left: 52, right: 14, top: 12, bottom: tradeMode ? 80 : 72 },
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'cross', lineStyle: { color: '#787b86', width: 1 } },
      backgroundColor: 'rgba(30,34,45,0.96)',
      borderColor: CHART.grid,
      textStyle: { color: CHART.textColor, fontSize: 11 },
      formatter: (params: unknown) => {
        const arr = params as Array<{
          dataIndex?: number
          data?: number[]
        }>
        const p = arr[0]
        const row = p?.data
        if (!row || row.length < 4) return ''
        const [open, close, low, high] = row
        const idx = p.dataIndex ?? 0
        const t = times[idx]
        const when = Number.isFinite(t)
          ? new Date(t * 1000).toLocaleString(undefined, {
              hour: '2-digit',
              minute: '2-digit',
              second: '2-digit',
            })
          : ''
        return [
          when,
          `O ${open.toFixed(4)}`,
          `H ${high.toFixed(4)}`,
          `L ${low.toFixed(4)}`,
          `C ${close.toFixed(4)}`,
        ].join('<br/>')
      },
    },
    xAxis: {
      type: 'category',
      data: category,
      boundaryGap: true,
      axisLine: { lineStyle: { color: CHART.grid } },
      axisLabel: {
        color: '#787b86',
        fontSize: 10,
        interval: 'auto',
        rotate,
      },
      splitLine: { show: false },
    },
    yAxis: {
      type: 'value',
      scale: true,
      splitLine: { lineStyle: { color: CHART.grid } },
      axisLabel: { color: '#787b86', fontSize: 10 },
    },
    dataZoom: [
      { type: 'inside', xAxisIndex: 0, filterMode: 'weakFilter' },
      {
        type: 'slider',
        xAxisIndex: 0,
        height: 20,
        bottom: 4,
        borderColor: CHART.grid,
        fillerColor: 'rgba(41,98,255,0.12)',
        handleStyle: { color: '#2962FF' },
        textStyle: { color: '#787b86', fontSize: 10 },
      },
    ],
    series: [
      {
        type: 'candlestick',
        name: 'ANT/WRT',
        data: values,
        itemStyle: {
          color: CHART.upColor,
          color0: CHART.downColor,
          borderColor: CHART.upColor,
          borderColor0: CHART.downColor,
        },
      },
    ],
  }
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
  const [intervalSec, setIntervalSec] = useState(0)
  const [fetchedHistory, setFetchedHistory] = useState<PriceTick[]>([])
  const [fetchedLast, setFetchedLast] = useState<number | null>(null)

  useEffect(() => {
    let cancelled = false
    fetch(`${API_BASE}/api/market/history?limit=20000`)
      .then((r) => (r.ok ? r.json() : Promise.reject(new Error(String(r.status)))))
      .then((d: MarketHistoryResponse) => {
        if (cancelled) return
        setFetchedHistory(Array.isArray(d.history) ? d.history : [])
        if (typeof d.last_price === 'number') setFetchedLast(d.last_price)
      })
      .catch(() => {
        if (!cancelled) setFetchedHistory([])
      })
    return () => {
      cancelled = true
    }
  }, [])

  const history = historyProp.length > 0 ? historyProp : fetchedHistory
  const displayLast =
    historyProp.length > 0 ? lastPrice : fetchedLast !== null ? fetchedLast : lastPrice

  const chartOption = useMemo(() => {
    const tradeMode = intervalSec <= 0
    const bars = ticksToOhlcBars(history, intervalSec)
    const { category, times, values } = barsToEchartsPayload(bars, tradeMode)
    if (values.length === 0) return null
    return buildCandlestickOption(category, values, times, tradeMode)
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
            {Number.isFinite(displayLast) ? displayLast.toFixed(2) : '—'}
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
