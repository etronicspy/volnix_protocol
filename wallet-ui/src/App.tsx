import React, { useState, useEffect } from 'react';
import { Wallet, Send, History, Settings, Shield, Users, Coins, Crown } from 'lucide-react';
import WalletConnect from './components/WalletConnect';
import Balance from './components/Balance';
import SendTokens from './components/SendTokens';
import TransactionHistory from './components/TransactionHistory';
import WalletTypes from './components/WalletTypes';
import ValidatorManagement from './components/ValidatorManagement';
import { WalletState } from './types/wallet';

function App() {
  const [walletState, setWalletState] = useState<WalletState>({
    isConnected: false,
    address: '',
    balance: {
      wrt: '0',
      lzn: '0',
      ant: '0'
    },
    walletType: 'guest',
    transactions: []
  });

  const [activeTab, setActiveTab] = useState<'wallet' | 'send' | 'history' | 'types' | 'validators'>('wallet');

  const connectWallet = async (address: string) => {
    // Симуляция подключения кошелька
    setWalletState(prev => ({
      ...prev,
      isConnected: true,
      address,
      balance: {
        wrt: '1000.50',
        lzn: '250.75',
        ant: '0' // Guest не имеет доступа к ANT токенам
      },
      transactions: [
        {
          id: '1',
          type: 'receive',
          amount: '100.00',
          token: 'WRT',
          from: 'volnix1abc...def',
          to: address,
          timestamp: new Date().toISOString(),
          status: 'completed'
        },
        {
          id: '2',
          type: 'send',
          amount: '50.25',
          token: 'LZN',
          from: address,
          to: 'volnix1xyz...uvw',
          timestamp: new Date(Date.now() - 3600000).toISOString(),
          status: 'completed'
        }
      ]
    }));
  };

  const disconnectWallet = () => {
    setWalletState({
      isConnected: false,
      address: '',
      balance: { wrt: '0', lzn: '0', ant: '0' },
      walletType: 'guest',
      transactions: []
    });
  };

  const upgradeWalletType = (newType: WalletType) => {
    setWalletState(prev => ({
      ...prev,
      walletType: newType,
      balance: {
        ...prev.balance,
        ant: newType === 'citizen' || newType === 'validator' ? '10' : '0'
      }
    }));
  };

  const sendTokens = async (to: string, amount: string, token: string) => {
    // Симуляция отправки токенов
    const newTransaction = {
      id: Date.now().toString(),
      type: 'send' as const,
      amount,
      token,
      from: walletState.address,
      to,
      timestamp: new Date().toISOString(),
      status: 'pending' as const
    };

    setWalletState(prev => ({
      ...prev,
      transactions: [newTransaction, ...prev.transactions]
    }));

    // Симуляция подтверждения через 2 секунды
    setTimeout(() => {
      setWalletState(prev => ({
        ...prev,
        transactions: prev.transactions.map(tx =>
          tx.id === newTransaction.id ? { ...tx, status: 'completed' } : tx
        )
      }));
    }, 2000);
  };

  return (
    <div className="container">
      <header className="text-center mb-4">
        <h1 style={{ 
          fontSize: '2.5rem', 
          fontWeight: 'bold', 
          color: 'white', 
          marginBottom: '8px',
          textShadow: '0 2px 4px rgba(0,0,0,0.3)'
        }}>
          <Wallet style={{ display: 'inline', marginRight: '12px' }} />
          Volnix Wallet
        </h1>
        <p style={{ color: 'rgba(255,255,255,0.9)', fontSize: '1.1rem' }}>
          Secure Multi-Token Blockchain Wallet
        </p>
      </header>

      {!walletState.isConnected ? (
        <WalletConnect onConnect={connectWallet} />
      ) : (
        <>
          <div className="card">
            <div className="flex" style={{ justifyContent: 'space-between', alignItems: 'center' }}>
              <div className="flex">
                <span className="status-connected">● Connected</span>
                <span style={{ marginLeft: '12px', fontFamily: 'monospace' }}>
                  {walletState.address}
                </span>
                <span style={{ 
                  marginLeft: '12px', 
                  padding: '2px 8px', 
                  borderRadius: '12px', 
                  fontSize: '12px',
                  fontWeight: '600',
                  background: walletState.walletType === 'guest' ? '#6b7280' : 
                             walletState.walletType === 'citizen' ? '#10b981' : '#f59e0b',
                  color: 'white'
                }}>
                  {walletState.walletType.toUpperCase()}
                </span>
              </div>
              <button 
                className="button" 
                onClick={disconnectWallet}
                style={{ background: '#ef4444' }}
              >
                Disconnect
              </button>
            </div>
          </div>

          <nav className="card">
            <div className="flex" style={{ justifyContent: 'center', gap: '8px' }}>
              <button
                className={`button ${activeTab === 'wallet' ? '' : 'button-secondary'}`}
                onClick={() => setActiveTab('wallet')}
                style={activeTab !== 'wallet' ? { background: '#6b7280', opacity: 0.7 } : {}}
              >
                <Coins size={20} />
                Balance
              </button>
              <button
                className={`button ${activeTab === 'send' ? '' : 'button-secondary'}`}
                onClick={() => setActiveTab('send')}
                style={activeTab !== 'send' ? { background: '#6b7280', opacity: 0.7 } : {}}
              >
                <Send size={20} />
                Send
              </button>
              <button
                className={`button ${activeTab === 'history' ? '' : 'button-secondary'}`}
                onClick={() => setActiveTab('history')}
                style={activeTab !== 'history' ? { background: '#6b7280', opacity: 0.7 } : {}}
              >
                <History size={20} />
                History
              </button>
              <button
                className={`button ${activeTab === 'types' ? '' : 'button-secondary'}`}
                onClick={() => setActiveTab('types')}
                style={activeTab !== 'types' ? { background: '#6b7280', opacity: 0.7 } : {}}
              >
                <Users size={20} />
                Wallet Types
              </button>
              <button
                className={`button ${activeTab === 'validators' ? '' : 'button-secondary'}`}
                onClick={() => setActiveTab('validators')}
                style={activeTab !== 'validators' ? { background: '#6b7280', opacity: 0.7 } : {}}
              >
                <Crown size={20} />
                Validators
              </button>
            </div>
          </nav>

          <div className="grid">
            {activeTab === 'wallet' && <Balance balance={walletState.balance} />}
            {activeTab === 'send' && <SendTokens onSend={sendTokens} />}
            {activeTab === 'history' && <TransactionHistory transactions={walletState.transactions} />}
            {activeTab === 'types' && <WalletTypes currentType={walletState.walletType} onUpgrade={upgradeWalletType} />}
            {activeTab === 'validators' && <ValidatorManagement />}
          </div>
        </>
      )}
    </div>
  );
}

export default App;