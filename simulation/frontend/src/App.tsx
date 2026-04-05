import { useEffect, useState } from 'react'
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts'
import { WalletPanel } from './components/WalletPanel'
import { TradingViewMarketWidget } from './components/TradingViewMarketWidget'
import { API_BASE, WS_URL } from './config'

interface Account {
  address: string;
  wrt_balance: number;
  lzn_balance: number;
  lzn_frozen_mining?: number;
  ant_balance: number;
  role: string;
  zkp_verified?: boolean;
}

interface Order {
  id: string;
  owner: string;
  order_type: string;
  price: number;
  amount: number;
  filled: number;
  timestamp: number;
}

interface Market {
  bids: Order[];
  asks: Order[];
  last_price: number;
  history: { time: string; price: number; ts?: number }[];
}

interface Transaction {
  tx_hash: string;
  tx_type: string;
  sender: string;
  receiver: string;
  amount: number;
  asset_type?: string;
  price: number;
  role: string;
  timestamp: number;
  buyer?: string;
  seller?: string;
  details?: string;
  stake_amount?: number;
}

/** Сырой блок с цепочки: транзакции — произвольные объекты (trade, declare, протокол и т.д.). */
interface Block {
  height: number;
  hash: string;
  tx_count: number;
  timestamp: number;
  transactions: Record<string, unknown>[];
}

interface CanonLogEntry {
  id: number
  ts: number
  source: string
  status: string
  category: string
  canon: string
  title: string
  detail: string
  tx_hash: string
  block_height: number | null
  meta?: Record<string, unknown>
}

interface NetworkState {
  status: string;
  block_height: number;
  block_time: number;
  accounts_count: number;
  mempool_size?: number;
  accounts: Record<string, Account>;
  recent_txs: Transaction[];
  market: Market;
  blocks: Block[];
  tps_history: {time: string, tps: number}[];
  blocks_per_epoch?: number;
  epoch_ant_sold_volume?: number;
  epoch_ant_sold_last?: number;
  epoch_emission_coefficient?: number;
  genesis_validator?: string;
  genesis_provider?: string;
  sim_treasury?: string;
  canon_log?: CanonLogEntry[];
  /** Эталон основной цепи, сек (обычно 60) */
  canonical_block_interval_sec?: number;
  /** Интервал блока в этой симуляции, сек */
  sim_block_interval_sec?: number;
}

/** Тело GET /api/state и поле `state` в сообщениях WebSocket. */
interface SimulatorApiState {
  height: number
  mempool_size?: number
  accounts_count: number
  accounts: Record<string, Account>
  market: Market
  blocks: Block[]
  tps_history: { time: string; tps: number }[]
  blocks_per_epoch?: number
  epoch_ant_sold_volume?: number
  epoch_ant_sold_last?: number
  epoch_emission_coefficient?: number
  genesis_validator?: string
  genesis_provider?: string
  sim_treasury?: string
  canon_log?: CanonLogEntry[]
  canonical_block_interval_sec?: number
  sim_block_interval_sec?: number
}

function mergeFromSimulatorApi(
  prev: NetworkState,
  snapshot: SimulatorApiState,
  blockTime: number,
  opts: { recent: 'append' | 'keep' | 'clear'; blockTxs?: Record<string, unknown>[] }
): NetworkState {
  const { recent, blockTxs = [] } = opts
  let recent_txs = prev.recent_txs
  if (recent === 'clear') {
    recent_txs = []
  } else if (recent === 'append') {
    recent_txs = [...(blockTxs as unknown as Transaction[]), ...prev.recent_txs].slice(0, 10)
  }
  return {
    ...prev,
    block_height: snapshot.height,
    block_time: blockTime,
    accounts_count: snapshot.accounts_count,
    mempool_size: snapshot.mempool_size,
    accounts: snapshot.accounts,
    market: snapshot.market,
    blocks: snapshot.blocks,
    tps_history: snapshot.tps_history,
    blocks_per_epoch: snapshot.blocks_per_epoch ?? prev.blocks_per_epoch,
    epoch_ant_sold_volume: snapshot.epoch_ant_sold_volume,
    epoch_ant_sold_last: snapshot.epoch_ant_sold_last,
    epoch_emission_coefficient: snapshot.epoch_emission_coefficient,
    genesis_validator: snapshot.genesis_validator,
    genesis_provider: snapshot.genesis_provider,
    sim_treasury: snapshot.sim_treasury,
    canonical_block_interval_sec: snapshot.canonical_block_interval_sec ?? prev.canonical_block_interval_sec,
    sim_block_interval_sec: snapshot.sim_block_interval_sec ?? prev.sim_block_interval_sec,
    canon_log: snapshot.canon_log ?? prev.canon_log ?? [],
    recent_txs,
  }
}

