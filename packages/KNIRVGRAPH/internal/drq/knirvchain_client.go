package drq

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// The types below mirror KNIRVCHAIN's event-bundle wire shape
// (packages/KNIRVCHAIN/internal/blockchain/eventbundle.go). KNIRVGRAPH
// cannot import KNIRVCHAIN's internal package directly (no cross-package
// imports between KNIRV_NETWORK services — communication is HTTP/gRPC/socket
// only), so this is an independent, structurally-identical type, matching
// the same precedent eventbundle.go itself uses for the CLI's resource refs.

const knirvchainEventBundleMintSchema = "knirv.event-bundle.v1"

type knirvchainEventBundleResourceRef struct {
	ID  string `json:"id"`
	Ref string `json:"ref,omitempty"`
}

type knirvchainEventBundleMintRequest struct {
	SchemaVersion string                             `json:"schema_version"`
	EventID       string                             `json:"event_id"`
	SessionID     string                             `json:"session_id"`
	ProjectID     string                             `json:"project_id"`
	EventKind     string                             `json:"event_kind"`
	Skills        []knirvchainEventBundleResourceRef `json:"skills,omitempty"`
	MinterAddress string                             `json:"minter_address"`
}

// knirvchainEventBundleReceipt mirrors eventBundleReceipt. Skill nodes store
// this verbatim (as JSON) in SkillNode.ValidationProof so a verifier can
// later re-fetch and re-check the bundle by EventID/BundleHash, the same
// pattern KNIRVSERVER's native_verifier.go uses for CLI commit proofs.
type knirvchainEventBundleReceipt struct {
	SchemaVersion string    `json:"schema_version"`
	EventID       string    `json:"event_id"`
	BundleHash    string    `json:"bundle_hash"`
	TokenID       string    `json:"token_id"`
	TransactionID string    `json:"transaction_id"`
	BlockID       string    `json:"block_id,omitempty"`
	Final         bool      `json:"final"`
	FinalizedAt   time.Time `json:"finalized_at,omitempty"`
}

// knirvchainDirectURL resolves KNIRVCHAIN's own HTTP API for internal,
// service-to-service calls. This intentionally bypasses KNIRVGATEWAY: the
// gateway's /api/v1/event-bundles/mint proxy requires a logged-in user's
// bearer session token (packages/KNIRVGATEWAY/internal/server/
// eventbundle_proxy.go) because it fronts the CLI's end-user-triggered
// decision-event flow on the public internet. DRQ's skill minting is an
// unattended, network-wide background process with no user session — it
// authenticates with the same shared KNIRV_INTERNAL_AUTH_TOKEN service key
// KNIRVCHAIN itself expects, calling it directly as an internal peer
// (the "explicitly registered service keys for unattended node operations"
// custody model, as opposed to user-action controller custody).
func knirvchainDirectURL() (string, error) {
	if raw := strings.TrimSpace(os.Getenv("KNIRVCHAIN_URL")); raw != "" {
		return strings.TrimRight(raw, "/"), nil
	}
	return "", errors.New("KNIRVCHAIN_URL is not configured; set it to KNIRVCHAIN's internal HTTP API address")
}

// mintSkillEventBundle mints (or, if already minted, fetches) an
// EventBundleNFT on KNIRVCHAIN for skillNode, giving the skill a
// verifiable, Merkle-checkable commit-bundle proof instead of the previous
// no-op mint. The mint is idempotent per skillNode.ID: minting the same
// skill twice returns the existing receipt rather than erroring or
// double-burning NRN, so this is safe to call from either of DRQ's two
// skill-minting code paths (SkillDiscoveryEngine.DiscoverSkill and
// SkillMintingProtocol.MintSkillFromCluster) without coordinating between
// them.
func (kc *KNIRVCHAINClient) mintSkillEventBundle(skillNode *SkillNode) ([]byte, error) {
	if skillNode == nil {
		return nil, errors.New("skill node is required")
	}
	eventID := strings.TrimSpace(skillNode.ID)
	if eventID == "" {
		return nil, errors.New("skill node id is required to mint an event bundle")
	}
	minter := strings.TrimSpace(skillNode.Creator)
	if minter == "" {
		return nil, errors.New("skill node creator (minter address) is required to mint an event bundle")
	}
	token := strings.TrimSpace(os.Getenv("KNIRV_INTERNAL_AUTH_TOKEN"))
	if token == "" {
		return nil, errors.New("KNIRV_INTERNAL_AUTH_TOKEN is not configured")
	}
	baseURL, err := knirvchainDirectURL()
	if err != nil {
		return nil, err
	}

	request := knirvchainEventBundleMintRequest{
		SchemaVersion: knirvchainEventBundleMintSchema,
		EventID:       eventID,
		SessionID:     eventID,
		ProjectID:     "knirvgraph-drq",
		EventKind:     "decision",
		Skills: []knirvchainEventBundleResourceRef{
			{ID: eventID, Ref: skillNode.CodePackageURI},
		},
		MinterAddress: minter,
	}
	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal event bundle mint request: %w", err)
	}

	httpRequest, err := http.NewRequest(http.MethodPost, baseURL+"/api/v1/event-bundles/mint", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build event bundle mint request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("X-KNIRV-Internal-Token", token)

	client := &http.Client{Timeout: 20 * time.Second}
	response, err := client.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("mint event bundle on KNIRVCHAIN: %w", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read event bundle mint response: %w", err)
	}
	if response.StatusCode/100 != 2 {
		return nil, fmt.Errorf("KNIRVCHAIN rejected event bundle mint (%d): %s", response.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	var receipt knirvchainEventBundleReceipt
	if err := json.Unmarshal(responseBody, &receipt); err != nil {
		return nil, fmt.Errorf("decode event bundle mint receipt: %w", err)
	}
	if receipt.EventID != eventID {
		return nil, fmt.Errorf("KNIRVCHAIN returned a receipt for a different event (%s)", receipt.EventID)
	}

	return responseBody, nil
}
