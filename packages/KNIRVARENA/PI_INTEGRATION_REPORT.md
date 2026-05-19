# Report: Integrating Prime Intellect (PI) into ERGO

## Executive Summary
This report outlines the integration of Prime Intellect's **verifiers.v1** and **BYO Harness** architecture into the ERGO gaming ecosystem. By leveraging PI's decentralized verification and compute capabilities, ERGO can transition from local, deterministic error handling and reward sculpting to a verifiable, distributed, and high-fidelity agent training pipeline. The BYO Harness provides the bridge between ERGO's reactive game UI and PI's production RL/eval infrastructure.

---

## 1. Architectural Mapping: ERGO ↔ Prime Intellect

### 1.1 Core Concept Mapping

| ERGO Concept | Prime Intellect Concept | Implementation Detail |
|---|---|---|
| `ErrorContext` (protobuf) | `Taskset Row` (JSON-serializable mapping) | `error_context.proto` serialized to dict with `stack_trace`, `input_hash`, `runtime_env`, `severity` |
| `Skill` / `LoRAAdapterSkill` | `Tool` / `Toolset` | Registered as `vf.Toolset` with MCP tools or Python callables in `state.get_tools()` |
| `RewardWeights` (w_c, w_l, w_s) | `@vf.reward(weight=N)` | `w_c` → correctness reward, `w_l` → latency reward, `w_s` → simplicity reward |
| `Tournament` / `TournamentController` | `Rollout Group` (`tasks`/`states` in group signals) | Collection of rollouts for competing agents against the same Taskset row |
| `Verifier.evaluateProposal()` | `@vf.reward` functions | LLM-as-judge path maps directly to PI's `JudgeRubric` |
| `ScoreWeights` (0.6/0.3/0.1) | `vf.Rubric(funcs=[...], weights=[...])` | Weighted multi-reward rubric |
| `AnchorDataset` | `Taskset` with `replay_program` | LLM-free replay for gold-solution verification |
| `DVE` (Distributed Virtual Environment) | PI Sandbox (`CliAgentEnv` / `SandboxEnv`) | Decentralized execution verification across peer nodes |
| `CDE` (Cognitive Development Environment) | PI `Harness(program={"sandbox": True})` | Local WASM sandbox → PI sandboxed CLI program |
| `ConsensusMechanism` | `@vf.reward(stage="group")` best-of-n | Group-level scoring replaces local reputation voting |
| `LoRAAdapterEngine.compileAdapter()` | PI Hosted Training | Local simulation → PI distributed GPU fine-tuning |
| `SabotageType.NOISE_INJECTION` | PI parallel rollouts with perturbed inputs | Adversarial robustness testing |
| `TrainingManager` (distill/harden/prune/denoise/quantize/stress) | PI `@vf.metric` + `prime eval` | Verifiable metrics replace local heuristics |
| `SkillMintingProcess` | PI LoRA Adapter Deployment | Chain-minted skill → PI inference endpoint |

### 1.2 Data Flow: End-to-End PI-Verified Pipeline

```
Game Error Thrown
  → CognitiveEngine.processInput()
    → ErrorContextManager.handleError()
      → queryKNIRVGraph()
        → [NEW] PI Harness Rollout: Reproduction Verification
          → ErrorContext as Taskset Row
          → Sandboxed execution via PI SandboxEnv
          → vf.get_messages(state.get("completion")) → Verified Ground Truth
    → [If verified] AnchorDataset populated
      → [NEW] PI Replay Harness: Gold-Solution Validation
        → program={"fn": "replay_program"} with state.answer preset
    → CognitiveEngine.processWithAdaline() → AdalineBridge → LLM
      → [NEW] PI Group Rollout: Tournament Scoring
        → Multiple agent models compete on same Taskset
        → @vf.reward(stage="group") best-of-n selection
    → Skill compiled via LoRAAdapterTrainingPipeline
      → [NEW] PI Hosted Training: Distributed LoRA Fine-Tuning
        → TrainingDataset exported as PI Taskset
        → Adapter deployed to PI Inference
    → SkillMintingProcess.submitForMinting()
      → [NEW] PI Verifier Network signs validation proof
```

