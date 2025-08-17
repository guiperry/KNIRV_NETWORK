# KNIRVORACLE User Guide

KNIRVORACLE is a blockchain-based platform for AI capabilities. It allows you to register, discover, and use tools, prompts, plugins, and more, all while maintaining a transparent and verifiable record on the blockchain.

## Getting Started

This guide will help you understand and use KNIRVORACLE. To begin, you'll need to:

### Install the KNIRVORACLE Node

The KNIRVORACLE Node runs the core blockchain software.

```bash
git clone <repository_url>
cd KNIRVORACLE
go build -o KNIRVORACLE_node ./cmd/KNIRVORACLE/main.go
./KNIRVORACLE_node --port 8080 --p2p_port 6001
```

### Install the Wallet Server

The Wallet Server helps manage your transactions.

```bash
go build -o wallet_server ./cmd/walletserver/main.go
./wallet_server --port 9090 --blockchain_server_ip http://localhost:8080
```

### Install an Agent Client

The Agent Client is an application that interacts with KNIRVORACLE to find and use AI capabilities. This client will handle downloading and running plugins, and interacting with other capabilities.

## Core Concepts

KNIRVORACLE is built around several core concepts:

### Capabilities

Capabilities are the building blocks of KNIRVORACLE. They include:

* **Plugins:** Executable code (like Go or Wasm) you can upload and others can use.
* **Tools:** Actions defined by input and output schemas.
* **Prompts:** Reusable templates for large language models.
* **Memory Services:** Persistent data storage.
* **Datasets & Model Artifacts:** References to external data and models.

### NRN Tokens

NRN Tokens are used to pay fees for using KNIRVORACLE.

## Using KNIRVORACLE

### Register a Capability

If you've developed a plugin, tool, or other capability, you can register it on KNIRVORACLE using the Wallet Server API. This involves providing metadata and paying a fee.

### Discover Capabilities

Use the KNIRVORACLE API to search for available capabilities. You can filter by type, owner, and other criteria.

### Use a Capability

Once you've found a capability, your agent client will interact with it. For plugins, this involves downloading, verifying, and running the code. For other capabilities, it involves sending input and receiving output.

### View Audit Trail

Every interaction is recorded on the blockchain as a `ContextRecord`. You can use the API to view this immutable record of all activity.

## API Overview

KNIRVORACLE provides APIs for interacting with the blockchain and managing capabilities. Key endpoints include:

* `/mcp/capabilities`: Search for available capabilities.
* `/mcp/capability/{id}`: Get details about a specific capability.
* `/mcp/context/{id}`: Get details about a specific interaction.
* `/transaction`: Submit a transaction.
* Wallet Server APIs: For creating and signing transactions.

(See the full API documentation for details.)

## Troubleshooting

### Common Issues

* **Error downloading plugin:** Verify the plugin's `ContentHash` matches the on-chain value. Check the plugin's location hint (URI) is accessible.
* **Transaction failure:** Ensure you have sufficient NRN tokens and the transaction is properly formatted. Check the KNIRVORACLE node logs for errors.
* **Capability not found:** Double-check the capability ID and ensure it's correctly registered.

### Advanced Troubleshooting

For more complex issues, refer to the KNIRVORACLE logs and API documentation.

## Contributing

See `CONTRIBUTING.md` for guidelines on contributing to KNIRVORACLE.

## License

This project is licensed under the MIT License.

Improvements made:

* Added clear headings and subheadings to improve structure and readability.
* Provided more detailed instructions for installing the KNIRVORACLE Node and Wallet Server.
* Added a section on troubleshooting common issues and advanced troubleshooting.
* Emphasized the importance of verifying the plugin's `ContentHash` and checking the plugin's location hint (URI) when troubleshooting plugin download issues.
* Clarified the role of the Agent Client and its interaction with KNIRVORACLE.
* Provided more detailed information on the KNIRVORACLE API and its endpoints.
* Added a section on viewing the audit trail and its importance in maintaining transparency and verifiability.

<div class="footer-links">
<a href="#/legal/CODE_OF_CONDUCT.md" class="footer-link">Contributor Covenant Code of Conduct</a> | <a href="#/legal/PRIVACY_POLICY.md" class="footer-link">PRIVACY_POLICY.md</a> | <a href="#/legal/TERMS_AND_CONDITIONS.md" class="footer-link">TERMS AND CONDITIONS</a>

© 2025 KNIRV Network
</div>
