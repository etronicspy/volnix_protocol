# Volnix Protocol — Whitepaper (English)

**English edition.** Russian: [WHITEPAPER_RU.md](./WHITEPAPER_RU.md).  
**Specification version:** 4.15 (aligned with [volnix_protocol.md](./volnix_protocol.md); full version-by-version changelog lives there).

---

## **Whitepaper: Volnix Protocol (H•P)**

**Version 4.15:** “Citizens” (DAO) / “Supplier” (ANT role); cross-references shortened (canonical detail in §4.1, §5, §6.1, §7.1).

### **Table of contents**
1. **Abstract**
2. **Introduction: Beyond the fairness trilemma**
3. **Foundational architecture principles**
   3.1. Singular verified identity (ZKP)  
   3.2. Role migration (“digital succession”)  
   3.3. Activity rules (“immune system”)
4. **Three-part economy: assets and roles**
   4.1. Assets: WRT, LZN, ANT  
   4.2. Roles: Guest, Supplier, Validator
5. **Economic engine: dual-loop model on the internal market**
   5.1. Loop 1: security and stability (validators’ passive income)  
   5.2. Loop 2: throughput and speed (active income and suppliers’ income)  
   5.3. Bridge: minimum obligation of activity (MOA)  
   5.4. ANT burn: LZN cap and fee share  
   5.5. ANT emission epoch: supplier balance reset and new issuance
6. **Consensus mechanism and network dynamics**
   6.1. CometBFT: proposer and fee distribution by ANT burn  
   6.2. Fixed block interval and halving (Bitcoin-style)
7. **Governance: constitutional protocol with bounded DAO**
   7.1. Constitutional layer (hard fork only)  
   7.2. DAO parameters (exhaustive list); **citizens** = WRT holders who vote
8. **Conclusion: a new paradigm**

---

### **1. Abstract**

Volnix Protocol is a sovereign Cosmos SDK blockchain for a fair, predictable economy with an **immutable monetary policy**. Its foundation is **Proof-of-Verified-Burn (PoVB)** and **ZKP**: mining rights attach to **verified** unique persons, mitigating Sybil attacks. **Roles:** **Guests**, **Suppliers**, **Validators**; **assets:** **WRT** (value and DAO vote), **LZN** (mining licence), **ANT** (**internal market** coin—circulation rules and **no direct transfers**, **§4.1**). **Two loops** and an **internal market:** state and execution on-chain; the reference client is the **Volnix wallet**; **chain, modules, and wallet** source is **open**. **CometBFT** orders blocks; the market is an **application layer** with **deterministic** matching (**§5**). **Consensus and proposer** follow **standard CometBFT** (**§6.1**). **Per-height fee collection** is **proportional to burned ANT** in **0…L** (**L** = activated **LZN**, **1 LZN = 1 ANT**); details and **B = 0** in **§5.4**. **Per-tx fees** are **flexible, market-driven** (**§5.2**). **MOA** (**§5.3**): **Supplier** places **sell ANT** orders; **validator** periodically **burns ANT**; windows set by **DAO**; on breach, role is removed, **ANT** burned, former validator’s **LZN remain**. **Block cadence is fixed**; **WRT halving** every **N** blocks (**§6.2**). **ANT issuance** is **epochal** to **Suppliers**, **without** a max-supply constant; scale is **indirect**—**LZN**, burn, coefficient (**§5.5**). **Bounded DAO** cannot change core tokenomics (**§7**). **Citizens** are **WRT holders who vote**; do **not** conflate with verified role **“Supplier”** (**§4.1–4.2**, **§7**).

### **2. Introduction: Beyond the fairness trilemma**

Today’s blockchains trade off security, decentralization, and scalability. Volnix adds a fourth axis—**fairness**—and refuses to treat it as optional. We hold that a truly decentralized system should rest not on anonymous capital or raw hash power alone, but on the **identity** of participants. Volnix is a digital social contract: rights and duties are explicit and enforced by code.

### **3. Foundational architecture principles**

