# KNIRVWALLET User Guide

Welcome to KNIRVWALLET, your gateway to the KNIRV Decentralized Trusted Execution Network (D-TEN)! This guide helps you install, configure, and use KNIRVWALLET on mobile and in your browser.

## Getting Started

KNIRVWALLET offers two interfaces: a mobile app (Agentic Wallet) and a browser extension/web app (Browser Wallet). Both provide access to your NRN tokens, AI agents, and the broader KNIRV ecosystem.

### Prerequisites

Before you begin, ensure you have the following installed:

* Node.js 18.14.2+
* Go 1.21+ (for Agentic Wallet backend only)
* PostgreSQL 13+ and Redis 6+ (for Agentic Wallet backend only)
* Yarn package manager

### Installation

#### Step 1: Clone the Repository

```bash
git clone <repository-url>
cd KNIRVWALLET
```

#### Step 2: Browser Wallet Setup

```bash
cd browser-wallet
yarn install
yarn build
```

Install the extension by loading the unpacked extension in your browser (Chrome: `chrome://extensions/`; Firefox: `about:debugging`).

#### Step 3: Agentic Wallet Setup

```bash
cd agentic-wallet
yarn install
cd go-backend
make deps
cp .env.example .env  # Edit .env with your configuration (see below)
make migrate-up
cd ..
yarn dev
```

### Configuration

**XION Integration:** You'll need to configure XION contract addresses. Copy the `.env.example` file to `.env` in the `agentic-wallet` directory and replace placeholders like `EXPO_PUBLIC_NRN_CONTRACT` and `EXPO_PUBLIC_FAUCET_CONTRACT` with your actual contract addresses.

### Troubleshooting

#### Common Issues

* **Installation Issues:** Ensure you have all prerequisites installed correctly. Check the console for error messages.
* **Connectivity Problems:** Verify your internet connection and that the necessary ports are open.
* **XION Integration Errors:** Double-check your XION contract addresses in the `.env` file.

#### Advanced Troubleshooting

For more complex issues, refer to the individual READMEs within each implementation directory or join the KNIRV community for further assistance.

## Using KNIRVWALLET

### Key Features

* **Multi-Chain Support:** Manage assets across BTC, ETH, Solana, and KNIRV-ORACLE.
* **NRN Token Management:** Send, receive, and burn NRN tokens.
* **AI Agent Management:** Control and interact with your KNIRV-CORTEX agents.
* **Secure Authentication:** Biometric login and hardware wallet (Ledger) support.
* **Cross-Platform Synchronization:** Seamlessly access your wallet from mobile and browser.

### Common Tasks

* **Creating a Wallet:** Follow the instructions within the application for creating a new wallet.
* **Managing NRN Tokens:** Use the in-app interface to check your balance, send, and receive NRN tokens.
* **Interacting with Agents:** Use the provided tools to execute your AI agents and submit data to the NRV system.

## Support

For further assistance, please refer to the individual READMEs within each implementation directory, open an issue on GitHub, or join the KNIRV community. Report security issues privately to the maintainers.

Improvements Needed:

1. **Simplify Prerequisites:** Break down the prerequisites into smaller, more manageable sections.
2. **Improve Installation Instructions:** Use clear, concise language and provide step-by-step instructions for each interface (mobile and browser).
3. **Enhance Troubleshooting Section:** Add more specific examples and solutions for common issues.
4. **Add Visual Aids:** Incorporate diagrams, screenshots, or videos to illustrate key concepts and features.
5. **Streamline Configuration Section:** Provide a clear, step-by-step guide for configuring XION contract addresses.
6. **Update Support Section:** Add more contact information and resources for users to access further assistance.

<div class="footer-links">
<a href="#/legal/CODE_OF_CONDUCT.md" class="footer-link">Contributor Covenant Code of Conduct</a> | <a href="#/legal/PRIVACY_POLICY.md" class="footer-link">PRIVACY_POLICY.md</a> | <a href="#/legal/TERMS_AND_CONDITIONS.md" class="footer-link">TERMS AND CONDITIONS</a>

© 2025 KNIRV Network
</div>
