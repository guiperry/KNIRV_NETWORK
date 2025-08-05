

---

**Source**: KNIRVROOT/docs/completedImplementations/Enhanced_Registration_Implementation.md

### Enhanced Registration Flow (Two-Step Signing):

**Client: Initiate Registration (POST /mcp/capability/register/initiate)**

*   Client sends:
    *   `desiredName`: The name they want for the capability.
    *   `capabilityType`: The type of capability (`RESOURCE`, `TOOL`, etc.).
    *   `descriptor`: The capability descriptor without an ID field set (or it can be ignored by the server at this stage).
    *   `from`: Sender's address.
    *   `fee`: Proposed fee.
*   No signature is sent at this stage.

**Server: Process Initiation**

*   Receives the initiation request.
*   Generates the canonical `capabilityID` using `GenerateCapabilityID(desiredName, capabilityType)`.
*   Checks if this `capabilityID` already exists in the database (and potentially a temporary "pending registrations" store to handle concurrent requests).
*   If ID is taken: Respond with HTTP 409 Conflict.
*   If ID is available:
    *   Construct the full `capabilityDescriptor` by populating its `ID` field with the generated `capabilityID` and using other details from the client's provided descriptor.
    *   Create an unsigned `Transaction` object (let's call it `pendingTxn`) with all details: `From`, `To` (empty), `Value` (0), `Data` (marshalled `capabilityDescriptor` with the server-generated ID), `Type` (`MCPRegisterCapability`), `Fee`, and `Timestamp`.
    *   Calculate the `TransactionHash` for this `pendingTxn`.
    *   Store this `pendingTxn` (or its hash and essential details) temporarily (e.g., in memory with a short TTL, or a lightweight DB table for pending registrations) associated with its `TransactionHash`.
    *   Respond to the client with HTTP 200 OK, including:
        *   `pendingTransactionHash`: The hash of the transaction the client needs to sign.
        *   `canonicalCapabilityID`: The server-generated ID.
        *   `fullDescriptor`: The complete descriptor (with the server-generated ID) that was used to create the hash. This allows the client to verify what it's about to sign.

**Client: Sign and Finalize (POST /mcp/capability/register/finalize)**

*   Receives the `pendingTransactionHash`, `canonicalCapabilityID`, and `fullDescriptor` from the server.
*   Crucially, the client reconstructs the transaction data exactly as the server did: It marshals the `fullDescriptor` received from the server to create the `Data` field.
*   It creates a local `Transaction` object using its original `From`, `Fee`, the server-provided `Timestamp` (if the server included it in the `fullDescriptor` or `pendingTxn` details), and the reconstructed `Data`.
*   It calculates the hash of this local transaction. This hash MUST match the `pendingTransactionHash` received from the server. If not, something is wrong (client or server misconstructed).
*   If hashes match, the client signs this hash (or the marshalled transaction data).
*   Client sends:
    *   `pendingTransactionHash`: To identify the pending registration.
    *   `publicKey`: Client's public key.
    *   `signature`: The signature over the data corresponding to `pendingTransactionHash`.

**Server: Process Finalization**

*   Receives the finalization request.
*   Retrieves the `pendingTxn` details using `pendingTransactionHash` from its temporary store. If not found or expired, respond with an error.
*   Populates the `pendingTxn` with the received `PublicKey` and `Signature`.
*   Verifies the signature on `pendingTxn`.
*   If invalid: Respond with HTTP 400/422.
*   If valid:
    *   Remove the transaction from the temporary pending store.
    *   Add the now fully signed and verified transaction to the main `TransactionPool`.
    *   Proceed with DHT announcement (using the `canonicalCapabilityID`).
    *   Respond with HTTP 201 Created (or 200 OK) with the final transaction hash and the `agent://` URI.

This two-step process ensures the client signs exactly what the server intends to record, including the server-generated ID.

**Refinement for `GenerateCapabilityID`:** You mentioned ensuring `capType` is not in the name since the URI scheme identifies `.Type`. The current `GenerateCapabilityID` already does this: `agent-<type>-<sanitized-name>`. The URI generation then uses this ID and the type to form `agent://<id>.<type>/`. This seems correct. The ID itself should be unique and can be formed from the name, but the URI construction adds the type.

Let's implement the changes. This will involve new handlers and modifications to existing ones.

## I. New Data Structures (if needed for pending store)

For a simple in-memory pending store (suitable for single-node tests, not production without TTLs/cleanup):

```go
// Potentially in blockchain_server.go or a new types file
var pendingRegistrations = struct {
	sync.RWMutex
	m map[string]*Transaction // Key: pendingTransactionHash
}{m: make(map[string]*Transaction)}

const pendingRegistrationTTL = 5 * time.Minute // Example TTL
```

## II. Code Modifications:

1.  **`uri_generation.go` (Ensure `GenerateCapabilityID` is robust)** The `GenerateCapabilityID` function you have is good. We'll use it. The key is that it doesn't include the type in the name part of the ID, but rather as a prefix or distinct component if needed for the ID itself. The URI then adds `.type`.

Current `GenerateCapabilityID`:

```go
// GenerateCapabilityID creates a standardized ID for a capability.
// Uses a fixed prefix "agent", the lowercase capability type, and a sanitized name.
func GenerateCapabilityID(name string, capType CapabilityType) string {
	prefix := "agent" // Standard prefix for agent capabilities
	typeName := strings.ToLower(string(capType))
	sanitizedName := SanitizeNameForID(name)

	if sanitizedName == "" {
		return fmt.Sprintf("%s-%s-%s", prefix, typeName, "unnamed-capability")
	}
	// ID format: agent-<type>-<sanitized-name>
	return fmt.Sprintf("%s-%s-%s", prefix, typeName, sanitizedName)
}
```

This ID format `agent-<type>-<sanitized-name>` is fine. The URI `agent://<id>.<type>/` will then be, for example, `agent://agent-tool-my-tool.tool/`. This is a bit redundant with "tool" appearing twice.

Alternative `GenerateCapabilityID` (Simpler ID, type only in URI): If you want the ID to be just `agent-<sanitized-name>` and let the URI carry the type:

```go
// In mcp_types.go or utils.go
func GenerateCapabilityID(name string /*, capType CapabilityType -- type no longer needed for ID itself */) string {
	prefix := "agent-cap" // Or just "cap-" or your preferred prefix for capabilities
	sanitizedName := SanitizeNameForID(name)

	if sanitizedName == "" {
		return fmt.Sprintf("%s-%s", prefix, "unnamed") // Fallback
	}
	return fmt.Sprintf("%s-%s", prefix, sanitizedName)
}
```

And then `GenerateMCPCapabilityURI` would be:

```go
// In uri_generation.go
func GenerateMCPCapabilityURI(capabilityID string, mcpCapabilityType CapabilityType, path string, params map[string]string) string {
	// URI: agent://<capabilityID>.<lowercaseMcCapabilityType>/path
	return GenerateResourceURI(capabilityID, strings.ToLower(string(mcpCapabilityType)), path, params)
}
```

This results in URIs like `agent://agent-cap-my-tool.tool/`. This seems cleaner. Let's proceed with this simpler ID generation.

2.  **`mcp_types.go` (or `utils.go`) - Update `GenerateCapabilityID`**

```go

// GenerateCapabilityID creates a standardized ID for a capability.
// Uses a fixed prefix "agent", the lowercase capability type, and a sanitized name.
func GenerateCapabilityID(name string /* capType CapabilityType - Type removed from ID itself */) string {
	prefix := "agent-cap" // Standard prefix for agent capabilities
	sanitizedName := SanitizeNameForID(name)

	if sanitizedName == "" {
		// Fallback if sanitization results in an empty string
		return fmt.Sprintf("%s-%s-%d", prefix, "unnamed", time.Now().UnixNano()%10000)
	}
	return fmt.Sprintf("%s-%s", prefix, sanitizedName)
}
```

3.  **`blockchain_server.go` - Add new handlers and modify existing**

```go
	p2pPort          int
}

var pendingRegistrations = struct {
	sync.RWMutex
	m map[string]*Transaction // Key: pendingTransactionHash
	// TODO: Add a mechanism to clean up expired pending registrations
}{m: make(map[string]*Transaction)}

const pendingRegistrationTTL = 5 * time.Minute // Example TTL
// Stop gracefully shuts down the blockchain server
func (bcs *BlockchainServer) Stop(ctx context.Context) error {
	var err error
Unchanged lines
	mux.HandleFunc("/test/faucet", bcs.handleTestFaucet)

	// Add MCP API endpoints
	mux.HandleFunc("/mcp/capability/register/initiate", bcs.handleMCPRegisterCapabilityInitiate) // New
	mux.HandleFunc("/mcp/capability/register/finalize", bcs.handleMCPRegisterCapabilityFinalize)   // New
	mux.HandleFunc("/mcp/capability/invoke", bcs.handleMCPInvokeCapability)
	mux.HandleFunc("/mcp/capability/update", bcs.handleMCPUpdateCapability)
	mux.HandleFunc("/mcp/capability/get", bcs.handleMCPGetCapability)
Unchanged lines
	json.NewEncoder(w).Encode(faucetTxn) // Respond with the created transaction
}

// --- MCP API Handlers ---

// handleMCPRegisterCapabilityInitiate - Step 1 of 2-step registration
func (bcs *BlockchainServer) handleMCPRegisterCapabilityInitiate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Client sends desiredName, type, descriptor (ID field empty/ignored), from, fee
	var requestData struct {
		From           string          `json:"from"`
		PublicKey      string          `json:"publicKey"`
Unchanged lines
		Fee            uint64          `json:"fee"`
		CapabilityType string          `json:"capabilityType"`
		Descriptor     json.RawMessage `json:"descriptor"`
		DesiredName    string          `json:"desiredName,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
Unchanged lines

	// Validate the capability type
	var capabilityDescriptor interface{}
	var capabilityName string
	var capabilityTimestamp int64
	var err error
Unchanged lines
			http.Error(w, fmt.Sprintf("Failed to parse ResourceDescriptor: %v", err), http.StatusBadRequest)
			return
		}
		capabilityName = descriptor.Name
		capabilityTimestamp = descriptor.Timestamp
		capabilityDescriptor = descriptor
	case string(CapabilityTypeTool):
