import { useOutletContext } from 'react-router-dom'

export interface LiveCtx {
  chainTick: number
  mempoolTick: number
}

export function useLive(): LiveCtx {
  return useOutletContext<LiveCtx>()
}
