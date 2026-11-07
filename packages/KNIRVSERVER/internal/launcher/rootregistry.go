package launcher

// Root tx-chain registry record publisher (root tx-chain design rev 2, section
// 2.1/2.2).
//
// When this launcher resolves the node as ROOT it derives the immutable root
// registry id (a fixed genesis constant, never operator-supplied and never taken
// from the registry or a caller), builds the RegistryRecord, signs it with the
// canonical KNIRVSDK secp256k1 path using the in-process root-owner signer, and
// POSTs the signed envelope to the deployed REGISTRY.KNIRV.NETWORK worker's
// signed registration endpoint. The Worker verifies the envelope against its
// configured trusted root identity and immutable root registry id; the launcher
// only ever reaches the Worker over HTTPS outbound.
//
// INTERIM (design rev 2 deviation, tracked in PendingImplementations.md): signing
// uses the in-process root-owner signer built from the already-decrypted root
// key, NOT the OpenBao RemoteSigner. The key is held only in process memory —
// it is never written to env vars, args, files, or logs. All signing goes
// through signRootRegistryRecord so the later swap to the Bao RemoteSigner (once
// owner-gated item 2a.6/8.8 delivers a root-key path) is a one-line change at
// the call site.
//
// If the node is not ROOT it publishes nothing for the root id.

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
)

// publisherNoncePathFunc returns the file used to persist the root-registry
// publisher nonce across restarts. Overridable in tests. The path itself is
// never logged.
var publisherNoncePathFunc = defaultPublisherNoncePath

// publisherNonceMu serializes nonce reads/writes so two launcher goroutines
// cannot race on the same nonce file.
var publisherNonceMu sync.Mutex

// registryHTTPClient is a bounded client for outbound registry calls: a finite
// timeout prevents a stalled Worker from hanging the launcher, and bodies are
// read through io.LimitReader at the call sites.
var registryHTTPClient = &http.Client{Timeout: 15 * time.Second}

// maxRegistryRespBytes bounds how much of a Worker response we read, so a
// malformed or hostile response cannot exhaust memory.
const maxRegistryRespBytes = 4 * 1024

func defaultPublisherNoncePath() string {
	dir, err := getAppDataDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "knirv-root-registry-nonce")
	}
	return filepath.Join(dir, "root_registry_nonce")
}

// deriveRootRegistryID returns the immutable root registry id: a fixed genesis
// constant derived deterministically from the root owner public key. It is not
// configurable, operator-supplied, or caller-influenced.
func deriveRootRegistryID(ownerPubKey []byte) string {
	sum := sha256.Sum256(ownerPubKey)
	return "knirvroot1" + hex.EncodeToString(sum[:8])
}

// rootPublicKeyFromHex returns the compressed public key for a hex-encoded root
// private key. It is used to derive this root node's immutable network-registry
// id. An error is returned for an invalid key.
func rootPublicKeyFromHex(hexKey string) ([]byte, error) {
	priv, err := hex.DecodeString(strings.TrimSpace(hexKey))
	if err != nil {
		return nil, err
	}
	signer, err := knirvsigning.NewLocalSigner(priv)
	if err != nil {
		return nil, err
	}
	return signer.PublicKey(), nil
}

// configureNodeRegistrationID sets KNIRV_NODE_REGISTRATION_ID in the launcher
// process for the node-registry heartbeat. A ROOT node derives it from the root
// owner public key (an immutable genesis constant) and overrides any inherited
// value, so an operator/attacker cannot make the node register under a foreign
// id. Non-root nodes clear it: a root-derived or inherited id must never be used
// by a non-root heartbeat. nodeRegistryRegistration reads this env at heartbeat
// time, so it must be set before startNodeRegistry runs.
func (app *ServerApp) configureNodeRegistrationID(isRoot bool, rootPrivKeyHex string) {
	if isRoot {
		pub, err := rootPublicKeyFromHex(rootPrivKeyHex)
		if err != nil {
			log.Printf("[KEY] could not derive root network-registry id: %v", err)
			return
		}
		os.Setenv(nodeRegistrationIDEnv, deriveRootRegistryID(pub))
		return
	}
	os.Unsetenv(nodeRegistrationIDEnv)
}

