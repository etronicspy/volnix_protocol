# Volnix Protocol - Architecture Diagrams

This document contains detailed architectural diagrams for the Volnix Protocol infrastructure.

## 1. System Architecture Overview

```mermaid
graph TB
    subgraph "External Layer"
        USERS[Users & Applications]
        EXCHANGES[Exchanges & Partners]
        DEVS[Developers & dApps]
    end
    
    subgraph "Interface Layer"
        WUI[Wallet UI<br/>React + TypeScript]
        BE[Blockchain Explorer<br/>HTML/JS + PowerShell]
        CLI[CLI Tools<br/>volnixd commands]
        SDK[Developer SDK<br/>CosmJS Integration]
    end
    
    subgraph "API Gateway Layer"
        REST[REST API<br/>Port 1317]
        GRPC[gRPC API<br/>Port 9090]
        GRPCWEB[gRPC-Web<br/>Port 9091]
        RPC[RPC Server<br/>Port 26657]
    end
    
    subgraph "Application Layer"
        APP[Volnix App<br/>Cosmos SDK v0.53.4]
        ABCI[ABCI Interface<br/>Application-Consensus Bridge]
        MM[Module Manager<br/>Lifecycle & Routing]
    end
    
    subgraph "Custom Modules"
        IDENT[x/ident<br/>Identity & ZKP<br/>Role Management]
        LIZENZ[x/lizenz<br/>License Management<br/>MOA Monitoring]
        ANTEIL[x/anteil<br/>Internal Market<br/>ANT Trading]
        CONSENSUS[x/consensus<br/>PoVB Logic<br/>Validator Selection]
    end
    
    subgraph "Consensus Layer"
        COMETBFT[CometBFT v0.38.17<br/>Byzantine Fault Tolerant<br/>Consensus Engine]
        MEMPOOL[Transaction Mempool<br/>Pending Transactions]
        VALIDATOR[Validator Set<br/>Active Block Creators]
    end
    
    subgraph "Storage Layer"
        STATE[State Store<br/>GoLevelDB<br/>Current State]
        BLOCKS[Block Store<br/>GoLevelDB<br/>Historical Blocks]
        IAVL[IAVL Tree<br/>Merkle Proofs<br/>State Verification]
    end
    
    subgraph "Network Layer"
        P2P[P2P Network<br/>Port 26656<br/>Inter-node Communication]
        SYNC[State Sync<br/>Fast Synchronization]
        GOSSIP[Block Gossip<br/>Block Propagation]
    end
    
    USERS --> WUI
    USERS --> BE
    DEVS --> CLI
    DEVS --> SDK
    EXCHANGES --> REST
    
    WUI --> REST
    BE --> RPC
    CLI --> APP
    SDK --> GRPC
    
    REST --> APP
    GRPC --> APP
    GRPCWEB --> APP
    RPC --> APP
    
    APP --> ABCI
    APP --> MM
    MM --> IDENT
    MM --> LIZENZ
    MM --> ANTEIL
    MM --> CONSENSUS
    
    ABCI --> COMETBFT
    COMETBFT --> MEMPOOL
    COMETBFT --> VALIDATOR
    
    IDENT --> STATE
    LIZENZ --> STATE
    ANTEIL --> STATE
    CONSENSUS --> STATE
    
    STATE --> IAVL
    COMETBFT --> BLOCKS
    
    COMETBFT --> P2P
    P2P --> SYNC
    P2P --> GOSSIP
```

## 2. Module Interaction Diagram

```mermaid
graph LR
    subgraph "Identity Management"
        IDENT[x/ident Module]
        ZKP[ZKP Verification]
        ROLES[Role Management]
        MIGRATION[Identity Migration]
    end
    
    subgraph "License Management"
        LIZENZ[x/lizenz Module]
        ACTIVATION[LZN Activation]
        MOA[MOA Monitoring]
        PENALTIES[Penalty System]
    end
    
    subgraph "Internal Market"
        ANTEIL[x/anteil Module]
        ORDERBOOK[Order Book]
        AUCTIONS[ANT Auctions]
        TRADING[Trade Execution]
    end
    
    subgraph "Consensus Logic"
        CONSENSUS[x/consensus Module]
        POVB[PoVB Algorithm]
        SELECTION[Block Creator Selection]
        HALVING[Halving Mechanism]
    end
    
    subgraph "External Modules"
        BANK[x/bank<br/>Token Transfers]
        STAKING[x/staking<br/>Validator Staking]
        GOV[x/gov<br/>Governance]
    end
    
    IDENT --> ZKP
    IDENT --> ROLES
    IDENT --> MIGRATION
    
    LIZENZ --> ACTIVATION
    LIZENZ --> MOA
    LIZENZ --> PENALTIES
    
    ANTEIL --> ORDERBOOK
    ANTEIL --> AUCTIONS
    ANTEIL --> TRADING
    
    CONSENSUS --> POVB
    CONSENSUS --> SELECTION
    CONSENSUS --> HALVING
    
    IDENT -.->|Role Verification| LIZENZ
    LIZENZ -.->|Validator Status| CONSENSUS
    CONSENSUS -.->|ANT Requirements| ANTEIL
    ANTEIL -.->|Market Data| CONSENSUS
    
    LIZENZ -.->|Token Operations| BANK
    CONSENSUS -.->|Validator Management| STAKING
    IDENT -.->|Governance Rights| GOV
```

