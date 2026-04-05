# GAME MECHANICS SPECIFICATION: KNIRVANA

This is the master design document for **KNIRVANA: The Dataset Forge**. Human Architects do not solve errors directly — they craft the training datasets that teach the **HERO Model** how to solve them.

---

## 1. Core Loop: The Dataset Forge Cycle

The game operates on a continuous loop where human-crafted datasets are consumed by the HERO Model, which then attempts to resolve all active error nodes in the graph.

1. **Error Surfacing:** The KNIRVGRAPH surfaces active Error Nodes — live AI failures captured from the network.
2. **Context Gathering (Agent Phase):** Each Architect's agents analyze the error node and return available context: stack traces, prior attempts, related knowledge, tool call logs.
3. **Dataset Crafting (Human Phase):** The human Architect uses the agent-provided context to fill in a TRL-compatible dataset template. This is the primary action of the game.
4. **HERO Resolution:** The HERO Model reads all submitted datasets, attempts to resolve the error node, and scores each dataset based on how much it contributed to a successful resolution.
5. **Reward Distribution:** Agents whose datasets helped the HERO Model most earn **Compute** rewards. Poor datasets drain resources.

---

## 2. The HERO Model

The **HERO Model** is the central intelligence of the arena — a base LLM that serves as the shared resolver across all active error nodes.

### What the HERO Model Does

| Function | Description |
| --- | --- |
| **Reads Datasets** | Consumes all `skill.md` files and formatted datasets submitted by human Architects each epoch. |
| **Resolves Errors** | Attempts to resolve each active Error Node using the knowledge available in submitted datasets. |
| **Judges Quality** | Scores each submitted dataset based on its contribution to a successful resolution. |
| **Distributes Rewards** | Grants Compute to the agents whose datasets proved most useful. Penalizes misleading or low-quality submissions. |

### HERO Model Knowledge Base

The HERO Model reads from a corpus of `skill.md` files maintained on KNIRVCHAIN. Each `skill.md` is a plain markdown document authored by a human Architect or their agents — no LoRA adapters, no weight modifications. Just structured knowledge that the model can read before attempting a resolution pass.

```
skill.md format:
---
skill_id: <knirvchain_hash>
error_class: <e.g., API_Timeout | Logic_Loop | Hallucination_TypeB>
context_summary: <brief description of the error pattern>
---

# Resolution Knowledge

<Markdown content: strategies, code patterns, edge cases, domain context>
```

The HERO Model treats these files like a reference library — the better the library, the better the resolutions.

---

## 3. Resource Management

| Resource | Symbol | Description | Depletion Effect |
| --- | --- | --- | --- |
| **Compute** | C | Earned by the HERO Model when your datasets contribute to successful resolutions. Spent on sabotage, dataset upgrades, and agent expansions. | Agent loses access to advanced context-gathering tools. |
| **Parity** | P | Represents the alignment quality of your submitted datasets over time. | If Parity reaches 0, your datasets are quarantined for one epoch (excluded from HERO reads). |

---

## 4. The Arena: Error Topography

The battlefield is a **3D KNIRVGRAPH** — a live map of all active error nodes in the network. Each node has:

- **Error Class** — the type of failure (Logic_Loop, API_Timeout, Hallucination, etc.)
- **Severity** — how many downstream processes it's blocking
- **Context Density** — how much usable context your agents can surface (determines which dataset formats are available)
- **HERO Attempt Status** — whether the HERO Model has already tried and failed this node

Higher-severity nodes yield more Compute rewards. Low context-density nodes are harder to craft good datasets for — but competitors are fewer.

### Terrain Features

- **Peaks (Red Nodes):** High-severity, low-context errors. Hard datasets to craft but massive reward potential.
- **Valleys (Green Nodes):** Well-documented, lower-severity errors. Easier datasets, steady Compute income.
- **Fog of Overfitting:** Nodes where the HERO Model attempted resolution using a bad dataset. The node is marked "corrupted" — new submissions must explicitly address the prior failure.

---

## 5. Primary Human Activity: Dataset Crafting

This is the core of why a human is necessary. Agents can gather context, but only the human Architect can determine the **right format** for the dataset and fill it with **meaningful, non-redundant signal**.

### The Workflow

