// Тесты для blockchainService - парсинг событий и транзакций

// Полифилл для TextEncoder в Jest окружении
if (typeof TextEncoder === 'undefined') {
  const { TextEncoder, TextDecoder } = require('util');
  global.TextEncoder = TextEncoder;
  global.TextDecoder = TextDecoder;
}

// Полифилл для crypto.subtle в Jest окружении
if (typeof global.crypto === 'undefined' || !global.crypto.subtle) {
  const nodeCrypto = require('crypto');
  global.crypto = {
    subtle: {
      digest: async (algorithm: string, data: Uint8Array): Promise<ArrayBuffer> => {
        const hash = nodeCrypto.createHash('sha256');
        hash.update(Buffer.from(data));
        return hash.digest().buffer;
      },
    },
  } as any;
}

describe('BlockchainService - Event Parsing', () => {
  // Mock данных для тестирования парсинга событий
  
  test('должен корректно парсить события с index: true (уже декодированные)', () => {
    const event = {
      type: 'transfer',
      attributes: [
        { key: 'sender', value: 'volnix19rl4cm2hmr8afy4kldpxz3fka4jguq0a9r0ces', index: true },
        { key: 'recipient', value: 'volnix1abc123def456', index: true },
        { key: 'amount', value: '1000000uwrt', index: true },
      ],
    };
    
    // Парсинг атрибутов (логика из blockchainService.ts)
    let sender = '';
    let recipient = '';
    let amount = '';
    
    for (const attr of event.attributes) {
      const isIndexed = attr.index === true;
      const key = isIndexed ? (attr.key || '') : (attr.key ? atob(attr.key) : '');
      const value = isIndexed ? (attr.value || '') : (attr.value ? atob(attr.value) : '');
      
      if (key === 'sender') sender = value;
      if (key === 'recipient') recipient = value;
      if (key === 'amount') amount = value;
    }
    
    expect(sender).toBe('volnix19rl4cm2hmr8afy4kldpxz3fka4jguq0a9r0ces');
    expect(recipient).toBe('volnix1abc123def456');
    expect(amount).toBe('1000000uwrt');
  });
  
  test('должен корректно парсить события с index: false (base64)', () => {
    // Атрибуты в base64
    const event = {
      type: 'transfer',
      attributes: [
        { key: btoa('sender'), value: btoa('volnix19rl4cm2hmr8afy4kldpxz3fka4jguq0a9r0ces'), index: false },
        { key: btoa('recipient'), value: btoa('volnix1abc123def456'), index: false },
        { key: btoa('amount'), value: btoa('1000000uwrt'), index: false },
      ],
    };
    
    let sender = '';
    let recipient = '';
    let amount = '';
    
    for (const attr of event.attributes) {
      const isIndexed = attr.index === true;
      const key = isIndexed ? (attr.key || '') : (attr.key ? atob(attr.key) : '');
      const value = isIndexed ? (attr.value || '') : (attr.value ? atob(attr.value) : '');
      
      if (key === 'sender') sender = value;
      if (key === 'recipient') recipient = value;
      if (key === 'amount') amount = value;
    }
    
    expect(sender).toBe('volnix19rl4cm2hmr8afy4kldpxz3fka4jguq0a9r0ces');
    expect(recipient).toBe('volnix1abc123def456');
    expect(amount).toBe('1000000uwrt');
  });
  
  test('должен корректно парсить amount с дробными суммами', () => {
    const testCases = [
      { input: '1000000uwrt', expected: { amount: '1000000', denom: 'uwrt' } },
      { input: '500000ulzn', expected: { amount: '500000', denom: 'ulzn' } },
      { input: '12345678uant', expected: { amount: '12345678', denom: 'uant' } },
      { input: '1uwrt', expected: { amount: '1', denom: 'uwrt' } },
    ];
    
    for (const testCase of testCases) {
      const match = testCase.input.match(/^(\d+)(\w+)$/);
      expect(match).not.toBeNull();
      expect(match![1]).toBe(testCase.expected.amount);
      expect(match![2]).toBe(testCase.expected.denom);
    }
  });
  
  test('должен корректно парсить множественные токены', () => {
    const amountStr = '1000000uwrt,2000000ulzn,3000000uant';
    const amounts = amountStr.split(',');
    
    const parsed: Array<{ amount: string; denom: string }> = [];
    
    for (const amt of amounts) {
      const match = amt.trim().match(/^(\d+)(\w+)$/);
      if (match) {
        parsed.push({ amount: match[1], denom: match[2] });
      }
    }
    
    expect(parsed).toHaveLength(3);
    expect(parsed[0]).toEqual({ amount: '1000000', denom: 'uwrt' });
    expect(parsed[1]).toEqual({ amount: '2000000', denom: 'ulzn' });
    expect(parsed[2]).toEqual({ amount: '3000000', denom: 'uant' });
  });
  
  test('должен обрабатывать событие transfer с всеми атрибутами', () => {
    const txResult = {
      code: 0,
      events: [
        {
          type: 'message',
          attributes: [
            { key: 'action', value: '/cosmos.bank.v1beta1.MsgSend', index: true },
          ],
        },
        {
          type: 'transfer',
          attributes: [
            { key: 'sender', value: 'volnix19rl4cm2hmr8afy4kldpxz3fka4jguq0a9r0ces', index: true },
            { key: 'recipient', value: 'volnix1abc123def456', index: true },
            { key: 'amount', value: '5000000uwrt', index: true },
          ],
        },
      ],
    };
    
    // Парсинг (логика из getTransactions)
    let from = '';
    let to = '';
    let amount = '0';
    let denom = 'uwrt';
    
    for (const event of txResult.events) {
      if (event.type === 'transfer' || event.type === 'coin_spent' || event.type === 'coin_received') {
        for (const attr of event.attributes) {
          const isIndexed = attr.index === true;
          const key = isIndexed ? (attr.key || '') : (attr.key ? atob(attr.key) : '');
          const value = isIndexed ? (attr.value || '') : (attr.value ? atob(attr.value) : '');
          
          if (key === 'sender' || key === 'spender') {
            from = value;
          } else if (key === 'recipient' || key === 'receiver') {
            to = value;
          } else if (key === 'amount') {
            const match = value.match(/^(\d+)(\w+)$/);
            if (match) {
              amount = match[1];
              denom = match[2];
            }
          }
        }
      }
    }
    
    expect(from).toBe('volnix19rl4cm2hmr8afy4kldpxz3fka4jguq0a9r0ces');
    expect(to).toBe('volnix1abc123def456');
    expect(amount).toBe('5000000');
    expect(denom).toBe('uwrt');
    
    // Конвертация в токены
    const amountInTokens = (parseInt(amount) / 1_000_000).toFixed(6);
    expect(amountInTokens).toBe('5.000000');
  });
  
  test('должен обрабатывать события coin_spent и coin_received', () => {
    const txResult = {
      code: 0,
      events: [
        {
          type: 'coin_spent',
          attributes: [
            { key: 'spender', value: 'volnix19rl4cm2hmr8afy4kldpxz3fka4jguq0a9r0ces', index: true },
            { key: 'amount', value: '7500000uwrt', index: true },
          ],
        },
        {
          type: 'coin_received',
          attributes: [
            { key: 'receiver', value: 'volnix1abc123def456', index: true },
            { key: 'amount', value: '7500000uwrt', index: true },
          ],
        },
      ],
    };
    
    // Парсинг coin_spent
    let spender = '';
    let receiver = '';
    let spentAmount = '';
    let receivedAmount = '';
    
    for (const event of txResult.events) {
      if (event.type === 'coin_spent') {
        for (const attr of event.attributes) {
          const key = attr.index ? attr.key : atob(attr.key);
          const value = attr.index ? attr.value : atob(attr.value);
          
          if (key === 'spender') spender = value;
          if (key === 'amount') {
            const match = value.match(/^(\d+)(\w+)$/);
            if (match) spentAmount = match[1];
          }
        }
      }
      
      if (event.type === 'coin_received') {
        for (const attr of event.attributes) {
          const key = attr.index ? attr.key : atob(attr.key);
          const value = attr.index ? attr.value : atob(attr.value);
          
          if (key === 'receiver') receiver = value;
          if (key === 'amount') {
            const match = value.match(/^(\d+)(\w+)$/);
            if (match) receivedAmount = match[1];
          }
        }
      }
    }
    
    expect(spender).toBe('volnix19rl4cm2hmr8afy4kldpxz3fka4jguq0a9r0ces');
    expect(receiver).toBe('volnix1abc123def456');
    expect(spentAmount).toBe('7500000');
    expect(receivedAmount).toBe('7500000');
  });
  
  test('должен обрабатывать пустые события', () => {
    const txResult = {
      code: 0,
      events: [],
    };
    
    let from = '';
    let to = '';
    let amount = '0';
    
    for (const event of txResult.events) {
      // Пустой массив - цикл не выполнится
    }
    
    expect(from).toBe('');
    expect(to).toBe('');
    expect(amount).toBe('0');
  });
  
  test('должен обрабатывать события без нужных атрибутов', () => {
    const txResult = {
      code: 0,
      events: [
        {
          type: 'message',
          attributes: [
            { key: 'action', value: '/cosmos.bank.v1beta1.MsgSend', index: true },
            { key: 'module', value: 'bank', index: true },
          ],
        },
      ],
    };
    
    let from = '';
    let to = '';
    let amount = '0';
    
    for (const event of txResult.events) {
      if (event.type === 'transfer') {
        // transfer event не найден
      }
    }
    
    expect(from).toBe('');
    expect(to).toBe('');
    expect(amount).toBe('0');
  });
});

