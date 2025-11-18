// Утилита для проверки сохраненных кошельков
// Скопируйте и вставьте этот код в консоль браузера на странице кошелька

(function() {
  console.log('🔍 Проверка сохраненных кошельков...\n');
  
  const WALLET_LIST_KEY = 'volnix_wallets_list';
  const WALLET_PREFIX = 'wallet_';
  const WALLET_ADDRESS_SUFFIX = '_address';
  
  try {
    // Получаем список имен кошельков
    const walletListJson = localStorage.getItem(WALLET_LIST_KEY);
    
    if (!walletListJson) {
      console.log('❌ Кошельки не найдены в localStorage');
      console.log('💡 Создайте новый кошелек через интерфейс');
      return;
    }

    const walletNames = JSON.parse(walletListJson);
    console.log(`✅ Найдено кошельков: ${walletNames.length}\n`);
    
    if (walletNames.length === 0) {
      console.log('📝 Список кошельков пуст');
      return;
    }
    
    // Выводим информацию о каждом кошельке
    const wallets = [];
    
    walletNames.forEach((name, index) => {
      const mnemonic = localStorage.getItem(`${WALLET_PREFIX}${name}`);
      const address = localStorage.getItem(`${WALLET_PREFIX}${name}${WALLET_ADDRESS_SUFFIX}`);
      const createdAt = localStorage.getItem(`${WALLET_PREFIX}${name}_created`) || 'Unknown';
      
      if (mnemonic && address) {
        const walletInfo = {
          '#': index + 1,
          'Имя': name,
          'Адрес': address,
          'Мнемоника (первые 30 символов)': mnemonic.substring(0, 30) + '...',
          'Дата создания': new Date(createdAt).toLocaleString('ru-RU')
        };
        
        wallets.push(walletInfo);
        
        console.log(`📛 Кошелек #${index + 1}: ${name}`);
        console.log(`   Адрес: ${address}`);
        console.log(`   Мнемоника: ${mnemonic.substring(0, 30)}...`);
        console.log(`   Создан: ${new Date(createdAt).toLocaleString('ru-RU')}`);
        console.log('');
      } else {
        console.warn(`⚠️  Кошелек "${name}" найден в списке, но данные неполные`);
      }
    });
    
    // Выводим таблицу
    if (wallets.length > 0) {
      console.table(wallets);
    }
    
    // Показываем все ключи связанные с кошельками
    console.log('\n🔑 Все ключи localStorage связанные с кошельками:');
    const allKeys = Object.keys(localStorage);
    const walletKeys = allKeys.filter(key => 
      key.startsWith(WALLET_PREFIX) || key === WALLET_LIST_KEY
    );
    
    walletKeys.forEach(key => {
      const value = localStorage.getItem(key);
      if (key.includes('_address') || key.includes('_created') || key === WALLET_LIST_KEY) {
        console.log(`  ${key}: ${value}`);
      } else {
        // Для мнемоник показываем только начало
        console.log(`  ${key}: ${value ? value.substring(0, 30) + '...' : 'null'}`);
      }
    });
    
  } catch (error) {
    console.error('❌ Ошибка при проверке кошельков:', error);
  }
})();

