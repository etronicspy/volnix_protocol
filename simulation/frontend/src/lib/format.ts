/** Чистые форматтеры — без React/состояния, чтобы их можно было покрыть vitest. */

/** Округление до N знаков с защитой от NaN/undefined. */
export function toFixedSafe(value: number | null | undefined, digits = 2): string {
  if (value === null || value === undefined || Number.isNaN(value)) return '0.00'
  return Number(value).toFixed(digits)
}

/** Короткий адрес: первые 6 + … + последние 4. */
export function shortAddress(addr: string | null | undefined, head = 6, tail = 4): string {
  if (!addr) return ''
  if (addr.length <= head + tail + 1) return addr
  return `${addr.slice(0, head)}…${addr.slice(-tail)}`
}

/** Короткий tx_hash: первые 8 + … + последние 4. */
export function shortHash(hash: string | null | undefined): string {
  return shortAddress(hash, 8, 4)
}

/** Человеческая длительность для UI (sec / min / hours). */
export function humanDuration(seconds: number): string {
  if (!isFinite(seconds) || seconds < 0) return '–'
  if (seconds < 60) return `${seconds.toFixed(1)}s`
  if (seconds < 3600) return `${(seconds / 60).toFixed(1)}m`
  return `${(seconds / 3600).toFixed(2)}h`
}

/** Цвет статуса канона (используется в UI). */
export type CanonStatus = 'ok' | 'reject' | 'warn' | 'queue' | string
export function canonStatusColor(status: CanonStatus): string {
  switch (status) {
    case 'ok':
      return 'text-green-400'
    case 'reject':
      return 'text-red-400'
    case 'warn':
      return 'text-yellow-300'
    case 'queue':
      return 'text-sky-300'
    default:
      return 'text-zinc-300'
  }
}

/** Безопасный парсер числа из input (e.g. формы кошелька). */
export function parseAmount(value: string | number | null | undefined): number {
  if (typeof value === 'number') return Number.isFinite(value) ? value : 0
  if (!value) return 0
  const n = Number(String(value).replace(',', '.'))
  return Number.isFinite(n) ? n : 0
}

/** Форматирование крупных чисел для UI: использует Intl.NumberFormat. */
export function formatAmount(value: number | null | undefined, digits = 2): string {
  if (value === null || value === undefined || Number.isNaN(value)) return '0'
  const absVal = Math.abs(value)
  if (absVal >= 1_000_000) return `${(value / 1_000_000).toFixed(digits)}M`
  if (absVal >= 1_000) return `${(value / 1_000).toFixed(digits)}K`
  return Number(value).toFixed(digits)
}
