package transformer

import (
	"crypto/sha256"
	"encoding/binary"
	"math"
	"math/rand"
	"sort"
)

const (
	LayerNormMin = -10.0
	LayerNormMax =  10.0
)

// ffnOutSalt salts derivation of a fallback FFN down-projection seed from a
// layer's existing FFNSeeds, for seed stores persisted before FFNOutSeeds
// existed.
const ffnOutSalt uint32 = 0xF0000000

// ffnOutSeedsOrDerive returns ffnOutSeeds if it already has dim entries.
// Otherwise it deterministically derives dim seeds from ffnSeeds so that old
// (pre-FFNOutSeeds) seed stores keep working without a migration step, in the
// same spirit as computeDecayChain treating empty DecaySeeds as identity.
func ffnOutSeedsOrDerive(ffnOutSeeds [][32]byte, ffnSeeds [][][32]byte, dim int) [][32]byte {
	if len(ffnOutSeeds) >= dim {
		return ffnOutSeeds[:dim]
	}
	derived := make([][32]byte, dim)
	for i := range derived {
		if len(ffnSeeds) == 0 {
			continue
		}
		row := ffnSeeds[i%len(ffnSeeds)]
		if len(row) == 0 {
			continue
		}
		derived[i] = expandSeed(row[i%len(row)], ffnOutSalt+uint32(i))
	}
	return derived
}

// biasSalt is a salt value reserved for deriving a neuron's additive bias from
// its weight seed via expandSeed. Input-dimension salts use the dimension
// index (0..dim-1), which for any realistic EmbedDim/FFNHiddenDim/VocabSize
// never collides with this sentinel.
const biasSalt uint32 = 0xFFFFFFFF

// expandSeed derives a per-connection 32-byte value from a compact seed and an
// integer salt (typically an input-dimension index) via keyed SHA-256
// expansion. This lets one stored 32-byte seed stand in for a full weight
// vector (one derived weight per input dimension) without materializing
// EmbedDim x EmbedDim seed matrices in memory — the same trick HashToVocab
// already used to turn one outputSeed into a full vocabulary of scores.
func expandSeed(seed [32]byte, salt uint32) [32]byte {
	var buf [36]byte
	copy(buf[:32], seed[:])
	binary.BigEndian.PutUint32(buf[32:], salt)
	return sha256.Sum256(buf[:])
}

// SeedToFloat maps a seed to a signed weight in [-1, 1]. Weights need a sign
// so a neuron can inhibit as well as excite a signal; weights restricted to
// [0, 1] can only ever add, never subtract or contrast.
func SeedToFloat(seed [32]byte) float32 {
	val := binary.BigEndian.Uint32(seed[:4])
	return (float32(val)/float32(^uint32(0)))*2 - 1
}

// SeedToUnitFloat maps a seed to [0, 1]. Used where a non-negative fraction is
// required rather than a signed weight — the FoX decay gate alpha_s must stay
// in [0, 1] or the "leaky faucet" forgetting chain could flip sign instead of
// just shrinking toward zero.
func SeedToUnitFloat(seed [32]byte) float32 {
	val := binary.BigEndian.Uint32(seed[:4])
	return float32(val) / float32(^uint32(0))
}

func Activate(x float32, activation string) float32 {
	switch activation {
	case "tanh":
		return float32(math.Tanh(float64(x)))
	case "sigmoid":
		return float32(1.0 / (1.0 + math.Exp(-float64(x))))
	default:
		// Bounded linear clamp to [-1, 1]. Earlier this took the absolute
		// value before clamping to [0, 1], which silently discarded the sign
		// of every weighted sum — pointless once weights (and therefore sums)
		// can be negative.
		if x > 1 {
			return 1
		}
		if x < -1 {
			return -1
		}
		return x
	}
}

// ProjectSeeds projects input through one seed per output neuron. Each seed is
// expanded per input dimension via expandSeed, so a neuron computes a genuine
// weighted combination of the whole input vector. Previously every output
// neuron reused a single scalar weight against the *sum* of the input
// (out[i] = hv_i * sum(input)), which made every neuron in the projection a
// scaled copy of the same underlying number — a rank-1 collapse regardless of
// how many seeds were stored. The bias term is derived from the same seed
// with a reserved salt, so no extra storage is needed.
func ProjectSeeds(input []float32, seeds [][32]byte, activation string) []float32 {
	out := make([]float32, len(seeds))
	for i, seed := range seeds {
		var sum float32
		for j, v := range input {
			sum += v * SeedToFloat(expandSeed(seed, uint32(j)))
		}
		bias := SeedToFloat(expandSeed(seed, biasSalt))
		out[i] = Activate(sum+bias, activation)
	}
	return out
}

