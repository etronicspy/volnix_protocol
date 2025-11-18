import React, { useState, useEffect } from 'react';
import { Wallet, Send, History, Users, Coins, Crown } from 'lucide-react';
import WalletConnect from './components/WalletConnect';
import Balance from './components/Balance';
import SendTokens from './components/SendTokens';
import TransactionHistory from './components/TransactionHistory';
import WalletTypes from './components/WalletTypes';
import ValidatorManagement from './components/ValidatorManagement';
import { WalletState, WalletType } from './types/wallet';
import { blockchainService } from './services/blockchainService';

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
    transactions: [],
    isVerified: false
  });

  const [activeTab, setActiveTab] = useState<'wallet' | 'send' | 'history' | 'types' | 'validators'>('wallet');
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');
  const [isRestoring, setIsRestoring] = useState(true); // Флаг для отслеживания восстановления состояния

  // Ключи для localStorage
  const CURRENT_WALLET_KEY = 'volnix_current_wallet';
  const CURRENT_WALLET_MNEMONIC_KEY = 'volnix_current_wallet_mnemonic';
  const ACTIVE_TAB_KEY = 'volnix_active_tab';

  // Восстановление состояния при загрузке страницы
  useEffect(() => {
    const restoreWalletState = async () => {
      try {
        // Восстанавливаем активную вкладку
        const savedTab = localStorage.getItem(ACTIVE_TAB_KEY);
        if (savedTab && ['wallet', 'send', 'history', 'types', 'validators'].includes(savedTab)) {
          setActiveTab(savedTab as typeof activeTab);
        }

        // Восстанавливаем подключенный кошелек
        const savedAddress = localStorage.getItem(CURRENT_WALLET_KEY);
        const savedMnemonic = localStorage.getItem(CURRENT_WALLET_MNEMONIC_KEY);

        if (savedAddress && savedMnemonic) {
          // Восстанавливаем подключение кошелька
          try {
            // Инициализируем signing client с сохраненной мнемоникой
            await blockchainService.initializeSigningClient(savedMnemonic);
            
            // Восстанавливаем состояние подключения ПЕРЕД загрузкой данных
            // Это предотвратит двойную загрузку (из restoreWalletState и из useEffect автообновления)
            setWalletState(prev => ({
              ...prev,
              isConnected: true,
              address: savedAddress
            }));
            
            // Загружаем данные кошелька (loadWalletData сам управляет isLoading)
            await loadWalletData(savedAddress);
          } catch (err: any) {
            console.error('Failed to restore wallet connection:', err);
            // Если не удалось восстановить, очищаем сохраненные данные
            localStorage.removeItem(CURRENT_WALLET_KEY);
            localStorage.removeItem(CURRENT_WALLET_MNEMONIC_KEY);
          }
        }
      } catch (err) {
        console.error('Error restoring wallet state:', err);
      } finally {
        // Помечаем, что восстановление завершено
        setIsRestoring(false);
      }
    };

    restoreWalletState();
  }, []); // Выполняется только при монтировании компонента

  // Сохранение активной вкладки при изменении
  useEffect(() => {
    localStorage.setItem(ACTIVE_TAB_KEY, activeTab);
  }, [activeTab]);

  // Загрузка балансов и транзакций
  const loadWalletData = async (address: string) => {
    if (!address || address.trim() === '') {
      setError('Invalid address');
      return;
    }

    setIsLoading(true);
    setError('');
    
    try {
      // Загружаем балансы
      const balances = await blockchainService.getBalances(address);
      
      // Проверяем что балансы валидны
      if (!balances || typeof balances !== 'object') {
        throw new Error('Invalid balance data received');
      }

      // Загружаем транзакции
      const blockchainTxs = await blockchainService.getTransactions(address);
      
      // Конвертируем транзакции в формат приложения
      const transactions = (blockchainTxs || []).map(tx => {
        if (!tx) return null;
        
        const amountValue = parseFloat(tx.amount || '0') || 0;
        const amountInTokens = amountValue > 0 ? (amountValue / 1_000_000).toFixed(6) : '0';
        
        return {
          id: tx.hash || `tx_${Date.now()}_${Math.random()}`,
          type: (tx.from === address ? 'send' : 'receive') as 'send' | 'receive',
          amount: amountInTokens,
          token: tx.denom === 'uwrt' ? 'WRT' : tx.denom === 'ulzn' ? 'LZN' : 'ANT',
          from: tx.from || address,
          to: tx.to || address,
          timestamp: tx.timestamp || new Date().toISOString(),
          status: (tx.status === 'success' ? 'completed' : 'failed') as 'completed' | 'failed' | 'pending'
        };
      }).filter((tx): tx is NonNullable<typeof tx> => tx !== null);

      setWalletState(prev => ({
        ...prev,
        balance: {
          wrt: balances.wrt || '0',
          lzn: balances.lzn || '0',
          ant: balances.ant || '0'
        },
        transactions
      }));
    } catch (err: any) {
      const errorMessage = err?.message || err?.toString() || 'Failed to load wallet data';
      setError(errorMessage);
      console.error('Error loading wallet data:', err);
      
      // Устанавливаем нулевые балансы при ошибке
      setWalletState(prev => ({
        ...prev,
        balance: {
          wrt: '0',
          lzn: '0',
          ant: '0'
        },
        transactions: prev.transactions || []
      }));
    } finally {
      setIsLoading(false);
    }
  };

  // Автообновление данных каждые 10 секунд
  useEffect(() => {
    // Не запускаем автообновление во время восстановления состояния
    if (isRestoring || !walletState.isConnected || !walletState.address) return;

    loadWalletData(walletState.address);
    const interval = setInterval(() => {
      loadWalletData(walletState.address);
    }, 10000);

    return () => clearInterval(interval);
  }, [isRestoring, walletState.isConnected, walletState.address]);

  const connectWallet = async (address: string, mnemonic?: string) => {
    setIsLoading(true);
    setError('');
    
    try {
      // Загружаем начальные данные
      await loadWalletData(address);
      
      setWalletState(prev => ({
        ...prev,
        isConnected: true,
        address
      }));

      // Сохраняем состояние подключения в localStorage
      localStorage.setItem(CURRENT_WALLET_KEY, address);
      if (mnemonic) {
        localStorage.setItem(CURRENT_WALLET_MNEMONIC_KEY, mnemonic);
      }
    } catch (err: any) {
      setError(err.message || 'Failed to connect wallet');
      setIsLoading(false);
    }
  };

  const disconnectWallet = () => {
    setWalletState({
      isConnected: false,
      address: '',
      balance: { wrt: '0', lzn: '0', ant: '0' },
      walletType: 'guest',
      transactions: [],
      isVerified: false
    });

    // Очищаем сохраненное состояние подключения
    localStorage.removeItem(CURRENT_WALLET_KEY);
    localStorage.removeItem(CURRENT_WALLET_MNEMONIC_KEY);
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
    if (!walletState.address) {
      setError('Wallet not connected');
      return;
    }

    setIsLoading(true);
    setError('');

    // Добавляем транзакцию в состояние как pending
    const newTransaction = {
      id: `pending_${Date.now()}`,
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

    try {
      // Отправляем транзакцию через blockchainService
      const txHash = await blockchainService.sendTokens(
        walletState.address,
        to,
        amount,
        token.toLowerCase() as 'wrt' | 'lzn' | 'ant'
      );

      // Обновляем транзакцию с реальным хешем
      setWalletState(prev => ({
        ...prev,
        transactions: prev.transactions.map(tx =>
          tx.id === newTransaction.id 
            ? { ...tx, id: txHash, status: 'completed' as const }
            : tx
        )
      }));

      // Перезагружаем балансы
      await loadWalletData(walletState.address);
      
      setError('');
    } catch (err: any) {
      setError(err.message || 'Failed to send transaction');
      
      // Обновляем транзакцию как failed
      setWalletState(prev => ({
        ...prev,
        transactions: prev.transactions.map(tx =>
          tx.id === newTransaction.id 
            ? { ...tx, status: 'failed' as const }
            : tx
        )
      }));
    } finally {
      setIsLoading(false);
    }
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

      {error && (
        <div className="card" style={{ background: '#fee2e2', color: '#dc2626', marginBottom: '16px' }}>
          {error}
        </div>
      )}

      {isLoading && (
        <div className="card" style={{ textAlign: 'center', padding: '20px' }}>
          <div style={{ fontSize: '18px', marginBottom: '10px' }}>⏳ Loading...</div>
          <div style={{ color: '#6b7280' }}>Please wait while we fetch blockchain data</div>
        </div>
      )}

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
            {activeTab === 'send' && <SendTokens onSend={sendTokens} balance={walletState.balance} />}
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