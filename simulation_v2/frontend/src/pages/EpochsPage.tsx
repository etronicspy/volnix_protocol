import { formatMicro } from '../lib/api'
import { useFetch } from '../hooks/useFetch'
import { useLive } from '../hooks/useLive'
import { HashLink } from '../components/HashLink'
import { DataTable } from '../components/ui/DataTable'
import { Page } from '../components/ui/Page'
import { Panel } from '../components/ui/Panel'
import type { EpochRecord } from '../types/api'

export function EpochsPage() {
  const { chainTick } = useLive()
  const { data, error, loading } = useFetch<{ epochs: EpochRecord[] }>('/api/v1/epochs', chainTick)
  const epochs = [...(data?.epochs ?? [])].reverse()

  return (
    <Page
      title="Epochs"
      lead="Boundary §5.5: cancel ANT orders → wipe all ANT → ANT_emit = L_total × EpochBlocks → LZN_emit."
      error={error}
      loading={loading && !data}
    >
      <Panel>
        <DataTable
          head={['Epoch', 'Height', 'L_total', 'ANT wiped', 'ANT emit', 'LZN emit', 'Suppliers']}
          empty="No epoch boundaries yet"
          count={epochs.length}
        >
          {epochs.map((e) => (
            <tr key={e.epoch}>
              <td>{e.epoch}</td>
              <td>
                <HashLink height={e.height} />
              </td>
              <td className="mono">{formatMicro(e.l_total)}</td>
              <td className="mono">{formatMicro(e.ant_wiped)}</td>
              <td className="mono">{formatMicro(e.ant_emit)}</td>
              <td className="mono">{e.lzn_emit.toLocaleString()}</td>
              <td>{e.suppliers.length}</td>
            </tr>
          ))}
        </DataTable>
      </Panel>
    </Page>
  )
}
