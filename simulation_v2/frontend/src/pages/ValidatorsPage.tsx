import { formatMicro } from '../lib/api'
import { useFetch } from '../hooks/useFetch'
import { useLive } from '../hooks/useLive'
import { HashLink } from '../components/HashLink'
import { Badge } from '../components/ui/Badge'
import { DataTable } from '../components/ui/DataTable'
import { Page } from '../components/ui/Page'
import { Panel } from '../components/ui/Panel'
import type { ValidatorView } from '../types/api'

export function ValidatorsPage() {
  const { chainTick } = useLive()
  const { data, error, loading } = useFetch<{
    proposer: string
    total_power: number
    validators: ValidatorView[]
  }>('/api/v1/validators', chainTick)

  return (
    <Page
      title="Validators"
      lead="Current ValidatorSet — weight w_i = s_i / L_i, integer power for proposer priority."
      error={error}
      loading={loading && !data}
    >
      {data ? (
        <Panel
          title={
            <>
              Proposer <HashLink address={data.proposer} /> · total power {data.total_power.toLocaleString()}
            </>
          }
        >
          <DataTable
            head={['Address', 'L_i', 's_i', 'w_i', 'Power', 'ANT', '']}
            empty="Empty set"
            count={data.validators.length}
          >
            {data.validators.map((v) => (
              <tr key={v.address}>
                <td>
                  <HashLink address={v.address} />
                </td>
                <td className="mono">{formatMicro(v.l_i)}</td>
                <td className="mono">{formatMicro(v.s_i)}</td>
                <td className="mono">{v.w_i.toFixed(4)}</td>
                <td className="mono">{v.power.toLocaleString()}</td>
                <td className="mono">{formatMicro(v.ant)}</td>
                <td>{v.is_proposer ? <Badge tone="ok">proposer</Badge> : null}</td>
              </tr>
            ))}
          </DataTable>
        </Panel>
      ) : null}
    </Page>
  )
}
