package transformer

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
)

// TrainingRecord is one (context, target) supervision example: Context is a
// sequence of token IDs and TargetTokenID is the token that should follow.
//
// When built from LoadTrainingRecordsFromLedger, Context is a stand-in: the
// mining ledger only records a quantized 12-slot context "bucket" per
// example, not the original token sequence (the pipeline discards raw tokens
// after encoding them into asic_slot buckets — see
// pipeline/1_DATA_MAPPER/.../embedder and 2_DATA_ENCODER). Each slot is
// folded into a synthetic token ID (slot mod VocabSize) so there is a
// concrete, computable input to run through the engine. This is an honest
// approximation, not a reconstruction of the real context — it gives the
// evolutionary search a real, reproducible fitness signal tied to actual
// mined (context-bucket, token) associations, which is strictly more signal
// than the crypto/rand baseline, but it is not equivalent to training on the
// original text.
type TrainingRecord struct {
	Context       []int
	TargetTokenID int
}

// EvolveConfig configures the evolution-strategies seed search.
type EvolveConfig struct {
	Generations     int   // mutate-and-test rounds
	MutationsPerGen int   // seed entries mutated per candidate
	NegativeSamples int   // contrastive negatives sampled per fitness evaluation
	Seed            int64 // RNG seed, for reproducible runs
}

// DefaultEvolveConfig returns modest defaults suitable for iterative use;
// callers training at production scale should raise Generations.
func DefaultEvolveConfig() EvolveConfig {
	return EvolveConfig{
		Generations:     50,
		MutationsPerGen: 8,
		NegativeSamples: 16,
		Seed:            1,
	}
}

// LoadTrainingRecordsFromLedger reads seed_writes.jsonl and turns each entry
// into a TrainingRecord (see TrainingRecord's doc for the asic_slots ->
// context caveat). maxRecords caps how many are read (<=0 means unbounded);
// the ledger has hundreds of thousands of lines, so capping is normally
// required to keep an evolutionary run's per-generation cost bounded.
func LoadTrainingRecordsFromLedger(framesDir string, vocabSize, maxRecords int) ([]TrainingRecord, error) {
	if vocabSize <= 0 {
		return nil, fmt.Errorf("LoadTrainingRecordsFromLedger: vocabSize must be positive")
	}
	path := filepath.Join(framesDir, seedWritesFile)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrLedgerNotFound, path)
		}
		return nil, fmt.Errorf("open seed ledger: %w", err)
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	var records []TrainingRecord
	for maxRecords <= 0 || len(records) < maxRecords {
		line, readErr := reader.ReadString('\n')
		trimmed := trimNewline(line)
		if len(trimmed) > 0 {
			var entry ledgerEntry
			if jsonErr := json.Unmarshal(trimmed, &entry); jsonErr == nil {
				context := make([]int, len(entry.AsicSlots))
				for i, slot := range entry.AsicSlots {
					context[i] = int(slot % uint32(vocabSize))
				}
				target := int(uint32(entry.TargetTokenID) % uint32(vocabSize))
				records = append(records, TrainingRecord{Context: context, TargetTokenID: target})
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return records, fmt.Errorf("read seed ledger: %w", readErr)
		}
	}
	return records, nil
}

