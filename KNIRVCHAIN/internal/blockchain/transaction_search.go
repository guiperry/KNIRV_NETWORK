package blockchain

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/philippgille/chromem-go"
)

// SearchTransactionsRequest defines the request structure for transaction search
type SearchTransactionsRequest struct {
	SemanticQuery string `json:"semantic_query,omitempty"` // Text for semantic search
	Type          string `json:"type,omitempty"`           // Filter by transaction type
	FromAddr      string `json:"from,omitempty"`           // Filter by sender address
	ToAddr        string `json:"to,omitempty"`             // Filter by recipient address
	StartTime     int64  `json:"start_time,omitempty"`     // Unix timestamp
	EndTime       int64  `json:"end_time,omitempty"`       // Unix timestamp
	MinAmount     uint64 `json:"min_amount,omitempty"`
	MaxAmount     uint64 `json:"max_amount,omitempty"`
	Page          int    `json:"page,omitempty"`      // For pagination (offset-based)
	PageSize      int    `json:"page_size,omitempty"` // N-Results for ChromemDB
}

// SearchTransactionsResponse defines the response structure for transaction search
type SearchTransactionsResponse struct {
	Results  []ApiTransaction `json:"results"` // Define ApiTransaction
	Total    int64            `json:"total"`   // Chromem might not give total matches easily for complex queries, this might be count of results returned
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}

// ApiTransaction represents a transaction in the API response
type ApiTransaction struct {
	TransactionHash string   `json:"transaction_hash"`
	BlockHash       string   `json:"block_hash,omitempty"`
	BlockNumber     uint64   `json:"block_number,omitempty"`
	From            string   `json:"from"`
	To              string   `json:"to,omitempty"`
	Value           uint64   `json:"value"`
	Type            string   `json:"type"`
	Timestamp       int64    `json:"timestamp"`
	Fee             uint64   `json:"fee"`
	Data            string   `json:"data,omitempty"`
	SimilarityScore *float32 `json:"similarity_score,omitempty"` // If semantic search was used
}

