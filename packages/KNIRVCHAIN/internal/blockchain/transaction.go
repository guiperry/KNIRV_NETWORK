package blockchain

import (
	"KNIRVCHAIN/internal/errors"
	pb "KNIRVCHAIN/internal/protocol/proto"
	"KNIRVCHAIN/internal/types"
	"KNIRVCHAIN/internal/utils"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/big"
	"os"
	"reflect"
	"strings"
	"time"

	knirvsigning "github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go/signing"
	"google.golang.org/protobuf/proto"
)

type TransactionProtoForHashing struct {
	From      string
	To        string
	Value     uint64
	Data      []byte
	Timestamp int64
	Fee       uint64
	Type      string
}

type Transaction struct {
	From            string       `json:"from"`
	To              string       `json:"to"`
	Value           uint64       `json:"value"`
	Data            []byte       `json:"data"`
	Status          string       `json:"status"`
	Timestamp       int64        `json:"timestamp"`
	TransactionHash string       `json:"transaction_hash"`
	PublicKey       string       `json:"public_key,omitempty"`
	Signature       []byte       `json:"signature"`
	BodyBytes       string       `json:"body_bytes,omitempty"`
	AuthInfoBytes   string       `json:"auth_info_bytes,omitempty"`
	ChainID         string       `json:"chain_id,omitempty"`
	AccountNumber   uint64       `json:"account_number,omitempty"`
	Sequence        uint64       `json:"sequence,omitempty"`
	DeepCopy        *Transaction `json:"deep_copy,omitempty"`
	Fee             uint64       `json:"fee"`                  // NRN token gas fee paid for this transaction
	Type            string       `json:"type,omitempty"`       // Transaction type for MCP transactions
	Verified        bool         `json:"verified"`             // Whether transaction has been verified
	BlockHash       string       `json:"block_hash,omitempty"` // Hash of the block containing this transaction
}

// ToProtoForHashing converts transaction to proto format for hashing
func (tx *Transaction) ToProtoForHashing() (*TransactionProtoForHashing, error) {
	return &TransactionProtoForHashing{
		From: tx.From, To: tx.To, Value: tx.Value, Data: tx.Data,
		Timestamp: tx.Timestamp, Fee: tx.Fee, Type: tx.Type,
	}, nil
}

// GetCanonicalBytesForHashingTransaction returns canonical bytes for transaction hashing
func GetCanonicalBytesForHashingTransaction(tx *TransactionProtoForHashing) ([]byte, error) {
	actionType := tx.Type
	if actionType == "" {
		actionType = "transfer"
	}
	return knirvsigning.MarshalAction(knirvsigning.Action{
		Action: actionType, Sender: tx.From, Recipient: tx.To, Amount: tx.Value,
		Payload: tx.Data, TimestampUnix: tx.Timestamp,
	})
}

func (tx *Transaction) DetermineDataType() string {
	if tx.Type != "" {
		return tx.Type
	}
	if len(tx.Data) > 0 {
		return "data"
	}
	if tx.Value > 0 {
		return "transfer"
	}
	return "unknown"
}

func NewTransaction(from, to string, value uint64, data []byte, optionalTimestamp ...int64) *Transaction {
	transaction := new(Transaction)

	transaction.From = from
	transaction.To = to
	transaction.Value = value
	if len(optionalTimestamp) > 0 && optionalTimestamp[0] != 0 {
		transaction.Timestamp = optionalTimestamp[0]
	} else {
		transaction.Timestamp = time.Now().Unix()
	}
	transaction.Data = data
	transaction.Status = utils.PENDING // for now by default
	transaction.Fee = 0                // Default fee is 0

	// Create unsigned transaction copy for hash calculation
	unsignedTx := &Transaction{
		From:      from,
		To:        to,
		Value:     value,
		Data:      data,
		Timestamp: transaction.Timestamp,
		Fee:       0,
	}

	// Convert to proto, marshal to bytes, then hash
	txProto, err := unsignedTx.ToProtoForHashing()
	if err != nil {
		log.Printf("Error converting unsigned tx to proto for hash: %v", err)
		// Handle error appropriately, maybe panic or return a special error hash
	}
	canonicalBytes, _ := GetCanonicalBytesForHashingTransaction(txProto)
	hash := sha256.Sum256(canonicalBytes)
	transaction.TransactionHash = utils.HEX_PREFIX + fmt.Sprintf("%x", hash[:])

	return transaction
}

