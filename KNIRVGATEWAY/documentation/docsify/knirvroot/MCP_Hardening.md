

---

**Source**: KNIRVROOT/docs/TODOs/MCP_Hardening.md

# MCP Hardening
Here are a few areas and potential considerations that you might want to double-check or think about, building upon what you've already designed:

**Capability Descriptor Updates & Versioning:**

*   While plugin updates are mentioned in the developer docs (registering a new version), formalize how updates to any capability descriptor are handled.
*   Does an update create a new on-chain entry, effectively versioning the capability? How do clients discover the "latest" vs. a "specific" version of a capability?
*   Consider if the `MCPUpdateCapabilityTransaction` mentioned in `Model_Context_Protocol.md` (Future Work) should be prioritized to handle this explicitly.

**ContextRecord Granularity and Querying:**

*   The current `ContextRecord` structure is good. As the system grows, you might find needs for more specific `InteractionTypes` or more structured `Details` fields.
*   The idea of a supplemental RealmDB for rich querying (`Model_Context_Protocol.md`) is excellent for future scalability of analytics. For now, ensure your LevelDB indexing for `ContextRecords` (`mcp:context:<tx_id>`, `mcp:capability:<id>/invocations`) is robust enough for common audit and billing use cases.

**Plugin Security (Client-Side):**

*   This is correctly identified as CRITICAL in `2_Plugins_Implementation.md` and `8_Core_Plugin_Dev_Documentation.md`.
*   Continuously emphasize and perhaps provide reference implementations or best-practice guides for client-side sandboxing, as the responsibility ultimately lies with the client application developer.

**Plugin Manifest Adoption and Evolution:**

*   The `agent-plugin-manifest.yaml` is a standout feature. Consider how to version the manifest specification itself.
*   How will clients be guided to correctly interpret and utilize these manifests, especially the `clientInjection` points? SDK support for manifest parsing and flow execution could be very helpful.

**Finalizing Pending Registrations (Error Handling & TTL):**

*   The `Enhanced_Registration_Hardening.md` correctly points out the need for TTL and cleanup of `pendingRegistrations`. This is vital.
*   What happens if a client initiates, the server reserves the ID and returns `pendingTransactionHash`, but the client never finalizes or the finalization fails (e.g., insufficient fee at that moment)? Is the ID locked indefinitely or until TTL? Clarify the "un-reserving" process.

**Gas Fee Mechanism for Invocation:**

*   The `BaseDescriptor.GasFeeNRN` specifies the fee to invoke a capability. How is this fee actually collected and transferred during an `MCPInvokeCapabilityTransaction`?
*   Does the `Transaction.Fee` field of the `MCPInvokeCapabilityTransaction` need to be at least `GasFeeNRN`?
*   Is the fee automatically transferred from the Initiator to the Owner of the capability by the blockchain logic when the `ContextRecord` is processed? This needs to be explicitly part of the transaction processing logic in `BlockchainStruct.AddBlock` or similar. The payment processor doc (`3_Payment_Processor_Implementation.md`) is for buying NRN, not for on-chain fee mechanics.

**Data Privacy in ContextRecords:**

*   While hashes provide some obfuscation, if `InputHash` or `OutputHash` correspond to sensitive data, or if the `Details` field contains PII, this could be a concern.
*   For future consideration: mechanisms for selective disclosure or zero-knowledge proofs if highly sensitive interactions need to be logged on a public (even if permissioned) chain. For now, clear documentation on what gets logged is important.

**DHT Robustness and Incentives:**

*   How are bootstrap nodes incentivized to remain stable for the private DHT?
*   Consider strategies for DHT health monitoring and potentially adding/removing bootstrap nodes over time through a governance mechanism.

**SDK Functionality - Beyond Fetching:**

*   The SDKs are planned for URI parsing and resource fetching. Will they also include helpers for:
    *   Constructing and signing MCP transactions (especially for the two-step registration)?
    *   Interpreting plugin manifests?
    *   Interacting with the `pendingRegistrations` flow?

**Governance for MCP Standards & Capability Types:**

*   As the ecosystem evolves, new `CapabilityTypes` or `InteractionTypes` for `ContextRecords` might be needed. How will these be proposed, standardized, and adopted by the network?

**Atomicity of Finalization and DHT Announcement:**

*   In `handleMCPRegisterCapabilityFinalize`, if `AddTransactionToTransactionPool` succeeds but the subsequent DHT announcement fails, the capability is registered but not immediately discoverable.
*   The suggestion for a retry mechanism or a periodic scanner for unannounced capabilities (`Enhanced_Registration_Hardening.md`) is good. This could be a background service on nodes.

Overall, your MCP implementation is shaping up to be very thorough. The two-step registration and the plugin manifest system are particularly strong points. Addressing the fee mechanics for invocation and ensuring robust handling of the pending registration lifecycle will be important next steps. Keep up the excellent work!
```

---

<div class="footer-links">


© 2025 KNIRV Network
</div>
