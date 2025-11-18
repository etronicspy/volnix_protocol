#!/usr/bin/env node
/**
 * Скрипт для получения адреса из мнемоники
 * Использование: node scripts/get-address-from-mnemonic.js '<mnemonic>'
 */

const { DirectSecp256k1HdWallet } = require('@cosmjs/proto-signing');

const MNEMONIC = process.argv[2] || 'abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about';
const PREFIX = 'volnix';

async function getAddress() {
  try {
    console.log('🔑 Генерация адреса из мнемоники...\n');
    
    const wallet = await DirectSecp256k1HdWallet.fromMnemonic(MNEMONIC, {
      prefix: PREFIX,
    });

    const [account] = await wallet.getAccounts();
    
    console.log('✅ Адрес:');
    console.log(account.address);
    console.log('');
    console.log('📋 Для создания genesis аккаунта используйте:');
    console.log(`   ./scripts/create-genesis-account.sh '${MNEMONIC}' 'testnet/node0/.volnix/config/genesis.json' '${account.address}'`);
    
  } catch (error) {
    console.error('❌ Ошибка:', error.message);
    process.exit(1);
  }
}

getAddress();

