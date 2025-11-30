// Network and RPC types
export interface NetworkStatus {
  node_info: {
    network: string;
    version: string;
  };
  sync_info: {
    latest_block_height: string;
    latest_block_hash: string;
    latest_block_time: string;
  };
}

export interface Block {
  height: number;
  hash: string;
  time: string;
  txs: number;
  proposer: string;
}

export interface Transaction {
  hash: string;
  height: number;
  time: string;
  blockHash: string;
}

// Validator types
export interface Validator {
  validator: string;
  ant_balance: string;
  status: 'VALIDATOR_STATUS_ACTIVE' | 'VALIDATOR_STATUS_INACTIVE' | 'VALIDATOR_STATUS_SLASHED' | 'VALIDATOR_STATUS_UNSPECIFIED';
  last_active?: string;
  last_block_height?: number;
  moa_score?: string;
  activity_score?: string;
  total_blocks_created?: number;
  total_burn_amount?: string;
}

export interface ConsensusParams {
  base_block_time?: string;
  high_activity_threshold?: string;
  low_activity_threshold?: string;
  min_burn_amount?: string;
  max_burn_amount?: string;
}

// API Response types
export interface ValidatorsResponse {
  validators: Validator[];
}

export interface ConsensusParamsResponse {
  params: ConsensusParams;
}

// Tab types
export type TabType = 'blocks' | 'transactions' | 'validators' | 'modules' | 'consensus';