// NewLLMRootingTransaction creates a new LLM rooting transaction
func NewLLMRootingTransaction(from string, llmData LLMRootingData, fee uint64, optionalTimestamp ...int64) (*Transaction, error) {
	// Generate CMU
	llmData.CMU = GenerateCMU(llmData, "mainnet")

	// Convert to protobuf
	protoData := &pb.LLMRootingDataProto{
		ModelName:   llmData.ModelName,
		ModelOwner:  llmData.ModelOwner,
		ApiEndpoint: llmData.APIEndpoint,
		MetadataCid: llmData.MetadataCID,
		Cmu:         llmData.CMU,
	}

	// Serialize the data using protobuf
	data, err := proto.Marshal(protoData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal LLM rooting data: %w", err)
	}

	// Use provided timestamp if available, otherwise use current time
	var timestamp int64
	if len(optionalTimestamp) > 0 && optionalTimestamp[0] != 0 {
		timestamp = optionalTimestamp[0]
	} else {
		timestamp = time.Now().Unix()
	}

	// Create transaction
	tx := &Transaction{
		From:      from,
		To:        "", // No recipient for rooting transactions
		Value:     0,  // No value transfer
		Data:      data,
		Timestamp: timestamp,
		Fee:       fee,
		Type:      TransactionTypeLLMRooting,
		Status:    utils.PENDING,
	}

	// Calculate hash
	tx.TransactionHash = tx.Hash()

	return tx, nil
}

// NewMCPTransaction creates a new transaction with MCP-specific fields
// If optionalTimestamp is provided and non-zero, it will be used instead of the current time
// Returns nil and error if validation fails
func NewMCPTransaction(from, to string, value uint64, data []byte, txType string, fee uint64, optionalTimestamp ...int64) (*Transaction, error) {
	// For registration transactions, validate capability descriptor first
	if txType == TransactionTypeMCPRegisterCapability {
		// Basic check: ensure capabilityDescriptor.ID is present and valid in the data
		var parsedData struct {
			CapabilityDescriptor struct {
				ID string `json:"id"`
			} `json:"capabilityDescriptor"`
		}
		if err := json.Unmarshal(data, &parsedData); err != nil {
			log.Printf("[ERROR] NewMCPTransaction: failed to unmarshal basic capability descriptor for ID check: %v", err)
			return nil, fmt.Errorf("failed to unmarshal MCP data: %w", err)
		}
		if parsedData.CapabilityDescriptor.ID == "" {
			log.Printf("[ERROR] NewMCPTransaction: empty capability ID in registration transaction data")
			return nil, errors.New("empty capability ID in registration transaction data")
		}
		if strings.ContainsAny(parsedData.CapabilityDescriptor.ID, "#$%&") {
			log.Printf("[ERROR] NewMCPTransaction: invalid characters in capability ID: %s", parsedData.CapabilityDescriptor.ID)
			return nil, errors.New("invalid characters in capability ID")
		}
	}

	// Use provided timestamp if available, otherwise use current time
	var timestamp int64
	if len(optionalTimestamp) > 0 && optionalTimestamp[0] != 0 {
		timestamp = optionalTimestamp[0]
	} else {
		timestamp = time.Now().Unix()
	}

	// Create a new transaction with the specified timestamp
	transaction := NewTransaction(from, to, value, data, timestamp)
	transaction.Type = txType
	transaction.Fee = fee

	// Recalculate hash using protobuf for consistency
	unsignedTx := &Transaction{
		From:      from,
		To:        to,
		Value:     value,
		Data:      data,
		Timestamp: timestamp,
		Fee:       fee,
		Type:      txType,
	}

	// Convert to proto, marshal to bytes, then hash
	txProto, err := unsignedTx.ToProtoForHashing()
	if err != nil {
		log.Printf("Error converting unsigned MCP tx to proto for hash: %v", err)
		return nil, fmt.Errorf("error calculating transaction hash: %w", err)
	}
	canonicalBytes, err := GetCanonicalBytesForHashingTransaction(txProto)
	if err != nil {
		return nil, fmt.Errorf("error getting canonical bytes for hashing: %w", err)
	}
	hash := sha256.Sum256(canonicalBytes)
	transaction.TransactionHash = utils.HEX_PREFIX + fmt.Sprintf("%x", hash[:])

	return transaction, nil
}

func (t Transaction) ToJson() string {
	nb, err := json.Marshal(t)

	if err != nil {
		return err.Error()
	} else {
		return string(nb)
	}
}

