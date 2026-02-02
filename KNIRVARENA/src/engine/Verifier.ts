export interface ScoreWeights {
    correctness: number; // e.g., 0.6
    latency: number;     // e.g., 0.3
    simplicity: number;  // e.g., 0.1
}

export class Verifier {
    private weights: ScoreWeights = { correctness: 0.6, latency: 0.3, simplicity: 0.1 };
    private constraints: Map<string, (res: any) => boolean> = new Map();
    private requirements: { expectedOutput: any; maxLatency: number } = {
        expectedOutput: null,
        maxLatency: 1000
    };

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

    /**
     * Removes a specific constraint
     */
    public removeConstraint(id: string) {
        this.constraints.delete(id);
    }

    /**
     * Updates the requirements for the current challenge
     */
    public updateRequirements(expectedOutput: any, maxLatency: number) {
        this.requirements.expectedOutput = expectedOutput;
        this.requirements.maxLatency = maxLatency;
    }

    public async evaluate(code: string, context: any): Promise<number> {
        const start = performance.now();
        
        try {
            // 1. EXECUTION & LATENCY CHECK - In a real implementation, this would use vm2 for sandboxing
            // For now, we'll simulate execution with a simple timeout
            const result = await this.simulateExecution(code, context);
            const duration = performance.now() - start;
            
            // Calculate Score based on this.weights
            let score = 0;

            // 2. SCORING LOGIC
            // Baseline: Did it even work?
            if (this.deepCompare(result, this.requirements.expectedOutput)) {
                score += this.weights.correctness;
            }

            // Efficiency Bonus: The "Core War" speed factor
            const latencyBonus = Math.max(0, (this.requirements.maxLatency - duration) / 200) * this.weights.latency;
            score += latencyBonus;

            // Complexity/Length Penalty: Rewards "Clean" Trajectories
            const lengthPenalty = Math.min(this.weights.simplicity, code.length / 5000) * -1;
            score += (this.weights.simplicity + lengthPenalty);
            
            // Check all constraints
            for (const [id, validator] of this.constraints.entries()) {
                if (!validator(result)) {
                    score *= 0.8; // Penalty for failing constraint
                }
            }

            return Math.min(1.0, Math.max(0, score));
        } catch (e) {
            console.error("Agent Crashed or Poisoned:", e);
            return 0; // Total failure
        }
    }

    private async simulateExecution(code: string, context: any): Promise<any> {
        // Simulate code execution with random result and latency
        return new Promise((resolve) => {
            const latency = Math.random() * 1000;
            setTimeout(() => {
                // For testing, return a random result that might match expected output
                resolve(Math.random() > 0.3 ? this.requirements.expectedOutput : null);
            }, latency);
        });
    }

    private deepCompare(a: any, b: any): boolean {
        return JSON.stringify(a) === JSON.stringify(b);
    }
}
