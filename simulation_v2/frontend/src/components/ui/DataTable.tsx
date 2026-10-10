import type { ReactNode } from 'react'

interface Props {
  head: string[]
  empty: string
  count: number
  children: ReactNode
}

export function DataTable({ head, empty, count, children }: Props) {
  if (count === 0) return <div className="empty">{empty}</div>
  return (
    <div className="table-wrap">
      <table className="data">
        <thead>
          <tr>
            {head.map((h, i) => (
              <th key={`${h}-${i}`}>{h}</th>
            ))}
          </tr>
        </thead>
        <tbody>{children}</tbody>
      </table>
    </div>
  )
}
