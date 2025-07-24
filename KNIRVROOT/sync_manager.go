package main

import (
	"context"
	"encoding/json"
	"fmt"

	"KNIRVROOT/config"
	"KNIRVROOT/errors"
	pb "KNIRVROOT/proto"
	"KNIRVROOT/types"
	"KNIRVROOT/utils" // Added to access constants
	"log"
	
	"sync"
	"time"

	chromem "github.com/philippgille/chromem-go" // New in-memory client with persistence
	"google.golang.org/protobuf/proto"
)

// ChromemConfig is defined in chromemDB_manager.go

// ChromemSyncManager handles synchronization between LevelDB and ChromemDB
type ChromemSyncManager struct {
	client                         *chromem.DB // Use chromem.DB as the client
	transactionCollection          *chromem.Collection
	contextRecordCollection        *chromem.Collection
	capabilityDescriptorCollection *chromem.Collection
	mu                             sync.RWMutex
	ef                             chromem.EmbeddingFunc    // Embedding function adapter
	cef                            *CerebrasEmbeddingClient // Deterministic Cerebras embedding client
	db                             *LevelDB                 // Add LevelDB client
}

// EmbedFunc is a function that implements the embedding functionality using deterministic embeddings
func EmbedFunc(ctx context.Context, cef *CerebrasEmbeddingClient, text string) ([]float32, error) {
	// Use the deterministic embedding client
	return cef.GenerateEmbedding(ctx, text)
}

// NewChromemSyncManager creates a new ChromemDB sync manager
func NewChromemSyncManager(chromemClient *chromem.DB, cerebrasAPICfg *config.CerebrasConfig, db *LevelDB) (*ChromemSyncManager, error) {
	// Initialize ChromemDB persistent client
	if db == nil {
		return nil, fmt.Errorf("LevelDB client cannot be nil for ChromemSyncManager")
	}
	if chromemClient == nil {
		return nil, fmt.Errorf("ChromemDB client (*chromem.DB) cannot be nil for ChromemSyncManager")
	}

	// Test environment detection based on test flag in config
	isTestEnv := cerebrasAPICfg != nil && cerebrasAPICfg.APIKey == "test"
	if isTestEnv {
		log.Printf("ChromemDB: Initializing TEST persistent client (shared instance)")
	} else {
		log.Printf("ChromemDB: Initializing persistent client (shared instance)")
	}

	// Use the provided chromemClient
	client := chromemClient
	var cef *CerebrasEmbeddingClient
	var embeddingFunc chromem.EmbeddingFunc

	// Check if running in a test environment (e.g., path contains "test_chromem")
	// Reuse existing isTestEnv variable

	if isTestEnv {
		log.Println("ChromemSyncManager: Test environment detected, using dummy embedding function.")
		embeddingFunc = func(ctx context.Context, text string) ([]float32, error) { //nolint:revive
			// Return a dummy embedding of a fixed small dimension, e.g., 10
			return make([]float32, 10), nil
		}
	} else {
		// Use deterministic Cerebras embedding client
		apiKey := utils.DEFAULT_CEREBRAS_API_KEY
		if apiKey == "" || apiKey == "your_default_or_public_cerebras_api_key_if_any" {
			apiKey = "deterministic" // Placeholder for deterministic mode
		}

		log.Println("ChromemSyncManager: Using deterministic Cerebras embedding client.") //nolint:revive
		cef = NewCerebrasEmbeddingClient(apiKey)
		embeddingFunc = func(ctx context.Context, text string) ([]float32, error) {
			return EmbedFunc(ctx, cef, text)
		}
	}

	csm := &ChromemSyncManager{
		client: client,
		ef:     embeddingFunc,
		cef:    cef, // Will be nil in test env
		mu:     sync.RWMutex{},
		db:     db, // Store the LevelDB client
	}

	// Get or create collections, passing the embedding function
	// chromem-go requires the EF when getting/creating collections for persistent DBs
	var err error
	csm.transactionCollection, err = client.GetOrCreateCollection("transactions", make(map[string]string), csm.ef)
	if err != nil {
		return nil, fmt.Errorf("failed to get/create transactions collection: %w", err)
	}
	log.Printf("ChromemDB: transactions collection ready.")

	csm.contextRecordCollection, err = client.GetOrCreateCollection("context_records", make(map[string]string), csm.ef)
	if err != nil {
		return nil, fmt.Errorf("failed to get/create context_records collection: %w", err)
	}
	log.Printf("ChromemDB: context_records collection ready.")

	csm.capabilityDescriptorCollection, err = client.GetOrCreateCollection("capability_descriptors", make(map[string]string), csm.ef)
	if err != nil {
		return nil, fmt.Errorf("failed to get/create capability_descriptors collection: %w", err)
	}
	log.Printf("ChromemDB: capability_descriptors collection ready.")

	return csm, nil
}

