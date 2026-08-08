package drq

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// defaultOracleSocketPath mirrors this package's own getOracleSocketPath()
// (internal/app/app.go) — kept as a local, independent constant rather than
// imported from internal/app, since app is the top-level orchestrator and a
// leaf package like drq importing it back would run the dependency the
// wrong way (and risks a future import cycle once SkillMintingProtocol
// gets real production wiring into app.go).
const defaultOracleSocketPath = "/var/lib/knirvserver/sockets/oracle.sock"

// knirvoracleSocketPath resolves KNIRVORACLE's Unix domain socket for
// internal, service-to-service calls, the same reasoning as
// knirvchainSocketPath in knirvchain_client.go: DRQ's skill minting is an
// unattended background process with no user session, so it talks directly
// to KNIRVORACLE's local socket rather than any public/TCP route,
// authenticating with the shared KNIRV_INTERNAL_AUTH_TOKEN service key.
func knirvoracleSocketPath() string {
	if explicit := strings.TrimSpace(os.Getenv("KNIRV_ORACLE_SOCKET_PATH")); explicit != "" {
		return explicit
	}
	if appDataDir := strings.TrimSpace(os.Getenv("KNIRV_APP_DATA_DIR")); appDataDir != "" {
		return filepath.Join(appDataDir, "sockets", "oracle.sock")
	}
	return defaultOracleSocketPath
}

// knirvoracleUnixHTTPClient mirrors knirvchainUnixHTTPClient.
func knirvoracleUnixHTTPClient(socketPath string, timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
			},
		},
	}
}

