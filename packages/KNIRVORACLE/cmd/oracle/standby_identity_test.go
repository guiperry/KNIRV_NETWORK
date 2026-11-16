package main

import (
	"context"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/guiperry/knirv-sdk-go/secrets"
	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
	"github.com/knirvcorp/knirvoracle/internal/oracle"
	"go.uber.org/zap"
)

// Source of the standby path, embedded so the guard inspects exactly what the
// compiler sees (this is what makes the overlay mutant test meaningful).
//
//go:embed standby_identity.go
var standbyIdentitySrc string

//go:embed rootkey.go
var rootkeySrc string

// standbyProbeProvider records any boot.key or root.key access. The standby path must
// never make these calls, so they fail the test if they happen.
type standbyProbeProvider struct {
	fakeProvider
	bootCalls int
	rootCalls int
}

func (p *standbyProbeProvider) Boot(ctx context.Context) (*secrets.BootSecrets, error) {
	p.bootCalls++
	return nil, errors.New("boot.key must not be read on the standby path")
}

func (p *standbyProbeProvider) Root(ctx context.Context) (*secrets.RootSecrets, error) {
	p.rootCalls++
	return nil, errors.New("root.key must not be read on the standby path")
}

// TestApplyStandbyIdentityUsesProviderSigner: the validator identity comes from the
// provider (a fake RemoteSigner), boot.key and root.key are never read, and the owner
// handle is a fresh key distinct from the validator identity.
func TestApplyStandbyIdentityUsesProviderSigner(t *testing.T) {
	signer := fakeValidatorSigner{addr: "knirv1w508d6qejxtdg4y5r3zarvary0c5xw7kaqx36e", pubkey: []byte("vb")}
	probe := &standbyProbeProvider{fakeProvider: fakeProvider{signer: signer}}
	cfg := &oracle.OracleConfig{ReplicationSources: []string{"https://bootnode.example"}}

	if err := applyStandbyIdentity(context.Background(), cfg, probe, zap.NewNop()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ReplicationSigner == nil || cfg.ReplicationSigner.Address() != signer.Address() {
		t.Fatalf("standby replication signer not taken from provider: %+v", cfg.ReplicationSigner)
	}
	if len(cfg.ReplicationSources) != 1 {
		t.Fatalf("replication sources should be kept when the identity resolves, got %v", cfg.ReplicationSources)
	}
	if probe.bootCalls != 0 || probe.rootCalls != 0 {
		t.Fatalf("standby read key material: boot=%d root=%d", probe.bootCalls, probe.rootCalls)
	}
	owner, err := hex.DecodeString(cfg.OwnerPrivateKey)
	if err != nil || len(owner) != 32 {
		t.Fatalf("owner handle is not a 32-byte in-memory key: %v", err)
	}
}

// TestApplyStandbyIdentityProviderErrorDisablesPulls: a failing provider degrades the
// standby (no pulls) without the ephemeral owner key being used to sign them.
func TestApplyStandbyIdentityProviderErrorDisablesPulls(t *testing.T) {
	probe := &standbyProbeProvider{fakeProvider: fakeProvider{signerErr: errors.New("no standby identity")}}
	cfg := &oracle.OracleConfig{ReplicationSources: []string{"https://bootnode.example"}}

	if err := applyStandbyIdentity(context.Background(), cfg, probe, zap.NewNop()); err != nil {
		t.Fatalf("a missing identity must not stop the standby: %v", err)
	}
	if cfg.ReplicationSigner != nil {
		t.Fatal("no signer expected when the provider fails")
	}
	if len(cfg.ReplicationSources) != 0 {
		t.Fatalf("pulls must be disabled when no standby identity resolves, got %v", cfg.ReplicationSources)
	}
	if probe.bootCalls != 0 {
		t.Fatalf("boot.key read on failure path: %d", probe.bootCalls)
	}
}

// TestApplyStandbyIdentityNoProviderDisablesPulls covers the case where OpenBao is not
// configured at all.
func TestApplyStandbyIdentityNoProviderDisablesPulls(t *testing.T) {
	cfg := &oracle.OracleConfig{ReplicationSources: []string{"https://bootnode.example"}}
	if err := applyStandbyIdentity(context.Background(), cfg, nil, zap.NewNop()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ReplicationSigner != nil || len(cfg.ReplicationSources) != 0 {
		t.Fatalf("expected no signer and no pulls, got signer=%v sources=%v", cfg.ReplicationSigner, cfg.ReplicationSources)
	}
	if cfg.OwnerPrivateKey == "" {
		t.Fatal("owner handle must still be set for the token constructor")
	}
}

func TestStandbyProviderRequiresAppRoleConfig(t *testing.T) {
	t.Setenv(envStandbyBaoSocket, "")
	t.Setenv(envStandbyRoleID, "")
	t.Setenv(envStandbySecretIDFile, "")
	if _, err := newStandbyProvider(context.Background()); err == nil {
		t.Fatal("expected error with no AppRole configuration")
	}
}

func TestReadStandbySecretIDRequiresOwnerOnlyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secret-id")
	if err := os.WriteFile(path, []byte("  synthetic-secret-id \n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readStandbySecretID(path); err == nil {
		t.Fatal("group/world-readable secret-id file accepted")
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := readStandbySecretID(path)
	if err != nil {
		t.Fatalf("0600 file rejected: %v", err)
	}
	if got != "synthetic-secret-id" {
		t.Fatalf("secret-id not trimmed: %q", got)
	}
}

// TestStandbyAppRoleLoginOverSocket runs a fake OpenBao on a unix socket and checks the
// login request shape and token handling. Error text must not echo the secret ID.
func TestStandbyAppRoleLoginOverSocket(t *testing.T) {
	dir, err := os.MkdirTemp("", "sbl")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	socket := filepath.Join(dir, "bao.sock")
	ln, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	const roleID = "synthetic-role"
	const secretID = "synthetic-secret"
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/auth/approle/login", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		if r.Method != http.MethodPut {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["role_id"] != roleID {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		if body["secret_id"] != secretID {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.Write([]byte(`{"auth":{"client_token":"tok-synthetic"}}`))
	})
	srv := &http.Server{Handler: mux}
	go srv.Serve(ln)
	defer srv.Close()

	token, err := standbyAppRoleLogin(context.Background(), socket, roleID, secretID)
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if token != "tok-synthetic" {
		t.Fatalf("unexpected token %q", token)
	}

	_, err = standbyAppRoleLogin(context.Background(), socket, roleID, "wrong-secret")
	if err == nil {
		t.Fatal("login with a wrong secret ID succeeded")
	}
	if strings.Contains(err.Error(), "wrong-secret") {
		t.Fatalf("error text echoes the secret ID: %v", err)
	}
}

// TestStandbyPathHoldsNoBootKeyMaterial is a source guard. The standby path (every
// function in standby_identity.go plus initStandbyOracle in rootkey.go) must not touch
// boot.key: no Boot() call, no FileProvider, no master wallet key, no boot.key path.
// It parses the embedded sources, so an overlay mutant is what gets checked.
func TestStandbyPathHoldsNoBootKeyMaterial(t *testing.T) {
	forbiddenSelectors := map[string]bool{
		"Boot":               true,
		"NewFileProvider":    true,
		"MasterWalletKeyHex": true,
		"FindBoot":           true,
		"BootSearchPaths":    true,
	}

	var checked int
	scan := func(node ast.Node, where string) {
		ast.Inspect(node, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.SelectorExpr:
				if forbiddenSelectors[x.Sel.Name] {
					t.Errorf("%s reads boot key material via selector %q", where, x.Sel.Name)
				}
			case *ast.BasicLit:
				if x.Kind == token.STRING && strings.Contains(strings.ToLower(x.Value), "boot.key") {
					t.Errorf("%s references boot.key by path literal %s", where, x.Value)
				}
			}
			return true
		})
	}

	fset := token.NewFileSet()
	standbyFile, err := parser.ParseFile(fset, "standby_identity.go", standbyIdentitySrc, 0)
	if err != nil {
		t.Fatalf("parse standby_identity.go: %v", err)
	}
	for _, decl := range standbyFile.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			checked++
			scan(fn, "standby_identity.go:"+fn.Name.Name)
		}
	}

	rootFile, err := parser.ParseFile(fset, "rootkey.go", rootkeySrc, 0)
	if err != nil {
		t.Fatalf("parse rootkey.go: %v", err)
	}
	found := false
	for _, decl := range rootFile.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "initStandbyOracle" {
			found = true
			checked++
			scan(fn, "rootkey.go:initStandbyOracle")
		}
	}
	if !found {
		t.Fatal("initStandbyOracle not found; guard would be vacuous")
	}
	if checked < 2 {
		t.Fatalf("guard inspected too few functions: %d", checked)
	}
}

