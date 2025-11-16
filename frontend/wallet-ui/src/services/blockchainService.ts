import { StargateClient, SigningStargateClient } from '@cosmjs/stargate';
import { DirectSecp256k1HdWallet } from '@cosmjs/proto-signing';
import { GasPrice } from '@cosmjs/stargate';
import { Comet38Client } from '@cosmjs/tendermint-rpc';

// Конфигурация сети
const RPC_ENDPOINT = process.env.REACT_APP_RPC_ENDPOINT || 'http://localhost:26657';
const CHAIN_ID = process.env.REACT_APP_CHAIN_ID || 'volnix-standalone';
const PREFIX = 'volnix';

// Типы для балансов
export interface TokenBalance {
  denom: string;
  amount: string;
}

export interface BlockchainTransaction {
  hash: string;
  height: number;
  timestamp: string;
  from: string;
  to: string;
  amount: string;
  denom: string;
  status: 'success' | 'failed';
}

class BlockchainService {
  private client: StargateClient | null = null;
  private signingClient: SigningStargateClient | null = null;
  private wallet: DirectSecp256k1HdWallet | null = null;

  // Инициализация клиента для чтения
  async initializeClient(): Promise<void> {
    if (!this.client) {
      try {
        // Connect with explicit chain-id to avoid "must provide a non-empty value" error
        this.client = await StargateClient.connect(RPC_ENDPOINT);
        // Verify chain-id matches
        const actualChainId = await this.client.getChainId();
        if (actualChainId !== CHAIN_ID) {
          console.warn(`Chain ID mismatch: expected ${CHAIN_ID}, got ${actualChainId}`);
        }
      } catch (error: any) {
        throw new Error(`Failed to connect to blockchain: ${error.message || 'Unknown error'}. Make sure the node is running on ${RPC_ENDPOINT}`);
      }
    }
  }

  // Инициализация клиента для подписи транзакций
  async initializeSigningClient(mnemonic: string): Promise<string> {
    try {
      this.wallet = await DirectSecp256k1HdWallet.fromMnemonic(mnemonic, {
        prefix: PREFIX,
      });

      const [account] = await this.wallet.getAccounts();
      
      try {
        // SIMPLE APPROACH: Just use StargateClient which works
        // Avoid Comet38Client entirely as it has issues with chain-id
        console.log('🔍 Connecting to RPC endpoint:', RPC_ENDPOINT);
        
        // Use connectWithSigner directly - simplest approach
        this.signingClient = await SigningStargateClient.connectWithSigner(
          RPC_ENDPOINT,
          this.wallet,
          {
            gasPrice: GasPrice.fromString('0.025uwrt'),
          }
        );
        console.log('✅ SigningStargateClient connected');
        
        // Verify chain-id is available
        const chainId = await this.signingClient.getChainId();
        console.log('✅ Chain ID:', chainId);
      } catch (error: any) {
        // Provide more helpful error message with detailed logging
        const errorMsg = error.message || 'Unknown error';
        console.error('❌ Error connecting signing client:', errorMsg);
        console.error('❌ Full error:', error);
        
        if (errorMsg.includes('must provide a non-empty value') || errorMsg.includes('empty chain-id')) {
          throw new Error(`Failed to connect signing client: Node returned empty chain-id. Make sure the node is running and properly initialized on ${RPC_ENDPOINT}. Error: ${errorMsg}`);
        }
        if (errorMsg.includes('fetch') || errorMsg.includes('network') || errorMsg.includes('ECONNREFUSED')) {
          throw new Error(`Failed to connect to node at ${RPC_ENDPOINT}. Make sure the RPC node is running. Error: ${errorMsg}`);
        }
        throw new Error(`Failed to connect signing client: ${errorMsg}. Make sure the node is running on ${RPC_ENDPOINT}`);
      }

      return account.address;
    } catch (error: any) {
      if (error.message && error.message.includes('Invalid mnemonic')) {
        throw new Error('Invalid mnemonic phrase. Please check your mnemonic and try again.');
      }
      throw new Error(`Failed to initialize wallet: ${error.message || 'Unknown error'}`);
    }
  }

  // Получение балансов всех токенов
  async getBalances(address: string): Promise<{ wrt: string; lzn: string; ant: string }> {
    try {
      await this.initializeClient();
      if (!this.client) throw new Error('Client not initialized');

      const balances = await this.client.getAllBalances(address);
      
      const result = {
        wrt: '0',
        lzn: '0',
        ant: '0',
      };

      if (balances && Array.isArray(balances)) {
        balances.forEach((balance) => {
          if (!balance || !balance.denom || !balance.amount) return;
          
          const amount = balance.amount;
          const amountNum = parseInt(amount, 10);
          if (isNaN(amountNum)) return;

          if (balance.denom === 'uwrt' || balance.denom === 'wrt') {
            result.wrt = (amountNum / 1_000_000).toFixed(6);
          } else if (balance.denom === 'ulzn' || balance.denom === 'lzn') {
            result.lzn = (amountNum / 1_000_000).toFixed(6);
          } else if (balance.denom === 'uant' || balance.denom === 'ant') {
            result.ant = (amountNum / 1_000_000).toFixed(6);
          }
        });
      }

      return result;
    } catch (error: any) {
      // Если аккаунт не существует, возвращаем нулевые балансы
      if (error.message && error.message.includes('account does not exist')) {
        return { wrt: '0', lzn: '0', ant: '0' };
      }
      throw new Error(`Failed to get balances: ${error.message || 'Unknown error'}`);
    }
  }

