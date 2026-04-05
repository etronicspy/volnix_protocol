# Volnix Protocol — Whitepaper (English)

**English edition.** Russian: [WHITEPAPER_RU.md](./WHITEPAPER_RU.md).  
**Specification version:** 4.20 (aligned with [volnix_protocol.md](./volnix_protocol.md); detailed per-version changelog there).

---

## **Whitepaper: Volnix Protocol (H•P)**

**Version 4.20:** §4.2 Type 1 **Guest** renamed **Citizen**; WRT-based governance uses **DAO voters** (distinct from wallet role **Citizen**). **(v4.19)** Genesis: two bootstrap wallets (Supplier + Validator, no ZKP), no Supervisor role; same MOA as for ZKP roles; genesis ANT = EpochBlocks × L_total

### **Table of contents**
1. **Abstract**
2. **Introduction: Beyond the fairness trilemma**
3. **Foundational architecture principles**
   3.1. Singular verified identity (ZKP)  
   3.2. Role migration (“digital succession”)  
   3.3. Activity rules (“immune system”)
4. **Three-part economy: assets and roles**
   4.1. Assets: WRT, LZN, ANT  
   4.2. Roles: Citizen, Supplier, Validator
5. **Economic engine: dual-loop model on the internal market**
   5.1. Loop 1: security and stability (validator income)  
   5.2. Loop 2: throughput and speed (active income and suppliers’ income)  
   5.3. Bridge: minimum obligation of activity (MOA)  
   5.4. ANT burn and participation stake: consensus weight and fee share  
   5.5. ANT emission epoch: supplier balance reset and new issuance
6. **Consensus mechanism and network dynamics**
   6.1. CometBFT: proposer, stake weight, and fee distribution  
   6.2. Fixed block interval and halving (Bitcoin-style)  
   6.3. Genesis block: two bootstrap wallets (no ZKP)
7. **Governance: constitutional protocol with bounded DAO**
   7.1. Constitutional layer (hard fork only)  
   7.2. DAO parameters (exhaustive list); **DAO voters** = WRT holders who vote
8. **Conclusion: a new paradigm**

---

### **1. Abstract**

Volnix Protocol is a sovereign Cosmos SDK blockchain for a fair, predictable economy with an **immutable monetary policy**. Its foundation is **Proof-of-Verified-Burn (PoVB)** and **ZKP**: mining rights attach to **verified** unique persons, mitigating Sybil attacks—**except** the **two** fixed genesis bootstrap addresses granted **Supplier** and **Validator** **without** ZKP (**§6.3**). **Roles (§4.2):** **Citizen**, **Supplier**, **Validator**; **assets:** **WRT** (value and DAO vote), **LZN** (mining licence), **ANT** (**internal market** coin—circulation rules and **no direct transfers**, **§4.1**). **Two loops** and an **internal market:** state and execution on-chain; the reference client is the **Volnix wallet**; **chain, modules, and wallet** source is **open**. **CometBFT** orders blocks; the market is an **application layer** with **deterministic** matching (**§5**). **Consensus and proposer** follow **standard CometBFT** (**§6.1**). At each height a validator declares **two numbers**: **b_i** (ANT volume for **burn**) and **s_i** (ANT as **participation stake**); **b_i + s_i ≤ L_i**; on a successful block **both** amounts are **burned irrevocably**; **consensus weight** = **s_i / L_i** (**§5.4**, **§6.1**). The **EndBlocker** of block N builds the `ValidatorSet` for block N+1: culling under global cap **λ** (genesis **1/3**), minimum threshold **Σ b_i ≥ λ × L_total**, **at most K** signers with the **largest w_i** (genesis **K = 150**, **DAO**, **§5.4**). **ANT emission epoch length** and other **periods** are counted in **consensus blocks** and set/changed via **WRT voting** (**§5.5**, **§6.2**). If **b_i = 0**, the validator receives **neither** base WRT reward **nor** a share of fees (**§5.1**, **§5.4**). **Per-height fee collection** is **proportional to b_i**. **Per-tx fees** are **flexible, market-driven** (**§5.2**). **MOA** (**§5.3**): **any signed transaction** from the address within a sliding window; durations—**DAO**; on breach the role is removed, **ANT** burned, a former validator’s **LZN remain**. **Block cadence is fixed**; **WRT halving** every **N** blocks (**§6.2**). **ANT issuance** is an **epochal** cycle to **Suppliers**, **without** a max-supply constant; scale is **indirect**—**LZN**, burn, coefficient (**§5.5**). The **genesis block** (**§6.3**): **two** bootstrap wallets—**Supplier** and **Validator**, **no ZKP**—**10,000 LZN** on the validator wallet (one activated), **10,080 ANT** initial emission to the supplier wallet. **Bounded DAO** cannot change core tokenomics (**§7**). **DAO voters** are **WRT holders who vote**; do **not** conflate wallet roles **Citizen** (Type 1) or **Supplier** with DAO participation (**§4.1–4.2**, **§7**).

