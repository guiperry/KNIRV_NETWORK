# KNIRVROOT User Guide

## Overview

KNIRVROOT is a blockchain-based platform for AI capabilities, allowing you to register, discover, and use AI tools, prompts, plugins, and more in a secure and transparent way. All interactions are recorded on the blockchain, creating an immutable audit trail.

## Prerequisites

Before using KNIRVROOT, ensure you have:

* Go installed on your system
* Familiarity with blockchain concepts and terminology

## Getting Started

To begin using KNIRVROOT, you'll need to interact with two main components: the KNIRVROOT node (the blockchain itself) and the Wallet Server (for managing transactions).

### Setting up the KNIRVROOT Node

This sets up the core blockchain network.

```bash
git clone <repository_url>
cd KNIRVROOT
go build -o KNIRVROOT_node ./cmd/KNIRVROOT/main.go
./KNIRVROOT_node --port 8080 --p2p_port 6001
```

### Setting up the Wallet Server

The Wallet Server helps you create and sign transactions. Run this after setting up the KNIRVROOT node.

```bash
go build -o wallet_server ./cmd/walletserver/main.go
./wallet_server --port 9090 --blockchain_server_ip http://localhost:8080
```

(Note: Replace `<repository_url>` with the actual Git repository URL.)

## Key Concepts

### Capabilities

* **Plugins:** Executable code (e.g., Go, Wasm) that extends KNIRVROOT's functionality.
* **Tools:** Actions defined with input/output schemas.
* **Prompts:** Reusable templates for LLMs.
* **Memory Services:** Persistent data storage.

### Context Records

Every interaction with a capability creates a Context Record on the blockchain, providing a complete audit trail. This includes who used the capability, what inputs and outputs were used, and any fees paid.

### NRN Tokens

The native currency used to pay transaction fees.

## Using KNIRVROOT

KNIRVROOT's capabilities are accessed via APIs. Here are some common use cases:

### Discovering Capabilities

Use the API to find available capabilities:

```bash
GET /mcp/capabilities?type=<capability_type>&owner=<owner_address>
```

Replace `<capability_type>` (e.g., PLUGIN, TOOL) and `<owner_address>` as needed.

### Using a Capability

Once you've found a capability, use the Wallet Server API to create and submit a transaction to invoke it. The specific API call will depend on the capability type. For example, to invoke a tool:

```bash
POST /wallet/mcp/create_invoke_capability
```

### Registering a Capability

Developers can register their own capabilities using the Wallet Server API:

```bash
POST /wallet/mcp/create_register_capability
```

This requires providing metadata about your capability (e.g., name, description, execution instructions). Large files (like plugin binaries) are stored off-chain, with only a hash stored on the blockchain for verification.

## Troubleshooting

* **Node not starting:** Check your Go installation and ensure the node's configuration is correct.
* **Transaction failures:** Verify you have sufficient NRN tokens and that the transaction is properly formatted. Check the node logs for error messages.
* **Capability not found:** Double-check the capability ID and ensure it's correctly registered on the blockchain.

## Advanced Features (Future Enhancements)

* **Enhanced Querying:** Future versions will improve querying capabilities using RealmDB for more complex searches.
* **Access Control:** Future implementations will add more granular access control mechanisms.

## Contributing

See `CONTRIBUTING.md` for guidelines on contributing to the project.

## License

This project is licensed under the MIT License.

Improvements made:

* Added an overview section to provide a brief introduction to KNIRVROOT
* Emphasized the importance of prerequisites, including Go installation and familiarity with blockchain concepts
* Clarified the purpose of the Wallet Server and its role in managing transactions
* Provided more detailed explanations of key concepts, including capabilities, context records, and NRN tokens
* Added code examples for common use cases, including discovering capabilities and using a capability
* Improved the troubleshooting section by providing more specific error messages and solutions
* Added a section on advanced features and future enhancements to provide a roadmap for future development.

<div class="footer-links">
<a href="#/legal/CODE_OF_CONDUCT.md" class="footer-link">Contributor Covenant Code of Conduct</a> | <a href="#/legal/PRIVACY_POLICY.md" class="footer-link">PRIVACY_POLICY.md</a> | <a href="#/legal/TERMS_AND_CONDITIONS.md" class="footer-link">TERMS AND CONDITIONS</a>

© 2025 KNIRV Network
</div>
