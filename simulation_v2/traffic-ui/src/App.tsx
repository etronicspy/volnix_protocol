import { useState } from 'react'
import { EXPLORER_URL } from './config'
import { BlockSpeedControl } from './components/BlockSpeedControl'
import { ResetChainButton } from './components/ResetChainButton'
import { TrafficPanel } from './components/TrafficPanel'
import { WalletsSection } from './components/wallets/WalletsSection'
import styles from './App.module.css'

type PageTab = 'traffic' | 'wallets'

export default function App() {
  const [error, setError] = useState<string | null>(null)
  const [tab, setTab] = useState<PageTab>('traffic')

  return (
    <div className={styles.shell}>
      <header className={styles.top}>
        <div className={styles.topInner}>
          <div className={styles.brandRow}>
            <span className={styles.brand}>
              Volnix <span>Traffic</span>
            </span>
            <div className={styles.tabs} role="tablist">
              <button
                type="button"
                role="tab"
                aria-selected={tab === 'traffic'}
                className={tab === 'traffic' ? 'btn' : 'btn secondary'}
                onClick={() => setTab('traffic')}
              >
                Traffic
              </button>
              <button
                type="button"
                role="tab"
                aria-selected={tab === 'wallets'}
                className={tab === 'wallets' ? 'btn' : 'btn secondary'}
                onClick={() => setTab('wallets')}
              >
                Кошельки
              </button>
            </div>
            <div className={styles.brandControls}>
              <BlockSpeedControl />
              <ResetChainButton onError={setError} />
              <a className={styles.explorerLink} href={EXPLORER_URL} target="_blank" rel="noreferrer">
                Открыть Explorer ↗
              </a>
            </div>
          </div>
          {error ? <span className="error">{error}</span> : null}
        </div>
      </header>
      <main className={styles.main}>
        <div className="page">
          {tab === 'traffic' ? (
            <>
              <h1 className="page-title">Traffic</h1>
              <p className="page-lead">
                Боты шлют обычные транзакции в мемпул. Блоки подтверждает узел — симуляция не правит
                цепь снаружи.
              </p>
              <TrafficPanel />
            </>
          ) : (
            <>
              <h1 className="page-title">Кошельки</h1>
              <p className="page-lead">
                Локальные сид-фразы хранятся в этом браузере. Боты трафика видны из реестра процесса
                и не копируются сюда. Операции идут через ноду <code>/api/v1/operator/*</code>.
              </p>
              <WalletsSection />
            </>
          )}
        </div>
      </main>
    </div>
  )
}