// nextPublisherNonce loads the last published nonce (0 if none), advances it by
// one (rejecting uint64 overflow), persists the new value atomically (O_EXCL
// random temp, fsync, rename), and validates the final file (regular, 0600, no
// symlink). It returns the new nonce. The in-memory lock and the atomic temp
// guard against concurrent writers and symlink/tmp races.
func nextPublisherNonce() (uint64, error) {
	path := publisherNoncePathFunc()
	if path == "" {
		return 0, fmt.Errorf("no publisher nonce path configured")
	}
	publisherNonceMu.Lock()
	defer publisherNonceMu.Unlock()

	// OS-backed lock so two processes (e.g. a restarted launcher racing a still
	// live one) cannot interleave read-increment-write on the nonce file. The
	// sidecar lock file is created 0600 and held for the whole operation.
	unlock, err := lockNonce(path)
	if err != nil {
		return 0, err
	}
	defer unlock()

	// Validate the final path BEFORE reading it: Lstat does not follow symlinks, so
	// a symlinked/non-regular/relaxed-mode file is rejected without ever being
	// opened or read. The old code read first, which silently followed the symlink.
	var current uint64
	if fi, serr := os.Lstat(path); serr == nil {
		if err := verifyNonceFileInfo(fi); err != nil {
			return 0, fmt.Errorf("publisher nonce: %w", err)
		}
		data, rerr := readNonceFile(path)
		if rerr != nil {
			return 0, fmt.Errorf("read publisher nonce: %w", rerr)
		}
		n, perr := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
		if perr != nil {
			return 0, fmt.Errorf("corrupt publisher nonce %q: %w", string(data), perr)
		}
		current = n
	} else if !os.IsNotExist(serr) {
		return 0, fmt.Errorf("stat publisher nonce: %w", serr)
	}

	if current == math.MaxUint64 {
		return 0, fmt.Errorf("publisher nonce overflow")
	}
	next := current + 1
	if err := writeNonceAtomic(path, next); err != nil {
		return 0, err
	}
	if err := verifyNonceFile(path); err != nil {
		return 0, err
	}
	return next, nil
}

// lockNonce takes an OS-backed exclusive lock on a sidecar lock file so the
// read-increment-write of the publisher nonce is serialized across processes, not
// just goroutines. It returns an unlock function that releases the lock.
func lockNonce(path string) (func(), error) {
	lockPath := path + ".lock"
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("nonce lock open: %w", err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return nil, fmt.Errorf("nonce lock: %w", err)
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		f.Close()
	}, nil
}

// readNonceFile reads the nonce with O_NOFOLLOW so a symlink at the path cannot be
// followed even in the narrow window after the Lstat validation.
func readNonceFile(path string) ([]byte, error) {
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, fmt.Errorf("open nonce: %w", err)
	}
	defer syscall.Close(fd)
	f := os.NewFile(uintptr(fd), path)
	defer f.Close()
	return io.ReadAll(f)
}

// writeNonceAtomic writes value to path via an exclusive random temp file with
// fsync then rename, so a reader never sees a partial nonce and a stale symlink
// at the target cannot redirect the write.
func writeNonceAtomic(path string, value uint64) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("nonce dir: %w", err)
	}
	randBytes := make([]byte, 8)
	if _, err := rand.Read(randBytes); err != nil {
		return fmt.Errorf("nonce rand: %w", err)
	}
	tmp := filepath.Join(dir, fmt.Sprintf(".knirv-root-registry-nonce-%x.tmp", randBytes))
	f, err := os.OpenFile(tmp, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("nonce temp: %w", err)
	}
	if _, err := f.WriteString(strconv.FormatUint(value, 10)); err != nil {
		f.Close()
		os.Remove(tmp)
		return fmt.Errorf("nonce write: %w", err)
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmp)
		return fmt.Errorf("nonce fsync: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("nonce close: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("nonce persist: %w", err)
	}
	return nil
}

// verifyNonceFile rejects a final nonce file that is not a regular 0600 file with
// no symlink, so a replaced/located symlink or relaxed mode cannot be used to
// exfiltrate or tamper the nonce.
func verifyNonceFile(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("nonce stat: %w", err)
	}
	return verifyNonceFileInfo(info)
}

// verifyNonceFileInfo is the symlink/mode check used both for the pre-read
// validation (the caller already has the Lstat result) and the post-write check.
func verifyNonceFileInfo(info os.FileInfo) error {
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("publisher nonce is a symlink")
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("publisher nonce is not a regular file")
	}
	if info.Mode().Perm() != 0o600 {
		return fmt.Errorf("publisher nonce has unsafe mode %v", info.Mode().Perm())
	}
	return nil
}

