package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVSHELL/dvepod/internal/tee"
)

type KNIRVAgent struct {
	ctx      *tee.Context
	endpoint string
	mode     string
	client   *http.Client
}

func New(ctx *tee.Context) *KNIRVAgent {
	return &KNIRVAgent{
		ctx:    ctx,
		mode:   "solo",
		client: &http.Client{},
	}
}

func (a *KNIRVAgent) SetEndpoint(url string) {
	a.endpoint = url
	a.mode = "tethered"
}

func (a *KNIRVAgent) Mode() string {
	return a.mode
}

func (a *KNIRVAgent) Register(att tee.Attestation) (string, error) {
	if a.endpoint == "" {
		return "", fmt.Errorf("no endpoint set — call SetEndpoint first")
	}

	payload := map[string]interface{}{
		"attestation": att,
		"node_id":     a.ctx.NodeID,
		"tee_type":    a.ctx.TEEType,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal registration: %w", err)
	}

	url := strings.TrimRight(a.endpoint, "/") + "/api/dve/pod/register"
	resp, err := a.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("registration request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read registration response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("registration failed (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse registration response: %w", err)
	}

	if result.SessionID == "" {
		return "", fmt.Errorf("registration response missing session_id")
	}

	return result.SessionID, nil
}

func (a *KNIRVAgent) Query(query string) (string, error) {
	if a.mode == "solo" {
		return a.soloResponse(query), nil
	}
	return a.tetheredQuery(query)
}

func (a *KNIRVAgent) soloResponse(query string) string {
	queryLower := strings.ToLower(query)

	switch {
	case strings.Contains(queryLower, "skill"):
		return `KNIRV Skill Nodes

Skill Nodes emerge from ErrorNodes through a mining process on KNIRVCHAIN. 
When an error is unresolvable by existing skills, it becomes an ErrorNode. 
Human Architects submit TRL-compatible datasets via KNIRVARENA. 
The HERO Model reads all submitted datasets and skill.md files, attempts error resolution, 
and distributes Compute rewards based on dataset contribution scores. 
Resolved errors mint skill.md files stored permanently on-chain.

I'm running in solo mode. Dock to a KNIRVSERVER with 'dock <url>' to unlock full capabilities.`

	case strings.Contains(queryLower, "dve"):
		return `Decentralized Virtual Environments

DVEs are isolated execution containers acting as TEE nodes in the KNIRV network. 
Each DVE runs a KNIRVAGENT supervisor process, communicates via Unix socket at 
/var/run/knirv/dve-{id}.sock, and can be addressed globally via DVEURI. 

You're currently running inside a DVE Pod — a portable WASM-native DVE that can 
run in any browser or WASI runtime.`

	case strings.Contains(queryLower, "nrn") || strings.Contains(queryLower, "token"):
		return `NRN Tokens

NRN (Node Resource Network) tokens are the compute currency of the KNIRV network. 
DVE operators earn NRN by providing validated compute, resolving error nodes, 
and contributing skill datasets. Tokens are staked to register DVEs and slashed on 
misbehavior. Your DVE Pod can accumulate NRN rewards when docked and running 
validated tasks.`

	default:
		return fmt.Sprintf(`I'm running in solo mode. In solo mode I can answer questions, run tools, and manage your workspace — all without a network connection.

%s

Dock to a KNIRVSERVER with 'dock <url>' to unlock full capabilities including inner agent sessions, on-chain task submission, and NRN rewards.`, a.ctx.NodeID)
	}
}

func (a *KNIRVAgent) tetheredQuery(query string) (string, error) {
	if a.endpoint == "" {
		return a.soloResponse(query), nil
	}

	payload := map[string]interface{}{
		"query":   query,
		"node_id": a.ctx.NodeID,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return a.soloResponse(query), nil
	}

	url := strings.TrimRight(a.endpoint, "/") + "/api/v1/dve/agent/query"
	resp, err := a.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return a.soloResponse(query) + "\n\n(remote query failed, falling back to solo mode)", nil
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return a.soloResponse(query), nil
	}

	var result struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return a.soloResponse(query), nil
	}

	return result.Response, nil
}
