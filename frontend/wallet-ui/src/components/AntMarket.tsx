import React, { useState, useEffect, useCallback } from 'react';
import { TrendingUp, TrendingDown, DollarSign, Clock } from 'lucide-react';
import { AntMarketOrder } from '../types/wallet';
import { blockchainService } from '../services/blockchainService';

interface AntMarketProps {
  walletType: 'citizen' | 'validator';
  antBalance: string;
  wrtBalance: string;
  onCreateOrder: (order: Partial<AntMarketOrder>) => void;
}

function mapApiOrderToAntMarketOrder(o: any): AntMarketOrder {
  const statusMap: Record<string, string> = {
    ORDER_STATUS_OPEN: 'OPEN',
    ORDER_STATUS_PARTIALLY_FILLED: 'PARTIAL',
    ORDER_STATUS_FILLED: 'FILLED',
    ORDER_STATUS_CANCELLED: 'CANCELLED',
    ORDER_STATUS_EXPIRED: 'CANCELLED',
    '1': 'OPEN',
    '2': 'PARTIAL',
    '3': 'FILLED',
    '4': 'CANCELLED',
    '5': 'CANCELLED',
  };
  const status = typeof o.status === 'number' ? statusMap[String(o.status)] : statusMap[o.status] || 'OPEN';
  return {
    orderId: String(o.order_id || o.orderId || ''),
    owner: o.owner || '',
    orderType: (o.order_type === 'ORDER_TYPE_MARKET' || o.orderType === 'MARKET') ? 'MARKET' : 'LIMIT',
    orderSide: (o.order_side === 'ORDER_SIDE_BUY' || o.orderSide === 'BUY') ? 'BUY' : 'SELL',
    antAmount: o.ant_amount || o.antAmount || '0',
    pricePerAnt: o.price || o.pricePerAnt || '0',
    status: status as AntMarketOrder['status'],
    createdAt: o.created_at || o.createdAt || new Date().toISOString(),
    expiresAt: o.expires_at || o.expiresAt || new Date(Date.now() + 86400000).toISOString(),
    filledAmount: o.filled_amount || o.filledAmount,
  };
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
  const [sellOrders, setSellOrders] = useState<AntMarketOrder[]>([]);
  const [buyOrders, setBuyOrders] = useState<AntMarketOrder[]>([]);
  const [marketPrice, setMarketPrice] = useState('0.5');
  const [stats, setStats] = useState({ volume24h: '0', maxPrice: '0', minPrice: '0', tradesCount: 0 });
  const [loading, setLoading] = useState(true);

  const fetchOrders = useCallback(async () => {
    setLoading(true);
    try {
      const { orders } = await blockchainService.getAntOrders();
      const open = orders.filter((o: any) => {
        const s = o.status;
        return s === 1 || s === 'ORDER_STATUS_OPEN' || s === 'OPEN';
      });
      const sells = open.filter((o: any) => o.order_side === 'ORDER_SIDE_SELL' || o.orderSide === 'SELL' || o.order_side === 2);
      const buys = open.filter((o: any) => o.order_side === 'ORDER_SIDE_BUY' || o.orderSide === 'BUY' || o.order_side === 1);
      setSellOrders(sells.map(mapApiOrderToAntMarketOrder));
      setBuyOrders(buys.map(mapApiOrderToAntMarketOrder));
      const prices = open.map((o: any) => parseFloat(o.price || o.pricePerAnt || '0')).filter((p) => !isNaN(p) && p > 0);
      const vol = orders.reduce((a: number, o: any) => a + parseFloat(o.ant_amount || o.antAmount || '0') * parseFloat(o.price || o.pricePerAnt || '0'), 0);
      if (prices.length > 0) {
        const mid = (Math.min(...prices) + Math.max(...prices)) / 2;
        setMarketPrice(mid.toFixed(2));
        setStats({
          volume24h: vol.toFixed(0),
          maxPrice: Math.max(...prices).toFixed(2),
          minPrice: Math.min(...prices).toFixed(2),
          tradesCount: open.length,
        });
      } else {
        setStats({
          volume24h: vol.toFixed(0),
          maxPrice: '0',
          minPrice: '0',
          tradesCount: 0,
        });
      }
    } catch {
      setSellOrders([]);
      setBuyOrders([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchOrders();
    const interval = setInterval(fetchOrders, 15000);
    return () => clearInterval(interval);
  }, [fetchOrders]);

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
            {loading ? '...' : `${marketPrice} WRT`}
          </div>
          <div style={{ fontSize: '14px', color: '#6b7280' }}>
            за 1 ANT
          </div>
          {!loading && sellOrders.length + buyOrders.length > 0 && (
            <div style={{ marginTop: '12px', fontSize: '13px', color: '#059669' }}>
              {sellOrders.length + buyOrders.length} открытых ордеров
            </div>
          )}
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
          {loading ? (
            <div style={{ padding: '16px', color: '#6b7280', textAlign: 'center' }}>Загрузка...</div>
          ) : sellOrders.length === 0 ? (
            <div style={{ padding: '16px', color: '#6b7280', textAlign: 'center' }}>Нет ордеров на продажу</div>
          ) : (
          sellOrders.map((order) => (
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
          )))}
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
          {loading ? (
            <div style={{ padding: '16px', color: '#6b7280', textAlign: 'center' }}>Загрузка...</div>
          ) : buyOrders.length === 0 ? (
            <div style={{ padding: '16px', color: '#6b7280', textAlign: 'center' }}>Нет ордеров на покупку</div>
          ) : (
          buyOrders.map((order) => (
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
          )))}
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
        <h5 style={{ marginBottom: '12px' }}>Статистика рынка</h5>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '16px', fontSize: '14px' }}>
          <div>
            <div style={{ color: '#6b7280', marginBottom: '4px' }}>Объём (WRT)</div>
            <div style={{ fontWeight: '600' }}>{loading ? '...' : stats.volume24h}</div>
          </div>
          <div>
            <div style={{ color: '#6b7280', marginBottom: '4px' }}>Макс. цена</div>
            <div style={{ fontWeight: '600', color: '#10b981' }}>{loading ? '...' : `${stats.maxPrice} WRT`}</div>
          </div>
          <div>
            <div style={{ color: '#6b7280', marginBottom: '4px' }}>Мин. цена</div>
            <div style={{ fontWeight: '600', color: '#ef4444' }}>{loading ? '...' : `${stats.minPrice} WRT`}</div>
          </div>
          <div>
            <div style={{ color: '#6b7280', marginBottom: '4px' }}>Ордеров</div>
            <div style={{ fontWeight: '600' }}>{loading ? '...' : stats.tradesCount}</div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default AntMarket;
