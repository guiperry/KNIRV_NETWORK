import { Verifier } from './Verifier';
import { LoraxClient } from '../networking/LoraxClient';
import { RFTAgent } from '../types/Agent';

export class Tournament {
    private verifier: Verifier;
    private lorax: LoraxClient;
    private skillSlotOwner: string | null = null; // Agent ID
    private incumbentScore: number = 0.8; // Baseline to beat

    constructor(verifier: Verifier, lorax: LoraxClient) {
        this.verifier = verifier;
        this.lorax = lorax;
    }

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

    public getAgent(agentId: string): RFTAgent | null {
        // This would typically fetch from the agent collection
        // For now, return null - will be implemented with actual agent management
        return null;
    }

    public getSkillSlotOwner(): string | null {
        return this.skillSlotOwner;
    }

    public getIncumbentScore(): number {
        return this.incumbentScore;
    }
}
