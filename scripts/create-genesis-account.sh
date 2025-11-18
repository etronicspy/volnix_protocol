#!/bin/bash
# Скрипт для создания genesis аккаунта с балансом
# Использование: ./scripts/create-genesis-account.sh <mnemonic> <address>

set -e

MNEMONIC="${1:-abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about}"
GENESIS_FILE="${2:-testnet/node0/.volnix/config/genesis.json}"

echo "🔑 Создание genesis аккаунта..."
echo "📝 Мнемоника: ${MNEMONIC:0:30}..."
echo "📄 Genesis файл: $GENESIS_FILE"

# Проверяем существование genesis файла
if [ ! -f "$GENESIS_FILE" ]; then
    echo "❌ Genesis файл не найден: $GENESIS_FILE"
    echo "💡 Убедитесь, что узел запущен и genesis файл создан"
    exit 1
fi

# Генерируем адрес из мнемоники используя Python (если доступен)
# Или используем готовый адрес
if [ -z "$3" ]; then
    echo "⚠️  Адрес не указан. Используйте:"
    echo "   ./scripts/create-genesis-account.sh '<mnemonic>' '<genesis_file>' '<address>'"
    echo ""
    echo "💡 Для получения адреса из мнемоники используйте CosmJS или CLI"
    exit 1
fi

ADDRESS="$3"
echo "📍 Адрес: $ADDRESS"

# Создаем резервную копию
cp "$GENESIS_FILE" "${GENESIS_FILE}.backup"
echo "✅ Резервная копия создана: ${GENESIS_FILE}.backup"

# Используем Python для модификации JSON
python3 << PYTHON_SCRIPT
import json
import sys

genesis_file = "$GENESIS_FILE"
address = "$ADDRESS"

# Балансы для genesis аккаунта (1000 каждого токена для отправки)
balances = [
    {"denom": "uwrt", "amount": "1000000000"},  # 1000 WRT
    {"denom": "ulzn", "amount": "1000000000"},  # 1000 LZN
    {"denom": "uant", "amount": "1000000000"}   # 1000 ANT
]

try:
    # Читаем genesis файл
    with open(genesis_file, 'r') as f:
        genesis = json.load(f)
    
    # Инициализируем app_state если его нет
    if 'app_state' not in genesis:
        genesis['app_state'] = {}
    
    # Инициализируем bank модуль
    if 'bank' not in genesis['app_state']:
        genesis['app_state']['bank'] = {
            "params": {
                "send_enabled": [],
                "default_send_enabled": True
            },
            "balances": [],
            "supply": []
        }
    
    # Добавляем баланс для адреса
    balance_entry = {
        "address": address,
        "coins": balances
    }
    
    # Проверяем, нет ли уже баланса для этого адреса
    existing_balance = None
    for i, bal in enumerate(genesis['app_state']['bank']['balances']):
        if bal.get('address') == address:
            existing_balance = i
            break
    
    if existing_balance is not None:
        # Обновляем существующий баланс
        genesis['app_state']['bank']['balances'][existing_balance] = balance_entry
        print(f"✅ Обновлен баланс для адреса: {address}")
    else:
        # Добавляем новый баланс
        genesis['app_state']['bank']['balances'].append(balance_entry)
        print(f"✅ Добавлен баланс для адреса: {address}")
    
    # Обновляем supply
    for coin in balances:
        # Ищем существующий supply для этого денома
        supply_found = False
        for i, sup in enumerate(genesis['app_state']['bank']['supply']):
            if sup.get('denom') == coin['denom']:
                # Обновляем supply
                current_amount = int(sup.get('amount', '0'))
                new_amount = current_amount + int(coin['amount'])
                genesis['app_state']['bank']['supply'][i]['amount'] = str(new_amount)
                supply_found = True
                break
        
        if not supply_found:
            # Добавляем новый supply
            genesis['app_state']['bank']['supply'].append({
                "denom": coin['denom'],
                "amount": coin['amount']
            })
    
    # Сохраняем обновленный genesis файл
    with open(genesis_file, 'w') as f:
        json.dump(genesis, f, indent=2)
    
    print("✅ Genesis файл обновлен!")
    print(f"💰 Балансы для {address}:")
    for coin in balances:
        amount = int(coin['amount']) / 1_000_000
        denom = coin['denom'].replace('u', '').upper()
        print(f"   {amount} {denom}")
    
except Exception as e:
    print(f"❌ Ошибка: {e}")
    sys.exit(1)
PYTHON_SCRIPT

echo ""
echo "✅ Genesis аккаунт создан!"
echo "⚠️  ВАЖНО: Перезапустите узел для применения изменений"
echo ""

