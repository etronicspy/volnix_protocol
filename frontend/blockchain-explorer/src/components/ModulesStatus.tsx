import React, { useState, useEffect } from 'react';

const REST_API_ENDPOINT = process.env.REACT_APP_REST_API_ENDPOINT || 'http://localhost:1317';

const MODULE_CONFIG: Record<string, { icon: string; name: string; description: string }> = {
  ident: { icon: '🔐', name: 'Identity (ident)', description: 'ZKP identity verification system' },
  lizenz: { icon: '📜', name: 'Lizenz (lizenz)', description: 'License activation and MOA tracking' },
  anteil: { icon: '💰', name: 'Anteil (anteil)', description: 'ANT rights trading and distribution' },
  consensus: { icon: '⚖️', name: 'Consensus (consensus)', description: 'PoVB consensus mechanism' },
};

export const ModulesStatus: React.FC = () => {
  const [modules, setModules] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const res = await fetch(`${REST_API_ENDPOINT}/status`);
        if (!res.ok) throw new Error('Failed to fetch');
        const data = await res.json();
        if (!cancelled && data.modules) {
          setModules(data.modules);
        }
      } catch {
        if (!cancelled) {
          setModules({ ident: 'unknown', lizenz: 'unknown', anteil: 'unknown', consensus: 'unknown' });
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => { cancelled = true; };
  }, []);

  const entries = Object.entries(MODULE_CONFIG);
  return (
    <div className="module-status">
      {loading ? (
        <div style={{ padding: '24px', textAlign: 'center', color: '#6b7280' }}>Загрузка статуса модулей...</div>
      ) : (
        entries.map(([key, cfg]) => {
          const status = modules[key] || 'unknown';
          return (
            <div key={key} className={`module-card ${status === 'active' ? 'active' : ''}`}>
              <div style={{ fontSize: '32px', marginBottom: '12px' }}>{cfg.icon}</div>
              <h4>{cfg.name}</h4>
              <div style={{ margin: '8px 0', color: status === 'active' ? '#10b981' : status === 'degraded' ? '#f59e0b' : '#6b7280' }}>
                {status === 'active' ? '✅ Active' : status === 'degraded' ? '⚠️ Degraded' : '❓ Unknown'}
              </div>
              <p style={{ fontSize: '14px', color: '#6b7280' }}>{cfg.description}</p>
            </div>
          );
        })
      )}
    </div>
  );
};

