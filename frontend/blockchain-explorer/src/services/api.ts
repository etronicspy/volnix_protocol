import { NetworkStatus, Block, Transaction, Validator, ConsensusParams } from '../types';

// API configuration
const RPC_ENDPOINT = process.env.REACT_APP_RPC_ENDPOINT || 'http://localhost:26657';
const REST_API_ENDPOINT = process.env.REACT_APP_REST_API_ENDPOINT || 'http://localhost:1317';

// Fetch network status from RPC
export async function fetchNetworkStatus(): Promise<NetworkStatus | null> {
  try {
    const response = await fetch(`${RPC_ENDPOINT}/status`);
    const data = await response.json();
    return data.result;
  } catch (error) {
    console.error('Error fetching network status:', error);
    return null;
  }
}

// Fetch latest block from RPC
export async function fetchLatestBlock(): Promise<any | null> {
  try {
    const response = await fetch(`${RPC_ENDPOINT}/block`);
    const data = await response.json();
    return data.result;
  } catch (error) {
    console.error('Error fetching latest block:', error);
    return null;
  }
}

// Fetch block by height from RPC
export async function fetchBlock(height: number): Promise<any | null> {
  try {
    const response = await fetch(`${RPC_ENDPOINT}/block?height=${height}`);
    const data = await response.json();
    return data.result;
  } catch (error) {
    console.error('Error fetching block:', error);
    return null;
  }
}

// Fetch recent blocks
export async function fetchRecentBlocks(limit: number = 10): Promise<Block[]> {
  try {
    const status = await fetchNetworkStatus();
    if (!status) return [];

    const latestHeight = parseInt(status.sync_info.latest_block_height);
    const blocks: Block[] = [];

    for (let i = 0; i < limit && i < latestHeight; i++) {
      const height = latestHeight - i;
      const block = await fetchBlock(height);
      if (block) {
        blocks.push({
          height: parseInt(block.block.header.height),
          hash: block.block_id.hash,
          time: block.block.header.time,
          txs: block.block.data.txs ? block.block.data.txs.length : 0,
          proposer: block.block.header.proposer_address || ''
        });
      }
    }

    return blocks;
  } catch (error) {
    console.error('Error fetching recent blocks:', error);
    return [];
  }
}

// Fetch transactions from blocks
export async function fetchTransactions(limit: number = 20): Promise<Transaction[]> {
  try {
    const blocks = await fetchRecentBlocks(limit);
    const transactions: Transaction[] = [];

    for (const block of blocks) {
      if (block.txs > 0) {
        const blockData = await fetchBlock(block.height);
        if (blockData && blockData.block.data.txs) {
          blockData.block.data.txs.forEach((tx: any, index: number) => {
            transactions.push({
              hash: tx.hash || `block-${block.height}-tx-${index}`,
              height: block.height,
              time: block.time,
              blockHash: block.hash
            });
          });
        }
      }
    }

    return transactions.slice(0, limit);
  } catch (error) {
    console.error('Error fetching transactions:', error);
    return [];
  }
}

// Fetch validators from REST API
export async function fetchValidators(): Promise<Validator[]> {
  try {
    const response = await fetch(`${REST_API_ENDPOINT}/volnix/consensus/v1/validators`);
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }
    const data = await response.json();
    return data.validators || [];
  } catch (error) {
    console.error('Error fetching validators from REST API:', error);
    return [];
  }
}

// Fetch consensus params from REST API
export async function fetchConsensusParams(): Promise<ConsensusParams | null> {
  try {
    const response = await fetch(`${REST_API_ENDPOINT}/volnix/consensus/v1/params`);
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }
    const data = await response.json();
    return data.params || null;
  } catch (error) {
    console.error('Error fetching consensus params from REST API:', error);
    return null;
  }
}

// Check REST API health
export async function checkRestApiHealth(): Promise<boolean> {
  try {
    const response = await fetch(`${REST_API_ENDPOINT}/health`, {
      method: 'GET',
      headers: {
        'Accept': 'application/json',
      },
      mode: 'cors',
    });
    return response.ok;
  } catch (error) {
    console.warn('REST API not available:', error);
    return false;
  }
}