## 3. Transaction Processing Pipeline

```mermaid
sequenceDiagram
    participant Client
    participant API
    participant App
    participant Modules
    participant CometBFT
    participant Network
    
    Client->>API: Submit Transaction
    API->>App: Route Transaction
    App->>App: Validate Basic
    App->>Modules: Execute AnteHandler
    Modules->>Modules: Check Signatures
    Modules->>Modules: Validate Permissions
    App->>CometBFT: Add to Mempool
    
    Note over CometBFT: Consensus Round
    CometBFT->>Network: Propose Block
    Network->>CometBFT: Vote on Block
    CometBFT->>App: Deliver Block
    
    App->>Modules: Execute Transactions
    Modules->>Modules: Update State
    Modules->>App: Return Results
    App->>CometBFT: Commit State
    CometBFT->>Client: Transaction Result
```

## 4. PoVB Consensus Flow

```mermaid
graph TB
    subgraph "Block Creation Cycle"
        START[New Round Start]
        BURN[ANT Burn Verification]
        AUCTION[Blind Auction Process]
        SELECT[Block Creator Selection]
        PROPOSE[Block Proposal]
        VOTE[Validator Voting]
        COMMIT[Block Commit]
        REWARD[Reward Distribution]
    end
    
    subgraph "Validator Pool"
        ACTIVE[Active Validators<br/>LZN Licensed]
        STANDBY[Standby Validators<br/>MOA Compliant]
        INACTIVE[Inactive Validators<br/>Penalized]
    end
    
    subgraph "Economic Mechanics"
        ANTBURN[ANT Token Burning]
        HALVING[Halving Schedule]
        DYNAMIC[Dynamic Block Time]
        MARKET[Internal ANT Market]
    end
    
    START --> BURN
    BURN --> AUCTION
    AUCTION --> SELECT
    SELECT --> PROPOSE
    PROPOSE --> VOTE
    VOTE --> COMMIT
    COMMIT --> REWARD
    REWARD --> START
    
    ACTIVE --> SELECT
    STANDBY --> ACTIVE
    INACTIVE --> STANDBY
    
    ANTBURN --> AUCTION
    HALVING --> ANTBURN
    DYNAMIC --> START
    MARKET --> ANTBURN
```

## 5. Network Topology

```mermaid
graph TB
    subgraph "Validator Network Core"
        V1[Validator 1<br/>Primary Creator]
        V2[Validator 2<br/>Backup Creator]
        V3[Validator 3<br/>Active Standby]
        V4[Validator N<br/>Active Standby]
    end
    
    subgraph "Full Node Network"
        F1[Full Node 1<br/>RPC Service]
        F2[Full Node 2<br/>API Gateway]
        F3[Full Node 3<br/>Archive Node]
        F4[Full Node 4<br/>Seed Node]
    end
    
    subgraph "Light Client Network"
        L1[Light Client 1<br/>Mobile Wallet]
        L2[Light Client 2<br/>Web App]
        L3[Light Client 3<br/>IoT Device]
    end
    
    subgraph "External Services"
        E1[Block Explorer]
        E2[Exchange API]
        E3[DeFi Protocol]
        E4[Analytics Service]
    end
    
    V1 -.->|Consensus| V2
    V2 -.->|Consensus| V3
    V3 -.->|Consensus| V4
    V4 -.->|Consensus| V1
    
    V1 -->|State Sync| F1
    V2 -->|State Sync| F2
    V3 -->|State Sync| F3
    V4 -->|State Sync| F4
    
    F1 -->|Header Sync| L1
    F2 -->|Header Sync| L2
    F3 -->|Header Sync| L3
    
    F1 -->|API Access| E1
    F2 -->|API Access| E2
    F3 -->|API Access| E3
    F4 -->|API Access| E4
```

## 6. Data Flow Architecture

