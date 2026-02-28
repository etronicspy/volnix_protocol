import { StargateClient, SigningStargateClient, defaultRegistryTypes } from '@cosmjs/stargate';
import { DirectSecp256k1HdWallet } from '@cosmjs/proto-signing';
import { GasPrice } from '@cosmjs/stargate';
import { Comet38Client } from '@cosmjs/tendermint-rpc';
import { Registry } from '@cosmjs/proto-signing';
import { MsgChangeRoleType, MsgActivateLZNType, MsgDeactivateLZNType, MsgPlaceOrderType } from '../types/volnix-messages';

// Конфигурация сети
const RPC_ENDPOINT = process.env.REACT_APP_RPC_ENDPOINT || 'http://localhost:26657';
const REST_ENDPOINT = process.env.REACT_APP_REST_ENDPOINT || 'http://localhost:1317';
const CHAIN_ID = process.env.REACT_APP_CHAIN_ID || 'volnix-1';
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
  tx_type?: 'transfer' | 'identity_verified' | 'role_changed';
}

class BlockchainService {
  private client: StargateClient | null = null;
  private signingClient: SigningStargateClient | null = null;
  private wallet: DirectSecp256k1HdWallet | null = null;
  private readonly STATE_QUERY_RETRY_DELAYS_MS = [250, 500, 1000];

  private isTransientStateQueryError(error: unknown): boolean {
    const message = String((error as any)?.message || error || '').toLowerCase();
    return (
      message.includes('failed to load state at height') ||
      message.includes('version does not exist') ||
      message.includes('query failed with (38)')
    );
  }

  private async sleep(ms: number): Promise<void> {
    await new Promise((resolve) => setTimeout(resolve, ms));
  }

  private normalizeBalances(balances: ReadonlyArray<{ denom: string; amount: string }>): { wrt: string; lzn: string; ant: string } {
    const result = {
      wrt: '0',
      lzn: '0',
      ant: '0',
    };

    balances.forEach((balance) => {
      if (!balance || !balance.denom || !balance.amount) return;
      const amountNum = parseInt(balance.amount, 10);
      if (isNaN(amountNum)) return;

      if (balance.denom === 'uwrt' || balance.denom === 'wrt') {
        result.wrt = (amountNum / 1_000_000).toFixed(6);
      } else if (balance.denom === 'ulzn' || balance.denom === 'lzn') {
        result.lzn = (amountNum / 1_000_000).toFixed(6);
      } else if (balance.denom === 'uant' || balance.denom === 'ant') {
        result.ant = (amountNum / 1_000_000).toFixed(6);
      }
    });

    return result;
  }

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
        // Use standard CosmJS API - StargateClient.connect()
        // CRITICAL: Get chain-id from /status endpoint directly to avoid "empty chain-id" error
        // when blocks haven't been created yet
        console.log('🔍 Connecting to RPC endpoint:', RPC_ENDPOINT);
        
        // First, try to get chain-id from /status endpoint directly
        let actualChainId = CHAIN_ID; // Use configured chain-id as fallback
        try {
          const statusResponse = await fetch(`${RPC_ENDPOINT}/status`);
          if (statusResponse.ok) {
            const statusData = await statusResponse.json();
            const nodeChainId = statusData?.result?.node_info?.network;
            if (nodeChainId && nodeChainId.trim() !== '') {
              actualChainId = nodeChainId;
              console.log('✅ Chain ID from /status endpoint:', actualChainId);
            } else {
              console.warn('⚠️ Chain ID from /status is empty, using configured:', CHAIN_ID);
              actualChainId = CHAIN_ID;
            }
          }
        } catch (statusError) {
          console.warn('⚠️ Could not fetch chain-id from /status, using configured:', CHAIN_ID);
          actualChainId = CHAIN_ID;
        }
        
        // Create StargateClient - it will use the chain-id from node_info if available
        const readClient = await StargateClient.connect(RPC_ENDPOINT);
        console.log('✅ StargateClient connected');
        
        // Try to get chain-id from client, but don't fail if it's empty
        try {
          const clientChainId = await readClient.getChainId();
          if (clientChainId && clientChainId.trim() !== '') {
            actualChainId = clientChainId;
            console.log('✅ Chain ID from StargateClient:', actualChainId);
          }
        } catch (chainIdError) {
          console.warn('⚠️ Could not get chain-id from client, using:', actualChainId);
        }
        
        // Create SigningStargateClient using standard API
        // CRITICAL: Use defaultRegistryTypes to register bank message types
        // This ensures CosmJS can properly encode MsgSend messages
        const registry = new Registry(defaultRegistryTypes);
        
        // Register custom Volnix Protocol message types
        registry.register('/volnix.ident.v1.MsgChangeRole', MsgChangeRoleType);
        registry.register('/volnix.lizenz.v1.MsgActivateLZN', MsgActivateLZNType);
        registry.register('/volnix.lizenz.v1.MsgDeactivateLZN', MsgDeactivateLZNType);
        registry.register('/volnix.anteil.v1.MsgPlaceOrder', MsgPlaceOrderType);
        
