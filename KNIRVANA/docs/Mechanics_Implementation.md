# 🚀 Mechanics Implementation


## 1. Project Overview & Architecture

**Goal:** Transform the existing game pilot into a competitive multi-agent arena where agents compete to optimize a "Skill Slot" (LoRA adapter) via a human-in-the-loop adversarial process.

**Core Stack:**

* **Language:** TypeScript (Node.js)
* **Sandboxing:** `vm2` (for Verifier safety)
* **Backend Integration:** REST API (Fetch) to LoRAX/vLLM

---

## 2. Directory Structure

Refactor the `src` folder to support the new modular mechanics:

```text
src/
├── engine/
│   ├── Tournament.ts       # Main game loop & controller
│   ├── Verifier.ts         # Sandbox & scoring logic
│   └── Sabotage.ts         # Adversarial effect logic
├── types/
│   ├── Agent.ts            # Interfaces for Warriors
│   ├── Trajectory.ts       # JSON Schema for winning data
│   └── GameState.ts        # Resource tracking (Compute/Parity)
├── networking/
│   └── LoraxClient.ts      # API wrapper for fine-tuning backend
└── utils/
    └── MathUtils.ts        # For scoring formulas

```

---

## 3. Core Data Structures (`src/types/`)

### A. The Agent & Resources

**File:** `src/types/Agent.ts`

```typescript
export interface AgentResources {
    compute: number;      // "Mana" for sabotage/human actions
    parity: number;       // "Health" - reaching 0 causes divergence (elimination)
    generation: number;   // Track evolutionary steps
}

export interface RFTAgent {
    id: string;
    name: string;
    policy: 'greedy' | 'bayesian' | 'stochastic'; // Determines behavior profile
    resources: AgentResources;
    
    // The core function: given a state, produce a CoT and Code
    proposeSolution(errorNodeContext: string): Promise<AgentResponse>;
}

export interface AgentResponse {
    chainOfThought: string[];
    code: string;
    estimatedLatency?: number;
}

```

### B. The Trajectory Schema (The Winner's Payload)

**File:** `src/types/Trajectory.ts`

```typescript
export interface TrajectoryStep {
    step: number;
    thought: string;
    action: string; // The code snippet or token output
}

export interface WinnerPayload {
    agent_metadata: {
        agent_id: string;
        generation: number;
        victory_type: 'convergence' | 'hijack' | 'defensive';
    };
    prompt_context: string;
    trajectory: TrajectoryStep[];
    reward_signal: {
        score: number;       // 0.0 to 1.0
        latency_ms: number;
        verifier_feedback: string;
    };
}

```

To turn a winning trajectory into a permanent "Skill," the data needs to be structured so the **LoRAX** backend can ingest it for a supervised fine-tuning (SFT) or reward-weighted fine-tuning (RWFT) pass.

## 🏗️ The Trajectory JSON Schema

The "Trajectory" is the record of the Agent's "thought process" () and the final output that satisfied the Verifier. This schema ensures that every win is documented with enough context for the model to learn *why* the Agent won.

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "RFTWinnerTrajectory",
  "type": "object",
  "properties": {
    "agent_metadata": {
      "type": "object",
      "properties": {
        "agent_id": { "type": "string", "format": "uuid" },
        "generation": { "type": "integer" },
        "base_adapter": { "type": "string" }
      }
    },
    "prompt_context": {
      "type": "string",
      "description": "The state of the Error Node/Task given to the Agent."
    },
    "trajectory": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "step": { "type": "integer" },
          "thought": { "type": "string", "description": "Internal reasoning (CoT)" },
          "action": { "type": "string", "description": "The actual token output" }
        }
      }
    },
    "reward_signal": {
      "type": "object",
      "properties": {
        "score": { "type": "number", "minimum": 0, "maximum": 1 },
        "latency_ms": { "type": "integer" },
        "verifier_feedback": { "type": "string" }
      }
    }
  },
  "required": ["agent_metadata", "prompt_context", "trajectory", "reward_signal"]
}