---

## 2. ErgoHarness: Full PI v1 Implementation

### 2.1 Package Structure

```
ergo_env/
├── ergo_env.py              # Environment implementation
├── ergo_tasksets.py          # Taskset loaders for error contexts
├── ergo_harness.py           # Custom BYO Harness
├── ergo_rewards.py           # Reward functions
├── ergo_tools.py             # MCP tool definitions
├── ergo_signals.py           # Metric/update/cleanup functions
├── pyproject.toml            # Package metadata + PI config
└── tasks/                    # Harbor-format task directories
    ├── memory-leak/
    │   ├── prompt.md
    │   └── solution.py
    ├── race-condition/
    └── api-timeout/
```

### 2.2 Taskset: Error Context as Task Rows

`ErrorContextManager.submitNewErrorNode()` triggers PI verification via a Taskset loader that consumes the protobuf-serialized context:

```python
import verifiers as vf
from typing import AsyncIterator


class ErgoErrorSource:
    """
    Consumes ErrorContext from KNIRVGRAPH or direct submission.
    Each row maps to one PI Rollout for reproduction verification.
    """
    def __init__(self, knirvgraph_client):
        self.client = knirvgraph_client

    async def __call__(self) -> AsyncIterator[dict]:
        # Stream error contexts from KNIRVGRAPH
        async for error_row in self.client.stream_unverified_errors():
            yield {
                "example_id": error_row["node_id"],
                "system_prompt": "You are an error reproduction agent. "
                                 "Analyze the stack trace and reproduce the error.",
                "prompt": [{
                    "role": "user",
                    "content": (
                        f"Error: {error_row['error_message']}\n"
                        f"Stack: {error_row['stack_trace']}\n"
                        f"OS: {error_row['runtime_env']['os']}\n"
                        f"Arch: {error_row['runtime_env']['arch']}\n"
                        f"Source: ```\n{error_row['source_snippet']}\n```"
                    )
                }],
                "answer": error_row.get("gold_solution", ""),
                "max_turns": 3,
                "sandbox": {
                    "language": error_row["runtime_env"].get("language", "typescript"),
                    "runtime": error_row["runtime_env"].get("runtime", "node"),
                },
                # Task-level runtime metadata for harness consumption
                "program": {
                    "env": {
                        "KNIRVGRAPH_NODE_ID": error_row["node_id"],
                        "ERROR_SEVERITY": error_row["severity"],
                    }
                },
            }


class ErgoTasksetConfig(vf.TasksetConfig):
    knirvgraph_endpoint: str = "http://localhost:8082"
    split: str = "unverified"
    min_severity: str = "medium"


def load_taskset(config: ErgoTasksetConfig) -> vf.Taskset:
    client = KNIRVGRAPHClient(config.knirvgraph_endpoint)
    source = ErgoErrorSource(client)

    return vf.Taskset(
        source=source,
        rewards=[
            error_reproduced,
            latency_score,
        ],
        toolsets=[vf.Toolset(tools=[vf.MCPTool(
            command="npx",
            args=["knirv-router", "--wasm"],
        )])],
        bindings={
            "error_reproduced.client": "objects.knirvgraph_client",
        },
        objects={"knirvgraph_client": client},
        config=config,
    )
```

### 2.3 Harness: Reproduction + Scoring

The harness owns the sandbox execution and WASM tool routing:

