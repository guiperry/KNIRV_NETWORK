package governance

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	secpECDSA "github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
	"go.uber.org/zap"
)

const defaultRegistryBaseURL = "https://registry.knirv.network"

// registrySigner mirrors the exact signing convention backend_server's
// registryclient package uses (dcrd SignCompact over a sha256 digest,
// stripped of its recovery byte, compressed pubkey, KNIRVSDK's bech32
// Address()) — duplicated rather than imported because this is a separate Go
// module (github.com/knirvcorp/knirvoracle) from backend_server, with no
// shared internal package between them.
type registrySigner struct {
	priv *secp256k1.PrivateKey
	pub  []byte
	id   string
}

func newRegistrySigner(hexKey string) (*registrySigner, error) {
	raw, err := hex.DecodeString(strings.TrimPrefix(strings.TrimSpace(hexKey), "0x"))
	if err != nil {
		return nil, fmt.Errorf("decode private key hex: %w", err)
	}
	if len(raw) != 32 {
		return nil, fmt.Errorf("private key must be 32 bytes, got %d", len(raw))
	}
	priv := secp256k1.PrivKeyFromBytes(raw)
	if priv.Key.IsZero() {
		return nil, fmt.Errorf("private key cannot be zero")
	}
	pub := priv.PubKey().SerializeCompressed()
	id, err := knirvsigning.Address(pub, knirvsigning.DefaultAddressPrefix)
	if err != nil {
		return nil, fmt.Errorf("derive validator address: %w", err)
	}
	return &registrySigner{priv: priv, pub: pub, id: id}, nil
}

func (s *registrySigner) sign(parts ...string) string {
	digest := sha256.Sum256([]byte(strings.Join(parts, "|")))
	compact := secpECDSA.SignCompact(s.priv, digest[:], true)
	return hex.EncodeToString(compact[1:])
}

// RegistryValidatorEntry mirrors registry.knirv.network's POST /validators
// entry shape (src/index.js).
type RegistryValidatorEntry struct {
	ValidatorID string `json:"validatorID"`
	PublicKey   string `json:"publicKey"`
	Active      bool   `json:"active"`
}

// RegistrySync pushes validator-set changes to the registry's §6.6
// write-time trust anchor whenever governance's own validator state changes
// (enrollment, jailing, unjailing, removal, stake-release demotion) — without
// this, the Worker's copy of "who's enrolled" only ever reflects a one-time
// genesis seed and silently drifts from reality.
type RegistrySync struct {
	baseURL string
	http    *http.Client
	chainID string
	logger  *zap.Logger

	signer  atomic.Pointer[registrySigner]
	version atomic.Int64
}

func NewRegistrySync(chainID string, logger *zap.Logger) *RegistrySync {
	base := defaultRegistryBaseURL
	if override := strings.TrimSpace(os.Getenv("KNIRV_REGISTRY_URL")); override != "" {
		base = override
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &RegistrySync{
		baseURL: strings.TrimRight(base, "/"),
		http:    &http.Client{Timeout: 10 * time.Second},
		chainID: chainID,
		logger:  logger,
	}
}

// SetSigningIdentity installs the key this process publishes validator-set
// updates as. KNIRVORACLE only ever runs with root.key's own
// RootPrivateKeyHex (fed in as OracleConfig.OwnerPrivateKey — see
// cmd/oracle/rootkey.go), matching §6.6's genesis trust model where root's
// pubkey is the Worker's only anchor before any set exists. If this process's
// node is later demoted (a takeover confirms a different root), the registry
// will simply reject further pushes signed by this identity — expected, not
// an error to work around.
func (rs *RegistrySync) SetSigningIdentity(privateKeyHex string) error {
	signer, err := newRegistrySigner(privateKeyHex)
	if err != nil {
		return err
	}
	rs.signer.Store(signer)
	return nil
}

// Push publishes the full current validator set with a freshly incremented
// version. Best-effort: failures are logged, never propagated to the
// governance mutation that triggered them — a registry outage must not block
// local enrollment/slashing decisions from taking effect.
func (rs *RegistrySync) Push(ctx context.Context, entries []RegistryValidatorEntry) {
	signer := rs.signer.Load()
	if signer == nil {
		return // no identity configured — most likely a non-root node; nothing to push.
	}
	version := rs.version.Add(1)

	encodedValidators, err := json.Marshal(entries)
	if err != nil {
		rs.logger.Warn("registry sync: failed to encode validators", zap.Error(err))
		return
	}
	payload := map[string]any{
		"chainID":    rs.chainID,
		"version":    version,
		"validators": entries,
		"signature":  signer.sign(rs.chainID, strconv.FormatInt(version, 10), string(encodedValidators)),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		rs.logger.Warn("registry sync: failed to encode request", zap.Error(err))
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rs.baseURL+"/validators", bytes.NewReader(body))
	if err != nil {
		rs.logger.Warn("registry sync: failed to build request", zap.Error(err))
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := rs.http.Do(req)
	if err != nil {
		rs.logger.Warn("registry sync: POST /validators failed", zap.Error(err))
		return
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		rs.logger.Warn("registry sync: POST /validators rejected",
			zap.Int("status", resp.StatusCode), zap.ByteString("body", respBody))
		return
	}
	rs.logger.Info("registry sync: validator set published", zap.Int64("version", version), zap.Int("count", len(entries)))
}
