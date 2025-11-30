import React from 'react';
import { ConsensusParams, Validator } from '../types';

interface ConsensusInfoProps {
  params: ConsensusParams | null;
  validators: Validator[];
  loading: boolean;
}

export const ConsensusInfo: React.FC<ConsensusInfoProps> = ({ params, validators, loading }) => {
  const activeValidators = validators.filter(v => v.status === 'VALIDATOR_STATUS_ACTIVE').length;
  const totalBurned = validators.reduce((sum, v) => sum + parseInt(v.total_burn_amount || '0'), 0);

  return (
    <>
      <div className="consensus-info">
        <h3>⚖️ Proof of Value Burn (PoVB) Consensus</h3>
        <p>Validators compete by burning ANT tokens to create blocks and secure the network</p>
        {params && (
          <div style={{ marginTop: '16px', fontSize: '14px', opacity: 0.9 }}>
            <div>Base Block Time: {params.base_block_time || 'N/A'}</div>
            <div>High Activity Threshold: {parseInt(params.high_activity_threshold || '0').toLocaleString()}</div>
            <div>Low Activity Threshold: {parseInt(params.low_activity_threshold || '0').toLocaleString()}</div>
            <div>Min Burn Amount: {parseInt(params.min_burn_amount || '0').toLocaleString()} ANT</div>
            <div>Max Burn Amount: {parseInt(params.max_burn_amount || '0').toLocaleString()} ANT</div>
          </div>
        )}
      </div>

      <div className="card">
        <h3 style={{ marginBottom: '20px' }}>🔥 Recent Burn Activity</h3>

        <div id="consensus-stats" style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(300px, 1fr))', gap: '20px', marginBottom: '20px' }}>
          <div style={{ background: '#fef3c7', padding: '16px', borderRadius: '8px' }}>
            <h4 style={{ color: '#92400e' }}>Current Status</h4>
            <div style={{ color: '#78350f', marginTop: '8px' }}>
              <div>Active Validators: {activeValidators}</div>
              <div>Total Validators: {validators.length}</div>
            </div>
          </div>

          <div style={{ background: '#ecfdf5', padding: '16px', borderRadius: '8px' }}>
            <h4 style={{ color: '#065f46' }}>Total Statistics</h4>
            <div style={{ color: '#047857', marginTop: '8px' }}>
              <div>Total ANT Burned: {totalBurned.toLocaleString()}</div>
              <div>Active Validators: {activeValidators}</div>
            </div>
          </div>
        </div>

        <div id="burn-history">
          {loading ? (
            <div className="loading">
              <div className="spinner"></div>
              <div>Loading burn history from RPC...</div>
            </div>
          ) : validators.filter(v => parseInt(v.total_burn_amount || '0') > 0).length === 0 ? (
            <div className="loading" style={{ color: '#6b7280' }}>
              <div style={{ marginBottom: '8px' }}>📋 No burn activity yet</div>
              <div style={{ fontSize: '12px', color: '#9ca3af' }}>
                Burn history will appear here once validators start burning ANT.
              </div>
            </div>
          ) : (
            validators
              .filter(v => parseInt(v.total_burn_amount || '0') > 0)
              .sort((a, b) => parseInt(b.total_burn_amount || '0') - parseInt(a.total_burn_amount || '0'))
              .slice(0, 10)
              .map(validator => (
                <div key={validator.validator} className="tx-item">
                  <div>
                    <div style={{ fontWeight: 600, marginBottom: '4px' }}>
                      {validator.validator || 'Unknown'}
                    </div>
                    <div style={{ color: '#6b7280', fontSize: '14px' }}>Validator Address</div>
                    <div style={{ color: '#9ca3af', fontSize: '12px' }}>
                      Blocks Created: {validator.total_blocks_created || 0}
                    </div>
                  </div>
                  <div style={{ textAlign: 'right' }}>
                    <div style={{ fontWeight: 600, color: '#ef4444' }}>
                      {parseInt(validator.total_burn_amount || '0').toLocaleString()} ANT
                    </div>
                    <div style={{ color: '#10b981', fontSize: '12px', fontWeight: 600 }}>Total Burned</div>
                  </div>
                </div>
              ))
          )}
        </div>
      </div>
    </>
  );
};

