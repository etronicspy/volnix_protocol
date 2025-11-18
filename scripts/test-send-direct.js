#!/usr/bin/env node

/**
 * Тестовый скрипт для проверки отправки транзакции напрямую через CosmJS
 * Без использования фронтенда
 */

const { SigningStargateClient, GasPrice } = require('@cosmjs/stargate');
const { DirectSecp256k1HdWallet, Registry } = require('@cosmjs/proto-signing');
const { defaultRegistryTypes } = require('@cosmjs/stargate');

const RPC_ENDPOINT = 'http://localhost:26657';
const CHAIN_ID = 'volnix-standalone';
const PREFIX = 'volnix';

// Тестовый мнемоник (genesis account)
const SENDER_MNEMONIC = 'abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about';

async function testSendDirect() {
  console.log('🧪 Тестирование отправки транзакции напрямую через CosmJS\n');
  console.log('📋 Параметры:');
  console.log(`   RPC: ${RPC_ENDPOINT}`);
  console.log(`   Chain ID: ${CHAIN_ID}`);
  console.log(`   Prefix: ${PREFIX}\n`);

  try {
    // 1. Создаем кошелек из мнемоника
    console.log('1️⃣  Создание кошелька из мнемоника...');
    const wallet = await DirectSecp256k1HdWallet.fromMnemonic(SENDER_MNEMONIC, {
      prefix: PREFIX,
    });
    const [account] = await wallet.getAccounts();
    console.log(`   ✅ Кошелек создан: ${account.address}\n`);

    // 2. Создаем SigningStargateClient с Registry
    console.log('2️⃣  Создание SigningStargateClient с Registry...');
    const registry = new Registry(defaultRegistryTypes);
    
    const signingClient = await SigningStargateClient.connectWithSigner(
      RPC_ENDPOINT,
      wallet,
      {
        gasPrice: GasPrice.fromString('0.025uwrt'),
        registry: registry, // CRITICAL: Register types for message encoding
      }
    );
    console.log('   ✅ SigningStargateClient создан\n');

    // 3. Проверяем chain-id
    console.log('3️⃣  Проверка chain-id...');
    const chainId = await signingClient.getChainId();
    console.log(`   ✅ Chain ID: ${chainId}\n`);

    // 4. Проверяем баланс
    console.log('4️⃣  Проверка баланса отправителя...');
    const balances = await signingClient.getAllBalances(account.address);
    console.log(`   ✅ Баланс: ${JSON.stringify(balances)}\n`);

    // 5. Создаем тестовое сообщение
    console.log('5️⃣  Создание тестового сообщения...');
    const testRecipient = 'volnix1abc123def456'; // Тестовый адрес
    const sendMsg = {
      typeUrl: '/cosmos.bank.v1beta1.MsgSend',
      value: {
        fromAddress: account.address,
        toAddress: testRecipient,
        amount: [
          {
            denom: 'uwrt',
            amount: '1000000', // 1 WRT
          },
        ],
      },
    };
    console.log(`   ✅ Сообщение создано:`);
    console.log(`      Type: ${sendMsg.typeUrl}`);
    console.log(`      From: ${sendMsg.value.fromAddress}`);
    console.log(`      To: ${testRecipient}`);
    console.log(`      Amount: ${sendMsg.value.amount[0].amount} ${sendMsg.value.amount[0].denom}\n`);

    // 6. Проверяем, что сообщение может быть закодировано
    console.log('6️⃣  Проверка кодирования сообщения...');
    try {
      // Попытка закодировать сообщение через registry
      const encoded = registry.encode(sendMsg);
      console.log(`   ✅ Сообщение успешно закодировано (${encoded.length} bytes)\n`);
    } catch (encodeError) {
      console.error(`   ❌ Ошибка кодирования: ${encodeError.message}\n`);
      throw encodeError;
    }

    // 7. Создаем fee
    console.log('7️⃣  Создание fee...');
    const fee = {
      amount: [
        {
          denom: 'uwrt',
          amount: '5000', // Минимальная комиссия
        },
      ],
      gas: '200000',
    };
    console.log(`   ✅ Fee создан: ${JSON.stringify(fee)}\n`);

    // 8. Пытаемся отправить транзакцию
    console.log('8️⃣  Отправка транзакции...');
    console.log(`   Messages: [${sendMsg.typeUrl}]`);
    console.log(`   Messages count: 1`);
    console.log(`   Is array: ${Array.isArray([sendMsg])}`);
    console.log(`   First message typeUrl: ${[sendMsg][0]?.typeUrl}\n`);

    try {
      const result = await signingClient.signAndBroadcast(
        account.address,
        [sendMsg], // Массив с одним сообщением
        fee
      );

      console.log('   ✅ Транзакция отправлена!');
      console.log(`      Code: ${result.code}`);
      console.log(`      Hash: ${result.transactionHash}`);
      console.log(`      Height: ${result.height}\n`);

      if (result.code === 0) {
        console.log('✅ ✅ ✅ УСПЕХ! Транзакция принята узлом!\n');
      } else {
        console.log(`⚠️  Транзакция отклонена узлом: ${result.rawLog}\n`);
      }
    } catch (broadcastError) {
      console.error(`   ❌ Ошибка при отправке: ${broadcastError.message}`);
      if (broadcastError.stack) {
        console.error(`   Stack: ${broadcastError.stack}\n`);
      }
      throw broadcastError;
    }

    // 9. Закрываем соединение
    signingClient.disconnect();
    console.log('✅ Тест завершен\n');

  } catch (error) {
    console.error('\n❌ ❌ ❌ ОШИБКА ТЕСТА!\n');
    console.error(`Ошибка: ${error.message}`);
    if (error.stack) {
      console.error(`Stack:\n${error.stack}`);
    }
    process.exit(1);
  }
}

// Запускаем тест
testSendDirect().catch((error) => {
  console.error('Критическая ошибка:', error);
  process.exit(1);
});

