import React, { useState, useEffect, useCallback } from 'react';
import { Shield, Activity, Wifi, WifiOff, RefreshCw, Server, TrendingUp } from 'lucide-react';

interface ValidatorData {
  validator: string;
  status: string;
  voting_power: string;
  ant_balance: string;
  activity_score: string;
  total_blocks_created: number;
  total_burn_amount: string;
}

interface NetworkInfo {
  latestBlockHeight: string;
  latestBlockTime: string;
  chainId: string;
  catching_up: boolean;
}

interface NetworkStakingProps {
  userAddress?: string;
}

const REST_ENDPOINT = process.env.REACT_APP_REST_ENDPOINT || 'http://localhost:1317';
const RPC_ENDPOINT = process.env.REACT_APP_RPC_ENDPOINT || 'http://localhost:26657';

const NetworkStaking: React.FC<NetworkStakingProps> = ({ userAddress }) => {
  const [validators, setValidators] = useState<ValidatorData[]>([]);
  const [networkInfo, setNetworkInfo] = useState<NetworkInfo | null>(null);
  const [networkOnline, setNetworkOnline] = useState<boolean | null>(null);
  const [loading, setLoading] = useState(true);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);

    let online = false;
    let netInfo: NetworkInfo | null = null;
    let vals: ValidatorData[] = [];

    try {
      const res = await fetch(`${RPC_ENDPOINT}/status`);
      if (res.ok) {
        const data = await res.json();
        const syncInfo = data.result?.sync_info;
        const nodeInfo = data.result?.node_info;
        if (syncInfo) {
          online = true;
          netInfo = {
            latestBlockHeight: syncInfo.latest_block_height || '0',
            latestBlockTime: syncInfo.latest_block_time || '',
            chainId: nodeInfo?.network || '',
            catching_up: syncInfo.catching_up || false,
          };
        }
      }
    } catch { /* node unreachable */ }

    if (online) {
      try {
        const res = await fetch(`${REST_ENDPOINT}/volnix/consensus/v1/validators`);
        if (res.ok) {
          const data = await res.json();
          vals = data.validators || [];
        }
      } catch { /* api unreachable */ }
    }

    setNetworkOnline(online);
    setNetworkInfo(netInfo);
    setValidators(vals);
    setLastUpdated(new Date());
    setLoading(false);
  }, []);

  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchData, 15000);
    return () => clearInterval(interval);
  }, [fetchData]);

  const formatTime = (isoString: string) => {
    if (!isoString) return '—';
    try {
      const date = new Date(isoString);
      return date.toLocaleTimeString();
    } catch { return '—'; }
  };

  const formatAddress = (address: string) => {
    if (!address) return '—';
    if (address.length <= 20) return address;
    return `${address.slice(0, 12)}...${address.slice(-8)}`;
  };

  const totalVotingPower = validators.reduce(
    (sum, v) => sum + parseInt(v.voting_power || '0', 10), 0
  );

  return (
    <div style={{ width: '100%' }}>
      {/* Network status banner */}
      <div className="card" style={{
        background: networkOnline === null
          ? '#f3f4f6'
          : networkOnline
            ? 'linear-gradient(135deg, #059669 0%, #047857 100%)'
            : 'linear-gradient(135deg, #dc2626 0%, #b91c1c 100%)',
        color: 'white',
      }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
            {networkOnline ? <Wifi size={28} /> : <WifiOff size={28} />}
            <div>
              <h3 style={{ margin: 0, fontSize: '1.4rem' }}>
                {networkOnline === null ? 'Checking...' : networkOnline ? 'Network Online' : 'Network Offline'}
              </h3>
              {networkInfo && (
                <p style={{ margin: '4px 0 0 0', opacity: 0.9, fontSize: '14px' }}>
                  {networkInfo.chainId} &middot; Block #{networkInfo.latestBlockHeight}
                  {networkInfo.catching_up && ' (syncing...)'}
                </p>
              )}
              {!networkOnline && networkOnline !== null && (
                <p style={{ margin: '4px 0 0 0', opacity: 0.9, fontSize: '14px' }}>
                  Nodes are not reachable. Start the blockchain node to see live data.
                </p>
              )}
            </div>
          </div>
          <button
            className="button"
            onClick={fetchData}
            disabled={loading}
            style={{
              background: 'rgba(255,255,255,0.2)',
              border: '1px solid rgba(255,255,255,0.4)',
              minWidth: 'auto',
              padding: '8px 12px',
            }}
          >
            <RefreshCw size={16} className={loading ? 'animate-spin' : ''} />
          </button>
        </div>
      </div>

      {/* Network stats row */}
      {networkOnline && networkInfo && (
        <div style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(4, 1fr)',
          gap: '12px',
          marginBottom: '20px',
        }}>
          <div className="card" style={{ textAlign: 'center', padding: '16px' }}>
            <div style={{ fontSize: '12px', color: '#6b7280', marginBottom: '4px' }}>Block Height</div>
            <div style={{ fontSize: '1.5rem', fontWeight: 'bold', color: '#3b82f6' }}>
              {parseInt(networkInfo.latestBlockHeight).toLocaleString()}
            </div>
          </div>
          <div className="card" style={{ textAlign: 'center', padding: '16px' }}>
            <div style={{ fontSize: '12px', color: '#6b7280', marginBottom: '4px' }}>Validators</div>
            <div style={{ fontSize: '1.5rem', fontWeight: 'bold', color: '#10b981' }}>
              {validators.length}
            </div>
          </div>
          <div className="card" style={{ textAlign: 'center', padding: '16px' }}>
            <div style={{ fontSize: '12px', color: '#6b7280', marginBottom: '4px' }}>Total Voting Power</div>
            <div style={{ fontSize: '1.5rem', fontWeight: 'bold', color: '#f59e0b' }}>
              {totalVotingPower.toLocaleString()}
            </div>
          </div>
          <div className="card" style={{ textAlign: 'center', padding: '16px' }}>
            <div style={{ fontSize: '12px', color: '#6b7280', marginBottom: '4px' }}>Last Block</div>
            <div style={{ fontSize: '1.2rem', fontWeight: 'bold', color: '#8b5cf6' }}>
              {formatTime(networkInfo.latestBlockTime)}
            </div>
          </div>
        </div>
      )}

      {/* Validator list */}
      <div className="card">
        <div style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: '16px',
        }}>
          <h4 style={{ margin: 0, display: 'flex', alignItems: 'center', gap: '8px' }}>
            <Shield size={20} />
            Active Validators
          </h4>
          {lastUpdated && (
            <span style={{ fontSize: '12px', color: '#9ca3af' }}>
              Updated {formatTime(lastUpdated.toISOString())}
            </span>
          )}
        </div>

        {!networkOnline && networkOnline !== null ? (
          <div style={{
            textAlign: 'center',
            padding: '40px 20px',
            color: '#6b7280',
          }}>
            <WifiOff size={48} style={{ marginBottom: '12px', opacity: 0.4 }} />
            <p style={{ fontSize: '16px', margin: '0 0 8px 0' }}>
              Cannot fetch validator data
            </p>
            <p style={{ fontSize: '14px', margin: 0 }}>
              The blockchain node is not running or not reachable.
            </p>
          </div>
        ) : validators.length === 0 && !loading ? (
          <div style={{
            textAlign: 'center',
            padding: '40px 20px',
            color: '#6b7280',
          }}>
            <Server size={48} style={{ marginBottom: '12px', opacity: 0.4 }} />
            <p style={{ fontSize: '16px', margin: '0 0 8px 0' }}>
              No validators found
            </p>
            <p style={{ fontSize: '14px', margin: 0 }}>
              The network has no active validators yet.
            </p>
          </div>
        ) : (
          <div style={{ overflowX: 'auto' }}>
            <table style={{
              width: '100%',
              borderCollapse: 'collapse',
              fontSize: '14px',
            }}>
              <thead>
                <tr style={{ borderBottom: '2px solid #e5e7eb' }}>
                  <th style={{ textAlign: 'left', padding: '10px 8px', color: '#6b7280', fontWeight: '600' }}>#</th>
                  <th style={{ textAlign: 'left', padding: '10px 8px', color: '#6b7280', fontWeight: '600' }}>Address</th>
                  <th style={{ textAlign: 'center', padding: '10px 8px', color: '#6b7280', fontWeight: '600' }}>Status</th>
                  <th style={{ textAlign: 'right', padding: '10px 8px', color: '#6b7280', fontWeight: '600' }}>Voting Power</th>
                  <th style={{ textAlign: 'right', padding: '10px 8px', color: '#6b7280', fontWeight: '600' }}>Power Share</th>
                  <th style={{ textAlign: 'right', padding: '10px 8px', color: '#6b7280', fontWeight: '600' }}>Blocks Created</th>
                </tr>
              </thead>
              <tbody>
                {validators.map((val, index) => {
                  const power = parseInt(val.voting_power || '0', 10);
                  const share = totalVotingPower > 0 ? (power / totalVotingPower * 100) : 0;
                  const isUser = userAddress && val.validator.toLowerCase() === userAddress.toLowerCase();
                  const isActive = val.status === 'VALIDATOR_STATUS_ACTIVE';

                  return (
                    <tr
                      key={val.validator}
                      style={{
                        borderBottom: '1px solid #f3f4f6',
                        background: isUser ? '#fef3c7' : 'transparent',
                      }}
                    >
                      <td style={{ padding: '12px 8px', fontWeight: '500' }}>{index + 1}</td>
                      <td style={{ padding: '12px 8px' }}>
                        <span style={{ fontFamily: 'monospace', fontSize: '13px' }}>
                          {formatAddress(val.validator)}
                        </span>
                        {isUser && (
                          <span style={{
                            marginLeft: '8px',
                            fontSize: '11px',
                            background: '#f59e0b',
                            color: 'white',
                            padding: '2px 6px',
                            borderRadius: '4px',
                            fontWeight: '600',
                          }}>YOU</span>
                        )}
                      </td>
                      <td style={{ padding: '12px 8px', textAlign: 'center' }}>
                        <span style={{
                          display: 'inline-flex',
                          alignItems: 'center',
                          gap: '4px',
                          fontSize: '12px',
                          fontWeight: '600',
                          color: isActive ? '#059669' : '#dc2626',
                          background: isActive ? '#ecfdf5' : '#fef2f2',
                          padding: '3px 8px',
                          borderRadius: '12px',
                        }}>
                          <Activity size={12} />
                          {isActive ? 'Active' : 'Inactive'}
                        </span>
                      </td>
                      <td style={{ padding: '12px 8px', textAlign: 'right', fontWeight: '600' }}>
                        {power.toLocaleString()}
                      </td>
                      <td style={{ padding: '12px 8px', textAlign: 'right' }}>
                        <div style={{
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'flex-end',
                          gap: '8px',
                        }}>
                          <div style={{
                            width: '60px',
                            height: '6px',
                            background: '#e5e7eb',
                            borderRadius: '3px',
                            overflow: 'hidden',
                          }}>
                            <div style={{
                              width: `${Math.min(share, 100)}%`,
                              height: '100%',
                              background: share > 33 ? '#f59e0b' : '#3b82f6',
                              borderRadius: '3px',
                            }} />
                          </div>
                          <span style={{ fontWeight: '500', minWidth: '48px' }}>
                            {share.toFixed(1)}%
                          </span>
                        </div>
                      </td>
                      <td style={{ padding: '12px 8px', textAlign: 'right' }}>
                        {(val.total_blocks_created || 0).toLocaleString()}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Protocol info */}
      <div className="card" style={{ background: '#f0f9ff', border: '1px solid #bfdbfe' }}>
        <h4 style={{ margin: '0 0 12px 0', color: '#1e40af', display: 'flex', alignItems: 'center', gap: '8px' }}>
          <TrendingUp size={20} />
          PoVB Consensus
        </h4>
        <div style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fit, minmax(250px, 1fr))',
          gap: '16px',
          fontSize: '14px',
          color: '#1e3a5f',
        }}>
          <div>
            <strong>How it works:</strong>
            <p style={{ margin: '4px 0 0 0', color: '#4b5563' }}>
              Validators activate LZN (mining licenses) to earn passive WRT income.
              They burn ANT tokens in auctions to win the right to create the next block.
            </p>
          </div>
          <div>
            <strong>MOA (Minimum Activity Obligation):</strong>
            <p style={{ margin: '4px 0 0 0', color: '#4b5563' }}>
              Validators must participate actively (burn ANT) to keep their LZN rewards.
              Failing MOA results in penalties or deactivation.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
};

export default NetworkStaking;