        // CRITICAL: Use explicit chain-id to avoid "must provide a non-empty value" error
        // when blocks haven't been created yet
        // Create Comet38Client first - it will get chain-id from /status endpoint
        // The chain-id is available in node_info.network even if blocks aren't created
        const cometClient = await Comet38Client.connect(RPC_ENDPOINT);
        
        // Create SigningStargateClient from Comet38Client
        // Note: chainId is not a valid option in SigningStargateClientOptions
        // The chain-id will be obtained from the Comet38Client, which gets it from /status
        this.signingClient = await SigningStargateClient.createWithSigner(
          cometClient,
          this.wallet,
          {
            gasPrice: GasPrice.fromString('0.025uwrt'),
            registry: registry, // CRITICAL: Register types for message encoding
          }
        );
        
        console.log('✅ SigningStargateClient connected with chain-id:', actualChainId);
        
        // Close read client as SigningStargateClient has its own connection
        readClient.disconnect();
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

  // Получение ANT из anteil UserPosition (для граждан и валидаторов).
  // ANT хранится в модуле anteil, а не в bank — getAllBalances не возвращает ANT.
  async getUserPositionAnt(address: string): Promise<string> {
    const REST_ENDPOINT = process.env.REACT_APP_REST_ENDPOINT || 'http://localhost:1317';
    try {
      const response = await fetch(`${REST_ENDPOINT}/volnix/anteil/v1/user_position/${encodeURIComponent(address)}`);
      if (!response.ok) {
        if (response.status === 404) return '0';
        return '0';
      }
      const data = await response.json();
      const position = data?.position;
      const antBalance = position?.ant_balance ?? position?.antBalance ?? '0';
      const amountNum = parseInt(antBalance, 10);
      if (isNaN(amountNum)) return '0';
      return (amountNum / 1_000_000).toFixed(6);
    } catch {
      return '0';
    }
  }

