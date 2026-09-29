# Volnix Wallet — simulation v2

Отдельный UI кошелька для ноды `simulation_v2/backend`. Explorer (`simulation_v2/frontend`, порт 5174) не используется.

Сид-фраза из 12 слов — строка-ключ стенда (`SHA256` текста, не BIP39). Та же фраза открывает тот же адрес `volnix1…`.

## Run

```bash
# terminal 1 — node
cd ../backend && python3 main.py

# terminal 2 — wallet
cd simulation_v2/wallet
npm install
npm run dev
```

Open [http://127.0.0.1:5175](http://127.0.0.1:5175).

Env overrides:

- `VITE_API_URL` — REST base (default `http://127.0.0.1:8001`)
- `VITE_WS_URL` — WebSocket (default `ws://127.0.0.1:8001/ws`)

`POST /operator/account` только выводит адрес и не создаёт счёт. Роль, перевод, сжигание и ордера попадают в состояние только транзакцией в блоке (`/operator/verify|declare|tx` → мемпул → DeliverTx). Фраза лежит в `sessionStorage`, пока вкладка открыта. «Запомнить на этом браузере» пишет её в `localStorage`. Выход стирает оба хранилища.
