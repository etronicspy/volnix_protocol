import { useEffect, useMemo, useState } from 'react'
import { normalizePhrase } from '../../lib/phrase'
import { unlockAccount } from '../../lib/wallet'
import { readActiveSeed, readWallets, setActiveSeed } from '../../lib/walletStore'
import { WalletOps } from './WalletOps'
import { WalletVault, type ActiveWallet } from './WalletVault'

export function WalletsSection() {
  const [active, setActive] = useState<ActiveWallet | null>(() => {
    const seed = readActiveSeed()
    if (!seed) return null
    const stored = readWallets().find((w) => w.seed === seed)
    if (!stored) return { seed, address: '' }
    return { seed: stored.seed, address: stored.address }
  })

  useEffect(() => {
    if (!active?.seed || active.address) return
    let cancelled = false
    unlockAccount(active.seed)
      .then((account) => {
        if (!cancelled) setActive({ seed: active.seed, address: account.address })
      })
      .catch(() => {
        /* keep hint until user picks a row */
      })
    return () => {
      cancelled = true
    }
  }, [active])

  const ready = useMemo(
    () => Boolean(active?.seed && active.address),
    [active],
  )

  function onSelect(wallet: ActiveWallet) {
    setActiveSeed(wallet.seed)
    setActive({ seed: normalizePhrase(wallet.seed), address: wallet.address })
  }

  return (
    <div className="stack">
      <WalletVault activeSeed={active?.seed ?? null} onSelect={onSelect} />
      {ready && active ? (
        <WalletOps key={active.seed} seed={active.seed} address={active.address} />
      ) : (
        <section className="panel">
          <div className="panel-title">Операции</div>
          <p className="muted">Выберите кошелёк в списке выше.</p>
        </section>
      )}
    </div>
  )
}
