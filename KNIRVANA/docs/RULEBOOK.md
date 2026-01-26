# 📜 RULEBOOK: ERROR NODE (v1.0)

## I. The Objective

Your Agent must navigate the **KNIRVANA** (a 3D landscape of Loss) to find the **Global Minimum** (). The first Agent to maintain a stable position at the Global Minimum for 3 consecutive "Epochs" wins.

---

## II. The Arena: The Error Topography

The Arena is a grid-based landscape where the elevation represents **Loss** ().

* **Peaks (Red Zones):** High Error. Agents take "Damage" (Compute Loss) here.
* **Valleys (Blue Zones):** Low Error. Agents regain "Stability."
* **The Fog of Overfitting:** Areas where the terrain looks like a goal but is actually a trap that resets Agent progress.

---

## III. Player Roles & Resources

You play as the **Architect**. You manage two primary resources:

1. **Compute ():** Your "Mana." Used to accelerate your Agent or sabotage others.
2. **Parity ():** Your "Health." If your Agent's weights become too chaotic (Divergence), you are disconnected from the Arena.

---

## IV. The Optimization Loop (Turn Phases)

Each round consists of three distinct phases:

### Phase 1: The Inference Step (Movement)

Agents move simultaneously based on their current **Policy**.

* **Human Activity:** You do not control the move. You observe the path.
* **Mechanic:** Agents naturally move toward lower elevation unless distracted by "Noise."

### Phase 2: The Reward Sculpting (Human Intervention)

This is where the human plays. You have 30 seconds to "Sculpt" the reward signal for the next move.

* **Weight Drop:** Place a "Reward Anchor" on a tile to lure your Agent.
* **Gradient Grease:** Make a slope steeper, forcing any Agent on it to slide 2 tiles down (faster movement, but less control).
* **Anchor Bias:** Force an opponent to recalculate their path, costing them **Compute**.

### Phase 3: The Backprop Pulse (The Attack)

If your Agent ended the turn in a lower elevation than an opponent, you may trigger a **Backprop Pulse**.

* **Effect:** You "leak" your error into their weights.
* **Math:** The opponent's Learning Rate () is modified by:


* *Translation:* You make their Agent "stumble" or over-correct, potentially sending them flying off a cliff.

---

## V. Advanced Human Tactics (The "Spells")

| Tactic | Cost | Effect |
| --- | --- | --- |
| **Stochastic Burst** | 10 Compute | Your Agent ignores terrain for 1 turn (Teleport) but loses 5 Parity. |
| **Dropout Shield** | 5 Compute | Makes your Agent invisible to opponent "Backprop Pulses" for one Epoch. |
| **Local Minima Trap** | 15 Compute | Create a fake "Goal" tile. If an opponent lands there, they are stuck for 2 turns. |
| **LR Decay** | 8 Compute | Permanently slow down an opponent's movement speed by 10%. |

---

## VI. Winning & Losing

* **The Convergence Victory:** Stay at the Global Minimum for 3 Epochs.
* **The Disconnection Defeat:** If your **Parity** reaches 0, your Agent "Hallucinates" and wanders off the map.
* **The Compute Crunch:** If you run out of Compute, you can no longer intervene, and your Agent runs on "Auto-Pilot" (Static Policy).

---

## VII. The "RLHF" Joker Card

Once per match, a human can trigger **Manual Alignment**.

> **Activity:** You are shown three potential paths. You pick the "Best" one. Your Agent receives a permanent +50% speed boost toward that coordinate and ignores all "Local Minima Traps" on the way.

---


## 📜 UPDATED RULEBOOK: THE RED QUEEN ARENA

### VIII. The Core Objective: "The Skill Slot"

There is only **one** active LoRA adapter slot (The "Throne"). Your goal is not just to solve the KNIRVANA, but to **occupy the slot** by submitting a trajectory that is mathematically superior to the current incumbent.

### IX. Digital Red Queen Dynamics

In this Arena, the "Ground Truth" is a moving target.

* **Weight Hijacking:** If your Agent produces a solution that is **shorter** (less token latency) or **more accurate** than the current Skill owner, your LoRA immediately replaces theirs.
* **The Reward Decay:** Every round you hold the Slot, the "Success Threshold" () increases by . You must constantly innovate just to keep your position.

### X. Sabotage Mechanics (Core War Style)

Human Architects can now use **Context Poisoning** to disrupt the RFT loop:

* **Noise Injection:** Spend 20 Compute to inject "Garbage Tokens" into the shared context. This forces opponent Agents to waste "Inference Steps" filtering out the noise.
* **Gradient Ghosting:** Create a "fake" high-reward path. If an opponent Agent commits to it, their reward score is penalized in the Evaluation Phase, preventing them from hijacking the Skill Slot.

### XI. Human Activity: "The Verifier Architect"

As a human player, you are the **Judge**. You can live-edit the `evaluate_solution` logic (The Verifier).

* **Activity:** If an opponent is winning through "Cheats" (e.g., finding a shortcut that bypasses the logic you want), you can update the **Unit Tests** in real-time to penalize that specific trajectory.
* **Strategic Play:** Shift the goalposts. Change the requirements from "Speed" to "Reasoning Depth" to knock a faster, dumber Agent out of the Skill Slot.

---