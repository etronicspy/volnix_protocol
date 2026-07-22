import { describe, expect, it } from 'vitest'
import {
  canonStatusColor,
  humanDuration,
  parseAmount,
  shortAddress,
  shortHash,
  toFixedSafe,
} from './format'

describe('toFixedSafe', () => {
  it('round to digits', () => {
    expect(toFixedSafe(1.2345, 2)).toBe('1.23')
    expect(toFixedSafe(0)).toBe('0.00')
  })
  it('null/NaN → 0.00', () => {
    expect(toFixedSafe(null)).toBe('0.00')
    expect(toFixedSafe(undefined)).toBe('0.00')
    expect(toFixedSafe(NaN)).toBe('0.00')
  })
})

describe('shortAddress / shortHash', () => {
  it('truncates long strings', () => {
    expect(shortAddress('abcdef1234567890', 4, 4)).toBe('abcd…7890')
    expect(shortHash('deadbeef1234567890abcdef')).toBe('deadbeef…cdef')
  })
  it('passes short strings through', () => {
    expect(shortAddress('abc')).toBe('abc')
    expect(shortAddress(null)).toBe('')
  })
})

describe('humanDuration', () => {
  it('formats seconds/minutes/hours', () => {
    expect(humanDuration(45)).toBe('45.0s')
    expect(humanDuration(120)).toBe('2.0m')
    expect(humanDuration(7200)).toBe('2.00h')
    expect(humanDuration(-1)).toBe('–')
  })
})

describe('canonStatusColor', () => {
  it('known statuses', () => {
    expect(canonStatusColor('ok')).toContain('green')
    expect(canonStatusColor('reject')).toContain('red')
    expect(canonStatusColor('warn')).toContain('yellow')
    expect(canonStatusColor('queue')).toContain('sky')
  })
  it('fallback default', () => {
    expect(canonStatusColor('other')).toContain('zinc')
  })
})

describe('parseAmount', () => {
  it('handles strings, comma decimals, junk', () => {
    expect(parseAmount('1.5')).toBe(1.5)
    expect(parseAmount('2,5')).toBe(2.5)
    expect(parseAmount('')).toBe(0)
    expect(parseAmount(null)).toBe(0)
    expect(parseAmount('abc')).toBe(0)
    expect(parseAmount(7)).toBe(7)
  })
})