```python
import verifiers.v1 as vf


class ErgoHarnessConfig(vf.HarnessConfig):
    wasm_router_endpoint: str = "http://localhost:8086"
    max_reproduction_attempts: int = 3
    sandbox_timeout_seconds: int = 120


def load_harness(config: ErgoHarnessConfig) -> vf.Harness:
    return vf.Harness(
        program={
            "fn": "ergo_harness:run_harness_program",
            "sandbox": True,
            "channels": "mcp",
        },
        config=config,
    )


async def run_harness_program(task: vf.Task, state: vf.State) -> vf.State:
    """
    The main harness program:
    1. Sets up sandbox from task.sandbox config
    2. Gets MCP tools (KNIRVROUTER for WASM, KNIRVGRAPH for context)
    3. Executes agent solution in sandbox
    4. Captures execution trace for reward functions
    """
    tools = state.get_tools()

    # Setup sandbox environment from task metadata
    runtime_config = task.get("sandbox", {})
    state.runtime["language"] = runtime_config.get("language", "typescript")
    state.runtime["start_time"] = time.time()

    # Execute via MCP tool (KNIRVROUTER WASM runtime)
    if "knirvrouter" in tools:
        result = await tools["knirvrouter"](
            action="execute",
            code=state.get("completion", ""),
            context=task.get("prompt", ""),
            sandbox_language=state.runtime["language"],
        )
        state.runtime["execution_result"] = result.get("status")
        state.runtime["duration_ms"] = result.get("duration_ms", 0)
        state.runtime["output"] = result.get("output", "")
    else:
        # Fallback: sandboxed CLI execution
        state.runtime["execution_result"] = "sandbox_executed"
        state.runtime["duration_ms"] = 0

    state.runtime["end_time"] = time.time()
    return state
```

### 2.4 Reward Functions: Three-Pillar Scoring

Maps directly to ERGO's `ScoreWeights` (correctness=0.6, latency=0.3, simplicity=0.1):

```python
@vf.reward(weight=0.6)
async def error_reproduced(task, state, knirvgraph_client) -> float:
    """
    Maps to w_c (Correctness).
    Checks if the agent's solution resolves the error in the stack trace.
    Uses KNIRVGRAPH to verify against known fix patterns.
    """
    execution_status = state.runtime.get("execution_result")
    if execution_status != "success":
        return 0.0

    # Verify against KNIRVGRAPH's known fix signatures
    completion = vf.get_messages(state.get("completion") or [], role="assistant")
    solution_text = str(completion[-1].content) if completion else ""

    verification = await knirvgraph_client.verify_fix(
        node_id=task["program"]["env"]["KNIRVGRAPH_NODE_ID"],
        solution=solution_text,
    )
    return verification.score


@vf.reward(weight=0.3)
async def latency_score(task, state) -> float:
    """
    Maps to w_l (Latency).
    Faster reproductions score higher.
    Mirrors Verifier.ts logic: max(0, (5000 - duration) / 5000).
    """
    duration = state.runtime.get("duration_ms", 5000)
    return max(0.0, (5000.0 - duration) / 5000.0)


@vf.reward(weight=0.1)
async def simplicity_score(task, state) -> float:
    """
    Maps to w_s (Simplicity).
    Shorter, cleaner solutions score higher.
    Mirrors Verifier.ts: max(0, 1 - length/3000).
    """
    completion = vf.get_messages(state.get("completion") or [], role="assistant")
    solution_text = str(completion[-1].content) if completion else ""
    length = len(solution_text)
    return max(0.0, 1.0 - length / 3000.0)
```

### 2.5 Group Reward: Tournament Winner Selection

Maps ERGO's `Tournament.runEpoch()` to PI's group-stage scoring:

```python
@vf.reward(stage="group", weight=1.0)
async def best_of_n(tasks: list, states: list) -> list[float]:
    """
    Digital Red Queen Mechanic — maps to Tournament.runEpoch().
    The agent must score higher than the incumbent (default 0.8).
    Returns one score per rollout in the group.
    """
    scores = []
    for state in states:
        correctness = state.get("rewards", {}).get("error_reproduced", 0)
        latency = state.get("rewards", {}).get("latency_score", 0)
        simplicity = state.get("rewards", {}).get("simplicity_score", 0)

        # ERGO formula: Score = C*w_c + L*w_l + S*w_s
        score = (correctness * 0.6) + (latency * 0.3) + (simplicity * 0.1)

        # Weight Hijacking Defense (Threshold: 0.8)
        # If score < 0.8 * previous, shift weights to favor correctness
        incumbent = state.get("incumbent_score", 0.8)
        if score < incumbent * 0.8:
            score = correctness * 0.8 + latency * 0.15 + simplicity * 0.05

        scores.append(min(1.0, max(0.0, score)))

    return scores
```

