package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/syndtr/goleveldb/leveldb/util"
	"google.golang.org/protobuf/proto"

	agentlog "KNIRVROOT/log"
	pb "KNIRVROOT/proto" // Matches proto package name "KNIRVROOT"
	"KNIRVROOT/types"
)

// MCPProcessor handles MCP-specific transaction validation and processing.
type MCPProcessor struct {
	db           *LevelDB
	agentManager *AgentManager
	wallet       *Wallet
	// bc *BlockchainStruct // Avoid circular dependency if possible.
	// Pass necessary blockchain context if methods need it.
}

// NewMCPProcessor creates a new MCPProcessor.
func NewMCPProcessor(db *LevelDB) *MCPProcessor {
	if db == nil {
		log.Printf("Warning: NewMCPProcessor called with a nil database connection.")
	}
	return &MCPProcessor{db: db}
}

// NewMCPProcessorWithAgentManager creates a new MCPProcessor with AgentManager support.
func NewMCPProcessorWithAgentManager(db *LevelDB, agentManager *AgentManager, wallet *Wallet) *MCPProcessor {
	if db == nil {
		log.Printf("Warning: NewMCPProcessorWithAgentManager called with a nil database connection.")
	}
	return &MCPProcessor{
		db:           db,
		agentManager: agentManager,
		wallet:       wallet,
	}
}

// validateMCPTransaction validates MCP-specific transactions
func (mcp *MCPProcessor) validateMCPTransaction(transaction *Transaction) bool {
	// Validate based on transaction type
	switch transaction.Type {
	case TransactionTypeMCPRegisterCapability:
		// Validate capability registration
		return mcp.validateCapabilityRegistration(transaction)
	case TransactionTypeMCPInvokeCapability:
		// Validate capability invocation
		return mcp.validateCapabilityInvocation(transaction)
	case TransactionTypeMCPUpdateCapability:
		// Validate capability update
		return mcp.validateCapabilityUpdate(transaction)
	default:
		// Unknown MCP transaction type
		return false
	}
}

// validateCapabilityRegistration validates a capability registration transaction
func (mcp *MCPProcessor) validateCapabilityRegistration(transaction *Transaction) bool {
	// First try to unmarshal as protobuf
	var registerProto pb.MCPRegisterCapabilityDataProto
	if err := proto.Unmarshal(transaction.Data, &registerProto); err == nil {
		// Successfully parsed as protobuf
		if registerProto.CapabilityDescriptor == nil {
			agentlog.LogError("Missing CapabilityDescriptor in protobuf data", errors.New("invalid protobuf data"))
			return false
		}

		// Convert protobuf to Go struct for validation
		capabilityDescriptor, err := ConvertProtoToCapability(registerProto.CapabilityDescriptor)
		if err != nil {
			agentlog.LogError("Failed to convert protobuf capability:", err)
			return false
		}

		// Check if the transaction sender matches the capability owner
		// Extract owner from the capability descriptor
		var owner string
		switch desc := capabilityDescriptor.(type) {
		case types.ResourceDescriptor:
			owner = desc.BaseDescriptor.Owner
		case types.ToolDescriptor:
			owner = desc.BaseDescriptor.Owner
		case types.PromptDescriptor:
			owner = desc.BaseDescriptor.Owner
		case types.MemoryServiceDescriptor:
			owner = desc.BaseDescriptor.Owner
		default:
			agentlog.LogError("Unsupported capability type from protobuf", errors.New("invalid capability type"))
			return false
		}

		// Verify that the transaction sender matches the capability owner
		if transaction.From != owner {
			agentlog.LogError(fmt.Sprintf("Transaction sender (%s) does not match capability owner (%s)", transaction.From, owner), errors.New("sender-owner mismatch"))
			return false
		}

		// Validate based on capability type
		switch desc := capabilityDescriptor.(type) {
		case types.ResourceDescriptor: // Ensure these types are accessible or defined in this package
			return mcp.validateResourceDescriptor(desc)
		case types.ToolDescriptor: // Ensure these types are accessible or defined in this package
			return mcp.validateToolDescriptor(desc)
		case types.PromptDescriptor: // Ensure these types are accessible or defined in this package
			return mcp.validatePromptDescriptor(desc)
		case types.MemoryServiceDescriptor: // Ensure these types are accessible or defined in this package
			return mcp.validateMemoryServiceDescriptor(desc)
		default:
			agentlog.LogError("Unsupported capability type from protobuf", errors.New("invalid capability type"))
			return false
		}
	}

	// If not protobuf, try JSON format (legacy)
	var capabilityData map[string]interface{}
	if err := json.Unmarshal(transaction.Data, &capabilityData); err != nil {
		agentlog.LogError("Failed to unmarshal capability data as JSON:", err)
		return false
	}

	// Check if the descriptor has the required fields
	descriptorData, ok := capabilityData["capabilityDescriptor"]
	if !ok {
		agentlog.LogError("Missing capabilityDescriptor in transaction data", errors.New("invalid transaction data"))
		return false
	}

	// Convert to JSON for further processing
	descriptorJSON, err := json.Marshal(descriptorData)
	if err != nil {
		agentlog.LogError("Failed to marshal descriptor data:", err)
		return false
	}

	// Get capability type from descriptor's BaseDescriptor
	var baseDesc types.BaseDescriptor
	if err := json.Unmarshal(descriptorJSON, &baseDesc); err != nil {
		agentlog.LogError("Failed to unmarshal BaseDescriptor:", err)
		return false
	}
	if baseDesc.CapabilityType == "" {
		agentlog.LogError("Missing CapabilityType in BaseDescriptor", errors.New("invalid capability type"))
		return false
	}

	// Verify that the transaction sender matches the capability owner
	if transaction.From != baseDesc.Owner {
		agentlog.LogError(fmt.Sprintf("Transaction sender (%s) does not match capability owner (%s)", transaction.From, baseDesc.Owner), errors.New("sender-owner mismatch"))
		return false
	}

	switch baseDesc.CapabilityType {
	case types.CapabilityTypeResource:
		var descriptor types.ResourceDescriptor
		if err := json.Unmarshal(descriptorJSON, &descriptor); err != nil {
			agentlog.LogError("Failed to unmarshal ResourceDescriptor:", err)
			return false // Ensure these types are accessible or defined in this package
		}
		// Validate resource descriptor
		return mcp.validateResourceDescriptor(descriptor)
	case types.CapabilityTypeTool:
		var descriptor types.ToolDescriptor
		if err := json.Unmarshal(descriptorJSON, &descriptor); err != nil {
			agentlog.LogError("Failed to unmarshal ToolDescriptor:", err)
			return false // Ensure these types are accessible or defined in this package
		}
		// Validate tool descriptor
		return mcp.validateToolDescriptor(descriptor)
	case types.CapabilityTypePrompt:
		var descriptor types.PromptDescriptor
		if err := json.Unmarshal(descriptorJSON, &descriptor); err != nil {
			agentlog.LogError("Failed to unmarshal PromptDescriptor:", err)
			return false // Ensure these types are accessible or defined in this package
		}
		// Validate prompt descriptor
		return mcp.validatePromptDescriptor(descriptor)
	case types.CapabilityTypeMemoryService:
		var descriptor types.MemoryServiceDescriptor
		if err := json.Unmarshal(descriptorJSON, &descriptor); err != nil {
			agentlog.LogError("Failed to unmarshal MemoryServiceDescriptor:", err)
			return false // Ensure these types are accessible or defined in this package
		}
		// Validate memory service descriptor
		return mcp.validateMemoryServiceDescriptor(descriptor)
	default:
		agentlog.LogError("Unknown capability type", fmt.Errorf("invalid type: %v", baseDesc.CapabilityType))
		return false
	}
}

