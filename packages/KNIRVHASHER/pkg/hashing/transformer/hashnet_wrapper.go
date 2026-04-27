package transformer

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

type HashNetworkWrapper struct {
	available bool
	hash     string
}

func NewHashNetworkWrapper() *HashNetworkWrapper {
	return &HashNetworkWrapper{
		available: false,
		hash:     "",
	}
}

func (hnw *HashNetworkWrapper) IsAvailable() bool {
	return hnw.available
}

func (hnw *HashNetworkWrapper) SetAvailable(available bool) {
	hnw.available = available
}

func (hnw *HashNetworkWrapper) VerifySource(source string) bool {
	if !hnw.available {
		return true
	}

	hash := hnw.ComputeHash(source)
	return hash != ""
}

func (hnw *HashNetworkWrapper) ComputeHash(source string) string {
	if source == "" {
		return ""
	}

	hasher := sha256.New()
	hasher.Write([]byte(source))
	hash := hasher.Sum(nil)
	hashStr := hex.EncodeToString(hash)

	hnw.hash = hashStr
	return hashStr[:16]
}

func (hnw *HashNetworkWrapper) GetHash() string {
	return hnw.hash
}

type FastHashNode struct {
	inputSize  int
	hiddenSize int
	outputSize int
	weights  []float32
}

func NewFastHashNode(inputSize, hiddenSize, outputSize int) *FastHashNode {
	return &FastHashNode{
		inputSize:  inputSize,
		hiddenSize: hiddenSize,
		outputSize: outputSize,
		weights:  make([]float32, inputSize*hiddenSize+hiddenSize*outputSize),
	}
}

func (fhn *FastHashNode) Forward(input []float32) []float32 {
	if len(input) != fhn.inputSize {
		return nil
	}

	hidden := make([]float32, fhn.hiddenSize)
	for i := 0; i < fhn.hiddenSize; i++ {
		sum := float32(0.0)
		for j := 0; j < fhn.inputSize; j++ {
			sum += input[j] * fhn.weights[i*fhn.inputSize+j]
		}
		hidden[i] = sigmoid(sum)
	}

	output := make([]float32, fhn.outputSize)
	for i := 0; i < fhn.outputSize; i++ {
		sum := float32(0.0)
		for j := 0; j < fhn.hiddenSize; j++ {
			sum += hidden[j] * fhn.weights[fhn.inputSize*fhn.hiddenSize+i*fhn.hiddenSize+j]
		}
		output[i] = sigmoid(sum)
	}

	return output
}

func sigmoid(x float32) float32 {
	return float32(1.0) / (float32(1.0) + exp32(-x))
}

func exp32(x float32) float32 {
	if x < -20 {
		return 0
	}
	result := float32(1.0)
	term := float32(1.0)
	for i := 1; i < 20; i++ {
		term *= x / float32(i)
		result += term
		if term < 0.0001 && term > -0.0001 {
			break
		}
	}
	return result
}

func (fhn *FastHashNode) GetWeights() []float32 {
	return fhn.weights
}

func (fhn *FastHashNode) SetWeights(weights []float32) {
	if len(weights) == len(fhn.weights) {
		copy(fhn.weights, weights)
	}
}

type FastHashLayer struct {
	nodes []*FastHashNode
}

func NewFastHashLayer(inputSize, hiddenSize, outputSize, numNodes int) *FastHashLayer {
	nodes := make([]*FastHashNode, numNodes)
	for i := 0; i < numNodes; i++ {
		nodes[i] = NewFastHashNode(inputSize, hiddenSize, outputSize)
	}

	return &FastHashLayer{
		nodes: nodes,
	}
}

func (fhl *FastHashLayer) Forward(input []float32) [][]float32 {
	outputs := make([][]float32, len(fhl.nodes))
	for i, node := range fhl.nodes {
		outputs[i] = node.Forward(input)
	}

	return outputs
}

type HashNetConfig struct {
	InputSize     int
	HiddenSize   int
	OutputSize   int
	NumLayers   int
	NumNodes    int
	Threshold  float32
}

func DefaultHashNetConfig() *HashNetConfig {
	return &HashNetConfig{
		InputSize:   256,
		HiddenSize: 128,
		OutputSize: 64,
		NumLayers: 3,
		NumNodes:  8,
		Threshold: 0.7,
	}
}

type HashNetworkFastPath struct {
	config *HashNetConfig
	layers []*FastHashLayer
}

func NewHashNetworkFastPath(cfg *HashNetConfig) *HashNetworkFastPath {
	layers := make([]*FastHashLayer, cfg.NumLayers)
	for i := 0; i < cfg.NumLayers; i++ {
		layers[i] = NewFastHashLayer(cfg.InputSize, cfg.HiddenSize, cfg.OutputSize, cfg.NumNodes)
	}

	return &HashNetworkFastPath{
		config: cfg,
		layers: layers,
	}
}

func (hnfp *HashNetworkFastPath) Process(input []float32) ([]float32, error) {
	if len(input) != hnfp.config.InputSize {
		return nil, fmt.Errorf("input size mismatch: got %d, want %d", len(input), hnfp.config.InputSize)
	}

	current := input
	for _, layer := range hnfp.layers {
		layerOutputs := layer.Forward(current)

		aggregated := make([]float32, hnfp.config.OutputSize)
		count := 0
		for _, output := range layerOutputs {
			if output != nil {
				for i, v := range output {
					aggregated[i] += v
				}
				count++
			}
		}

		if count > 0 {
			for i := range aggregated {
				aggregated[i] /= float32(count)
			}
		}

		current = aggregated
	}

	return current, nil
}

func (hnfp *HashNetworkFastPath) GetConfidence() float32 {
	return hnfp.config.Threshold
}

type HashSourceIngestion struct {
	sources []string
	index   int
}

func NewHashSourceIngestion() *HashSourceIngestion {
	return &HashSourceIngestion{
		sources: []string{},
		index:   0,
	}
}

func (hsi *HashSourceIngestion) AddSource(source string) {
	hsi.sources = append(hsi.sources, source)
}

func (hsi *HashSourceIngestion) GetSource() string {
	if hsi.index >= len(hsi.sources) {
		return ""
	}
	source := hsi.sources[hsi.index]
	hsi.index++
	return source
}

func (hsi *HashSourceIngestion) HasNext() bool {
	return hsi.index < len(hsi.sources)
}

func (hsi *HashSourceIngestion) Reset() {
	hsi.index = 0
}

func (hsi *HashSourceIngestion) Count() int {
	return len(hsi.sources)
}

type HashSourceMerger struct {
	merges [][]string
}

func NewHashSourceMerger() *HashSourceMerger {
	return &HashSourceMerger{
		merges: [][]string{},
	}
}

func (hsm *HashSourceMerger) Merge(sources ...string) string {
	if len(sources) == 0 {
		return ""
	}
	if len(sources) == 1 {
		return sources[0]
	}

	var result strings.Builder
	for i, s := range sources {
		if i > 0 {
			result.WriteString("\n---\n")
		}
		result.WriteString(s)
	}

	merged := result.String()
	hsm.merges = append(hsm.merges, sources)

	return merged
}

func (hsm *HashSourceMerger) GetMergeCount() int {
	return len(hsm.merges)
}