```
## Example: A "Winner's Payload"

Imagine an Agent just solved a "Refactoring" node in the Arena. Here is what is sent to the `commit_to_lorax` function:

```json
{
  "agent_metadata": {
    "agent_id": "9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d",
    "generation": 42,
    "base_adapter": "skill-v41-optimizer"
  },
  "prompt_context": "Optimize this nested loop for O(n) complexity: [Code Snippet]",
  "trajectory": [
    {
      "step": 1,
      "thought": "The current nested loop creates O(n^2). I will use a Hash Map to store seen values.",
      "action": "const seen = new Map();"
    },
    {
      "step": 2,
      "thought": "Iterate once and check map for O(1) lookups.",
      "action": "for (let x of data) { if (seen.has(target - x)) return [seen.get(target - x), x]; }"
    }
  ],
  "reward_signal": {
    "score": 0.98,
    "latency_ms": 140,
    "verifier_feedback": "All unit tests passed. Time complexity verified as linear."
  }
}

```

### C. The Data Flow: From Victory to Weights

The following flow illustrates how the human's verifier and the agent's logic collide to update the LoRA adapter.

### How the Human "Gamifies" this Step:

* **Trajectory Pruning:** Before the data hits LoRAX, the human Architect can "edit" the trajectory. If the Agent found the right answer but its "thought process" was messy or redundant, the human can delete those steps to make the resulting Skill "Cleaner" and "Faster."
* **Distillation:** Humans can choose to combine the top 3 winning trajectories into a single "Batch Update," creating a **Ensemble Skill** that is more robust than any single Agent's solution.


## Agent Training

Your AI coding agent needs to add a `TrainingManager` to handle these intermediate states.

```typescript
// src/engine/TrainingManager.ts

export class TrainingManager {
    /**
     * The Generalize mechanic: Reduces latency by trimming the trajectory.
     */
    public distill(trajectory: TrajectoryStep[]): TrajectoryStep[] {
        // Implementation: Logic to remove steps with low 'importance' scores
        // Human Architects trigger this via the UI to 'clean' their DNA.
        return trajectory.filter(step => step.thought.length > 10); 
    }

    /**
     * The Hibernate mechanic: Trades speed for Parity.
     */
    public harden(agent: RFTAgent) {
        agent.resources.parity += 20; // Build defense
        agent.resources.compute += 50; // Generate passive income
    }
}

```
---

## 4. The Verifier Engine (`src/engine/Verifier.ts`)

To make the game truly competitive, the Verifier can’t just be a "Pass/Fail" check. It needs to be a **Grader** that evaluates efficiency, elegance, and resilience against the "Noise" other players have injected.This is the "Physics Engine" of the game. It must support dynamic weight adjustment and sandboxed execution.

**Requirements:**

1. Use `vm2` to safely execute Agent code.
2. Implement the weighted scoring formula.
3. Allow "Hot Swapping" of test constraints (Human Interaction).

```typescript
import { VM } from 'vm2';

export interface ScoreWeights {
    correctness: number; // e.g., 0.6
    latency: number;     // e.g., 0.3
    simplicity: number;  // e.g., 0.1
}

export class Verifier {
    private weights: ScoreWeights = { correctness: 0.6, latency: 0.3, simplicity: 0.1 };
    private constraints: Map<string, (res: any) => boolean> = new Map();

    /**
     * Updates the physics of the arena.
     * Called by Human Player via UI.
     */
    public updateWeights(newWeights: ScoreWeights) {
        this.weights = newWeights;
    }

    /**
     * Adds a "Trap" or specific test case.
     */
    public addConstraint(id: string, validator: (res: any) => boolean) {
        this.constraints.set(id, validator);
    }

    public async evaluate(code: string, context: any): Promise<number> {
        const vm = new VM({ timeout: 1000, sandbox: { context } });
        const start = performance.now();
        
        try {
            // 1. EXECUTION & LATENCY CHECK
            const result = vm.run(code);
            const duration = performance.now() - start;
            
            // Calculate Score based on this.weights
            // 2. SCORING LOGIC
            // Baseline: Did it even work?
            if (this.deepCompare(result, this.requirements.expectedOutput)) {
                score += 0.6; // 60% for correctness
            }

            // Efficiency Bonus: The "Core War" speed factor
            const latencyBonus = Math.max(0, (this.requirements.maxLatency - duration) / 200) * 0.2;
            score += latencyBonus;

            // Complexity/Length Penalty: Rewards "Clean" Trajectories
            const lengthPenalty = Math.min(0.2, agentCode.length / 5000) * -1;
            score += (0.2 + lengthPenalty);
            
            return finalScore;
        } catch (e) {
            console.error("Agent Crashed or Poisoned:", e);
            return 0; // Total failure
        }

        return Math.min(1.0, Math.max(0, score));
    }

