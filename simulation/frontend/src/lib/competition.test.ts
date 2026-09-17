import { describe, expect, it } from 'vitest'

import type { PovbCompetition } from '../types'
import {
  competitionStatusLabel,
  corridorFill,
  isGenesisCompetition,
} from './competition'

const base: PovbCompetition = {
  kind: 'povb',
  lambda: 1 / 3,
  K: 150,
  L_total: 200,
  floor: 200 / 3,
  cap: (2 * 200) / 3,
  B_candidates: 160,
  B_selected: 80,
  candidates_count: 2,
  selected_count: 1,
  culled_lambda_count: 1,
  culled_k_count: 0,
  entries: [],
}

describe('competitionStatusLabel', () => {
  it('maps known statuses', () => {
    expect(competitionStatusLabel('selected')).toBe('в наборе')
    expect(competitionStatusLabel('culled_lambda')).toBe('λ-отсев')
    expect(competitionStatusLabel('culled_k')).toBe('вне top-K')
  })
  it('passes unknown through', () => {
    expect(competitionStatusLabel('other')).toBe('other')
  })
})

describe('corridorFill', () => {
  it('puts B at 0 when equal to floor', () => {
    expect(corridorFill({ ...base, B_selected: base.floor })).toBeCloseTo(0)
  })
  it('puts B at 1 when equal to cap', () => {
    expect(corridorFill({ ...base, B_selected: base.cap })).toBeCloseTo(1)
  })
  it('clamps outside the corridor', () => {
    expect(corridorFill({ ...base, B_selected: 0 })).toBe(0)
    expect(corridorFill({ ...base, B_selected: 10_000 })).toBe(1)
  })
})

describe('isGenesisCompetition', () => {
  it('detects genesis kind', () => {
    expect(isGenesisCompetition({ ...base, kind: 'genesis' })).toBe(true)
    expect(isGenesisCompetition(base)).toBe(false)
    expect(isGenesisCompetition(null)).toBe(false)
  })
})
