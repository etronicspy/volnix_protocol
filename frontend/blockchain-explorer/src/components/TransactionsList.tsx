import React from 'react';
import { Transaction } from '../types';

interface TransactionsListProps {
  transactions: Transaction[];
  loading: boolean;
}

export const TransactionsList: React.FC<TransactionsListProps> = ({ transactions, loading }) => {
  if (loading) {
    return (
      <div className="loading">
        <div className="spinner"></div>
        <div>Loading transactions from RPC...</div>
      </div>
    );
  }

  if (transactions.length === 0) {
    return (
      <div className="loading" style={{ color: '#6b7280' }}>
        <div style={{ marginBottom: '8px' }}>📋 No transactions found</div>
        <div style={{ fontSize: '12px', color: '#9ca3af' }}>
          Transactions will appear here once blocks contain them.
        </div>
      </div>
    );
  }

  return (
    <>
      <input
        type="text"
        className="search-box"
        placeholder="Search by transaction hash or address..."
        id="tx-search"
      />
      <div id="transactions-list">
        {transactions.map((tx, index) => (
          <div key={tx.hash || index} className="tx-item">
            <div>
              <div style={{ fontWeight: 600, marginBottom: '4px' }}>Transaction</div>
              <div style={{ color: '#6b7280', fontSize: '14px' }}>
                Hash: <span className="hash">{tx.hash.substring(0, 20)}...</span>
              </div>
              <div style={{ color: '#9ca3af', fontSize: '12px' }}>
                Block #{tx.height.toLocaleString()} • {new Date(tx.time).toLocaleString()}
              </div>
            </div>
            <div style={{ textAlign: 'right' }}>
              <div style={{ color: '#10b981', fontSize: '12px', fontWeight: 600 }}>✅ Confirmed</div>
            </div>
          </div>
        ))}
      </div>
    </>
  );
};

