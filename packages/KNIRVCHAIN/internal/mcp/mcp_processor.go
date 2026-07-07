package mcp

import (
	"encoding/json"
	"fmt"
	"log"

	"KNIRVCHAIN/internal/blockchain"
	"KNIRVCHAIN/internal/wallet"

	"github.com/syndtr/goleveldb/leveldb"
)

// Stub type definitions - these need to be properly implemented
type AgentManager struct{}
type Wallet = wallet.WalletImpl

// Transaction type alias for MCP processing - use blockchain.Transaction which has Type field
type Transaction = blockchain.Transaction

// MCP Transaction type constants
const (
	TransactionTypeMCPRegisterCapability = "register_capability_txn"
	TransactionTypeMCPInvokeCapability   = "invoke_capability_txn"
	TransactionTypeMCPUpdateCapability   = "update_capability_txn"
	TransactionTypeMCPDeleteCapability   = "delete_capability_txn"
)

// MCPProcessorImpl handles MCP-specific transaction validation and processing.
type MCPProcessorImpl struct {
	db           *leveldb.DB
	agentManager *AgentManager
	wallet       *Wallet
}

// NewMCPProcessor creates a new MCPProcessor.
func NewMCPProcessor(db *leveldb.DB) *MCPProcessorImpl {
	if db == nil {
		log.Printf("Warning: NewMCPProcessor called with a nil database connection.")
	}
	return &MCPProcessorImpl{db: db}
}

// NewMCPProcessorWithAgentManager creates a new MCPProcessor with AgentManager support.
func NewMCPProcessorWithAgentManager(db *leveldb.DB, agentManager *AgentManager, wallet *Wallet) *MCPProcessorImpl {
	if db == nil {
		log.Printf("Warning: NewMCPProcessorWithAgentManager called with a nil database connection.")
	}
	return &MCPProcessorImpl{
		db:           db,
		agentManager: agentManager,
		wallet:       wallet,
	}
}

// validateMCPTransaction validates MCP-specific transactions
func (mcp *MCPProcessorImpl) validateMCPTransaction(transaction *Transaction) bool {
	// Basic validation checks
	if transaction == nil {
		log.Printf("Error: Transaction is nil")
		return false
	}

	if len(transaction.Data) == 0 {
		log.Printf("Error: Transaction data is empty")
		return false
	}

	if transaction.From == "" {
		log.Printf("Error: Transaction from address is empty")
		return false
	}

	// Parse transaction data to determine type and validate accordingly
	var txData map[string]interface{}
	if err := json.Unmarshal(transaction.Data, &txData); err != nil {
		log.Printf("Error: Failed to unmarshal transaction data: %v", err)
		return false
	}

	// Determine transaction type from the transaction Type field or data content
	txType := transaction.Type
	if txType == "" {
		// Try to infer type from data content
		if _, hasCapabilityDescriptor := txData["capabilityDescriptor"]; hasCapabilityDescriptor {
			txType = TransactionTypeMCPRegisterCapability
		} else if _, hasContextRecord := txData["contextRecord"]; hasContextRecord {
			txType = TransactionTypeMCPInvokeCapability
		}
	}

	// Validate based on transaction type
	switch txType {
	case TransactionTypeMCPRegisterCapability:
		return mcp.validateCapabilityRegistration(transaction)
	case TransactionTypeMCPInvokeCapability:
		return mcp.validateCapabilityInvocation(transaction)
	case TransactionTypeMCPUpdateCapability:
		return mcp.validateCapabilityUpdate(transaction)
	case TransactionTypeMCPDeleteCapability:
		return mcp.validateCapabilityDeletion(transaction)
	default:
		log.Printf("Error: Unknown MCP transaction type: %s", txType)
		return false
	}
}

