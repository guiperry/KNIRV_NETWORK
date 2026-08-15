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
	"knirvhasher/pkg/hashing/core"
)

// ---- Real backprop-trained LM (Phase 4) ----
//
// This replaces a previous Gorgonia scaffold that looked complete (real
// matmuls, real softmax, real gradient calls) but was not: EmbeddingLayer
// ignored token IDs entirely and returned a fixed (seqLen x embedDim) matrix
// regardless of input; GPT.loss was mean(logits) with no dependency on any
// target token; TrainModel's inner loop never read dataset.inputs/targets
// into the forward pass. The graph was, in effect, the same for every input
// and never learned anything — the same "real ops, disconnected from real
// data" failure pattern found elsewhere in this package before this session's
// fixes. See docs/hasher_validation_patch.md Phase 4.
//
// Design: GPT owns its learnable weights as plain *tensor.Dense values (not
// permanently-bound gorgonia nodes). Each Forward/TrainStep call builds a
// fresh gorgonia graph sized to the actual input, binds fresh nodes to the
// current weight values via gorgonia.WithValue, runs it, and — for
// TrainStep — writes the post-gradient-step values back into the persisted
// tensors. This mirrors the "rebuild the graph, carry the values" pattern
// DynamicGraph already used elsewhere in this package, applied directly so
// the full attention/FFN math (which needs ops DynamicGraph doesn't wrap —
// SoftMax, Transpose, HadamardProd) can be expressed with plain Gorgonia
// calls instead of extending that wrapper's limited op set.

// GorgoniteConfig defines architecture for the real, backprop-trained LM.
type GorgoniteConfig struct {
	VocabSize    int
	EmbedDim     int
	NumHeads     int
	NumLayers    int
	ContextLen   int
	DropoutRate  float64 // reserved; dropout is not applied in this pass
	FFNHiddenDim int
	// DecayAlpha is the FoX temporal-decay rate applied per position step
	// (see foxAttention below), in (0, 1]. 1.0 disables decay entirely.
	DecayAlpha float32
}

func DefaultGorgoniteConfig() *GorgoniteConfig {
	return &GorgoniteConfig{
		VocabSize:    100277,
		EmbedDim:     768,
		NumHeads:     12,
		NumLayers:    12,
		ContextLen:   2048,
		DropoutRate:  0.1,
		FFNHiddenDim: 3072,
		DecayAlpha:   0.95,
	}
}

// layerParams holds one transformer block's learnable weights as plain
// tensors. No biases, matching the prior scaffold's shape (a bias-free
// transformer still trains; adding biases is a reasonable future increment,
// not required for the forward/backward path to be real and correct).
type layerParams struct {
	wQuery  []*tensor.Dense // one per head, each [EmbedDim, HeadDim]
	wKey    []*tensor.Dense
	wValue  []*tensor.Dense
	wOutput *tensor.Dense // [EmbedDim, EmbedDim]
	w1      *tensor.Dense // [EmbedDim, FFNHiddenDim]
	w2      *tensor.Dense // [FFNHiddenDim, EmbedDim]
}

// GPT is a small, real, backprop-trained transformer language model.
type GPT struct {
	config     *GorgoniteConfig
	embeddings *tensor.Dense // [VocabSize, EmbedDim]
	outputProj *tensor.Dense // [EmbedDim, VocabSize]
	layers     []*layerParams

	trained          bool
	trainingProgress float64
}

// NewGPT creates a GPT with freshly (Glorot-style) initialized weights.
func NewGPT(config *GorgoniteConfig) *GPT {
	if config == nil {
		config = DefaultGorgoniteConfig()
	}
	headDim := config.EmbedDim / config.NumHeads

	gpt := &GPT{
		config:     config,
		embeddings: randDense(config.VocabSize, config.EmbedDim),
		outputProj: randDense(config.EmbedDim, config.VocabSize),
		layers:     make([]*layerParams, config.NumLayers),
	}
	for l := 0; l < config.NumLayers; l++ {
		lp := &layerParams{
			wQuery:  make([]*tensor.Dense, config.NumHeads),
			wKey:    make([]*tensor.Dense, config.NumHeads),
			wValue:  make([]*tensor.Dense, config.NumHeads),
			wOutput: randDense(config.EmbedDim, config.EmbedDim),
			w1:      randDense(config.EmbedDim, config.FFNHiddenDim),
			w2:      randDense(config.FFNHiddenDim, config.EmbedDim),
		}
		for h := 0; h < config.NumHeads; h++ {
			lp.wQuery[h] = randDense(config.EmbedDim, headDim)
			lp.wKey[h] = randDense(config.EmbedDim, headDim)
			lp.wValue[h] = randDense(config.EmbedDim, headDim)
		}
		gpt.layers[l] = lp
	}
	return gpt
}

// randDense creates a [rows, cols] tensor with small Gaussian-ish random
// init (Box-Muller), scaled like Glorot/Xavier (1/sqrt(fanIn)) to keep
// activations from exploding or vanishing at the start of training.
func randDense(rows, cols int) *tensor.Dense {
	data := make([]float32, rows*cols)
	scale := float32(1.0 / math.Sqrt(float64(rows)))
	for i := range data {
		u1, u2 := rand.Float64(), rand.Float64()
		if u1 < 1e-12 {
			u1 = 1e-12
		}
		z := math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)
		data[i] = float32(z) * scale
	}
	return tensor.New(tensor.WithBacking(data), tensor.WithShape(rows, cols))
}

