package knirvbao

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"time"
)

type baoAPI struct{ client *http.Client }

func newBaoAPI(socket string) *baoAPI {
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}
	return &baoAPI{client: &http.Client{Transport: transport, Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

// request deliberately omits response bodies from errors: bootstrap replies
// can contain recovery keys or tokens and must never reach logs.
func (a *baoAPI) request(ctx context.Context, method, path, token string, body any, out any) (int, error) {
	var input io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		input = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, "http://openbao/v1/"+path, input)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("X-Vault-Token", token)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("OpenBao request %s failed: %w", path, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return resp.StatusCode, err
	}
	defer clear(data)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, fmt.Errorf("OpenBao request %s returned HTTP %d", path, resp.StatusCode)
	}
	if out != nil && len(data) != 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return resp.StatusCode, errors.New("invalid OpenBao response")
		}
	}
	return resp.StatusCode, nil
}

type sealStatus struct {
	Initialized bool `json:"initialized"`
	Sealed      bool `json:"sealed"`
}

func (a *baoAPI) sealStatus(ctx context.Context) (sealStatus, error) {
	var status sealStatus
	_, err := a.request(ctx, http.MethodGet, "sys/seal-status", "", nil, &status)
	return status, err
}

// initialize returns the bootstrap token and the one-time recovery keys. The
// caller must keep both only in memory long enough to persist and must never
// log them.
func (a *baoAPI) initialize(ctx context.Context) (string, []string, error) {
	var response struct {
		RootToken    string   `json:"root_token"`
		RecoveryKeys []string `json:"recovery_keys"`
	}
	_, err := a.request(ctx, http.MethodPut, "sys/init", "", map[string]int{"recovery_shares": 1, "recovery_threshold": 1}, &response)
	if err != nil {
		return "", nil, err
	}
	if response.RootToken == "" {
		return "", nil, errors.New("OpenBao initialization returned no bootstrap token")
	}
	return response.RootToken, response.RecoveryKeys, nil
}

// Policy paths are explicit. The OpenBao plugin's key policy remains an
// independent server-side limit on the allowed domains, chains and fees.
var bootstrapPolicies = []struct{ Name, Policy string }{
	{"gateway", `path "knirv-kv/data/node/cloudflare" { capabilities = ["read"] }`},
	{"oracle", `path "knirv-kv/data/node/payments" { capabilities = ["read"] }
path "knirv-kv/data/node/validator" { capabilities = ["read"] }
path "knirv/sign/direct/root-owner" { capabilities = ["update"] }
path "knirv/sign/message/root-owner" { capabilities = ["update"] }`},
	{"validation", `path "knirv/sign/direct/checkpoint-signer" { capabilities = ["update"] }
path "knirv/sign/message/checkpoint-signer" { capabilities = ["update"] }
path "knirv/keys/checkpoint-signer" { capabilities = ["read"] }`},
	{"backend", `path "knirv-kv/data/node/cloudflare" { capabilities = ["read"] }
path "knirv-kv/data/node/payments" { capabilities = ["read"] }
path "knirv-kv/data/node/llm" { capabilities = ["read"] }
path "knirv-kv/data/node/auth" { capabilities = ["read"] }
path "knirv-kv/data/node/tls" { capabilities = ["read"] }
path "knirv-kv/data/node/device" { capabilities = ["read"] }
path "knirv-kv/data/node/registration" { capabilities = ["read"] }
path "knirv/sign/direct/failover-identity" { capabilities = ["update"] }
path "knirv/sign/message/failover-identity" { capabilities = ["update"] }
path "knirv/sign/raw/failover-identity" { capabilities = ["update"] }`},
	// oracle-standby: a bootnode's mirrored (failover) Oracle. It may read the
	// validator settings and sign replication-pull on bootnode-master only; the
	// plugin key policy enforces the domain/purpose allowlist server-side, so a
	// widened token still cannot obtain an active-mode signature. OpenBao.md §4.7.
	{"oracle-standby", `path "knirv-kv/data/node/validator" { capabilities = ["read"] }
path "knirv/sign/message/bootnode-master" { capabilities = ["update"] }`},
	// oracle-active: the oracle policy plus the signer the node is acting as. A Root
	// always runs active; a bootnode only after promotion swaps its AppRole.
	{"oracle-active", `path "knirv-kv/data/node/payments" { capabilities = ["read"] }
path "knirv-kv/data/node/validator" { capabilities = ["read"] }
path "knirv/sign/direct/root-owner" { capabilities = ["update"] }
path "knirv/sign/message/root-owner" { capabilities = ["update"] }
path "knirv/sign/direct/failover-identity" { capabilities = ["update"] }
path "knirv/sign/message/failover-identity" { capabilities = ["update"] }`},
}

