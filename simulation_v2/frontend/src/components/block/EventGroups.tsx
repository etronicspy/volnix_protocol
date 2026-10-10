import type { ChainEvent } from '../../types/api'
import { EventList } from '../ui/EventList'
import { Panel } from '../ui/Panel'

function groupEvents(events: ChainEvent[]): Record<string, ChainEvent[]> {
  const groups: Record<string, ChainEvent[]> = {}
  for (const e of events) {
    const prefix = e.type.includes('.') ? e.type.split('.')[0] : e.type
    ;(groups[prefix] ??= []).push(e)
  }
  return groups
}

export function EventGroups({ title, events }: { title: string; events: ChainEvent[] }) {
  if (!events.length) return null
  const groups = groupEvents(events)
  return (
    <Panel title={title}>
      <div className="stack">
        {Object.entries(groups).map(([prefix, list]) => (
          <details key={prefix} open>
            <summary className="event-group-sum">
              <span className="badge muted">{prefix}</span>
              <span className="muted"> {list.length}</span>
            </summary>
            <div style={{ marginTop: 8 }}>
              <EventList events={list} />
            </div>
          </details>
        ))}
      </div>
    </Panel>
  )
}
