import React from 'react';
import { Crown, Activity, TrendingUp, AlertTriangle, CheckCircle, XCircle } from 'lucide-react';
import { ValidatorInfo } from '../types/wallet';

interface ValidatorDashboardProps {
  validatorInfo: ValidatorInfo;
  onActivateLzn: (amount: string) => void;
  onDeactivateLzn: (amount: string) => void;
}

const ValidatorDashboard: React.FC<ValidatorDashboardProps> = ({ 
  validatorInfo,
  onActivateLzn,
  onDeactivateLzn
}) => {
  const moaPercentage = validatorInfo.moaCompliance * 100;
  const getMoaStatus = () => {
    if (moaPercentage >= 100) return { color: '#10b981', text: 'Выполнено', icon: <CheckCircle size={20} /> };
    if (moaPercentage >= 90) return { color: '#f59e0b', text: 'Предупреждение', icon: <AlertTriangle size={20} /> };
    if (moaPercentage >= 70) return { color: '#ef4444', text: 'Штраф 25%', icon: <AlertTriangle size={20} /> };
    if (moaPercentage >= 50) return { color: '#dc2626', text: 'Штраф 50%', icon: <XCircle size={20} /> };
    return { color: '#991b1b', text: 'Риск деактивации!', icon: <XCircle size={20} /> };
  };

  const moaStatus = getMoaStatus();

  return (
    <div style={{ width: '100%' }}>
      {/* Заголовок */}
      <div className="card" style={{ 
        background: 'linear-gradient(135deg, #f59e0b 0%, #d97706 100%)',
        color: 'white'
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          <Crown size={32} />
          <div>
            <h3 style={{ margin: 0, fontSize: '1.5rem' }}>Панель Валидатора</h3>
            <p style={{ margin: '4px 0 0 0', opacity: 0.9 }}>
              Контур 1 (Пассивный доход) + Контур 2 (Активный доход)
            </p>
          </div>
        </div>
      </div>

      {/* MOA Статус - КРИТИЧЕСКИ ВАЖНО */}
      <div className="card" style={{ 
        background: moaPercentage >= 90 ? '#f0fdf4' : '#fef2f2',
        border: `3px solid ${moaStatus.color}`
      }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '12px' }}>
          <h4 style={{ margin: 0, display: 'flex', alignItems: 'center', gap: '8px' }}>
            <Activity size={24} />
            MOA - Минимальное Обязательство Активности
          </h4>
          <div style={{ 
            display: 'flex', 
            alignItems: 'center', 
            gap: '8px',
            color: moaStatus.color,
            fontWeight: '600'
          }}>
            {moaStatus.icon}
            {moaStatus.text}
          </div>
        </div>

        <div style={{ marginBottom: '16px' }}>
          <div style={{ 
            background: '#e5e7eb', 
            height: '24px', 
            borderRadius: '12px',
            overflow: 'hidden',
            position: 'relative'
          }}>
            <div style={{ 
              background: moaStatus.color,
              height: '100%',
              width: `${Math.min(moaPercentage, 100)}%`,
              transition: 'width 0.3s ease',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: 'white',
              fontSize: '12px',
              fontWeight: '600'
            }}>
              {moaPercentage.toFixed(1)}%
            </div>
          </div>
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px', marginBottom: '16px' }}>
          <div>
            <div style={{ fontSize: '14px', color: '#6b7280', marginBottom: '4px' }}>
              Требуется за эпоху (7 дней)
            </div>
            <div style={{ fontSize: '1.5rem', fontWeight: 'bold' }}>
              {validatorInfo.moaRequired} ANT
            </div>
          </div>
          <div>
            <div style={{ fontSize: '14px', color: '#6b7280', marginBottom: '4px' }}>
              Использовано
            </div>
            <div style={{ fontSize: '1.5rem', fontWeight: 'bold', color: moaStatus.color }}>
              {validatorInfo.moaCurrent} ANT
            </div>
          </div>
        </div>

        <div style={{ 
          background: moaPercentage >= 90 ? '#dbeafe' : '#fee2e2',
          padding: '12px',
          borderRadius: '6px',
          fontSize: '13px'
        }}>
          <strong>Важно:</strong> MOA связывает ваш пассивный доход (Контур 1) с активным участием (Контур 2). 
          {moaPercentage < 90 && (
            <span style={{ color: '#dc2626', display: 'block', marginTop: '4px' }}>
              ⚠️ Невыполнение MOA приведет к штрафам или деактивации LZN!
            </span>
          )}
        </div>
      </div>

      {/* Доходы */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '20px', marginBottom: '20px' }}>
        {/* Контур 1: Пассивный доход */}
        <div className="card" style={{ background: '#f0f9ff' }}>
          <h5 style={{ marginBottom: '12px', color: '#1e40af' }}>
            💰 Контур 1: Пассивный доход
          </h5>
          <div style={{ fontSize: '2rem', fontWeight: 'bold', color: '#3b82f6', marginBottom: '8px' }}>
            {validatorInfo.passiveIncome} WRT
          </div>
          <div style={{ fontSize: '14px', color: '#6b7280', marginBottom: '12px' }}>
            За текущую эпоху
          </div>
          <div style={{ fontSize: '13px', color: '#1e40af', background: '#dbeafe', padding: '8px', borderRadius: '4px' }}>
            Доля в сети: {validatorInfo.shareOfNetwork}%<br/>
            Источник: Базовая эмиссия WRT
          </div>
        </div>

        {/* Контур 2: Активный доход */}
        <div className="card" style={{ background: '#fef3c7' }}>
          <h5 style={{ marginBottom: '12px', color: '#92400e' }}>
            ⚡ Контур 2: Активный доход
          </h5>
          <div style={{ fontSize: '2rem', fontWeight: 'bold', color: '#f59e0b', marginBottom: '8px' }}>
            {validatorInfo.activeIncome} WRT
          </div>
          <div style={{ fontSize: '14px', color: '#6b7280', marginBottom: '12px' }}>
            За текущую эпоху
          </div>
          <div style={{ fontSize: '13px', color: '#92400e', background: '#fef3c7', padding: '8px', borderRadius: '4px' }}>
            Блоков выиграно: {validatorInfo.blocksWonTotal}<br/>
            Источник: Комиссии из аукционов
          </div>
        </div>
      </div>

      {/* Активация LZN */}
      <div className="card">
        <h4 style={{ marginBottom: '16px', display: 'flex', alignItems: 'center', gap: '8px' }}>
          <TrendingUp size={24} />
          Активация LZN (Лицензия на майнинг)
        </h4>

        <div style={{ 
          background: '#f9fafb', 
          padding: '16px', 
          borderRadius: '8px',
          marginBottom: '16px'
        }}>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: '16px' }}>
            <div>
              <div style={{ fontSize: '14px', color: '#6b7280', marginBottom: '4px' }}>
                Всего LZN
              </div>
              <div style={{ fontSize: '1.3rem', fontWeight: 'bold' }}>
                {validatorInfo.lznTotal}
              </div>
            </div>
            <div>
              <div style={{ fontSize: '14px', color: '#6b7280', marginBottom: '4px' }}>
                Активировано
              </div>
              <div style={{ fontSize: '1.3rem', fontWeight: 'bold', color: '#10b981' }}>
                {validatorInfo.lznActivated}
              </div>
            </div>
            <div>
              <div style={{ fontSize: '14px', color: '#6b7280', marginBottom: '4px' }}>
                Доступно
              </div>
              <div style={{ fontSize: '1.3rem', fontWeight: 'bold', color: '#3b82f6' }}>
                {(parseFloat(validatorInfo.lznTotal) - parseFloat(validatorInfo.lznActivated)).toFixed(2)}
              </div>
            </div>
          </div>
        </div>

        <div style={{ 
          background: '#fef3c7', 
          padding: '12px', 
          borderRadius: '6px',
          marginBottom: '16px',
          fontSize: '13px',
          color: '#92400e'
        }}>
          <strong>⚠️ Ограничение:</strong> Один валидатор может активировать максимум 33% от общего пула LZN в сети. 
          Активация LZN дает право на долю от базовой эмиссии WRT (Контур 1).
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '12px' }}>
          <button
            className="button"
            onClick={() => {
              const amount = prompt('Введите количество LZN для активации:');
              if (amount) onActivateLzn(amount);
            }}
            style={{ background: '#10b981' }}
          >
            Активировать LZN
          </button>
          <button
            className="button"
            onClick={() => {
              const amount = prompt('Введите количество LZN для деактивации:');
              if (amount) onDeactivateLzn(amount);
            }}
            style={{ background: '#ef4444' }}
            disabled={parseFloat(validatorInfo.lznActivated) === 0}
          >
            Деактивировать LZN
          </button>
        </div>
      </div>

      {/* Статистика */}
      <div className="card" style={{ background: '#f9fafb' }}>
        <h5 style={{ marginBottom: '16px' }}>Статистика валидатора</h5>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: '16px', fontSize: '14px' }}>
          <div>
            <div style={{ color: '#6b7280', marginBottom: '4px' }}>Доля в сети</div>
            <div style={{ fontWeight: '600', fontSize: '1.1rem' }}>{validatorInfo.shareOfNetwork}%</div>
          </div>
          <div>
            <div style={{ color: '#6b7280', marginBottom: '4px' }}>Всего блоков</div>
            <div style={{ fontWeight: '600', fontSize: '1.1rem' }}>{validatorInfo.blocksWonTotal}</div>
          </div>
          <div>
            <div style={{ color: '#6b7280', marginBottom: '4px' }}>Последний блок</div>
            <div style={{ fontWeight: '600', fontSize: '1.1rem' }}>
              {validatorInfo.lastBlockWon || 'N/A'}
            </div>
          </div>
        </div>
      </div>

      {/* Правила активности */}
      <div className="card" style={{ background: '#fef2f2', border: '2px solid #fecaca' }}>
        <h5 style={{ color: '#dc2626', marginBottom: '12px' }}>
          ⚠️ Правила активности валидатора
        </h5>
        <ul style={{ margin: 0, paddingLeft: '20px', fontSize: '14px', color: '#991b1b' }}>
          <li>Валидатор должен участвовать в консенсусе хотя бы раз в <strong>6 месяцев</strong></li>
          <li>При неактивности статус валидатора аннулируется</li>
          <li>LZN принудительно размораживаются</li>
          <li>Теряется право на получение доли от базовой эмиссии</li>
          <li>Для восстановления требуется повторная ZKP-верификация</li>
        </ul>
      </div>
    </div>
  );
};

export default ValidatorDashboard;