// Close closes the ChromemSyncManager
func (csm *ChromemSyncManager) Close() error {
	return nil
}

// GetCapabilityDescriptorFromChromemDB retrieves a capability descriptor from ChromemDB
func (csm *ChromemSyncManager) GetCapabilityDescriptorFromChromemDB(capabilityID string) (*types.CapabilityDescriptor, error) {
	if csm.client == nil {
		return nil, fmt.Errorf("ChromemSyncManager client is not initialized")
	}

	csm.mu.RLock()
	defer csm.mu.RUnlock()

	// Use the already initialized collection from the manager
	if csm.capabilityDescriptorCollection == nil {
		return nil, fmt.Errorf("capability_descriptors collection is not initialized")
	}

	// Attempt to get the capability by ID
	doc, err := csm.capabilityDescriptorCollection.GetByID(context.Background(), capabilityID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving capability ID '%s' from ChromemDB: %w", capabilityID, err)
	}

	if doc.Content == "" {
		return nil, fmt.Errorf("capability ID '%s' not found in ChromemDB", capabilityID)
	}

	// Convert document content to CapabilityDescriptor
	var capDesc types.CapabilityDescriptor

	// If content is a JSON string, unmarshal it
	if err := json.Unmarshal([]byte(doc.Content), &capDesc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal document data to CapabilityDescriptor for ID '%s': %w", capabilityID, err)
	}

	return &capDesc, nil
}

// Get retrieves documents from ChromemDB collections
func (csm *ChromemSyncManager) Get(ctx context.Context, ids []string, include []string, exclude []string) ([]chromem.Document, error) {
	csm.mu.RLock()
	defer csm.mu.RUnlock()

	var results []chromem.Document
	var found bool

	for _, id := range ids {
		// Try capability descriptors first
		docCap, errCap := csm.capabilityDescriptorCollection.GetByID(ctx, id)
		log.Printf("[DEBUG] ChromemSyncManager.Get: GetByID for ID '%s' in capability_descriptors returned doc: (ID: '%s', Content: '%.30s...'), err: %v", id, docCap.ID, docCap.Content, errCap)
		if errCap == nil {
			// Create enhanced document with both content and metadata
			enhancedDoc := chromem.Document{
				ID:        docCap.ID,
				Content:   docCap.Content,
				Metadata:  docCap.Metadata,
				Embedding: docCap.Embedding,
			}
			results = append(results, enhancedDoc)
			found = true
			continue
		} else {
			log.Printf("[DEBUG] ChromemSyncManager.Get: GetByID for ID '%s' in capability_descriptors failed, trying other collections. Error: %v", id, errCap)

			// Then try context records
			docCtx, errCtx := csm.contextRecordCollection.GetByID(ctx, id)
			log.Printf("[DEBUG] ChromemSyncManager.Get: GetByID for ID '%s' in context_records returned doc: (ID: '%s', Content: '%.30s...'), err: %v", id, docCtx.ID, docCtx.Content, errCtx)
			if errCtx == nil {
				enhancedDoc := chromem.Document{
					ID:        docCtx.ID,
					Content:   docCtx.Content,
					Metadata:  docCtx.Metadata,
					Embedding: docCtx.Embedding,
				}
				results = append(results, enhancedDoc)
				found = true
				continue
			}

			// Finally try transactions
			docTx, errTx := csm.transactionCollection.GetByID(ctx, id)
			log.Printf("[DEBUG] ChromemSyncManager.Get: GetByID for ID '%s' in transactions returned doc: (ID: '%s', Content: '%.30s...'), err: %v", id, docTx.ID, docTx.Content, errTx)
			if errTx == nil {
				enhancedDoc := chromem.Document{
					ID:        docTx.ID,
					Content:   docTx.Content,
					Metadata:  docTx.Metadata,
					Embedding: docTx.Embedding,
				}
				results = append(results, enhancedDoc)
				found = true
				continue
			}
		}
	}

	if !found {
		log.Printf("[DEBUG] ChromemSyncManager.Get: No documents found for ids: %v. Returning error.", ids)
		return nil, fmt.Errorf("no documents found for ids: %v", ids)
	}

	log.Printf("[DEBUG] ChromemSyncManager.Get: Returning %d results, error: nil. Found: %t", len(results), found)
	return results, nil
}

