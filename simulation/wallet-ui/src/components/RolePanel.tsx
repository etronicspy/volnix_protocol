import type { Account, Feedback } from '../types'
import { Btn, FeedbackBanner, Panel } from './ui'

interface RolePanelProps {
  account: Account
  pending: boolean
  feedback: Feedback | null
  isSim: boolean
  genesisValidator?: string
  genesisProvider?: string
  onSetRole: (role: 'citizen' | 'provider' | 'validator') => void
}

export function RolePanel({
  account,
  pending,
  feedback,
  isSim,
  onSetRole,
}: RolePanelProps) {
  const roles: Array<{
    id: 'citizen' | 'provider' | 'validator'
    title: string
    body: string
  }> = [
    {
      id: 'citizen',
      title: 'Гражданин',
      body: 'WRT/LZN. ANT не хранится. Базовый тип кошелька §4.2.',
    },
    {
      id: 'provider',
      title: 'Поставщик',
      body: 'Продажа ANT на рынке. Узел проверит ZKP (кроме genesis provider).',
    },
    {
      id: 'validator',
      title: 'Валидатор',
      body: 'Покупка ANT, активация LZN, declare. Узел проверит ZKP и LZN (кроме genesis).',
    },
  ]

  return (
    <Panel
      title="Тип кошелька"
      hint={isSim
        ? 'Tx set_role → мемпул; канон проверяется при сборке блока.'
        : 'Tx set_role → x/ident модуль блокчейна.'}
    >
      {feedback ? <div className="mb-4"><FeedbackBanner {...feedback} /></div> : null}

      <div className="grid gap-3 md:grid-cols-3">
        {roles.map((r) => {
          const active = account.role === r.id
          return (
            <div
              key={r.id}
              className={`rounded-2xl border p-4 ${
                active
                  ? 'border-[var(--mint)] bg-[color-mix(in_oklab,var(--mint)_10%,transparent)]'
                  : 'border-[var(--line)] bg-[var(--panel-2)]'
              }`}
            >
              <h3 className="font-semibold text-[var(--sand)]">{r.title}</h3>
              <p className="mt-2 text-sm text-[var(--muted)]">{r.body}</p>
              <Btn
                className="mt-4 w-full"
                variant={active ? 'ghost' : 'primary'}
                disabled={pending || active}
                data-tip={r.body}
                onClick={() => onSetRole(r.id)}
              >
                {active ? 'Текущая роль' : 'Сменить'}
              </Btn>
            </div>
          )
        })}
      </div>
    </Panel>
  )
}
