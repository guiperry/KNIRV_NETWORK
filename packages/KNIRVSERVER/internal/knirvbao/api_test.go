package knirvbao

import (
	"context"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestBootstrapUsesNarrowRoles(t *testing.T) {
	var paths []string
	api := &baoAPI{client: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		paths = append(paths, req.URL.Path)
		if req.Header.Get("X-Vault-Token") != "test-bootstrap-token" {
			t.Error("missing bootstrap authorization")
		}
		if strings.HasPrefix(req.URL.Path, "/v1/auth/approle/role/") {
			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Error(err)
			}
			if !strings.Contains(string(body), `"token_no_default_policy":true`) || !strings.Contains(string(body), `"secret_id_num_uses":1`) {
				t.Error("role grants a default policy or reusable secret ID")
			}
		}
		if req.Method == http.MethodGet {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"data":{}}`)), Header: make(http.Header)}, nil
		}
		return &http.Response{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
	})}}
	if err := api.bootstrap(context.Background(), "test-bootstrap-token", nil); err != nil {
		t.Fatal(err)
	}
	// 3 mounts (GET+POST each) + 6 policies (PUT) + 6 roles (POST) = 18 requests.
	if len(paths) != 18 || !strings.HasSuffix(paths[len(paths)-1], "/auth/approle/role/oracle-active") {
		t.Fatalf("unexpected bootstrap request count or final operation: %d %s", len(paths), paths[len(paths)-1])
	}
	// OB-R7: the persisted admin token is intentionally NOT revoked here; a later
	// owner-approved recovery/control identity is required before any revocation.
	if strings.Contains(strings.Join(paths, " "), "revoke-self") {
		t.Fatal("bootstrap must not revoke the persisted admin token (OB-R7)")
	}
	for _, policy := range bootstrapPolicies {
		if strings.Contains(policy.Policy, "sudo") || strings.Contains(policy.Policy, "delete") || strings.Contains(policy.Policy, "root-owner") && policy.Name != "oracle" && policy.Name != "oracle-active" {
			t.Fatalf("policy %s exceeds its intended permissions", policy.Name)
		}
	}
}

func TestRequestDoesNotExposeErrorBody(t *testing.T) {
	api := &baoAPI{client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusForbidden, Body: io.NopCloser(strings.NewReader("sensitive-test-value")), Header: make(http.Header)}, nil
	})}}
	_, err := api.request(context.Background(), http.MethodGet, "sys/example", "", nil, nil)
	if err == nil || strings.Contains(err.Error(), "sensitive-test-value") {
		t.Fatal("error response body was exposed")
	}
}

// TestValidationPolicyGrantsCheckpointKeyRead covers openbao_2a.4 Blocker 2:
// the validation AppRole must be able to read the checkpoint-signer key
// metadata (so the binary can resolve its service address at startup) but must
// not gain read on unrelated key metadata. A live-vault integration test that
// boots with this token and proves the metadata lookup + direct sign succeed
// while unrelated lookups stay 403 is tracked separately (requires a real
// OpenBao — not started here per the no-live-vault constraint).
func TestValidationPolicyGrantsCheckpointKeyRead(t *testing.T) {
	var validation string
	for _, policy := range bootstrapPolicies {
		if policy.Name == "validation" {
			validation = policy.Policy
		}
	}
	if validation == "" {
		t.Fatal("validation policy missing")
	}
	if !strings.Contains(validation, `path "knirv/keys/checkpoint-signer" { capabilities = ["read"] }`) {
		t.Fatal("validation policy must grant read on knirv/keys/checkpoint-signer")
	}
	if !strings.Contains(validation, `knirv/sign/direct/checkpoint-signer`) ||
		!strings.Contains(validation, `knirv/sign/message/checkpoint-signer`) {
		t.Fatal("validation policy must retain direct/message sign grants")
	}
	// Unrelated key metadata must remain forbidden for the validation role.
	// The only key-metadata path it may reference is checkpoint-signer.
	for _, forbidden := range []string{
		`knirv/keys/root-owner`,
		`knirv/keys/failover-identity`,
		`knirv/keys/oracle`,
	} {
		if strings.Contains(validation, forbidden) {
			t.Fatalf("validation policy must not grant access to %q", forbidden)
		}
	}
	// Exactly one key-metadata path is permitted, and it must be the
	// checkpoint-signer read grant.
	if cnt := strings.Count(validation, `knirv/keys/`); cnt != 1 {
		t.Fatalf("validation policy must reference exactly one knirv/keys/ path, got %d", cnt)
	}
}

// TestRawFailoverGrantOnlyInBackendPolicy confirms the raw compact signing
// endpoint is granted only to the backend/failover AppRole (least-privilege).
// Other roles — oracle, validation, gateway — must not be able to reach it.
func TestRawFailoverGrantOnlyInBackendPolicy(t *testing.T) {
	var backend string
	policies := map[string]string{}
	for _, policy := range bootstrapPolicies {
		policies[policy.Name] = policy.Policy
		if policy.Name == "backend" {
			backend = policy.Policy
		}
	}
	if backend == "" {
		t.Fatal("backend policy missing")
	}
	if !strings.Contains(backend, `path "knirv/sign/raw/failover-identity" { capabilities = ["update"] }`) {
		t.Fatal("backend policy must grant knirv/sign/raw/failover-identity")
	}
	for _, role := range []string{"oracle", "validation", "gateway"} {
		if strings.Contains(policies[role], "knirv/sign/raw/") {
			t.Fatalf("%s policy must not grant raw signing", role)
		}
	}
}

// TestValidatorPolicies covers openbao_2a.6: the standby/active Oracle role split
// (OpenBao.md §4.7). oracle-standby may read node/validator and sign
// bootnode-master (replication-pull, enforced server-side by the plugin), but must
// not be able to sign root-owner. oracle-active adds the failover-identity signer
// to the oracle policy. The launcher swap between them is Task 8.6 and is not
// covered here (requires a live OpenBao).
func TestValidatorPolicies(t *testing.T) {
	policies := map[string]string{}
	for _, policy := range bootstrapPolicies {
		policies[policy.Name] = policy.Policy
	}
	// oracle-standby: read-only on validator settings, sign bootnode-master only.
	if !strings.Contains(policies["oracle-standby"], `path "knirv-kv/data/node/validator" { capabilities = ["read"] }`) {
		t.Fatal("oracle-standby must read node/validator")
	}
	if !strings.Contains(policies["oracle-standby"], `path "knirv/sign/message/bootnode-master" { capabilities = ["update"] }`) {
		t.Fatal("oracle-standby must sign bootnode-master (replication-pull)")
	}
	for _, forbidden := range []string{
		`knirv/sign/direct/root-owner`,
		`knirv/sign/message/root-owner`,
		`knirv/sign/raw/`,
		`knirv/sign/direct/failover-identity`,
	} {
		if strings.Contains(policies["oracle-standby"], forbidden) {
			t.Fatalf("oracle-standby must not grant %q", forbidden)
		}
	}
	// oracle-active: oracle policy + the failover-identity signer.
	for _, want := range []string{
		`knirv/sign/direct/root-owner`,
		`knirv/sign/message/root-owner`,
		`knirv/sign/direct/failover-identity`,
		`knirv/sign/message/failover-identity`,
	} {
		if !strings.Contains(policies["oracle-active"], want) {
			t.Fatalf("oracle-active must grant %q", want)
		}
	}
	// The oracle policy itself gains read on node/validator (no write).
	if !strings.Contains(policies["oracle"], `path "knirv-kv/data/node/validator" { capabilities = ["read"] }`) {
		t.Fatal("oracle policy must read node/validator")
	}
}

// roleCapabilities parses the named bootstrap policy into path -> capabilities. It
// derives expectations from the single canonical bootstrapPolicies source of truth so
// negative tests cannot silently drift from the policy text.
func roleCapabilities(t *testing.T, role string) map[string][]string {
	t.Helper()
	var pol string
	for _, p := range bootstrapPolicies {
		if p.Name == role {
			pol = p.Policy
		}
	}
	if pol == "" {
		t.Fatalf("role %q not found in bootstrapPolicies", role)
	}
	out := map[string][]string{}
	re := regexp.MustCompile(`path\s+"([^"]+)"\s*\{\s*capabilities\s*=\s*\[([^\]]*)\]`)
	for _, m := range re.FindAllStringSubmatch(pol, -1) {
		var caps []string
		for _, c := range strings.Split(m[2], ",") {
			c = strings.TrimSpace(strings.Trim(c, `"`))
			if c != "" {
				caps = append(caps, c)
			}
		}
		out[m[1]] = caps
	}
	return out
}

