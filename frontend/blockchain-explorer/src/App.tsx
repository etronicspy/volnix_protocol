import React, { useState, useMemo } from 'react';
import { TabType } from './types';
import { useNetworkData } from './hooks/useNetworkData';
import { NetworkOverview } from './components/NetworkOverview';
import { StatsGrid } from './components/StatsGrid';
import { Tabs } from './components/Tabs';
import { BlocksList } from './components/BlocksList';
import { BlockDetail } from './components/BlockDetail';
import { TransactionsList } from './components/TransactionsList';
import { ValidatorsList } from './components/ValidatorsList';
import { ConsensusInfo } from './components/ConsensusInfo';
import { ModulesStatus } from './components/ModulesStatus';
import './App.css';

function App() {
  const [activeTab, setActiveTab] = useState<TabType>('blocks');
  const [selectedBlockHeight, setSelectedBlockHeight] = useState<number | null>(null);
  const { status, blocks, transactions, totalTransactionCount, validators, consensusParams, restApiAvailable, loading, refreshData } = useNetworkData();

  // Calculate statistics
  const stats = useMemo(() => {
    const totalBlocks = status ? parseInt(status.sync_info.latest_block_height) : -1;
    const totalTransactions = totalTransactionCount;
    const activeValidators = validators.filter(v => v.status === 'VALIDATOR_STATUS_ACTIVE').length;
    const burnedAnt = validators.reduce((sum, v) => sum + parseInt(v.total_burn_amount || '0'), 0);
    const avgBlockTime = '-'; // TODO: Calculate from blocks
    const networkHealth = status ? '100%' : '-';

    return {
      totalBlocks,
      totalTransactions,
      activeValidators,
      burnedAnt,
      avgBlockTime,
      networkHealth
    };
  }, [status, transactions, validators]);

  const handleViewBlock = (height: number) => {
    setSelectedBlockHeight(height);
  };

  const handleTabChange = (tab: TabType) => {
    setActiveTab(tab);
  };

  return (
    <div className="container">
      <div className="header">
        <h1>🔍 Volnix Blockchain Explorer</h1>
        <p>Real-time network monitoring and blockchain analysis</p>
      </div>

      <NetworkOverview
        status={status}
        activeValidatorsCount={stats.activeValidators}
        restApiAvailable={restApiAvailable}
        rpcAvailable={!!status}
      />

      <StatsGrid {...stats} />

      <Tabs activeTab={activeTab} onTabChange={handleTabChange} />

      {activeTab === 'blocks' && (
        <div className="card">
          {selectedBlockHeight != null ? (
            <BlockDetail height={selectedBlockHeight} onBack={() => setSelectedBlockHeight(null)} />
          ) : (
            <>
              <h3 style={{ marginBottom: '20px' }}>📦 Latest Blocks</h3>
              <BlocksList blocks={blocks} loading={loading} onViewBlock={handleViewBlock} />
            </>
          )}
        </div>
      )}

      {activeTab === 'transactions' && (
        <div className="card">
          <h3 style={{ marginBottom: '20px' }}>💸 Recent Transactions</h3>
          <TransactionsList transactions={transactions} loading={loading} />
        </div>
      )}

      {activeTab === 'validators' && (
        <div className="card">
          <h3 style={{ marginBottom: '20px' }}>👑 Network Validators</h3>
          <ValidatorsList validators={validators} loading={loading} />
        </div>
      )}

      {activeTab === 'modules' && (
        <div className="card">
          <h3 style={{ marginBottom: '20px' }}>🔧 Protocol Modules Status</h3>
          <ModulesStatus />
        </div>
      )}

      {activeTab === 'consensus' && (
        <>
          <ConsensusInfo params={consensusParams} validators={validators} loading={loading} />
        </>
      )}

      <button className="refresh-btn" onClick={refreshData} title="Refresh Data">
        🔄
      </button>
    </div>
  );
}

export default App;

