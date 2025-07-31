

---

**Source**: KNIRVROOT/docs/pending_implementation_plans/Data_Consistency_Implementation_Plan.md

# Transaction.Data Consistency Implementation Plan

## Overview

This document outlines a comprehensive plan to address the inconsistencies in Transaction.Data representation across the KNIRVCHAIN system. The goal is to establish clear boundaries between JSON and Protocol Buffer (Protobuf) usage, ensuring consistent data handling throughout the application lifecycle.

## Current Issues

1. **Inconsistent Serialization Formats**:
   - Transaction.Data uses both JSON and Protobuf depending on context
   - Hashing and signature verification use Protobuf while API handlers use JSON

2. **Two-Step Registration Flow Complexity**:
   - The legacy initiate/finalize endpoints create serialization inconsistencies
   - Different serialization formats between transaction creation and verification
   - The new `/prepare_registration` endpoint needs consistent handling

3. **Missing Protobuf Definitions**:
   - No Protobuf definitions for `MCPRegisterCapabilityData` and `MCPUpdateCapabilityData`
   - Incomplete Protobuf coverage across transaction types

4. **Inconsistent Data Processing**:
   - JSON unmarshaling is used in transaction validation
   - Protobuf is used for hashing and signatures

## Guiding Principles

To address these issues, we will follow these clear principles:

1. **JSON for Transit, Protobuf for Storage and Hashing**:
   - All network API requests/responses will use JSON
   - All database storage will use Protobuf
   - All cryptographic operations (hashing, signatures) will use Protobuf

2. **Consistent Transaction.Data Handling**:
   - Transaction.Data will remain in JSON format for transit and invocation
   - Transaction.Data will be consistently processed using JSON unmarshaling
   - The entire Transaction structure can be protobuf'd for storage

3. **Improved Two-Step Registration Flow**:
   - Remove the legacy initiate/finalize endpoints
   - Use the new `/prepare_registration` endpoint for server-generated IDs
   - Ensure consistent serialization between preparation and submission

4. **Complete Protobuf Coverage**:
   - Define Protobuf messages for all data structures
   - Ensure consistent conversion between Go structs and Protobuf messages

## Implementation Plan

### Phase 1: Define Clear Boundaries and Complete Protobuf Definitions

#### 1.1 Update Proto Files

Add missing Protobuf definitions for transaction data structures in `proto/mcp_transactions.proto`:

```protobuf
syntax = "proto3";

package proto;

option go_package = "./;proto";

import "google/protobuf/struct.proto";
import "proto/mcp_descriptors.proto";
import "proto/mcp_context.proto";

// MCPRegisterCapabilityDataProto corresponds to MCPRegisterCapabilityData
message MCPRegisterCapabilityDataProto {
  // We use oneof to handle different capability descriptor types
  oneof capability_descriptor {
    ResourceDescriptorProto resource = 1;
    ToolDescriptorProto tool = 2;
    PromptDescriptorProto prompt = 3;
    MemoryServiceDescriptorProto memory_service = 4;
  }
}

// MCPUpdateCapabilityDataProto corresponds to MCPUpdateCapabilityData
message MCPUpdateCapabilityDataProto {
  string capability_id = 1;
  string capability_type = 2;
  // We use oneof to handle different capability descriptor types
  oneof capability_descriptor {
    ResourceDescriptorProto resource = 3;
    ToolDescriptorProto tool = 4;
    PromptDescriptorProto prompt = 5;
    MemoryServiceDescriptorProto memory_service = 6;
  }
}

// TransactionDataProto is a container for different transaction data types
message TransactionDataProto {
  oneof data {
    MCPRegisterCapabilityDataProto register_capability = 1;
    MCPInvokeCapabilityDataProto invoke_capability = 2;
    MCPUpdateCapabilityDataProto update_capability = 3;
    // Add other transaction data types as needed
    bytes raw_data = 99; // For non-MCP transaction data or legacy support
  }
}
```

#### 1.2 Create Conversion Functions

Add conversion functions in `proto_converters.go` for the new Protobuf message types:

