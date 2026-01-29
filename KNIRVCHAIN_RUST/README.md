# KNIRVCHAIN: A Custom Blockchain Implementation in Rust

This project implements a custom blockchain in Rust, designed for learning and experimentation. It provides a basic foundation for understanding core blockchain concepts like block mining, transaction processing, and decentralized data storage. This version uses the `sled` embedded database for persistent storage and `actix-web` for the API. It leverages `tokio` for asynchronous tasks, and `tracing` for debugging information.

## Features

- **Basic Blockchain Structure:** Implements blocks, a blockchain, and basic hashing (SHA-256) with configurable difficulty for mining.
- **Transaction Handling:** Allows clients to submit transactions, which are then included in new blocks.
- **Automatic Block Mining:** New blocks are automatically created at a regular time interval or when new transactions are available in the pool.
- **Customizable Settings**: The application's behaviour is customized via environmental variables.
- **API Endpoints:**
  - `/send_txn` : Accepts a JSON payload representing a transaction to be added into the pool for mining. Returns the transaction hash once added to the pool.
  - `/balance` : Retrieves a user balance from the blockchain (currently using a mock response).
  - `/blocks`: Returns the current blockchain information in JSON format.
- **`sled` Embedded Database:** Stores blockchain data using the `sled` embedded database for persistent local storage.
- **`actix-web` framework**: A simple but complete http server using actix.
- **`tracing` library:** Collects data via custom events for debug.
- **Asynchronous Processing:** Uses tokio for async tasks to ensure performance and non-blocking IO, with use of thread safe primitives to ensure data consistency.

## Getting Started

1.  **Prerequisites:**

    - Rust toolchain installed (`rustup`): [https://www.rust-lang.org/tools/install](https://www.rust-lang.org/tools/install)
    - Basic understanding of Rust and command line tools

2.  **Clone the Repository:**

    ```bash
    git clone <repository_url>
    cd knirvchain
    ```

3.  **Build the Project:**
    ```bash
    cargo build
    ```
4.  **Run the KNIRVCHAIN Node:**

You can execute the compiled binary directly in a terminal after a successful build:

```bash
   ./target/debug/knirvchain


   You can customize various settings via environment variables:

   *   **`BLOCK_DIFFICULTY`**: Difficulty setting for the proof of work, default is 0 (easy mining, for testing), change it for higher numbers.

*  **`KNIRVCHAIN_ID`** Integer that is used as an identifier of this instance of KNIRVCHAIN, default is `1`

*   **`BLOCK_TIME`**: The time (in seconds) to wait between automatic mining attempts (when no transactions arrive), default is 5 seconds.

Here's an example to set the endpoint, chain id and difficulty when starting:

KNIRVCHAIN_RPC_ENDPOINT=0.0.0.0:8080 BLOCK_DIFFICULTY=3 KNIRVCHAIN_ID=2 ./target/debug/knirvchain
```

This example will configure the chain to:

- Run at address: `0.0.0.0:8080`
- Set Difficulty: `3`
- Set Chain ID: `2`

The logs will be visible directly on the command line after the application has started. You should see the `Starting Server`, the configured `KNIRVCHAIN Chain ID`, automatic blocks that are added when the timer completes, and then transactions that are processed when sent via the `/send_txn` endpoint.

## Interacting with KNIRVCHAIN

You can interact with your KNIRVCHAIN node by making API requests using clients like `curl`, `Postman`, or other API testing tools.

1.  **Sending Transactions (POST /send_txn):**

_Method:_ `POST`

_Headers:_
`Content-Type: application/json`

_Body:_ (JSON)

```json
{
  "data": "Your transaction data",
  "signature": "Your transaction signature"
}
```

_Example using PowerShell:_

````powershell
    # Configuration
    #If the environment variable KNIRVCHAIN_RPC_ENDPOINT is not set, set it to a default value.
     $rpcEndpoint = if ($env:KNIRVCHAIN_RPC_ENDPOINT) { $env:KNIRVCHAIN_RPC_ENDPOINT } else { "http://localhost:8000" }

     # Create a transaction object as a hashtable
    $transaction = @{
        data = "test transaction from powershell";
         signature = "mocked signature from powershell"
    }
    # Convert hashtable to JSON
   $jsonPayload = $transaction | ConvertTo-Json
   # Set up headers for the HTTP request.
  $headers = @{
     "Content-Type" = "application/json"
  }

   # Send the HTTP POST request using Invoke-WebRequest.
 try {
      $response = Invoke-WebRequest -Uri "$rpcEndpoint/send_txn" -Method Post -Body $jsonPayload -Headers $headers -UseBasicParsing
     Write-Host "Request to server successful."
     # Convert the JSON response to a Powershell object, and parse the message field
      $responseObject = $response.Content | ConvertFrom-Json;
    Write-Host "Response: $($responseObject.message)"

 } catch {
     Write-Host "Error sending request:"
   Write-Host $_.Exception.Message
}

1. Checking Balance (POST /balance):
Method: POST
Headers: Content-Type: application/json
Body:

```json
    {
    "address": "0xAddressHere"
 }
````

2. Retrieving Blocks (GET /blocks):
   Method: GET
   This endpoint requires no request body. It returns the whole blockchain.

**The application logs will include, in all those cases:**

    - Received transaction

    - Clearing transaction pool

    - Previous hash

    - Mining start/finish for individual blocks

    - Blocks added to the chain

    - Skips mining if the transaction pool is empty.

**Design Decisions:**
Asynchronous design: This is done using actix and tokio primitives which require extra attention in how locks and async calls are handled. By understanding asynchronous rust and the usage of primitives like .await, spawn and the thread safe data structures like tokio::sync::Mutex you can avoid common programming errors like deadlocks.

Immutability: Immutability is an underlying concept for better handling concurrency as that means that less data requires specific lock patterns. Data that does not require updates can be passed via cloneable data structures safely (such as configurations), or if a reference is sufficient (like with a cloned web::Data<>). This concept is widely used in functional programming.

Error Handling: Extensive usage of the Result<T, E> type to improve code quality, by addressing any errors that may be presented while also keeping a clean code with consistent function signatures, allowing for errors to bubble up in cases of failures, instead of having implicit handling of panics.

Performance: This version uses thread safe mutex locks that while guaranteeing correct results will have performance overhead when compared to lower level code with less abstraction. These are used here to focus on correct functionality for this demonstration project. You can consider using other low level lock strategies using primitives from the std::sync and std::sync::atomic crates, to fine-tune performance for any project requiring optimized synchronization.

Simple API: Actix is excellent to create a simple HTTP endpoint. Libraries with a reduced learning curve were selected to make it easy to understand and build on top of, instead of making it complex. For example, sled is much easier to learn than other embedded database frameworks.

**Future Enhancements**
Implement a proper difficulty adjustment mechanism.

Improve networking capabilities for decentralized operation.

Implement a more realistic transaction verification, and smart contracts to introduce more complete features (using the Xelis VM).

Extend API for more transaction data access.

**Contributions**
Contributions to KNIRVCHAIN are welcome! If you have ideas for improvements or find any issues, please feel free to submit pull requests or create issues. Make sure you are using rustfmt to apply rustfmt defaults on code and format the codebase, before opening pull requests.

License
This project is licensed under the MIT License - see the LICENSE file for details.
