# RULEBOOK: KNIRVANA — THE DATASET FORGE (v2.0)

---

## I. The Objective

You are an **Architect**. Your goal is not to solve errors yourself — it is to craft training datasets good enough that the **HERO Model** can solve them. The HERO Model reads every dataset submitted to the arena, attempts to resolve active error nodes, and rewards the Architects whose datasets contributed the most.

**Win by being the best teacher, not the best solver.**

---

## II. The Arena: The KNIRVGRAPH

The Arena is a live 3D graph of active error nodes — real AI failures captured from the KNIRV network.

- **Red Nodes (High Severity):** Complex, high-reward errors. Low context density — harder datasets to craft.
- **Green Nodes (Low Severity):** Well-documented errors. Easier datasets, steady Compute income.
- **Corrupted Nodes (Fog):** Nodes where a bad dataset caused the HERO Model to attempt a wrong resolution. Any new dataset submitted here must include a `rejected` field referencing the prior failure.

Each node displays:
- Error class (e.g., `API_Timeout`, `Logic_Loop`, `Hallucination_TypeB`)
- Context density (determines which dataset formats you can use)
- HERO attempt history (how many times it's been tried, and with what results)

---

## III. Player Role & Resources

You play as the **Architect**. You manage two primary resources:

1. **Compute (C):** Earned when your datasets help the HERO Model succeed. Spent on sabotage and agent upgrades.
2. **Parity (P):** Represents your dataset quality track record. Submitting consistently poor or irrelevant datasets drains Parity. Reaching 0 quarantines your submissions for one epoch.

---

## IV. Turn Phases

Each epoch consists of four phases:

### Phase 1: Error Surfacing (Automatic)

The KNIRVGRAPH updates. New error nodes appear; resolved ones are cleared. Each Architect sees:
- Active nodes available to claim
- Updated HERO attempt logs for corrupted nodes
- Competitor activity (which nodes are being targeted this epoch)

### Phase 2: Context Gathering (Agent Phase)

Your agents analyze the error nodes you're targeting and return context packages:
- Raw error logs and stack traces
- Related nodes from KNIRVGRAPH history
- Prior resolution attempts (successful and failed)
- Relevant `skill.md` files already on KNIRVCHAIN

Better agents return richer context. Context density determines which **dataset formats** are available to you this epoch.

### Phase 3: Dataset Crafting (Human Phase — 60 seconds)

**This is where you play.**

You have 60 seconds to craft and propose a dataset for your target error node. You must:

1. Select a TRL-compatible format (based on available context density).
2. Fill the template with entries derived from your agent's context package.
3. Optionally embed a `skill.md` inside the `.nrv` — narrative resolution knowledge that the HERO Model reads before attempting the error node.
4. **Propose the `.nrv` to KNIRVGRAPH** — it is now a pending resolution candidate, visible to all Architects. It is not yet on KNIRVCHAIN.

**Available Formats:**

| Format | Min Context Density | Best For |
| --- | --- | --- |
| Standard (Prompt/Completion) | Any | Simple error-fix pairs |
| Conversational (Chat) | Medium | Multi-turn reasoning failures |
| Preference (DPO) | High | Nodes with prior HERO failed attempts |
| Process Reward | High + HERO history | Step-level reasoning annotation |
| Unpaired Preference (KTO) | Any | Batch labeling of agent-generated attempts |

You may only submit **one dataset per node per epoch**. Quality beats quantity.

### Phase 4: HERO Resolution & Reward (Automatic)

The HERO Model runs its resolution pass across all active nodes using all **proposed `.nrv` files** on the KNIRVGRAPH plus the committed `skill.md` library already on KNIRVCHAIN.

- **Nodes that resolve:** Contributing `.nrv` files are **committed to KNIRVCHAIN**. Compute is distributed to their Architects proportionally to `DatasetScore`. Embedded `skill.md` files are indexed as standalone skills and begin earning passive Compute income.
- **Nodes that fail:** No `.nrv` files are committed. Architects with low-quality proposals lose Parity. The node becomes "Corrupted" — next epoch proposals must use DPO or Process Reward format and directly address the prior failure.

---

## V. Dataset Crafting Rules

1. **Format Validity:** Every entry in your dataset must conform exactly to the selected TRL format schema. Malformed entries score zero — the entire dataset is penalized.

2. **Relevance:** Entries must directly address the target error node's class and context. Off-topic entries reduce your DatasetScore even if correctly formatted.

3. **No Duplication:** If another Architect already submitted the same prompt/completion pair, your entry scores zero. The HERO Model rewards novelty.

4. **Preference Format Restriction:** The `rejected` field in DPO format must be sourced from an actual prior HERO failed attempt log for this node. Fabricated rejections are flagged and penalized.

---

## VI. Sabotage Tactics

Sabotage is a secondary activity. Spending Compute on sabotage instead of dataset quality is always a trade-off.

| Tactic | Cost | Effect |
| --- | --- | --- |
| **Context Poisoning** | 20 C | Inject a misleading entry into a competitor's agent context feed. Their next dataset may contain corrupted data. |
| **Format Trap** | 15 C | Lock one dataset format for a target node this epoch, forcing competitors into a harder format. |
| **Backprop Pulse** | 10 C | Flag a competitor's submitted dataset for re-evaluation. Borderline datasets may be downgraded. |
| **Fog Drop** | 25 C | Corrupt a low-severity node — competitors who target it waste their crafting cycle. |

**Adversarial Drift:** Architects who spend more than 50% of their Compute on sabotage over 3 consecutive epochs have their Parity penalized. The HERO Model tracks contribution history — a saboteur's datasets are weighted lower automatically.

---

## VII. The .nrv Commit

Datasets travel as **`.nrv` (Noted Resolution Vector)** files — the binary dataset container format used by KNIRVBASE. A `skill.md` may be embedded inside the `.nrv` to provide the HERO Model with narrative knowledge alongside structured dataset entries.

**Two-stage lifecycle:**
1. **Proposed** (on KNIRVGRAPH) — your `.nrv` is live and visible to competitors. The HERO Model can read it this epoch. Nothing is permanent yet.
2. **Committed** (on KNIRVCHAIN) — only happens if the HERO Model achieves a successful resolution using your `.nrv`. The file is signed with your KNIRVCHAIN address via Dilithium-3 PQC and stored permanently.

Committed `.nrv` files are read by the HERO Model in every future resolution pass. Embedded `skill.md` files are indexed as standalone skills. Passive Compute income flows to you each time either is referenced.

You may optionally **edit a proposed `.nrv`** before the epoch ends (before the HERO pass runs):
- Costs no Compute; replaces the proposal on KNIRVGRAPH
- If your revision improves the contribution score that epoch: +5 Parity
- If your revision makes it worse: −10 Parity

---

## VIII. The RLHF Joker: Manual Alignment

Once per match, an Architect can trigger **Manual Alignment** on a corrupted node.

> You are shown the HERO Model's full failed resolution attempt (token by token). You annotate which reasoning steps were correct and which were the wrong turns. This creates a **Process Reward dataset** automatically — submitted at no Compute cost and with a 2x DatasetScore multiplier for this node.

This is the most powerful move in the game. Use it on a high-severity corrupted node.

---

## IX. Winning

| Victory Condition | Description |
| --- | --- |
| **The Curator** | 10 of your `skill.md` files are still actively read by the HERO Model after 5 epochs. |
| **The Resolver** | Your datasets contribute to 3 consecutive high-severity (Red) node resolutions. |
| **The Monopolist** | Hold the highest Compute balance for 5 consecutive epochs. |

**Elimination:** If your Parity reaches 0, your datasets are quarantined for one epoch. Two consecutive quarantines = disconnection from the arena.

---

## X. Quick Reference: DatasetScore

```
DatasetScore = (ResolutionContribution × 0.6)
             + (FormatCorrectness × 0.2)
             + (Novelty × 0.2)
```

- **ResolutionContribution:** Did the HERO succeed? How much did your dataset move the needle?
- **FormatCorrectness:** Were all entries structurally valid?
- **Novelty:** Was your submission meaningfully different from competitors' for the same node?

---
