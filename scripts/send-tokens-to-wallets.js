#!/usr/bin/env node
/**
 * Скрипт для отправки токенов на три кошелька
 * Использование: node scripts/send-tokens-to-wallets.js
 */

const { StargateClient, SigningStargateClient } = require('@cosmjs/stargate');
const { DirectSecp256k1HdWallet } = require('@cosmjs/proto-signing');
const { GasPrice } = require('@cosmjs/stargate');

const RPC_ENDPOINT = process.env.RPC_ENDPOINT || 'http://localhost:26657';
const CHAIN_ID = process.env.CHAIN_ID || 'volnix-standalone';
const PREFIX = 'volnix';

// Адреса получателей (из скриншотов)
const RECIPIENTS = [
  'vo1n1x18xxeuuqd37xtp52luuqpw3acfw0cgk3vvea3v',
  'vo1nix19tvhq59sfffvm37cm0d9pkf6jyl3sn7ev5try9q',
  'volnix1kfm2jun5v4lacd4xrzpnsepm7y0eesrmf3e41r'
];

// Количество токенов для отправки (100 каждого типа)
const AMOUNT = 100;
const AMOUNT_IN_MICRO = AMOUNT * 1_000_000; // Конвертация в микро-единицы

// Тестовая мнемоника для отправителя (нужно заменить на реальную с балансом)
// Или использовать genesis аккаунт
const SENDER_MNEMONIC = process.env.SENDER_MNEMONIC || 'abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about';

async function sendTokens() {
  console.log('🚀 Начинаем отправку токенов...\n');
  console.log(`📡 RPC Endpoint: ${RPC_ENDPOINT}`);
  console.log(`⛓️  Chain ID: ${CHAIN_ID}\n`);

  try {
    // Создаем кошелек отправителя
    console.log('🔑 Создание кошелька отправителя...');
    const wallet = await DirectSecp256k1HdWallet.fromMnemonic(SENDER_MNEMONIC, {
      prefix: PREFIX,
    });

    const [account] = await wallet.getAccounts();
    console.log(`✅ Адрес отправителя: ${account.address}\n`);

    // Подключаемся к сети
    console.log('🔌 Подключение к сети...');
    const client = await StargateClient.connect(RPC_ENDPOINT);
    const chainId = await client.getChainId();
    console.log(`✅ Подключено. Chain ID: ${chainId}\n`);

    // Проверяем баланс отправителя
    const senderBalances = await client.getAllBalances(account.address);
    console.log('💰 Баланс отправителя:');
    senderBalances.forEach(b => {
      console.log(`   ${b.amount} ${b.denom}`);
    });
    console.log('');

    // Создаем подписывающий клиент
    console.log('✍️  Создание подписывающего клиента...');
    const signingClient = await SigningStargateClient.connectWithSigner(
      RPC_ENDPOINT,
      wallet,
      {
        gasPrice: GasPrice.fromString('0.025uwrt'),
      }
    );
    console.log('✅ Готов к отправке транзакций\n');

    // Отправляем токены на каждый адрес
    const tokens = [
      { denom: 'uwrt', name: 'WRT' },
      { denom: 'ulzn', name: 'LZN' },
      { denom: 'uant', name: 'ANT' }
    ];

    for (const recipient of RECIPIENTS) {
      console.log(`\n📤 Отправка токенов на ${recipient}...`);
      
      for (const token of tokens) {
        try {
          const sendMsg = {
            typeUrl: '/cosmos.bank.v1beta1.MsgSend',
            value: {
              fromAddress: account.address,
              toAddress: recipient,
              amount: [
                {
                  denom: token.denom,
                  amount: AMOUNT_IN_MICRO.toString(),
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

          console.log(`   Отправка ${AMOUNT} ${token.name}...`);
          const result = await signingClient.signAndBroadcast(
            account.address,
            [sendMsg],
            fee
          );

          if (result.code === 0) {
            console.log(`   ✅ ${token.name} отправлено успешно! Hash: ${result.transactionHash}`);
          } else {
            console.error(`   ❌ Ошибка отправки ${token.name}: ${result.rawLog}`);
          }

          // Небольшая задержка между транзакциями
          await new Promise(resolve => setTimeout(resolve, 1000));
        } catch (error) {
          console.error(`   ❌ Ошибка при отправке ${token.name}:`, error.message);
        }
      }
    }

    console.log('\n✅ Все транзакции обработаны!');
    console.log('\n📊 Проверка балансов получателей...');

    // Проверяем балансы получателей
    for (const recipient of RECIPIENTS) {
      try {
        const balances = await client.getAllBalances(recipient);
        console.log(`\n💰 ${recipient}:`);
        if (balances.length === 0) {
          console.log('   Баланс: 0 (аккаунт еще не создан в блокчейне)');
        } else {
          balances.forEach(b => {
            const amount = parseInt(b.amount) / 1_000_000;
            console.log(`   ${amount} ${b.denom.replace('u', '').toUpperCase()}`);
          });
        }
      } catch (error) {
        console.error(`   ❌ Ошибка проверки баланса: ${error.message}`);
      }
    }

    // Закрываем соединения
    await signingClient.disconnect();
    await client.disconnect();

    console.log('\n✅ Готово!');
  } catch (error) {
    console.error('\n❌ Критическая ошибка:', error);
    process.exit(1);
  }
}

// Запускаем скрипт
sendTokens();