### 2.6 Metric Functions: TrainingManager Mechanics

ERGO's TrainingManager operations (distill, harden, prune, denoise, quantize, stressTest) become verifiable PI metrics:

```python
@vf.metric
async def trajectory_compression_ratio(task, state) -> float:
    """Maps to TrainingManager.distill() — ratio of pruned steps."""
    trajectory = state.get("trajectory", [])
    if not trajectory:
        return 1.0
    # Count high-importance steps (thought length > 10, mirroring distill())
    important_steps = [s for s in trajectory if len(s.get("thought", "")) > 10]
    return len(important_steps) / len(trajectory) if trajectory else 1.0


@vf.metric
async def denoising_effectiveness(task, state) -> float:
    """Maps to TrainingManager.denoise() — adversarial char removal."""
    completion = vf.get_messages(state.get("completion") or [], role="assistant")
    text = str(completion[-1].content) if completion else ""
    # Count noise pattern replacements (0→o, 1→l)
    noise_chars = sum(1 for c in text if c in "01")
    total = len(text)
    return 1.0 - (noise_chars / total) if total > 0 else 1.0


@vf.metric
async def stress_test_diversity(task, state) -> float:
    """Maps to TrainingManager.stressTest() — robustness across variations."""
    trajectory = state.get("trajectory", [])
    unique_approaches = len(set(
        s.get("action", "") for s in trajectory if s.get("action")
    ))
    return min(1.0, unique_approaches / 5.0)  # Normalize to 0-1
```

### 2.7 Lifecycle Hooks: Tournament Phases

ERGO's tournament lifecycle (epoch start → proposal → verification → selection → reinforcement) maps to PI lifecycle hooks:

```python
@vf.setup(priority=10)
async def tournament_epoch_setup(task, state) -> None:
    """Runs before the harness program. Prepares the agent."""
    state["incumbent_score"] = 0.8
    state["epoch_start"] = time.time()
    state["agent_resources"] = {
        "compute": 100,
        "parity": 50,
        "latency": 50,
    }


@vf.update(priority=5)
async def reward_anchor_sync(task, state) -> None:
    """Runs before scoring. Syncs player-placed RewardAnchors from UI."""
    # TournamentControllerIntegration.updateRewardWeights() would push
    # weights to PI via this hook
    anchor_weights = state.runtime.get("reward_anchors", {})
    if anchor_weights:
        state["active_weights"] = anchor_weights


@vf.cleanup
async def save_tournament_result(task, state) -> None:
    """Runs after scoring. Persists to KNIRVGRAPH."""
    if state.get("error"):
        return  # Don't save errored rollouts
    await knirvgraph_client.save_rollout_result({
        "node_id": task["program"]["env"]["KNIRVGRAPH_NODE_ID"],
        "score": state.get("reward", 0),
        "duration": state.runtime.get("duration_ms", 0),
        "trajectory_hash": hash_trajectory(state.get("trajectory", [])),
    })


@vf.teardown
async def close_knirvgraph_connection():
    """Runs once at environment shutdown."""
    await knirvgraph_client.close()
```

---

## 3. Integration Points with Existing ERGO Systems

### 3.1 PI Integration with ErrorContextManager (TypeScript)

The existing `ErrorContextManager.submitNewErrorNode()` triggers a PI Rollout for verification:

