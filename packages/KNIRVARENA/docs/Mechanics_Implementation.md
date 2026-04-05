# Mechanics Implementation

## 1. Project Overview & Architecture

**Goal:** Implement the KNIRVANA game engine where human Architects craft TRL-compatible training datasets for active error nodes. The **HERO Model** consumes all submitted datasets, attempts to resolve error nodes, and distributes Compute rewards based on each dataset's contribution score.

**Core Stack:**

- **Language:** TypeScript (Node.js)
- **Backend Integration:** REST API (Fetch) to HERO Model API + KNIRVCHAIN
- **Real-time:** Phoenix Channels (WebSocket) for game state streaming

---

## 2. Directory Structure

```text
src/
├── engine/
│   ├── Tournament.ts         # Main game loop & epoch controller
│   ├── DatasetValidator.ts   # TRL format validation logic
│   ├── Sabotage.ts           # Adversarial effect logic
│   └── RewardDistributor.ts  # Compute reward calculation
├── types/
│   ├── Agent.ts              # Agent interfaces and context packages
│   ├── Dataset.ts            # TRL dataset format schemas
│   ├── ErrorNode.ts          # Error node and FailureContext types
│   └── GameState.ts          # Resource tracking (Compute/Parity)
├── networking/
│   ├── HeroClient.ts         # API wrapper for HERO Model
│   └── KnirvChainClient.ts   # API wrapper for dataset/skill.md commits
└── utils/
    └── DatasetScorer.ts      # Scoring formula utilities
```

---

## 3. Core Data Structures (`src/types/`)

### A. Error Node & Failure Context

**File:** `src/types/ErrorNode.ts`

```typescript
export type ErrorClass =
  | 'API_Timeout'
  | 'Logic_Loop'
  | 'Hallucination_TypeB'
  | 'Missing_Handler'
  | 'Syntax_Error'
  | string;

export interface HeroAttempt {
  epoch: number;
  output: string;
  chainOfThought: string[];
  score: number; // 0.0–1.0 — how close HERO got before failing
}

export interface FailureContext {
  errorId: string;           // SHA-256 hash from KNIRVCHAIN
  errorClass: ErrorClass;
  inputPrompt: string;       // Original task that failed
  failedResponse: string;    // The bad output
  trace: string[];           // Tool call logs, API responses
  heroAttempts: HeroAttempt[]; // Prior HERO resolution attempts
  contextDensity: number;    // 1–10, determines available dataset formats
  severity: number;          // 1–10, determines Compute reward multiplier
  status: 'PENDING_DATASET' | 'CORRUPTED' | 'RESOLVED';
}
```

---

### B. TRL Dataset Formats

**File:** `src/types/Dataset.ts`

```typescript
// Format 1: Standard
export interface StandardDataset {
  format: 'standard';
  entries: Array<{
    prompt: string;
    completion: string;
  }>;
}

// Format 2: Conversational
export interface ConversationalDataset {
  format: 'conversational';
  entries: Array<{
    messages: Array<{
      role: 'system' | 'user' | 'assistant';
      content: string;
    }>;
  }>;
}

// Format 3: Preference / DPO
export interface PreferenceDataset {
  format: 'preference';
  entries: Array<{
    prompt: string;
    chosen: string;
    rejected: string; // Must reference a real HeroAttempt output
  }>;
}

// Format 4: Process Reward
export interface ProcessRewardDataset {
  format: 'process_reward';
  entries: Array<{
    prompt: string;
    completions: Array<{
      content: string;
      rating: 0 | 1;
    }>;
  }>;
}

// Format 5: Unpaired Preference (KTO/ORPO)
export interface UnpairedPreferenceDataset {
  format: 'unpaired_preference';
  entries: Array<{
    prompt: string;
    completion: string;
    label: boolean;
  }>;
}

export type DatasetSubmission =
  | StandardDataset
  | ConversationalDataset
  | PreferenceDataset
  | ProcessRewardDataset
  | UnpairedPreferenceDataset;

export interface SubmittedDataset {
  datasetId: string;         // SHA-256 hash = .nrv file ID
  architectId: string;       // KNIRVCHAIN address
  targetNodeId: string;
  epoch: number;
  dataset: DatasetSubmission;
  skillMd?: string;          // Optional embedded skill.md markdown content
  submittedAt: string;
  // Lifecycle: 'proposed' on KNIRVGRAPH until HERO resolves; then 'committed' to KNIRVCHAIN
  status: 'proposed' | 'committed' | 'failed';
}
```

