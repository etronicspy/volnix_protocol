import { formatMicro, toNumber } from '../../lib/api'
import { HashLink } from '../HashLink'
import type { PovbTrace } from '../../types/api'
import { Badge } from '../ui/Badge'
import { DataTable } from '../ui/DataTable'
import { Panel } from '../ui/Panel'
import { Stat } from '../ui/Stat'
import { PovbCorridor } from './PovbCorridor'

export function PovbPanel({ povb }: { povb: PovbTrace }) {
  const bMin = toNumber(povb.b_min ?? povb.lower)
  const bMax = toNumber(povb.b_max ?? povb.upper)
  const sumB = toNumber(povb.sum_b)
  const fill = toNumber(povb.fill)
  const lDecl = toNumber(povb.l_decl ?? povb.l_total)
  const declares = povb.declares ?? []

  return (
    <Panel id="povb" title="PoVB (§5.4)">
      <div className="grid-3" style={{ marginBottom: '0.75rem' }}>
        <Stat label="Set updated" sm>
          <Badge tone={povb.set_updated ? 'ok' : 'muted'}>{povb.set_updated ? 'yes' : 'no'}</Badge>
        </Stat>
        <Stat label="Fill / L_decl" sm>
          {formatMicro(fill)} / {formatMicro(lDecl)}
        </Stat>
        <Stat label="Σ b_i" sm>
          {formatMicro(sumB)}
        </Stat>
        <Stat label="α / λ / K" sm asWritten>
          {povb.alpha ?? '—'} / {povb.lambda ?? '—'} / {povb.k ?? '—'}
        </Stat>
        <Stat label="Corridor" sm>
          {formatMicro(bMin)} … {formatMicro(bMax)}
        </Stat>
        <Stat label="Passed" sm>
          {povb.passed?.length ?? 0}/{declares.length}
        </Stat>
      </div>

      <PovbCorridor bMin={bMin} bMax={bMax} sumB={sumB} setUpdated={Boolean(povb.set_updated)} />

      <div style={{ marginTop: declares.length ? '0.85rem' : '0.75rem' }}>
        <DataTable
          head={['Validator', 'L_i', 'f_i', 'b_i', 's_i', 'w_i', 'Status']}
          empty="No declares this height"
          count={declares.length}
        >
          {declares.map((d) => (
            <tr key={d.validator}>
              <td>
                <HashLink address={d.validator} />
              </td>
              <td className="mono">{formatMicro(d.l_i)}</td>
              <td className="mono">{formatMicro(d.f_i)}</td>
              <td className="mono">{formatMicro(d.b_i)}</td>
              <td className="mono">{formatMicro(d.s_i)}</td>
              <td className="mono">{d.w_i.toFixed(4)}</td>
              <td>
                {!d.valid ? (
                  <Badge tone="bad">{d.reason || 'invalid'}</Badge>
                ) : d.passed ? (
                  <Badge tone="ok">passed</Badge>
                ) : (
                  <Badge tone="muted">{d.excluded || 'out'}</Badge>
                )}
              </td>
            </tr>
          ))}
        </DataTable>
      </div>
    </Panel>
  )
}