#### **3.1. Singular verified identity (ZKP)**
The protocol enforces **one person—one verified role**. Using zero-knowledge proofs, users prove uniqueness via accredited providers without exposing personal data. At verification they choose one of two mutually exclusive roles: **Supplier** or **Validator**. If the **active Supplier cap** (§7.2 item 9) is **reached**, **Supplier** is **unavailable** for **new** verification until a slot frees (§4.2). This design blocks Sybil farming of verified accounts.

#### **3.2. Role migration (“digital succession”)**
To limit irreversible key loss, the protocol allows role recovery. After re-verification with ZKP, a user may migrate **Supplier** or **Validator** and linked frozen or accounted assets (e.g. activated LZN or protocol-tracked **ANT**) to a **new** wallet—**ANT moves only via protocol migration** (not a user `MsgSend`; see §4.1). Liquid assets (WRT, non-activated LZN) on the lost wallet stay inaccessible, preserving cryptographic integrity and incentivizing key hygiene.

#### **3.3. Activity rules (“immune system”)**
The economy is protected from “dead” verified roles through **MOA** (§5.3): criteria and durations are **not arbitrary** but **determined** by modules and **DAO** parameters (§7.2). On **MOA sanction**, the protocol **burns all ANT** tied to that role (free balance and order escrow per snapshot rules). **LZN are not confiscated:** a former **validator** keeps **LZN** on-wallet (including activated—no network seizure for third parties); only **validator rights** end (consensus and **Loop 1**). A former **Supplier**’s ZKP identifier may be freed for re-verification.

### **4. Three-part economy: assets and roles**

#### **4.1. Assets**
* **Wert (WRT): value and vote.** Primary network asset with fixed, Bitcoin-like issuance. Store of value, settlement on the internal market, and **the sole DAO voting instrument**. **Citizens** are **WRT holders** who participate in **DAO** (weight and quorum per governance module)—**political** participation, **not** the same as verified wallet role **Supplier** (§4.2).
* **Lizenz (LZN): licence / capacity.** Tradable token with fixed one-off issuance; “share” or “mining licence.” Activation yields a share of base network emission and caps a Validator’s mining power. **Activated LZN lock-up** (how long activated amount stays **locked** until normal deactivation/unlock per module) is a **DAO-tunable** parameter (§7) within code **min/max**.
* **Anteil (ANT): internal market coin.** **Tradable** in-protocol; the **only** venue is Volnix **internal market** (on-chain order book + **Volnix wallet** client). **Direct ANT transfers** (analogue of `MsgSend` / arbitrary address-to-address) are **forbidden:** ANT **changes hands** only on **order fills** on the internal market and **protocol service** flows—**emission** and **epoch reset** (§5.5), market **reserve and clearing**, validator **burn** (§5.4), **role migration** (§3.2). Generic third-party wallets **do not** support ANT custody/transfer outside this model given the asset’s economic and security role. **Supplier ANT emission is epoch-discrete** (§5.5): **at each emission epoch end** the protocol **burns all ANT** on **Supplier** accounts (free and order-reserved per **deterministic** epoch snapshot), then **credits new** ANT. Within an epoch **Suppliers** trade ANT for WRT. There is **no** separate fixed **max ANT supply** constant: scale is bounded **indirectly**—(1) validator **burn** capped by **LZN** (§5.4), (2) **new epoch issuance formula** and bounded **coefficient** (§5.5). Validators burn ANT **per block height** up to activated LZN for fee-share (see 5.4) and **MOA** (§5.3).

