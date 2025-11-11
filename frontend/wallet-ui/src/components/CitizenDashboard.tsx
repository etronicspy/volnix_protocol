import React from 'react';
import { Shield, TrendingUp, Clock, AlertCircle } from 'lucide-react';
import { CitizenInfo } from '../types/wallet';

interface CitizenDashboardProps {
  citizenInfo: CitizenInfo;
}

const CitizenDashboard: React.FC<CitizenDashboardProps> = ({ citizenInfo }) => {
  const accumulationPercentage = (parseFloat(citizenInfo.antAccumulated) / parseFloat(citizenInfo.antLimit)) * 100;
  const isNearLimit = accumulationPercentage >= 80;

  return (
    <div style={{ width: '100%' }}>
      {/* Заголовок */}
      <div className="card" style={{ 
        background: 'linear-gradient(135deg, #10b981 0%, #059669 100%)',
        color: 'white'
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          <Shield size={32} />
          <div>
            <h3 style={{ margin: 0, fontSize: '1.5rem' }}>Панель Гражданина</h3>
            <p style={{ margin: '4px 0 0 0', opacity: 0.9 }}>
              Продавец прав на производительность (ANT)
            </p>
          </div>
        </div>
      </div>

      {/* Информационный баннер */}
      <div style={{ 
        background: '#dbeafe', 
        padding: '16px', 
        borderRadius: '8px', 
        marginBottom: '20px',
        border: '2px solid #3b82f6'
      }}>
        <h4 style={{ color: '#1e40af', marginBottom: '8px' }}>
          💡 Как работает доход Гражданина
        </h4>
        <p style={{ color: '#1e3a8a', margin: 0, fontSize: '14px' }}>
          Вы автоматически получаете <strong>10 ANT каждый день</strong>. Эти права на производительность 
          можно продать валидаторам на внутреннем рынке за WRT. Валидаторы покупают ANT для участия 
          в аукционах блоков и выполнения MOA (Минимального Обязательства Активности).
        </p>
      </div>

      {/* Накопление ANT */}
      <div className="card" style={{ 
        background: isNearLimit ? '#fef3c7' : '#f0fdf4',
        border: isNearLimit ? '2px solid #f59e0b' : '2px solid #10b981'
      }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '12px' }}>
          <h4 style={{ margin: 0 }}>Накопление ANT</h4>
          <div style={{ 
            fontSize: '14px', 
            fontWeight: '600',
            color: isNearLimit ? '#f59e0b' : '#10b981'
          }}>
            {citizenInfo.antAccumulated} / {citizenInfo.antLimit} ANT
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
              background: isNearLimit ? '#f59e0b' : '#10b981',
              height: '100%',
              width: `${Math.min(accumulationPercentage, 100)}%`,
              transition: 'width 0.3s ease',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: 'white',
              fontSize: '12px',
              fontWeight: '600'
            }}>
              {accumulationPercentage.toFixed(1)}%
            </div>
          </div>
        </div>

        {isNearLimit && (
          <div style={{ 
            background: '#fef3c7',
            padding: '12px',
            borderRadius: '6px',
            fontSize: '13px',
            color: '#92400e',
            marginBottom: '12px'
          }}>
            <AlertCircle size={16} style={{ display: 'inline', marginRight: '4px' }} />
            <strong>Внимание:</strong> Вы приближаетесь к лимиту накопления! 
            Продайте ANT на рынке, чтобы продолжать получать ежедневные начисления.
          </div>
        )}

        <div style={{ 
          background: '#f9fafb',
          padding: '12px',
          borderRadius: '6px',
          fontSize: '13px',
          color: '#6b7280'
        }}>
          <strong>Лимит накопления:</strong> Граждане могут накопить максимум 1000 ANT. 
          Это предотвращает концентрацию прав и стимулирует активную торговлю на рынке.
        </div>
      </div>

      {/* Статистика дохода */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '20px', marginBottom: '20px' }}>
        {/* Ежедневное начисление */}
        <div className="card" style={{ background: '#f0fdf4' }}>
          <h5 style={{ marginBottom: '12px', color: '#166534', display: 'flex', alignItems: 'center', gap: '8px' }}>
            <Clock size={20} />
            Ежедневное начисление
          </h5>
          <div style={{ fontSize: '2rem', fontWeight: 'bold', color: '#10b981', marginBottom: '8px' }}>
            {citizenInfo.dailyAntRate} ANT
          </div>
          <div style={{ fontSize: '14px', color: '#6b7280', marginBottom: '12px' }}>
            Автоматически каждый день
          </div>
          <div style={{ fontSize: '13px', color: '#166534', background: '#dcfce7', padding: '8px', borderRadius: '4px' }}>
            Последнее начисление:<br/>
            {new Date(citizenInfo.lastAntAccrual).toLocaleString()}
          </div>
        </div>

        {/* Доход от продаж */}
        <div className="card" style={{ background: '#fef3c7' }}>
          <h5 style={{ marginBottom: '12px', color: '#92400e', display: 'flex', alignItems: 'center', gap: '8px' }}>
            <TrendingUp size={20} />
            Доход от продаж ANT
          </h5>
          <div style={{ fontSize: '2rem', fontWeight: 'bold', color: '#f59e0b', marginBottom: '8px' }}>
            {citizenInfo.incomeFromAntSales} WRT
          </div>
          <div style={{ fontSize: '14px', color: '#6b7280', marginBottom: '12px' }}>
            За все время
          </div>
          <div style={{ fontSize: '13px', color: '#92400e', background: '#fef3c7', padding: '8px', borderRadius: '4px' }}>
            Продано ANT: {citizenInfo.antSoldTotal}<br/>
            Средняя цена: {(parseFloat(citizenInfo.incomeFromAntSales) / parseFloat(citizenInfo.antSoldTotal || '1')).toFixed(2)} WRT/ANT
          </div>
        </div>
      </div>

      {/* Прогноз дохода */}
      <div className="card" style={{ background: '#f0f9ff' }}>
        <h5 style={{ marginBottom: '16px', color: '#1e40af' }}>
          📊 Прогноз дохода
        </h5>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '16px' }}>
          <div>
            <div style={{ fontSize: '14px', color: '#6b7280', marginBottom: '4px' }}>
              За неделю
            </div>
            <div style={{ fontSize: '1.2rem', fontWeight: 'bold', color: '#3b82f6' }}>
              70 ANT
            </div>
            <div style={{ fontSize: '12px', color: '#6b7280' }}>
              ~35 WRT*
            </div>
          </div>
          <div>
            <div style={{ fontSize: '14px', color: '#6b7280', marginBottom: '4px' }}>
              За месяц
            </div>
            <div style={{ fontSize: '1.2rem', fontWeight: 'bold', color: '#3b82f6' }}>
              300 ANT
            </div>
            <div style={{ fontSize: '12px', color: '#6b7280' }}>
              ~150 WRT*
            </div>
          </div>
          <div>
            <div style={{ fontSize: '14px', color: '#6b7280', marginBottom: '4px' }}>
              За год
            </div>
            <div style={{ fontSize: '1.2rem', fontWeight: 'bold', color: '#3b82f6' }}>
              3,650 ANT
            </div>
            <div style={{ fontSize: '12px', color: '#6b7280' }}>
              ~1,825 WRT*
            </div>
          </div>
          <div>
            <div style={{ fontSize: '14px', color: '#6b7280', marginBottom: '4px' }}>
              Текущая цена
            </div>
            <div style={{ fontSize: '1.2rem', fontWeight: 'bold', color: '#10b981' }}>
              0.5 WRT
            </div>
            <div style={{ fontSize: '12px', color: '#6b7280' }}>
              за 1 ANT
            </div>
          </div>
        </div>
        <div style={{ 
          marginTop: '12px', 
          fontSize: '12px', 
          color: '#6b7280',
          fontStyle: 'italic'
        }}>
          * Прогноз основан на текущей рыночной цене ANT. Фактический доход зависит от цены продажи.
        </div>
      </div>

      {/* Стратегии продаж */}
      <div className="card" style={{ background: '#f9fafb' }}>
        <h5 style={{ marginBottom: '16px' }}>💡 Стратегии максимизации дохода</h5>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px', fontSize: '14px' }}>
          <div style={{ 
            background: 'white', 
            padding: '12px', 
            borderRadius: '6px',
            border: '1px solid #e5e7eb'
          }}>
            <h6 style={{ color: '#10b981', marginBottom: '8px' }}>✓ Активная торговля</h6>
            <p style={{ margin: 0, color: '#6b7280', fontSize: '13px' }}>
              Продавайте ANT регулярно по рыночной цене. Стабильный доход без риска достижения лимита.
            </p>
          </div>
          <div style={{ 
            background: 'white', 
            padding: '12px', 
            borderRadius: '6px',
            border: '1px solid #e5e7eb'
          }}>
            <h6 style={{ color: '#3b82f6', marginBottom: '8px' }}>📈 Накопление и продажа</h6>
            <p style={{ margin: 0, color: '#6b7280', fontSize: '13px' }}>
              Накапливайте ANT и продавайте крупными партиями при высоком спросе. Выше риск, но потенциально больше прибыль.
            </p>
          </div>
          <div style={{ 
            background: 'white', 
            padding: '12px', 
            borderRadius: '6px',
            border: '1px solid #e5e7eb'
          }}>
            <h6 style={{ color: '#f59e0b', marginBottom: '8px' }}>⚡ Лимитные ордера</h6>
            <p style={{ margin: 0, color: '#6b7280', fontSize: '13px' }}>
              Размещайте лимитные ордера выше рыночной цены. Ждите, пока валидаторы купят по вашей цене.
            </p>
          </div>
          <div style={{ 
            background: 'white', 
            padding: '12px', 
            borderRadius: '6px',
            border: '1px solid #e5e7eb'
          }}>
            <h6 style={{ color: '#8b5cf6', marginBottom: '8px' }}>📊 Мониторинг рынка</h6>
            <p style={{ margin: 0, color: '#6b7280', fontSize: '13px' }}>
              Следите за активностью сети и спросом валидаторов. Продавайте, когда спрос высокий.
            </p>
          </div>
        </div>
      </div>

      {/* Правила активности */}
      <div className="card" style={{ background: '#fef2f2', border: '2px solid #fecaca' }}>
        <h5 style={{ color: '#dc2626', marginBottom: '12px' }}>
          ⚠️ Правила активности гражданина
        </h5>
        <ul style={{ margin: 0, paddingLeft: '20px', fontSize: '14px', color: '#991b1b' }}>
          <li>Гражданин должен совершить хотя бы одну подписанную транзакцию в течение <strong>1 года</strong></li>
          <li>При неактивности статус гражданина аннулируется</li>
          <li>Все накопленные права на ANT сгорают</li>
          <li>ZKP-идентификатор освобождается для повторной верификации</li>
          <li>Для восстановления требуется новая ZKP-верификация</li>
        </ul>
      </div>

      {/* Преимущества роли */}
      <div className="card" style={{ background: '#f0fdf4', border: '2px solid #10b981' }}>
        <h5 style={{ color: '#166534', marginBottom: '12px' }}>
          ✨ Преимущества роли Гражданина
        </h5>
        <ul style={{ margin: 0, paddingLeft: '20px', fontSize: '14px', color: '#166534' }}>
          <li><strong>Пассивный доход:</strong> Автоматическое начисление 10 ANT/день без усилий</li>
          <li><strong>Нет инфраструктуры:</strong> Не требуется техническое оборудование или знания</li>
          <li><strong>Низкий порог входа:</strong> Минимальные начальные инвестиции</li>
          <li><strong>Гибкость:</strong> Продавайте ANT когда угодно по выгодной цене</li>
          <li><strong>Участие в DAO:</strong> Голосуйте через WRT токены</li>
          <li><strong>Защита от инфляции:</strong> Лимит накопления защищает ценность ANT</li>
        </ul>
      </div>
    </div>
  );
};

export default CitizenDashboard;