Unchanged lines
			http.Error(w, fmt.Sprintf("Failed to parse ToolDescriptor: %v", err), http.StatusBadRequest)
			return
		}
		capabilityName = descriptor.Name
		capabilityTimestamp = descriptor.Timestamp
		capabilityDescriptor = descriptor
	case string(CapabilityTypePrompt):
Unchanged lines
			http.Error(w, fmt.Sprintf("Failed to parse PromptDescriptor: %v", err), http.StatusBadRequest)
			return
		}
		capabilityName = descriptor.Name
		capabilityTimestamp = descriptor.Timestamp
		capabilityDescriptor = descriptor
	case string(CapabilityTypeMemoryService):
Unchanged lines
			http.Error(w, fmt.Sprintf("Failed to parse MemoryServiceDescriptor: %v", err), http.StatusBadRequest)
			return
		}
		capabilityName = descriptor.Name
		capabilityTimestamp = descriptor.Timestamp
		capabilityDescriptor = descriptor
	default:
Unchanged lines
		return
	}

	nameForIDGeneration := requestData.DesiredName
	if nameForIDGeneration == "" {
		if capabilityName == "" {
			http.Error(w, "Either 'desiredName' in request or 'name' in descriptor must be provided for ID generation.", http.StatusBadRequest)
			return
		}
		nameForIDGeneration = capabilityName
	}

	generatedCapabilityID := GenerateCapabilityID(nameForIDGeneration /*, CapabilityType(requestData.CapabilityType) - type removed from ID gen */)

	// Check if this server-generated ID already exists
	if _, errDb := bcs.db.GetCapabilityByID(generatedCapabilityID); errDb == nil {
		http.Error(w, fmt.Sprintf("Capability ID '%s' (generated from name '%s' and type '%s') already exists.", generatedCapabilityID, nameForIDGeneration, requestData.CapabilityType), http.StatusConflict)
		return
	}

	// Update the descriptor with the server-generated ID.
	// This requires modifying the specific descriptor type.
	var finalDescriptorForHashing interface{}
	switch desc := capabilityDescriptor.(type) {
	case ResourceDescriptor:
		desc.ID = generatedCapabilityID
		finalDescriptorForHashing = desc
	case ToolDescriptor:
		desc.ID = generatedCapabilityID
		finalDescriptorForHashing = desc
	case PromptDescriptor:
		desc.ID = generatedCapabilityID
		finalDescriptorForHashing = desc
	case MemoryServiceDescriptor:
		desc.ID = generatedCapabilityID
		finalDescriptorForHashing = desc
	default:
		http.Error(w, "Internal server error: could not set generated ID on descriptor.", http.StatusInternalServerError)
		return
	}

	// Create transaction data with the server-finalized descriptor
	txnData, err := json.Marshal(map[string]interface{}{
		"capabilityDescriptor": finalDescriptorForHashing, // Use the descriptor with the server-generated ID
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal transaction data: %v", err), http.StatusInternalServerError)
Unchanged lines
	}

	// Create a new MCP transaction (unsigned at this stage)
	pendingTxn := NewMCPTransaction(
		requestData.From,
		"", // No recipient for capability registration
		0,  // No value transfer
Unchanged lines
		capabilityTimestamp, // Pass the timestamp from the descriptor
	)

	// Store this pending transaction (or its hash and necessary data)
	pendingRegistrations.Lock()
	pendingRegistrations.m[pendingTxn.TransactionHash] = pendingTxn
	pendingRegistrations.Unlock()

	// TODO: Add a cleanup mechanism for expired pendingRegistrations

	log.Printf("[INFO] Initiated capability registration. Pending hash: %s, Generated ID: %s", pendingTxn.TransactionHash, generatedCapabilityID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":                   "pending_signature",
		"pendingTransactionHash":   pendingTxn.TransactionHash,
		"canonicalCapabilityID":    generatedCapabilityID,
		"fullDescriptorForSigning": finalDescriptorForHashing, // Send back the descriptor client needs to use for hash consistency
	})
}