func (t Transaction) VerifyTxn() bool {
	if t.From == utils.BLOCKCHAIN_ADDRESS {
		return t.isValidProtocolTransaction()
	}
	// Skip value check for MCP transactions that might have zero value
	if t.Type == "" {
		if t.Value <= 0 {
			return false
		}

		if t.Value > math.MaxUint32 {
			return false
		}
	}

	if t.From == t.To {
		return false
	}

	// Validate MCP transaction types
	if t.Type != "" {
		switch t.Type {
		case TransactionTypeMCPRegisterCapability:
			// Check capability descriptor structure and validate sender matches owner
			var regData struct {
				CapabilityDescriptor types.BaseDescriptor `json:"capabilityDescriptor"`
			}
			if err := json.Unmarshal(t.Data, &regData); err != nil {
				log.Printf("[ERROR] VerifyTxn: failed to unmarshal MCPRegisterCapabilityData for basic structural check: %v", err)
				return false
			}
			if regData.CapabilityDescriptor.ID == "" { // ID check
				log.Printf("[ERROR] VerifyTxn: ID missing in CapabilityDescriptor for MCPRegisterCapability")
				return false
			}
			// CapabilityType check
			if !types.IsValidCapabilityType(regData.CapabilityDescriptor.CapabilityType) {
				log.Printf("[ERROR] VerifyTxn: Invalid CapabilityType '%s' in CapabilityDescriptor for MCPRegisterCapability", regData.CapabilityDescriptor.CapabilityType)
				return false
			}
			// Verify transaction sender matches capability owner
			if t.From != regData.CapabilityDescriptor.Owner {
				log.Printf("[ERROR] VerifyTxn: Transaction sender (%s) does not match capability owner (%s)", t.From, regData.CapabilityDescriptor.Owner)
				return false
			}
			// Fall through to signature verification

		case TransactionTypeMCPInvokeCapability:
			var mcpInvokeData types.MCPInvokeCapabilityData
			if err := json.Unmarshal(t.Data, &mcpInvokeData); err != nil {
				log.Printf("[ERROR] VerifyTxn: failed to unmarshal MCPInvokeCapabilityData for basic check: %v", err)
				return false
			}
			// Fall through to signature verification

		case TransactionTypeMCPUpdateCapability:
			var mcpUpdateData types.MCPUpdateCapabilityData // Assuming this type exists
			if err := json.Unmarshal(t.Data, &mcpUpdateData); err != nil {
				log.Printf("[ERROR] VerifyTxn: failed to unmarshal MCPUpdateCapabilityData for basic check: %v", err)
				return false
			}
			// Fall through to signature verification

		case TransactionTypeLLMRooting:
			var protoData pb.LLMRootingDataProto
			if err := proto.Unmarshal(t.Data, &protoData); err != nil {
				log.Printf("[ERROR] VerifyTxn: failed to unmarshal LLMRootingDataProto: %v", err)
				return false
			}
			llmData := LLMRootingData{
				ModelName:   protoData.ModelName,
				ModelOwner:  protoData.ModelOwner,
				APIEndpoint: protoData.ApiEndpoint,
				MetadataCID: protoData.MetadataCid,
				CMU:         protoData.Cmu,
			}
			// Validate required fields
			if llmData.ModelName == "" || llmData.ModelOwner == "" ||
				llmData.APIEndpoint == "" || llmData.MetadataCID == "" {
				log.Printf("[ERROR] VerifyTxn: missing required fields in LLM rooting data")
				return false
			}
			// Verify CMU is correctly generated
			expectedCMU := GenerateCMU(llmData, "mainnet")
			if llmData.CMU != expectedCMU {
				log.Printf("[ERROR] VerifyTxn: invalid CMU. Expected: %s, Got: %s", expectedCMU, llmData.CMU)
				return false
			}
			// Verify transaction sender matches model owner
			if t.From != llmData.ModelOwner {
				log.Printf("[ERROR] VerifyTxn: transaction sender (%s) does not match model owner (%s)", t.From, llmData.ModelOwner)
				return false
			}
			// Fall through to signature verification

		default:
			// Invalid transaction type
			log.Printf("[ERROR] VerifyTxn: unknown transaction type: %s", t.Type)
			return false
		}
	}

	valid, err := t.VerifySignature()
	if err != nil {
		return false
	}

	return valid
}

