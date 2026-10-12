# FoX Upgrade Scope

## Objective

Upgrade the Linear Attention mechanism in `UnifiedHasherEngine.attnForward` to **FoX (Formula X)** — combining sharp softmax-style attention weights with a scalar decay chain for temporal forgetting.

## FoX Equation

```
sum_{j=1}^{t} exp(q_t^T k_j) * (prod_{s=j+1}^{t} alpha_s) * v_j
```

Where:
- `q_t^T k_j` — dot product similarity between query at position t and key at position j
- `exp(q_t^T k_j)` — sharp attention weight (Softmax-style, no normalization)
- `prod_{s=j+1}^{t} alpha_s` — scalar decay chain from position j+1 to t
- `alpha_s` — scalar forget gate in [0, 1] for each position s
- `v_j` — value vector at position j

## Current State (Post Linear Attention Patch)

The `attnForward` in `unified_engine.go` now implements:

```
sum_{j=1}^{t} (q_t^T k_j) * v_j
```

This is Linear Attention (Equation 3) — proper cross-token sequence mixing with causal masking, but without:
1. Exponential sharpening (`exp`)
2. Temporal decay (`prod alpha_s`)

## What Needs to Change

### 1. Decay Parameter Storage

**File:** `pkg/hashing/transformer/unified_engine.go`

**Current:** `TransformerLayerSeeds` stores QuerySeeds, KeySeeds, ValueSeeds, OutputSeeds, FFNSeeds.

**Change:** Add `DecaySeeds [][32]byte` to `TransformerLayerSeeds`:

```go
type TransformerLayerSeeds struct {
    QuerySeeds  [][][32]byte
    KeySeeds    [][][32]byte
    ValueSeeds  [][][32]byte
    OutputSeeds [][][32]byte
    FFNSeeds    [][][32]byte
    DecaySeeds  [][32]byte  // NEW: one scalar decay per position
}
```

The `DecaySeeds` array has length `ContextLen` (2048 by default). Each seed produces a scalar `alpha_s` in [0, 1] via `SeedToFloat`.

### 2. Decay Computation

**File:** `pkg/hashing/transformer/unified_engine.go`

**New function:** `computeDecayChain(startPos, endPos int, decaySeeds [][32]byte) float32`

```go
func computeDecayChain(startPos, endPos int, decaySeeds [][32]byte) float32 {
    if startPos >= endPos || len(decaySeeds) == 0 {
        return 1.0
    }
    var product float32 = 1.0
    for s := startPos; s < endPos; s++ {
        if s < len(decaySeeds) {
            alpha := SeedToFloat(decaySeeds[s])
            product *= alpha
        }
    }
    return product
}
```

### 3. Modified attnForward

**File:** `pkg/hashing/transformer/unified_engine.go`

Replace the current Linear Attention with FoX:

```go
func (e *UnifiedHasherEngine) attnForward(hidden [][]float32, layer TransformerLayerSeeds, router *HardwareRouter) [][]float32 {
    seqLen := len(hidden)
    dim := e.config.EmbedDim
    numHeads := e.config.NumHeads
    out := make([][]float32, seqLen)
    for i := range out {
        out[i] = make([]float32, dim)
    }
    for i := 0; i < seqLen; i++ {
        for h := 0; h < numHeads; h++ {
            q := ProjectSeeds(hidden[i], layer.QuerySeeds[h], e.config.Activation)
            for j := 0; j <= i; j++ {
                k := ProjectSeeds(hidden[j], layer.KeySeeds[h], e.config.Activation)
                v := ProjectSeeds(hidden[j], layer.ValueSeeds[h], e.config.Activation)
                match := dotProduct(q, k)
                attentionWeight := float32(math.Exp(float64(match)))
                decay := computeDecayChain(j+1, i+1, layer.DecaySeeds)
                for d := 0; d < dim && d < len(v); d++ {
                    out[i][d] += attentionWeight * decay * v[d] / float32(numHeads)
                }
            }
        }
        proj := ProjectSeeds2D(out[i], layer.OutputSeeds, e.config.Activation)
        copy(out[i], proj)
    }
    return out
}
```

### 4. Seed Generation

**File:** `pkg/hashing/transformer/unified_engine.go`

**Function:** `BuildDefaultSeedStore` and `buildDefaultLayerSeeds`

**Change:** Initialize `DecaySeeds` for each layer:

