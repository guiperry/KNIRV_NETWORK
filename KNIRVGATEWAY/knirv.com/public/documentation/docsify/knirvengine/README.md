# KNIRVENGINE User Guide: Setup, Usage, and Troubleshooting

KNIRVENGINE is a three-engine cognitive architecture for deploying and managing autonomous agents. It combines Human-like Reasoning Models (HRM), QR code linkage, and WebAssembly (WASM) for efficient cognitive processing. This guide helps you set up, use, and troubleshoot KNIRVENGINE.

## Prerequisites

Before installing KNIRVENGINE, ensure you have the following:

* Go 1.21+
* Node.js 18+
* Rust 1.70+
* Python 3.9+ (for HRM model)

## Installation

To install KNIRVENGINE, follow these steps:

1. **Clone the repository:**

```bash
git clone https://github.com/guiperry/KNIRV_NETWORK.git
cd KNIRV_NETWORK/KNIRVENGINE
```

2. **Build the components:**

```bash
cd desktop-client
go build -o desktop-client main.go

cd ../mobile-controller
npm install
npm run build

cd ../agent-core/rust-wasm
cargo build --release --target wasm32-unknown-unknown
```

## Running KNIRVENGINE

To run KNIRVENGINE, follow these steps:

1. **Start the Desktop Host:**

```bash
cd ../desktop-client
./desktop-client
```

2. **Start the Mobile Tool (development):**

```bash
cd ../mobile-controller
npm run dev
```

3. **Access the system:**

* Desktop Host: `http://localhost:8082`
* Mobile Tool: `http://localhost:5173`
* MCP WebSocket: `ws://localhost:8082/api/mcp/ws`

## Using KNIRVENGINE

KNIRVENGINE consists of three core engines:

* **Desktop-Host Engine:** Central coordination and HRM processing.
* **Mobile-Tool Engine:** Enhanced mobile client with sensory processing (QR code scanning, voice and visual processing).
* **Agent-Core Engine (WASM):** A lightweight cognitive shell for autonomous agents.

### QR Code Linkage

KNIRVENGINE uses QR codes for secure pairing between desktop and mobile devices. Two types of QR codes are used:

* **Target Assignment QR:** Links a mobile device to a specific system, authorizing agent deployment.
* **Transaction Signing QR:** Authorizes blockchain transactions.

### Human-like Reasoning Model (HRM)

The HRM provides advanced cognitive capabilities. You can send requests to the HRM via the API (see API Reference below).

### Model Context Protocol (MCP)

KNIRVENGINE uses MCP for standardized agent communication. You can interact with the MCP using a WebSocket connection (see example below).

## Troubleshooting

* **Desktop Host not starting:** Check your Go installation and ensure the HRM model is correctly located (`./dist/weights.safetensors`).
* **Mobile Tool errors:** Check your Node.js and npm installations. Ensure you have the necessary device permissions (camera access).
* **MCP connection issues:** Verify that the WebSocket endpoint (`ws://localhost:8082/api/mcp/ws`) is accessible.

## Configuration

You can configure various aspects of KNIRVENGINE (see detailed configuration options in the original README).

## API Reference

The full API reference is available in the original README.

## Deployment

For production deployment, build all components, configure environment variables, set up a reverse proxy (Nginx recommended), enable HTTPS/WSS, and configure monitoring and logging. A sample Dockerfile is provided in the original README.

## Integrated Wallet Functionality

KNIRVENGINE includes a wallet that integrates with the KNIRV ecosystem, supporting multiple cryptocurrencies and NRN tokens. It requires linking with KNIRVCONTROLLER for full functionality. See the original README for detailed information on wallet features, architecture, and usage.

## Support

For support and questions, please refer to the GitHub Issues or the KNIRV Network Discord community.

Improvements:

1. **Simplified installation instructions:** Break down the installation process into smaller, more manageable steps.
2. **Enhanced troubleshooting section:** Provide more detailed error messages and solutions for common issues.
3. **Improved API documentation:** Include more examples and use cases for the API endpoints.
4. **Clearer deployment instructions:** Provide a step-by-step guide for deploying KNIRVENGINE in a production environment.
5. **Consistent formatting:** Use a consistent formatting style throughout the guide to improve readability.

<div class="footer-links">
<a href="documentation/static/legal/CODE_OF_CONDUCT.html" class="footer-link">Contributor Covenant Code of Conduct</a> | <a href="documentation/static/legal/PRIVACY_POLICY.html" class="footer-link">PRIVACY_POLICY.md</a> | <a href="documentation/static/legal/TERMS_AND_CONDITIONS.html" class="footer-link">TERMS AND CONDITIONS</a>

© 2025 KNIRV Network
</div>