```typescript
// packages/KNIRVARENA/src/core/cortex/ErrorContextManager.ts

interface PIRolloutConfig {
  taskset_id: string;
  max_attempts: number;
  sandbox_language: string;
  reward_weights: {
    correctness: number;  // w_c
    latency: number;      // w_l
    simplicity: number;   // w_s
  };
}

interface PIVerificationResult {
  reproduced: boolean;
  execution_trace: string;
  score: number;
  rewards: Record<string, number>;
  sandbox_logs: string[];
}

class ErrorContextManager {
  private piService: PIService;

  async submitNewErrorNode(errorContext: ErrorContext): Promise<SubmissionResponse> {
    const response = await super.submitNewErrorNode(errorContext);

    if (response.status === 'SUBMISSION_SUCCESS') {
      // Trigger PI BYO Harness Rollout for reproduction verification
      const verification: PIVerificationResult = await this.piService.createRollout({
        taskset_id: 'ergo-error-verification',
        rows: [{
          error_context: {
            error_message: errorContext.errorMessage,
            stack_trace: errorContext.stackTrace,
            runtime_env: {
              os: errorContext.os,
              arch: errorContext.arch,
              language: errorContext.language || 'typescript',
              runtime: errorContext.runtime || 'node',
            },
            source_snippet: errorContext.sourceSnippet,
            severity: errorContext.severity,
          },
          node_id: response.errorNodeId,
          bounty: errorContext.bountyAmount || 1,
        }],
        harness_config: {
          sandbox: true,
          max_turns: 3,
          channels: 'mcp',
        },
      });

      // If verified, store ground truth
      if (verification.reproduced || verification.score > 0.7) {
        await this.knirvgraphClient.submitVerificationProof({
          nodeId: response.errorNodeId,
          piRolloutId: verification.rolloutId,
          executionTrace: verification.execution_trace,
          verifiedScore: verification.score,
        });
      }
    }
    return response;
  }
}
```

### 3.2 PI Integration with AnchorDatasetManager

The existing `AnchorDatasetManager` (5 categories: error_resolution, combat, exploration, dialogue, crafting) gains PI-powered validation:

```typescript
// packages/KNIRVARENA/src/sensory-shell/AnchorDatasetManager.ts

class AnchorDatasetManager {
  async populateTemplate(
    templateId: string,
    errorContext: ErrorContextForAnchor,
    additionalContext?: Record<string, unknown>,
  ): Promise<PopulatedAnchorDataset | null> {
    const populated = await this._populateTemplate(templateId, errorContext, additionalContext);
    if (!populated) return null;

    // NEW: Validate populated anchor via PI Replay Harness (LLM-free)
    if (populated.entry.populatedTemplate.includes('{{resolved}}')) {
      const isValid = await this.piReplayService.validate({
        taskset_id: 'ergo-anchor-validation',
        program: 'replay_program',  // LLM-free replay
        gold_solution: populated.entry.populatedTemplate,
        reward_conditions: {
          correctness_threshold: 0.8,
        },
      });

      if (!isValid) {
        console.warn(`Anchor ${templateId} failed PI replay validation`);
        populated.confidence *= 0.5;  // Demote confidence
      }
    }

    return populated;
  }
}
```

### 3.3 PI Integration with KNIRVSERVERClient (DVE/CDE)

The existing `KNIRVSERVERClient` (DVE and CDE validation paths) gains PI as a verification backend:

```typescript
// packages/KNIRVARENA/src/services/KNIRVSERVERClient.ts

class KNIRVSERVERClient {
  async validateWithDVE(request: DVERequest): Promise<DVEResult> {
    // Try PI Sandbox first (if configured)
    if (this.piConfig?.enabled) {
      try {
        const piResult = await this.piService.sandboxValidate({
          code: request.skillCode,
          test_cases: request.testCases,
          sandbox_type: 'dve',
          tee_type: request.requiredTEEType,
        });
        return {
          success: piResult.passed,
          score: piResult.score,
          passed: piResult.passed,
          output: piResult.output,
          executionTime: piResult.execution_time_ms,
        };
      } catch (piError) {
        console.warn('PI sandbox validation failed, falling back to DVE:', piError);
      }
    }
    // Fallback to original DVE implementation
    return this._originalValidateWithDVE(request);
  }
}
```

### 3.4 PI Integration with TournamentController

Reward weights are synced bidirectionally between the UI and PI Verifier config:

```typescript
// packages/KNIRVARENA/src/engine/TournamentControllerIntegration.ts

class TournamentControllerImpl {
  async updateRewardWeights(newWeights: RewardWeights): Promise<void> {
    this.weights = { ...newWeights };

    // Sync to PI Harness config for next rollout group
    await this.piService.updateHarnessConfig({
      rewards: [
        { name: 'error_reproduced', weight: newWeights.w_c },
        { name: 'latency_score', weight: newWeights.w_l },
        { name: 'simplicity_score', weight: newWeights.w_s },
      ],
    });
  }
}
```

