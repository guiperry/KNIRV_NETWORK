package transformer

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"sort"
	"strings"

	tiktoken "github.com/pkoukk/tiktoken-go"
	"gorgonia.org/gorgonia"
	"gorgonia.org/tensor"
)

type GorgoniteConfig struct {
	VocabSize    int
	EmbedDim     int
	NumHeads     int
	NumLayers    int
	ContextLen   int
	DropoutRate  float64
	FFNHiddenDim int
}

func DefaultGorgoniteConfig() *GorgoniteConfig {
	return &GorgoniteConfig{
		VocabSize:    100277,
		EmbedDim:     768,
		NumHeads:     12,
		NumLayers:    12,
		ContextLen:  2048,
		DropoutRate: 0.1,
		FFNHiddenDim: 3072,
	}
}

type EmbeddingLayer struct {
	graph      *gorgonia.ExprGraph
	embeddings *gorgonia.Node
	vocabSize  int
	embedDim   int
	seqLen     int
}

func NewEmbeddingLayer(g *gorgonia.ExprGraph, vocabSize, embedDim, seqLen int) *EmbeddingLayer {
	embeddings := gorgonia.NewMatrix(g,
		tensor.Float32,
		gorgonia.WithShape(seqLen, embedDim),
		gorgonia.WithName("embeddings"),
		gorgonia.WithInit(gorgonia.GlorotN(1.0)),
	)

	return &EmbeddingLayer{
		graph:     g,
		embeddings: embeddings,
		vocabSize: vocabSize,
		embedDim:  embedDim,
		seqLen:    seqLen,
	}
}

func (e *EmbeddingLayer) Forward(seqLen int) (*gorgonia.Node, error) {
	return e.embeddings, nil
}

type PositionalEncoding struct {
	graph    *gorgonia.ExprGraph
	encoding *gorgonia.Node
	seqLen   int
	embedDim int
}

func NewPositionalEncoding(g *gorgonia.ExprGraph, seqLen, embedDim int) *PositionalEncoding {
	encoding := make([]float32, seqLen*embedDim)

	for pos := 0; pos < seqLen; pos++ {
		for i := 0; i < embedDim; i++ {
			angle := float64(pos) / math.Pow(10000, float64(2*i)/float64(embedDim))
			if i%2 == 0 {
				encoding[pos*embedDim+i] = float32(math.Sin(angle))
			} else {
				encoding[pos*embedDim+i] = float32(math.Cos(angle))
			}
		}
	}

	posEncTensor := tensor.New(
		tensor.WithBacking(encoding),
		tensor.WithShape(seqLen, embedDim),
	)

	posEncNode := gorgonia.NewConstant(posEncTensor, gorgonia.WithName("pos_encoding"))

	return &PositionalEncoding{
		graph:    g,
		encoding: posEncNode,
		seqLen:   seqLen,
		embedDim: embedDim,
	}
}

func (p *PositionalEncoding) Forward(seqLen int) (*gorgonia.Node, error) {
	return p.encoding, nil
}

type SelfAttention struct {
	graph    *gorgonia.ExprGraph
	wQuery   *gorgonia.Node
	wKey     *gorgonia.Node
	wValue   *gorgonia.Node
	embedDim int
	headDim  int
	dropout  float64
}

func NewSelfAttention(g *gorgonia.ExprGraph, embedDim, headDim int, dropout float64, seqLen int) *SelfAttention {
	wq := gorgonia.NewMatrix(g, tensor.Float32,
		gorgonia.WithShape(embedDim, headDim),
		gorgonia.WithName("w_query"),
		gorgonia.WithInit(gorgonia.GlorotN(1.0)),
	)

	wk := gorgonia.NewMatrix(g, tensor.Float32,
		gorgonia.WithShape(embedDim, headDim),
		gorgonia.WithName("w_key"),
		gorgonia.WithInit(gorgonia.GlorotN(1.0)),
	)

	wv := gorgonia.NewMatrix(g, tensor.Float32,
		gorgonia.WithShape(embedDim, headDim),
		gorgonia.WithName("w_value"),
		gorgonia.WithInit(gorgonia.GlorotN(1.0)),
	)

	return &SelfAttention{
		graph:    g,
		wQuery:   wq,
		wKey:     wk,
		wValue:   wv,
		embedDim: embedDim,
		headDim:  headDim,
		dropout:  dropout,
	}
}

