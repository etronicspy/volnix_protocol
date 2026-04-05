/** Эталон эмиссии LZN в симуляции (§4.2); потолок активированных на адрес = ⌊ref/3⌋ (= 3333 при ref 10 000). */
export const LZN_TOTAL_SUPPLY_REF = 10_000
export const LZN_MAX_FROZEN_PER_ADDRESS = Math.floor(LZN_TOTAL_SUPPLY_REF / 3)

/** REST API (Vite: set VITE_API_URL for non-localhost). */
export const API_BASE = import.meta.env.VITE_API_URL ?? 'http://localhost:8000'

/** WebSocket endpoint for live blocks. */
export const WS_URL = import.meta.env.VITE_WS_URL ?? 'ws://localhost:8000/ws'
