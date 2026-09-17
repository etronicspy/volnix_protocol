/** Эталон эмиссии LZN в симуляции (§4.2); потолок активированных на адрес = ⌊ref/3⌋. */
export const LZN_TOTAL_SUPPLY_REF = 10_000
export const LZN_MAX_FROZEN_PER_ADDRESS = Math.floor(LZN_TOTAL_SUPPLY_REF / 3)

export const API_BASE = import.meta.env.VITE_API_URL ?? 'http://localhost:8000'
export const WS_URL = import.meta.env.VITE_WS_URL ?? 'ws://localhost:8000/ws'

export const CHAIN_RPC_URL = import.meta.env.VITE_CHAIN_RPC ?? 'http://localhost:26657'
export const CHAIN_REST_URL = import.meta.env.VITE_CHAIN_REST ?? 'http://localhost:1317'
export const CHAIN_PREFIX = import.meta.env.VITE_CHAIN_PREFIX ?? 'volnix'
export const CHAIN_ID = import.meta.env.VITE_CHAIN_ID ?? 'volnix-1'

export const SELECTED_ADDRESS_KEY = 'volnix_sim_wallet_address'