// validateCapabilityRegistration validates a capability registration transaction
func (mcp *MCPProcessorImpl) validateCapabilityRegistration(transaction *Transaction) bool {
	if len(transaction.Data) == 0 {
		log.Printf("Error: Missing transaction data")
		return false
	}

	// Parse transaction data
	var txData map[string]interface{}
	if err := json.Unmarshal(transaction.Data, &txData); err != nil {
		log.Printf("Error: Failed to unmarshal registration transaction data: %v", err)
		return false
	}

	// Check for capability descriptor in the data
	capabilityDescriptor, hasCapabilityDescriptor := txData["capabilityDescriptor"]
	if !hasCapabilityDescriptor {
		log.Printf("Error: Missing capabilityDescriptor in registration transaction")
		return false
	}

	// Convert to map for validation
	capDescMap, ok := capabilityDescriptor.(map[string]interface{})
	if !ok {
		log.Printf("Error: capabilityDescriptor is not a valid object")
		return false
	}

	// Validate required fields in capability descriptor
	capabilityID, ok := capDescMap["id"].(string)
	if !ok || capabilityID == "" {
		log.Printf("Error: Missing or invalid capability ID")
		return false
	}

	name, ok := capDescMap["name"].(string)
	if !ok || name == "" {
		log.Printf("Error: Missing or invalid capability name")
		return false
	}

	owner, ok := capDescMap["owner"].(string)
	if !ok || owner == "" {
		log.Printf("Error: Missing or invalid capability owner")
		return false
	}

	// Validate owner matches transaction sender
	if owner != transaction.From {
		log.Printf("Error: Transaction sender (%s) does not match capability owner (%s)", transaction.From, owner)
		return false
	}

	// Validate capability type
	capType, ok := capDescMap["capabilityType"].(string)
	if !ok || capType == "" {
		log.Printf("Error: Missing or invalid capabilityType")
		return false
	}

	if !IsValidCapabilityType(CapabilityType(capType)) {
		log.Printf("Error: Invalid capability type: %s", capType)
		return false
	}

	// Validate version
	version, ok := capDescMap["version"].(string)
	if !ok || version == "" {
		log.Printf("Error: Missing or invalid version")
		return false
	}

	// Validate description
	description, ok := capDescMap["description"].(string)
	if !ok || description == "" {
		log.Printf("Error: Missing or invalid description")
		return false
	}

	// Validate gas fee (optional, can be 0)
	if gasFeeInterface, exists := capDescMap["gasFeeNRN"]; exists {
		switch gasFee := gasFeeInterface.(type) {
		case float64:
			if gasFee < 0 {
				log.Printf("Error: Gas fee cannot be negative")
				return false
			}
		case uint64:
			// Already valid
		default:
			log.Printf("Error: Invalid gas fee type")
			return false
		}
	}

	log.Printf("Capability registration validation passed for ID: %s, Type: %s", capabilityID, capType)
	return true
}

