import { useState } from 'react'

import { useBackend } from '../adapters'
import type { Feedback } from '../types'
import { Btn, FeedbackBanner, Field, Panel, TextInput, TextSelect } from './ui'

interface FaucetPanelProps {
  address: string
  onDone: (feedback: Feedback) => void
}

export function FaucetPanel({ address, onDone }: FaucetPanelProps) {
  const { adapter } = useBackend()
  const [asset, setAsset] = useState<'wrt' | 'lzn' | 'ant'>('wrt')
  const [amount, setAmount] = useState('1000')
  const [pending, setPending] = useState(false)
  const [feedback, setFeedback] = useState<Feedback | null>(null)

  const run = async () => {
    if (!adapter.mintAsset) return
    setPending(true)
    setFeedback(null)
    try {
      const res = await adapter.mintAsset(address, Number(amount), asset)
      const fb = {
        ok: res.accepted,
        text: res.accepted
          ? `Mint queued${res.tx_hash ? ` · ${res.tx_hash.slice(0, 12)}…` : ''}`
          : res.message || 'Mint failed',
      }
      setFeedback(fb)
      onDone(fb)
    } catch (err) {
      const fb = { ok: false, text: err instanceof Error ? err.message : 'Network error' }
      setFeedback(fb)
      onDone(fb)
    } finally {
      setPending(false)
    }
  }

  return (
    <Panel
      title="Faucet (sim-operator)"
      hint="POST /api/sim-operator/mint — только для симуляции."
    >
      {feedback ? <div className="mb-4"><FeedbackBanner {...feedback} /></div> : null}

      <div className="grid gap-4 sm:grid-cols-2">
        <Field label="Актив" tip="Mint из казны симуляции. ANT только поставщику или валидатору — иначе отказ admission.">
          <TextSelect value={asset} onChange={(e) => setAsset(e.target.value as 'wrt' | 'lzn' | 'ant')}>
            <option value="wrt">WRT</option>
            <option value="lzn">LZN</option>
            <option value="ant">ANT</option>
          </TextSelect>
        </Field>
        <Field label="Сумма">
          <TextInput type="number" min={0} step="any" value={amount} onChange={(e) => setAmount(e.target.value)} />
        </Field>
      </div>

      <div className="mt-5">
        <Btn data-tip="POST /api/sim-operator/mint → мемпул. Не часть протоколного кошелька." disabled={pending || !(Number(amount) > 0)} onClick={() => void run()}>
          Mint в мемпул
        </Btn>
      </div>
    </Panel>
  )
}
