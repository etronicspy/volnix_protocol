# Документация инфраструктуры Volnix Wallet UI

## Обзор

Volnix Wallet UI представляет собой современное React-приложение, предоставляющее пользователям интуитивный интерфейс для взаимодействия с блокчейном Volnix Protocol. Кошелек поддерживает все три типа токенов (WRT, LZN, ANT), различные роли пользователей и полный спектр операций от базовых транзакций до управления валидаторами.

## Архитектура приложения

### Технологический стек

```
Frontend Framework: React 18 + TypeScript
UI Components: Custom components with Lucide React icons
Blockchain Integration: CosmJS (@cosmjs/stargate, @cosmjs/proto-signing)
Styling: CSS3 с современными градиентами и анимациями
Build Tool: Create React App (react-scripts)
Package Manager: npm
```

### Структура проекта

```
volnix-protocol/
└── frontend/                   # Frontend приложения
    └── wallet-ui/              # Веб-интерфейс кошелька
        ├── public/
        │   └── index.html              # HTML шаблон
        ├── src/
        │   ├── components/             # React компоненты
        │   │   ├── WalletConnect.tsx   # Подключение кошелька
        │   │   ├── Balance.tsx         # Отображение балансов
        │   │   ├── SendTokens.tsx      # Отправка токенов
        │   │   ├── TransactionHistory.tsx # История транзакций
        │   │   ├── WalletTypes.tsx     # Управление типами кошельков
        │   │   └── ValidatorManagement.tsx # Управление валидаторами
        │   ├── types/
        │   │   └── wallet.ts           # TypeScript определения
        │   ├── App.tsx                 # Главный компонент приложения
        │   ├── index.tsx               # Точка входа
        │   ├── index.css               # Глобальные стили
        │   └── react-app-env.d.ts      # TypeScript конфигурация
        ├── package.json                # Зависимости и скрипты
        ├── tsconfig.json              # TypeScript конфигурация
        └── README.md                  # Документация проекта
```

### Архитектурные принципы

1. **Компонентная архитектура**: Модульная структура с переиспользуемыми компонентами
2. **Типизация**: Полная типизация с TypeScript для безопасности разработки
3. **Состояние приложения**: Централизованное управление состоянием через React hooks
4. **Responsive Design**: Адаптивный дизайн для различных устройств
5. **Безопасность**: Локальная обработка приватных ключей и валидация данных

## Компоненты и их взаимодействие

### Главный компонент (App.tsx)

Центральный компонент приложения, управляющий:

```typescript
interface WalletState {
  isConnected: boolean;
  address: string;
  balance: Balance;
  walletType: WalletType;
  transactions: Transaction[];
}
```

**Основные функции:**
- Управление состоянием кошелька
- Навигация между разделами
- Обработка подключения/отключения кошелька
- Координация между компонентами

### Компонент подключения (WalletConnect.tsx)

**Назначение**: Обеспечивает безопасное подключение к кошельку

**Функциональность:**
- Создание нового кошелька с пользовательским именем
- Подключение существующего кошелька
- Демо-режим для тестирования
- Генерация mock-адресов для разработки

**Интеграция с блокчейном:**
```typescript
// Будущая интеграция с CosmJS
import { SigningStargateClient } from "@cosmjs/stargate";
import { DirectSecp256k1HdWallet } from "@cosmjs/proto-signing";
```

### Компонент баланса (Balance.tsx)

**Назначение**: Отображение балансов всех токенов и портфеля

**Поддерживаемые токены:**
- **WRT (Wealth Rights Token)**: Основной утилитарный токен
- **LZN (Lizenz Token)**: Токен стейкинга для валидаторов
- **ANT (Anteil Rights)**: Токен управления для граждан

**Особенности:**
- Расчет общей стоимости портфеля
- Блокировка ANT токенов для гостевых кошельков
- Быстрые действия (стейкинг, обмен, клейм)
- Индикаторы доступности функций по ролям

### Компонент отправки (SendTokens.tsx)

**Назначение**: Интерфейс для отправки токенов

**Функциональность:**
- Выбор токена из доступных балансов
- Валидация адреса получателя (формат volnix1...)
- Проверка достаточности средств
- Расчет и отображение комиссий сети
- Предварительный просмотр транзакции

