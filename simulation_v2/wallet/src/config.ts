/** REST API (Vite: set VITE_API_URL for non-localhost). */
export const API_BASE = import.meta.env.VITE_API_URL ?? 'http://127.0.0.1:8001'

/** WebSocket endpoint for live blocks. */
export const WS_URL = import.meta.env.VITE_WS_URL ?? 'ws://127.0.0.1:8001/ws'

export const SCALE = 1_000_000

export const SESSION_KEY = 'volnix.wallet.seed'
export const LOCAL_KEY = 'volnix.wallet.seed.remember'

/** Stand genesis validator. Same string as genesis.default.json. */
export const GENESIS_SEED = 'volnix-genesis-validator-v2'