func (t *Transaction) VerifySignature() (bool, error) {
	if t.From == utils.BLOCKCHAIN_ADDRESS {
		if !t.isValidProtocolTransaction() {
			return false, fmt.Errorf("invalid unsigned protocol transaction")
		}
		return true, nil
	}

	// Basic checks
	if len(t.Signature) == 0 {
		log.Println("Verification failed: Signature is nil or empty")
		return false, fmt.Errorf("signature is nil or empty")
	}
	if t.PublicKey == "" || t.BodyBytes == "" || t.AuthInfoBytes == "" || t.ChainID == "" {
		return false, fmt.Errorf("canonical signing fields are incomplete")
	}
	body, err := base64.StdEncoding.DecodeString(t.BodyBytes)
	if err != nil {
		return false, fmt.Errorf("decode body_bytes: %w", err)
	}
	auth, err := base64.StdEncoding.DecodeString(t.AuthInfoBytes)
	if err != nil {
		return false, fmt.Errorf("decode auth_info_bytes: %w", err)
	}
	pub, err := base64.StdEncoding.DecodeString(t.PublicKey)
	if err != nil {
		return false, fmt.Errorf("decode public_key: %w", err)
	}
	action, err := knirvsigning.ParseActionBody(body)
	if err != nil {
		return false, fmt.Errorf("decode signed action: %w", err)
	}
	expectedType := t.Type
	if expectedType == "" {
		expectedType = "transfer"
	}
	if action.Action != expectedType || action.Sender != t.From || action.Recipient != t.To ||
		action.Amount != t.Value || action.TimestampUnix != t.Timestamp || !reflect.DeepEqual(action.Payload, t.Data) {
		return false, fmt.Errorf("signed action does not match transaction fields")
	}
	address, err := knirvsigning.Address(pub, knirvsigning.DefaultAddressPrefix)
	if err != nil || address != t.From {
		return false, fmt.Errorf("public key does not match transaction sender")
	}
	signed := knirvsigning.SignedTransaction{
		BodyBytes: body, AuthInfoBytes: auth, Signatures: [][]byte{t.Signature},
		PublicKey: pub, Address: t.From,
	}
	if err := knirvsigning.VerifyTransaction(signed, t.ChainID, t.AccountNumber); err != nil {
		return false, err
	}
	raw := knirvsigning.MarshalTxRaw(body, auth, [][]byte{t.Signature})
	hash := sha256.Sum256(raw)
	if t.TransactionHash != "" && !strings.EqualFold(strings.TrimPrefix(t.TransactionHash, "0x"), fmt.Sprintf("%x", hash)) {
		return false, fmt.Errorf("transaction hash does not match signed TxRaw")
	}
	return true, nil
}

func (t *Transaction) isValidProtocolTransaction() bool {
	if t == nil || t.From != utils.BLOCKCHAIN_ADDRESS || t.Fee != 0 || t.To == "" {
		return false
	}
	switch t.Type {
	case "protocol_mining_reward":
		return t.Value == utils.MINING_REWARD && len(t.Data) == 0
	case "protocol_validation_proof":
		return t.Value == 0 && len(t.Data) > 0
	case "protocol_uri_mint":
		return t.Value == 0 && len(t.Data) > 0
	case "demo_faucet":
		return strings.EqualFold(strings.TrimSpace(os.Getenv("KNIRV_ENABLE_DEMO")), "true") && t.Value > 0
	default:
		return false
	}
}

func (t Transaction) Hash() string {
	if t.BodyBytes != "" && t.AuthInfoBytes != "" && len(t.Signature) == 64 {
		body, bodyErr := base64.StdEncoding.DecodeString(t.BodyBytes)
		auth, authErr := base64.StdEncoding.DecodeString(t.AuthInfoBytes)
		if bodyErr == nil && authErr == nil {
			sum := sha256.Sum256(knirvsigning.MarshalTxRaw(body, auth, [][]byte{t.Signature}))
			return utils.HEX_PREFIX + hex.EncodeToString(sum[:])
		}
	}
	// Create unsigned transaction copy for hash calculation
	unsignedTx := &Transaction{
		From:      t.From,
		To:        t.To,
		Value:     t.Value,
		Data:      t.Data,
		Timestamp: t.Timestamp,
		Fee:       t.Fee,
		Type:      t.Type,
	}

	// Convert to proto, marshal to bytes, then hash
	txProto, err := unsignedTx.ToProtoForHashing()
	if err != nil {
		log.Printf("Error in Transaction.Hash() converting to proto: %v", err)
		// Return a dummy error hash or handle more gracefully
		return "error_hash"
	}
	canonicalBytes, _ := GetCanonicalBytesForHashingTransaction(txProto)
	sum := sha256.Sum256(canonicalBytes)
	hexRep := hex.EncodeToString(sum[:32])
	formattedHexRep := utils.HEX_PREFIX + hexRep

	return formattedHexRep
}

func (tx *Transaction) Clone() *Transaction {

	copiedSignature := make([]byte, len(tx.Signature))
	copy(copiedSignature, tx.Signature)

	copiedPublicKey := tx.PublicKey

	copiedData := make([]byte, len(tx.Data))
	copy(copiedData, tx.Data)

	return &Transaction{ // Create a completely new Transaction instance
		From:            tx.From,
		To:              tx.To,
		Value:           tx.Value,
		Data:            copiedData, // Copy the byte slice
		Status:          tx.Status,
		Timestamp:       tx.Timestamp,
		TransactionHash: tx.TransactionHash,
		Fee:             tx.Fee,
		Type:            tx.Type,
		BodyBytes:       tx.BodyBytes,
		AuthInfoBytes:   tx.AuthInfoBytes,
		ChainID:         tx.ChainID,
		AccountNumber:   tx.AccountNumber,
		Sequence:        tx.Sequence,
		Verified:        tx.Verified,
		BlockHash:       tx.BlockHash,

		PublicKey: copiedPublicKey,
		Signature: copiedSignature,
	}
}

