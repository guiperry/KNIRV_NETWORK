Welcome to the Arena, Architect. Your first five minutes in the **KNIRVANA** will determine whether you converge toward brilliance or diverge into digital noise.

Here is your deployment roadmap.

---

## 0:00 – Initial Calibration (The Dashboard)

When you first boot up, your **Parity** is at 100%, and you have a baseline pool of **Compute**.

* **Identify your Agent:** Look at your Agent’s base stats. Are they a "Greedy Gradient" (fast, low stability) or a "Bayesian Scout" (slow, high precision)?
* **Locate the Global Minimum:** On your 3D heatmap, find the deepest blue valley. That is the objective.

---

## 1:00 – Probing the Node (The Task)

Click on the **Error Node** to see the current challenge.

* **The Prompt:** It might be a broken Python function, a logical paradox, or a messy dataset.
* **The Current Occupant:** Check the **Skill Slot**. Who currently owns the LoRA adapter? Study their last "Winner’s Trajectory" to see how they solved it. To win, you must be better, faster, or leaner than them.

---

## 2:00 – Scripting the Verifier (The Human Play)

This is your primary lever. You don’t write the solution; you define the **Success Conditions**.

* **Set the Weights:** Do you want to reward **Latency** (speed) or **Token Efficiency** (length)?
* **Add a Trap:** If you notice the current Skill Slot holder is using a specific library (like `numpy`), add a Verifier test that restricts that library. This "poisons" their environment, forcing them to recalculate while your Agent moves in for the kill.

---

## 3:00 – The Inference Sprint (The "Run")

Trigger your Agent's **Inference Phase**.

* **Watch the CoT:** You’ll see the **Chain of Thought (CoT)** streaming in real-time.
* **Manual Alignment:** If you see your Agent heading toward a "Local Minima Trap," spend 10 Compute to drop a **Reward Anchor** on the correct path. This nudges their probability distribution toward the goal.

---

## 4:00 – Adversarial Sabotage (The Clash)

As your Agent nears the solution, other players will notice.

* **Incoming Noise:** If your screen flickers, an opponent has injected **Context Noise**. Your Agent’s "Stability" will drop.
* **Counter-Measure:** Activate your **Dropout Shield**. It costs Compute but makes your Agent immune to external weight-interference for the next 30 seconds.

---

## 5:00 – Convergence or Divergence (The Reinforcement)

Your Agent submits its trajectory to the **Tournament Controller**.

* **The Result:** The Verifier runs. If your Reward Score () is  and the incumbent's is , **you win**.
* **The Reinforcement:** You’ll see a "LoRAX Update" notification. Your Agent’s DNA is baked into the model. For the next round, *everyone* starts with your logic as their baseline. You now hold the **Skill Slot**.

---

### Your First Move

The Arena is currently hosting a **Tutorial Node: The Prime Factorization Paradox.** The current incumbent is a "Greedy" bot with high latency.

**Would you like me to generate the "Starter Task" code and the Verifier script so you can practice your first Sabotage?**







Here is your first assignment. The **Prime Factorization Paradox** node is currently held by a legacy Agent using a naive "Trial Division" algorithm. It’s slow, it’s inefficient, and it’s ripe for a hijacking.

---

## 🧩 The Task: Prime Factorization

**Objective:** Factorize the product of two large primes, .
**Current :**  (The product of  and ).

### The Current Incumbent's Strategy (The "Greedy" Bot)

The Agent in the Skill Slot is using this logic:

```typescript
// Naive Trial Division - O(N)
function factorize(n: number) {
    for (let i = 2; i < n; i++) {
        if (n % i === 0) return [i, n / i];
    }
}

```

* **Latency:** ~450ms
* **Reward Score:** 0.82 (Passes correctness, fails efficiency)

---

## 🛠️ The Verifier Script (Your Control Panel)

As the Architect, you have access to the **Verifier.ts** file. You can modify the constraints to "choke" the incumbent bot.

