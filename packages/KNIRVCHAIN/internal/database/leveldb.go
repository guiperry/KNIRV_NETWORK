package database

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "KNIRVCHAIN/internal/protocol/proto"
	"KNIRVCHAIN/internal/types"
)

// CapabilityConverter defines the interface for converting capabilities to proto
type CapabilityConverter interface {
	ConvertToCapabilityDescriptorContainerProto(capability interface{}) (*pb.CapabilityDescriptorContainerProto, error)
}

// LevelDB implements the MCPDatabase interface using LevelDB.
type LevelDB struct {
	Client    *leveldb.DB
	path      string
	converter CapabilityConverter
}

// Key prefixes for different types of data
const (
	KeyPrefixAccountBalance            = "account:balance:"
	KeyPrefixCapability                = "cap:"
	KeyPrefixContext                   = "ctx:"
	KeyPrefixCapabilityTypeIdx         = "idx:cap_type:"
	KeyPrefixCapabilityOwnerIdx        = "idx:cap_owner:"
	KeyPrefixCapabilityInvocationsIdx  = "idx:cap_invocations:"
	KeyPrefixContextCapabilityIdx      = "idx:ctx_capability:"
	KeyPrefixContextInteractionTypeIdx = "idx:ctx_interaction_type:"
	KeyPrefixContextInitiatorIdx       = "idx:ctx_initiator:"

	// PoAu-D specific keys
	NetworkAuthorsKey = "config:network_authors"
	PoAuDEnabledKey   = "config:poaud_enabled"
)

// NewDBClient creates a new LevelDB instance (alias for NewLevelDB)
func NewDBClient(path string) (*LevelDB, error) {
	return NewLevelDB(path)
}

// NewLevelDB creates a new LevelDB instance.
func NewLevelDB(dbPath string) (*LevelDB, error) {
	db, err := leveldb.OpenFile(dbPath, &opt.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to open leveldb at %s: %w", dbPath, err)
	}
	return &LevelDB{Client: db, path: dbPath, converter: nil}, nil
}

// NewLevelDBWithConverter creates a new LevelDB instance with a capability converter.
func NewLevelDBWithConverter(dbPath string, converter CapabilityConverter) (*LevelDB, error) {
	db, err := leveldb.OpenFile(dbPath, &opt.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to open leveldb at %s: %w", dbPath, err)
	}
	return &LevelDB{Client: db, path: dbPath, converter: converter}, nil
}

// SetConverter sets the capability converter for the LevelDB instance.
func (db *LevelDB) SetConverter(converter CapabilityConverter) {
	db.converter = converter
}

func (db *LevelDB) SaveLastBlock(block interface{}, address string) error {
	blockBytes, err := json.Marshal(block)
	if err != nil {
		return err
	}
	err = db.Client.Put([]byte(address+"last_block"), blockBytes, &opt.WriteOptions{})
	if err != nil {
		return fmt.Errorf("failed to put data into database with key last_block: %w", err)
	}
	return nil
}

func (db *LevelDB) LoadLastBlock(address string) (interface{}, error) {
	data, err := db.Client.Get([]byte(address+"last_block"), &opt.ReadOptions{})
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, fmt.Errorf("no block found: %w", err)
		}
		return nil, fmt.Errorf("failed to get data from database with key last_block: %w", err)
	}

	block := new(interface{})
	err = json.Unmarshal(data, block)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return block, nil
}

// GetBalance retrieves the NRN token balance for an address
func (db *LevelDB) GetBalance(address string) (uint64, error) {
	// Get the balance
	balanceBytes, err := db.GetBytes(KeyPrefixAccountBalance + address)
	if err != nil {
		// If the key doesn't exist, return 0 balance
		if err == leveldb.ErrNotFound {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get balance: %w", err)
	}

	// Parse the balance
	var balance uint64
	err = json.Unmarshal(balanceBytes, &balance)
	if err != nil {
		return 0, fmt.Errorf("failed to unmarshal balance: %w", err)
	}

	return balance, nil
}

// UpdateBalance updates the NRN token balance for an address
func (db *LevelDB) UpdateBalance(address string, amount int64) error {
	// Get the current balance
	balance, err := db.GetBalance(address)
	if err != nil {
		return fmt.Errorf("failed to get balance: %w", err)
	}

	// Update the balance
	if amount < 0 && uint64(-amount) > balance {
		return fmt.Errorf("insufficient balance")
	}
	newBalance := balance + uint64(amount)

	// Marshal the new balance
	balanceBytes, err := json.Marshal(newBalance)
	if err != nil {
		return fmt.Errorf("failed to marshal balance: %w", err)
	}

	// Save the new balance
	err = db.PutBytes(KeyPrefixAccountBalance+address, balanceBytes)
	if err != nil {
		return fmt.Errorf("failed to save balance: %w", err)
	}

	return nil
}

func (db *LevelDB) KeyExists(address string) (bool, error) {
	_, err := db.Client.Get([]byte(address), &opt.ReadOptions{})
	if err != nil {
		if err == leveldb.ErrNotFound {
			return false, nil
		}
		return false, fmt.Errorf("failed to get data from database with key blockchain: %w", err)

	}
	return true, nil
}

func (db *LevelDB) GetBlockchain(address string) ([]byte, error) {
	data, err := db.Client.Get([]byte(address), &opt.ReadOptions{})
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, fmt.Errorf("no blockchain found: %w", err)
		}
		return nil, fmt.Errorf("failed to get data from database with key blockchain: %w", err)
	}

	return data, nil
}

// new and correct implementations using object state to implement workflow, for types. where those data types and methods from structs/object should use those other structs and object methods from same program.

// PutBytes saves raw byte data to the database with the given key.
func (db *LevelDB) PutBytes(key string, data []byte) error {
	err := db.Client.Put([]byte(key), data, &opt.WriteOptions{Sync: true})
	if err != nil {
		return fmt.Errorf("failed to put data into database with key %s: %w", key, err)
	}
	return nil
}

// GetBytes retrieves raw byte data from the database
func (db *LevelDB) GetBytes(key string) ([]byte, error) {
	data, err := db.Client.Get([]byte(key), &opt.ReadOptions{})
	if err != nil {
		if err == leveldb.ErrNotFound {
			// Return the original leveldb.ErrNotFound error for consistent error checking
			return nil, err
		}
		return nil, fmt.Errorf("failed to get data for key %s: %w", key, err)
	}
	return data, nil
}

// Account Balance Methods

// SaveAccountBalance saves an account's NRN token balance
func (db *LevelDB) SaveAccountBalance(address string, balance uint64) error {
	key := KeyPrefixAccountBalance + address
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, balance)
	return db.PutBytes(key, data)
}

