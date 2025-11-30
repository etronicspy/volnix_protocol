import React from 'react';
import { Block, Validator } from '../types';

interface StatsGridProps {
  totalBlocks: number;
  totalTransactions: number;
  activeValidators: number;
  burnedAnt: number;
  avgBlockTime: string;
  networkHealth: string;
}

export const StatsGrid: React.FC<StatsGridProps> = ({
  totalBlocks,
  totalTransactions,
  activeValidators,
  burnedAnt,
  avgBlockTime,
  networkHealth
}) => {
  const stats = [
    { label: 'Total Blocks', value: totalBlocks === -1 ? '-' : totalBlocks.toLocaleString() },
    { label: 'Total Transactions', value: totalTransactions === -1 ? '-' : totalTransactions.toLocaleString() },
    { label: 'Active Validators', value: activeValidators === -1 ? '-' : activeValidators.toString() },
    { label: 'ANT Burned (PoVB)', value: burnedAnt === -1 ? '-' : burnedAnt.toLocaleString() },
    { label: 'Avg Block Time', value: avgBlockTime },
    { label: 'Network Health', value: networkHealth }
  ];

  return (
    <div className="stats-grid">
      {stats.map((stat, index) => (
        <div key={index} className="stat-card">
          <div className="stat-value">{stat.value}</div>
          <div className="stat-label">{stat.label}</div>
        </div>
      ))}
    </div>
  );
};