// validateCapabilityInvocation validates a capability invocation transaction
func (mcp *MCPProcessor) validateCapabilityInvocation(transaction *Transaction) bool {
	// Parse the MCPInvokeCapabilityData from transaction data
	var invokeData types.MCPInvokeCapabilityData
	if err := json.Unmarshal(transaction.Data, &invokeData); err != nil {
		agentlog.LogError("Failed to unmarshal MCPInvokeCapabilityData:", err)
		return false
	}

	// Get the context record directly from the MCPInvokeCapabilityData
	record := invokeData.ContextRecord

	// Validate context record
	if record.CapabilityID == "" {
		agentlog.LogError("Missing CapabilityID in context record", errors.New("invalid context record"))
		return false
	}

	if record.InteractionType == "" {
		agentlog.LogError("Missing InteractionType in context record", errors.New("invalid context record"))
		return false
	}

	if record.Initiator == "" {
		agentlog.LogError("Missing Initiator in context record", errors.New("invalid context record"))
		return false
	}

	// Verify that the capability exists
	capability, err := mcp.getCapabilityByID(record.CapabilityID)
	if err != nil {
		agentlog.LogError("Failed to get capability:", err)
		return false
	}

	// Verify that the transaction fee is sufficient
	if transaction.Fee < capability.GasFeeNRN {
		agentlog.LogError("Insufficient fee", fmt.Errorf("provided: %d < required: %d", transaction.Fee, capability.GasFeeNRN))
		return false
	}

	return true
}

// validateCapabilityUpdate validates a capability update transaction
func (mcp *MCPProcessor) validateCapabilityUpdate(transaction *Transaction) bool {
	// Parse the capability descriptor from transaction data
	var updateData map[string]interface{}
	if err := json.Unmarshal(transaction.Data, &updateData); err != nil {
		agentlog.LogError("Failed to unmarshal update data:", err)
		return false
	}

	// Debug log the update data structure
	updateDataJSON, _ := json.MarshalIndent(updateData, "", "  ")
	agentlog.LogInfo(fmt.Sprintf("Update data structure: %s", string(updateDataJSON)))

	// Check if the update data has the required fields - try multiple possible field names
	var capabilityID string
	var ok bool

	// Try different case variations
	if capabilityID, ok = updateData["capabilityID"].(string); !ok || capabilityID == "" {
		if capabilityID, ok = updateData["CapabilityID"].(string); !ok || capabilityID == "" {
			if capabilityID, ok = updateData["capabilityId"].(string); !ok || capabilityID == "" {
				agentlog.LogError("Missing or invalid capabilityID in update data (tried capabilityID, CapabilityID, capabilityId)", errors.New("invalid update data"))
				return false
			}
		}
	}

	// Verify that the capability exists with retry logic
	var capability types.BaseDescriptor
	var lastErr error
	for i := 0; i < 5; i++ { // Increased retry attempts
		capability, lastErr = mcp.getCapabilityByID(capabilityID)
		if lastErr == nil {
			break
		}
		time.Sleep(50 * time.Millisecond) // Increased delay
	}
	if lastErr != nil {
		agentlog.LogError(fmt.Sprintf("Failed to get capability %s for update after retries: %v", capabilityID, lastErr), lastErr)
		return false
	}

	// Verify that the sender is the owner of the capability
	if transaction.From != capability.Owner {
		agentlog.LogError(fmt.Sprintf("Sender %s is not the owner of the capability %s (owner: %s)",
			transaction.From, capabilityID, capability.Owner), errors.New("unauthorized access"))
		return false
	}

	// Log successful validation with more details
	agentlog.LogInfo(fmt.Sprintf("Successfully validated capability update for ID %s by owner %s (version: %s)",
		capabilityID, transaction.From, capability.Version))

	return true
}

// Helper functions for descriptor validation

func (mcp *MCPProcessor) validateResourceDescriptor(descriptor types.ResourceDescriptor) bool {
	// Validate base descriptor fields
	if !mcp.validateBaseDescriptor(descriptor.BaseDescriptor) {
		return false
	}

	// Validate resource-specific fields
	if descriptor.BaseDescriptor.CapabilityType != types.CapabilityTypeResource {
		agentlog.LogError("Invalid CapabilityType for ResourceDescriptor", errors.New("invalid descriptor type"))
		return false
	}

	if descriptor.ResourceType == "" {
		agentlog.LogError("Missing ResourceType", errors.New("invalid resource descriptor"))
		return false
	}

	return true
}

func (mcp *MCPProcessor) validateToolDescriptor(descriptor types.ToolDescriptor) bool {
	// Validate base descriptor fields
	if !mcp.validateBaseDescriptor(descriptor.BaseDescriptor) {
		return false
	}

	// Validate tool-specific fields
	if descriptor.BaseDescriptor.CapabilityType != types.CapabilityTypeTool {
		agentlog.LogError("Invalid CapabilityType for ToolDescriptor", errors.New("invalid descriptor type"))
		return false
	}

	if descriptor.InputSchemaJSON == "" {
		agentlog.LogError("Missing InputSchemaJSON", errors.New("invalid tool descriptor"))
		return false
	}

	if descriptor.OutputSchemaJSON == "" {
		agentlog.LogError("Missing OutputSchemaJSON", errors.New("invalid tool descriptor"))
		return false
	}

	return true
}

func (mcp *MCPProcessor) validatePromptDescriptor(descriptor types.PromptDescriptor) bool {
	// Validate base descriptor fields
	if !mcp.validateBaseDescriptor(descriptor.BaseDescriptor) {
		return false
	}

	// Validate prompt-specific fields
	if descriptor.BaseDescriptor.CapabilityType != types.CapabilityTypePrompt {
		agentlog.LogError("Invalid CapabilityType for PromptDescriptor", errors.New("invalid descriptor type"))
		return false
	}

	if descriptor.Template == "" {
		agentlog.LogError("Missing Template", errors.New("invalid prompt descriptor"))
		return false
	}

	if descriptor.ParametersSchemaJSON == "" {
		agentlog.LogError("Missing ParametersSchemaJSON", errors.New("invalid prompt descriptor"))
		return false
	}

	return true
}

func (mcp *MCPProcessor) validateMemoryServiceDescriptor(descriptor types.MemoryServiceDescriptor) bool {
	// Validate base descriptor fields
	if !mcp.validateBaseDescriptor(descriptor.BaseDescriptor) {
		return false
	}

	// Validate memory service-specific fields
	if descriptor.BaseDescriptor.CapabilityType != types.CapabilityTypeMemoryService {
		agentlog.LogError("Invalid CapabilityType for MemoryServiceDescriptor", errors.New("invalid descriptor type"))
		return false
	}

	return true
}

