import React, { useState } from 'react';
import { Wallet, Key, Shield } from 'lucide-react';
import { blockchainService } from '../services/blockchainService';

interface WalletConnectProps {
  onConnect: (address: string, mnemonic?: string) => void;
}

const WalletConnect: React.FC<WalletConnectProps> = ({ onConnect }) => {
  const [isConnecting, setIsConnecting] = useState(false);
  const [walletName, setWalletName] = useState('');
  const [mnemonic, setMnemonic] = useState('');
  const [error, setError] = useState('');

  const handleCreateWallet = async () => {
    if (!walletName.trim()) {
      setError('Please enter a wallet name');
      return;
    }
    
    setIsConnecting(true);
    setError('');
    
    try {
      // Генерация новой мнемоники (в продакшене используйте библиотеку для генерации)
      // Для демо используем тестовую мнемонику
      const testMnemonic = 'abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about';
      
      const address = await blockchainService.initializeSigningClient(testMnemonic);
      
      // Сохраняем мнемонику локально (в продакшене используйте безопасное хранилище)
      localStorage.setItem(`wallet_${walletName}`, testMnemonic);
      localStorage.setItem(`wallet_${walletName}_address`, address);
      
      onConnect(address, testMnemonic);
      setIsConnecting(false);
    } catch (err: any) {
      setError(err.message || 'Failed to create wallet');
      setIsConnecting(false);
    }
  };

  const handleConnectExisting = async () => {
    if (!mnemonic.trim()) {
      setError('Please enter your mnemonic phrase');
      return;
    }
    
    setIsConnecting(true);
    setError('');
    
    try {
      const address = await blockchainService.initializeSigningClient(mnemonic.trim());
      onConnect(address, mnemonic.trim());
      setIsConnecting(false);
    } catch (err: any) {
      setError(err.message || 'Failed to connect wallet. Please check your mnemonic.');
      setIsConnecting(false);
    }
  };

  const connectDemoWallet = async () => {
    setIsConnecting(true);
    setError('');
    
    try {
      // Используем тестовую мнемонику для демо
      const demoMnemonic = 'abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about';
      const address = await blockchainService.initializeSigningClient(demoMnemonic);
      onConnect(address, demoMnemonic);
      setIsConnecting(false);
    } catch (err: any) {
      setError(err.message || 'Failed to connect demo wallet');
      setIsConnecting(false);
    }
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
          onClick={handleCreateWallet}
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
        <textarea
          className="input"
          placeholder="Enter your mnemonic phrase (12 or 24 words)"
          value={mnemonic}
          onChange={(e) => setMnemonic(e.target.value)}
          rows={3}
          style={{ fontFamily: 'monospace', fontSize: '12px' }}
        />
        <button
          className="button"
          onClick={handleConnectExisting}
          disabled={isConnecting || !mnemonic.trim()}
          style={{ width: '100%', background: '#10b981', marginTop: '8px' }}
        >
          {isConnecting ? 'Connecting...' : 'Connect Wallet'}
        </button>
      </div>

      <div style={{ textAlign: 'center', margin: '24px 0' }}>
        <span style={{ color: '#6b7280' }}>or</span>
      </div>

      <button
        className="button"
        onClick={connectDemoWallet}
        disabled={isConnecting}
        style={{ width: '100%', background: '#6b7280' }}
      >
        {isConnecting ? 'Connecting...' : 'Connect Demo Wallet'}
      </button>

      {error && (
        <div style={{
          marginTop: '16px',
          padding: '12px',
          background: '#fee2e2',
          borderRadius: '8px',
          color: '#dc2626',
          fontSize: '14px'
        }}>
          {error}
        </div>
      )}

      <div style={{ 
        marginTop: '24px', 
        padding: '16px', 
        background: '#f3f4f6', 
        borderRadius: '8px',
        fontSize: '14px',
        color: '#6b7280'
      }}>
        <strong>Note:</strong> For testing, you can use a test mnemonic. In production, use secure key management.
      </div>
    </div>
  );
};

export default WalletConnect;