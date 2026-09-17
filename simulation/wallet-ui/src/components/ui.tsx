import type { ButtonHTMLAttributes, InputHTMLAttributes, ReactNode, SelectHTMLAttributes } from 'react'

interface PanelProps {
  title: string
  hint?: string
  children: ReactNode
}

export function Panel({ title, hint, children }: PanelProps) {
  return (
    <section className="rise rounded-2xl border border-[var(--line)] bg-[color-mix(in_oklab,var(--panel)_92%,black)] p-5 shadow-[0_20px_50px_rgba(0,0,0,0.25)]">
      <header className="mb-4 border-b border-[var(--line)] pb-3">
        <h2 className="text-lg font-semibold tracking-tight text-[var(--sand)]">{title}</h2>
        {hint ? <p className="mt-1 text-sm text-[var(--muted)]">{hint}</p> : null}
      </header>
      {children}
    </section>
  )
}

interface FieldProps {
  label: string
  tip?: string
  children: ReactNode
}

export function Field({ label, tip, children }: FieldProps) {
  return (
    <label className="block space-y-1.5 text-sm">
      <span className="text-[var(--muted)]" {...(tip ? { 'data-tip': tip } : {})}>
        {label}
      </span>
      {children}
    </label>
  )
}

const inputClass =
  'w-full rounded-xl border border-[var(--line)] bg-[var(--panel-2)] px-3 py-2.5 text-[var(--sand)] outline-none transition focus:border-[var(--mint-dim)]'

export function TextInput(props: InputHTMLAttributes<HTMLInputElement>) {
  return <input {...props} className={`${inputClass} ${props.className ?? ''}`} />
}

export function TextSelect(props: SelectHTMLAttributes<HTMLSelectElement>) {
  return <select {...props} className={`${inputClass} ${props.className ?? ''}`} />
}

interface BtnProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'ghost' | 'danger'
}

export function Btn({ variant = 'primary', className = '', ...props }: BtnProps) {
  const styles =
    variant === 'primary'
      ? 'bg-[var(--mint)] text-[#062019] hover:brightness-110 disabled:opacity-40'
      : variant === 'danger'
        ? 'bg-[color-mix(in_oklab,var(--danger)_25%,transparent)] text-[var(--danger)] border border-[color-mix(in_oklab,var(--danger)_40%,transparent)]'
        : 'bg-transparent text-[var(--sand)] border border-[var(--line)] hover:bg-[var(--panel-2)]'
  return (
    <button
      type="button"
      {...props}
      className={`rounded-xl px-4 py-2.5 text-sm font-semibold transition disabled:cursor-not-allowed ${styles} ${className}`}
    />
  )
}

export function FeedbackBanner({ ok, text }: { ok: boolean; text: string }) {
  return (
    <div
      className={`rounded-xl px-3 py-2 text-sm ${
        ok
          ? 'bg-[color-mix(in_oklab,var(--ok)_18%,transparent)] text-[var(--ok)]'
          : 'bg-[color-mix(in_oklab,var(--danger)_18%,transparent)] text-[var(--danger)]'
      }`}
    >
      {text}
    </div>
  )
}

export function formatAddr(addr: string, head = 10, tail = 6): string {
  if (addr.length <= head + tail + 1) return addr
  return `${addr.slice(0, head)}…${addr.slice(-tail)}`
}

export function formatAmt(n: number, digits = 4): string {
  if (!Number.isFinite(n)) return '—'
  return n.toLocaleString(undefined, {
    minimumFractionDigits: 0,
    maximumFractionDigits: digits,
  })
}
