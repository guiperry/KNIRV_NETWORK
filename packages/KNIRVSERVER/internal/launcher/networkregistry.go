package launcher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"knirv-server/pkg/embedded/validationchain"
)

// Network registry integration (pi_2d2_bond_server, owner tx-chain address
// registry decision). Every node publishes its Transaction Chain wallet
// address (derived from the same checkpoint signer identity used for funding)
// to REGISTRY.KNIRV.NETWORK so that any node — in particular a boot node
// completing its bond — can resolve another node's (e.g. the root's) tx-chain
// address without a hardcoded flag.
//
// The worker now accepts `txChainAddress` on POST /register and returns it on
// /lookup, /nodes, /bootnodes and /status. See OpenBao.md / d14_server_handoff.md.

const (
	// networkRegistryEnvOverride lets local/dev point the client at `wrangler dev`.
	networkRegistryEnvOverride = "KNIRV_REGISTRY_URL"
	networkRegistryDefaultURL = "https://registry.knirv.com"

	// nodeRegistryInterval is the periodic re-registration cadence. The worker
	// prunes nodes after nodeTtlMs (default 1h), so a 15-minute heartbeat keeps
	// the entry alive between restarts.
	nodeRegistryInterval = 15 * time.Minute

	// nodeRegistrationIDEnv mirrors the env the launcher sets from boot.key's
	// RegistrationID (launcher.go). It is the operator-assigned network identity
	// used as the registry `chainID` key.
	nodeRegistrationIDEnv = "KNIRV_NODE_REGISTRATION_ID"
)

// nodeRegistryBaseURL resolves the registry base URL, honouring the same
// KNIRV_REGISTRY_URL override convention used by the failover registryclient.
func nodeRegistryBaseURL() string {
	if override := strings.TrimSpace(os.Getenv(networkRegistryEnvOverride)); override != "" {
		return strings.TrimRight(override, "/")
	}
	return networkRegistryDefaultURL
}

// validatedRegistryURL appends suffix to base and requires HTTPS, permitting plain
// HTTP only for an explicit loopback test endpoint (127.0.0.1 / localhost / ::1).
// The node registry and the root-registry publisher both must reject plaintext
// endpoints so a misconfigured or hostile KNIRV_REGISTRY_URL cannot exfiltrate
// the node identity over cleartext.
func validatedRegistryURL(base, suffix string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("registry URL: %w", err)
	}
	if u.Scheme != "https" {
		host := u.Hostname()
		if host != "127.0.0.1" && host != "localhost" && host != "::1" {
			return "", fmt.Errorf("registry URL must be https, got %q", base)
		}
	}
	return strings.TrimRight(base, "/") + suffix, nil
}

// TxChainAddress derives this node's Transaction Chain wallet address from the
// checkpoint signer identity. This is the same address funded in
// fundRootKeyHolderWallet (launcher.go). Returns "" if the signer cannot be
// loaded (e.g. before keys are available).
func TxChainAddress() string {
	signer, err := validationchain.LoadCheckpointSigner()
	if err != nil {
		return ""
	}
	return validationchain.CheckpointAddress(signer)
}

// NodeRegistration is the payload POSTed to POST /register.
type NodeRegistration struct {
	ChainID        string `json:"chainID"`
	IP             string `json:"ip,omitempty"`
	Port           int    `json:"port"`
	Type           string `json:"type"` // "bootnode" | "client"
	PeerID         string `json:"peerID,omitempty"`
	TxChainAddress string `json:"txChainAddress,omitempty"`
}

// RegisterNode posts the node registration (including TxChainAddress) to the
// network registry. It is safe to call repeatedly. The URL must be HTTPS except
// for an explicit loopback test endpoint, and the response is read under a bound.
func RegisterNode(ctx context.Context, baseURL string, reg NodeRegistration) error {
	endpoint, err := validatedRegistryURL(baseURL, "/register")
	if err != nil {
		return err
	}
	body, err := json.Marshal(reg)
	if err != nil {
		return fmt.Errorf("encode node registration: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build node registration request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := registryHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("node registration request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, maxRegistryRespBytes))
		return fmt.Errorf("node registration returned %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxRegistryRespBytes))
	return nil
}

// LookupTxChainAddress resolves a node's Transaction Chain address from the
// registry via GET /lookup/{chainID}. Returns ("", nil) when the node is not
// registered.
func LookupTxChainAddress(ctx context.Context, baseURL, chainID string) (string, error) {
	endpoint, err := validatedRegistryURL(baseURL, "/lookup/"+chainID)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("build lookup request: %w", err)
	}
	resp, err := registryHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("lookup request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return "", nil
	}
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("lookup returned %d", resp.StatusCode)
	}
	var out struct {
		TxChainAddress string `json:"txChainAddress"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxRegistryRespBytes)).Decode(&out); err != nil {
		return "", fmt.Errorf("decode lookup response: %w", err)
	}
	return out.TxChainAddress, nil
}

// nodeRegistryRegistration builds the registration for this server from its
// resolved role and config. chainID is the operator-assigned node identity; if
// unset the node cannot be keyed in the registry and registration is skipped.
func (app *ServerApp) nodeRegistryRegistration() (NodeRegistration, bool) {
	chainID := strings.TrimSpace(os.Getenv(nodeRegistrationIDEnv))
	if chainID == "" {
		return NodeRegistration{}, false
	}
	addr := TxChainAddress()
	nodeType := "client"
	if resolved := app.roleResolution.Load(); resolved != nil {
		if resolved.IsRoot() || resolved.IsBootnode() {
			nodeType = "bootnode"
		}
	}
	return NodeRegistration{
		ChainID:        chainID,
		IP:             app.config.Host,
		Port:           app.config.Port,
		Type:           nodeType,
		TxChainAddress: addr,
	}, true
}

// startNodeRegistry derives this node's tx-chain address and publishes it to the
// network registry immediately and on a 15-minute heartbeat. Failures are logged
// and never fatal: a missing registry must not block node startup. The bond
// completion flow (pi_2d2_bond_server part b) depends on the root node's entry
// being present here.
func (app *ServerApp) startNodeRegistry(ctx context.Context) {
	reg, ok := app.nodeRegistryRegistration()
	if !ok {
		log.Printf("Warning: %s not set; skipping network registry registration (tx-chain address will not be published)", nodeRegistrationIDEnv)
		return
	}
	baseURL := nodeRegistryBaseURL()

	registerOnce := func() {
		if err := RegisterNode(ctx, baseURL, reg); err != nil {
			log.Printf("Warning: network registry registration failed for %s: %v", reg.ChainID, err)
			return
		}
		log.Printf("Registered node %s (type=%s) with tx-chain address %s to %s", reg.ChainID, reg.Type, reg.TxChainAddress, baseURL)
	}

	registerOnce()
	ticker := time.NewTicker(nodeRegistryInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			registerOnce()
		}
	}
}
