import { useState } from 'react'

import type { Feedback } from '../types'
import { Btn, FeedbackBanner, Field, Panel, TextInput, TextSelect } from './ui'

interface SendPanelProps {
  address: string
  pending: boolean
  feedback: Feedback | null
  isSim: boolean
  onSend: (payload: { address: string; to_address: string; amount: number; asset: string }) => void
}

export function SendPanel({ address, pending, feedback, isSim, onSend }: SendPanelProps) {
  const [to, setTo] = useState('')
  const [amount, setAmount] = useState('10')
  const [asset, setAsset] = useState<'wrt' | 'lzn' | 'ant'>('wrt')

  return (
    <Panel
      title="Перевод"
      hint={isSim
        ? 'Узел примет в блоке только WRT и LZN; прямой ANT отклонит (§4.1).'
        : 'WRT / LZN через MsgSend. ANT через x/anteil.'}
    >
      {feedback ? <div className="mb-4"><FeedbackBanner {...feedback} /></div> : null}

      <div className="grid gap-4 sm:grid-cols-2">
        <Field label="Получатель" tip="Адрес volnix… Получатель появится в состоянии, если его ещё нет.">
          <TextInput
            className="mono"
            placeholder="volnix…"
            value={to}
            onChange={(e) => setTo(e.target.value.trim())}
          />
        </Field>
        <Field label="Актив" tip="WRT и LZN проходят. ANT — прямым MsgSend узел отклонит (§4.1).">
          <TextSelect value={asset} onChange={(e) => setAsset(e.target.value as 'wrt' | 'lzn' | 'ant')}>
            <option value="wrt">WRT</option>
            <option value="lzn">LZN</option>
            <option value="ant">ANT</option>
          </TextSelect>
        </Field>
        <Field label="Сумма">
          <TextInput
            type="number"
            min={0}
            step="any"
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
          />
        </Field>
      </div>

      <div className="mt-5">
        <Btn
          data-tip={isSim ? 'Попадёт в мемпул. Узел примет или отклонит в следующем блоке.' : 'Подпись CosmJS и broadcast в ноду.'}
          disabled={pending || !to || !(Number(amount) > 0)}
          onClick={() =>
            onSend({
              address,
              to_address: to,
              amount: Number(amount),
              asset,
            })
          }
        >
          {isSim ? 'Отправить в мемпул' : 'Подписать и отправить'}
        </Btn>
      </div>
    </Panel>
  )
}