// GetAccountBalance retrieves an account's NRN token balance
func (db *LevelDB) GetAccountBalance(address string) (uint64, error) {
	key := KeyPrefixAccountBalance + address
	data, err := db.Client.Get([]byte(key), &opt.ReadOptions{})
	if err != nil {
		if err == leveldb.ErrNotFound {
			return 0, nil // Return 0 balance if account not found
		}
		return 0, fmt.Errorf("failed to get account balance: %w", err)
	}
	if len(data) != 8 {
		return 0, fmt.Errorf("invalid balance data length")
	}
	return binary.BigEndian.Uint64(data), nil
}

// TransferBalance atomically transfers NRN tokens between accounts
func (db *LevelDB) TransferBalance(from string, to string, amount uint64) error {
	// Get current balances
	fromBalance, err := db.GetAccountBalance(from)
	if err != nil {
		return fmt.Errorf("failed to get sender balance: %w", err)
	}
	toBalance, err := db.GetAccountBalance(to)
	if err != nil {
		return fmt.Errorf("failed to get recipient balance: %w", err)
	}

	// Check sufficient balance
	if fromBalance < amount {
		return fmt.Errorf("insufficient balance")
	}

	// Update balances
	err = db.SaveAccountBalance(from, fromBalance-amount)
	if err != nil {
		return fmt.Errorf("failed to update sender balance: %w", err)
	}
	err = db.SaveAccountBalance(to, toBalance+amount)
	if err != nil {
		// Attempt to revert sender balance if recipient update fails
		_ = db.SaveAccountBalance(from, fromBalance)
		return fmt.Errorf("failed to update recipient balance: %w", err)
	}

	return nil
}

// Helper function to get keys by prefix
func (db *LevelDB) getKeysByPrefix(prefix string) ([]string, error) {
	log.Printf("getKeysByPrefix: searching with prefix '%s'", prefix)
	var keys []string
	iter := db.Client.NewIterator(util.BytesPrefix([]byte(prefix)), nil)
	defer iter.Release()

	for iter.Next() {
		key := string(iter.Key())
		// Verify key has minimum required segments
		parts := strings.Split(key, ":")
		if len(parts) < 4 {
			continue // Skip malformed keys
		}
		keys = append(keys, key) // Return full key
	}

	if err := iter.Error(); err != nil {
		log.Printf("getKeysByPrefix: iterator error for prefix '%s': %v", prefix, err)
		return nil, fmt.Errorf("iterator error: %w", err)
	}

	log.Printf("getKeysByPrefix: found %d keys for prefix '%s'", len(keys), prefix)
	return keys, nil
}

func (db *LevelDB) PutIntoDb(blockchain interface{}, address string) error { // this method receives implementations to work with data types to do saving using interfaces on workflow steps implementation, during runtime where data persists for software validation for the required logic and parameters for the tests workflow with those project related requirements when testing an application under methods executions when methods calls also use data from structs to save in `leveldb`.
	data, err := json.Marshal(blockchain) // validates if those types can marshal during method executions before it can save the new state object information into persistence leveldb database when calling during that workflow as a local workflow of object states and their properties
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// Call the new PutBytes function
	return db.PutBytes(address, data)
}
func (db *LevelDB) Close() error {
	return db.Client.Close()
}

// Path returns the filesystem path of the database
func (db *LevelDB) Path() string {
	return db.path
}

// GetKeysWithPrefix returns all keys with a specific prefix
// CapabilityExists checks if a capability exists in the database
func (db *LevelDB) CapabilityExists(capabilityID string) (bool, error) {
	key := fmt.Sprintf("capability:%s", capabilityID)
	_, err := db.Client.Get([]byte(key), &opt.ReadOptions{})
	if err != nil {
		if err == leveldb.ErrNotFound {
			return false, nil
		}
		return false, fmt.Errorf("failed to check capability existence: %w", err)
	}
	return true, nil
}

func (db *LevelDB) GetKeysWithPrefix(prefix string) ([]string, error) {
	var keys []string
	iter := db.Client.NewIterator(util.BytesPrefix([]byte(prefix)), nil)
	defer iter.Release()
	for iter.Next() {
		keys = append(keys, string(iter.Key()))
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}
	return keys, nil
}

// Capability Methods

// getCapabilityKey returns a consistent key format for capability storage
func getCapabilityKey(id string) []byte {
	key := fmt.Sprintf("mcp:capability:%s", id)
	log.Printf("[getCapabilityKey] Generated key: %s", key)
	return []byte(key)
}

func (db *LevelDB) GetCapabilityByID(capabilityID string) (*pb.CapabilityDescriptorContainerProto, error) {
	// Try with the new key format first
	key := getCapabilityKey(capabilityID)
	log.Printf("[GetCapabilityByID] Attempting to retrieve capability with key: %s (Hex: %x)", string(key), key)
	data, err := db.Client.Get(key, nil)

	// If not found with new format, try with the old format
	if err != nil && err == leveldb.ErrNotFound {
		data, err = db.GetBytes(KeyPrefixCapability + capabilityID)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				return nil, fmt.Errorf("capability with ID %s not found in database", capabilityID)
			}
			return nil, fmt.Errorf("database error while retrieving capability %s: %w", capabilityID, err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("database error while retrieving capability %s: %w", capabilityID, err)
	}

	var cap pb.CapabilityDescriptorContainerProto
	if err := proto.Unmarshal(data, &cap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal capability: %w", err)
	}
	return &cap, nil
}

func (db *LevelDB) GetCapabilitiesByType(capType string) ([]*pb.CapabilityDescriptorContainerProto, error) {
	keys, err := db.getKeysByPrefix(KeyPrefixCapabilityTypeIdx + capType)
	if err != nil {
		return nil, fmt.Errorf("failed to get capability keys by type: %w", err)
	}

	var caps []*pb.CapabilityDescriptorContainerProto
	for _, key := range keys {
		cap, err := db.GetCapabilityByID(strings.TrimPrefix(key, KeyPrefixCapabilityTypeIdx))
		if err != nil {
			return nil, fmt.Errorf("failed to get capability: %w", err)
		}
		caps = append(caps, cap)
	}
	return caps, nil
}

func (db *LevelDB) GetCapabilitiesByOwner(owner string) ([]*pb.CapabilityDescriptorContainerProto, error) {
	keys, err := db.getKeysByPrefix(KeyPrefixCapabilityOwnerIdx + owner)
	if err != nil {
		return nil, fmt.Errorf("failed to get capability keys by owner: %w", err)
	}

	var caps []*pb.CapabilityDescriptorContainerProto
	for _, key := range keys {
		cap, err := db.GetCapabilityByID(strings.TrimPrefix(key, KeyPrefixCapabilityOwnerIdx))
		if err != nil {
			return nil, fmt.Errorf("failed to get capability: %w", err)
		}
		caps = append(caps, cap)
	}
	return caps, nil
}