func (sa *SelfAttention) Forward(x *gorgonia.Node, mask *gorgonia.Node, training bool) (*gorgonia.Node, error) {
	q, err := gorgonia.Mul(x, sa.wQuery)
	if err != nil {
		return nil, err
	}

	k, err := gorgonia.Mul(x, sa.wKey)
	if err != nil {
		return nil, err
	}

	v, err := gorgonia.Mul(x, sa.wValue)
	if err != nil {
		return nil, err
	}

	kT, err := gorgonia.Transpose(k)
	if err != nil {
		return nil, err
	}

	scores, err := gorgonia.Mul(q, kT)
	if err != nil {
		return nil, err
	}

	scale := float32(1.0 / math.Sqrt(float64(sa.headDim)))
	scaleNode := gorgonia.NewScalar(sa.graph, tensor.Float32, gorgonia.WithValue(scale))
	scores, err = gorgonia.HadamardProd(scores, scaleNode)
	if err != nil {
		return nil, err
	}

	// if mask != nil {
	// 	scores, err = gorgonia.Add(scores, mask)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// }

	attnWeights, err := gorgonia.SoftMax(scores)
	if err != nil {
		return nil, err
	}

	// if training && sa.dropout > 0 {
	// 	attnWeights, err = gorgonia.Dropout(attnWeights, sa.dropout)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// }

	output, err := gorgonia.Mul(attnWeights, v)
	if err != nil {
		return nil, err
	}

	return output, nil
}

type MultiHeadAttention struct {
	graph       *gorgonia.ExprGraph
	heads       []*SelfAttention
	wOutput     *gorgonia.Node
	numHeads    int
	embedDim    int
	headDim     int
}

func NewMultiHeadAttention(g *gorgonia.ExprGraph, embedDim, numHeads int, dropout float64, seqLen int) *MultiHeadAttention {
	headDim := embedDim / numHeads

	heads := make([]*SelfAttention, numHeads)
	for i := 0; i < numHeads; i++ {
		heads[i] = NewSelfAttention(g, embedDim, headDim, dropout, seqLen)
	}

	wOut := gorgonia.NewMatrix(g, tensor.Float32,
		gorgonia.WithShape(embedDim, embedDim),
		gorgonia.WithName("w_output"),
		gorgonia.WithInit(gorgonia.GlorotN(1.0)),
	)

	return &MultiHeadAttention{
		graph:    g,
		heads:    heads,
		wOutput:  wOut,
		numHeads: numHeads,
		embedDim: embedDim,
		headDim:  headDim,
	}
}

func (mha *MultiHeadAttention) Forward(x *gorgonia.Node, mask *gorgonia.Node, training bool) (*gorgonia.Node, error) {
	headOutputs := make([]*gorgonia.Node, mha.numHeads)
	for i := 0; i < mha.numHeads; i++ {
		out, err := mha.heads[i].Forward(x, mask, training)
		if err != nil {
			return nil, err
		}
		headOutputs[i] = out
	}

	result, err := gorgonia.Concat(1, headOutputs...)
	if err != nil {
		return nil, err
	}

	output, err := gorgonia.Mul(result, mha.wOutput)
	if err != nil {
		return nil, err
	}

	return output, nil
}

type FeedForward struct {
	graph      *gorgonia.ExprGraph
	w1         *gorgonia.Node
	w2         *gorgonia.Node
	dropout    float64
}

func NewFeedForward(g *gorgonia.ExprGraph, embedDim int, dropout float64) *FeedForward {
	hiddenDim := embedDim * 4

	w1 := gorgonia.NewMatrix(g, tensor.Float32,
		gorgonia.WithShape(embedDim, hiddenDim),
		gorgonia.WithName("ffn_w1"),
		gorgonia.WithInit(gorgonia.GlorotN(1.0)),
	)

	w2 := gorgonia.NewMatrix(g, tensor.Float32,
		gorgonia.WithShape(hiddenDim, embedDim),
		gorgonia.WithName("ffn_w2"),
		gorgonia.WithInit(gorgonia.GlorotN(1.0)),
	)

	return &FeedForward{
		graph:   g,
		w1:      w1,
		w2:      w2,
		dropout: dropout,
	}
}