---

### C. Agent & Resources

**File:** `src/types/Agent.ts`

```typescript
export interface AgentResources {
  compute: number;   // Earned via dataset quality; spent on sabotage/upgrades
  parity: number;    // Quality track record; 0 = quarantined for one epoch
}

export interface ContextPackage {
  rawTrace: string[];
  relatedNodes: string[];        // Similar past error node IDs from KNIRVGRAPH
  priorResolutions: string[];    // Successful resolution patterns for similar errors
  heroAttempts: HeroAttempt[];   // Surfaced from FailureContext
  availableFormats: DatasetSubmission['format'][];  // Based on contextDensity
}

export interface KnirvAgent {
  id: string;
  architectId: string;
  resources: AgentResources;
  contextDepth: number;    // Upgrade level 1–3, affects context richness
  crossNodeLinking: boolean; // Upgrade: surfaces related node history
  noiseFilter: boolean;      // Upgrade: pre-filters low-signal context

  gatherContext(node: FailureContext): Promise<ContextPackage>;
}
```

---

### D. Game State

**File:** `src/types/GameState.ts`

```typescript
export interface PlayerState {
  playerId: string;
  role: 'Architect';
  resources: {
    compute: number;
    parity: number;
  };
  cooldowns: {
    sabotage: number;
    manualAlignment: number; // Once per match
  };
  committedSkills: string[]; // skill.md IDs on KNIRVCHAIN
}

export interface WorldState {
  activeNodes: FailureContext[];
  currentEpoch: number;
  isPaused: boolean;
}
```

---

## 4. Dataset Validator (`src/engine/DatasetValidator.ts`)

Validates submitted datasets before they are sent to the HERO Model. A single malformed entry invalidates the entire submission.

```typescript
import { DatasetSubmission, PreferenceDataset } from '../types/Dataset';
import { FailureContext } from '../types/ErrorNode';

export class DatasetValidator {
  public validate(
    submission: DatasetSubmission,
    node: FailureContext
  ): { valid: boolean; errors: string[] } {
    const errors: string[] = [];

    // Format availability check
    if (submission.format === 'preference' || submission.format === 'process_reward') {
      if (node.contextDensity < 7) {
        errors.push(`Format '${submission.format}' requires contextDensity >= 7. Node has ${node.contextDensity}.`);
      }
    }

    if (submission.format === 'conversational' && node.contextDensity < 4) {
      errors.push(`Format 'conversational' requires contextDensity >= 4.`);
    }

    // Preference format: rejected must come from a real HERO attempt
    if (submission.format === 'preference') {
      const pref = submission as PreferenceDataset;
      const heroOutputs = node.heroAttempts.map(a => a.output);
      for (const entry of pref.entries) {
        if (!heroOutputs.includes(entry.rejected)) {
          errors.push(`Preference entry 'rejected' field must match a real HERO attempt output for node ${node.errorId}.`);
        }
      }
    }

    // Non-empty entries
    if (!('entries' in submission) || submission.entries.length === 0) {
      errors.push('Dataset must contain at least one entry.');
    }

    return { valid: errors.length === 0, errors };
  }
}
```

---

## 5. HERO Model Client (`src/networking/HeroClient.ts`)

Wraps the HERO Model API. The Tournament Controller sends batched datasets per epoch and receives resolution results with per-dataset contribution scores.

```typescript
import { SubmittedDataset } from '../types/Dataset';
import { FailureContext } from '../types/ErrorNode';

export interface HeroResolutionResult {
  nodeId: string;
  resolved: boolean;
  resolution: string | null;
  datasetContributions: Array<{
    datasetId: string;
    contributionScore: number; // 0.0–1.0
  }>;
}

export class HeroClient {
  constructor(private baseUrl: string) {}

  public async runResolutionPass(
    nodes: FailureContext[],
    datasets: SubmittedDataset[]
  ): Promise<HeroResolutionResult[]> {
    const response = await fetch(`${this.baseUrl}/hero/resolve`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ nodes, datasets }),
    });

    if (!response.ok) {
      throw new Error(`HERO Model API error: ${response.status}`);
    }

    return response.json();
  }

  public async generateSkillMd(
    nodeId: string,
    resolution: string
  ): Promise<string> {
    const response = await fetch(`${this.baseUrl}/hero/skill`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ nodeId, resolution }),
    });

    return response.text(); // Returns skill.md content
  }
}
```

