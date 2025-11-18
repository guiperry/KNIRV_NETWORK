 The `ContextRecord` and its associated endpoints are really the heart of the "verifiable audit trail" aspect of your Model Context Protocol (MCP) implementation. They provide the on-chain proof of what happened, when, by whom, and with what (at least in terms of references like hashes and fees).

Essentially, a `ContextRecord` is created and logged as part of an `MCPInvokeCapabilityTransaction` every time a registered MCP capability (like a Tool, Prompt, Plugin, or Memory Service) is used or interacted with, or when a significant MCP event like a capability registration occurs.

Here are some scenarios where you'd utilize a `ContextRecord` and its endpoints:

**General Purpose of ContextRecords:**

*   **Immutable Audit Trail:** To have an undeniable, on-chain record of every significant interaction with an MCP capability.
*   **Verifiability:** To allow anyone to check the history of a capability's usage or an initiator's actions.
*   **Billing/Accounting:** To track NRN token fee payments for capability usage.
*   **Provenance:** To trace the lineage of data or models generated through a series of MCP interactions.
*   **Debugging:** To reconstruct the sequence of operations if an AI system behaves unexpectedly.
*   **Analytics:** To understand how capabilities are being used.

Let's dive into specific scenarios based on the MCP primitives:

**1. Using a Registered Tool**

*   **Scenario:** An AI agent, "AgentX," needs to find the current price of "NRN\_TOKEN". It uses a registered MCP Tool called "CryptoPriceOracleTool" (ID: `tool-crypto-price-001`).
*   **Action:** AgentX (or the client application acting on its behalf) would make a call that results in an `MCPInvokeCapabilityTransaction`.
    *   This would likely be via an endpoint like `POST /wallet/mcp/create_invoke_capability` (if the wallet server handles signing) or directly to `POST /mcp/invoke/tool/tool-crypto-price-001` (if the client signs).
*   **ContextRecord Created:**
    *   `CapabilityID`: `"tool-crypto-price-001"`
    *   `InteractionType`: `"TOOL_INVOCATION"`
    *   `Initiator`: AgentX's KNIRVORACLE address.
    *   `Timestamp`: Current time.
    *   `InputHash`: Hash of the input `{"token_symbol": "NRN_TOKEN"}`.
    *   `OutputHash`: Hash of the output `{"price_usd": 0.75}`.
    *   `Details`: Could include `{"input_params": {"token_symbol": "NRN_TOKEN"}, "status": "success"}`.
    *   The transaction would also include the `Fee` (NRN tokens) specified by `tool-crypto-price-001`.
*   **Endpoint Usage (Querying):**
    *   `GET /mcp/context/{tx_id_of_invocation}`: To retrieve the specific details of this price query.
    *   `GET /mcp/capability/tool-crypto-price-001/invocations`: The owner of the tool could use this to see all invocations of their tool, perhaps for billing reconciliation or usage analytics.
    *   `GET /mcp/contexts?initiator={AgentX_address}&interactionType=TOOL_INVOCATION`: To audit all tools AgentX has used.

**2. Utilizing a Registered Prompt**

*   **Scenario:** A user in an application wants to generate a short story. The application uses a registered MCP Prompt called "CreativeStoryPrompt" (ID: `prompt-story-gen-v2`).
*   **Action:** The application, on behalf of the user, triggers an `MCPInvokeCapabilityTransaction`.
*   **ContextRecord Created:**
    *   `CapabilityID`: `"prompt-story-gen-v2"`
    *   `InteractionType`: `"PROMPT_USAGE"`
    *   `Initiator`: The user's (or application's service) KNIRVORACLE address.
    *   `Timestamp`: Current time.
    *   `InputHash`: Hash of the parameters provided to the prompt template (e.g., `{"genre": "sci-fi", "character_name": "Zorp"}`).
    *   `OutputHash`: Hash of the generated story.
    *   `Details`: `{"prompt_params": {"genre": "sci-fi", "character_name": "Zorp"}}`.
*   **Endpoint Usage (Querying):**
    *   `GET /mcp/context/{tx_id_of_usage}`: To see the parameters used for a specific story generation instance.
    *   `GET /mcp/capability/prompt-story-gen-v2/invocations`: To track how often this prompt template is used.

**3. Executing a Registered Plugin (Resource)**

*   **Scenario:** A agent Client (your "Inference Engine") needs to validate a user-uploaded game mod file. It uses a registered MCP Plugin (a `ResourceDescriptor` with `ResourceTypePlugin`) called "SkyrimModValidatorPlugin" (ID: `plugin-skyrim-validator-007`). The plugin is downloaded and executed locally by the client as per `plugin_templates.md`.
*   **Action:** After local execution, the agent Client logs this usage by creating an `MCPInvokeCapabilityTransaction`.
*   **ContextRecord Created:**
    *   `CapabilityID`: `"plugin-skyrim-validator-007"`
    *   `InteractionType`: `"PLUGIN_EXECUTION"` (or `"RESOURCE_ACCESS"`)
    *   `Initiator`: The agent Client's KNIRVORACLE address.
    *   `Timestamp`: Current time.
    *   `InputHash`: Hash of the mod file that was validated.
    *   `OutputHash`: Hash of the validation report (e.g., the `ValidateAssetOutput` from your example).
    *   `Details`: `{"input_filename": "super_cool_sword.esp", "execution_status": "success", "errors_found": 0}`.