```go
// ConvertMCPRegisterCapabilityDataToProto converts MCPRegisterCapabilityData to MCPRegisterCapabilityDataProto
func ConvertMCPRegisterCapabilityDataToProto(data MCPRegisterCapabilityData) (*MCPRegisterCapabilityDataProto, error) {
    protoData := &MCPRegisterCapabilityDataProto{}
    
    // Convert the capability descriptor based on its type
    container, err := ConvertToCapabilityDescriptorContainerProto(data.CapabilityDescriptor)
    if err != nil {
        return nil, fmt.Errorf("failed to convert capability descriptor: %w", err)
    }
    
    // Set the appropriate field in the oneof based on the descriptor type
    switch d := container.Descriptor_.(type) {
    case *pb.CapabilityDescriptorContainerProto_Resource:
        protoData.CapabilityDescriptor = &MCPRegisterCapabilityDataProto_Resource{
            Resource: d.Resource,
        }
    case *pb.CapabilityDescriptorContainerProto_Tool:
        protoData.CapabilityDescriptor = &MCPRegisterCapabilityDataProto_Tool{
            Tool: d.Tool,
        }
    case *pb.CapabilityDescriptorContainerProto_Prompt:
        protoData.CapabilityDescriptor = &MCPRegisterCapabilityDataProto_Prompt{
            Prompt: d.Prompt,
        }
    case *pb.CapabilityDescriptorContainerProto_MemoryService:
        protoData.CapabilityDescriptor = &MCPRegisterCapabilityDataProto_MemoryService{
            MemoryService: d.MemoryService,
        }
    }
    
    return protoData, nil
}

// ConvertMCPUpdateCapabilityDataToProto converts MCPUpdateCapabilityData to MCPUpdateCapabilityDataProto
func ConvertMCPUpdateCapabilityDataToProto(data MCPUpdateCapabilityData) (*MCPUpdateCapabilityDataProto, error) {
    protoData := &MCPUpdateCapabilityDataProto{
        CapabilityId:   data.CapabilityID,
        CapabilityType: data.CapabilityType,
    }
    
    // Convert the capability descriptor based on its type
    container, err := ConvertToCapabilityDescriptorContainerProto(data.CapabilityDescriptor)
    if err != nil {
        return nil, fmt.Errorf("failed to convert capability descriptor: %w", err)
    }
    
    // Set the appropriate field in the oneof based on the descriptor type
    switch d := container.Descriptor_.(type) {
    case *pb.CapabilityDescriptorContainerProto_Resource:
        protoData.CapabilityDescriptor = &MCPUpdateCapabilityDataProto_Resource{
            Resource: d.Resource,
        }
    case *pb.CapabilityDescriptorContainerProto_Tool:
        protoData.CapabilityDescriptor = &MCPUpdateCapabilityDataProto_Tool{
            Tool: d.Tool,
        }
    case *pb.CapabilityDescriptorContainerProto_Prompt:
        protoData.CapabilityDescriptor = &MCPUpdateCapabilityDataProto_Prompt{
            Prompt: d.Prompt,
        }
    case *pb.CapabilityDescriptorContainerProto_MemoryService:
        protoData.CapabilityDescriptor = &MCPUpdateCapabilityDataProto_MemoryService{
            MemoryService: d.MemoryService,
        }
    }
    
    return protoData, nil
}

// Add corresponding functions for converting from Protobuf to Go structs
```

#### 1.3 Document the Boundaries

Create a clear documentation of the boundaries between JSON and Protobuf usage:

```go
// In a new file: data_serialization.go

// SerializationBoundaries defines the serialization format to use for different operations
type SerializationBoundaries struct {
    // Network transit (API requests/responses)
    NetworkTransit string
    
    // Database storage
    DatabaseStorage string
    
    // Cryptographic operations (hashing, signatures)
    Cryptographic string
    
    // Transaction.Data in memory and transit
    TransactionData string
}

// GetSerializationBoundaries returns the defined serialization boundaries
func GetSerializationBoundaries() SerializationBoundaries {
    return SerializationBoundaries{
        NetworkTransit:   "JSON",
        DatabaseStorage:  "Protobuf",
        Cryptographic:    "Protobuf",
        TransactionData:  "JSON",
    }
}
```

### Phase 2: Implement Improved Two-Step Registration Flow

#### 2.1 Remove Legacy Endpoints

Remove the legacy initiate/finalize endpoints from `blockchain_server.go`:

```go
// Remove these handlers
// mux.HandleFunc("/mcp/capability/register/initiate", bcs.handleMCPRegisterCapabilityInitiate)
// mux.HandleFunc("/mcp/capability/register/finalize", bcs.handleMCPRegisterCapabilityFinalize)

// Remove the handler functions themselves
// func (bcs *BlockchainServer) handleMCPRegisterCapabilityInitiate(w http.ResponseWriter, r *http.Request) { ... }
// func (bcs *BlockchainServer) handleMCPRegisterCapabilityFinalize(w http.ResponseWriter, r *http.Request) { ... }

// Remove the pendingRegistrations map and related code
// var pendingRegistrations = struct { ... }
```

#### 2.2 Implement Prepare Registration Endpoint

Ensure the `/prepare_registration` endpoint is properly implemented with consistent serialization:

