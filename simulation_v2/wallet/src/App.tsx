import { useEffect, useState } from 'react'
import { AuthGate } from './AuthGate'
import { readStoredSeed, unlockAccount } from './lib/wallet'
import type { WalletSession } from './lib/wallet'
import { WalletHome } from './WalletHome'

export default function App() {
  const [session, setSession] = useState<WalletSession | null>(null)
  const [booting, setBooting] = useState(true)
  const [bootError, setBootError] = useState<string | null>(null)

  useEffect(() => {
    const stored = readStoredSeed()
    if (!stored) {
      setBooting(false)
      return
    }
    unlockAccount(stored.seed)
      .then((account) => {
        setSession({
          seed: stored.seed,
          address: account.address,
          remembered: stored.remembered,
        })
      })
      .catch((err: unknown) => {
        setBootError(err instanceof Error ? err.message : 'Нода недоступна')
      })
      .finally(() => setBooting(false))
  }, [])

  if (booting) {
    return (
      <div className="page">
        <p className="muted">Открываем кошелёк…</p>
      </div>
    )
  }

  if (!session) {
    return (
      <>
        {bootError ? (
          <div className="page" style={{ paddingBottom: 0 }}>
            <p className="error">{bootError}</p>
          </div>
        ) : null}
        <AuthGate onUnlock={setSession} />
      </>
    )
  }

  return <WalletHome session={session} onLogout={() => setSession(null)} />
}
