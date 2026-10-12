package transformer

import (
	"fmt"
	"math"
	"sort"
)

type ModelPruner struct {
	threshold   float32
	method      PruneMethod
	sparsity    float32
	initialized bool
}

type PruneMethod int

const (
	PruneMagnitude PruneMethod = iota
	PruneThreshold
	PruneGradient
	PruneLottery
)

func NewModelPruner(threshold float32, method PruneMethod) *ModelPruner {
	return &ModelPruner{
		threshold:   threshold,
		method:      method,
		sparsity:    0.0,
		initialized: true,
	}
}

func (mp *ModelPruner) PruneWeights(weights []float32) ([]float32, error) {
	if !mp.initialized {
		return nil, fmt.Errorf("pruner not initialized")
	}

	switch mp.method {
	case PruneMagnitude:
		return mp.pruneMagnitude(weights), nil
	case PruneThreshold:
		return mp.pruneThreshold(weights), nil
	case PruneGradient:
		return mp.pruneGradient(weights), nil
	case PruneLottery:
		return mp.pruneLottery(weights), nil
	}

	return weights, nil
}

func (mp *ModelPruner) pruneMagnitude(weights []float32) []float32 {
	result := make([]float32, len(weights))
	copy(result, weights)

	absWeights := make([]float32, len(weights))
	for i, w := range weights {
		absWeights[i] = absFloat32(w)
	}

	sort.Slice(absWeights, func(i, j int) bool {
		return absWeights[i] < absWeights[j]
	})

	thresholdIdx := int(float32(len(absWeights)) * mp.threshold)
	if thresholdIdx >= len(absWeights) {
		thresholdIdx = len(absWeights) - 1
	}
	thresholdValue := absWeights[thresholdIdx]

	for i, w := range result {
		if absFloat32(w) < thresholdValue {
			result[i] = 0.0
			mp.sparsity += 1.0
		}
	}

	if len(weights) > 0 {
		mp.sparsity /= float32(len(weights))
	}

	return result
}

func (mp *ModelPruner) pruneThreshold(weights []float32) []float32 {
	result := make([]float32, len(weights))

	for i, w := range weights {
		if absFloat32(w) < mp.threshold {
			result[i] = 0.0
		} else {
			result[i] = w
		}
	}

	return result
}

func (mp *ModelPruner) pruneGradient(weights []float32) []float32 {
	result := make([]float32, len(weights))

	gradients := make([]float32, len(weights))
	for i := range gradients {
		gradients[i] = float32(i + 1)
	}

	threshold := calculateGradientThreshold(gradients, mp.threshold)

	for i, w := range weights {
		if absFloat32(gradients[i]) < threshold {
			result[i] = 0.0
		} else {
			result[i] = w
		}
	}

	return result
}

func (mp *ModelPruner) pruneLottery(weights []float32) []float32 {
	result := make([]float32, len(weights))

	scores := make([]float32, len(weights))
	for i := range weights {
		scores[i] = absFloat32(weights[i]) * lotteryScore(i, len(weights))
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i] > scores[j]
	})

	numToKeep := int(float32(len(weights)) * (float32(1.0) - mp.threshold))
	if numToKeep > len(weights) {
		numToKeep = len(weights)
	}

	for i := range result {
		if i < int(numToKeep) {
			result[i] = weights[i]
		} else {
			result[i] = 0.0
		}
	}

	return result
}

func absFloat32(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

func calculateGradientThreshold(gradients []float32, percentile float32) float32 {
	if len(gradients) == 0 {
		return 0.0
	}

	sorted := make([]float32, len(gradients))
	copy(sorted, gradients)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	idx := int(float32(len(sorted)) * percentile)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}

	return sorted[idx]
}

func lotteryScore(index, total int) float32 {
	normalizedIndex := float32(index) / float32(total)
	return float32(1.0) - normalizedIndex
}

type PruningSchedule struct {
	initialThreshold float32
	finalThreshold  float32
	currentEpoch   int
	totalEpochs    int
	method         PruneMethod
}

func NewPruningSchedule(initialThreshold, finalThreshold float32, totalEpochs int, method PruneMethod) *PruningSchedule {
	return &PruningSchedule{
		initialThreshold: initialThreshold,
		finalThreshold:   finalThreshold,
		currentEpoch:    0,
		totalEpochs:     totalEpochs,
		method:          method,
	}
}

func (ps *PruningSchedule) GetThreshold() float32 {
	progress := float32(ps.currentEpoch) / float32(ps.totalEpochs)
	threshold := ps.initialThreshold + (ps.finalThreshold-ps.initialThreshold)*progress

	return threshold
}

func (ps *PruningSchedule) Step() {
	if ps.currentEpoch < ps.totalEpochs {
		ps.currentEpoch++
	}
}

func (ps *PruningSchedule) IsDone() bool {
	return ps.currentEpoch >= ps.totalEpochs
}

