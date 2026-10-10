import type { ReactNode } from 'react'

interface Props {
  label: string
  children: ReactNode
  sm?: boolean
  asWritten?: boolean
}

export function Stat({ label, children, sm, asWritten }: Props) {
  return (
    <div className="stat">
      <div className={asWritten ? 'stat-label as-written' : 'stat-label'}>{label}</div>
      <div className={sm ? 'stat-value sm' : 'stat-value'}>{children}</div>
    </div>
  )
}