#### **4.2. Roles (wallet types)**
* **Type 1 — Guest:** Unverified wallet. Holds/trades WRT and LZN; **no ANT balance**, **no direct ANT transfers**. Entry point for all users.
* **Type 2 — Supplier:** Verified role of **supplying ANT** to the internal market (**not** synonymous with **citizen** in the DAO sense; **§4.1**, **§7**). Protocol-minted ANT is **sold** to validators for WRT and **burned** as **PoVB fuel** (fees, **MOA**); epoch issuance/reset—**§4.1**, **§5.5**. Within an epoch—**ANT seller** for WRT. **MOA** (§5.3): within **DAO** windows, **place sell-ANT orders**. **Max active Suppliers** is set by **DAO** (§7.2 item 9): while active **Supplier** count is **not below** that cap, **new** ZKP verification choosing **Supplier** is **closed**. If the cap is **lowered**, **existing** **Suppliers** **keep** status—no mass revocation; only **new** admissions freeze while active count **≥** new cap; slots free via **Supplier MOA** (§5.3) and other protocol exits. **Per-epoch ANT accumulation cap** for a Supplier (if used) is **DAO** (§7.2 item 4). **No** carry of ANT across epochs—remainder burns at epoch boundary.
* **Type 3 — Validator:** Verified role. Activates LZN (≤33% per wallet) and **buys** ANT for consensus participation. Each height, a Validator **chooses** burn **b** in **0 … activated LZN** (same denomination as LZN); **b** sets **fee share** for that height (see 5.4). **MOA** (§5.3): within **DAO** window, **record ANT burn** on-chain (at least one protocol-counted burn in the window). Block **proposer** follows **CometBFT** by consensus weight (see 6.1). On MOA loss, **LZN remain** on-wallet (§3.3).

### **5. Economic engine: dual-loop model on the internal market**

The economy splits into two linked loops. The **internal market** has **two layers:** (1) **chain**—modules/state for orders, execution, settlement; (2) **Volnix wallet**—reference **open-source** client for order-book UX and signing market txs. Anyone may audit the wallet or build **alternative** clients to the same on-chain API; the reference stack has no black boxes.

**CometBFT / Cosmos SDK fit.** CometBFT orders txs and agrees state; the **market** does not alter consensus (**§6.1**). Orders execute in **app modules** as txs land in blocks. **Determinism:** given fixed tx order in a block, matching is **identical** on all nodes; tie-breaks (price, time/block index) are implementation-defined.

**ANT transfers between users** use **only** the internal market; direct **MsgSend** and **bank**/IBC bypass for ANT **denom** are **rejected** by the app. Service flows—**§4.1**.

#### **5.1. Loop 1: security and stability (validators’ passive income)**
Validators activate LZN, committing capital to long-run security. The protocol pays **base block reward (WRT emission)** proportional to each Validator’s share of total activated LZN—stable, predictable income.

#### **5.2. Loop 2: throughput and speed (active income and suppliers’ income)**
On-chain **order book** and **Volnix wallet**—see §5 intro. Market txs enter blocks **like any other**: the **proposer** orders by **gas/fees** and block limits (**§6.1**; **§7.2** items 7–8).
* **Supply:** During an **ANT emission epoch**, **Suppliers** post limit or market **sell ANT** orders (until epoch end when balances reset—§5.5).
* **Demand:** Validators buy ANT to burn **per height** (within LZN cap) for **fee share** and **MOA** (§5.3).
* **Flexible fees (Bitcoin-style):** senders set fees; validators/mempool prioritize inclusion (higher bid wins ties). This yields a **floating market block fee**, not a protocol-fixed surcharge.
* **Redistribution:** Fills move **WRT** to **Suppliers**; validators **burn** **ANT** per height. **Block fee split**—**§5.4**. **F** is driven by **block-space market** (Bitcoin spirit).

#### **5.3. Bridge: minimum obligation of activity (MOA)**
**MOA** ties **Supplier** and **Validator** status to the **internal market** and **ANT burn**. Observation windows (**T_g** for **Supplier**, **T_v** for validator) are set by **DAO** (§7.2 items 2–3) within code **min/max**. Reference **genesis** may use **1 year** / **6 months** for **T_g** / **T_v** if so configured. **Last-event time** for a new role before first order/burn is set **unambiguously** by the module (e.g. status grant time) so **T_g** / **T_v** checks are **deterministic**.

**(1) Supplier MOA.** The protocol stores the last **counted** **sell-ANT order** on the internal market (message type and “sell ANT” criterion fixed in the market module). If **now − last ≥ T_g**, status **Supplier** is **revoked**. **All Supplier ANT** (free + order **escrow**) **burns**; ZKP id may be freed for re-verification.