### **2. Introduction: Beyond the fairness trilemma**

Today’s blockchains trade off security, decentralization, and scalability. Volnix adds a fourth axis—**fairness**—and refuses to treat it as optional. We hold that a truly decentralized system should rest not on anonymous capital or raw hash power alone, but on the **identity** of participants. Volnix is a digital social contract: rights and duties are explicit and enforced by code.

### **3. Foundational architecture principles**

#### **3.1. Singular verified identity (ZKP)**
The protocol enforces **one person—one verified role**. Using zero-knowledge proofs, users prove uniqueness via accredited providers without exposing personal data. At verification they choose one of two mutually exclusive roles: **Supplier** or **Validator**. If the **active Supplier cap** (§7.2 item 8) is **reached**, **Supplier** is **unavailable** for **new** verification until a slot frees (§4.2). This design blocks Sybil farming of verified accounts.

#### **3.2. Role migration (“digital succession”)**
To limit irreversible key loss, the protocol allows role recovery. After re-verification with ZKP, a user may migrate **Supplier** or **Validator** and linked frozen or accounted assets (e.g. activated LZN or protocol-tracked **ANT**) to a **new** wallet—**ANT moves only via protocol migration** (not a user `MsgSend`; see §4.1). Liquid assets (WRT, non-activated LZN) on the lost wallet stay inaccessible, preserving cryptographic integrity and incentivizing key hygiene.

#### **3.3. Activity rules (“immune system”)**
The economy is protected from “dead” verified roles through **MOA** (§5.3): criteria and durations are **not arbitrary** but **determined** by modules and **DAO** parameters (§7.2). On **MOA sanction**, the protocol **burns all ANT** tied to that role (free balance and order escrow per snapshot rules). **LZN are not confiscated:** a former **validator** keeps **LZN** on-wallet (including activated—no network seizure for third parties); only **validator rights** end (consensus and **Loop 1**). A former **Supplier**’s ZKP identifier may be freed for re-verification **when** that Supplier was ZKP-bound; the genesis bootstrap **Supplier** (**§6.3**) has **no** ZKP link—MOA only ends **Supplier** status (nothing to “release”).

### **4. Three-part economy: assets and roles**

#### **4.1. Assets**
* **Wert (WRT): value and vote.** Primary network asset with fixed, Bitcoin-like issuance. Store of value, settlement on the internal market, and **the sole DAO voting instrument**. **DAO voters** are **WRT holders** who participate in **governance** (weight and quorum per governance module)—**political** participation, **not** the same as wallet roles **Supplier** or **Citizen** (§4.2).
* **Lizenz (LZN): licence / capacity.** Tradable token with fixed one-off issuance; “share” or “mining licence.” Activation sets a Validator’s **maximum mining power**—the upper cap **b_i + s_i** per height (§5.4). Activation **alone** yields **no** income: base WRT reward and fee share require **burning ANT** (**b_i > 0**, §5.1, §5.4). **Activated LZN lock-up** (how long activated amount stays **locked** until normal deactivation/unlock per module) is a **DAO-tunable** parameter (§7) within code **min/max**.
* **Anteil (ANT): internal market coin.** **Tradable** in-protocol; the **only** venue is Volnix **internal market** (on-chain order book + **Volnix wallet** client). **Direct ANT transfers** (analogue of `MsgSend` / arbitrary address-to-address) are **forbidden:** ANT **changes hands** only on **order fills** on the internal market and **protocol service** flows—**emission** and **epoch reset** (§5.5), market **reserve and clearing**, validator **burn** (§5.4), **role migration** (§3.2). Generic third-party wallets **do not** support ANT custody/transfer outside this model given the asset’s economic and security role. **Supplier ANT emission is epoch-discrete** (§5.5): **at each emission epoch end** the protocol **burns all ANT** on **Supplier** accounts (free and order-reserved per **deterministic** epoch snapshot), then **credits new** ANT. Within an epoch **Suppliers** trade ANT for WRT. There is **no** separate fixed **max ANT supply** constant: scale is bounded **indirectly**—(1) validator **burn** capped by **LZN** (§5.4), (2) **new epoch issuance formula** and bounded **coefficient** (§5.5). Validators burn ANT **per block height** up to activated LZN for fee-share (see 5.4) and **MOA** (§5.3).