func (ff *FeedForward) Forward(x *gorgonia.Node, training bool) (*gorgonia.Node, error) {
	hidden, err := gorgonia.Mul(x, ff.w1)
	if err != nil {
		return nil, err
	}

	// Simplified activation (should be GELU)
	hidden, err = gorgonia.Tanh(hidden)
	if err != nil {
		return nil, err
	}

	// if training && ff.dropout > 0 {
	// 	hidden, err = gorgonia.Dropout(hidden, ff.dropout)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// }

	output, err := gorgonia.Mul(hidden, ff.w2)
	if err != nil {
		return nil, err
	}

	return output, nil
}

type TransformerBlock struct {
	graph       *gorgonia.ExprGraph
	attention   *MultiHeadAttention
	feedForward *FeedForward
}

func NewTransformerBlock(g *gorgonia.ExprGraph, embedDim, numHeads int, dropout float64, seqLen int) *TransformerBlock {
	return &TransformerBlock{
		graph:       g,
		attention:   NewMultiHeadAttention(g, embedDim, numHeads, dropout, seqLen),
		feedForward: NewFeedForward(g, embedDim, dropout),
	}
}

func (tb *TransformerBlock) Forward(x *gorgonia.Node, mask *gorgonia.Node, training bool) (*gorgonia.Node, error) {
	attnOut, err := tb.attention.Forward(x, mask, training)
	if err != nil {
		return nil, err
	}

	x, err = gorgonia.Add(x, attnOut)
	if err != nil {
		return nil, err
	}

	ffOut, err := tb.feedForward.Forward(x, training)
	if err != nil {
		return nil, err
	}

	x, err = gorgonia.Add(x, ffOut)
	if err != nil {
		return nil, err
	}

	return x, nil
}

type GPT struct {
	graph          *gorgonia.ExprGraph
	config         *GorgoniteConfig
	embedding      *EmbeddingLayer
	posEncoding    *PositionalEncoding
	blocks         []*TransformerBlock
	outputLayer    *gorgonia.Node
	logits         *gorgonia.Node
	loss           *gorgonia.Node
learnables     []*gorgonia.Node
}

func NewGPT(g *gorgonia.ExprGraph, config *GorgoniteConfig) *GPT {
	embedding := NewEmbeddingLayer(g, config.VocabSize, config.EmbedDim, config.ContextLen)
	posEncoding := NewPositionalEncoding(g, config.ContextLen, config.EmbedDim)

	blocks := make([]*TransformerBlock, config.NumLayers)
	for i := 0; i < config.NumLayers; i++ {
		blocks[i] = NewTransformerBlock(g, config.EmbedDim, config.NumHeads, config.DropoutRate, config.ContextLen)
	}

	outputLayer := gorgonia.NewMatrix(g, tensor.Float32,
		gorgonia.WithShape(config.EmbedDim, config.VocabSize),
		gorgonia.WithName("output_projection"),
		gorgonia.WithInit(gorgonia.GlorotN(1.0)),
	)

	// Collect learnable nodes (weights)
	learnables := []*gorgonia.Node{embedding.embeddings, outputLayer}
	// Attention and FFN weights from blocks
	for _, block := range blocks {
		learnables = append(learnables, block.attention.wOutput)
		for _, head := range block.attention.heads {
			learnables = append(learnables, head.wQuery, head.wKey, head.wValue)
		}
		learnables = append(learnables, block.feedForward.w1, block.feedForward.w2)
	}

	gpt := &GPT{
		graph:       g,
		config:      config,
		embedding:   embedding,
		posEncoding: posEncoding,
		blocks:      blocks,
		outputLayer: outputLayer,
		learnables:  learnables,
	}

	// Build the graph
	seqLen := config.ContextLen
	x, err := embedding.Forward(seqLen)
	if err != nil {
		panic(err)
	}

	posEnc, err := posEncoding.Forward(seqLen)
	if err != nil {
		panic(err)
	}

	x, err = gorgonia.Add(x, posEnc)
	if err != nil {
		panic(err)
	}

	mask := createCausalMask(g, seqLen)

	for _, block := range blocks {
		x, err = block.Forward(x, mask, false)
		if err != nil {
			panic(err)
		}
	}

	logits, err := gorgonia.Mul(x, outputLayer)
	if err != nil {
		panic(err)
	}

	gpt.logits = logits

	loss, err := gorgonia.Mean(logits)
	if err != nil {
		panic(err)
	}
	gpt.loss = loss

	fmt.Printf("Graph has %d nodes\n", len(g.AllNodes()))

	return gpt
}

func (gpt *GPT) Forward(training bool) (*gorgonia.Node, error) {
	return gpt.logits, nil
}