// ValidateTransactionInBlockContext performs final validation before applying a transaction.
// This is a simplified version. Real validation is more complex.
func (bc *BlockchainStruct) ValidateTransactionInBlockContext(tx *Transaction, currentBlockAccounts map[string]*big.Int) error {
	// First validate general transaction properties
	if !bc.validateTransaction(tx) {
		return fmt.Errorf("transaction %s failed basic validation", tx.TransactionHash)
	}
	// Check signature (should have been done before pool, but good for defense)
	if !tx.VerifyTxn() {
		// Get detailed error from VerifySignature() if available
		if _, err := tx.VerifySignature(); err != nil {
			log.Printf("Transaction %s failed basic validation: %v", tx.TransactionHash, err)
			return fmt.Errorf("transaction %s failed basic validation: %w", tx.TransactionHash, err)
		}
		return fmt.Errorf("transaction %s failed basic validation", tx.TransactionHash)
	}

	// Check sufficient funds for network fee (tx.Fee)
	senderBalance, ok := currentBlockAccounts[tx.From]
	if !ok {
		senderBalance = big.NewInt(0)
	}
	networkFee := new(big.Int).SetUint64(tx.Fee)
	if senderBalance.Cmp(networkFee) < 0 {
		return fmt.Errorf("insufficient balance for network fee for tx %s. From: %s, Balance: %s, Fee: %s",
			tx.TransactionHash, tx.From, senderBalance.String(), networkFee.String())
	}

	// Additional validation for MCP registration transactions
	if tx.Type == TransactionTypeMCPRegisterCapability && bc.db != nil {
		var desc struct {
			CapabilityDescriptor types.BaseDescriptor `json:"capabilityDescriptor"`
		}
		if err := json.Unmarshal(tx.Data, &desc); err != nil {
			return fmt.Errorf("failed to unmarshal capability descriptor: %w", err)
		}

		// Check if capability already exists
		existingCap, err := bc.db.GetCapabilityByID(desc.CapabilityDescriptor.ID)
		if err != nil {
			// For registrations, "not found" is the desired state
			if !strings.Contains(err.Error(), "not found") {
				return fmt.Errorf("failed to check capability existence: %w", err)
			}
		} else if existingCap != nil {
			return fmt.Errorf("capability ID already exists: %s", desc.CapabilityDescriptor.ID)
		}
	}

	// If MCPInvoke, check for GasFeeNRN as well
	if tx.Type == TransactionTypeMCPInvokeCapability && bc.db != nil { // Ensure db is available
		var mcpInvokeData types.MCPInvokeCapabilityData
		if err := json.Unmarshal(tx.Data, &mcpInvokeData); err != nil {
			return fmt.Errorf("validateTransactionInBlockContext: failed to unmarshal MCPInvokeCapabilityData for tx %s: %w", tx.TransactionHash, err)
		} // Ensure bc.db is initialized and passed correctly
		capDescInterface, err := bc.db.GetCapabilityByID(mcpInvokeData.ContextRecord.CapabilityID) // Fetch from DB
		if err != nil {
			return fmt.Errorf("validateTransactionInBlockContext: capability %s not found for tx %s: %w", mcpInvokeData.ContextRecord.CapabilityID, tx.TransactionHash, err)
		}
		baseDesc, err := getBaseDescriptorFromInterface(capDescInterface) // Use unexported name
		if err != nil {
			return fmt.Errorf("validateTransactionInBlockContext: failed to get base descriptor for tx %s: %w", tx.TransactionHash, err)
		}

		totalCost := new(big.Int).SetUint64(tx.Fee)
		if baseDesc.GasFeeNRN > 0 {
			totalCost.Add(totalCost, new(big.Int).SetUint64(baseDesc.GasFeeNRN))
		}
		if senderBalance.Cmp(totalCost) < 0 {
			return fmt.Errorf("insufficient balance for network fee + capability fee for tx %s. From: %s, Balance: %s, TotalCost: %s",
				tx.TransactionHash, tx.From, senderBalance.String(), totalCost.String())
		}
	}
	// Add other validation: nonce, etc.
	return nil
}