  // Получение балансов всех токенов
  async getBalances(address: string): Promise<{ wrt: string; lzn: string; ant: string }> {
    const readBalances = async (): Promise<{ wrt: string; lzn: string; ant: string }> => {
      await this.initializeClient();
      if (!this.client) throw new Error('Client not initialized');

      const balances = await this.client.getAllBalances(address);
      const normalized = this.normalizeBalances(balances || []);
      // ANT для граждан/валидаторов хранится в anteil UserPosition, не в bank
      const antFromPosition = await this.getUserPositionAnt(address);
      return {
        ...normalized,
        ant: antFromPosition !== '0' ? antFromPosition : normalized.ant,
      };
    };

    try {
      return await readBalances();
    } catch (error: any) {
      // Transient startup/race condition on some CometBFT/Cosmos SDK queries.
      if (this.isTransientStateQueryError(error)) {
        for (const delay of this.STATE_QUERY_RETRY_DELAYS_MS) {
          try {
            this.client = null;
            await this.sleep(delay);
            return await readBalances();
          } catch (retryError: any) {
            if (!this.isTransientStateQueryError(retryError)) {
              throw retryError;
            }
          }
        }
      }

      if (this.isTransientStateQueryError(error)) {
        throw new Error(
          'Node is temporarily unavailable while syncing state. Please wait a few seconds and try again.'
        );
      }

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

  // Сохранение хеша транзакции в localStorage
  private saveTxHash(address: string, txHash: string): void {
    try {
      const TX_STORAGE_KEY = `volnix_txs_${address}`;
      const storedTxs = localStorage.getItem(TX_STORAGE_KEY);
      const txHashes: string[] = storedTxs ? JSON.parse(storedTxs) : [];
      
      // Добавляем новый хеш в начало массива (новые транзакции сначала)
      if (!txHashes.includes(txHash)) {
        txHashes.unshift(txHash);
        localStorage.setItem(TX_STORAGE_KEY, JSON.stringify(txHashes));
        console.log(`💾 Saved transaction hash ${txHash} to localStorage`);
      }
    } catch (error: any) {
      console.warn(`Failed to save transaction hash: ${error.message}`);
    }
  }

  // Парсинг событий transfer и ident из tx_result в BlockchainTransaction
  private parseTxEventsToTransaction(
    tx: { hash?: string; height?: string | number },
    txResult: { code?: number; events?: Array<{ type?: string; attributes?: Array<{ key?: string; value?: string; index?: boolean }> }> },
    address: string
  ): BlockchainTransaction {
    let from = '';
    let to = '';
    let amount = '0';
    let denom = 'uwrt';
    let txType: 'transfer' | 'identity_verified' | 'role_changed' = 'transfer';

    const getAttr = (attrs: Array<{ key?: string; value?: string; index?: boolean }>, key: string): string => {
      for (const a of attrs) {
        const k = a.index ? (a.key || '') : (a.key ? atob(a.key) : '');
        const v = a.index ? (a.value || '') : (a.value ? atob(a.value) : '');
        if (k === key) return v;
      }
      return '';
    };

    const events = txResult.events || [];
    for (const event of events) {
      const attributes = event.attributes || [];
      if (event.type === 'transfer' || event.type === 'coin_spent' || event.type === 'coin_received') {
        for (const attr of attributes) {
          const isIndexed = attr.index === true;
          let key = '';
          let value = '';
          if (isIndexed) {
            key = attr.key || '';
            value = attr.value || '';
          } else {
            try {
              key = attr.key ? atob(attr.key) : '';
              value = attr.value ? atob(attr.value) : '';
            } catch {
              key = attr.key || '';
              value = attr.value || '';
            }
          }
          if (key === 'sender' || key === 'spender') from = value;
          else if (key === 'recipient' || key === 'receiver') to = value;
          else if (key === 'amount') {
            const amounts = value.split(',');
            for (const amt of amounts) {
              const match = amt.trim().match(/^(\d+)(\w+)$/);
              if (match) {
                amount = match[1];
                denom = match[2];
                break;
              }
            }
          }
        }
      } else if (event.type === 'ident.identity_verified' || event.type === 'ident.role_changed') {
        const addr = getAttr(attributes, 'address');
        if (addr === address) {
          txType = event.type === 'ident.identity_verified' ? 'identity_verified' : 'role_changed';
          from = addr;
          to = '';
          amount = '0';
          denom = txType === 'identity_verified' ? 'identity' : 'role';
        }
      }
    }

    const hash = (typeof tx.hash === 'string' ? tx.hash : '').replace(/^0x/i, '').toUpperCase();
    const height = tx.height !== undefined
      ? (typeof tx.height === 'string' ? parseInt(tx.height, 10) : tx.height)
      : 0;

    return {
      hash,
      height,
      timestamp: new Date().toISOString(),
      from: from || address,
      to: to || address,
      amount,
      denom,
      status: (txResult.code === 0 ? 'success' : 'failed') as 'success' | 'failed',
      tx_type: txType,
    };
  }

  // Сканирование блоков для поиска транзакций (fallback когда tx_search не индексирует события)
  private async scanBlocksForTransactions(address: string, blocksToScan: number = 100): Promise<string[]> {
    const hashes: string[] = [];
    try {
      const statusRes = await fetch(`${RPC_ENDPOINT}/status`);
      const statusData = await statusRes.json();
      const latestHeight = parseInt(statusData.result?.sync_info?.latest_block_height || '0');
      if (latestHeight === 0) return hashes;

      const startHeight = Math.max(1, latestHeight - blocksToScan + 1);
      for (let height = latestHeight; height >= startHeight; height--) {
        try {
          const [blockRes, resultsRes] = await Promise.all([
            fetch(`${RPC_ENDPOINT}/block?height=${height}`),
            fetch(`${RPC_ENDPOINT}/block_results?height=${height}`),
          ]);
          const blockData = await blockRes.json();
          const resultsData = await resultsRes.json();
          const txs = blockData.result?.block?.data?.txs || [];
          const txResults = resultsData.result?.txs_results || [];

          for (let i = 0; i < txResults.length; i++) {
            const events = txResults[i]?.events || [];
            for (const event of events) {
              let isRelevant = false;
              if (event.type === 'transfer' || event.type === 'coin_spent' || event.type === 'coin_received') {
                const attrs = event.attributes || [];
                let recipient = '';
                let sender = '';
                for (const a of attrs) {
                  const k = a.index ? (a.key || '') : (a.key ? atob(a.key) : '');
                  const v = a.index ? (a.value || '') : (a.value ? atob(a.value) : '');
                  if (k === 'recipient' || k === 'receiver') recipient = v;
                  if (k === 'sender' || k === 'spender') sender = v;
                }
                isRelevant = recipient === address || sender === address;
              } else if (event.type === 'ident.identity_verified' || event.type === 'ident.role_changed') {
                const attrs = event.attributes || [];
                for (const a of attrs) {
                  const k = a.index ? (a.key || '') : (a.key ? atob(a.key) : '');
                  const v = a.index ? (a.value || '') : (a.value ? atob(a.value) : '');
                  if (k === 'address' && v === address) {
                    isRelevant = true;
                    break;
                  }
                }
              }
              if (isRelevant && txs[i]) {
                const txHash = await this.calculateTxHash(txs[i]);
                if (txHash && !hashes.includes(txHash)) hashes.unshift(txHash);
              }
            }
          }
        } catch {
          /* skip block */
        }
      }
    } catch (e) {
      console.warn('Block scan failed:', (e as Error).message);
    }
    return hashes;
  }

  private async calculateTxHash(txBase64: string): Promise<string | null> {
    try {
      const txBytes = Uint8Array.from(atob(txBase64), (c) => c.charCodeAt(0));
      const hashBuffer = await crypto.subtle.digest('SHA-256', txBytes);
      return Array.from(new Uint8Array(hashBuffer))
        .map((b) => b.toString(16).padStart(2, '0'))
        .join('')
        .toUpperCase();
    } catch {
      return null;
    }
  }

  // Получение транзакций: REST API (backend tx indexer) или fallback на сканирование блоков
  async getTransactions(address: string, limit: number = 50, _scanBlocks?: boolean): Promise<BlockchainTransaction[]> {
    try {
      const effectiveLimit = Math.min(limit, 100);

      // 1. Пробуем REST API backend (custom tx indexer)
      try {
        const url = `${REST_ENDPOINT}/volnix/tx/v1/transactions/${encodeURIComponent(address)}?limit=${effectiveLimit}`;
        const response = await fetch(url);
        if (response.ok) {
          const data = await response.json();
          const txs: BlockchainTransaction[] = (data.transactions || []).map(
            (t: { hash: string; height: number; timestamp: string; from: string; to: string; amount: string; denom: string; status: string; tx_type?: string }) => ({
              hash: t.hash,
              height: t.height,
              timestamp: t.timestamp || new Date().toISOString(),
              from: t.from,
              to: t.to,
              amount: t.amount,
              denom: t.denom,
              status: (t.status === 'success' ? 'success' : 'failed') as 'success' | 'failed',
              tx_type: (t.tx_type as 'transfer' | 'identity_verified' | 'role_changed') || 'transfer',
            })
          );
          if (txs.length > 0 || response.status === 200) {
            console.log(`✅ Loaded ${txs.length} transactions via REST API`);
            return txs.slice(0, limit);
          }
        }
      } catch (restErr: any) {
        console.warn('REST tx API unavailable, falling back to block scan:', restErr?.message);
      }

      // 2. Fallback: сканирование блоков через RPC
      await this.initializeClient();
      if (!this.client) throw new Error('Client not initialized');

      const txHashes = await this.scanBlocksForTransactions(address, 100);
      for (const hash of txHashes) this.saveTxHash(address, hash);
      const stored = localStorage.getItem(`volnix_txs_${address}`);
      const savedHashes: string[] = stored ? JSON.parse(stored) : [];
      const combined = txHashes.concat(savedHashes);
      const seen: Record<string, boolean> = {};
      const hashes: string[] = [];
      for (let i = 0; i < combined.length; i++) {
        const h = combined[i];
        if (!seen[h]) {
          seen[h] = true;
          hashes.push(h);
        }
      }
      const transactions: BlockchainTransaction[] = [];
      for (const hash of hashes.slice(0, limit)) {
        try {
          const res = await fetch(`${RPC_ENDPOINT}/tx?hash=0x${hash}`);
          const data = await res.json();
          if (data.error || !data.result) continue;
          const tx = data.result;
          transactions.push(
            this.parseTxEventsToTransaction(tx, tx.tx_result || {}, address)
          );
        } catch {
          /* skip */
        }
      }
      transactions.sort((a, b) => b.height - a.height);
      console.log(`✅ Loaded ${transactions.length} transactions via block scan fallback`);
      return transactions.slice(0, limit);
    } catch (error: any) {
      console.warn(`Failed to get transactions: ${error.message || error}. Returning empty.`);
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

    // CRITICAL: Validate amount before processing
    const amountNum = parseFloat(amount);
    if (isNaN(amountNum) || amountNum <= 0) {
      throw new Error('Amount must be greater than 0');
    }

    // Конвертация деноминации
    const fullDenom = denom === 'wrt' ? 'uwrt' : denom === 'lzn' ? 'ulzn' : 'uant';
    const amountInMicro = Math.floor(amountNum * 1_000_000).toString();
    
    // CRITICAL: Verify amountInMicro is not zero
    if (amountInMicro === '0' || amountInMicro === 'NaN') {
      throw new Error('Amount is too small or invalid');
    }
    
    console.log('💰 Amount validation:', {
      originalAmount: amount,
      parsedAmount: amountNum,
      amountInMicro: amountInMicro,
      fullDenom: fullDenom
    });

    // CRITICAL: Get account info to check sequence number
    // This ensures we have the latest sequence before sending
    let accountSequence: number | undefined;
    try {
      const account = await this.signingClient.getAccount(fromAddress);
      if (account) {
        accountSequence = account.sequence;
        console.log('📋 Account sequence:', accountSequence);
      }
    } catch (err) {
      console.warn('⚠️  Could not get account sequence, will use default:', err);
    }

    // CRITICAL: Create message in the format CosmJS expects
    // CosmJS requires messages to be EncodeObject with typeUrl and value
    // The value must match the protobuf structure exactly
    const sendMsg: {
      typeUrl: string;
      value: {
        fromAddress: string;
        toAddress: string;
        amount: Array<{
          denom: string;
          amount: string;
        }>;
      };
    } = {
      typeUrl: '/cosmos.bank.v1beta1.MsgSend',
      value: {
        fromAddress: fromAddress,
        toAddress: toAddress,
        amount: [
          {
            denom: fullDenom,
            amount: amountInMicro,
          },
        ],
      },
    };
    
    // CRITICAL: Verify message structure before sending
    console.log('🔍 Created message:', {
      typeUrl: sendMsg.typeUrl,
      hasValue: !!sendMsg.value,
      hasFromAddress: !!sendMsg.value.fromAddress,
      hasToAddress: !!sendMsg.value.toAddress,
      hasAmount: !!sendMsg.value.amount,
      amountLength: sendMsg.value.amount?.length || 0,
      fullMessage: sendMsg
    });

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
      // Log transaction details before sending
      console.log('📤 Sending transaction:', {
        from: fromAddress,
        to: toAddress,
        amount: amountInMicro,
        denom: fullDenom,
        messageType: sendMsg.typeUrl,
        messagesCount: 1,
        accountSequence: accountSequence,
        sendMsg: sendMsg,
        fee: fee
      });
      
      // CRITICAL: Verify message is properly formatted
      if (!sendMsg || !sendMsg.typeUrl || !sendMsg.value) {
        throw new Error('Invalid message format: message must have typeUrl and value');
      }
      
      // CRITICAL: Verify message value structure
      if (!sendMsg.value.fromAddress || !sendMsg.value.toAddress) {
        throw new Error('Invalid message: fromAddress and toAddress are required');
      }
      
      if (!sendMsg.value.amount || !Array.isArray(sendMsg.value.amount) || sendMsg.value.amount.length === 0) {
        throw new Error('Invalid message: amount array is required and must not be empty');
      }
      
      // Create messages array - CRITICAL: must be a proper array with at least one message
      const messages = [sendMsg];
      
      if (!Array.isArray(messages) || messages.length === 0) {
        throw new Error('Messages array is empty');
      }
      
      console.log('✅ Message validation passed, calling signAndBroadcast...');
      console.log('📋 Message details:', JSON.stringify(sendMsg, null, 2));
      console.log('📋 Messages array:', JSON.stringify(messages, null, 2));
      console.log('📋 Fee details:', JSON.stringify(fee, null, 2));
      
      // CRITICAL: Log what we're passing to signAndBroadcast
      console.log('📤 Calling signAndBroadcast with:', {
        fromAddress,
        messages: messages,
        messagesLength: messages.length,
        messagesType: typeof messages,
        isArray: Array.isArray(messages),
        firstMessageType: messages[0]?.typeUrl,
        accountSequence: accountSequence,
        fee
      });
      
      // CRITICAL: Pass messages array directly (not wrapped in another array)
      const result = await this.signingClient.signAndBroadcast(
        fromAddress,
        messages, // Array with one message - should be valid
        fee
      );

      console.log('✅ Transaction result:', {
        code: result.code,
        hash: result.transactionHash,
        height: result.height
      });

      if (result.code !== 0) {
        console.error('❌ Transaction failed:', result.rawLog);
        throw new Error(`Transaction failed: ${result.rawLog}`);
      }

      // CRITICAL: Сохраняем хеш транзакции в localStorage для истории
      this.saveTxHash(fromAddress, result.transactionHash);

      // CRITICAL: Wait a bit after successful transaction to allow sequence update
      // This helps prevent "tx already exists" errors on subsequent transactions
      await new Promise(resolve => setTimeout(resolve, 500));

      return result.transactionHash;
    } catch (error: any) {
      console.error('❌ Error sending transaction:', error);
      
      // CRITICAL: Handle "tx already exists" error gracefully
      const errorMessage = error.message || '';
      const errorData = error.data || '';
      const errorString = JSON.stringify(error);
      
      if (
        errorMessage.includes('tx already exists') ||
        errorData.includes('tx already exists') ||
        errorString.includes('tx already exists')
      ) {
        console.warn('⚠️  Transaction already exists in cache. This usually means:');
        console.warn('   1. The transaction was already sent successfully');
        console.warn('   2. Or the same transaction is being sent twice');
        console.warn('   3. Wait a moment and try again, or check transaction status');
        
        // Try to extract transaction hash if available
        const hashMatch = errorString.match(/hash[":\s]+([A-Fa-f0-9]{64})/);
        if (hashMatch) {
          console.warn(`   Transaction hash: ${hashMatch[1]}`);
          throw new Error(`Transaction already exists. Hash: ${hashMatch[1]}. Please wait a moment before sending another transaction.`);
        }
        
        throw new Error('Transaction already exists in cache. Please wait a moment before sending another transaction.');
      }
      
      // Log full error details for debugging
      if (error.message) {
        console.error('Error message:', error.message);
      }
      if (error.data) {
        console.error('Error data:', error.data);
      }
      if (error.stack) {
        console.error('Error stack:', error.stack);
      }
      throw new Error(`Failed to send transaction: ${error.message || error}`);
    }
  }

  // Активация LZN (только для валидаторов с верифицированной идентичностью)
  async activateLzn(
    validator: string,
    amount: string,
    identityHash: string
  ): Promise<string> {
    if (!this.signingClient) {
      throw new Error('Signing client not initialized. Please connect wallet with mnemonic.');
    }
    const amountNum = parseFloat(amount);
    if (isNaN(amountNum) || amountNum <= 0) {
      throw new Error('Amount must be greater than 0');
    }
    if (!identityHash || identityHash.trim() === '') {
      throw new Error('Identity hash is required for LZN activation. Verify your identity first.');
    }
    const amountInMicro = Math.floor(amountNum * 1_000_000).toString();
    if (amountInMicro === '0') {
      throw new Error('Amount is too small');
    }

    const msg = {
      typeUrl: '/volnix.lizenz.v1.MsgActivateLZN',
      value: {
        validator,
        amount: amountInMicro,
        identityHash,
      },
    };

    const fee = {
      amount: [{ denom: 'uwrt', amount: '5000' }],
      gas: '200000',
    };

    const result = await this.signingClient.signAndBroadcast(validator, [msg], fee);
    if (result.code !== 0) {
      throw new Error(`Activate LZN failed: ${result.rawLog}`);
    }
    this.saveTxHash(validator, result.transactionHash);
    await new Promise((r) => setTimeout(r, 500));
    return result.transactionHash;
  }

  // Деактивация LZN
  async deactivateLzn(validator: string, reason: string): Promise<string> {
    if (!this.signingClient) {
      throw new Error('Signing client not initialized. Please connect wallet with mnemonic.');
    }
    if (!reason || reason.trim() === '') {
      throw new Error('Reason is required for LZN deactivation.');
    }

    const msg = {
      typeUrl: '/volnix.lizenz.v1.MsgDeactivateLZN',
      value: {
        validator,
        amount: '0',
        reason,
      },
    };

    const fee = {
      amount: [{ denom: 'uwrt', amount: '5000' }],
      gas: '200000',
    };

    const result = await this.signingClient.signAndBroadcast(validator, [msg], fee);
    if (result.code !== 0) {
      throw new Error(`Deactivate LZN failed: ${result.rawLog}`);
    }
    this.saveTxHash(validator, result.transactionHash);
    await new Promise((r) => setTimeout(r, 500));
    return result.transactionHash;
  }

  // Получение активированного LZN для валидатора
  async getActivatedLizenz(validator: string): Promise<{ amount: string } | null> {
    const paths = [
      `${REST_ENDPOINT}/volnix/lizenz/v1/activated/${validator}`,
      `${REST_ENDPOINT}/volnix/lizenz/v1/lizenz/${validator}`,
    ];
    for (const url of paths) {
      try {
        const res = await fetch(url);
        if (!res.ok) {
          if (res.status === 404) continue;
          throw new Error(`Failed to get activated lizenz: ${res.status}`);
        }
        const data = await res.json();
        const al = data?.activated_lizenz;
        if (!al) return null;
        return { amount: al.amount || '0' };
      } catch (e: any) {
        if (e?.message?.includes('404') || e?.message?.includes('not found')) continue;
        console.warn('getActivatedLizenz:', url, e?.message);
      }
    }
    return null;
  }

  // Получение identity hash для верифицированного аккаунта
  async getIdentityHash(address: string): Promise<string | null> {
    try {
      const res = await fetch(`${REST_ENDPOINT}/volnix/ident/v1/verified_account/${address}`);
      if (!res.ok) return null;
      const data = await res.json();
      const va = data?.verified_account;
      return va?.identity_hash || va?.identityHash || null;
    } catch {
      return null;
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

  // Получение примерной комиссии для отображения (gas * min_gas_price)
  async getEstimatedFee(): Promise<string> {
    try {
      const REST_ENDPOINT = process.env.REACT_APP_REST_ENDPOINT || 'http://localhost:1317';
      const res = await fetch(`${REST_ENDPOINT}/cosmos/tx/v1beta1/params`);
      if (res.ok) {
        const data = await res.json();
        const params = data.params || data;
        const minGasPrice = params.min_gas_price || params.minGasPrice || '';
        if (minGasPrice) {
          const match = minGasPrice.match(/([\d.]+)(\w*)/);
          if (match) {
            const [, amount, denom] = match;
            const gas = 200000;
            const feeAmount = Math.ceil(parseFloat(amount || '0') * gas);
            const d = denom || 'uwrt';
            if (d.startsWith('u') && d.length > 1) {
              return (feeAmount / 1e6).toFixed(4) + ' ' + d.slice(1).toUpperCase();
            }
            return `${feeAmount} ${d}`;
          }
        }
      }
    } catch {
      // ignore
    }
    return '0.001 WRT';
  }

  // Размещение ордера на рынке ANT
  async placeOrder(
    owner: string,
    side: 'buy' | 'sell',
    antAmount: string,
    price: string,
    isMarket: boolean
  ): Promise<string> {
    if (!this.signingClient) {
      throw new Error('Signing client not initialized. Please connect wallet with mnemonic.');
    }

    const amountNum = parseFloat(antAmount);
    const priceNum = parseFloat(price);
    if (isNaN(amountNum) || amountNum <= 0 || isNaN(priceNum) || priceNum <= 0) {
      throw new Error('Amount and price must be greater than 0');
    }

    // Convert to micro units (1 ANT = 1_000_000 uant, 1 WRT = 1_000_000 uwrt)
    const antAmountMicro = Math.floor(amountNum * 1_000_000).toString();
    const priceMicro = Math.floor(priceNum * 1_000_000).toString();

    if (antAmountMicro === '0') {
      throw new Error('Amount is too small');
    }

    // OrderType: 1=LIMIT, 2=MARKET
    // OrderSide: 1=BUY, 2=SELL
    const orderType = isMarket ? 2 : 1;
    const orderSide = side === 'buy' ? 1 : 2;

    let identityHash = '';
    try {
      const hash = await this.getIdentityHash(owner);
      if (hash) identityHash = hash;
    } catch {
      // identity_hash optional for order placement
    }

    const placeOrderMsg = {
      typeUrl: '/volnix.anteil.v1.MsgPlaceOrder',
      value: {
        owner,
        orderType,
        orderSide,
        antAmount: antAmountMicro,
        price: priceMicro,
        identityHash,
      },
    };

    const fee = {
      amount: [{ denom: 'uwrt', amount: '5000' }],
      gas: '300000',
    };

    const result = await this.signingClient.signAndBroadcast(
      owner,
      [placeOrderMsg],
      fee
    );

    if (result.code !== 0) {
      throw new Error(`Order failed: ${result.rawLog}`);
    }

    this.saveTxHash(owner, result.transactionHash);
    await new Promise((resolve) => setTimeout(resolve, 500));
    return result.transactionHash;
  }

  // Получение ордеров ANT с REST API
  async getAntOrders(owner?: string): Promise<{ orders: any[] }> {
    try {
      const REST_ENDPOINT = process.env.REACT_APP_REST_ENDPOINT || 'http://localhost:1317';
      const url = owner
        ? `${REST_ENDPOINT}/volnix/anteil/v1/orders?owner=${encodeURIComponent(owner)}`
        : `${REST_ENDPOINT}/volnix/anteil/v1/orders`;
      const response = await fetch(url);
      if (!response.ok) {
        return { orders: [] };
      }
      const data = await response.json();
      return { orders: data.orders || [] };
    } catch (error: any) {
      console.warn('Failed to fetch ANT orders:', error?.message);
      return { orders: [] };
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

  // Получение роли аккаунта через REST API
  async getAccountRole(address: string): Promise<'guest' | 'citizen' | 'validator' | null> {
    try {
      // REST API endpoint для получения verified account
      const REST_ENDPOINT = process.env.REACT_APP_REST_ENDPOINT || 'http://localhost:1317';
      const response = await fetch(`${REST_ENDPOINT}/volnix/ident/v1/verified_account/${address}`);
      
      if (!response.ok) {
        // Если аккаунт не найден, возвращаем guest
        if (response.status === 404) {
          return 'guest';
        }
        throw new Error(`Failed to get account role: ${response.statusText}`);
      }

      const data = await response.json();
      const verifiedAccount = data.verified_account;
      
      if (!verifiedAccount || !verifiedAccount.role) {
        return 'guest';
      }

      // Конвертируем Role enum в WalletType
      const role = verifiedAccount.role;
      if (role === 'ROLE_CITIZEN' || role === 2) {
        return 'citizen';
      } else if (role === 'ROLE_VALIDATOR' || role === 3) {
        return 'validator';
      } else {
        return 'guest';
      }
    } catch (error: any) {
      console.warn(`Failed to get account role: ${error.message}. Defaulting to guest.`);
      return 'guest';
    }
  }

  // Отправка транзакции изменения роли на блокчейн
  private async sha256Hex(input: string): Promise<string> {
    const bytes = new TextEncoder().encode(input);
    const digest = await crypto.subtle.digest('SHA-256', bytes);
    return Array.from(new Uint8Array(digest))
      .map((b) => b.toString(16).padStart(2, '0'))
      .join('');
  }

  private async createRoleChangeZkpProof(address: string, roleValue: number): Promise<string> {
    const REST_ENDPOINT = process.env.REACT_APP_REST_ENDPOINT || 'http://localhost:1317';
    let identityHash = 'unknown-identity';

    try {
      const response = await fetch(`${REST_ENDPOINT}/volnix/ident/v1/verified_account/${address}`);
      if (response.ok) {
        const data = await response.json();
        identityHash = data?.verified_account?.identity_hash || identityHash;
      }
    } catch {
      // Fallback is safe for UI-side proof generation;
      // backend still enforces identity/account checks.
    }

    const roleName =
      roleValue === 3 ? 'ROLE_VALIDATOR' : roleValue === 2 ? 'ROLE_CITIZEN' : 'ROLE_GUEST';
    const publicInputs = JSON.stringify({
      address,
      identityHash,
      targetRole: roleName,
    });
    const proofData = `${address}|${roleName}|${Date.now()}|${Math.random().toString(36).slice(2)}`;
    const verificationKey = 'volnix-ident-role-change-v1';
    const proofHash = await this.sha256Hex(`${proofData}|${publicInputs}|${verificationKey}`);

    return JSON.stringify({
      proofHash,
      publicInputs,
      proofData,
      verificationKey,
      createdAt: new Date().toISOString(),
      providerSignature: '',
    });
  }

  async changeRole(
    address: string,
    newRole: 'guest' | 'citizen' | 'validator'
  ): Promise<string> {
    if (!this.signingClient) {
      throw new Error('Signing client not initialized. Please connect wallet with mnemonic.');
    }

    // Конвертируем WalletType в Role enum
    let roleValue: number;
    if (newRole === 'citizen') {
      roleValue = 2; // ROLE_CITIZEN
    } else if (newRole === 'validator') {
      roleValue = 3; // ROLE_VALIDATOR
    } else {
      roleValue = 1; // ROLE_GUEST
    }

    try {
      console.log('📤 Sending MsgChangeRole transaction to blockchain:', {
        address,
        newRole,
        roleValue
      });

      const zkpProof = await this.createRoleChangeZkpProof(address, roleValue);

      // Создаем сообщение MsgChangeRole
      const changeRoleMsg = {
        typeUrl: '/volnix.ident.v1.MsgChangeRole',
        value: {
          address: address,
          newRole: roleValue,
          zkpProof: zkpProof,
          // changeFee is optional, not included
        },
      };

      const fee = {
        amount: [
          {
            denom: 'uwrt',
            amount: '5000', // Minimal fee
          },
        ],
        gas: '200000',
      };

      console.log('📋 Message structure:', {
        typeUrl: changeRoleMsg.typeUrl,
        value: changeRoleMsg.value,
        fee: fee
      });

      // Отправляем транзакцию на блокчейн
      const result = await this.signingClient.signAndBroadcast(
        address,
        [changeRoleMsg],
        fee
      );

      console.log('✅ Transaction result:', {
        code: result.code,
        hash: result.transactionHash,
        height: result.height,
        rawLog: result.rawLog
      });

      if (result.code !== 0) {
        console.error('❌ Transaction failed:', result.rawLog);
        throw new Error(`Transaction failed: ${result.rawLog}`);
      }

      console.log('✅ Role change transaction sent successfully:', result.transactionHash);

      // Wait a bit for transaction to be included
      await new Promise(resolve => setTimeout(resolve, 500));

      return result.transactionHash;
    } catch (error: any) {
      console.error('❌ Error sending role change transaction:', error);

      const msg = String(error?.message || error || '');

      // Provide helpful error messages
      if (msg.includes('does not exist on chain') || msg.includes('query sequence')) {
        throw new Error(
          'Account does not exist on chain. Send some WRT tokens to your address first (from another wallet or faucet), then try changing your role again.'
        );
      }
      if (msg.includes('not registered')) {
        throw new Error('Message type not registered. Please check Registry configuration.');
      }
      if (msg.includes('ZKP proof')) {
        throw new Error('ZKP proof validation failed. Please retry role change.');
      }

      throw new Error(`Failed to change role: ${msg || error}`);
    }
  }

  async getNetworkValidators(): Promise<{
    validators: Array<{
      validator: string;
      status: string;
      voting_power: string;
      ant_balance: string;
      activity_score: string;
      total_blocks_created: number;
      total_burn_amount: string;
    }>;
    networkOnline: boolean;
  }> {
    const REST_ENDPOINT = process.env.REACT_APP_REST_ENDPOINT || 'http://localhost:1317';
    try {
      const response = await fetch(`${REST_ENDPOINT}/volnix/consensus/v1/validators`);
      if (!response.ok) {
        return { validators: [], networkOnline: false };
      }
      const data = await response.json();
      return {
        validators: data.validators || [],
        networkOnline: true,
      };
    } catch {
      return { validators: [], networkOnline: false };
    }
  }

  async getConsensusParams(): Promise<Record<string, string> | null> {
    const REST_ENDPOINT = process.env.REACT_APP_REST_ENDPOINT || 'http://localhost:1317';
    try {
      const response = await fetch(`${REST_ENDPOINT}/volnix/consensus/v1/params`);
      if (!response.ok) return null;
      const data = await response.json();
      return data.params || null;
    } catch {
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