func createCausalMask(_ *gorgonia.ExprGraph, seqLen int) *gorgonia.Node {
	maskData := make([]float32, seqLen*seqLen)
	for i := 0; i < seqLen; i++ {
		for j := i + 1; j < seqLen; j++ {
			maskData[i*seqLen+j] = -1e10
		}
	}

	maskTensor := tensor.New(
		tensor.WithBacking(maskData),
		tensor.WithShape(seqLen, seqLen),
	)

	return gorgonia.NewConstant(maskTensor)
}

type Dataset struct {
	inputs  [][]int
	targets [][]int
}

func PrepareDataset(text string, tokenizer Tokenizer, contextLen, stride int) *Dataset {
	tokens := tokenizer.Encode(text)

	dataset := &Dataset{
		inputs:  make([][]int, 0),
		targets: make([][]int, 0),
	}

	for i := 0; i+contextLen < len(tokens); i += stride {
		input := tokens[i : i+contextLen]
		target := tokens[i+1 : i+contextLen+1]
		dataset.inputs = append(dataset.inputs, input)
		dataset.targets = append(dataset.targets, target)
	}

	return dataset
}

type TrainConfig struct {
	LearningRate   float64
	WeightDecay    float64
	NumEpochs      int
	BatchSize      int
	WarmupSteps    int
	MaxLR          float64
	MinLR          float64
	GradientClip   float64
	EvalFreq       int
	SaveFreq       int
}

func GetPretrainingConfig() TrainConfig {
	return TrainConfig{
		LearningRate: 3e-4,
		WeightDecay:  0.1,
		NumEpochs:    300,
		BatchSize:    32,
		WarmupSteps:  1500,
		MaxLR:        3e-4,
		MinLR:        3e-5,
		GradientClip: 1.0,
		EvalFreq:     200,
		SaveFreq:     1000,
	}
}

func TrainModel(model *GPT, dataset *Dataset, config TrainConfig) error {
	vm := gorgonia.NewTapeMachine(model.graph, gorgonia.BindDualValues())
	defer vm.Close()

	solver := gorgonia.NewVanillaSolver(gorgonia.WithLearnRate(config.LearningRate))

	for epoch := 0; epoch < config.NumEpochs; epoch++ {
		epochLoss := 0.0

		for i := 0; i < len(dataset.inputs); i += config.BatchSize {
			_, err := model.Forward(true)
			if err != nil {
				return fmt.Errorf("forward pass error: %w", err)
			}

			loss := model.loss

			if err := vm.RunAll(); err != nil {
				return fmt.Errorf("VM execution error: %w", err)
			}

			fmt.Printf("Debug: Loss node ID: %d, Shape: %v\n", loss.ID(), loss.Shape())
			fmt.Printf("Debug: Computing gradients for %d learnables\n", len(model.learnables))

			grads, err := gorgonia.Grad(loss, model.learnables...)
			if err != nil {
				fmt.Printf("Debug: gorgonia.Grad failed: %v\n", err)
				return fmt.Errorf("backward pass error: %w", err)
			}

			fmt.Printf("Debug: Gradients computed successfully for %d nodes\n", len(grads))
			for i, g := range grads {
				if g != nil {
					fmt.Printf("Debug: Grad %d ID: %d, Shape: %v\n", i, g.ID(), g.Shape())
				} else {
					fmt.Printf("Debug: Grad %d is nil\n", i)
				}
			}

			// Run backward pass to compute gradient values
			if err := vm.RunAll(); err != nil {
				fmt.Printf("Debug: Backward VM run failed: %v\n", err)
				return fmt.Errorf("backward VM run error: %w", err)
			}

			valueGrads := gorgonia.NodesToValueGrads(grads)
			fmt.Printf("Debug: Converted to %d ValueGrads\n", len(valueGrads))
			for i, vg := range valueGrads {
				gradVal, err := vg.Grad()
				if err != nil {
					fmt.Printf("Debug: ValueGrad %d grad error: %v\n", i, err)
				} else if gradVal != nil {
					fmt.Printf("Debug: ValueGrad %d has grad value, shape: %v\n", i, gradVal.Shape())
				} else {
					fmt.Printf("Debug: ValueGrad %d has nil grad value\n", i)
				}
			}

			if err := solver.Step(valueGrads); err != nil {
				fmt.Printf("Debug: Solver step failed: %v\n", err)
				// Print more details about the graph
				fmt.Printf("Debug: Graph nodes: %d\n", len(model.graph.AllNodes()))
				for _, node := range model.graph.AllNodes() {
					if node.Value() != nil {
						fmt.Printf("Debug: Node %d (%s): shape %v, has value\n", node.ID(), node.Name(), node.Shape())
					} else {
						fmt.Printf("Debug: Node %d (%s): shape %v, no value\n", node.ID(), node.Name(), node.Shape())
					}
				}
				return fmt.Errorf("solver step error: %w", err)
			}

			vm.Reset()

			lossValue := loss.Value().Data().(float32)
			epochLoss += float64(lossValue)
		}

		avgLoss := epochLoss / float64(len(dataset.inputs)/config.BatchSize)
		fmt.Printf("Epoch %d: Loss = %.4f\n", epoch, avgLoss)

		if epoch%config.SaveFreq == 0 {
			SaveModel(model, fmt.Sprintf("checkpoint_epoch_%d.bin", epoch))
		}
	}

	return nil
}


