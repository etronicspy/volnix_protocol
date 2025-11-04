import React, { useState } from 'react';
import { Send, ArrowRight } from 'lucide-react';

interface SendTokensProps {
  onSend: (to: string, amount: string, token: string) => void;
}

const SendTokens: React.FC<SendTokensProps> = ({ onSend }) => {
  const [recipient, setRecipient] = useState('');
  const [amount, setAmount] = useState('');
  const [selectedToken, setSelectedToken] = useState('WRT');
  const [isSending, setIsSending] = useState(false);

  const handleSend = async () => {
    if (!recipient || !amount || parseFloat(amount) <= 0) return;

    setIsSending(true);
    await onSend(recipient, amount, selectedToken);
    
    // Очистка формы после отправки
    setRecipient('');
    setAmount('');
    setIsSending(false);
  };

  const tokens = [
    { symbol: 'WRT', name: 'Wealth Rights Token', balance: '1000.50', available: true },
    { symbol: 'LZN', name: 'Lizenz Token', balance: '250.75', available: true },
    { symbol: 'ANT', name: 'Anteil Rights', balance: '0', available: false }
  ];

  const selectedTokenData = tokens.find(t => t.symbol === selectedToken);

  return (
    <div className="card" style={{ maxWidth: '500px', margin: '0 auto' }}>
      <h3 style={{ marginBottom: '24px', display: 'flex', alignItems: 'center', gap: '8px' }}>
        <Send size={24} />
        Send Tokens
      </h3>

      <div style={{ marginBottom: '20px' }}>
        <label style={{ display: 'block', marginBottom: '8px', fontWeight: '600' }}>
          Select Token
        </label>
        <select
          className="input"
          value={selectedToken}
          onChange={(e) => setSelectedToken(e.target.value)}
          style={{ cursor: 'pointer' }}
        >
          {tokens.map((token) => (
            <option 
              key={token.symbol} 
              value={token.symbol}
              disabled={!token.available}
              style={!token.available ? { color: '#9ca3af' } : {}}
            >
              {token.symbol} - {token.name} (Balance: {token.balance})
              {!token.available && ' 🔒 Requires Citizen status'}
            </option>
          ))}
        </select>
      </div>

      <div style={{ marginBottom: '20px' }}>
        <label style={{ display: 'block', marginBottom: '8px', fontWeight: '600' }}>
          Recipient Address
        </label>
        <input
          type="text"
          className="input"
          placeholder="volnix1abc123def456..."
          value={recipient}
          onChange={(e) => setRecipient(e.target.value)}
        />
      </div>

      <div style={{ marginBottom: '20px' }}>
        <label style={{ display: 'block', marginBottom: '8px', fontWeight: '600' }}>
          Amount
        </label>
        <div style={{ position: 'relative' }}>
          <input
            type="number"
            className="input"
            placeholder="0.00"
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
            step="0.01"
            min="0"
          />
          <div style={{ 
            position: 'absolute', 
            right: '16px', 
            top: '50%', 
            transform: 'translateY(-50%)',
            color: '#6b7280',
            fontWeight: '600'
          }}>
            {selectedToken}
          </div>
        </div>
        {selectedTokenData && (
          <div style={{ 
            fontSize: '14px', 
            color: '#6b7280', 
            marginTop: '4px',
            display: 'flex',
            justifyContent: 'space-between'
          }}>
            <span>Available: {selectedTokenData.balance} {selectedToken}</span>
            <button
              type="button"
              onClick={() => setAmount(selectedTokenData.balance)}
              style={{
                background: 'none',
                border: 'none',
                color: '#667eea',
                cursor: 'pointer',
                textDecoration: 'underline'
              }}
            >
              Max
            </button>
          </div>
        )}
      </div>

      <div style={{ 
        background: '#f3f4f6', 
        padding: '16px', 
        borderRadius: '8px', 
        marginBottom: '20px' 
      }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '8px' }}>
          <span>Amount:</span>
          <span>{amount || '0'} {selectedToken}</span>
        </div>
        <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '8px' }}>
          <span>Network Fee:</span>
          <span>0.001 WRT</span>
        </div>
        <hr style={{ margin: '8px 0', border: 'none', borderTop: '1px solid #d1d5db' }} />
        <div style={{ display: 'flex', justifyContent: 'space-between', fontWeight: '600' }}>
          <span>Total:</span>
          <span>{amount || '0'} {selectedToken} + 0.001 WRT</span>
        </div>
      </div>

      <button
        className="button"
        onClick={handleSend}
        disabled={isSending || !recipient || !amount || parseFloat(amount) <= 0}
        style={{ width: '100%', fontSize: '16px', padding: '16px' }}
      >
        {isSending ? (
          'Sending...'
        ) : (
          <>
            Send {selectedToken}
            <ArrowRight size={20} />
          </>
        )}
      </button>

      <div style={{ 
        marginTop: '16px', 
        fontSize: '14px', 
        color: '#6b7280',
        textAlign: 'center'
      }}>
        Double-check the recipient address before sending. Transactions cannot be reversed.
      </div>
    </div>
  );
};

export default SendTokens;