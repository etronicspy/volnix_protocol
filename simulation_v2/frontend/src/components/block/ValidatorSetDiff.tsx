import { HashLink } from '../HashLink'
import type { ValidatorSetView, ValidatorView } from '../../types/api'
import { Badge } from '../ui/Badge'
import { DataTable } from '../ui/DataTable'
import { ValidatorTable } from './ValidatorTable'

interface Props {
  signingSet: ValidatorSetView | null
  nextSet: ValidatorSetView | null
  updates: ValidatorView[]
}

type DiffKind = 'same' | 'in' | 'out' | 'power'

interface DiffRow {
  address: string
  kind: DiffKind
  before: ValidatorView | null
  after: ValidatorView | null
}

function buildDiff(signing: ValidatorSetView | null, next: ValidatorSetView | null): DiffRow[] {
  const before = new Map((signing?.validators ?? []).map((v) => [v.address, v]))
  const after = new Map((next?.validators ?? []).map((v) => [v.address, v]))
  const addrs = new Set([...before.keys(), ...after.keys()])
  const rows: DiffRow[] = []
  for (const address of [...addrs].sort()) {
    const b = before.get(address) ?? null
    const a = after.get(address) ?? null
    let kind: DiffKind = 'same'
    if (b && !a) kind = 'out'
    else if (!b && a) kind = 'in'
    else if (b && a && b.power !== a.power) kind = 'power'
    rows.push({ address, kind, before: b, after: a })
  }
  return rows
}

function kindTone(kind: DiffKind) {
  if (kind === 'in') return 'ok' as const
  if (kind === 'out') return 'bad' as const
  if (kind === 'power') return 'warn' as const
  return 'muted' as const
}

export function ValidatorSetDiff({ signingSet, nextSet, updates }: Props) {
  const rows = buildDiff(signingSet, nextSet)
  const signing = signingSet?.validators ?? []
  const next = nextSet?.validators ?? updates

  return (
    <div className="panel" id="valset">
      <div className="panel-title">ValidatorSet</div>
      <p className="muted" style={{ marginBottom: '0.75rem' }}>
        Signing set for this height vs next set after EndBlocker (for H+1). Proposer now:{' '}
        {signingSet?.proposer ? <HashLink address={signingSet.proposer} /> : '—'} → next:{' '}
        {nextSet?.proposer ? <HashLink address={nextSet.proposer} /> : '—'}
      </p>

      <div className="grid-2" style={{ marginBottom: '1rem' }}>
        <div>
          <div className="panel-title">Signing ({signing.length})</div>
          <ValidatorTable validators={signing} empty="Empty" />
        </div>
        <div>
          <div className="panel-title">Next ({next.length})</div>
          <ValidatorTable validators={next} empty="Empty / set held" />
        </div>
      </div>

      <div className="panel-title">Diff</div>
      <DataTable head={['Address', 'Change', 'Power before', 'Power after']} empty="No validators" count={rows.length}>
        {rows.map((r) => (
          <tr key={r.address}>
            <td>
              <HashLink address={r.address} />
            </td>
            <td>
              <Badge tone={kindTone(r.kind)}>{r.kind}</Badge>
            </td>
            <td className="mono">{r.before?.power?.toLocaleString() ?? '—'}</td>
            <td className="mono">{r.after?.power?.toLocaleString() ?? '—'}</td>
          </tr>
        ))}
      </DataTable>
    </div>
  )
}