**Безопасность:**
- Валидация всех входных данных
- Предупреждения о необратимости операций
- Двойная проверка адресов

### Компонент истории (TransactionHistory.tsx)

**Назначение**: Отображение истории всех транзакций

**Типы транзакций:**
```typescript
interface Transaction {
  id: string;
  type: 'send' | 'receive';
  amount: string;
  token: string;
  from: string;
  to: string;
  timestamp: string;
  status: 'pending' | 'completed' | 'failed';
}
```

**Функции:**
- Хронологический список транзакций
- Фильтрация по типу и статусу
- Детальная информация о каждой операции
- Обновление статусов в реальном времени

### Компонент типов кошельков (WalletTypes.tsx)

**Назначение**: Управление ролями и правами пользователей

**Поддерживаемые типы:**

1. **Guest (Гость)**
   - Базовая функциональность
   - Доступ к WRT и LZN токенам
   - Ограниченные права управления

2. **Citizen (Гражданин)**
   - Верифицированный пользователь
   - Доступ к ANT токенам
   - Права участия в управлении
   - Требует ZKP верификацию

3. **Validator (Валидатор)**
   - Участник консенсуса
   - Возможность стейкинга LZN
   - Заработок на валидации блоков
   - Требует техническую настройку

### Компонент управления валидаторами (ValidatorManagement.tsx)

**Назначение**: Полное управление валидаторскими кошельками

**Функциональность:**
- Создание кошельков для каждого узла сети
- Подключение кошельков к узлам валидации
- Мониторинг заработка и статистики
- Управление стейкингом и наградами
- Экспорт ключей валидаторов

**Интеграция с сетью:**
- Поддержка множественных узлов (Node-0, Node-1, Node-2)
- Автоматическое начисление наград
- Мониторинг консенсуса PoVB
- Статистика валидации блоков

## Интеграция с блокчейном через CosmJS

### Настройка подключения

```typescript
// Конфигурация подключения к Volnix Protocol
const rpcEndpoint = "http://localhost:26657";
const chainId = "volnix-testnet-1";

// Создание клиента
const client = await SigningStargateClient.connectWithSigner(
  rpcEndpoint,
  signer,
  {
    gasPrice: GasPrice.fromString("0.025uwrt"),
  }
);
```

### Управление кошельками

```typescript
// Создание кошелька из мнемоники
const wallet = await DirectSecp256k1HdWallet.fromMnemonic(
  mnemonic,
  {
    prefix: "volnix",
  }
);

// Получение адресов
const [firstAccount] = await wallet.getAccounts();
const address = firstAccount.address;
```

### Отправка транзакций

```typescript
// Отправка токенов
const sendMsg = {
  typeUrl: "/cosmos.bank.v1beta1.MsgSend",
  value: {
    fromAddress: senderAddress,
    toAddress: recipientAddress,
    amount: [
      {
        denom: "uwrt", // или "ulzn", "uant"
        amount: amount,
      },
    ],
  },
};

const result = await client.signAndBroadcast(
  senderAddress,
  [sendMsg],
  fee
);
```

### Запросы балансов

```typescript
// Получение баланса всех токенов
const balances = await client.getAllBalances(address);

// Получение баланса конкретного токена
const wrtBalance = await client.getBalance(address, "uwrt");
const lznBalance = await client.getBalance(address, "ulzn");
const antBalance = await client.getBalance(address, "uant");
```

### Интеграция с кастомными модулями

```typescript
// Запросы к модулю Identity (x/ident)
const identityQuery = {
  identity: { address: userAddress }
};

// Запросы к модулю Lizenz (x/lizenz)
const lizenzQuery = {
  validator_info: { address: validatorAddress }
};

// Запросы к модулю Anteil (x/anteil)
const anteilQuery = {
  market_orders: { address: userAddress }
};
```

## Управление ключами и безопасность

### Архитектура безопасности

1. **Локальное хранение ключей**
   - Приватные ключи никогда не покидают устройство пользователя
   - Использование браузерного localStorage с шифрованием
   - Поддержка аппаратных кошельков (будущая функция)

2. **Шифрование данных**
   ```typescript
   // Шифрование приватных ключей
   const encryptedKey = await encrypt(privateKey, userPassword);
   localStorage.setItem('volnix_wallet_key', encryptedKey);
   
   // Расшифровка при использовании
   const decryptedKey = await decrypt(encryptedKey, userPassword);
   ```

