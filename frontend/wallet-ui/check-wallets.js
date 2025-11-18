// Утилита для проверки сохраненных кошельков
// Запустите в консоли браузера на странице кошелька

function checkWallets() {
  const walletListKey = 'volnix_wallets_list';
  const walletPrefix = 'wallet_';
  const walletAddressPrefix = 'wallet_';
  const walletAddressSuffix = '_address';
  
  try {
    const walletListJson = localStorage.getItem(walletListKey);
    console.log('📋 Wallet List Key:', walletListJson);
    
    if (!walletListJson) {
      console.log('❌ No wallets found in localStorage');
      return [];
    }

    const walletNames = JSON.parse(walletListJson);
    console.log('📝 Wallet Names:', walletNames);
    
    const wallets = [];
    
    for (const name of walletNames) {
      const mnemonic = localStorage.getItem(`${walletPrefix}${name}`);
      const address = localStorage.getItem(`${walletAddressPrefix}${name}${walletAddressSuffix}`);
      const createdAt = localStorage.getItem(`${walletPrefix}${name}_created`) || 'Unknown';
      
      if (mnemonic && address) {
        wallets.push({
          name,
          address,
          mnemonic: mnemonic.substring(0, 20) + '...', // Показываем только начало для безопасности
          createdAt
        });
      }
    }
    
    console.log('✅ Found wallets:', wallets.length);
    console.table(wallets);
    
    return wallets;
  } catch (error) {
    console.error('❌ Error checking wallets:', error);
    return [];
  }
}

// Проверяем все ключи в localStorage связанные с кошельками
function checkAllWalletKeys() {
  console.log('🔍 All localStorage keys related to wallets:');
  const allKeys = Object.keys(localStorage);
  const walletKeys = allKeys.filter(key => key.startsWith('wallet_') || key === 'volnix_wallets_list');
  walletKeys.forEach(key => {
    const value = localStorage.getItem(key);
    if (key.includes('mnemonic') || key.includes('wallet_') && !key.includes('_address') && !key.includes('_created')) {
      console.log(`  ${key}: ${value ? value.substring(0, 20) + '...' : 'null'}`);
    } else {
      console.log(`  ${key}: ${value}`);
    }
  });
}

console.log('💡 Run checkWallets() to see all saved wallets');
console.log('💡 Run checkAllWalletKeys() to see all wallet-related keys in localStorage');