    private deepCompare(a: any, b: any): boolean {
        return JSON.stringify(a) === JSON.stringify(b);
    }
}

```

---

## 5. Adversarial Mechanics (`src/engine/Sabotage.ts`)

Implement the functions that allow players to spend **Compute** to disrupt others.

```typescript
import { RFTAgent } from '../types/Agent';

export enum SabotageType {
    NOISE_INJECTION = 'NOISE_INJECTION', // Adds random chars to context
    BACKPROP_PULSE = 'BACKPROP_PULSE',   // Reduces target Parity
    GRADIENT_GHOSTING = 'GRADIENT_GHOSTING' // Creates fake high-reward target
}

export class SabotageEngine {
    
    public static applyEffect(type: SabotageType, target: RFTAgent, magnitude: number) {
        switch (type) {
            case SabotageType.NOISE_INJECTION:
                // Logic: Return a "Decorator" function that corrupts the input string
                // for the target's next Inference Step.
                break;
                
            case SabotageType.BACKPROP_PULSE:
                // Logic: Directly reduce Parity
                target.resources.parity -= (10 * magnitude);
                break;
        }
    }
}

```

---

## 6. The Tournament Loop (`src/engine/Tournament.ts`)

The main controller that ties it all together.

**Logic Flow:**

1. **Load Phase:** Fetch current Error Node state.
2. **Sabotage Phase:** Check queue for active sabotage effects.
3. **Inference:** Call `proposeSolution` on all active Agents.
4. **Verification:** Pass results to `Verifier`.
5. **Ranking:** Sort by Reward Score.
6. **Red Queen Check:**
* IF `Winner.score > Incumbent.score + Threshold` THEN `Hijack`.


7. **Reinforcement:** Call `LoraxClient` to update weights.

```typescript
import { Verifier } from './Verifier';
import { LoraxClient } from '../networking/LoraxClient';
import { RFTAgent } from '../types/Agent';

export class Tournament {
    private verifier: Verifier;
    private lorax: LoraxClient;
    private skillSlotOwner: string | null = null; // Agent ID
    private incumbentScore: number = 0.8; // Baseline to beat

    public async runEpoch(agents: RFTAgent[], nodeContext: string) {
        // 1. Inference
        const proposals = await Promise.all(agents.map(a => a.proposeSolution(nodeContext)));

        // 2. Verification
        const results = await Promise.all(proposals.map(async (p, index) => {
            const score = await this.verifier.evaluate(p.code, nodeContext);
            return { agent: agents[index], proposal: p, score };
        }));

        // 3. Selection
        results.sort((a, b) => b.score - a.score);
        const winner = results[0];

        // 4. Digital Red Queen Mechanic
        if (winner.score > this.incumbentScore) {
            console.log(`🚀 SKILL SLOT HIJACKED by ${winner.agent.id}`);
            this.skillSlotOwner = winner.agent.id;
            this.incumbentScore = winner.score; // The bar is raised

            // 5. Reinforcement
            await this.lorax.fineTune(winner.agent, winner.proposal, winner.score);
        }
    }
}

```

---

## 7. LoRAX Integration (`src/networking/LoraxClient.ts`)

Standardize the API calls to the inference/training backend.

```typescript
import { WinnerPayload } from '../types/Trajectory';

export class LoraxClient {
    private baseUrl: string;

    constructor(url: string) {
        this.baseUrl = url;
    }

    public async fineTune(agent: any, proposal: any, score: number) {
        const payload: WinnerPayload = {
            agent_metadata: {
                 agent_id: agent.id, 
                 generation: agent.resources.generation,
                 victory_type: 'convergence' 
            },
            prompt_context: "...",
            trajectory: [ 
                { step: 1, thought: proposal.chainOfThought, action: proposal.code } 
            ],
            reward_signal: { score: score, latency_ms: 0, verifier_feedback: "Success" }
        };

        // POST to LoRAX backend
        await fetch(`${this.baseUrl}/adapters`, {
            method: 'POST',
            body: JSON.stringify({
                adapter_id: `skill-${agent.id}`,
                action: 'fine_tune',
                data: payload
            })
        });
    }
}

