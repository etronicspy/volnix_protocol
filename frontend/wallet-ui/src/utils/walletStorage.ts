// Утилита для работы с сохраненными кошельками в localStorage

export interface SavedWallet {
  name: string;
  address: string;
  mnemonic: string;
  createdAt: string;
}

const WALLET_PREFIX = 'wallet_';
const WALLET_ADDRESS_PREFIX = 'wallet_';
const WALLET_ADDRESS_SUFFIX = '_address';
const WALLET_LIST_KEY = 'volnix_wallets_list';

/**
 * Получить список всех сохраненных кошельков
 */
export function getAllWallets(): SavedWallet[] {
  try {
    const walletListJson = localStorage.getItem(WALLET_LIST_KEY);
    if (!walletListJson) {
      return [];
    }

    const walletNames: string[] = JSON.parse(walletListJson);
    const wallets: SavedWallet[] = [];

    for (const name of walletNames) {
      const mnemonic = localStorage.getItem(`${WALLET_PREFIX}${name}`);
      const address = localStorage.getItem(`${WALLET_ADDRESS_PREFIX}${name}${WALLET_ADDRESS_SUFFIX}`);
      const createdAt = localStorage.getItem(`${WALLET_PREFIX}${name}_created`) || new Date().toISOString();

      if (mnemonic && address) {
        wallets.push({
          name,
          address,
          mnemonic,
          createdAt,
        });
      }
    }

    return wallets;
  } catch (error) {
    console.error('Error loading wallets:', error);
    return [];
  }
}

/**
 * Сохранить кошелек
 */
export function saveWallet(name: string, address: string, mnemonic: string): void {
  try {
    // Сохраняем данные кошелька
    localStorage.setItem(`${WALLET_PREFIX}${name}`, mnemonic);
    localStorage.setItem(`${WALLET_ADDRESS_PREFIX}${name}${WALLET_ADDRESS_SUFFIX}`, address);
    localStorage.setItem(`${WALLET_PREFIX}${name}_created`, new Date().toISOString());

    // Обновляем список кошельков
    const walletListJson = localStorage.getItem(WALLET_LIST_KEY);
    let walletNames: string[] = [];

    if (walletListJson) {
      try {
        walletNames = JSON.parse(walletListJson);
      } catch (e) {
        // Если список поврежден, создаем новый
        walletNames = [];
      }
    }

    // Добавляем имя кошелька, если его еще нет
    if (!walletNames.includes(name)) {
      walletNames.push(name);
      localStorage.setItem(WALLET_LIST_KEY, JSON.stringify(walletNames));
    }
  } catch (error) {
    console.error('Error saving wallet:', error);
    throw new Error('Failed to save wallet');
  }
}

/**
 * Удалить кошелек
 */
export function deleteWallet(name: string): void {
  try {
    // Удаляем данные кошелька
    localStorage.removeItem(`${WALLET_PREFIX}${name}`);
    localStorage.removeItem(`${WALLET_ADDRESS_PREFIX}${name}${WALLET_ADDRESS_SUFFIX}`);
    localStorage.removeItem(`${WALLET_PREFIX}${name}_created`);

    // Обновляем список кошельков
    const walletListJson = localStorage.getItem(WALLET_LIST_KEY);
    if (walletListJson) {
      try {
        const walletNames: string[] = JSON.parse(walletListJson);
        const updatedNames = walletNames.filter(n => n !== name);
        localStorage.setItem(WALLET_LIST_KEY, JSON.stringify(updatedNames));
      } catch (e) {
        // Если список поврежден, очищаем его
        localStorage.removeItem(WALLET_LIST_KEY);
      }
    }
  } catch (error) {
    console.error('Error deleting wallet:', error);
    throw new Error('Failed to delete wallet');
  }
}

/**
 * Проверить, существует ли кошелек с таким именем
 */
export function walletExists(name: string): boolean {
  const mnemonic = localStorage.getItem(`${WALLET_PREFIX}${name}`);
  return mnemonic !== null;
}

/**
 * Получить кошелек по имени
 */
export function getWallet(name: string): SavedWallet | null {
  try {
    const mnemonic = localStorage.getItem(`${WALLET_PREFIX}${name}`);
    const address = localStorage.getItem(`${WALLET_ADDRESS_PREFIX}${name}${WALLET_ADDRESS_SUFFIX}`);
    const createdAt = localStorage.getItem(`${WALLET_PREFIX}${name}_created`) || new Date().toISOString();

    if (mnemonic && address) {
      return {
        name,
        address,
        mnemonic,
        createdAt,
      };
    }

    return null;
  } catch (error) {
    console.error('Error getting wallet:', error);
    return null;
  }
}

