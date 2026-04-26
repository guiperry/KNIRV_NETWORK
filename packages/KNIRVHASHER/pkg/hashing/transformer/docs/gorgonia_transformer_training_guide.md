# Complete Training Guide: Transformers with Go Libraries

This guide shows you how to build and train a Transformer using real Go ML libraries: **cudago** (CUDA), **Gorgonia** (computation graphs), and **autograd** (automatic differentiation).

## 📋 Prerequisites & Setup

### 1. System Requirements

```bash
# NVIDIA GPU with CUDA support
# CUDA 12.6+ installed
# Go 1.21+

# Check CUDA installation
nvcc --version

# Set environment variables
export PATH=/usr/local/cuda/bin:$PATH
export LD_LIBRARY_PATH=/usr/local/cuda/lib64:$LD_LIBRARY_PATH
```

### 2. Install Go Libraries

```bash
# Install Gorgonia (main ML framework)
go get -u gorgonia.org/gorgonia
go get -u gorgonia.org/tensor

# Install cudago for CUDA acceleration
go get -u github.com/InternatBlackhole/cudago/cuda

# Install autograd for automatic differentiation
go get -u github.com/itsubaki/autograd

# Optional: Additional dependencies
go get -u gonum.org/v1/gonum/mat
```

## 🏗️ Part 1: Understanding the Architecture

### Library Roles

**Gorgonia**: 
- Core computation graph framework
- Tensor operations
- Automatic differentiation (built-in)
- GPU support through CUDA backend

**cudago**:
- Low-level CUDA bindings
- Custom GPU kernels
- Memory management
- Direct GPU operations

**autograd**:
- Simple automatic differentiation
- Good for learning concepts
- Lightweight alternative

## 💻 Part 2: Building the Transformer with Gorgonia

### Step 1: Initialize Gorgonia Environment

```go
package main

import (
    "fmt"
    "log"
    
    "gorgonia.org/gorgonia"
    "gorgonia.org/tensor"
    "github.com/InternatBlackhole/cudago/cuda"
)

type TransformerConfig struct {
    VocabSize    int
    EmbedDim     int
    NumHeads     int
    NumLayers    int
    ContextLen   int
    DropoutRate  float64
    FFNHiddenDim int
}

func main() {
    // Initialize CUDA
    dev, err := cuda.Init(0) // 0 = first GPU
    if err != nil {
        log.Fatal("CUDA initialization failed:", err)
    }
    defer dev.Close()
    
    fmt.Printf("Using GPU: %s\n", dev.Name())
    fmt.Printf("Total Memory: %.2f GB\n", float64(dev.TotalMem())/(1024*1024*1024))
    
    // Your training code here
}
```

### Step 2: Create Transformer Components