// OnNewBlockConfirmed is called when a new block is added to the canonical chain
func (csm *ChromemSyncManager) OnNewBlockConfirmed(block *Block, registrationContextRecords ...*pb.ContextRecordProto) error {
	csm.mu.Lock()
	defer csm.mu.Unlock()

	log.Printf("ChromemDB: Processing new confirmed block %d", block.BlockNumber)

	// Process transactions in the block
	for _, tx := range block.Transactions {
		// Skip invalid transactions
		if _, exists := block.InvalidTxHashes[tx.TransactionHash]; exists {
			log.Printf("ChromemDB: Skipping invalid transaction %s", tx.TransactionHash)
			continue
		}

		// Index the transaction
		txID, txDoc, txMetadata := PrepareTransactionForChromem(tx, int64(block.BlockNumber), block.Timestamp)

		// Add to ChromemDB
		err := csm.transactionCollection.Add(
			context.Background(),
			[]string{txID},
			nil, // Let the embedding function handle this
			[]map[string]string{convertMetadata(txMetadata)},
			[]string{txDoc},
		)

		if err != nil {
			log.Printf("Error adding transaction %s to ChromemDB: %v", txID, err)
		}

		// Handle capability registration transactions
		if string(tx.Type) == "register_capability_txn" {
			log.Printf("[DEBUG] ChromemSyncManager: Processing register_capability_txn: %s in block %d", tx.TransactionHash, block.BlockNumber)
			// First try to unmarshal as protobuf
			var registerData pb.MCPRegisterCapabilityDataProto
			if err := proto.Unmarshal(tx.Data, &registerData); err != nil {
				log.Printf("[DEBUG] ChromemSyncManager: Protobuf unmarshal failed for tx %s (expected for JSON tx.Data): %v", tx.TransactionHash, err)
				// If protobuf fails, try JSON unmarshal as fallback
				var regDataJSON types.MCPRegisterCapabilityData // Changed from map[string]interface{}
				if jsonErr := json.Unmarshal(tx.Data, &regDataJSON); jsonErr != nil {
					log.Printf("Error unmarshaling capability registration data (both proto and JSON failed): protoErr=%v jsonErr=%v", err, jsonErr)
					continue
				}

				// Use our new conversion helper
				capDesc := regDataJSON.CapabilityDescriptor
				if capDesc == nil {
					log.Printf("[ERROR] ChromemSyncManager: CapabilityDescriptor in JSON is not a map or is nil for tx %s. Type: %T", tx.TransactionHash, regDataJSON.CapabilityDescriptor)
					continue
				}

				converted, err := types.ConvertMapToCapability(capDesc)
				if err != nil {
					log.Printf("[ERROR] ChromemSyncManager: Failed to convert JSON capability descriptor for tx %s: %v", tx.TransactionHash, err)
					continue
				}

				// Convert to protobuf format
				switch desc := converted.(type) {
				case *types.ResourceDescriptor:
					resourceProto, err := ConvertResourceDescriptorToProto(*desc)
					if err != nil {
						log.Printf("[ERROR] ChromemSyncManager: Failed to convert resource descriptor to proto for tx %s: %v", tx.TransactionHash, err)
						continue
					}
					registerData = pb.MCPRegisterCapabilityDataProto{
						CapabilityDescriptor: &pb.CapabilityDescriptorContainerProto{
							Descriptor_: &pb.CapabilityDescriptorContainerProto_Resource{
								Resource: resourceProto,
							},
						},
					}
				case *types.ToolDescriptor:
					toolProto, err := ConvertToolDescriptorToProto(*desc)
					if err != nil {
						log.Printf("[ERROR] ChromemSyncManager: Failed to convert tool descriptor to proto for tx %s: %v", tx.TransactionHash, err)
						continue
					}
					registerData = pb.MCPRegisterCapabilityDataProto{
						CapabilityDescriptor: &pb.CapabilityDescriptorContainerProto{
							Descriptor_: &pb.CapabilityDescriptorContainerProto_Tool{
								Tool: toolProto,
							},
						},
					}
				case *types.PromptDescriptor:
					promptProto, err := ConvertPromptDescriptorToProto(*desc)
					if err != nil {
						log.Printf("[ERROR] ChromemSyncManager: Failed to convert prompt descriptor to proto for tx %s: %v", tx.TransactionHash, err)
						continue
					}
					registerData = pb.MCPRegisterCapabilityDataProto{
						CapabilityDescriptor: &pb.CapabilityDescriptorContainerProto{
							Descriptor_: &pb.CapabilityDescriptorContainerProto_Prompt{
								Prompt: promptProto,
							},
						},
					}
				case *types.MemoryServiceDescriptor:
					memoryProto, err := ConvertMemoryServiceDescriptorToProto(*desc)
					if err != nil {
						log.Printf("[ERROR] ChromemSyncManager: Failed to convert memory service descriptor to proto for tx %s: %v", tx.TransactionHash, err)
						continue
					}
					registerData = pb.MCPRegisterCapabilityDataProto{
						CapabilityDescriptor: &pb.CapabilityDescriptorContainerProto{
							Descriptor_: &pb.CapabilityDescriptorContainerProto_MemoryService{
								MemoryService: memoryProto,
							},
						},
					}
				default:
					log.Printf("[ERROR] ChromemSyncManager: Unsupported capability type '%T' in JSON fallback for tx %s", converted, tx.TransactionHash)
					continue
				}
			} else {
				log.Printf("[DEBUG] ChromemSyncManager: Protobuf unmarshal SUCCEEDED for tx %s (this path might be taken if tx.Data was already proto)", tx.TransactionHash)
			}

			capID, capDoc, capMetadata, err := PrepareCapabilityDescriptorForChromemFromRegister(
				&registerData,
				tx.TransactionHash,
				block.BlockNumber,
				block.Timestamp,
			)
			if err != nil {
				log.Printf("[ERROR] ChromemSyncManager: Error preparing capability descriptor for tx %s: %v", tx.TransactionHash, err)
				continue
			}
			log.Printf("[DEBUG] ChromemSyncManager: Prepared for ChromemDB. CapID: %s, Doc: %.60s..., Metadata: %v", capID, capDoc, capMetadata)

			err = csm.capabilityDescriptorCollection.Add(
				context.Background(),
				[]string{capID},
				nil,
				[]map[string]string{convertMetadata(capMetadata)},
				[]string{capDoc},
			)
			if err != nil {
				log.Printf("[ERROR] ChromemSyncManager: Error adding capability descriptor %s to ChromemDB for tx %s: %v", capID, tx.TransactionHash, err)
			} else {
				log.Printf("[DEBUG] ChromemSyncManager: Successfully added capability %s to ChromemDB for tx %s.", capID, tx.TransactionHash)

				// After successfully adding the capability descriptor,
				// try to get the context record from LevelDB and add it to ChromemDB.
				var levelDBContextRecordProto *pb.ContextRecordProto
				var errDbGet error

				// Find the matching pre-fetched context record for THIS transaction
				foundPrefetched := false
				for _, preFetchedProto := range registrationContextRecords {
					if preFetchedProto != nil && preFetchedProto.Id == tx.TransactionHash {
						levelDBContextRecordProto = preFetchedProto
						foundPrefetched = true
						log.Printf("[DEBUG] ChromemSyncManager: Using pre-fetched context record for registration tx %s.", tx.TransactionHash)
						break
					}
				}

				if !foundPrefetched {
					log.Printf("[DEBUG] ChromemSyncManager: No pre-fetched context record found for tx %s. Attempting LevelDB fetch.", tx.TransactionHash)
					// Retry fetching from LevelDB if not pre-fetched
					for i := 0; i < 3; i++ { // Retry up to 3 times
						levelDBContextRecordProto, errDbGet = csm.db.GetContextRecord(tx.TransactionHash)
						if errDbGet == nil && levelDBContextRecordProto != nil {
							break
						}
						log.Printf("[WARN] ChromemSyncManager: Attempt %d to get context record from LevelDB for reg tx %s failed: %v. Retrying in 100ms...", i+1, tx.TransactionHash, errDbGet)
						time.Sleep(100 * time.Millisecond)
					}
				}

				if levelDBContextRecordProto == nil {
					log.Printf("[ERROR] ChromemSyncManager: Failed to get context record from LevelDB for registration tx %s after retries. Error: %v. It will not be synced to ChromemDB.", tx.TransactionHash, errDbGet)
				} else {
					// Convert proto to types.ContextRecord
					contextRecord, errConv := ConvertProtoToContextRecord(levelDBContextRecordProto)
					if errConv != nil {
						log.Printf("[ERROR] ChromemSyncManager: Failed to convert context record proto to type for tx %s: %v", tx.TransactionHash, errConv)
					} else {
						ctxIDReg, ctxDocReg, ctxMetadataReg := PrepareContextRecordForChromemEnhanced(
							&contextRecord,
							tx.TransactionHash,
							int64(block.BlockNumber),
							block.Timestamp,
						)
						errAddCtx := csm.contextRecordCollection.Add(
							context.Background(),
							[]string{ctxIDReg},
							nil,
							[]map[string]string{convertMetadata(ctxMetadataReg)},
							[]string{ctxDocReg},
						)
						if errAddCtx != nil {
							log.Printf("[ERROR] ChromemSyncManager: Error adding registration context record %s to ChromemDB for tx %s: %v", ctxIDReg, tx.TransactionHash, errAddCtx)
						} else {
							log.Printf("[DEBUG] ChromemSyncManager: Successfully added registration context record %s to ChromemDB for tx %s.", ctxIDReg, tx.TransactionHash)
						}
					}
				}
			}
		} else if string(tx.Type) == TransactionTypeMCPInvokeCapability {
			// Ensure this constant is "invoke_capability_txn"
			log.Printf("ChromemDB Sync: Processing INVOKE_CAPABILITY_TXN: %s", tx.TransactionHash) // DEBUG LOG

			// Handle capability invocation to store ContextRecord
			var invokeData types.MCPInvokeCapabilityData
			if err := json.Unmarshal(tx.Data, &invokeData); err != nil {
				log.Printf("Error unmarshaling MCPInvokeCapabilityData for tx %s: %v", tx.TransactionHash, err)
				log.Printf("ChromemDB Sync: Error unmarshaling MCPInvokeCapabilityData for tx %s: %v", tx.TransactionHash, err) // DEBUG LOG
				continue
			}

			log.Printf("ChromemDB Sync: Successfully unmarshaled invokeData for tx %s. ContextRecord CapID: %s", tx.TransactionHash, invokeData.ContextRecord.CapabilityID) // DEBUG LOG

			// Prepare ContextRecord for ChromemDB
			// Ensure the ContextRecord has its ID set to the transaction hash.
			if invokeData.ContextRecord.ID == "" {
				log.Printf("ChromemDB Sync: ContextRecord.ID is empty for tx %s, setting to tx.TransactionHash", tx.TransactionHash) // DEBUG LOG
				invokeData.ContextRecord.ID = tx.TransactionHash
			}

			ctxID, ctxDoc, ctxMetadata := PrepareContextRecordForChromemEnhanced(
				&invokeData.ContextRecord,
				tx.TransactionHash,
				int64(block.BlockNumber),
				block.Timestamp)

			log.Printf("ChromemDB Sync: Prepared ContextRecord for ChromemDB. ID: %s", ctxID) // DEBUG LOG

			// Add with timeout and retry logic
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			maxRetries := 3
			var lastErr error

			for i := 0; i < maxRetries; i++ {
				err = csm.contextRecordCollection.Add(
					ctx,
					[]string{ctxID},
					nil, // embeddings
					[]map[string]string{convertMetadata(ctxMetadata)},
					[]string{ctxDoc}, // documents
				)

				if err == nil {
					log.Printf("ChromemDB Sync: Successfully added context record %s to ChromemDB for invoke tx %s", ctxID, tx.TransactionHash)
					break
				}

				lastErr = err
				log.Printf("ChromemDB Sync: Retry %d - Error adding context record %s to ChromemDB: %v", i+1, ctxID, err)
				time.Sleep(100 * time.Millisecond) // Small delay before retry
			}

			if lastErr != nil {
				log.Printf("ChromemDB Sync: Failed to add context record %s to ChromemDB after %d retries: %v", ctxID, maxRetries, lastErr)
			}
		}
	}

	return nil
}