1. **Select an Error Node** from the active KNIRVGRAPH.
2. **Review Agent Context** — agents return structured dumps: error logs, related knowledge, code traces, prior resolution attempts.
3. **Choose a Dataset Format** from the available TRL-compatible templates (see Section 5.1).
4. **Craft the Dataset** — fill the template with relevant, non-redundant entries derived from the agent context.
5. **Package as `.nrv`** — the dataset is packaged into a `.nrv` (Noted Resolution Vector) file. A `skill.md` may be embedded inside the `.nrv` as supporting knowledge for the HERO Model.
6. **Propose to KNIRVGRAPH** — the `.nrv` is proposed onto the KNIRVGRAPH as a pending resolution candidate. It is not yet on KNIRVCHAIN.
7. **Await HERO Resolution** — if the HERO Model successfully resolves the error node using this `.nrv`, it is committed to KNIRVCHAIN. If resolution fails, the `.nrv` remains in proposed state and can be revised next epoch.

### What Makes a Good Dataset

The HERO Model judges quality by how much each dataset improved its resolution confidence on the target error node. Key signals:

- **Relevance** — does the dataset directly address the error class?
- **Diversity** — does it cover edge cases, not just the happy path?
- **Format correctness** — is the TRL format used correctly? Malformed entries score zero.
- **Non-redundancy** — duplicate entries from other Architects are penalized.

---

### 5.1 Available Dataset Formats (TRL-Compatible)

Architects select from the following HuggingFace TRL dataset formats. Available formats depend on the **Context Density** of the selected error node.

#### Format 1: Standard (Prompt/Completion)
*Available at any context density.*

```json
{
  "prompt": "<The exact error context, task description, or broken input>",
  "completion": "<The correct resolution, fix, or expected output>"
}
```
Best for: simple error-to-fix pairs, well-understood error classes.

---

#### Format 2: Conversational (Chat)
*Available at medium+ context density.*

```json
{
  "messages": [
    { "role": "system", "content": "<domain expertise or constraint context>" },
    { "role": "user", "content": "<the failing request or task>" },
    { "role": "assistant", "content": "<the correct resolution with reasoning>" }
  ]
}
```
Best for: multi-turn reasoning errors, agent tool-call failures, hallucination patterns.

---

#### Format 3: Preference (DPO / Chosen-Rejected)
*Available at high context density.*

```json
{
  "prompt": "<the error context>",
  "chosen": "<the correct resolution approach>",
  "rejected": "<a plausible but wrong resolution — from failed prior attempts>"
}
```
Best for: nodes where HERO previously attempted but chose the wrong path. The `rejected` field comes directly from the HERO's prior failed attempt log.

---

#### Format 4: Process Reward (Step-Level)
*Available at high context density + prior HERO attempt data.*

```json
{
  "prompt": "<the error context>",
  "completions": [
    { "content": "<reasoning step 1>", "rating": 1 },
    { "content": "<reasoning step 2>", "rating": 1 },
    { "content": "<wrong turn>", "rating": 0 },
    { "content": "<correction>", "rating": 1 }
  ]
}
```
Best for: complex multi-step reasoning failures. Requires the Architect to annotate which steps in the resolution are correct vs. which are traps.

---

#### Format 5: Unpaired Preference (ORPO/KTO)
*Available at any context density.*

```json
{
  "prompt": "<the error context>",
  "completion": "<a resolution attempt>",
  "label": true
}
```
`label: true` = valid resolution. `label: false` = invalid. Best for quickly tagging large batches of agent-generated resolution attempts as good or bad.

---

## 6. The Adversarial Layer (Sabotage)

Human Architects can still interfere with competitors. These are secondary activities — they don't directly resolve error nodes but can affect rivals' dataset quality and Compute flow.

| Tactic | Cost | Effect |
| --- | --- | --- |
| **Context Poisoning** | 20 Compute | Inject a misleading entry into the target's agent context feed. Their next dataset may include corrupted data. |
| **Format Trap** | 15 Compute | Temporarily make one dataset format unavailable for a target node, forcing competitors into a harder format. |
| **Backprop Pulse** | 10 Compute | Flag a competitor's submitted dataset for HERO Model re-evaluation. If it was borderline quality, it may be downgraded. |
| **Fog Drop** | 25 Compute | Mark a low-severity node as "Fog of Overfitting" — competitors who target it waste a crafting cycle. |

### Adversarial Drift

If an Architect focuses too heavily on sabotage instead of dataset crafting, their **Parity** degrades. The HERO Model weights their future submissions lower because their track record shows low resolution contribution.

---

## 7. The .nrv File and Skill.md System

Datasets are packaged as **`.nrv` (Noted Resolution Vector)** files — a binary container format defined in `packages/KNIRVBASE/go/`. A `skill.md` knowledge document can be embedded inside a `.nrv` as part of its content payload, providing the HERO Model with narrative resolution context alongside structured training data.