// nameRecordingProvider wraps fakeProvider and records every identity name requested
// through Signer. The standby must ask for the replication identity by its real name.
type nameRecordingProvider struct {
	fakeProvider
	names []string
}

func (p *nameRecordingProvider) Signer(ctx context.Context, name string) (knirvsigning.RemoteSigner, error) {
	p.names = append(p.names, name)
	return p.fakeProvider.Signer(ctx, name)
}

// TestStandbyResolvesReplicationSignerByBootName: the standby asks the provider for
// "boot" (bootnode-master), never "validation" (checkpoint-signer, which the
// oracle-standby policy denies). The fake provider ignores names, so this test checks
// the name directly.
func TestStandbyResolvesReplicationSignerByBootName(t *testing.T) {
	signer := fakeValidatorSigner{addr: "knirv1w508d6qejxtdg4y5r3zarvary0c5xw7kaqx36e", pubkey: []byte("vb")}
	p := &nameRecordingProvider{fakeProvider: fakeProvider{signer: signer}}
	if _, err := resolveStandbyValidatorSigner(context.Background(), p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.names) != 1 || p.names[0] != "boot" {
		t.Fatalf("standby requested signer names %v, want [boot]", p.names)
	}
}

// TestReadStandbySecretIDRejectsSymlink: a symlink to an owner-only file must not be
// accepted, even though its target has mode 0600.
func TestReadStandbySecretIDRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "real-secret-id")
	if err := os.WriteFile(target, []byte("synthetic-secret-id\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "secret-id")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := readStandbySecretID(link); err == nil {
		t.Fatal("symlinked secret-id file accepted")
	}
	if _, err := readStandbySecretID(target); err != nil {
		t.Fatalf("direct 0600 file rejected: %v", err)
	}
}