// namedParams returns every learnable tensor with a stable name, in a fixed
// order — used to build matching gorgonia nodes each forward call and to
// serialize/deserialize the model.
func (gpt *GPT) namedParams() ([]string, []*tensor.Dense) {
	names := []string{"embeddings", "output_projection"}
	values := []*tensor.Dense{gpt.embeddings, gpt.outputProj}
	for l, lp := range gpt.layers {
		for h := range lp.wQuery {
			names = append(names, fmt.Sprintf("layer%d.head%d.wq", l, h))
			values = append(values, lp.wQuery[h])
			names = append(names, fmt.Sprintf("layer%d.head%d.wk", l, h))
			values = append(values, lp.wKey[h])
			names = append(names, fmt.Sprintf("layer%d.head%d.wv", l, h))
			values = append(values, lp.wValue[h])
		}
		names = append(names, fmt.Sprintf("layer%d.wOutput", l))
		values = append(values, lp.wOutput)
		names = append(names, fmt.Sprintf("layer%d.w1", l))
		values = append(values, lp.w1)
		names = append(names, fmt.Sprintf("layer%d.w2", l))
		values = append(values, lp.w2)
	}
	return names, values
}

// graphBuild is what buildGraph returns: the fresh graph, the parameter
// nodes bound to it (parallel to namedParams' order, so gradients can be
// mapped back to the persisted tensors), the logits node, and — only when
// targetIDs was non-nil — the scalar loss node.
type graphBuild struct {
	graph      *gorgonia.ExprGraph
	paramNodes []*gorgonia.Node
	logits     *gorgonia.Node
	loss       *gorgonia.Node
}

// buildGraph constructs a fresh computational graph for exactly the given
// token sequence. Unlike the previous scaffold, embedding lookup is a real
// one-hot(tokenIDs) x embeddings matmul — the graph's output genuinely
// depends on which tokens were passed in — and, when targetIDs is provided,
// the loss is real cross-entropy against those targets, not mean(logits).
func (gpt *GPT) buildGraph(tokenIDs []int, targetIDs []int) (*graphBuild, error) {
	if len(tokenIDs) == 0 {
		return nil, fmt.Errorf("buildGraph: empty token sequence")
	}
	cfg := gpt.config
	seqLen := len(tokenIDs)
	g := gorgonia.NewGraph()

	names, values := gpt.namedParams()
	paramNodes := make([]*gorgonia.Node, len(values))
	byName := make(map[string]*gorgonia.Node, len(values))
	for i, v := range values {
		n := gorgonia.NewMatrix(g, tensor.Float32,
			gorgonia.WithShape(v.Shape()...),
			gorgonia.WithName(names[i]),
			gorgonia.WithValue(v),
		)
		paramNodes[i] = n
		byName[names[i]] = n
	}

	// One-hot(tokenIDs) x embeddings -> [seqLen, EmbedDim]. This is the
	// standard trick for a differentiable embedding lookup without a
	// gather op: the gradient w.r.t. embeddings flows back correctly
	// through the matmul (only the rows for tokens that actually appeared
	// get a nonzero gradient).
	oneHot := make([]float32, seqLen*cfg.VocabSize)
	for i, tok := range tokenIDs {
		idx := tok % cfg.VocabSize
		if idx < 0 {
			idx += cfg.VocabSize
		}
		oneHot[i*cfg.VocabSize+idx] = 1
	}
	oneHotTensor := tensor.New(tensor.WithBacking(oneHot), tensor.WithShape(seqLen, cfg.VocabSize))
	oneHotNode := gorgonia.NewConstant(oneHotTensor, gorgonia.WithName("token_one_hot"))

	x, err := gorgonia.Mul(oneHotNode, byName["embeddings"])
	if err != nil {
		return nil, fmt.Errorf("embedding lookup: %w", err)
	}

	posEnc := sinusoidalPositionalEncoding(g, seqLen, cfg.EmbedDim)
	x, err = gorgonia.Add(x, posEnc)
	if err != nil {
		return nil, fmt.Errorf("add positional encoding: %w", err)
	}

	decay := foxDecayMatrix(g, seqLen, cfg.DecayAlpha)
	headDim := cfg.EmbedDim / cfg.NumHeads

	for l, lp := range gpt.layers {
		attnOut, err := foxMultiHeadAttention(g, x, decay, lp, l, byName, cfg.NumHeads, headDim)
		if err != nil {
			return nil, fmt.Errorf("layer %d attention: %w", l, err)
		}
		x, err = gorgonia.Add(x, attnOut)
		if err != nil {
			return nil, fmt.Errorf("layer %d attention residual: %w", l, err)
		}

		ffOut, err := feedForward(x, byName[fmt.Sprintf("layer%d.w1", l)], byName[fmt.Sprintf("layer%d.w2", l)])
		if err != nil {
			return nil, fmt.Errorf("layer %d feedforward: %w", l, err)
		}
		x, err = gorgonia.Add(x, ffOut)
		if err != nil {
			return nil, fmt.Errorf("layer %d feedforward residual: %w", l, err)
		}
	}

	logits, err := gorgonia.Mul(x, byName["output_projection"])
	if err != nil {
		return nil, fmt.Errorf("output projection: %w", err)
	}

	build := &graphBuild{graph: g, paramNodes: paramNodes, logits: logits}

	if targetIDs != nil {
		if len(targetIDs) != seqLen {
			return nil, fmt.Errorf("buildGraph: %d targets for %d tokens", len(targetIDs), seqLen)
		}
		loss, err := crossEntropyLoss(g, logits, targetIDs, cfg.VocabSize)
		if err != nil {
			return nil, fmt.Errorf("cross entropy loss: %w", err)
		}
		build.loss = loss
	}

	return build, nil
}

