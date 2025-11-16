# Быстрый запуск для Mac

## Проблема: "Нет соединения" или "must provide a non-empty value"

### Решение 1: Полный перезапуск

Откройте терминал и выполните:

```bash
cd /Users/etronicspy/volnix-protocol/helvetia_protocol

# Остановите все процессы
pkill -9 volnixd-standalone
pkill -9 node

# Запустите заново
./scripts/start-local-network.sh
```

Скрипт автоматически:
- Остановит старые процессы
- Очистит базу данных
- Запустит RPC узел на порту 26657
- Запустит Wallet UI на порту 3000

### Решение 2: Ручной запуск

Если автоматический скрипт не работает:

#### Терминал 1: RPC узел
```bash
cd /Users/etronicspy/volnix-protocol/helvetia_protocol
cd testnet/node0
rm -rf .volnix/data/*.db
VOLNIX_HOME=".volnix" ../../build/volnixd-standalone start
```

#### Терминал 2: Фронтенд
```bash
cd /Users/etronicspy/volnix-protocol/helvetia_protocol/frontend/wallet-ui
rm -rf node_modules/.cache
npm start
```

### Проверка

1. RPC узел: `curl http://localhost:26657/status`
   - Должен вернуть JSON с chain_id: "volnix-standalone"

2. Фронтенд: откройте `http://localhost:3000` в браузере

3. Консоль браузера (Cmd+Option+I):
   ```javascript
   fetch('http://localhost:26657/status')
     .then(r=>r.json())
     .then(d=>console.log('Chain ID:', d.result.node_info.network))
   ```

### Если ошибка сохраняется

Пришлите:
1. Вывод команды: `curl http://localhost:26657/status`
2. Логи из консоли браузера (Cmd+Option+I → Console)
3. Что показывает: `lsof -i :26657` и `lsof -i :3000`

