# Error Node Dataset Pipeline

This document describes how raw AI failures from the KNIRV network are transformed into structured training datasets by human Architects in KNIRVANA, consumed by the **HERO Model** for error resolution, and committed to KNIRVCHAIN as permanent `skill.md` knowledge files.

The goal is not to hardcode solutions — it is to build a dataset library rich enough that the HERO Model can generalize resolution strategies across the entire error graph.

---

## 1. The FailureContext Object

Every error node in the arena is backed by a `FailureContext` — the environmental state captured at the moment of AI failure. This is what your agents surface during the Context Gathering phase.

| Field | Description |
| --- | --- |
| `error_id` | Unique hash from KNIRVCHAIN (verifiable, tamper-proof). |
| `error_class` | Taxonomy tag (e.g., `API_Timeout`, `Logic_Loop`, `Hallucination_TypeB`). |
| `input_prompt` | The original task the AI was given when it failed. |
| `failed_response` | The actual incorrect or broken output. |
| `trace` | Tool call logs, API responses, agent reasoning steps captured at failure time. |
| `hero_attempts` | Array of prior HERO Model resolution attempts with scores (used for DPO/Process Reward formats). |
| `context_density` | Integer 1–10. Determines which TRL dataset formats are available for this node. |

---

## 2. TRL Dataset Formats

Architects fill these templates with data sourced from the `FailureContext`. The format you choose determines what the HERO Model learns and how it learns it.

All formats are sourced from HuggingFace TRL dataset specifications.

---

### Format 1: Standard (Prompt/Completion)
**Min context_density:** 1 (always available)

The simplest format. Map the failure directly to its correct resolution.

```json
{
  "prompt": "Generate a smart contract for a time-locked withdrawal. [full failing context]",
  "completion": "The error was a missing reentrancy guard. Correct implementation: [fixed code]"
}
```

**When to use:** The error class is well-understood and has a clear single correct answer. Best for `API_Timeout`, `Syntax_Error`, `Missing_Handler` classes.

---

### Format 2: Conversational (Chat)
**Min context_density:** 4

Models the resolution as a dialogue. Use when the failure involved multi-turn reasoning or tool use.

```json
{
  "messages": [
    {
      "role": "system",
      "content": "You are resolving a Logic_Loop error in a recursive agent call chain. The agent has no base case termination."
    },
    {
      "role": "user",
      "content": "The agent called `resolve_dependency()` 847 times before stack overflow. Input: [full trace from FailureContext.trace]"
    },
    {
      "role": "assistant",
      "content": "The missing base case is [X]. The correct termination condition is [Y]. Here is the fixed call chain: [resolution]"
    }
  ]
}
```

**When to use:** Failures involving tool call chains, multi-step agent reasoning, or hallucination patterns where context matters.

---

### Format 3: Preference / DPO (Chosen–Rejected)
**Min context_density:** 7
**Requires:** At least one entry in `FailureContext.hero_attempts`

Directly targets nodes where the HERO Model has already tried and failed. The `rejected` field must come from the prior HERO attempt log — not fabricated.

```json
{
  "prompt": "[FailureContext.input_prompt + full error context]",
  "chosen": "The correct resolution approach: [your crafted resolution]",
  "rejected": "[FailureContext.hero_attempts[0].output — the HERO's previous wrong answer]"
}
```

**When to use:** "Corrupted" nodes in the arena — marked with HERO failed attempt history. This format teaches the HERO Model what NOT to do as much as what to do. Highest DatasetScore multiplier for corrupted nodes.

---

### Format 4: Process Reward (Step-Level Annotation)
**Min context_density:** 8
**Requires:** HERO attempt history with chain-of-thought logs

Annotate each reasoning step in the HERO Model's prior attempt as correct or incorrect.

```json
{
  "prompt": "[FailureContext.input_prompt]",
  "completions": [
    { "content": "First, I'll check if the API endpoint is reachable.", "rating": 1 },
    { "content": "The endpoint returns 200, so the issue must be in parsing.", "rating": 1 },
    { "content": "I'll assume the response is always JSON.", "rating": 0 },
    { "content": "Instead, I should validate content-type before parsing.", "rating": 1 }
  ]
}
```

**When to use:** Complex multi-step failures where the HERO Model's reasoning partially succeeded but went wrong at a specific step. Requires the Architect to carefully read the full CoT trace and annotate each step. High effort, high reward.

---

### Format 5: Unpaired Preference (KTO / ORPO)
**Min context_density:** 1 (always available)

Label resolution attempts as valid or invalid. Best for rapid batch annotation of agent-generated candidates.

```json
{ "prompt": "[error context]", "completion": "[resolution attempt A]", "label": true }
{ "prompt": "[error context]", "completion": "[resolution attempt B]", "label": false }
{ "prompt": "[error context]", "completion": "[resolution attempt C]", "label": true }
```