func (ps *PruningSchedule) GetProgress() float32 {
	return float32(ps.currentEpoch) / float32(ps.totalEpochs)
}

type PruningStats struct {
	numParameters    int
	numZeroed      int
	sparsity      float32
	totalFlops    int64
	prunedFlops   int64
	compression float32
}

func NewPruningStats(numParams int) *PruningStats {
	return &PruningStats{
		numParameters: numParams,
		numZeroed:     0,
		sparsity:      0.0,
		totalFlops:    int64(numParams * 2),
		prunedFlops:   0,
		compression: 0.0,
	}
}

func (ps *PruningStats) Update(weights []float32) {
	zeroCount := 0
	for _, w := range weights {
		if absFloat32(w) < 0.0001 {
			zeroCount++
		}
	}

	ps.numZeroed = zeroCount
	ps.sparsity = float32(zeroCount) / float32(ps.numParameters)

	if ps.numParameters > 0 {
		ps.prunedFlops = int64((ps.numParameters - zeroCount) * 2)
		ps.compression = float32(ps.numParameters) / float32(ps.numParameters-zeroCount)
	}
}

func (ps *PruningStats) GetSparsity() float32 {
	return ps.sparsity
}

func (ps *PruningStats) GetCompression() float32 {
	return ps.compression
}

func (ps *PruningStats) String() string {
	return fmt.Sprintf("PruningStats{sparsity=%.2f, compression=%.2f, zeroed=%d/%d}",
		ps.sparsity, ps.compression, ps.numZeroed, ps.numParameters)
}

type TransformerPruner struct {
	model          *GPT
	pruner        *ModelPruner
	schedule     *PruningSchedule
	stats        *PruningStats
	prunedWeights map[string][]float32
}

func NewTransformerPruner(model *GPT, initialThreshold, finalThreshold float32, epochs int, method PruneMethod) *TransformerPruner {
	numParams := countModelParameters(model)

	return &TransformerPruner{
		model:      model,
		pruner:     NewModelPruner(initialThreshold, method),
		schedule:  NewPruningSchedule(initialThreshold, finalThreshold, epochs, method),
		stats:     NewPruningStats(numParams),
		prunedWeights: make(map[string][]float32),
	}
}

func (tp *TransformerPruner) Prune() (map[string][]float32, error) {
	threshold := tp.schedule.GetThreshold()
	tp.pruner.threshold = threshold

	if tp.model == nil {
		return nil, fmt.Errorf("model is nil")
	}

	weights := collectModelWeights(tp.model)
	prunedWeights := make(map[string][]float32)

	for name, w := range weights {
		pruned, err := tp.pruner.PruneWeights(w)
		if err != nil {
			return nil, err
		}
		prunedWeights[name] = pruned
	}

	tp.prunedWeights = prunedWeights
	tp.stats.Update(flattenWeights(prunedWeights))

	return prunedWeights, nil
}

func (tp *TransformerPruner) Step() {
	tp.schedule.Step()
}

func (tp *TransformerPruner) GetProgress() float32 {
	return tp.schedule.GetProgress()
}

func (tp *TransformerPruner) GetStats() *PruningStats {
	return tp.stats
}

func countModelParameters(model *GPT) int {
	count := 0
	if model == nil {
		return 0
	}

	if model.config != nil {
		count += model.config.VocabSize * model.config.EmbedDim
		count += model.config.EmbedDim * model.config.FFNHiddenDim * 2 * model.config.NumLayers
		count += model.config.EmbedDim * model.config.VocabSize
	}

	return count
}

func collectModelWeights(model *GPT) map[string][]float32 {
	weights := make(map[string][]float32)

	if model == nil || model.config == nil {
		return weights
	}

	embedSize := model.config.VocabSize * model.config.EmbedDim
	weights["embedding"] = make([]float32, embedSize)
	for i := 0; i < embedSize && i < len(weights["embedding"]); i++ {
		weights["embedding"][i] = float32(i % 100)
	}

	ffnSize := model.config.EmbedDim * model.config.FFNHiddenDim
	for layer := 0; layer < model.config.NumLayers; layer++ {
		key := fmt.Sprintf("layer_%d_ffn_w1", layer)
		weights[key] = make([]float32, ffnSize)
		for i := 0; i < ffnSize && i < len(weights[key]); i++ {
			weights[key][i] = float32(i % 100)
		}
	}

	outSize := model.config.EmbedDim * model.config.VocabSize
	weights["output"] = make([]float32, outSize)
	for i := 0; i < outSize && i < len(weights["output"]); i++ {
		weights["output"][i] = float32(i % 100)
	}

	return weights
}

func flattenWeights(weights map[string][]float32) []float32 {
	var flat []float32
	for _, w := range weights {
		flat = append(flat, w...)
	}
	return flat
}

var _ = math.Log