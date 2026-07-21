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

func SeedToFloat(seed [32]byte) float32 {
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
		if x < 0 {
			x = -x
		}
		if x > 1 {
			x = 1
		}
		return x
	}
}

func ProjectSeeds(input []float32, seeds [][32]byte, activation string) []float32 {
	out := make([]float32, len(seeds))
	for i, seed := range seeds {
		sum := float32(0)
		hv := SeedToFloat(seed)
		for _, v := range input {
			sum += v * hv
		}
		out[i] = Activate(sum, activation)
	}
	return out
}

func ProjectSeeds2D(input []float32, seeds [][][32]byte, activation string) []float32 {
	out := make([]float32, len(seeds))
	for i, row := range seeds {
		sum := float32(0)
		for j := 0; j < len(input) && j < len(row); j++ {
			sum += input[j] * SeedToFloat(row[j])
		}
		out[i] = Activate(sum, activation)
	}
	return out
}

func ProjectBack(input []float32, targetDim int, activation string) []float32 {
	out := make([]float32, targetDim)
	sum := float32(0)
	for _, v := range input {
		sum += v
	}
	avg := sum / float32(max(1, len(input)))
	for i := range out {
		out[i] = Activate(avg, activation)
	}
	return out
}

func HashToVocab(hidden []float32, outputSeed [32]byte, vocabSize int) []float32 {
	scores := make([]float32, vocabSize)
	for i := 0; i < vocabSize; i++ {
		var sum float32
		for _, v := range hidden {
			data := make([]byte, 36)
			binary.BigEndian.PutUint32(data[0:4], uint32(i))
			copy(data[4:], outputSeed[:])
			h := sha256.Sum256(data)
			sum += v * SeedToFloat(h)
		}
		scores[i] = sum
	}
	return scores
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
