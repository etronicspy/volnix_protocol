import { useMemo, useState } from 'react'
import type { FormEvent } from 'react'
import { GENESIS_SEED } from './config'
import { generatePhrase, normalizePhrase, phrasesMatch } from './lib/phrase'
import { persistSeed, unlockAccount } from './lib/wallet'
import type { WalletSession } from './lib/wallet'

interface AuthGateProps {
  onUnlock: (session: WalletSession) => void
}

export function AuthGate({ onUnlock }: AuthGateProps) {
  const [mode, setMode] = useState<'create' | 'login'>('create')
  const [created, setCreated] = useState(() => generatePhrase())
  const [confirm, setConfirm] = useState('')
  const [loginPhrase, setLoginPhrase] = useState('')
  const [remember, setRemember] = useState(false)
  const [pending, setPending] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const confirmed = useMemo(() => phrasesMatch(created, confirm), [created, confirm])

  async function enter(seed: string) {
    const phrase = normalizePhrase(seed)
    if (!phrase) {
      setError('Введите сид-фразу')
      return
    }
    setPending(true)
    setError(null)
    try {
      const account = await unlockAccount(phrase)
      persistSeed(phrase, remember)
      onUnlock({ seed: phrase, address: account.address, remembered: remember })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не удалось войти')
    } finally {
      setPending(false)
    }
  }

  function onCreate(e: FormEvent) {
    e.preventDefault()
    if (!confirmed) return
    void enter(created)
  }

  function onLogin(e: FormEvent) {
    e.preventDefault()
    void enter(loginPhrase)
  }

  return (
    <div className="page">
      <div className="brand">
        Volnix <span>Wallet</span>
      </div>
      <p className="page-lead">
        Сид-фраза из 12 слов — это ключ стенда. Нода считает адрес как SHA256 текста, не по BIP39.
        Та же фраза всегда открывает тот же адрес.
      </p>
      <div className="tabs">
        <button
          type="button"
          className={mode === 'create' ? 'btn' : 'btn secondary'}
          onClick={() => setMode('create')}
        >
          Создать
        </button>
        <button
          type="button"
          className={mode === 'login' ? 'btn' : 'btn secondary'}
          onClick={() => setMode('login')}
        >
          Войти
        </button>
      </div>
      <div className="row" style={{ marginBottom: '1rem' }}>
        <button
          type="button"
          className="btn secondary"
          disabled={pending}
          onClick={() => void enter(GENESIS_SEED)}
        >
          {pending ? 'Вход…' : 'Войти в genesis'}
        </button>
      </div>

      {mode === 'create' ? (
        <form className="panel" onSubmit={onCreate}>
          <div className="panel-title">Новая фраза</div>
          <p>Запишите слова. Кнопка «Я записал» включается после повторного ввода.</p>
          <div className="words">
            {created.split(' ').map((word, i) => (
              <div className="word" key={`${word}-${i}`}>
                <b>{i + 1}</b>
                {word}
              </div>
            ))}
          </div>
          <div className="row" style={{ marginBottom: '0.75rem' }}>
            <button
              type="button"
              className="btn secondary"
              onClick={() => {
                setCreated(generatePhrase())
                setConfirm('')
              }}
            >
              Другая фраза
            </button>
          </div>
          <div className="field">
            <label htmlFor="confirm">Повторите фразу</label>
            <textarea
              id="confirm"
              value={confirm}
              onChange={(e) => setConfirm(e.target.value)}
              autoComplete="off"
              spellCheck={false}
            />
          </div>
          <label className="check">
            <input
              type="checkbox"
              checked={remember}
              onChange={(e) => setRemember(e.target.checked)}
            />
            Запомнить на этом браузере
          </label>
          {error ? <p className="error">{error}</p> : null}
          <button className="btn" type="submit" disabled={!confirmed || pending}>
            {pending ? 'Вход…' : 'Я записал'}
          </button>
        </form>
      ) : (
        <form className="panel" onSubmit={onLogin}>
          <div className="panel-title">Вход по фразе</div>
          <div className="field">
            <label htmlFor="login">Сид-фраза</label>
            <textarea
              id="login"
              value={loginPhrase}
              onChange={(e) => setLoginPhrase(e.target.value)}
              autoComplete="off"
              spellCheck={false}
            />
          </div>
          <label className="check">
            <input
              type="checkbox"
              checked={remember}
              onChange={(e) => setRemember(e.target.checked)}
            />
            Запомнить на этом браузере
          </label>
          {error ? <p className="error">{error}</p> : null}
          <button className="btn" type="submit" disabled={pending || !normalizePhrase(loginPhrase)}>
            {pending ? 'Вход…' : 'Войти'}
          </button>
        </form>
      )}
    </div>
  )
}