```go
func buildDefaultLayerSeeds(cfg *UnifiedConfig) TransformerLayerSeeds {
    layer := TransformerLayerSeeds{
        QuerySeeds:  make([][][32]byte, cfg.NumHeads),
        KeySeeds:    make([][][32]byte, cfg.NumHeads),
        ValueSeeds:  make([][][32]byte, cfg.NumHeads),
        OutputSeeds: make([][][32]byte, cfg.EmbedDim),
        FFNSeeds:    make([][][32]byte, cfg.FFNHiddenDim),
        DecaySeeds:  make([][32]byte, cfg.ContextLen),  // NEW
    }
    // ... existing seed generation ...
    
    for s := 0; s < cfg.ContextLen; s++ {
        rand.Read(layer.DecaySeeds[s][:])
    }
    
    return layer
}
```

### 5. Legacy HasherTransformer Update

**File:** `pkg/hashing/transformer/gpt.go`

Apply the same changes to `hasherTransformerLayer` and `hasherMultiHeadAttention` for backward compatibility.

## Interaction with Existing Systems

### 21-Pass Temporal Ensemble

The FoX decay chain (`prod alpha_s`) operates **within a single forward pass** across token positions. The 21-pass recursive ensemble operates **across inference passes** with jitter and seed rotation. These are complementary:

- **FoX decay** — handles temporal forgetting within a sequence (position-level)
- **21-pass ensemble** — handles prediction robustness across inference runs (pass-level)

### Hardware Acceleration

The `exp()` operation is not natively supported by the hash-based hardware interface (`ComputeBatch`). Options:

1. **Software fallback** — compute `math.Exp` in Go (current approach for non-hardware paths)
2. **Lookup table** — precompute exp values for the [0, dim] range and use hash lookup
3. **Approximation** — use `1 + x + x*x/2` for small x (first-order Taylor)

Recommendation: Start with option 1 (software fallback) since `math.Exp` is fast for float32 and the attention loop is already in software for the dot product.

### Numerical Stability

- `exp(match)` can overflow for large dot products. Add clamping:
  ```go
  if match > 88.0 { match = 88.0 } // exp(88) ~ 1e38, near float32 max
  ```
- `prod alpha_s` can underflow to 0 for long sequences. Add minimum epsilon:
  ```go
  if product < 1e-30 { product = 1e-30 }
  ```

## Estimated Performance Impact

| Operation | Linear Attention | FoX |
|-----------|-----------------|-----|
| Dot product | 1 multiply + add per dim | Same |
| Attention weight | Raw scalar | `math.Exp` call |
| Decay | None | Loop over (i-j) positions |
| Total per head per token | O(dim * seqLen) | O(dim * seqLen^2) worst case |

The decay chain adds a nested loop: for each (query position i, key position j), we compute `prod_{s=j+1}^{i} alpha_s`. This is O(seqLen^2) per head.

**Optimization:** Precompute cumulative decay products:
```
cumDecay[t] = prod_{s=0}^{t} alpha_s
decay(j, i) = cumDecay[i] / cumDecay[j]
```
This reduces decay computation to O(1) per (i, j) pair, making FoX O(dim * seqLen^2) total — same asymptotic complexity as standard attention, but with linear memory for the cumulative array.

## Testing Requirements

1. **Unit tests** for `computeDecayChain`:
   - Identity decay (all alpha=1.0) → product = 1.0
   - Zero decay (all alpha=0.0) → product = 0.0
   - Mixed decay → verify product correctness
   - Edge cases: empty range, single element

2. **Integration tests** for FoX attention:
   - Sharp attention concentrates on relevant tokens vs Linear Attention
   - Long-range forgetting (alpha < 1.0 reduces distant token influence)
   - Causal masking still enforced

3. **Regression tests**:
   - Ensure backward compatibility with seed stores that lack DecaySeeds (default to alpha=1.0)

## Rollout Plan

1. **Phase 1:** Add DecaySeeds to types and seed generation (no behavior change)
2. **Phase 2:** Implement `computeDecayChain` and integrate into `attnForward`
3. **Phase 3:** Add tests and benchmark against Linear Attention baseline
4. **Phase 4:** Tune decay initialization (currently random; could use learned or scheduled values)

## Risk Assessment

| Risk | Severity | Mitigation |
|------|----------|------------|
| `exp()` overflow | Medium | Clamp match to [-88, 88] |
| Decay underflow | Medium | Use cumulative product with epsilon floor |
| Performance regression | High | Precompute cumulative decay array |
| Seed compatibility | Low | Default DecaySeeds to all-ones (identity decay) |