#### **4.2. Roles (wallet types)**
* **Type 1 — Citizen:** Unverified wallet. Holds/trades WRT and LZN; **no ANT balance**, **no direct ANT transfers**. Entry point for all users.
* **Type 2 — Supplier:** Verified role of **supplying ANT** to the internal market (**not** the same as **DAO voter** status as a **WRT holder**; **§4.1**, **§7**). Protocol-minted ANT is **sold** to validators for WRT and **burned** as **PoVB fuel** (fees, **MOA**); epoch issuance/reset—**§4.1**, **§5.5**. Within an epoch—**ANT seller** for WRT. **MOA** (§5.3): within **DAO** windows, **any signed transaction** from the address. **Max active Suppliers** is set by **DAO** (§7.2 item 8): while active **Supplier** count is **not below** that cap, **new** ZKP verification choosing **Supplier** is **closed**. If the cap is **lowered**, **existing** **Suppliers** **keep** status—no mass revocation; only **new** admissions freeze while active count **≥** new cap; slots free via **Supplier MOA** (§5.3) and other protocol exits. **Per-epoch ANT accumulation cap** for a Supplier (if used) is **DAO** (§7.2 item 4). **No** carry of ANT across epochs—remainder burns at epoch boundary.
* **Type 3 — Validator:** Verified role. Activates LZN (≤33% per wallet) and **buys** ANT for consensus. Each block height the Validator declares **two numbers** (**§5.4**): **(1) b_i**—ANT for **burn**; **(2) s_i**—ANT as **participation stake** in consensus. Both obey **b_i + s_i ≤ L_i** (**L_i** = activated LZN, **1 LZN = 1 ANT**); on a successful block **both** are **burned irrevocably**. **Weight** in CometBFT `ValidatorSet` is **stake share**: **w_i = s_i / L_i** (**§6.1**). If **b_i = 0**, the validator receives **neither** base WRT (**§5.1**) **nor** fee share (**§5.4**) for that height. **MOA** (§5.3): within **DAO** window, **at least one signed transaction** from the address (any on-chain record). Block **proposer** follows **standard CometBFT** by **w_i** (see 6.1). On MOA loss, **LZN remain** on-wallet (§3.3).

**Genesis bootstrap (§6.3).** Two fixed genesis addresses hold **Supplier** and **Validator** **without** ZKP. There is **no** separate **Supervisor** role. **MOA** applies to both (**§5.3**).

### **5. Economic engine: dual-loop model on the internal market**

The economy splits into two linked loops. The **internal market** has **two layers:** (1) **chain**—modules/state for orders, execution, settlement; (2) **Volnix wallet**—reference **open-source** client for order-book UX and signing market txs. Anyone may audit the wallet or build **alternative** clients to the same on-chain API; the reference stack has no black boxes.

**CometBFT / Cosmos SDK fit.** CometBFT orders txs and agrees state; the **market** does not alter consensus (**§6.1**). Orders execute in **app modules** as txs land in blocks. **Determinism:** given fixed tx order in a block, matching is **identical** on all nodes; tie-breaks (price, time/block index) are implementation-defined.

**ANT transfers between users** use **only** the internal market; direct **MsgSend** and **bank**/IBC bypass for ANT **denom** are **rejected** by the app. Service flows—**§4.1**.

#### **5.1. Loop 1: security and stability (validator income)**
Validators activate LZN, committing capital to long-run security. The protocol distributes **base block reward (WRT emission)** proportional to each Validator’s share of total activated LZN, **but only among validators** that recorded **ANT burn** at that height (**b_i > 0**, §5.4). A validator with **b_i = 0** receives **no** share of base WRT for that height **regardless** of activated LZN. Thus activated LZN without ANT burn **earns nothing**—passive LZN without PoVB participation is **ruled out**.