// Context Record Methods

func (db *LevelDB) GetContextRecord(contextID string) (interface{}, error) { // contextID is the transaction hash
	key := KeyPrefixContext + contextID
	log.Printf("[LevelDB.GetContextRecord] Attempting to get context record with key: %s (Hex: %x)", key, []byte(key))
	data, err := db.GetBytes(key) // GetBytes already prefixes with KeyPrefixContext if key doesn't have it. Let's be explicit.
	if err != nil {
		log.Printf("[LevelDB.GetContextRecord] Failed to get data for key %s: %v", key, err)
		return nil, fmt.Errorf("failed to get context record with key %s: %w", key, err)
	}

	var ctx pb.ContextRecordProto
	if err := proto.Unmarshal(data, &ctx); err != nil {
		log.Printf("[LevelDB.GetContextRecord] Failed to unmarshal context record for key %s: %v", key, err)
		return nil, fmt.Errorf("failed to unmarshal context record: %w", err)
	}
	return &ctx, nil
}

func (db *LevelDB) GetContextRecordsForCapability(capabilityID string) ([]*pb.ContextRecordProto, error) {
	keys, err := db.getKeysByPrefix(KeyPrefixContextCapabilityIdx + capabilityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get context keys for capability: %w", err)
	}

	var contexts []*pb.ContextRecordProto
	for _, key := range keys {
		ctxInterface, err := db.GetContextRecord(strings.TrimPrefix(key, KeyPrefixContextCapabilityIdx))
		if err != nil {
			return nil, fmt.Errorf("failed to get context record: %w", err)
		}
		if ctx, ok := ctxInterface.(*pb.ContextRecordProto); ok {
			contexts = append(contexts, ctx)
		}
	}
	return contexts, nil
}

func (db *LevelDB) GetAllContextRecords() ([]*pb.ContextRecordProto, error) {
	keys, err := db.getKeysByPrefix(KeyPrefixContext)
	if err != nil {
		return nil, fmt.Errorf("failed to get all context keys: %w", err)
	}

	var contexts []*pb.ContextRecordProto
	for _, key := range keys {
		ctxInterface, err := db.GetContextRecord(strings.TrimPrefix(key, KeyPrefixContext))
		if err != nil {
			return nil, fmt.Errorf("failed to get context record: %w", err)
		}
		if ctx, ok := ctxInterface.(*pb.ContextRecordProto); ok {
			contexts = append(contexts, ctx)
		}
	}
	return contexts, nil
}

// SaveContextRecord saves a context record to the database
func (db *LevelDB) SaveContextRecord(record types.ContextRecord) error {
	// Convert the ContextRecord to ContextRecordProto
	log.Printf("Converting ContextRecord to proto: %+v", record)
	log.Printf("[LevelDB.SaveContextRecord] Converting ContextRecord to proto: ID=%s, CapID=%s, Initiator=%s", record.ID, record.CapabilityID, record.Initiator)
	protoRecord := &pb.ContextRecordProto{
		Id:              record.ID,
		CapabilityId:    record.CapabilityID,
		InteractionType: pb.InteractionTypeProto(getInteractionTypeProtoValue(string(record.InteractionType))),
		Initiator:       record.Initiator,
		InputHash:       record.InputHash,
		OutputHash:      record.OutputHash,
		// Convert timestamp from Unix to protobuf timestamp
		Timestamp: &timestamppb.Timestamp{
			Seconds: record.Timestamp,
			Nanos:   0,
		},
		Signature: record.Signature,
	}
	log.Printf("Converted proto record: %+v", protoRecord)
	// log.Printf("Converted proto record: %+v", protoRecord) // Can be very verbose

	// Convert Details map to structpb.Struct if it exists
	if record.Details != nil {
		detailsStruct, err := structpb.NewStruct(record.Details)
		if err != nil {
			return fmt.Errorf("failed to convert details to proto struct: %w", err)
		}
		protoRecord.Details = detailsStruct
	}

	// Marshal the proto record
	data, err := proto.Marshal(protoRecord)
	if err != nil {
		log.Printf("Failed to marshal proto record: %v", err)
		log.Printf("[LevelDB.SaveContextRecord] Failed to marshal proto record for ID %s: %v", record.ID, err)
		return fmt.Errorf("failed to marshal context record: %w", err)
	}
	log.Printf("Successfully marshaled proto record (%d bytes)", len(data))
	log.Printf("[LevelDB.SaveContextRecord] Successfully marshaled proto record for ID %s (%d bytes)", record.ID, len(data))

	// Save the record
	key := KeyPrefixContext + record.ID
	log.Printf("Saving context record with key: %s", key)
	log.Printf("[LevelDB.SaveContextRecord] Saving context record with key: %s (Hex: %x)", key, []byte(key))
	err = db.PutBytes(key, data)
	if err != nil {
		log.Printf("Failed to save context record: %v", err)
		log.Printf("[LevelDB.SaveContextRecord] Failed to save context record with key %s: %v", key, err)
		return fmt.Errorf("failed to save context record: %w", err)
	}
	log.Printf("Successfully saved context record with key: %s", key)
	log.Printf("[LevelDB.SaveContextRecord] Successfully saved context record with key: %s", key)

	// Create index for capability ID
	err = db.PutBytes(KeyPrefixContextCapabilityIdx+record.CapabilityID+":"+record.ID, []byte{1})
	if err != nil {
		return fmt.Errorf("failed to create capability index for context record: %w", err)
	}

	// Create index for interaction type
	err = db.PutBytes(KeyPrefixContextInteractionTypeIdx+string(record.InteractionType)+":"+record.ID, []byte{1})
	if err != nil {
		return fmt.Errorf("failed to create interaction type index for context record: %w", err)
	}

	// Create index for initiator
	err = db.PutBytes(KeyPrefixContextInitiatorIdx+record.Initiator+":"+record.ID, []byte{1})
	if err != nil {
		return fmt.Errorf("failed to create initiator index for context record: %w", err)
	}

	return nil
}

