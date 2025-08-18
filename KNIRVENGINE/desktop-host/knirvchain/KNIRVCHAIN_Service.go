package knirvchain // Or your preferred package name for the client's service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	knirvtx "github.com/cloud-equities/KNIRVCHAIN/sdk/go/transaction"
	// "github.com/cloud-equities/KNIRVCHAIN/sdk/go/transmission" // Comment out if not needed
)

var errNotFound = fmt.Errorf("not found")

// KNIRVServiceConfig holds the configuration for the KNIRVService.
type KNIRVServiceConfig struct {
	WalletServerURL     string // e.g., "http://localhost:9090"
	BlockchainServerURL string // e.g., "http://localhost:8080" (for direct queries if needed)
	DefaultTimeout      time.Duration
}

// KNIRVService provides methods to interact with the KNIRVCHAIN network.
type KNIRVService struct {
	config     KNIRVServiceConfig
	httpClient *http.Client
	appWallet  *knirvtx.Wallet // The wallet instance for the client application
}

// NewKNIRVService creates a new instance of KNIRVService.
// It requires a wallet instance for signing transactions.
func NewKNIRVService(config KNIRVServiceConfig, appWallet *knirvtx.Wallet) (*KNIRVService, error) {
	if config.WalletServerURL == "" {
		return nil, fmt.Errorf("WalletServerURL is required in config")
	}
	if appWallet == nil {
		return nil, fmt.Errorf("application wallet instance is required")
	}
	if config.DefaultTimeout == 0 {
		config.DefaultTimeout = 30 * time.Second // Default timeout
	}

	return &KNIRVService{
		config:    config,
		appWallet: appWallet,
		httpClient: &http.Client{
			Timeout: config.DefaultTimeout,
		},
	}, nil
}

// --- Capability Registration (Two-Step Process) ---

// InitiateRegisterCapabilityResponse is the expected response from the initiate step.
type InitiateRegisterCapabilityResponse struct {
	Status                   string          `json:"status"`
	PendingTransactionHash   string          `json:"pendingTransactionHash"`
	CanonicalCapabilityID    string          `json:"canonicalCapabilityID"`
	FullDescriptorForSigning json.RawMessage `json:"fullDescriptorForSigning"`
}

// FinalizeRegisterCapabilityResponse is the expected response from the finalize step.
type FinalizeRegisterCapabilityResponse struct {
	Status          string `json:"status"`
	TransactionHash string `json:"transaction_hash"`
	CapabilityURI   string `json:"capability_uri"`
}