```go
// In blockchain_server.go

// MCPPrepareCapabilityRegistrationRequest defines the request structure for prepare_registration
type MCPPrepareCapabilityRegistrationRequest struct {
    OwnerAddress string          `json:"ownerAddress"`
    Descriptor   json.RawMessage `json:"descriptor"` // Raw descriptor JSON
    Fee          uint64          `json:"fee"`
    Message      string          `json:"message,omitempty"`
}

// UnsignedTransactionDetails defines the response structure for prepare_registration
type UnsignedTransactionDetails struct {
    From      string `json:"from"`
    To        string `json:"to,omitempty"`
    Value     string `json:"value"`
    Data      string `json:"data"` // base64 encoded JSON bytes
    Timestamp int64  `json:"timestamp"`
    Fee       uint64 `json:"fee"`
    Type      string `json:"type"`
}

// MCPPrepareCapabilityRegistrationResponse defines the response structure for prepare_registration
type MCPPrepareCapabilityRegistrationResponse struct {
    PendingTransactionHash string                   `json:"pendingTransactionHash"`
    UnsignedTransaction    UnsignedTransactionDetails `json:"unsignedTransaction"`
}

// handleMCPPrepareCapabilityRegistration handles the preparation of a capability registration
func (bcs *BlockchainServer) handleMCPPrepareCapabilityRegistration(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    
    // Parse the request body
    var requestData MCPPrepareCapabilityRegistrationRequest
    if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
        http.Error(w, fmt.Sprintf("Failed to parse request body: %v", err), http.StatusBadRequest)
        return
    }
    
    // Validate the request data
    if requestData.OwnerAddress == "" {
        http.Error(w, "Missing ownerAddress", http.StatusBadRequest)
        return
    }
    
    // Parse and validate the descriptor
    var descriptor interface{}
    if err := json.Unmarshal(requestData.Descriptor, &descriptor); err != nil {
        http.Error(w, fmt.Sprintf("Failed to parse descriptor: %v", err), http.StatusBadRequest)
        return
    }
    
    // Generate a unique capability ID
    // This is a simplified example - actual implementation would use a more robust ID generation method
    capabilityID := GenerateCapabilityID(fmt.Sprintf("%v", descriptor))
    
    // Check if this capability ID already exists
    if _, err := bcs.db.GetCapabilityByID(capabilityID); err == nil {
        http.Error(w, fmt.Sprintf("Capability ID '%s' already exists", capabilityID), http.StatusConflict)
        return
    }
    
    // Update the descriptor with the server-generated ID
    // This is a simplified example - actual implementation would update the specific descriptor type
    descriptorMap, ok := descriptor.(map[string]interface{})
    if !ok {
        http.Error(w, "Invalid descriptor format", http.StatusBadRequest)
        return
    }
    descriptorMap["id"] = capabilityID
    
    // Create transaction data with the server-finalized descriptor
    txnData, err := json.Marshal(map[string]interface{}{
        "capabilityDescriptor": descriptorMap,
    })
    if err != nil {
        http.Error(w, fmt.Sprintf("Failed to marshal transaction data: %v", err), http.StatusInternalServerError)
        return
    }
    
    // Create an unsigned transaction
    timestamp := time.Now().Unix()
    txnType := string(TransactionTypeMCPRegisterCapability)
    
    // Calculate the transaction hash
    // This is a simplified example - actual implementation would use the proper hashing method
    txnHash := calculateTransactionHash(requestData.OwnerAddress, "", "0", txnData, timestamp, requestData.Fee, txnType)
    
    // Create the response
    unsignedTx := UnsignedTransactionDetails{
        From:      requestData.OwnerAddress,
        To:        "",
        Value:     "0",
        Data:      base64.StdEncoding.EncodeToString(txnData),
        Timestamp: timestamp,
        Fee:       requestData.Fee,
        Type:      txnType,
    }
    
    response := MCPPrepareCapabilityRegistrationResponse{
        PendingTransactionHash: txnHash,
        UnsignedTransaction:    unsignedTx,
    }
    
    // Respond with the unsigned transaction details
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

// calculateTransactionHash calculates the hash of an unsigned transaction
// This is a simplified example - actual implementation would use the proper hashing method
func calculateTransactionHash(from, to, value string, data []byte, timestamp int64, fee uint64, txnType string) string {
    // In a real implementation, this would use the same hashing method as Transaction.Hash()
    // For now, we'll just create a simple hash
    h := sha256.New()
    h.Write([]byte(from))
    h.Write([]byte(to))
    h.Write([]byte(value))
    h.Write(data)
    h.Write([]byte(fmt.Sprintf("%d", timestamp)))
    h.Write([]byte(fmt.Sprintf("%d", fee)))
    h.Write([]byte(txnType))
    return hex.EncodeToString(h.Sum(nil))
}

// handleTransaction handles the submission of a signed transaction
func (bcs *BlockchainServer) handleTransaction(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    
    // Parse the request body
    var tx Transaction
    if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
        http.Error(w, fmt.Sprintf("Failed to parse request body: %v", err), http.StatusBadRequest)
        return
    }
    
    // Verify the transaction signature
    if !tx.VerifyTxn() {
        http.Error(w, "Invalid transaction signature", http.StatusBadRequest)
        return
    }
    
    // Add the transaction to the pool
    bcs.blockchain.AddTransactionToTransactionPool(&tx)
    
    // Respond with success
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "transactionHash": tx.TransactionHash,
        "message":         "Transaction submitted to mempool",
    })
}
```

