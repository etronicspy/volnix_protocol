import React from 'react';
import { TabType } from '../types';

interface TabsProps {
  activeTab: TabType;
  onTabChange: (tab: TabType) => void;
}

const tabs: { id: TabType; label: string; icon: string }[] = [
  { id: 'blocks', label: 'Blocks', icon: '📦' },
  { id: 'transactions', label: 'Transactions', icon: '💸' },
  { id: 'validators', label: 'Validators', icon: '👑' },
  { id: 'modules', label: 'Modules', icon: '🔧' },
  { id: 'consensus', label: 'Consensus', icon: '⚖️' }
];

export const Tabs: React.FC<TabsProps> = ({ activeTab, onTabChange }) => {
  return (
    <div className="tabs">
      {tabs.map(tab => (
        <button
          key={tab.id}
          className={`tab ${activeTab === tab.id ? 'active' : ''}`}
          onClick={() => onTabChange(tab.id)}
        >
          {tab.icon} {tab.label}
        </button>
      ))}
    </div>
  );
};