**(2) Validator MOA.** The protocol stores the last **counted** **ANT burn** (§5.4 rules, app state). If **now − last ≥ T_v**, **validator** status is **revoked** (consensus and **Loop 1**). **LZN stay** with the owner (**not** confiscated). **All ANT** tied to validator role at sanction **burns**.

**(3) Loop 2 and MOA.** Share in **F** at a height follows **§5.4** only. **MOA** does **not** require burn **every block**: **T_v** and **T_g** are **sliding** windows ((1)–(2)), formalized in-module.

**(4) Link to Loop 1.** **Base WRT emission** goes only to active **validators**; on (2) revocation, Loop 1 share **ends** while **LZN remain** (§3.3).

#### **5.4. ANT burn: LZN cap and fee share**
Each **height**, each Validator may burn **b** with **0 ≤ b ≤ L**, **L** = **activated LZN** in the same denomination (**1 LZN = 1 ANT** at minimum unit). The validator **chooses** **b** (including **b = L** or **b = 0**). ANT is bought on the internal market from **Suppliers** and destroyed. Let **F** = total tx fees in the block at that height, **B = Σ b_i** over validators recording burn at that height. Validator **i**’s share of **F** is **F · (b_i / B)** when **B > 0**. When **B = 0**, **F** is **not** split among validators by burn rule (destination—**community pool**, roll to later heights, or other **deterministic** **DAO** policy, §7.2 item 6; constitution only records **no** proportional split on zero burn). **b_i** fixation is at **app** (Cosmos SDK) layer, consistent with CometBFT. **No** extra fee **bonuses** beyond **F · (b_i / B)**.

#### **5.5. ANT emission epoch: supplier reset and new issuance**
An **ANT emission epoch** is a calendar period ending with a **full** refresh of Supplier ANT supply. In the reference implementation, epoch length matches the burn window below (**one calendar week**); if changed, it must be **explicit** and **deterministic** in code.

**Step 1 — Supplier reset.** At epoch end the protocol **burns all ANT** credited to **active Suppliers**: free balances and market **escrow** (open orders handled **deterministically** in the end-epoch tx—cancel, partial fill, or other fixed rule, identical on all nodes). **No** carry of Supplier ANT across epochs.

**Step 2 — new issuance.** **Immediately after** reset, the protocol **credits** **Suppliers** with new ANT. **Total** epoch emission = (ANT burned by validators and other protocol rules in the **prior** epoch) × **distribution coefficient**. The coefficient updates from burn dynamics: ratio = burn_current_epoch / burn_previous_epoch; new_coefficient = old_coefficient / ratio, clamped each step to **DAO** lower/upper bounds (§7.2 item 5); reference genesis bounds **0.75–1.5**, initial coefficient **1**. New ANT is split **evenly** across **active Suppliers** (or per module-fixed rule). Consensus emits a **special block** (or atomic phase at epoch boundary) performing **reset and mint**.

**Indirect emission cap.** There is **no** “max supply ANT” constant. Upper **flow** to **Suppliers** tracks **prior-epoch burn** and the **coefficient**; burn intensity is capped by aggregate **activated LZN** (§5.4). **LZN** and the **epoch coefficient** thus bound ANT supply **without** a fixed monetary ceiling.

### **6. Consensus mechanism and network dynamics**

#### **6.1. CometBFT: proposer and fee distribution by ANT burn**
**Proposer** and **consensus** are **standard CometBFT**: **round-robin** by weights in `ValidatorSet` (staking / Cosmos SDK; CometBFT core is **not** customized for Volnix). The proposer includes market and other txs per **§5.2** and **§7.2** items 7–8. **Per-height fee split** is **app rule** **§5.4**. The market is part of **one** app state (**§5**).

#### **6.2. Fixed block interval and halving (Bitcoin-style)**
* **Block cadence.** **Target** inter-block interval is **fixed** by consensus/app (predictable “clock” like **Bitcoin**). Cadence is **not** tied to ANT burn or “economic speed”—separate from WRT/LZN/ANT policy. CometBFT timeouts/commit tuning targets the network goal.
* **Halving.** **Base block reward** (WRT in Loop 1) **halves** strictly **every N blocks**, Bitcoin-style by **height**. With fixed mean interval, halving dates are **calendar-predictable**, reinforcing **immutable** policy and familiar participant expectations.