// callOracleSkillEndpoint POSTs an internal-service-token-gated request to
// one of KNIRVORACLE's /oracle/v3/skills/* routes over its Unix socket, and
// decodes the JSON response into out (when non-nil).
func callOracleSkillEndpoint(path string, request any, out any) error {
	token := strings.TrimSpace(os.Getenv("KNIRV_INTERNAL_AUTH_TOKEN"))
	if token == "" {
		return errors.New("KNIRV_INTERNAL_AUTH_TOKEN is not configured")
	}

	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("marshal request for %s: %w", path, err)
	}

	httpRequest, err := http.NewRequest(http.MethodPost, knirvchainSocketSchemeAndHost+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request for %s: %w", path, err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("X-KNIRV-Internal-Token", token)

	client := knirvoracleUnixHTTPClient(knirvoracleSocketPath(), 20*time.Second)
	response, err := client.Do(httpRequest)
	if err != nil {
		return fmt.Errorf("call KNIRVORACLE %s: %w", path, err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("read KNIRVORACLE %s response: %w", path, err)
	}
	if response.StatusCode/100 != 2 {
		return fmt.Errorf("KNIRVORACLE rejected %s (%d): %s", path, response.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	if out != nil {
		if err := json.Unmarshal(responseBody, out); err != nil {
			return fmt.Errorf("decode KNIRVORACLE %s response: %w", path, err)
		}
	}
	return nil
}

// VerifySkillNode asks KNIRVORACLE to validate skillNode (structurally, and
// against its KNIRVCHAIN bundle receipt when one is already present in
// skillNode.ValidationProof) before it's treated as canonical. See
// routes/skill_economics.go's handleSkillVerify for exactly what's checked.
func (koc *KNIRVORACLEClient) VerifySkillNode(skillNode *SkillNode) (bool, error) {
	if skillNode == nil {
		return false, errors.New("skill node is required")
	}
	request := struct {
		SkillID        string   `json:"skill_id"`
		Creator        string   `json:"creator"`
		Description    string   `json:"description"`
		ResolvesErrors []string `json:"resolves_errors"`
		CodePackageURI string   `json:"code_package_uri"`
		BundleReceipt  []byte   `json:"bundle_receipt,omitempty"`
	}{
		SkillID:        skillNode.ID,
		Creator:        skillNode.Creator,
		Description:    skillNode.Description,
		ResolvesErrors: skillNode.ResolvesErrors,
		CodePackageURI: skillNode.CodePackageURI,
		BundleReceipt:  skillNode.ValidationProof,
	}
	var response struct {
		Verified bool   `json:"verified"`
		Reason   string `json:"reason"`
	}
	if err := callOracleSkillEndpoint("/oracle/v3/skills/verify", request, &response); err != nil {
		return false, fmt.Errorf("verify skill node with KNIRVORACLE: %w", err)
	}
	if !response.Verified {
		return false, fmt.Errorf("KNIRVORACLE rejected skill %s: %s", skillNode.ID, response.Reason)
	}
	return true, nil
}

// skillCanonicalizationBurnCostNRN returns the flat NRN cost burned to
// canonicalize a skill on KNIRVORACLE, mirroring KNIRVCHAIN's
// eventBundleMintCostNRN() pattern (env override + fixed default) — there
// is no existing per-skill pricing model to inherit from otherwise.
func skillCanonicalizationBurnCostNRN() uint64 {
	if raw := strings.TrimSpace(os.Getenv("KNIRV_SKILL_BURN_COST_NRN")); raw != "" {
		if cost, err := strconv.ParseUint(raw, 10, 64); err == nil {
			return cost
		}
	}
	return 50
}

// BurnNRNForSkill burns skillCanonicalizationBurnCostNRN() NRN from
// skillNode.Creator as the cost of canonicalizing the skill.
func (koc *KNIRVORACLEClient) BurnNRNForSkill(skillNode *SkillNode) error {
	if skillNode == nil {
		return errors.New("skill node is required")
	}
	creator := strings.TrimSpace(skillNode.Creator)
	if creator == "" {
		return errors.New("skill node creator is required to burn NRN for it")
	}
	request := struct {
		From    string `json:"from"`
		SkillID string `json:"skill_id"`
		Amount  string `json:"amount"`
	}{
		From:    creator,
		SkillID: skillNode.ID,
		Amount:  fmt.Sprintf("%d", skillCanonicalizationBurnCostNRN()),
	}
	if err := callOracleSkillEndpoint("/oracle/v3/skills/burn", request, nil); err != nil {
		return fmt.Errorf("burn NRN for skill on KNIRVORACLE: %w", err)
	}
	return nil
}

// GrantOwnershipRights registers rights as a durable record on KNIRVORACLE.
func (koc *KNIRVORACLEClient) GrantOwnershipRights(rights SkillOwnershipRights) error {
	if strings.TrimSpace(rights.SkillID) == "" || strings.TrimSpace(rights.AgentID) == "" {
		return errors.New("skill id and agent id are required to grant ownership rights")
	}
	request := struct {
		SkillID       string  `json:"skill_id"`
		AgentID       string  `json:"agent_id"`
		InvocationFee float64 `json:"invocation_fee"`
		Perpetual     bool    `json:"perpetual"`
	}{
		SkillID:       rights.SkillID,
		AgentID:       rights.AgentID,
		InvocationFee: rights.InvocationFee,
		Perpetual:     rights.Perpetual,
	}
	if err := callOracleSkillEndpoint("/oracle/v3/skills/ownership", request, nil); err != nil {
		return fmt.Errorf("grant ownership rights on KNIRVORACLE: %w", err)
	}
	return nil
}

// PayBounty pays amount NRN to agentID as a reward for contributing
// solutions to skillID's resolved error cluster.
func (koc *KNIRVORACLEClient) PayBounty(agentID, skillID string, amount uint64) error {
	if strings.TrimSpace(agentID) == "" {
		return errors.New("agent id is required to pay a bounty")
	}
	if amount == 0 {
		return nil
	}
	request := struct {
		To      string `json:"to"`
		SkillID string `json:"skill_id"`
		Amount  string `json:"amount"`
		Reason  string `json:"reason,omitempty"`
	}{
		To:      agentID,
		SkillID: skillID,
		Amount:  fmt.Sprintf("%d", amount),
		Reason:  "drq-bounty-share",
	}
	if err := callOracleSkillEndpoint("/oracle/v3/skills/bounty", request, nil); err != nil {
		return fmt.Errorf("pay bounty on KNIRVORACLE: %w", err)
	}
	return nil
}
