# Руководство по инструментам разработчика Volnix Protocol

## Обзор

Volnix Protocol предоставляет полный набор инструментов для разработчиков, включающий CLI утилиты, SDK, библиотеки для интеграции, инструменты тестирования и отладки. Данное руководство описывает все доступные инструменты и методы их использования.

## 1. CLI инструменты и команды volnixd

### 1.1 Основные команды

#### Инициализация и управление узлом

```bash
# Инициализация нового узла
volnixd init [moniker] --chain-id volnix-mainnet

# Запуск узла
volnixd start

# Проверка статуса узла
volnixd status

# Показать версию
volnixd version
```

#### Управление ключами

```bash
# Создание нового ключа
volnixd keys add [key-name]

# Список всех ключей
volnixd keys list

# Показать адрес ключа
volnixd keys show [key-name] --address

# Экспорт ключа
volnixd keys export [key-name]

# Импорт ключа
volnixd keys import [key-name] [keyfile]
```

#### Транзакции

```bash
# Отправка токенов
volnixd tx bank send [from-key] [to-address] [amount] --chain-id volnix-mainnet

# Identity модуль - создание аккаунта
volnixd tx ident create-account [identity-type] [identity-hash] --from [key-name]

# Lizenz модуль - активация лицензии
volnixd tx lizenz activate-lizenz [amount] [identity-hash] --from [key-name]

# Anteil модуль - создание ордера
volnixd tx anteil create-order [order-type] [order-side] [ant-amount] [price] --from [key-name]

# Consensus модуль - регистрация валидатора
volnixd tx consensus register-validator [validator-info] --from [key-name]
```

#### Запросы

```bash
# Баланс аккаунта
volnixd query bank balances [address]

# Identity запросы
volnixd query ident account [address]
volnixd query ident all-accounts

# Lizenz запросы
volnixd query lizenz activated-lizenz [validator]
volnixd query lizenz all-activated-lizenz
volnixd query lizenz moa-status [validator]

# Anteil запросы
volnixd query anteil order [order-id]
volnixd query anteil all-orders
volnixd query anteil user-position [owner]
volnixd query anteil auction [auction-id]

# Consensus запросы
volnixd query consensus validator [validator-address]
volnixd query consensus all-validators
volnixd query consensus consensus-state
```

### 1.2 Конфигурация

#### Файлы конфигурации

```bash
# Основная конфигурация
~/.volnix/config/config.toml

# Конфигурация приложения
~/.volnix/config/app.toml

# Genesis файл
~/.volnix/config/genesis.json

# Ключи валидатора
~/.volnix/config/priv_validator_key.json
```

#### Настройка сети

```toml
# config.toml
[p2p]
laddr = "tcp://0.0.0.0:26656"
persistent_peers = "node1@ip1:26656,node2@ip2:26656"

[rpc]
laddr = "tcp://127.0.0.1:26657"
cors_allowed_origins = ["*"]

[consensus]
timeout_propose = "3s"
timeout_prevote = "1s"
timeout_precommit = "1s"
timeout_commit = "5s"
```

### 1.3 Standalone версия

```bash
# Инициализация standalone узла
volnixd-standalone init [moniker]

# Запуск standalone узла
volnixd-standalone start

# Проверка статуса
volnixd-standalone status

# Версия standalone
volnixd-standalone version
```

## 2. SDK и библиотеки для разработчиков

### 2.1 Go SDK

#### Установка

```bash
go mod init your-project
go get github.com/volnix-protocol/volnix-protocol
```

#### Основные компоненты

```go
package main

import (
    "github.com/volnix-protocol/volnix-protocol/app"
    "github.com/volnix-protocol/volnix-protocol/x/ident"
    "github.com/volnix-protocol/volnix-protocol/x/lizenz"
    "github.com/volnix-protocol/volnix-protocol/x/anteil"
    "github.com/volnix-protocol/volnix-protocol/x/consensus"
)

// Создание приложения
func NewApp() *app.VolnixApp {
    encodingConfig := app.MakeEncodingConfig()
    return app.NewVolnixApp(logger, db, nil, encodingConfig)
}
```

#### Работа с модулями

