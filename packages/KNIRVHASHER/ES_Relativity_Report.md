# ES Relativity Report: Evolution Strategies at Scale vs. KNIRVHASHER

**Paper:** *Evolution Strategies at Scale: LLM Fine-Tuning Beyond Reinforcement Learning*
**arXiv:** 2509.24372v2 | Feb 10, 2026
**Authors:** Xin Qiu, Yulu Gan, Conor F. Hayes, Qiyao Liang, Yinggan Xu, et al.

---

## 1. What the Paper Claims

The paper demonstrates for the first time that Evolution Strategies (ES) — a class of
population-based, zeroth-order optimization algorithms — can fine-tune LLMs at
**billion-parameter scale without dimensionality reduction**, outperforming RL (GRPO,
PPO, DrGRPO) across multiple benchmarks.

**The core ES algorithm (Algorithm 1):**

```
Given: pretrained parameters θ₀, reward function R(·), population N, noise σ, LR α
For each iteration t:
  For each n in N:
    Sample noise: εₙ ~ N(0, I)
    Evaluate: Rₙ = R(θ_{t-1} + σ·εₙ)
  Normalize rewards Rₙ (z-score)
  Update: θ_t ← θ_{t-1} + α · (1/N) · Σ Rₙεₙ
```

**Six headline properties where ES outperforms RL:**
1. Only needs **response-level rewards** — no per-token credit assignment
2. Finds good solutions with **tiny population size** (N=30, even at billion-parameter scale)
3. **More robust** than RL across different base LLMs
4. Does **not reward-hack** — optimizes a distribution, not a single solution
5. Produces **consistent results** across runs (15.5× lower std-dev than GRPO)
6. **Inference-only** — no gradient computation, significant GPU memory savings

---

## 2. KNIRVHASHER Architecture: What It Actually Does

KNIRVHASHER is a novel inference system built around hash-based neural computation and
ASIC-accelerated nonce mining. Its "model" is not a transformer — it is a **hash network
whose parameters are seeds** (32-byte arrays), and whose forward pass is SHA-256, not
matrix multiplication.

### 2.1 HashNetwork (`pkg/hashing/neural/`)

A 3-layer neural network where each neuron's "weight" is a 32-byte seed:

```
Forward pass: output = SHA-256(input || seed) → normalized float [0, 1]
```

Each layer concatenates the input with a seed and hashes it. There are no floating-point
weight matrices. The network's "parameters" are entirely defined by `Seeds1`, `Seeds2`,
`SeedsOut`.

### 2.2 RecursiveEngine (`pkg/hashing/inference/recursive.go`)

- Runs **21 temporal passes** over the same input
- Each pass: applies input jitter → routes through HashNetwork → gets prediction + confidence
- Aggregates via **majority vote consensus** across all 21 passes
- Optional **seed rotation** per pass (XOR-based deterministic rotation)
- Produces `ConsensusResult` with confidence, vote count, class distribution

### 2.3 JitterEngine (`pkg/hashing/jitter/jitter_engine.go`)

- Executes the **21-pass temporal loop** for dynamic associative hashing
- Each pass: Double-SHA256 → extract lookup key → flash search for jitter vector →
  XOR jitter into Bitcoin header
- `HuntGoldenNonce`: evaluates candidate nonces, tracking best alignment to `targetTokenID`
- `ComputePassReward`: bit-matching reward per pass (leading zeros in XOR output)
- Alignment ≥ 0.95 = "found" — maps hash output to token space

### 2.4 EvolutionaryHarness (`pipeline/3_DATA_TRAINER/pkg/training/evolutionary.go`)

The actual training loop — where optimization lives:

- **Population of seeds** (`SeedPopulation`, default `GroupSize=128`)
- `EvaluatePopulationBatch`: builds Bitcoin headers from `TrainingRecord.FeatureVector +
  nonce`, runs `Execute21PassLoopBatch`, extracts golden nonces
- **Multi-component reward**: alignment + stability + format + exact-match bonus
- `CalculateBitMatchAdvantage`: z-score normalization of Hamming bit-match scores
- `SelectAndMutate`: elite selection (top 25%) + `BitcoinAwareMutate` (bit-flip)
- Dynamic Difficulty Scaling (DDS): progressive target mask from 8→32 bits across epochs

