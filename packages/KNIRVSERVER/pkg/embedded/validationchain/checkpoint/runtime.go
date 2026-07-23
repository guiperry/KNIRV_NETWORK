package checkpoint

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
)

const chainID = "validation-chain"

// Runtime polls the embedded Validation Chain's local HTTP API (over its own
// Unix domain socket — it's genuinely co-located with this process) for new
// blocks, and posts signed checkpoints to KNIRVORACLE over the public
// KNIRVGATEWAY (see Poster) — never a local socket, since KNIRVORACLE only
// runs on the root node and this code runs on every node. It intentionally
// does not implement the transition-proof / local-verifier handoff
// KNIRVCHAIN's equivalent runtime does (merkle-math.md Phase 4/5) — that is
// a separate, more advanced finality-attestation layer; this runtime only
// covers checkpoint submission, which is enough for Validation Chain to
// become the registered merkle source.
type Runtime struct {
	poster     *Poster
	signer     *ecdsa.PrivateKey
	httpClient *http.Client

	lastEnd  uint64
	prevHash [32]byte
}

// NewRuntime builds a Runtime. chainSocketPath is the Validation Chain's own
// local Unix domain socket (genuinely co-located with this process);
// gatewayURL is the public KNIRVGATEWAY base URL (e.g. KNIRVSERVER's
// own resolvePublicURL() result), and oracleURL is the actual KNIRVORACLE
// endpoint (KNIRV_ORACLE_URL override if set, otherwise the same as gatewayURL).
// KNIRVORACLE is never assumed local.
func NewRuntime(chainSocketPath, gatewayURL, oracleURL string, signer *ecdsa.PrivateKey) *Runtime {
	return &Runtime{
		poster: NewPoster(gatewayURL, oracleURL),
		signer: signer,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					return (&net.Dialer{}).DialContext(ctx, "unix", chainSocketPath)
				},
			},
		},
	}
}

// Run registers the chain once, then polls for new blocks and posts a
// checkpoint whenever the tip advances, until ctx is cancelled. Intended to
// run in its own goroutine.
func (rt *Runtime) Run(ctx context.Context, pollInterval time.Duration) {
	if err := rt.ensureRegistered(ctx); err != nil {
		log.Printf("Warning: Validation Chain checkpoint registration failed (will retry on next poll): %v", err)
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := rt.ensureRegistered(ctx); err != nil {
				log.Printf("Warning: Validation Chain checkpoint registration failed: %v", err)
				continue
			}
			if err := rt.tick(ctx); err != nil {
				log.Printf("Warning: Validation Chain checkpoint post failed: %v", err)
			}
		}
	}
}

func (rt *Runtime) ensureRegistered(ctx context.Context) error {
	pub := rt.signer.Public().(*ecdsa.PublicKey)
	address := oracleAddress(pub)
	reg := &ChainRegistration{
		ChainID: chainID,
		Authors: []RegisteredAuthor{
			{Address: address, PubKey: crypto.CompressPubkey(pub), Weight: 1},
		},
		QuorumNumer: 1,
		QuorumDenom: 1,
		ProofWindow: 0,
	}
	if err := SignRegistration(reg, rt.signer); err != nil {
		return fmt.Errorf("sign registration: %w", err)
	}
	if _, err := rt.poster.RegisterChain(ctx, reg); err != nil {
		if strings.Contains(err.Error(), "already registered") {
			return nil
		}
		return err
	}
	return nil
}

func (rt *Runtime) tick(ctx context.Context) error {
	height, err := rt.chainHeight(ctx)
	if err != nil {
		return fmt.Errorf("query validation chain height: %w", err)
	}
	if height <= rt.lastEnd {
		return nil
	}
	root, err := rt.tipRoot(ctx)
	if err != nil {
		return fmt.Errorf("query validation chain tip root: %w", err)
	}

	cp := &Checkpoint{
		SchemaVersion: SchemaVersion,
		ChainID:       chainID,
		StartHeight:   rt.lastEnd + 1,
		EndHeight:     height,
		Root:          root,
		PrevCheckHash: rt.prevHash,
		Proposer:      oracleAddress(rt.signer.Public().(*ecdsa.PublicKey)),
	}
	if err := SignCheckpoint(cp, rt.signer); err != nil {
		return fmt.Errorf("sign checkpoint: %w", err)
	}
	if _, err := rt.poster.PostCheckpoint(ctx, cp); err != nil {
		return err
	}

	rt.lastEnd = cp.EndHeight
	rt.prevHash = cp.Digest()
	log.Printf("Validation Chain checkpoint posted to KNIRVORACLE: height %d-%d", cp.StartHeight, cp.EndHeight)
	return nil
}

func (rt *Runtime) chainHeight(ctx context.Context) (uint64, error) {
	var out struct {
		Height uint64 `json:"height"`
	}
	if err := rt.getJSON(ctx, "/chain/height", &out); err != nil {
		return 0, err
	}
	return out.Height, nil
}

// tipRoot fetches the current block list and returns the latest block's own
// hash as the checkpoint root. Each Validation Chain block hash is computed
// over (index, previous_hash, timestamp, data, nonce), so the tip hash is
// already a hash-chained accumulator over the chain's full history —
// equivalent in spirit to KNIRVCHAIN's AccumRoot.
func (rt *Runtime) tipRoot(ctx context.Context) ([32]byte, error) {
	var blocks []struct {
		Index uint64 `json:"index"`
		Hash  string `json:"hash"`
	}
	if err := rt.getJSON(ctx, "/blocks", &blocks); err != nil {
		return [32]byte{}, err
	}
	if len(blocks) == 0 {
		return [32]byte{}, fmt.Errorf("validation chain has no blocks")
	}
	tip := blocks[len(blocks)-1]
	raw, err := hex.DecodeString(tip.Hash)
	if err != nil || len(raw) != 32 {
		return [32]byte{}, fmt.Errorf("validation chain tip hash %q is not 32 bytes hex", tip.Hash)
	}
	var root [32]byte
	copy(root[:], raw)
	return root, nil
}

func (rt *Runtime) getJSON(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://unix"+path, nil)
	if err != nil {
		return err
	}
	resp, err := rt.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s returned %d", path, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
