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

// defaultChainSocketPath is KNIRVSERVER's well-known fallback chain socket
// location (packages/KNIRVSERVER/pkg/knirvchain/manager.go's
// appDataDir/"sockets"/"chain.sock", where appDataDir defaults to
// /var/lib/knirvserver) — the same default KNIRVGATEWAY's CHAIN_SOCKET_PATH
// and this package's own getOracleSocketPath()/embedding_model.go resolve
// to when nothing more specific is configured.
const defaultChainSocketPath = "/var/lib/knirvserver/sockets/chain.sock"

// knirvchainSocketSchemeAndHost is the placeholder base URL used with the
// Unix-socket-dialing http.Client below — the host is never actually
// resolved over the network, matching GraphRAGEmbedder's "http://unix"
// convention in this same package (embedding_model.go).
const knirvchainSocketSchemeAndHost = "http://unix"

// knirvchainSocketPath resolves KNIRVCHAIN's Unix domain socket for
// internal, service-to-service calls. This intentionally bypasses
// KNIRVGATEWAY's public HTTP proxy: the gateway's /api/v1/event-bundles/mint
// route (packages/KNIRVGATEWAY/internal/server/eventbundle_proxy.go)
// requires a logged-in user's bearer session token on top of the internal
// token, because it fronts the CLI's end-user-triggered decision-event flow
// on the public internet. DRQ's skill minting is an unattended,
// network-wide background process with no user session, so — like every
// other internal-to-internal KNIRV service call (KNIRVGATEWAY→KNIRVCHAIN,
// this package's own KNIRVORACLE/GraphRAG clients) — it talks directly over
// the sibling service's local Unix socket rather than a public/TCP URL,
// authenticating with the same shared KNIRV_INTERNAL_AUTH_TOKEN service key
// KNIRVCHAIN's handler expects (the "explicitly registered service keys for
// unattended node operations" custody model, as opposed to user-action
// controller custody).
func knirvchainSocketPath() string {
	if explicit := strings.TrimSpace(os.Getenv("KNIRV_CHAIN_SOCKET_PATH")); explicit != "" {
		return explicit
	}
	if appDataDir := strings.TrimSpace(os.Getenv("KNIRV_APP_DATA_DIR")); appDataDir != "" {
		return filepath.Join(appDataDir, "sockets", "chain.sock")
	}
	return defaultChainSocketPath
}

// unixHTTPClient builds an http.Client that dials a Unix domain socket
// regardless of the URL host given to it, mirroring this package's
// GraphRAGEmbedder helper of the same name (embedding_model.go) — kept
// separate here (rather than exported/shared) since the two call sites
// don't otherwise share a type.
func knirvchainUnixHTTPClient(socketPath string, timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
			},
		},
	}
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

	httpRequest, err := http.NewRequest(http.MethodPost, knirvchainSocketSchemeAndHost+"/api/v1/event-bundles/mint", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build event bundle mint request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("X-KNIRV-Internal-Token", token)

	client := knirvchainUnixHTTPClient(knirvchainSocketPath(), 20*time.Second)
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
