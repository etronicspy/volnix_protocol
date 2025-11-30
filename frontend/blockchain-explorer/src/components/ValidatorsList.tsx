import React from 'react';
import { Validator } from '../types';

interface ValidatorsListProps {
  validators: Validator[];
  loading: boolean;
}

export const ValidatorsList: React.FC<ValidatorsListProps> = ({ validators, loading }) => {
  if (loading) {
    return (
      <div className="loading">
        <div className="spinner"></div>
        <div>Loading validators from REST API...</div>
      </div>
    );
  }

  if (validators.length === 0) {
    return (
      <div className="loading" style={{ color: '#6b7280' }}>
        <div style={{ marginBottom: '8px' }}>📋 No validators registered yet</div>
        <div style={{ fontSize: '12px', color: '#9ca3af' }}>
          Validators will appear here once they register in the consensus module.
        </div>
      </div>
    );
  }

  return (
    <div id="validators-list">
      {validators.map(validator => {
        const statusClass = validator.status === 'VALIDATOR_STATUS_ACTIVE' ? 'status-active' : 'status-inactive';
        const statusText = validator.status === 'VALIDATOR_STATUS_ACTIVE' ? '🟢 Active' : '🔴 Inactive';
        const antBalance = parseInt(validator.ant_balance || '0');
        const totalBurn = parseInt(validator.total_burn_amount || '0');

        return (
          <div key={validator.validator} className="validator-item">
            <div>
              <div style={{ fontWeight: 600, marginBottom: '4px' }}>
                {validator.validator || 'Unknown'}
              </div>
              <div style={{ color: '#6b7280', fontSize: '14px' }}>
                Address: <span className="hash">{(validator.validator || '').substring(0, 30)}...</span>
              </div>
              <div style={{ color: '#9ca3af', fontSize: '12px' }}>
                ANT Balance: {antBalance.toLocaleString()} • Activity Score: {validator.activity_score || '0'}
              </div>
            </div>
            <div style={{ textAlign: 'right' }}>
              <div className={statusClass}>{statusText}</div>
              <div style={{ color: '#6b7280', fontSize: '14px' }}>ANT: {antBalance.toLocaleString()}</div>
              <div style={{ color: '#6b7280', fontSize: '14px' }}>Blocks: {validator.total_blocks_created || 0}</div>
              <div style={{ color: '#ef4444', fontSize: '12px' }}>Burned: {totalBurn.toLocaleString()} ANT</div>
            </div>
          </div>
        );
      })}
    </div>
  );
};

