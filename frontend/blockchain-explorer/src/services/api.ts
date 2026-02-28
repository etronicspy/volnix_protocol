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

// Fetch transactions via tx_search (indexed, finds txs in any block)
export async function fetchTransactionsViaTxSearch(limit: number = 20): Promise<{ transactions: Transaction[]; totalCount: number }> {
  try {
    const params = new URLSearchParams({
      query: 'tx.height>=1',
      prove: 'false',
      page: '1',
      per_page: String(Math.min(limit, 100)),
      order_by: 'desc'
    });
    const response = await fetch(`${RPC_ENDPOINT}/tx_search?${params}`);
    const data = await response.json();
    if (data.error) {
      console.warn('tx_search failed, falling back to block scan:', data.error);
      return fetchTransactionsFromBlocks(limit);
    }
    const result = data.result || {};
    const txItems = result.txs || [];
    const transactions: Transaction[] = [];
    for (const item of txItems) {
      const height = parseInt(item.height || '0', 10);
      const block = height > 0 ? await fetchBlock(height) : null;
      transactions.push({
        hash: item.hash || '',
        height,
        time: block?.block?.header?.time || new Date().toISOString(),
        blockHash: block?.block_id?.hash || ''
      });
    }
    const totalCount = parseInt(result.total_count || '0', 10);
    return { transactions, totalCount };
  } catch (error) {
    console.error('Error fetching transactions via tx_search:', error);
    return fetchTransactionsFromBlocks(limit);
  }
}

// Fallback: fetch transactions from recent blocks only
async function fetchTransactionsFromBlocks(limit: number): Promise<{ transactions: Transaction[]; totalCount: number }> {
  try {
    const blocks = await fetchRecentBlocks(50);
    const transactions: Transaction[] = [];
    for (const block of blocks) {
      if (block.txs > 0) {
        const blockData = await fetchBlock(block.height);
        if (blockData?.block?.data?.txs) {
          blockData.block.data.txs.forEach((_: any, index: number) => {
            transactions.push({
              hash: `block-${block.height}-tx-${index}`,
              height: block.height,
              time: block.time,
              blockHash: block.hash
            });
          });
        }
      }
    }
    const totalTxs = blocks.reduce((s, b) => s + b.txs, 0);
    return { transactions: transactions.slice(0, limit), totalCount: totalTxs };
  } catch {
    return { transactions: [], totalCount: 0 };
  }
}

// Legacy: fetch transactions from blocks (for backward compatibility)
export async function fetchTransactions(limit: number = 20): Promise<Transaction[]> {
  const { transactions } = await fetchTransactionsViaTxSearch(limit);
  return transactions;
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

