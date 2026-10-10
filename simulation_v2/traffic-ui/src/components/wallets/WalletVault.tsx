import { useCallback, useEffect, useMemo, useState } from 'react'
import type { FormEvent } from 'react'
import { GENESIS_SEED } from '../../config'
import { ApiError, trafficApi } from '../../lib/api'
import { generatePhrase, normalizePhrase } from '../../lib/phrase'
import { unlockAccount } from '../../lib/wallet'
import {
  addWallet,
  readWallets,
  removeWallet,
  type StoredWallet,
} from '../../lib/walletStore'
import type { TrafficWallet, TrafficWalletsResponse } from '../../types/api'
import { HashLink } from '../HashLink'
import styles from './WalletVault.module.css'

export interface ActiveWallet {
  seed: string
  address: string
}

interface WalletVaultProps {
  activeSeed: string | null
  onSelect: (wallet: ActiveWallet) => void
}

interface VaultRow {
  seed: string
  address: string
  label: string
  local: boolean
  traffic: boolean
  role?: string
  kind?: string
  genesis?: boolean
}

function kindLabel(kind: string | undefined): string {
  if (kind === 'citizen') return 'гражданин'
  if (kind === 'enrichment') return 'enrichment'
  if (kind === 'retired') return 'retired'
  return kind ?? ''
}

function roleLabel(role: string | undefined): string {
  if (role === 'supplier') return 'поставщик'
  if (role === 'validator') return 'валидатор'
  if (role === 'citizen') return 'гражданин'
  return role ?? '—'
}

function mergeRows(local: StoredWallet[], traffic: TrafficWallet[]): VaultRow[] {
  const bySeed = new Map<string, VaultRow>()
  for (const w of local) {
    bySeed.set(w.seed, {
      seed: w.seed,
      address: w.address,
      label: w.label,
      local: true,
      traffic: false,
    })
  }
  for (const w of traffic) {
    const seed = normalizePhrase(w.seed)
    if (!seed) continue
    const existing = bySeed.get(seed)
    if (existing) {
      existing.traffic = true
      existing.address = w.address || existing.address
      existing.role = w.role
      existing.kind = w.kind
      existing.genesis = Boolean(w.genesis)
      if (w.genesis) existing.label = existing.label || 'genesis'
    } else {
      bySeed.set(seed, {
        seed,
        address: w.address,
        label: w.genesis ? 'genesis' : kindLabel(w.kind) || 'traffic',
        local: false,
        traffic: true,
        role: w.role,
        kind: w.kind,
        genesis: Boolean(w.genesis),
      })
    }
  }
  return [...bySeed.values()].sort((a, b) => {
    if (Boolean(a.genesis) !== Boolean(b.genesis)) return a.genesis ? -1 : 1
    if (a.local !== b.local) return a.local ? -1 : 1
    return a.address.localeCompare(b.address)
  })
}

