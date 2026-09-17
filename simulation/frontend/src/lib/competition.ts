import type { PovbCompetition, PovbCompetitionStatus } from '../types'

export const COMPETITION_STATUS_LABEL: Record<PovbCompetitionStatus, string> = {
  selected: 'в наборе',
  culled_lambda: 'λ-отсев',
  culled_k: 'вне top-K',
  deferred: 'отложено',
}

export function competitionStatusLabel(status: string): string {
  if (status in COMPETITION_STATUS_LABEL) {
    return COMPETITION_STATUS_LABEL[status as PovbCompetitionStatus]
  }
  return status
}

/** Доля коридора [floor, cap], куда попало Σb выбранных. 0 = пол, 1 = потолок. */
export function corridorFill(comp: PovbCompetition): number {
  const span = comp.cap - comp.floor
  if (!Number.isFinite(span) || span <= 0) return 0
  const t = (comp.B_selected - comp.floor) / span
  if (!Number.isFinite(t)) return 0
  return Math.min(1, Math.max(0, t))
}

export function isGenesisCompetition(comp: PovbCompetition | null | undefined): boolean {
  return comp?.kind === 'genesis'
}
