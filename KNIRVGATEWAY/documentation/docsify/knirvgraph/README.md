# KNIRVGRAPH User Guide

## Introduction

Welcome to the KNIRVGRAPH User Guide, a comprehensive resource for setting up, using, and troubleshooting the KNIRVGRAPH application, a graph-based blockchain explorer. This guide is designed to help both end-users and developers successfully use KNIRVGRAPH.

## Getting Started

### Installation

Before you begin, ensure you have the following prerequisites installed:

* Go 1.21+
* Node.js 18+
* npm or yarn

To install KNIRVGRAPH, follow these steps:

1. Clone the repository using `git clone [repository_url]`.
2. Navigate to the `graphchain-app` directory and run `make deps` followed by `make build`.
3. In the `src` directory, run `npm install`.
4. Start the application by opening two terminal windows. In the first, run `./build/graphchain-node -home ./data -rpc-port 8080`. In the second, run `npm run dev`.
5. Access the application by visiting `http://localhost:5173` for the frontend and `http://localhost:8080` for the backend API.

### Prerequisites

* Ensure you have the necessary dependencies installed, including Go, Node.js, and npm or yarn.
* Verify that ports 8080 and 5173 are accessible.

### Troubleshooting Installation

* Check the backend logs for error messages.
* Verify that the `make deps` and `make build` commands completed successfully.
* Ensure that the `npm install` command completed without errors.

## Using KNIRVGRAPH

### Exploring the Dashboard

The KNIRVGRAPH explorer provides a user-friendly interface to interact with the graph database. You can:

* View real-time graph statistics.
* Browse and search for nodes within the graph.
* Examine transaction details.
* Check account balances and transaction history.
* Interact with a visual representation of the graph structure.
* Discover routes between nodes.

### Using the Command-Line Interface (CLI)

The CLI provides additional functionality for interacting with the KNIRVGRAPH network. Key commands include:

* `height`: Get the current blockchain height.
* `node {nodeID}`: Get information about a specific node.
* `edge {edgeID}`: Get information about a specific edge.
* `heads`: Get the current graph heads.
* `neighbors {nodeID}`: Get the neighbors of a node.
* `path {node1} {node2}`: Find a path between two nodes.
* `create-node`: Create a new node.
* `create-edge`: Create a new edge.
* `account {address}`: Get account balance.
* `send-tx`: Send a transaction.

### API Reference

The KNIRVGRAPH backend exposes a REST API for programmatic access. Key endpoints include:

* `/height`: Get current GraphChain height (GET)
* `/node/{nodeID}`: Get node by ID (GET)
* `/edge/{edgeID}`: Get edge by ID (GET)
* `/graph/heads`: Get current graph heads (GET)
* `/graph/neighbors/{nodeID}`: Get node neighbors (GET)
* `/graph/path/{from}/{to}`: Find path between nodes (GET)
* `/account/{address}`: Get account information (GET)
* `/transaction`: Submit new graph transaction (POST)
* `/node`: Create new graph node (POST)
* `/edge`: Create new graph edge (POST)

### Troubleshooting

* **Backend Errors:** Check the backend logs for error messages.
* **Frontend Errors:** Check the browser's developer console for errors.
* **Network Issues:** Ensure that the backend and frontend are communicating correctly.
* **Database Issues:** Check the BluntDB configuration and ensure sufficient disk space.

## Configuration

The backend configuration is managed via `config/default.toml`. The frontend automatically connects to `http://localhost:8080`; modify `src/services/api.ts` to change this.

## Contributing

Contributions are welcome! Please refer to the original README's contribution guidelines.

## License

MIT License (see LICENSE file).

Improvements Needed:

* Add more detailed explanations for each section.
* Provide additional troubleshooting steps for common issues.
* Consider adding a FAQ section.
* Update the API reference to include more detailed information about each endpoint.
* Improve the overall formatting and organization of the guide.

<div class="footer-links">
<a href="#/legal/CODE_OF_CONDUCT.md" class="footer-link">Contributor Covenant Code of Conduct</a> | <a href="#/legal/PRIVACY_POLICY.md" class="footer-link">PRIVACY_POLICY.md</a> | <a href="#/legal/TERMS_AND_CONDITIONS.md" class="footer-link">TERMS AND CONDITIONS</a>

© 2025 KNIRV Network
</div>
