export interface Balance {
  wrt: string;
  lzn: string;
  lznActivated?: string; // Только для валидаторов
  ant: string;
}

export interface Transaction {
  id: string;
  type: 'send' | 'receive' | 'ant_buy' | 'ant_sell' | 'lzn_activate' | 'lzn_deactivate';
  amount: string;
  token: string;
  from: string;
  to: string;
  timestamp: string;
  status: 'pending' | 'completed' | 'failed';
  price?: string; // Для ANT сделок
}

// Роли взаимоисключающие после верификации!
export type WalletType = 'guest' | 'citizen' | 'validator';

export interface WalletState {
  isConnected: boolean;
  address: string;
  balance: Balance;
  walletType: WalletType;
  transactions: Transaction[];
  isVerified: boolean; // ZKP верификация пройдена
  antAccumulationLimit?: number; // 1000 для граждан
  moaRequired?: number; // Требуемое MOA для валидаторов
  moaCurrent?: number; // Текущее выполнение MOA
  lastActivity?: string; // Для правил активности
}

// Ордер на внутреннем рынке ANT
export interface AntMarketOrder {
  orderId: string;
  owner: string;
  orderType: 'LIMIT' | 'MARKET';
  orderSide: 'BUY' | 'SELL';
  antAmount: string;
  pricePerAnt: string; // В WRT
  status: 'OPEN' | 'FILLED' | 'CANCELLED' | 'PARTIAL';
  createdAt: string;
  expiresAt: string;
  filledAmount?: string;
}

// Информация о валидаторе
export interface ValidatorInfo {
  address: string;
  lznActivated: string;
  lznTotal: string;
  shareOfNetwork: string; // % от общего пула активированных LZN
  passiveIncome: string; // Доход из Контура 1
  activeIncome: string; // Доход из Контура 2 (аукционы)
  moaRequired: string;
  moaCurrent: string;
  moaCompliance: number; // 0-1
  lastBlockWon?: string;
  blocksWonTotal: number;
}

// Информация о гражданине
export interface CitizenInfo {
  address: string;
  antAccumulated: string;
  antLimit: string; // 1000 ANT
  antSoldTotal: string;
  incomeFromAntSales: string; // В WRT
  lastAntAccrual: string;
  dailyAntRate: string; // 10 ANT/день
}