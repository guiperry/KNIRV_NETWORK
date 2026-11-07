package launcher

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
	"github.com/guiperry/knirv-sdk-go/noderole"
)

// newTestRootSigner returns a freshly generated in-process signer plus its hex
// private key, mirroring how the launcher builds the root-owner signer from the
// decrypted root key.
func newTestRootSigner(t *testing.T) (knirvsigning.RemoteSigner, string) {
	t.Helper()
	priv := make([]byte, 32)
	if _, err := rand.Read(priv); err != nil {
		t.Fatalf("rand: %v", err)
	}
	signer, err := knirvsigning.NewLocalSigner(priv)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}
	return signer, hex.EncodeToString(priv)
}

// fakeWorker emulates REGISTRY.KNIRV.NETWORK's signed registration + signed lookup
// using the same KNIRVSDK verifier the real Worker uses. It enforces the trusted
// root identity (expectedAddr) and immutable root registry id (expectedRegID).
func fakeWorker(t *testing.T, expectedAddr, expectedRegID string) (*httptest.Server, map[string][]byte) {
	t.Helper()
	store := map[string][]byte{}
	mux := http.NewServeMux()
	mux.HandleFunc("/registry/register", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Signed knirvsigning.SignedMessage `json:"signed"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			w.WriteHeader(400)
			return
		}
		rec, err := knirvsigning.VerifyRegistryRecord(body.Signed, expectedAddr, expectedRegID, time.Now())
		if err != nil {
			w.WriteHeader(401)
			return
		}
		raw, _ := json.Marshal(body.Signed)
		store[rec.ChainID] = raw
		w.WriteHeader(201)
	})
	mux.HandleFunc("/registry/lookup/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/registry/lookup/")
		if v, ok := store[id]; ok {
			w.Write(v)
			return
		}
		w.WriteHeader(404)
	})
	return httptest.NewServer(mux), store
}

// TestPublishRootRegistryRecord_RootKeyStartup covers: a ROOT node publishes a
// signed root-tx-chain record to the Worker, and a subsequent CLI-style lookup
// with root verification (VerifyRegistryRecord against the trusted root identity
// and immutable root registry id) succeeds.
func TestPublishRootRegistryRecord_RootKeyStartup(t *testing.T) {
	signer, privHex := newTestRootSigner(t)
	expectedRegID := deriveRootRegistryID(signer.PublicKey())
	ts, _ := fakeWorker(t, signer.Address(), expectedRegID)
	defer ts.Close()

	t.Setenv("KNIRV_REGISTRY_URL", ts.URL)
	t.Setenv("KNIRV_VALIDATION_SERVICE_PRIVATE_KEY", privHex)

	nonceFile := filepath.Join(t.TempDir(), "nonce")
	old := publisherNoncePathFunc
	publisherNoncePathFunc = func() string { return nonceFile }
	defer func() { publisherNoncePathFunc = old }()

	app := &ServerApp{}
	if err := app.publishRootRegistryRecord(context.Background(), privHex); err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	resp, err := http.Get(ts.URL + "/registry/lookup/" + expectedRegID)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("lookup status = %d, want 200", resp.StatusCode)
	}
	var signed knirvsigning.SignedMessage
	if err := json.NewDecoder(resp.Body).Decode(&signed); err != nil {
		t.Fatalf("decode signed: %v", err)
	}
	// CLI-style root verification: trusted root address + immutable root registry id.
	if _, err := knirvsigning.VerifyRegistryRecord(signed, signer.Address(), expectedRegID, time.Now()); err != nil {
		t.Fatalf("root verification of published record failed: %v", err)
	}
}

// TestPublishRootRegistryRecord_NonRootPublishesNothing covers the gate: a node
// that is not ROOT (here a bootnode) must not publish any root record.
func TestPublishRootRegistryRecord_NonRootPublishesNothing(t *testing.T) {
	signer, privHex := newTestRootSigner(t)
	ts, _ := fakeWorker(t, signer.Address(), deriveRootRegistryID(signer.PublicKey()))
	defer ts.Close()

	var hits int32
	gate := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(200)
	}))
	defer gate.Close()
	t.Setenv("KNIRV_REGISTRY_URL", gate.URL)
	t.Setenv("KNIRV_VALIDATION_SERVICE_PRIVATE_KEY", privHex)

	app := &ServerApp{}
	app.roleResolution.Store(&noderole.Resolution{Kind: noderole.KindBootnode})
	app.maybePublishRootRegistry(context.Background(), privHex)

	if atomic.LoadInt32(&hits) != 0 {
		t.Fatalf("non-root node published a root record (%d hits)", hits)
	}
}

// TestMaybePublishRootRegistryRecord_EmptyKeySkips covers the fail-closed guard:
// an empty root key publishes nothing even when the role is ROOT.
func TestMaybePublishRootRegistryRecord_EmptyKeySkips(t *testing.T) {
	var hits int32
	gate := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(200)
	}))
	defer gate.Close()
	t.Setenv("KNIRV_REGISTRY_URL", gate.URL)

	app := &ServerApp{}
	app.roleResolution.Store(&noderole.Resolution{Kind: noderole.KindRoot})
	app.maybePublishRootRegistry(context.Background(), "")

	if atomic.LoadInt32(&hits) != 0 {
		t.Fatalf("empty root key published a record (%d hits)", hits)
	}
}

// --- Audit txchain_p45 P4/P5/P6 tests ---

// P4: a ROOT node derives its network-registry id from the root owner public key
// and overrides any inherited KNIRV_NODE_REGISTRATION_ID, so an operator/attacker
// cannot make it register under a foreign id.
func TestConfigureNodeRegistrationIDRootOverridesInherited(t *testing.T) {
	t.Setenv(nodeRegistrationIDEnv, "foreign-inherited-id")
	app := &ServerApp{}
	_, hexKey := newTestRootSigner(t)
	pub, err := rootPublicKeyFromHex(hexKey)
	if err != nil {
		t.Fatal(err)
	}
	want := deriveRootRegistryID(pub)
	app.configureNodeRegistrationID(true, hexKey)
	if got := os.Getenv(nodeRegistrationIDEnv); got != want {
		t.Fatalf("registration id = %q, want derived %q (inherited value must be overridden)", got, want)
	}
}

// P4: a non-root node must clear KNIRV_NODE_REGISTRATION_ID — a root-derived or
// inherited id must never drive a non-root heartbeat.
func TestConfigureNodeRegistrationIDNonRootClears(t *testing.T) {
	t.Setenv(nodeRegistrationIDEnv, "some-inherited-id")
	app := &ServerApp{}
	app.configureNodeRegistrationID(false, "")
	if _, ok := os.LookupEnv(nodeRegistrationIDEnv); ok {
		t.Fatalf("non-root node must clear KNIRV_NODE_REGISTRATION_ID, got %q", os.Getenv(nodeRegistrationIDEnv))
	}
}

// P5: the registry URL must be HTTPS except for an explicit loopback test endpoint.
func TestRegistryURLRejectsPlaintext(t *testing.T) {
	t.Setenv(networkRegistryEnvOverride, "http://example.com")
	if _, err := registryRegisterURL(); err == nil {
		t.Fatal("expected plaintext non-loopback registry URL to be rejected")
	}
}

func TestRegistryURLAllowsLoopbackHTTP(t *testing.T) {
	t.Setenv(networkRegistryEnvOverride, "http://127.0.0.1:9999")
	u, err := registryRegisterURL()
	if err != nil {
		t.Fatalf("loopback http should be allowed: %v", err)
	}
	if !strings.HasSuffix(u, "/registry/register") {
		t.Fatalf("unexpected registry url %q", u)
	}
}

func TestRegistryURLAllowsHTTPS(t *testing.T) {
	t.Setenv(networkRegistryEnvOverride, "https://registry.knirv.network")
	if _, err := registryRegisterURL(); err != nil {
		t.Fatalf("https should be allowed: %v", err)
	}
}

// P5: a plaintext (non-loopback) registry base URL is rejected by RegisterNode
// before any request is issued.
func TestRegisterNodeRejectsPlaintext(t *testing.T) {
	if err := RegisterNode(context.Background(), "http://192.0.2.1", NodeRegistration{}); err == nil {
		t.Fatal("expected RegisterNode to reject a plaintext non-loopback URL")
	}
}

// P5: the publisher nonce advances, persists, and the stored file is a regular
// 0600 file (no symlink, no partial write).
func TestNextPublisherNonceIncrements(t *testing.T) {
	nonceFile := filepath.Join(t.TempDir(), "nonce")
	old := publisherNoncePathFunc
	publisherNoncePathFunc = func() string { return nonceFile }
	defer func() { publisherNoncePathFunc = old }()

	n1, err := nextPublisherNonce()
	if err != nil {
		t.Fatal(err)
	}
	if n1 != 1 {
		t.Fatalf("first nonce = %d, want 1", n1)
	}
	n2, err := nextPublisherNonce()
	if err != nil {
		t.Fatal(err)
	}
	if n2 != 2 {
		t.Fatalf("second nonce = %d, want 2", n2)
	}
	fi, err := os.Stat(nonceFile)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("nonce file mode = %v, want 0600", fi.Mode().Perm())
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		t.Fatal("nonce file must not be a symlink")
	}
}

// P5: uint64 nonce overflow is rejected rather than wrapping.
func TestNextPublisherNonceOverflow(t *testing.T) {
	nonceFile := filepath.Join(t.TempDir(), "nonce")
	old := publisherNoncePathFunc
	publisherNoncePathFunc = func() string { return nonceFile }
	defer func() { publisherNoncePathFunc = old }()
	if err := os.WriteFile(nonceFile, []byte(strconv.FormatUint(math.MaxUint64, 10)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := nextPublisherNonce(); err == nil {
		t.Fatal("expected nonce overflow to be rejected")
	}
}

// P5: a symlinked nonce file is rejected (no tmp/symlink redirect of the write).
func TestVerifyNonceFileRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("1"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "nonce")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err := verifyNonceFile(link); err == nil {
		t.Fatal("expected symlinked nonce file to be rejected")
	}
}

// P6: the root-registry publish path must not export the root key (or any key) to
// the process environment. Pieces 4/5 derive the id/signer in process memory only.
func TestPublishRootRegistryRecordNoEnvExposure(t *testing.T) {
	nonceFile := filepath.Join(t.TempDir(), "nonce")
	old := publisherNoncePathFunc
	publisherNoncePathFunc = func() string { return nonceFile }
	defer func() { publisherNoncePathFunc = old }()

	signer, hexKey := newTestRootSigner(t)
	expectedRegID := deriveRootRegistryID(signer.PublicKey())
	ts, _ := fakeWorker(t, signer.Address(), expectedRegID)
	defer ts.Close()

	// Pre-existing (validation-chain 2a.x) env handoff: the launcher sets this from
	// the root key so the validation chain signer can load. Pieces 4/5 add nothing
	// on top of it.
	t.Setenv("KNIRV_VALIDATION_SERVICE_PRIVATE_KEY", hexKey)
	t.Setenv(networkRegistryEnvOverride, ts.URL)

	app := &ServerApp{}
	before := map[string]bool{}
	for _, kv := range os.Environ() {
		before[kv] = true
	}
	if err := app.publishRootRegistryRecord(context.Background(), hexKey); err != nil {
		t.Fatalf("publish: %v", err)
	}
	for _, kv := range os.Environ() {
		if !before[kv] {
			t.Fatalf("process environment changed after publish (key exposure risk): new entry %q", kv)
		}
	}
}

func rootPublicKeyFromHexMust(t *testing.T, hexKey string) []byte {
	t.Helper()
	pub, err := rootPublicKeyFromHex(hexKey)
	if err != nil {
		t.Fatal(err)
	}
	return pub
}

// P1#3: a symlinked nonce file is rejected before it is read (Lstat validation
// runs before any open), so the symlink target is never read.
func TestNextPublisherNonceRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("1"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "nonce")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	old := publisherNoncePathFunc
	publisherNoncePathFunc = func() string { return link }
	defer func() { publisherNoncePathFunc = old }()
	if _, err := nextPublisherNonce(); err == nil {
		t.Fatal("expected symlinked nonce to be rejected")
	}
}

// P1#3: a nonce file with a non-0600 mode is rejected.
func TestNextPublisherNonceRejectsWrongMode(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "nonce")
	if err := os.WriteFile(p, []byte("1"), 0o644); err != nil {
		t.Fatal(err)
	}
	old := publisherNoncePathFunc
	publisherNoncePathFunc = func() string { return p }
	defer func() { publisherNoncePathFunc = old }()
	if _, err := nextPublisherNonce(); err == nil {
		t.Fatal("expected wrong-mode nonce to be rejected")
	}
}

// P1#3: a non-regular nonce path (here a directory) is rejected.
func TestNextPublisherNonceRejectsNonRegular(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "nonce")
	if err := os.Mkdir(p, 0o700); err != nil {
		t.Fatal(err)
	}
	old := publisherNoncePathFunc
	publisherNoncePathFunc = func() string { return p }
	defer func() { publisherNoncePathFunc = old }()
	if _, err := nextPublisherNonce(); err == nil {
		t.Fatal("expected non-regular nonce to be rejected")
	}
}

// P1#3 child: advances the nonce five times, writing each value to the file named
// by KNIRV_TEST_NONCE_OUT. Run as a separate process by
// TestPublisherNonceTwoProcessContention to exercise the cross-process flock.
func TestPublisherNonceChild(t *testing.T) {
	path := os.Getenv("KNIRV_TEST_NONCE_PATH")
	if path == "" {
		t.Skip("not a child process")
	}
	outPath := os.Getenv("KNIRV_TEST_NONCE_OUT")
	f, err := os.Create(outPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	old := publisherNoncePathFunc
	publisherNoncePathFunc = func() string { return path }
	defer func() { publisherNoncePathFunc = old }()
	for i := 0; i < 5; i++ {
		n, err := nextPublisherNonce()
		if err != nil {
			t.Fatalf("next nonce: %v", err)
		}
		fmt.Fprintln(f, n)
	}
}

// P1#3: two processes racing on the same nonce file (via an OS-backed flock) must
// produce unique, strictly increasing nonces — never a duplicate or a lost update.
func TestPublisherNonceTwoProcessContention(t *testing.T) {
	dir := t.TempDir()
	noncePath := filepath.Join(dir, "nonce")
	old := publisherNoncePathFunc
	publisherNoncePathFunc = func() string { return noncePath }
	defer func() { publisherNoncePathFunc = old }()

	bin, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	var all []uint64
	var wg sync.WaitGroup
	for p := 0; p < 2; p++ {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()
			outPath := filepath.Join(dir, fmt.Sprintf("child%d.out", p))
			cmd := exec.Command(bin, "-test.run=^TestPublisherNonceChild$")
			cmd.Env = append(os.Environ(),
				"KNIRV_TEST_NONCE_PATH="+noncePath,
				"KNIRV_TEST_NONCE_OUT="+outPath,
			)
			cmd.Stdout = io.Discard
			cmd.Stderr = io.Discard
			if err := cmd.Run(); err != nil {
				t.Errorf("child %d: %v", p, err)
				return
			}
			data, rerr := os.ReadFile(outPath)
			if rerr != nil {
				t.Errorf("read child %d out: %v", p, rerr)
				return
			}
			mu.Lock()
			defer mu.Unlock()
			for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
				if line == "" {
					continue
				}
				v, perr := strconv.ParseUint(strings.TrimSpace(line), 10, 64)
				if perr != nil {
					t.Errorf("parse nonce %q: %v", line, perr)
					continue
				}
				all = append(all, v)
			}
		}(p)
	}
	wg.Wait()

	if len(all) != 10 {
		t.Fatalf("expected 10 nonces, got %d: %v", len(all), all)
	}
	// Children finish in arbitrary order, so sort by value before checking the
	// sequence. With the flock serializing every read-increment-write, the ten
	// values must be exactly {1..10} — unique and gap-free — proving no lost
	// update and no duplicate.
	sort.Slice(all, func(i, j int) bool { return all[i] < all[j] })
	seen := map[uint64]bool{}
	for i, v := range all {
		if seen[v] {
			t.Fatalf("duplicate nonce %d (lost lock / race)", v)
		}
		seen[v] = true
		if v != uint64(i)+1 {
			t.Fatalf("nonce sequence gap: at index %d got %d (expected %d)", i, v, uint64(i)+1)
		}
	}
}
