import { HashLink } from '../HashLink'
import type { CommitView, ValidatorSetView } from '../../types/api'
import { Badge } from '../ui/Badge'
import { DataTable } from '../ui/DataTable'
import { Panel } from '../ui/Panel'
import { Stat } from '../ui/Stat'

interface Props {
  commit: CommitView | null
  signingSet: ValidatorSetView | null
  votedPower: number
  totalPower: number
  round: number
}

function flagBadge(flag: string) {
  if (flag === 'commit') return <Badge tone="ok">commit</Badge>
  if (flag === 'nil') return <Badge tone="warn">nil</Badge>
  return <Badge tone="muted">{flag || 'absent'}</Badge>
}

export function CommitVotesTable({ commit, signingSet, votedPower, totalPower, round }: Props) {
  const threshold = totalPower > 0 ? Math.floor((totalPower * 2) / 3) + 1 : 0
  const quorum = totalPower > 0 && votedPower >= threshold
  const powerByAddr = new Map((signingSet?.validators ?? []).map((v) => [v.address, v.power]))

  if (!commit) {
    return (
      <Panel id="commit" title="CometBFT commit">
        <div className="empty">No commit recorded for this height yet (tip after restart may omit it).</div>
      </Panel>
    )
  }

  return (
    <Panel id="commit" title="CometBFT commit">
      <div className="grid-3" style={{ marginBottom: '0.75rem' }}>
        <Stat label="Round">{round}</Stat>
        <Stat label="Voted / total" sm>
          {votedPower.toLocaleString()} / {totalPower.toLocaleString()}
        </Stat>
        <Stat label={`Quorum (≥2/3+1 = ${threshold.toLocaleString()})`} sm>
          <Badge tone={quorum ? 'ok' : 'bad'}>{quorum ? 'met' : 'miss'}</Badge>
        </Stat>
      </div>
      <DataTable head={['Validator', 'Flag', 'Power', 'Signature']} empty="Empty signature list" count={commit.signatures.length}>
        {commit.signatures.map((s) => (
          <tr key={s.validator_address}>
            <td>
              <HashLink address={s.validator_address} />
            </td>
            <td>{flagBadge(s.block_id_flag)}</td>
            <td className="mono">{(powerByAddr.get(s.validator_address) ?? 0).toLocaleString()}</td>
            <td className="mono muted">{s.signature ? `${s.signature.slice(0, 12)}…` : '—'}</td>
          </tr>
        ))}
      </DataTable>
    </Panel>
  )
}