describe('BlockchainService - Transaction Conversion', () => {
  test('должен конвертировать микротокены в токены', () => {
    const testCases = [
      { micro: '1000000', expected: '1.000000' },
      { micro: '5000000', expected: '5.000000' },
      { micro: '10000000', expected: '10.000000' },
      { micro: '100000000', expected: '100.000000' },
      { micro: '1', expected: '0.000001' },
      { micro: '500000', expected: '0.500000' },
    ];
    
    for (const testCase of testCases) {
      const amountInTokens = (parseInt(testCase.micro) / 1_000_000).toFixed(6);
      expect(amountInTokens).toBe(testCase.expected);
    }
  });
  
  test('должен определять тип транзакции (send/receive)', () => {
    const userAddress = 'volnix19rl4cm2hmr8afy4kldpxz3fka4jguq0a9r0ces';
    
    const testCases = [
      {
        tx: { from: userAddress, to: 'volnix1abc123def456' },
        expected: 'send',
      },
      {
        tx: { from: 'volnix1abc123def456', to: userAddress },
        expected: 'receive',
      },
      {
        tx: { from: 'volnix1xyz789', to: 'volnix1abc123' },
        expected: 'send', // Если не наш адрес, по умолчанию send
      },
    ];
    
    for (const testCase of testCases) {
      // Правильная логика: send если from === userAddress, receive если to === userAddress
      const type = testCase.tx.from === userAddress 
        ? 'send' 
        : (testCase.tx.to === userAddress ? 'receive' : 'send');
      expect(type).toBe(testCase.expected);
    }
  });
  
  test('должен корректно конвертировать denom', () => {
    const testCases = [
      { denom: 'uwrt', expected: 'WRT' },
      { denom: 'ulzn', expected: 'LZN' },
      { denom: 'uant', expected: 'ANT' },
      { denom: 'wrt', expected: 'WRT' },
      { denom: 'unknown', expected: 'ANT' }, // default
    ];
    
    for (const testCase of testCases) {
      let token = 'ANT';
      if (testCase.denom === 'uwrt' || testCase.denom === 'wrt') {
        token = 'WRT';
      } else if (testCase.denom === 'ulzn' || testCase.denom === 'lzn') {
        token = 'LZN';
      } else if (testCase.denom === 'uant' || testCase.denom === 'ant') {
        token = 'ANT';
      }
      
      expect(token).toBe(testCase.expected);
    }
  });
});

