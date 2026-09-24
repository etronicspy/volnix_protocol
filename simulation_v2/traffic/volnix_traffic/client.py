"""HTTP client for simulation_v2 node explorer + operator APIs."""

from __future__ import annotations

from typing import Any, Dict, List, Optional, Union

import httpx


class NodeClient:
    def __init__(self, base_url: str, timeout: float = 30.0) -> None:
        self.base_url = base_url.rstrip("/")
        self._client = httpx.AsyncClient(base_url=self.base_url, timeout=timeout)

    async def aclose(self) -> None:
        await self._client.aclose()

    async def get(self, path: str, **params: Any) -> Dict[str, Any]:
        r = await self._client.get(path, params={k: v for k, v in params.items() if v is not None})
        r.raise_for_status()
        return r.json()

    async def post_json(
        self,
        path: str,
        body: Optional[Union[Dict[str, Any], List[Any]]] = None,
        **params: Any,
    ) -> Dict[str, Any]:
        r = await self._client.post(
            path,
            json=body if body is not None else {},
            params={k: v for k, v in params.items() if v is not None},
        )
        r.raise_for_status()
        return r.json()

    # --- explorer ---

    async def chain_summary(self) -> dict[str, Any]:
        return await self.get("/api/v1/chain/summary")

    async def params(self) -> dict[str, Any]:
        return await self.get("/api/v1/params")

    async def accounts(self, role: Optional[str] = None) -> list[dict[str, Any]]:
        data = await self.get("/api/v1/accounts", role=role)
        return list(data.get("accounts") or [])

    async def account(self, address: str) -> dict[str, Any]:
        return await self.get(f"/api/v1/accounts/{address}")

    async def orderbook(self, market: str = "ANT/WRT") -> dict[str, Any]:
        return await self.get("/api/v1/market/orderbook", market=market)

    async def unconfirmed(self) -> dict[str, Any]:
        return await self.get("/unconfirmed_txs")

    # --- operator ---

    async def create_account(self, seed: str) -> dict[str, Any]:
        return await self.post_json("/api/v1/operator/account", {"seed": seed})

    async def mint_wrt(self, address: str, amount: int) -> dict[str, Any]:
        return await self.post_json(
            "/api/v1/operator/mint",
            {"address": address, "denom": "uwrt", "amount": int(amount)},
        )

    async def verify(self, seed: str, desired_role: str, zkp_proof: str) -> dict[str, Any]:
        return await self.post_json(
            "/api/v1/operator/verify",
            {
                "seed": seed,
                "desired_role": desired_role,
                "zkp_proof": zkp_proof,
            },
        )

    async def declare(self, seed: str, b_i: int, s_i: int) -> dict[str, Any]:
        return await self.post_json(
            "/api/v1/operator/declare",
            {"seed": seed, "b_i": int(b_i), "s_i": int(s_i)},
        )

    async def place_order(
        self,
        seed: str,
        *,
        market: str,
        side: str,
        amount: int,
        price: int = 0,
        order_type: str = "LIMIT",
    ) -> dict[str, Any]:
        return await self.post_json(
            "/api/v1/operator/order",
            {
                "seed": seed,
                "market": market,
                "side": side,
                "order_type": order_type,
                "amount": int(amount),
                "price": int(price),
            },
        )

    async def signed_tx(self, seed: str, messages: list[dict[str, Any]]) -> dict[str, Any]:
        return await self.post_json("/api/v1/operator/tx", messages, seed=seed)

    async def produce(self, count: int = 1) -> dict[str, Any]:
        return await self.post_json("/api/v1/operator/produce", {"count": int(count)})