**When to use:** When your agents generate multiple resolution candidates and you need to quickly curate which are valid before submitting. Efficient but scores lower than Preference format for corrupted nodes.

---

## 3. The Dataset Pipeline

```
AI fails in KNIRV network
        ↓
FailureContext captured → Error Node added to KNIRVGRAPH
        ↓
Architect's agents surface context (Phase 2)
        ↓
Architect selects TRL format and crafts dataset (Phase 3 — 60 seconds)
        ↓
Dataset packaged as .nrv file
  └── skill.md may be embedded inside .nrv as narrative context
        ↓
.nrv proposed to KNIRVGRAPH  ← PENDING, not yet on chain
  └── visible to all Architects; HERO can read it this epoch
        ↓
HERO Model reads all proposed .nrv files + committed skill.md library
        ↓
HERO attempts resolution on each active Error Node
        ↓
Resolution SUCCESS → contributing .nrv files committed to KNIRVCHAIN ← PERMANENT
  └── embedded skill.md indexed as standalone skill
  └── Compute distributed to contributing Architects
Resolution FAILURE → .nrv files stay proposed (not committed)
  └── Node marked Corrupted; next epoch requires DPO/Process Reward format
```

---

## 4. The .nrv File and Embedded Skill.md

Datasets are packaged and proposed as **`.nrv` (Noted Resolution Vector)** files — the binary dataset container defined in `packages/KNIRVBASE/go/`. A `.nrv` is committed to KNIRVCHAIN only after the HERO Model achieves a successful resolution using it. Prior to that, it lives on KNIRVGRAPH as a pending proposal.

### Embedding a Skill.md Inside .nrv

A `skill.md` is a plain markdown knowledge document that can be embedded inside a `.nrv` file. It provides the HERO Model with narrative resolution context — strategies, edge cases, code patterns — that supplements the structured TRL dataset entries. The HERO Model reads embedded `skill.md` content before running its resolution attempt on the associated error node.

Once the `.nrv` is committed, the embedded `skill.md` is extracted and indexed as a standalone skill on KNIRVCHAIN. Architects earn passive Compute income each time it's read by a future HERO resolution pass.

### Skill.md Format (when embedded in .nrv)

```markdown
---
skill_id: <assigned by KNIRVCHAIN on .nrv commit>
nrv_id: <parent .nrv file hash>
error_class: API_Timeout
author: <architect_knirvchain_address>
resolved_node: <error_node_id>
epoch: 47
status: proposed | committed
---

# API Timeout Resolution Pattern

## Error Context
Timeouts occur when downstream service SLA is under 200ms but the agent 
assumes 1000ms default. The `fetch` call has no explicit timeout parameter.

## Resolution
Always pass an explicit `AbortController` signal with a timeout derived 
from the downstream SLA registered in KNIRVGRAPH for that service class.

## Edge Cases
- If SLA is unknown, default to 500ms, not the system default
- Retry logic must use exponential backoff, not fixed intervals
- Auth token refresh must happen outside the timeout window

## Code Pattern
```typescript
const controller = new AbortController();
const timeout = setTimeout(() => controller.abort(), sla ?? 500);
try {
  const res = await fetch(url, { signal: controller.signal });
} finally {
  clearTimeout(timeout);
}
```
```

A `skill.md` embedded in a **proposed** `.nrv` has `status: proposed` — the HERO Model reads it, but it earns no Compute yet. Once the parent `.nrv` is committed to KNIRVCHAIN, the `skill.md` status becomes `committed` and begins earning passive Compute income on each future HERO read.

---

## 5. Data Collector Reference (Backend)

The backend automatically packages failures into `FailureContext` objects. Architects do not write this code — it runs on KNIRVSERVER and populates the KNIRVGRAPH.

```python
class KnirvDataCollector:
    def capture_failure(self, prompt, failed_output, model_settings, trace_log):
        context_hash = hashlib.sha256(f"{prompt}{datetime.now()}".encode()).hexdigest()
        return {
            "error_id": context_hash,
            "error_class": self._classify(failed_output),
            "input_prompt": prompt,
            "failed_response": failed_output,
            "metadata": {
                "temp": model_settings.get("temperature"),
                "top_p": model_settings.get("top_p"),
            },
            "trace": trace_log,
            "hero_attempts": [],           # populated as HERO attempts resolution
            "context_density": self._score_density(trace_log),
            "status": "PENDING_DATASET"    # ready for Architect crafting
        }
```

`context_density` is computed from the richness of the trace log — number of tool calls logged, stack trace depth, number of related nodes found in KNIRVGRAPH. This directly determines which TRL formats are available to Architects targeting this node.

---