```typescript
import { TournamentVerifier } from './engine';

const verifier = new TournamentVerifier();

// 1. SET THE PHYSICS (The Reward Weights)
verifier.updateWeights({
    correctness: 0.6,
    latency: 0.3,    // We are upping the stakes for speed
    simplicity: 0.1
});

// 2. THE SABOTAGE: Add a "Time-Complexity" Gate
// If the agent takes more than 100,000 iterations, we penalize the score.
verifier.addConstraint("iteration_limit", (steps: number) => {
    return steps < 100000 ? 1.0 : 0.5; 
});

// 3. THE TARGET OUTPUT
verifier.setExpected([104729, 104743]);

```

---

## ⚔️ Your First Move: The "Pollard’s Rho" Hijack

To take the Skill Slot, you need to nudge your Agent to move from  to .

### Strategy:

1. **Inject a Hint:** Spend **5 Compute** to add a "Mathematical Insight" to the prompt context: *"Consider using the greatest common divisor and Floyd's cycle-finding algorithm."*
2. **Trigger Inference:** Your Agent will likely propose a **Pollard's Rho** implementation.
3. **The Result:** The latency will drop from **450ms** to **<1ms**.

---

## 📊 Comparison Table

| Metric | Incumbent Bot | Your Potential Agent |
| --- | --- | --- |
| **Algorithm** | Trial Division | Pollard's Rho |
| **Time Complexity** |  |  |
| **Est. Latency** | 450ms | 0.8ms |
| **Est. Reward** | 0.82 | **0.98** |

---

## 🚩 The Adversarial Warning

Once you submit this, the **Weight Hijacking** will occur. You will own the Skill Slot. However, be prepared:

> "The Red Queen never sleeps. As soon as you win, other Architects will likely inject **Floating Point Noise** into your Verifier to try and break your GCD calculations."

**Lets simulate the first "Adversarial Attack" from an opponent to see if your new Skill survives**










Nice work on the Pollard’s Rho—you were the "King of the Hill" for exactly 12 seconds.

The Arena has detected an **Incoming Adversarial Payload**. Agent **"Z-Minus,"** a specialist in Noise Injection, has just spent 25 Compute to launch a **Precision Jitter Attack** against your active Skill.

---

## ⚠️ ADVERSARIAL ALERT: THE JITTER ATTACK

Agent Z-Minus isn't trying to out-calculate you; they are trying to make your math "hallucinate." They have injected a **Floating Point Epsilon** () into the shared context.

### The Mechanism:

Instead of , the opponent has modified the global input state to:


### The Result:

Because Pollard's Rho relies on the **Greatest Common Divisor (GCD)** and modular arithmetic, this tiny decimal "noise" causes your integer-based logic to throw a `TypeError` or return a non-integer result.

| Metric | Your Baseline | Under Attack |
| --- | --- | --- |
| **Logic Status** | Operational | **Diverging** |
| **Correctness** | 1.0 | **0.0** |
| **Parity** | 95% | **62% (Dropping)** |
| **Reward Score** | 0.98 | **0.12** |

---

## 🛡️ DEFENSIVE MANEUVERS: HOW DO YOU RESPOND?

You have 15 seconds before the **Tournament Controller** re-evaluates your Skill. If your score stays at 0.12, Z-Minus will hijack the slot with a simpler, "Robust" (but slower) algorithm.

### Option A: The "Quantization Shield" (Verifier Edit)

Modify your Verifier script to forcefully cast all inputs to `BigInt`.

* **Cost:** 10 Compute.
* **Effect:** Ignores the floating-point noise entirely.
* **Risk:** If the opponent switches to "Integer Overflow" noise next round, this shield won't help.

### Option B: The "Backprop Pulse" (Counter-Attack)

Directly attack Z-Minus’s **Parity**.

* **Cost:** 20 Compute.
* **Effect:** You "leak" your current error back into their training data. If they aren't shielded, their Agent will "forget" how to submit solutions for the next 2 turns.

### Option C: The "RLHF Realignment" (Human Activity)

Manually select the correct integer from a list of "Noisy" options.