3. **Валидация транзакций**
   - Проверка формата адресов
   - Валидация сумм и балансов
   - Подтверждение критических операций
   - Предупреждения о высоких комиссиях

### Управление сессиями

```typescript
interface WalletSession {
  address: string;
  walletType: WalletType;
  connectedAt: Date;
  lastActivity: Date;
  autoLockTimeout: number;
}

// Автоматическая блокировка после неактивности
const AUTO_LOCK_TIMEOUT = 30 * 60 * 1000; // 30 минут

// Проверка активности сессии
const checkSessionActivity = () => {
  const lastActivity = localStorage.getItem('last_activity');
  const now = Date.now();
  
  if (now - parseInt(lastActivity) > AUTO_LOCK_TIMEOUT) {
    lockWallet();
  }
};
```

### Резервное копирование

```typescript
// Экспорт мнемонической фразы
const exportMnemonic = async (password: string) => {
  const encryptedMnemonic = await encrypt(mnemonic, password);
  return {
    mnemonic: encryptedMnemonic,
    timestamp: new Date().toISOString(),
    version: "1.0"
  };
};

// Импорт кошелька
const importWallet = async (
  encryptedMnemonic: string, 
  password: string
) => {
  const mnemonic = await decrypt(encryptedMnemonic, password);
  const wallet = await DirectSecp256k1HdWallet.fromMnemonic(mnemonic);
  return wallet;
};
```

## Кастомизация и расширение функциональности

### Добавление новых компонентов

1. **Создание компонента**
   ```typescript
   // src/components/NewFeature.tsx
   import React from 'react';
   import { SomeIcon } from 'lucide-react';
   
   interface NewFeatureProps {
     // Определение пропсов
   }
   
   const NewFeature: React.FC<NewFeatureProps> = ({ props }) => {
     return (
       <div className="card">
         {/* Реализация компонента */}
       </div>
     );
   };
   
   export default NewFeature;
   ```

2. **Интеграция в приложение**
   ```typescript
   // src/App.tsx
   import NewFeature from './components/NewFeature';
   
   // Добавление в навигацию
   const [activeTab, setActiveTab] = useState<'wallet' | 'send' | 'history' | 'newfeature'>('wallet');
   
   // Добавление в рендер
   {activeTab === 'newfeature' && <NewFeature />}
   ```

### Расширение типов данных

```typescript
// src/types/wallet.ts
export interface ExtendedWalletState extends WalletState {
  // Новые поля состояния
  stakingRewards: StakingReward[];
  governanceProposals: Proposal[];
  nftCollections: NFTCollection[];
}

// Новые типы для расширений
export interface StakingReward {
  validatorAddress: string;
  amount: string;
  claimableAt: Date;
}
```

### Кастомизация стилей

```css
/* src/index.css - Переменные для кастомизации */
:root {
  --primary-color: #667eea;
  --secondary-color: #764ba2;
  --success-color: #10b981;
  --warning-color: #f59e0b;
  --error-color: #ef4444;
  --background-gradient: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

/* Кастомные темы */
.theme-dark {
  --background-color: #1f2937;
  --text-color: #f9fafb;
  --card-background: #374151;
}

.theme-light {
  --background-color: #f9fafb;
  --text-color: #1f2937;
  --card-background: #ffffff;
}
```

### Добавление новых токенов

```typescript
// Расширение поддержки токенов
interface ExtendedBalance extends Balance {
  // Новые токены
  gov: string;    // Governance token
  nft: string;    // NFT collection count
  defi: string;   // DeFi protocol tokens
}

// Конфигурация токенов
const TOKEN_CONFIG = {
  wrt: { name: 'Wealth Rights Token', decimals: 6, icon: 'coins' },
  lzn: { name: 'Lizenz Token', decimals: 6, icon: 'trending-up' },
  ant: { name: 'Anteil Rights', decimals: 6, icon: 'shield' },
  gov: { name: 'Governance Token', decimals: 6, icon: 'vote' },
};
```

### Интеграция с внешними сервисами

