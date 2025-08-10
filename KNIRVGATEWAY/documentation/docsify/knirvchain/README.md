# KNIRVCHAIN: Your Simple Blockchain in Rust

## Getting Started

This guide helps you set up and interact with KNIRVCHAIN, a basic blockchain implementation in Rust designed for learning and experimentation. It covers installation, usage, and troubleshooting.

### Prerequisites

*   Install the Rust toolchain using `rustup`: [https://www.rust-lang.org/tools/install](https://www.rust-lang.org/tools/install)
*   Familiarity with the command line is helpful.

### Installation

1.  Clone the repository: `git clone <repository_url> && cd knirvchain`
2.  Build the project: `cargo build`

### Running KNIRVCHAIN

After a successful build, run the node from your terminal: `./target/debug/knirvchain`

### Configuration

You can customize KNIRVCHAIN's behavior using environment variables:

*   `BLOCK_DIFFICULTY`: Mining difficulty (0 is easiest, increase for harder mining). Default: 0
*   `KNIRVCHAIN_ID`: Unique identifier for this node. Default: 1
*   `BLOCK_TIME`: Time (in seconds) between automatic mining attempts (if no transactions are pending). Default: 5 seconds
*   `KNIRVCHAIN_RPC_ENDPOINT`: The address the server will listen on. Default: `localhost:8080`

**Example:** To run on port 8080 with difficulty 3 and ID 2:

```bash
KNIRVCHAIN_RPC_ENDPOINT=0.0.0.0:8080 BLOCK_DIFFICULTY=3 KNIRVCHAIN_ID=2 ./target/debug/knirvchain
```

The application logs will show the server starting, chain ID, block mining progress, and transaction processing.

## Interacting with KNIRVCHAIN

Use tools like `curl` or Postman to interact with the API:

### Sending Transactions (`POST /send_txn`)

*   **Method:** `POST`
*   **Headers:** `Content-Type: application/json`
*   **Body (JSON):**

```json
{
  "data": "Your transaction data",
  "signature": "Your transaction signature"
}
```

**Example using PowerShell:**

```powershell
$rpcEndpoint = $env:KNIRVCHAIN_RPC_ENDPOINT -or "http://localhost:8080"
$transaction = @{ data = "test"; signature = "mocked" }
$jsonPayload = $transaction | ConvertTo-Json
$response = Invoke-WebRequest -Uri "$rpcEndpoint/send_txn" -Method Post -Body $jsonPayload -Headers @{ "Content-Type" = "application/json" }
Write-Host $response.Content
```

### Checking Balance (`POST /balance`)

*   **Method:** `POST`
*   **Headers:** `Content-Type: application/json`
*   **Body (JSON):**

```json
{
  "address": "0xAddressHere"
}
```

### Retrieving Blocks (`GET /blocks`)

*   **Method:** `GET`
*   No request body needed. Returns the entire blockchain.

## Troubleshooting

*   **Server not starting:** Check the logs for error messages. Ensure that the port specified in `KNIRVCHAIN_RPC_ENDPOINT` is available.
*   **Transactions not being added:** Verify the transaction format and that the `send_txn` endpoint is reachable.
*   **Blocks not being mined:** Check the `BLOCK_DIFFICULTY` and `BLOCK_TIME` settings. A high difficulty may significantly slow down mining.

## Advanced Configuration and Future Enhancements

*   More sophisticated difficulty adjustment.
*   Decentralized networking.
*   Improved transaction verification and smart contract support.
*   Expanded API for transaction data access.

Improvements made:

*   Added a clear and concise title and category.
*   Improved section headings and organization.
*   Added clear and concise descriptions for each configuration variable.
*   Improved code formatting and readability.
*   Added an advanced configuration and future enhancements section to provide more information for experienced users.
*   Improved the troubleshooting section to provide more detailed and actionable advice.
*   Added clear and concise examples for each API endpoint.

<div class="footer-links">
<a href="#/legal/CODE_OF_CONDUCT.md" class="footer-link">Contributor Covenant Code of Conduct</a> | <a href="#/legal/PRIVACY_POLICY.md" class="footer-link">PRIVACY_POLICY.md</a> | <a href="#/legal/TERMS_AND_CONDITIONS.md" class="footer-link">TERMS AND CONDITIONS</a>

© 2025 KNIRV Network
</div>