func SaveModel(model *GPT, filepath string) error {
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)

	params := make(map[string][]float32)

	for _, node := range model.graph.AllNodes() {
		if node.Value() != nil {
			data := node.Value().Data()
			switch v := data.(type) {
			case []float32:
				params[node.Name()] = v
			case []float64:
				f32 := make([]float32, len(v))
				for i, val := range v {
					f32[i] = float32(val)
				}
				params[node.Name()] = f32
			}
		}
	}

	return encoder.Encode(params)
}

func LoadModel(model *GPT, filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)
	params := make(map[string][]float32)

	if err := decoder.Decode(&params); err != nil {
		return err
	}

	for _, node := range model.graph.AllNodes() {
		if data, exists := params[node.Name()]; exists {
			t := tensor.New(
				tensor.WithBacking(data),
				tensor.WithShape(node.Shape()...),
			)
			gorgonia.Let(node, t)
		}
	}

	return nil
}

func Generate(model *GPT, startTokens []int, maxNewTokens int, temperature float32, topK int) []int {
	tokens := make([]int, len(startTokens))
	copy(tokens, startTokens)

	vm := gorgonia.NewTapeMachine(model.graph)
	defer vm.Close()

	for i := 0; i < maxNewTokens; i++ {
		context := tokens
		if len(context) > model.config.ContextLen {
			context = context[len(context)-model.config.ContextLen:]
		}

		logits, err := model.Forward(false)
		if err != nil {
			log.Fatal(err)
		}

		if err := vm.RunAll(); err != nil {
			log.Fatal(err)
		}

		logitsData := logits.Value().Data().([]float32)
		lastLogits := logitsData[len(context)*model.config.VocabSize:]

		nextToken := SampleToken(lastLogits[:model.config.VocabSize], temperature, topK)
		tokens = append(tokens, nextToken)

		vm.Reset()
	}

	return tokens
}

func SampleToken(logits []float32, temperature float32, topK int) int {
	if temperature != 1.0 {
		for i := range logits {
			logits[i] /= temperature
		}
	}

	maxLogit := logits[0]
	for _, l := range logits {
		if l > maxLogit {
			maxLogit = l
		}
	}

	expSum := float32(0)
	probs := make([]float32, len(logits))
	for i, l := range logits {
		probs[i] = float32(math.Exp(float64(l - maxLogit)))
		expSum += probs[i]
	}

	for i := range probs {
		probs[i] /= expSum
	}

	if topK > 0 && topK < len(probs) {
		type indexedProb struct {
			idx  int
			prob float32
		}

		indexed := make([]indexedProb, len(probs))
		for i, p := range probs {
			indexed[i] = indexedProb{i, p}
		}

		sort.Slice(indexed, func(i, j int) bool {
			return indexed[i].prob > indexed[j].prob
		})

		for i := topK; i < len(indexed); i++ {
			probs[indexed[i].idx] = 0
		}

		sum := float32(0)
		for _, p := range probs {
			sum += p
		}
		for i := range probs {
			probs[i] /= sum
		}
	}

	r := rand.Float32()
	cumulative := float32(0)
	for i, p := range probs {
		cumulative += p
		if r < cumulative {
			return i
		}
	}

	return len(probs) - 1
}

type Tokenizer interface {
	Encode(text string) []int
	Decode(tokens []int) string
}

type BPETokenizer struct {
	vocab     map[string]int
	invVocab  map[int]string
	vocabSize int
}