// RegisterCapability handles the full two-step capability registration process.
// descriptor should be the specific type (e.g., mcp_types.ResourceDescriptor, mcp_types.ToolDescriptor)
// with its ID field empty or ignored.
func (s *KNIRVService) RegisterCapability(
	ctx context.Context,
	desiredName string,
	capabilityType knirvtx.CapabilityType,
	descriptor interface{}, // e.g., ResourceDescriptor, ToolDescriptor
	fee uint64,
) (*FinalizeRegisterCapabilityResponse, error) {

	// Step 1: Initiate Registration
	initiateData := map[string]interface{}{
		"desiredName":    desiredName,
		"capabilityType": string(capabilityType),
		"descriptor":     descriptor, // Server will set the ID
		"from":           s.appWallet.GetAddress(),
		"fee":            fee,
		"publicKey":      s.appWallet.GetPublicKeyHex(), // Server might use this for initial checks or logging
	}

	initiateURL := s.config.WalletServerURL + "/wallet/mcp/initiate_register_capability"
	// Fallback or direct if WalletServer doesn't have this specific endpoint yet:
	// initiateURL := s.config.BlockchainServerURL + "/mcp/capability/register/initiate"

	reqBodyBytes, err := json.Marshal(initiateData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal initiate registration request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", initiateURL, bytes.NewBuffer(reqBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create initiate registration request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("initiate registration request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("initiate registration failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var initResponse InitiateRegisterCapabilityResponse
	if err := json.NewDecoder(resp.Body).Decode(&initResponse); err != nil {
		return nil, fmt.Errorf("failed to decode initiate registration response: %w", err)
	}

	if initResponse.Status != "pending_signature" {
		return nil, fmt.Errorf("initiate registration returned unexpected status: %s", initResponse.Status)
	}

	// Step 2: Client Reconstructs, Signs, and Finalizes
	var serverFinalizedDescriptorMap map[string]interface{} // Use map to preserve structure for hashing
	if err := json.Unmarshal(initResponse.FullDescriptorForSigning, &serverFinalizedDescriptorMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal FullDescriptorForSigning into map: %w", err)
	}

	// Extract timestamp from the server-provided descriptor for transaction construction
	var baseDesc knirvtx.BaseDescriptor
	if err := json.Unmarshal(initResponse.FullDescriptorForSigning, &baseDesc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal BaseDescriptor from FullDescriptorForSigning: %w", err)
	}
	if baseDesc.Timestamp == 0 {
		// Fallback if timestamp not in descriptor, though server should provide it
		baseDesc.Timestamp = time.Now().Unix()
	}

	clientFinalizationDataBytes, err := json.Marshal(map[string]interface{}{
		"capabilityDescriptor": serverFinalizedDescriptorMap,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal client finalization data: %w", err)
	}

	// Create the transaction object that the server expects for the pending hash
	// The server calculated pendingTransactionHash based on an unsigned transaction.
	// The client needs to create a similar structure to sign.
	txnToSign := knirvtx.NewMCPTransaction(
		s.appWallet.GetAddress(),
		"", // No recipient for capability registration
		0,  // No value transfer
		clientFinalizationDataBytes,
		knirvtx.TransactionTypeMCPRegisterCapability,
		fee,
		baseDesc.Timestamp, // Use timestamp from server-provided descriptor
	)

	// Verify client-side hash matches server's pending hash (optional sanity check)
	// Note: NewMCPTransaction already calculates and sets the hash for the unsigned transaction.
	if txnToSign.TransactionHash != initResponse.PendingTransactionHash {
		// This indicates a potential mismatch in how the transaction data or its components
		// (like timestamp) are being handled between client and server before signing.
		// Or, the server's NewMCPTransaction hashing logic for the pending hash differs.
		fmt.Printf("Warning: Client-side pending hash (%s) does not match server's pending hash (%s). Ensure data and timestamp are consistent.\n", txnToSign.TransactionHash, initResponse.PendingTransactionHash)
		// Depending on strictness, you might return an error here.
	}

	// Sign the transaction (this will re-calculate the hash with PublicKey if not already set)
	txnToSign.PublicKey = s.appWallet.GetPublicKeyHex() // Ensure PublicKey is set before final signing
	if err := s.appWallet.SignTransaction(txnToSign); err != nil {
		return nil, fmt.Errorf("failed to sign finalization transaction: %w", err)
	}

	finalizeData := map[string]interface{}{
		"pendingTransactionHash": initResponse.PendingTransactionHash,
		"publicKey":              s.appWallet.GetPublicKeyHex(),
		"signature":              txnToSign.Signature, // Signature from the signed transaction
	}

	finalizeURL := s.config.WalletServerURL + "/wallet/mcp/finalize_register_capability"
	// Fallback or direct:
	// finalizeURL := s.config.BlockchainServerURL + "/mcp/capability/register/finalize"

	finalizeReqBodyBytes, err := json.Marshal(finalizeData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal finalize registration request: %w", err)
	}

	finalizeReq, err := http.NewRequestWithContext(ctx, "POST", finalizeURL, bytes.NewBuffer(finalizeReqBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create finalize registration request: %w", err)
	}
	finalizeReq.Header.Set("Content-Type", "application/json")

	finalizeResp, err := s.httpClient.Do(finalizeReq)
	if err != nil {
		return nil, fmt.Errorf("finalize registration request failed: %w", err)
	}
	defer finalizeResp.Body.Close()

	if finalizeResp.StatusCode != http.StatusCreated && finalizeResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(finalizeResp.Body)
		return nil, fmt.Errorf("finalize registration failed with status %d: %s", finalizeResp.StatusCode, string(bodyBytes))
	}

	var finalResponse FinalizeRegisterCapabilityResponse
	if err := json.NewDecoder(finalizeResp.Body).Decode(&finalResponse); err != nil {
		return nil, fmt.Errorf("failed to decode finalize registration response: %w", err)
	}

	return &finalResponse, nil
}

// --- Capability Invocation ---

// InvokeCapabilityData is the data sent to the wallet server for invoking a capability.
type InvokeCapabilityData struct {
	CapabilityID    string                 `json:"capabilityID"`
	InteractionType string                 `json:"interactionType"` // e.g., "PLUGIN_EXECUTION", "TOOL_INVOCATION"
	Initiator       string                 `json:"initiator"`       // Should be s.appWallet.GetAddress()
	InputHash       string                 `json:"inputHash,omitempty"`
	OutputHash      string                 `json:"outputHash,omitempty"`
	Details         map[string]interface{} `json:"details,omitempty"`
	Fee             uint64                 `json:"fee"` // This is the GasFeeNRN of the capability
	// Wallet server will add PublicKey, sign, and construct the full transaction
}

// InvokeCapabilityResponse is the expected response from invoking a capability.
type InvokeCapabilityResponse struct {
	Status          string `json:"status"`
	TransactionHash string `json:"transaction_hash"`
	ContextID       string `json:"context_id"` // Usually same as transaction_hash
}

// InvokeCapability logs the usage of a capability on-chain.
func (s *KNIRVService) InvokeCapability(
	ctx context.Context,
	capabilityID string,
	interactionType string,
	inputData []byte, // Raw data for hashing
	outputData []byte, // Raw data for hashing
	details map[string]interface{},
	capabilityGasFee uint64, // The GasFeeNRN defined in the capability's descriptor
) (*InvokeCapabilityResponse, error) {

	var inputHashStr, outputHashStr string
	if len(inputData) > 0 {
		inputHash := sha256.Sum256(inputData)
		inputHashStr = hex.EncodeToString(inputHash[:])
	}
	if len(outputData) > 0 {
		outputHash := sha256.Sum256(outputData)
		outputHashStr = hex.EncodeToString(outputHash[:])
	}

	invokeData := InvokeCapabilityData{
		CapabilityID:    capabilityID,
		InteractionType: interactionType,
		Initiator:       s.appWallet.GetAddress(),
		InputHash:       inputHashStr,
		OutputHash:      outputHashStr,
		Details:         details,
		Fee:             capabilityGasFee,
	}

	invokeURL := s.config.WalletServerURL + "/wallet/mcp/create_invoke_capability"

	reqBodyBytes, err := json.Marshal(invokeData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal invoke capability request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", invokeURL, bytes.NewBuffer(reqBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create invoke capability request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("invoke capability request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("invoke capability failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var invokeResponse InvokeCapabilityResponse
	if err := json.NewDecoder(resp.Body).Decode(&invokeResponse); err != nil {
		return nil, fmt.Errorf("failed to decode invoke capability response: %w", err)
	}

	return &invokeResponse, nil
}

// --- Capability Discovery & Querying ---

// ListCapabilitiesFilter allows filtering when listing capabilities.
type ListCapabilitiesFilter struct {
	Type         knirvtx.CapabilityType `url:"type,omitempty"`
	ResourceType knirvtx.ResourceType   `url:"resourceType,omitempty"`
	Owner        string                 `url:"owner,omitempty"`
	Name         string                 `url:"name,omitempty"`
	Limit        int                    `url:"limit,omitempty"`
	// Add other filterable fields as your API supports
}

// ListCapabilities retrieves a list of capability descriptors based on filters.
// Returns a slice of json.RawMessage, as the concrete type of each descriptor varies.
func (s *KNIRVService) ListCapabilities(ctx context.Context, filter ListCapabilitiesFilter) ([]json.RawMessage, error) {
	listURL := s.config.BlockchainServerURL + "/mcp/capabilities"

	params := url.Values{}
	if filter.Type != "" {
		params.Add("type", string(filter.Type))
	}
	if filter.ResourceType != "" {
		params.Add("resourceType", string(filter.ResourceType))
	}
	if filter.Owner != "" {
		params.Add("owner", filter.Owner)
	}
	if filter.Name != "" {
		params.Add("name", filter.Name)
	}
	if filter.Limit > 0 {
		params.Add("limit", fmt.Sprintf("%d", filter.Limit))
	}

	if len(params) > 0 {
		listURL += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", listURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list capabilities request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list capabilities request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list capabilities failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var response struct {
		Capabilities []json.RawMessage `json:"capabilities"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode list capabilities response: %w", err)
	}
	return response.Capabilities, nil
}

// GetCapabilityDescriptor retrieves a specific capability descriptor by its ID.
// Returns json.RawMessage as the concrete type varies.
func (s *KNIRVService) GetCapabilityDescriptor(ctx context.Context, capabilityID string) (json.RawMessage, error) {
	getURL := fmt.Sprintf("%s/mcp/capability/%s", s.config.BlockchainServerURL, url.PathEscape(capabilityID))

	req, err := http.NewRequestWithContext(ctx, "GET", getURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get capability descriptor request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get capability descriptor request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("capability descriptor %s not found: %w", capabilityID, errNotFound) // Define errNotFound
		}
		return nil, fmt.Errorf("get capability descriptor failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read get capability descriptor response body: %w", err)
	}
	return bodyBytes, nil
}

// --- Context Record Querying ---

// GetContextRecord retrieves a specific context record by its ID (transaction hash).
func (s *KNIRVService) GetContextRecord(ctx context.Context, contextID string) (*knirvtx.ContextRecord, error) {
	getURL := fmt.Sprintf("%s/mcp/context/%s", s.config.BlockchainServerURL, url.PathEscape(contextID))

	req, err := http.NewRequestWithContext(ctx, "GET", getURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get context record request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get context record request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("context record %s not found: %w", contextID, errNotFound)
		}
		return nil, fmt.Errorf("get context record failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var contextRecord knirvtx.ContextRecord
	if err := json.NewDecoder(resp.Body).Decode(&contextRecord); err != nil {
		return nil, fmt.Errorf("failed to decode context record response: %w", err)
	}
	return &contextRecord, nil
}

// GetCapabilityInvocations retrieves all context records for a given capability ID.
func (s *KNIRVService) GetCapabilityInvocations(ctx context.Context, capabilityID string) ([]knirvtx.ContextRecord, error) {
	getURL := fmt.Sprintf("%s/mcp/capability/%s/invocations", s.config.BlockchainServerURL, url.PathEscape(capabilityID))

	req, err := http.NewRequestWithContext(ctx, "GET", getURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get capability invocations request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get capability invocations request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get capability invocations failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var response struct {
		ContextRecords []knirvtx.ContextRecord `json:"context_records"` // Assuming this is the API response structure
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode get capability invocations response: %w", err)
	}
	return response.ContextRecords, nil
}

// --- Plugin Handling (Conceptual - Download part might use P2P SDK later) ---

// DownloadPluginPackage attempts to download a plugin package.
// For now, it assumes HTTP(S) URIs in LocationHints. P2P via knirv:// would require libp2p integration.
func (s *KNIRVService) DownloadPluginPackage(ctx context.Context, descriptor knirvtx.ResourceDescriptor) ([]byte, error) {
	if descriptor.ResourceType != knirvtx.ResourceTypePlugin {
		return nil, fmt.Errorf("descriptor is not for a plugin resource type")
	}
	if len(descriptor.Schema.LocationHints) == 0 {
		return nil, fmt.Errorf("no location hints provided for plugin %s", descriptor.ID)
	}

	var lastErr error
	for _, hint := range descriptor.Schema.LocationHints {
		// Basic check for HTTP/HTTPS, knirv:// would need different handling
		if !(strings.HasPrefix(hint, "http://") || strings.HasPrefix(hint, "https://")) {
			// Log or skip non-HTTP hints for now
			fmt.Printf("Skipping non-HTTP location hint: %s\n", hint)
			lastErr = fmt.Errorf("unsupported URI scheme in hint: %s", hint)
			continue
		}

		req, err := http.NewRequestWithContext(ctx, "GET", hint, nil)
		if err != nil {
			lastErr = fmt.Errorf("failed to create request for hint %s: %w", hint, err)
			continue
		}

		resp, err := s.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to download from hint %s: %w", hint, err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			lastErr = fmt.Errorf("download from hint %s failed with status %d", hint, resp.StatusCode)
			continue
		}

		packageBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed to read package data from hint %s: %w", hint, err)
			continue
		}

		// Verify ContentHash
		hash := sha256.Sum256(packageBytes)
		if hex.EncodeToString(hash[:]) != descriptor.ContentHash {
			lastErr = fmt.Errorf("content hash mismatch for package from hint %s. Expected %s, got %s", hint, descriptor.ContentHash, hex.EncodeToString(hash[:]))
			continue // Try next hint
		}

		return packageBytes, nil // Success
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed to download plugin package %s after trying all hints: %w", descriptor.ID, lastErr)
	}
	return nil, fmt.Errorf("no suitable location hints found or all failed for plugin %s", descriptor.ID)
}

// TODO: Add methods for advanced context querying, and potentially direct transaction submission
// if not using the wallet server for everything.
