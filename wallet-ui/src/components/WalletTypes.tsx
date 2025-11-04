import React from 'react';
import { Users, User, Shield, Crown, CheckCircle, Lock } from 'lucide-react';
import { WalletType } from '../types/wallet';

interface WalletTypesProps {
  currentType: WalletType;
  onUpgrade?: (newType: WalletType) => void;
}

const WalletTypes: React.FC<WalletTypesProps> = ({ currentType, onUpgrade }) => {
  const walletTypes = [
    {
      type: 'guest' as WalletType,
      name: 'Guest',
      icon: <User size={32} />,
      color: '#6b7280',
      description: 'Basic wallet functionality',
      features: [
        'Store and trade WRT and LZN tokens',
        'Basic transaction capabilities',
        'Entry point for all users',
        'No verification required'
      ],
      limitations: [
        'Cannot access ANT tokens',
        'Limited governance participation',
        'No staking rewards'
      ],
      requirements: 'None - Default wallet type'
    },
    {
      type: 'citizen' as WalletType,
      name: 'Citizen',
      icon: <Shield size={32} />,
      color: '#10b981',
      description: 'Verified user with governance rights',
      features: [
        'All Guest features',
        'Receive ANT tokens from protocol',
        'Sell ANT rights on internal market',
        'Participate in governance voting',
        'Access to citizen-only features'
      ],
      limitations: [
        'ANT accumulation limits apply',
        'Must maintain activity (1 year rule)',
        'Cannot validate transactions'
      ],
      requirements: 'ZKP identity verification required'
    },
    {
      type: 'validator' as WalletType,
      name: 'Validator',
      icon: <Crown size={32} />,
      color: '#f59e0b',
      description: 'Network validator with consensus rights',
      features: [
        'All Citizen features',
        'Activate LZN tokens for staking',
        'Participate in network consensus',
        'Earn base emission rewards',
        'Buy ANT rights for consensus participation'
      ],
      limitations: [
        'Maximum 33% LZN activation per wallet',
        'Must maintain 6-month activity',
        'Higher responsibility and requirements'
      ],
      requirements: 'ZKP verification + Technical setup + Minimum stake'
    }
  ];

  const getCurrentTypeData = () => {
    return walletTypes.find(wt => wt.type === currentType) || walletTypes[0];
  };

  const currentTypeData = getCurrentTypeData();

  return (
    <div style={{ width: '100%' }}>
      {/* Current Wallet Type Status */}
      <div className="card" style={{ 
        background: `linear-gradient(135deg, ${currentTypeData.color}20 0%, ${currentTypeData.color}10 100%)`,
        border: `2px solid ${currentTypeData.color}40`
      }}>
        <div className="flex" style={{ alignItems: 'center', marginBottom: '16px' }}>
          <div style={{ color: currentTypeData.color }}>
            {currentTypeData.icon}
          </div>
          <div>
            <h3 style={{ margin: 0, fontSize: '1.5rem' }}>
              Current Status: {currentTypeData.name}
            </h3>
            <p style={{ margin: 0, color: '#6b7280' }}>
              {currentTypeData.description}
            </p>
          </div>
          <CheckCircle size={24} style={{ color: currentTypeData.color, marginLeft: 'auto' }} />
        </div>
      </div>

      {/* All Wallet Types */}
      <div className="card">
        <h3 style={{ marginBottom: '24px', display: 'flex', alignItems: 'center', gap: '8px' }}>
          <Users size={24} />
          Wallet Types in Volnix Protocol
        </h3>

        <div style={{ display: 'grid', gap: '20px' }}>
          {walletTypes.map((walletType) => (
            <div 
              key={walletType.type}
              style={{
                border: `2px solid ${walletType.type === currentType ? walletType.color : '#e5e7eb'}`,
                borderRadius: '12px',
                padding: '20px',
                background: walletType.type === currentType ? `${walletType.color}10` : 'white'
              }}
            >
              <div className="flex" style={{ alignItems: 'center', marginBottom: '16px' }}>
                <div style={{ color: walletType.color }}>
                  {walletType.icon}
                </div>
                <div>
                  <h4 style={{ margin: 0, fontSize: '1.3rem' }}>
                    {walletType.name}
                    {walletType.type === currentType && (
                      <span style={{ 
                        marginLeft: '8px', 
                        fontSize: '12px', 
                        background: walletType.color,
                        color: 'white',
                        padding: '2px 8px',
                        borderRadius: '12px'
                      }}>
                        CURRENT
                      </span>
                    )}
                  </h4>
                  <p style={{ margin: 0, color: '#6b7280' }}>
                    {walletType.description}
                  </p>
                </div>
              </div>

              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
                <div>
                  <h5 style={{ color: '#10b981', marginBottom: '8px' }}>✓ Features</h5>
                  <ul style={{ margin: 0, paddingLeft: '16px', fontSize: '14px' }}>
                    {walletType.features.map((feature, index) => (
                      <li key={index} style={{ marginBottom: '4px', color: '#374151' }}>
                        {feature}
                      </li>
                    ))}
                  </ul>
                </div>

                <div>
                  <h5 style={{ color: '#ef4444', marginBottom: '8px' }}>⚠ Limitations</h5>
                  <ul style={{ margin: 0, paddingLeft: '16px', fontSize: '14px' }}>
                    {walletType.limitations.map((limitation, index) => (
                      <li key={index} style={{ marginBottom: '4px', color: '#6b7280' }}>
                        {limitation}
                      </li>
                    ))}
                  </ul>
                </div>
              </div>

              <div style={{ 
                marginTop: '16px', 
                padding: '12px', 
                background: '#f9fafb', 
                borderRadius: '8px' 
              }}>
                <strong style={{ fontSize: '14px' }}>Requirements: </strong>
                <span style={{ fontSize: '14px', color: '#6b7280' }}>
                  {walletType.requirements}
                </span>
              </div>

              {walletType.type !== currentType && (
                <button
                  className="button"
                  style={{ 
                    width: '100%', 
                    marginTop: '16px',
                    background: walletType.color,
                    opacity: walletType.type === 'citizen' || walletType.type === 'validator' ? 0.6 : 1
                  }}
                  disabled={walletType.type === 'citizen' || walletType.type === 'validator'}
                >
                  {walletType.type === 'citizen' || walletType.type === 'validator' ? (
                    <>
                      <Lock size={16} />
                      Requires Verification
                    </>
                  ) : (
                    `Upgrade to ${walletType.name}`
                  )}
                </button>
              )}
            </div>
          ))}
        </div>
      </div>

      {/* Upgrade Information */}
      <div className="card" style={{ background: '#fef3c7' }}>
        <h4 style={{ color: '#92400e', marginBottom: '12px' }}>🚀 Want to Upgrade?</h4>
        <p style={{ color: '#78350f', marginBottom: '16px' }}>
          Upgrade your wallet type to unlock more features and participate more actively in the Volnix ecosystem.
        </p>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: '12px' }}>
          <button className="button" style={{ background: '#10b981' }}>
            Start Verification Process
          </button>
          <button className="button" style={{ background: '#6b7280' }}>
            Learn More
          </button>
        </div>
      </div>
    </div>
  );
};

export default WalletTypes;