import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { Layout } from './components/Layout'
import { OverviewPage } from './pages/OverviewPage'
import { BlocksPage } from './pages/BlocksPage'
import { BlockDetailPage } from './pages/BlockDetailPage'
import { TxDetailPage } from './pages/TxDetailPage'
import { AccountPage } from './pages/AccountPage'
import { WalletsPage } from './pages/WalletsPage'
import { ValidatorsPage } from './pages/ValidatorsPage'
import { MarketPage } from './pages/MarketPage'
import { EpochsPage } from './pages/EpochsPage'
import { OperatorPage } from './pages/OperatorPage'

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<Layout />}>
          <Route index element={<OverviewPage />} />
          <Route path="blocks" element={<BlocksPage />} />
          <Route path="blocks/:height" element={<BlockDetailPage />} />
          <Route path="txs/:hash" element={<TxDetailPage />} />
          <Route path="wallets" element={<WalletsPage />} />
          <Route path="accounts/:address" element={<AccountPage />} />
          <Route path="validators" element={<ValidatorsPage />} />
          <Route path="market" element={<MarketPage />} />
          <Route path="epochs" element={<EpochsPage />} />
          <Route path="operator" element={<OperatorPage />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}
