import type { Challenge } from '../types/challenge';
import type { GameLLMService, SolutionProposal, AgentPersona } from '../services/gameLLMService';

export interface ScoreWeights {
  correctness: number; // e.g., 0.6
  latency: number;     // e.g., 0.3
  simplicity: number;  // e.g., 0.1
}

export class Verifier {
  private weights: ScoreWeights = { correctness: 0.6, latency: 0.3, simplicity: 0.1 };
  private constraints: Map<string, (res: any) => boolean> = new Map();
  private llmService: GameLLMService | null = null;

  /**
   * Wire in the LLM judge. Call once after construction.
   */
  public setLLMService(service: GameLLMService) {
    this.llmService = service;
  }

  /**
   * Updates scoring weights (called by player via UI reward anchors).
   */
  public updateWeights(newWeights: ScoreWeights) {
    this.weights = newWeights;
  }

  /**
   * Adds a constraint/trap.
   */
  public addConstraint(id: string, validator: (res: any) => boolean) {
    this.constraints.set(id, validator);
  }

  public removeConstraint(id: string) {
    this.constraints.delete(id);
  }

  /**
   * Core evaluation — uses LLM-as-judge when available,
   * falls back to heuristic scoring when not.
   *
   * @param proposal  The structured proposal from the agent
   * @param challenge The challenge being solved
   * @param persona   The agent's persona (for LLM judge context)
   */
  public async evaluateProposal(
    proposal: SolutionProposal,
    challenge: Challenge,
    persona: AgentPersona
  ): Promise<number> {
    const start = performance.now();

    try {
      let correctnessScore: number;
      let reasoning = '';

      if (this.llmService) {
        // LLM-as-judge path
        const evalResult = await this.llmService.evaluate(challenge, proposal, persona);
        correctnessScore = evalResult.correctness;
        reasoning = evalResult.reasoning;
      } else {
        // Heuristic fallback: score based on solution length and key terms
        correctnessScore = this.heuristicScore(proposal.solution, challenge);
      }

      const duration = performance.now() - start;

      // Correctness (LLM judge or heuristic)
      let score = correctnessScore * this.weights.correctness;

      // Latency bonus: faster proposals get more points
      const maxLatency = 5000;
      const latencyBonus = Math.max(0, (maxLatency - proposal.estimatedLatency) / maxLatency) * this.weights.latency;
      score += latencyBonus;

      // Simplicity bonus: shorter, cleaner solutions score better
      const solutionLength = proposal.solution.length;
      const simplicityScore = Math.max(0, 1 - solutionLength / 3000);
      score += simplicityScore * this.weights.simplicity;

      // Apply any active constraints
      for (const [, validator] of this.constraints.entries()) {
        if (!validator({ proposal, challenge, reasoning })) {
          score *= 0.8;
        }
      }

      return Math.min(1.0, Math.max(0, score));
    } catch (e) {
      console.error('[Verifier] Evaluation failed:', e);
      return 0;
    }
  }

  /**
   * Legacy evaluate method for backwards compatibility.
   * Used when raw code string + context string are passed directly.
   */
  public async evaluate(code: string, context: string): Promise<number> {
    const start = performance.now();

    try {
      const result = await this.simulateExecution(code, context);
      const duration = performance.now() - start;

      let score = 0;

      if (result !== null) {
        score += this.weights.correctness;
      }

      const latencyBonus = Math.max(0, (1000 - duration) / 200) * this.weights.latency;
      score += latencyBonus;

      const lengthPenalty = Math.min(this.weights.simplicity, code.length / 5000) * -1;
      score += this.weights.simplicity + lengthPenalty;

      for (const [, validator] of this.constraints.entries()) {
        if (!validator(result)) {
          score *= 0.8;
        }
      }

      return Math.min(1.0, Math.max(0, score));
    } catch (e) {
      console.error('[Verifier] Legacy evaluate crashed:', e);
      return 0;
    }
  }

  // ── Private helpers ───────────────────────────────────────────────────────

  private heuristicScore(solution: string, challenge: Challenge): number {
    const lower = solution.toLowerCase();

    // Look for key fix patterns by challenge type
    const fixSignals: Record<string, string[]> = {
      'Memory Leak':    ['cleanup', 'clearinterval', 'clearimmediate', 'abort', 'return () =>', 'removeEventListener'],
      'Logic Error':    ['math.max', 'math.min', '??', '?.', 'filter', 'reduce', 'concat', 'spread'],
      'Race Condition': ['abortcontroller', 'signal', 'ismounted', 'promise.all', 'functional', 'prev =>'],
      'Buffer Overflow':['maxpages', 'maxbytes', 'maxdepth', 'limit', 'join(', 'iterative'],
      'API Timeout':    ['abortsignal', 'settimeout', 'promise.race', 'retry', 'backoff', 'promise.all'],
    };

    const signals = fixSignals[challenge.type] ?? [];
    const matches = signals.filter(s => lower.includes(s)).length;
    return Math.min(1.0, 0.3 + (matches / signals.length) * 0.7);
  }

  private async simulateExecution(code: string, _context: string): Promise<any> {
    return new Promise(resolve => {
      setTimeout(() => {
        // Non-random: code that is longer than 10 chars passes
        resolve(code.length > 10 ? 'ok' : null);
      }, Math.random() * 200);
    });
  }
}