#### **5.2. Loop 2: throughput and speed (active income and suppliers’ income)**
On-chain **order book** and **Volnix wallet**—see §5 intro. Market txs enter blocks **like any other**: the **proposer** orders by **gas/fees** and block limits (**§6.1**; **§7.2** items 6–7).
* **Supply:** During an **ANT emission epoch**, **Suppliers** post limit or market **sell ANT** orders (until epoch end when balances reset—§5.5).
* **Demand:** Validators buy ANT to burn **per height** (within LZN cap) for **fee share** and **MOA** (§5.3).
* **Flexible fees (Bitcoin-style):** senders set fees; validators/mempool prioritize inclusion (higher bid wins ties). This yields a **floating market block fee**, not a protocol-fixed surcharge.
* **Redistribution:** Fills move **WRT** to **Suppliers**; validators **burn** **ANT** per height. **Block fee split**—**§5.4**. **F** is driven by **block-space market** (Bitcoin spirit).

#### **5.3. Bridge: minimum obligation of activity (MOA)**
**MOA** ties **Supplier** and **Validator** to **real presence** on-chain. The protocol has **no** separate **Supervisor** role. **Genesis** uses **two** fixed addresses—one **Supplier**, one **Validator**—each granted its status **without** ZKP (**§6.3**); they are ordinary **Supplier** / **Validator** statuses on **distinct** wallets and are subject to the **same** MOA rules below as ZKP-granted roles (they **can** lose status and face the same sanctions).

The activity criterion is **unified**: **any signed transaction** from the address, recorded on-chain, counts as presence. **Minimum activity windows** (**T_g** for **Supplier**, **T_v** for validator) are set by **DAO** (§7.2 items 2–3) within code **min/max**. Reference **genesis** may use **1 year** / **6 months** for **T_g** / **T_v**. **Last-event time** for a new role before the first tx is set **unambiguously** by the module (e.g. status grant time) so checks are **deterministic** on all nodes.

**(1) Supplier MOA.** The protocol stores the time (height) of the last **signed transaction** from the **Supplier** address. If **now − last ≥ T_g**, **Supplier** status is **revoked**. **All Supplier ANT** (free + order **escrow**) **burns**. The ZKP identifier may be freed for re-verification **if** the Supplier was ZKP-bound; the genesis bootstrap Supplier (**§6.3**) has **no** ZKP binding—only **Supplier** status ends.

**(2) Validator MOA.** The protocol stores the time (height) of the last **signed transaction** from the **validator** address. If **now − last ≥ T_v**, **validator** status is **revoked** (consensus and **Loop 1**). **LZN stay** with the owner (**not** confiscated). **All ANT** tied to the validator role **burns**.

**(3) MOA and per-block participation.** MOA does **not** require action **every block**: **T_v** and **T_g** are **sliding** windows ((1)–(2)). Share in block fees **F** at a height follows **§5.4** (**b_i**, **s_i**); MOA governs **long-run** presence only.

**(4) Link to Loop 1.** **Base WRT emission** goes only to active **validators**; on (2) revocation Loop 1 share **ends** while **LZN remain** (§3.3).

#### **5.4. ANT burn and participation stake: consensus weight and fee share**

**Declaration and culling (CometBFT-compatible).** The validator sends **`MsgDeclareParticipation{b_i, s_i}`** in block **N**. **EndBlocker** of N collects declarations, applies constraints, global cap **λ**, and signer cap **K**, and returns to CometBFT an **updated `ValidatorSet`** for block **N+1**—only validators that pass culling **sign** N+1. This **physically bounds** signer count (O(n²) mitigation). On N+1 the declared **b_i** and **s_i** are **fixed and burned**—**both** amounts destroyed; rewards and fees are distributed among participants.

**Top-K by weight (K).** Among validators with valid declarations (**b_i + s_i ≤ L_i**) and other EndBlocker checks, **rank by descending** **w_i = s_i / L_i**. The active `ValidatorSet` for the next block includes **at most K** validators with the **largest** **w_i** (ties broken **deterministically**, e.g. by address/key). Genesis reference **K = 150** keeps consensus load manageable. **K** is **DAO**-tunable (§7.2 item 10) within code **min/max**. **λ** limits and minimum **Σ b_i** apply **coherently** with top-K selection (**one deterministic** step order in implementation—identical on all nodes).