---

## 6. KNIRVCHAIN Client (`src/networking/KnirvChainClient.ts`)

Commits datasets and skill.md files to the chain. Every submission is signed with the Architect's address.

```typescript
export interface SkillMdCommit {
  skillId: string;
  architectAddress: string;
  nodeId: string;
  epoch: number;
  content: string; // Full skill.md markdown
}

export class KnirvChainClient {
  constructor(private baseUrl: string) {}

  // Phase 3: propose a .nrv onto KNIRVGRAPH (pending — not yet committed to chain)
  public async proposeNrv(dataset: SubmittedDataset): Promise<string> {
    const response = await fetch(`${this.baseUrl}/graph/propose`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ...dataset, status: 'proposed' }),
    });
    const result = await response.json();
    return result.nrvId;
  }

  // Phase 4: commit a .nrv to KNIRVCHAIN after successful HERO resolution
  // Only called when HERO resolution succeeds and this .nrv contributed
  public async commitNrv(nrvId: string, architectAddress: string): Promise<string> {
    const response = await fetch(`${this.baseUrl}/chain/commit`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ nrvId, architectAddress }),
    });
    const result = await response.json();
    return result.txHash;
  }

  // Commit embedded skill.md (extracted from committed .nrv)
  public async commitSkillMd(skill: SkillMdCommit): Promise<string> {
    const response = await fetch(`${this.baseUrl}/chain/skill`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(skill),
    });
    const result = await response.json();
    return result.skillId;
  }

  // Fetch all committed skill.md files (HERO Model reference library)
  public async getSkillLibrary(): Promise<SkillMdCommit[]> {
    const response = await fetch(`${this.baseUrl}/chain/skills`);
    return response.json();
  }

  // Fetch all proposed .nrv files for a given error node (KNIRVGRAPH)
  public async getProposedNrvs(nodeId: string): Promise<SubmittedDataset[]> {
    const response = await fetch(`${this.baseUrl}/graph/proposals/${nodeId}`);
    return response.json();
  }
}
```

---

## 7. Reward Distributor (`src/engine/RewardDistributor.ts`)

Calculates final Compute rewards from HERO Model contribution scores.

```typescript
import { HeroResolutionResult } from '../networking/HeroClient';
import { SubmittedDataset } from '../types/Dataset';

export interface RewardAllocation {
  architectId: string;
  computeEarned: number;
  parityDelta: number;
}

export class RewardDistributor {
  public calculate(
    results: HeroResolutionResult[],
    submissions: SubmittedDataset[],
    nodeSeverityMap: Map<string, number>
  ): RewardAllocation[] {
    const allocations = new Map<string, RewardAllocation>();

    for (const result of results) {
      const severity = nodeSeverityMap.get(result.nodeId) ?? 1;
      const baseReward = severity * 10; // Higher severity = more Compute

      for (const contrib of result.datasetContributions) {
        const submission = submissions.find(s => s.datasetId === contrib.datasetId);
        if (!submission) continue;

        const current = allocations.get(submission.architectId) ?? {
          architectId: submission.architectId,
          computeEarned: 0,
          parityDelta: 0,
        };

        if (result.resolved) {
          current.computeEarned += Math.floor(baseReward * contrib.contributionScore);
          current.parityDelta += contrib.contributionScore > 0.5 ? 2 : 0;
        } else {
          // Failed resolution — penalize low-scoring contributors
          if (contrib.contributionScore < 0.2) {
            current.parityDelta -= 5;
          }
        }

        allocations.set(submission.architectId, current);
      }
    }

    return Array.from(allocations.values());
  }
}
```

---

## 8. Tournament Loop (`src/engine/Tournament.ts`)

The main epoch controller.