### Phase 3: Update Transaction Processing

#### 3.1 Update Transaction.Data Handling

Ensure consistent handling of Transaction.Data in JSON format:

```go
// In blockchain_struct.go

// validateMCPTransaction validates MCP-specific transactions
func (bc *BlockchainStruct) validateMCPTransaction(transaction *Transaction) bool {
    // Validate based on transaction type
    switch transaction.Type {
    case TransactionTypeMCPRegisterCapability:
        // Parse the capability descriptor from transaction data using JSON
        var registerData MCPRegisterCapabilityData
        if err := json.Unmarshal(transaction.Data, &registerData); err != nil {
            agentlog.LogError("Failed to unmarshal capability registration data:", err)
            return false
        }
        
        // Validate the capability registration
        return bc.validateCapabilityRegistration(transaction, registerData)
        
    case TransactionTypeMCPInvokeCapability:
        // Parse the context record from transaction data using JSON
        var invokeData MCPInvokeCapabilityData
        if err := json.Unmarshal(transaction.Data, &invokeData); err != nil {
            agentlog.LogError("Failed to unmarshal capability invocation data:", err)
            return false
        }
        
        // Validate the capability invocation
        return bc.validateCapabilityInvocation(transaction, invokeData)
        
    case TransactionTypeMCPUpdateCapability:
        // Parse the update data from transaction data using JSON
        var updateData MCPUpdateCapabilityData
        if err := json.Unmarshal(transaction.Data, &updateData); err != nil {
            agentlog.LogError("Failed to unmarshal capability update data:", err)
            return false
        }
        
        // Validate the capability update
        return bc.validateCapabilityUpdate(transaction, updateData)
        
    default:
        // Not an MCP transaction
        return true
    }
}
```

#### 3.2 Update Transaction Hashing and Signature Verification

Ensure consistent hashing and signature verification using Protobuf:

```go
// In transaction.go

// ToProtoForHashing creates a TransactionProto specifically for hashing (used in signing/verification).
// It omits fields that are not part of the signed content.
func (t *Transaction) ToProtoForHashing() (*TransactionProto, error) {
    ts := timestamppb.New(time.Unix(t.Timestamp, 0))
    
    // For hashing and signature verification, we use the raw Data bytes
    // This ensures consistent hashing regardless of the Data content
    return &TransactionProto{
        From:      t.From,
        To:        t.To,
        Value:     t.Value,
        Data:      t.Data, // Use the raw Data bytes as-is
        Timestamp: ts,
        Fee:       t.Fee,
        Type:      t.Type,
    }, nil
}
```

### Phase 4: Update Database Operations

#### 4.1 Update Capability Storage

Ensure capabilities are stored using Protobuf:

```go
// In leveldb_mcp.go

// SaveCapability saves a capability descriptor to the database
func (db *LevelDB) SaveCapability(capability interface{}) error {
    // Get the capability ID
    baseDesc, err := getBaseDescriptorFromInterface(capability)
    if err != nil {
        return fmt.Errorf("failed to get base descriptor: %w", err)
    }
    
    // Convert the capability to Protobuf
    container, err := ConvertToCapabilityDescriptorContainerProto(capability)
    if err != nil {
        return fmt.Errorf("failed to convert capability to protobuf: %w", err)
    }
    
    // Marshal the Protobuf message
    data, err := proto.Marshal(container)
    if err != nil {
        return fmt.Errorf("failed to marshal capability protobuf: %w", err)
    }
    
    // Save the capability to the database
    key := fmt.Sprintf("capability:%s", baseDesc.ID)
    return db.Put([]byte(key), data, nil)
}

// GetCapabilityByID retrieves a capability descriptor from the database by ID
func (db *LevelDB) GetCapabilityByID(id string) (interface{}, error) {
    // Get the capability from the database
    key := fmt.Sprintf("capability:%s", id)
    data, err := db.Get([]byte(key), nil)
    if err != nil {
        return nil, fmt.Errorf("failed to get capability from database: %w", err)
    }
    
    // Unmarshal the Protobuf message
    container := &pb.CapabilityDescriptorContainerProto{}
    if err := proto.Unmarshal(data, container); err != nil {
        // Try JSON as fallback for backward compatibility
        var jsonMap map[string]interface{}
        if jsonErr := json.Unmarshal(data, &jsonMap); jsonErr == nil {
            // Convert JSON to Go struct based on capability type
            return convertJSONToCapability(jsonMap)
        }
        return nil, fmt.Errorf("failed to unmarshal capability protobuf: %w", err)
    }
    
    // Convert the Protobuf message to a Go struct
    return ConvertProtoToCapabilityDescriptor(container)
}
```

