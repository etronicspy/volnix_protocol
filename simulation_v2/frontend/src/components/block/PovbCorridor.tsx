interface Props {
  bMin: number
  bMax: number
  sumB: number
  setUpdated: boolean
}

/** Visual λ corridor: marker for Σb between b_min and b_max. */
export function PovbCorridor({ bMin, bMax, sumB, setUpdated }: Props) {
  const span = Math.max(bMax - bMin, 1)
  const pad = span * 0.15
  const lo = Math.min(bMin, sumB) - pad
  const hi = Math.max(bMax, sumB) + pad
  const width = Math.max(hi - lo, 1)
  const pct = (v: number) => `${Math.max(0, Math.min(100, ((v - lo) / width) * 100))}%`

  return (
    <div className="corridor">
      <div className="corridor-track">
        <div className="corridor-zone" style={{ left: pct(bMin), width: `calc(${pct(bMax)} - ${pct(bMin)})` }} />
        <div
          className={`corridor-marker ${setUpdated ? 'ok' : 'bad'}`}
          style={{ left: pct(sumB) }}
          title={`Σb = ${sumB}`}
        />
      </div>
      <div className="corridor-labels mono muted">
        <span>b_min {bMin}</span>
        <span>Σb {sumB}</span>
        <span>b_max {bMax}</span>
      </div>
    </div>
  )
}