```go
// Embedding Layer
type EmbeddingLayer struct {
    graph      *gorgonia.ExprGraph
    weights    *gorgonia.Node
    vocabSize  int
    embedDim   int
}

func NewEmbeddingLayer(g *gorgonia.ExprGraph, vocabSize, embedDim int) *EmbeddingLayer {
    // Initialize embedding weights with Xavier initialization
    weights := gorgonia.NewMatrix(g,
        tensor.Float32,
        gorgonia.WithShape(vocabSize, embedDim),
        gorgonia.WithName("embedding_weights"),
        gorgonia.WithInit(gorgonia.GlorotN(1.0)),
    )
    
    return &EmbeddingLayer{
        graph:     g,
        weights:   weights,
        vocabSize: vocabSize,
        embedDim:  embedDim,
    }
}

func (e *EmbeddingLayer) Forward(tokenIDs *gorgonia.Node) (*gorgonia.Node, error) {
    // Lookup embeddings for token IDs
    embedded, err := gorgonia.Gather(e.weights, tokenIDs, 0)
    if err != nil {
        return nil, err
    }
    return embedded, nil
}

// Positional Encoding
type PositionalEncoding struct {
    graph      *gorgonia.ExprGraph
    encoding   *gorgonia.Node
    contextLen int
    embedDim   int
}

func NewPositionalEncoding(g *gorgonia.ExprGraph, contextLen, embedDim int) *PositionalEncoding {
    // Create sinusoidal positional encodings
    encoding := make([]float32, contextLen*embedDim)
    
    for pos := 0; pos < contextLen; pos++ {
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
        tensor.WithShape(contextLen, embedDim),
    )
    
    posEncNode := gorgonia.NewConstant(posEncTensor, gorgonia.WithName("pos_encoding"))
    
    return &PositionalEncoding{
        graph:      g,
        encoding:   posEncNode,
        contextLen: contextLen,
        embedDim:   embedDim,
    }
}

func (p *PositionalEncoding) Forward(seqLen int) (*gorgonia.Node, error) {
    // Slice positional encoding to match sequence length
    sliced, err := gorgonia.Slice(p.encoding, gorgonia.S(0, seqLen))
    if err != nil {
        return nil, err
    }
    return sliced, nil
}

// Self-Attention Mechanism
type SelfAttention struct {
    graph       *gorgonia.ExprGraph
    wQuery      *gorgonia.Node
    wKey        *gorgonia.Node
    wValue      *gorgonia.Node
    embedDim    int
    headDim     int
    dropout     float64
}

func NewSelfAttention(g *gorgonia.ExprGraph, embedDim, headDim int, dropout float64) *SelfAttention {
    // Initialize Q, K, V projection matrices
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
    // Q = X @ W_Q
    q, err := gorgonia.Mul(x, sa.wQuery)
    if err != nil {
        return nil, err
    }
    
    // K = X @ W_K
    k, err := gorgonia.Mul(x, sa.wKey)
    if err != nil {
        return nil, err
    }
    
    // V = X @ W_V
    v, err := gorgonia.Mul(x, sa.wValue)
    if err != nil {
        return nil, err
    }
    
    // Attention scores = Q @ K^T / sqrt(d_k)
    kT, err := gorgonia.Transpose(k)
    if err != nil {
        return nil, err
    }
    
    scores, err := gorgonia.Mul(q, kT)
    if err != nil {
        return nil, err
    }
    
    // Scale by sqrt(head_dim)
    scale := float32(1.0 / math.Sqrt(float64(sa.headDim)))
    scaleNode := gorgonia.NewScalar(sa.graph, tensor.Float32, gorgonia.WithValue(scale))
    scores, err = gorgonia.HadamardProd(scores, scaleNode)
    if err != nil {
        return nil, err
    }
    
    // Apply causal mask (if provided)
    if mask != nil {
        scores, err = gorgonia.Add(scores, mask)
        if err != nil {
            return nil, err
        }
    }
    
    // Softmax
    attnWeights, err := gorgonia.SoftMax(scores)
    if err != nil {
        return nil, err
    }
    
    // Apply dropout during training
    if training && sa.dropout > 0 {
        attnWeights, err = gorgonia.Dropout(attnWeights, sa.dropout)
        if err != nil {
            return nil, err
        }
    }
    
    // Output = Attention_weights @ V
    output, err := gorgonia.Mul(attnWeights, v)
    if err != nil {
        return nil, err
    }
    
    return output, nil
}

// Multi-Head Attention
type MultiHeadAttention struct {
    graph       *gorgonia.ExprGraph
    heads       []*SelfAttention
    wOutput     *gorgonia.Node
    numHeads    int
    embedDim    int
    headDim     int
}

func NewMultiHeadAttention(g *gorgonia.ExprGraph, embedDim, numHeads int, dropout float64) *MultiHeadAttention {
    headDim := embedDim / numHeads
    
    // Create attention heads
    heads := make([]*SelfAttention, numHeads)
    for i := 0; i < numHeads; i++ {
        heads[i] = NewSelfAttention(g, embedDim, headDim, dropout)
    }
    
    // Output projection
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
    // Process each head
    headOutputs := make([]*gorgonia.Node, mha.numHeads)
    for i := 0; i < mha.numHeads; i++ {
        out, err := mha.heads[i].Forward(x, mask, training)
        if err != nil {
            return nil, err
        }
        headOutputs[i] = out
    }
    
    // Concatenate heads
    concat, err := gorgonia.Concat(1, headOutputs...)
    if err != nil {
        return nil, err
    }
    
    // Output projection
    output, err := gorgonia.Mul(concat, mha.wOutput)
    if err != nil {
        return nil, err
    }
    
    return output, nil
}

// Feed-Forward Network
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
    // First linear layer
    hidden, err := gorgonia.Mul(x, ff.w1)
    if err != nil {
        return nil, err
    }
    
    // GELU activation
    // GELU(x) = 0.5 * x * (1 + tanh(sqrt(2/π) * (x + 0.044715 * x^3)))
    hidden, err = gorgonia.Tanh(hidden) // Simplified - use proper GELU
    if err != nil {
        return nil, err
    }
    
    // Dropout
    if training && ff.dropout > 0 {
        hidden, err = gorgonia.Dropout(hidden, ff.dropout)
        if err != nil {
            return nil, err
        }
    }
    
    // Second linear layer
    output, err := gorgonia.Mul(hidden, ff.w2)
    if err != nil {
        return nil, err
    }
    
    return output, nil
}

// Transformer Block
type TransformerBlock struct {
    graph       *gorgonia.ExprGraph
    attention   *MultiHeadAttention
    feedForward *FeedForward
    layerNorm1  *gorgonia.Node
    layerNorm2  *gorgonia.Node
}

func NewTransformerBlock(g *gorgonia.ExprGraph, embedDim, numHeads int, dropout float64) *TransformerBlock {
    return &TransformerBlock{
        graph:       g,
        attention:   NewMultiHeadAttention(g, embedDim, numHeads, dropout),
        feedForward: NewFeedForward(g, embedDim, dropout),
    }
}

func (tb *TransformerBlock) Forward(x *gorgonia.Node, mask *gorgonia.Node, training bool) (*gorgonia.Node, error) {
    // Self-attention with residual connection
    attnOut, err := tb.attention.Forward(x, mask, training)
    if err != nil {
        return nil, err
    }
    
    // Add & Norm
    x, err = gorgonia.Add(x, attnOut)
    if err != nil {
        return nil, err
    }
    
    x, err = gorgonia.LayerNorm(x, nil, nil) // Simplified
    if err != nil {
        return nil, err
    }
    
    // Feed-forward with residual connection
    ffOut, err := tb.feedForward.Forward(x, training)
    if err != nil {
        return nil, err
    }
    
    // Add & Norm
    x, err = gorgonia.Add(x, ffOut)
    if err != nil {
        return nil, err
    }
    
    x, err = gorgonia.LayerNorm(x, nil, nil)
    if err != nil {
        return nil, err
    }
    
    return x, nil
}

// Complete GPT Model
type GPT struct {
    graph          *gorgonia.ExprGraph
    config         TransformerConfig
    embedding      *EmbeddingLayer
    posEncoding    *PositionalEncoding
    blocks         []*TransformerBlock
    outputLayer    *gorgonia.Node
}

func NewGPT(config TransformerConfig) *GPT {
    g := gorgonia.NewGraph()
    
    // Create layers
    embedding := NewEmbeddingLayer(g, config.VocabSize, config.EmbedDim)
    posEncoding := NewPositionalEncoding(g, config.ContextLen, config.EmbedDim)
    
    // Create transformer blocks
    blocks := make([]*TransformerBlock, config.NumLayers)
    for i := 0; i < config.NumLayers; i++ {
        blocks[i] = NewTransformerBlock(g, config.EmbedDim, config.NumHeads, config.DropoutRate)
    }
    
    // Output projection (to vocabulary)
    outputLayer := gorgonia.NewMatrix(g, tensor.Float32,
        gorgonia.WithShape(config.EmbedDim, config.VocabSize),
        gorgonia.WithName("output_projection"),
        gorgonia.WithInit(gorgonia.GlorotN(1.0)),
    )
    
    return &GPT{
        graph:       g,
        config:      config,
        embedding:   embedding,
        posEncoding: posEncoding,
        blocks:      blocks,
        outputLayer: outputLayer,
    }
}

func (gpt *GPT) Forward(tokenIDs *gorgonia.Node, training bool) (*gorgonia.Node, error) {
    // Embed tokens
    x, err := gpt.embedding.Forward(tokenIDs)
    if err != nil {
        return nil, err
    }
    
    // Add positional encoding
    seqLen := tokenIDs.Shape()[0]
    posEnc, err := gpt.posEncoding.Forward(seqLen)
    if err != nil {
        return nil, err
    }
    
    x, err = gorgonia.Add(x, posEnc)
    if err != nil {
        return nil, err
    }
    
    // Create causal mask
    mask := createCausalMask(gpt.graph, seqLen)
    
    // Pass through transformer blocks
    for _, block := range gpt.blocks {
        x, err = block.Forward(x, mask, training)
        if err != nil {
            return nil, err
        }
    }
    
    // Output projection
    logits, err := gorgonia.Mul(x, gpt.outputLayer)
    if err != nil {
        return nil, err
    }
    
    return logits, nil
}

func createCausalMask(g *gorgonia.ExprGraph, seqLen int) *gorgonia.Node {
    // Create upper triangular mask for causal attention
    // Returns large negative values for future positions
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
```