// fitness scores how well seeds rank each record's true target token against
// a sample of random negative tokens: the fraction of negatives the true
// token's score beats, averaged across records, in [0, 1]. This is a
// contrastive (noise-contrastive-style) estimate rather than a full softmax
// loss over the vocabulary — evaluating the true HashToVocab over the whole
// vocabulary for every candidate in every generation would be
// O(Generations * VocabSize * EmbedDim) hash calls, which is impractical.
// Sampling negatives is a standard, real technique for exactly this
// trade-off (as in word2vec's negative sampling): a candidate that
// consistently outranks a modest random sample makes real progress against
// the full ranking too.
//
// This replaces the previous "training" signal in this codebase (pruner.go's
// pruneGradient, which used gradients[i] = i+1 — the array index, not
// anything derived from data or loss) with one that actually depends on the
// ledger's real (context, target) associations and on the candidate seeds
// being evaluated.
func fitness(cfg *UnifiedConfig, seeds *SeedStore, records []TrainingRecord, evCfg EvolveConfig, rng *rand.Rand) float64 {
	if len(records) == 0 {
		return 0
	}
	engine := NewUnifiedHasherEngineWithConfig(cfg, seeds, nil, ModeTransformer)
	var total float64
	for _, rec := range records {
		hidden := engine.Forward(rec.Context)
		targetScore := vocabTokenScore(hidden, seeds.OutputSeed, rec.TargetTokenID)
		negatives := evCfg.NegativeSamples
		if negatives <= 0 {
			negatives = 1
		}
		wins := 0
		tried := 0
		for n := 0; n < negatives; n++ {
			neg := rng.Intn(cfg.VocabSize)
			if neg == rec.TargetTokenID {
				continue
			}
			tried++
			if targetScore > vocabTokenScore(hidden, seeds.OutputSeed, neg) {
				wins++
			}
		}
		if tried == 0 {
			total += 1 // no valid negatives to compare against; treat as a win
			continue
		}
		total += float64(wins) / float64(tried)
	}
	return total / float64(len(records))
}

// cloneSeedStore deep-copies a SeedStore via JSON round-trip. Correctness
// over speed: SeedStore's nested slice-of-slice-of-[32]byte shape is
// error-prone to hand-copy, and clones only happen once per generation, not
// per fitness evaluation.
func cloneSeedStore(store *SeedStore) *SeedStore {
	data, err := json.Marshal(store)
	if err != nil {
		return store
	}
	var clone SeedStore
	if err := json.Unmarshal(data, &clone); err != nil {
		return store
	}
	return &clone
}

func randomSeedBytes(rng *rand.Rand) [32]byte {
	var s [32]byte
	rng.Read(s[:])
	return s
}

// tokensInRecords collects the distinct in-range token IDs referenced by a
// batch of training records, so mutation can target embeddings that are
// actually exercised by the fitness function instead of picking blindly out
// of a 100k-token vocabulary.
func tokensInRecords(records []TrainingRecord, vocabSize int) []int {
	seen := make(map[int]bool)
	for _, r := range records {
		for _, tok := range r.Context {
			if tok >= 0 && tok < vocabSize {
				seen[tok] = true
			}
		}
		if r.TargetTokenID >= 0 && r.TargetTokenID < vocabSize {
			seen[r.TargetTokenID] = true
		}
	}
	toks := make([]int, 0, len(seen))
	for t := range seen {
		toks = append(toks, t)
	}
	sort.Ints(toks)
	return toks
}

// mutate2D replaces one random [32]byte seed inside a 2D seed matrix.
func mutate2D(seeds [][][32]byte, rng *rand.Rand) {
	if len(seeds) == 0 {
		return
	}
	row := seeds[rng.Intn(len(seeds))]
	if len(row) == 0 {
		return
	}
	row[rng.Intn(len(row))] = randomSeedBytes(rng)
}

// mutate1D replaces one random [32]byte seed inside a flat seed vector.
func mutate1D(seeds [][32]byte, rng *rand.Rand) {
	if len(seeds) == 0 {
		return
	}
	seeds[rng.Intn(len(seeds))] = randomSeedBytes(rng)
}

