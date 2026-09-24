import { HashLink } from '../HashLink'
import type { ValidatorSetView, ValidatorView } from '../../types/api'

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

function buildDiff(
  signing: ValidatorSetView | null,
  next: ValidatorSetView | null,
): DiffRow[] {
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

function kindBadge(kind: DiffKind) {
  if (kind === 'in') return <span className="badge ok">in</span>
  if (kind === 'out') return <span className="badge bad">out</span>
  if (kind === 'power') return <span className="badge warn">power</span>
  return <span className="badge muted">same</span>
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
          {signing.length ? (
            <div className="table-wrap">
              <table className="data">
                <thead>
                  <tr>
                    <th>Address</th>
                    <th>L_i</th>
                    <th>s_i</th>
                    <th>w_i</th>
                    <th>Power</th>
                  </tr>
                </thead>
                <tbody>
                  {signing.map((v) => (
                    <tr key={v.address}>
                      <td>
                        <HashLink address={v.address} />
                      </td>
                      <td className="mono">{v.l_i}</td>
                      <td className="mono">{v.s_i}</td>
                      <td className="mono">{v.w_i?.toFixed?.(4) ?? v.w_i}</td>
                      <td className="mono">{v.power?.toLocaleString?.() ?? v.power}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <div className="empty">Empty</div>
          )}
        </div>
        <div>
          <div className="panel-title">Next ({next.length})</div>
          {next.length ? (
            <div className="table-wrap">
              <table className="data">
                <thead>
                  <tr>
                    <th>Address</th>
                    <th>L_i</th>
                    <th>s_i</th>
                    <th>w_i</th>
                    <th>Power</th>
                  </tr>
                </thead>
                <tbody>
                  {next.map((v) => (
                    <tr key={v.address}>
                      <td>
                        <HashLink address={v.address} />
                      </td>
                      <td className="mono">{v.l_i}</td>
                      <td className="mono">{v.s_i}</td>
                      <td className="mono">{typeof v.w_i === 'number' ? v.w_i.toFixed(4) : v.w_i}</td>
                      <td className="mono">
                        {typeof v.power === 'number' ? v.power.toLocaleString() : v.power}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <div className="empty">Empty / set held</div>
          )}
        </div>
      </div>

      <div className="panel-title">Diff</div>
      {rows.length ? (
        <div className="table-wrap">
          <table className="data">
            <thead>
              <tr>
                <th>Address</th>
                <th>Change</th>
                <th>Power before</th>
                <th>Power after</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => (
                <tr key={r.address}>
                  <td>
                    <HashLink address={r.address} />
                  </td>
                  <td>{kindBadge(r.kind)}</td>
                  <td className="mono">{r.before?.power?.toLocaleString() ?? '—'}</td>
                  <td className="mono">{r.after?.power?.toLocaleString() ?? '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <div className="empty">No validators</div>
      )}
    </div>
  )
}