Each **height**, each Validator declares **two numbers**:

**Joint cap: b_i + s_i ≤ L_i.** Burn plus stake **cannot exceed** activated LZN (**L_i**, **1 LZN = 1 ANT**). Both draw from one ANT pool—a **direct trade-off**: more burn leaves less for stake (and vice versa).

**(1) b_i — ANT burn volume.** Range **0 ≤ b_i ≤ L_i** (subject to the joint cap). The validator **chooses** **b_i**. If **b_i = 0**, **no** fee share **nor** base WRT (**§5.1**) for that height.

**(2) s_i — participation stake (ANT).** ANT amount setting **consensus weight** in CometBFT `ValidatorSet`: **w_i = s_i / L_i**. Range **0 ≤ s_i ≤ L_i − b_i**. A **larger** fraction of LZN staked means **higher** consensus weight. On a successful block the stake **burns** like **b_i**—**b_i + s_i** is destroyed.

**Fee split.** Let **F** = total tx fees at that height, **B = Σ b_i** over participants. Validator **i**’s share of **F** is **F · (b_i / B)**.

**Global burn cap (λ) and minimum threshold.** Total burn per block: **Σ b_i ≤ λ × L_total**, **L_total = Σ L_i** aggregate activated LZN, **λ** (**BurnCapLambda**) set by **DAO** (§7.2 item 9; genesis **λ = 1/3**). If proposed burn **exceeds** the cap, validators are **sorted by w_i ascending**; **lowest-weight** validators are **excluded** from the next `ValidatorSet` (via EndBlocker `ValidatorUpdates`) until **Σ b_i of the remainder ≤ λ × L_total**. Then the **K** cap applies (**§5.4** above). Excluded validators’ **ANT is not lost**—it stays on wallets.

**Minimum burn threshold** to form a block: **Σ b_i ≥ λ × L_total**. If EndBlocker **cannot** reach it, the set for the next block **is not updated**—EndBlocker **runs again** on the next height; validators must **coordinate** sufficient aggregate burn. Validator **ANT is perpetual** (epoch reset in §5.5 applies only to **Suppliers**).

**Two-number trade-off.** With **b_i + s_i ≤ L_i** and **both** burned each successful block, the validator **loses** all declared ANT every time. **b_i** drives **income rights** (base WRT and fees); **s_i** drives **consensus weight** (proposer, votes, λ culling order, top-K). Global **λ** and cap **K** create **competition**: over the λ cap, **low-weight** validators drop out; **K** caps signers with highest **w_i** (**§7.2 item 10**). Large LZN does **not** auto-dominate—**stake fraction** does. **No** extra fee **bonuses** beyond **F · (b_i / B)**.

#### **5.5. ANT emission epoch: supplier reset and new issuance**
An **ANT emission epoch** is measured in **consensus blocks** (**height**): after **EpochBlocks** since the last boundary, a **full** refresh of Supplier ANT supply runs. **EpochBlocks** is a **DAO** parameter (§7.2 item 11), **proposed and approved** by WRT holders; genesis reference can follow target block time (e.g. **1 block/min** and “one week” → **EpochBlocks = 60 × 24 × 7 = 10 080**). “Week” is only a **guide** for choosing initial **EpochBlocks**; the protocol’s source of truth is the **block counter**. **Epoch reset affects Suppliers only**—validator **ANT is perpetual** and **not** touched at epoch edges.

**Step 1 — Supplier reset.** At epoch end the protocol **burns all ANT** credited to **active Suppliers**: free balances and market **escrow** (open orders handled **deterministically** in the end-epoch tx—cancel, partial fill, or other fixed rule, identical on all nodes). **No** carry of Supplier ANT across epochs.

**Step 2 — new issuance.** **Immediately after** reset, the protocol **credits** **Suppliers** with new ANT. **Total** epoch emission = (**ANT volume sold by Suppliers** on the internal market in the **prior** epoch—i.e. filled sell orders) × **distribution coefficient**. The coefficient updates from sales dynamics: ratio = sold_current_epoch / sold_previous_epoch; new_coefficient = old_coefficient / ratio, clamped each step to **DAO** bounds (§7.2 item 5); reference genesis **0.75–1.5**, initial coefficient **1**. New ANT is split **evenly** across **active Suppliers** (or per module-fixed rule). Consensus emits a **special block** (or atomic phase at epoch boundary) performing **reset and mint**.