#### 4.2 Update Context Record Storage

Ensure context records are stored using Protobuf:

```go
// In leveldb_mcp.go

// SaveContextRecord saves a context record to the database
func (db *LevelDB) SaveContextRecord(record *ContextRecord) error {
    // Convert the context record to Protobuf
    protoRecord, err := ConvertContextRecordToProto(*record)
    if err != nil {
        return fmt.Errorf("failed to convert context record to protobuf: %w", err)
    }
    
    // Marshal the Protobuf message
    data, err := proto.Marshal(protoRecord)
    if err != nil {
        return fmt.Errorf("failed to marshal context record protobuf: %w", err)
    }
    
    // Save the context record to the database
    key := fmt.Sprintf("context:%s", record.ID)
    return db.Put([]byte(key), data, nil)
}

// GetContextRecord retrieves a context record from the database by ID
func (db *LevelDB) GetContextRecord(id string) (*ContextRecord, error) {
    // Get the context record from the database
    key := fmt.Sprintf("context:%s", id)
    data, err := db.Get([]byte(key), nil)
    if err != nil {
        return nil, fmt.Errorf("failed to get context record from database: %w", err)
    }
    
    // Unmarshal the Protobuf message
    protoRecord := &pb.ContextRecordProto{}
    if err := proto.Unmarshal(data, protoRecord); err != nil {
        // Try JSON as fallback for backward compatibility
        var record ContextRecord
        if jsonErr := json.Unmarshal(data, &record); jsonErr == nil {
            return &record, nil
        }
        return nil, fmt.Errorf("failed to unmarshal context record protobuf: %w", err)
    }
    
    // Convert the Protobuf message to a Go struct
    record, err := ConvertProtoToContextRecord(protoRecord)
    if err != nil {
        return nil, fmt.Errorf("failed to convert protobuf to context record: %w", err)
    }
    
    return &record, nil
}
```

### Phase 5: Update Tests

#### 5.1 Update Test Helpers

Create test helpers to ensure consistent data handling in tests, including support for the two-step registration process:

```go
// In mcp_test_helpers.go

// CreateTestCapabilityRegistrationTransaction creates a test transaction for capability registration
func CreateTestCapabilityRegistrationTransaction(from string, capability interface{}, fee uint64) (*Transaction, error) {
    // Create transaction data using JSON
    txnData, err := json.Marshal(map[string]interface{}{
        "capabilityDescriptor": capability,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to marshal transaction data: %w", err)
    }
    
    // Create MCP transaction
    return NewMCPTransaction(from, "", 0, txnData, TransactionTypeMCPRegisterCapability, fee), nil
}

// CreateTestCapabilityInvocationTransaction creates a test transaction for capability invocation
func CreateTestCapabilityInvocationTransaction(from string, contextRecord ContextRecord, fee uint64) (*Transaction, error) {
    // Create transaction data using JSON
    txnData, err := json.Marshal(map[string]interface{}{
        "contextRecord": contextRecord,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to marshal transaction data: %w", err)
    }
    
    // Create MCP transaction
    return NewMCPTransaction(from, "", 0, txnData, TransactionTypeMCPInvokeCapability, fee), nil
}

// CreateTestCapabilityUpdateTransaction creates a test transaction for capability update
func CreateTestCapabilityUpdateTransaction(from string, capabilityID string, capabilityType string, capability interface{}, fee uint64) (*Transaction, error) {
    // Create transaction data using JSON
    txnData, err := json.Marshal(map[string]interface{}{
        "capabilityID":         capabilityID,
        "capabilityType":       capabilityType,
        "capabilityDescriptor": capability,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to marshal transaction data: %w", err)
    }
    
    // Create MCP transaction
    return NewMCPTransaction(from, "", 0, txnData, TransactionTypeMCPUpdateCapability, fee), nil
}

// PrepareTestCapabilityRegistration simulates the prepare_registration endpoint for testing
func PrepareTestCapabilityRegistration(ownerAddress string, descriptor interface{}, fee uint64) (*MCPPrepareCapabilityRegistrationResponse, error) {
    // Convert descriptor to JSON
    descriptorJSON, err := json.Marshal(descriptor)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal descriptor: %w", err)
    }
    
    // Create request data
    requestData := MCPPrepareCapabilityRegistrationRequest{
        OwnerAddress: ownerAddress,
        Descriptor:   descriptorJSON,
        Fee:          fee,
    }
    
    // Generate a unique capability ID
    // This is a simplified example - actual implementation would use a more robust ID generation method
    descriptorMap, ok := descriptor.(map[string]interface{})
    if !ok {
        return nil, fmt.Errorf("invalid descriptor format")
    }
    
    // Set a server-generated ID
    capabilityID := fmt.Sprintf("test-cap-%d", time.Now().UnixNano())
    descriptorMap["id"] = capabilityID
    
    // Create transaction data with the server-finalized descriptor
    txnData, err := json.Marshal(map[string]interface{}{
        "capabilityDescriptor": descriptorMap,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to marshal transaction data: %w", err)
    }
    
    // Create an unsigned transaction
    timestamp := time.Now().Unix()
    txnType := string(TransactionTypeMCPRegisterCapability)
    
    // Calculate the transaction hash
    h := sha256.New()
    h.Write([]byte(ownerAddress))
    h.Write([]byte(""))
    h.Write([]byte("0"))
    h.Write(txnData)
    h.Write([]byte(fmt.Sprintf("%d", timestamp)))
    h.Write([]byte(fmt.Sprintf("%d", fee)))
    h.Write([]byte(txnType))
    txnHash := hex.EncodeToString(h.Sum(nil))
    
    // Create the response
    unsignedTx := UnsignedTransactionDetails{
        From:      ownerAddress,
        To:        "",
        Value:     "0",
        Data:      base64.StdEncoding.EncodeToString(txnData),
        Timestamp: timestamp,
        Fee:       fee,
        Type:      txnType,
    }
    
    response := MCPPrepareCapabilityRegistrationResponse{
        PendingTransactionHash: txnHash,
        UnsignedTransaction:    unsignedTx,
    }
    
    return &response, nil
}
```

