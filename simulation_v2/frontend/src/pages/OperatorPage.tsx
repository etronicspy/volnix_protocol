import { TrafficPanel } from '../components/TrafficPanel'

export function OperatorPage() {
  return (
    <div className="page">
      <h1 className="page-title">Traffic</h1>
      <p className="page-lead">
        Боты шлют обычные транзакции в мемпул. Блоки подтверждает узел — симуляция не правит
        цепь снаружи.
      </p>
      <TrafficPanel />
    </div>
  )
}
