package transformer

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// DefaultFramesDir is the host directory where the 3_DATA_SEEDER pipeline
// persists mined seeds for this node, per the KNIRV_NETWORK production
// layout. It is one level below the "dataPath" that
// pipeline/3_DATA_SEEDER/pkg/storage/seed_writer.go accepts (that package
// joins "frames" onto dataPath itself); this constant already points at the
// frames directory.
const DefaultFramesDir = "/var/lib/knirvserver/knirvhasher/data/frames"

// seedWritesFile is the append-only ledger written by
// pipeline/3_DATA_SEEDER's DualSeedWriter. It is authoritative for "every
// winning seed ever found," per the comment on ledgerSeedsByKey in that
// package, so it — not the periodically-rewritten training_frames*.json
// snapshots — is what this loader reads.
const seedWritesFile = "seed_writes.jsonl"

// ErrLedgerNotFound is returned when the frames directory or its ledger file
// does not exist, e.g. on a dev machine without /var/lib/knirvserver mounted.
// Callers should treat this as "no mined seeds available yet" and fall back
// to BuildDefaultSeedStore rather than treat it as fatal.
var ErrLedgerNotFound = errors.New("transformer: seed ledger not found")

// ledgerEntry mirrors pipeline/3_DATA_SEEDER/pkg/storage/seed_writer.go's
// SeedWriteLedgerEntry. It is redefined here (rather than imported) because
// KNIRVHASHER's pipeline stages are independent Go modules with no
// cross-package imports, per repo convention.
type ledgerEntry struct {
	SchemaVersion    int32      `json:"schema_version"`
	Timestamp        time.Time  `json:"timestamp"`
	SourceFile       string     `json:"source_file"`
	TargetTokenID    int32      `json:"target_token_id"`
	AssertionKey     string     `json:"assertion_key"`
	ContextTokens    []int32    `json:"context_tokens"`
	AssertionSpan    []int32    `json:"assertion_span"`
	CommitmentTarget uint32     `json:"commitment_target"`
	ContextHash      uint32     `json:"context_hash"`
	AsicSlots        [12]uint32 `json:"asic_slots"`
	BestSeed         string     `json:"best_seed"`
	SeedBytes        int        `json:"seed_bytes"`
}

// FramesLoadStats reports how much of a loaded SeedStore actually came from
// mined ledger data versus random fallback, so operators can tell a "real"
// engine from a freshly-randomized one at a glance.
type FramesLoadStats struct {
	LedgerPath    string
	LedgerRecords int
	TokensCovered int
	TokensTotal   int
}

// ledgerCacheTTL bounds how long a parsed ledger is reused before being
// re-read. The ledger is a 70MB+/250k-line append-only file in production;
// without caching, every call to LoadSeedStoreFromFrames would re-parse it
// end to end, and callers like inferTransformer in
// pkg/hashing/inference/recursive.go construct a fresh engine (and therefore
// fresh seeds) per inference request — that would turn every single
// inference call into a multi-second file scan.
const ledgerCacheTTL = 5 * time.Minute

type ledgerCacheEntry struct {
	tokenSeeds map[int32][]byte
	records    int
	err        error
	loadedAt   time.Time
}

var (
	ledgerCacheMu sync.Mutex
	ledgerCache   = map[string]ledgerCacheEntry{}
)

// InvalidateFramesCache drops any cached ledger parse for framesDir (or all
// cached directories if framesDir is empty), forcing the next
// LoadSeedStoreFromFrames call to re-read from disk. Useful after a known
// 3_DATA_SEEDER write burst if callers don't want to wait out ledgerCacheTTL.
func InvalidateFramesCache(framesDir string) {
	ledgerCacheMu.Lock()
	defer ledgerCacheMu.Unlock()
	if framesDir == "" {
		ledgerCache = map[string]ledgerCacheEntry{}
		return
	}
	delete(ledgerCache, framesDir)
}

func loadLedgerTokenSeedsCached(framesDir string) (map[int32][]byte, int, error) {
	ledgerCacheMu.Lock()
	if entry, ok := ledgerCache[framesDir]; ok && time.Since(entry.loadedAt) < ledgerCacheTTL {
		ledgerCacheMu.Unlock()
		return entry.tokenSeeds, entry.records, entry.err
	}
	ledgerCacheMu.Unlock()

	tokenSeeds, records, err := loadLedgerTokenSeeds(framesDir)

	ledgerCacheMu.Lock()
	ledgerCache[framesDir] = ledgerCacheEntry{tokenSeeds: tokenSeeds, records: records, err: err, loadedAt: time.Now()}
	ledgerCacheMu.Unlock()

	return tokenSeeds, records, err
}

