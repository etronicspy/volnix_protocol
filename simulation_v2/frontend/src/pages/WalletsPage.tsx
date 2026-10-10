import { useMemo, useState } from 'react'
import { useFetch } from '../hooks/useFetch'
import { useLive } from '../hooks/useLive'
import { HashLink } from '../components/HashLink'
import { Badge } from '../components/ui/Badge'
import type { BadgeTone } from '../components/ui/Badge'
import { DataTable } from '../components/ui/DataTable'
import { Page } from '../components/ui/Page'
import { Panel } from '../components/ui/Panel'
import { Stat } from '../components/ui/Stat'
import type { AccountListResponse } from '../types/api'

type RoleFilter = 'all' | 'citizen' | 'supplier' | 'validator'

const ROLE_TONE: Record<string, BadgeTone> = {
  validator: 'ok',
  supplier: 'warn',
  citizen: 'muted',
}

const FILTERS = [
  ['all', 'All'],
  ['validator', 'Validators'],
  ['supplier', 'Suppliers'],
  ['citizen', 'Citizens'],
] as const

export function WalletsPage() {
  const { chainTick } = useLive()
  const [role, setRole] = useState<RoleFilter>('all')
  const { data, error, loading } = useFetch<AccountListResponse>('/api/v1/accounts', chainTick)

  const filtered = useMemo(() => {
    const all = data?.accounts ?? []
    if (role === 'all') return all
    return all.filter((a) => a.role === role)
  }, [data, role])

  const counts = useMemo(() => {
    const next = { total: 0, citizen: 0, supplier: 0, validator: 0 }
    for (const a of data?.accounts ?? []) {
      next.total += 1
      if (a.role === 'citizen') next.citizen += 1
      else if (a.role === 'supplier') next.supplier += 1
      else if (a.role === 'validator') next.validator += 1
    }
    return next
  }, [data])

  return (
    <Page
      title="Wallets"
      lead="All simulation accounts — role/status and WRT / LZN / ANT balances."
      error={error}
      loading={loading && !data}
    >
      {data ? (
        <>
          <div className="grid-3" style={{ marginBottom: '1rem' }}>
            <Stat label="Accounts">{counts.total}</Stat>
            <Stat label="Height">{data.height}</Stat>
            <Stat label="By role" sm>
              V {counts.validator} · S {counts.supplier} · C {counts.citizen}
            </Stat>
          </div>

          <div className="row" style={{ marginBottom: '0.75rem', flexWrap: 'wrap', gap: '0.5rem' }}>
            {FILTERS.map(([value, label]) => (
              <button
                key={value}
                type="button"
                className={`btn ${role === value ? '' : 'secondary'}`}
                onClick={() => setRole(value)}
              >
                {label}
              </button>
            ))}
          </div>

          <Panel title={`${filtered.length} wallet${filtered.length === 1 ? '' : 's'}${role !== 'all' ? ` · ${role}` : ''}`}>
            <DataTable
              head={['Address', 'Role', 'Status', 'WRT', 'ANT', 'LZN', 'LZN act.', 'Last tx']}
              empty="No wallets match this filter"
              count={filtered.length}
            >
              {filtered.map((a) => (
                <tr key={a.address}>
                  <td>
                    <HashLink address={a.address} />
                  </td>
                  <td>
                    <Badge tone={ROLE_TONE[a.role] ?? 'muted'}>{a.role}</Badge>
                  </td>
                  <td>
                    <span className="row" style={{ gap: '0.35rem', flexWrap: 'wrap' }}>
                      {a.genesis_no_zkp ? <Badge tone="muted">genesis</Badge> : null}
                      {a.in_validator_set ? (
                        <Badge tone="ok">{a.is_proposer ? 'proposer' : 'in set'}</Badge>
                      ) : a.role === 'validator' ? (
                        <Badge tone="muted">not in set</Badge>
                      ) : null}
                      {a.open_orders > 0 ? <Badge tone="warn">{a.open_orders} orders</Badge> : null}
                    </span>
                  </td>
                  <td className="mono">{a.wrt_display}</td>
                  <td className="mono">{a.ant_display}</td>
                  <td className="mono">{a.lzn_display}</td>
                  <td className="mono">{a.lzn_activated_display}</td>
                  <td className="mono muted">{a.last_tx_height > 0 ? `h${a.last_tx_height}` : '—'}</td>
                </tr>
              ))}
            </DataTable>
          </Panel>
        </>
      ) : null}
    </Page>
  )
}
