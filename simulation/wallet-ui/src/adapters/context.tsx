import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react'

import { CHAIN_ID, CHAIN_PREFIX, CHAIN_REST_URL, CHAIN_RPC_URL } from '../config'
import { SimAdapter } from './SimAdapter'
import type { BackendAdapter, BackendMode, WalletState } from './types'

export interface BackendContextValue {
  mode: BackendMode
  adapter: BackendAdapter
  state: WalletState | null
  connected: boolean
  error: string | null
  refresh: () => Promise<void>
  switchMode: (mode: BackendMode) => void
  connectChain: (mnemonic: string) => Promise<string>
  chainReady: boolean
}

const Ctx = createContext<BackendContextValue | null>(null)

const MODE_KEY = 'volnix_backend_mode'

export function BackendProvider({ children }: { children: ReactNode }) {
  const [mode, setMode] = useState<BackendMode>(
    () => (localStorage.getItem(MODE_KEY) as BackendMode) || 'sim',
  )

  const [adapter, setAdapter] = useState<BackendAdapter>(() => new SimAdapter())
  const [state, setState] = useState<WalletState | null>(null)
  const [connected, setConnected] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [chainReady, setChainReady] = useState(false)
  const chainAdapterRef = useRef<BackendAdapter | null>(null)
  const unsubRef = useRef<(() => void) | null>(null)

  const refresh = useCallback(async () => {
    const a = mode === 'chain' ? chainAdapterRef.current : adapter
    if (!a) return
    try {
      const s = await a.getState()
      setState(s)
      setError(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load state')
    }
  }, [adapter, mode])

  const startSubscription = useCallback((a: BackendAdapter) => {
    unsubRef.current?.()
    const unsub = a.subscribe({
      onState: (s) => {
        setState(s)
        setError(null)
      },
      onConnected: setConnected,
      onError: (e) => setError(e),
    })
    unsubRef.current = unsub
  }, [])

  useEffect(() => {
    if (mode === 'sim') {
      void refresh()
      startSubscription(adapter)
    }
    return () => {
      unsubRef.current?.()
      unsubRef.current = null
    }
  }, [adapter, mode, refresh, startSubscription])

  const switchMode = useCallback(
    (m: BackendMode) => {
      unsubRef.current?.()
      unsubRef.current = null
      adapter.disconnect()
      chainAdapterRef.current?.disconnect()
      chainAdapterRef.current = null

      localStorage.setItem(MODE_KEY, m)
      setMode(m)
      setState(null)
      setConnected(false)
      setError(null)
      setChainReady(false)

      if (m === 'sim') {
        const sim = new SimAdapter()
        setAdapter(sim)
      } else {
        setAdapter({ mode: 'chain' } as BackendAdapter)
      }
    },
    [adapter],
  )

  const connectChain = useCallback(
    async (mnemonic: string) => {
      const { ChainAdapter } = await import('./ChainAdapter')
      const ca = new ChainAdapter(CHAIN_RPC_URL, CHAIN_REST_URL, CHAIN_PREFIX, CHAIN_ID)
      const address = await ca.connectWithMnemonic(mnemonic)
      chainAdapterRef.current = ca
      setAdapter(ca)
      setChainReady(true)
      const s = await ca.getState()
      setState(s)
      setConnected(true)
      startSubscription(ca)
      return address
    },
    [startSubscription],
  )

  const value = useMemo(
    (): BackendContextValue => ({
      mode,
      adapter,
      state,
      connected,
      error,
      refresh,
      switchMode,
      connectChain,
      chainReady,
    }),
    [mode, adapter, state, connected, error, refresh, switchMode, connectChain, chainReady],
  )

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>
}

export function useBackend(): BackendContextValue {
  const ctx = useContext(Ctx)
  if (!ctx) throw new Error('useBackend must be inside <BackendProvider>')
  return ctx
}