// mutateLayerSeed point-mutates one seed somewhere in a single transformer
// layer (chosen uniformly across the layer's seed categories, including
// per-head Q/K/V).
func mutateLayerSeed(layer *TransformerLayerSeeds, rng *rand.Rand) {
	switch rng.Intn(6) {
	case 0:
		if len(layer.QuerySeeds) > 0 {
			mutate1D(layer.QuerySeeds[rng.Intn(len(layer.QuerySeeds))], rng)
		}
	case 1:
		if len(layer.KeySeeds) > 0 {
			mutate1D(layer.KeySeeds[rng.Intn(len(layer.KeySeeds))], rng)
		}
	case 2:
		if len(layer.ValueSeeds) > 0 {
			mutate1D(layer.ValueSeeds[rng.Intn(len(layer.ValueSeeds))], rng)
		}
	case 3:
		mutate2D(layer.OutputSeeds, rng)
	case 4:
		mutate2D(layer.FFNSeeds, rng)
	case 5:
		if rng.Intn(2) == 0 {
			mutate1D(layer.DecaySeeds, rng)
		} else {
			mutate1D(layer.FFNOutSeeds, rng)
		}
	}
}

// mutateOnce applies one point mutation to store, chosen among: an embedding
// row for a token actually exercised by the current training batch, the
// output seed, or a random seed inside a random layer.
func mutateOnce(store *SeedStore, rng *rand.Rand, batchTokens []int) {
	switch rng.Intn(3) {
	case 0:
		if len(batchTokens) == 0 || len(store.Embeddings) == 0 {
			return
		}
		tok := batchTokens[rng.Intn(len(batchTokens))]
		if tok < 0 || tok >= len(store.Embeddings) {
			return
		}
		mutate1D(store.Embeddings[tok], rng)
	case 1:
		store.OutputSeed = randomSeedBytes(rng)
	case 2:
		if len(store.Layers) == 0 {
			return
		}
		mutateLayerSeed(&store.Layers[rng.Intn(len(store.Layers))], rng)
	}
}

// EvolveSeeds runs a (1+1) evolution-strategy search over seeds: each
// generation mutates a clone, scores it against records with fitness, and
// keeps the mutation only if it does not make fitness worse. This is a real,
// if simple, optimizer — every accepted step is backed by a measured
// improvement (or no change) against real ledger-derived data, unlike the
// prior state of this codebase where seeds were pure crypto/rand noise for
// the lifetime of the process (BuildDefaultSeedStore, called fresh on every
// engine construction and on every /reload without a custom payload) and the
// only code claiming to be a trainer (HasherTrainer.Train) was a one-line
// no-op.
//
// Returns the evolved SeedStore and the per-generation fitness history
// (monotonically non-decreasing by construction, since a generation's result
// only replaces the incumbent when it is at least as good).
func EvolveSeeds(cfg *UnifiedConfig, seeds *SeedStore, records []TrainingRecord, evCfg EvolveConfig) (*SeedStore, []float64, error) {
	if cfg == nil {
		return nil, nil, fmt.Errorf("EvolveSeeds: nil config")
	}
	if seeds == nil {
		return nil, nil, fmt.Errorf("EvolveSeeds: nil seeds")
	}
	if len(records) == 0 {
		return nil, nil, fmt.Errorf("EvolveSeeds: no training records")
	}
	if evCfg.Generations <= 0 {
		evCfg = DefaultEvolveConfig()
	}

	rng := rand.New(rand.NewSource(evCfg.Seed))
	batchTokens := tokensInRecords(records, cfg.VocabSize)

	current := cloneSeedStore(seeds)
	currentFitness := fitness(cfg, current, records, evCfg, rng)
	history := make([]float64, 0, evCfg.Generations+1)
	history = append(history, currentFitness)

	for gen := 0; gen < evCfg.Generations; gen++ {
		candidate := cloneSeedStore(current)
		mutations := evCfg.MutationsPerGen
		if mutations <= 0 {
			mutations = 1
		}
		for m := 0; m < mutations; m++ {
			mutateOnce(candidate, rng, batchTokens)
		}
		candidateFitness := fitness(cfg, candidate, records, evCfg, rng)
		if candidateFitness >= currentFitness {
			current = candidate
			currentFitness = candidateFitness
		}
		history = append(history, currentFitness)
	}

	return current, history, nil
}
