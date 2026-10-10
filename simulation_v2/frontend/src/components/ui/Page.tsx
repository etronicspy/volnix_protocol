import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'

interface Props {
  title: string
  lead?: ReactNode
  back?: { to: string; label: string }
  afterBack?: ReactNode
  error?: string | null
  loading?: boolean
  wide?: boolean
  children: ReactNode
}

export function Page({ title, lead, back, afterBack, error, loading, wide, children }: Props) {
  return (
    <div className={wide ? 'page page-wide' : 'page'}>
      {back ? (
        <p className="muted">
          <Link to={back.to}>← {back.label}</Link>
          {afterBack}
        </p>
      ) : null}
      <h1 className="page-title">{title}</h1>
      {lead ? <p className="page-lead">{lead}</p> : null}
      {error ? <p className="error">{error}</p> : null}
      {loading ? <p className="muted">Loading…</p> : null}
      {children}
    </div>
  )
}
