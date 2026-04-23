package validationchain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type CommitValidationResultRequest struct {
	ValidationID string                 `json:"validation_id"`
	NodeID       string                 `json:"node_id,omitempty"`
	ResultType   string                 `json:"result_type,omitempty"`
	Payload      map[string]interface{} `json:"payload,omitempty"`
}

type PolicyCommitRequest struct {
	Name      string                 `json:"name"`
	Type      string                 `json:"type,omitempty"`
	TargetDVE string                 `json:"target_dve,omitempty"`
	Rules     map[string]interface{} `json:"rules,omitempty"`
	Priority  int                    `json:"priority,omitempty"`
	Enabled   bool                   `json:"enabled"`
}

type EvidenceAnchorRequest struct {
	EvidenceID   string                 `json:"evidence_id,omitempty"`
	EvidenceType string                 `json:"evidence_type,omitempty"`
	NodeID       string                 `json:"node_id,omitempty"`
	ValidationID string                 `json:"validation_id,omitempty"`
	Payload      map[string]interface{} `json:"payload,omitempty"`
	Signature    string                 `json:"signature,omitempty"`
	Algorithm    string                 `json:"algorithm,omitempty"`
	PublicKey    string                 `json:"public_key,omitempty"`
}

type Client struct {
	baseURL string
	client  *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Client) postJSON(path string, payload interface{}, out interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal validation-chain payload: %w", err)
	}

	resp, err := c.client.Post(c.baseURL+path, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("validation chain request failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	if out == nil {
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("failed to decode validation-chain response: %w", err)
	}
	return nil
}

func (c *Client) SubmitTransaction(tx interface{}) (string, error) {
	payload, err := json.Marshal(tx)
	if err != nil {
		return "", fmt.Errorf("failed to marshal validation-chain transaction: %w", err)
	}

	req := map[string]string{
		"data":      string(payload),
		"signature": "validation-chain",
	}
	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal validation-chain request: %w", err)
	}

	resp, err := c.client.Post(c.baseURL+"/send_txn", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("failed to submit to validation chain: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("validation chain submit failed with status %d", resp.StatusCode)
	}

	var result struct {
		TransactionHash string `json:"transaction_hash"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode validation-chain response: %w", err)
	}
	return result.TransactionHash, nil
}

func (c *Client) CommitValidationResult(req CommitValidationResultRequest) (string, error) {
	var result struct {
		TransactionHash string `json:"transaction_hash"`
	}

	if err := c.postJSON("/validation/commit", req, &result); err == nil && result.TransactionHash != "" {
		return result.TransactionHash, nil
	}

	return c.SubmitTransaction(map[string]interface{}{
		"type": "validation_commit",
		"data": req,
	})
}

func (c *Client) CommitPolicy(req PolicyCommitRequest) (string, error) {
	var result struct {
		TransactionHash string `json:"transaction_hash"`
	}

	if err := c.postJSON("/policy/commit", req, &result); err == nil && result.TransactionHash != "" {
		return result.TransactionHash, nil
	}

	return c.SubmitTransaction(map[string]interface{}{
		"type": "policy_commit",
		"data": req,
	})
}

func (c *Client) AnchorEvidencePack(req EvidenceAnchorRequest) (string, error) {
	var result struct {
		TransactionHash string `json:"transaction_hash"`
	}

	if err := c.postJSON("/evidence/anchor", req, &result); err == nil && result.TransactionHash != "" {
		return result.TransactionHash, nil
	}

	return c.SubmitTransaction(map[string]interface{}{
		"type": "evidence_anchor",
		"data": req,
	})
}

func (c *Client) GetRecord(recordID string) (map[string]interface{}, error) {
	resp, err := c.client.Get(c.baseURL + "/records/" + recordID)
	if err != nil {
		return nil, fmt.Errorf("failed to query validation-chain record: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("validation chain record lookup failed with status %d: %s", resp.StatusCode, string(body))
	}

	var record map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&record); err != nil {
		return nil, fmt.Errorf("failed to decode validation-chain record: %w", err)
	}
	return record, nil
}

func (c *Client) GetBlockHeight() (uint64, error) {
	resp, err := c.client.Get(c.baseURL + "/chain/height")
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			var result struct {
				Height uint64 `json:"height"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
				return result.Height, nil
			}
		}
	}

	resp, err = c.client.Get(c.baseURL + "/blocks")
	if err != nil {
		return 0, fmt.Errorf("failed to query validation chain blocks: %w", err)
	}
	defer resp.Body.Close()

	var blocks []json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&blocks); err != nil {
		return 0, fmt.Errorf("failed to decode validation chain blocks: %w", err)
	}
	if len(blocks) == 0 {
		return 0, nil
	}
	return uint64(len(blocks) - 1), nil
}

func (c *Client) Health() error {
	resp, err := c.client.Get(c.baseURL + "/health")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("validation chain health returned status %d", resp.StatusCode)
	}
	return nil
}

type DVEURIRecord struct {
	DVEID      string `json:"dve_id"`
	FullURI    string `json:"full_uri"`
	WalletAddr string `json:"wallet_address"`
	CreatedAt  int64  `json:"created_at"`
	TxHash     string `json:"tx_hash"`
}

func (c *Client) SubmitDVEURI(req DVEURIRecord) (string, error) {
	var result struct {
		TransactionHash string `json:"transaction_hash"`
	}

	if err := c.postJSON("/dve_uri/submit", req, &result); err == nil && result.TransactionHash != "" {
		return result.TransactionHash, nil
	}

	return c.SubmitTransaction(map[string]interface{}{
		"type": "dve_uri_submit",
		"data": req,
	})
}

func (c *Client) GetDVEURI(dveID string) (*DVEURIRecord, error) {
	resp, err := c.client.Get(c.baseURL + "/dve_uri/" + dveID)
	if err != nil {
		return nil, fmt.Errorf("failed to query DVE URI: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("DVE URI lookup failed with status %d: %s", resp.StatusCode, string(body))
	}

	var record DVEURIRecord
	if err := json.NewDecoder(resp.Body).Decode(&record); err != nil {
		return nil, fmt.Errorf("failed to decode DVE URI record: %w", err)
	}
	return &record, nil
}

func (c *Client) GetDVEURIByWallet(walletAddr string) ([]*DVEURIRecord, error) {
	resp, err := c.client.Get(c.baseURL + "/dve_uri/by_wallet/" + walletAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to query DVE URIs by wallet: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("DVE URI by wallet lookup failed with status %d: %s", resp.StatusCode, string(body))
	}

	var records []*DVEURIRecord
	if err := json.NewDecoder(resp.Body).Decode(&records); err != nil {
		return nil, fmt.Errorf("failed to decode DVE URI records: %w", err)
	}
	return records, nil
}
