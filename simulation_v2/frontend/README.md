# Volnix Explorer — simulation v2 frontend

Read-only blockchain explorer UI for the v2 simulator node (`simulation_v2/backend`).

Talks to explorer REST `/api/v1` and WebSocket `/ws` (default backend `http://127.0.0.1:8001`).
Stand controls (traffic bots, time scale, reset chain) live in a separate app:
[`../traffic-ui`](../traffic-ui/README.md) (port 5176); the header links to it.

## Run

```bash
# terminal 1 — node
cd ../backend && python3 main.py

# terminal 2 — traffic / economy (optional)
cd ../traffic && pip install -r requirements.txt && python3 main.py

# terminal 3 — UI
cd simulation_v2/frontend
npm install
npm run dev

# terminal 4 — stand control UI (optional)
cd ../traffic-ui && npm install && npm run dev
```

Open [http://127.0.0.1:5174](http://127.0.0.1:5174).

Env overrides:

- `VITE_API_URL` — REST base (default `http://127.0.0.1:8001`)
- `VITE_WS_URL` — WebSocket (default `ws://127.0.0.1:8001/ws`)
- `VITE_TRAFFIC_UI_URL` — stand control UI for the header link (default `http://127.0.0.1:5176`)

## Pages

| Route | Purpose |
|-------|---------|
| `/` | Chain overview, supplies, recent blocks/txs |
| `/blocks`, `/blocks/:height` | Full block tape (PoVB summary) + consensus world (commit, PoVB corridor, set diff, txs, events) |
| `/txs/:hash` | Transaction detail |
| `/wallets` | All simulation wallets: role, status, balances |
| `/accounts/:address` | Balances, orders, history |
| `/validators` | Current ValidatorSet |
| `/market` | ANT/WRT and LZN/WRT books + trades |
| `/epochs` | Epoch boundary records |

API contract: [`../docs/API.md`](../docs/API.md).
