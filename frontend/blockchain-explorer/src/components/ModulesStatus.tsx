import React from 'react';

export const ModulesStatus: React.FC = () => {
  const modules = [
    {
      icon: '🔐',
      name: 'Identity (ident)',
      description: 'ZKP identity verification system',
      status: 'active'
    },
    {
      icon: '📜',
      name: 'Lizenz (lizenz)',
      description: 'License activation and MOA tracking',
      status: 'active'
    },
    {
      icon: '💰',
      name: 'Anteil (anteil)',
      description: 'ANT rights trading and distribution',
      status: 'active'
    },
    {
      icon: '⚖️',
      name: 'Consensus (consensus)',
      description: 'PoVB consensus mechanism',
      status: 'active'
    }
  ];

  return (
    <div className="module-status">
      {modules.map((module, index) => (
        <div key={index} className={`module-card ${module.status === 'active' ? 'active' : ''}`}>
          <div style={{ fontSize: '32px', marginBottom: '12px' }}>{module.icon}</div>
          <h4>{module.name}</h4>
          <div className="status-active" style={{ margin: '8px 0' }}>✅ Active</div>
          <p style={{ fontSize: '14px', color: '#6b7280' }}>{module.description}</p>
        </div>
      ))}
    </div>
  );
};