```typescript
// Интеграция с DeFi протоколами
interface DeFiIntegration {
  protocol: string;
  tvl: string;
  apy: number;
  userStake: string;
}

// Интеграция с NFT маркетплейсами
interface NFTIntegration {
  marketplace: string;
  collections: NFTCollection[];
  userNFTs: NFT[];
}

// Интеграция с аналитикой
interface AnalyticsIntegration {
  portfolioValue: string;
  performance24h: number;
  transactions30d: number;
  stakingRewards: string;
}
```

## Развертывание и конфигурация

### Локальная разработка

```bash
# Клонирование репозитория
git clone https://github.com/volnix-protocol/volnix-protocol.git
cd volnix-protocol/frontend/wallet-ui

# Установка зависимостей
npm install

# Настройка переменных окружения
cp .env.example .env.local

# Запуск в режиме разработки
npm start
```

### Переменные окружения

```bash
# .env.local
REACT_APP_CHAIN_ID=volnix-testnet-1
REACT_APP_RPC_ENDPOINT=http://localhost:26657
REACT_APP_REST_ENDPOINT=http://localhost:1317
REACT_APP_WEBSOCKET_ENDPOINT=ws://localhost:26657/websocket
REACT_APP_EXPLORER_URL=http://localhost:3001
REACT_APP_FAUCET_URL=http://localhost:4500
```

### Сборка для продакшена

```bash
# Сборка оптимизированной версии
npm run build

# Результат в папке build/
# Готов для развертывания на веб-сервере
```

### Docker развертывание

```dockerfile
# Dockerfile
FROM node:18-alpine as builder

WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production

COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=builder /app/build /usr/share/nginx/html
COPY nginx.conf /etc/nginx/nginx.conf

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
```

### Nginx конфигурация

```nginx
# nginx.conf
server {
    listen 80;
    server_name localhost;
    
    location / {
        root /usr/share/nginx/html;
        index index.html index.htm;
        try_files $uri $uri/ /index.html;
    }
    
    # API проксирование
    location /api/ {
        proxy_pass http://volnix-node:1317/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
    
    # WebSocket поддержка
    location /websocket {
        proxy_pass http://volnix-node:26657;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

## Мониторинг и аналитика

### Метрики производительности

```typescript
// Мониторинг производительности
interface PerformanceMetrics {
  loadTime: number;
  renderTime: number;
  transactionTime: number;
  errorRate: number;
  userSessions: number;
}

// Сбор метрик
const collectMetrics = () => {
  const navigation = performance.getEntriesByType('navigation')[0];
  return {
    loadTime: navigation.loadEventEnd - navigation.loadEventStart,
    renderTime: navigation.domContentLoadedEventEnd - navigation.domContentLoadedEventStart,
    // Дополнительные метрики
  };
};
```

### Логирование ошибок

```typescript
// Централизованное логирование
interface ErrorLog {
  timestamp: Date;
  level: 'error' | 'warning' | 'info';
  message: string;
  stack?: string;
  userAgent: string;
  url: string;
}

// Обработчик ошибок
const errorHandler = (error: Error, errorInfo: any) => {
  const errorLog: ErrorLog = {
    timestamp: new Date(),
    level: 'error',
    message: error.message,
    stack: error.stack,
    userAgent: navigator.userAgent,
    url: window.location.href,
  };
  
  // Отправка на сервер логирования
  sendErrorLog(errorLog);
};
```

### Пользовательская аналитика

```typescript
// Отслеживание действий пользователей
interface UserAction {
  action: string;
  timestamp: Date;
  userId?: string;
  metadata?: Record<string, any>;
}

// Трекинг событий
const trackEvent = (action: string, metadata?: Record<string, any>) => {
  const event: UserAction = {
    action,
    timestamp: new Date(),
    userId: getCurrentUserId(),
    metadata,
  };
  
  // Отправка аналитики
  sendAnalytics(event);
};
```

## Заключение

Volnix Wallet UI представляет собой современное, безопасное и расширяемое решение для взаимодействия с блокчейном Volnix Protocol. Архитектура приложения обеспечивает:

- **Безопасность**: Локальное управление ключами и шифрование данных
- **Масштабируемость**: Модульная архитектура для легкого расширения
- **Производительность**: Оптимизированный React код с TypeScript
- **Пользовательский опыт**: Интуитивный интерфейс с современным дизайном
- **Интеграция**: Полная поддержка всех функций протокола Volnix

Документация обеспечивает полное понимание архитектуры, возможностей интеграции и путей расширения функциональности для разработчиков и системных администраторов.