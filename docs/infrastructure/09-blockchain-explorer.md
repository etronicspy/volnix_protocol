# 🔍 Blockchain Explorer - Документация инфраструктуры

## Обзор

Volnix Blockchain Explorer представляет собой полнофункциональный веб-интерфейс для мониторинга и анализа блокчейна Volnix Protocol в режиме реального времени. Эксплорер обеспечивает прозрачность сети, позволяя пользователям отслеживать блоки, транзакции, валидаторов и специфичные для протокола операции, включая консенсус PoVB и работу кастомных модулей.

## Архитектура и функциональность

### Архитектурная схема

```
┌─────────────────────────────────────────────────────────────┐
│                    Volnix Blockchain Explorer               │
├─────────────────────────────────────────────────────────────┤
│  Frontend Layer (HTML/CSS/JavaScript)                      │
│  ┌─────────────┬─────────────┬─────────────┬─────────────┐  │
│  │   Blocks    │Transactions │ Validators  │  Modules    │  │
│  │   Monitor   │   Tracker   │  Dashboard  │  Status     │  │
│  └─────────────┴─────────────┴─────────────┴─────────────┘  │
├─────────────────────────────────────────────────────────────┤
│  HTTP Server Layer (PowerShell)                            │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │  Static File Server + API Proxy                        │ │
│  └─────────────────────────────────────────────────────────┘ │
├─────────────────────────────────────────────────────────────┤
│  Integration Layer                                          │
│  ┌─────────────┬─────────────┬─────────────┬─────────────┐  │
│  │ RPC Client  │ REST API    │ WebSocket   │ gRPC Client │  │
│  │ (CometBFT)  │ (Cosmos)    │ (Real-time) │ (Modules)   │  │
│  └─────────────┴─────────────┴─────────────┴─────────────┘  │
├─────────────────────────────────────────────────────────────┤
│  Volnix Protocol Node                                       │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │  volnixd + Custom Modules (ident/lizenz/anteil/consensus) │ │
│  └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### Основные компоненты

#### 1. Frontend Interface
- **Технология**: Vanilla HTML5/CSS3/JavaScript
- **Дизайн**: Responsive UI с градиентным дизайном
- **Компоненты**:
  - Network Overview Dashboard
  - Blocks Browser
  - Transactions Monitor
  - Validators Dashboard
  - Modules Status Panel
  - PoVB Consensus Monitor

#### 2. HTTP Server
- **Платформа**: PowerShell HTTP Listener
- **Порт**: 8080 (по умолчанию)
- **Функции**:
  - Статический файловый сервер
  - CORS поддержка
  - Логирование запросов
  - Error handling

#### 3. Data Integration Layer
- **RPC Integration**: Подключение к CometBFT RPC
- **REST API**: Cosmos SDK REST endpoints
- **gRPC**: Прямое подключение к модулям
- **WebSocket**: Real-time обновления (планируется)

## Функциональные возможности

### 📊 Network Overview
Центральная панель с ключевыми метриками сети:

```javascript
// Основные метрики
const networkMetrics = {
    totalBlocks: "Общее количество блоков",
    totalTransactions: "Общее количество транзакций", 
    activeValidators: "Активные валидаторы",
    burnedANT: "Сожженные ANT токены (PoVB)",
    avgBlockTime: "Среднее время блока",
    networkHealth: "Здоровье сети (%)"
};
```

**Отображаемые данные**:
- Chain ID и статус сети
- Высота блока в реальном времени
- Количество активных валидаторов
- Статистика PoVB консенсуса
- Общие метрики производительности

### 📦 Blocks Monitor
Детальный просмотр блоков с возможностями поиска:

**Функции**:
- Список последних блоков с пагинацией
- Поиск по номеру блока или хешу
- Детальная информация о каждом блоке:
  - Номер блока и хеш
  - Время создания
  - Валидатор-создатель
  - Количество транзакций
  - Размер блока
  - Gas usage

**Интерфейс блока**:
```html
<div class="block-item">
    <div class="block-info">
        <div class="block-number">Block #12,345</div>
        <div class="block-hash">0x1a2b3c4d5e6f7890abcdef...</div>
        <div class="block-time">2 minutes ago</div>
    </div>
    <div class="block-stats">
        <div class="tx-count">25 txs</div>
        <div class="validator">Validator: Node-1</div>
        <button onclick="viewBlock(12345)">View Details</button>
    </div>
