import React, { useState } from 'react';
import { Wallet, Key, Shield } from 'lucide-react';

interface WalletConnectProps {
  onConnect: (address: string) => void;
}

const WalletConnect: React.FC<WalletConnectProps> = ({ onConnect }) => {
  const [isConnecting, setIsConnecting] = useState(false);
  const [walletName, setWalletName] = useState('');

  const handleConnect = async () => {
    if (!walletName.trim()) return;
    
    setIsConnecting(true);
    
    // Симуляция подключения к кошельку
    setTimeout(() => {
      const mockAddress = `volnix1${Math.random().toString(36).substring(2, 15)}`;
      onConnect(mockAddress);
      setIsConnecting(false);
    }, 1500);
  };

  const connectExistingWallet = () => {
    const mockAddress = 'volnix1abc123def456ghi789jkl012mno345pqr678stu';
    onConnect(mockAddress);
  };

  return (
    <div className="card" style={{ maxWidth: '500px', margin: '0 auto' }}>
      <div className="text-center mb-4">
        <Shield size={64} style={{ color: '#667eea', margin: '0 auto 16px' }} />
        <h2 style={{ fontSize: '1.8rem', marginBottom: '8px' }}>Connect Your Wallet</h2>
        <p style={{ color: '#6b7280' }}>
          Connect to your Volnix wallet to manage your tokens
        </p>
      </div>

      <div style={{ marginBottom: '24px' }}>
        <h3 style={{ marginBottom: '16px', display: 'flex', alignItems: 'center', gap: '8px' }}>
          <Key size={20} />
          Create New Wallet
        </h3>
        <input
          type="text"
          className="input"
          placeholder="Enter wallet name"
          value={walletName}
          onChange={(e) => setWalletName(e.target.value)}
        />
        <button
          className="button"
          onClick={handleConnect}
          disabled={isConnecting || !walletName.trim()}
          style={{ width: '100%' }}
        >
          {isConnecting ? 'Creating Wallet...' : 'Create & Connect'}
        </button>
      </div>

      <div style={{ textAlign: 'center', margin: '24px 0' }}>
        <span style={{ color: '#6b7280' }}>or</span>
      </div>

      <div>
        <h3 style={{ marginBottom: '16px', display: 'flex', alignItems: 'center', gap: '8px' }}>
          <Wallet size={20} />
          Connect Existing Wallet
        </h3>
        <button
          className="button"
          onClick={connectExistingWallet}
          style={{ width: '100%', background: '#10b981' }}
        >
          Connect Demo Wallet
        </button>
      </div>

      <div style={{ 
        marginTop: '24px', 
        padding: '16px', 
        background: '#f3f4f6', 
        borderRadius: '8px',
        fontSize: '14px',
        color: '#6b7280'
      }}>
        <strong>Note:</strong> This is a demo interface. In production, this would connect to your actual Volnix wallet using secure key management.
      </div>
    </div>
  );
};

export default WalletConnect;