**Maximum ANT destroyed per epoch (planning ceiling).** With **1 LZN = 1 ANT** and **b_i + s_i ≤ L_i** each block, an **upper bound** on total ANT the network **could** destroy in one epoch (if **every** block fully “spends” activated LZN as **b_i + s_i**) is:

**ANT_max_epoch = EpochBlocks × L_total**,

**EpochBlocks** in **blocks** (§7.2 item 11), **L_total = Σ L_i** aggregate activated LZN at the epoch boundary. This is **not** actual burn but a **deterministic ceiling** for planning Supplier ANT supply.

**First epoch (genesis).** At launch **sold_previous = 0**—the trade-based formula does not apply. Initial Supplier ANT is **`ANT_genesis = EpochBlocks × L_total_genesis`**, **L_total_genesis** = aggregate activated LZN in genesis (§6.3). With **EpochBlocks = 10 080** and **L_total_genesis = 1** (one LZN activated on the genesis **Validator** wallet): **`ANT_genesis = 10,080 ANT`**. Credited to the genesis **Supplier** wallet (§6.3); covers the **ceiling** of validator demand for the first epoch at those parameters. From the **second** epoch, standard mechanics apply (trade volume × coefficient).

**Indirect emission cap.** There is **no** “max supply ANT” constant. Upper **flow** to **Suppliers** tracks **prior-epoch ANT sales** and the **coefficient**; sales are constrained by validator **demand** and Supplier **supply** (itself bounded by prior emission). **Market demand**, **LZN**, **λ** (§5.4), and the **epoch coefficient** bound ANT supply **without** a fixed monetary ceiling.

### **6. Consensus mechanism and network dynamics**

#### **6.1. CometBFT: proposer, stake weight, and fee distribution**
**Proposer** and **consensus** are **standard CometBFT**: **round-robin** by weights in `ValidatorSet`; CometBFT core is **not** customized for Volnix. Validator **i**’s **weight** in `ValidatorSet` is **participation stake**: **w_i = s_i / L_i**, **s_i** from **§5.4**, **L_i** activated LZN. Thus **dominance** follows **not** absolute LZN but **stake fraction**—**curbing** concentration at large validators. The proposer includes market and other txs per **§5.2** and **§7.2** items 6–7. **Per-height fee split** is **app rule** **§5.4**. The market is part of **one** app state (**§5**).

#### **6.2. Fixed block interval and halving (Bitcoin-style)**
* **Block cadence.** **Target** inter-block interval is **fixed** by consensus/app (predictable “clock” like **Bitcoin**). Cadence is **not** tied to ANT burn or “economic speed”—separate from WRT/LZN/ANT policy.
* **Adaptive timeouts per height / round.** Wall-clock time to finalize (CometBFT rounds per height, network delay) **varies**. Nodes **after each** committed block (and between rounds on the same height if needed) **deterministically** adjust CometBFT timeout parameters (e.g. propose / prevote / precommit / commit and aligned node config), targeting **BaseBlockTime** and related app consensus parameters (within code and optional **DAO** bounds). The goal is **mean** inter-block time for **successful** blocks **converges** to the target despite delay (including when **EndBlocker** temporarily skips `ValidatorSet` updates for insufficient burn, §5.4, and extra rounds occur). Thus **height-based halving** and **EpochBlocks** stay unambiguous; calendar predictability is a **derivative** of stable mean block time.
* **Halving and other periods—in blocks and via DAO.** **Base block reward** (WRT in Loop 1) **halves** every **N** blocks (**N = HalvingInterval**), **N** a **consensus** block count, **not** calendar days. Concrete **N** is **proposed and approved** by WRT holders through **DAO** within code **min/max** (§7.2 item 12). Likewise **ANT emission epoch length** is **EpochBlocks** (**§5.5**, §7.2 item 11). Key **periods** are **block-anchored** and transparently **voted** by **DAO voters** (WRT holders); calendar estimates are **derived** from mean block interval. With **stable** mean interval, halving moments are **calendar-predictable** as a consequence of fixed height **N**.

#### **6.3. Genesis block: two bootstrap wallets (no ZKP)**