</div>
```

### 💸 Transactions Tracker
Мониторинг всех типов транзакций в сети:

**Поддерживаемые типы транзакций**:
- **Transfer**: Переводы WRT токенов
- **Stake LZN**: Стейкинг лицензий валидаторами
- **ANT Burn**: Сжигание ANT в рамках PoVB
- **Identity Verification**: ZKP верификация через x/ident
- **License Activation**: Активация лицензий через x/lizenz
- **ANT Trading**: Торговля правами через x/anteil

**Отображаемая информация**:
- Хеш транзакции
- Тип операции
- Сумма и токен
- Комиссия
- Статус выполнения (Success/Failed)
- Блок и время

### 👑 Validators Dashboard
Мониторинг валидаторов и их производительности:

**Метрики валидаторов**:
```javascript
const validatorMetrics = {
    address: "volnix1validator0abc123...",
    moniker: "Node-0 (Validator)",
    status: "Active/Inactive",
    stakedLZN: "1,000 LZN",
    uptime: "99.8%",
    blocksProduced: "4,123",
    port: "26650",
    lastSeen: "2 minutes ago"
};
```

**Функции мониторинга**:
- Статус валидатора (Active/Jailed/Unbonding)
- Количество застейканных LZN
- Uptime и производительность
- Количество созданных блоков
- MOA (Minimum Obligation Activity) статус
- Сетевые параметры (порты, эндпоинты)

### 🔧 Modules Status Panel
Мониторинг кастомных модулей Volnix Protocol:

#### Identity Module (x/ident)
```javascript
const identityStatus = {
    status: "Active",
    verifiedUsers: 1234,
    successRate: "98.5%",
    zkpVerifications: "Daily count",
    roleMigrations: "Pending migrations"
};
```

#### Lizenz Module (x/lizenz)
```javascript
const lizenzStatus = {
    status: "Active", 
    activeLicenses: 567,
    totalStaked: "45,678 LZN",
    moaCompliance: "95.2%",
    penaltiesIssued: "12 this epoch"
};
```

#### Anteil Module (x/anteil)
```javascript
const anteilStatus = {
    status: "Active",
    marketOrders: 89,
    tradingVolume: "12,345 ANT",
    activeAuctions: 5,
    priceRange: "0.95-1.05 WRT/ANT"
};
```

#### Consensus Module (x/consensus)
```javascript
const consensusStatus = {
    status: "Active",
    totalBurned: "1,250 ANT",
    efficiency: "99.2%",
    currentRound: 2468,
    roundTimeRemaining: "3m 45s"
};
```

### ⚖️ PoVB Consensus Monitor
Специализированный мониторинг консенсуса Proof-of-Verified-Burn:

**Отслеживаемые данные**:
- Текущий раунд консенсуса
- Время до завершения раунда
- Лидирующий валидатор
- История сжигания ANT
- Статистика эффективности
- Участники текущего раунда

**Визуализация раунда**:
```html
<div class="consensus-round">
    <h4>Current Round #2,468</h4>
    <div class="round-info">
        <div>Time Remaining: 3m 45s</div>
        <div>Leading: Node-1 (150 ANT burned)</div>
        <div>Participants: 3 validators</div>
    </div>
    <div class="burn-history">
        <!-- История сжигания в текущем раунде -->
    </div>
</div>
```

## Мониторинг сети в реальном времени

### Auto-Refresh System
Эксплорер автоматически обновляет данные каждые 30 секунд:

```javascript
function startAutoRefresh() {
    refreshInterval = setInterval(() => {
        refreshData();
    }, 30000); // 30 секунд
}