## 🎯 Part 3: Training Loop

### Step 1: Prepare Dataset

```go
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
```

### Step 2: Define Training Configuration

```go
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
```

### Step 3: Training Loop with Gorgonia

```go
func TrainModel(model *GPT, dataset *Dataset, config TrainConfig) error {
    // Create VM for executing the graph
    vm := gorgonia.NewTapeMachine(model.graph, gorgonia.BindDualValues())
    defer vm.Close()
    
    // Create solver (optimizer)
    solver := gorgonia.NewAdamSolver(
        gorgonia.WithLearnRate(config.LearningRate),
        gorgonia.WithBatchSize(float64(config.BatchSize)),
        gorgonia.WithClip(config.GradientClip),
    )
    
    // Training loop
    for epoch := 0; epoch < config.NumEpochs; epoch++ {
        epochLoss := 0.0
        
        for i := 0; i < len(dataset.inputs); i += config.BatchSize {
            // Get batch
            batchEnd := min(i+config.BatchSize, len(dataset.inputs))
            batchInputs := dataset.inputs[i:batchEnd]
            batchTargets := dataset.targets[i:batchEnd]
            
            // Convert to tensors
            inputTensor := intsToTensor(batchInputs)
            targetTensor := intsToTensor(batchTargets)
            
            // Create nodes for this batch
            inputNode := gorgonia.NewMatrix(model.graph, tensor.Int,
                gorgonia.WithShape(len(batchInputs), len(batchInputs[0])),
                gorgonia.WithValue(inputTensor),
            )
            
            targetNode := gorgonia.NewMatrix(model.graph, tensor.Int,
                gorgonia.WithShape(len(batchTargets), len(batchTargets[0])),
                gorgonia.WithValue(targetTensor),
            )
            
            // Forward pass
            logits, err := model.Forward(inputNode, true) // training=true
            if err != nil {
                return fmt.Errorf("forward pass error: %w", err)
            }
            
            // Compute loss (cross-entropy)
            loss, err := gorgonia.SoftMaxCrossEntropy(logits, targetNode)
            if err != nil {
                return fmt.Errorf("loss computation error: %w", err)
            }
            
            // Backward pass
            if _, err := gorgonia.Grad(loss, model.graph.AllNodes()...); err != nil {
                return fmt.Errorf("backward pass error: %w", err)
            }
            
            // Run the VM
            if err := vm.RunAll(); err != nil {
                return fmt.Errorf("VM execution error: %w", err)
            }
            
            // Update weights
            if err := solver.Step(gorgonia.NodesToValueGrads(model.graph.AllNodes())); err != nil {
                return fmt.Errorf("solver step error: %w", err)
            }
            
            // Reset for next iteration
            vm.Reset()
            
            // Accumulate loss
            lossValue := loss.Value().Data().(float32)
            epochLoss += float64(lossValue)
        }
        
        avgLoss := epochLoss / float64(len(dataset.inputs)/config.BatchSize)
        fmt.Printf("Epoch %d: Loss = %.4f\n", epoch, avgLoss)
        
        // Save checkpoint
        if epoch%config.SaveFreq == 0 {
            SaveModel(model, fmt.Sprintf("checkpoint_epoch_%d.bin", epoch))
        }
    }
    
    return nil
}
```

## ⚡ Part 4: CUDA Acceleration with cudago

### Custom CUDA Kernels for Performance

```go
// Example: Custom matrix multiplication kernel
func MatMulCUDA(a, b []float32, m, n, k int) ([]float32, error) {
    // Initialize CUDA
    dev, err := cuda.Init(0)
    if err != nil {
        return nil, err
    }
    defer dev.Close()
    
    // Allocate device memory
    aSize := uint64(m * k * 4) // float32 = 4 bytes
    bSize := uint64(k * n * 4)
    cSize := uint64(m * n * 4)
    
    dA, _ := cuda.MemAlloc[float32](uint64(len(a)), 4, 0)
    defer dA.Free()
    
    dB, _ := cuda.MemAlloc[float32](uint64(len(b)), 4, 0)
    defer dB.Free()
    
    dC, _ := cuda.MemAlloc[float32](uint64(m*n), 4, 0)
    defer dC.Free()
    
    // Copy data to device
    dA.CopyHtoD(a)
    dB.CopyHtoD(b)
    
    // Load and execute CUDA kernel
    // (You would need to write the .cu kernel file separately)
    module, _ := cuda.LoadLibraryFromPath("matmul.ptx", nil, nil)
    defer module.Unload()
    
    kernel, _ := module.GetKernel("matmul_kernel")
    
    // Configure kernel launch
    blockSize := 16
    gridX := (m + blockSize - 1) / blockSize
    gridY := (n + blockSize - 1) / blockSize
    
    // Launch kernel
    kernel.Launch(
        gridX, gridY, 1,    // grid dimensions
        blockSize, blockSize, 1,  // block dimensions
        0, nil,             // shared memory, stream
        dA, dB, dC, m, n, k, // kernel arguments
    )
    
    // Copy result back
    result := make([]float32, m*n)
    dC.CopyDtoH(result)
    
    return result, nil
}
```

### CUDA Kernel File (matmul.cu)

```cuda
extern "C" __global__
void matmul_kernel(const float* A, const float* B, float* C,
                   int m, int n, int k) {
    int row = blockIdx.y * blockDim.y + threadIdx.y;
    int col = blockIdx.x * blockDim.x + threadIdx.x;
    
    if (row < m && col < n) {
        float sum = 0.0f;
        for (int i = 0; i < k; i++) {
            sum += A[row * k + i] * B[i * n + col];
        }
        C[row * n + col] = sum;
    }
}
```