  // Получение информации об аккаунте
  async getAccount(address: string): Promise<any> {
    await this.initializeClient();
    if (!this.client) throw new Error('Client not initialized');

    try {
      const account = await this.client.getAccount(address);
      return account;
    } catch (error) {
      // Аккаунт может не существовать
      return null;
    }
  }

  // Получение транзакций аккаунта
  async getTransactions(address: string, limit: number = 50): Promise<BlockchainTransaction[]> {
    await this.initializeClient();
    if (!this.client) throw new Error('Client not initialized');

    try {
      // Используем RPC напрямую для получения транзакций
      const response = await fetch(`${RPC_ENDPOINT}/tx_search?query="transfer.recipient='${address}' OR transfer.sender='${address}'"&per_page=${limit}`);
      const data = await response.json();

      if (!data.result || !data.result.txs) {
        return [];
      }

      const transactions: BlockchainTransaction[] = data.result.txs.map((tx: any) => {
        // Парсинг транзакции из Cosmos SDK формата
        const txHash = tx.hash || '';
        const height = tx.height || 0;
        const timestamp = tx.timestamp || new Date().toISOString();

        // Извлечение данных из сообщений
        // Это упрощенная версия, в реальности нужно парсить protobuf
        // Пытаемся извлечь данные из tx_result
        let from = address;
        let to = address;
        let amount = '0';
        let denom = 'uwrt';
        let status: 'success' | 'failed' = 'success';

        if (tx.tx_result) {
          if (tx.tx_result.code !== 0) {
            status = 'failed';
          }
          // Здесь можно добавить парсинг событий для получения from/to/amount
        }

        return {
          hash: txHash,
          height: typeof height === 'string' ? parseInt(height) : height,
          timestamp,
          from,
          to,
          amount,
          denom,
          status,
        };
      });

      return transactions;
    } catch (error) {
      console.error('Error fetching transactions:', error);
      return [];
    }
  }

  // Отправка токенов
  async sendTokens(
    fromAddress: string,
    toAddress: string,
    amount: string,
    denom: 'wrt' | 'lzn' | 'ant'
  ): Promise<string> {
    if (!this.signingClient) {
      throw new Error('Signing client not initialized. Please connect wallet with mnemonic.');
    }

    // Конвертация деноминации
    const fullDenom = denom === 'wrt' ? 'uwrt' : denom === 'lzn' ? 'ulzn' : 'uant';
    const amountInMicro = Math.floor(parseFloat(amount) * 1_000_000).toString();

    const sendMsg = {
      typeUrl: '/cosmos.bank.v1beta1.MsgSend',
      value: {
        fromAddress,
        toAddress,
        amount: [
          {
            denom: fullDenom,
            amount: amountInMicro,
          },
        ],
      },
    };

    const fee = {
      amount: [
        {
          denom: 'uwrt',
          amount: '5000', // Минимальная комиссия
        },
      ],
      gas: '200000',
    };

    try {
      const result = await this.signingClient.signAndBroadcast(
        fromAddress,
        [sendMsg],
        fee
      );

      if (result.code !== 0) {
        throw new Error(`Transaction failed: ${result.rawLog}`);
      }

      return result.transactionHash;
    } catch (error: any) {
      throw new Error(`Failed to send transaction: ${error.message}`);
    }
  }

  // Получение статуса сети
  async getNetworkStatus(): Promise<any> {
    await this.initializeClient();
    if (!this.client) throw new Error('Client not initialized');

    try {
      const response = await fetch(`${RPC_ENDPOINT}/status`);
      const data = await response.json();
      return data.result;
    } catch (error) {
      console.error('Error fetching network status:', error);
      return null;
    }
  }

  // Получение последнего блока
  async getLatestBlock(): Promise<any> {
    await this.initializeClient();
    if (!this.client) throw new Error('Client not initialized');

    try {
      const block = await this.client.getBlock();
      return block;
    } catch (error) {
      console.error('Error fetching latest block:', error);
      return null;
    }
  }

  // Очистка клиентов
  disconnect(): void {
    this.client = null;
    this.signingClient = null;
    this.wallet = null;
  }
}

export const blockchainService = new BlockchainService();