func NewBPETokenizer(vocabPath string) *BPETokenizer {
	vocab := loadVocab(vocabPath)

	invVocab := make(map[int]string)
	for token, idx := range vocab {
		invVocab[idx] = token
	}

	return &BPETokenizer{
		vocab:     vocab,
		invVocab:  invVocab,
		vocabSize: len(vocab),
	}
}

func (t *BPETokenizer) Encode(text string) []int {
	tokens := make([]int, 0)

	words := strings.Fields(text)
	for _, word := range words {
		if idx, exists := t.vocab[word]; exists {
			tokens = append(tokens, idx)
		} else {
			tokens = append(tokens, t.vocab["<UNK>"])
		}
	}

	return tokens
}

func (t *BPETokenizer) Decode(tokens []int) string {
	words := make([]string, len(tokens))
	for i, tok := range tokens {
		if word, exists := t.invVocab[tok]; exists {
			words[i] = word
		} else {
			words[i] = "<UNK>"
		}
	}
	return strings.Join(words, " ")
}

func loadVocab(path string) map[string]int {
	vocab := make(map[string]int)
	if path == "" {
		return vocab
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return vocab
	}
	_ = json.Unmarshal(data, &vocab)
	return vocab
}

// TiktokenTokenizer wraps tiktoken for use as a Tokenizer.
type TiktokenTokenizer struct {
	enc *tiktoken.Tiktoken
}

// NewTiktokenTokenizer creates a tokenizer using the given tiktoken encoding (default: cl100k_base).
func NewTiktokenTokenizer(encoding string) (*TiktokenTokenizer, error) {
	if encoding == "" {
		encoding = "cl100k_base"
	}
	enc, err := tiktoken.GetEncoding(encoding)
	if err != nil {
		return nil, err
	}
	return &TiktokenTokenizer{enc: enc}, nil
}

func (t *TiktokenTokenizer) Encode(text string) []int {
	return t.enc.Encode(text, nil, nil)
}

func (t *TiktokenTokenizer) Decode(tokens []int) string {
	return t.enc.Decode(tokens)
}
// RunGorgoniteProtocol builds and runs a quick validation of the Gorgonite graph.
func RunGorgoniteProtocol(cfg *GorgoniteConfig) error {
	if cfg == nil {
		cfg = DefaultGorgoniteConfig()
	}
	g := gorgonia.NewGraph()
	model := NewGPT(g, cfg)

	logits, err := model.Forward(false)
	if err != nil {
		return fmt.Errorf("forward pass failed: %w", err)
	}

	vm := gorgonia.NewTapeMachine(model.graph, gorgonia.BindDualValues())
	defer vm.Close()
	if err := vm.RunAll(); err != nil {
		return fmt.Errorf("VM execution failed: %w", err)
	}

	log.Printf("RunGorgoniteProtocol OK — logits shape: %v", logits.Shape())
	return nil
}

// CreateDummyDataset generates a random dataset for testing.
func CreateDummyDataset(numSamples, vocabSize, contextLen int) *Dataset {
	dataset := &Dataset{
		inputs:  make([][]int, numSamples),
		targets: make([][]int, numSamples),
	}

	for i := 0; i < numSamples; i++ {
		input := make([]int, contextLen)
		target := make([]int, contextLen)
		for j := 0; j < contextLen; j++ {
			input[j] = rand.Intn(vocabSize)
			if j < contextLen-1 {
				target[j] = input[j+1]
			} else {
				target[j] = rand.Intn(vocabSize)
			}
		}
		dataset.inputs[i] = input
		dataset.targets[i] = target
	}

	return dataset
}

// RunPretraining runs one epoch of pretraining on a dummy dataset for smoke-testing.
func RunPretraining(cfg *GorgoniteConfig) error {
	if cfg == nil {
		cfg = DefaultGorgoniteConfig()
		cfg.VocabSize = 50257
		cfg.EmbedDim = 256
		cfg.NumHeads = 8
		cfg.NumLayers = 4
		cfg.ContextLen = 32
		cfg.FFNHiddenDim = 256
	}
	g := gorgonia.NewGraph()
	model := NewGPT(g, cfg)
	dataset := CreateDummyDataset(10, cfg.VocabSize, cfg.ContextLen)
	trainCfg := TrainConfig{
		LearningRate: 3e-4,
		NumEpochs:    1,
		BatchSize:    1,
		GradientClip: 1.0,
		SaveFreq:     5,
	}
	return TrainModel(model, dataset, trainCfg)
}