At launch the **first block** (genesis) is created **deterministically**, like Bitcoin’s genesis—all initial records in **that block**. There is **no** distinct **Supervisor** role in the protocol—only **Citizen**, **Supplier**, and **Validator** (**§4.2**). Bootstrap is **two** fixed addresses in genesis config: one holds **Supplier**, the other **Validator**; both statuses are issued **without** ZKP as a **one-time** network seed. **MOA** applies to **both** the same as to ZKP-granted roles (**§5.3**): either address **can** lose its role if activity rules are breached.

**(1) Genesis Supplier wallet.** A fixed genesis address with **Supplier** status **without** ZKP. It receives the initial **ANT** emission (step (4)). As ZKP-verified **Suppliers** join, the economy follows normal rules; the genesis Supplier is **not** exempt from **MOA**.

**(2) Genesis Validator wallet.** A **second** fixed genesis address with **Validator** status **without** ZKP. It holds LZN (**(3)**) and enters `ValidatorSet` as in **(5)**. **Not** exempt from **MOA**.

**(3) LZN.** **10,000 LZN** (full one-off issuance) credited to the genesis **Validator** wallet; **1 LZN activated** (locked for mining)—minimal bootstrap, like Bitcoin’s first miner. **9,999 LZN** stay **non-activated** for trading or later activation.

**(4) ANT — initial emission.** Per **§5.5**: **`ANT_genesis = EpochBlocks × L_total_genesis`**, **L_total_genesis** = aggregate activated LZN in genesis (**1** on the genesis Validator wallet → **L_total_genesis = 1**), **EpochBlocks** from genesis (reference **10,080**). Hence **ANT_genesis = 10,080 × 1 = 10,080 ANT**, credited to the genesis **Supplier** wallet, matching per-epoch destruction **ceiling** at that **L_total** (**§5.5**). Implementation: standard Cosmos SDK genesis state (`InitGenesis`), **CometBFT-compatible**.

**(5) ValidatorSet.** The genesis **Validator** wallet with one activated LZN and initial **b_i** / **s_i** forms the **sole** genesis `ValidatorSet` validator. From block two, standard EndBlocker applies (§5.4).

**(6) Steady state.** New **Suppliers** and **Validators** appear via ZKP, receive roles and ANT per normal rules (§5.5), trade on the internal market. While the genesis Supplier is the only Supplier, it trades ANT **to** the genesis Validator (two addresses, internal market); from the **second** epoch, emission uses trade volume × coefficient (§5.5).

### **7. Governance: constitutional protocol with bounded DAO**

Volnix uses a strict rule stack. **DAO voters** hold governance power; voting power belongs **only** to **WRT holders** (weighting per governance module). Adopted proposals take effect only after a **long timelock**. DAO parameters have code **min/max**; votes cannot set values outside them.

#### **7.1. Constitutional layer (no vote—hard fork only)**

* **WRT & LZN tokenomics:** total issuance, **halving schedule** (including **every N blocks** for WRT), immutability of base monetary policy for these assets.
* **PoVB & ANT structure:** **two-number** per-height mechanism—**b_i** (ANT burn) and **s_i** (participation stake); **both** **burned** on successful block; **b_i + s_i ≤ L_i** (**1 LZN = 1 ANT**); **global cap Σ b_i ≤ λ × L_total** with **lowest-weight culling** if exceeded; **minimum Σ b_i ≥ λ × L_total**—if unmet, **EndBlocker** retries; **consensus weight w_i = s_i / L_i**; **b_i = 0 → zero** share (no WRT, no fees); **ANT** only via **internal market**—**§4.1**, **§5**; validator **ANT perpetual**; **Supplier emission epoch** from **trade volume**—**§5.5**; **MOA** (**T_g**, **T_v**)—**any signed transaction** in a sliding window; sanctions; **LZN** not seized—**§5.3**, **§3.3**.
* **Consensus:** proposer and block order—**standard CometBFT** by **w_i = s_i / L_i** (§6.1); `ValidatorSet` updated by **EndBlocker** of N for N+1; **at most K** signers with largest **w_i** (genesis **K = 150**), **K** is **DAO** (§7.2 item 10).
* **Periods in blocks:** **HalvingInterval**, **EpochBlocks**, and **K** are **consensus-block** counters, changeable via **DAO** (WRT vote) within code (§6.2, §7.2 items 10–12).
* **Genesis block** (§6.3): **two** bootstrap wallets (Supplier + Validator, **no** ZKP, **no** Supervisor role), **10,000 LZN** on the validator wallet (one activated), **10,080 ANT** to the supplier wallet = **EpochBlocks × L_total** at launch parameters; all records in the **first block**.
* **No** constitutional “hard max supply ANT”; ANT scale is **indirect** (§5.4–5.5).