Compile with:
```bash
nvcc -ptx -o matmul.ptx matmul.cu
```

## 🎓 Part 5: Using itsubaki/autograd for Learning

The autograd library is great for understanding automatic differentiation:

```go
package main

import (
    "fmt"
    "github.com/itsubaki/autograd"
)

func LearnAutograd() {
    // Create variables
    x := autograd.NewVariable(2.0)
    w := autograd.NewVariable(3.0)
    b := autograd.NewVariable(1.0)
    
    // Define computation: y = w*x + b
    y := autograd.Add(autograd.Mul(w, x), b)
    
    // Compute gradients
    y.Backward()
    
    // Access gradients
    fmt.Printf("dy/dx = %.2f\n", x.Grad.Data[0])  // Should be 3.0
    fmt.Printf("dy/dw = %.2f\n", w.Grad.Data[0])  // Should be 2.0
    fmt.Printf("dy/db = %.2f\n", b.Grad.Data[0])  // Should be 1.0
}
```

## 📊 Part 6: Complete Training Pipeline

### Full Example

```go
package main

import (
    "fmt"
    "log"
)

func main() {
    // 1. Initialize CUDA
    dev, err := cuda.Init(0)
    if err != nil {
        log.Fatal(err)
    }
    defer dev.Close()
    
    // 2. Define model configuration
    config := TransformerConfig{
        VocabSize:    50257,  // GPT-2 vocab size
        EmbedDim:     512,
        NumHeads:     8,
        NumLayers:    6,
        ContextLen:   512,
        DropoutRate:  0.1,
        FFNHiddenDim: 2048,
    }
    
    // 3. Create model
    model := NewGPT(config)
    
    // 4. Load and prepare dataset
    text := loadTextData("dataset.txt")
    tokenizer := NewBPETokenizer("tokenizer.json")
    dataset := PrepareDataset(text, tokenizer, config.ContextLen, config.ContextLen)
    
    fmt.Printf("Dataset: %d samples\n", len(dataset.inputs))
    
    // 5. Configure training
    trainConfig := GetPretrainingConfig()
    
    // 6. Train model
    fmt.Println("Starting training...")
    if err := TrainModel(model, dataset, trainConfig); err != nil {
        log.Fatal("Training error:", err)
    }
    
    // 7. Save final model
    SaveModel(model, "final_model.bin")
    
    // 8. Test generation
    prompt := []int{1, 2, 3, 4, 5}
    generated := Generate(model, prompt, 100)
    fmt.Println("Generated:", tokenizer.Decode(generated))
}
```

## 🔧 Part 7: Helper Functions & Utilities

### Model Saving and Loading

```go
import (
    "encoding/gob"
    "os"
)

func SaveModel(model *GPT, filepath string) error {
    file, err := os.Create(filepath)
    if err != nil {
        return err
    }
    defer file.Close()
    
    encoder := gob.NewEncoder(file)
    
    // Extract all learnable parameters
    params := make(map[string][]float32)
    
    for _, node := range model.graph.AllNodes() {
        if node.Value() != nil {
            // Convert tensor to float32 slice
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
    
    // Load parameters back into model
    for _, node := range model.graph.AllNodes() {
        if data, exists := params[node.Name()]; exists {
            // Create tensor from data
            t := tensor.New(
                tensor.WithBacking(data),
                tensor.WithShape(node.Shape()...),
            )
            gorgonia.Let(node, t)
        }
    }
    
    return nil
}
```

### Text Generation Functions

```go
func Generate(model *GPT, startTokens []int, maxNewTokens int, 
              temperature float32, topK int) []int {
    tokens := make([]int, len(startTokens))
    copy(tokens, startTokens)
    
    // Create VM for inference
    vm := gorgonia.NewTapeMachine(model.graph)
    defer vm.Close()
    
    for i := 0; i < maxNewTokens; i++ {
        // Get context window
        context := tokens
        if len(context) > model.config.ContextLen {
            context = context[len(context)-model.config.ContextLen:]
        }
        
        // Create input tensor
        inputTensor := tensor.New(
            tensor.WithBacking(context),
            tensor.WithShape(len(context)),
        )
        
        inputNode := gorgonia.NewMatrix(model.graph, tensor.Int,
            gorgonia.WithShape(len(context)),
            gorgonia.WithValue(inputTensor),
        )
        
        // Forward pass (training=false)
        logits, err := model.Forward(inputNode, false)
        if err != nil {
            log.Fatal(err)
        }
        
        if err := vm.RunAll(); err != nil {
            log.Fatal(err)
        }
        
        // Get logits for last token
        logitsData := logits.Value().Data().([]float32)
        lastLogits := logitsData[len(context)*model.config.VocabSize:]
        
        // Apply temperature and sample
        nextToken := SampleToken(lastLogits[:model.config.VocabSize], 
                                temperature, topK)
        tokens = append(tokens, nextToken)
        
        vm.Reset()
    }
    
    return tokens
}

func SampleToken(logits []float32, temperature float32, topK int) int {
    // Apply temperature
    if temperature != 1.0 {
        for i := range logits {
            logits[i] /= temperature
        }
    }
    
    // Softmax
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
    
    // Top-K sampling
    if topK > 0 && topK < len(probs) {
        type indexedProb struct {
            idx  int
            prob float32
        }
        
        indexed := make([]indexedProb, len(probs))
        for i, p := range probs {
            indexed[i] = indexedProb{i, p}
        }
        
        // Sort by probability (descending)
        sort.Slice(indexed, func(i, j int) bool {
            return indexed[i].prob > indexed[j].prob
        })
        
        // Zero out everything except top-K
        for i := topK; i < len(indexed); i++ {
            probs[indexed[i].idx] = 0
        }
        
        // Renormalize
        sum := float32(0)
        for _, p := range probs {
            sum += p
        }
        for i := range probs {
            probs[i] /= sum
        }
    }
    
    // Sample from distribution
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
```

### Tokenizer Implementation