function App() {
  const [state, setState] = useState<NetworkState>({
    status: 'Connecting...',
    block_height: 0,
    block_time: 0,
    accounts_count: 0,
    accounts: {},
    recent_txs: [],
    market: { bids: [], asks: [], last_price: 0, history: [] },
    blocks: [],
    tps_history: [],
    blocks_per_epoch: 10080,
    canon_log: [],
  })
  
  const [blockTimeInput, setBlockTimeInput] = useState<string>("60")
  const [botStatus, setBotStatus] = useState({ is_running: false, intensity: 1.0 })
  const [botIntensityInput, setBotIntensityInput] = useState<string>("1.0")
  const [selectedBlockHeight, setSelectedBlockHeight] = useState<number | null>(null)

  const selectedBlock = state.blocks.find(b => b.height === selectedBlockHeight)

  // Manual Order Form State
  const [orderAccount, setOrderAccount] = useState<string>("")
  const [orderType, setOrderType] = useState<string>("buy")
  const [orderPrice, setOrderPrice] = useState<string>("10.0")
  const [orderAmount, setOrderAmount] = useState<string>("5.0")
  const [orderMarket, setOrderMarket] = useState(false)
  const [orderMaxWrt, setOrderMaxWrt] = useState<string>("")
  const [selectedWallet, setSelectedWallet] = useState<string>("")

  useEffect(() => {
    // Fetch initial bot status
    fetch(`${API_BASE}/api/bot/status`)
      .then(res => res.json())
      .then(data => {
        setBotStatus(data)
        setBotIntensityInput(data.intensity.toString())
      })

    // Connect to WebSocket
    const ws = new WebSocket(WS_URL)

    ws.onopen = () => {
      setState(prev => ({ ...prev, status: 'Connected (Live)' }))
    }

    ws.onmessage = (event) => {
      const msg = JSON.parse(event.data) as {
        type: string
        data: {
          state: SimulatorApiState
          block_time: number
          block?: { height?: number; transactions?: Record<string, unknown>[] }
        }
      }
      if (msg.type === 'init' || msg.type === 'new_block') {
        const block = msg.data.block
        const txs = block?.transactions ?? []
        const isResetSnapshot =
          msg.type === 'new_block' &&
          msg.data.state.height === 0 &&
          block?.height === 0
        const recent: 'append' | 'keep' | 'clear' = isResetSnapshot
          ? 'clear'
          : msg.type === 'init'
            ? 'keep'
            : 'append'
        setState(prev =>
          mergeFromSimulatorApi(prev, msg.data.state, msg.data.block_time, { recent, blockTxs: txs })
        )
        if (msg.type === 'init') {
          setBlockTimeInput(msg.data.block_time.toString())
        }
      }
    }

    ws.onclose = () => {
      setState(prev => ({ ...prev, status: 'Disconnected' }))
    }

    return () => {
      ws.close()
    }
  }, [])

  const handleBotControl = async (action: 'start' | 'stop') => {
    const parsed = parseFloat(botIntensityInput)
    const intensity = Number.isFinite(parsed) ? parsed : 1.0
    await fetch(`${API_BASE}/api/bot/control`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ action, intensity }),
    })
    const statusRes = await fetch(`${API_BASE}/api/bot/status`)
    const data = (await statusRes.json()) as { is_running: boolean; intensity: number }
    setBotStatus({ is_running: data.is_running, intensity: data.intensity })
    setBotIntensityInput(String(data.intensity))
  }

  const handleCreateOrder = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!orderAccount) {
      alert("Please select an account")
      return
    }
    const amt = parseFloat(orderAmount)
    if (Number.isNaN(amt) || amt <= 0) {
      alert('Amount must be positive')
      return
    }
    const body: Record<string, unknown> = {
      op: 'create_order',
      address: orderAccount,
      side: orderType,
      amount: amt,
    }
    if (orderMarket) {
      body.market = true
      if (orderType === 'buy' && orderMaxWrt.trim() !== '') {
        const cap = parseFloat(orderMaxWrt)
        if (Number.isNaN(cap) || cap <= 0) {
          alert('max_wrt must be positive')
          return
        }
        body.max_wrt = cap
      }
    } else {
      const p = parseFloat(orderPrice)
      if (Number.isNaN(p) || p <= 0) {
        alert('Price must be positive for limit orders')
        return
      }
      body.price = p
    }
    const res = await fetch(`${API_BASE}/api/wallet/submit`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    })
    const data = await res.json()
    alert(data.accepted ? `Принято в мемпул. tx ${(data.tx_hash || '').slice(0, 14)}…` : data.message || 'Отклонено')
  }

  const handleCreateAccounts = async () => {
    await fetch(`${API_BASE}/api/sim-operator/accounts`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ count: 5 })
    })
  }

  const handleResetState = async () => {
    if (!confirm('Are you sure you want to completely reset and delete the blockchain state?')) {
      return
    }
    try {
      const res = await fetch(`${API_BASE}/api/sim-operator/reset`, { method: 'POST' })
      if (!res.ok) {
        alert(`Сброс не удался (HTTP ${res.status}).`)
        return
      }
      const root = (await fetch(`${API_BASE}/`).then((r) => r.json())) as { block_time: number }
      const st = (await fetch(`${API_BASE}/api/state`).then((r) => r.json())) as SimulatorApiState
      setState((prev) => mergeFromSimulatorApi(prev, st, root.block_time, { recent: 'clear' }))
      setBlockTimeInput(String(root.block_time))
      setSelectedBlockHeight(null)
    } catch {
      alert('Сброс: ошибка сети или сервера.')
    }
  }

  const handleUpdateBlockTime = async () => {
    const time = parseFloat(blockTimeInput)
    if (!isNaN(time) && time >= 0.1 && time <= 300) {
      await fetch(`${API_BASE}/api/sim-operator/block-time`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ time_sec: time })
      })
      // Update local state to reflect the change immediately
      setState(prev => ({ ...prev, block_time: time }))
    } else {
      alert("Интервал симуляции: от 0.1 до 300 с. В эталонной цепи блок ≈ 60 с (1 мин).")
    }
  }

  const handleMint = async (address: string, asset_type: string = "wrt") => {
    const res = await fetch(`${API_BASE}/api/sim-operator/mint`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ address, amount: 1000, asset_type })
    })
    const data = await res.json()
    if (data.status === "error") {
      alert(data.message)
    } else if (data.status === "queued") {
      alert(`В мемпул: ${data.message}\ntx ${(data.tx_hash || "").slice(0, 16)}… — исполнение в следующем блоке`)
    }
  }

  const selectedAccount = selectedWallet ? state.accounts[selectedWallet] : undefined

  const MAIN_TABS = [
    { id: 'overview' as const, label: 'Обзор' },
    { id: 'chain' as const, label: 'Блокчейн' },
    { id: 'market' as const, label: 'Рынок' },
    { id: 'accounts' as const, label: 'Счета' },
    { id: 'sim' as const, label: 'Симуляция' },
  ]
  const [mainTab, setMainTab] = useState<(typeof MAIN_TABS)[number]['id']>('overview')

  return (
    <div className="min-h-screen bg-gray-900 text-white p-8">
      <div className="max-w-6xl mx-auto">
        <div className="flex justify-between items-center mb-6">
          <div>
            <h1 className="text-4xl font-bold text-blue-400">Volnix Protocol Simulation</h1>
            <p className="text-gray-500 text-sm mt-1">
              §5.5: {state.blocks_per_epoch ?? 10080} blocks/epoch (эталон 1 блок/мин × 7 сут) · Mempool: {state.mempool_size ?? 0} tx · Sold epoch: {Number(state.epoch_ant_sold_volume ?? 0).toFixed(2)} ANT · Prev epoch sold: {Number(state.epoch_ant_sold_last ?? 0).toFixed(2)} · coeff: {Number(state.epoch_emission_coefficient ?? 1).toFixed(4)}
            </p>
          </div>
          <div className={`px-4 py-2 rounded-full font-semibold ${state.status.includes('Live') ? 'bg-green-500/20 text-green-400' : 'bg-red-500/20 text-red-400'}`}>
            {state.status}
          </div>
        </div>

        <div className="flex flex-wrap gap-1 border-b border-gray-700 mb-6 -mx-1 px-1">
          {MAIN_TABS.map((t) => (
            <button
              key={t.id}
              type="button"
              onClick={() => setMainTab(t.id)}
              className={`px-4 py-2.5 rounded-t-lg text-sm font-medium transition-colors border border-b-0 ${
                mainTab === t.id
                  ? 'bg-gray-800 text-blue-300 border-gray-600 relative z-[1] mb-[-1px]'
                  : 'bg-gray-900/80 text-gray-400 border-transparent hover:text-gray-200'
              }`}
            >
              {t.label}
            </button>
          ))}
        </div>

        {mainTab === 'overview' && (
          <>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
          <div className="bg-gray-800 p-6 rounded-lg border border-gray-700">
            <h2 className="text-gray-400 text-sm uppercase tracking-wider mb-2">Block Height</h2>
            <p className="text-3xl font-mono text-white">{state.block_height}</p>
          </div>

          <div className="bg-gray-800 p-6 rounded-lg border border-gray-700">
            <h2 className="text-gray-400 text-sm uppercase tracking-wider mb-2">Mempool</h2>
            <p className="text-3xl font-mono text-amber-300">{state.mempool_size ?? 0}</p>
            <p className="text-xs text-gray-500 mt-1">ожидают следующий блок</p>
            <p className="text-xs text-gray-500 mt-2 leading-snug">
              Не в цепи: <span className="text-amber-200/90 font-mono">{state.mempool_size ?? 0}</span> tx — после
              приёма через API мемпул сразу пишется в state.json; после каждого блока — вместе с цепочкой.
            </p>
          </div>

          <div className="bg-gray-800 p-6 rounded-lg border border-gray-700">
            <h2 className="text-gray-400 text-sm uppercase tracking-wider mb-2">Интервал блока (симуляция)</h2>
            <p className="text-xs text-gray-500 mb-2">
              Порядок фаз блока — как в каноне (BeginBlock → DeliverTx → EndBlock). В основной цепи —{' '}
              <span className="text-gray-400">{state.canonical_block_interval_sec ?? 60} с</span> на блок; здесь можно
              задать другое значение для нагрузочных тестов (0.1–300 с), сохраняется в state.
            </p>
            <div className="flex items-center gap-2 flex-wrap">
              <input 
                type="number" 
                step="0.1" 
                min="0.1" 
                max="300"
                value={blockTimeInput}
                onChange={(e) => setBlockTimeInput(e.target.value)}
                className="bg-gray-700 text-white px-3 py-1 rounded w-24 border border-gray-600 focus:outline-none focus:border-blue-500"
              />
              <span className="text-gray-400">s</span>
              <button 
                onClick={handleUpdateBlockTime}
                className="ml-2 bg-blue-600 hover:bg-blue-700 text-white px-3 py-1 rounded text-sm transition-colors"
              >
                Set
              </button>
              {state.block_time > 0 ? (
                <span className="text-xs text-gray-500">активно: {state.block_time}s</span>
              ) : null}
            </div>
          </div>

          <div className="bg-gray-800 p-6 rounded-lg border border-gray-700">
            <h2 className="text-gray-400 text-sm uppercase tracking-wider mb-2">Total Accounts</h2>
            <p className="text-3xl font-mono text-white">{state.accounts_count}</p>
          </div>
        </div>

        <div className="mb-8">
          <h2 className="text-lg font-bold text-emerald-400/90 mb-3">Кошелёк / аккаунт</h2>
          <WalletPanel
            apiBase={API_BASE}
            address={selectedWallet}
            account={selectedAccount}
            blockHeight={state.block_height}
            simTreasury={state.sim_treasury}
            genesisValidator={state.genesis_validator}
            genesisProvider={state.genesis_provider}
          />
        </div>
          </>
        )}

        {mainTab === 'chain' && (
        <>
        <div className="bg-gray-800 p-6 rounded-lg border border-gray-700 mb-8">
          <h2 className="text-xl font-bold mb-4 text-blue-300">🔗 Blockchain & Consensus Explorer</h2>
          
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            {/* Live Block Tape */}
            <div className="lg:col-span-2">
              <h3 className="text-gray-400 text-sm uppercase tracking-wider mb-3">Live Block Tape</h3>
              <div className="flex gap-4 overflow-x-auto pb-4 snap-x">
                {state.blocks.length === 0 ? (
                  <p className="text-gray-500">Waiting for blocks...</p>
                ) : (
                  state.blocks.slice().reverse().map(b => (
                    <div 
                      key={b.height} 
                      onClick={() => setSelectedBlockHeight(b.height === selectedBlockHeight ? null : b.height)}
                      className={`min-w-[200px] bg-gray-900 p-4 rounded border flex flex-col gap-2 snap-start shrink-0 shadow-lg relative overflow-hidden cursor-pointer transition-colors ${b.height === selectedBlockHeight ? 'border-blue-500' : 'border-gray-700 hover:border-gray-500'}`}
                    >
                      <div className="absolute top-0 left-0 w-1 h-full bg-blue-500"></div>
                      <div className="flex justify-between items-center pl-2">
                        <span className="text-blue-400 font-bold text-lg">#{b.height}</span>
                        <span className="text-xs text-gray-400">{new Date(b.timestamp * 1000).toLocaleTimeString()}</span>
                      </div>
                      <div className="text-xs font-mono text-gray-400 truncate pl-2" title={b.hash}>
                        Hash: {b.hash}
                      </div>
                      <div className="text-sm font-semibold text-green-400 pl-2 mt-1">
                        {b.tx_count} TXs
                      </div>
                    </div>
                  ))
                )}
              </div>
            </div>

            {/* TPS Chart */}
            <div className="lg:col-span-1">
              <h3 className="text-gray-400 text-sm uppercase tracking-wider mb-3">Network Load (TPS)</h3>
              <div className="h-32 min-h-32 min-w-0 w-full bg-gray-900 rounded p-2 border border-gray-700">
                {state.tps_history && state.tps_history.length > 0 ? (
                  <ResponsiveContainer width="100%" height="100%" minHeight={120}>
                    <LineChart data={state.tps_history}>
                      <CartesianGrid strokeDasharray="3 3" stroke="#374151" vertical={false} />
                      <XAxis dataKey="time" hide />
                      <YAxis stroke="#9CA3AF" tick={{fontSize: 10}} width={30} />
                      <Tooltip 
                        contentStyle={{backgroundColor: '#1F2937', border: '1px solid #374151', color: '#fff', fontSize: '12px'}}
                        itemStyle={{color: '#60A5FA'}}
                      />
                      <Line type="monotone" dataKey="tps" stroke="#60A5FA" strokeWidth={2} dot={false} isAnimationActive={false} />
                    </LineChart>
                  </ResponsiveContainer>
                ) : (
                  <div className="h-full flex items-center justify-center text-gray-500 text-sm">No data</div>
                )}
              </div>
              <div className="mt-2 text-right">
                <span className="text-xs text-gray-400">Current: </span>
                <span className="text-sm font-bold text-blue-400">
                  {state.tps_history.length > 0 ? state.tps_history[state.tps_history.length - 1].tps.toFixed(2) : '0.00'} tx/s
                </span>
              </div>
            </div>
          </div>

          {/* Selected Block Details — полное содержимое блока (JSON) */}
          {selectedBlock && (
            <div className="mt-6 bg-gray-900 p-4 rounded border border-blue-500/50">
              <div className="flex flex-wrap justify-between items-center gap-2 mb-3">
                <h3 className="text-lg font-bold text-blue-300">Block #{selectedBlock.height}</h3>
                <button
                  type="button"
                  onClick={() => setSelectedBlockHeight(null)}
                  className="text-gray-500 hover:text-white"
                >
                  ✕
                </button>
              </div>
              <dl className="grid grid-cols-1 sm:grid-cols-2 gap-2 text-sm mb-4 border-b border-gray-700/80 pb-4">
                <div>
                  <dt className="text-gray-500 text-xs uppercase">Hash</dt>
                  <dd className="font-mono text-xs text-gray-200 break-all">{selectedBlock.hash}</dd>
                </div>
                <div>
                  <dt className="text-gray-500 text-xs uppercase">Time (UTC)</dt>
                  <dd className="font-mono text-gray-300">
                    {new Date(selectedBlock.timestamp * 1000).toISOString()}
                  </dd>
                </div>
                <div>
                  <dt className="text-gray-500 text-xs uppercase">tx_count</dt>
                  <dd className="font-mono text-gray-300">{selectedBlock.tx_count}</dd>
                </div>
                <div>
                  <dt className="text-gray-500 text-xs uppercase">transactions.length</dt>
                  <dd className="font-mono text-gray-300">{selectedBlock.transactions.length}</dd>
                </div>
              </dl>
              <p className="text-xs text-gray-500 mb-2">Полное тело блока (все поля каждой записи в ленте):</p>
              <div className="max-h-[min(70vh,720px)] overflow-auto rounded border border-gray-700 bg-black/40 p-3">
                <pre className="text-xs font-mono text-gray-300 whitespace-pre-wrap break-words">
                  {JSON.stringify(selectedBlock, null, 2)}
                </pre>
              </div>
            </div>
          )}
        </div>

        <div className="bg-gray-800 p-6 rounded-lg border border-gray-700 mb-8">
          <h2 className="text-xl font-bold mb-4">Recent Transactions (Last 10)</h2>
          {state.recent_txs.length === 0 ? (
            <p className="text-gray-500">No recent transactions in blocks.</p>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-left">
                <thead>
                  <tr className="border-b border-gray-700 text-gray-400">
                    <th className="pb-3 font-medium">Type</th>
                    <th className="pb-3 font-medium">Sender</th>
                    <th className="pb-3 font-medium">Receiver</th>
                    <th className="pb-3 font-medium">Amount</th>
                  </tr>
                </thead>
                <tbody>
                  {state.recent_txs.map((tx, idx) => (
                    <tr key={idx} className="border-b border-gray-700/50">
                      <td className="py-2">
                        <span className="bg-blue-900/50 text-blue-300 px-2 py-1 rounded text-xs uppercase">
                          {tx.tx_type}
                        </span>
                      </td>
                      <td className="py-2 font-mono text-xs text-gray-400">{tx.sender || '-'}</td>
                      <td className="py-2 font-mono text-xs text-gray-400">{tx.receiver || '-'}</td>
                      <td className="py-2 text-green-400 font-mono">
                        {tx.amount ? `${tx.amount.toFixed(2)}` : '-'}
                        {tx.asset_type ? ` ${tx.asset_type.toUpperCase()}` : ''}
                        {tx.tx_type === 'epoch_emission' ? ` (Epoch)` : ''}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
          </>
        )}

        {mainTab === 'sim' && (
        <>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-8">
          <div className="bg-gray-800 p-6 rounded-lg border border-gray-700">
            <h2 className="text-xl font-bold mb-4 text-purple-400">Панель оператора симуляции</h2>
            <p className="text-xs text-gray-500 mb-3">
              Mint из казначейства и смена роли через панель оператора тоже идут в мемпул и исполняются в блоке (как на узле).
              Прямое изменение балансов снято. Эпохальная эмиссия ANT — только системные tx в блоке границы эпохи (§5.5: на кошельки Поставщиков).
              Цепочка и счета сохраняются на диск автоматически после каждого блока; при остановке сервера — ещё раз при завершении.
              Мемпул сериализуется в state.json (лимит 1000 tx) и поднимается при старте; tx бота до следующего блока на диск не сбрасываются отдельно.
            </p>
            <p className="text-xs text-gray-500 mb-3">
              Перед сбросом цепочки текущий <span className="font-mono text-gray-400">state.json</span> копируется в{' '}
              <span className="font-mono text-gray-400">state.json.bak</span> (если файл был).
            </p>
            <div className="flex flex-wrap gap-4">
              <button 
                onClick={handleCreateAccounts}
                className="bg-purple-600 hover:bg-purple-700 text-white px-4 py-2 rounded transition-colors"
              >
                + Generate 5 Accounts
              </button>
              <button 
                onClick={handleResetState}
                className="bg-red-600 hover:bg-red-700 text-white px-4 py-2 rounded transition-colors"
              >
                🗑️ Reset Blockchain
              </button>
            </div>
          </div>

          <div className="bg-gray-800 p-6 rounded-lg border border-gray-700">
            <h2 className="text-xl font-bold mb-4 text-orange-400">🤖 Bot Engine (Traffic Generator)</h2>
            <div className="flex flex-col gap-4">
              <div className="flex items-center gap-4">
                <span className="text-gray-300">Status:</span>
                <span className={`px-2 py-1 rounded text-xs font-bold uppercase ${botStatus.is_running ? 'bg-green-900/50 text-green-400' : 'bg-gray-700 text-gray-400'}`}>
                  {botStatus.is_running ? 'Running' : 'Stopped'}
                </span>
              </div>
              <div className="flex items-center gap-2">
                <span className="text-gray-300">Intensity (tx/s):</span>
                <input 
                  type="number" 
                  step="0.1" 
                  min="0.1" 
                  max="100"
                  value={botIntensityInput}
                  onChange={(e) => setBotIntensityInput(e.target.value)}
                  className="bg-gray-700 text-white px-3 py-1 rounded w-24 border border-gray-600 focus:outline-none focus:border-orange-500"
                  disabled={botStatus.is_running}
                />
              </div>
              <div className="flex gap-2 mt-2">
                {!botStatus.is_running ? (
                  <button 
                    onClick={() => handleBotControl('start')}
                    className="bg-orange-600 hover:bg-orange-700 text-white px-4 py-2 rounded transition-colors w-full"
                  >
                    ▶️ Start Traffic
                  </button>
                ) : (
                  <button 
                    onClick={() => handleBotControl('stop')}
                    className="bg-gray-600 hover:bg-gray-500 text-white px-4 py-2 rounded transition-colors w-full"
                  >
                    ⏹️ Stop Traffic
                  </button>
                )}
              </div>
              <p className="text-xs text-gray-500 mt-1">
                Бот ставит tx в мемпул и логирует намерение; в блоке узел проверяет канон (docs/volnix_protocol.md). Есть
                редкие <span className="text-amber-600/90">canon_probe</span> — ожидаемые отклонения (§4.1–4.2).
              </p>
            </div>
          </div>
        </div>

        {/* Канон-аудит / логи проверок */}
        <div className="bg-gray-800 p-6 rounded-lg border border-cyan-800/40 mb-8">
          <h2 className="text-xl font-bold mb-2 text-cyan-300">📜 Канон-аудит симуляции</h2>
          <p className="text-xs text-gray-500 mb-4">
            Записи: отклонения до мемпула (кошелёк / панель оператора), намерения бота, пост-разбор ленты блока (переводы §4.1, рынок
            §5.2, declare §5.4, эпоха §5.5). Статус <span className="text-emerald-400">ok</span> — соответствует правилам;
            <span className="text-red-400/90"> reject</span> — tx не прошла или отклонена намеренно при тесте;
            <span className="text-amber-400/90"> warn</span> — внимание.
          </p>
          <div className="max-h-[420px] overflow-y-auto rounded border border-gray-700 bg-gray-950/80">
            {(state.canon_log ?? []).length === 0 ? (
              <p className="p-4 text-gray-600 text-sm">Пока нет записей — дождитесь блока или включите бота.</p>
            ) : (
              <ul className="divide-y divide-gray-800">
                {(state.canon_log ?? []).map((e) => (
                  <li key={e.id} className="px-3 py-2.5 text-sm hover:bg-gray-900/50">
                    <div className="flex flex-wrap items-baseline gap-2 gap-y-1">
                      <span
                        className={`text-[10px] font-bold uppercase px-1.5 py-0.5 rounded ${
                          e.status === 'ok'
                            ? 'bg-emerald-900/50 text-emerald-300'
                            : e.status === 'reject'
                              ? 'bg-red-900/40 text-red-300'
                              : e.status === 'warn'
                                ? 'bg-amber-900/40 text-amber-200'
                                : 'bg-slate-700 text-slate-300'
                        }`}
                      >
                        {e.status}
                      </span>
                      <span className="text-[10px] text-gray-500">{e.source}</span>
                      <span className="text-[10px] text-cyan-700/90 font-mono">{e.canon}</span>
                      <span className="text-[10px] text-gray-600">
                        {e.block_height != null ? `h=${e.block_height}` : ''}{' '}
                        {e.tx_hash ? `· ${e.tx_hash.slice(0, 10)}…` : ''}
                      </span>
                    </div>
                    <div className="text-gray-200 mt-1">{e.title}</div>
                    {e.detail ? <div className="text-xs text-gray-500 mt-0.5 break-words">{e.detail}</div> : null}
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>
        </>
        )}

        {mainTab === 'market' && (
        <div className="bg-gray-800 p-6 rounded-lg border border-gray-700 mb-8">
          <div className="flex justify-between items-center mb-4">
            <h2 className="text-xl font-bold text-blue-300">📈 Anteil Market</h2>
            <div className="bg-gray-900 px-4 py-2 rounded border border-gray-700">
              <span className="text-gray-400 text-sm">Last Price: </span>
              <span className="text-xl font-mono text-green-400">{state.market?.last_price.toFixed(2)} WRT</span>
            </div>
          </div>
          
          {/* Виджет графика (стиль TradingView + Lightweight Charts, данные симуляции) */}
          <div className="min-w-0 w-full mb-8">
            <TradingViewMarketWidget
              history={state.market?.history ?? []}
              lastPrice={state.market?.last_price ?? 0}
              height={360}
            />
          </div>

          {/* Manual Order Form */}
          <div className="bg-gray-900/50 p-4 rounded border border-gray-700/50 mb-8">
            <h3 className="text-lg font-semibold text-gray-300 mb-3">Быстрый ордер (тот же API кошелька)</h3>
            <form onSubmit={handleCreateOrder} className="flex flex-wrap gap-4 items-end">
              <div className="flex flex-col gap-1">
                <label className="text-xs text-gray-400">Account</label>
                <select 
                  className="bg-gray-700 text-white border border-gray-600 rounded px-3 py-2 text-sm focus:outline-none w-48"
                  value={orderAccount}
                  onChange={(e) => setOrderAccount(e.target.value)}
                  required
                >
                  <option value="" disabled>Select Account</option>
                  {Object.values(state.accounts)
                    .filter((a) => (orderType === 'buy' ? a.role === 'validator' : a.role === 'provider'))
                    .map((acc) => (
                    <option key={acc.address} value={acc.address}>{acc.address.substring(0, 12)}... ({acc.wrt_balance?.toFixed(0)} WRT / {acc.ant_balance?.toFixed(0)} ANT)</option>
                  ))}
                </select>
              </div>
              
              <div className="flex flex-col gap-1">
                <label className="text-xs text-gray-400">Type</label>
                <select 
                  className="bg-gray-700 text-white border border-gray-600 rounded px-3 py-2 text-sm focus:outline-none"
                  value={orderType}
                  onChange={(e) => {
                    setOrderType(e.target.value)
                    setOrderAccount('')
                  }}
                >
                  <option value="buy">Buy ANT (Validator)</option>
                  <option value="sell">Sell ANT (Provider)</option>
                </select>
              </div>

              <div className="flex flex-col gap-1">
                <label className="text-xs text-gray-400 flex items-center gap-2">
                  <input
                    type="checkbox"
                    checked={orderMarket}
                    onChange={(e) => setOrderMarket(e.target.checked)}
                    className="rounded"
                  />
                  Market (IOC по книге)
                </label>
              </div>

              {!orderMarket && (
                <div className="flex flex-col gap-1">
                  <label className="text-xs text-gray-400">Price (WRT per ANT)</label>
                  <input
                    type="number"
                    step="0.1"
                    min="0.1"
                    required={!orderMarket}
                    value={orderPrice}
                    onChange={(e) => setOrderPrice(e.target.value)}
                    className="bg-gray-700 text-white border border-gray-600 rounded px-3 py-2 text-sm focus:outline-none w-32"
                  />
                </div>
              )}

              {orderMarket && orderType === 'buy' && (
                <div className="flex flex-col gap-1">
                  <label className="text-xs text-gray-400">Max WRT (optional)</label>
                  <input
                    type="number"
                    step="0.1"
                    min="0.1"
                    placeholder="All WRT if empty"
                    value={orderMaxWrt}
                    onChange={(e) => setOrderMaxWrt(e.target.value)}
                    className="bg-gray-700 text-white border border-gray-600 rounded px-3 py-2 text-sm focus:outline-none w-36"
                  />
                </div>
              )}

              <div className="flex flex-col gap-1">
                <label className="text-xs text-gray-400">
                  {orderMarket && orderType === 'buy' ? 'Max ANT to buy' : 'Amount (ANT)'}
                </label>
                <input
                  type="number"
                  step="0.1"
                  min="0.1"
                  required
                  value={orderAmount}
                  onChange={(e) => setOrderAmount(e.target.value)}
                  className="bg-gray-700 text-white border border-gray-600 rounded px-3 py-2 text-sm focus:outline-none w-24"
                />
              </div>

              <button type="submit" className="bg-blue-600 hover:bg-blue-700 text-white px-6 py-2 rounded text-sm font-semibold transition-colors h-[38px]">
                Submit Order
              </button>
            </form>
          </div>
          
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            {/* Bids (Buy Orders) */}
            <div>
              <h3 className="text-green-400 font-semibold mb-2 border-b border-green-900/50 pb-2">Bids (Buy)</h3>
              <div className="overflow-hidden">
                <table className="w-full text-left text-sm">
                  <thead>
                    <tr className="text-gray-500">
                      <th className="pb-2">Price</th>
                      <th className="pb-2">Amount</th>
                      <th className="pb-2">Filled</th>
                    </tr>
                  </thead>
                  <tbody>
                    {state.market?.bids.map((order) => (
                      <tr key={order.id} className="border-t border-gray-700/30">
                        <td className="py-1 text-green-400 font-mono">{order.price.toFixed(2)}</td>
                        <td className="py-1 font-mono">{order.amount.toFixed(2)}</td>
                        <td className="py-1 text-gray-400 font-mono">{(order.filled / order.amount * 100).toFixed(0)}%</td>
                      </tr>
                    ))}
                    {(!state.market?.bids || state.market.bids.length === 0) && (
                      <tr><td colSpan={3} className="py-4 text-center text-gray-600">No buy orders</td></tr>
                    )}
                  </tbody>
                </table>
              </div>
            </div>

            {/* Asks (Sell Orders) */}
            <div>
              <h3 className="text-red-400 font-semibold mb-2 border-b border-red-900/50 pb-2">Asks (Sell)</h3>
              <div className="overflow-hidden">
                <table className="w-full text-left text-sm">
                  <thead>
                    <tr className="text-gray-500">
                      <th className="pb-2">Price</th>
                      <th className="pb-2">Amount</th>
                      <th className="pb-2">Filled</th>
                    </tr>
                  </thead>
                  <tbody>
                    {state.market?.asks.map((order) => (
                      <tr key={order.id} className="border-t border-gray-700/30">
                        <td className="py-1 text-red-400 font-mono">{order.price.toFixed(2)}</td>
                        <td className="py-1 font-mono">{order.amount.toFixed(2)}</td>
                        <td className="py-1 text-gray-400 font-mono">{(order.filled / order.amount * 100).toFixed(0)}%</td>
                      </tr>
                    ))}
                    {(!state.market?.asks || state.market.asks.length === 0) && (
                      <tr><td colSpan={3} className="py-4 text-center text-gray-600">No sell orders</td></tr>
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        </div>
        )}

        {mainTab === 'accounts' && (
        <div className="bg-gray-800 p-6 rounded-lg border border-gray-700 mb-8">
          <h2 className="text-xl font-bold mb-4">Accounts</h2>
          {Object.keys(state.accounts).length === 0 ? (
            <p className="text-gray-500">Нет аккаунтов — создайте через вкладку «Симуляция» (панель оператора).</p>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-left">
                <thead>
                  <tr className="border-b border-gray-700 text-gray-400">
                    <th className="pb-3 font-medium">Address</th>
                    <th className="pb-3 font-medium">WRT Balance</th>
                    <th className="pb-3 font-medium">LZN (liq. / frozen)</th>
                    <th className="pb-3 font-medium">ANT Balance</th>
                    <th className="pb-3 font-medium">ZKP</th>
                    <th className="pb-3 font-medium">Role</th>
                    <th className="pb-3 font-medium">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {Object.values(state.accounts).map((acc) => (
                    <tr key={acc.address} className="border-b border-gray-700/50">
                      <td className="py-3 font-mono text-sm text-blue-300">{acc.address}</td>
                      <td className="py-3 text-green-400">{acc.wrt_balance?.toFixed(2) || '0.00'}</td>
                      <td className="py-3 text-purple-400 text-sm">
                        {(acc.lzn_balance ?? 0).toFixed(2)}
                        <span className="text-gray-500"> / </span>
                        <span className="text-amber-400/90" title="LZN frozen for mining">{(acc.lzn_frozen_mining ?? 0).toFixed(0)}</span>
                      </td>
                      <td className="py-3 text-orange-400">{acc.ant_balance?.toFixed(2) || '0.00'}</td>
                      <td className="py-3 text-center text-lg" title="ZKP (симуляция): verify_zkp из кошелька">
                        {acc.zkp_verified ? (
                          <span className="text-emerald-400">✓</span>
                        ) : (
                          <span className="text-gray-600">—</span>
                        )}
                      </td>
                      <td className="py-3">
                        <span className={`px-2 py-1 rounded text-xs uppercase tracking-wider ${
                          acc.role === 'guest' || acc.role === 'citizen' ? 'bg-blue-900/50 text-blue-300' :
                          acc.role === 'provider' ? 'bg-orange-900/50 text-orange-300' :
                          'bg-green-900/50 text-green-300'
                        }`}>
                          {acc.role}
                        </span>
                      </td>
                      <td className="py-3">
                        <div className="flex flex-wrap gap-2">
                          <button
                            type="button"
                            onClick={() => setSelectedWallet(acc.address)}
                            className={`px-2 py-1 rounded text-xs transition-colors ${
                              selectedWallet === acc.address
                                ? 'bg-emerald-800 text-white'
                                : 'bg-gray-700 hover:bg-gray-600'
                            }`}
                          >
                            Кошелёк
                          </button>
                          <button onClick={() => handleMint(acc.address, 'wrt')} className="bg-gray-700 hover:bg-gray-600 px-2 py-1 rounded text-xs transition-colors">+1000 WRT</button>
                          <button onClick={() => handleMint(acc.address, 'lzn')} className="bg-gray-700 hover:bg-gray-600 px-2 py-1 rounded text-xs transition-colors">+100 LZN</button>
                          <button onClick={() => handleMint(acc.address, 'ant')} className="bg-gray-700 hover:bg-gray-600 px-2 py-1 rounded text-xs transition-colors">+100 ANT</button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
        )}

      </div>
    </div>
  )
}

export default App
