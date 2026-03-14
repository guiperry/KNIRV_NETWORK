# KNIRVCHAIN Major Refactor Plan: LLM Memory Blockchain

## 1. Vision: A Cross-Model Memory Bank

This document outlines a major refactoring of the KNIRVCHAIN application to transform it into a specialized **LLM Memory Blockchain**. The primary use-case is to provide a secure, private, and persistent memory bank for users, accessible across any LLM that integrates with the Model Context Protocol (MCP).

The core idea is to shift from a system of active "agents" to a repository of passive but structured "memories". These memories will be stored on-chain in the form of `GLB` files, enabling rich, 3D-native representations of information.

## 2. Core Architectural Changes

### 2.1. From "Agent" to "Memory"

The concept of an "Agent" will be completely removed and replaced with "Memory". This is not just a naming change but a fundamental shift in the architecture.

*   **New `Memory` Struct:** A new, simpler `Memory` struct will be created. It will replace the current `Agent` struct.

    ```go
    // In a new file, e.g., internal/memory/memory.go
    package memory

    import "time"

    type Memory struct {
        ID          string                 `json:"id"`
        Owner       string                 `json:"owner"`
        Name        string                 `json:"name"`
        Description string                 `json:"description"`
        CreatedAt   time.Time              `json:"created_at"`
        // CID for the off-chain GLB file (e.g., on IPFS)
        GLB_CID     string                 `json:"glb_cid"`
        // Small, on-chain metadata
        Metadata    map[string]interface{} `json:"metadata"`
        // Classification of the memory
        Classification string                `json:"classification,omitempty"` // "Error", "Context", "Idea"
    }
    ```

*   **Refactor `internal/agent` to `internal/memory`:**
    *   The `internal/agent` directory will be renamed to `internal/memory`.
    *   `AgentManager` will become `MemoryManager`. Its responsibility will be simplified to creating, retrieving, and managing `Memory` objects. All complex logic related to agent capabilities, plugins, and consensus will be removed.

### 2.2. On-Chain Storage: GLB via Off-Chain CIDs

To handle the `GLB` file format efficiently and avoid bloating the blockchain, we will adopt an off-chain storage pattern.

*   **Storage Mechanism:** The raw `GLB` file data will NOT be stored directly in a transaction. Instead, it will be stored on a decentralized file system like IPFS.
*   **On-Chain Data:** The transaction's `Data` field will contain the `GLB_CID` (Content Identifier) of the file on the decentralized storage system. The `Memory` struct will store this CID.
*   **Transaction Type:** A new transaction type, e.g., `memory_create`, will be introduced. The data of this transaction will contain the `Memory` object (including the `GLB_CID`).

### 2.3. KNIRVCHAIN as an MCP Server for Memories

The chain will expose an MCP server to allow external LLMs to interact with a user's memories.

*   **Refactor `internal/mcp`:** The `MCPProcessor` will be updated to handle memory-related capabilities.
    *   **New Capabilities:** New MCP capabilities will be defined for memories:
        *   `register_memory`: Creates a new memory on the blockchain.
        *   `get_memory`: Retrieves a memory by its ID.
        *   `query_memories`: Queries memories based on metadata.
    *   The MCP server will be the primary interface for LLMs to access the memory bank.

### 2.4. Integration with KNIRVGRAPH

The internal graph implementation will be replaced with an interface to an external `KNIRVGRAPH` application.

*   **Refactor `internal/graph`:**
    *   The `NodeStore`, `GraphQueries`, and `RelationshipManager` will be rewritten.
    *   Instead of accessing local LevelDB and ChromaDB, they will become API clients that make HTTP requests to the `KNIRVGRAPH` server.
    *   This decouples the `KNIRVCHAIN` from the graph implementation, allowing each to evolve independently.

## 3. New Features

### 3.1. Memory Classification

The KNIRVCHAIN will classify certain memories and forward them to the `KNIRVGRAPH`.

*   **Classification Logic:** A new component, the `MemoryClassifier`, will be introduced. This component will be responsible for analyzing new memories and classifying them into one of three initial categories: `Errors`, `Context`, and `Ideas`.
*   **Transaction to `KNIRVGRAPH`:** When a memory is classified as an Error, Context, or Idea, the `MemoryClassifier` will create a new transaction targeted at the `KNIRVGRAPH`. This transaction will contain the classified memory data, instructing the `KNIRVGRAPH` to process and incorporate it into its graph.
    *   This will likely involve a new transaction type, e.g., `graph_add_node`.

## 4. Refactoring Roadmap

This refactoring will be executed in phases:

### Phase 1: Core Data Structure and MCP Refactoring

1.  **Implement `Memory` struct:** Create the new `Memory` struct and the `internal/memory` package.
2.  **Refactor `AgentManager` to `MemoryManager`:** Simplify the manager to handle basic CRUD operations for `Memory` objects.
3.  **Update MCP:**
    *   Define the new memory-related MCP capabilities.
    *   Update the `MCPProcessor` to handle these new capabilities.
4.  **Introduce `memory_create` transaction:** Create the new transaction type for adding memories.

### Phase 2: Graph Integration

1.  **Develop `KNIRVGRAPH` client:** Rewrite `internal/graph` to act as an API client for the `KNIRVGRAPH` application.
2.  **Abstract graph interactions:** Ensure all graph-related operations in the codebase go through the new `KNIRVGRAPH` client.
3.  **Remove old graph implementation:** Delete the old `NodeStore` and related database dependencies (LevelDB/ChromaDB for the graph).

### Phase 3: Memory Classification and `KNIRVGRAPH` Transactions

1.  **Implement `MemoryClassifier`:** Create the new component for classifying memories.
2.  **Implement transaction to `KNIRVGRAPH`:** Create the mechanism to send "Error", "Context", and "Idea" classified memories as transactions to the `KNIRVGRAPH`.
3.  **End-to-end testing:** Test the full flow from memory creation, classification, to its appearance in the `KNIRVGRAPH`.

### Phase 4: Cleanup and Deprecation

1.  **Remove all "agent" references:** Perform a full codebase search and remove any remaining "agent" terminology and code.
2.  **Remove unused packages:** Delete any packages that are no longer needed after the refactoring (e.g., parts of the old agent system).
3.  **Update documentation:** Update all relevant documentation, including `README.md`, to reflect the new architecture.