```

### The API Handshake (Simplified)

When the **Tournament Controller** decides a winner, it sends this command to the **LoRAX Server**:

```bash
# The Controller tells LoRAX to "Bake" the winning trajectory
curl -X POST http://lorax-backend:8080/adapters \
  -d '{
    "adapter_id": "architect-v1-pollards-rho",
    "parent_model": "base-llm-13b",
    "action": "commit_to_main_slot"
  }'

```


## 8. The Human Interface API (`src/api/GameServer.ts`)

To make this playable, the game engine needs a REST/WebSocket layer so the Human UI (The Dashboard) can send commands to the Tournament Controller.

**Responsibilities:**

1. Handle "Reward Sculpting" (Weight updates).
2. Handle "Sabotage" commands (spending Compute).
3. Stream real-time game state to the frontend.

```typescript
import express from 'express';
import { Tournament } from '../engine/Tournament';
import { Verifier } from '../engine/Verifier';
import { SabotageEngine, SabotageType } from '../engine/Sabotage';

export class GameServer {
    private app = express();
    private tournament: Tournament;
    private verifier: Verifier;

    constructor(tournament: Tournament, verifier: Verifier) {
        this.tournament = tournament;
        this.verifier = verifier;
        this.setupRoutes();
    }

    private setupRoutes() {
        this.app.use(express.json());

        // 1. HUMAN ACTION: Update Reward Weights
        this.app.post('/architect/weights', (req, res) => {
            const { weights } = req.body;
            // Validates that weights sum to roughly 1.0 or valid range
            this.verifier.updateWeights(weights); 
            console.log(`[UI] Architect updated weights: ${JSON.stringify(weights)}`);
            res.sendStatus(200);
        });

        // 2. HUMAN ACTION: Execute Sabotage
        this.app.post('/architect/sabotage', (req, res) => {
            const { targetAgentId, type, magnitude } = req.body;
            const target = this.tournament.getAgent(targetAgentId);
            
            if (target) {
                SabotageEngine.applyEffect(type as SabotageType, target, magnitude);
                res.json({ success: true, message: `${type} deployed against ${targetAgentId}` });
            } else {
                res.status(404).json({ error: "Agent not found" });
            }
        });

        // 3. UI POLLING: Get Dashboard State
        this.app.get('/state', (req, res) => {
            res.json({
                epoch: this.tournament.currentEpoch,
                incumbent: this.tournament.skillSlotOwner,
                incumbentScore: this.tournament.incumbentScore,
                agents: this.tournament.getAgentStatuses() // Returns IDs, Parity, Compute
            });
        });
    }

    public start(port: number) {
        this.app.listen(port, () => console.log(`🎮 KNIRVANA Server running on port ${port}`));
    }
}

```

---

## 9. Global Game State (`src/types/GameState.ts`)

We need a unified store for the "World Data" that persists between HTTP requests and game ticks.

```typescript
export interface ErrorNode {
    id: string;
    description: string;   // The task prompt
    baseContext: string;   // Shared context (can be poisoned)
    difficulty: number;    // 1-10
}

export interface PlayerState {
    playerId: string;
    role: 'Architect';
    resources: {
        compute: number;
        parity: number;
    };
    // Cooldowns for special abilities
    cooldowns: {
        sabotage: number;
        shield: number;
    };
}

export interface WorldState {
    currentNode: ErrorNode;
    activeEpoch: number;
    isPaused: boolean;
    weather: 'CLEAR' | 'FOG_OF_OVERFITTING'; // Environmental modifiers
}

