import React, { useState } from 'react';
import { Crown, Server, Key, Activity, Shield, TrendingUp } from 'lucide-react';

interface ValidatorWallet {
  name: string;
  address: string;
  balance: {
    wrt: number;
    lzn: number;
  };
  staked: number;
  created: Date;
}

interface NodeConnection {
  connected: boolean;
  connectedAt: Date;
  rewardsEarned: number;
  blocksValidated: number;
}

interface ValidatorManagementProps {
  onValidatorCreated?: (nodeId: number, wallet: ValidatorWallet) => void;
}

const ValidatorManagement: React.FC<ValidatorManagementProps> = ({ onValidatorCreated }) => {
  const [validatorWallets, setValidatorWallets] = useState<{ [key: number]: ValidatorWallet }>({});
  const [nodeConnections, setNodeConnections] = useState<{ [key: number]: NodeConnection }>({});
  const [creatingWallet, setCreatingWallet] = useState<number | null>(null);
  const [connectingNode, setConnectingNode] = useState<number | null>(null);
  const [walletNames, setWalletNames] = useState<{ [key: number]: string }>({
    0: 'validator-node-0',
    1: 'validator-node-1',
    2: 'validator-node-2'
  });

  const validators = [
    { id: 0, name: 'Node-0 Validator', port: 26650, status: 'Active' },
    { id: 1, name: 'Node-1 Validator', port: 26651, status: 'Active' },
    { id: 2, name: 'Node-2 Validator', port: 26652, status: 'Active' }
  ];

  const createValidatorWallet = async (nodeId: number) => {
    const walletName = walletNames[nodeId];
    if (!walletName.trim()) {
      alert('Please enter a wallet name for the validator');
      return;
    }

    setCreatingWallet(nodeId);

    // Simulate wallet creation
    setTimeout(() => {
      const mockAddress = `volnix1val${nodeId}${Math.random().toString(36).substring(2, 15)}validator`;
      
      const newWallet: ValidatorWallet = {
        name: walletName,
        address: mockAddress,
        balance: {
          wrt: nodeId === 0 ? 1000 : nodeId === 1 ? 1200 : 800,
          lzn: nodeId === 0 ? 500 : nodeId === 1 ? 750 : 600
        },
        staked: nodeId === 0 ? 500 : nodeId === 1 ? 750 : 600,
        created: new Date()
      };

      setValidatorWallets(prev => ({
        ...prev,
        [nodeId]: newWallet
      }));

      setCreatingWallet(null);
      onValidatorCreated?.(nodeId, newWallet);

      alert(`🎉 Validator wallet created successfully!\n\n` +
            `📛 Name: ${walletName}\n` +
            `📍 Address: ${mockAddress.substring(0, 20)}...\n` +
            `💰 Initial Balance: ${newWallet.balance.wrt} WRT, ${newWallet.balance.lzn} LZN\n` +
            `🔒 Auto-staked: ${newWallet.staked} LZN\n\n` +
            `⚠️ IMPORTANT: Connect wallet to node to start earning rewards!`);
    }, 2000);
  };

  const stakeAllValidators = () => {
    const createdValidators = Object.keys(validatorWallets);
    if (createdValidators.length === 0) {
      alert('❌ No validator wallets created yet!\n\nPlease create validator wallets first.');
      return;
    }

    let stakeInfo = '🔒 Auto-staking all validators:\n\n';
    createdValidators.forEach(nodeId => {
      const validator = validatorWallets[parseInt(nodeId)];
      stakeInfo += `• Node-${nodeId}: ${validator.staked} LZN staked\n`;
    });
    stakeInfo += '\n✅ All validators are now participating in consensus!';
    
    alert(stakeInfo);
  };

  const checkValidatorStatus = () => {
    const createdValidators = Object.keys(validatorWallets);
    if (createdValidators.length === 0) {
      alert('❌ No validator wallets created yet!');
      return;
    }

    let statusInfo = '🌐 Network Validator Status:\n\n';
    statusInfo += `📊 Active Validators: ${createdValidators.length}/3\n`;
    statusInfo += `⚡ Network Health: ${createdValidators.length === 3 ? '100%' : Math.round((createdValidators.length/3)*100) + '%'}\n`;
    statusInfo += `🔥 Consensus: ${createdValidators.length >= 2 ? 'Active (PoVB)' : 'Waiting for validators'}\n\n`;
    
    createdValidators.forEach(nodeId => {
      const validator = validatorWallets[parseInt(nodeId)];
      statusInfo += `🟢 Node-${nodeId}: ${validator.name}\n`;
      statusInfo += `   Staked: ${validator.staked} LZN\n`;
      statusInfo += `   Status: Active & Validating\n\n`;
    });

    alert(statusInfo);
  };

  const exportValidatorKeys = () => {
    const createdValidators = Object.keys(validatorWallets);
    if (createdValidators.length === 0) {
      alert('❌ No validator wallets to export!');
      return;
    }

    let exportData = '🔐 Validator Keys Export:\n\n';
    createdValidators.forEach(nodeId => {
      const validator = validatorWallets[parseInt(nodeId)];
      exportData += `Node-${nodeId} (${validator.name}):\n`;
      exportData += `Address: ${validator.address}\n`;
      exportData += `Created: ${validator.created.toLocaleString()}\n`;
      exportData += `Staked: ${validator.staked} LZN\n\n`;
    });
    
    exportData += '⚠️ In production, this would export actual private keys securely.';
    alert(exportData);
  };

  const connectWalletToNode = async (nodeId: number) => {
    const wallet = validatorWallets[nodeId];
    if (!wallet) {
      alert('❌ No wallet found for this node!');
      return;
    }

    setConnectingNode(nodeId);

    // Simulate connection process
    setTimeout(() => {
      const newConnection: NodeConnection = {
        connected: true,
        connectedAt: new Date(),
        rewardsEarned: 0,
        blocksValidated: 0
      };

      setNodeConnections(prev => ({
        ...prev,
        [nodeId]: newConnection
      }));

      setConnectingNode(null);

      // Start earning simulation
      startEarningRewards(nodeId);

      alert(`🎉 Wallet successfully connected to Node-${nodeId}!\n\n` +
            `✅ Node is now authorized to earn rewards\n` +
            `💰 Rewards will be sent to: ${wallet.address.substring(0, 20)}...\n` +
            `🔥 Participating in PoVB consensus\n` +
            `📊 Block validation rewards: ~5-10 WRT per block\n\n` +
            `🚀 Your validator is now fully operational!`);
    }, 3000);
  };

  const startEarningRewards = (nodeId: number) => {
    // Simulate earning rewards every 10 seconds
    const interval = setInterval(() => {
      const connection = nodeConnections[nodeId];
      if (connection && connection.connected) {
        const reward = Math.floor(Math.random() * 6) + 5;
        
        setNodeConnections(prev => ({
          ...prev,
          [nodeId]: {
            ...prev[nodeId],
            rewardsEarned: prev[nodeId].rewardsEarned + reward,
            blocksValidated: prev[nodeId].blocksValidated + 1
          }
        }));

        setValidatorWallets(prev => ({
          ...prev,
          [nodeId]: {
            ...prev[nodeId],
            balance: {
              ...prev[nodeId].balance,
              wrt: prev[nodeId].balance.wrt + reward
            }
          }
        }));
      } else {
        clearInterval(interval);
      }
    }, 10000);
  };

  const disconnectWalletFromNode = (nodeId: number) => {
    const connection = nodeConnections[nodeId];
    if (!connection || !connection.connected) {
      alert('❌ Wallet is not connected to this node!');
      return;
    }

    if (window.confirm(`⚠️ Disconnect wallet from Node-${nodeId}?\n\nThis will stop earning rewards until reconnected.`)) {
      setNodeConnections(prev => ({
        ...prev,
        [nodeId]: {
          ...prev[nodeId],
          connected: false
        }
      }));

      alert(`🔌 Wallet disconnected from Node-${nodeId}\n\n❌ No longer earning rewards\n⚠️ Node continues validating but rewards are not distributed`);
    }
  };

  const viewConsensusActivity = () => {
    const createdValidators = Object.keys(validatorWallets);
    if (createdValidators.length === 0) {
      alert('❌ No validator wallets created yet!');
      return;
    }

    let consensusInfo = '⚖️ PoVB Consensus Activity:\n\n';
    consensusInfo += `🔥 Current Round: #2,468\n`;
    consensusInfo += `⏱️ Time Remaining: 3m 45s\n`;
    consensusInfo += `🏆 Leading: Node-1 (150 ANT burned)\n\n`;
    
    consensusInfo += '📊 Validator Participation:\n';
    createdValidators.forEach(nodeIdStr => {
      const nodeId = parseInt(nodeIdStr);
      const connection = nodeConnections[nodeId];
      const isConnected = connection && connection.connected;
      const burnAmount = Math.floor(Math.random() * 100) + 50;
      
      consensusInfo += `• Node-${nodeId}: ${burnAmount} ANT burned`;
      if (isConnected) {
        consensusInfo += ` ✅ (Earning rewards)`;
      } else {
        consensusInfo += ` ⚠️ (No rewards - wallet not connected)`;
      }
      consensusInfo += '\n';
    });
    
    const connectedCount = Object.values(nodeConnections).filter(c => c && c.connected).length;
    consensusInfo += `\n💰 Earning validators: ${connectedCount}/${createdValidators.length}`;
    alert(consensusInfo);
  };

  return (
    <div style={{ width: '100%' }}>
      <div style={{ 
        background: '#fef3c7', 
        padding: '16px', 
        borderRadius: '8px', 
        marginBottom: '20px' 
      }}>
        <h4 style={{ color: '#92400e', marginBottom: '8px', display: 'flex', alignItems: 'center', gap: '8px' }}>
          <Server size={20} />
          Validator Wallet Management
        </h4>
        <p style={{ color: '#78350f', margin: '0 0 12px 0' }}>
          Create validator wallets and connect them to nodes to start earning rewards from block validation.
        </p>
        <div style={{ 
          display: 'grid', 
          gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))', 
          gap: '12px',
          fontSize: '12px'
        }}>
          <div style={{ color: '#78350f' }}>
            <strong>Step 1:</strong> Create wallet
          </div>
          <div style={{ color: '#78350f' }}>
            <strong>Step 2:</strong> Connect to node
          </div>
          <div style={{ color: '#78350f' }}>
            <strong>Step 3:</strong> Start earning rewards
          </div>
        </div>
      </div>

      <div style={{ 
        display: 'grid', 
        gridTemplateColumns: 'repeat(auto-fit, minmax(300px, 1fr))', 
        gap: '20px', 
        marginBottom: '20px' 
      }}>
        {validators.map((validator) => {
          const wallet = validatorWallets[validator.id];
          const connection = nodeConnections[validator.id];
          const isCreated = !!wallet;
          const isCreating = creatingWallet === validator.id;
          const isConnecting = connectingNode === validator.id;
          const isConnected = connection && connection.connected;

          return (
            <div 
              key={validator.id}
              style={{
                border: `2px solid ${isConnected ? '#10b981' : isCreated ? '#f59e0b' : '#e5e7eb'}`,
                borderRadius: '12px',
                padding: '20px',
                background: isConnected ? '#f0fdf4' : isCreated ? '#fefbf3' : 'white'
              }}
            >
              <div style={{ 
                display: 'flex', 
                justifyContent: 'space-between', 
                alignItems: 'center', 
                marginBottom: '12px' 
              }}>
                <div>
                  <div style={{ fontSize: '24px', marginBottom: '8px' }}>
                    {isCreated ? '🟢' : '⚪'}
                  </div>
                  <h4 style={{ margin: 0 }}>{validator.name}</h4>
                  <p style={{ color: '#6b7280', fontSize: '14px', margin: '4px 0' }}>
                    Port: {validator.port} • Status: {validator.status}
                  </p>
                </div>
                <div style={{ textAlign: 'right' }}>
                  <div style={{ 
                    fontSize: '12px', 
                    fontWeight: '600',
                    color: isConnected ? '#10b981' : isCreated ? '#f59e0b' : '#ef4444'
                  }}>
                    {isConnected ? 'CONNECTED & EARNING' : isCreated ? 'WALLET CREATED' : 'NO WALLET'}
                  </div>
                </div>
              </div>
              
              {!isCreated && (
                <div style={{ margin: '12px 0' }}>
                  <input
                    type="text"
                    className="input"
                    placeholder="Validator wallet name"
                    value={walletNames[validator.id]}
                    onChange={(e) => setWalletNames(prev => ({
                      ...prev,
                      [validator.id]: e.target.value
                    }))}
                    style={{ marginBottom: '8px' }}
                  />
                  <button
                    className="button"
                    onClick={() => createValidatorWallet(validator.id)}
                    disabled={isCreating}
                    style={{ width: '100%' }}
                  >
                    {isCreating ? (
                      <>
                        <Activity size={16} className="animate-spin" />
                        Creating...
                      </>
                    ) : (
                      <>
                        <Key size={16} />
                        Create Validator Wallet
                      </>
                    )}
                  </button>
                </div>
              )}

              {isCreated && !isConnected && (
                <div style={{ margin: '12px 0' }}>
                  <button
                    className="button"
                    onClick={() => connectWalletToNode(validator.id)}
                    disabled={isConnecting}
                    style={{ width: '100%', background: '#f59e0b' }}
                  >
                    {isConnecting ? (
                      <>
                        <Activity size={16} className="animate-spin" />
                        Connecting...
                      </>
                    ) : (
                      <>
                        🔗 Connect to Node-{validator.id}
                      </>
                    )}
                  </button>
                </div>
              )}

              {isConnected && (
                <div style={{ margin: '12px 0' }}>
                  <button
                    className="button"
                    onClick={() => disconnectWalletFromNode(validator.id)}
                    style={{ width: '100%', background: '#ef4444' }}
                  >
                    🔌 Disconnect from Node
                  </button>
                </div>
              )}
              
              {isCreated && (
                <div style={{ 
                  background: isConnected ? '#f0fdf4' : '#fefbf3', 
                  padding: '12px', 
                  borderRadius: '6px', 
                  marginTop: '12px' 
                }}>
                  <div style={{ fontSize: '12px', color: isConnected ? '#166534' : '#92400e' }}>
                    <div style={{ marginBottom: '4px' }}>
                      <strong>Address:</strong> 
                      <span style={{ fontFamily: 'monospace', marginLeft: '4px' }}>
                        {wallet.address.substring(0, 20)}...
                      </span>
                    </div>
                    <div style={{ marginBottom: '4px' }}>
                      <strong>Balance:</strong> {wallet.balance.wrt} WRT, {wallet.balance.lzn} LZN
                    </div>
                    <div style={{ marginBottom: '4px' }}>
                      <strong>Staked:</strong> {wallet.staked} LZN
                    </div>
                    {isConnected && connection && (
                      <>
                        <div style={{ marginTop: '8px', paddingTop: '8px', borderTop: '1px solid #d1d5db' }}>
                          <div style={{ marginBottom: '4px' }}>
                            <strong>Status:</strong> Connected & Earning Rewards
                          </div>
                          <div style={{ marginBottom: '4px' }}>
                            <strong>Connected:</strong> {connection.connectedAt.toLocaleTimeString()}
                          </div>
                          <div style={{ marginBottom: '4px' }}>
                            <strong>Rewards Earned:</strong> {connection.rewardsEarned} WRT
                          </div>
                          <div>
                            <strong>Blocks Validated:</strong> {connection.blocksValidated}
                          </div>
                        </div>
                      </>
                    )}
                  </div>
                </div>
              )}
            </div>
          );
        })}
      </div>

      <div className="card" style={{ background: '#f0f9ff' }}>
        <h4 style={{ 
          color: '#1e40af', 
          marginBottom: '12px',
          display: 'flex',
          alignItems: 'center',
          gap: '8px'
        }}>
          <Crown size={20} />
          Validator Management Actions
        </h4>
        <div style={{ 
          display: 'grid', 
          gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', 
          gap: '12px' 
        }}>
          <button 
            className="button" 
            onClick={stakeAllValidators}
            style={{ background: '#10b981' }}
          >
            <TrendingUp size={16} />
            Auto-Stake All Validators
          </button>
          <button 
            className="button" 
            onClick={checkValidatorStatus}
            style={{ background: '#3b82f6' }}
          >
            <Activity size={16} />
            Check Network Status
          </button>
          <button 
            className="button" 
            onClick={exportValidatorKeys}
            style={{ background: '#8b5cf6' }}
          >
            <Key size={16} />
            Export Validator Keys
          </button>
          <button 
            className="button" 
            onClick={viewConsensusActivity}
            style={{ background: '#f59e0b' }}
          >
            <Shield size={16} />
            View Consensus Activity
          </button>
        </div>
      </div>
    </div>
  );
};

export default ValidatorManagement;