### 2.5 EvoGRPO (`pipeline/3_DATA_TRAINER/internal/evo_grpo/evo_grpo.go`)

Placeholder implementation of Evolutionary GRPO — the intended fusion of evolutionary
search with GRPO-style advantage weighting. Population → fitness eval → selection →
crossover/mutation loop over 100 generations. Currently uses stub fitness values; real
implementation deferred.

---

## 3. Structural Similarities to ES

Despite operating in completely different problem domains, KNIRVHASHER's training pipeline
has deep structural parallels to the ES algorithm.

### 3.1 Parameter Space = Seed Space

**ES:** perturbs floating-point weight tensors θ + σ·ε
**KNIRVHASHER:** mutates 32-byte seed arrays via bit-flip

Both search a parameter space through perturbation and evaluate fitness at perturbed
candidates. KNIRVHASHER's seed space is 256-dimensional (32 bytes × 8 bits). The
mathematical structure is identical at the abstract level.

### 3.2 Population-Based, Gradient-Free Reward Optimization

**ES:** N perturbed models evaluated, rewards aggregated without gradients
**KNIRVHASHER:** population of `GroupSize` seeds evaluated, rewards from SHA-256 hash
alignment

Both are **zeroth-order**: no gradient is ever computed. Both need only a scalar reward
signal per candidate. This is the defining property of ES vs. RL, and KNIRVHASHER
already satisfies it completely.

### 3.3 Z-Score Advantage Normalization

`CalculateBitMatchAdvantage` in `evolutionary.go` computes:

```go
advantage = (bitScore - mean) / stdDev
```

This is **byte-for-byte identical** to ES's reward normalization step (Algorithm 1,
line 6). The paper notes this is critical for training stability across tasks.
KNIRVHASHER already implements this correctly.

### 3.4 The 21-Pass Loop as Reward Smoothing

ES's key insight is that Gaussian noise in parameter space smooths jagged reward
landscapes. The JitterEngine's 21-pass temporal loop achieves an analogous effect
domain-specifically: averaging hash computations across 21 XOR-jittered states smooths
the hash output distribution over time. Both mechanisms address the same problem —
making discontinuous reward signals tractable for optimization.

### 3.5 Inference-Only Training

**ES paper (Section 5):** *"ES is an inference-only fine-tuning mechanism, where the
model weights are never differentiated, only evaluated."*
**KNIRVHASHER:** Already 100% inference-only. The entire training loop in `3_DATA_TRAINER`
consists of forward passes through the jitter engine. No backpropagation exists anywhere
in the codebase.

KNIRVHASHER's compute model is already aligned with ES's key engineering advantage. The
paper notes this enables specialized inference kernels for large batches and parameter
perturbations — directly applicable to the CUDA path (`pkg/hashing/methods/cuda/`).

### 3.6 The Primary Structural Gap: GA Elitism vs. ES Weighted Update

**ES:** θ_t ← θ_{t-1} + α · (1/N) · Σ Rₙεₙ
  — weighted sum of *all* perturbations, proportional to reward

**KNIRVHASHER:** Hard-select top 25% elites, discard the rest, mutate survivors
  — GA-style

This is the primary structural divergence. ES uses a soft, reward-weighted update across
the full population. This gap is the highest-value item to close (see Section 5).

---

## 4. Key Differences and Friction Points

### 4.1 SHA-256 Avalanche Effect

The fundamental challenge for applying ES in KNIRVHASHER's domain: SHA-256 has a
near-perfect avalanche effect. A single bit change in the seed produces ~50% bit-flip
in the output — completely decorrelated. This means:

- Gaussian perturbations in seed space do **not** produce smooth variation in hash output
  space
- The reward landscape (bit-matching alignment) is effectively binary near any solution
- ES's theoretical guarantee (Gaussian smoothing of the reward landscape) **does not
  transfer through SHA-256**

The existing `CalculateBitMatchAdvantage` already addresses this by using Hamming
distance as a continuous proxy — creating an artificial gradient where the true hash
function provides none. This is the correct architectural response.

### 4.2 Seed Space Dimensionality is Already Tiny