// validateCapabilityInvocation validates a capability invocation transaction
func (mcp *MCPProcessorImpl) validateCapabilityInvocation(transaction *Transaction) bool {
	if len(transaction.Data) == 0 {
		log.Printf("Error: Missing transaction data")
		return false
	}

	// Parse transaction data
	var txData map[string]interface{}
	if err := json.Unmarshal(transaction.Data, &txData); err != nil {
		log.Printf("Error: Failed to unmarshal invocation transaction data: %v", err)
		return false
	}

	// Check for context record in the data
	contextRecordData, hasContextRecord := txData["contextRecord"]
	if !hasContextRecord {
		log.Printf("Error: Missing contextRecord in invocation transaction")
		return false
	}

	// Convert to map for validation
	contextMap, ok := contextRecordData.(map[string]interface{})
	if !ok {
		log.Printf("Error: contextRecord is not a valid object")
		return false
	}

	// Validate required fields in context record
	capabilityID, ok := contextMap["capabilityID"].(string)
	if !ok || capabilityID == "" {
		log.Printf("Error: Missing or invalid capabilityID in context record")
		return false
	}

	initiator, ok := contextMap["initiator"].(string)
	if !ok || initiator == "" {
		log.Printf("Error: Missing or invalid initiator in context record")
		return false
	}

	// Validate initiator matches transaction sender
	if initiator != transaction.From {
		log.Printf("Error: Transaction sender (%s) does not match context record initiator (%s)", transaction.From, initiator)
		return false
	}

	// Validate interaction type
	interactionTypeStr, ok := contextMap["interactionType"].(string)
	if !ok || interactionTypeStr == "" {
		log.Printf("Error: Missing or invalid interactionType in context record")
		return false
	}

	// Validate that it's a valid interaction type
	validInteractionTypes := []string{
		string(InteractionTypeToolInvocation),
		string(InteractionTypePromptUsage),
		string(InteractionTypeResourceAccess),
		string(InteractionTypePluginExecution),
		string(InteractionTypeSamplingRequestSent),
		string(InteractionTypeSamplingResponseReceived),
		string(InteractionTypeMemoryWrite),
	}

	isValidInteraction := false
	for _, validType := range validInteractionTypes {
		if interactionTypeStr == validType {
			isValidInteraction = true
			break
		}
	}

	if !isValidInteraction {
		log.Printf("Error: Invalid interaction type: %s", interactionTypeStr)
		return false
	}

	// Validate timestamp (should be recent)
	if timestampInterface, exists := contextMap["timestamp"]; exists {
		switch timestamp := timestampInterface.(type) {
		case float64:
			if timestamp <= 0 {
				log.Printf("Error: Invalid timestamp")
				return false
			}
		case int64:
			if timestamp <= 0 {
				log.Printf("Error: Invalid timestamp")
				return false
			}
		default:
			log.Printf("Error: Invalid timestamp type")
			return false
		}
	}

	log.Printf("Capability invocation validation passed for capability: %s, interaction: %s", capabilityID, interactionTypeStr)
	return true
}

// validateCapabilityUpdate validates a capability update transaction
func (mcp *MCPProcessorImpl) validateCapabilityUpdate(transaction *Transaction) bool {
	if len(transaction.Data) == 0 {
		log.Printf("Error: Missing transaction data")
		return false
	}

	// Parse transaction data
	var txData map[string]interface{}
	if err := json.Unmarshal(transaction.Data, &txData); err != nil {
		log.Printf("Error: Failed to unmarshal update transaction data: %v", err)
		return false
	}

	// Validate required fields for capability update
	capabilityID, ok := txData["capabilityID"].(string)
	if !ok || capabilityID == "" {
		log.Printf("Error: Missing or invalid capabilityID in update transaction")
		return false
	}

	// Validate owner matches transaction sender
	owner, ok := txData["owner"].(string)
	if !ok || owner == "" {
		log.Printf("Error: Missing or invalid owner in update transaction")
		return false
	}

	if owner != transaction.From {
		log.Printf("Error: Transaction sender (%s) does not match capability owner (%s)", transaction.From, owner)
		return false
	}

	// Validate capability type
	capType, ok := txData["capabilityType"].(string)
	if !ok || capType == "" {
		log.Printf("Error: Missing or invalid capabilityType in update transaction")
		return false
	}

	if !IsValidCapabilityType(CapabilityType(capType)) {
		log.Printf("Error: Invalid capability type: %s", capType)
		return false
	}

	return true
}

// validateCapabilityDeletion validates a capability deletion transaction
func (mcp *MCPProcessorImpl) validateCapabilityDeletion(transaction *Transaction) bool {
	if len(transaction.Data) == 0 {
		log.Printf("Error: Missing transaction data")
		return false
	}

	// Parse transaction data
	var txData map[string]interface{}
	if err := json.Unmarshal(transaction.Data, &txData); err != nil {
		log.Printf("Error: Failed to unmarshal deletion transaction data: %v", err)
		return false
	}

	// Validate required fields for capability deletion
	capabilityID, ok := txData["capabilityID"].(string)
	if !ok || capabilityID == "" {
		log.Printf("Error: Missing or invalid capabilityID in deletion transaction")
		return false
	}

	// Validate owner matches transaction sender
	owner, ok := txData["owner"].(string)
	if !ok || owner == "" {
		log.Printf("Error: Missing or invalid owner in deletion transaction")
		return false
	}

	if owner != transaction.From {
		log.Printf("Error: Transaction sender (%s) does not match capability owner (%s)", transaction.From, owner)
		return false
	}

	return true
}

