import { useEffect, useState } from 'react'
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts'

interface Account {
  address: string;
  wrt_balance: number;
  lzn_balance: number;
  ant_balance: number;
  role: string;
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
  history: {time: string, price: number}[];
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
}

interface Block {
  height: number;
  hash: string;
  tx_count: number;
  timestamp: number;
  transactions: Transaction[];
}

interface NetworkState {
  status: string;
  block_height: number;
  block_time: number;
  accounts_count: number;
  accounts: Record<string, Account>;
  recent_txs: Transaction[];
  market: Market;
  blocks: Block[];
  tps_history: {time: string, tps: number}[];
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
    tps_history: []
  })
  
  const [blockTimeInput, setBlockTimeInput] = useState<string>("5.0")
  const [botStatus, setBotStatus] = useState({ is_running: false, intensity: 1.0 })
  const [botIntensityInput, setBotIntensityInput] = useState<string>("1.0")
  const [selectedBlockHeight, setSelectedBlockHeight] = useState<number | null>(null)

  const selectedBlock = state.blocks.find(b => b.height === selectedBlockHeight)

  // Manual Order Form State
  const [orderAccount, setOrderAccount] = useState<string>("")
  const [orderType, setOrderType] = useState<string>("buy")
  const [orderPrice, setOrderPrice] = useState<string>("10.0")
  const [orderAmount, setOrderAmount] = useState<string>("5.0")

  useEffect(() => {
    // Fetch initial bot status
    fetch('http://localhost:8000/api/bot/status')
      .then(res => res.json())
      .then(data => {
        setBotStatus(data)
        setBotIntensityInput(data.intensity.toString())
      })

    // Connect to WebSocket
    const ws = new WebSocket('ws://localhost:8000/ws')

    ws.onopen = () => {
      setState(prev => ({ ...prev, status: 'Connected (Live)' }))
    }

    ws.onmessage = (event) => {
      const msg = JSON.parse(event.data)
      if (msg.type === 'init' || msg.type === 'new_block') {
        const txs = msg.data.block?.transactions || []
        setState(prev => ({
          ...prev,
          block_height: msg.data.state.height,
          block_time: msg.data.block_time,
          accounts_count: msg.data.state.accounts_count,
          accounts: msg.data.state.accounts,
          market: msg.data.state.market,
          blocks: msg.data.state.blocks,
          tps_history: msg.data.state.tps_history,
          recent_txs: [...txs, ...prev.recent_txs].slice(0, 10) // Keep last 10 txs
        }))
        // Only set the block time input on initial load, not on every block
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
    const intensity = parseFloat(botIntensityInput)
    await fetch('http://localhost:8000/api/bot/control', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ action, intensity })
    })
    setBotStatus({ is_running: action === 'start', intensity })
  }

  const handleCreateOrder = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!orderAccount) {
      alert("Please select an account")
      return
    }
    await fetch('http://localhost:8000/api/god-mode/order', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        address: orderAccount,
        order_type: orderType,
        price: parseFloat(orderPrice),
        amount: parseFloat(orderAmount)
      })
    })
    alert("Order added to mempool! Will be processed in the next block.")
  }

  const handleCreateAccounts = async () => {
    await fetch('http://localhost:8000/api/god-mode/accounts', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ count: 5 })
    })
  }

  const handleSaveState = async () => {
    await fetch('http://localhost:8000/api/god-mode/save', { method: 'POST' })
    alert('State saved to disk!')
  }

  const handleResetState = async () => {
    if (confirm('Are you sure you want to completely reset and delete the blockchain state?')) {
      await fetch('http://localhost:8000/api/god-mode/reset', { method: 'POST' })
    }
  }

  const handleUpdateBlockTime = async () => {
    const time = parseFloat(blockTimeInput)
    if (!isNaN(time) && time >= 0.1 && time <= 300) {
      await fetch('http://localhost:8000/api/god-mode/block-time', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ time_sec: time })
      })
      // Update local state to reflect the change immediately
      setState(prev => ({ ...prev, block_time: time }))
    } else {
      alert("Block time must be between 0.1 and 300 seconds.")
    }
  }

  const handleMint = async (address: string, asset_type: string = "wrt") => {
    const res = await fetch('http://localhost:8000/api/god-mode/mint', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ address, amount: 1000, asset_type })
    })
    const data = await res.json()
    if (data.status === "error") {
      alert(data.message)
    }
  }

  const handleSetRole = async (address: string, role: string) => {
    await fetch('http://localhost:8000/api/god-mode/role', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ address, role })
    })
  }

  return (
    <div className="min-h-screen bg-gray-900 text-white p-8">
      <div className="max-w-6xl mx-auto">
        <div className="flex justify-between items-center mb-8">
          <h1 className="text-4xl font-bold text-blue-400">Volnix Protocol Simulation</h1>
          <div className={`px-4 py-2 rounded-full font-semibold ${state.status.includes('Live') ? 'bg-green-500/20 text-green-400' : 'bg-red-500/20 text-red-400'}`}>
            {state.status}
          </div>
        </div>
        
        {/* Network Stats */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
          <div className="bg-gray-800 p-6 rounded-lg border border-gray-700">
            <h2 className="text-gray-400 text-sm uppercase tracking-wider mb-2">Block Height</h2>
            <p className="text-3xl font-mono text-white">{state.block_height}</p>
          </div>

          <div className="bg-gray-800 p-6 rounded-lg border border-gray-700">
            <h2 className="text-gray-400 text-sm uppercase tracking-wider mb-2">Block Time</h2>
            <div className="flex items-center gap-2">
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
            </div>
          </div>

          <div className="bg-gray-800 p-6 rounded-lg border border-gray-700">
            <h2 className="text-gray-400 text-sm uppercase tracking-wider mb-2">Total Accounts</h2>
            <p className="text-3xl font-mono text-white">{state.accounts_count}</p>
          </div>
        </div>

        {/* Blockchain & Consensus Explorer */}
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
              <div className="h-32 w-full bg-gray-900 rounded p-2 border border-gray-700">
                {state.tps_history && state.tps_history.length > 0 ? (
                  <ResponsiveContainer width="100%" height="100%">
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

          {/* Selected Block Details */}
          {selectedBlock && (
            <div className="mt-6 bg-gray-900 p-4 rounded border border-blue-500/50">
              <div className="flex justify-between items-center mb-4">
                <h3 className="text-lg font-bold text-blue-300">Block #{selectedBlock.height} Details</h3>
                <button onClick={() => setSelectedBlockHeight(null)} className="text-gray-500 hover:text-white">✕</button>
              </div>
              {selectedBlock.transactions.length === 0 ? (
                <p className="text-gray-500">Empty block.</p>
              ) : (
                <div className="overflow-x-auto max-h-96 overflow-y-auto">
                  <table className="w-full text-left text-sm">
                    <thead className="sticky top-0 bg-gray-900">
                      <tr className="border-b border-gray-700 text-gray-400">
                        <th className="pb-2">Tx Hash</th>
                        <th className="pb-2">Type</th>
                        <th className="pb-2">Sender</th>
                        <th className="pb-2">Receiver</th>
                        <th className="pb-2">Details</th>
                      </tr>
                    </thead>
                    <tbody>
                      {selectedBlock.transactions.map((tx, idx) => (
                        <tr key={idx} className="border-b border-gray-800">
                          <td className="py-2 font-mono text-xs text-gray-500" title={tx.tx_hash}>{tx.tx_hash.substring(0, 8)}...</td>
                          <td className="py-2">
                            <span className="bg-blue-900/50 text-blue-300 px-2 py-1 rounded text-xs uppercase">{tx.tx_type}</span>
                          </td>
                          <td className="py-2 font-mono text-xs text-gray-400">{tx.sender ? tx.sender.substring(0, 12) + '...' : '-'}</td>
                          <td className="py-2 font-mono text-xs text-gray-400">{tx.receiver ? tx.receiver.substring(0, 12) + '...' : '-'}</td>
                          <td className="py-2 text-green-400 font-mono text-xs">
                            {tx.amount ? `${tx.amount.toFixed(2)}` : ''}
                            {tx.asset_type ? ` ${tx.asset_type.toUpperCase()}` : ''}
                            {tx.role ? ` Role:${tx.role}` : ''}
                            {tx.price ? ` @ ${tx.price.toFixed(2)} WRT` : ''}
                            {tx.tx_type === 'epoch_emission' ? ` (Epoch)` : ''}
                            {tx.details ? ` - ${tx.details}` : ''}
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

        {/* God Mode Panel */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-8">
          <div className="bg-gray-800 p-6 rounded-lg border border-gray-700">
            <h2 className="text-xl font-bold mb-4 text-purple-400">⚡ God Mode Control Panel</h2>
            <div className="flex flex-wrap gap-4">
              <button 
                onClick={handleCreateAccounts}
                className="bg-purple-600 hover:bg-purple-700 text-white px-4 py-2 rounded transition-colors"
              >
                + Generate 5 Accounts
              </button>
              <button 
                onClick={handleSaveState}
                className="bg-green-600 hover:bg-green-700 text-white px-4 py-2 rounded transition-colors"
              >
                💾 Save State
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
                * Bots only send tokens if they have a role (Citizen/Provider/Validator) and balance &gt; 0.
              </p>
            </div>
          </div>
        </div>

        {/* Market / Orderbook */}
        <div className="bg-gray-800 p-6 rounded-lg border border-gray-700 mb-8">
          <div className="flex justify-between items-center mb-4">
            <h2 className="text-xl font-bold text-blue-300">📈 Anteil Market</h2>
            <div className="bg-gray-900 px-4 py-2 rounded border border-gray-700">
              <span className="text-gray-400 text-sm">Last Price: </span>
              <span className="text-xl font-mono text-green-400">{state.market?.last_price.toFixed(2)} WRT</span>
            </div>
          </div>
          
          {/* Chart */}
          <div className="h-64 w-full mb-8 bg-gray-900/50 rounded p-4 border border-gray-700/50">
            {state.market?.history && state.market.history.length > 0 ? (
              <ResponsiveContainer width="100%" height="100%">
                <LineChart data={state.market.history}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#374151" />
                  <XAxis dataKey="time" stroke="#9CA3AF" tick={{fontSize: 12}} />
                  <YAxis stroke="#9CA3AF" tick={{fontSize: 12}} domain={['auto', 'auto']} />
                  <Tooltip 
                    contentStyle={{backgroundColor: '#1F2937', border: '1px solid #374151', color: '#fff'}}
                    itemStyle={{color: '#4ADE80'}}
                  />
                  <Line type="monotone" dataKey="price" stroke="#4ADE80" strokeWidth={2} dot={false} isAnimationActive={false} />
                </LineChart>
              </ResponsiveContainer>
            ) : (
              <div className="h-full flex items-center justify-center text-gray-500">No price history yet. Start trading!</div>
            )}
          </div>

          {/* Manual Order Form */}
          <div className="bg-gray-900/50 p-4 rounded border border-gray-700/50 mb-8">
            <h3 className="text-lg font-semibold text-gray-300 mb-3">Manual Order Creation</h3>
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
                  {Object.values(state.accounts).filter(a => a.role !== 'guest').map(acc => (
                    <option key={acc.address} value={acc.address}>{acc.address.substring(0, 12)}... ({acc.wrt_balance?.toFixed(0)} WRT / {acc.ant_balance?.toFixed(0)} ANT)</option>
                  ))}
                </select>
              </div>
              
              <div className="flex flex-col gap-1">
                <label className="text-xs text-gray-400">Type</label>
                <select 
                  className="bg-gray-700 text-white border border-gray-600 rounded px-3 py-2 text-sm focus:outline-none"
                  value={orderType}
                  onChange={(e) => setOrderType(e.target.value)}
                >
                  <option value="buy">Buy ANT for WRT</option>
                  <option value="sell">Sell ANT for WRT</option>
                </select>
              </div>

              <div className="flex flex-col gap-1">
                <label className="text-xs text-gray-400">Price (WRT per ANT)</label>
                <input 
                  type="number" step="0.1" min="0.1" required
                  value={orderPrice}
                  onChange={(e) => setOrderPrice(e.target.value)}
                  className="bg-gray-700 text-white border border-gray-600 rounded px-3 py-2 text-sm focus:outline-none w-32"
                />
              </div>

              <div className="flex flex-col gap-1">
                <label className="text-xs text-gray-400">Amount (ANT)</label>
                <input 
                  type="number" step="0.1" min="0.1" required
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

        {/* Recent Transactions */}
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

        {/* Accounts List */}
        <div className="bg-gray-800 p-6 rounded-lg border border-gray-700">
          <h2 className="text-xl font-bold mb-4">Accounts</h2>
          {Object.keys(state.accounts).length === 0 ? (
            <p className="text-gray-500">No accounts yet. Use God Mode to generate some.</p>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-left">
                <thead>
                  <tr className="border-b border-gray-700 text-gray-400">
                    <th className="pb-3 font-medium">Address</th>
                    <th className="pb-3 font-medium">WRT Balance</th>
                    <th className="pb-3 font-medium">LZN Balance</th>
                    <th className="pb-3 font-medium">ANT Balance</th>
                    <th className="pb-3 font-medium">Role</th>
                    <th className="pb-3 font-medium">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {Object.values(state.accounts).map((acc) => (
                    <tr key={acc.address} className="border-b border-gray-700/50">
                      <td className="py-3 font-mono text-sm text-blue-300">{acc.address}</td>
                      <td className="py-3 text-green-400">{acc.wrt_balance?.toFixed(2) || '0.00'}</td>
                      <td className="py-3 text-purple-400">{acc.lzn_balance?.toFixed(2) || '0.00'}</td>
                      <td className="py-3 text-orange-400">{acc.ant_balance?.toFixed(2) || '0.00'}</td>
                      <td className="py-3">
                        <span className={`px-2 py-1 rounded text-xs uppercase tracking-wider ${
                          acc.role === 'guest' ? 'bg-gray-700 text-gray-300' :
                          acc.role === 'citizen' ? 'bg-blue-900/50 text-blue-300' :
                          acc.role === 'provider' ? 'bg-orange-900/50 text-orange-300' :
                          'bg-green-900/50 text-green-300'
                        }`}>
                          {acc.role}
                        </span>
                      </td>
                      <td className="py-3">
                        <div className="flex gap-2">
                          <button onClick={() => handleMint(acc.address, 'wrt')} className="bg-gray-700 hover:bg-gray-600 px-2 py-1 rounded text-xs transition-colors">+1000 WRT</button>
                          <button onClick={() => handleMint(acc.address, 'lzn')} className="bg-gray-700 hover:bg-gray-600 px-2 py-1 rounded text-xs transition-colors">+100 LZN</button>
                          <button onClick={() => handleMint(acc.address, 'ant')} className="bg-gray-700 hover:bg-gray-600 px-2 py-1 rounded text-xs transition-colors">+100 ANT</button>
                          <select 
                            className="bg-gray-700 border border-gray-600 rounded text-xs px-2 py-1 focus:outline-none"
                            value={acc.role}
                            onChange={(e) => handleSetRole(acc.address, e.target.value)}
                          >
                            <option value="guest">Guest</option>
                            <option value="citizen">Citizen</option>
                            <option value="provider">Provider</option>
                            <option value="validator">Validator</option>
                          </select>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>

      </div>
    </div>
  )
}

export default App