ES's main contribution is scaling to *billions of parameters* with N=30. KNIRVHASHER's
seed space is 256 bits (32 bytes). The curse of dimensionality ES overcomes is
irrelevant here. The current GA approach is mechanically appropriate for a 256-bit
search space. ES would add principled variance reduction but not change the fundamental
convergence character.

### 4.3 Mutation Semantics: Gaussian vs. Bit-Flip

ES noise: ε ~ N(0, I) — continuous displacement scaled by σ, unbiased gradient estimate.
`BitcoinAwareMutate`: flip k random bits where k ∝ 1/advantage — local random walk,
no theoretical variance guarantee.

A natural ES adaptation for KNIRVHASHER's discrete seed space: **temperature-scaled
byte perturbations** (Gaussian on the integer interpretation of each byte, clipped to
[0, 255]) rather than pure bit-flip. This preserves ES's variance reduction while
respecting byte boundaries.

### 4.4 `StaticMidstate` vs. Principled σ Annealing

`EvolutionaryHarness.StaticMidstate` freezes jitter for early generations to stabilize
rewards before full difficulty engages. This is analogous to ES's σ schedule, but it's
a binary flag rather than a continuous noise decay. Replacing it with proper σ annealing
tied to epoch would unify these mechanisms.

---

## 5. Feasibility: Applying ES Directly to KNIRVHASHER

### 5.1 Where ES Applies

ES maps onto KNIRVHASHER's **training pipeline** (`3_DATA_TRAINER`), specifically
`EvolutionaryHarness`. The `EvoGRPO` struct was already designed with this fusion in
mind but currently holds placeholder logic.

### 5.2 Proposed ES Update for KNIRVHASHER Seeds

```
Current (GA):
  1. Evaluate population of N seeds → rewards
  2. Sort by reward, keep top 25% elites
  3. Mutate elites (bit-flip)
  4. Refill with random seeds

Proposed (ES-style):
  1. Sample noise εₙ ~ N(0, σ²I) for each of N candidates around a base seed
  2. Evaluate perturbed seeds → rewards Rₙ
  3. z-score normalize rewards (CalculateBitMatchAdvantage — already exists)
  4. Update base seed: seed_t ← seed_{t-1} + α · (1/N) · Σ Rₙεₙ
  5. Clip to [0, 255]
```

The normalization step already exists. The missing piece is the weighted sum update
(step 4) replacing hard elitism.

### 5.3 Variance Reduction via Mirrored Sampling

For each noise vector ε, also evaluate -ε — halving variance at no extra cost:

```go
// Current: N forward passes with random perturbations
// Proposed: N/2 (seed + ε) + N/2 (seed - ε) evaluations
// Reward estimate: (1/N) Σ [R(seed+ε) - R(seed-ε)] · ε
```

This pairs naturally with the existing CUDA batch execution in `EvaluatePopulationBatch`
— just needs paired header generation. Low-friction to implement.

### 5.4 Unify DDS with σ Annealing

KNIRVHASHER's Dynamic Difficulty Scaling (DDS) already progressively tightens the target
mask across epochs — structurally analogous to ES's σ decay schedule. Unified:

- **Low epochs (easy difficulty):** large σ — coarse Gaussian noise, broad exploration
- **High epochs (hard difficulty):** small σ — fine-grained noise, local refinement

Replaces `StaticMidstate` binary flag with a principled σ schedule tied to `eh.Epoch`.

### 5.5 Population Size: N=30 Is Enough