### 3.5 PI Integration with LoRAAdapterTrainingPipeline

The local LoRA training pipeline (Xavier init, simplified gradient descent) exports datasets to PI for distributed training:

```typescript
// packages/KNIRVARENA/src/core/knirvgraph/LoRAAdapterTrainingPipeline.ts

class LoRAAdapterTrainingPipeline {
  async trainLoRAAdapter(
    dataset: TrainingDataset,
    config: LoRATrainingConfig,
  ): Promise<LoRAAdapterSkill> {
    // For complex clusters, delegate to PI Hosted Training
    if (dataset.datasetMetrics.complexityScore > 0.7 && this.piConfig?.enabled) {
      return this.trainViaPI(dataset, config);
    }
    // Fallback to local training
    return this._localTrain(dataset, config);
  }

  private async trainViaPI(
    dataset: TrainingDataset,
    config: LoRATrainingConfig,
  ): Promise<LoRAAdapterSkill> {
    // 1. Export dataset as PI Taskset
    const piRun = await this.piService.startHostedTraining({
      model: 'Qwen/Qwen3-4B-Instruct-2507',
      taskset: this.exportToPITaskset(dataset),
      training_config: {
        rank: config.rank,
        alpha: config.alpha,
        learning_rate: config.learningRate,
        epochs: config.epochs,
        max_steps: 100,
        batch_size: 128,
        rollouts_per_example: 8,
        lora: true,
      },
    });

    // 2. Wait for training completion
    const result = await this.piService.waitForTraining(piRun.run_id);

    // 3. Import trained adapter weights
    return this.importPIAdapter(result.adapter_id);
  }

  private exportToPITaskset(dataset: TrainingDataset): object {
    return {
      rows: dataset.trainingPairs.map(pair => ({
        system_prompt: `Fix this ${pair.errorContext.errorType}: ${pair.errorContext.errorMessage}`,
        prompt: [{ role: 'user', content: pair.errorContext.stackTrace }],
        answer: pair.solutionContext.solutionCode,
        max_turns: 1,
      })),
      rewards: ['correctness', 'latency_score', 'simplicity_score'],
    };
  }
}
```

### 3.6 PI Integration with SkillMintingProcess

The Skill Minting validation pipeline gains PI as a verifiable validation oracle:

```typescript
// packages/KNIRVARENA/src/core/knirvgraph/SkillMintingProcess.ts

class SkillMintingProcess {
  private async performPerformanceValidation(
    loraAdapter: LoRAAdapterSkill,
  ): Promise<PerformanceValidation> {
    // Use PI evaluation for verifiable performance metrics
    if (this.piConfig?.enabled) {
      const piEval = await this.piService.runEvaluation({
        taskset_id: 'ergo-skill-validation',
        model: `${loraAdapter.baseModelCompatibility}:${loraAdapter.skillId}`,
        num_examples: 50,
        rollouts_per_example: 5,
      });

      return {
        expectedAccuracy: piEval.metrics.correctness_mean,
        inferenceLatency: piEval.metrics.latency_mean,
        memoryEfficiency: piEval.metrics.efficiency_mean,
        scalabilityScore: piEval.metrics.scalability_mean,
        robustnessScore: piEval.metrics.robustness_mean,
      };
    }
    // Fallback to local simulated validation
    return this._localPerformanceValidation(loraAdapter);
  }
}
```

---

## 4. TOML Configuration: ERGO Environment on PI

### 4.1 Environment Package Config

```toml
# ergo_env/pyproject.toml
[project]
name = "ergo-arena"
description = "ERGO Arena — Decentralized Trusted Execution Environment for Agent Training"
tags = ["multi-turn", "tool-use", "sandboxed", "train", "eval"]
version = "1.0.0"
requires-python = ">=3.10"
dependencies = [
    "verifiers>=0.1.8",
    "knirvgraph-client>=0.1",
]

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"

[tool.hatch.build]
include = ["ergo_env.py", "ergo_tasksets.py", "ergo_harness.py",
           "ergo_rewards.py", "ergo_tools.py", "ergo_signals.py",
           "pyproject.toml", "tasks/"]

[tool.verifiers.eval]
num_examples = 10
rollouts_per_example = 3
```

