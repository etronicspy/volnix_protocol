# Volnix Simulation Wallet UI

Standalone кошелёк для **off-chain симулятора** (`simulation/backend`, порт 8000).

Это не `frontend/wallet-ui` (CosmJS → Cosmos RPC `:26657`). Здесь нет мнемоник и
подписей: адрес — строка из состояния симуляции, транзакции идут в мемпул через
`POST /api/wallet/submit`.

## Возможности

- Выбор / создание аккаунта (`GET /api/state`, `POST /api/sim-operator/accounts`)
- Live-балансы по WebSocket `/ws`
- ZKP verify, смена роли, перевод WRT/LZN, активация LZN, declare
- Рынок ANT (limit/market + cancel)
- История (`GET /api/account/{addr}/history`)
- Faucet mint через sim-operator (явно помечен как demo)

## Запуск

Нужен запущенный симулятор на `:8000`.

```bash
cd simulation/wallet-ui
npm install
npm run dev          # http://127.0.0.1:5174
```

Env (опционально):

```bash
VITE_API_URL=http://localhost:8000
VITE_WS_URL=ws://localhost:8000/ws
```

## Связанные URL

| Сервис | URL |
|--------|-----|
| Sim backend | http://127.0.0.1:8000 |
| Sim dashboard | http://127.0.0.1:5173 |
| **Sim wallet** | http://127.0.0.1:5174 |
| Cosmos wallet-ui | http://127.0.0.1:3000 (отдельный продукт) |
