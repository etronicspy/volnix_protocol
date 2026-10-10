import type { ReactNode } from 'react'

interface Props {
  title?: ReactNode
  id?: string
  children: ReactNode
}

export function Panel({ title, id, children }: Props) {
  return (
    <div className="panel" id={id}>
      {title != null ? <div className="panel-title">{title}</div> : null}
      {children}
    </div>
  )
}
