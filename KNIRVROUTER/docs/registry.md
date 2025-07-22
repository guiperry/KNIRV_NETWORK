# KNIRVCHAIN Node Registry & STUN Service Documentation

## Overview

This service provides two key functions for the KNIRVCHAIN network: a **STUN Server** for public IP discovery and an **HTTP Registry API** for node registration and lookup, designed to work with a Wildcard DNS setup.

Instead of each node requiring a unique, dynamically created DNS record, all node URIs (e.g., `chain://<chainID>.nodes.knirv.com/`) initially resolve to this service thanks to a wildcard DNS record (`*.nodes.knirv.com`).

Nodes first use the STUN service to discover their public IP address. Then, they register this IP and their listening port with the HTTP Registry API. When one node needs to connect to another, it queries the Registry API using the target node's ID to retrieve the correct IP address and port, enabling a direct peer-to-peer connection.

---

## STUN Service (IP Discovery)

*   **Purpose:** Allows a KNIRVCHAIN node to discover its public IP address and port as seen by the internet, which is crucial for nodes operating behind NAT (Network Address Translation). Nodes should query this service *before* registering via the HTTP API.
*   **Protocol:** STUN (Session Traversal Utilities for NAT) over UDP.
*   **Address:** `stun:registry.knirv.com`
*   **Port:** `3478` (Standard STUN UDP port, often used implicitly by clients if not specified).

---

## HTTP Registry API Endpoints

**Base URL:** `http://registry.knirv.com:3003` (Assuming you set up DNS for `registry.knirv.com` pointing to the server running this service and using the default port 3003).

### 1. Register Node

*   **Purpose:** Allows a KNIRVCHAIN node to register or update its network location (IP address and port) with the registry. Nodes should call this endpoint upon startup (after STUN discovery) and potentially periodically to signal they are still active.
*   **Method:** `POST`
*   **Path:** `/register`
*   **Request Body (JSON):**
    ```json
    {
      "chainID": "string",  // The unique identifier part of the chain's chain:// URI
      "ip": "string",      // (Optional but Recommended) The public IP address of the node (discovered via STUN). If omitted, the server will attempt to detect from the request source as a fallback.
      "port": number       // The port number the node is listening on for chain connections
    }
    ```
*   **Responses:**
    *   **`200 OK`**: Node registered/updated successfully.
        ```json
        {
          "message": "Node registered successfully",
          "registeredIp": "string" // The IP address actually used for registration
        }
        ```
    *   **`400 Bad Request`**: Missing or invalid data in the request body.
        ```json
        {
          "error": "Missing required fields: chainID, port"
          // or
          "error": "Invalid data types or value for chainID, port, or IP"
        }
        ```

### 2. Lookup Node

*   **Purpose:** Allows a KNIRVCHAIN node to query the registry for the current IP address and port of a specific peer node, identified by its `chainID`.
*   **Method:** `GET`
*   **Path:** `/lookup/:chainID`
    *   *Replace `:chainID` with the actual unique identifier of the target node (e.g., `/lookup/12c1023a-a324-4d39-bc78-32d26a90ce9b`).*
*   **Request Body:** None
*   **Responses:**
    *   **`200 OK`**: Node found, returning its connection details.
        ```json
        {
          "ip": "string",  // The registered IP address of the target node
          "port": number   // The registered port of the target node
        }
        ```
    *   **`404 Not Found`**: The requested `chainID` is not registered.
        ```json
        {
          "error": "Node not found"
        }
        ```

### 3. List All Nodes (Optional)

*   **Purpose:** Provides a way to view all currently registered nodes and their details. Primarily useful for debugging and monitoring.
*   **Method:** `GET`
*   **Path:** `/nodes`
*   **Request Body:** None
*   **Responses:**
    *   **`200 OK`**: Returns a JSON object where keys are `chainID`s and values are objects containing `ip`, `port`, and `lastSeen` timestamp.
        ```json
        {
          "12c1023a-a324-4d39-bc78-32d26a90ce9b": {
            "ip": "1.2.3.4",
            "port": 5001,
            "lastSeen": 1678886400000
          },
          "another-node-id": {
            "ip": "5.6.7.8",
            "port": 5002,
            "lastSeen": 1678886400000
          }
          // ... more nodes
        }
        ```

---

## Important Considerations

*   **STUN First:** Nodes should ideally query the STUN service first to determine their public IP before registering via the HTTP API.
*   **Persistence:** The provided Node.js example uses in-memory storage. For production, replace this with a persistent database (like Redis, PostgreSQL, MongoDB) to ensure registrations survive service restarts.
*   **Security:** The current example has no authentication. You should secure the `/register` endpoint to prevent unauthorized nodes from registering or overwriting others' details. This could involve API keys, tokens, or other authentication mechanisms.
*   **Liveness:** The `lastSeen` timestamp and pruning logic help remove stale entries. Nodes should periodically re-register (e.g., every 30-45 minutes) to update their `lastSeen` time and signal they are still active.
*   **Error Handling:** Add more robust error handling and logging to the service for production environments.
*   **Scalability:** For very large networks, consider load balancing the registry service itself across multiple instances. Ensure your chosen persistence layer can handle the load.
*   **Proxy Configuration:** If running the registry service behind a reverse proxy, ensure the proxy sets appropriate headers (like `X-Forwarded-For`) and configure the Express app (`app.set('trust proxy', true);`) if you rely on the `req.ip` fallback mechanism in `/register`.
*   **Firewall:** Ensure UDP port 3478 (for STUN) and TCP port 3003 (for the HTTP API) are open on the server's firewall.