// TestStandbyPullSignerAllowedByOracleStandbyPolicy: a fake OpenBao on a unix socket
// enforces the oracle-standby policy (only bootnode-master may sign, for the
// oracle-standby token). The real secrets.BaoProvider is used, so the identity name
// goes through the SDK core mapping. Replication-pull signing must succeed, and the
// signed identity must be bootnode-master.
func TestStandbyPullSignerAllowedByOracleStandbyPolicy(t *testing.T) {
	// Short prefix on purpose: unix socket paths are limited to ~108 bytes.
	dir, err := os.MkdirTemp("", "sbp")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	socket := filepath.Join(dir, "b.sock")
	ln, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	const standbyToken = "tok-standby-policy"
	var seen []string
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/knirv/sign/message/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/v1/knirv/sign/message/")
		seen = append(seen, name)
		if r.Header.Get("X-Vault-Token") != standbyToken || name != "bootnode-master" {
			http.Error(w, "permission denied", http.StatusForbidden)
			return
		}
		out, _ := json.Marshal(knirvsigning.SignedMessage{Signature: []byte{1}, PublicKey: []byte{2}, Address: "knirv1fake"})
		w.Write([]byte(`{"data":` + string(out) + `}`))
	})
	srv := &http.Server{Handler: mux}
	go srv.Serve(ln)
	defer srv.Close()

	bao := secrets.NewBaoProvider(socket, standbyToken)
	cfg := &oracle.OracleConfig{ReplicationSources: []string{"https://bootnode.example"}}
	ctx := context.Background()
	if err := applyStandbyIdentity(ctx, cfg, bao, zap.NewNop()); err != nil {
		t.Fatalf("applyStandbyIdentity: %v", err)
	}
	if cfg.ReplicationSigner == nil || len(cfg.ReplicationSources) == 0 {
		t.Fatalf("standby identity degraded: signer=%v sources=%v", cfg.ReplicationSigner, cfg.ReplicationSources)
	}
	_, err = cfg.ReplicationSigner.SignMessage(ctx, knirvsigning.MessageEnvelope{
		SchemaVersion: "1", Domain: "knirv.replication", Purpose: "replication-pull",
		ChainID: "chain", Nonce: "n1", Payload: []byte("p"),
	})
	if err != nil {
		t.Fatalf("replication-pull signing denied under oracle-standby policy (signed names seen=%v): %v", seen, err)
	}
	if len(seen) != 1 || seen[0] != "bootnode-master" {
		t.Fatalf("pull signed with wrong identity: %v", seen)
	}
}