function refreshData() {
    // Обновление метрик сети
    updateNetworkMetrics();
    // Обновление списка блоков
    updateBlocksList();
    // Обновление транзакций
    updateTransactionsList();
    // Обновление статуса валидаторов
    updateValidatorsList();
    // Обновление статуса модулей
    updateModulesStatus();
}
```

### Real-time Notifications
Система уведомлений о важных событиях:

```javascript
function showNotification(message, type) {
    const notification = document.createElement('div');
    notification.className = type; // 'success', 'error', 'warning'
    notification.textContent = message;
    // Позиционирование и автоудаление
}
```

**Типы уведомлений**:
- Новые блоки
- Крупные транзакции
- Изменения статуса валидаторов
- Системные события модулей
- Ошибки сети

### Performance Monitoring
Мониторинг производительности самого эксплорера:

```javascript
const performanceMetrics = {
    loadTime: "Время загрузки страницы",
    apiResponseTime: "Время ответа API",
    updateFrequency: "Частота обновлений",
    errorRate: "Частота ошибок",
    memoryUsage: "Использование памяти браузера"
};
```

## Руководство по настройке и развертыванию

### Системные требования

**Минимальные требования**:
- Windows 10/11 или Windows Server 2019+
- PowerShell 5.1 или PowerShell Core 7+
- 512 MB RAM
- 100 MB свободного места
- Сетевое подключение к узлу Volnix

**Рекомендуемые требования**:
- Windows 11 или Windows Server 2022
- PowerShell Core 7.3+
- 2 GB RAM
- 1 GB свободного места
- Gigabit Ethernet подключение

### Установка и запуск

#### Шаг 1: Подготовка окружения
```powershell
# Проверка версии PowerShell
$PSVersionTable.PSVersion

# Проверка политики выполнения
Get-ExecutionPolicy

# При необходимости изменить политику
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

#### Шаг 2: Запуск эксплорера
```powershell
# Переход в директорию эксплорера
cd blockchain-explorer

# Запуск сервера
powershell -ExecutionPolicy Bypass -File start-explorer.ps1
```

#### Шаг 3: Проверка работы
```
Откройте браузер и перейдите по адресу:
http://localhost:8080/

Ожидаемый результат:
- Загрузка интерфейса эксплорера
- Отображение статистики сети
- Успешное уведомление о загрузке
```

### Конфигурация сервера

#### Изменение порта
Отредактируйте файл `start-explorer.ps1`:

```powershell
# Изменить порт по умолчанию
$port = 8080  # Замените на желаемый порт
```

#### Настройка CORS
Для интеграции с внешними приложениями:

```powershell
# Добавить CORS заголовки
$response.Headers.Add("Access-Control-Allow-Origin", "*")
$response.Headers.Add("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
$response.Headers.Add("Access-Control-Allow-Headers", "Content-Type")
```

#### Логирование
Настройка детального логирования:

```powershell
# Включить детальное логирование
$logFile = "explorer-$(Get-Date -Format 'yyyy-MM-dd').log"
$logEntry = "$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') - $($request.HttpMethod) $($request.Url.AbsolutePath)"
Add-Content -Path $logFile -Value $logEntry
```

### Развертывание в продакшене

#### IIS Integration
Для продакшен развертывания рекомендуется использовать IIS:

```xml
<!-- web.config для IIS -->
<?xml version="1.0" encoding="UTF-8"?>
<configuration>
    <system.webServer>
        <defaultDocument>
            <files>
                <clear />
                <add value="index.html" />
            </files>
        </defaultDocument>
        <staticContent>
            <mimeMap fileExtension=".json" mimeType="application/json" />
        </staticContent>
    </system.webServer>
</configuration>
```

#### Nginx Proxy
Конфигурация Nginx для проксирования:

```nginx
server {
    listen 80;
    server_name explorer.volnix.local;
    
    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
```

#### Docker Deployment
Dockerfile для контейнеризации:

```dockerfile
FROM mcr.microsoft.com/powershell:lts-alpine

WORKDIR /app
COPY . .

EXPOSE 8080

CMD ["pwsh", "-File", "start-explorer.ps1"]
```

## Интеграция с RPC эндпоинтами и API

### CometBFT RPC Integration

#### Основные эндпоинты
```javascript
const cometBFTEndpoints = {
    status: "/status",                    // Статус узла
    block: "/block?height={height}",      // Данные блока
    blockResults: "/block_results?height={height}", // Результаты блока
    validators: "/validators?height={height}",      // Валидаторы
    consensus: "/consensus_state",        // Состояние консенсуса
    netInfo: "/net_info",                // Сетевая информация
    abciInfo: "/abci_info"               // ABCI информация
};
```