```go
type BPETokenizer struct {
    vocab      map[string]int
    invVocab   map[int]string
    merges     [][]string
    vocabSize  int
}

func NewBPETokenizer(vocabPath string) *BPETokenizer {
    // Load vocabulary and merges from file
    // This is simplified - use actual BPE implementation
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
    // Simplified encoding - implement proper BPE
    tokens := make([]int, 0)
    
    // For now, just split by whitespace and lookup
    words := strings.Fields(text)
    for _, word := range words {
        if idx, exists := t.vocab[word]; exists {
            tokens = append(tokens, idx)
        } else {
            // Handle unknown tokens
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
    // Load from JSON or text file
    vocab := make(map[string]int)
    // Implementation here...
    return vocab
}
```

## 📈 Part 8: Monitoring and Evaluation

### Training Metrics

```go
type TrainingMetrics struct {
    TrainLosses []float64
    ValLosses   []float64
    Perplexities []float64
    LearningRates []float64
    TokensSeen  []int
}

func (m *TrainingMetrics) Log(epoch int, trainLoss, valLoss, lr float64, tokens int) {
    m.TrainLosses = append(m.TrainLosses, trainLoss)
    m.ValLosses = append(m.ValLosses, valLoss)
    m.Perplexities = append(m.Perplexities, math.Exp(valLoss))
    m.LearningRates = append(m.LearningRates, lr)
    m.TokensSeen = append(m.TokensSeen, tokens)
    
    fmt.Printf("Epoch %d | Train Loss: %.4f | Val Loss: %.4f | PPL: %.2f | LR: %.6f\n",
        epoch, trainLoss, valLoss, math.Exp(valLoss), lr)
}

func (m *TrainingMetrics) SaveToFile(filepath string) error {
    file, err := os.Create(filepath)
    if err != nil {
        return err
    }
    defer file.Close()
    
    encoder := json.NewEncoder(file)
    encoder.SetIndent("", "  ")
    return encoder.Encode(m)
}

func (m *TrainingMetrics) PlotMetrics() {
    // Use a plotting library like gonum/plot
    // Example structure:
    // - Plot train/val loss over time
    // - Plot perplexity
    // - Plot learning rate schedule
}
```

### Validation Loop

```go
func ValidateModel(model *GPT, valDataset *Dataset) float64 {
    vm := gorgonia.NewTapeMachine(model.graph)
    defer vm.Close()
    
    totalLoss := 0.0
    numBatches := 0
    
    for i := 0; i < len(valDataset.inputs); i += 32 {
        batchEnd := min(i+32, len(valDataset.inputs))
        batchInputs := valDataset.inputs[i:batchEnd]
        batchTargets := valDataset.targets[i:batchEnd]
        
        // Convert to tensors
        inputTensor := intsToTensor(batchInputs)
        targetTensor := intsToTensor(batchTargets)
        
        inputNode := gorgonia.NewMatrix(model.graph, tensor.Int,
            gorgonia.WithShape(len(batchInputs), len(batchInputs[0])),
            gorgonia.WithValue(inputTensor),
        )
        
        targetNode := gorgonia.NewMatrix(model.graph, tensor.Int,
            gorgonia.WithShape(len(batchTargets), len(batchTargets[0])),
            gorgonia.WithValue(targetTensor),
        )
        
        // Forward pass (no training)
        logits, _ := model.Forward(inputNode, false)
        loss, _ := gorgonia.SoftMaxCrossEntropy(logits, targetNode)
        
        vm.RunAll()
        
        lossValue := loss.Value().Data().(float32)
        totalLoss += float64(lossValue)
        numBatches++
        
        vm.Reset()
    }
    
    return totalLoss / float64(numBatches)
}
```

## ⚙️ Part 9: Advanced Training Techniques

### Mixed Precision Training

```go
type MixedPrecisionTrainer struct {
    model       *GPT
    optimizer   gorgonia.Solver
    scaler      *GradScaler
    useAMP      bool
}

type GradScaler struct {
    scale       float32
    growthRate  float32
    backoffRate float32
    growthInterval int
    stepsSinceGrowth int
}

func NewGradScaler() *GradScaler {
    return &GradScaler{
        scale:       65536.0, // 2^16
        growthRate:  2.0,
        backoffRate: 0.5,
        growthInterval: 2000,
    }
}

func (gs *GradScaler) Scale(loss *gorgonia.Node) *gorgonia.Node {
    // Scale loss for mixed precision
    scaleNode := gorgonia.NewScalar(loss.Graph(), 
        tensor.Float32, 
        gorgonia.WithValue(gs.scale))
    scaled, _ := gorgonia.Mul(loss, scaleNode)
    return scaled
}

func (gs *GradScaler) Step(optimizer gorgonia.Solver, grads gorgonia.ValueGrad) error {
    // Unscale gradients
    for _, vg := range grads {
        if grad, ok := vg.Grad().(tensor.Tensor); ok {
            data := grad.Data().([]float32)
            for i := range data {
                data[i] /= gs.scale
            }
        }
    }
    
    // Check for NaN/Inf
    hasNaN := false
    for _, vg := range grads {
        if grad, ok := vg.Grad().(tensor.Tensor); ok {
            data := grad.Data().([]float32)
            for _, v := range data {
                if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
                    hasNaN = true
                    break
                }
            }
        }
    }
    
    if hasNaN {
        // Skip update and reduce scale
        gs.scale *= gs.backoffRate
        gs.stepsSinceGrowth = 0
        return nil
    }
    
    // Perform optimizer step
    if err := optimizer.Step(grads); err != nil {
        return err
    }
    
    // Grow scale periodically
    gs.stepsSinceGrowth++
    if gs.stepsSinceGrowth >= gs.growthInterval {
        gs.scale *= gs.growthRate
        gs.stepsSinceGrowth = 0
    }
    
    return nil
}
```

### Gradient Accumulation

