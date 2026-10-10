/** REST API (Vite: set VITE_API_URL for non-localhost). */
export const API_BASE = import.meta.env.VITE_API_URL ?? 'http://127.0.0.1:8001'

/** WebSocket endpoint for live blocks. */
export const WS_URL = import.meta.env.VITE_WS_URL ?? 'ws://127.0.0.1:8001/ws'

/** Stand control UI (simulation_v2/traffic-ui): traffic, time scale, reset chain. */
export const TRAFFIC_UI_URL = import.meta.env.VITE_TRAFFIC_UI_URL ?? 'http://127.0.0.1:5176'

export const SCALE = 1_000_000

/** Mempool poll + WS-offline fallback for chain REST (node pace is adaptive §6.2). */
export const MEMPOOL_POLL_MS = 1000
