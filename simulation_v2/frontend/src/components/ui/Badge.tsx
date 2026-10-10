import type { ReactNode } from 'react'

export type BadgeTone = 'ok' | 'bad' | 'warn' | 'muted'

interface Props {
  tone?: BadgeTone
  children: ReactNode
}

export function Badge({ tone, children }: Props) {
  return <span className={tone ? `badge ${tone}` : 'badge'}>{children}</span>
}