```go
func TrainWithGradientAccumulation(model *GPT, dataset *Dataset, 
                                   config TrainConfig, accumSteps int) error {
    vm := gorgonia.NewTapeMachine(model.graph, gorgonia.BindDualValues())
    defer vm.Close()
    
    solver := gorgonia.NewAdamSolver(
        gorgonia.WithLearnRate(config.LearningRate),
    )
    
    // Accumulated gradients
    accGrads := make(map[*gorgonia.Node][]float32)
    
    for epoch := 0; epoch < config.NumEpochs; epoch++ {
        for i := 0; i < len(dataset.inputs); i++ {
            // Forward pass
            inputNode := createInputNode(dataset.inputs[i])
            targetNode := createTargetNode(dataset.targets[i])
            
            logits, _ := model.Forward(inputNode, true)
            loss, _ := gorgonia.SoftMaxCrossEntropy(logits, targetNode)
            
            // Scale loss by accumulation steps
            scaledLoss := gorgonia.Must(gorgonia.Div(loss, 
                gorgonia.NewScalar(model.graph, tensor.Float32, 
                gorgonia.WithValue(float32(accumSteps)))))
            
            // Backward pass
            gorgonia.Grad(scaledLoss, model.graph.AllNodes()...)
            vm.RunAll()
            
            // Accumulate gradients
            for _, node := range model.graph.AllNodes() {
                if node.Deriv() != nil {
                    grad := node.Deriv().Value().Data().([]float32)
                    if accGrads[node] == nil {
                        accGrads[node] = make([]float32, len(grad))
                    }
                    for j := range grad {
                        accGrads[node][j] += grad[j]
                    }
                }
            }
            
            vm.Reset()
            
            // Update weights every accumSteps
            if (i+1)%accumSteps == 0 {
                // Apply accumulated gradients
                for node, grad := range accGrads {
                    if node.Deriv() != nil {
                        t := tensor.New(
                            tensor.WithBacking(grad),
                            tensor.WithShape(node.Shape()...),
                        )
                        gorgonia.Let(node.Deriv(), t)
                    }
                }
                
                // Optimizer step
                solver.Step(gorgonia.NodesToValueGrads(model.graph.AllNodes()))
                
                // Clear accumulated gradients
                for node := range accGrads {
                    accGrads[node] = nil
                }
            }
        }
    }
    
    return nil
}
```

### Learning Rate Scheduling

```go
type LRScheduler interface {
    GetLR(step int) float64
}

type CosineScheduler struct {
    maxLR       float64
    minLR       float64
    warmupSteps int
    totalSteps  int
}

func (cs *CosineScheduler) GetLR(step int) float64 {
    if step < cs.warmupSteps {
        return cs.maxLR * float64(step) / float64(cs.warmupSteps)
    }
    
    progress := float64(step-cs.warmupSteps) / float64(cs.totalSteps-cs.warmupSteps)
    return cs.minLR + 0.5*(cs.maxLR-cs.minLR)*(1+math.Cos(math.Pi*progress))
}

type OneCycleScheduler struct {
    maxLR      float64
    totalSteps int
    pctStart   float64
}

func (ocs *OneCycleScheduler) GetLR(step int) float64 {
    pctStep := float64(step) / float64(ocs.totalSteps)
    
    if pctStep < ocs.pctStart {
        // Increasing phase
        return ocs.maxLR * pctStep / ocs.pctStart
    } else {
        // Decreasing phase
        progress := (pctStep - ocs.pctStart) / (1.0 - ocs.pctStart)
        return ocs.maxLR * (1.0 - progress)
    }
}
```

## 🎯 Part 10: Practical Training Example

### Complete Training Script