// applyTransactionToState applies the transaction's effects to the provided accounts map.
func (bc *BlockchainStruct) applyTransactionToState(tx *Transaction, accounts map[string]*big.Int, blockProposerAddress string) error {
	sender := tx.From
	networkFee := new(big.Int).SetUint64(tx.Fee)

	// Handle network fee deduction for non-faucet senders
	if sender != utils.BLOCKCHAIN_ADDRESS {
		if _, ok := accounts[sender]; !ok {
			// This should ideally not happen if pre-loading is correct and sender is not faucet
			accounts[sender] = big.NewInt(0)
		}

		// 1. Deduct Network Fee from sender
		accounts[sender].Sub(accounts[sender], networkFee)
	} else {
		// For faucet, network fee is effectively absorbed/ignored if non-zero.
	}

	// 2. Credit Network Fee to block proposer (miner/validator)
	if _, ok := accounts[blockProposerAddress]; !ok {
		accounts[blockProposerAddress] = big.NewInt(0)
	}
	accounts[blockProposerAddress] = new(big.Int).Add(accounts[blockProposerAddress], networkFee)

	// 3. Handle MCP specific logic
	if strings.HasPrefix(tx.Type, "mcp_") { // Corrected prefix to lowercase "mcp_"
		// Delegate MCP transaction effects to the MCPProcessor
		if bc.mcpProcessor == nil { // Should not happen if initialized correctly
			return fmt.Errorf("MCPProcessor not initialized in BlockchainStruct for tx %s", tx.TransactionHash)
		}
		if _, err := bc.mcpProcessor.ApplyMCPTransactionEffects(tx, accounts); err != nil {
			return err
		} // Convert tx.Value to big.Int for comparison
	} else if new(big.Int).SetUint64(tx.Value).Cmp(big.NewInt(0)) > 0 { // Standard value transfer
		if _, ok := accounts[tx.To]; !ok {
			accounts[tx.To] = big.NewInt(0)
		}
		txValueBigInt := new(big.Int).SetUint64(tx.Value)

		// Deduct value from sender only if sender is not the faucet
		if sender != utils.BLOCKCHAIN_ADDRESS {
			if accounts[sender].Cmp(txValueBigInt) < 0 {
				return fmt.Errorf("insufficient balance for value transfer in tx %s after fee deduction", tx.TransactionHash)
			}
			accounts[sender].Sub(accounts[sender], txValueBigInt)
		}
		// Credit value to recipient
		accounts[tx.To] = new(big.Int).Add(accounts[tx.To], txValueBigInt)
	}

	return nil
}

// getBaseDescriptorFromInterface extracts BaseDescriptor from various capability types.
func getBaseDescriptorFromInterface(capDescInterface interface{}) (types.BaseDescriptor, error) {
	// Check for nil interface
	if capDescInterface == nil {
		return types.BaseDescriptor{}, fmt.Errorf("capability descriptor is nil")
	}

	// Check for proto.CapabilityDescriptorContainerProto type
	if container, ok := capDescInterface.(*pb.CapabilityDescriptorContainerProto); ok {
		if container == nil {
			return types.BaseDescriptor{}, fmt.Errorf("CapabilityDescriptorContainerProto is nil")
		}

		// Extract the base descriptor from the appropriate oneof field
		switch desc := container.Descriptor_.(type) {
		case *pb.CapabilityDescriptorContainerProto_Resource:
			if desc == nil || desc.Resource == nil || desc.Resource.BaseDescriptor == nil {
				return types.BaseDescriptor{}, fmt.Errorf("resource descriptor or its base descriptor is nil")
			}
			return protoToBaseDescriptor(desc.Resource.BaseDescriptor), nil
		case *pb.CapabilityDescriptorContainerProto_Tool:
			if desc == nil || desc.Tool == nil || desc.Tool.BaseDescriptor == nil {
				return types.BaseDescriptor{}, fmt.Errorf("tool descriptor or its base descriptor is nil")
			}
			return protoToBaseDescriptor(desc.Tool.BaseDescriptor), nil
		case *pb.CapabilityDescriptorContainerProto_Prompt:
			if desc == nil || desc.Prompt == nil || desc.Prompt.BaseDescriptor == nil {
				return types.BaseDescriptor{}, fmt.Errorf("prompt descriptor or its base descriptor is nil")
			}
			return protoToBaseDescriptor(desc.Prompt.BaseDescriptor), nil
		case *pb.CapabilityDescriptorContainerProto_MemoryService:
			if desc == nil || desc.MemoryService == nil || desc.MemoryService.BaseDescriptor == nil {
				return types.BaseDescriptor{}, fmt.Errorf("memory service descriptor or its base descriptor is nil")
			}
			return protoToBaseDescriptor(desc.MemoryService.BaseDescriptor), nil
		default:
			return types.BaseDescriptor{}, fmt.Errorf("unknown or nil descriptor type in CapabilityDescriptorContainerProto")
		}
	}

	// Use reflection for non-proto types
	val := reflect.ValueOf(capDescInterface)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return types.BaseDescriptor{}, fmt.Errorf("capability descriptor is not a struct: %T", capDescInterface)
	}
	baseDescField := val.FieldByName("BaseDescriptor")
	if !baseDescField.IsValid() {
		// If BaseDescriptor is not found directly, check if the interface itself is a BaseDescriptor
		if bd, ok := capDescInterface.(types.BaseDescriptor); ok {
			return bd, nil
		}
		return types.BaseDescriptor{}, fmt.Errorf("no 'BaseDescriptor' field in capability descriptor type: %T: %v", capDescInterface, capDescInterface)
	}
	if baseDesc, ok := baseDescField.Interface().(types.BaseDescriptor); ok {
		return baseDesc, nil
	}
	return types.BaseDescriptor{}, fmt.Errorf("field 'BaseDescriptor' is not of type BaseDescriptor in %T", capDescInterface)
}