func hasCap(caps map[string][]string, path, cap string) bool {
	for _, c := range caps[path] {
		if c == cap {
			return true
		}
	}
	return false
}

// TestAppRoleNegativeStandby enforces DEC-1 §2a.6 / OpenBao.md §4.7: a mirrored
// (failover) Oracle may only read validator settings and sign replication-pull. It
// must NOT write node/validator, and must NOT sign root-owner or failover-identity in
// any form. These are vault-independent assertions over the parsed ACL policy.
func TestAppRoleNegativeStandby(t *testing.T) {
	caps := roleCapabilities(t, "oracle-standby")
	if !hasCap(caps, `knirv-kv/data/node/validator`, "read") {
		t.Fatal("oracle-standby must read node/validator")
	}
	if hasCap(caps, `knirv-kv/data/node/validator`, "write") || hasCap(caps, `knirv-kv/data/node/validator`, "update") {
		t.Fatal("oracle-standby must NOT write node/validator")
	}
	if !hasCap(caps, `knirv/sign/message/bootnode-master`, "update") {
		t.Fatal("oracle-standby must sign bootnode-master (replication-pull)")
	}
	for _, forbidden := range []string{
		`knirv/sign/direct/root-owner`,
		`knirv/sign/message/root-owner`,
		`knirv/sign/direct/failover-identity`,
		`knirv/sign/message/failover-identity`,
		`knirv/sign/raw/failover-identity`,
	} {
		if _, ok := caps[forbidden]; ok {
			t.Fatalf("oracle-standby must NOT grant %q", forbidden)
		}
	}
}

// TestAppRoleNegativeActive confirms the active Oracle gains only the intended signers
// (root-owner + failover-identity) on top of the oracle policy, and still never
// receives a default or sudo capability.
func TestAppRoleNegativeActive(t *testing.T) {
	caps := roleCapabilities(t, "oracle-active")
	for _, want := range []string{
		`knirv/sign/direct/root-owner`,
		`knirv/sign/message/root-owner`,
		`knirv/sign/direct/failover-identity`,
		`knirv/sign/message/failover-identity`,
	} {
		if !hasCap(caps, want, "update") {
			t.Fatalf("oracle-active must grant %q", want)
		}
	}
	// Active must still only read validator settings, never write them.
	if hasCap(caps, `knirv-kv/data/node/validator`, "write") || hasCap(caps, `knirv-kv/data/node/validator`, "update") {
		t.Fatal("oracle-active must NOT write node/validator")
	}
}