```go
// Identity модуль
import identtypes "github.com/volnix-protocol/volnix-protocol/x/ident/types"

// Создание аккаунта
account := identtypes.NewAccount(
    "citizen",
    "identity_hash_here",
    "volnix1address...",
)

// Lizenz модуль
import lizenztypes "github.com/volnix-protocol/volnix-protocol/x/lizenz/types"

// Создание активированной лицензии
lizenz := lizenztypes.NewActivatedLizenz(
    "volnix1validator...",
    "1000000ulzn",
    "identity_hash",
)

// Anteil модуль
import anteiltypes "github.com/volnix-protocol/volnix-protocol/x/anteil/types"

// Создание ордера
order := anteiltypes.NewOrder(
    "volnix1owner...",
    anteiltypes.OrderType_ORDER_TYPE_LIMIT,
    anteiltypes.OrderSide_ORDER_SIDE_BUY,
    "100",
    "1.5",
    "identity_hash",
)
```

### 2.2 JavaScript/TypeScript SDK

#### Установка

```bash
npm install @cosmjs/stargate @cosmjs/proto-signing
```

#### Подключение к сети

```typescript
import { StargateClient } from "@cosmjs/stargate";

// Подключение к RPC
const client = await StargateClient.connect("http://localhost:26657");

// Получение баланса
const balance = await client.getAllBalances("volnix1address...");

// Получение информации о блоке
const block = await client.getBlock(12345);
```

#### Отправка транзакций

```typescript
import { SigningStargateClient } from "@cosmjs/stargate";
import { DirectSecp256k1HdWallet } from "@cosmjs/proto-signing";

// Создание кошелька
const wallet = await DirectSecp256k1HdWallet.fromMnemonic(mnemonic);
const [firstAccount] = await wallet.getAccounts();

// Создание signing client
const client = await SigningStargateClient.connectWithSigner(
  "http://localhost:26657",
  wallet
);

// Отправка токенов
const result = await client.sendTokens(
  firstAccount.address,
  "volnix1recipient...",
  [{ denom: "uvx", amount: "1000000" }],
  "auto"
);
```

### 2.3 Python SDK

#### Установка

```bash
pip install cosmpy
```

#### Базовое использование

```python
from cosmpy.aerial.client import LedgerClient
from cosmpy.aerial.wallet import LocalWallet

# Подключение к сети
client = LedgerClient("http://localhost:26657")

# Создание кошелька
wallet = LocalWallet.from_mnemonic(mnemonic)

# Получение баланса
balance = client.query_bank_balance(wallet.address(), "uvx")

# Отправка транзакции
tx = client.send_tokens(
    wallet.address(),
    "volnix1recipient...",
    1000000,
    "uvx"
)
```

## 3. Руководство по тестированию и отладке

### 3.1 Unit тестирование

#### Структура тестов

```
tests/
├── integration_test.go      # Интеграционные тесты
├── security_test.go         # Тесты безопасности
├── benchmark_test.go        # Бенчмарки
└── end_to_end_test.go       # E2E тесты

x/
├── ident/keeper/
│   ├── keeper_test.go       # Unit тесты keeper
│   ├── msg_server_test.go   # Тесты msg server
│   └── query_server_test.go # Тесты query server
```

#### Запуск тестов

```bash
# Все тесты
make test

# Unit тесты
make test-unit

# Интеграционные тесты
make test-integration

# Тесты с покрытием
make test-coverage

# Бенчмарки
go test -bench=. ./tests/...
```

#### Пример unit теста

```go
func TestCreateAccount(t *testing.T) {
    app := setupTestApp()
    ctx := app.NewContext(false)
    
    // Создание аккаунта
    msg := &identtypes.MsgCreateAccount{
        Creator:      "volnix1creator...",
        IdentityType: "citizen",
        IdentityHash: "test_hash",
    }
    
    _, err := app.IdentKeeper.CreateAccount(ctx, msg)
    require.NoError(t, err)
    
    // Проверка создания
    account, found := app.IdentKeeper.GetAccount(ctx, msg.Creator)
    require.True(t, found)
    require.Equal(t, msg.IdentityType, account.IdentityType)
}
```

### 3.2 Интеграционное тестирование

#### Полный экономический цикл

```go
func TestFullEconomicCycle(t *testing.T) {
    // 1. Создание аккаунтов
    citizenAddr := createCitizen(t, app, ctx)
    validatorAddr := createValidator(t, app, ctx)
    
    // 2. Активация лицензии
    activateLizenz(t, app, ctx, validatorAddr)
    
    // 3. Создание ANT позиции
    createANTPosition(t, app, ctx, citizenAddr)
    
    // 4. Торговля
    createAndExecuteOrder(t, app, ctx, citizenAddr)
    
    // 5. Проверка состояния
    verifyFinalState(t, app, ctx)
}
```