// protoToBaseDescriptor converts a proto BaseDescriptor to domain BaseDescriptor
func protoToBaseDescriptor(protoBase *pb.BaseDescriptorProto) types.BaseDescriptor {
	if protoBase == nil {
		return types.BaseDescriptor{}
	}

	// Convert capability type from proto enum to domain type
	var capType types.CapabilityType
	switch protoBase.CapabilityType {
	case pb.CapabilityTypeProto_CAPABILITY_TYPE_PROTO_RESOURCE:
		capType = types.CapabilityTypeResource
	case pb.CapabilityTypeProto_CAPABILITY_TYPE_PROTO_TOOL:
		capType = types.CapabilityTypeTool
	case pb.CapabilityTypeProto_CAPABILITY_TYPE_PROTO_PROMPT:
		capType = types.CapabilityTypePrompt
	case pb.CapabilityTypeProto_CAPABILITY_TYPE_PROTO_MEMORY_SERVICE:
		capType = types.CapabilityTypeMemoryService
	default:
		capType = types.CapabilityTypeUnspecified
	}

	// Convert custom metadata from proto map to Go map
	customMetadata := make(map[string]interface{})
	for k, v := range protoBase.CustomMetadata.AsMap() {
		customMetadata[k] = v
	}

	return types.BaseDescriptor{
		ID:             protoBase.Id,
		Name:           protoBase.Name,
		Owner:          protoBase.Owner,
		Version:        protoBase.Version,
		Description:    protoBase.Description,
		GasFeeNRN:      protoBase.GasFeeNrn,
		Timestamp:      protoBase.Timestamp.AsTime().Unix(),
		CustomMetadata: customMetadata,
		CapabilityType: capType,
	}
}

const (
	// ... existing transaction types ...
	TransactionTypeNFTCapabilityAttachment = "NFT_CAPABILITY_ATTACHMENT"
	TransactionTypeLLMRooting              = "llm_rooting"

	// KNIRVCHAIN ErrorNode → SkillNode Mining Flow
	TransactionTypeErrorNodeSubmit     = "error_node_submit"
	TransactionTypeSkillMiningProposal = "skill_mining_proposal"
	TransactionTypeSkillValidation     = "skill_validation"
	TransactionTypeSkillConfirmation   = "skill_confirmation"

	// KNIRVCHAIN ContextNode → CapabilityNode Minting Flow
	TransactionTypeContextNodeCreate      = "context_node_create"
	TransactionTypeCapabilityMintProposal = "capability_mint_proposal"
	TransactionTypeCapabilityValidation   = "capability_validation"
	TransactionTypeCapabilityMint         = "capability_mint"

	// KNIRVCHAIN IdeaNode → PropertyNode Making Flow
	TransactionTypeIdeaNodeSubmit      = "idea_node_submit"
	TransactionTypeNoveltyAssessment   = "novelty_assessment"
	TransactionTypePropertyMint        = "property_mint"
	TransactionTypePropertyTransfer    = "property_transfer"
	TransactionTypeRoyaltyDistribution = "royalty_distribution"

	// TransactionTypeEventBundleMint mints one KNIRVCHAIN EventBundleNFT per
	// CLI decision/error event, bundling the Skills/Capabilities(MCPs)/
	// Assets/Context/Credentials used to produce it (chain_refactor.md §3.2).
	TransactionTypeEventBundleMint = "event_bundle_mint"

	// Actuarial Syndicate transaction types follow UPPER_SNAKE_CASE to visually
	// distinguish financial-authority operations from the mostly-snake_case
	// non-financial types above. This is a deliberate deviation, not an oversight.
	TransactionTypeRISKSNAPSHOTCommit    = "RISK_SNAPSHOT_COMMIT"
	TransactionTypeMODELActivate         = "MODEL_ACTIVATE"
	TransactionTypePOOLCREATE            = "POOL_CREATE"
	TransactionTypeSTAKEDEPOSIT          = "STAKE_DEPOSIT"
	TransactionTypeSTAKEEXITREQUEST      = "STAKE_EXIT_REQUEST"
	TransactionTypeSUBMISSIONCOMMIT      = "SUBMISSION_COMMIT"
	TransactionTypePRICINGDECISIONCOMMIT = "PRICING_DECISION_COMMIT"
	TransactionTypeRESERVECOMMIT         = "RESERVE_COMMIT"
	TransactionTypeSETTLEMENTCOMMIT      = "SETTLEMENT_COMMIT"
	TransactionTypeLOSSALLOCATIONCOMMIT  = "LOSS_ALLOCATION_COMMIT"
)

