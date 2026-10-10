import { useState } from 'react'
import type { FormEvent } from 'react'
import { Link, NavLink, Outlet, useNavigate } from 'react-router-dom'
import { MEMPOOL_POLL_MS, TRAFFIC_UI_URL } from '../config'
import { api } from '../lib/api'
import { useChainWebSocket } from '../hooks/useChainWebSocket'
import { useLiveRefresh } from '../hooks/useFetch'
import type { SearchResult } from '../types/api'
import styles from './Layout.module.css'

const NAV = [
  { to: '/', label: 'Overview', end: true },
  { to: '/blocks', label: 'Blocks' },
  { to: '/wallets', label: 'Wallets' },
  { to: '/validators', label: 'Validators' },
  { to: '/market', label: 'Market' },
  { to: '/epochs', label: 'Epochs' },
]

export function Layout() {
  const live = useChainWebSocket()
  const pollTick = useLiveRefresh(MEMPOOL_POLL_MS)
  const chainTick = live.connected ? live.height : pollTick
  const navigate = useNavigate()
  const [q, setQ] = useState('')
  const [searchError, setSearchError] = useState<string | null>(null)

  async function onSearch(e: FormEvent) {
    e.preventDefault()
    const query = q.trim()
    if (!query) return
    setSearchError(null)
    try {
      const res = await api.get<SearchResult>(`/api/v1/search?q=${encodeURIComponent(query)}`)
      if (res.kind === 'block' && res.height !== undefined) {
        navigate(`/blocks/${res.height}`)
      } else if (res.kind === 'tx' && res.hash) {
        navigate(`/txs/${res.hash}`)
      } else if (res.kind === 'account' && res.address) {
        navigate(`/accounts/${res.address}`)
      } else if (/^[0-9a-fA-F]{64}$/.test(query)) {
        navigate(`/txs/${query}`)
      } else {
        setSearchError('Nothing found')
      }
    } catch {
      setSearchError('Search failed')
    }
  }

  return (
    <div className={styles.shell}>
      <header className={styles.top}>
        <div className={styles.topInner}>
          <div className={styles.brandRow}>
            <Link to="/" className={styles.brand}>
              Volnix <span>Explorer</span>
            </Link>
            <div className={styles.brandControls}>
              <a
                className={styles.external}
                href={TRAFFIC_UI_URL}
                target="_blank"
                rel="noreferrer"
                title="Пульт стенда: traffic, time scale, reset chain"
              >
                Traffic ↗
              </a>
              <div className={styles.live}>
                <span className={`${styles.dot} ${live.connected ? styles.on : ''}`} />
                {live.connected ? 'live' : 'offline'}
                {live.chainId ? ` · ${live.chainId}` : ''}
                {` · h${live.height}`}
              </div>
            </div>
          </div>
          <nav className={styles.nav}>
            {NAV.map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
                end={item.end}
                className={({ isActive }) => (isActive ? styles.active : undefined)}
              >
                {item.label}
              </NavLink>
            ))}
          </nav>
          <form onSubmit={onSearch} className={`row ${styles.search}`}>
            <input
              value={q}
              onChange={(e) => setQ(e.target.value)}
              placeholder="Search height, tx hash, address…"
              aria-label="Search"
              className={styles.searchInput}
            />
            <button type="submit" className="btn secondary">
              Search
            </button>
            {searchError ? <span className="error">{searchError}</span> : null}
          </form>
        </div>
      </header>
      <main className={styles.main}>
        <Outlet context={{ chainTick, mempoolTick: pollTick }} />
      </main>
    </div>
  )
}
