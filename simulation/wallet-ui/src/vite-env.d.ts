/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_URL?: string
  readonly VITE_WS_URL?: string
  readonly VITE_CHAIN_RPC?: string
  readonly VITE_CHAIN_REST?: string
  readonly VITE_CHAIN_PREFIX?: string
  readonly VITE_CHAIN_ID?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
