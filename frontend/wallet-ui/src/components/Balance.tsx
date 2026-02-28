import React from 'react';
import { Coins, TrendingUp, Shield } from 'lucide-react';
import { Balance as BalanceType, WalletType } from '../types/wallet';

type QuickActionTab = 'staking' | 'market' | 'types';

interface BalanceProps {
  balance: BalanceType;
  walletType?: WalletType;
  onQuickAction?: (tab: QuickActionTab) => void;
}

const Balance: React.FC<BalanceProps> = ({ balance, walletType = 'guest', onQuickAction }) => {
  const tokens = [
    {
      symbol: 'WRT',
      name: 'Wealth Rights Token',
      amount: balance.wrt,
      icon: <Coins size={24} />,
      color: '#667eea',
      description: 'Primary utility token for transactions and fees'
    },
    {
      symbol: 'LZN',
      name: 'Lizenz Token',
      amount: balance.lzn,
      icon: <TrendingUp size={24} />,
      color: '#10b981',
      description: 'Staking token for validators and governance'
    },
    {
      symbol: 'ANT',
      name: 'Anteil Rights',
      amount: balance.ant,
      icon: <Shield size={24} />,
      color: '#f59e0b',
      description: 'Governance rights for verified citizens'
    }
  ];

  return (
    <div style={{ width: '100%' }}>
      <div className="balance-card">
        <div className="balance-amount">{balance.wrt} WRT</div>
        <div className="balance-label">Wealth Rights Token</div>
      </div>

      <div className="card">
        <h3 style={{ marginBottom: '20px', display: 'flex', alignItems: 'center', gap: '8px' }}>
          <Coins size={24} />
          Token Balances
        </h3>
        
        {tokens.map((token) => {
          // ANT: Guest — locked. Citizen receives from protocol. Validator buys on market (per whitepaper §4.2)
          const isAntLocked = token.symbol === 'ANT' && walletType === 'guest';
          const isLocked = isAntLocked;
          return (
            <div key={token.symbol} className="transaction-item" style={isLocked ? { opacity: 0.6 } : {}}>
              <div className="flex">
                <div style={{ color: isLocked ? '#9ca3af' : token.color }}>
                  {token.icon}
                </div>
                <div>
                  <div style={{ fontWeight: '600', fontSize: '16px', color: isLocked ? '#9ca3af' : 'inherit' }}>
                    {token.amount} {token.symbol}
                    {isLocked && <span style={{ marginLeft: '8px', fontSize: '12px', background: '#ef4444', color: 'white', padding: '2px 6px', borderRadius: '4px' }}>🔒 LOCKED</span>}
                  </div>
                  <div style={{ color: '#6b7280', fontSize: '14px' }}>
                    {token.name}
                  </div>
                  <div style={{ color: isLocked ? '#ef4444' : '#9ca3af', fontSize: '12px', marginTop: '4px' }}>
                    {isLocked && token.symbol === 'ANT' 
                      ? 'Requires Citizen status to access (Validators buy ANT on Market)'
                      : token.description}
                  </div>
                </div>
              </div>
            </div>
          );
        })}
      </div>

      <div className="card">
        <h3 style={{ marginBottom: '16px' }}>Quick Actions</h3>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))', gap: '12px' }}>
          <button
            className="button"
            style={{ background: '#10b981' }}
            onClick={() => onQuickAction?.('staking')}
          >
            Stake LZN
          </button>
          <button
            className="button"
            style={{
              background: walletType === 'guest' ? '#6b7280' : '#10b981',
              opacity: walletType === 'guest' ? 0.6 : 1
            }}
            disabled={walletType === 'guest'}
            title={walletType === 'guest' ? 'Requires Citizen status' : walletType === 'validator' ? 'Validators buy ANT on Market' : 'Claim ANT tokens from protocol'}
            onClick={() => walletType !== 'guest' && onQuickAction?.('market')}
          >
            {walletType === 'guest' ? '🔒 Claim ANT' : walletType === 'validator' ? 'Buy ANT (Market)' : 'Claim ANT'}
          </button>
          <button
            className="button"
            style={{ background: '#8b5cf6' }}
            onClick={() => onQuickAction?.('market')}
          >
            Swap Tokens
          </button>
        </div>
      </div>
    </div>
  );
};

export default Balance;