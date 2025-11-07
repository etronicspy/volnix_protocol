import React from 'react';
import { History, ArrowUpRight, ArrowDownLeft, Clock, CheckCircle, XCircle } from 'lucide-react';
import { Transaction } from '../types/wallet';

interface TransactionHistoryProps {
  transactions: Transaction[];
}

const TransactionHistory: React.FC<TransactionHistoryProps> = ({ transactions }) => {
  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'completed':
        return <CheckCircle size={16} style={{ color: '#10b981' }} />;
      case 'pending':
        return <Clock size={16} style={{ color: '#f59e0b' }} />;
      case 'failed':
        return <XCircle size={16} style={{ color: '#ef4444' }} />;
      default:
        return <Clock size={16} style={{ color: '#6b7280' }} />;
    }
  };

  const getTypeIcon = (type: string) => {
    return type === 'send' ? 
      <ArrowUpRight size={20} style={{ color: '#ef4444' }} /> : 
      <ArrowDownLeft size={20} style={{ color: '#10b981' }} />;
  };

  const formatDate = (timestamp: string) => {
    return new Date(timestamp).toLocaleString();
  };

  const truncateAddress = (address: string) => {
    return `${address.slice(0, 8)}...${address.slice(-6)}`;
  };

  if (transactions.length === 0) {
    return (
      <div className="card" style={{ textAlign: 'center', padding: '48px' }}>
        <History size={48} style={{ color: '#d1d5db', margin: '0 auto 16px' }} />
        <h3 style={{ color: '#6b7280', marginBottom: '8px' }}>No Transactions Yet</h3>
        <p style={{ color: '#9ca3af' }}>
          Your transaction history will appear here once you start using your wallet.
        </p>
      </div>
    );
  }

  return (
    <div className="card">
      <h3 style={{ marginBottom: '24px', display: 'flex', alignItems: 'center', gap: '8px' }}>
        <History size={24} />
        Transaction History
      </h3>

      <div>
        {transactions.map((tx) => (
          <div key={tx.id} className="transaction-item">
            <div className="flex">
              {getTypeIcon(tx.type)}
              <div>
                <div style={{ fontWeight: '600', fontSize: '16px', marginBottom: '4px' }}>
                  {tx.type === 'send' ? 'Sent' : 'Received'} {tx.amount} {tx.token}
                </div>
                <div style={{ color: '#6b7280', fontSize: '14px', marginBottom: '2px' }}>
                  {tx.type === 'send' ? 'To: ' : 'From: '}
                  <span style={{ fontFamily: 'monospace' }}>
                    {truncateAddress(tx.type === 'send' ? tx.to : tx.from)}
                  </span>
                </div>
                <div style={{ color: '#9ca3af', fontSize: '12px' }}>
                  {formatDate(tx.timestamp)}
                </div>
              </div>
            </div>
            <div style={{ textAlign: 'right' }}>
              <div style={{ 
                display: 'flex', 
                alignItems: 'center', 
                gap: '8px',
                marginBottom: '4px'
              }}>
                {getStatusIcon(tx.status)}
                <span style={{ 
                  fontSize: '14px',
                  fontWeight: '600',
                  textTransform: 'capitalize',
                  color: tx.status === 'completed' ? '#10b981' : 
                         tx.status === 'pending' ? '#f59e0b' : '#ef4444'
                }}>
                  {tx.status}
                </span>
              </div>
              <div style={{ 
                fontSize: '16px', 
                fontWeight: '600',
                color: tx.type === 'send' ? '#ef4444' : '#10b981'
              }}>
                {tx.type === 'send' ? '-' : '+'}{tx.amount} {tx.token}
              </div>
            </div>
          </div>
        ))}
      </div>

      <div style={{ 
        marginTop: '20px', 
        textAlign: 'center',
        padding: '16px',
        background: '#f9fafb',
        borderRadius: '8px'
      }}>
        <button 
          className="button" 
          style={{ background: '#6b7280' }}
        >
          Load More Transactions
        </button>
      </div>
    </div>
  );
};

export default TransactionHistory;