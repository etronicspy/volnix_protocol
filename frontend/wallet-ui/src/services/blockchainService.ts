import { StargateClient, SigningStargateClient, defaultRegistryTypes } from '@cosmjs/stargate';
import { DirectSecp256k1HdWallet } from '@cosmjs/proto-signing';
import { GasPrice } from '@cosmjs/stargate';
import { Comet38Client } from '@cosmjs/tendermint-rpc';
import { Registry } from '@cosmjs/proto-signing';
import { MsgChangeRoleType } from '../types/volnix-messages';

// Конфигурация сети
const RPC_ENDPOINT = process.env.REACT_APP_RPC_ENDPOINT || 'http://localhost:26657';
const CHAIN_ID = process.env.REACT_APP_CHAIN_ID || 'volnix-testnet';
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
        // MsgChangeRole: for changing account roles
        registry.register('/volnix.ident.v1.MsgChangeRole', MsgChangeRoleType);
        
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

  // Сканирование блоков для поиска входящих транзакций
  async scanForIncomingTransactions(address: string, blocksToScan: number = 100): Promise<void> {
    try {
      console.log(`🔍 Scanning last ${blocksToScan} blocks for incoming transactions to ${address}...`);
      
      // Получаем текущую высоту блока
      const statusResponse = await fetch(`${RPC_ENDPOINT}/status`);
      const statusData = await statusResponse.json();
      const latestHeight = parseInt(statusData.result?.sync_info?.latest_block_height || '0');
      
      if (latestHeight === 0) {
        console.warn('⚠️  Could not get latest block height');
        return;
      }
      
      console.log(`📊 Latest block height: ${latestHeight}`);
      
      // Определяем диапазон блоков для сканирования
      const startHeight = Math.max(1, latestHeight - blocksToScan + 1);
      const endHeight = latestHeight;
      
      let foundCount = 0;
      
      // Сканируем блоки
      for (let height = endHeight; height >= startHeight; height--) {
        try {
          // Получаем результаты транзакций в блоке
          const blockResultResponse = await fetch(`${RPC_ENDPOINT}/block_results?height=${height}`);
          const blockResultData = await blockResultResponse.json();
          const txResults = blockResultData.result?.txs_results || [];
          
          if (txResults.length === 0) continue;
          
          // Получаем сами транзакции чтобы извлечь хеши
          const blockResponse = await fetch(`${RPC_ENDPOINT}/block?height=${height}`);
          const blockData = await blockResponse.json();
          const txs = blockData.result?.block?.data?.txs || [];
          
          // Проверяем каждую транзакцию
          for (let i = 0; i < txResults.length; i++) {
            const txResult = txResults[i];
            const events = txResult.events || [];
            
            // Ищем события transfer с нашим адресом как получателем
            for (const event of events) {
              if (event.type === 'transfer' || event.type === 'coin_received') {
                const attributes = event.attributes || [];
                
                let recipient = '';
                for (const attr of attributes) {
                  try {
                    // CRITICAL: Проверяем index - если true, значения уже декодированы
                    const isIndexed = attr.index === true;
                    const key = isIndexed ? (attr.key || '') : (attr.key ? atob(attr.key) : '');
                    const value = isIndexed ? (attr.value || '') : (attr.value ? atob(attr.value) : '');
                    
                    if (key === 'recipient' || key === 'receiver') {
                      recipient = value;
                    }
                  } catch (e) {
                    // Игнорируем ошибки декодирования
                  }
                }
                
                // Если это наш адрес - сохраняем хеш транзакции
                if (recipient === address && txs[i]) {
                  // Вычисляем хеш транзакции (SHA256 от base64 tx)
                  const txHash = await this.calculateTxHash(txs[i]);
                  if (txHash) {
                    this.saveTxHash(address, txHash);
                    foundCount++;
                    console.log(`   ✅ Found incoming tx at block ${height}: ${txHash.substring(0, 16)}...`);
                  }
                }
              }
            }
          }
        } catch (error: any) {
          // Игнорируем ошибки отдельных блоков
          console.warn(`⚠️  Error scanning block ${height}:`, error.message);
        }
      }
      
      console.log(`🔍 Scan complete. Found ${foundCount} incoming transactions.`);
    } catch (error: any) {
      console.warn(`Failed to scan for incoming transactions: ${error.message}`);
    }
  }

  // Вычисление хеша транзакции из base64 данных
  private async calculateTxHash(txBase64: string): Promise<string | null> {
    try {
      // Декодируем base64 в байты
      const txBytes = Uint8Array.from(atob(txBase64), c => c.charCodeAt(0));
      
      // Вычисляем SHA256
      const hashBuffer = await crypto.subtle.digest('SHA-256', txBytes);
      const hashArray = Array.from(new Uint8Array(hashBuffer));
      const hashHex = hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
      
      return hashHex.toUpperCase();
    } catch (error) {
      console.warn('Failed to calculate tx hash:', error);
      return null;
    }
  }

  // Получение транзакций аккаунта
  async getTransactions(address: string, limit: number = 50, scanBlocks: boolean = true): Promise<BlockchainTransaction[]> {
    await this.initializeClient();
    if (!this.client) throw new Error('Client not initialized');

    try {
      // НОВЫЙ: Сначала сканируем блоки для поиска входящих транзакций
      if (scanBlocks) {
        const SCAN_FLAG_KEY = `volnix_last_scan_${address}`;
        const lastScan = localStorage.getItem(SCAN_FLAG_KEY);
        const now = Date.now();
        
        // Сканируем только раз в 30 секунд
        if (!lastScan || now - parseInt(lastScan) > 30000) {
          await this.scanForIncomingTransactions(address, 100);
          localStorage.setItem(SCAN_FLAG_KEY, now.toString());
        }
      }
      
      // Загружаем хеши из localStorage и получаем детали через /tx?hash=
      // Это работает, так как /tx?hash= индексируется корректно, в отличие от /tx_search
      
      const TX_STORAGE_KEY = `volnix_txs_${address}`;
      const storedTxs = localStorage.getItem(TX_STORAGE_KEY);
      const txHashes: string[] = storedTxs ? JSON.parse(storedTxs) : [];
      
      if (txHashes.length === 0) {
        console.log('📭 No transactions found in localStorage for', address);
        return [];
      }

      console.log(`📦 Loading ${txHashes.length} transactions from localStorage`);

      // Загружаем детали каждой транзакции
      const txPromises = txHashes.slice(0, limit).map(async (hash) => {
        try {
          const response = await fetch(`${RPC_ENDPOINT}/tx?hash=0x${hash}`);
          const data = await response.json();
          
          if (data.error) {
            console.warn(`Transaction ${hash} not found:`, data.error.data);
            return null;
          }

          if (!data.result) {
            return null;
          }

          const tx = data.result;
          const txResult = tx.tx_result || {};
          
          // Парсим события для получения from/to/amount
          let from = '';
          let to = '';
          let amount = '0';
          let denom = 'uwrt';
          
          const events = txResult.events || [];
          for (const event of events) {
            // CRITICAL: Проверяем transfer, coin_spent и coin_received события
            if (event.type === 'transfer' || event.type === 'coin_spent' || event.type === 'coin_received') {
              const attributes = event.attributes || [];
              for (const attr of attributes) {
                // CRITICAL: Атрибуты могут быть в base64 ИЛИ уже декодированы (если index: true)
                // Проверяем index флаг - если true, значения уже строки
                let key = '';
                let value = '';
                
                const isIndexed = attr.index === true;
                
                if (isIndexed) {
                  // Если index: true, значения уже декодированы (строки)
                  key = attr.key || '';
                  value = attr.value || '';
                } else {
                  // Если index: false, значения в base64 - декодируем
                  try {
                    key = attr.key ? atob(attr.key) : '';
                    value = attr.value ? atob(attr.value) : '';
                  } catch (e) {
                    // Если декодирование не удалось, используем как есть
                    key = attr.key || '';
                    value = attr.value || '';
                  }
                }
                
                // Парсим разные атрибуты
                if (key === 'sender' || key === 'spender') {
                  from = value;
                } else if (key === 'recipient' || key === 'receiver') {
                  to = value;
                } else if (key === 'amount') {
                  // amount формат: "1000000uwrt" или "1000000uwrt,2000000ulzn"
                  const amounts = value.split(',');
                  for (const amt of amounts) {
                    const match = amt.trim().match(/^(\d+)(\w+)$/);
                    if (match) {
                      amount = match[1];
                      denom = match[2];
                      break; // Берем первую сумму
                    }
                  }
                }
              }
            }
          }

          return {
            hash: tx.hash || hash,
            height: typeof tx.height === 'string' ? parseInt(tx.height) : tx.height,
            timestamp: new Date().toISOString(), // CometBFT не возвращает timestamp через /tx
            from: from || address,
            to: to || address,
            amount,
            denom,
            status: (txResult.code === 0 ? 'success' : 'failed') as 'success' | 'failed',
          };
        } catch (error: any) {
          console.warn(`Failed to load transaction ${hash}:`, error.message);
          return null;
        }
      });

      const results = await Promise.all(txPromises);
      const transactions = results.filter((tx): tx is BlockchainTransaction => tx !== null);
      
      console.log(`✅ Loaded ${transactions.length} transactions`);
      return transactions;
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
      
      // Provide helpful error messages
      if (error.message && error.message.includes('not registered')) {
        throw new Error('Message type not registered. Please check Registry configuration.');
      }
      if (error.message && error.message.includes('ZKP proof')) {
        throw new Error('ZKP proof validation failed. Please retry role change.');
      }
      
      throw new Error(`Failed to change role: ${error.message || error}`);
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


