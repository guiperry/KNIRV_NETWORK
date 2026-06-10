package governance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type GovernanceClient struct {
	oracleURL  string
	serverURL  string
	httpClient *http.Client
}

func NewGovernanceClient(oracleURL, serverURL string) *GovernanceClient {
	return &GovernanceClient{
		oracleURL: oracleURL,
		serverURL: serverURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// DID operations

func (c *GovernanceClient) RegisterDID(docJSON string) error {
	resp, err := c.httpClient.Post(c.oracleURL+"/oracle/v3/did/register", "application/json", strings.NewReader(docJSON))
	if err != nil {
		return fmt.Errorf("did register request failed: %w", err)
	}
	defer resp.Body.Close()
	return checkResponse(resp)
}

func (c *GovernanceClient) ResolveDID(did string) ([]byte, error) {
	resp, err := c.httpClient.Get(c.oracleURL + "/oracle/v3/did/" + did)
	if err != nil {
		return nil, fmt.Errorf("did resolve request failed: %w", err)
	}
	defer resp.Body.Close()
	if err := checkResponse(resp); err != nil {
		return nil, err
	}
	return io.ReadAll(resp.Body)
}

// Identity operations

func (c *GovernanceClient) CreateEnvelope(nodeID, agentID, source string) ([]byte, error) {
	body := map[string]string{"node_id": nodeID, "agent_id": agentID, "source": source}
	data, _ := json.Marshal(body)
	resp, err := c.httpClient.Post(c.serverURL+"/api/v1/governance/identity/envelopes", "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create envelope failed: %w", err)
	}
	defer resp.Body.Close()
	return handleResponse(resp)
}

func (c *GovernanceClient) RevokeIdentity(identityID, nodeID, reason, revokedBy string) ([]byte, error) {
	body := map[string]string{
		"identity_id": identityID,
		"node_id":     nodeID,
		"reason":      reason,
		"revoked_by":  revokedBy,
	}
	data, _ := json.Marshal(body)
	resp, err := c.httpClient.Post(c.serverURL+"/api/v1/governance/identity/revoke", "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("revoke request failed: %w", err)
	}
	defer resp.Body.Close()
	return handleResponse(resp)
}

func (c *GovernanceClient) CheckRevoked(nodeID string) (bool, error) {
	resp, err := c.httpClient.Get(c.serverURL + "/api/v1/governance/identity/revoked/" + nodeID)
	if err != nil {
		return false, fmt.Errorf("check revoked failed: %w", err)
	}
	defer resp.Body.Close()
	var result struct {
		Revoked bool `json:"revoked"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}
	return result.Revoked, nil
}

// Policy operations

func (c *GovernanceClient) NormalizeInput(nodeID, action, actionType string) ([]byte, error) {
	body := map[string]string{"node_id": nodeID, "action": action, "action_type": actionType}
	data, _ := json.Marshal(body)
	resp, err := c.httpClient.Post(c.serverURL+"/api/v1/governance/policy/inputs", "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("normalize input failed: %w", err)
	}
	defer resp.Body.Close()
	return handleResponse(resp)
}

func (c *GovernanceClient) GetContract() ([]byte, error) {
	resp, err := c.httpClient.Get(c.serverURL + "/api/v1/governance/policy/contract")
	if err != nil {
		return nil, fmt.Errorf("get contract failed: %w", err)
	}
	defer resp.Body.Close()
	return handleResponse(resp)
}

// Reliability operations

func (c *GovernanceClient) RecordBreakerSuccess(breakerID string) error {
	resp, err := c.httpClient.Post(c.serverURL+"/api/v1/governance/reliability/breakers/"+breakerID+"/success", "application/json", nil)
	if err != nil {
		return fmt.Errorf("record success failed: %w", err)
	}
	defer resp.Body.Close()
	return checkResponse(resp)
}

func (c *GovernanceClient) RecordBreakerFailure(breakerID string) error {
	resp, err := c.httpClient.Post(c.serverURL+"/api/v1/governance/reliability/breakers/"+breakerID+"/failure", "application/json", nil)
	if err != nil {
		return fmt.Errorf("record failure failed: %w", err)
	}
	defer resp.Body.Close()
	return checkResponse(resp)
}

func (c *GovernanceClient) IsBreakerAllowed(breakerID string) (bool, error) {
	resp, err := c.httpClient.Get(c.serverURL + "/api/v1/governance/reliability/breakers/" + breakerID + "/allow")
	if err != nil {
		return false, fmt.Errorf("breaker check failed: %w", err)
	}
	defer resp.Body.Close()
	var result struct {
		Allowed bool `json:"allowed"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}
	return result.Allowed, nil
}

func (c *GovernanceClient) DefineSLO(name string, target float64) error {
	body := map[string]interface{}{"name": name, "target": target}
	data, _ := json.Marshal(body)
	resp, err := c.httpClient.Post(c.serverURL+"/api/v1/governance/reliability/slos/define", "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("define SLO failed: %w", err)
	}
	defer resp.Body.Close()
	return checkResponse(resp)
}

// Compliance operations

func (c *GovernanceClient) RecordComplianceEvent(nodeID, agentID, eventType, severity string) ([]byte, error) {
	body := map[string]string{
		"node_id":    nodeID,
		"agent_id":   agentID,
		"event_type": eventType,
		"severity":   severity,
	}
	data, _ := json.Marshal(body)
	resp, err := c.httpClient.Post(c.serverURL+"/api/v1/governance/compliance/events", "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("compliance event failed: %w", err)
	}
	defer resp.Body.Close()
	return handleResponse(resp)
}

func (c *GovernanceClient) GetComplianceStatus(nodeID string) ([]byte, error) {
	resp, err := c.httpClient.Get(c.serverURL + "/api/v1/governance/compliance/status?node_id=" + nodeID)
	if err != nil {
		return nil, fmt.Errorf("compliance status failed: %w", err)
	}
	defer resp.Body.Close()
	return handleResponse(resp)
}

func (c *GovernanceClient) VerifyComplianceChain() (bool, error) {
	resp, err := c.httpClient.Get(c.serverURL + "/api/v1/governance/compliance/chain/verify")
	if err != nil {
		return false, fmt.Errorf("verify chain failed: %w", err)
	}
	defer resp.Body.Close()
	var result struct {
		Valid bool `json:"valid"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}
	return result.Valid, nil
}

func checkResponse(resp *http.Response) error {
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed (status %d): %s", resp.StatusCode, string(body))
	}
	return nil
}

func handleResponse(resp *http.Response) ([]byte, error) {
	if err := checkResponse(resp); err != nil {
		return nil, err
	}
	return io.ReadAll(resp.Body)
}