#### 5.2 Update Test Cases

Update test cases to use the new helpers and ensure consistent data handling, including tests for the two-step registration process:

```go
// In mcp_blockchain_test.go

func TestMCPTransactionProcessing(t *testing.T) {
    // Create a test database
    dbPath := fmt.Sprintf("testdb_%d", time.Now().UnixNano())
    db, err := NewLevelDB(dbPath)
    if err != nil {
        t.Fatalf("Failed to create test database: %v", err)
    }
    
    // Create a genesis block
    genesisBlock := NewBlock(nil, 0, 0)
    
    // Create a test blockchain
    chainID := fmt.Sprintf("test_chain_%d", time.Now().UnixNano())
    bc, err := NewBlockchain(genesisBlock, chainID, "test_miner_mcp_processing", db)
    if err != nil {
        t.Fatalf("Failed to create test blockchain: %v", err)
    }
    
    // Create a test wallet
    wallet, _ := NewWallet()
    from := wallet.GetAddress()
    registrationNetworkFee := uint64(10) // Network fee for registration
    capabilityGasFee := uint64(50)       // GasFeeNRN for the capability itself
    
    // Create a test resource descriptor
    resourceDesc := ResourceDescriptor{
        BaseDescriptor: BaseDescriptor{
            // ID will be set by the server during preparation
            Name:           "Test Resource",
            Owner:          from,
            Version:        "1.0.0",
            Description:    "Test resource for unit testing",
            GasFeeNRN:      capabilityGasFee, // Set the capability's invocation fee
            Timestamp:      time.Now().Unix(),
            CustomMetadata: map[string]interface{}{"key1": "value1"},
            CapabilityType: CapabilityTypeResource,
        },
        ResourceType: ResourceTypeFile,
        ContentHash:  "sha256:1234567890abcdef",
        Schema: PluginSchemaDetail{
            Summary:       "This is resource 2 (API)",
            LocationHints: []string{"http://api.example.com/res2"},
        },
    }
    
    // Test the two-step registration process
    
    // Step 1: Prepare registration
    prepareResponse, err := PrepareTestCapabilityRegistration(from, resourceDesc, registrationNetworkFee)
    if err != nil {
        t.Fatalf("Failed to prepare registration: %v", err)
    }
    
    // Step 2: Create and sign the transaction
    // Decode the base64-encoded transaction data
    txnDataBytes, err := base64.StdEncoding.DecodeString(prepareResponse.UnsignedTransaction.Data)
    if err != nil {
        t.Fatalf("Failed to decode transaction data: %v", err)
    }
    
    // Create a transaction with the server-provided data
    txn := &Transaction{
        From:      prepareResponse.UnsignedTransaction.From,
        To:        prepareResponse.UnsignedTransaction.To,
        Value:     0, // Value is "0" in the response, convert to uint64
        Data:      txnDataBytes,
        Timestamp: prepareResponse.UnsignedTransaction.Timestamp,
        Fee:       prepareResponse.UnsignedTransaction.Fee,
        Type:      TransactionType(prepareResponse.UnsignedTransaction.Type),
    }
    
    // Sign the transaction
    err = wallet.SignTransaction(txn)
    if err != nil {
        t.Fatalf("Failed to sign transaction: %v", err)
    }
    
    // Add the transaction to the pool
    bc.AddTransactionToTransactionPool(txn)
    
    // ... rest of the test ...
}

// TestPrepareRegistrationEndpoint tests the prepare_registration endpoint
func TestPrepareRegistrationEndpoint(t *testing.T) {
    // Create a test server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Simulate the prepare_registration endpoint
        if r.URL.Path != "/mcp/capability/prepare_registration" {
            t.Fatalf("Expected request to /mcp/capability/prepare_registration, got %s", r.URL.Path)
        }
        
        // Parse the request body
        var requestData MCPPrepareCapabilityRegistrationRequest
        if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
            t.Fatalf("Failed to parse request body: %v", err)
        }
        
        // Generate a response
        response, err := PrepareTestCapabilityRegistration(requestData.OwnerAddress, requestData.Descriptor, requestData.Fee)
        if err != nil {
            t.Fatalf("Failed to prepare registration: %v", err)
        }
        
        // Send the response
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(response)
    }))
    defer server.Close()
    
    // Create a test client
    client := server.Client()
    
    // Create a test resource descriptor
    resourceDesc := map[string]interface{}{
        "name":           "Test Resource",
        "owner":          "test-owner",
        "version":        "1.0.0",
        "description":    "Test resource for unit testing",
        "gasFeeNRN":      uint64(50),
        "timestamp":      time.Now().Unix(),
        "customMetadata": map[string]interface{}{"key1": "value1"},
        "capabilityType": "RESOURCE",
        "resourceType":   "FILE",
        "contentHash":    "sha256:1234567890abcdef",
        "schema": map[string]interface{}{
            "summary":       "This is resource 2 (API)",
            "locationHints": []string{"http://api.example.com/res2"},
        },
    }
    
    // Create the request body
    requestBody, err := json.Marshal(map[string]interface{}{
        "ownerAddress": "test-owner",
        "descriptor":   resourceDesc,
        "fee":          uint64(10),
    })
    if err != nil {
        t.Fatalf("Failed to marshal request body: %v", err)
    }
    
    // Send the request
    resp, err := client.Post(server.URL+"/mcp/capability/prepare_registration", "application/json", bytes.NewBuffer(requestBody))
    if err != nil {
        t.Fatalf("Failed to send request: %v", err)
    }
    defer resp.Body.Close()
    
    // Check the response status code
    if resp.StatusCode != http.StatusOK {
        t.Fatalf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
    }
    
    // Parse the response
    var response MCPPrepareCapabilityRegistrationResponse
    if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
        t.Fatalf("Failed to parse response: %v", err)
    }
    
    // Verify the response
    if response.PendingTransactionHash == "" {
        t.Fatalf("Expected non-empty pendingTransactionHash")
    }
    if response.UnsignedTransaction.From != "test-owner" {
        t.Fatalf("Expected From to be 'test-owner', got '%s'", response.UnsignedTransaction.From)
    }
    if response.UnsignedTransaction.Fee != uint64(10) {
        t.Fatalf("Expected Fee to be 10, got %d", response.UnsignedTransaction.Fee)
    }
    
    // Decode the transaction data
    txnDataBytes, err := base64.StdEncoding.DecodeString(response.UnsignedTransaction.Data)
    if err != nil {
        t.Fatalf("Failed to decode transaction data: %v", err)
    }
    
    // Parse the transaction data
    var txnData map[string]interface{}
    if err := json.Unmarshal(txnDataBytes, &txnData); err != nil {
        t.Fatalf("Failed to parse transaction data: %v", err)
    }
    
    // Verify the transaction data
    descriptor, ok := txnData["capabilityDescriptor"].(map[string]interface{})
    if !ok {
        t.Fatalf("Expected capabilityDescriptor to be a map, got %T", txnData["capabilityDescriptor"])
    }
    
    // Verify that the server set an ID
    if descriptor["id"] == nil {
        t.Fatalf("Expected server to set an ID")
    }
}
```

