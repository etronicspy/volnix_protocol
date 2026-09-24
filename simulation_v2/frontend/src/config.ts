/** REST API (Vite: set VITE_API_URL for non-localhost). */
export const API_BASE = import.meta.env.VITE_API_URL ?? 'http://127.0.0.1:8001'

/** WebSocket endpoint for live blocks. */
export const WS_URL = import.meta.env.VITE_WS_URL ?? 'ws://127.0.0.1:8001/ws'

/** Traffic / economy control process (simulation_v2/traffic). */
export const TRAFFIC_BASE = import.meta.env.VITE_TRAFFIC_URL ?? 'http://127.0.0.1:8002'

export const SCALE = 1_000_000

/** Explorer REST refresh interval (backend/traffic stay on Sim speed). */
export const EXPLORER_POLL_MS = 1000