```typescript
import { HeroClient } from '../networking/HeroClient';
import { KnirvChainClient } from '../networking/KnirvChainClient';
import { DatasetValidator } from './DatasetValidator';
import { RewardDistributor } from './RewardDistributor';
import { SubmittedDataset } from '../types/Dataset';
import { FailureContext } from '../types/ErrorNode';
import { WorldState, PlayerState } from '../types/GameState';

export class Tournament {
  public currentEpoch = 0;
  public worldState: WorldState;
  private pendingDatasets: SubmittedDataset[] = [];

  constructor(
    private hero: HeroClient,
    private chain: KnirvChainClient,
    private validator: DatasetValidator,
    private rewardDistributor: RewardDistributor
  ) {}

  // Called when an Architect proposes a .nrv during Phase 3
  // Proposes to KNIRVGRAPH — not committed to KNIRVCHAIN yet
  public proposeDataset(
    submission: SubmittedDataset,
    node: FailureContext
  ): { accepted: boolean; errors: string[] } {
    const validation = this.validator.validate(submission.dataset, node);
    if (!validation.valid) return { accepted: false, errors: validation.errors };

    submission.status = 'proposed';
    this.pendingDatasets.push(submission);
    this.chain.proposeNrv(submission); // Async, non-blocking — lands on KNIRVGRAPH
    return { accepted: true, errors: [] };
  }

  // Phase 4: HERO resolution pass (runs automatically at epoch end)
  public async runResolutionPass(players: Map<string, PlayerState>) {
    const activeNodes = this.worldState.activeNodes;

    let results;
    try {
      results = await this.hero.runResolutionPass(activeNodes, this.pendingDatasets);
    } catch {
      this.worldState.isPaused = true;
      return;
    }

    // Commit .nrv files and skill.md for resolved nodes
    for (const result of results) {
      if (result.resolved && result.resolution) {
        // Commit contributing .nrv files to KNIRVCHAIN
        for (const contrib of result.datasetContributions) {
          if (contrib.contributionScore > 0) {
            const submission = this.pendingDatasets.find(s => s.datasetId === contrib.datasetId);
            if (submission) {
              submission.status = 'committed';
              await this.chain.commitNrv(submission.datasetId, submission.architectId);

              // If .nrv contains an embedded skill.md, extract and commit it separately
              if (submission.skillMd) {
                await this.chain.commitSkillMd({
                  skillId: `skill-${result.nodeId}-${submission.datasetId.slice(0, 8)}`,
                  architectAddress: submission.architectId,
                  nodeId: result.nodeId,
                  epoch: this.currentEpoch,
                  content: submission.skillMd,
                });
              }
            }
          }
        }

        // Remove resolved node from active set
        this.worldState.activeNodes = this.worldState.activeNodes.filter(
          n => n.errorId !== result.nodeId
        );
      } else {
        // Resolution failed — mark node Corrupted; .nrv proposals remain on KNIRVGRAPH (not committed)
        const node = this.worldState.activeNodes.find(n => n.errorId === result.nodeId);
        if (node) node.status = 'CORRUPTED';
        for (const contrib of result.datasetContributions) {
          const submission = this.pendingDatasets.find(s => s.datasetId === contrib.datasetId);
          if (submission) submission.status = 'failed';
        }
      }
    }

    // Distribute rewards
    const severityMap = new Map(activeNodes.map(n => [n.errorId, n.severity]));
    const allocations = this.rewardDistributor.calculate(results, this.pendingDatasets, severityMap);

    for (const alloc of allocations) {
      const player = players.get(alloc.architectId);
      if (!player) continue;
      player.resources.compute += alloc.computeEarned;
      player.resources.parity = Math.max(0, Math.min(100, player.resources.parity + alloc.parityDelta));
    }

    // Reset for next epoch
    this.pendingDatasets = [];
    this.currentEpoch++;
  }
}
```

---

## 9. Adversarial Mechanics (`src/engine/Sabotage.ts`)

Secondary activity — Architects spend Compute to disrupt competitors.

```typescript
import { KnirvAgent } from '../types/Agent';
import { FailureContext } from '../types/ErrorNode';

export enum SabotageType {
  CONTEXT_POISONING = 'CONTEXT_POISONING',
  FORMAT_TRAP = 'FORMAT_TRAP',
  BACKPROP_PULSE = 'BACKPROP_PULSE',
  FOG_DROP = 'FOG_DROP',
}

export class SabotageEngine {
  public static applyEffect(
    type: SabotageType,
    target: KnirvAgent,
    node: FailureContext,
    magnitude: number
  ) {
    switch (type) {
      case SabotageType.CONTEXT_POISONING:
        // Inject a misleading entry into the target agent's next context gather
        target['_poisonedContext'] = `[CORRUPTED]: ${magnitude} false entries injected`;
        break;

      case SabotageType.FORMAT_TRAP:
        // Lock one available format for this node for the target this epoch
        // Implementation: add a per-agent format exclusion list checked by DatasetValidator
        break;

      case SabotageType.BACKPROP_PULSE:
        // Flag target's submitted dataset for HERO re-evaluation
        // Implementation: add dataset ID to a re-evaluation queue processed before reward calc
        break;

      case SabotageType.FOG_DROP:
        // Corrupt a node — mark it as CORRUPTED preemptively
        node.status = 'CORRUPTED';
        node.heroAttempts.push({
          epoch: -1,
          output: '[FOG_DROP: artificially corrupted]',
          chainOfThought: [],
          score: 0,
        });
        break;
    }
  }
}
```

