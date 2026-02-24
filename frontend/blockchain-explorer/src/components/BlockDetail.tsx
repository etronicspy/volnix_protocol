import React, { useState, useEffect } from 'react';
import { fetchBlock } from '../services/api';

interface BlockDetailProps {
  height: number;
  onBack: () => void;
}

export const BlockDetail: React.FC<BlockDetailProps> = ({ height, onBack }) => {
  const [block, setBlock] = useState<any>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      setLoading(true);
      try {
        const data = await fetchBlock(height);
        if (!cancelled) setBlock(data);
      } catch {
        if (!cancelled) setBlock(null);
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => { cancelled = true; };
  }, [height]);

  if (loading) {
    return (
      <div className="card" style={{ padding: '24px' }}>
        <div className="loading">
          <div className="spinner"></div>
          <div>Загрузка блока #{height}...</div>
        </div>
      </div>
    );
  }

  if (!block) {
    return (
      <div className="card" style={{ padding: '24px' }}>
        <div style={{ color: '#ef4444', marginBottom: '16px' }}>Не удалось загрузить блок #{height}</div>
        <button className="button" onClick={onBack}>← Назад</button>
      </div>
    );
  }

  const inner = block.block || block;
  const header = inner?.header || {};
  const blockData = inner?.data || {};
  const blockId = block.block_id || block.blockId || {};
  const txs = blockData.txs || [];
  const proposer = header.proposer_address || header.proposer || '—';

  return (
    <div className="card" style={{ padding: '24px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
        <h3 style={{ margin: 0 }}>📦 Блок #{height}</h3>
        <button className="button" onClick={onBack}>← Назад к списку</button>
      </div>

      <div style={{ display: 'grid', gap: '16px', marginBottom: '24px' }}>
        <div style={{ padding: '12px', background: '#f9fafb', borderRadius: '8px' }}>
          <div style={{ display: 'grid', gridTemplateColumns: '140px 1fr', gap: '8px' }}>
            <span style={{ color: '#6b7280' }}>Hash:</span>
            <span className="hash" style={{ wordBreak: 'break-all' }}>{blockId.hash || '—'}</span>
            <span style={{ color: '#6b7280' }}>Время:</span>
            <span>{header.time ? new Date(header.time).toLocaleString() : '—'}</span>
            <span style={{ color: '#6b7280' }}>Валидатор:</span>
            <span style={{ wordBreak: 'break-all' }}>{proposer}</span>
            <span style={{ color: '#6b7280' }}>Транзакций:</span>
            <span>{Array.isArray(txs) ? txs.length : 0}</span>
          </div>
        </div>

        <div>
          <h4 style={{ marginBottom: '12px' }}>Транзакции в блоке</h4>
          {Array.isArray(txs) && txs.length > 0 ? (
            <div style={{ fontSize: '14px' }}>
              {txs.map((tx: any, i: number) => (
                <div key={i} style={{ padding: '8px', background: '#f3f4f6', borderRadius: '4px', marginBottom: '4px' }}>
                  {typeof tx === 'string' ? `Tx: ${tx.substring(0, 40)}...` : `Tx #${i + 1}`}
                </div>
              ))}
            </div>
          ) : (
            <div style={{ color: '#6b7280', padding: '12px' }}>Нет транзакций</div>
          )}
        </div>
      </div>
    </div>
  );
};
