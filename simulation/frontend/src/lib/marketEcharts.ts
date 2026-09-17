/**
 * Агрегация тиков → OHLC и payload для Apache ECharts candlestick
 * (согласовано с simulation/backend/core/market_bars.py).
 *
 * Ось X всегда category по индексу бара — равные слоты как на TradingView/Binance.
 * Время только в подписи/tooltip; дубликаты «MM-DD HH:MM» больше не схлопывают бары.
 */

export interface PriceTick {
  time: string
  price: number
  ts?: number | string
}

export interface OhlcBar {
  t: number
  open: number
  high: number
  low: number
  close: number
}

const MAX_TRADE_BARS = 8000
const MAX_BUCKET_BARS = 4000

function pad2(n: number): string {
  return n.toString().padStart(2, '0')
}

export function formatBarTime(tSec: number, tradeMode: boolean): string {
  const d = new Date(tSec * 1000)
  if (tradeMode) {
    return `${pad2(d.getHours())}:${pad2(d.getMinutes())}:${pad2(d.getSeconds())}`
  }
  // Секунды в подписи: на 1s/1m соседние бары иначе выглядят как один и тот же «HH:MM».
  return `${pad2(d.getMonth() + 1)}-${pad2(d.getDate())} ${pad2(d.getHours())}:${pad2(d.getMinutes())}:${pad2(d.getSeconds())}`
}

export function normalizeTickTimes(ticks: PriceTick[]): { ts: number; price: number }[] {
  if (ticks.length === 0) return []
  const now = Date.now() / 1000
  const normalized = ticks.map((t, i) => {
    let ts: number
    if (typeof t.ts === 'number' && Number.isFinite(t.ts)) {
      ts = t.ts
    } else if (typeof t.ts === 'string') {
      const n = Number(t.ts)
      ts = Number.isFinite(n) ? n : now - (ticks.length - 1 - i)
    } else {
      ts = now - (ticks.length - 1 - i)
    }
    return { price: t.price, ts }
  })
  normalized.sort((a, b) => a.ts - b.ts)
  return normalized
}

export function ticksToOhlcBars(ticks: PriceTick[], intervalSec: number): OhlcBar[] {
  if (ticks.length === 0) return []
  const normalized = normalizeTickTimes(ticks)

  if (intervalSec <= 0) {
    const slice =
      normalized.length > MAX_TRADE_BARS
        ? normalized.slice(-MAX_TRADE_BARS)
        : normalized
    let lastT = -Infinity
    return slice.map(({ ts, price }) => {
      let t = ts
      if (t <= lastT) {
        t = lastT + 1e-6
      }
      lastT = t
      return { t, open: price, high: price, low: price, close: price }
    })
  }

  const buckets = new Map<
    number,
    { open: number; high: number; low: number; close: number }
  >()

  for (const { ts, price } of normalized) {
    const bucket = Math.floor(ts / intervalSec) * intervalSec
    const existing = buckets.get(bucket)
    if (!existing) {
      buckets.set(bucket, { open: price, high: price, low: price, close: price })
    } else {
      existing.high = Math.max(existing.high, price)
      existing.low = Math.min(existing.low, price)
      existing.close = price
    }
  }

  const ordered = [...buckets.keys()].sort((a, b) => a - b)
  const keys =
    ordered.length > MAX_BUCKET_BARS ? ordered.slice(-MAX_BUCKET_BARS) : ordered

  return keys.map((time) => {
    const ohlc = buckets.get(time)!
    return {
      t: time,
      open: ohlc.open,
      high: ohlc.high,
      low: ohlc.low,
      close: ohlc.close,
    }
  })
}

/**
 * Payload для ECharts: category = уникальные индексы (равные слоты),
 * времена — параллельный массив для подписей и tooltip.
 */
export function barsToEchartsPayload(
  bars: OhlcBar[],
  tradeMode: boolean,
): { category: string[]; times: number[]; labels: string[]; values: number[][] } {
  const category: string[] = []
  const times: number[] = []
  const labels: string[] = []
  const values: number[][] = []
  for (let i = 0; i < bars.length; i++) {
    const bar = bars[i]
    const t = bar.t
    times.push(t)
    // Уникальный ключ категории — индекс. Иначе одинаковые «MM-DD HH:MM»
    // схлопывают соседние 1s-бары в ECharts.
    category.push(String(i))
    labels.push(formatBarTime(t, tradeMode))
    values.push([bar.open, bar.close, bar.low, bar.high])
  }
  return { category, times, labels, values }
}