```mermaid
graph LR
    subgraph "Data Sources"
        TX[Transactions]
        BLOCKS[Blocks]
        STATE[State Changes]
        EVENTS[Events]
    end
    
    subgraph "Processing Layer"
        VALIDATE[Validation]
        EXECUTE[Execution]
        STORE[Storage]
        INDEX[Indexing]
    end
    
    subgraph "Storage Systems"
        LEVELDB[(GoLevelDB<br/>Primary Storage)]
        IAVL[(IAVL Tree<br/>State Proofs)]
        CACHE[(Memory Cache<br/>Hot Data)]
    end
    
    subgraph "Query Layer"
        GRPC_Q[gRPC Queries]
        REST_Q[REST Queries]
        RPC_Q[RPC Queries]
        WS[WebSocket Streams]
    end
    
    subgraph "Client Applications"
        WALLET[Wallet UI]
        EXPLORER[Block Explorer]
        DAPPS[dApps]
        TOOLS[CLI Tools]
    end
    
    TX --> VALIDATE
    BLOCKS --> VALIDATE
    STATE --> EXECUTE
    EVENTS --> EXECUTE
    
    VALIDATE --> STORE
    EXECUTE --> STORE
    STORE --> INDEX
    
    STORE --> LEVELDB
    STORE --> IAVL
    INDEX --> CACHE
    
    LEVELDB --> GRPC_Q
    IAVL --> REST_Q
    CACHE --> RPC_Q
    CACHE --> WS
    
    GRPC_Q --> WALLET
    REST_Q --> EXPLORER
    RPC_Q --> DAPPS
    WS --> TOOLS
```

## 7. Security Architecture

```mermaid
graph TB
    subgraph "Identity Layer"
        ZKP_VERIFY[ZKP Verification<br/>Zero-Knowledge Proofs]
        ROLE_AUTH[Role Authorization<br/>Guest/Citizen/Validator]
        IDENTITY_MIGRATE[Identity Migration<br/>Digital Inheritance]
    end
    
    subgraph "Consensus Security"
        BFT[Byzantine Fault Tolerance<br/>1/3 Malicious Tolerance]
        ECONOMIC[Economic Security<br/>Staking + Burning]
        SYBIL[Sybil Resistance<br/>ZKP Identity Verification]
    end
    
    subgraph "Network Security"
        P2P_ENCRYPT[P2P Encryption<br/>TLS/Noise Protocol]
        DDOS_PROTECT[DDoS Protection<br/>Rate Limiting]
        FIREWALL[Network Firewall<br/>Port Management]
    end
    
    subgraph "Application Security"
        TX_VALIDATE[Transaction Validation<br/>Signature Verification]
        STATE_VERIFY[State Verification<br/>Merkle Proofs]
        ANTE_HANDLER[Ante Handler<br/>Pre-execution Checks]
    end
    
    subgraph "Economic Security"
        SLASH[Slashing Conditions<br/>MOA Violations]
        BURN_VERIFY[Burn Verification<br/>ANT Token Burning]
        MARKET_INTEGRITY[Market Integrity<br/>Order Book Protection]
    end
    
    ZKP_VERIFY --> ROLE_AUTH
    ROLE_AUTH --> IDENTITY_MIGRATE
    
    BFT --> ECONOMIC
    ECONOMIC --> SYBIL
    
    P2P_ENCRYPT --> DDOS_PROTECT
    DDOS_PROTECT --> FIREWALL
    
    TX_VALIDATE --> STATE_VERIFY
    STATE_VERIFY --> ANTE_HANDLER
    
    SLASH --> BURN_VERIFY
    BURN_VERIFY --> MARKET_INTEGRITY
    
    ZKP_VERIFY -.->|Identity Proofs| BFT
    ROLE_AUTH -.->|Validator Rights| SLASH
    TX_VALIDATE -.->|Signature Check| P2P_ENCRYPT
```

## 8. Deployment Architecture

```mermaid
graph TB
    subgraph "Production Environment"
        LB[Load Balancer<br/>HAProxy/Nginx]
        
        subgraph "Validator Cluster"
            V1[Validator Node 1<br/>Primary]
            V2[Validator Node 2<br/>Backup]
            V3[Validator Node 3<br/>Standby]
        end
        
        subgraph "Full Node Cluster"
            F1[Full Node 1<br/>Public RPC]
            F2[Full Node 2<br/>Archive]
            F3[Full Node 3<br/>Seed]
        end
        
        subgraph "Monitoring Stack"
            PROM[Prometheus<br/>Metrics Collection]
            GRAF[Grafana<br/>Visualization]
            ALERT[AlertManager<br/>Notifications]
        end
    end
    
    subgraph "Development Environment"
        DEV[Development Node<br/>Local Testing]
        TEST[Testnet Node<br/>Integration Testing]
        STAGE[Staging Node<br/>Pre-production]
    end
    
    subgraph "External Services"
        DNS[DNS Service<br/>Domain Management]
        CDN[CDN Service<br/>Static Assets]
        BACKUP[Backup Service<br/>Data Protection]
    end
    
    LB --> V1
    LB --> F1
    LB --> F2
    
    V1 -.-> V2
    V2 -.-> V3
    
    F1 --> F3
    F2 --> F3
    
    V1 --> PROM
    F1 --> PROM
    PROM --> GRAF
    PROM --> ALERT
    
    DEV --> TEST
    TEST --> STAGE
    STAGE --> V1
    
    LB --> DNS
    F1 --> CDN
    V1 --> BACKUP
```

---

*These diagrams provide visual representations of the Volnix Protocol architecture at different levels of detail. They complement the system overview documentation and serve as reference materials for developers, operators, and stakeholders.*