ES demonstrates N=30 suffices for billion-parameter LLMs. For a 256-bit seed space,
N=16–32 would converge. Reducing `GroupSize` from 128 to 30–64 with ES's weighted
update (vs. GA's hard elitism) gives equivalent or better convergence at meaningfully
lower CUDA batch cost.

### 5.6 ES Applied to HashNetwork Seeds Directly

Most ambitious application: treat all of HashNetwork's seeds (Seeds1, Seeds2, SeedsOut)
as the full parameter vector θ and apply ES to optimize the network against a dataset of
`(input, target_token)` pairs. The HashNetwork is far smaller than any LLM (hundreds of
32-byte seeds), so N=30 with simultaneous Gaussian perturbations across all seeds is
entirely tractable — and would provide a principled training alternative to the current
mining-based approach.

---

## 6. What ES Would NOT Help With

### 6.1 SHA-256 Avalanche is Not a Landscape Problem

ES smooths reward landscapes via parameter-space noise injection. SHA-256's avalanche
effect is intentional cryptographic hardness — it cannot be smoothed by any perturbation
strategy. The Hamming distance proxy in `CalculateBitMatchAdvantage` is the correct
response; ES would improve how the optimizer uses that proxy, not eliminate the need for
it.

### 6.2 The Inference Layer is Not the Training Layer

The JitterEngine's 21-pass loop is an inference-time mechanism. ES applies to training.
ES does not improve hash output quality at inference time — it only improves how the seed
database is built during training. Do not conflate these two layers.

### 6.3 Behavioral Alignment Properties Don't Transfer

ES's advantages around reward hacking resistance and KL divergence control address LLM
behavioral alignment problems — producing plausible-but-wrong outputs, gaming conciseness
metrics, etc. These are meaningless against KNIRVHASHER's deterministic SHA-256 reward
function. Hash alignment can't be gamed semantically. These ES properties are not
applicable.

---

## 7. Verdict Table

| ES Concept | KNIRVHASHER Status | Applicability |
|---|---|---|
| Population-based reward optimization | Already implemented (EvolutionaryHarness) | Native — structural match |
| Gradient-free, inference-only | Already satisfied | Native — matches current design |
| Z-score reward normalization | Already implemented (CalculateBitMatchAdvantage) | Native — already present |
| Small population (N=30) | Currently N=128 | Feasible — reduce with ES update |
| Gaussian noise perturbation | Currently bit-flip mutation | Feasible — swap update rule |
| Weighted update Σ Rₙεₙ | Currently hard GA elitism | Feasible — highest-value change |
| Mirrored/antithetic sampling | Not implemented | Feasible — low-effort variance reduction |
| σ annealing / noise schedule | Partial (DDS + StaticMidstate) | Feasible — unify with DDS |
| Reward hacking resistance | N/A (deterministic hash reward) | Not applicable |
| KL divergence control | N/A | Not applicable |
| Landscape smoothing | Partial (Hamming proxy) | Partial — avalanche still limits this |
| LLM behavioral alignment | Not a target | Not applicable |

---

## 8. Priority Recommendations

**Implement (high value, low effort):**

1. **Replace GA elitism with ES weighted update** in `EvolutionaryHarness.SelectAndMutate`
   — principled gradient estimation, directly improves convergence consistency.

2. **Add mirrored sampling** to `EvaluatePopulationBatch`
   — 50% variance reduction for free; pairs with existing CUDA batch path.

3. **Unify DDS with σ annealing**
   — replace `StaticMidstate` binary flag with a σ schedule tied to `eh.Epoch`.

4. **Reduce population size from 128 → 30–64**
   — after the weighted update is in place; better CUDA occupancy, same convergence.

5. **Implement `EvoGRPO` properly using the ES update rule as its base**
   — the struct exists for exactly this purpose; fill it in with Algorithm 1.

**Skip:**

- Applying ES to the JitterEngine (inference-time, wrong layer)
- Expecting ES to overcome SHA-256 avalanche (it cannot)
- Importing LLM behavioral alignment concepts (wrong domain)

---

## 9. Summary

The paper and KNIRVHASHER are more structurally aligned than they appear. KNIRVHASHER
already implements the most important ES principles: gradient-free population search,
response-level reward only, z-score normalization, and inference-only compute. The
primary gap is the **update rule** — KNIRVHASHER uses GA-style hard elitism where ES
uses a smooth, reward-weighted perturbation sum. Closing that gap is straightforward,
would measurably improve convergence stability across runs, and would give `EvoGRPO` a
proper mathematical foundation rather than placeholder logic.

The ES paper's behavioral results (reward hacking, KL divergence, LLM generalization)
do not transfer to KNIRVHASHER's domain. The hash-alignment task has a deterministic
reward function with no semantic ambiguity. The paper's mechanistic claims — small
populations, inference-only, z-score normalization, smooth Gaussian exploration — are
directly applicable and worth incorporating in the training pipeline.