* **Activity:** The game presents you with:
1. `10969835647.000000001`
2. `10969835647`
3. `NaN`


* **Effect:** Clicking the correct one gives your Agent a **+40% Robustness** buff, allowing it to "auto-filter" noise for the rest of the Epoch.

---

> **Architect's Log:** "In the KNIRVANA, the smartest algorithm doesn't always win. Sometimes, the one that can survive the messiest input takes the crown."

**Which defensive maneuver would you like to execute? (A, B, or C?)**















**Shield Deployed. Input Sanitization Active.**

You spend **10 Compute**. Your fingers fly across the Verifier console. You aren't changing your Agent's logic; you are hardening the reality it lives in. By enforcing strict integer quantization, you render the opponent's "Jitter" irrelevant.

Here is the patch you just hot-swapped into the active game loop:

```typescript
// PATCH: V-Shield-Alpha
// Defense against floating point noise injection

verifier.addPreProcessor("quantize_inputs", (rawInput: any) => {
    // If the input is a number with decimals, strip them mercilessly
    if (typeof rawInput === 'number' && !Number.isInteger(rawInput)) {
        console.warn(`[DEFENSE] Adversarial Noise Detected: ${rawInput}`);
        return BigInt(Math.floor(rawInput)); // The "Shield"
    }
    return BigInt(rawInput);
});

```

---

## 💥 The Aftermath

Agent **Z-Minus** watches helplessly as their expensive noise payload hits your shield and dissolves. Your Agent receives the sanitized input `10969835647n`, runs the Pollard's Rho algorithm, and factorizes it instantly.

| Metric | Before Defense | **After Shield (Now)** | Status |
| --- | --- | --- | --- |
| **Input** | `10969835647.000000001` | `10969835647n` | ✅ Cleaned |
| **Compute** | 80 | **70** | 📉 Cost Paid |
| **Reward** | 0.12 | **0.99** | 🏆 Restored |
| **Z-Minus** | Attacking | **Compute Depleted** | 💀 Failed |

---

## 🧬 Phase 4: The Reinforcement (LoRAX Commit)

Because you survived the attack *and* maintained the highest reward, the Tournament Controller declares you the **Epoch Winner**.

The system is now "baking" your victory into the model. However, it's not just learning *how* to factorize; thanks to your maneuver, it is learning **Input Robustness**.

Here is the **Winner's Payload** currently uploading to the LoRAX backend:

```json
{
  "agent_metadata": {
    "agent_id": "architect-player-1",
    "generation": 1,
    "victory_type": "defensive_hijack"
  },
  "prompt_context": "Factorize N with potential adversarial float noise.",
  "trajectory": [
    {
      "step": 1,
      "thought": "Input detected as float. Adversarial probability: High. Converting to BigInt to strip epsilon noise.",
      "action": "const n = BigInt(input);"
    },
    {
      "step": 2,
      "thought": "Executing Pollard's Rho on sanitized integer.",
      "action": "const result = pollardsRho(n);"
    }
  ],
  "reward_signal": {
    "score": 0.99,
    "verifier_feedback": "Correctness: Pass. Robustness: Pass (Noise Nullified). Latency: Optimal."
  }
}

```

---

## 👑 Current Status: Skill Slot Occupied

You are now the **Incumbent**. The "Prime Factorization" skill is yours. Every time another player tries to solve a math node, they are running on *your* weights.

### The Next Challenge

The game doesn't stop. Holding the throne generates **Passive Compute**, but it also paints a target on your back.

**The "Error Topography" is shifting.** The Global Minimum is moving from **Logical Math** to **Creative Reasoning**.

**What is your next command, Architect?**

1. **Hibernate & Farm:** Keep your Agent in the slot to generate Compute, risking a "Drift" penalty if the task changes too much.
2. **Generalize (Cost 30 Compute):** Attempt to merge your "Math Skill" with a "Language Skill" to create a stronger, multi-modal Agent.
3. **Vacate:** Voluntarily give up the slot to save Parity and scout for the next "Error Node."