```go
package main

import (
    "fmt"
    "log"
    "time"
)

func main() {
    fmt.Println("=== Transformer Training Pipeline ===")
    
    // Step 1: Initialize
    startTime := time.Now()
    fmt.Println("\n[1/8] Initializing CUDA...")
    
    dev, err := cuda.Init(0)
    if err != nil {
        log.Fatal("CUDA init failed:", err)
    }
    defer dev.Close()
    fmt.Printf("✓ Using GPU: %s (%.2f GB)\n", dev.Name(), 
        float64(dev.TotalMem())/(1024*1024*1024))
    
    // Step 2: Load Data
    fmt.Println("\n[2/8] Loading dataset...")
    text := loadTextFromFile("data/training_corpus.txt")
    fmt.Printf("✓ Loaded %d characters\n", len(text))
    
    // Step 3: Initialize Tokenizer
    fmt.Println("\n[3/8] Initializing tokenizer...")
    tokenizer := NewBPETokenizer("tokenizer.json")
    fmt.Printf("✓ Vocabulary size: %d\n", tokenizer.vocabSize)
    
    // Step 4: Prepare Dataset
    fmt.Println("\n[4/8] Preparing dataset...")
    trainDataset := PrepareDataset(text[:int(0.9*float64(len(text)))], 
        tokenizer, 512, 512)
    valDataset := PrepareDataset(text[int(0.9*float64(len(text))):], 
        tokenizer, 512, 512)
    fmt.Printf("✓ Train samples: %d\n", len(trainDataset.inputs))
    fmt.Printf("✓ Val samples: %d\n", len(valDataset.inputs))
    
    // Step 5: Create Model
    fmt.Println("\n[5/8] Creating model...")
    config := TransformerConfig{
        VocabSize:    tokenizer.vocabSize,
        EmbedDim:     768,
        NumHeads:     12,
        NumLayers:    12,
        ContextLen:   512,
        DropoutRate:  0.1,
        FFNHiddenDim: 3072,
    }
    
    model := NewGPT(config)
    numParams := countParameters(model)
    fmt.Printf("✓ Model created with %d parameters (%.2f M)\n", 
        numParams, float64(numParams)/1e6)
    
    // Step 6: Configure Training
    fmt.Println("\n[6/8] Configuring training...")
    trainConfig := TrainConfig{
        LearningRate: 3e-4,
        WeightDecay:  0.1,
        NumEpochs:    10,
        BatchSize:    16,
        WarmupSteps:  500,
        MaxLR:        3e-4,
        MinLR:        3e-5,
        GradientClip: 1.0,
        EvalFreq:     100,
        SaveFreq:     1000,
    }
    fmt.Println("✓ Configuration set")
    
    // Step 7: Train
    fmt.Println("\n[7/8] Starting training...")
    fmt.Println("=" * 60)
    
    metrics := &TrainingMetrics{}
    
    if err := TrainModelWithMetrics(model, trainDataset, valDataset, 
        trainConfig, metrics); err != nil {
        log.Fatal("Training failed:", err)
    }
    
    fmt.Println("=" * 60)
    fmt.Println("✓ Training completed")
    
    // Step 8: Save and Evaluate
    fmt.Println("\n[8/8] Saving model and generating samples...")
    
    if err := SaveModel(model, "final_model.bin"); err != nil {
        log.Fatal("Save failed:", err)
    }
    fmt.Println("✓ Model saved")
    
    // Generate sample text
    prompt := "Once upon a time"
    promptTokens := tokenizer.Encode(prompt)
    generated := Generate(model, promptTokens, 100, 0.8, 50)
    generatedText := tokenizer.Decode(generated)
    
    fmt.Println("\n--- Generated Sample ---")
    fmt.Println(generatedText)
    fmt.Println("------------------------")
    
    // Save metrics
    metrics.SaveToFile("training_metrics.json")
    
    elapsed := time.Since(startTime)
    fmt.Printf("\n✓ Total time: %s\n", elapsed)
    fmt.Println("\nTraining pipeline completed successfully!")
}

func countParameters(model *GPT) int {
    count := 0
    for _, node := range model.graph.AllNodes() {
        if node.Value() != nil {
            count += node.Shape().TotalSize()
        }
    }
    return count
}

func loadTextFromFile(path string) string {
    data, err := os.ReadFile(path)
    if err != nil {
        log.Fatal(err)
    }
    return string(data)
}

func TrainModelWithMetrics(model *GPT, trainDataset, valDataset *Dataset,
                          config TrainConfig, metrics *TrainingMetrics) error {
    vm := gorgonia.NewTapeMachine(model.graph, gorgonia.BindDualValues())
    defer vm.Close()
    
    solver := gorgonia.NewAdamSolver(
        gorgonia.WithLearnRate(config.LearningRate),
        gorgonia.WithBatchSize(float64(config.BatchSize)),
        gorgonia.WithClip(config.GradientClip),
    )
    
    scheduler := &CosineScheduler{
        maxLR:       config.MaxLR,
        minLR:       config.MinLR,
        warmupSteps: config.WarmupSteps,
        totalSteps:  config.NumEpochs * len(trainDataset.inputs) / config.BatchSize,
    }
    
    globalStep := 0
    tokensSeen := 0
    
    for epoch := 0; epoch < config.NumEpochs; epoch++ {
        epochLoss := 0.0
        numBatches := 0
        
        for i := 0; i < len(trainDataset.inputs); i += config.BatchSize {
            batchEnd := min(i+config.BatchSize, len(trainDataset.inputs))
            
            // Training step (simplified - full implementation above)
            batchLoss := trainStep(model, vm, solver, 
                trainDataset.inputs[i:batchEnd],
                trainDataset.targets[i:batchEnd])
            
            epochLoss += batchLoss
            numBatches++
            globalStep++
            tokensSeen += config.BatchSize * len(trainDataset.inputs[0])
            
            // Update learning rate
            lr := scheduler.GetLR(globalStep)
            solver = gorgonia.NewAdamSolver(
                gorgonia.WithLearnRate(lr),
                gorgonia.WithBatchSize(float64(config.BatchSize)),
                gorgonia.WithClip(config.GradientClip),
            )
            
            // Evaluate periodically
            if globalStep%config.EvalFreq == 0 {
                valLoss := ValidateModel(model, valDataset)
                trainLoss := epochLoss / float64(numBatches)
                
                metrics.Log(epoch, trainLoss, valLoss, lr, tokensSeen)
            }
            
            // Save checkpoint
            if globalStep%config.SaveFreq == 0 {
                checkpoint := fmt.Sprintf("checkpoint_step_%d.bin", globalStep)
                SaveModel(model, checkpoint)
                fmt.Printf("Checkpoint saved: %s\n", checkpoint)
            }
        }
    }
    
    return nil
}

func trainStep(model *GPT, vm gorgonia.VM, solver gorgonia.Solver,
               inputs, targets [][]int) float64 {
    // Simplified training step
    // Full implementation combines all previous examples
    return 0.0
}
```

## 🚀 Part 11: Optimization Tips & Troubleshooting

### Memory Management

```go
// Monitor GPU memory usage
func MonitorGPUMemory(dev *cuda.Device) {
    ticker := time.NewTicker(5 * time.Second)
    go func() {
        for range ticker.C {
            free, total := dev.MemInfo()
            used := total - free
            pct := float64(used) / float64(total) * 100
            
            fmt.Printf("GPU Memory: %.2f / %.2f GB (%.1f%% used)\n",
                float64(used)/(1024*1024*1024),
                float64(total)/(1024*1024*1024),
                pct)
        }
    }()
}

// Clear GPU memory between batches if needed
func ClearGPUCache(dev *cuda.Device) {
    // Force garbage collection
    runtime.GC()
    
    // Synchronize device
    dev.Synchronize()
}
```

### Debugging NaN/Inf Issues

```go
func CheckForNaN(node *gorgonia.Node, name string) bool {
    if node.Value() == nil {
        return false
    }
    
    data := node.Value().Data()
    switch v := data.(type) {
    case []float32:
        for i, val := range v {
            if math.IsNaN(float64(val)) || math.IsInf(float64(val), 0) {
                fmt.Printf("NaN/Inf detected in %s at index %d: %v\n", name, i, val)
                return true
            }
        }
    }
    return false
}

// Add to training loop
func TrainWithNaNChecks(model *GPT, dataset *Dataset) {
    // ... training code ...
    
    // After each forward pass
    if CheckForNaN(logits, "logits") {
        log.Fatal("NaN detected in logits - stopping training")
    }
    
    // After computing loss
    if CheckForNaN(loss, "loss") {
        log.Fatal("NaN detected in loss - stopping training")
    }
}
```

### Common Issues and Solutions

**Issue 1: CUDA Out of Memory**
```go
// Solution: Reduce batch size or use gradient accumulation
config.BatchSize = 8  // Instead of 32
accumSteps := 4       // Effective batch size = 32
```

**Issue 2: Training Loss Not Decreasing**
```go
// Solutions:
// 1. Check learning rate
config.LearningRate = 1e-4  // Try lower

// 2. Verify data preprocessing
fmt.Printf("Sample input: %v\n", dataset.inputs[0])
fmt.Printf("Sample target: %v\n", dataset.targets[0])

// 3. Start with smaller model
config.NumLayers = 4   // Instead of 12
config.EmbedDim = 256  // Instead of 768
```

**Issue 3: Gradient Explosion**
```go
// Solution: Clip gradients more aggressively
config.GradientClip = 0.5  // Instead of 1.0

// Add gradient norm monitoring
func ComputeGradNorm(grads gorgonia.ValueGrad) float64 {
    norm := 0.0
    for _, vg := range grads {
        if grad, ok := vg.Grad().(tensor.Tensor); ok {
            data := grad.Data().([]float32)
            for _, v := range data {
                norm += float64(v * v)
            }
        }
    }
    return math.Sqrt(norm)
}
```