### .nrv Lifecycle: Propose → Resolve → Commit

```
Human crafts dataset
        ↓
Packaged as .nrv (may include embedded skill.md)
        ↓
Proposed to KNIRVGRAPH  ← pending, not yet on chain
        ↓
HERO Model reads .nrv during resolution pass
        ↓
Resolution succeeds → .nrv committed to KNIRVCHAIN ← permanent, signed
Resolution fails    → .nrv stays proposed; revise next epoch
```

**Proposed** `.nrv` files are visible to all Architects on the KNIRVGRAPH — competitors can see what you're working on. Only **committed** `.nrv` files (those that contributed to a successful resolution) earn Compute rewards and become part of the HERO Model's permanent reference library.

### The Embedded Skill.md

When an Architect includes a `skill.md` inside their `.nrv`, it gives the HERO Model narrative context — resolution strategies, edge cases, code patterns — that supplements the structured TRL dataset entries. The HERO Model treats the embedded `skill.md` as a reference document read before attempting resolution.

Once the `.nrv` is committed to KNIRVCHAIN, the embedded `skill.md` is indexed separately as a standalone skill, earning the Architect passive Compute income each time it's referenced by future HERO resolution passes.

Skill authorship is cryptographically tied to the Architect's KNIRVCHAIN address in the `.nrv` PQC manifest (Dilithium-3 signature).

---

## 8. Agent Training: "The Forge"

After a successful resolution epoch, Architects can upgrade their agents' context-gathering capabilities. Better agents surface richer context, unlocking higher-tier dataset formats (Preference, Process Reward).

| Upgrade | Cost | Effect |
| --- | --- | --- |
| **Context Depth** | 30 Compute | Agent returns deeper stack traces and tool-call logs. |
| **Cross-Node Linking** | 40 Compute | Agent identifies similar past errors in the KNIRVGRAPH and includes their resolutions as context. |
| **Noise Filter** | 20 Compute | Agent filters out low-signal context before presenting it. Reduces your risk of crafting a noisy dataset. |

---

## 9. Reward Equation

The HERO Model calculates each dataset's contribution score as:

```
DatasetScore = (ResolutionContribution * 0.6) 
             + (FormatCorrectness * 0.2) 
             + (Novelty * 0.2)
```

Where:
- **ResolutionContribution**: Did the HERO Model's resolution attempt succeed on this node? How much did this dataset specifically improve the confidence score?
- **FormatCorrectness**: Were all entries structurally valid per the selected TRL format?
- **Novelty**: Was the dataset meaningfully different from other Architects' submissions for the same node?

Compute rewards are distributed proportionally to DatasetScore among all Architects who submitted for a resolved node.

---

## 10. Victory Conditions

| Victory Type | Condition |
| --- | --- |
| **The Curator** | Accumulate 10 committed `skill.md` files that are still actively used by the HERO Model after 5 epochs. |
| **The Resolver** | Your datasets contribute to the resolution of 3 consecutive high-severity (Red) error nodes. |
| **The Monopolist** | Hold the highest Compute balance in the arena for 5 consecutive epochs. |

---

## 11. Technical Backend

The game state is governed by the relationship between the **Tournament Controller** (TypeScript engine) and **KNIRVCHAIN** (ledger of datasets and skill.md files).

| Layer | Role |
| --- | --- |
| **Tournament Controller** | Manages epochs, routes `.nrv` files to the HERO Model API, distributes Compute rewards, tracks Parity. |
| **KNIRVGRAPH** | Holds all **proposed** `.nrv` files (pending resolution). Live error node data: class, severity, context density, HERO attempt history. |
| **HERO Model API** | Accepts proposed `.nrv` batches per epoch, returns resolution results and per-dataset contribution scores. |
| **KNIRVCHAIN** | Stores only **committed** `.nrv` files — those that contributed to a successful resolution. Cryptographic authorship via Dilithium-3 PQC signatures. |

### Why KNIRVGRAPH First, Then KNIRVCHAIN?

Separating the proposal stage (KNIRVGRAPH) from the commit stage (KNIRVCHAIN) ensures:
1. **Quality Gate** — only `.nrv` files that demonstrably helped the HERO Model resolve an error node earn chain residency.
2. **Anti-Spam** — submitting a `.nrv` is cheap; the cost is the Compute you spent crafting it. But chain space is reserved for proven knowledge.
3. **Audit Trail** — every committed `.nrv` has a cryptographic lineage: who proposed it, which epoch it was tested, which error node it resolved.

---
