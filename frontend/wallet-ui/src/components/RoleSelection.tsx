import React, { useState } from 'react';
import { Shield, Crown, AlertTriangle, CheckCircle } from 'lucide-react';
import { WalletType } from '../types/wallet';

interface RoleSelectionProps {
  onRoleSelected: (role: 'citizen' | 'validator') => void;
}

const RoleSelection: React.FC<RoleSelectionProps> = ({ onRoleSelected }) => {
  const [selectedRole, setSelectedRole] = useState<'citizen' | 'validator' | null>(null);
  const [isVerifying, setIsVerifying] = useState(false);

  const handleVerification = async () => {
    if (!selectedRole) return;
    
    setIsVerifying(true);
    
    // Симуляция ZKP верификации
    setTimeout(() => {
      onRoleSelected(selectedRole);
      setIsVerifying(false);
    }, 2000);
  };

  return (
    <div className="card" style={{ maxWidth: '900px', margin: '0 auto' }}>
      <div style={{ 
        background: '#fef3c7', 
        padding: '16px', 
        borderRadius: '8px', 
        marginBottom: '24px',
        border: '2px solid #f59e0b'
      }}>
        <h4 style={{ color: '#92400e', marginBottom: '8px', display: 'flex', alignItems: 'center', gap: '8px' }}>
          <AlertTriangle size={20} />
          ⚠️ ВАЖНО: Выбор роли необратим!
        </h4>
        <p style={{ color: '#78350f', margin: 0 }}>
          После ZKP-верификации вы должны выбрать ОДНУ роль: Гражданин ИЛИ Валидатор. 
          Эти роли взаимоисключающие согласно принципу "один человек - одна верифицированная роль".
          Изменить роль можно только через механизм миграции при утере доступа.
        </p>
      </div>

      <h3 style={{ marginBottom: '24px', textAlign: 'center' }}>
        Выберите свою роль в протоколе Volnix
      </h3>

      <div style={{ 
        display: 'grid', 
        gridTemplateColumns: '1fr 1fr', 
        gap: '24px',
        marginBottom: '24px'
      }}>
        {/* Гражданин */}
        <div 
          onClick={() => setSelectedRole('citizen')}
          style={{
            border: `3px solid ${selectedRole === 'citizen' ? '#10b981' : '#e5e7eb'}`,
            borderRadius: '12px',
            padding: '24px',
            cursor: 'pointer',
            background: selectedRole === 'citizen' ? '#f0fdf4' : 'white',
            transition: 'all 0.3s ease',
            position: 'relative'
          }}
        >
          {selectedRole === 'citizen' && (
            <div style={{ 
              position: 'absolute', 
              top: '12px', 
              right: '12px',
              color: '#10b981'
            }}>
              <CheckCircle size={24} />
            </div>
          )}
          
          <div style={{ textAlign: 'center', marginBottom: '16px' }}>
            <Shield size={48} style={{ color: '#10b981', margin: '0 auto' }} />
            <h4 style={{ margin: '12px 0 8px 0', fontSize: '1.5rem' }}>Гражданин</h4>
            <p style={{ color: '#6b7280', fontSize: '14px' }}>Продавец прав на производительность</p>
          </div>

          <div style={{ marginBottom: '16px' }}>
            <h5 style={{ color: '#10b981', marginBottom: '8px' }}>✓ Доходы:</h5>
            <ul style={{ margin: 0, paddingLeft: '20px', fontSize: '14px', color: '#374151' }}>
              <li>Автоматическое начисление 10 ANT/день</li>
              <li>Продажа ANT на внутреннем рынке за WRT</li>
              <li>Пассивный доход без инфраструктуры</li>
              <li>Участие в DAO голосовании (через WRT)</li>
            </ul>
          </div>

          <div style={{ marginBottom: '16px' }}>
            <h5 style={{ color: '#ef4444', marginBottom: '8px' }}>⚠ Ограничения:</h5>
            <ul style={{ margin: 0, paddingLeft: '20px', fontSize: '14px', color: '#6b7280' }}>
              <li>Лимит накопления: 1000 ANT</li>
              <li>Нельзя майнить блоки</li>
              <li>Требуется активность раз в год</li>
            </ul>
          </div>

          <div style={{ 
            background: '#f0fdf4', 
            padding: '12px', 
            borderRadius: '6px',
            fontSize: '13px',
            color: '#166534'
          }}>
            <strong>Идеально для:</strong> Пользователей, желающих получать пассивный доход 
            без технических знаний и инфраструктуры
          </div>
        </div>

        {/* Валидатор */}
        <div 
          onClick={() => setSelectedRole('validator')}
          style={{
            border: `3px solid ${selectedRole === 'validator' ? '#f59e0b' : '#e5e7eb'}`,
            borderRadius: '12px',
            padding: '24px',
            cursor: 'pointer',
            background: selectedRole === 'validator' ? '#fefbf3' : 'white',
            transition: 'all 0.3s ease',
            position: 'relative'
          }}
        >
          {selectedRole === 'validator' && (
            <div style={{ 
              position: 'absolute', 
              top: '12px', 
              right: '12px',
              color: '#f59e0b'
            }}>
              <CheckCircle size={24} />
            </div>
          )}
          
          <div style={{ textAlign: 'center', marginBottom: '16px' }}>
            <Crown size={48} style={{ color: '#f59e0b', margin: '0 auto' }} />
            <h4 style={{ margin: '12px 0 8px 0', fontSize: '1.5rem' }}>Валидатор</h4>
            <p style={{ color: '#6b7280', fontSize: '14px' }}>Покупатель прав и оператор узла</p>
          </div>

          <div style={{ marginBottom: '16px' }}>
            <h5 style={{ color: '#10b981', marginBottom: '8px' }}>✓ Доходы:</h5>
            <ul style={{ margin: 0, paddingLeft: '20px', fontSize: '14px', color: '#374151' }}>
              <li><strong>Контур 1:</strong> Пассивный доход от активации LZN</li>
              <li><strong>Контур 2:</strong> Комиссии из выигранных блоков</li>
              <li>Доля от базовой эмиссии WRT</li>
              <li>Максимальный потенциал прибыли</li>
            </ul>
          </div>

          <div style={{ marginBottom: '16px' }}>
            <h5 style={{ color: '#ef4444', marginBottom: '8px' }}>⚠ Требования:</h5>
            <ul style={{ margin: 0, paddingLeft: '20px', fontSize: '14px', color: '#6b7280' }}>
              <li>Покупка и активация LZN токенов</li>
              <li>Покупка ANT для участия в аукционах</li>
              <li>Выполнение MOA (Минимальное Обязательство Активности)</li>
              <li>Техническая инфраструктура (узел)</li>
              <li>Активность каждые 6 месяцев</li>
            </ul>
          </div>

          <div style={{ 
            background: '#fefbf3', 
            padding: '12px', 
            borderRadius: '6px',
            fontSize: '13px',
            color: '#92400e'
          }}>
            <strong>Идеально для:</strong> Технически подкованных участников с капиталом, 
            готовых активно участвовать в консенсусе
          </div>
        </div>
      </div>

      {/* Сравнительная таблица */}
      <div style={{ 
        background: '#f9fafb', 
        padding: '16px', 
        borderRadius: '8px',
        marginBottom: '24px'
      }}>
        <h5 style={{ marginBottom: '12px' }}>Сравнение ролей:</h5>
        <table style={{ width: '100%', fontSize: '14px' }}>
          <thead>
            <tr style={{ borderBottom: '2px solid #d1d5db' }}>
              <th style={{ textAlign: 'left', padding: '8px' }}>Параметр</th>
              <th style={{ textAlign: 'center', padding: '8px', color: '#10b981' }}>Гражданин</th>
              <th style={{ textAlign: 'center', padding: '8px', color: '#f59e0b' }}>Валидатор</th>
            </tr>
          </thead>
          <tbody>
            <tr style={{ borderBottom: '1px solid #e5e7eb' }}>
              <td style={{ padding: '8px' }}>Доступ к ANT</td>
              <td style={{ textAlign: 'center', padding: '8px' }}>Получает (продает)</td>
              <td style={{ textAlign: 'center', padding: '8px' }}>Покупает (использует)</td>
            </tr>
            <tr style={{ borderBottom: '1px solid #e5e7eb' }}>
              <td style={{ padding: '8px' }}>Активация LZN</td>
              <td style={{ textAlign: 'center', padding: '8px' }}>❌ Нет</td>
              <td style={{ textAlign: 'center', padding: '8px' }}>✅ Да (макс 33%)</td>
            </tr>
            <tr style={{ borderBottom: '1px solid #e5e7eb' }}>
              <td style={{ padding: '8px' }}>Майнинг блоков</td>
              <td style={{ textAlign: 'center', padding: '8px' }}>❌ Нет</td>
              <td style={{ textAlign: 'center', padding: '8px' }}>✅ Да (через аукционы)</td>
            </tr>
            <tr style={{ borderBottom: '1px solid #e5e7eb' }}>
              <td style={{ padding: '8px' }}>Требуется инфраструктура</td>
              <td style={{ textAlign: 'center', padding: '8px' }}>❌ Нет</td>
              <td style={{ textAlign: 'center', padding: '8px' }}>✅ Да (узел)</td>
            </tr>
            <tr>
              <td style={{ padding: '8px' }}>Начальный капитал</td>
              <td style={{ textAlign: 'center', padding: '8px' }}>Минимальный</td>
              <td style={{ textAlign: 'center', padding: '8px' }}>Значительный</td>
            </tr>
          </tbody>
        </table>
      </div>

      <button
        className="button"
        onClick={handleVerification}
        disabled={!selectedRole || isVerifying}
        style={{ 
          width: '100%', 
          fontSize: '18px', 
          padding: '16px',
          background: selectedRole === 'citizen' ? '#10b981' : selectedRole === 'validator' ? '#f59e0b' : '#6b7280'
        }}
      >
        {isVerifying ? (
          'Проверка ZKP верификации...'
        ) : selectedRole ? (
          `Подтвердить роль: ${selectedRole === 'citizen' ? 'Гражданин' : 'Валидатор'}`
        ) : (
          'Выберите роль для продолжения'
        )}
      </button>

      <div style={{ 
        marginTop: '16px', 
        fontSize: '13px', 
        color: '#6b7280',
        textAlign: 'center'
      }}>
        🔐 Для верификации используется Zero-Knowledge Proof (ZKP) - ваши личные данные не раскрываются
      </div>
    </div>
  );
};

export default RoleSelection;
