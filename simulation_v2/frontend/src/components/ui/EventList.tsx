import type { ChainEvent } from '../../types/api'

interface Props {
  events: ChainEvent[]
}

export function EventList({ events }: Props) {
  return (
    <div className="stack">
      {events.map((e, i) => (
        <div key={`${e.type}-${i}`}>
          <span className="badge muted">{e.type}</span>
          <div className="mono muted" style={{ marginTop: 4, fontSize: '0.8rem' }}>
            {e.attributes.map((a) => `${a.key}=${a.value}`).join(' · ')}
          </div>
        </div>
      ))}
    </div>
  )
}
