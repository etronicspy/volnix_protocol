import React from 'react';
import { Block } from '../types';

interface BlocksListProps {
  blocks: Block[];
  loading: boolean;
  onViewBlock: (height: number) => void;
}

export const BlocksList: React.FC<BlocksListProps> = ({ blocks, loading, onViewBlock }) => {
  if (loading) {
    return (
      <div className="loading">
        <div className="spinner"></div>
        <div>Loading blocks from RPC...</div>
      </div>
    );
  }

  if (blocks.length === 0) {
    return (
      <div className="loading" style={{ color: '#6b7280' }}>
        <div style={{ marginBottom: '8px' }}>📋 No blocks found</div>
        <div style={{ fontSize: '12px', color: '#9ca3af' }}>
          Blocks will appear here once the network starts producing them.
        </div>
      </div>
    );
  }

  return (
    <>
      <input
        type="text"
        className="search-box"
        placeholder="Search by block height or hash..."
        id="block-search"
      />
      <div id="blocks-list">
        {blocks.map(block => (
          <div key={block.height} className="block-item">
            <div>
              <div style={{ fontWeight: 600, marginBottom: '4px' }}>
                Block #{block.height.toLocaleString()}
              </div>
              <div style={{ color: '#6b7280', fontSize: '14px' }}>
                Hash: <span className="hash">{block.hash.substring(0, 20)}...</span>
              </div>
              <div style={{ color: '#9ca3af', fontSize: '12px' }}>
                {new Date(block.time).toLocaleString()} • {block.txs} transaction{block.txs !== 1 ? 's' : ''}
              </div>
            </div>
            <div style={{ textAlign: 'right' }}>
              <div className="status-active">✅ Confirmed</div>
              <div style={{ color: '#6b7280', fontSize: '14px' }}>
                Validator: {block.proposer ? block.proposer.substring(0, 20) + '...' : 'Unknown'}
              </div>
              <button className="button" onClick={() => onViewBlock(block.height)} style={{ marginTop: '4px' }}>
                View
              </button>
            </div>
          </div>
        ))}
      </div>
    </>
  );
};

