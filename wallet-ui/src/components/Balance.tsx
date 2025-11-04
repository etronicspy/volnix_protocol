import React from 'react';
import { Coins, TrendingUp, Shield } from 'lucide-react';
import { Balance as BalanceType } from '../types/wallet';

interface BalanceProps {
  balance: BalanceType;
}

const Balance: React.FC<BalanceProps> = ({ balance }) => {
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

  const totalValue = (
    parseFloat(balance.wrt) * 1.0 + 
    parseFloat(balance.lzn) * 2.5 + 
    parseFloat(balance.ant) * 10.0
  ).toFixed(2);

  return (
    <div style={{ width: '100%' }}>
      <div className="balance-card">
        <div className="balance-amount">${totalValue}</div>
        <div className="balance-label">Total Portfolio Value</div>
      </div>

      <div className="card">
        <h3 style={{ marginBottom: '20px', display: 'flex', alignItems: 'center', gap: '8px' }}>
          <Coins size={24} />
          Token Balances
        </h3>
        
        {tokens.map((token) => {
          const isLocked = token.symbol === 'ANT' && parseFloat(token.amount) === 0;
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
                    {isLocked ? 'Requires Citizen status to access' : token.description}
                  </div>
                </div>
              </div>
              <div style={{ textAlign: 'right' }}>
                <div style={{ fontWeight: '600', color: isLocked ? '#9ca3af' : 'inherit' }}>
                  ${(parseFloat(token.amount) * (token.symbol === 'WRT' ? 1.0 : token.symbol === 'LZN' ? 2.5 : 10.0)).toFixed(2)}
                </div>
                <div style={{ color: '#6b7280', fontSize: '14px' }}>
                  ${token.symbol === 'WRT' ? '1.00' : token.symbol === 'LZN' ? '2.50' : '10.00'} per token
                </div>
              </div>
            </div>
          );
        })}
      </div>

      <div className="card">
        <h3 style={{ marginBottom: '16px' }}>Quick Actions</h3>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))', gap: '12px' }}>
          <button className="button" style={{ background: '#10b981' }}>
            Stake LZN
          </button>
          <button 
            className="button" 
            style={{ background: '#6b7280', opacity: 0.6 }}
            disabled
            title="Requires Citizen status"
          >
            🔒 Claim ANT
          </button>
          <button className="button" style={{ background: '#8b5cf6' }}>
            Swap Tokens
          </button>
        </div>
      </div>
    </div>
  );
};

export default Balance;