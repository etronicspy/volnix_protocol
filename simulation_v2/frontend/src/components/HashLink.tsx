import { Link } from 'react-router-dom'
import { shortAddress, shortHash } from '../lib/api'

interface Props {
  hash?: string | null
  address?: string | null
  height?: number | null
  className?: string
}

export function HashLink({ hash, address, height, className }: Props) {
  const cls = `mono ${className ?? ''}`
  if (height !== undefined && height !== null) {
    return (
      <Link to={`/blocks/${height}`} className={cls}>
        #{height}
      </Link>
    )
  }
  if (address) {
    return (
      <Link to={`/accounts/${address}`} className={cls} title={address}>
        {shortAddress(address)}
      </Link>
    )
  }
  if (hash) {
    return (
      <Link to={`/txs/${hash}`} className={cls} title={hash}>
        {shortHash(hash)}
      </Link>
    )
  }
  return <span className="muted">—</span>
}
