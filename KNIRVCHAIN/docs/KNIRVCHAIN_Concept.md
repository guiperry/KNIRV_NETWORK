# KNIRVBASE Solution Design Document

## 1. Introduction

**KNIRVBASE** is a specialized, private blockchain designed to function as a persistent "Long-Term Memory" (LTM) for Large Language Models. By leveraging the **Model Context Protocol (MCP)**, it allows any compatible LLM to read from and write to a secure, immutable ledger of user experiences, facts, and insights.

### 1.1 Purpose

To provide a decentralized (or private-node) storage layer where memories are not just stored as raw text, but as structured objects (GLB) that can be categorized and bridged to **KNIRVGRAPH** for relational analysis.

---

## 2. System Architecture

The system consists of three primary layers: the **Interface Layer** (MCP Server), the **Core Logic Layer** (The Blockchain), and the **Integration Layer** (KNIRVGRAPH Bridge).

### 2.1 High-Level Component Diagram

* **MCP Server:** The gateway for LLMs. It exposes "tools" to the LLM for memory retrieval and storage.
* **KNIRVBASE Core:** A Proof-of-Authority (PoA) or private consensus ledger that stores blocks containing GLB files.
* **Memory Classifier:** An internal logic engine that parses incoming data into specific classes.
* **KNIRVGRAPH Bridge:** An outbound transaction handler that sends specific memory types to the KNIRVGRAPH API.

---

## 3. Data Specification

### 3.1 Storage Format (GLB)

All primary memory payloads are stored in the **GLB (Binary glTF)** format.

* **Why GLB?** While traditionally for 3D data, GLB allows for a self-contained binary package. In KNIRVBASE, the GLB can encapsulate spatial context, metadata, and even 3D representations of conceptual "memory spaces."
* **Structure:** Each block contains a `blob` field holding the GLB data and a `header` containing the cryptographic hash and metadata.

### 3.2 Block Structure

| Field | Type | Description |
| --- | --- | --- |
| `block_id` | UUID | Unique identifier for the memory block. |
| `timestamp` | Uint64 | Unix timestamp of memory creation. |
| `payload_hash` | String | SHA-256 hash of the GLB file. |
| `data` | Binary (GLB) | The actual memory content. |
| `category` | Enum | Errors, Context, Ideas, or General. |
| `prev_hash` | String | Reference to the previous block. |

---

## 4. MCP Server Implementation

The KNIRVBASE functions as an MCP Host. It provides the following tools to any connected LLM:

* **`store_memory(content, type)`**: Takes text/data, packages it into a GLB container, and commits it to the chain.
* **`retrieve_memory(query)`**: Performs a semantic search across the blockchain and returns relevant GLB metadata/content.
* **`sync_graph()`**: Manually triggers a push of pending categorized transactions to KNIRVGRAPH.

---

## 5. Logic & Classification

### 5.1 Memory Categorization

Upon the creation of a new memory, the internal engine evaluates the content. If it matches specific criteria, it is flagged for KNIRVGRAPH:

1. **Errors:** Logic failures, API timeouts, or incorrect LLM outputs.
2. **Context:** Environmental data, user preferences, or situational history.
3. **Ideas:** Creative sparks, hypotheses, or "to-be-explored" concepts.

### 5.2 KNIRVGRAPH Interface

When a memory is classified as one of the three types above, the **KNIRVBASE Bridge** initiates a transaction:

> `POST /api/v1/transaction`
> **Payload:** `{ source: "KNIRVBASE", type: "IDEA", data: [GLB_REF], timestamp: 1734612252 }`

---

## 6. Security and Privacy

* **Encryption at Rest:** All GLB files are encrypted using the user's private key before being written to the chain.
* **Access Control:** Only LLMs providing a valid MCP-Token authorized by the user can decrypt and read the memory stream.
* **Immutability:** Once a memory is "hashed" into KNIRVBASE, it cannot be altered, ensuring a truthful audit log of the AI's "thought process."

---

## 7. Future Roadmap

* Expansion of memory classes (e.g., "Tasks," "Emotional State").
* Zero-Knowledge Proofs (ZKP) for sharing memories between different users' KNIRVBASEs without revealing raw data.
* Native 3D visualization of GLB memories within a VR/AR interface.

---