// handleMCPRegisterCapabilityFinalize - Step 2 of 2-step registration
func (bcs *BlockchainServer) handleMCPRegisterCapabilityFinalize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var requestData struct {
		PendingTransactionHash string `json:"pendingTransactionHash"`
		PublicKey              string `json:"publicKey"`
		Signature              []byte `json:"signature"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse request body: %v", err), http.StatusBadRequest)
		return
	}

	pendingRegistrations.Lock()
	transaction, ok := pendingRegistrations.m[requestData.PendingTransactionHash]
	if ok {
		delete(pendingRegistrations.m, requestData.PendingTransactionHash) // Remove once retrieved
	}
	pendingRegistrations.Unlock()

	if !ok {
		http.Error(w, "Pending registration not found or expired.", http.StatusNotFound)
		return
	}

	// Set the public key and signature
	transaction.PublicKey = requestData.PublicKey
	transaction.Signature = requestData.Signature

	err := bcs.BlockchainPtr.AddTransactionToTransactionPool(transaction)
	if err != nil {
		log.Printf("[ERROR] Failed to add MCP Register transaction to pool: %v\nTransaction: %+v\nSignature: %x",
			err, transaction, transaction.Signature)
Unchanged lines
	log.Printf("[INFO] Successfully added transaction %s to pool", transaction.TransactionHash)

	// --- Generate and Announce URI ---
	// Extract the ID and Type from the finalized transaction's data
	var finalizedTxData struct {
		CapabilityDescriptor BaseDescriptor `json:"capabilityDescriptor"` // Just need Base for ID and Type
	}
	if err := json.Unmarshal(transaction.Data, &finalizedTxData); err != nil {
		log.Printf("[ERROR] Failed to unmarshal finalized transaction data for DHT announcement: %v", err)
		// Continue with response even if DHT announcement prep fails
	}
	finalCapabilityID := finalizedTxData.CapabilityDescriptor.ID
	finalCapabilityType := finalizedTxData.CapabilityDescriptor.CapabilityType
	// Use capabilityID and its specific MCP type (e.g., "RESOURCE", "TOOL")
	capabilityURI := GenerateMCPCapabilityURI(finalCapabilityID, finalCapabilityType, "", nil)

	if bcs.discoveryManager != nil {
		go func(idToAnnounce string, capTypeStr string) { // Announce in background
Unchanged lines
			} else {
				log.Printf("[INFO] Successfully announced MCP capability %s (type: %s, URI: %s) on DHT", idToAnnounce, strings.ToLower(capTypeStr), capabilityURI)
			}
		}(finalCapabilityID, string(finalCapabilityType))
	}

	w.WriteHeader(http.StatusCreated) // Or http.StatusOK
 	w.Header().Set("Content-Type", "application/json")
 	json.NewEncoder(w).Encode(map[string]interface{}{
 		"status":           "success",
Use code with care. Learn more
4.  **Update `TestMCPRegisterAndDiscoverCapability` (in `mcp_api_test.go`)**
```

```go
	server, bc, _, wallet := setupTestBlockchainServer(t) // This now initializes MockDiscoveryService.SelfID

	// 2. Prepare capability registration data
	from := wallet.GetAddress()
	fee := uint64(100)
	fixedTimestamp := time.Now().Unix()
	capType := CapabilityTypeTool
	desiredNameForID := "My Super Tool" // Client suggests this name
	// Client does NOT set the ID in the descriptor it sends initially.

	initialToolDesc := ToolDescriptor{ // Descriptor sent by client will have ID field empty
		BaseDescriptor: BaseDescriptor{
			// ID:             "", // Explicitly empty
			Name:           desiredNameForID, // Name is still provided
			Owner:          from,
			Version:        "1.0.0",
			CapabilityType: capType, // Correctly set here
Unchanged lines
		OutputSchemaJSON: `{}`,
	}

	// --- Step 1: Initiate Registration ---
	initiatePayload := map[string]interface{}{
		"desiredName":    desiredNameForID,
		"capabilityType": string(capType),
		"descriptor":     initialToolDesc, // Descriptor with empty ID
		"from":           from,
		"fee":            fee,
		// No signature/publicKey needed for initiation
	}
	initiateBody, _ := json.Marshal(initiatePayload)
	initReq, _ := http.NewRequest("POST", "/mcp/capability/register/initiate", bytes.NewBuffer(initiateBody))
	initReq.Header.Set("Content-Type", "application/json")
	initRR := httptest.NewRecorder()
	server.handleMCPRegisterCapabilityInitiate(initRR, initReq)

	if status := initRR.Code; status != http.StatusOK {
		t.Fatalf("Initiate handler returned wrong status code: got %v want %v. Body: %s",
			status, http.StatusOK, initRR.Body.String())
	}

	var initResponse struct {
		Status                   string          `json:"status"`
		PendingTransactionHash   string          `json:"pendingTransactionHash"`
		CanonicalCapabilityID    string          `json:"canonicalCapabilityID"`
		FullDescriptorForSigning json.RawMessage `json:"fullDescriptorForSigning"` // Server provides the descriptor to sign
	}
	if err := json.Unmarshal(initRR.Body.Bytes(), &initResponse); err != nil {
		t.Fatalf("Failed to parse initiate response: %v. Body: %s", err, initRR.Body.String())
	}
	if initResponse.Status != "pending_signature" {
		t.Fatalf("Expected status 'pending_signature', got '%s'", initResponse.Status)
	}
	if initResponse.PendingTransactionHash == "" {
		t.Fatal("PendingTransactionHash is empty in initiate response")
	}
	if initResponse.CanonicalCapabilityID == "" {
		t.Fatal("CanonicalCapabilityID is empty in initiate response")
	}

	// --- Step 2: Client Signs and Finalizes ---
	// Client reconstructs the transaction data using the descriptor from the server
	var serverFinalizedDescriptor ToolDescriptor // Assuming ToolDescriptor
	if err := json.Unmarshal(initResponse.FullDescriptorForSigning, &serverFinalizedDescriptor); err != nil {
		t.Fatalf("Failed to unmarshal FullDescriptorForSigning: %v", err)
	}

	// The ID in serverFinalizedDescriptor should match initResponse.CanonicalCapabilityID
	if serverFinalizedDescriptor.ID != initResponse.CanonicalCapabilityID {
		t.Fatalf("ID mismatch: serverFinalizedDescriptor.ID (%s) != initResponse.CanonicalCapabilityID (%s)",
			serverFinalizedDescriptor.ID, initResponse.CanonicalCapabilityID)
	}

	clientFinalizationData, _ := json.Marshal(map[string]interface{}{"capabilityDescriptor": serverFinalizedDescriptor})

	// Client creates a local transaction matching what the server expects for the pending hash
	clientTxnForFinalize := &Transaction{
		Type:      TransactionTypeMCPRegisterCapability,
		Data:      clientFinalizationData, // Data based on server-provided descriptor
		Fee:       fee,
		Timestamp: serverFinalizedDescriptor.Timestamp, // Use timestamp from server-provided descriptor
		From:      from,
	}

	// Client calculates hash to verify it matches pendingTransactionHash
	// Note: NewMCPTransaction sets the hash, or we can call .Hash()
	// For this check, let's ensure our local hash matches the server's pending hash.
	// We need to ensure NewMCPTransaction's internal hashing for the *unsigned* part matches.
	// The `Transaction.Hash()` method is the canonical one for the *final* hash.
	// The hash in `pendingTransactionHash` was calculated by the server's `NewMCPTransaction`
	// before signature.
	tempServerTxn := NewMCPTransaction(from, "", 0, clientFinalizationData, TransactionTypeMCPRegisterCapability, fee, serverFinalizedDescriptor.Timestamp)
	if tempServerTxn.TransactionHash != initResponse.PendingTransactionHash {
		t.Fatalf("Client-calculated pending hash '%s' does not match server's pending hash '%s'", tempServerTxn.TransactionHash, initResponse.PendingTransactionHash)
	}

	if err := wallet.SignTransaction(clientTxnForFinalize); err != nil { // Sign the transaction that includes the server-generated ID
		t.Fatalf("Failed to sign client transaction: %v", err)
	}

	finalizePayload := map[string]interface{}{
		"pendingTransactionHash": initResponse.PendingTransactionHash,
		"publicKey":              wallet.GetPublicKeyHex(),
		"signature":              base64.StdEncoding.EncodeToString(clientTxnForFinalize.Signature),
	}
	finalizeBody, _ := json.Marshal(finalizePayload)
	finalizeReq, _ := http.NewRequest("POST", "/mcp/capability/register/finalize", bytes.NewBuffer(finalizeBody))
	finalizeReq.Header.Set("Content-Type", "application/json")
	finalizeRR := httptest.NewRecorder()
	server.handleMCPRegisterCapabilityFinalize(finalizeRR, finalizeReq)

	if status := finalizeRR.Code; status != http.StatusCreated { // Expect 201 Created on successful finalization
		t.Fatalf("Register handler returned wrong status code: got %v want %v. Body: %s",
			status, http.StatusCreated, finalizeRR.Body.String())
	}

	// Verify transaction was added to pool (means validation passed)
Unchanged lines
	if len(bc.TransactionPool) == 0 {
		t.Fatalf("Expected transaction in pool after registration, but pool is empty")
	}
	// The hash in the pool should be the one from clientTxnForFinalize (which includes the server-generated ID)
	t.Logf("Transaction in pool hash: %s, Client finalized hash: %s", bc.TransactionPool[0].TransactionHash, clientTxnForFinalize.TransactionHash)
	// Note: A direct hash comparison might still be tricky if the server re-creates the transaction object
	// instead of using the exact one from the pending store. The key is that the signature verifies.

	// 4. Attempt to discover the capability using the mock discovery manager
	time.Sleep(200 * time.Millisecond) // Small delay for the announcement goroutine in handler

	mockDiscovery := server.discoveryManager.(*MockDiscoveryService) // Cast to access mock's state

	providers, err := mockDiscovery.FindMCPCapabilityProviders(context.Background(), initResponse.CanonicalCapabilityID, strings.ToLower(string(capType)))
	if err != nil {
		t.Fatalf("FindMCPCapabilityProviders failed: %v", err)
	}
	if len(providers) == 0 {
		t.Fatalf("Expected to find providers for capability %s, but found none. AnnouncedCaps: %v", initResponse.CanonicalCapabilityID, mockDiscovery.AnnouncedCaps)
	}
	if providers[0].ID != mockDiscovery.SelfID {
		t.Errorf("Expected provider ID %s, got %s", mockDiscovery.SelfID, providers[0].ID)
```

**Important Considerations and Next Steps:**

*   **Pending Registrations Store:** The `pendingRegistrations` map is a very basic in-memory store. For a production system, you'd want something more robust:
    *   **TTL/Expiration:** Entries should expire after a timeout (e.g., 5 minutes) to prevent memory leaks if clients don't finalize.
    *   **Persistence (Optional):** If you need pending registrations to survive server restarts, you'd store them in LevelDB or another persistent store.
*   **Error Handling:** The error handling in the new handlers can be made more granular.
*   **Concurrency:** Ensure the `pendingRegistrations` map is accessed concurrently safely (the `sync.RWMutex` helps).
*   **`capabilityDescriptor` Update in Handler:** The way `capabilityDescriptor` has its ID set in `handleMCPRegisterCapabilityInitiate` uses type assertions. This is okay but can be made more generic if you have many capability types. You might consider having an interface method like `SetID(string)` on all descriptor types.

```go
// Example of setting ID via interface
type IdentifiableCapability interface {
    SetID(id string)
    GetBaseDescriptor() BaseDescriptor // To get name, type, timestamp
}
// Each descriptor (ResourceDescriptor, ToolDescriptor) would implement this.
// Then in the handler:
// if ic, ok := capabilityDescriptor.(IdentifiableCapability); ok {
//     ic.SetID(generatedCapabilityID)
// } else { /* error */ }
```

*   **Transaction Hash Consistency:** The most complex part is ensuring the `pendingTransactionHash` generated by the server matches the hash the client calculates before signing. This requires that the `Transaction` object (specifically the fields that go into the hash for signing) is constructed identically on both sides. The server sending back `fullDescriptorForSigning` is key to this. The client must use this exact descriptor.
*   **Timestamp Handling:** The server should probably dictate the `Timestamp` for the transaction when it creates the `pendingTxn`. The `fullDescriptorForSigning` should include this server-set timestamp, and the client should use it when reconstructing the transaction for signing. This ensures timestamp consistency. The current diff for `blockchain_server.go` already extracts `capabilityTimestamp` from the descriptor and passes it to `NewMCPTransaction`.

This two-step approach is more involved but significantly improves security and clarity by ensuring the client signs exactly what the server has finalized, including the server-authoritative ID.
```

---

<div class="footer-links">


© 2025 KNIRV Network
</div>