### 3.3 Отладка

#### Логирование

```bash
# Включение debug логов
volnixd start --log_level debug

# Логирование в файл
volnixd start --log_level debug > volnix.log 2>&1
```

#### Профилирование

```go
import _ "net/http/pprof"
import "net/http"

// Запуск pprof сервера
go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

```bash
# CPU профиль
go tool pprof http://localhost:6060/debug/pprof/profile

# Memory профиль
go tool pprof http://localhost:6060/debug/pprof/heap
```

## 4. Примеры интеграции с внешними системами

### 4.1 REST API интеграция

#### Получение данных через REST

```bash
# Информация о блоке
curl http://localhost:1317/cosmos/base/tendermint/v1beta1/blocks/latest

# Баланс аккаунта
curl http://localhost:1317/cosmos/bank/v1beta1/balances/volnix1address...

# Identity запросы
curl http://localhost:1317/volnix/ident/account/volnix1address...

# Lizenz запросы
curl http://localhost:1317/volnix/lizenz/activated_lizenz/volnix1validator...

# Anteil запросы
curl http://localhost:1317/volnix/anteil/order/order_id_here
```

#### JavaScript интеграция

```javascript
// Получение баланса
async function getBalance(address) {
    const response = await fetch(
        `http://localhost:1317/cosmos/bank/v1beta1/balances/${address}`
    );
    const data = await response.json();
    return data.balances;
}

// Получение информации об аккаунте
async function getAccount(address) {
    const response = await fetch(
        `http://localhost:1317/volnix/ident/account/${address}`
    );
    const data = await response.json();
    return data.account;
}
```

### 4.2 WebSocket интеграция

#### Подписка на события

```javascript
const ws = new WebSocket('ws://localhost:26657/websocket');

// Подписка на новые блоки
ws.send(JSON.stringify({
    jsonrpc: "2.0",
    method: "subscribe",
    id: 1,
    params: {
        query: "tm.event='NewBlock'"
    }
}));

// Подписка на транзакции
ws.send(JSON.stringify({
    jsonrpc: "2.0",
    method: "subscribe",
    id: 2,
    params: {
        query: "tm.event='Tx'"
    }
}));
```

### 4.3 Мониторинг и метрики

#### Prometheus метрики

```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'volnix'
    static_configs:
      - targets: ['localhost:26660']
```

#### Grafana дашборд

```json
{
  "dashboard": {
    "title": "Volnix Protocol Metrics",
    "panels": [
      {
        "title": "Block Height",
        "type": "stat",
        "targets": [
          {
            "expr": "tendermint_consensus_height"
          }
        ]
      },
      {
        "title": "Transaction Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(tendermint_consensus_total_txs[5m])"
          }
        ]
      }
    ]
  }
}
```

## 5. Инструменты для создания dApps на Volnix

### 5.1 Шаблон dApp

#### Структура проекта

```
volnix-dapp/
├── contracts/              # Smart contracts (будущее)
├── frontend/              # React frontend
│   ├── src/
│   │   ├── components/    # UI компоненты
│   │   ├── hooks/         # React hooks для Volnix
│   │   ├── services/      # API сервисы
│   │   └── types/         # TypeScript типы
│   └── package.json
├── backend/               # Node.js backend
│   ├── src/
│   │   ├── routes/        # API маршруты
│   │   ├── services/      # Бизнес логика
│   │   └── utils/         # Утилиты
│   └── package.json
└── docker-compose.yml     # Развертывание
```

#### React hooks для Volnix

```typescript
// useVolnixClient.ts
import { useState, useEffect } from 'react';
import { StargateClient } from '@cosmjs/stargate';

export function useVolnixClient() {
    const [client, setClient] = useState<StargateClient | null>(null);
    
    useEffect(() => {
        StargateClient.connect('http://localhost:26657')
            .then(setClient);
    }, []);
    
    return client;
}

// useBalance.ts
export function useBalance(address: string) {
    const [balance, setBalance] = useState([]);
    const client = useVolnixClient();
    
    useEffect(() => {
        if (client && address) {
            client.getAllBalances(address)
                .then(setBalance);
        }
    }, [client, address]);
    
    return balance;
}
```

### 5.2 Компоненты UI

#### Wallet Connect компонент

```typescript
// WalletConnect.tsx
import React, { useState } from 'react';
import { DirectSecp256k1HdWallet } from '@cosmjs/proto-signing';