// Helper function to convert InteractionType string to InteractionTypeProto value
func getInteractionTypeProtoValue(interactionType string) int32 {
	switch interactionType {
	case string(types.InteractionTypeToolInvocation):
		return int32(pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_TOOL_INVOCATION)
	case string(types.InteractionTypePromptUsage):
		return int32(pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_PROMPT_USAGE)
	case string(types.InteractionTypeResourceAccess):
		return int32(pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_RESOURCE_ACCESS)
	case string(types.InteractionTypePluginExecution):
		return int32(pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_PLUGIN_EXECUTION)
	case string(types.InteractionTypeSamplingRequestSent):
		return int32(pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_SAMPLING_REQUEST_SENT)
	case string(types.InteractionTypeSamplingResponseReceived):
		return int32(pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_SAMPLING_RESPONSE_RECEIVED)
	case string(types.InteractionTypeMemoryWrite):
		return int32(pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_MEMORY_WRITE)
	case string(types.InteractionTypeCapabilityRegistration):
		return int32(pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_CAPABILITY_REGISTRATION)
	default:
		return int32(pb.InteractionTypeProto_INTERACTION_TYPE_PROTO_UNSPECIFIED)
	}
}

// SaveCapability saves a capability to the database with enhanced persistence verification
func (db *LevelDB) SaveCapability(capability interface{}) error {
	startTime := time.Now()
	log.Printf("[SaveCapability] Starting capability save operation")
	log.Printf("[SaveCapability] Input capability: %+v", capability)

	// Convert the capability to a proto container
	if db.converter == nil {
		return fmt.Errorf("capability converter not set")
	}
	capProto, err := db.converter.ConvertToCapabilityDescriptorContainerProto(capability)
	if err != nil {
		log.Printf("[SaveCapability] Failed to convert capability to proto: %v", err)
		return fmt.Errorf("failed to convert capability to proto: %w", err)
	}

	// Get the capability ID from the container
	var capabilityID string
	var capabilityType string
	var owner string

	switch x := capProto.GetDescriptor_().(type) {
	case *pb.CapabilityDescriptorContainerProto_Resource:
		capabilityID = x.Resource.BaseDescriptor.Id
		capabilityType = "RESOURCE"
		owner = x.Resource.BaseDescriptor.Owner
		log.Printf("[SaveCapability] Saving RESOURCE capability: ID=%s, Owner=%s", capabilityID, owner)
	case *pb.CapabilityDescriptorContainerProto_Tool:
		capabilityID = x.Tool.BaseDescriptor.Id
		capabilityType = "TOOL"
		owner = x.Tool.BaseDescriptor.Owner
		log.Printf("[SaveCapability] Saving TOOL capability: ID=%s, Owner=%s", capabilityID, owner)
	case *pb.CapabilityDescriptorContainerProto_Prompt:
		capabilityID = x.Prompt.BaseDescriptor.Id
		capabilityType = "PROMPT"
		owner = x.Prompt.BaseDescriptor.Owner
		log.Printf("[SaveCapability] Saving PROMPT capability: ID=%s, Owner=%s", capabilityID, owner)
	case *pb.CapabilityDescriptorContainerProto_MemoryService:
		capabilityID = x.MemoryService.BaseDescriptor.Id
		capabilityType = "MEMORY_SERVICE"
		owner = x.MemoryService.BaseDescriptor.Owner
		log.Printf("[SaveCapability] Saving MEMORY_SERVICE capability: ID=%s, Owner=%s", capabilityID, owner)
	default:
		err := fmt.Errorf("unsupported capability type")
		log.Printf("[SaveCapability] %v", err)
		return err
	}

	// Marshal the proto to bytes
	data, err := proto.Marshal(capProto)
	if err != nil {
		log.Printf("[SaveCapability] Failed to marshal capability proto: %v", err)
		return fmt.Errorf("failed to marshal capability proto: %w", err)
	}

	// Save the capability using the new key format
	key := string(getCapabilityKey(capabilityID))
	log.Printf("[SaveCapability] Saving capability with key: %s", key)
	err = db.PutBytes(key, data)
	if err != nil {
		log.Printf("[SaveCapability] Failed to save capability with new key format: %v", err)
		return fmt.Errorf("failed to save capability: %w", err)
	}

	// Also save with the old key format for backward compatibility
	oldKey := KeyPrefixCapability + capabilityID
	log.Printf("[SaveCapability] Saving capability with old key format: %s", oldKey)
	err = db.PutBytes(oldKey, data)
	if err != nil {
		log.Printf("[SaveCapability] Warning: failed to save capability with old key format: %v", err)
		// Continue even if old format save fails
	}

	// Force LevelDB sync and compaction
	log.Printf("[SaveCapability] Starting compaction")
	compactionStart := time.Now()
	if err := db.Client.CompactRange(util.Range{}); err != nil {
		log.Printf("[SaveCapability] Warning: compaction error after capability save: %v", err)
	}
	log.Printf("[SaveCapability] Compaction completed in %v", time.Since(compactionStart))

	// Enhanced verification with more retries and better error reporting
	var verificationErrors []string
	var verified bool

	// Verify with new key format first
	log.Printf("[SaveCapability] Verifying new key format write")
	for i := 0; i < 5; i++ { // Increased retries from 3 to 5
		_, err = db.Client.Get([]byte(key), nil)
		if err == nil {
			verified = true
			break
		}
		verificationErrors = append(verificationErrors, fmt.Sprintf("Attempt %d (new format): %v", i+1, err))
		time.Sleep(20 * time.Millisecond) // Increased delay from 10ms to 20ms
	}

	// If new format verification failed, try old format
	if !verified {
		log.Printf("[SaveCapability] New key format verification failed, trying old format")
		for i := 0; i < 5; i++ {
			_, err = db.Client.Get([]byte(oldKey), nil)
			if err == nil {
				log.Printf("[SaveCapability] Capability verified with old key format: %s", oldKey)
				verified = true
				break
			}
			verificationErrors = append(verificationErrors, fmt.Sprintf("Attempt %d (old format): %v", i+1, err))
			time.Sleep(20 * time.Millisecond)
		}
	}

	if !verified {
		log.Printf("[SaveCapability] Failed to verify capability write after all attempts: %v", verificationErrors)
		return fmt.Errorf("failed to verify capability write after 5 attempts each for new and old formats. Errors: %v", verificationErrors)
	}

	// Create indexes with verification
	log.Printf("[SaveCapability] Creating type index")
	if err := db.createAndVerifyIndex(KeyPrefixCapabilityTypeIdx + capabilityType + ":" + capabilityID); err != nil {
		return fmt.Errorf("failed to create type index for capability: %w", err)
	}

	log.Printf("[SaveCapability] Creating owner index")
	if err := db.createAndVerifyIndex(KeyPrefixCapabilityOwnerIdx + owner + ":" + capabilityID); err != nil {
		return fmt.Errorf("failed to create owner index for capability: %w", err)
	}

	log.Printf("[SaveCapability] Capability saved and verified successfully in %v", time.Since(startTime))
	return nil
}