```

---

## 10. Implementation Roadmap (Step-by-Step)

For the AI Coding Agent, follow this strict implementation order to ensure dependencies are met.

### Phase 1: The Physics Engine (The Verifier)

1. **Task:** Create `src/engine/Verifier.ts`.
2. **Validation:** Write a unit test that feeds it a correct JS function and asserts a score > 0.6.
3. **Validation:** Write a unit test that feeds it an infinite loop and asserts it terminates via `vm2` timeout with score 0.

### Phase 2: The Warriors (Agents)

1. **Task:** Create `src/types/Agent.ts` and a mock `GreedyAgent` class.
2. **Validation:** Ensure the agent can return a valid string response when `proposeSolution()` is called.

### Phase 3: The Arena (Tournament Loop)

1. **Task:** Create `src/engine/Tournament.ts`.
2. **Logic:** Connect the Agent output to the Verifier input.
3. **Validation:** Run a simulation where two dummy agents compete. Ensure the one with the higher mock score is stored as `skillSlotOwner`.

### Phase 4: The Network (LoRAX & API)

1. **Task:** Implement `src/networking/LoraxClient.ts` (Mock the actual fetch call for now).
2. **Task:** Implement `src/api/GameServer.ts`.
3. **Validation:** Use `curl` or Postman to hit `POST /architect/weights` and verify `Verifier.weights` changes in memory.

---

## 11. Configuration & Dependencies

Create a `package.json` with these core dependencies:

```json
{
  "name": "error-node-engine",
  "version": "0.1.0",
  "scripts": {
    "start": "ts-node src/index.ts",
    "test": "jest"
  },
  "dependencies": {
    "express": "^4.18.2",
    "vm2": "^3.9.19",  // CRITICAL: For sandboxing
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

## 12. Persistence & History (`src/engine/HistoryManager.ts`)

To make the RFT loop meaningful, we must track the evolution of the model. This allows players to see the "Lineage" of the current Skill Slot.

```typescript
import fs from 'fs/promises';
import { WinnerPayload } from '../types/Trajectory';

export class HistoryManager {
    private historyPath = './data/epoch_history.json';

    /**
     * Records every successful Hijack for future training passes.
     */
    public async recordVictory(payload: WinnerPayload) {
        const history = await this.loadHistory();
        history.push({
            timestamp: new Date().toISOString(),
            ...payload
        });
        await fs.writeFile(this.historyPath, JSON.stringify(history, null, 2));
    }

    private async loadHistory(): Promise<any[]> {
        try {
            const data = await fs.readFile(this.historyPath, 'utf-8');
            return JSON.parse(data);
        } catch {
            return [];
        }
    }
}

```

---

## 13. Adversarial Synchronization (The Middleware)

The game must ensure that **Sabotage** effects (like Noise) are actually injected into the prompt before the Agent sees it.

**Logic Flow in `Tournament.ts`:**

```typescript
// Inside the runEpoch method:
const finalProposals = await Promise.all(agents.map(async (agent) => {
    let context = worldState.currentNode.baseContext;

    // Apply active Sabotage effects from the SabotageEngine
    if (SabotageEngine.hasActiveEffect(agent.id, SabotageType.NOISE_INJECTION)) {
        context = SabotageEngine.injectNoise(context); 
    }

    return agent.proposeSolution(context);
}));

```

---

## 14. Final Integration Checklist for the Coding Agent

Before concluding the build, the AI agent must verify the following "Adversarial Integrity" points:

1. **Parity Decay:** Does an Agent's `parity` resource actually drop when they fail a verifier check?
* *Implementation:* In `Tournament.ts`, add a decrement logic: `agent.resources.parity -= 5` on every failed epoch.


2. **Compute Regeneration:** Does the `skillSlotOwner` receive a bonus?
* *Implementation:* At the end of each `runEpoch`, loop through all agents and add `+10` compute, but `+20` for the current owner.


3. **LoRAX Error Handling:** If the LoRAX backend is down, does the game pause gracefully?
* *Implementation:* Wrap the `LoraxClient` call in a try/catch that sets `worldState.isPaused = true`.



---

# 🏁 Final Agent Instructions (The Handover)

> "You have the full specification. Your final objective is to implement the **HistoryManager** and integrate the **SabotageEngine** into the main **Tournament** loop.
> **Final Task:** Ensure that the `src/index.ts` file correctly initializes the `Tournament`, `Verifier`, and `GameServer` in the correct order. The server should start, and the tournament loop should begin running on a `setInterval` or a recursive `async` loop.
> Once implemented, provide a sample `curl` command that a human can use to sabotage an agent during a live epoch."

---