export function WalletConnect() {
    const [wallet, setWallet] = useState(null);
    const [address, setAddress] = useState('');
    
    const connectWallet = async () => {
        const wallet = await DirectSecp256k1HdWallet.fromMnemonic(mnemonic);
        const [account] = await wallet.getAccounts();
        
        setWallet(wallet);
        setAddress(account.address);
    };
    
    return (
        <div>
            {!wallet ? (
                <button onClick={connectWallet}>
                    Connect Wallet
                </button>
            ) : (
                <div>
                    Connected: {address}
                </div>
            )}
        </div>
    );
}
```

#### Balance Display компонент

```typescript
// BalanceDisplay.tsx
import React from 'react';
import { useBalance } from '../hooks/useBalance';

interface Props {
    address: string;
}

export function BalanceDisplay({ address }: Props) {
    const balance = useBalance(address);
    
    return (
        <div>
            <h3>Balances</h3>
            {balance.map(coin => (
                <div key={coin.denom}>
                    {coin.amount} {coin.denom}
                </div>
            ))}
        </div>
    );
}
```

### 5.3 Backend сервисы

#### API для работы с Volnix

```typescript
// volnix.service.ts
import { StargateClient } from '@cosmjs/stargate';

export class VolnixService {
    private client: StargateClient;
    
    constructor() {
        this.connect();
    }
    
    private async connect() {
        this.client = await StargateClient.connect('http://localhost:26657');
    }
    
    async getAccount(address: string) {
        return await this.client.getAccount(address);
    }
    
    async getBalance(address: string) {
        return await this.client.getAllBalances(address);
    }
    
    async getBlock(height?: number) {
        return await this.client.getBlock(height);
    }
}
```

#### Express API маршруты

```typescript
// routes/api.ts
import express from 'express';
import { VolnixService } from '../services/volnix.service';

const router = express.Router();
const volnixService = new VolnixService();

router.get('/account/:address', async (req, res) => {
    try {
        const account = await volnixService.getAccount(req.params.address);
        res.json(account);
    } catch (error) {
        res.status(500).json({ error: error.message });
    }
});

router.get('/balance/:address', async (req, res) => {
    try {
        const balance = await volnixService.getBalance(req.params.address);
        res.json(balance);
    } catch (error) {
        res.status(500).json({ error: error.message });
    }
});

export default router;
```

### 5.4 Развертывание dApp

#### Docker Compose

```yaml
# docker-compose.yml
version: '3.8'

services:
  volnix-node:
    image: volnix/node:latest
    ports:
      - "26657:26657"
      - "1317:1317"
    volumes:
      - volnix-data:/root/.volnix
    
  frontend:
    build: ./frontend
    ports:
      - "3000:3000"
    depends_on:
      - backend
    
  backend:
    build: ./backend
    ports:
      - "8000:8000"
    depends_on:
      - volnix-node
    environment:
      - VOLNIX_RPC_URL=http://volnix-node:26657

volumes:
  volnix-data:
```

#### Makefile для разработки

```makefile
# Makefile
.PHONY: dev build deploy

dev:
	docker-compose up -d volnix-node
	cd backend && npm run dev &
	cd frontend && npm start

build:
	docker-compose build

deploy:
	docker-compose up -d

test:
	cd backend && npm test
	cd frontend && npm test

clean:
	docker-compose down -v
```

## 6. Инструменты командной строки

### 6.1 Скрипты автоматизации

#### Тестирование функциональности

```bash
# scripts/test_current_functionality.sh
#!/bin/bash

echo "🧪 Testing Volnix Protocol Functionality"

# Проверка prerequisites
check_prerequisites() {
    command -v go >/dev/null 2>&1 || { echo "Go not installed"; exit 1; }
    [ -f "./volnixd" ] || { echo "volnixd binary not found"; exit 1; }
}

# Инициализация узла
init_node() {
    echo "🚀 Initializing node..."
    ./volnixd init testnode
}

# Запуск узла
start_node() {
    echo "📡 Starting node..."
    ./volnixd start > /tmp/volnixd.log 2>&1 &
    NODE_PID=$!
}

# Тестирование CLI команд
test_cli() {
    echo "⌨️ Testing CLI commands..."
    ./volnixd keys add testkey
    ./volnixd status
}

check_prerequisites
init_node
start_node
test_cli
```

#### Развертывание тестовой сети

```bash
# scripts/setup_testnet.sh
#!/bin/bash

CHAIN_ID="volnix-testnet"
NODES=4