### 4.2 RL Training Config

```toml
# configs/rl/ergo-qwen-training.toml
model = "Qwen/Qwen3-4B-Instruct-2507"
max_steps = 200
batch_size = 128
rollouts_per_example = 8

[sampling]
max_tokens = 4096
temperature = 0.7

[[env]]
id = "ergo-arena"

[env.harness]
max_turns = 5
sandbox = true
channels = "mcp"

[env.harness.program]
env = { KNIRVGRAPH_ENDPOINT = "http://knirvgraph:8082" }

[env.taskset]
split = "unverified"
min_severity = "medium"

[env.taskset.toolsets.knirvrouter]
tools = ["ergo_tools:execute_wasm"]
bindings = { "execute_wasm.router_endpoint" = "http://knirvrouter:8086" }

[[env.taskset.rewards]]
fn = "ergo_rewards:error_reproduced"
weight = 0.6

[[env.taskset.rewards]]
fn = "ergo_rewards:latency_score"
weight = 0.3

[[env.taskset.rewards]]
fn = "ergo_rewards:simplicity_score"
weight = 0.1
```

### 4.3 Evaluation Config

```toml
# configs/eval/ergo-benchmark.toml
model = "openai/gpt-4.1-mini"
num_examples = 100
rollouts_per_example = 5

[[eval]]
id = "ergo-arena"
name = "ergo-error-resolution"

[eval.harness]
max_turns = 3

[eval.taskset]
split = "verified"  # Only evaluate on already-verified errors
```

---

## 5. Advanced Integration: Adversarial Noise Injection via PI

PI's parallel rollout capability enables ERGO's `SabotageEngine` to run verifiable adversarial testing:

```python
@vf.reward(weight=0.3)
async def adversarial_robustness(task, state) -> float:
    """
    Tests resistance to SabotageType.NOISE_INJECTION.
    Parallel rollouts with perturbed inputs.
    """
    original_prompt = task["prompt"]
    noise_variants = [
        inject_noise(original_prompt, magnitude=0.1),
        inject_noise(original_prompt, magnitude=0.3),
        inject_noise(original_prompt, magnitude=0.5),
    ]

    # Run parallel rollouts for each noise variant
    results = await asyncio.gather(*[
        run_rollout_with_prompt(variant, state)
        for variant in noise_variants
    ])

    # Score: ratio of successful reproductions under noise
    successes = sum(1 for r in results if r.get("execution_result") == "success")
    return successes / len(results) if results else 0.0
```

This maps directly to ERGO's `SabotageType.NOISE_INJECTION` mechanic where opponents corrupt an agent's input context. With PI, the noise injection becomes a verifiable benchmark dimension.

---

## 6. Verifiable Leaderboard: Tamper-Proof NRN Rewards

PI's group scoring creates a cryptographically signed tournament result:

```python
@vf.setup(priority=20)
async def leaderboard_setup(task, state) -> None:
    """Initialize leaderboard entry for this rollout."""
    state["leaderboard"] = {
        "agent_id": task.get("agent_id", "unknown"),
        "tournament_id": task.get("tournament_id", "unknown"),
        "epoch": time.time(),
    }


@vf.reward(stage="group")
async def leaderboard_scores(tasks: list, states: list) -> list[float]:
    """
    Produces a signed leaderboard entry for the tournament.
    Each agent's score is verifiable on-chain.
    """
    scores = []
    for i, (task, state) in enumerate(zip(tasks, states)):
        score = state.get("reward", 0)
        agent_id = task.get("agent_id", f"agent_{i}")

        # Create signed proof
        proof = create_verifiable_proof({
            "agent_id": agent_id,
            "score": score,
            "tournament_id": task.get("tournament_id"),
            "trajectory_hash": hash_trajectory(state.get("trajectory", [])),
            "validator": "prime-intellect-verifier",
        })

        await knirvgraph_client.submit_leaderboard_entry(proof)
        scores.append(score)

    return scores
```

