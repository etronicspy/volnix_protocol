import React, { useState, useEffect } from 'react';
import { Wallet, Key, Shield, Trash2, RefreshCw, Copy, Eye, EyeOff, Check } from 'lucide-react';
import { blockchainService } from '../services/blockchainService';
import { getAllWallets, saveWallet, deleteWallet, walletExists, SavedWallet, getWallet } from '../utils/walletStorage';
import { generateMnemonic, validateMnemonic } from '../utils/mnemonicGenerator';

interface WalletConnectProps {
  onConnect: (address: string, mnemonic?: string) => void;
}

const WalletConnect: React.FC<WalletConnectProps> = ({ onConnect }) => {
  const [isConnecting, setIsConnecting] = useState(false);
  const [walletName, setWalletName] = useState('');
  const [mnemonic, setMnemonic] = useState('');
  const [error, setError] = useState('');
  const [savedWallets, setSavedWallets] = useState<SavedWallet[]>([]);
  const [showSavedWallets, setShowSavedWallets] = useState(true);
  const [showMnemonicModal, setShowMnemonicModal] = useState(false);
  const [newWalletMnemonic, setNewWalletMnemonic] = useState('');
  const [newWalletName, setNewWalletName] = useState('');
  const [newWalletAddress, setNewWalletAddress] = useState('');
  const [mnemonicCopied, setMnemonicCopied] = useState(false);
  const [showMnemonic, setShowMnemonic] = useState(false);

  // Загрузка списка сохраненных кошельков
  useEffect(() => {
    loadSavedWallets();
  }, []);

  const loadSavedWallets = () => {
    const wallets = getAllWallets();
    setSavedWallets(wallets);
  };

  const handleCreateWallet = async () => {
    if (!walletName.trim()) {
      setError('Please enter a wallet name');
      return;
    }

    // Проверяем, не существует ли уже кошелек с таким именем
    if (walletExists(walletName.trim())) {
      setError(`Wallet "${walletName.trim()}" already exists. Please choose a different name.`);
      return;
    }
    
    setIsConnecting(true);
    setError('');
    
    try {
      // Генерация новой уникальной мнемоники (синхронная функция)
      const newMnemonic = generateMnemonic();
      
      const address = await blockchainService.initializeSigningClient(newMnemonic);
      
      // Сохраняем кошелек
      saveWallet(walletName.trim(), address, newMnemonic);
      
      // Обновляем список кошельков
      loadSavedWallets();
      
      // Показываем модальное окно с мнемоникой
      setNewWalletMnemonic(newMnemonic);
      setNewWalletName(walletName.trim());
      setNewWalletAddress(address);
      setShowMnemonicModal(true);
      setShowMnemonic(false); // Скрываем мнемонику по умолчанию для безопасности
      setMnemonicCopied(false);
      
      setIsConnecting(false);
    } catch (err: any) {
      setError(err.message || 'Failed to create wallet');
      setIsConnecting(false);
    }
  };

  const handleConnectSavedWallet = async (wallet: SavedWallet) => {
    setIsConnecting(true);
    setError('');
    
    try {
      const address = await blockchainService.initializeSigningClient(wallet.mnemonic);
      onConnect(address, wallet.mnemonic);
      setIsConnecting(false);
    } catch (err: any) {
      setError(err.message || 'Failed to connect wallet');
      setIsConnecting(false);
    }
  };

  const handleDeleteWallet = (walletName: string, e: React.MouseEvent) => {
    e.stopPropagation(); // Предотвращаем подключение при удалении
    
    if (window.confirm(`Are you sure you want to delete wallet "${walletName}"? This action cannot be undone.`)) {
      try {
        deleteWallet(walletName);
        loadSavedWallets();
        setError('');
      } catch (err: any) {
        setError(err.message || 'Failed to delete wallet');
      }
    }
  };

  const handleConnectExisting = async () => {
    if (!mnemonic.trim()) {
      setError('Please enter your mnemonic phrase');
      return;
    }

    // Валидация мнемоники
    if (!validateMnemonic(mnemonic.trim())) {
      setError('Invalid mnemonic phrase. Please check your mnemonic and try again.');
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

  const handleCopyMnemonic = async () => {
    try {
      await navigator.clipboard.writeText(newWalletMnemonic);
      setMnemonicCopied(true);
      setTimeout(() => setMnemonicCopied(false), 2000);
    } catch (err) {
      console.error('Failed to copy mnemonic:', err);
      setError('Failed to copy mnemonic to clipboard');
    }
  };

  const handleCloseMnemonicModal = () => {
    // Сохраняем значения перед очисткой
    const address = newWalletAddress;
    const mnemonic = newWalletMnemonic;
    
    // Закрываем модальное окно и очищаем состояние
    setShowMnemonicModal(false);
    setNewWalletMnemonic('');
    setNewWalletName('');
    setNewWalletAddress('');
    setShowMnemonic(false);
    setMnemonicCopied(false);
    setWalletName(''); // Очищаем имя кошелька
    
    // Подключаем кошелек после закрытия модального окна
    if (address && mnemonic) {
      onConnect(address, mnemonic);
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

      {/* Список сохраненных кошельков */}
      {savedWallets.length > 0 && showSavedWallets && (
        <div style={{ marginBottom: '24px' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '12px' }}>
            <h3 style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
              <Wallet size={20} />
              Saved Wallets ({savedWallets.length})
            </h3>
            <button
              onClick={() => setShowSavedWallets(!showSavedWallets)}
              style={{
                background: 'transparent',
                border: 'none',
                color: '#6b7280',
                cursor: 'pointer',
                fontSize: '14px'
              }}
            >
              {showSavedWallets ? 'Hide' : 'Show'}
            </button>
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
            {savedWallets.map((wallet) => (
              <div
                key={wallet.name}
                onClick={() => handleConnectSavedWallet(wallet)}
                style={{
                  padding: '12px',
                  background: '#f3f4f6',
                  borderRadius: '8px',
                  cursor: 'pointer',
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'center',
                  transition: 'background 0.2s'
                }}
                onMouseEnter={(e) => {
                  e.currentTarget.style.background = '#e5e7eb';
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.background = '#f3f4f6';
                }}
              >
                <div style={{ flex: 1 }}>
                  <div style={{ fontWeight: 600, marginBottom: '4px' }}>{wallet.name}</div>
                  <div style={{ fontSize: '12px', color: '#6b7280', fontFamily: 'monospace' }}>
                    {wallet.address.substring(0, 20)}...
                  </div>
                </div>
                <button
                  onClick={(e) => handleDeleteWallet(wallet.name, e)}
                  style={{
                    background: 'transparent',
                    border: 'none',
                    color: '#dc2626',
                    cursor: 'pointer',
                    padding: '4px',
                    display: 'flex',
                    alignItems: 'center'
                  }}
                  title="Delete wallet"
                >
                  <Trash2 size={16} />
                </button>
              </div>
            ))}
          </div>
          <div style={{ textAlign: 'center', margin: '16px 0' }}>
            <span style={{ color: '#6b7280' }}>or</span>
          </div>
        </div>
      )}

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

      {/* Модальное окно с мнемоникой */}
      {showMnemonicModal && (
        <div style={{
          position: 'fixed',
          top: 0,
          left: 0,
          right: 0,
          bottom: 0,
          background: 'rgba(0, 0, 0, 0.7)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          zIndex: 1000,
          padding: '20px'
        }}>
          <div style={{
            background: 'white',
            borderRadius: '16px',
            padding: '32px',
            maxWidth: '600px',
            width: '100%',
            maxHeight: '90vh',
            overflow: 'auto',
            boxShadow: '0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04)'
          }}>
            <div style={{ textAlign: 'center', marginBottom: '24px' }}>
              <Shield size={48} style={{ color: '#667eea', margin: '0 auto 16px' }} />
              <h2 style={{ fontSize: '1.5rem', marginBottom: '8px' }}>Wallet Created Successfully!</h2>
              <p style={{ color: '#6b7280', fontSize: '14px' }}>
                Save your mnemonic phrase securely. You'll need it to restore your wallet.
              </p>
            </div>

            <div style={{ marginBottom: '20px' }}>
              <div style={{ marginBottom: '8px', fontSize: '14px', fontWeight: 600, color: '#374151' }}>
                Wallet Name:
              </div>
              <div style={{
                padding: '12px',
                background: '#f3f4f6',
                borderRadius: '8px',
                fontFamily: 'monospace',
                fontSize: '14px'
              }}>
                {newWalletName}
              </div>
            </div>

            <div style={{ marginBottom: '20px' }}>
              <div style={{ marginBottom: '8px', fontSize: '14px', fontWeight: 600, color: '#374151' }}>
                Address:
              </div>
              <div style={{
                padding: '12px',
                background: '#f3f4f6',
                borderRadius: '8px',
                fontFamily: 'monospace',
                fontSize: '12px',
                wordBreak: 'break-all'
              }}>
                {newWalletAddress}
              </div>
            </div>

            <div style={{ marginBottom: '20px' }}>
              <div style={{ 
                display: 'flex', 
                justifyContent: 'space-between', 
                alignItems: 'center',
                marginBottom: '8px' 
              }}>
                <div style={{ fontSize: '14px', fontWeight: 600, color: '#374151' }}>
                  Mnemonic Phrase:
                </div>
                <div style={{ display: 'flex', gap: '8px' }}>
                  <button
                    onClick={() => setShowMnemonic(!showMnemonic)}
                    style={{
                      background: 'transparent',
                      border: '1px solid #d1d5db',
                      borderRadius: '6px',
                      padding: '6px 12px',
                      cursor: 'pointer',
                      display: 'flex',
                      alignItems: 'center',
                      gap: '4px',
                      fontSize: '12px',
                      color: '#6b7280'
                    }}
                  >
                    {showMnemonic ? <EyeOff size={16} /> : <Eye size={16} />}
                    {showMnemonic ? 'Hide' : 'Show'}
                  </button>
                  <button
                    onClick={handleCopyMnemonic}
                    style={{
                      background: 'transparent',
                      border: '1px solid #d1d5db',
                      borderRadius: '6px',
                      padding: '6px 12px',
                      cursor: 'pointer',
                      display: 'flex',
                      alignItems: 'center',
                      gap: '4px',
                      fontSize: '12px',
                      color: '#6b7280'
                    }}
                  >
                    {mnemonicCopied ? <Check size={16} /> : <Copy size={16} />}
                    {mnemonicCopied ? 'Copied!' : 'Copy'}
                  </button>
                </div>
              </div>
              <div style={{
                padding: '16px',
                background: showMnemonic ? '#fef3c7' : '#f3f4f6',
                borderRadius: '8px',
                fontFamily: 'monospace',
                fontSize: '14px',
                lineHeight: '1.8',
                minHeight: '120px',
                border: showMnemonic ? '2px solid #fbbf24' : '2px solid transparent',
                wordBreak: 'break-word'
              }}>
                {showMnemonic ? (
                  <div style={{ color: '#92400e' }}>
                    {newWalletMnemonic}
                  </div>
                ) : (
                  <div style={{ color: '#6b7280', textAlign: 'center', padding: '20px 0' }}>
                    Click "Show" to reveal your mnemonic phrase
                  </div>
                )}
              </div>
            </div>

            <div style={{
              padding: '16px',
              background: '#fef2f2',
              borderRadius: '8px',
              border: '1px solid #fecaca',
              marginBottom: '24px'
            }}>
              <div style={{ 
                display: 'flex', 
                alignItems: 'start', 
                gap: '8px',
                color: '#991b1b',
                fontSize: '14px'
              }}>
                <Shield size={20} style={{ flexShrink: 0, marginTop: '2px' }} />
                <div>
                  <strong style={{ display: 'block', marginBottom: '8px' }}>⚠️ Important Security Warning:</strong>
                  <ul style={{ margin: 0, paddingLeft: '20px' }}>
                    <li>Never share your mnemonic phrase with anyone</li>
                    <li>Store it in a safe place (offline is best)</li>
                    <li>If you lose your mnemonic, you cannot recover your wallet</li>
                    <li>Anyone with your mnemonic can access your funds</li>
                  </ul>
                </div>
              </div>
            </div>

            <button
              onClick={handleCloseMnemonicModal}
              className="button"
              style={{ width: '100%' }}
            >
              I've Saved My Mnemonic - Continue
            </button>
          </div>
        </div>
      )}
    </div>
  );
};

export default WalletConnect;