// Canonical AppRole / policy names created by bootstrap. The launcher and the
// bootstrap must agree on these names exactly, so they are defined once here and
// referenced by both packages (bao_handoff.go derives its AppRole names from these).
const (
	RoleGateway       = "gateway"
	RoleOracle        = "oracle"
	RoleValidation    = "validation"
	RoleBackend       = "backend"
	RoleOracleStandby = "oracle-standby"
	RoleOracleActive  = "oracle-active"
)

// BootstrapRoleNames returns the AppRole / policy names the bootstrap creates.
func BootstrapRoleNames() []string {
	names := make([]string, 0, len(bootstrapPolicies))
	for _, p := range bootstrapPolicies {
		names = append(names, p.Name)
	}
	return names
}

// BootstrapRolePaths returns the OpenBao paths granted by the named role's policy.
// It is parsed from the canonical bootstrapPolicies definition so ACL tests derive
// their expectations from a single source of truth rather than a disconnected copy.
func BootstrapRolePaths(role string) ([]string, bool) {
	for _, p := range bootstrapPolicies {
		if p.Name == role {
			return parsePolicyPaths(p.Policy), true
		}
	}
	return nil, false
}

var policyPathRe = regexp.MustCompile(`path\s+"([^"]+)"`)

func parsePolicyPaths(policy string) []string {
	var paths []string
	for _, m := range policyPathRe.FindAllStringSubmatch(policy, -1) {
		paths = append(paths, m[1])
	}
	return paths
}

// bootstrap finishes the vault configuration (mounts, policies, roles) and then
// invokes the beforeFinish hook, which the launcher uses for AppRole / service-identity
// work. The persisted admin token is intentionally NOT revoked: owner decision OB-R7
// keeps it available for later AppRole work and interrupted-bootstrap recovery, and no
// AppRole substitute exists without the owner's sign-off.
func (a *baoAPI) bootstrap(ctx context.Context, token string, beforeFinish func(context.Context, *baoAPI, string) error) error {
	if token == "" {
		return errors.New("missing OpenBao bootstrap token")
	}
	if err := a.ensureMount(ctx, token, "sys/mounts", "knirv", "knirv", map[string]any{"type": "knirv", "plugin_version": "v0.1.0"}); err != nil {
		return err
	}
	if err := a.ensureMount(ctx, token, "sys/mounts", "knirv-kv", "kv", map[string]any{"type": "kv", "options": map[string]string{"version": "2"}}); err != nil {
		return err
	}
	if err := a.ensureMount(ctx, token, "sys/auth", "approle", "approle", map[string]any{"type": "approle"}); err != nil {
		return err
	}
	for _, policy := range bootstrapPolicies {
		if _, err := a.request(ctx, http.MethodPut, "sys/policies/acl/"+policy.Name, token, map[string]string{"policy": policy.Policy}, nil); err != nil {
			return err
		}
		role := map[string]any{"token_policies": []string{policy.Name}, "token_no_default_policy": true, "secret_id_num_uses": 1, "secret_id_ttl": "5m", "token_ttl": "15m", "token_max_ttl": "1h"}
		if _, err := a.request(ctx, http.MethodPost, "auth/approle/role/"+policy.Name, token, role, nil); err != nil {
			return err
		}
	}
	if beforeFinish != nil {
		if err := beforeFinish(ctx, a, token); err != nil {
			return err
		}
	}
	return nil
}

// Existing mounts are checked before use so a transient bootstrap failure
// can be retried with the same in-memory token without widening permissions.
func (a *baoAPI) ensureMount(ctx context.Context, token, base, name, kind string, body any) error {
	var response struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if _, err := a.request(ctx, http.MethodGet, base, token, nil, &response); err != nil {
		return err
	}
	if raw, ok := response.Data[name+"/"]; ok {
		var existing struct {
			Type           string            `json:"type"`
			Options        map[string]string `json:"options"`
			PluginVersion  string            `json:"plugin_version"`
			RunningVersion string            `json:"running_plugin_version"`
		}
		if err := json.Unmarshal(raw, &existing); err != nil {
			return errors.New("invalid OpenBao mount metadata")
		}
		if existing.Type != kind {
			return fmt.Errorf("OpenBao mount %s has unexpected type", name)
		}
		if kind == "kv" && existing.Options["version"] != "2" {
			return errors.New("OpenBao KV mount is not version 2")
		}
		if kind == "knirv" && existing.PluginVersion != "v0.1.0" && existing.RunningVersion != "v0.1.0" {
			return errors.New("OpenBao signing plugin version mismatch")
		}
		return nil
	}
	_, err := a.request(ctx, http.MethodPost, base+"/"+name, token, body, nil)
	return err
}
