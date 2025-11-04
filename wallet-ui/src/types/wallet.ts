export interface Balance {
  wrt: string;
  lzn: string;
  ant: string;
}

export interface Transaction {
  id: string;
  type: 'send' | 'receive';
  amount: string;
  token: string;
  from: string;
  to: string;
  timestamp: string;
  status: 'pending' | 'completed' | 'failed';
}

export type WalletType = 'guest' | 'citizen' | 'validator';

export interface WalletState {
  isConnected: boolean;
  address: string;
  balance: Balance;
  walletType: WalletType;
  transactions: Transaction[];
}