// foxMultiHeadAttention computes one multi-head attention block using the
// FoX-style ingredients — sharp (softmax) content matching combined
// multiplicatively with a temporal decay matrix — then concatenates heads
// and projects back through wOutput.
//
// This adapts the original hash-based FoX formula
// (sum_j exp(q.k_j) * decay(j,i) * v_j, unnormalized) for a differentiable,
// numerically-stable port: raw unnormalized exp(scores) can overflow before
// the decay multiply ever gets a chance to shrink it back down, so this
// computes softmax(scores) (Gorgonia's SoftMax is already numerically
// stable — subtracts the row max internally) and multiplies that by the
// decay matrix afterward, rather than exponentiating raw scores directly.
// The result no longer sums to exactly 1 per row (decay attenuates it
// further), which is the intended behavior: distant positions are
// down-weighted both by content match *and* by explicit recency decay.
func foxMultiHeadAttention(g *gorgonia.ExprGraph, x, decay *gorgonia.Node, lp *layerParams, layerIdx int, byName map[string]*gorgonia.Node, numHeads, headDim int) (*gorgonia.Node, error) {
	seqLen := x.Shape()[0]
	heads := make([]*gorgonia.Node, numHeads)

	causalMask := causalMaskAdd(g, seqLen)

	for h := 0; h < numHeads; h++ {
		wq := byName[fmt.Sprintf("layer%d.head%d.wq", layerIdx, h)]
		wk := byName[fmt.Sprintf("layer%d.head%d.wk", layerIdx, h)]
		wv := byName[fmt.Sprintf("layer%d.head%d.wv", layerIdx, h)]

		q, err := gorgonia.Mul(x, wq)
		if err != nil {
			return nil, err
		}
		k, err := gorgonia.Mul(x, wk)
		if err != nil {
			return nil, err
		}
		v, err := gorgonia.Mul(x, wv)
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

		scale := float32(1.0 / math.Sqrt(float64(headDim)))
		scaleNode := gorgonia.NewScalar(g, tensor.Float32, gorgonia.WithValue(scale))
		scores, err = gorgonia.HadamardProd(scores, scaleNode)
		if err != nil {
			return nil, err
		}

		scores, err = gorgonia.Add(scores, causalMask)
		if err != nil {
			return nil, err
		}

		weights, err := gorgonia.SoftMax(scores)
		if err != nil {
			return nil, err
		}
		weights, err = gorgonia.HadamardProd(weights, decay)
		if err != nil {
			return nil, err
		}

		headOut, err := gorgonia.Mul(weights, v)
		if err != nil {
			return nil, err
		}
		heads[h] = headOut
	}

	concat, err := gorgonia.Concat(1, heads...)
	if err != nil {
		return nil, err
	}
	return gorgonia.Mul(concat, byName[fmt.Sprintf("layer%d.wOutput", layerIdx)])
}

func feedForward(x, w1, w2 *gorgonia.Node) (*gorgonia.Node, error) {
	hidden, err := gorgonia.Mul(x, w1)
	if err != nil {
		return nil, err
	}
	hidden, err = gorgonia.Tanh(hidden) // simplified activation (GELU would be closer to a real GPT FFN)
	if err != nil {
		return nil, err
	}
	return gorgonia.Mul(hidden, w2)
}

// causalMaskAdd returns a [seqLen, seqLen] constant with 0 on/below the
// diagonal and a large negative value above it, added to scores before
// softmax so future positions get ~0 probability.
func causalMaskAdd(g *gorgonia.ExprGraph, seqLen int) *gorgonia.Node {
	data := make([]float32, seqLen*seqLen)
	for i := 0; i < seqLen; i++ {
		for j := i + 1; j < seqLen; j++ {
			data[i*seqLen+j] = -1e10
		}
	}
	t := tensor.New(tensor.WithBacking(data), tensor.WithShape(seqLen, seqLen))
	return gorgonia.NewConstant(t, gorgonia.WithName("causal_mask"))
}

// foxDecayMatrix returns a [seqLen, seqLen] constant where entry (i, j) is
// alpha^(i-j) for j <= i (temporal forgetting of older positions) and 0 for
// j > i (redundant with the causal mask, but keeps this matrix correct on
// its own if ever reused without one).
func foxDecayMatrix(g *gorgonia.ExprGraph, seqLen int, alpha float32) *gorgonia.Node {
	if alpha <= 0 || alpha > 1 {
		alpha = 1
	}
	data := make([]float32, seqLen*seqLen)
	for i := 0; i < seqLen; i++ {
		v := float32(1.0)
		for j := i; j >= 0; j-- {
			data[i*seqLen+j] = v
			v *= alpha
		}
	}
	t := tensor.New(tensor.WithBacking(data), tensor.WithShape(seqLen, seqLen))
	return gorgonia.NewConstant(t, gorgonia.WithName("fox_decay"))
}

// sinusoidalPositionalEncoding returns a fixed (non-learnable) [seqLen,
// embedDim] constant, standard Transformer sin/cos positional encoding.
func sinusoidalPositionalEncoding(g *gorgonia.ExprGraph, seqLen, embedDim int) *gorgonia.Node {
	encoding := make([]float32, seqLen*embedDim)
	for pos := 0; pos < seqLen; pos++ {
		for i := 0; i < embedDim; i++ {
			angle := float64(pos) / math.Pow(10000, float64(2*(i/2))/float64(embedDim))
			if i%2 == 0 {
				encoding[pos*embedDim+i] = float32(math.Sin(angle))
			} else {
				encoding[pos*embedDim+i] = float32(math.Cos(angle))
			}
		}
	}
	t := tensor.New(tensor.WithBacking(encoding), tensor.WithShape(seqLen, embedDim))
	return gorgonia.NewConstant(t, gorgonia.WithName("pos_encoding"))
}

