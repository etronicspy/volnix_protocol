import { EXPLORER_URL } from '../config'
import { shortAddress, shortHash } from '../lib/api'

interface Props {
  hash?: string | null
  address?: string | null
  height?: number | null
  className?: string
}

/** Opens the entity in the read-only explorer (separate app, new tab). */
export function HashLink({ hash, address, height, className }: Props) {
  const cls = `mono ${className ?? ''}`
  if (height !== undefined && height !== null) {
    return (
      <a href={`${EXPLORER_URL}/blocks/${height}`} className={cls} target="_blank" rel="noreferrer">
        #{height}
      </a>
    )
  }
  if (address) {
    return (
      <a
        href={`${EXPLORER_URL}/accounts/${address}`}
        className={cls}
        title={address}
        target="_blank"
        rel="noreferrer"
      >
        {shortAddress(address)}
      </a>
    )
  }
  if (hash) {
    return (
      <a href={`${EXPLORER_URL}/txs/${hash}`} className={cls} title={hash} target="_blank" rel="noreferrer">
        {shortHash(hash)}
      </a>
    )
  }
  return <span className="muted">—</span>
}
