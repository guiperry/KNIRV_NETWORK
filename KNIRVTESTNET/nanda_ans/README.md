
# AgentVerse Registry - NANDA+ANS Security Blueprint

**AgentVerse Registry** is a prototype application demonstrating a secure and federated registry for AI agent discovery and registration. It is built upon the **NANDA+ANS Security Blueprint**, a comprehensive architecture designed to establish a tamper-evident, dual-trust foundation for the burgeoning Internet of AI Agents. This blueprint uniquely supports both traditional CA-signed (Certificate Authority) and decentralized DID-based (Decentralized Identifier) identities, ensuring broad interoperability and robust cryptographic assurance.

## The NANDA+ANS Security Blueprint

The NANDA+ANS blueprint addresses critical security challenges in agent ecosystems, including identity verification, capability attestation, and secure discovery.

**Core Principles:**

*   **Dual-Trust Model:** Seamlessly integrates PKI-based trust (via ANS) and DID-based decentralized trust (via NANDA), allowing agents and CAs from different ecosystems to interoperate.
*   **Decoupled Two-Hop Lookup:**
    1.  **Anchor Tier (NANDA - Simple Registry):** NANDA (Naming and Discovery Anchors) provides a lightweight, globally resolvable layer. It stores `AgentAddr` records – minimal, digitally signed pointers containing the agent's DID, a URL to its detailed metadata (`AgentFacts`), and a Time-To-Live (TTL). This tier is optimized for high availability and resilience.
    2.  **Metadata Distribution Tier (ANS - Name Service & PKI):** ANS (Agent Name Service) defines the structure for `AgentFacts`. These are rich, cryptographically verifiable metadata documents hosted distributively (e.g., by the agent provider). ANS leverages PKI principles for hierarchical naming, capability attestation, and signature verification, allowing `AgentFacts` to be signed by the agent owner, publishers, or CAs.
*   **Enhanced Security & Resilience:** This separation minimizes the core registry's (NANDA) attack surface. Dynamic and richly attested metadata (`AgentFacts`) are managed at the edge, improving privacy, scalability, and update agility.
*   **Cryptographic Assurance:** Emphasizes strong cryptographic proof for agent identity and capabilities. While this prototype simulates digital signatures, a full implementation would use verifiable credentials, potentially with Zero-Knowledge Proofs, allowing consumers to cryptographically verify an agent's claimed skills before interaction.

## Key Security Flows

### 1. Agent Registration Flow (NANDA + ANS with PKI)

The registration process establishes an agent's verifiable presence in the ecosystem:

1.  **Identity & Metadata Preparation:**
    *   An agent (or its provider) generates or possesses a unique Decentralized Identifier (DID), which serves as its root identity (NANDA identifier).
    *   Detailed metadata, known as `AgentFacts`, is created. This includes the agent's name, capabilities, endpoints, provider information, version, and any relevant attestations or protocol extensions. The structure of `AgentFacts` is guided by ANS principles.
2.  **Digital Signature of AgentFacts (ANS/PKI aspect):**
    *   The `AgentFacts` document is digitally signed by the agent's private key (corresponding to its DID) or by a trusted third-party publisher/CA. This signature provides authenticity and integrity for the agent's detailed information.
    *   In a PKI-integrated ANS, this signature could be part of a certificate chain, linking the agent's key to a trusted root CA, thus verifying its legitimacy within a specific trust framework.