### Phase 6: Documentation and Guidelines

#### 6.1 Update API Documentation

Update the API documentation to reflect the new data handling approach:

```yaml
# In openapi.yaml
paths:
  /transaction:
    post:
      tags:
        - blockchain
        - mcp
      summary: Submit a Signed Transaction
      description: |
        Submits a pre-signed transaction to the blockchain network.
        This endpoint is used for various transaction types, including
        standard transfers and MCP operations like capability registration,
        invocation, or updates. The `Transaction.type` field and `Transaction.data`
        structure will determine how it's processed.
        
        Note: Transaction.Data should be in JSON format for all MCP transactions.
      operationId: writeTransaction
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Transaction'
      responses:
        '202':
          description: Transaction accepted by the node and broadcasted. Does not guarantee inclusion in a block.
          content:
            application/json:
              schema:
                type: object
                properties:
                  transactionHash:
                    type: string
                  message:
                    type: string
                    example: "Transaction submitted to mempool"
```

#### 6.2 Create Developer Guidelines

Create a developer guide for working with the new data handling approach:

```markdown
# Developer Guidelines for Data Serialization

## JSON vs. Protobuf Usage

- **JSON for Transit**: All network API requests/responses use JSON format.
- **Protobuf for Storage**: All database storage uses Protobuf format.
- **Protobuf for Cryptographic Operations**: All hashing and signature operations use Protobuf.
- **JSON for Transaction.Data**: Transaction.Data remains in JSON format for transit and invocation.

## Working with Transaction.Data

When creating a transaction with MCP-specific data:

1. Create your data structure (e.g., `MCPRegisterCapabilityData`, `MCPInvokeCapabilityData`).
2. Serialize it to JSON using `json.Marshal()`.
3. Set the resulting bytes as `Transaction.Data`.

Example:
```go
// Create transaction data
txnData, err := json.Marshal(map[string]interface{}{
    "capabilityDescriptor": resourceDesc,
})
if err != nil {
    return nil, fmt.Errorf("failed to marshal transaction data: %w", err)
}

