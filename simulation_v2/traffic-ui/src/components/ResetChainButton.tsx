import { useState } from 'react'
import { api, trafficApi } from '../lib/api'
import styles from './ResetChainButton.module.css'

interface Props {
  onError?: (message: string | null) => void
}

/** Wipe the blockstore to genesis, then drop the traffic bot registry (best effort). */
export function ResetChainButton({ onError }: Props) {
  const [resetting, setResetting] = useState(false)

  async function onResetChain() {
    if (resetting) return
    if (!window.confirm('Reset the chain to genesis? Blocks and accounts will be wiped.')) return
    setResetting(true)
    onError?.(null)
    try {
      await api.post('/api/v1/operator/reset-chain')
      try {
        await trafficApi.post('/reset')
      } catch {
        /* traffic process is optional */
      }
    } catch {
      onError?.('Reset failed')
    } finally {
      setResetting(false)
    }
  }

  return (
    <button type="button" className={styles.reset} disabled={resetting} onClick={() => void onResetChain()}>
      {resetting ? 'Resetting…' : 'Reset chain'}
    </button>
  )
}