# Создание директорий для узлов
for i in $(seq 1 $NODES); do
    mkdir -p testnet/node$i
    volnixd init node$i --chain-id $CHAIN_ID --home testnet/node$i
done

# Генерация genesis файла
volnixd collect-gentxs --home testnet/node1
cp testnet/node1/config/genesis.json testnet/

# Копирование genesis во все узлы
for i in $(seq 2 $NODES); do
    cp testnet/genesis.json testnet/node$i/config/
done

echo "✅ Testnet setup completed"
```

### 6.2 Утилиты разработчика

#### Генератор тестовых данных

```bash
# scripts/generate_test_data.sh
#!/bin/bash

# Создание тестовых аккаунтов
create_test_accounts() {
    for i in {1..10}; do
        volnixd keys add test-account-$i
    done
}

# Создание тестовых транзакций
create_test_transactions() {
    for i in {1..5}; do
        volnixd tx bank send test-account-1 \
            $(volnixd keys show test-account-2 -a) \
            1000uvx --chain-id volnix-testnet
    done
}

create_test_accounts
create_test_transactions
```

#### Мониторинг производительности

```bash
# scripts/monitor_performance.sh
#!/bin/bash

# Мониторинг использования ресурсов
monitor_resources() {
    while true; do
        echo "$(date): CPU: $(top -bn1 | grep volnixd | awk '{print $9}')%"
        echo "$(date): Memory: $(ps -o rss= -p $(pgrep volnixd) | awk '{print $1/1024}') MB"
        sleep 30
    done
}

# Мониторинг блоков
monitor_blocks() {
    while true; do
        HEIGHT=$(curl -s http://localhost:26657/status | jq -r '.result.sync_info.latest_block_height')
        echo "$(date): Block height: $HEIGHT"
        sleep 10
    done
}

monitor_resources &
monitor_blocks &
wait
```

## 7. Отладка и диагностика

### 7.1 Логирование

#### Конфигурация логов

```toml
# config.toml
[log]
level = "debug"
format = "json"
```

#### Анализ логов

```bash
# Фильтрация по модулям
grep "module=ident" volnix.log

# Поиск ошибок
grep "ERROR" volnix.log

# Анализ производительности
grep "block_time" volnix.log | awk '{print $NF}' | sort -n
```

### 7.2 Профилирование

#### CPU профилирование

```go
// main.go
import _ "net/http/pprof"

func main() {
    go func() {
        log.Println(http.ListenAndServe("localhost:6060", nil))
    }()
    
    // Основная логика приложения
}
```

```bash
# Сбор CPU профиля
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Анализ
(pprof) top10
(pprof) web
```

#### Memory профилирование

```bash
# Сбор memory профиля
go tool pprof http://localhost:6060/debug/pprof/heap

# Анализ утечек памяти
(pprof) list functionName
(pprof) png > memory_profile.png
```

### 7.3 Диагностика сети

#### Проверка подключений

```bash
# Статус P2P соединений
curl http://localhost:26657/net_info

# Список пиров
curl http://localhost:26657/net_info | jq '.result.peers'

# Проверка синхронизации
curl http://localhost:26657/status | jq '.result.sync_info'
```

#### Анализ консенсуса

```bash
# Информация о валидаторах
curl http://localhost:26657/validators

# Статус консенсуса
curl http://localhost:26657/consensus_state

# Дамп консенсуса
curl http://localhost:26657/dump_consensus_state
```

## 8. Лучшие практики

### 8.1 Разработка

- Используйте типизированные клиенты для безопасности
- Всегда проверяйте ошибки и обрабатывайте исключения
- Применяйте unit тесты для всей бизнес-логики
- Используйте интеграционные тесты для проверки взаимодействий
- Документируйте все публичные API

### 8.2 Тестирование

- Тестируйте на локальной тестовой сети перед mainnet
- Используйте автоматизированные тесты в CI/CD
- Проводите нагрузочное тестирование
- Тестируйте сценарии восстановления после сбоев

### 8.3 Производство

- Мониторьте все критические метрики
- Настройте алерты для важных событий
- Регулярно создавайте резервные копии
- Используйте логирование для диагностики проблем
- Планируйте масштабирование заранее

## Заключение

Volnix Protocol предоставляет полный набор инструментов для разработчиков, от CLI утилит до SDK и библиотек интеграции. Данное руководство покрывает все основные аспекты разработки на платформе Volnix, включая тестирование, отладку и создание dApps.

Для получения дополнительной информации обращайтесь к документации API и примерам кода в репозитории проекта.