// Create MCP transaction
txn := NewMCPTransaction(from, "", 0, txnData, TransactionTypeMCPRegisterCapability, fee)
```

When processing a transaction with MCP-specific data:

1. Deserialize `Transaction.Data` from JSON using `json.Unmarshal()`.
2. Process the resulting data structure.

Example:
```go
var registerData MCPRegisterCapabilityData
if err := json.Unmarshal(transaction.Data, &registerData); err != nil {
    return fmt.Errorf("failed to unmarshal capability registration data: %w", err)
}
```

## Working with Database Storage

When storing data in the database:

1. Convert your Go struct to a Protobuf message using the appropriate conversion function.
2. Serialize the Protobuf message using `proto.Marshal()`.
3. Store the resulting bytes in the database.

Example:
```go
// Convert the capability to Protobuf
container, err := ConvertToCapabilityDescriptorContainerProto(capability)
if err != nil {
    return fmt.Errorf("failed to convert capability to protobuf: %w", err)
}

// Marshal the Protobuf message
data, err := proto.Marshal(container)
if err != nil {
    return fmt.Errorf("failed to marshal capability protobuf: %w", err)
}

// Save the capability to the database
key := fmt.Sprintf("capability:%s", baseDesc.ID)
return db.Put([]byte(key), data, nil)
```

When retrieving data from the database:

1. Deserialize the bytes from the database using `proto.Unmarshal()`.
2. Convert the Protobuf message to a Go struct using the appropriate conversion function.

Example:
```go
// Unmarshal the Protobuf message
container := &pb.CapabilityDescriptorContainerProto{}
if err := proto.Unmarshal(data, container); err != nil {
    return nil, fmt.Errorf("failed to unmarshal capability protobuf: %w", err)
}

// Convert the Protobuf message to a Go struct
return ConvertProtoToCapabilityDescriptor(container)
```
```

## Implementation Timeline

### Week 1: Preparation and Definition

1. Define clear boundaries between JSON and Protobuf usage
2. Create missing Protobuf definitions
3. Update conversion functions
4. Document the new approach

### Week 2: Core Implementation

1. Implement the improved two-step registration flow with `/prepare_registration`
2. Remove legacy initiate/finalize endpoints
3. Update transaction processing to ensure consistent data handling
4. Update database operations to use Protobuf for storage

### Week 3: Testing and Validation

1. Update test helpers and test cases
2. Add specific tests for the two-step registration process
3. Validate the implementation against the defined boundaries
4. Ensure backward compatibility with existing data

### Week 4: Documentation and Deployment

1. Update API documentation
2. Create developer guidelines
3. Deploy the changes to the test environment
4. Validate the implementation in the test environment

## Conclusion

This implementation plan addresses the inconsistencies in Transaction.Data representation by establishing clear boundaries between JSON and Protobuf usage. By following this plan, we will ensure consistent data handling throughout the application lifecycle, improving data integrity, reducing developer confusion, and enhancing the overall robustness of the system.

The key principles of this plan are:

1. **JSON for Transit, Protobuf for Storage and Hashing**
2. **Consistent Transaction.Data Handling**
3. **Improved Two-Step Registration Flow**
4. **Complete Protobuf Coverage**

The improved two-step registration flow using the `/prepare_registration` endpoint provides several benefits:

1. **Server-Generated IDs**: The server can generate unique, canonical IDs for capabilities, ensuring consistency and preventing ID collisions.
2. **Consistent Serialization**: By providing the client with the exact data structure to sign, we ensure consistent serialization between preparation and submission.
3. **Simplified Client Implementation**: Clients don't need to implement complex ID generation logic, they just need to sign the data provided by the server.
4. **Improved Security**: The server can validate the capability descriptor before generating an ID, preventing malformed or malicious descriptors from being registered.

By adhering to these principles, we will create a more maintainable and reliable system that leverages the strengths of both JSON and Protobuf while minimizing the potential for inconsistencies and errors.

---

<div class="footer-links">


© 2025 KNIRV Network
</div>