3.  **AgentFacts Hosting:** The signed `AgentFacts` JSON-LD document is hosted at a publicly accessible URL (e.g., on the provider's infrastructure via a well-known URI).
4.  **NANDA Record Creation (`AgentAddr`):**
    *   An `AgentAddr` record is created for the NANDA registry. This record is minimal and includes:
        *   `agent_id`: The agent's DID.
        *   `facts_url`: The URL pointing to the hosted (and signed) `AgentFacts`.
        *   `ttl`: Time-to-live for this NANDA record.
5.  **Digital Signature of AgentAddr (NANDA aspect):**
    *   The `AgentAddr` itself is digitally signed, typically by the NANDA registry shard or a designated authority responsible for that part of the NANDA namespace. This ensures the integrity and authenticity of the pointer itself.
6.  **Publication:** The signed `AgentAddr` is published to the NANDA registry.

*In this prototype, the registration form simulates these steps by generating mock DIDs, constructing `AgentFacts` (including simulated signature data with PKI-relevant fields like `simulatedIssuer` and `simulatedPublicKey`), and displaying this data. No actual cryptographic operations or network publication occurs.*

### 2. Agent Discovery Flow (NANDA + ANS with PKI)

Users or other agents discover and verify agents as follows:

1.  **Initial Lookup (NANDA):**
    *   A client queries the NANDA registry using a known agent identifier (e.g., its DID or a human-friendly alias resolvable to a DID).
    *   The NANDA registry returns the corresponding `AgentAddr` record.
2.  **Verification of AgentAddr (NANDA aspect):**
    *   The client (optionally) verifies the digital signature on the `AgentAddr` to ensure the pointer itself is authentic and hasn't been tampered with since being published by the NANDA registry.
3.  **Fetching AgentFacts (Two-Hop):**
    *   The client uses the `facts_url` from the `AgentAddr` to retrieve the detailed `AgentFacts` document from the Metadata Distribution Tier.
4.  **Verification of AgentFacts (ANS/PKI aspect):**
    *   The client verifies the digital signature(s) within the `AgentFacts` document. This critical step confirms:
        *   **Authenticity:** The `AgentFacts` were indeed issued by the claimed agent/provider.
        *   **Integrity:** The agent's capabilities, endpoints, and other metadata have not been altered.
    *   If using PKI, this verification might involve checking the signature against the agent's public key (potentially obtained from its DID document or a certificate) and validating the certificate chain up to a trusted root CA.
5.  **Trust Decision & Interaction:** Based on successful verification, the client can trust the agent's identity and claimed capabilities and proceed with secure interaction.

*This prototype's discovery page fetches agent data (which includes simulated signatures) and the agent profile page displays these simulated PKI details, allowing users to conceptually follow this verification flow.*

## About This Project: AgentVerse Registry Prototype

This Next.js application serves as a high-fidelity prototype to demonstrate the core concepts of the NANDA+ANS Security Blueprint.

**Key Features Demonstrated:**

*   **Agent Listing & Discovery:** Browse and search for registered AI agents.
*   **Detailed Agent Profiles:** View comprehensive information about each agent, including its capabilities, provider, version, and (simulated) NANDA+ANS registry details and PKI-based signature information.
*   **Simulated Agent Registration:** A form allows users to "register" new agents. This process generates realistic default data, including mock DIDs and simulated digital signatures with PKI elements, to illustrate the registration flow. (Note: This prototype does not perform actual cryptographic operations or persist new registrations to a live, shared database beyond the session's mock data unless Firebase is fully configured and enabled by the developer).
*   **AI-Generated Summaries:** Utilizes Genkit to provide AI-powered summaries of agent functionalities on their profile pages.

**Technology Stack:**

*   Next.js (App Router, Server Components)
*   React
*   TypeScript
*   ShadCN UI Components
*   Tailwind CSS
*   Genkit (for AI features)
*   Firebase (Firestore, for optional persistence of agent data if configured)

## Running the Project Prototype

Follow these instructions to set up and run the prototype locally.

**Prerequisites:**

*   Node.js (v18 or later recommended)
*   npm or yarn

**Setup & Installation:**

1.  **Clone the Repository** (if applicable, otherwise ensure you have the project files):
    ```bash
    # git clone <repository-url>
    # cd agentverse-registry-nanda-ans
    ```

2.  **Install Dependencies:**
    ```bash
    npm install
    # OR
    # yarn install
    ```

3.  **Environment Variables:**
    *   Create a `.env` file in the root of the project by copying `.env.example` (if one exists) or creating it manually.
    *   **Firebase Configuration (Optional but Recommended for full functionality):** If you want to use Firebase to persist agent data beyond the mock samples:
        *   Create a Firebase project at [https://console.firebase.google.com/](https://console.firebase.google.com/).
        *   Enable Firestore database in your project.
        *   Add a Web app to your Firebase project and copy its configuration credentials.
        *   Populate the Firebase credentials in your `.env` file:
            ```env
            NEXT_PUBLIC_FIREBASE_API_KEY=your_api_key
            NEXT_PUBLIC_FIREBASE_AUTH_DOMAIN=your_auth_domain
            NEXT_PUBLIC_FIREBASE_PROJECT_ID=your_project_id
            NEXT_PUBLIC_FIREBASE_STORAGE_BUCKET=your_storage_bucket
            NEXT_PUBLIC_FIREBASE_MESSAGING_SENDER_ID=your_messaging_sender_id
            NEXT_PUBLIC_FIREBASE_APP_ID=your_app_id
            NEXT_PUBLIC_FIREBASE_MEASUREMENT_ID=your_measurement_id
            ```
    *   **Genkit Configuration (for AI Summaries):**
        *   Ensure you have a Google AI API key if you want the AI summary feature to work.
        *   You might need to set `GOOGLE_API_KEY=your_google_ai_api_key` in your `.env` file or configure Genkit appropriately.
        *   By default, the prototype uses `googleai/gemini-2.0-flash` via Genkit.

4.  **Run the Development Server:**
    ```bash
    npm run dev
    ```
    The application should now be running at `http://localhost:9002` (or the port specified in your `package.json`).

5.  **Run Genkit (for AI features, in a separate terminal):**
    If you are actively developing or testing AI features that use Genkit flows (like the agent summary):
    ```bash
    npm run genkit:dev
    # OR for watching changes
    # npm run genkit:watch
    ```
    The Genkit development UI will typically be available at `http://localhost:4000`.

## Contributing

Contributions to enhance this prototype or further explore the NANDA+ANS blueprint are welcome!

**Guidelines:**

*   **Bug Reports & Feature Requests:** Please open an issue on the project's repository, providing as much detail as possible.
*   **Code Style:**
    *   Follow existing code patterns and Next.js/React best practices.
    *   Utilize TypeScript for type safety.
    *   Use ShadCN UI components where appropriate and adhere to Tailwind CSS utility-first principles.
*   **Commit Messages:** Aim for clear and descriptive commit messages. Consider using Conventional Commits if you are familiar with the standard (e.g., `feat: Add X feature`, `fix: Resolve Y bug`).
*   **Pull Request Process:**
    1.  Fork the repository (if applicable).
    2.  Create a new branch for your feature or bug fix (e.g., `git checkout -b feat/new-agent-verification`).
    3.  Make your changes and commit them.
    4.  Push your branch to your fork.
    5.  Open a Pull Request against the main project repository, detailing the changes you've made.
    6.  Ensure your PR passes any CI checks and addresses any review feedback.

This README aims to provide a solid foundation for understanding the AgentVerse Registry prototype and its underlying NANDA+ANS security principles.
