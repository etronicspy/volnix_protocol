import React from 'react';
import { NetworkStatus } from '../types';

interface NetworkOverviewProps {
  status: NetworkStatus | null;
  activeValidatorsCount: number;
  restApiAvailable: boolean;
  rpcAvailable: boolean;
}

export const NetworkOverview: React.FC<NetworkOverviewProps> = ({
  status,
  activeValidatorsCount,
  restApiAvailable,
  rpcAvailable
}) => {
  const chainId = status?.node_info.network || '-';
  const blockHeight = status?.sync_info.latest_block_height || '-';
  const networkStatus = status ? 'Connected' : 'Disconnected';

  return (
    <div className="network-overview">
      <h3>🌐 Network Status: <span>{networkStatus}</span></h3>
      <p>
        Chain ID: {chainId} • <span>{activeValidatorsCount}</span> Validators Active • Block Height: <span>{typeof blockHeight === 'string' ? parseInt(blockHeight).toLocaleString() : blockHeight}</span>
      </p>
      <p style={{ fontSize: '14px', marginTop: '8px', opacity: 0.9 }}>
        RPC: <span>{rpcAvailable ? '🟢' : '🔴'}</span>
        {' '}REST API: <span style={{ color: restApiAvailable ? '#10b981' : '#ef4444' }}>
          {restApiAvailable ? '🟢 Connected' : '🔴 Not Available'}
        </span>
      </p>
    </div>
  );
};

