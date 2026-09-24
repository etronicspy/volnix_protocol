import { HashLink } from '../HashLink'
import type { CommitView, ValidatorSetView } from '../../types/api'

interface Props {
  commit: CommitView | null
  signingSet: ValidatorSetView | null
  votedPower: number
  totalPower: number
  round: number
}

function flagBadge(flag: string) {
  if (flag === 'commit') return <span className="badge ok">commit</span>
  if (flag === 'nil') return <span className="badge warn">nil</span>
  return <span className="badge muted">{flag || 'absent'}</span>
}

export function CommitVotesTable({ commit, signingSet, votedPower, totalPower, round }: Props) {
  const threshold = totalPower > 0 ? Math.floor((totalPower * 2) / 3) + 1 : 0
  const quorum = totalPower > 0 && votedPower >= threshold
  const powerByAddr = new Map((signingSet?.validators ?? []).map((v) => [v.address, v.power]))

  if (!commit) {
    return (
      <div className="panel" id="commit">
        <div className="panel-title">CometBFT commit</div>
        <div className="empty">No commit recorded for this height yet (tip after restart may omit it).</div>
      </div>
    )
  }

  return (
    <div className="panel" id="commit">
      <div className="panel-title">CometBFT commit</div>
      <div className="grid-3" style={{ marginBottom: '0.75rem' }}>
        <div className="stat">
          <div className="stat-label">Round</div>
          <div className="stat-value">{round}</div>
        </div>
        <div className="stat">
          <div className="stat-label">Voted / total</div>
          <div className="stat-value sm">
            {votedPower.toLocaleString()} / {totalPower.toLocaleString()}
          </div>
        </div>
        <div className="stat">
          <div className="stat-label">Quorum (≥2/3+1 = {threshold.toLocaleString()})</div>
          <div className="stat-value sm">
            <span className={`badge ${quorum ? 'ok' : 'bad'}`}>{quorum ? 'met' : 'miss'}</span>
          </div>
        </div>
      </div>
      {commit.signatures.length ? (
        <div className="table-wrap">
          <table className="data">
            <thead>
              <tr>
                <th>Validator</th>
                <th>Flag</th>
                <th>Power</th>
                <th>Signature</th>
              </tr>
            </thead>
            <tbody>
              {commit.signatures.map((s) => (
                <tr key={s.validator_address}>
                  <td>
                    <HashLink address={s.validator_address} />
                  </td>
                  <td>{flagBadge(s.block_id_flag)}</td>
                  <td className="mono">{(powerByAddr.get(s.validator_address) ?? 0).toLocaleString()}</td>
                  <td className="mono muted">
                    {s.signature ? `${s.signature.slice(0, 12)}…` : '—'}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <div className="empty">Empty signature list</div>
      )}
    </div>
  )
}