// ProjectSeeds2D projects input through a full per-output-neuron weight row
// (already full-rank: one seed per input dimension per output neuron). A bias
// is derived from the row's first seed via a reserved salt.
func ProjectSeeds2D(input []float32, seeds [][][32]byte, activation string) []float32 {
	out := make([]float32, len(seeds))
	for i, row := range seeds {
		var sum float32
		for j := 0; j < len(input) && j < len(row); j++ {
			sum += input[j] * SeedToFloat(row[j])
		}
		var bias float32
		if len(row) > 0 {
			bias = SeedToFloat(expandSeed(row[0], biasSalt))
		}
		out[i] = Activate(sum+bias, activation)
	}
	return out
}

// ProjectBack projects an expanded hidden vector back down to len(seeds)
// output dimensions using one seed per output neuron (see ProjectSeeds).
// Previously this ignored seeds entirely: it averaged every input into one
// scalar and copied that same scalar into every output slot, so every
// dimension of an FFN's output was forced to be identical — a rank-1
// bottleneck independent of any training. It is now the same operation as
// ProjectSeeds; kept as a distinctly-named wrapper for readability at call
// sites that are conceptually "projecting back down".
func ProjectBack(input []float32, seeds [][32]byte, activation string) []float32 {
	return ProjectSeeds(input, seeds, activation)
}

// HashToVocab projects pooled hidden state to per-vocab-token logits. Each
// vocab index derives its own seed from outputSeed via expandSeed, and that
// seed is itself expanded per hidden dimension so a token's score depends on
// the pattern of hidden activations, not just their sum. Previously the hash
// for a vocab index was computed independent of which hidden dimension was
// being summed (recomputed identically inside the loop), so
// scores[i] = weight_i * sum(hidden) for every token — meaning the predicted
// token was almost entirely determined by the fixed per-token weight and
// barely moved with the actual input content.
func HashToVocab(hidden []float32, outputSeed [32]byte, vocabSize int) []float32 {
	scores := make([]float32, vocabSize)
	for i := 0; i < vocabSize; i++ {
		scores[i] = vocabTokenScore(hidden, outputSeed, i)
	}
	return scores
}

// vocabTokenScore computes a single vocab index's score, factored out of
// HashToVocab so the evolutionary trainer's contrastive fitness function
// (evolve.go) can score a handful of candidate tokens without paying for a
// full VocabSize-length pass.
func vocabTokenScore(hidden []float32, outputSeed [32]byte, tokenID int) float32 {
	tokenSeed := expandSeed(outputSeed, uint32(tokenID))
	var sum float32
	for j, v := range hidden {
		sum += v * SeedToFloat(expandSeed(tokenSeed, uint32(j)))
	}
	bias := SeedToFloat(expandSeed(tokenSeed, biasSalt))
	return sum + bias
}

func LayerNorm(x float32, min, max float32) float32 {
	if x < min {
		return min
	}
	if x > max {
		return max
	}
	return x
}

func Argmax32(s []float32) int {
	best := 0
	for i, v := range s {
		if v > s[best] {
			best = i
		}
	}
	return best
}

func SampleTemp32(scores []float32, temp float32) int {
	maxS := scores[0]
	for _, v := range scores {
		if v > maxS {
			maxS = v
		}
	}
	var sum float32
	probs := make([]float32, len(scores))
	for i, v := range scores {
		probs[i] = float32(math.Exp(float64((v - maxS) / temp)))
		sum += probs[i]
	}
	for i := range probs {
		probs[i] /= sum
	}
	var cum float32
	for i, p := range probs {
		cum += p
		if rand.Float32() < cum {
			return i
		}
	}
	return len(scores) - 1
}

func SortFloat32(s []float32) {
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func dotProduct(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var sum float32
	for i := range a {
		sum += a[i] * b[i]
	}
	return sum
}
