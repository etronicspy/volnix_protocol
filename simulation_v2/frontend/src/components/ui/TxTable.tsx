import { HashLink } from '../HashLink'
import type { PendingTx } from '../../lib/pending'
import type { TxListItem } from '../../types/api'
import { DataTable } from './DataTable'

interface Props {
  pending: PendingTx[]
  txs: TxListItem[]
  extraCol?: 'sender' | 'index'
  empty: string
  rejects?: string[]
}

export function TxTable({ pending, txs, extraCol = 'sender', empty, rejects }: Props) {
  const extraHead = extraCol === 'index' ? 'Index' : 'Sender'
  return (
    <>
      {rejects?.length ? <p className="error">{rejects.join(' · ')}</p> : null}
      <DataTable head={['Hash', 'Height', extraHead]} empty={empty} count={pending.length + txs.length}>
        {pending.map((t) => (
          <tr key={t.hash}>
            <td>
              <HashLink hash={t.hash} />
            </td>
            <td>
              <span className="badge warn">in mempool</span>
              <div className="muted" style={{ marginTop: 4, fontSize: '0.75rem' }}>
                {t.label}
              </div>
            </td>
            <td>
              {extraCol === 'index' ? '—' : <HashLink address={t.sender} />}
            </td>
          </tr>
        ))}
        {txs.map((t) => (
          <tr key={t.hash}>
            <td>
              <HashLink hash={t.hash} />
            </td>
            <td>
              <HashLink height={t.height} />
            </td>
            <td>{extraCol === 'index' ? t.index : <HashLink address={t.sender} />}</td>
          </tr>
        ))}
      </DataTable>
    </>
  )
}