---

## 7. Implementation Roadmap

### Phase 1: Foundation (Week 1-2)
- [ ] Create `ergo_env/` package structure with `pyproject.toml`
- [ ] Implement `ErgoErrorSource` taskset loading from KNIRVGRAPH
- [ ] Implement basic `ErgoHarness` with sandbox execution
- [ ] Port `Verifier.ts` scoring to PI reward functions (error_reproduced, latency_score, simplicity_score)
- [ ] Write unit tests mapping ERGO test cases to PI rollouts

### Phase 2: Verification Pipeline (Week 3-4)
- [ ] Integrate `ErrorContextManager.submitNewErrorNode()` with PI Rollout trigger
- [ ] Implement LLM-free replay validation for AnchorDatasets
- [ ] Wire `KNIRVSERVERClient` DVE/CDE to PI Sandbox as fallback chain
- [ ] Add PI verification proof storage to KNIRVGRAPH

### Phase 3: Tournament & Rewards (Week 5-6)
- [ ] Implement `@vf.reward(stage="group")` best-of-n tournament scoring
- [ ] Sync `TournamentController.updateRewardWeights()` bidirectionally with PI
- [ ] Add adversarial robustness rewards (NOISE_INJECTION testing)
- [ ] Implement verifiable leaderboard proofs via PI group scoring

### Phase 4: Distributed Training (Week 7-8)
- [ ] Export `LoRAAdapterTrainingPipeline` datasets to PI Hosted Training
- [ ] Import trained PI adapters back into `LoRAAdapterEngine`
- [ ] Deploy minted skills as PI LoRA Inference endpoints
- [ ] Implement fallback chain: PI Hosted → PI Eval → Local training

### Phase 5: Production Hardening (Week 9-10)
- [ ] Add `@vf.setup`/`@vf.cleanup`/`@vf.teardown` lifecycle hooks to all tournament phases
- [ ] TOML config for RL training and evaluation
- [ ] Stress test with parallel rollouts across noise variants
- [ ] Benchmark: compare PI-verified vs local heuristic scores

---

## 8. Performance Considerations

### 8.1 Concurrency Model
ERGO's `CognitiveEngine` runs on a single event loop. PI's BYO Harness uses `asyncio` with `asyncio.to_thread()` for CPU-bound operations:
- **Sandbox rollouts**: PI SandboxEnv handles lifecycle — no blocking
- **WASM execution**: Offload to thread pool (`asyncio.to_thread`)
- **Embedding computation**: Use `ProcessPoolExecutor` for heavy transformations

### 8.2 Caching Strategy
- **Repeated error contexts**: PI caches rollouts by `input_hash` (already computed by `ErrorContextHandler.simulateSha256`)
- **Anchor validation results**: PI replay cache with TTL based on anchor confidence decay
- **Adapter weights**: PI inference endpoint caches deployed adapters — no re-load per request

### 8.3 Fallback Chain
```
PI Sandbox (fastest, preferred)
  → PI Hosted Training (for complex clusters)
    → Local DVE validation (KNIRVSERVERClient original path)
      → Local CDE sandbox (WASM fallback)
        → Heuristic scoring (Verifier.ts legacy path)
```

---

## 9. Conclusion

Integrating Prime Intellect transforms ERGO from a visual simulation into a rigorous, verifiable training platform. The BYO Harness provides the bridge between the game's reactive UI and PI's production RL/eval infrastructure. Key outcomes:

1. **Verifiable Error Contexts**: Every `ErrorNode` is reproduced in a PI sandbox before becoming training data
2. **Tamper-Proof Rewards**: Tournament scores are computed via PI verifier network, signed for on-chain NRN distribution
3. **Distributed LoRA Training**: ERGO's local LoRA pipeline delegates complex training to PI's GPU clusters
4. **Adversarial Robustness**: Noise injection becomes a quantifiable, verifiable benchmark dimension
5. **Signed Leaderboards**: Group scoring creates cryptographically verifiable tournament results
6. **Full Lifecycle**: Error capture → PI verification → Anchor validation → Tournament scoring → LoRA training → PI deployment → On-chain minting