// ProcessMCPTransaction processes an MCP transaction
func (mcp *MCPProcessorImpl) ProcessMCPTransaction(transaction *Transaction) error {
	// First validate the transaction
	if !mcp.validateMCPTransaction(transaction) {
		return fmt.Errorf("invalid MCP transaction")
	}

	// Parse transaction data
	var txData map[string]interface{}
	if err := json.Unmarshal(transaction.Data, &txData); err != nil {
		return fmt.Errorf("failed to unmarshal transaction data: %w", err)
	}

	// Determine transaction type and process accordingly
	txType := transaction.Type
	if txType == "" {
		// Try to infer type from data content
		if _, hasCapabilityDescriptor := txData["capabilityDescriptor"]; hasCapabilityDescriptor {
			txType = TransactionTypeMCPRegisterCapability
		} else if _, hasContextRecord := txData["contextRecord"]; hasContextRecord {
			txType = TransactionTypeMCPInvokeCapability
		}
	}

	log.Printf("Processing MCP transaction: %s, Type: %s", transaction.TransactionHash, txType)

	// Process based on transaction type
	switch txType {
	case TransactionTypeMCPRegisterCapability:
		return mcp.processCapabilityRegistration(transaction, txData)
	case TransactionTypeMCPInvokeCapability:
		return mcp.processCapabilityInvocation(transaction, txData)
	case TransactionTypeMCPUpdateCapability:
		return mcp.processCapabilityUpdate(transaction, txData)
	case TransactionTypeMCPDeleteCapability:
		return mcp.processCapabilityDeletion(transaction, txData)
	default:
		return fmt.Errorf("unknown MCP transaction type: %s", txType)
	}
}

// processCapabilityRegistration processes a capability registration transaction
func (mcp *MCPProcessorImpl) processCapabilityRegistration(transaction *Transaction, txData map[string]interface{}) error {
	log.Printf("Processing capability registration for transaction: %s", transaction.TransactionHash)

	// Extract capability descriptor
	capabilityDescriptor, ok := txData["capabilityDescriptor"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid capability descriptor")
	}

	// Convert to concrete capability type
	capability, err := ConvertMapToCapability(capabilityDescriptor)
	if err != nil {
		return fmt.Errorf("failed to convert capability descriptor: %w", err)
	}

	// Store capability in database (if database is available)
	if mcp.db != nil {
		capabilityID := capabilityDescriptor["id"].(string)
		capabilityJSON, err := json.Marshal(capability)
		if err != nil {
			return fmt.Errorf("failed to marshal capability: %w", err)
		}

		if err := mcp.db.Put([]byte("capability:"+capabilityID), capabilityJSON, nil); err != nil {
			return fmt.Errorf("failed to store capability: %w", err)
		}

		log.Printf("Successfully registered capability: %s", capabilityID)
	} else {
		log.Printf("Warning: No database available, capability registration not persisted")
	}

	return nil
}

// processCapabilityInvocation processes a capability invocation transaction
func (mcp *MCPProcessorImpl) processCapabilityInvocation(transaction *Transaction, txData map[string]interface{}) error {
	log.Printf("Processing capability invocation for transaction: %s", transaction.TransactionHash)

	// Extract context record
	contextRecordData, ok := txData["contextRecord"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid context record")
	}

	// Store context record in database (if database is available)
	if mcp.db != nil {
		contextID := contextRecordData["id"].(string)
		contextJSON, err := json.Marshal(contextRecordData)
		if err != nil {
			return fmt.Errorf("failed to marshal context record: %w", err)
		}

		if err := mcp.db.Put([]byte("context:"+contextID), contextJSON, nil); err != nil {
			return fmt.Errorf("failed to store context record: %w", err)
		}

		log.Printf("Successfully recorded capability invocation: %s", contextID)
	} else {
		log.Printf("Warning: No database available, context record not persisted")
	}

	return nil
}

