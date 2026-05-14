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