---

## 10. Game Server API (`src/api/GameServer.ts`)

REST/WebSocket layer for the human UI.

```typescript
import express from 'express';
import { Tournament } from '../engine/Tournament';
import { SabotageEngine, SabotageType } from '../engine/Sabotage';

export class GameServer {
  private app = express();

  constructor(private tournament: Tournament) {
    this.app.use(express.json());
    this.setupRoutes();
  }

  private setupRoutes() {
    // Human proposes a .nrv dataset for a node (lands on KNIRVGRAPH, pending resolution)
    this.app.post('/architect/propose', (req, res) => {
      const { submission, nodeId } = req.body;
      const node = this.tournament.worldState.activeNodes.find(n => n.errorId === nodeId);
      if (!node) return res.status(404).json({ error: 'Node not found' });

      const result = this.tournament.proposeDataset(submission, node);
      res.json(result);
    });

    // Human executes a sabotage action
    this.app.post('/architect/sabotage', (req, res) => {
      const { targetAgentId, type, nodeId, magnitude } = req.body;
      // Validate Compute cost, apply effect
      // ... 
      res.json({ success: true });
    });

    // Get current game state
    this.app.get('/state', (req, res) => {
      res.json({
        epoch: this.tournament.currentEpoch,
        activeNodes: this.tournament.worldState.activeNodes.map(n => ({
          errorId: n.errorId,
          errorClass: n.errorClass,
          severity: n.severity,
          contextDensity: n.contextDensity,
          status: n.status,
          heroAttemptCount: n.heroAttempts.length,
        })),
        isPaused: this.tournament.worldState.isPaused,
      });
    });
  }

  public start(port: number) {
    this.app.listen(port, () => console.log(`KNIRVANA server running on port ${port}`));
  }
}
```

---

## 11. Implementation Order

### Phase 1: Data Layer
1. Create `src/types/ErrorNode.ts`, `src/types/Dataset.ts`, `src/types/GameState.ts`.
2. Create `src/engine/DatasetValidator.ts`.
3. Unit test: valid Preference format with real HERO attempt reference → passes. Fabricated `rejected` field → fails.

### Phase 2: HERO Integration
1. Create `src/networking/HeroClient.ts` with mocked resolution responses.
2. Create `src/networking/KnirvChainClient.ts` with mocked commit responses.
3. Unit test: submit a Standard dataset, mock HERO returns `resolved: true` with contribution score 0.8 → Compute reward calculated correctly.

### Phase 3: Tournament Loop
1. Create `src/engine/RewardDistributor.ts`.
2. Create `src/engine/Tournament.ts`.
3. Integration test: two Architects submit datasets for the same node. HERO resolves it. Architect with higher contribution score receives more Compute.

### Phase 4: Sabotage & API
1. Create `src/engine/Sabotage.ts`.
2. Create `src/api/GameServer.ts`.
3. Test: `POST /architect/sabotage` with FORMAT_TRAP — confirm target's available formats are restricted for that node next epoch.

---

## 12. Configuration

```json
{
  "name": "knirvana-engine",
  "version": "2.0.0",
  "scripts": {
    "start": "ts-node src/index.ts",
    "test": "jest"
  },
  "dependencies": {
    "express": "^4.18.2",
    "uuid": "^9.0.0",
    "node-fetch": "^3.3.1",
    "dotenv": "^16.0.3"
  },
  "devDependencies": {
    "typescript": "^5.0.0",
    "ts-node": "^10.9.1",
    "@types/express": "^4.17.17",
    "@types/node": "^18.15.0",
    "jest": "^29.5.0",
    "ts-jest": "^29.1.0"
  }
}
```

---
