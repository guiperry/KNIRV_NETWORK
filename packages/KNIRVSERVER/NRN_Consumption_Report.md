# NRN Token Consumption Report: Value Delivery to End Clients

## 1. Executive Summary
The NRN token serves as the primary economic fuel for the KNIRV Network, facilitating a "Truth-as-a-Service" model. Within the **KNIRVSERVER** and broader ecosystem, NRN is consumed to guarantee the integrity, factuality, and security of AI-driven workflows. Consumption occurs at two primary levels: **Infrastructure Allocation** (renting secure compute) and **Service Invocation** (high-confidence AI operations).

---

## 2. Infrastructure Consumption (DVE Utility)
End clients and developers consume NRN to access the **Deterministic Validation Environment (DVE)** infrastructure, which provides TEE (Trusted Execution Environment) security for sensitive operations.

### 2.1 DVE Rental (On-Demand Compute)
Users pay NRN to rent DVE instances for specific durations.
*   **Mechanism:** Hourly rental rates (e.g., 10, 25, or 50 NRN/hour depending on resource tier).
*   **Value Delivery:** Provides the client with a private, secure, and isolated environment (via TEE containers) to run sensitive models or handle confidential data without exposure to the host.
*   **Implementation:** Managed by `DVERentalService` in `KNIRVSERVER`.

### 2.2 DVE Node Registration & Staking
To participate as a validator or service provider, users must stake NRN.
*   **Requirement:** 1,000 NRN minimum stake.
*   **Value Delivery:** Ensures economic alignment. Higher stakes contribute to a higher **Reputation Score**, which in turn allows the node to be selected for higher-value validation tasks requested by clients.
*   **Implementation:** Managed by `DVECreationService` in `KNIRVSERVER`.

---

## 3. High-Value Service Consumption
End clients consume NRN to "buy up" to higher levels of AI certainty and specialized agent capabilities.

### 3.1 Factuality QA (High-Confidence Inference)
The "Factuality Slice" allows clients to request answers that are cross-referenced against evidence and validated by multiple DVEs.
*   **Cost:** **0.25 NRN** per invocation.
*   **Value Delivery:** Clients receive a "Factuality Score" and a cryptographic proof that the answer is grounded in verifiable evidence, significantly reducing the risk of AI hallucinations.
*   **Implementation:** Triggered via `KNIRVSERVER` validation handlers and settled on `KNIRV-ORACLE`.

### 3.2 Skill Invocation Fees
Agents within the network can possess specialized "Skills" (e.g., complex financial analysis, code auditing, or medical data extraction).
*   **Cost:** **1 NRN** base fee per skill invocation.
*   **Value Delivery:** Access to expert-level automated logic that goes beyond standard LLM capabilities.
*   **Implementation:** Enforced by the `FeeCollector` in `KNIRV-ORACLE`.

### 3.3 Model-to-Model Transitions
For complex tasks requiring multiple models (e.g., Planning -> Execution -> Verification), the network facilitates seamless handoffs.
*   **Cost:** **5 NRN** for systemic transitions.
*   **Value Delivery:** Orchestrates a "Mixture of Agents" (MoA) or "Task Orchestrator" workflow, ensuring the best model is used for each sub-task of a client's complex request.

---

## 4. Deflationary Mechanics & Network Integrity
Token consumption is designed to be deflationary, directly linking network utility to token value.

### 4.1 Fee Burning
The `KNIRV-ORACLE` acts as the economic regulator of the network.
*   **Burn Rate:** **50% of all service and transaction fees** are automatically burned (sent to a zero address).
*   **Value Delivery:** Long-term value preservation for token holders and active participants by reducing the circulating supply as network usage increases.

### 4.2 Slashing (Risk Mitigation)
If a DVE provider fails to provide accurate results or misbehaves:
*   **Penalty:** A portion of their staked NRN is slashed.
*   **Value Delivery:** Protects the end client by providing a financial guarantee of service quality. Slashing events reinforce the reliability of the "Truth" provided by the network.

---

## 5. Conclusion: The NRN Value Loop
The NRN token creates a closed-loop economy where:
1.  **Clients** pay NRN for guaranteed **Factuality** and **Security**.
2.  **DVE Operators** earn NRN for providing **Validation** and **Compute**.
3.  **The Network** burns NRN to ensure **Scarcity** and **Sustainability**.

By consuming NRN, the end client is not just buying "tokens," but is purchasing **verifiable certainty** in an era of AI uncertainty.
