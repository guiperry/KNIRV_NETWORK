package bridge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/knirvchain/internal/blockchain"
)

type KNIRVGraphBridge struct {
	apiURL     string
	apiKey     string
	httpClient *http.Client
}

func NewKNIRVGraphBridge(apiURL, apiKey string) *KNIRVGraphBridge {
	return &KNIRVGraphBridge{
		apiURL: apiURL,
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type GraphTransaction struct {
	Source        string                    `json:"source"`
	Type          blockchain.MemoryCategory `json:"type"`
	Data          GraphData                 `json:"data"`
	Timestamp     int64                     `json:"timestamp"`
	Relationships []Relationship            `json:"relationships"`
}

type GraphData struct {
	BlockID        string    `json:"block_id"`
	GLBRef         string    `json:"glb_ref"`
	ContentSummary string    `json:"content_summary"`
	SemanticVector []float32 `json:"semantic_vector"`
	Tags           []string  `json:"tags"`
}

type Relationship struct {
	Type   string `json:"type"`
	Target string `json:"target"`
}

func (b *KNIRVGraphBridge) SendTransaction(
	ctx context.Context,
	block *blockchain.Block,
) error {
	summary := b.extractSummary(block)
	relationships := b.extractRelationships(block)

	transaction := GraphTransaction{
		Source: "KNIRVBASE",
		Type:   block.Category,
		Data: GraphData{
			BlockID:        block.BlockID.String(),
			GLBRef:         block.PayloadHash,
			ContentSummary: summary,
			SemanticVector: block.SemanticVector,
			Tags:           []string{},
		},
		Timestamp:     block.Timestamp,
		Relationships: relationships,
	}

	payload, err := json.Marshal(transaction)
	if err != nil {
		return fmt.Errorf("failed to marshal transaction: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		fmt.Sprintf("%s/api/v1/transaction", b.apiURL),
		bytes.NewReader(payload),
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", b.apiKey))

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (b *KNIRVGraphBridge) extractSummary(block *blockchain.Block) string {
	return fmt.Sprintf("Memory block %s", block.BlockID.String())
}

func (b *KNIRVGraphBridge) extractRelationships(block *blockchain.Block) []Relationship {
	relationships := []Relationship{}

	if block.PrevHash != "" {
		relationships = append(relationships, Relationship{
			Type:   "FOLLOWS",
			Target: block.PrevHash,
		})
	}

	return relationships
}