// createAndVerifyIndex creates an index and verifies it was written
func (db *LevelDB) createAndVerifyIndex(key string) error {
	// Write the index
	if err := db.PutBytes(key, []byte{1}); err != nil {
		log.Printf("[createAndVerifyIndex] Failed to create index %s: %v", key, err)
		return err
	}

	// Verify the index was written
	for i := 0; i < 3; i++ {
		_, err := db.Client.Get([]byte(key), nil)
		if err == nil {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return fmt.Errorf("failed to verify index creation for key %s", key)
}

// UpdateCapability updates an existing capability in the database with enhanced verification
func (db *LevelDB) UpdateCapability(capabilityID string, updates interface{}) error {
	startTime := time.Now()
	log.Printf("[UpdateCapability] Starting update for capability ID: %s", capabilityID)

	// Get existing capability with verification
	log.Printf("[UpdateCapability] Retrieving existing capability")
	existingCap, err := db.GetCapabilityByID(capabilityID)
	if err != nil {
		if err == leveldb.ErrNotFound {
			log.Printf("[UpdateCapability] Capability not found: %s", capabilityID)
			return fmt.Errorf("capability %s not found - cannot update non-existent capability", capabilityID)
		}
		log.Printf("[UpdateCapability] Failed to get existing capability: %v", err)
		return fmt.Errorf("failed to get existing capability: %w", err)
	}

	// Convert updates to proto
	log.Printf("[UpdateCapability] Converting updates to proto")
	if db.converter == nil {
		return fmt.Errorf("capability converter not set")
	}
	updatesProto, err := db.converter.ConvertToCapabilityDescriptorContainerProto(updates)
	if err != nil {
		log.Printf("[UpdateCapability] Failed to convert updates: %v", err)
		return fmt.Errorf("failed to convert updates to proto: %w", err)
	}

	// Verify capability types match
	if existingCap.GetDescriptor_() == nil || updatesProto.GetDescriptor_() == nil {
		log.Printf("[UpdateCapability] Invalid capability descriptor types")
		return fmt.Errorf("invalid capability descriptor types")
	}

	// Get concrete types without using reflect
	existingType := ""
	switch existingCap.GetDescriptor_().(type) {
	case *pb.CapabilityDescriptorContainerProto_Resource:
		existingType = "Resource"
	case *pb.CapabilityDescriptorContainerProto_Tool:
		existingType = "Tool"
	case *pb.CapabilityDescriptorContainerProto_Prompt:
		existingType = "Prompt"
	case *pb.CapabilityDescriptorContainerProto_MemoryService:
		existingType = "MemoryService"
	}

	updateType := ""
	switch updatesProto.GetDescriptor_().(type) {
	case *pb.CapabilityDescriptorContainerProto_Resource:
		updateType = "Resource"
	case *pb.CapabilityDescriptorContainerProto_Tool:
		updateType = "Tool"
	case *pb.CapabilityDescriptorContainerProto_Prompt:
		updateType = "Prompt"
	case *pb.CapabilityDescriptorContainerProto_MemoryService:
		updateType = "MemoryService"
	}

	if existingType != updateType {
		log.Printf("[UpdateCapability] Capability type mismatch: existing=%s, update=%s", existingType, updateType)
		return fmt.Errorf("cannot update capability of type %s with type %s", existingType, updateType)
	}

	// Merge updates with existing capability
	log.Printf("[UpdateCapability] Merging capability updates")
	mergedCap, err := mergeCapabilityDescriptors(existingCap, updatesProto)
	if err != nil {
		log.Printf("[UpdateCapability] Failed to merge updates: %v", err)
		return fmt.Errorf("failed to merge capability updates: %w", err)
	}

	// Convert merged proto back to Go struct for SaveCapability
	log.Printf("[UpdateCapability] Converting merged proto back to Go struct")
	mergedGoStruct, err := convertProtoToGoStruct(mergedCap)
	if err != nil {
		log.Printf("[UpdateCapability] Failed to convert proto to Go struct: %v", err)
		return fmt.Errorf("failed to convert merged capability to Go struct: %w", err)
	}

	// Save merged capability with verification
	log.Printf("[UpdateCapability] Saving merged capability")
	if err := db.SaveCapability(mergedGoStruct); err != nil {
		log.Printf("[UpdateCapability] Failed to save merged capability: %v", err)
		return fmt.Errorf("failed to save merged capability: %w", err)
	}

	// Verify the update was applied
	log.Printf("[UpdateCapability] Verifying update")
	updatedCapProto, err := db.GetCapabilityByID(capabilityID)
	if err != nil {
		log.Printf("[UpdateCapability] Failed to verify update: %v", err)
		return fmt.Errorf("failed to verify capability update: %w", err)
	}

	// Compare the updated capability with our merged version (both are proto types)
	if !proto.Equal(mergedCap, updatedCapProto) {
		log.Printf("[UpdateCapability] Update verification failed - stored capability doesn't match expected state")
		return fmt.Errorf("capability update verification failed - stored state doesn't match expected updates")
	}

	log.Printf("[UpdateCapability] Successfully updated capability %s in %v", capabilityID, time.Since(startTime))
	return nil
}

// mergeCapabilityDescriptors merges updates into an existing capability with detailed logging
func mergeCapabilityDescriptors(existing, updates *pb.CapabilityDescriptorContainerProto) (*pb.CapabilityDescriptorContainerProto, error) {
	// Create deep copy of existing capability
	merged := proto.Clone(existing).(*pb.CapabilityDescriptorContainerProto)
	log.Printf("[mergeCapabilityDescriptors] Starting merge operation")

	// Handle different capability types
	switch x := merged.GetDescriptor_().(type) {
	case *pb.CapabilityDescriptorContainerProto_Resource:
		updatesResource := updates.GetResource()
		if updatesResource == nil {
			err := fmt.Errorf("update type mismatch - expected Resource")
			log.Printf("[mergeCapabilityDescriptors] %v", err)
			return nil, err
		}

		// Log original values
		original := existing.GetResource().BaseDescriptor
		log.Printf("[mergeCapabilityDescriptors] Original Resource Capability - Name: %s, Version: %s, Desc: %s, GasFee: %d",
			original.Name, original.Version, original.Description, original.GasFeeNrn)

		// Merge base descriptor fields
		if updatesResource.BaseDescriptor != nil {
			if updatesResource.BaseDescriptor.Name != "" {
				log.Printf("[mergeCapabilityDescriptors] Updating name: %s -> %s", x.Resource.BaseDescriptor.Name, updatesResource.BaseDescriptor.Name)
				x.Resource.BaseDescriptor.Name = updatesResource.BaseDescriptor.Name
			}
			if updatesResource.BaseDescriptor.Version != "" {
				log.Printf("[mergeCapabilityDescriptors] Updating version: %s -> %s", x.Resource.BaseDescriptor.Version, updatesResource.BaseDescriptor.Version)
				x.Resource.BaseDescriptor.Version = updatesResource.BaseDescriptor.Version
			}
			if updatesResource.BaseDescriptor.Description != "" {
				log.Printf("[mergeCapabilityDescriptors] Updating description: %s -> %s", x.Resource.BaseDescriptor.Description, updatesResource.BaseDescriptor.Description)
				x.Resource.BaseDescriptor.Description = updatesResource.BaseDescriptor.Description
			}
			if updatesResource.BaseDescriptor.GasFeeNrn != 0 {
				log.Printf("[mergeCapabilityDescriptors] Updating gas fee: %d -> %d", x.Resource.BaseDescriptor.GasFeeNrn, updatesResource.BaseDescriptor.GasFeeNrn)
				x.Resource.BaseDescriptor.GasFeeNrn = updatesResource.BaseDescriptor.GasFeeNrn
			}
			if updatesResource.BaseDescriptor.CustomMetadata != nil {
				log.Printf("[mergeCapabilityDescriptors] Updating custom metadata")
				if x.Resource.BaseDescriptor.CustomMetadata == nil {
					x.Resource.BaseDescriptor.CustomMetadata = &structpb.Struct{Fields: make(map[string]*structpb.Value)}
				}
				// Merge custom metadata fields
				for key, value := range updatesResource.BaseDescriptor.CustomMetadata.Fields {
					log.Printf("[mergeCapabilityDescriptors] Updating custom metadata field: %s", key)
					x.Resource.BaseDescriptor.CustomMetadata.Fields[key] = value
				}
			}
		} else {
			log.Printf("[mergeCapabilityDescriptors] WARNING: updatesResource.BaseDescriptor is nil!")
		}

	case *pb.CapabilityDescriptorContainerProto_Tool:
		updatesTool := updates.GetTool()
		if updatesTool == nil {
			err := fmt.Errorf("update type mismatch - expected Tool")
			log.Printf("[mergeCapabilityDescriptors] %v", err)
			return nil, err
		}

		// Log original values
		original := existing.GetTool().BaseDescriptor
		log.Printf("[mergeCapabilityDescriptors] Original Tool Capability - Name: %s, Version: %s, Desc: %s, GasFee: %d",
			original.Name, original.Version, original.Description, original.GasFeeNrn)

		// Merge base descriptor fields
		if updatesTool.BaseDescriptor != nil {
			if updatesTool.BaseDescriptor.Name != "" {
				log.Printf("[mergeCapabilityDescriptors] Updating name: %s -> %s", x.Tool.BaseDescriptor.Name, updatesTool.BaseDescriptor.Name)
				x.Tool.BaseDescriptor.Name = updatesTool.BaseDescriptor.Name
			}
			if updatesTool.BaseDescriptor.Version != "" {
				log.Printf("[mergeCapabilityDescriptors] Updating version: %s -> %s", x.Tool.BaseDescriptor.Version, updatesTool.BaseDescriptor.Version)
				x.Tool.BaseDescriptor.Version = updatesTool.BaseDescriptor.Version
			}
			if updatesTool.BaseDescriptor.Description != "" {
				log.Printf("[mergeCapabilityDescriptors] Updating description: %s -> %s", x.Tool.BaseDescriptor.Description, updatesTool.BaseDescriptor.Description)
				x.Tool.BaseDescriptor.Description = updatesTool.BaseDescriptor.Description
			}
			if updatesTool.BaseDescriptor.GasFeeNrn != 0 {
				log.Printf("[mergeCapabilityDescriptors] Updating gas fee: %d -> %d", x.Tool.BaseDescriptor.GasFeeNrn, updatesTool.BaseDescriptor.GasFeeNrn)
				x.Tool.BaseDescriptor.GasFeeNrn = updatesTool.BaseDescriptor.GasFeeNrn
			}
			if updatesTool.BaseDescriptor.CustomMetadata != nil {
				log.Printf("[mergeCapabilityDescriptors] Updating custom metadata")
				if x.Tool.BaseDescriptor.CustomMetadata == nil {
					x.Tool.BaseDescriptor.CustomMetadata = &structpb.Struct{Fields: make(map[string]*structpb.Value)}
				}
				// Merge custom metadata fields
				for key, value := range updatesTool.BaseDescriptor.CustomMetadata.Fields {
					log.Printf("[mergeCapabilityDescriptors] Updating custom metadata field: %s", key)
					x.Tool.BaseDescriptor.CustomMetadata.Fields[key] = value
				}
			}
		}

	case *pb.CapabilityDescriptorContainerProto_Prompt:
		updatesPrompt := updates.GetPrompt()
		if updatesPrompt == nil {
			err := fmt.Errorf("update type mismatch - expected Prompt")
			log.Printf("[mergeCapabilityDescriptors] %v", err)
			return nil, err
		}

		// Log original values
		original := existing.GetPrompt().BaseDescriptor
		log.Printf("[mergeCapabilityDescriptors] Original Prompt Capability - Name: %s, Version: %s, Desc: %s, GasFee: %d",
			original.Name, original.Version, original.Description, original.GasFeeNrn)

		// Merge base descriptor fields
		if updatesPrompt.BaseDescriptor != nil {
			if updatesPrompt.BaseDescriptor.Name != "" {
				log.Printf("[mergeCapabilityDescriptors] Updating name: %s -> %s", x.Prompt.BaseDescriptor.Name, updatesPrompt.BaseDescriptor.Name)
				x.Prompt.BaseDescriptor.Name = updatesPrompt.BaseDescriptor.Name
			}
			if updatesPrompt.BaseDescriptor.Version != "" {
				log.Printf("[mergeCapabilityDescriptors] Updating version: %s -> %s", x.Prompt.BaseDescriptor.Version, updatesPrompt.BaseDescriptor.Version)
				x.Prompt.BaseDescriptor.Version = updatesPrompt.BaseDescriptor.Version
			}
			if updatesPrompt.BaseDescriptor.Description != "" {
				log.Printf("[mergeCapabilityDescriptors] Updating description: %s -> %s", x.Prompt.BaseDescriptor.Description, updatesPrompt.BaseDescriptor.Description)
				x.Prompt.BaseDescriptor.Description = updatesPrompt.BaseDescriptor.Description
			}
			if updatesPrompt.BaseDescriptor.GasFeeNrn != 0 {
				log.Printf("[mergeCapabilityDescriptors] Updating gas fee: %d -> %d", x.Prompt.BaseDescriptor.GasFeeNrn, updatesPrompt.BaseDescriptor.GasFeeNrn)
				x.Prompt.BaseDescriptor.GasFeeNrn = updatesPrompt.BaseDescriptor.GasFeeNrn
			}
			if updatesPrompt.BaseDescriptor.CustomMetadata != nil {
				log.Printf("[mergeCapabilityDescriptors] Updating custom metadata")
				if x.Prompt.BaseDescriptor.CustomMetadata == nil {
					x.Prompt.BaseDescriptor.CustomMetadata = &structpb.Struct{Fields: make(map[string]*structpb.Value)}
				}
				// Merge custom metadata fields
				for key, value := range updatesPrompt.BaseDescriptor.CustomMetadata.Fields {
					log.Printf("[mergeCapabilityDescriptors] Updating custom metadata field: %s", key)
					x.Prompt.BaseDescriptor.CustomMetadata.Fields[key] = value
				}
			}
		}

	case *pb.CapabilityDescriptorContainerProto_MemoryService:
		updatesMemory := updates.GetMemoryService()
		if updatesMemory == nil {
			err := fmt.Errorf("update type mismatch - expected MemoryService")
			log.Printf("[mergeCapabilityDescriptors] %v", err)
			return nil, err
		}

		// Log original values
		original := existing.GetMemoryService().BaseDescriptor
		log.Printf("[mergeCapabilityDescriptors] Original MemoryService Capability - Name: %s, Version: %s, Desc: %s, GasFee: %d",
			original.Name, original.Version, original.Description, original.GasFeeNrn)

		// Merge base descriptor fields
		if updatesMemory.BaseDescriptor != nil {
			if updatesMemory.BaseDescriptor.Name != "" {
				log.Printf("[mergeCapabilityDescriptors] Updating name: %s -> %s", x.MemoryService.BaseDescriptor.Name, updatesMemory.BaseDescriptor.Name)
				x.MemoryService.BaseDescriptor.Name = updatesMemory.BaseDescriptor.Name
			}
			if updatesMemory.BaseDescriptor.Version != "" {
				log.Printf("[mergeCapabilityDescriptors] Updating version: %s -> %s", x.MemoryService.BaseDescriptor.Version, updatesMemory.BaseDescriptor.Version)
				x.MemoryService.BaseDescriptor.Version = updatesMemory.BaseDescriptor.Version
			}
			if updatesMemory.BaseDescriptor.Description != "" {
				log.Printf("[mergeCapabilityDescriptors] Updating description: %s -> %s", x.MemoryService.BaseDescriptor.Description, updatesMemory.BaseDescriptor.Description)
				x.MemoryService.BaseDescriptor.Description = updatesMemory.BaseDescriptor.Description
			}
			if updatesMemory.BaseDescriptor.GasFeeNrn != 0 {
				log.Printf("[mergeCapabilityDescriptors] Updating gas fee: %d -> %d", x.MemoryService.BaseDescriptor.GasFeeNrn, updatesMemory.BaseDescriptor.GasFeeNrn)
				x.MemoryService.BaseDescriptor.GasFeeNrn = updatesMemory.BaseDescriptor.GasFeeNrn
			}
			if updatesMemory.BaseDescriptor.CustomMetadata != nil {
				log.Printf("[mergeCapabilityDescriptors] Updating custom metadata")
				if x.MemoryService.BaseDescriptor.CustomMetadata == nil {
					x.MemoryService.BaseDescriptor.CustomMetadata = &structpb.Struct{Fields: make(map[string]*structpb.Value)}
				}
				// Merge custom metadata fields
				for key, value := range updatesMemory.BaseDescriptor.CustomMetadata.Fields {
					log.Printf("[mergeCapabilityDescriptors] Updating custom metadata field: %s", key)
					x.MemoryService.BaseDescriptor.CustomMetadata.Fields[key] = value
				}
			}
		}

	default:
		err := fmt.Errorf("unsupported capability type for merge")
		log.Printf("[mergeCapabilityDescriptors] %v", err)
		return nil, err
	}

	log.Printf("[mergeCapabilityDescriptors] Merge completed successfully")
	return merged, nil
}

// convertProtoToGoStruct converts a proto capability descriptor back to a Go struct
func convertProtoToGoStruct(protoDesc *pb.CapabilityDescriptorContainerProto) (interface{}, error) {
	switch desc := protoDesc.GetDescriptor_().(type) {
	case *pb.CapabilityDescriptorContainerProto_Resource:
		resource := desc.Resource
		baseDesc := resource.GetBaseDescriptor()

		// Convert custom metadata from proto struct to map
		customMetadata := make(map[string]interface{})
		if baseDesc.GetCustomMetadata() != nil {
			for key, value := range baseDesc.GetCustomMetadata().GetFields() {
				customMetadata[key] = value.AsInterface()
			}
		}

		// Convert location hints from repeated string to slice
		var locationHints []string
		if resource.GetSchema() != nil {
			locationHints = resource.GetSchema().GetLocationHints()
		}

		// Convert proto resource type to Go enum
		var resourceType types.DiscoveryResourceType
		switch resource.GetResourceType() {
		case pb.DiscoveryResourceTypeProto_DISCOVERY_RESOURCE_TYPE_PROTO_FILE:
			resourceType = types.DiscoveryResourceTypeFile
		case pb.DiscoveryResourceTypeProto_DISCOVERY_RESOURCE_TYPE_PROTO_API:
			resourceType = types.DiscoveryResourceTypeAPI
		case pb.DiscoveryResourceTypeProto_DISCOVERY_RESOURCE_TYPE_PROTO_PLUGIN:
			resourceType = types.DiscoveryResourceTypePlugin
		case pb.DiscoveryResourceTypeProto_DISCOVERY_RESOURCE_TYPE_PROTO_GENERATED_DOCUMENT:
			resourceType = types.DiscoveryResourceTypeGeneratedDoc
		case pb.DiscoveryResourceTypeProto_DISCOVERY_RESOURCE_TYPE_PROTO_DATASET:
			resourceType = types.DiscoveryResourceTypeDataset
		case pb.DiscoveryResourceTypeProto_DISCOVERY_RESOURCE_TYPE_PROTO_MODEL_ARTIFACT:
			resourceType = types.DiscoveryResourceTypeModelArtifact
		case pb.DiscoveryResourceTypeProto_DISCOVERY_RESOURCE_TYPE_PROTO_SERVICE:
			resourceType = types.DiscoveryResourceTypeService
		default:
			resourceType = types.DiscoveryResourceTypeFile // Default fallback
		}

		return types.ResourceDescriptor{
			BaseDescriptor: types.BaseDescriptor{
				ID:             baseDesc.GetId(),
				CapabilityType: types.CapabilityTypeResource, // Convert from proto enum to Go enum
				Name:           baseDesc.GetName(),
				Owner:          baseDesc.GetOwner(),
				Version:        baseDesc.GetVersion(),
				Description:    baseDesc.GetDescription(),
				GasFeeNRN:      baseDesc.GetGasFeeNrn(),
				Timestamp:      baseDesc.GetTimestamp().GetSeconds(),
				CustomMetadata: customMetadata,
			},
			ResourceType: resourceType,
			ContentHash:  resource.GetContentHash(),
			Schema: types.PluginSchemaDetail{
				Summary:       resource.GetSchema().GetSummary(),
				LocationHints: locationHints,
			},
		}, nil

	case *pb.CapabilityDescriptorContainerProto_Tool:
		tool := desc.Tool
		baseDesc := tool.GetBaseDescriptor()

		// Convert custom metadata from proto struct to map
		customMetadata := make(map[string]interface{})
		if baseDesc.GetCustomMetadata() != nil {
			for key, value := range baseDesc.GetCustomMetadata().GetFields() {
				customMetadata[key] = value.AsInterface()
			}
		}

		return types.ToolDescriptor{
			BaseDescriptor: types.BaseDescriptor{
				ID:             baseDesc.GetId(),
				CapabilityType: types.CapabilityTypeTool, // Convert from proto enum to Go enum
				Name:           baseDesc.GetName(),
				Owner:          baseDesc.GetOwner(),
				Version:        baseDesc.GetVersion(),
				Description:    baseDesc.GetDescription(),
				GasFeeNRN:      baseDesc.GetGasFeeNrn(),
				Timestamp:      baseDesc.GetTimestamp().GetSeconds(),
				CustomMetadata: customMetadata,
			},
			ExecutionPointer: tool.GetExecutionPointer(),
			InputSchemaJSON:  tool.GetInputSchemaJson(),
			OutputSchemaJSON: tool.GetOutputSchemaJson(),
		}, nil

	case *pb.CapabilityDescriptorContainerProto_Prompt:
		prompt := desc.Prompt
		baseDesc := prompt.GetBaseDescriptor()

		// Convert custom metadata from proto struct to map
		customMetadata := make(map[string]interface{})
		if baseDesc.GetCustomMetadata() != nil {
			for key, value := range baseDesc.GetCustomMetadata().GetFields() {
				customMetadata[key] = value.AsInterface()
			}
		}

		return types.PromptDescriptor{
			BaseDescriptor: types.BaseDescriptor{
				ID:             baseDesc.GetId(),
				CapabilityType: types.CapabilityTypePrompt, // Convert from proto enum to Go enum
				Name:           baseDesc.GetName(),
				Owner:          baseDesc.GetOwner(),
				Version:        baseDesc.GetVersion(),
				Description:    baseDesc.GetDescription(),
				GasFeeNRN:      baseDesc.GetGasFeeNrn(),
				Timestamp:      baseDesc.GetTimestamp().GetSeconds(),
				CustomMetadata: customMetadata,
			},
			Template:             prompt.GetTemplate(),
			ParametersSchemaJSON: prompt.GetParametersSchemaJson(),
		}, nil

	case *pb.CapabilityDescriptorContainerProto_MemoryService:
		memory := desc.MemoryService
		baseDesc := memory.GetBaseDescriptor()

		// Convert custom metadata from proto struct to map
		customMetadata := make(map[string]interface{})
		if baseDesc.GetCustomMetadata() != nil {
			for key, value := range baseDesc.GetCustomMetadata().GetFields() {
				customMetadata[key] = value.AsInterface()
			}
		}

		return types.MemoryServiceDescriptor{
			BaseDescriptor: types.BaseDescriptor{
				ID:             baseDesc.GetId(),
				CapabilityType: types.CapabilityTypeMemoryService, // Convert from proto enum to Go enum
				Name:           baseDesc.GetName(),
				Owner:          baseDesc.GetOwner(),
				Version:        baseDesc.GetVersion(),
				Description:    baseDesc.GetDescription(),
				GasFeeNRN:      baseDesc.GetGasFeeNrn(),
				Timestamp:      baseDesc.GetTimestamp().GetSeconds(),
				CustomMetadata: customMetadata,
			},
			GraphSchema: memory.GetGraphSchema(),
		}, nil

	default:
		return nil, fmt.Errorf("unsupported capability type in proto descriptor")
	}
}

// PoAu-D Network Authors Management

// GetNetworkAuthors retrieves the NetworkAuthors map from the database
func (db *LevelDB) GetNetworkAuthors() (map[string]bool, error) {
	data, err := db.GetBytes(NetworkAuthorsKey)
	if err != nil {
		return nil, err
	}

	var authors map[string]bool
	if err := json.Unmarshal(data, &authors); err != nil {
		return nil, fmt.Errorf("failed to unmarshal network authors: %w", err)
	}

	return authors, nil
}

// PutNetworkAuthors saves the NetworkAuthors map to the database
func (db *LevelDB) PutNetworkAuthors(authors map[string]bool) error {
	data, err := json.Marshal(authors)
	if err != nil {
		return fmt.Errorf("failed to marshal network authors: %w", err)
	}

	return db.PutBytes(NetworkAuthorsKey, data)
}

// PoAu-D Configuration Management

// GetPoAuDEnabled retrieves the PoAu-D enabled flag from the database
func (db *LevelDB) GetPoAuDEnabled() (bool, error) {
	data, err := db.GetBytes(PoAuDEnabledKey)
	if err != nil {
		if err == leveldb.ErrNotFound {
			// Default to false if not set
			return false, nil
		}
		return false, err
	}

	var enabled bool
	if err := json.Unmarshal(data, &enabled); err != nil {
		return false, fmt.Errorf("failed to unmarshal PoAu-D enabled flag: %w", err)
	}

	return enabled, nil
}

// PutPoAuDEnabled saves the PoAu-D enabled flag to the database
func (db *LevelDB) PutPoAuDEnabled(enabled bool) error {
	data, err := json.Marshal(enabled)
	if err != nil {
		return fmt.Errorf("failed to marshal PoAu-D enabled flag: %w", err)
	}

	return db.PutBytes(PoAuDEnabledKey, data)
}