// HandleSearchTransactions handles searching for transactions
func HandleSearchTransactions(cm *ChromemManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var searchReq SearchTransactionsRequest
		if err := json.NewDecoder(r.Body).Decode(&searchReq); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if searchReq.Page <= 0 {
			searchReq.Page = 1
		}
		if searchReq.PageSize <= 0 || searchReq.PageSize > 100 {
			searchReq.PageSize = 20
		}

		// Call the ChromemDB search function
		apiResults, err := cm.SearchTransactionsChromem(
			context.Background(),
			searchReq.SemanticQuery,
			searchReq.PageSize,
			searchReq.Page,
			searchReq.Type,
			searchReq.FromAddr,
			searchReq.ToAddr,
			searchReq.StartTime,
			searchReq.EndTime,
			searchReq.MinAmount,
			searchReq.MaxAmount,
		)

		if err != nil {
			log.Printf("Failed to search transactions in ChromemDB: %v", err)
			http.Error(w, "Failed to search transactions", http.StatusInternalServerError)
			return
		}

		response := SearchTransactionsResponse{
			Results:  apiResults,
			Total:    int64(len(apiResults)), // Chromem returns N results, not total matches for a filter.
			Page:     searchReq.Page,
			PageSize: searchReq.PageSize,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// RegisterTransactionSearchHandlers registers the transaction search handlers
func (cm *ChromemManager) RegisterTransactionSearchHandlers() error {
	http.HandleFunc("/api/transactions/search", HandleSearchTransactions(cm))
	return nil
}

// SearchTransactionsChromem searches transactions in ChromemDB
func (m *ChromemManager) SearchTransactionsChromem(
	ctx context.Context,
	semanticQuery string,
	nResults int,
	page int, // For offset calculation
	txType string,
	fromAddr string,
	toAddr string,
	startTime int64, // Unix timestamp
	endTime int64, // Unix timestamp
	minAmount uint64,
	maxAmount uint64,
) ([]ApiTransaction, error) {
	m.mu.RLock() // Use RLock for read operations
	defer m.mu.RUnlock()

	// Prepare query options
	options := chromem.QueryOptions{
		Where: make(map[string]string),
	}

	// Add filters
	if txType != "" {
		options.Where["Type"] = txType
	}
	if fromAddr != "" {
		options.Where["From"] = fromAddr
	}
	if toAddr != "" {
		options.Where["To"] = toAddr
	}

	// Handle time range
	if startTime > 0 || endTime > 0 {
		timeRange := ""
		if startTime > 0 {
			timeRange += fmt.Sprintf(">=%d", startTime)
		}
		if endTime > 0 {
			if startTime > 0 {
				timeRange += ","
			}
			timeRange += fmt.Sprintf("<=%d", endTime)
		}
		options.Where["Timestamp"] = timeRange
	}

	// Handle amount range
	if minAmount > 0 || maxAmount > 0 {
		amountRange := ""
		if minAmount > 0 {
			amountRange += fmt.Sprintf(">=%d", minAmount)
		}
		if maxAmount > 0 {
			if minAmount > 0 {
				amountRange += ","
			}
			amountRange += fmt.Sprintf("<=%d", maxAmount)
		}
		options.Where["Value"] = amountRange
	}

	// Ensure we have at least one filter if no semantic query
	if semanticQuery == "" && len(options.Where) == 0 {
		return nil, fmt.Errorf("search requires a semantic query or at least one filter")
	}

	// Query ChromemDB with options
	// Use nResults directly in the Query call instead of storing it in options.NResults
	results, err := m.transactionCollection.Query(ctx, semanticQuery, page*nResults, options.Where, nil)
	if err != nil {
		return nil, fmt.Errorf("chromemdb query failed: %w", err)
	}

	// Apply pagination (manual slicing after fetching enough results)
	startIndex := (page - 1) * nResults
	endIndex := startIndex + nResults
	if startIndex >= len(results) {
		return []ApiTransaction{}, nil // Page out of bounds
	}
	if endIndex > len(results) {
		endIndex = len(results)
	}

	paginatedResults := results[startIndex:endIndex]

	// Convert results to API transactions
	apiResults := make([]ApiTransaction, len(paginatedResults))
	for i, doc := range paginatedResults {
		// Extract metadata fields directly from document
		var txHash, blockHash, from, to, txType string
		var blockNumber, value, timestamp, fee float64

		if doc.Metadata != nil {
			if val, exists := doc.Metadata["TransactionHash"]; exists {
				txHash = val
			}
			if val, exists := doc.Metadata["BlockHash"]; exists {
				blockHash = val
			}
			if val, exists := doc.Metadata["From"]; exists {
				from = val
			}
			if val, exists := doc.Metadata["To"]; exists {
				to = val
			}
			if val, exists := doc.Metadata["Type"]; exists {
				txType = val
			}
			if val, exists := doc.Metadata["BlockNumber"]; exists {
				if num, err := strconv.ParseFloat(val, 64); err == nil {
					blockNumber = num
				}
			}
			if val, exists := doc.Metadata["Value"]; exists {
				if num, err := strconv.ParseFloat(val, 64); err == nil {
					value = num
				}
			}
			if val, exists := doc.Metadata["Timestamp"]; exists {
				if num, err := strconv.ParseFloat(val, 64); err == nil {
					timestamp = num
				}
			}
			if val, exists := doc.Metadata["Fee"]; exists {
				if num, err := strconv.ParseFloat(val, 64); err == nil {
					fee = num
				}
			}
		}
		// Convert float64 numbers back to uint64/int64
		valueUint := uint64(value)
		timestampInt := int64(timestamp)
		feeUint := uint64(fee)

		// Create API transaction
		apiResults[i] = ApiTransaction{
			TransactionHash: txHash,
			BlockHash:       blockHash,
			BlockNumber:     uint64(blockNumber),
			From:            from,
			To:              to, // To can be empty string
			Value:           valueUint,
			Type:            txType,
			Timestamp:       timestampInt,
			Fee:             feeUint,
			Data:            doc.Content,
		}

		// Add similarity score if semantic query was used
		if semanticQuery != "" {
			score := float32(1.0) // Default score since chromem-go doesn't provide distances
			apiResults[i].SimilarityScore = &score
		}
	}

	return apiResults, nil
}