// processCapabilityUpdate processes a capability update transaction
func (mcp *MCPProcessorImpl) processCapabilityUpdate(transaction *Transaction, txData map[string]interface{}) error {
	log.Printf("Processing capability update for transaction: %s", transaction.TransactionHash)

	// Extract capability ID
	capabilityID, ok := txData["capabilityID"].(string)
	if !ok {
		return fmt.Errorf("missing capability ID")
	}

	// Store updated capability in database (if database is available)
	if mcp.db != nil {
		updateJSON, err := json.Marshal(txData)
		if err != nil {
			return fmt.Errorf("failed to marshal update data: %w", err)
		}

		if err := mcp.db.Put([]byte("capability:"+capabilityID), updateJSON, nil); err != nil {
			return fmt.Errorf("failed to update capability: %w", err)
		}

		log.Printf("Successfully updated capability: %s", capabilityID)
	} else {
		log.Printf("Warning: No database available, capability update not persisted")
	}

	return nil
}

// processCapabilityDeletion processes a capability deletion transaction
func (mcp *MCPProcessorImpl) processCapabilityDeletion(transaction *Transaction, txData map[string]interface{}) error {
	log.Printf("Processing capability deletion for transaction: %s", transaction.TransactionHash)

	// Extract capability ID
	capabilityID, ok := txData["capabilityID"].(string)
	if !ok {
		return fmt.Errorf("missing capability ID")
	}

	// Delete capability from database (if database is available)
	if mcp.db != nil {
		if err := mcp.db.Delete([]byte("capability:"+capabilityID), nil); err != nil {
			return fmt.Errorf("failed to delete capability: %w", err)
		}

		log.Printf("Successfully deleted capability: %s", capabilityID)
	} else {
		log.Printf("Warning: No database available, capability deletion not persisted")
	}

	return nil
}

// Additional stub methods that might be called by other parts of the system

// GetCapability returns a capability by ID
func (mcp *MCPProcessorImpl) GetCapability(id string) (interface{}, error) {
	if mcp.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	// Retrieve capability from database
	data, err := mcp.db.Get([]byte("capability:"+id), nil)
	if err != nil {
		return nil, fmt.Errorf("capability not found: %s", id)
	}

	// Unmarshal the capability data
	var capabilityData map[string]interface{}
	if err := json.Unmarshal(data, &capabilityData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal capability data: %w", err)
	}

	// Convert to concrete capability type
	capability, err := ConvertMapToCapability(capabilityData)
	if err != nil {
		return nil, fmt.Errorf("failed to convert capability: %w", err)
	}

	return capability, nil
}

// RegisterCapability registers a new capability
func (mcp *MCPProcessorImpl) RegisterCapability(transaction *Transaction) error {
	log.Printf("Registering capability from transaction: %s", transaction.TransactionHash)

	// Parse transaction data
	var txData map[string]interface{}
	if err := json.Unmarshal(transaction.Data, &txData); err != nil {
		return fmt.Errorf("failed to unmarshal transaction data: %w", err)
	}

	return mcp.processCapabilityRegistration(transaction, txData)
}

// InvokeCapability invokes a capability
func (mcp *MCPProcessorImpl) InvokeCapability(transaction *Transaction) error {
	log.Printf("Invoking capability from transaction: %s", transaction.TransactionHash)

	// Parse transaction data
	var txData map[string]interface{}
	if err := json.Unmarshal(transaction.Data, &txData); err != nil {
		return fmt.Errorf("failed to unmarshal transaction data: %w", err)
	}

	return mcp.processCapabilityInvocation(transaction, txData)
}