// crossEntropyLoss computes mean over positions of -log(softmax(logits)[i,
// target[i]]), the standard next-token-prediction loss. Built the same way
// embedding lookup is (one-hot times the value, so the constant one-hot
// target vector selects out the right column), rather than requiring a
// gather op.
func crossEntropyLoss(g *gorgonia.ExprGraph, logits *gorgonia.Node, targetIDs []int, vocabSize int) (*gorgonia.Node, error) {
	seqLen := len(targetIDs)
	// LogSoftMax (not SoftMax followed by a separate Log) is required here,
	// not just a style preference: computing log(softmax(x)) as two steps
	// evaluates log(0) = -Inf whenever a probability underflows to exactly
	// zero, and HadamardProd against the one-hot target vector then hits
	// 0 * -Inf = NaN at every OTHER (non-target) vocab position — which
	// contaminates the sum even though those positions "shouldn't" matter.
	// LogSoftMax computes x - max(x) - logsumexp(x) directly and never
	// produces -Inf for finite input.
	logProbs, err := gorgonia.LogSoftMax(logits)
	if err != nil {
		return nil, err
	}

	oneHot := make([]float32, seqLen*vocabSize)
	for i, tgt := range targetIDs {
		idx := tgt % vocabSize
		if idx < 0 {
			idx += vocabSize
		}
		oneHot[i*vocabSize+idx] = 1
	}
	oneHotTensor := tensor.New(tensor.WithBacking(oneHot), tensor.WithShape(seqLen, vocabSize))
	oneHotNode := gorgonia.NewConstant(oneHotTensor, gorgonia.WithName("target_one_hot"))

	picked, err := gorgonia.HadamardProd(logProbs, oneHotNode)
	if err != nil {
		return nil, err
	}
	summed, err := gorgonia.Sum(picked) // sum over the one nonzero entry per row = sum of picked log-probs
	if err != nil {
		return nil, err
	}
	negSummed, err := gorgonia.Neg(summed)
	if err != nil {
		return nil, err
	}
	n := gorgonia.NewScalar(g, tensor.Float32, gorgonia.WithValue(float32(seqLen)))
	return gorgonia.HadamardDiv(negSummed, n)
}

// Forward runs inference for tokenIDs and returns flattened [seqLen x
// VocabSize] logits (row i = position i's distribution over the vocabulary).
func (gpt *GPT) Forward(tokenIDs []int) ([]float32, error) {
	build, err := gpt.buildGraph(tokenIDs, nil)
	if err != nil {
		return nil, err
	}
	vm := gorgonia.NewTapeMachine(build.graph)
	defer vm.Close()
	if err := vm.RunAll(); err != nil {
		return nil, fmt.Errorf("forward VM run: %w", err)
	}
	data, ok := build.logits.Value().Data().([]float32)
	if !ok {
		return nil, fmt.Errorf("unexpected logits value type %T", build.logits.Value().Data())
	}
	return data, nil
}

// TrainStep runs one forward+backward+SGD-update step on a single (context,
// target) example and returns the scalar loss. Parameter values are updated
// in place on gpt's persisted tensors, so the next call — whether Forward
// or another TrainStep — sees the updated weights.
func (gpt *GPT) TrainStep(tokenIDs, targetIDs []int, learningRate float32) (float32, error) {
	build, err := gpt.buildGraph(tokenIDs, targetIDs)
	if err != nil {
		return 0, err
	}
	if build.loss == nil {
		return 0, fmt.Errorf("TrainStep: no target IDs given")
	}

	// gorgonia.Grad performs symbolic differentiation: it inserts gradient
	// computation nodes into build.graph and must be called BEFORE the
	// TapeMachine is constructed, so those nodes are part of what the
	// machine compiles and runs. Calling it after (as the previous scaffold
	// this replaced did) leaves the added nodes outside the machine's
	// already-fixed instruction tape — their gradients silently never get
	// computed, which surfaces as a nil-Value panic in the solver step.
	if _, err := gorgonia.Grad(build.loss, build.paramNodes...); err != nil {
		return 0, fmt.Errorf("gradient computation: %w", err)
	}

	vm := gorgonia.NewTapeMachine(build.graph, gorgonia.BindDualValues())
	defer vm.Close()
	if err := vm.RunAll(); err != nil {
		return 0, fmt.Errorf("forward+backward VM run: %w", err)
	}

	// Gradient clipping (norm 1.0, matching GetPretrainingConfig's default)
	// is not optional here: an unclipped step on a freshly-initialized
	// small model can produce a large-enough weight update that the next
	// forward pass's logits overflow into a non-finite loss within a
	// handful of steps.
	solver := gorgonia.NewVanillaSolver(gorgonia.WithLearnRate(float64(learningRate)), gorgonia.WithClip(1.0))
	if err := solver.Step(gorgonia.NodesToValueGrads(build.paramNodes)); err != nil {
		return 0, fmt.Errorf("solver step: %w", err)
	}

	// Copy each parameter node's post-step value back into gpt's persisted
	// tensors, so the update actually survives past this graph's lifetime.
	_, values := gpt.namedParams()
	for i, node := range build.paramNodes {
		updated, ok := node.Value().Data().([]float32)
		if !ok {
			return 0, fmt.Errorf("param %d: unexpected value type after step", i)
		}
		copy(values[i].Data().([]float32), updated)
	}

	lossVal, ok := build.loss.Value().Data().(float32)
	if !ok {
		return 0, fmt.Errorf("unexpected loss value type")
	}
	return lossVal, nil
}