#### **7.2. Legislative layer — DAO-tunable parameters (reference exhaustive list)**

1. **Activated LZN lock-up duration** after activation (and module-aligned unlock rules); §4.1, §5.1.
2. **Supplier MOA: T_g** — max time without a **signed transaction** from the **Supplier** address before revocation; §5.3.
3. **Validator MOA: T_v** — max time without a **signed transaction** from the **validator** address before revocation; §5.3.
4. **Per-epoch ANT accumulation cap** for a Supplier (if enabled); §4.2, §5.5.
5. **Lower and upper bounds** on the dynamic ANT **epoch emission coefficient** (reference genesis **0.75–1.5**); coefficient updates per §5.5 inside bounds.
6. **Max gas per block** (`BlockGasLimit` / Cosmos equivalent); §5.2, §6.1.
7. **Max block size in bytes.**
8. **Active Supplier count cap** — upper bound on concurrent **Supplier** slots; §4.2. While **active Suppliers ≥ cap**, **new** **Supplier** status is **not** granted (ZKP with **Supplier** for a new slot). If DAO **lowers** the cap, **existing** **Suppliers** are **not** auto-revoked for “being over cap”; only **new** **Supplier** grants freeze until active count is **strictly below** the new cap (including via **MOA**, role migration, other deterministic exits).
9. **BurnCapLambda (λ)** — global burn cap and minimum per block: **Σ b_i ≤ λ × L_total** (upper), **Σ b_i ≥ λ × L_total** (lower); if upper exceeded—weight-based culling; if lower missed—EndBlocker retries (§5.4). Genesis **λ = 1/3**; **DAO** within code **min/max**.
10. **Max active signers (MaxActiveValidators, K)** — upper bound on `ValidatorSet` size after EndBlocker: among validators passing other checks, **at most K** with **largest** **w_i** (§5.4). Genesis **K = 150**; **DAO** within code **min/max**.
11. **ANT emission epoch length in blocks (`EpochBlocks`)** — **consensus** blocks between Supplier ANT reset/mint boundaries (§5.5); reference **10,080** at ~1 block/min “week” target. **Proposed and approved** via WRT **DAO** within code **min/max**; **`ANT_max_epoch`** reference scales with **EpochBlocks** and **L_total** (**§5.5**).
12. **WRT halving interval (`HalvingInterval`, N blocks)** — **consensus** blocks between halvings of base WRT reward (§6.2); reference and **min/max** in code, concrete **N** **proposed and approved** by WRT holders via **DAO**.

Items **6–7** do not change WRT/LZN tokenomics or replace **burn** economics; they set **operational** chain limits. Item **8** caps **Supplier** role capacity. Item **9** sets **burn intensity**. Item **10** caps **consensus signer load**. Items **11–12** anchor key **periods in blocks** (ANT epoch, WRT halving) to **WRT votes**. **Hard max supply ANT** is **not** introduced via DAO.

### **8. Conclusion**

Volnix unifies **identity** (**ZKP**), **markets**, and **consensus**: security through personal accountability, fairness through role separation, efficiency through a **built-in** market without extra intermediaries. **ANT**, **PoVB**, **fee split**, and **Supplier epoch emission**—**§4.1**, **§5.4–5.5**; **CometBFT**, **block cadence**, **WRT halving**, and **genesis**—**§6**; **DAO**, **DAO voters** (**WRT**), **MOA** (**T_g** / **T_v**)—**§7**, **§5.3**, **§7.2**. **Open** reference stack—**§5**. Immutable **base** tokenomics with **bounded** DAO adaptation balances **reliability**, **performance**, and **decentralization**.

---

*End of whitepaper body (v4.20). For Russian see [WHITEPAPER_RU.md](./WHITEPAPER_RU.md). For detailed per-version change notes see [volnix_protocol.md](./volnix_protocol.md).*