export function WalletVault({ activeSeed, onSelect }: WalletVaultProps) {
  const [local, setLocal] = useState<StoredWallet[]>(() => readWallets())
  const [traffic, setTraffic] = useState<TrafficWallet[]>([])
  const [trafficError, setTrafficError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [phrase, setPhrase] = useState('')
  const [label, setLabel] = useState('')
  const [error, setError] = useState<string | null>(null)

  const refreshTraffic = useCallback(async () => {
    try {
      const body = await trafficApi.get<TrafficWalletsResponse>('/wallets')
      setTraffic(body.wallets ?? [])
      setTrafficError(null)
    } catch (err) {
      setTraffic([])
      setTrafficError(
        err instanceof ApiError ? err.message : 'Процесс трафика не запущен',
      )
    }
  }, [])

  useEffect(() => {
    void refreshTraffic()
    const id = window.setInterval(() => void refreshTraffic(), 5000)
    return () => window.clearInterval(id)
  }, [refreshTraffic])

  const rows = useMemo(() => mergeRows(local, traffic), [local, traffic])

  async function persist(seed: string, nextLabel: string) {
    const phraseNorm = normalizePhrase(seed)
    if (!phraseNorm) {
      setError('Введите сид-фразу')
      return
    }
    setBusy(true)
    setError(null)
    try {
      const account = await unlockAccount(phraseNorm)
      const next = addWallet({
        seed: phraseNorm,
        address: account.address,
        label: nextLabel,
      })
      setLocal(next)
      setPhrase('')
      setLabel('')
      onSelect({ seed: phraseNorm, address: account.address })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось добавить кошелёк')
    } finally {
      setBusy(false)
    }
  }

  function onAdd(e: FormEvent) {
    e.preventDefault()
    void persist(phrase, label)
  }

  function onRemove(seed: string) {
    setLocal(removeWallet(seed))
  }

  return (
    <div className={styles.wrap}>
      {trafficError ? (
        <div className={styles.offline} role="status">
          Кошельки ботов недоступны: процесс трафика не отвечает. Локальные кошельки
          работают через ноду. {trafficError !== 'Процесс трафика не запущен' ? (
            <span className="muted">{trafficError}</span>
          ) : null}
        </div>
      ) : null}

      <section className="panel">
        <div className="panel-title">Список кошельков</div>
        <p className={styles.hint}>
          Локальные фразы хранятся в браузере (стенд, открытый текст). Боты приходят
          из traffic <code>GET /wallets</code> и не копируются в localStorage.
        </p>
        <div className="row" style={{ marginBottom: '0.75rem' }}>
          <button
            type="button"
            className="btn secondary"
            disabled={busy}
            onClick={() => void refreshTraffic()}
          >
            Обновить из traffic
          </button>
        </div>
        {rows.length === 0 ? (
          <p className="muted">Нет кошельков. Добавьте фразу или дождитесь ботов.</p>
        ) : (
          <div className="table-wrap">
            <table className="data">
              <thead>
                <tr>
                  <th>Метка</th>
                  <th>Адрес</th>
                  <th>Роль</th>
                  <th>Источник</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                {rows.map((row) => {
                  const active = activeSeed === row.seed
                  return (
                    <tr key={row.seed} className={active ? styles.active : undefined}>
                      <td>{row.label || '—'}</td>
                      <td>
                        {row.address ? <HashLink address={row.address} /> : <span className="muted">—</span>}
                      </td>
                      <td>{roleLabel(row.role)}</td>
                      <td>
                        <span className={styles.badges}>
                          {row.local ? <span className="badge muted">local</span> : null}
                          {row.traffic ? (
                            <span className="badge muted">
                              traffic{row.kind ? `: ${kindLabel(row.kind)}` : ''}
                            </span>
                          ) : null}
                          {row.genesis ? <span className="badge ok">genesis</span> : null}
                          {active ? <span className="badge ok">активен</span> : null}
                        </span>
                      </td>
                      <td>
                        <div className={styles.actions}>
                          <button
                            type="button"
                            className="btn secondary"
                            disabled={active || !row.address}
                            onClick={() => onSelect({ seed: row.seed, address: row.address })}
                          >
                            Выбрать
                          </button>
                          {row.local ? (
                            <button
                              type="button"
                              className="btn secondary"
                              disabled={busy}
                              onClick={() => onRemove(row.seed)}
                            >
                              Удалить
                            </button>
                          ) : null}
                        </div>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <form className="panel" onSubmit={onAdd}>
        <div className="panel-title">Добавить локальный кошелёк</div>
        <p className={styles.hint}>
          Сид — 12 слов стенда или любая строка (нода считает SHA256 текста, не BIP39).
        </p>
        <div className="field">
          <label htmlFor="wallet-label">Метка</label>
          <input
            id="wallet-label"
            value={label}
            onChange={(e) => setLabel(e.target.value)}
            placeholder="необязательно"
          />
        </div>
        <div className="field">
          <label htmlFor="wallet-phrase">Сид-фраза</label>
          <textarea
            id="wallet-phrase"
            className={styles.textarea}
            value={phrase}
            onChange={(e) => setPhrase(e.target.value)}
            autoComplete="off"
            spellCheck={false}
          />
        </div>
        {error ? <p className="error">{error}</p> : null}
        <div className="row">
          <button className="btn" type="submit" disabled={busy || !normalizePhrase(phrase)}>
            {busy ? 'Добавление…' : 'Добавить'}
          </button>
          <button
            type="button"
            className="btn secondary"
            disabled={busy}
            onClick={() => setPhrase(generatePhrase())}
          >
            Сгенерировать
          </button>
          <button
            type="button"
            className="btn secondary"
            disabled={busy}
            onClick={() => void persist(GENESIS_SEED, 'genesis')}
          >
            Добавить genesis
          </button>
        </div>
      </form>
    </div>
  )
}
