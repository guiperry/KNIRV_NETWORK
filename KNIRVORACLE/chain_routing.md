This revised Software Design Document (SDD) focuses on building a secure, high-performance, multi-chain routing mechanism in Rust, powered by a native utility token.

## Software Design Document: KNIRV Inter-Chain Router (K-ICR)

### 1. Introduction ✨

#### 1.1 Purpose
The purpose of this document is to detail the design and architecture of the **KNIRV Inter-Chain Router (K-ICR)**. K-ICR is a decentralized oracle built in **Rust** using the Substrate framework. Its function is to create a universally verifiable registry of canonical endpoints for external blockchain networks and resolve them using the **`knirv://`** URI scheme.

#### 1.2 Core Mechanism
The system is driven by **KNIRVROUTER Operators**, who compete to submit verifiable proof that a specific `knirv://` URI resolves to a canonical external chain endpoint (e.g., an RPC URL for Ethereum). Successful minting of these valid route entries results in the issuance of **NRN tokens** as a reward, establishing a decentralized and incentivized routing layer.

#### 1.3 Scope
* **Rust/Substrate Runtime:** Core logic, consensus, and state management.
* **Chain Registration:** A secure, on-chain registry for external blockchain metadata.
* **Route Minting:** The logic for claiming and verifying a `knirv://` route to an external chain endpoint.
* **NRN Token:** The utility token for incentivizing Router Operators.
* **Resolution API:** A high-speed interface for resolving `knirv://` URIs to network endpoints.

### 2. System Architecture 🏗️

K-ICR is structured around the modular, robust architecture of **Substrate**, utilizing its framework (FRAME) for runtime development in Rust.



#### 2.1 Technology Stack
| Component | Technology/Library | Purpose |
| :--- | :--- | :--- |
| **Blockchain Framework** | **Substrate (Rust)** | Provides the core blockchain infrastructure, consensus, and P2P networking. |
| **Runtime Logic** | **FRAME Pallets** | Modular components: `ChainRegistry`, `RouterMinting`, `NRNToken`. |
| **Off-chain Oracle** | **Off-chain Workers (OCW)** | Handles asynchronous, non-deterministic tasks like fetching external chain RPC status and submitting proof-of-work/attestations. |
| **Networking** | `libp2p` (Substrate Integration) | Handles peer discovery and block synchronization. |
| **HTTP Client** | `reqwest` (Rust crate) | Used by the OCW for high-performance HTTP requests to external chain endpoints. |

#### 2.2 Core Pallets

1.  **`ChainRegistry` Pallet:** Manages the list of external blockchains K-ICR supports.
    * **Extrinsic:** `register_chain(name, canonical_rpc_url, governance_proof_logic)`.
2.  **`RouterMinting` Pallet:** Contains the core business logic for URI-to-Endpoint mapping and NRN token issuance.
    * **Extrinsic:** `mint_route(chain_id, knirv_hash, target_endpoint)`. This is the operator's claim.
3.  **`NRNToken` Pallet:** Manages the native utility token balance, minting, and transfer.

### 3. Data Structures and Logic 💾

#### 3.1 Chain Registry Structure


http://googleusercontent.com/immersive_entry_chip/0

#### 3.2 Alternate URI (CMU): `knirv://`

The `knirv://` URI serves as the canonical, human-readable, and chain-verifiable route identifier.

**CMU Format:**
`knirv://[ChainName]/[RouteHash]`

* **`ChainName`:** A URL-safe identifier (e.g., `knirv://ethereum-mainnet/`).
* **`RouteHash`:** A deterministic hash derived from the concatenation of the `ChainID` and a unique, canonical identifier for the endpoint type (e.g., "primary-rpc", "websocket-endpoint").

**CMU Minting Hash Logic (Simplified):**
$$
\text{RouteHash} = \text{SHA256}(\text{ChainID} + \text{EndpointType} + \text{Timestamp}_{\text{InitialMint}})
$$

#### 3.3 Route Mapping and Operator Stake

The core storage of the `RouterMinting` Pallet maps the canonical URI to the operator and its status.


http://googleusercontent.com/immersive_entry_chip/1

---

### 4. Operational Flow 🔄