// loadLedgerTokenSeeds reads seed_writes.jsonl and folds every best_seed
// found for a given target_token_id into one aggregate 32-byte value per
// token (sha256 of the sorted, concatenated seed bytes — sorted so the result
// doesn't depend on ledger read order, since a token is typically mined
// from many different context buckets across the file).
func loadLedgerTokenSeeds(framesDir string) (map[int32][]byte, int, error) {
	path := filepath.Join(framesDir, seedWritesFile)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, fmt.Errorf("%w: %s", ErrLedgerNotFound, path)
		}
		return nil, 0, fmt.Errorf("open seed ledger: %w", err)
	}
	defer f.Close()

	perToken := make(map[int32][][]byte)
	scanner := bufio.NewReader(f)
	records := 0
	for {
		line, err := scanner.ReadString('\n')
		trimmed := trimNewline(line)
		if len(trimmed) > 0 {
			var entry ledgerEntry
			if jsonErr := json.Unmarshal(trimmed, &entry); jsonErr == nil && entry.BestSeed != "" {
				seedBytes, decErr := base64.StdEncoding.DecodeString(entry.BestSeed)
				if decErr == nil && len(seedBytes) > 0 {
					perToken[entry.TargetTokenID] = append(perToken[entry.TargetTokenID], seedBytes)
					records++
				}
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, records, fmt.Errorf("read seed ledger: %w", err)
		}
	}

	aggregated := make(map[int32][]byte, len(perToken))
	for tokenID, seeds := range perToken {
		sort.Slice(seeds, func(i, j int) bool {
			return string(seeds[i]) < string(seeds[j])
		})
		h := sha256.New()
		for _, s := range seeds {
			h.Write(s)
		}
		aggregated[tokenID] = h.Sum(nil)
	}
	return aggregated, records, nil
}

func trimNewline(s string) []byte {
	b := []byte(s)
	for len(b) > 0 && (b[len(b)-1] == '\n' || b[len(b)-1] == '\r') {
		b = b[:len(b)-1]
	}
	return b
}

// expandAggregateSeed turns a token's folded ledger material into dim
// distinct 32-byte seeds via keyed SHA-256 expansion (the same technique
// expandSeed uses elsewhere in this package), so a real, data-derived value
// backs every embedding dimension instead of just one.
func expandAggregateSeed(aggregate []byte, dim int) [][32]byte {
	out := make([][32]byte, dim)
	for j := 0; j < dim; j++ {
		buf := make([]byte, len(aggregate)+4)
		copy(buf, aggregate)
		binary.BigEndian.PutUint32(buf[len(aggregate):], uint32(j))
		out[j] = sha256.Sum256(buf)
	}
	return out
}

// LoadSeedStoreFromFrames builds a SeedStore the same shape as
// BuildDefaultSeedStore, but for every vocabulary token the mining ledger in
// framesDir has evidence for, the token's embedding row is derived
// deterministically from the real "golden seeds" 3_DATA_SEEDER's
// evolutionary Hamming-similarity search found for it — instead of being
// pure crypto/rand noise. Tokens absent from the ledger, and all
// non-token-indexed seeds (positional, per-layer Q/K/V/Output/FFN/Decay),
// still fall back to crypto/rand: the ledger only records associations
// between quantized context buckets and target tokens, so it has no
// equivalent signal for those.
//
// Returns ErrLedgerNotFound (wrapped) if framesDir has no seed_writes.jsonl —
// callers should treat that as "nothing mined yet" and use
// BuildDefaultSeedStore directly rather than failing startup.
func LoadSeedStoreFromFrames(framesDir string, cfg *UnifiedConfig) (*SeedStore, *FramesLoadStats, error) {
	if cfg == nil {
		cfg = DefaultUnifiedConfig()
	}
	tokenSeeds, records, err := loadLedgerTokenSeedsCached(framesDir)
	if err != nil {
		return nil, nil, err
	}

	store := BuildDefaultSeedStore(cfg)
	covered := 0
	for tokenID, aggregate := range tokenSeeds {
		if tokenID < 0 || int(tokenID) >= len(store.Embeddings) {
			continue
		}
		store.Embeddings[tokenID] = expandAggregateSeed(aggregate, cfg.EmbedDim)
		covered++
	}

	stats := &FramesLoadStats{
		LedgerPath:    filepath.Join(framesDir, seedWritesFile),
		LedgerRecords: records,
		TokensCovered: covered,
		TokensTotal:   cfg.VocabSize,
	}
	return store, stats, nil
}

// LoadOrBuildSeedStore tries LoadSeedStoreFromFrames(framesDir, cfg) and
// falls back to BuildDefaultSeedStore (pure crypto/rand) when the ledger is
// unavailable — e.g. a dev machine without /var/lib/knirvserver mounted.
// This is the constructor every engine entry point should go through instead
// of calling BuildDefaultSeedStore directly, so a node with mined data
// actually uses it instead of silently re-randomizing on every startup and
// every seedless /reload.
func LoadOrBuildSeedStore(framesDir string, cfg *UnifiedConfig) (*SeedStore, *FramesLoadStats) {
	if cfg == nil {
		cfg = DefaultUnifiedConfig()
	}
	store, stats, err := LoadSeedStoreFromFrames(framesDir, cfg)
	if err != nil {
		log.Printf("[transformer] seed ledger unavailable (%v); using unmined crypto/rand seeds", err)
		return BuildDefaultSeedStore(cfg), nil
	}
	log.Printf("[transformer] loaded seeds from mining ledger %s: %d records, %d/%d vocab tokens backed by mined data",
		stats.LedgerPath, stats.LedgerRecords, stats.TokensCovered, stats.TokensTotal)
	return store, stats
}