// SyndicateCommitment is the only payload accepted for financial-authority
// syndicate transactions. It contains commitments and public terms only; raw
// telemetry, PoCs, identities, and provider credentials are never chain data.
type SyndicateCommitment struct {
	SchemaVersion   string `json:"schema_version"`
	EntityID        string `json:"entity_id"`
	CommitmentHash  string `json:"commitment_hash"`
	PredecessorHash string `json:"predecessor_hash,omitempty"`
	Amount          uint64 `json:"amount,omitempty"`
	Currency        string `json:"currency,omitempty"`
	ModelVersion    string `json:"model_version,omitempty"`
	SnapshotID      string `json:"snapshot_id,omitempty"`
}

func IsSyndicateTransactionType(txType string) bool {
	switch txType {
	case TransactionTypeRISKSNAPSHOTCommit, TransactionTypeMODELActivate, TransactionTypePOOLCREATE, TransactionTypeSTAKEDEPOSIT, TransactionTypeSTAKEEXITREQUEST, TransactionTypeSUBMISSIONCOMMIT, TransactionTypePRICINGDECISIONCOMMIT, TransactionTypeRESERVECOMMIT, TransactionTypeSETTLEMENTCOMMIT, TransactionTypeLOSSALLOCATIONCOMMIT:
		return true
	default:
		return false
	}
}

func ValidateSyndicateCommitment(txType string, data []byte) (*SyndicateCommitment, error) {
	if !IsSyndicateTransactionType(txType) {
		return nil, fmt.Errorf("not a syndicate transaction type: %s", txType)
	}
	var commitment SyndicateCommitment
	if err := json.Unmarshal(data, &commitment); err != nil {
		return nil, fmt.Errorf("decode syndicate commitment: %w", err)
	}
	if commitment.SchemaVersion != "knirv.syndicate.v1" || commitment.EntityID == "" || len(commitment.CommitmentHash) != 64 {
		return nil, fmt.Errorf("invalid syndicate commitment envelope")
	}
	if (txType == TransactionTypeSTAKEDEPOSIT || txType == TransactionTypeRESERVECOMMIT || txType == TransactionTypeSETTLEMENTCOMMIT || txType == TransactionTypeLOSSALLOCATIONCOMMIT) && commitment.Amount == 0 {
		return nil, fmt.Errorf("%s requires a positive amount", txType)
	}
	if txType == TransactionTypePRICINGDECISIONCOMMIT && (commitment.ModelVersion == "" || commitment.SnapshotID == "") {
		return nil, fmt.Errorf("pricing decision requires model version and snapshot")
	}
	return &commitment, nil
}

// NFTCapabilityAttachmentData represents the data for an NFT capability attachment transaction
type NFTCapabilityAttachmentData struct {
	NFTID        string                 `json:"nft_id"`
	CapabilityID string                 `json:"capability_id"`
	Params       map[string]interface{} `json:"params,omitempty"`
}

// LLMRootingData contains the core metadata for an LLM model rooting transaction
type LLMRootingData struct {
	ModelName   string `json:"model_name"`   // e.g., "OpenAI-GPT4-Turbo"
	ModelOwner  string `json:"model_owner"`  // Public key or verified entity ID
	APIEndpoint string `json:"api_endpoint"` // The actual chat completion URL
	MetadataCID string `json:"metadata_cid"` // Content Identifier for off-chain metadata
	CMU         string `json:"cmu"`          // The Chain Minted URL
}

// GenerateCMU generates a Chain Minted URL for LLM rooting data
func GenerateCMU(data LLMRootingData, networkID string) string {
	// ModelHash is SHA256 of ModelName + ModelOwner + MetadataCID
	// APIEndpoint is excluded to allow updates
	modelHashData := data.ModelName + data.ModelOwner + data.MetadataCID
	modelHash := sha256.Sum256([]byte(modelHashData))

	return fmt.Sprintf("knirv://%s/%s", networkID, hex.EncodeToString(modelHash[:]))
}