// convertMetadata converts a map[string]interface{} to map[string]string for ChromemDB
func convertMetadata(metadata map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for k, v := range metadata {
		result[k] = fmt.Sprintf("%v", v)
	}
	return result
}

// PrepareTransactionForChromem prepares a transaction for ChromemDB
func PrepareTransactionForChromem(tx *Transaction, blockNumber int64, blockTimestamp int64) (string, string, map[string]interface{}) {
	// Use transaction hash as ID
	id := tx.TransactionHash

	// Create a document with transaction details
	doc := fmt.Sprintf("Transaction %s: Type=%s, From=%s, Block=%d, Timestamp=%d",
		tx.TransactionHash, tx.Type, tx.From, blockNumber, blockTimestamp)

	// Create metadata for filtering/sorting
	metadata := map[string]interface{}{
		"type":        string(tx.Type),
		"from":        tx.From,
		"blockNumber": blockNumber,
		"timestamp":   blockTimestamp,
	}
	return id, doc, metadata
}

// PrepareCapabilityDescriptorForChromemFromRegister prepares a capability descriptor for ChromemDB from a register transaction
func PrepareCapabilityDescriptorForChromemFromRegister(registerData *pb.MCPRegisterCapabilityDataProto, txHash string, blockNumber uint64, blockTimestamp int64) (string, string, map[string]interface{}, error) {
	if registerData == nil || registerData.CapabilityDescriptor == nil {
		return "", "", nil, fmt.Errorf("invalid register data: nil pointer")
	}

	var capID string
	var capType string
	var capName string
	var capOwner string
	var resourceType string

	// Extract data based on the capability type
	switch desc := registerData.CapabilityDescriptor.Descriptor_.(type) {
	case *pb.CapabilityDescriptorContainerProto_Resource:
		if desc.Resource == nil || desc.Resource.BaseDescriptor == nil {
			return "", "", nil, fmt.Errorf("invalid resource descriptor: nil pointer")
		}
		capID = desc.Resource.BaseDescriptor.Id
		capType = "RESOURCE"
		capName = desc.Resource.BaseDescriptor.Name
		capOwner = desc.Resource.BaseDescriptor.Owner
		resourceType = desc.Resource.ResourceType.String()

	case *pb.CapabilityDescriptorContainerProto_Tool:
		if desc.Tool == nil || desc.Tool.BaseDescriptor == nil {
			return "", "", nil, fmt.Errorf("invalid tool descriptor: nil pointer")
		}
		capID = desc.Tool.BaseDescriptor.Id
		capType = "TOOL"
		capName = desc.Tool.BaseDescriptor.Name
		capOwner = desc.Tool.BaseDescriptor.Owner

	case *pb.CapabilityDescriptorContainerProto_Prompt:
		if desc.Prompt == nil || desc.Prompt.BaseDescriptor == nil {
			return "", "", nil, fmt.Errorf("invalid prompt descriptor: nil pointer")
		}
		capID = desc.Prompt.BaseDescriptor.Id
		capType = "PROMPT"
		capName = desc.Prompt.BaseDescriptor.Name
		capOwner = desc.Prompt.BaseDescriptor.Owner

	case *pb.CapabilityDescriptorContainerProto_MemoryService:
		if desc.MemoryService == nil || desc.MemoryService.BaseDescriptor == nil {
			return "", "", nil, fmt.Errorf("invalid memory service descriptor: nil pointer")
		}
		capID = desc.MemoryService.BaseDescriptor.Id
		capType = "MEMORY_SERVICE"
		capName = desc.MemoryService.BaseDescriptor.Name
		capOwner = desc.MemoryService.BaseDescriptor.Owner

	default:
		return "", "", nil, fmt.Errorf("unknown capability type")
	}

	if capID == "" {
		return "", "", nil, fmt.Errorf("capability ID is empty")
	}

	// Create a document with capability details
	doc := fmt.Sprintf("Capability %s: Type=%s, Name=%s, Owner=%s, ResourceType=%s, RegisterTx=%s, Block=%d",
		capID, capType, capName, capOwner, resourceType, txHash, blockNumber)

	// Create metadata for filtering/sorting
	metadata := map[string]interface{}{
		"type":           capType,
		"name":           capName,
		"owner":          capOwner,
		"resourceType":   resourceType,
		"registerTxHash": txHash,
		"blockNumber":    blockNumber,
		"timestamp":      blockTimestamp,
	}

	return capID, doc, metadata, nil
}

type SyncManager struct {
	blockchain *BlockchainStruct
	mu         sync.RWMutex
	stop       bool
}

// NewSyncManager creates a new SyncManager
func NewSyncManager(blockchain *BlockchainStruct) *SyncManager {
	return &SyncManager{
		blockchain: blockchain,
		stop:       false,
	}
}

// Start begins the synchronization process
func (sm *SyncManager) Start() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.blockchain == nil {
		return errors.New("blockchain is not initialized")
	}

	sm.stop = false
	log.Println("SyncManager: Started synchronization")
	return nil
}

// Stop halts the synchronization process
func (sm *SyncManager) Stop() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.stop = true
	log.Println("SyncManager: Stopped synchronization")
}

// IsStopped returns whether the sync manager is stopped
func (sm *SyncManager) IsStopped() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return sm.stop
}