#### Пример запроса блока
```javascript
async function fetchBlock(height) {
    try {
        const response = await fetch(`http://localhost:26657/block?height=${height}`);
        const data = await response.json();
        return {
            height: data.result.block.header.height,
            hash: data.result.block_id.hash,
            time: data.result.block.header.time,
            proposer: data.result.block.header.proposer_address,
            txs: data.result.block.data.txs || []
        };
    } catch (error) {
        console.error('Error fetching block:', error);
        return null;
    }
}
```

### Cosmos SDK REST API

#### Основные эндпоинты
```javascript
const cosmosEndpoints = {
    // Базовые эндпоинты
    nodeInfo: "/cosmos/base/tendermint/v1beta1/node_info",
    blocks: "/cosmos/base/tendermint/v1beta1/blocks/{height}",
    
    // Банковский модуль
    balances: "/cosmos/bank/v1beta1/balances/{address}",
    supply: "/cosmos/bank/v1beta1/supply",
    
    // Стейкинг
    validators: "/cosmos/staking/v1beta1/validators",
    delegations: "/cosmos/staking/v1beta1/delegations/{delegator}",
    
    // Транзакции
    txs: "/cosmos/tx/v1beta1/txs",
    txByHash: "/cosmos/tx/v1beta1/txs/{hash}"
};
```

#### Пример запроса валидаторов
```javascript
async function fetchValidators() {
    try {
        const response = await fetch('http://localhost:1317/cosmos/staking/v1beta1/validators');
        const data = await response.json();
        return data.validators.map(validator => ({
            address: validator.operator_address,
            moniker: validator.description.moniker,
            status: validator.status,
            tokens: validator.tokens,
            commission: validator.commission.commission_rates.rate
        }));
    } catch (error) {
        console.error('Error fetching validators:', error);
        return [];
    }
}
```

### Custom Modules gRPC Integration

#### Identity Module (x/ident)
```javascript
const identityQueries = {
    // Получить информацию о пользователе
    getUserInfo: "/volnix.ident.v1.Query/UserInfo",
    // Получить статистику верификации
    getVerificationStats: "/volnix.ident.v1.Query/VerificationStats",
    // Получить список ролей
    getRoles: "/volnix.ident.v1.Query/Roles"
};

async function fetchIdentityStats() {
    const response = await grpcQuery(identityQueries.getVerificationStats, {});
    return {
        totalUsers: response.total_users,
        verifiedUsers: response.verified_users,
        successRate: response.success_rate
    };
}
```

#### Lizenz Module (x/lizenz)
```javascript
const lizenzQueries = {
    // Получить активные лицензии
    getActiveLicenses: "/volnix.lizenz.v1.Query/ActiveLicenses",
    // Получить статистику MOA
    getMOAStats: "/volnix.lizenz.v1.Query/MOAStats",
    // Получить информацию о лицензии
    getLicenseInfo: "/volnix.lizenz.v1.Query/LicenseInfo"
};
```

#### Anteil Module (x/anteil)
```javascript
const anteilQueries = {
    // Получить рыночные ордера
    getMarketOrders: "/volnix.anteil.v1.Query/MarketOrders",
    // Получить статистику торговли
    getTradingStats: "/volnix.anteil.v1.Query/TradingStats",
    // Получить активные аукционы
    getActiveAuctions: "/volnix.anteil.v1.Query/ActiveAuctions"
};
```

#### Consensus Module (x/consensus)
```javascript
const consensusQueries = {
    // Получить статистику сжигания
    getBurnStats: "/volnix.consensus.v1.Query/BurnStats",
    // Получить текущий раунд
    getCurrentRound: "/volnix.consensus.v1.Query/CurrentRound",
    // Получить историю раундов
    getRoundHistory: "/volnix.consensus.v1.Query/RoundHistory"
};
```

### WebSocket Integration (Планируется)

#### Real-time Events
```javascript
const wsConnection = new WebSocket('ws://localhost:26657/websocket');

// Подписка на события
const subscriptions = {
    newBlock: "tm.event='NewBlock'",
    newTx: "tm.event='Tx'",
    validatorSetUpdates: "tm.event='ValidatorSetUpdates'",
    // Кастомные события модулей
    identityVerification: "volnix.ident.verification",
    antBurn: "volnix.consensus.burn",
    antTrade: "volnix.anteil.trade"
};