func (mcp *MCPProcessor) validateBaseDescriptor(descriptor types.BaseDescriptor) bool {
	if descriptor.ID == "" {
		agentlog.LogError("Missing ID", errors.New("invalid descriptor"))
		return false
	}

	if descriptor.Name == "" {
		agentlog.LogError("Missing Name", errors.New("invalid descriptor"))
		return false
	}

	if descriptor.Owner == "" {
		agentlog.LogError("Missing Owner", errors.New("invalid descriptor"))
		return false
	}

	if descriptor.Version == "" {
		agentlog.LogError("Missing Version", errors.New("invalid descriptor"))
		return false
	}

	// Check if the ID is already in use
	_, err := mcp.getCapabilityByID(descriptor.ID)
	if err == nil {
		agentlog.LogError("Capability ID already exists", errors.New("duplicate capability ID"))
		return false
	}

	return true
}

// GetCapabilityDescriptor retrieves a full capability descriptor by its ID (public method)
func (mcp *MCPProcessor) GetCapabilityDescriptor(id string) (interface{}, error) {
	// Check if we have a database connection
	if mcp.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// Get the capability from the database
	capability, err := mcp.db.GetCapabilityByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get capability %s: %w", id, err)
	}

	return capability, nil
}

// getCapabilityByID retrieves a capability descriptor by its ID with retry logic
func (mcp *MCPProcessor) getCapabilityByID(id string) (types.BaseDescriptor, error) {
	// Check if we have a database connection
	if mcp.db == nil {
		return types.BaseDescriptor{}, fmt.Errorf("database not initialized")
	}

	var retryErr error
	var capability interface{}

	// Retry with exponential backoff
	for i := 0; i < 5; i++ {
		// Get the capability from the database
		capability, retryErr = mcp.db.GetCapabilityByID(id)
		if retryErr == nil {
			break
		}

		// Exponential backoff: 50ms, 100ms, 200ms, 400ms, 800ms
		time.Sleep(time.Duration(50*(1<<i)) * time.Millisecond)
	}

	if retryErr != nil {
		return types.BaseDescriptor{}, fmt.Errorf("failed to get capability after retries: %w", retryErr)
	}

	// Convert proto to capability interface
	capProto, ok := capability.(*pb.CapabilityDescriptorContainerProto)
	if !ok {
		return types.BaseDescriptor{}, fmt.Errorf("invalid capability type %T", capability)
	}
	capabilityInterface, err := ConvertProtoToCapability(capProto)
	if err != nil {
		return types.BaseDescriptor{}, fmt.Errorf("failed to convert capability: %w", err)
	}

	// Extract the BaseDescriptor from the capability
	switch c := capabilityInterface.(type) {
	case types.ResourceDescriptor:
		return c.BaseDescriptor, nil
	case types.ToolDescriptor:
		return c.BaseDescriptor, nil
	case types.PromptDescriptor:
		return c.BaseDescriptor, nil
	case types.MemoryServiceDescriptor:
		return c.BaseDescriptor, nil
	default:
		return types.BaseDescriptor{}, fmt.Errorf("unsupported capability type")
	}
}

