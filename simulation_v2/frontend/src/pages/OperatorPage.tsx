import { useState } from 'react'
import type { FormEvent } from 'react'
import { TrafficPanel } from '../components/TrafficPanel'
import { api, ApiError } from '../lib/api'

export function OperatorPage() {
  const [log, setLog] = useState<string>('')
  const [busy, setBusy] = useState(false)

  const [seed, setSeed] = useState('supplier-1')
  const [role, setRole] = useState('supplier')
  const [zkp, setZkp] = useState('zkp-demo-1')
  const [b_i, setBi] = useState('400000')
  const [s_i, setSi] = useState('200000')
  const [market, setMarket] = useState('ANT/WRT')
  const [side, setSide] = useState('SELL')
  const [amount, setAmount] = useState('1000000')
  const [price, setPrice] = useState('20000')
  const [mintAddr, setMintAddr] = useState('')
  const [mintAmt, setMintAmt] = useState('1000000')
  const [produceCount, setProduceCount] = useState('1')

  async function run(label: string, fn: () => Promise<unknown>) {
    setBusy(true)
    try {
      const res = await fn()
      setLog(`${label}\n${JSON.stringify(res, null, 2)}`)
    } catch (e) {
      setLog(`${label} FAILED\n${e instanceof ApiError ? e.message : String(e)}`)
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="page">
      <h1 className="page-title">Operator</h1>
      <p className="page-lead">
        Stand helpers — derive accounts, send earned WRT, verify roles, declare, place orders, produce
        blocks. Genesis has no WRT premint (§6.3); subsidy accrues per block, then transfer. Traffic
        bots live in a separate process on :8002.
      </p>

      <div className="grid-2">
        <TrafficPanel
          onLog={(label, payload) =>
            setLog(
              `${label}\n${typeof payload === 'string' ? payload : JSON.stringify(payload, null, 2)}`,
            )
          }
        />

        <div className="panel">
          <div className="panel-title">Produce blocks</div>
          <div className="field">
            <label htmlFor="count">Count</label>
            <input id="count" value={produceCount} onChange={(e) => setProduceCount(e.target.value)} />
          </div>
          <button
            type="button"
            className="btn"
            disabled={busy}
            onClick={() =>
              run('produce', () =>
                api.post('/api/v1/operator/produce', { count: Number(produceCount) || 1 }),
              )
            }
          >
            Produce
          </button>
        </div>

        <div className="panel">
          <div className="panel-title">Derive account</div>
          <div className="field">
            <label htmlFor="seed">Seed</label>
            <input id="seed" value={seed} onChange={(e) => setSeed(e.target.value)} />
          </div>
          <button
            type="button"
            className="btn secondary"
            disabled={busy}
            onClick={() => run('account', () => api.post('/api/v1/operator/account', { seed }))}
          >
            Derive
          </button>
        </div>

        <div className="panel">
          <div className="panel-title">Send WRT (from genesis subsidy)</div>
          <div className="field">
            <label htmlFor="mintAddr">Address</label>
            <input id="mintAddr" value={mintAddr} onChange={(e) => setMintAddr(e.target.value)} />
          </div>
          <div className="field">
            <label htmlFor="mintAmt">Amount (micro)</label>
            <input id="mintAmt" value={mintAmt} onChange={(e) => setMintAmt(e.target.value)} />
          </div>
          <button
            type="button"
            className="btn secondary"
            disabled={busy}
            onClick={() =>
              run('mint', () =>
                api.post('/api/v1/operator/mint', {
                  address: mintAddr,
                  denom: 'uwrt',
                  amount: Number(mintAmt) || 0,
                }),
              )
            }
          >
            Send WRT
          </button>
        </div>

        <div className="panel">
          <div className="panel-title">Verify identity</div>
          <div className="field">
            <label htmlFor="vseed">Seed</label>
            <input id="vseed" value={seed} onChange={(e) => setSeed(e.target.value)} />
          </div>
          <div className="field">
            <label htmlFor="role">Role</label>
            <select id="role" value={role} onChange={(e) => setRole(e.target.value)}>
              <option value="supplier">supplier</option>
              <option value="validator">validator</option>
            </select>
          </div>
          <div className="field">
            <label htmlFor="zkp">ZKP proof id</label>
            <input id="zkp" value={zkp} onChange={(e) => setZkp(e.target.value)} />
          </div>
          <button
            type="button"
            className="btn"
            disabled={busy}
            onClick={() =>
              run('verify', () =>
                api.post('/api/v1/operator/verify', {
                  seed,
                  desired_role: role,
                  zkp_proof: zkp,
                }),
              )
            }
          >
            Submit verify tx
          </button>
        </div>

        <div className="panel">
          <div className="panel-title">Declare participation</div>
          <div className="field">
            <label htmlFor="dseed">Validator seed</label>
            <input
              id="dseed"
              value={seed}
              onChange={(e) => setSeed(e.target.value)}
              placeholder="volnix-genesis-validator-v2"
            />
          </div>
          <div className="field">
            <label htmlFor="bi">b_i (micro)</label>
            <input id="bi" value={b_i} onChange={(e) => setBi(e.target.value)} />
          </div>
          <div className="field">
            <label htmlFor="si">s_i (micro)</label>
            <input id="si" value={s_i} onChange={(e) => setSi(e.target.value)} />
          </div>
          <button
            type="button"
            className="btn"
            disabled={busy}
            onClick={() =>
              run('declare', () =>
                api.post('/api/v1/operator/declare', {
                  seed,
                  b_i: Number(b_i) || 0,
                  s_i: Number(s_i) || 0,
                }),
              )
            }
          >
            Submit declare
          </button>
          <p className="muted" style={{ marginTop: '0.75rem', fontSize: '0.85rem' }}>
            Genesis seed: <code>volnix-genesis-validator-v2</code>. With L_i = 1e6, try b_i=400000, s_i=200000.
          </p>
        </div>

        <div className="panel">
          <div className="panel-title">Place order</div>
          <OrderForm
            seed={seed}
            setSeed={setSeed}
            market={market}
            setMarket={setMarket}
            side={side}
            setSide={setSide}
            amount={amount}
            setAmount={setAmount}
            price={price}
            setPrice={setPrice}
            busy={busy}
            onSubmit={(e) => {
              e.preventDefault()
              void run('order', () =>
                api.post('/api/v1/operator/order', {
                  seed,
                  market,
                  side,
                  order_type: 'LIMIT',
                  amount: Number(amount) || 0,
                  price: Number(price) || 0,
                }),
              )
            }}
          />
        </div>
      </div>

      <div className="panel" style={{ marginTop: '1rem' }}>
        <div className="panel-title">Response</div>
        <pre className="pre">{log || '—'}</pre>
      </div>
    </div>
  )
}

interface OrderFormProps {
  seed: string
  setSeed: (v: string) => void
  market: string
  setMarket: (v: string) => void
  side: string
  setSide: (v: string) => void
  amount: string
  setAmount: (v: string) => void
  price: string
  setPrice: (v: string) => void
  busy: boolean
  onSubmit: (e: FormEvent) => void
}

function OrderForm(props: OrderFormProps) {
  const {
    seed,
    setSeed,
    market,
    setMarket,
    side,
    setSide,
    amount,
    setAmount,
    price,
    setPrice,
    busy,
    onSubmit,
  } = props
  return (
    <form onSubmit={onSubmit}>
      <div className="field">
        <label htmlFor="oseed">Seed</label>
        <input id="oseed" value={seed} onChange={(e) => setSeed(e.target.value)} />
      </div>
      <div className="field">
        <label htmlFor="market">Market</label>
        <select id="market" value={market} onChange={(e) => setMarket(e.target.value)}>
          <option value="ANT/WRT">ANT/WRT</option>
          <option value="LZN/WRT">LZN/WRT</option>
        </select>
      </div>
      <div className="field">
        <label htmlFor="side">Side</label>
        <select id="side" value={side} onChange={(e) => setSide(e.target.value)}>
          <option value="SELL">SELL</option>
          <option value="BUY">BUY</option>
        </select>
      </div>
      <div className="field">
        <label htmlFor="amount">Amount (micro)</label>
        <input id="amount" value={amount} onChange={(e) => setAmount(e.target.value)} />
      </div>
      <div className="field">
        <label htmlFor="price">Price (micro WRT per micro base)</label>
        <input id="price" value={price} onChange={(e) => setPrice(e.target.value)} />
      </div>
      <button type="submit" className="btn" disabled={busy}>
        Place order
      </button>
    </form>
  )
}
