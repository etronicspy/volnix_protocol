/** Node REST API (Vite: set VITE_API_URL for non-localhost). */
export const API_BASE = import.meta.env.VITE_API_URL ?? 'http://127.0.0.1:8001'

/** Traffic / economy control process (simulation_v2/traffic). */
export const TRAFFIC_BASE = import.meta.env.VITE_TRAFFIC_URL ?? 'http://127.0.0.1:8002'

/** Read-only explorer (simulation_v2/frontend) — target for address / block links. */
export const EXPLORER_URL = import.meta.env.VITE_EXPLORER_URL ?? 'http://127.0.0.1:5174'

/** Node WebSocket for live blocks (wallet section refreshes on `new_block`). */
export const WS_URL = import.meta.env.VITE_WS_URL ?? 'ws://127.0.0.1:8001/ws'

export const SCALE = 1_000_000

/** Stand genesis validator. Same string as backend genesis.default.json and traffic GENESIS_SEED. */
export const GENESIS_SEED = 'volnix-genesis-validator-v2'

/** localStorage keys for the wallet section (plain-text seeds, stand only). */
export const WALLETS_KEY = 'volnix.traffic.wallets'
export const WALLET_ACTIVE_KEY = 'volnix.traffic.wallet.active'