wsConnection.onmessage = function(event) {
    const data = JSON.parse(event.data);
    handleRealtimeUpdate(data);
};
```

## Инструкции по кастомизации интерфейса

### Темы и стили

#### Цветовая схема
Основная цветовая палитра определена в CSS:

```css
:root {
    /* Основные цвета */
    --primary-blue: #1e3a8a;
    --secondary-blue: #3730a3;
    --accent-green: #10b981;
    --accent-purple: #8b5cf6;
    --accent-orange: #f59e0b;
    --accent-red: #ef4444;
    
    /* Градиенты */
    --gradient-primary: linear-gradient(135deg, #1e3a8a 0%, #3730a3 100%);
    --gradient-success: linear-gradient(135deg, #10b981 0%, #059669 100%);
    --gradient-warning: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
}
```

#### Кастомизация темы
Для создания новой темы:

```css
/* Темная тема */
.dark-theme {
    --bg-primary: #1a1a1a;
    --bg-secondary: #2d2d2d;
    --text-primary: #ffffff;
    --text-secondary: #cccccc;
    --border-color: #404040;
}

/* Применение темы */
body.dark-theme {
    background: var(--bg-primary);
    color: var(--text-primary);
}
```

### Добавление новых компонентов

#### Создание нового виджета
```javascript
function createCustomWidget(title, data, type) {
    const widget = document.createElement('div');
    widget.className = `custom-widget widget-${type}`;
    widget.innerHTML = `
        <div class="widget-header">
            <h4>${title}</h4>
            <button class="widget-refresh" onclick="refreshWidget('${type}')">🔄</button>
        </div>
        <div class="widget-content">
            ${renderWidgetContent(data, type)}
        </div>
    `;
    return widget;
}
```

#### Добавление новой вкладки
```javascript
function addCustomTab(tabName, tabContent) {
    // Добавить кнопку вкладки
    const tabButton = document.createElement('button');
    tabButton.className = 'tab';
    tabButton.textContent = tabName;
    tabButton.onclick = () => showTab(tabName.toLowerCase());
    
    // Добавить контент вкладки
    const tabDiv = document.createElement('div');
    tabDiv.id = `${tabName.toLowerCase()}-tab`;
    tabDiv.className = 'tab-content';
    tabDiv.innerHTML = tabContent;
    
    // Вставить в DOM
    document.querySelector('.tabs').appendChild(tabButton);
    document.querySelector('.container').appendChild(tabDiv);
}
```

### Конфигурация отображения

#### Настройка метрик
```javascript
const customMetrics = {
    // Добавить новую метрику
    addMetric: function(name, value, label, color) {
        const statsGrid = document.querySelector('.stats-grid');
        const metricCard = document.createElement('div');
        metricCard.className = 'stat-card';
        metricCard.style.borderLeftColor = color;
        metricCard.innerHTML = `
            <div class="stat-value" style="color: ${color}">${value}</div>
            <div class="stat-label">${label}</div>
        `;
        statsGrid.appendChild(metricCard);
    },
    
    // Обновить существующую метрику
    updateMetric: function(id, value) {
        const element = document.getElementById(id);
        if (element) {
            element.textContent = value;
        }
    }
};
```

#### Фильтры и поиск
```javascript
const customFilters = {
    // Добавить фильтр по типу транзакции
    addTransactionFilter: function(type, label) {
        const filterContainer = document.createElement('div');
        filterContainer.innerHTML = `
            <label>
                <input type="checkbox" value="${type}" onchange="filterTransactions()">
                ${label}
            </label>
        `;
        document.querySelector('.filters').appendChild(filterContainer);
    },
    
    // Применить фильтры
    applyFilters: function() {
        const activeFilters = Array.from(document.querySelectorAll('.filters input:checked'))
            .map(input => input.value);
        
        document.querySelectorAll('.tx-item').forEach(item => {
            const txType = item.dataset.type;
            item.style.display = activeFilters.length === 0 || activeFilters.includes(txType) 
                ? 'flex' : 'none';
        });
    }
};
```

### Интеграция с внешними системами

#### API для внешних приложений
```javascript
// Экспорт данных для внешних приложений
window.VolnixExplorer = {
    // Получить текущие метрики
    getNetworkMetrics: function() {
        return {
            blockHeight: document.getElementById('block-height').textContent,
            totalBlocks: document.getElementById('total-blocks').textContent,
            totalTxs: document.getElementById('total-txs').textContent,
            activeValidators: document.getElementById('active-validators').textContent,
            burnedANT: document.getElementById('burned-ant').textContent
        };
    },
    
    // Подписаться на обновления
    onUpdate: function(callback) {
        document.addEventListener('explorerUpdate', callback);
    },
    
    // Получить данные блока
    getBlockData: function(height) {
        return fetchBlock(height);
    }
};
```

#### Webhook интеграция
```javascript
const webhookConfig = {
    endpoints: [],
    
    // Добавить webhook
    addWebhook: function(url, events) {
        this.endpoints.push({ url, events });
    },
    
    // Отправить webhook
    sendWebhook: function(event, data) {
        this.endpoints.forEach(endpoint => {
            if (endpoint.events.includes(event)) {
                fetch(endpoint.url, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ event, data, timestamp: Date.now() })
                });
            }
        });
    }
};
```

## Безопасность и производительность

### Меры безопасности

#### Input Validation
```javascript
function sanitizeInput(input) {
    return input.replace(/[<>\"']/g, '');
}

function validateBlockHeight(height) {
    const num = parseInt(height);
    return !isNaN(num) && num > 0 && num <= getCurrentBlockHeight();
}

function validateAddress(address) {
    return /^volnix1[a-z0-9]{38}$/.test(address);
}
```

#### Rate Limiting
```javascript
const rateLimiter = {
    requests: new Map(),
    limit: 100, // запросов в минуту
    
    checkLimit: function(ip) {
        const now = Date.now();
        const requests = this.requests.get(ip) || [];
        const recentRequests = requests.filter(time => now - time < 60000);
        
        if (recentRequests.length >= this.limit) {
            return false;
        }
        
        recentRequests.push(now);
        this.requests.set(ip, recentRequests);
        return true;
    }
};
```

### Оптимизация производительности

#### Кэширование данных
```javascript
const cache = {
    data: new Map(),
    ttl: 30000, // 30 секунд
    
    get: function(key) {
        const item = this.data.get(key);
        if (item && Date.now() - item.timestamp < this.ttl) {
            return item.value;
        }
        return null;
    },
    
    set: function(key, value) {
        this.data.set(key, {
            value: value,
            timestamp: Date.now()
        });
    }
};
```

#### Lazy Loading
```javascript
const lazyLoader = {
    observers: new Map(),
    
    observe: function(element, callback) {
        const observer = new IntersectionObserver((entries) => {
            entries.forEach(entry => {
                if (entry.isIntersecting) {
                    callback(entry.target);
                    observer.unobserve(entry.target);
                }
            });
        });
        
        observer.observe(element);
        this.observers.set(element, observer);
    }
};
```

## Мониторинг и диагностика

### Системные метрики
```javascript
const systemMonitor = {
    startTime: Date.now(),
    
    getUptime: function() {
        return Date.now() - this.startTime;
    },
    
    getMemoryUsage: function() {
        if (performance.memory) {
            return {
                used: performance.memory.usedJSHeapSize,
                total: performance.memory.totalJSHeapSize,
                limit: performance.memory.jsHeapSizeLimit
            };
        }
        return null;
    },
    
    getPerformanceMetrics: function() {
        return {
            uptime: this.getUptime(),
            memory: this.getMemoryUsage(),
            loadTime: performance.timing.loadEventEnd - performance.timing.navigationStart
        };
    }
};
```

### Error Handling
```javascript
const errorHandler = {
    errors: [],
    
    logError: function(error, context) {
        const errorInfo = {
            message: error.message,
            stack: error.stack,
            context: context,
            timestamp: Date.now(),
            userAgent: navigator.userAgent
        };
        
        this.errors.push(errorInfo);
        console.error('Explorer Error:', errorInfo);
        
        // Отправить на сервер мониторинга (если настроен)
        this.reportError(errorInfo);
    },
    
    reportError: function(errorInfo) {
        // Отправка ошибки на сервер мониторинга
        fetch('/api/errors', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(errorInfo)
        }).catch(() => {
            // Игнорировать ошибки отправки
        });
    }
};

// Глобальный обработчик ошибок
window.addEventListener('error', (event) => {
    errorHandler.logError(event.error, 'Global error handler');
});
```

## Заключение

Volnix Blockchain Explorer представляет собой мощный инструмент для мониторинга и анализа блокчейна Volnix Protocol. Благодаря модульной архитектуре, простоте развертывания и богатым возможностям кастомизации, эксплорер обеспечивает полную прозрачность сети и удобный интерфейс для всех участников экосистемы.

Ключевые преимущества:
- **Полная интеграция** с Volnix Protocol и его кастомными модулями
- **Real-time мониторинг** всех аспектов сети
- **Простое развертывание** на Windows платформах
- **Гибкая кастомизация** интерфейса и функциональности
- **Высокая производительность** и безопасность

Эксплорер продолжает развиваться вместе с протоколом, добавляя новые возможности мониторинга и анализа по мере развития экосистемы Volnix.