## 📊 Part 12: Benchmarking & Performance

### Measure Training Speed

```go
type PerformanceMetrics struct {
    TokensPerSecond float64
    BatchesPerSecond float64
    TimePerEpoch    time.Duration
    GPUUtilization  float64
}

func BenchmarkTraining(model *GPT, dataset *Dataset, numSteps int) PerformanceMetrics {
    startTime := time.Now()
    totalTokens := 0
    
    vm := gorgonia.NewTapeMachine(model.graph)
    defer vm.Close()
    
    for i := 0; i < numSteps; i++ {
        batch := dataset.inputs[i]
        totalTokens += len(batch) * len(batch[0])
        
        // Forward pass
        inputNode := createInputNode(batch)
        model.Forward(inputNode, true)
        vm.RunAll()
        vm.Reset()
    }
    
    elapsed := time.Since(startTime)
    tokensPerSec := float64(totalTokens) / elapsed.Seconds()
    batchesPerSec := float64(numSteps) / elapsed.Seconds()
    
    return PerformanceMetrics{
        TokensPerSecond:  tokensPerSec,
        BatchesPerSecond: batchesPerSec,
        TimePerEpoch:     elapsed,
    }
}
```

### Compare Configurations

```go
func CompareConfigurations() {
    configs := []TransformerConfig{
        {VocabSize: 10000, EmbedDim: 256, NumHeads: 4, NumLayers: 4},
        {VocabSize: 10000, EmbedDim: 512, NumHeads: 8, NumLayers: 6},
        {VocabSize: 10000, EmbedDim: 768, NumHeads: 12, NumLayers: 12},
    }
    
    for i, cfg := range configs {
        model := NewGPT(cfg)
        params := countParameters(model)
        
        // Benchmark
        perf := BenchmarkTraining(model, dataset, 100)
        
        fmt.Printf("\nConfig %d:\n", i+1)
        fmt.Printf("  Parameters: %.2f M\n", float64(params)/1e6)
        fmt.Printf("  Speed: %.2f tokens/sec\n", perf.TokensPerSecond)
        fmt.Printf("  Memory: %.2f GB\n", estimateMemory(model))
    }
}
```

## 🎓 Part 13: Best Practices Summary

### DO ✅

1. **Start Small**
```go
// Begin with tiny model
smallConfig := TransformerConfig{
    VocabSize:   1000,
    EmbedDim:    128,
    NumHeads:    4,
    NumLayers:   2,
    ContextLen:  128,
    DropoutRate: 0.1,
}
```

2. **Monitor Everything**
```go
// Log all metrics
metrics := &TrainingMetrics{}
// Check GPU memory
MonitorGPUMemory(dev)
// Watch for NaN
CheckForNaN(loss, "loss")
```

3. **Save Frequently**
```go
// Save every N steps
if step%1000 == 0 {
    SaveModel(model, fmt.Sprintf("checkpoint_%d.bin", step))
}
```

4. **Validate Regularly**
```go
// Evaluate on validation set
if step%100 == 0 {
    valLoss := ValidateModel(model, valDataset)
    fmt.Printf("Validation Loss: %.4f\n", valLoss)
}
```

### DON'T ❌

1. **Don't Skip Warmup**
```go
// BAD: No warmup
config.WarmupSteps = 0

// GOOD: Gradual warmup
config.WarmupSteps = 1000
```

2. **Don't Use Fixed LR**
```go
// BAD: Fixed learning rate
lr := 3e-4

// GOOD: Scheduled learning rate
scheduler := CosineScheduler{...}
lr := scheduler.GetLR(step)
```

3. **Don't Ignore Data Quality**
```go
// GOOD: Clean and validate data
text = CleanText(text)
ValidateTokens(tokens)
```

## 🔍 Part 14: Example Workflow

### Day-by-Day Training Plan

**Week 1: Setup & Testing**
```bash
# Day 1-2: Environment setup
go get required packages
test CUDA installation
verify GPU access

# Day 3-4: Data preparation
collect training data
implement tokenizer
create datasets

# Day 5-7: Small model test
train 2-layer, 128-dim model
verify training loop works
debug any issues
```

**Week 2-3: Scaling Up**
```bash
# Day 8-14: Medium model
train 6-layer, 512-dim model
tune hyperparameters
monitor metrics

# Day 15-21: Optimization
implement mixed precision
add gradient accumulation
optimize data loading
```

**Week 4+: Production Training**
```bash
# Day 22+: Full model
train 12-layer, 768-dim model
run for many epochs
evaluate on test set
fine-tune for specific task
```

## 📚 Part 15: Resources & References

### Essential Reading

1. **Transformer Architecture**
   - "Attention is All You Need" (Vaswani et al., 2017)
   - https://arxiv.org/abs/1706.03762

2. **GPT Papers**
   - GPT-1: "Improving Language Understanding"
   - GPT-2: "Language Models are Unsupervised Multitask Learners"
   - GPT-3: "Language Models are Few-Shot Learners"

3. **Training Techniques**
   - "Accurate, Large Minibatch SGD" (Goyal et al.)
   - "Mixed Precision Training" (Micikevicius et al.)

### Go Libraries Documentation

```bash
# Gorgonia
https://gorgonia.org/
https://github.com/gorgonia/gorgonia

# cudago
https://github.com/InternatBlackhole/cudago

# autograd
https://github.com/itsubaki/autograd

# Additional tools
gonum.org/v1/gonum
```

### Community Resources

- Gorgonia Slack/Discord
- r/golang machine learning discussions
- Stack Overflow: [gorgonia] tag

## 🎯 Conclusion

You now have a complete guide to building and training Transformers in Go! The key steps are:

1. ✅ Set up CUDA environment with cudago
2. ✅ Build model architecture with Gorgonia
3. ✅ Prepare and tokenize datasets
4. ✅ Implement training loop with automatic differentiation
5. ✅ Monitor metrics and optimize performance
6. ✅ Save checkpoints and generate text

Remember: **Start small, validate often, scale gradually!**

Good luck with your Transformer training! 🚀