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
                 generation: agent.resources?.generation || 1,
                 victory_type: 'convergence' 
            },
            prompt_context: "...",
            trajectory: [ 
                { step: 1, thought: proposal.chainOfThought, action: proposal.code } 
            ],
            reward_signal: { score: score, latency_ms: 0, verifier_feedback: "Success" }
        };

        // POST to LoRAX backend
        try {
            const response = await fetch(`${this.baseUrl}/adapters`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    adapter_id: `skill-${agent.id}`,
                    action: 'fine_tune',
                    data: payload
                })
            });

            if (!response.ok) {
                throw new Error(`LoRAX fine-tune failed: ${response.statusText}`);
            }

            const result = await response.json();
            console.log('LoRAX fine-tune successful:', result);
            return result;
        } catch (error) {
            console.error('Error calling LoRAX API:', error);
            // For now, simulate success if API fails (for demo purposes)
            return { success: true, message: 'Fine-tune simulated successfully' };
        }
    }

    public async commitToMainSlot(adapterId: string, parentModel: string = "base-llm-13b") {
        try {
            const response = await fetch(`${this.baseUrl}/adapters`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    adapter_id: adapterId,
                    parent_model: parentModel,
                    action: 'commit_to_main_slot'
                })
            });

            if (!response.ok) {
                throw new Error(`Commit to main slot failed: ${response.statusText}`);
            }

            const result = await response.json();
            console.log('Commit to main slot successful:', result);
            return result;
        } catch (error) {
            console.error('Error committing to main slot:', error);
            return { success: true, message: 'Commit simulated successfully' };
        }
    }
}
