import React, { useState } from 'react';
import { TrendingUp, TrendingDown, DollarSign, Clock } from 'lucide-react';
import { AntMarketOrder } from '../types/wallet';

interface AntMarketProps {
  walletType: 'citizen' | 'validator';
  antBalance: string;
  wrtBalance: string;
  onCreateOrder: (order: Partial<AntMarketOrder>) => void;
}

const AntMarket: React.FC<AntMarketProps> = ({ 
  walletType, 
  antBalance, 
  wrtBalance,
  onCreateOrder 
}) => {
  const [orderType, setOrderType] = useState<'LIMIT' | 'MARKET'>('LIMIT');
  const [amount, setAmount] = useState('');
  const [price, setPrice] = useState('');

  // Мок данные для демонстрации
  const marketPrice = '0.5'; // WRT за 1 ANT
  const [sellOrders] = useState<AntMarketOrder[]>([
    {
      orderId: '1',
      owner: 'volnix1abc...def',
      orderType: 'LIMIT',
      orderSide: 'SELL',
      antAmount: '100',
      pricePerAnt: '0.48',
      status: 'OPEN',
      createdAt: new Date().toISOString(),
      expiresAt: new Date(Date.now() + 86400000).toISOString()
    },
    {
      orderId: '2',
      owner: 'volnix1ghi...jkl',
      orderType: 'LIMIT',
      orderSide: 'SELL',
      antAmount: '250',
      pricePerAnt: '0.50',
      status: 'OPEN',
      createdAt: new Date().toISOString(),
      expiresAt: new Date(Date.now() + 86400000).toISOString()
    },
    {
      orderId: '3',
      owner: 'volnix1mno...pqr',
      orderType: 'LIMIT',
      orderSide: 'SELL',
      antAmount: '150',
      pricePerAnt: '0.52',
      status: 'OPEN',
      createdAt: new Date().toISOString(),
      expiresAt: new Date(Date.now() + 86400000).toISOString()
    }
  ]);

  const [buyOrders] = useState<AntMarketOrder[]>([
    {
      orderId: '4',
      owner: 'volnix1stu...vwx',
      orderType: 'LIMIT',
      orderSide: 'BUY',
      antAmount: '200',
      pricePerAnt: '0.47',
      status: 'OPEN',
      createdAt: new Date().toISOString(),
      expiresAt: new Date(Date.now() + 86400000).toISOString()
    },
    {
      orderId: '5',
      owner: 'volnix1yza...bcd',
      orderType: 'LIMIT',
      orderSide: 'BUY',
      antAmount: '300',
      pricePerAnt: '0.45',
      status: 'OPEN',
      createdAt: new Date().toISOString(),
      expiresAt: new Date(Date.now() + 86400000).toISOString()
    }
  ]);

  const handleCreateOrder = () => {
    if (!amount || (orderType === 'LIMIT' && !price)) return;

    const newOrder: Partial<AntMarketOrder> = {
      orderType,
      orderSide: walletType === 'citizen' ? 'SELL' : 'BUY',
      antAmount: amount,
      pricePerAnt: orderType === 'LIMIT' ? price : marketPrice,
      status: 'OPEN',
      createdAt: new Date().toISOString(),
      expiresAt: new Date(Date.now() + 86400000).toISOString()
    };

    onCreateOrder(newOrder);
    setAmount('');
    setPrice('');
  };

  const isCitizen = walletType === 'citizen';
  const canTrade = isCitizen ? parseFloat(antBalance) > 0 : parseFloat(wrtBalance) > 0;

  return (
    <div style={{ width: '100%' }}>
      {/* Информационный баннер */}
      <div style={{ 
        background: '#dbeafe', 
        padding: '16px', 
        borderRadius: '8px', 
        marginBottom: '20px',
        border: '2px solid #3b82f6'
      }}>
        <h4 style={{ color: '#1e40af', marginBottom: '8px', display: 'flex', alignItems: 'center', gap: '8px' }}>
          <DollarSign size={20} />
          Внутренний рынок ANT (Контур 2)
        </h4>
        <p style={{ color: '#1e3a8a', margin: 0, fontSize: '14px' }}>
          {isCitizen ? (
            <>
              <strong>Вы - Гражданин (Продавец):</strong> Продавайте свои права на ANT валидаторам за WRT. 
              Вы получаете 10 ANT/день (лимит накопления: 1000 ANT).
            </>
          ) : (
            <>
              <strong>Вы - Валидатор (Покупатель):</strong> Покупайте права на ANT у граждан для участия в аукционах блоков. 
              Помните о MOA - минимальном обязательстве активности!
            </>
          )}
        </p>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '20px', marginBottom: '20px' }}>
        {/* Текущая цена */}
        <div className="card" style={{ background: '#f0fdf4' }}>
          <h5 style={{ marginBottom: '12px', color: '#166534' }}>Рыночная цена ANT</h5>
          <div style={{ fontSize: '2rem', fontWeight: 'bold', color: '#10b981', marginBottom: '8px' }}>
            {marketPrice} WRT
          </div>
          <div style={{ fontSize: '14px', color: '#6b7280' }}>
            за 1 ANT
          </div>
          <div style={{ marginTop: '12px', fontSize: '13px', color: '#059669' }}>
            ↑ +5.2% за 24ч
          </div>
        </div>

        {/* Ваш баланс */}
        <div className="card" style={{ background: isCitizen ? '#fef3c7' : '#fefbf3' }}>
          <h5 style={{ marginBottom: '12px', color: '#92400e' }}>Ваш баланс</h5>
          <div style={{ marginBottom: '8px' }}>
            <span style={{ fontSize: '1.2rem', fontWeight: 'bold' }}>{antBalance} ANT</span>
            {isCitizen && (
              <div style={{ fontSize: '12px', color: '#78350f', marginTop: '4px' }}>
                Лимит: 1000 ANT
              </div>
            )}
          </div>
          <div style={{ fontSize: '14px', color: '#6b7280' }}>
            {wrtBalance} WRT
          </div>
        </div>
      </div>

      {/* Книга ордеров */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '20px', marginBottom: '20px' }}>
        {/* Ордера на продажу */}
        <div className="card">
          <h4 style={{ marginBottom: '16px', display: 'flex', alignItems: 'center', gap: '8px', color: '#ef4444' }}>
            <TrendingDown size={20} />
            Ордера на продажу (ASK)
          </h4>
          <div style={{ fontSize: '12px', color: '#6b7280', marginBottom: '8px', display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: '8px' }}>
            <span>Цена (WRT)</span>
            <span style={{ textAlign: 'right' }}>Количество (ANT)</span>
            <span style={{ textAlign: 'right' }}>Сумма (WRT)</span>
          </div>
          {sellOrders.map((order) => (
            <div 
              key={order.orderId}
              style={{ 
                padding: '8px',
                background: '#fef2f2',
                borderRadius: '4px',
                marginBottom: '4px',
                display: 'grid',
                gridTemplateColumns: '1fr 1fr 1fr',
                gap: '8px',
                fontSize: '14px'
              }}
            >
              <span style={{ color: '#ef4444', fontWeight: '600' }}>{order.pricePerAnt}</span>
              <span style={{ textAlign: 'right' }}>{order.antAmount}</span>
              <span style={{ textAlign: 'right', color: '#6b7280' }}>
                {(parseFloat(order.pricePerAnt) * parseFloat(order.antAmount)).toFixed(2)}
              </span>
            </div>
          ))}
        </div>

        {/* Ордера на покупку */}
        <div className="card">
          <h4 style={{ marginBottom: '16px', display: 'flex', alignItems: 'center', gap: '8px', color: '#10b981' }}>
            <TrendingUp size={20} />
            Ордера на покупку (BID)
          </h4>
          <div style={{ fontSize: '12px', color: '#6b7280', marginBottom: '8px', display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: '8px' }}>
            <span>Цена (WRT)</span>
            <span style={{ textAlign: 'right' }}>Количество (ANT)</span>
            <span style={{ textAlign: 'right' }}>Сумма (WRT)</span>
          </div>
          {buyOrders.map((order) => (
            <div 
              key={order.orderId}
              style={{ 
                padding: '8px',
                background: '#f0fdf4',
                borderRadius: '4px',
                marginBottom: '4px',
                display: 'grid',
                gridTemplateColumns: '1fr 1fr 1fr',
                gap: '8px',
                fontSize: '14px'
              }}
            >
              <span style={{ color: '#10b981', fontWeight: '600' }}>{order.pricePerAnt}</span>
              <span style={{ textAlign: 'right' }}>{order.antAmount}</span>
              <span style={{ textAlign: 'right', color: '#6b7280' }}>
                {(parseFloat(order.pricePerAnt) * parseFloat(order.antAmount)).toFixed(2)}
              </span>
            </div>
          ))}
        </div>
      </div>

      {/* Форма создания ордера */}
      <div className="card" style={{ background: isCitizen ? '#f0fdf4' : '#fefbf3' }}>
        <h4 style={{ marginBottom: '16px' }}>
          {isCitizen ? 'Продать ANT' : 'Купить ANT'}
        </h4>

        <div style={{ marginBottom: '16px' }}>
          <label style={{ display: 'block', marginBottom: '8px', fontWeight: '600' }}>
            Тип ордера
          </label>
          <div style={{ display: 'flex', gap: '12px' }}>
            <button
              className="button"
              onClick={() => setOrderType('LIMIT')}
              style={{ 
                background: orderType === 'LIMIT' ? '#3b82f6' : '#6b7280',
                flex: 1
              }}
            >
              Лимитный
            </button>
            <button
              className="button"
              onClick={() => setOrderType('MARKET')}
              style={{ 
                background: orderType === 'MARKET' ? '#3b82f6' : '#6b7280',
                flex: 1
              }}
            >
              Рыночный
            </button>
          </div>
        </div>

        <div style={{ marginBottom: '16px' }}>
          <label style={{ display: 'block', marginBottom: '8px', fontWeight: '600' }}>
            Количество ANT
          </label>
          <input
            type="number"
            className="input"
            placeholder="0.00"
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
            step="0.01"
            min="0"
            max={isCitizen ? antBalance : undefined}
          />
          {isCitizen && (
            <div style={{ fontSize: '14px', color: '#6b7280', marginTop: '4px' }}>
              Доступно: {antBalance} ANT
            </div>
          )}
        </div>

        {orderType === 'LIMIT' && (
          <div style={{ marginBottom: '16px' }}>
            <label style={{ display: 'block', marginBottom: '8px', fontWeight: '600' }}>
              Цена за 1 ANT (в WRT)
            </label>
            <input
              type="number"
              className="input"
              placeholder="0.00"
              value={price}
              onChange={(e) => setPrice(e.target.value)}
              step="0.01"
              min="0"
            />
            <div style={{ fontSize: '14px', color: '#6b7280', marginTop: '4px' }}>
              Рыночная цена: {marketPrice} WRT
            </div>
          </div>
        )}

        <div style={{ 
          background: '#f9fafb', 
          padding: '12px', 
          borderRadius: '6px',
          marginBottom: '16px'
        }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '4px' }}>
            <span>Количество:</span>
            <span>{amount || '0'} ANT</span>
          </div>
          <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '4px' }}>
            <span>Цена:</span>
            <span>{orderType === 'LIMIT' ? (price || '0') : marketPrice} WRT</span>
          </div>
          <hr style={{ margin: '8px 0', border: 'none', borderTop: '1px solid #d1d5db' }} />
          <div style={{ display: 'flex', justifyContent: 'space-between', fontWeight: '600' }}>
            <span>Итого:</span>
            <span>
              {(parseFloat(amount || '0') * parseFloat(orderType === 'LIMIT' ? (price || '0') : marketPrice)).toFixed(2)} WRT
            </span>
          </div>
        </div>

        <button
          className="button"
          onClick={handleCreateOrder}
          disabled={!canTrade || !amount || (orderType === 'LIMIT' && !price)}
          style={{ 
            width: '100%',
            background: isCitizen ? '#10b981' : '#f59e0b'
          }}
        >
          {isCitizen ? 'Разместить ордер на продажу' : 'Разместить ордер на покупку'}
        </button>

        <div style={{ 
          marginTop: '12px', 
          fontSize: '13px', 
          color: '#6b7280',
          textAlign: 'center'
        }}>
          <Clock size={14} style={{ display: 'inline', marginRight: '4px' }} />
          Ордер действителен 24 часа
        </div>
      </div>

      {/* Статистика рынка */}
      <div className="card" style={{ background: '#f9fafb', marginTop: '20px' }}>
        <h5 style={{ marginBottom: '12px' }}>Статистика рынка за 24ч</h5>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '16px', fontSize: '14px' }}>
          <div>
            <div style={{ color: '#6b7280', marginBottom: '4px' }}>Объем торгов</div>
            <div style={{ fontWeight: '600' }}>12,450 ANT</div>
          </div>
          <div>
            <div style={{ color: '#6b7280', marginBottom: '4px' }}>Макс. цена</div>
            <div style={{ fontWeight: '600', color: '#10b981' }}>0.55 WRT</div>
          </div>
          <div>
            <div style={{ color: '#6b7280', marginBottom: '4px' }}>Мин. цена</div>
            <div style={{ fontWeight: '600', color: '#ef4444' }}>0.42 WRT</div>
          </div>
          <div>
            <div style={{ color: '#6b7280', marginBottom: '4px' }}>Сделок</div>
            <div style={{ fontWeight: '600' }}>1,247</div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default AntMarket;