*   **Endpoint Usage (Querying):**
    *   `GET /mcp/capability/plugin-skyrim-validator-007/invocations`: The plugin developer uses this to see how many times their plugin has been invoked and to verify they've received the NRN fees.
    *   `GET /mcp/contexts?initiator={client_address}&capabilityId=plugin-skyrim-validator-007`: The client can use this to get a history of all mods it has validated with this specific plugin.

**4. Interacting with a Memory Service**

*   **Scenario:** An AI assistant, "HelperBot," learns a new user preference: "UserA dislikes reminders before 9 AM." It stores this in a registered MCP Memory Service (ID: `mem-user-prefs-global`).
*   **Action:** HelperBot triggers an `MCPInvokeCapabilityTransaction` to log the memory write.
*   **ContextRecord Created:**
    *   `CapabilityID`: `"mem-user-prefs-global"`
    *   `InteractionType`: `"MEMORY_WRITE"`
    *   `Initiator`: HelperBot's KNIRVORACLE address.
    *   `Timestamp`: Current time.
    *   `InputHash`: Hash of the data being written (e.g., `{"user": "UserA", "preference_type": "reminder_timing", "value": "dislikes_before_9am"}`).
    *   `Details`: `{"operation": "add_preference", "user_id_hash": "hash_of_UserA"}`.
*   **Endpoint Usage (Querying):**
    *   `GET /mcp/contexts?capabilityId=mem-user-prefs-global&interactionType=MEMORY_WRITE`: To audit all writes to this memory service.

Later, if HelperBot reads from this memory (`InteractionType`: `"MEMORY_READ"`), another `ContextRecord` would be logged.

**5. Logging a Sampling Event (Server-Side)**

*   **Scenario:** Your MCP server (acting as a sophisticated agent) needs to ask a client-side LLM (via the "Sampling" primitive) to brainstorm ideas for a marketing slogan. The server initiates this.
*   **Action:** The MCP server itself logs this event by creating an `MCPInvokeCapabilityTransaction` via `POST /mcp/log_sampling_event`.
*   **ContextRecord Created (for `SAMPLING_REQUEST_SENT`):**
    *   `CapabilityID`: Could be the server's own registered ID or a generic "sampling-service" ID.
    *   `InteractionType`: `"SAMPLING_REQUEST_SENT"`
    *   `Initiator`: The MCP server's KNIRVORACLE address.
    *   `Timestamp`: Current time.
    *   `InputHash`: Hash of the structured prompt/task sent to the client's LLM.
    *   `Details`: `{"target_client_id_hash": "hash_of_client_identifier", "purpose": "marketing_slogan_brainstorm"}`.
*   **Endpoint Usage (Querying):**
    *   `GET /mcp/contexts?interactionType=SAMPLING_REQUEST_SENT&initiator={server_address}`: To track all sampling requests initiated by this server.

A corresponding `"SAMPLING_RESPONSE_RECEIVED"` record would be logged when the client returns the results.

**6. Auditing Capability Registrations**

*   **Scenario:** You want to see a chronological log of all new capabilities (Tools, Plugins, etc.) registered on the KNIRVORACLE network.
*   **Action:** While the `MCPRegisterCapabilityTransaction` itself registers the capability, you could design your system so that a `ContextRecord` is also created to log this event for a unified audit trail of all MCP-related activities.
*   **ContextRecord Created (Optional, for enhanced auditing):**
    *   `CapabilityID`: The ID of the newly registered capability.
    *   `InteractionType`: `"CAPABILITY_REGISTRATION"`
    *   `Initiator`: The address of the developer/entity who registered the capability.
    *   `Timestamp`: Time of registration.
    *   `Details`: `{"capability_type": "TOOL", "registered_name": "MyNewTool"}`.
*   **Endpoint Usage (Querying):**
    *   `GET /mcp/contexts?interactionType=CAPABILITY_REGISTRATION`: To get a feed of all new capabilities.
    *   `GET /mcp/contexts?owner={developer_address}&interactionType=CAPABILITY_REGISTRATION`: To see all capabilities registered by a specific developer. (Note: owner is not a direct field in `ContextRecord` but could be inferred or queried if `CapabilityID` is used to look up the descriptor).

These scenarios illustrate that `ContextRecords` are versatile. They are not just about logging tool use but about creating a verifiable history for any defined interaction within your MCP ecosystem, which is crucial for transparency, billing, and building trust in complex AI systems. The GET endpoints then provide the means to 