// ProcessMCPRegisterCapability handles the logic for an MCPRegisterCapability transaction.
// This method is typically called by the BlockchainServer when a transaction is received.
func (mcp *MCPProcessor) ProcessMCPRegisterCapability(transaction *Transaction) error {
	agentlog.LogInfo(fmt.Sprintf("[ProcessMCPRegisterCapability] Validating transaction: %s", transaction.TransactionHash))
	agentlog.LogInfo(fmt.Sprintf("[ProcessMCPRegisterCapability] Capability registration data: %s", string(transaction.Data)))

	// 1. Unmarshal tx.Data (JSON) into MCPRegisterCapabilityData
	var mcpRegData types.MCPRegisterCapabilityData
	if err := json.Unmarshal(transaction.Data, &mcpRegData); err != nil {
		return fmt.Errorf("failed to unmarshal MCPRegisterCapabilityData from transaction %s: %w", transaction.TransactionHash, err)
	}

	// Handle both capabilityDescriptor and descriptor fields for backward compatibility
	var descriptor interface{}
	if mcpRegData.CapabilityDescriptor != nil {
		descriptor = mcpRegData.CapabilityDescriptor
	} else if mcpRegData.Descriptor != nil {
		descriptor = mcpRegData.Descriptor
	} else {
		return fmt.Errorf("both capabilityDescriptor and descriptor are nil in transaction data for tx %s", transaction.TransactionHash)
	}

	// 2. Convert the capability descriptor to a concrete type
	descriptorMap, ok := descriptor.(map[string]interface{})
	if !ok {
		return fmt.Errorf("capabilityDescriptor is not a map for tx %s, type: %T", transaction.TransactionHash, mcpRegData.Descriptor)
	}

	descriptorBytes, err := json.Marshal(descriptorMap)
	if err != nil {
		return fmt.Errorf("failed to re-marshal descriptorMap for tx %s: %w", transaction.TransactionHash, err)
	}

	var tempBaseDesc types.BaseDescriptor
	if err := json.Unmarshal(descriptorBytes, &tempBaseDesc); err != nil {
		return fmt.Errorf("failed to unmarshal into tempBaseDesc for tx %s: %w. Bytes: %s", transaction.TransactionHash, err, string(descriptorBytes))
	}

	var concreteCapability interface{}
	switch tempBaseDesc.CapabilityType {
	case types.CapabilityTypeResource:
		var rd types.ResourceDescriptor
		if err := json.Unmarshal(descriptorBytes, &rd); err != nil {
			return fmt.Errorf("failed to unmarshal into ResourceDescriptor for tx %s: %w", transaction.TransactionHash, err)
		}
		concreteCapability = rd
	case types.CapabilityTypeTool:
		var td types.ToolDescriptor
		if err := json.Unmarshal(descriptorBytes, &td); err != nil {
			return fmt.Errorf("failed to unmarshal into ToolDescriptor for tx %s: %w", transaction.TransactionHash, err)
		}
		concreteCapability = td
	case types.CapabilityTypePrompt:
		var pd types.PromptDescriptor
		if err := json.Unmarshal(descriptorBytes, &pd); err != nil {
			return fmt.Errorf("failed to unmarshal into PromptDescriptor for tx %s: %w", transaction.TransactionHash, err)
		}
		concreteCapability = pd
	case types.CapabilityTypeMemoryService:
		var msd types.MemoryServiceDescriptor
		if err := json.Unmarshal(descriptorBytes, &msd); err != nil {
			return fmt.Errorf("failed to unmarshal into MemoryServiceDescriptor for tx %s: %w", transaction.TransactionHash, err)
		}
		concreteCapability = msd
	default:
		return fmt.Errorf("unsupported capability type '%s' in ProcessMCPRegisterCapability for tx %s", tempBaseDesc.CapabilityType, transaction.TransactionHash)
	}

	baseDesc, err := getBaseDescriptorFromInterface(concreteCapability)
	if err != nil {
		// This should not happen if the above switch was successful
		return fmt.Errorf("failed to extract base descriptor from transaction data for tx %s: %w", transaction.TransactionHash, err)
	}

	// Perform any additional validation specific to this processing step if needed.
	// For example, check if baseDesc.Owner matches transaction.From.
	if baseDesc.Owner != transaction.From {
		return fmt.Errorf("descriptor owner (%s) does not match transaction signer (%s) for tx %s", baseDesc.Owner, transaction.From, transaction.TransactionHash)
	}

	// 3. Save the capability to the database
	if mcp.db == nil {
		return fmt.Errorf("database not initialized for tx %s", transaction.TransactionHash)
	}

	// Save with verification
	if err := mcp.db.SaveCapability(concreteCapability); err != nil {
		return fmt.Errorf("failed to save capability from tx %s: %w", transaction.TransactionHash, err)
	}

	// Verify the capability was saved
	var verifyErr error
	for i := 0; i < 3; i++ {
		_, verifyErr = mcp.getCapabilityByID(baseDesc.ID)
		if verifyErr == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if verifyErr != nil {
		return fmt.Errorf("failed to verify saved capability %s exists after saving: %w", baseDesc.ID, verifyErr)
	}

	agentlog.LogInfo(fmt.Sprintf("[ProcessMCPRegisterCapability] Successfully saved and verified capability %s in DB", baseDesc.ID))

	// Remove Flush call since LevelDB handles this automatically
	return nil
}

// ProcessMCPInvokeCapability handles the logic for an MCPInvokeCapability transaction.
func (mcp *MCPProcessor) ProcessMCPInvokeCapability(transaction *Transaction, contextRecord types.ContextRecord) error {
	txHash := transaction.TransactionHash
	agentlog.LogInfo(fmt.Sprintf("[ProcessMCPInvokeCapability] Processing transaction: %s", txHash))

	// 1. Validate transaction (already done by validateCapabilityInvocation before adding to pool)
	// 2. Extract capability ID from context record
	capabilityID := contextRecord.CapabilityID
	agentlog.LogInfo(fmt.Sprintf("[ProcessMCPInvokeCapability] Capability ID: %s", capabilityID))

	// 3. Validate capability exists (already done by validateCapabilityInvocation before adding to pool)
	// 4. Validate initiator has rights/funds (partially done by simulatedBalanceCheck and validateTransactionInBlockContext)

	// Ensure context record ID matches transaction hash and uses ctx: prefix
	if contextRecord.ID != "ctx:"+txHash {
		contextRecord.ID = "ctx:" + txHash
	}

	// 5. Store ContextRecord
	if err := mcp.db.SaveContextRecord(contextRecord); err != nil { // SaveContextRecord expects ContextRecord
		return fmt.Errorf("failed to save context record for invoke tx %s: %w", txHash, err)
	}

	// Verify the record was actually saved with increased retry delay
	var savedRecord *pb.ContextRecordProto
	var lastErr error
	for i := 0; i < 5; i++ {
		savedRecord, lastErr = mcp.db.GetContextRecord(contextRecord.ID)
		if lastErr == nil {
			break
		}
		time.Sleep(100 * time.Millisecond) // Increased from 10ms to 100ms
	}
	if lastErr != nil {
		agentlog.LogError(fmt.Sprintf("Failed to verify context record storage for tx %s after retries. Key: ctx:%s", txHash, contextRecord.ID), lastErr)
		return fmt.Errorf("failed to verify context record storage for tx %s (key: ctx:%s): %w", txHash, contextRecord.ID, lastErr)
	}
	agentlog.LogInfo(fmt.Sprintf("Successfully verified ContextRecord %s for tx %s", savedRecord.Id, txHash))

	agentlog.LogInfo(fmt.Sprintf("[ProcessMCPInvokeCapability] Successfully saved and verified ContextRecord %s for tx %s", contextRecord.ID, txHash))

	return nil
}

// ApplyMCPTransactionEffects applies the effects of an MCP transaction to the state.
// This is called during block processing after a transaction has been validated and included in a block.
func (mcp *MCPProcessor) ApplyMCPTransactionEffects(tx *Transaction, accounts map[string]*big.Int) (*pb.ContextRecordProto, error) {
	agentlog.LogInfo(fmt.Sprintf("[ApplyMCPTransactionEffects] Applying effects for tx: %s", tx.TransactionHash))

	switch tx.Type {
	case TransactionTypeMCPRegisterCapability:
		agentlog.LogInfo(fmt.Sprintf("[ApplyMCPTransactionEffects] Processing MCPRegisterCapability transaction: %s", tx.TransactionHash))
		agentlog.LogInfo(fmt.Sprintf("[ApplyMCPTransactionEffects] Transaction data: %s", string(tx.Data)))
		var mcpRegData types.MCPRegisterCapabilityData
		if err := json.Unmarshal(tx.Data, &mcpRegData); err != nil {
			return nil, fmt.Errorf("ApplyMCPTransactionEffects: failed to unmarshal MCPRegisterCapabilityData for tx %s: %w", tx.TransactionHash, err)
		}

		if mcp.db == nil {
			return nil, fmt.Errorf("ApplyMCPTransactionEffects: database not available for register tx %s", tx.TransactionHash)
		}

		// Handle both capabilityDescriptor and descriptor fields for backward compatibility
		var descriptor interface{}
		if mcpRegData.CapabilityDescriptor != nil {
			descriptor = mcpRegData.CapabilityDescriptor
		} else if mcpRegData.Descriptor != nil {
			descriptor = mcpRegData.Descriptor
		} else {
			errMsg := "ApplyMCPTransactionEffects: both capabilityDescriptor and descriptor are nil in tx " +
				tx.TransactionHash + ". Raw tx.Data: " + string(tx.Data)
			err := errors.New(errMsg)
			agentlog.LogError(errMsg, err) // Keep LogError for consistency
			return nil, err
		}

		// Additional validation for descriptor type
		if _, ok := descriptor.(map[string]interface{}); !ok {
			errMsg := "ApplyMCPTransactionEffects: capabilityDescriptor is not a map for tx " +
				tx.TransactionHash + ", type: " + fmt.Sprintf("%T", descriptor) + ". Raw descriptor: " + fmt.Sprintf("%+v", descriptor)
			err := errors.New(errMsg)
			agentlog.LogError(errMsg, err) // Keep LogError
			return nil, err
		}

		descriptorMap, ok := descriptor.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("ApplyMCPTransactionEffects: capabilityDescriptor is not a map for tx %s, type: %T", tx.TransactionHash, mcpRegData.Descriptor)
		}

		descriptorBytes, err := json.Marshal(descriptorMap)
		if err != nil {
			return nil, fmt.Errorf("ApplyMCPTransactionEffects: failed to re-marshal descriptorMap for tx %s: %w", tx.TransactionHash, err)
		}

		var tempBaseDesc types.BaseDescriptor
		if err := json.Unmarshal(descriptorBytes, &tempBaseDesc); err != nil {
			return nil, fmt.Errorf("ApplyMCPTransactionEffects: failed to unmarshal into tempBaseDesc for tx %s: %w. Bytes: %s", tx.TransactionHash, err, string(descriptorBytes))
		}

		var concreteDescriptorToSave interface{}
		switch tempBaseDesc.CapabilityType {
		case types.CapabilityTypeResource:
			var rd types.ResourceDescriptor
			if err := json.Unmarshal(descriptorBytes, &rd); err != nil {
				return nil, fmt.Errorf("ApplyMCPTransactionEffects: failed to unmarshal into ResourceDescriptor for save: %w", err)
			}
			concreteDescriptorToSave = rd
		case types.CapabilityTypeTool:
			var td types.ToolDescriptor
			if err := json.Unmarshal(descriptorBytes, &td); err != nil {
				return nil, fmt.Errorf("ApplyMCPTransactionEffects: failed to unmarshal into ToolDescriptor for save: %w", err)
			}
			concreteDescriptorToSave = td
		case types.CapabilityTypePrompt:
			var pd types.PromptDescriptor
			if err := json.Unmarshal(descriptorBytes, &pd); err != nil {
				return nil, fmt.Errorf("ApplyMCPTransactionEffects: failed to unmarshal into PromptDescriptor for save: %w", err)
			}
			concreteDescriptorToSave = pd
		case types.CapabilityTypeMemoryService:
			var msd types.MemoryServiceDescriptor
			if err := json.Unmarshal(descriptorBytes, &msd); err != nil {
				return nil, fmt.Errorf("ApplyMCPTransactionEffects: failed to unmarshal into MemoryServiceDescriptor for save: %w", err)
			}
			concreteDescriptorToSave = msd
		default:
			return nil, fmt.Errorf("ApplyMCPTransactionEffects: unsupported capability type '%s' for saving: %s", tempBaseDesc.CapabilityType, tx.TransactionHash)
		}

		capID := getCapabilityIDFromDescriptor(concreteDescriptorToSave)
		log.Printf("[DEBUG] Attempting to save capability %s from tx %s", capID, tx.TransactionHash)
		log.Printf("[DEBUG] Capability details: %+v", concreteDescriptorToSave)

		// Check database connection status
		if mcp.db == nil {
			log.Printf("[CRITICAL] Database connection is nil when trying to save capability %s", capID)
			return nil, fmt.Errorf("ApplyMCPTransactionEffects: database connection is nil for tx %s", tx.TransactionHash)
		}
		if mcp.db.Client == nil {
			log.Printf("[CRITICAL] LevelDB client is nil when trying to save capability %s", capID)
			return nil, fmt.Errorf("ApplyMCPTransactionEffects: LevelDB client is nil for tx %s", tx.TransactionHash)
		}

		// Enhanced capability saving with verification
		log.Printf("[DEBUG] Saving capability %s with descriptor: %+v", capID, concreteDescriptorToSave)
		startTime := time.Now()
		if err := mcp.db.SaveCapability(concreteDescriptorToSave); err != nil {
			log.Printf("[ERROR] Failed to save capability %s after %v: %v", capID, time.Since(startTime), err)
			log.Printf("[ERROR] Capability that failed to save: %+v", concreteDescriptorToSave)

			// Check database status
			dbStats, _ := mcp.db.Client.GetProperty("leveldb.stats")
			log.Printf("[DEBUG] LevelDB stats: %s", dbStats)

			return nil, fmt.Errorf("ApplyMCPTransactionEffects: failed to save capability from tx %s: %w", tx.TransactionHash, err)
		}
		log.Printf("[INFO] Successfully saved capability %s to DB in %v", capID, time.Since(startTime))

		// Verify the capability was saved with content matching
		var savedCapability interface{}
		var lastErr2 error
		for i := 0; i < 10; i++ { // Increased retry attempts
			startGetTime := time.Now()
			savedCapability, lastErr2 = mcp.db.GetCapabilityByID(capID)
			if lastErr2 == nil {
				// Convert concreteDescriptorToSave (Go struct) to its proto representation for comparison
				expectedCapabilityProto, err := ConvertToCapabilityDescriptorContainerProto(concreteDescriptorToSave)
				if err != nil {
					log.Printf("[ERROR] Failed to convert expected capability %s to proto for verification: %v", capID, err)
					lastErr2 = fmt.Errorf("failed to convert expected capability to proto: %w", err)
					continue
				}

				// Compare the protobuf messages directly
				savedProto, ok := savedCapability.(*pb.CapabilityDescriptorContainerProto)
				if !ok {
					lastErr2 = fmt.Errorf("saved capability is not a proto message")
					continue
				}
				if !proto.Equal(expectedCapabilityProto, savedProto) {
					lastErr2 = fmt.Errorf("saved capability proto does not match expected proto")
					// For debugging, log the JSON of both if they don't match
					savedJSON, _ := json.Marshal(savedProto)
					expectedJSON, _ := json.Marshal(expectedCapabilityProto)
					log.Printf("[ERROR] Capability protobuf mismatch for %s (attempt %d)\nSaved: %s\nExpected: %s",
						capID, i+1, string(savedJSON), string(expectedJSON))
					continue
				}
				log.Printf("[INFO] Successfully verified capability %s exists in DB with matching content (attempt %d, took %v)",
					capID, i+1, time.Since(startGetTime))
				break
			}
			log.Printf("[WARN] Failed to get capability %s (attempt %d): %v", capID, i+1, lastErr2)
			time.Sleep(200 * time.Millisecond) // Increased delay
		}
		if lastErr2 != nil {
			log.Printf("[ERROR] Failed to verify saved capability %s after retries: %v", capID, lastErr2)

			// Check if the key exists at all
			exists, err := mcp.db.Client.Has([]byte("mcp:capability:"+capID), nil)
			if err != nil {
				log.Printf("[ERROR] Failed to check key existence for capability %s: %v", capID, err)
			} else if !exists {
				log.Printf("[CRITICAL] Key mcp:capability:%s does not exist in database", capID)
			} else {
				log.Printf("[DEBUG] Key mcp:capability:%s exists but GetCapabilityByID failed", capID)
			}

			return nil, fmt.Errorf("ApplyMCPTransactionEffects: failed to verify saved capability %s after retries: %w", capID, lastErr2)
		}

		// Force LevelDB sync with compaction
		log.Printf("[DEBUG] Starting LevelDB compaction for tx %s", tx.TransactionHash)
		for i := 0; i < 3; i++ { // Multiple compaction attempts
			if err := mcp.db.Client.CompactRange(util.Range{}); err != nil {
				log.Printf("[ERROR] Failed to compact LevelDB (attempt %d): %v", i+1, err)
				if i == 2 { // Final attempt failed
					agentlog.LogError(fmt.Sprintf("ApplyMCPTransactionEffects: Failed to compact LevelDB for registration tx %s", tx.TransactionHash), err)
					return nil, fmt.Errorf("ApplyMCPTransactionEffects: failed to compact LevelDB for registration tx %s: %w", tx.TransactionHash, err)
				}
				time.Sleep(500 * time.Millisecond)
				continue
			}
			break
		}
		log.Printf("[INFO] Successfully compacted LevelDB for tx %s", tx.TransactionHash)
		agentlog.LogInfo(fmt.Sprintf("ApplyMCPTransactionEffects: Successfully synced and compacted LevelDB for registration tx %s", tx.TransactionHash))

		// Additional verification with retries
		var verifyErr error
		for i := 0; i < 5; i++ {
			_, verifyErr = mcp.db.GetCapabilityByID(capID)
			if verifyErr == nil {
				break
			}
			time.Sleep(200 * time.Millisecond)
		} // Keep this verification
		if verifyErr != nil {
			return nil, fmt.Errorf("ApplyMCPTransactionEffects: failed to verify capability %s exists after compaction: %w", capID, verifyErr)
		}

		// Final verification after compaction with content check
		savedCapabilityProtoAfterCompaction, err := mcp.db.GetCapabilityByID(capID)
		if err != nil {
			log.Printf("[ERROR] Capability %s not found after compaction: %v", capID, err)
			return nil, fmt.Errorf("ApplyMCPTransactionEffects: capability %s not found after compaction: %w", capID, err)
		}

		expectedCapabilityProtoAfterCompaction, err := ConvertToCapabilityDescriptorContainerProto(concreteDescriptorToSave)
		if err != nil {
			log.Printf("[ERROR] Failed to convert expected capability %s to proto for final verification: %v", capID, err)
			return nil, fmt.Errorf("ApplyMCPTransactionEffects: failed to convert expected capability to proto for final verification: %w", err)
		}

		if !proto.Equal(expectedCapabilityProtoAfterCompaction, savedCapabilityProtoAfterCompaction) {
			savedJSON, _ := json.Marshal(savedCapabilityProtoAfterCompaction)
			expectedJSON, _ := json.Marshal(expectedCapabilityProtoAfterCompaction)
			log.Printf("[ERROR] Capability content mismatch after compaction for %s\nSaved: %s\nExpected: %s",
				capID, string(savedJSON), string(expectedJSON))
			return nil, fmt.Errorf("ApplyMCPTransactionEffects: capability %s content mismatch after compaction", capID)
		}
		log.Printf("[INFO] Final verification passed for capability %s with matching content", capID)

		// Optionally, create and save a ContextRecord for the registration event
		baseDesc, _ := getBaseDescriptorFromInterface(concreteDescriptorToSave) // Safe to ignore error if already validated
		contextRecord := types.NewContextRecord(tx.TransactionHash, baseDesc.ID, types.InteractionTypeCapabilityRegistration, tx.From, "", "", nil)
		// Convert to proto before returning
		contextRecordProto, errProto := ConvertContextRecordToProto(*contextRecord)
		if errProto != nil {
			agentlog.LogWarning(fmt.Sprintf("Warning: Failed to convert context record to proto for capability registration %s: %v", baseDesc.ID, errProto))
			return nil, nil // Return nil proto, but not a fatal error for the overall transaction
		}
		if err := mcp.db.SaveContextRecord(*contextRecord); err != nil { // Save the types.ContextRecord
			agentlog.LogWarning(fmt.Sprintf("Warning: Failed to save context record for capability registration %s: %v", baseDesc.ID, err))
			return nil, nil // Return nil proto, but not a fatal error for the overall transaction
		}
		return contextRecordProto, nil // Return the saved proto

	case TransactionTypeMCPInvokeCapability:
		var mcpInvokeData types.MCPInvokeCapabilityData
		if err := json.Unmarshal(tx.Data, &mcpInvokeData); err != nil {
			agentlog.LogError(fmt.Sprintf("ApplyMCPTransactionEffects: Failed to unmarshal MCPInvokeCapabilityData for tx %s", tx.TransactionHash), err)
			return nil, fmt.Errorf("ApplyMCPTransactionEffects: failed to unmarshal MCPInvokeCapabilityData for tx %s: %w", tx.TransactionHash, err)
		}

		if mcp.db == nil {
			err := fmt.Errorf("database not available")
			agentlog.LogError("ApplyMCPTransactionEffects: Database connection not initialized", err)
			return nil, fmt.Errorf("ApplyMCPTransactionEffects: database not available for invoke tx %s: %w", tx.TransactionHash, err)
		}

		// Validate required fields in context record
		if mcpInvokeData.ContextRecord.CapabilityID == "" {
			err := fmt.Errorf("empty capability ID")
			agentlog.LogError(fmt.Sprintf("ApplyMCPTransactionEffects: Empty CapabilityID in ContextRecord for tx %s", tx.TransactionHash), err)
			return nil, fmt.Errorf("ApplyMCPTransactionEffects: empty capability ID in context record for tx %s: %w", tx.TransactionHash, err)
		}
		if mcpInvokeData.ContextRecord.Initiator == "" {
			err := fmt.Errorf("empty initiator")
			agentlog.LogError(fmt.Sprintf("ApplyMCPTransactionEffects: Empty Initiator in ContextRecord for tx %s", tx.TransactionHash), err)
			return nil, fmt.Errorf("ApplyMCPTransactionEffects: empty initiator in context record for tx %s: %w", tx.TransactionHash, err)
		}
		if mcpInvokeData.ContextRecord.InteractionType == "" {
			err := fmt.Errorf("empty interaction type")
			agentlog.LogError(fmt.Sprintf("ApplyMCPTransactionEffects: Empty InteractionType in ContextRecord for tx %s", tx.TransactionHash), err)
			return nil, fmt.Errorf("ApplyMCPTransactionEffects: empty interaction type in context record for tx %s: %w", tx.TransactionHash, err)
		}

		// Create a fresh ContextRecord with all fields explicitly set and ctx: prefix
		contextRecord := types.ContextRecord{
			ID:              tx.TransactionHash,
			CapabilityID:    mcpInvokeData.ContextRecord.CapabilityID,
			InteractionType: mcpInvokeData.ContextRecord.InteractionType,
			Initiator:       mcpInvokeData.ContextRecord.Initiator,
			InputHash:       mcpInvokeData.ContextRecord.InputHash,
			OutputHash:      mcpInvokeData.ContextRecord.OutputHash,
			Timestamp:       mcpInvokeData.ContextRecord.Timestamp, // Use original timestamp
			Signature:       mcpInvokeData.ContextRecord.Signature,
			Details:         mcpInvokeData.ContextRecord.Details,
		}

		agentlog.LogInfo(fmt.Sprintf("ApplyMCPTransactionEffects: Saving ContextRecord for tx %s - CapabilityId: %s, Initiator: %s, InteractionType: %s, Details: %v",
			tx.TransactionHash,
			contextRecord.CapabilityID,
			contextRecord.Initiator,
			contextRecord.InteractionType,
			contextRecord.Details))

		// Save with explicit sync verification
		startSaveTime := time.Now()
		if err := mcp.db.SaveContextRecord(contextRecord); err != nil {
			err := fmt.Errorf("ApplyMCPTransactionEffects: failed to save context record for tx %s: %w", tx.TransactionHash, err)
			agentlog.LogError(fmt.Sprintf("Failed to save context record for tx %s. ContextRecord: %+v", tx.TransactionHash, contextRecord), err)
			return nil, err
		}
		log.Printf("[INFO] Successfully saved context record for tx %s to DB in %v", tx.TransactionHash, time.Since(startSaveTime))

		// Explicitly verify it's in the DB immediately after saving for debugging this specific issue
		if _, err := mcp.db.GetContextRecord(tx.TransactionHash); err != nil {
			log.Printf("[ERROR] ApplyMCPTransactionEffects: CRITICAL - FAILED to retrieve context record %s immediately after save: %v", tx.TransactionHash, err)
			return nil, fmt.Errorf("failed to verify context record save for %s: %w", tx.TransactionHash, err)
		}

		// Force LevelDB sync
		if err := mcp.db.Client.CompactRange(util.Range{}); err != nil {
			agentlog.LogError(fmt.Sprintf("ApplyMCPTransactionEffects: Failed to compact LevelDB for tx %s", tx.TransactionHash), err)
			return nil, fmt.Errorf("ApplyMCPTransactionEffects: failed to compact LevelDB for tx %s: %w", tx.TransactionHash, err)
		}
		agentlog.LogInfo(fmt.Sprintf("ApplyMCPTransactionEffects: Successfully synced and compacted LevelDB for tx %s", tx.TransactionHash))

		agentlog.LogInfo(fmt.Sprintf("ApplyMCPTransactionEffects: Successfully called SaveContextRecord for tx %s. Verifying after sync...", tx.TransactionHash))

		// Verify the record was saved with retry logic
		var savedRecord *pb.ContextRecordProto
		var lastErr error
		for i := 0; i < 3; i++ {
			savedRecord, lastErr = mcp.db.GetContextRecord(tx.TransactionHash)
			if lastErr == nil {
				break
			}
			time.Sleep(100 * time.Millisecond) // Increased from 10ms to 100ms
		}
		if lastErr != nil {
			agentlog.LogError(fmt.Sprintf("ApplyMCPTransactionEffects: Failed to verify saved context record for tx %s: %v", tx.TransactionHash, lastErr), lastErr)
			return nil, fmt.Errorf("ApplyMCPTransactionEffects: failed to verify saved context record for tx %s (key: %s): %w", tx.TransactionHash, tx.TransactionHash, lastErr)
		}

		agentlog.LogInfo(fmt.Sprintf("ApplyMCPTransactionEffects: Successfully verified ContextRecord for tx %s - CapabilityId: %s, Initiator: %s, InteractionType: %s",
			tx.TransactionHash,
			savedRecord.CapabilityId,
			savedRecord.Initiator,
			savedRecord.InteractionType))

		// Implement GasFeeNRN transfer
		capDescInterface, errDbGet := mcp.db.GetCapabilityByID(mcpInvokeData.ContextRecord.CapabilityID)
		if errDbGet != nil {
			log.Printf("[ERROR] ApplyMCPTransactionEffects: Could not retrieve capability %s for fee transfer: %v", mcpInvokeData.ContextRecord.CapabilityID, errDbGet)
			return nil, fmt.Errorf("could not retrieve capability %s for fee transfer: %w", mcpInvokeData.ContextRecord.CapabilityID, errDbGet)
		}
		baseDesc, errConv := getBaseDescriptorFromInterface(capDescInterface)
		if errConv != nil {
			log.Printf("[ERROR] ApplyMCPTransactionEffects: Could not get base descriptor for capability %s: %v", mcpInvokeData.ContextRecord.CapabilityID, errConv)
			return nil, fmt.Errorf("could not get base descriptor for capability %s: %w", mcpInvokeData.ContextRecord.CapabilityID, errConv)
		}

		if baseDesc.GasFeeNRN > 0 {
			capabilityGasFee := new(big.Int).SetUint64(baseDesc.GasFeeNRN)
			// Ensure both accounts exist in the map
			if _, ok := accounts[tx.From]; !ok {
				accounts[tx.From] = big.NewInt(0)
				log.Printf("[WARN] Account for %s not found in accounts map, initializing with 0", tx.From)
			}
			if _, ok := accounts[baseDesc.Owner]; !ok {
				accounts[baseDesc.Owner] = big.NewInt(0)
				log.Printf("[INFO] Account for capability owner %s not found in accounts map, initializing with 0", baseDesc.Owner)
			}

			if accounts[tx.From].Cmp(capabilityGasFee) < 0 {
				return nil, fmt.Errorf("insufficient funds for capability gas fee for tx %s by %s", tx.TransactionHash, tx.From)
			}

			// Always deduct capability gas fee from initiator
			accounts[tx.From] = new(big.Int).Sub(accounts[tx.From], capabilityGasFee)
			log.Printf("[INFO] Debited %d NRN from initiator %s for capability gas fee (tx: %s)",
				baseDesc.GasFeeNRN, tx.From, tx.TransactionHash)

			// Credit to capability owner ONLY if they are different from the initiator
			if tx.From != baseDesc.Owner {
				accounts[baseDesc.Owner] = new(big.Int).Add(accounts[baseDesc.Owner], capabilityGasFee)
				log.Printf("[INFO] Credited %d NRN to owner %s for capability gas fee (tx: %s)",
					baseDesc.GasFeeNRN, baseDesc.Owner, tx.TransactionHash)
			} else {
				log.Printf("[INFO] Initiator %s is also owner. Capability gas fee of %d NRN is effectively burned/collected by network (tx: %s).",
					tx.From, baseDesc.GasFeeNRN, tx.TransactionHash)
			}
		}
		return nil, nil // No specific context record proto to return for invoke, it's part of tx.Data
	case TransactionTypeMCPUpdateCapability:
		// First try to parse as generic map to determine format
		var updateDataMap map[string]interface{}
		if err := json.Unmarshal(tx.Data, &updateDataMap); err != nil {
			return nil, fmt.Errorf("ApplyMCPTransactionEffects: failed to unmarshal update data for tx %s: %w", tx.TransactionHash, err)
		}

		if mcp.db == nil {
			return nil, fmt.Errorf("ApplyMCPTransactionEffects: database not available for update tx %s", tx.TransactionHash)
		}

		var capabilityID string
		var concreteDescriptor interface{}

		// Check if this is the test format with capabilityDescriptor
		if capDescriptor, hasCapDesc := updateDataMap["capabilityDescriptor"]; hasCapDesc {
			// Test format: { "capabilityID": "...", "capabilityDescriptor": { ... } }
			capabilityID = updateDataMap["capabilityID"].(string)

			// Convert the full capability descriptor
			descriptorBytes, err := json.Marshal(capDescriptor)
			if err != nil {
				return nil, fmt.Errorf("ApplyMCPTransactionEffects: failed to marshal capability descriptor: %w", err)
			}

			// Determine the type from the descriptor itself
			var tempBaseDesc types.BaseDescriptor
			if err := json.Unmarshal(descriptorBytes, &tempBaseDesc); err != nil {
				return nil, fmt.Errorf("ApplyMCPTransactionEffects: failed to unmarshal base descriptor: %w", err)
			}

			switch tempBaseDesc.CapabilityType {
			case types.CapabilityTypeResource:
				var resourceDesc types.ResourceDescriptor
				if err := json.Unmarshal(descriptorBytes, &resourceDesc); err != nil {
					return nil, fmt.Errorf("ApplyMCPTransactionEffects: failed to unmarshal resource descriptor: %w", err)
				}
				concreteDescriptor = resourceDesc
			case types.CapabilityTypeTool:
				var toolDesc types.ToolDescriptor
				if err := json.Unmarshal(descriptorBytes, &toolDesc); err != nil {
					return nil, fmt.Errorf("ApplyMCPTransactionEffects: failed to unmarshal tool descriptor: %w", err)
				}
				concreteDescriptor = toolDesc
			case types.CapabilityTypePrompt:
				var promptDesc types.PromptDescriptor
				if err := json.Unmarshal(descriptorBytes, &promptDesc); err != nil {
					return nil, fmt.Errorf("ApplyMCPTransactionEffects: failed to unmarshal prompt descriptor: %w", err)
				}
				concreteDescriptor = promptDesc
			case types.CapabilityTypeMemoryService:
				var memoryDesc types.MemoryServiceDescriptor
				if err := json.Unmarshal(descriptorBytes, &memoryDesc); err != nil {
					return nil, fmt.Errorf("ApplyMCPTransactionEffects: failed to unmarshal memory service descriptor: %w", err)
				}
				concreteDescriptor = memoryDesc
			default:
				return nil, fmt.Errorf("ApplyMCPTransactionEffects: unsupported capability type: %s", tempBaseDesc.CapabilityType)
			}
		} else {
			// Standard format: MCPUpdateCapabilityData
			var mcpUpdateData types.MCPUpdateCapabilityData
			if err := json.Unmarshal(tx.Data, &mcpUpdateData); err != nil {
				return nil, fmt.Errorf("ApplyMCPTransactionEffects: failed to unmarshal MCPUpdateCapabilityData for tx %s: %w", tx.TransactionHash, err)
			}

			capabilityID = mcpUpdateData.CapabilityID

			// Get the existing capability to determine type
			existingCapability, err := mcp.db.GetCapabilityByID(mcpUpdateData.CapabilityID)
			if err != nil {
				return nil, fmt.Errorf("ApplyMCPTransactionEffects: failed to get existing capability %s: %w", mcpUpdateData.CapabilityID, err)
			}

			existingBaseDesc, err := getBaseDescriptorFromInterface(existingCapability)
			if err != nil {
				return nil, fmt.Errorf("ApplyMCPTransactionEffects: failed to get base descriptor from existing capability: %w", err)
			}

			// Convert the update descriptor based on the existing capability type
			descriptorBytes, err := json.Marshal(mcpUpdateData.Descriptor)
			if err != nil {
				return nil, fmt.Errorf("ApplyMCPTransactionEffects: failed to marshal descriptor: %w", err)
			}

			switch existingBaseDesc.CapabilityType {
			case types.CapabilityTypeResource:
				var resourceDesc types.ResourceDescriptor
				if err := json.Unmarshal(descriptorBytes, &resourceDesc); err != nil {
					return nil, fmt.Errorf("ApplyMCPTransactionEffects: failed to unmarshal resource descriptor: %w", err)
				}
				concreteDescriptor = resourceDesc
			case types.CapabilityTypeTool:
				var toolDesc types.ToolDescriptor
				if err := json.Unmarshal(descriptorBytes, &toolDesc); err != nil {
					return nil, fmt.Errorf("ApplyMCPTransactionEffects: failed to unmarshal tool descriptor: %w", err)
				}
				concreteDescriptor = toolDesc
			case types.CapabilityTypePrompt:
				var promptDesc types.PromptDescriptor
				if err := json.Unmarshal(descriptorBytes, &promptDesc); err != nil {
					return nil, fmt.Errorf("ApplyMCPTransactionEffects: failed to unmarshal prompt descriptor: %w", err)
				}
				concreteDescriptor = promptDesc
			case types.CapabilityTypeMemoryService:
				var memoryDesc types.MemoryServiceDescriptor
				if err := json.Unmarshal(descriptorBytes, &memoryDesc); err != nil {
					return nil, fmt.Errorf("ApplyMCPTransactionEffects: failed to unmarshal memory service descriptor: %w", err)
				}
				concreteDescriptor = memoryDesc
			default:
				return nil, fmt.Errorf("ApplyMCPTransactionEffects: unsupported capability type: %s", existingBaseDesc.CapabilityType)
			}
		}

		// Get the existing capability for ownership verification
		existingCapability, err := mcp.db.GetCapabilityByID(capabilityID)
		if err != nil {
			return nil, fmt.Errorf("ApplyMCPTransactionEffects: failed to get existing capability %s: %w", capabilityID, err)
		}

		// Extract the base descriptor for ownership verification
		baseDesc, err := getBaseDescriptorFromInterface(existingCapability)
		if err != nil {
			return nil, fmt.Errorf("ApplyMCPTransactionEffects: failed to extract base descriptor from existing capability: %w", err)
		}

		// Verify ownership
		if baseDesc.Owner != tx.From {
			return nil, fmt.Errorf("ApplyMCPTransactionEffects: sender %s is not the owner of capability %s", tx.From, capabilityID)
		}

		if err := mcp.db.UpdateCapability(capabilityID, concreteDescriptor); err != nil {
			return nil, fmt.Errorf("ApplyMCPTransactionEffects: failed to update capability %s: %w", capabilityID, err)
		}
		// Explicitly verify it's in the DB immediately after saving
		if _, err := mcp.db.GetCapabilityByID(capabilityID); err != nil {
			log.Printf("[ERROR] ApplyMCPTransactionEffects: CRITICAL - FAILED to retrieve updated capability %s immediately after save: %v", capabilityID, err)
			return nil, fmt.Errorf("failed to verify updated capability save for %s: %w", capabilityID, err)
		}

		// Create a context record for the update
		contextRecord := types.NewContextRecord(tx.TransactionHash, capabilityID, types.InteractionTypeCapabilityUpdate, tx.From, "", "", nil)
		startSaveTime := time.Now()
		if err := mcp.db.SaveContextRecord(*contextRecord); err != nil {
			agentlog.LogWarning(fmt.Sprintf("Warning: Failed to save context record for capability update %s: %v", capabilityID, err))
			return nil, nil // Return nil proto, but not a fatal error for the overall transaction
		}
		saveDuration := time.Since(startSaveTime)
		if saveDuration > 100*time.Millisecond {
			agentlog.LogWarning(fmt.Sprintf("Slow context record save for capability %s: took %v", capabilityID, saveDuration))
		}
		// Convert to proto before returning
		contextRecordProto, errProto := ConvertContextRecordToProto(*contextRecord)
		if errProto != nil {
			agentlog.LogWarning(fmt.Sprintf("Warning: Failed to convert context record to proto for capability update %s: %v", capabilityID, errProto))
			return nil, nil
		}
		return contextRecordProto, nil
	default:
		return nil, fmt.Errorf("ApplyMCPTransactionEffects: unsupported transaction type: %s", tx.Type)
	}

}
