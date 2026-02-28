import { useState, useEffect, useCallback } from 'react';
import {
  fetchNetworkStatus,
  fetchRecentBlocks,
  fetchTransactionsViaTxSearch,
  fetchValidators,
  fetchConsensusParams,
  checkRestApiHealth
} from '../services/api';
import { NetworkStatus, Block, Transaction, Validator, ConsensusParams } from '../types';

interface NetworkData {
  status: NetworkStatus | null;
  blocks: Block[];
  transactions: Transaction[];
  totalTransactionCount: number;
  validators: Validator[];
  consensusParams: ConsensusParams | null;
  restApiAvailable: boolean;
  loading: boolean;
  error: string | null;
}

export function useNetworkData(refreshInterval: number = 30000) {
  const [data, setData] = useState<NetworkData>({
    status: null,
    blocks: [],
    transactions: [],
    totalTransactionCount: 0,
    validators: [],
    consensusParams: null,
    restApiAvailable: false,
    loading: true,
    error: null
  });

  const refreshData = useCallback(async () => {
    try {
      setData(prev => ({ ...prev, loading: true, error: null }));

      // Check REST API health
      const restApiAvailable = await checkRestApiHealth();

      // Fetch network status
      const status = await fetchNetworkStatus();

      // Fetch blocks and transactions (tx_search finds txs in any block)
      const blocks = await fetchRecentBlocks(10);
      const { transactions, totalCount } = await fetchTransactionsViaTxSearch(20);

      // Fetch validators and consensus params if REST API is available
      let validators: Validator[] = [];
      let consensusParams: ConsensusParams | null = null;

      if (restApiAvailable) {
        validators = await fetchValidators();
        consensusParams = await fetchConsensusParams();
      }

      setData({
        status,
        blocks,
        transactions,
        totalTransactionCount: totalCount,
        validators,
        consensusParams,
        restApiAvailable,
        loading: false,
        error: null
      });
    } catch (error) {
      console.error('Error refreshing network data:', error);
      setData(prev => ({
        ...prev,
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to refresh data'
      }));
    }
  }, []);

  useEffect(() => {
    refreshData();

    const interval = setInterval(refreshData, refreshInterval);
    return () => clearInterval(interval);
  }, [refreshData, refreshInterval]);

  return { ...data, refreshData };
}