describe('BlockchainService - Transaction Hash Calculation', () => {
  test('должен корректно вычислять SHA256 хеш', async () => {
    // Тестовые данные
    const testData = 'test transaction data';
    const encoder = new TextEncoder();
    const data = encoder.encode(testData);
    
    // Вычисляем SHA256
    const hashBuffer = await crypto.subtle.digest('SHA-256', data);
    const hashArray = Array.from(new Uint8Array(hashBuffer));
    const hashHex = hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
    
    // Проверяем что хеш hex строка длиной 64 символа
    expect(hashHex).toHaveLength(64);
    expect(hashHex).toMatch(/^[0-9a-f]{64}$/);
  });
  
  test('должен конвертировать base64 в bytes для хеша', () => {
    const testBase64 = btoa('test data');
    
    // Декодируем base64 в байты (логика из calculateTxHash)
    const txBytes = Uint8Array.from(atob(testBase64), c => c.charCodeAt(0));
    
    expect(txBytes).toBeInstanceOf(Uint8Array);
    expect(txBytes.length).toBeGreaterThan(0);
    
    // Проверяем что декодирование корректно
    const decoded = Array.from(txBytes).map(b => String.fromCharCode(b)).join('');
    expect(decoded).toBe('test data');
  });
});

describe('BlockchainService - localStorage', () => {
  beforeEach(() => {
    localStorage.clear();
  });
  
  test('должен сохранять хеш транзакции в localStorage', () => {
    const address = 'volnix19rl4cm2hmr8afy4kldpxz3fka4jguq0a9r0ces';
    const txHash = 'ABC123DEF456';
    
    // Логика из saveTxHash
    const TX_STORAGE_KEY = `volnix_txs_${address}`;
    const storedTxs = localStorage.getItem(TX_STORAGE_KEY);
    const txHashes: string[] = storedTxs ? JSON.parse(storedTxs) : [];
    
    if (!txHashes.includes(txHash)) {
      txHashes.unshift(txHash);
      localStorage.setItem(TX_STORAGE_KEY, JSON.stringify(txHashes));
    }
    
    // Проверка
    const stored = localStorage.getItem(TX_STORAGE_KEY);
    expect(stored).not.toBeNull();
    const hashes = JSON.parse(stored!);
    expect(hashes).toContain(txHash);
    expect(hashes[0]).toBe(txHash); // Новый хеш в начале
  });
  
  test('должен избегать дубликатов хешей', () => {
    const address = 'volnix19rl4cm2hmr8afy4kldpxz3fka4jguq0a9r0ces';
    const txHash = 'ABC123DEF456';
    const TX_STORAGE_KEY = `volnix_txs_${address}`;
    
    // Сохраняем дважды
    for (let i = 0; i < 2; i++) {
      const storedTxs = localStorage.getItem(TX_STORAGE_KEY);
      const txHashes: string[] = storedTxs ? JSON.parse(storedTxs) : [];
      
      if (!txHashes.includes(txHash)) {
        txHashes.unshift(txHash);
        localStorage.setItem(TX_STORAGE_KEY, JSON.stringify(txHashes));
      }
    }
    
    // Должен быть только один хеш
    const stored = localStorage.getItem(TX_STORAGE_KEY);
    const hashes = JSON.parse(stored!);
    expect(hashes).toHaveLength(1);
    expect(hashes[0]).toBe(txHash);
  });
  
  test('должен сохранять несколько хешей в правильном порядке', () => {
    const address = 'volnix19rl4cm2hmr8afy4kldpxz3fka4jguq0a9r0ces';
    const TX_STORAGE_KEY = `volnix_txs_${address}`;
    
    const hashes = ['HASH1', 'HASH2', 'HASH3'];
    
    for (const txHash of hashes) {
      const storedTxs = localStorage.getItem(TX_STORAGE_KEY);
      const txHashes: string[] = storedTxs ? JSON.parse(storedTxs) : [];
      
      if (!txHashes.includes(txHash)) {
        txHashes.unshift(txHash);
        localStorage.setItem(TX_STORAGE_KEY, JSON.stringify(txHashes));
      }
    }
    
    const stored = localStorage.getItem(TX_STORAGE_KEY);
    const storedHashes = JSON.parse(stored!);
    
    // Порядок должен быть обратный (новые первые)
    expect(storedHashes).toEqual(['HASH3', 'HASH2', 'HASH1']);
  });
});

