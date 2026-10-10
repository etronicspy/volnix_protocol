import { HashLink } from '../HashLink'
import type { ValidatorView } from '../../types/api'
import { DataTable } from '../ui/DataTable'

interface Props {
  validators: ValidatorView[]
  empty: string
}

export function ValidatorTable({ validators, empty }: Props) {
  return (
    <DataTable head={['Address', 'L_i', 's_i', 'w_i', 'Power']} empty={empty} count={validators.length}>
      {validators.map((v) => (
        <tr key={v.address}>
          <td>
            <HashLink address={v.address} />
          </td>
          <td className="mono">{v.l_i}</td>
          <td className="mono">{v.s_i}</td>
          <td className="mono">{typeof v.w_i === 'number' ? v.w_i.toFixed(4) : v.w_i}</td>
          <td className="mono">{typeof v.power === 'number' ? v.power.toLocaleString() : v.power}</td>
        </tr>
      ))}
    </DataTable>
  )
}
