# Volnix Traffic UI — simulation v2

Отдельный пульт управления стендом `simulation_v2`. Explorer (`simulation_v2/frontend`, порт 5174)
остаётся read-only; всё, что меняет цепь или экономику снаружи, живёт здесь:

| Секция | Куда ходит | Что делает |
|--------|------------|------------|
| Time scale | нода `GET /api/v1/operator/pace`, `POST /api/v1/operator/time-scale` | Стендовое ускорение `wall_sleep = T / scale` (1…3600×). Traffic следует за `produce_interval_sec` |
| Reset chain | нода `POST /api/v1/operator/reset-chain`, затем traffic `POST /reset` | Сброс блокстора к genesis и очистка реестра ботов |
| Traffic-панель | traffic `GET /status`, `POST /start|stop|tick|intensity|release|reset` | Запуск/пауза экономики, Intensity, Release/Burn, таблица агентов |
| Кошельки | localStorage + traffic `GET /wallets`; нода `/api/v1/operator/*` (в т.ч. `mint`) и `/ws` | Список локальных сид-фраз и ботов; перевод, роль, declare, LZN, ордера, пополнение WRT из genesis |

Ссылки на адреса и высоты в таблице агентов открывают explorer в новой вкладке.

## Run

```bash
# terminal 1 — node (с traffic держите VOLNIX_SIM2_AUTO_DECLARE=false)
cd ../backend && python3 main.py

# terminal 2 — traffic / economy
cd ../traffic && pip install -r requirements.txt && python3 main.py

# terminal 3 — explorer (read-only, для ссылок)
cd ../frontend && npm install && npm run dev

# terminal 4 — traffic UI
cd simulation_v2/traffic-ui
npm install
npm run dev
```

Open [http://127.0.0.1:5176](http://127.0.0.1:5176).

Пока процесс traffic не запущен, панель показывает «Процесс трафика не запущен», кнопки и слайдер
Intensity неактивны; Time scale и Reset chain работают через ноду независимо.

Env overrides:

- `VITE_API_URL` — нода (default `http://127.0.0.1:8001`)
- `VITE_TRAFFIC_URL` — traffic control API (default `http://127.0.0.1:8002`)
- `VITE_EXPLORER_URL` — explorer для внешних ссылок (default `http://127.0.0.1:5174`)
- `VITE_WS_URL` — WebSocket ноды для live-блоков в разделе Кошельки (default `ws://127.0.0.1:8001/ws`)

Сид-фразы локальных кошельков хранятся в `localStorage` (`volnix.traffic.wallets`) открытым текстом — только для стенда. Боты из traffic `GET /wallets` в браузер не копируются.

Контракт traffic API: [`../traffic/README.md`](../traffic/README.md); нода: [`../docs/API.md`](../docs/API.md).
