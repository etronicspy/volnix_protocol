import { describe, expect, it } from 'vitest'
import {
  barsToEchartsPayload,
  formatBarTime,
  ticksToOhlcBars,
  type PriceTick,
} from './marketEcharts'

function tick(ts: number, price: number): PriceTick {
  return { time: '', price, ts }
}

describe('ticksToOhlcBars', () => {
  it('aggregates into 1s buckets without filling gaps', () => {
    const ticks = [
      tick(1000.1, 1),
      tick(1000.9, 2),
      tick(1005.0, 3),
    ]
    const bars = ticksToOhlcBars(ticks, 1)
    expect(bars).toHaveLength(2)
    expect(bars[0]).toMatchObject({ t: 1000, open: 1, high: 2, low: 1, close: 2 })
    expect(bars[1]).toMatchObject({ t: 1005, open: 3, close: 3 })
  })

  it('trade mode: one bar per tick', () => {
    const ticks = [tick(1, 10), tick(1, 11), tick(2, 12)]
    const bars = ticksToOhlcBars(ticks, 0)
    expect(bars).toHaveLength(3)
    expect(bars[1].t).toBeGreaterThan(bars[0].t)
  })
})

describe('barsToEchartsPayload', () => {
  it('uses unique category indices so duplicate clock labels cannot collapse bars', () => {
    const bars = ticksToOhlcBars(
      [tick(1_700_000_000, 1), tick(1_700_000_001, 2), tick(1_700_000_002, 3)],
      1,
    )
    const payload = barsToEchartsPayload(bars, false)
    expect(payload.category).toEqual(['0', '1', '2'])
    expect(new Set(payload.category).size).toBe(payload.category.length)
    expect(payload.labels).toHaveLength(3)
    expect(payload.values).toHaveLength(3)
    expect(payload.values[0]).toEqual([1, 1, 1, 1])
  })
})

describe('formatBarTime', () => {
  it('renders local clock parts', () => {
    const s = formatBarTime(1_700_000_000, false)
    expect(s).toMatch(/^\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/)
  })
})

describe('padBarsToSlots (via widget helper)', () => {
  it('pads short series so candles pack to the right', async () => {
    const { padBarsToSlots } = await import('../components/TradingViewMarketWidget')
    const padded = padBarsToSlots(['0', '1'], ['a', 'b'], [[1, 1, 1, 1], [2, 2, 2, 2]], 5)
    expect(padded.category).toHaveLength(5)
    expect(padded.candleData.slice(0, 3)).toEqual(['-', '-', '-'])
    expect(padded.candleData.slice(3)).toEqual([[1, 1, 1, 1], [2, 2, 2, 2]])
    expect(padded.labels.slice(3)).toEqual(['a', 'b'])
  })

  it('does not pad when already long enough', async () => {
    const { padBarsToSlots } = await import('../components/TradingViewMarketWidget')
    const values = Array.from({ length: 5 }, (_, i) => [i, i, i, i])
    const padded = padBarsToSlots(
      values.map((_, i) => String(i)),
      values.map((_, i) => `t${i}`),
      values,
      5,
    )
    expect(padded.category).toHaveLength(5)
    expect(padded.candleData.every((v) => Array.isArray(v))).toBe(true)
  })
})