// signRootRegistryRecord signs a root-tx-chain registry record with the given
// signer. This is the single seam through which all root-registry signing flows;
// swapping to the Bao RemoteSigner later changes only how `signer` is
// constructed at the call site. The envelope matches the Rust-core
// `sign_registry_record` shape the Worker verifies (domain knirv.registry,
// purpose root-tx-chain-record, nonce "root-tx-chain:<n>", canonical JSON
// payload).
func signRootRegistryRecord(ctx context.Context, signer knirvsigning.RemoteSigner, record knirvsigning.RegistryRecord, chainID string, issuedAt uint64) (knirvsigning.SignedMessage, error) {
	payload, err := json.Marshal(record)
	if err != nil {
		return knirvsigning.SignedMessage{}, fmt.Errorf("marshal registry record: %w", err)
	}
	envelope := knirvsigning.MessageEnvelope{
		SchemaVersion: knirvsigning.MessageSchemaVersion,
		Domain:        knirvsigning.RegistryDomain,
		Purpose:       knirvsigning.PurposeRootTxChainRecord,
		ChainID:       chainID,
		Nonce:         fmt.Sprintf("root-tx-chain:%d", record.Nonce),
		IssuedAtUnix:  int64(issuedAt),
		ExpiresAtUnix: int64(record.ExpiresAtUnix),
		Payload:       payload,
	}
	return signer.SignMessage(ctx, envelope)
}

// publishRootRegistryRecord derives the root registry id, builds and signs the
// RegistryRecord with the in-process root-owner signer, and POSTs it to the
// Worker's signed registration endpoint. It returns nil on success. The Worker
// URL must be HTTPS except for an explicit loopback test endpoint.
func (app *ServerApp) publishRootRegistryRecord(ctx context.Context, rootPrivKeyHex string) error {
	priv, err := hex.DecodeString(rootPrivKeyHex)
	if err != nil {
		return fmt.Errorf("decode root key: %w", err)
	}
	// In-process signer only; the key is never written to env/args/files/logs.
	signer, err := knirvsigning.NewLocalSigner(priv)
	if err != nil {
		return fmt.Errorf("root signer: %w", err)
	}
	ownerPub := signer.PublicKey()
	rootRegistryID := deriveRootRegistryID(ownerPub)

	txAddr := TxChainAddress()
	if txAddr == "" {
		return fmt.Errorf("root tx-chain address unavailable; cannot publish root registry record")
	}

	nonce, err := nextPublisherNonce()
	if err != nil {
		return fmt.Errorf("publisher nonce: %w", err)
	}

	now := time.Now().Unix()
	// Bounded lifetime (<= REGISTRY_RECORD_MAX_TTL_SECONDS = 1 year) per the
	// Worker's verification.
	expiresAt := now + int64((365*24*60*60)-60)
	record := knirvsigning.RegistryRecord{
		ChainID:        rootRegistryID,
		TxChainAddress: txAddr,
		OwnerPubKey:    base64.StdEncoding.EncodeToString(ownerPub),
		Nonce:          nonce,
		ExpiresAtUnix:  uint64(expiresAt),
	}
	signed, err := signRootRegistryRecord(ctx, signer, record, rootRegistryID, uint64(now))
	if err != nil {
		return fmt.Errorf("sign root registry record: %w", err)
	}

	body := struct {
		Signed knirvsigning.SignedMessage `json:"signed"`
	}{Signed: signed}
	raw, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encode registry record: %w", err)
	}

	endpoint, err := registryRegisterURL()
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("build registry request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := registryHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("registry request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, maxRegistryRespBytes))
		return fmt.Errorf("registry rejected root record (%d): %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	// Drain and discard any remaining body within the read bound.
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxRegistryRespBytes))
	return nil
}

// registryRegisterURL returns the Worker's signed registration endpoint,
// requiring HTTPS. Plain HTTP is only permitted for an explicit loopback test
// endpoint (127.0.0.1 / localhost / ::1) so the launcher test suite can use an
// httptest server.
func registryRegisterURL() (string, error) {
	base := nodeRegistryBaseURL()
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
	return strings.TrimRight(base, "/") + "/registry/register", nil
}

// publishRootRegistryWithRetry publishes the root registry record asynchronously
// with bounded retries and exponential backoff, so a stalled or temporarily
// unreachable Worker cannot block node startup. It stops on context cancel.
func (app *ServerApp) publishRootRegistryWithRetry(ctx context.Context, rootPrivKeyHex string) {
	const maxAttempts = 3
	backoff := time.Second
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if err := app.publishRootRegistryRecord(ctx, rootPrivKeyHex); err != nil {
			log.Printf("Warning: root tx-chain registry record not published (attempt %d/%d): %v", attempt+1, maxAttempts, err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			backoff *= 2
			continue
		}
		log.Printf("[REGISTRY] Published root tx-chain registry record")
		return
	}
}

// maybePublishRootRegistry gates the root-record publish on node role and the
// presence of a root key. Non-root nodes (and nodes without a decrypted root
// key) publish nothing for the root id.
func (app *ServerApp) maybePublishRootRegistry(ctx context.Context, rootPrivKeyHex string) {
	resolved := app.roleResolution.Load()
	if resolved == nil || !resolved.IsRoot() {
		return
	}
	if strings.TrimSpace(rootPrivKeyHex) == "" {
		return
	}
	app.publishRootRegistryWithRetry(ctx, rootPrivKeyHex)
}