#### 4.1 Chain Registration
1.  A proposal is submitted via a runtime extrinsic (`register_chain`) with the chain's config details (name, base RPC, min stake).
2.  The proposal undergoes a governance vote (e.g., via the Substrate **`Governance Pallet`**).
3.  Once approved, the new `ExternalChainConfig` is stored in the **`ChainRegistry`**.

#### 4.2 Route Minting and NRN Issuance (The KNIRVROUTER Challenge)
This process is executed by the **KNIRVROUTER Operator** and utilizes the **Off-chain Workers (OCW)** for proof.

1.  **Operator Stake:** A Router Operator stakes the required `min_stake_nrn` for the target chain.
2.  **Mint Claim:** The Operator submits a `mint_route(chain_id, knirv_hash, claimed_endpoint_url)` extrinsic.
3.  **OCW Challenge:** The `RouterMinting` pallet triggers an OCW task for the validator nodes.
4.  **OCW Verification:**
    * The OCW attempts to connect to the `claimed_endpoint_url` using `reqwest`.
    * It performs a deterministic check (e.g., fetch the latest block hash and compare it against the canonical endpoint's hash).
    * If verification is successful, the OCW generates a signed attestation (an *unsigned extrinsic*).
5.  **Attestation Submission:** The OCW submits the signed attestation back to the chain.
6.  **NRN Reward:** If the attestation is verified on-chain, the `RouterMinting` pallet:
    * Updates the `RouterEntry` with the new endpoint and operator ID.
    * Triggers the **`NRNToken`** pallet to **mint and transfer NRN** to the successful Router Operator.

#### 4.3 `knirv://` Resolution
The K-ICR node provides a public-facing API for resolution.

1.  **Request:** A client sends a request (e.g., `GET /resolve/knirv/ethereum-mainnet/primary-rpc-hash`).
2.  **Query:** The API service queries the **`RouterMinting` Pallet** storage based on the `RouteHash`.
3.  **Response:** The node returns the active, verified `current_endpoint` (the actual RPC URL), allowing the client to connect directly to the external chain.

### 5. Token Economics (NRN Token) 🪙

#### 5.1 NRN Token Use Cases
* **Router Stake:** Operators must stake NRN to participate in the route minting process, which prevents sybil attacks and secures the integrity of the routing data.
* **Operator Rewards:** NRN is minted and rewarded to successful operators who prove the validity of a route, compensating them for the computational and networking cost (the oracle service).
* **Transaction Fees:** NRN is used to pay transaction fees on the K-ICR chain.

#### 5.2 Minting Mechanism
NRN is minted *programmatically* by the **`RouterMinting` Pallet** only when a new, successful route attestation is processed. The issuance rate is dynamically adjusted based on the current staking ratio, network demand, and the complexity/value of the external chain being rooted.

### 6. API and Inter-Chain Interface 🌐

#### 6.1 RPC/REST Endpoints
The K-ICR node exposes interfaces for users and external services.

| Method | Endpoint | Description |
| :--- | :--- | :--- |
| `GET` | `/api/v1/resolve/{knirv_uri}` | Resolves the full `knirv://` URI string to the current verified endpoint URL. |
| `GET` | `/api/v1/chains` | Retrieves the full list of supported `ExternalChainConfig` entries. |
| `POST` | `/api/v1/router/claim` | Submits a new route minting claim (requires NRN stake). |
| `GET` | `/api/v1/router/status/{knirv_hash}` | Checks the current status (endpoint, operator, last verified) of a route. |

#### 6.2 Cross-Chain Transfers (XCM)
The K-ICR chain will use **XCM (Cross-Consensus Messaging)** to facilitate the transfer of NRN tokens and routing proofs if it connects to a Polkadot/Kusama ecosystem. For non-Substrate chains (like Ethereum), a specific **Bridge Pallet** would be required to lock NRN and mint equivalent wrapped tokens on the external chain, allowing cross-chain payment for routing services.

### 7. Security and Validation 🛡️

* **Finality:** Use Substrate's **GRANDPA** finality mechanism for guaranteed state immutability.
* **OCW Integrity:** Off-chain Workers are executed by trusted validators, and their attestations are cryptographically signed, preventing malicious data injection.
* **Economic Security:** The mandatory NRN stake acts as a bond; any Router Operator who submits a fraudulent attestation is subject to **slashing** (loss of staked NRN), ensuring economic alignment with honest behavior.