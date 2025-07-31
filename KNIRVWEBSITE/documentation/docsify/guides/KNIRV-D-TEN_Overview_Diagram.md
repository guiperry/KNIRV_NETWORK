```mermaid
graph TD
    subgraph KNIRV D-TEN Ecosystem
        U[User] -- Voice/Visual Input --> KS[KNIRV-SHELL]
        KS -- Manages LoRA Adapters --> L[Rust WASM LoRA Adapters]
        KS -- Rents DVEs for Validation --> DVE[KNIRV-NEXUS DVEs]
        KS -- Uses for NRN/Transactions --> KW["KNIRV-WALLET (XION Meta Account)"]

        KW -- Acquires NRN from Faucet --> KR["KNIRV-ROOT (NRN Oracle & Orchestrator)"]
        KR -- Provides USDC Faucet --> R[KNIRV-ROUTERS]
        R -- Mints NRNs --> KR

        KS -- Submits/Queries Data --> KG["KNIRV-GRAPH (Problem/Solution Fabric)"]
        KG -- Feeds Data --> KR

        KR -- Propagates Canonical State --> KS
        KS -- "Proposes Base LLM Updates / SkillNodes" --> KXC["KNIRVCHAIN (CosmWasm on XION)"]
        KXC -- "Provides Canonical Base LLM / Skill Registry" --> KS

        KXC -- "Handles NRN Minting/Burning" --> KW
        KXC -- "Records All NRN Transactions" --> KR

        KS -- Controls Agent Units --> KN["KNIRVANA (Game Client)"]
        KN -- Uses NRNs for Skill Invocation --> KXC

        DVE -- "Generates Proofs for Base LLM / Skills" --> KS
        KS -- "Submits Proofs to KNIRVCHAIN" --> KXC

        style KXC fill:#2c7bb6,stroke:#333,stroke-width:2px,color:#fff
        style KS fill:#d85450,stroke:#333,stroke-width:2px
        style KR fill:#2d7336,stroke:#333,stroke-width:2px,color:#fff
        style KW fill:#663399,stroke:#333,stroke-width:2px,color:#fff
        style KG fill:#008080,stroke:#333,stroke-width:2px,color:#fff
        style DVE fill:#996633,stroke:#333,stroke-width:2px,color:#fff
        style R fill:#ff9900,stroke:#333,stroke-width:2px
        style KN fill:#cc6699,stroke:#333,stroke-width:2px
    end
```
# KNIRV-SHELL Architecture Diagram
```mermaid
 graph TD
    A[User Input: Voice / Screenshot / Camera] --> B[KNIRV-SHELL UI Module]
    B --> C[Voice Control Module]
    B --> D[Visual Input Module]
    C --> E[KNIRV-SHELL Core Module]
    D --> E
    E -- Internal Data Flow --> F["The Fabric" Algorithm Module]
    E -- Manages LoRA Adapters --> G[Rust WASM LoRA Adapters]
    E -- Interacts with --> H[Network Interaction Module]

    F -- Translates into --> I[Visual NRV Objects]
    I --> B

    H -- "Queries Base LLM / Submits Base LLM Updates" --> J["KNIRVCHAIN (on XION)"]
    H -- "Submits SkillNodes / ErrorNodes" --> K[KNIRV-GRAPH]
    H -- Rents DVEs --> L[KNIRV-NEXUS DVEs]
    H -- "Manages NRNs / Transactions" --> M["KNIRV-WALLET (XION Meta Account)"]
    H -- "Routes Game/Agent P2P Traffic" --> N[KNIRV-ROUTERS]
    E -- Controls Agent Units --> O["KNIRVANA (Game Client)"]

    J -- Syncs --> P["KNIRV-ROOT (NRN Oracle & Orchestrator)"]
    K -- Feeds Data To --> P

    %% Added relationships for KNIRV-ROOT's faucet and state propagation
    P -- "Provides USDC Faucet" --> N
    M -- "Acquires NRN from Faucet" --> P
    P -- "Propagates Canonical Base LLM / NRN State" --> E
```

---

<div class="footer-links">


© 2025 KNIRV Network
</div>
