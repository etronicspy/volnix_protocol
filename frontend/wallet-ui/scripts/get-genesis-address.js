#!/usr/bin/env node
/**
 * Скрипт для получения адреса из тестовой мнемоники
 * Использование: node scripts/get-genesis-address.js
 */

const { DirectSecp256k1HdWallet } = require('@cosmjs/proto-signing');

const TEST_MNEMONIC = 'abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about';
const PREFIX = 'volnix';

async function getAddress() {
  try {
    console.log('🔑 Генерация адреса из тестовой мнемоники...\n');
    console.log('📝 Мнемоника:', TEST_MNEMONIC);
    console.log('');
    
    const wallet = await DirectSecp256k1HdWallet.fromMnemonic(TEST_MNEMONIC, {
      prefix: PREFIX,
    });

    const [account] = await wallet.getAccounts();
    
    console.log('✅ Адрес genesis аккаунта:');
    console.log(account.address);
    console.log('');
    console.log('💡 Используйте эту мнемонику для подключения кошелька:');
    console.log(TEST_MNEMONIC);
    console.log('');
    console.log('⚠️  ВАЖНО: Этот адрес должен совпадать с genesis адресом в коде узла!');
    console.log('   Если не совпадает, нужно обновить genesis адрес в cmd/volnixd-standalone/main.go');
    
  } catch (error) {
    console.error('❌ Ошибка:', error.message);
    process.exit(1);
  }
}

getAddress();