// SetTrained marks the GPT model as trained (or not), enabling the
// GeneratorSwitcher to route generation requests to the internal model.
func (gpt *GPT) SetTrained(v bool) {
	gpt.trained = v
}

// SetProgress updates the training progress fraction [0.0, 1.0].
func (gpt *GPT) SetProgress(p float64) {
	if p < 0 {
		p = 0
	}
	if p > 1 {
		p = 1
	}
	gpt.trainingProgress = p
	if p >= 1 {
		gpt.trained = true
	}
}

// IsReady implements TrainingStateProvider.
func (gpt *GPT) IsReady() bool {
	return gpt.trained
}

// Progress implements TrainingStateProvider.
func (gpt *GPT) Progress() float64 {
	return gpt.trainingProgress
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
	LearningRate float64
	WeightDecay  float64
	NumEpochs    int
	BatchSize    int
	WarmupSteps  int
	MaxLR        float64
	MinLR        float64
	GradientClip float64
	EvalFreq     int
	SaveFreq     int
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

// TrainModel runs real gradient-descent training: each example's real input
// and target token IDs are fed through GPT.TrainStep, which is where the
// actual forward+backward+update happens. BatchSize is honored as a
// logging/reporting granularity (average loss per BatchSize examples) — each
// example still gets its own SGD step; true mini-batched gradient averaging
// across variable-length sequences is a reasonable future increment but not
// required for the loop to be real and for loss to actually decrease.
func TrainModel(model *GPT, dataset *Dataset, config TrainConfig) error {
	if len(dataset.inputs) == 0 {
		return fmt.Errorf("TrainModel: empty dataset")
	}
	batchSize := config.BatchSize
	if batchSize <= 0 {
		batchSize = 1
	}

	for epoch := 0; epoch < config.NumEpochs; epoch++ {
		epochLoss := 0.0
		steps := 0

		for i := 0; i < len(dataset.inputs); i++ {
			loss, err := model.TrainStep(dataset.inputs[i], dataset.targets[i], float32(config.LearningRate))
			if err != nil {
				return fmt.Errorf("train step %d: %w", i, err)
			}
			epochLoss += float64(loss)
			steps++

			if steps%batchSize == 0 {
				log.Printf("Epoch %d step %d: loss=%.4f", epoch, steps, loss)
			}
		}

		avgLoss := epochLoss / float64(steps)
		log.Printf("Epoch %d: avg loss = %.4f", epoch, avgLoss)

		if config.SaveFreq > 0 && epoch%config.SaveFreq == 0 {
			if err := SaveModel(model, fmt.Sprintf("checkpoint_epoch_%d.bin", epoch)); err != nil {
				log.Printf("checkpoint save failed at epoch %d: %v", epoch, err)
			}
		}
	}

	return nil
}

// savedParam is the gob-serializable form of one named tensor.
type savedParam struct {
	Name  string
	Shape []int
	Data  []float32
}

func SaveModel(model *GPT, filepath string) error {
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	names, values := model.namedParams()
	saved := make([]savedParam, len(names))
	for i, name := range names {
		saved[i] = savedParam{
			Name:  name,
			Shape: values[i].Shape(),
			Data:  values[i].Data().([]float32),
		}
	}

	return gob.NewEncoder(file).Encode(saved)
}

func LoadModel(model *GPT, filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	var saved []savedParam
	if err := gob.NewDecoder(file).Decode(&saved); err != nil {
		return err
	}

	byName := make(map[string][]float32, len(saved))
	for _, sp := range saved {
		byName[sp.Name] = sp.Data
	}

	names, values := model.namedParams()
	for i, name := range names {
		data, ok := byName[name]
		if !ok {
			continue
		}
		dst, ok := values[i].Data().([]float32)
		if !ok || len(dst) != len(data) {
			return fmt.Errorf("LoadModel: param %q shape mismatch", name)
		}
		copy(dst, data)
	}

	return nil
}

// Generate autoregressively samples maxNewTokens tokens continuing from
// startTokens, using the model's own (real, content-dependent) forward pass
// at each step.
func Generate(model *GPT, startTokens []int, maxNewTokens int, temperature float32, topK int) []int {
	tokens := make([]int, len(startTokens))
	copy(tokens, startTokens)

	for i := 0; i < maxNewTokens; i++ {
		context := tokens
		if len(context) > model.config.ContextLen {
			context = context[len(context)-model.config.ContextLen:]
		}

		logits, err := model.Forward(context)
		if err != nil {
			log.Printf("Generate: forward failed: %v", err)
			return tokens
		}

		lastStart := (len(context) - 1) * model.config.VocabSize
		lastLogits := logits[lastStart : lastStart+model.config.VocabSize]

		nextToken := SampleToken(lastLogits, temperature, topK)
		tokens = append(tokens, nextToken)
	}

	return tokens
}

func SampleToken(logits []float32, temperature float32, topK int) int {
	scored := make([]float32, len(logits))
	copy(scored, logits)
	if temperature != 1.0 && temperature > 0 {
		for i := range scored {
			scored[i] /= temperature
		}
	}

	maxLogit := scored[0]
	for _, l := range scored {
		if l > maxLogit {
			maxLogit = l
		}
	}

	expSum := float32(0)
	probs := make([]float32, len(scored))
	for i, l := range scored {
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
	model := NewGPT(cfg)
	if cfg == nil {
		cfg = model.config
	}
	dummy := make([]int, minInt(cfg.ContextLen, 8))
	for i := range dummy {
		dummy[i] = i % cfg.VocabSize
	}
	logits, err := model.Forward(dummy)
	if err != nil {
		return fmt.Errorf("forward pass failed: %w", err)
	}
	log.Printf("RunGorgoniteProtocol OK — logits len: %d (want %d)", len(logits), len(dummy)*cfg.VocabSize)
	return nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
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
	model := NewGPT(cfg)
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


// ---- HasherTransformer: hash-seed-based transformer (hardware-accelerated path) ----

// HasherTransformerConfig defines architecture for the hash-seed transformer.
// Named HasherTransformerConfig (vs GorgoniteConfig) to distinguish the two implementations.
type HasherTransformerConfig struct {
	VocabSize    int
	EmbedDim     int
	NumLayers    int
	NumHeads     int
	ContextLen   int
	DropoutRate  float32
	FFNHiddenDim int
	Activation   string // "hash", "tanh", "sigmoid"
}

// HasherTransformer implements a transformer whose weights are [32]byte seeds
// operated on via a core.HashMethod for hardware-accelerated inference.
type HasherTransformer struct {
	Config     *HasherTransformerConfig
	Embeddings [][][32]byte
	Positional [][][32]byte
	Layers     []hasherTransformerLayer
	OutputSeed [32]byte
	hashMethod core.HashMethod
}

type hasherTransformerLayer struct {
	QuerySeeds  [][][32]byte
	KeySeeds    [][][32]byte
	ValueSeeds  [][][32]byte
	OutputSeeds [][][32]byte
	FFNSeeds    [][][32]byte
	DecaySeeds  [][32]byte
	FFNOutSeeds [][32]byte
}

// NewHasherTransformer creates a randomly-seeded HasherTransformer.
func NewHasherTransformer(config *HasherTransformerConfig, hashMethod core.HashMethod) *HasherTransformer {
	model := &HasherTransformer{Config: config, hashMethod: hashMethod}

	model.Embeddings = make([][][32]byte, config.VocabSize)
	for i := 0; i < config.VocabSize; i++ {
		model.Embeddings[i] = make([][32]byte, config.EmbedDim)
		for j := 0; j < config.EmbedDim; j++ {
			rand.Read(model.Embeddings[i][j][:])
		}
	}
	model.Positional = make([][][32]byte, config.ContextLen)
	for i := 0; i < config.ContextLen; i++ {
		model.Positional[i] = make([][32]byte, config.EmbedDim)
		for j := 0; j < config.EmbedDim; j++ {
			rand.Read(model.Positional[i][j][:])
		}
	}
	model.Layers = make([]hasherTransformerLayer, config.NumLayers)
	for i := 0; i < config.NumLayers; i++ {
		model.Layers[i] = newHasherLayer(config)
	}
	rand.Read(model.OutputSeed[:])
	return model
}

func newHasherLayer(cfg *HasherTransformerConfig) hasherTransformerLayer {
	layer := hasherTransformerLayer{
		QuerySeeds:  make([][][32]byte, cfg.NumHeads),
		KeySeeds:    make([][][32]byte, cfg.NumHeads),
		ValueSeeds:  make([][][32]byte, cfg.NumHeads),
		OutputSeeds: make([][][32]byte, cfg.EmbedDim),
		FFNSeeds:    make([][][32]byte, cfg.FFNHiddenDim),
		DecaySeeds:  make([][32]byte, cfg.ContextLen),
		FFNOutSeeds: make([][32]byte, cfg.EmbedDim),
	}
	for h := 0; h < cfg.NumHeads; h++ {
		layer.QuerySeeds[h] = make([][32]byte, cfg.EmbedDim)
		layer.KeySeeds[h] = make([][32]byte, cfg.EmbedDim)
		layer.ValueSeeds[h] = make([][32]byte, cfg.EmbedDim)
		for j := 0; j < cfg.EmbedDim; j++ {
			rand.Read(layer.QuerySeeds[h][j][:])
			rand.Read(layer.KeySeeds[h][j][:])
			rand.Read(layer.ValueSeeds[h][j][:])
		}
	}
	for j := 0; j < cfg.EmbedDim; j++ {
		layer.OutputSeeds[j] = make([][32]byte, cfg.EmbedDim)
		for k := 0; k < cfg.EmbedDim; k++ {
			rand.Read(layer.OutputSeeds[j][k][:])
		}
	}
	for j := 0; j < cfg.FFNHiddenDim; j++ {
		layer.FFNSeeds[j] = make([][32]byte, cfg.EmbedDim)
		for k := 0; k < cfg.EmbedDim; k++ {
			rand.Read(layer.FFNSeeds[j][k][:])
		}
	}
	for s := 0; s < cfg.ContextLen; s++ {
		rand.Read(layer.DecaySeeds[s][:])
	}
	for j := 0; j < cfg.EmbedDim; j++ {
		rand.Read(layer.FFNOutSeeds[j][:])
	}
	return layer
}

// SetHashMethod updates the HashMethod used for hardware-accelerated inference.
func (ht *HasherTransformer) SetHashMethod(method core.HashMethod) { ht.hashMethod = method }

// Forward runs token IDs through all layers and returns a pooled float32 vector.
// Uses HardwareRouter for hardware-accelerated projections when a HashMethod is available.
func (ht *HasherTransformer) Forward(tokenIDs []int) []float32 {
	if len(tokenIDs) == 0 {
		return make([]float32, ht.Config.EmbedDim)
	}
	router := NewHardwareRouter(ht.hashMethod, FallbackMixed)
	hidden := make([][]float32, len(tokenIDs))
	for i, id := range tokenIDs {
		hidden[i] = ht.embedToken(id, i, router)
	}
	for l := 0; l < ht.Config.NumLayers; l++ {
		hidden = ht.forwardHasherLayer(hidden, l, router)
	}
	return ht.averagePool(hidden)
}

func (ht *HasherTransformer) embedToken(tokenID, position int, router *HardwareRouter) []float32 {
	dim := ht.Config.EmbedDim
	out := make([]float32, dim)
	tokenID = tokenID % ht.Config.VocabSize
	if tokenID < len(ht.Embeddings) {
		seeds := ht.Embeddings[tokenID]
		proj, err := router.Project(make([]float32, dim), seeds)
		if err == nil {
			copy(out, proj)
		}
	}
	if position < ht.Config.ContextLen && position < len(ht.Positional) {
		seeds := ht.Positional[position]
		add, err := router.Project(make([]float32, dim), seeds)
		if err == nil {
			for j := 0; j < dim; j++ {
				out[j] += add[j]
			}
		}
	}
	return out
}

func (ht *HasherTransformer) forwardHasherLayer(hidden [][]float32, layerIdx int, router *HardwareRouter) [][]float32 {
	seqLen := len(hidden)
	dim := ht.Config.EmbedDim
	layer := ht.Layers[layerIdx]

	attn := ht.hasherMultiHeadAttention(hidden, layer, router)
	for i := 0; i < seqLen; i++ {
		for j := 0; j < dim; j++ {
			hidden[i][j] = ht.hasherLayerNorm(hidden[i][j] + attn[i][j])
		}
	}
	ffn := ht.hasherFFN(hidden, layer, router)
	for i := 0; i < seqLen; i++ {
		for j := 0; j < dim; j++ {
			hidden[i][j] = ht.hasherLayerNorm(hidden[i][j] + ffn[i][j])
		}
	}
	return hidden
}

func (ht *HasherTransformer) hasherMultiHeadAttention(hidden [][]float32, layer hasherTransformerLayer, router *HardwareRouter) [][]float32 {
	seqLen := len(hidden)
	dim := ht.Config.EmbedDim
	out := make([][]float32, seqLen)
	for i := range out {
		out[i] = make([]float32, dim)
	}

	cumDecay := precomputeCumulativeDecay(layer.DecaySeeds, seqLen)

	for i := 0; i < seqLen; i++ {
		for h := 0; h < ht.Config.NumHeads; h++ {
			q := ht.projectSeeds(hidden[i], layer.QuerySeeds[h], router)
			for j := 0; j <= i; j++ {
				k := ht.projectSeeds(hidden[j], layer.KeySeeds[h], router)
				v := ht.projectSeeds(hidden[j], layer.ValueSeeds[h], router)
				match := dotProduct(q, k)
				if match > 88.0 {
					match = 88.0
				} else if match < -88.0 {
					match = -88.0
				}
				attentionWeight := float32(math.Exp(float64(match)))
				decay := getDecayFromCumulative(j+1, i+1, cumDecay)
				for d := 0; d < dim && d < len(v); d++ {
					out[i][d] += attentionWeight * decay * v[d] / float32(ht.Config.NumHeads)
				}
			}
		}
		out[i] = ht.projectSeeds2D(out[i], layer.OutputSeeds, router)
	}
	return out
}

func (ht *HasherTransformer) hasherFFN(hidden [][]float32, layer hasherTransformerLayer, router *HardwareRouter) [][]float32 {
	outSeeds := ffnOutSeedsOrDerive(layer.FFNOutSeeds, layer.FFNSeeds, ht.Config.EmbedDim)
	out := make([][]float32, len(hidden))
	for i, h := range hidden {
		expanded := ht.projectSeeds2D(h, layer.FFNSeeds, router)
		out[i] = ht.projectBack(expanded, outSeeds, router)
	}
	return out
}

func (ht *HasherTransformer) projectSeeds(input []float32, seeds [][32]byte, router *HardwareRouter) []float32 {
	if router != nil {
		if out, err := router.Project(input, seeds); err == nil && len(out) > 0 {
			return out
		}
	}
	return ht.projectSeedsFallback(input, seeds)
}

func (ht *HasherTransformer) projectSeedsFallback(input []float32, seeds [][32]byte) []float32 {
	return ProjectSeeds(input, seeds, ht.Config.Activation)
}

func (ht *HasherTransformer) projectSeeds2D(input []float32, seeds [][][32]byte, router *HardwareRouter) []float32 {
	if router != nil {
		if out, err := router.ProjectBatch2D(input, seeds); err == nil && len(out) > 0 {
			return out
		}
	}
	return ht.projectSeeds2DFallback(input, seeds)
}

func (ht *HasherTransformer) projectSeeds2DFallback(input []float32, seeds [][][32]byte) []float32 {
	return ProjectSeeds2D(input, seeds, ht.Config.Activation)
}

// projectBack projects an FFN-expanded vector back down to len(seeds) output
// dimensions. Earlier this discarded seeds entirely on the software path
// (averaged every input into one scalar and broadcast it to every output —
// see ProjectBack in seed_utils.go for why that was a rank-1 bottleneck), and
// on the "hardware" path treated the raw float32 bit-patterns of a zeroed
// input as fake seeds, which was disconnected from any real seed material.
// Both paths now go through the same per-output-neuron seeds as every other
// projection.
func (ht *HasherTransformer) projectBack(input []float32, seeds [][32]byte, router *HardwareRouter) []float32 {
	if router != nil {
		if out, err := router.Project(input, seeds); err == nil && len(out) > 0 {
			return out
		}
	}
	return ht.projectBackFallback(input, seeds)
}

func (ht *HasherTransformer) projectBackFallback(input []float32, seeds [][32]byte) []float32 {
	return ProjectBack(input, seeds, ht.Config.Activation)
}

func (ht *HasherTransformer) averagePool(hidden [][]float32) []float32 {
	if len(hidden) == 0 {
		return nil
	}
	out := make([]float32, len(hidden[0]))
	for _, row := range hidden {
		for j, v := range row {
			out[j] += v
		}
	}
	n := float32(len(hidden))
	for j := range out {
		out[j] /= n
	}
	return out
}

// seedToFloat delegates to the shared, signed [-1,1] SeedToFloat (seed_utils.go)
// so HasherTransformer and UnifiedHasherEngine can never drift apart on what a
// seed means numerically.
func (ht *HasherTransformer) seedToFloat(seed [32]byte) float32 {
	return SeedToFloat(seed)
}

func (ht *HasherTransformer) activate(x float32) float32 {
	return Activate(x, ht.Config.Activation)
}

func (ht *HasherTransformer) hasherLayerNorm(x float32) float32 {
	if x < -10 {
		return -10
	}
	if x > 10 {
		return 10
	}
	return x
}

// GenerateToken produces the next token given a context and temperature.
func (ht *HasherTransformer) GenerateToken(ctx []int, temperature float32) (int, []float32) {
	hidden := ht.Forward(ctx)
	router := NewHardwareRouter(ht.hashMethod, FallbackMixed)
	scores := router.HashToVocab(hidden, ht.OutputSeed, ht.Config.VocabSize)
	if temperature <= 0 {
		return argmax32(scores), scores
	}
	return sampleTemp32(scores, temperature), scores
}

func argmax32(s []float32) int {
	best := 0
	for i, v := range s {
		if v > s[best] {
			best = i
		}
	}
	return best
}

func sampleTemp32(scores []float32, temp float32) int {
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
	return len(probs) - 1
}

// ---- HasherTransformer backward-compatibility wrappers ----

// ForwardWrapper runs Forward() via the unified engine for backward compatibility.
// When hashMethod is non-nil, the unified engine attempts hardware acceleration.
func (ht *HasherTransformer) ForwardWrapper(tokenIDs []int) []float32 {
	engine, err := NewUnifiedHasherEngineFromHasherTransformer(ht)
	if err != nil {
		return ht.Forward(tokenIDs)
	}
	return engine.Forward(tokenIDs)
}

// GenerateTokenWrapper produces the next token via the unified engine.
func (ht *HasherTransformer) GenerateTokenWrapper(ctx []int, temperature float32) (int, []float32) {
	engine, err := NewUnifiedHasherEngineFromHasherTransformer(ht)
	if err != nil {
		return ht.GenerateToken(ctx, temperature)
	}
	return engine.GenerateToken(ctx, temperature)
}

// hasherTransformerConfigToUnified converts a HasherTransformerConfig to the
// equivalent UnifiedConfig.
func hasherTransformerConfigToUnified(cfg *HasherTransformerConfig) *UnifiedConfig {
	return &UnifiedConfig{
		VocabSize:    cfg.VocabSize,
		EmbedDim:     cfg.EmbedDim,
		NumHeads:     cfg.NumHeads,
		NumLayers:    cfg.NumLayers,
		ContextLen:   cfg.ContextLen,
		FFNHiddenDim: cfg.FFNHiddenDim,
		Activation:   cfg.Activation,
		Passes:       21,
		Jitter:       0.01,
		SeedRotation: false,
	}
}

// hasherTransformerToSeedStore extracts ht's seeds into a standalone
// SeedStore, used by NewUnifiedHasherEngineFromHasherTransformer for the
// HasherTransformer → UnifiedHasherEngine conversion path.
func hasherTransformerToSeedStore(ht *HasherTransformer) *SeedStore {
	seeds := &SeedStore{
		Embeddings: ht.Embeddings,
		Positional: ht.Positional,
		Layers:     make([]TransformerLayerSeeds, len(ht.Layers)),
		OutputSeed: ht.OutputSeed,
	}
	for i, l := range ht.Layers {
		seeds.Layers[i] = TransformerLayerSeeds{
			QuerySeeds:  l.QuerySeeds,
			KeySeeds:    l.KeySeeds,
			ValueSeeds:  l.ValueSeeds,
			OutputSeeds: l.OutputSeeds,
			FFNSeeds:    l.FFNSeeds,
			DecaySeeds:  l.DecaySeeds,
			FFNOutSeeds: l.FFNOutSeeds,
		}
	}
	return seeds
}

// NewUnifiedHasherEngineFromHasherTransformer converts a legacy HasherTransformer
// into a UnifiedHasherEngine in transformer mode.
func NewUnifiedHasherEngineFromHasherTransformer(ht *HasherTransformer) (*UnifiedHasherEngine, error) {
	if ht == nil || ht.Config == nil {
		return nil, fmt.Errorf("nil HasherTransformer")
	}
	cfg := hasherTransformerConfigToUnified(ht.Config)
	seeds := hasherTransformerToSeedStore(ht)
	return NewUnifiedHasherEngineWithConfig(cfg, seeds, ht.hashMethod, ModeTransformer), nil
}