### **7. Governance: constitutional protocol with bounded DAO**

Volnix uses a strict rule stack. **Citizens** are **DAO voters**; voting power belongs **only** to **WRT holders** (weighting per governance module). Adopted proposals take effect only after a **long timelock**. DAO parameters have code **min/max**; votes cannot set values outside them.

#### **7.1. Constitutional layer (no vote—hard fork only)**

* **WRT & LZN tokenomics:** total issuance, **halving schedule** (including **every N blocks** for WRT), immutability of base monetary policy for these assets.
* **PoVB & ANT structure:** **ANT burn**, **0 ≤ b ≤ L**, **1 LZN = 1 ANT**, **F** split when **B > 0**—**§5.4**; **ANT** only via **internal market**—**§4.1**, **§5**; **Supplier emission epoch**—**§5.5**; **MOA** (**T_g**, **T_v**), sanctions, **LZN** not seized—**§5.3**, **§3.3**.
* **Invariant at B = 0:** **F** is **not** split among validators **proportionally** to “zero burn”; **where** **F** goes is not constitution but **DAO** item 6 below.
* **Consensus:** proposer choice and block order—**standard CometBFT** (§6.1).
* **No** constitutional “hard max supply ANT”; ANT scale is **indirect** (§5.4–5.5).

#### **7.2. Legislative layer — DAO-tunable parameters (reference exhaustive list)**

1. **Activated LZN lock-up duration** after activation (and module-aligned unlock rules); §4.1, §5.1.
2. **Supplier MOA: T_g** — max time without **sell-ANT order** on internal market before **Supplier** revocation; §5.3.
3. **Validator MOA: T_v** — max time without **counted ANT burn** before validator revocation; §5.3.
4. **Per-epoch ANT accumulation cap** for a Supplier (if enabled); §4.2, §5.5.
5. **Lower and upper bounds** on the dynamic ANT **epoch emission coefficient** (reference genesis **0.75–1.5**); coefficient updates per §5.5 inside bounds.
6. **Policy for fees F when B = 0** — community pool, roll-forward, or other **deterministic** variant per module under DAO; §5.4.
7. **Max gas per block** (`BlockGasLimit` / Cosmos equivalent); §5.2, §6.1.
8. **Max block size in bytes.**
9. **Active Supplier count cap** — upper bound on concurrent **Supplier** slots; §4.2. While **active Suppliers ≥ cap**, **new** **Supplier** status is **not** granted (ZKP with **Supplier** for a new slot). If DAO **lowers** the cap, **existing** **Suppliers** are **not** auto-revoked for “being over cap”; only **new** **Supplier** grants freeze until active count is **strictly below** the new cap (including via **MOA**, role migration, other deterministic exits).

Items **7–8** do not change WRT/LZN tokenomics or replace **burn** economics; they set **operational** chain limits. Item **9** caps **Supplier** role capacity. **Hard max supply ANT** is **not** introduced via DAO.

### **8. Conclusion**

Volnix unifies **identity** (**ZKP**), **markets**, and **consensus**: security through personal accountability, fairness through role separation, efficiency through a **built-in** market without extra intermediaries. **ANT**, **PoVB**, **fee split**, and **Supplier epoch emission**—**§4.1**, **§5.4–5.5**; **CometBFT**, **block cadence**, **WRT halving**—**§6**; **DAO**, **citizens** (**WRT**), **MOA** (**T_g** / **T_v**)—**§7**, **§5.3**, **§7.2**. **Open** reference stack—**§5**. Immutable **base** tokenomics with **bounded** DAO adaptation balances **reliability**, **performance**, and **decentralization**.

---

*End of whitepaper body (v4.15). For Russian text see [WHITEPAPER_RU.md](./WHITEPAPER_RU.md). For detailed per-version change notes see [volnix_protocol.md](./volnix_protocol.md).*
