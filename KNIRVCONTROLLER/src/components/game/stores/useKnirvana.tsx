import { create } from "zustand";
import { subscribeWithSelector } from "zustand/middleware";
import { Tournament } from "../../../engine/Tournament";
import { Verifier } from "../../../engine/Verifier";
import { LoraxClient } from "../../../networking/LoraxClient";
import { TrainingManager } from "../../../engine/TrainingManager";
import { SabotageEngine, SabotageType } from "../../../engine/Sabotage";

export interface DeployAnimation {
  id: string;
  agentId: string;
  agentName: string;
  startPosition: { x: number; y: number; z: number };
  endPosition: { x: number; y: number; z: number };
  startTime: number;
  duration: number;
}

export type GamePhase = "menu" | "playing" | "paused" | "training";

export interface AgentResources {
  compute: number;      // "Mana" for sabotage/human actions
  parity: number;       // "Health" - reaching 0 causes divergence (elimination)
  generation: number;   // Track evolutionary steps
}

export interface ErrorNode {
  id: string;
  position: { x: number; y: number; z: number };
  type: string;
  difficulty: number;
  bounty: number;
  isBeingSolved: boolean;
  progress: number;
  solverAgent?: string;
}

export interface SkillNode {
  id: string;
  position: { x: number; y: number; z: number };
  name: string;
  creator: string;
  usageCount: number;
}

export interface IdeaNode {
  id: string;
  position: { x: number; y: number; z: number };
  name: string;
  ideaType: 'asset' | 'characteristic' | 'attribute' | 'innovation' | 'improvement' | 'feature';
  description: string;
  feasibilityScore: number;
  existenceCheck: boolean;
  collaborators: string[];
  stakes: Record<string, number>;
  collaborationValue: number;
  status: 'pending' | 'collaborative' | 'property_created' | 'abandoned';
}

export interface PropertyNode {
  id: string;
  position: { x: number; y: number; z: number };
  name: string;
  propertyType: 'asset' | 'characteristic' | 'attribute' | 'license' | 'patent' | 'trademark';
  sourceIdea: string;
  valueType: string;
  immutable: boolean;
  category: string;
  owners: Record<string, number>;
  marketValue: number;
  usageCount: number;
}

export interface Agent {
  id: string;
  name: string;
  position: { x: number; y: number; z: number };
  target: string | null;
  status: 'idle' | 'moving' | 'working' | 'upgrading';
  type: string;
  efficiency: number;
  accuracy?: number;
  experience: number;
  resources: AgentResources;
  policy: 'greedy' | 'bayesian' | 'stochastic';
  stats?: {
    errorsResolved: number;
    skillsLearned: number;
    ideasDeveloped: number;
    propertiesCreated: number;
    sabotageApplied: number;
    trainingTime: number;
  };
  proposeSolution: (errorNodeContext: string) => Promise<{
    chainOfThought: string[];
    code: string;
    estimatedLatency: number;
  }>;
}

export interface KnirvanaState {
  // Game state
  gamePhase: GamePhase;
  gameTime: number;
  
  // Resources
  nrnBalance: number;
  skillsLearned: number;
  errorsResolved: number;
  ideasDeveloped: number;
  propertiesCreated: number;

  // Game objects
  errorNodes: ErrorNode[];
  skillNodes: SkillNode[];
  ideaNodes: IdeaNode[];
  propertyNodes: PropertyNode[];
  agents: Agent[];
  
  // Deploy animations
  deployAnimations: DeployAnimation[];

  // Selection
  selectedErrorNode: string | null;
  selectedIdeaNode: string | null;
  selectedPropertyNode: string | null;
  selectedAgent: string | null;

  // Tournament specific
  skillSlotOwner: string | null;
  incumbentScore: number;
  
  // Engine instances
  tournament: Tournament;
  verifier: Verifier;
  loraxClient: LoraxClient;
  trainingManager: TrainingManager;
  
  // Actions
  startGame: () => void;
  pauseGame: () => void;
  updateGameTime: (delta: number) => void;
  
  // Node actions
  selectErrorNode: (id: string) => void;
  selectAgent: (id: string) => void;
  
  // Agent actions
  createAgent: (type: string) => void;
  deployAgent: (agentId: string, nodeId: string) => void;
  moveAgent: (agentId: string, position: { x: number; y: number; z: number }) => void;
  
  // Resource actions
  addNRN: (amount: number) => void;
  spendNRN: (amount: number) => boolean;
  
  // Tournament actions
  runEpoch: () => Promise<void>;
  
  // Sabotage actions
  applySabotage: (targetAgentId: string, type: SabotageType, magnitude: number) => void;
  
  // Training actions
  startTraining: (agentId: string) => void;
  distillTrajectory: (agentId: string) => void;
  hardenAgent: (agentId: string) => void;
  
  // Deploy animation actions
  startDeployAnimation: (agentId: string, agentName: string, endPosition: { x: number; y: number; z: number }) => void;
  updateDeployAnimations: () => void;
  
  // Verifier actions
  updateVerifierWeights: (weights: { correctness: number; latency: number; simplicity: number }) => void;
  addVerifierConstraint: (id: string, validator: (res: unknown) => boolean) => void;
}

// Generate initial game data
const generateErrorNodes = (): ErrorNode[] => {
  const types = ['Memory Leak', 'Logic Error', 'Race Condition', 'Buffer Overflow', 'API Timeout'];
  const nodes: ErrorNode[] = [];
  
  for (let i = 0; i < 15; i++) {
    nodes.push({
      id: `error-${i}`,
      position: {
        x: (Math.random() - 0.5) * 40,
        y: 0,
        z: (Math.random() - 0.5) * 40
      },
      type: types[Math.floor(Math.random() * types.length)],
      difficulty: Math.random(),
      bounty: Math.round(10 + Math.random() * 90),
      isBeingSolved: false,
      progress: 0
    });
  }
  
  return nodes;
};

const generateSkillNodes = (): SkillNode[] => {
  const skills = ['Memory Optimization', 'Logic Verification', 'Concurrency Control', 'Security Hardening', 'Performance Tuning'];
  const nodes: SkillNode[] = [];
  
  for (let i = 0; i < 8; i++) {
    nodes.push({
      id: `skill-${i}`,
      position: {
        x: (Math.random() - 0.5) * 35,
        y: 0,
        z: (Math.random() - 0.5) * 35
      },
      name: skills[Math.floor(Math.random() * skills.length)],
      creator: `Agent-${Math.floor(Math.random() * 100)}`,
      usageCount: Math.floor(Math.random() * 50)
    });
  }
  
  return nodes;
};

const generateInitialAgents = (): Agent[] => {
  const types = ['Analyzer', 'Optimizer'];
  const policies: Array<'greedy' | 'bayesian' | 'stochastic'> = ['greedy', 'bayesian', 'stochastic'];
  const agents: Agent[] = [];
  
  for (let i = 0; i < 3; i++) {
    agents.push({
      id: `agent-${i}`,
      name: `Agent ${i}`,
      position: {
        x: (Math.random() - 0.5) * 10,
        y: 1,
        z: (Math.random() - 0.5) * 10
      },
      target: null,
      status: 'idle',
      type: types[Math.floor(Math.random() * types.length)],
      efficiency: 0.6 + Math.random() * 0.4,
      experience: Math.floor(Math.random() * 100),
      resources: {
        compute: 100,
        parity: 100,
        generation: 1
      },
      policy: policies[i % policies.length],
      proposeSolution: async (errorNodeContext: string) => {
        // Default implementation
        return {
          chainOfThought: [`Thinking about ${errorNodeContext}`, 'Found a solution'],
          code: `function solve(${errorNodeContext}) { return 42; }`,
          estimatedLatency: Math.random() * 1000
        };
      }
    });
  }
  
  return agents;
};

// Initialize engine instances
const verifier = new Verifier();
const loraxClient = new LoraxClient('http://localhost:8080');
const tournament = new Tournament(verifier, loraxClient);
const trainingManager = new TrainingManager();

export const useKnirvana = create<KnirvanaState>()(
  subscribeWithSelector((set, get) => ({
    // Initial state
    gamePhase: "menu",
    gameTime: 0,
    
    nrnBalance: 500,
    skillsLearned: 0,
    errorsResolved: 0,
    ideasDeveloped: 0,
    propertiesCreated: 0,
    
    errorNodes: generateErrorNodes(),
    skillNodes: generateSkillNodes(),
    ideaNodes: [],
    propertyNodes: [],
    agents: generateInitialAgents(),
    deployAnimations: [],
    
    selectedErrorNode: null,
    selectedIdeaNode: null,
    selectedPropertyNode: null,
    selectedAgent: null,
    
    skillSlotOwner: null,
    incumbentScore: 0.8,
    
    tournament,
    verifier,
    loraxClient,
    trainingManager,
    
    // Actions
    startGame: () => {
      console.log("Starting KNIRVANA game");
      set({ gamePhase: "playing" });
    },
    
    pauseGame: () => {
      set({ gamePhase: "paused" });
    },
    
    updateGameTime: (delta) => {
      set((state) => ({ gameTime: state.gameTime + delta }));
      
      // Update deploy animations
      get().updateDeployAnimations();
      
      // Update game simulation
      const state = get();
      const updatedErrorNodes = state.errorNodes.map(node => {
        if (node.isBeingSolved && node.progress < 1) {
          const agent = state.agents.find(a => a.id === node.solverAgent);
          const progressRate = agent ? agent.efficiency * 0.1 : 0.05;
          const newProgress = Math.min(1, node.progress + progressRate * delta);
          
          // Check if node is completed
          if (newProgress >= 1 && node.progress < 1) {
            console.log(`ErrorNode ${node.id} resolved!`);
            // Award NRN to player
            get().addNRN(node.bounty);
            set((s) => ({ errorsResolved: s.errorsResolved + 1 }));
          }
          
          return { ...node, progress: newProgress };
        }
        return node;
      });
      
      set({ errorNodes: updatedErrorNodes });
    },
    
    selectErrorNode: (id) => {
      console.log(`Selected ErrorNode: ${id}`);
      set({ selectedErrorNode: id });
    },
    
    selectAgent: (id) => {
      console.log(`Selected Agent: ${id}`);
      set({ selectedAgent: id });
    },
    
    startDeployAnimation: (agentId, agentName, endPosition) => {
      console.log(`Starting deploy animation for agent ${agentId} to position (${endPosition.x}, ${endPosition.y}, ${endPosition.z})`);
      
      const animation: DeployAnimation = {
        id: `deploy-${Date.now()}`,
        agentId,
        agentName,
        startPosition: { x: 0, y: 20, z: 0 }, // Camera viewport position
        endPosition,
        startTime: Date.now(),
        duration: 1500 // 1.5 seconds
      };
      
      set(state => ({
        deployAnimations: [...state.deployAnimations, animation]
      }));
    },
    
    updateDeployAnimations: () => {
      const now = Date.now();
      set(state => {
        const activeAnimations = state.deployAnimations.filter(anim => 
          now - anim.startTime < anim.duration
        );
        
        // For completed animations, add the agent to the game
        const completedAnimations = state.deployAnimations.filter(anim => 
          now - anim.startTime >= anim.duration
        );
        
        if (completedAnimations.length > 0) {
          const newAgents = [...state.agents];
          completedAnimations.forEach(anim => {
            const newAgent: Agent = {
              id: anim.agentId,
              name: anim.agentName,
              position: anim.endPosition,
              target: null,
              status: 'idle',
              type: 'deployed',
              policy: 'greedy',
              efficiency: Math.random() * 0.5 + 0.5,
              accuracy: Math.random() * 0.3 + 0.7,
              experience: 0,
              resources: {
                compute: 100,
                parity: 100,
                generation: 1
              },
              stats: {
                errorsResolved: 0,
                skillsLearned: 0,
                ideasDeveloped: 0,
                propertiesCreated: 0,
                sabotageApplied: 0,
                trainingTime: 0
              },
              proposeSolution: async (errorNodeContext: string) => {
                return {
                  chainOfThought: [`Thinking about ${errorNodeContext}`, 'Found a solution'],
                  code: `function solve(${errorNodeContext}) { return 42; }`,
                  estimatedLatency: Math.random() * 1000
                };
              }
            };
            newAgents.push(newAgent);
          });
          
          return {
            deployAnimations: activeAnimations,
            agents: newAgents
          };
        }
        
        return { deployAnimations: activeAnimations };
      });
    },
    
    createAgent: (type) => {
      const cost = 50;
      if (get().spendNRN(cost)) {
        const policies: Array<'greedy' | 'bayesian' | 'stochastic'> = ['greedy', 'bayesian', 'stochastic'];
        const newAgent: Agent = {
          id: `agent-${Date.now()}`,
          name: `${type} Agent ${Date.now()}`,
          position: {
            x: (Math.random() - 0.5) * 5,
            y: 1,
            z: (Math.random() - 0.5) * 5
          },
          target: null,
          status: 'idle',
          type,
          efficiency: 0.5 + Math.random() * 0.3,
          experience: 0,
          resources: {
            compute: 100,
            parity: 100,
            generation: 1
          },
          policy: policies[Math.floor(Math.random() * policies.length)],
          proposeSolution: async (errorNodeContext: string) => {
            return {
              chainOfThought: [`Thinking about ${errorNodeContext}`, 'Found a solution'],
              code: `function solve(${errorNodeContext}) { return 42; }`,
              estimatedLatency: Math.random() * 1000
            };
          }
        };
        
        console.log(`Created new ${type} agent`);
        set((state) => ({ agents: [...state.agents, newAgent] }));
      }
    },
    
    deployAgent: (agentId, nodeId) => {
      const cost = 10;
      if (get().spendNRN(cost)) {
        console.log(`Deploying agent ${agentId} to node ${nodeId}`);
        
        set((state) => ({
          agents: state.agents.map(agent =>
            agent.id === agentId
              ? { ...agent, target: nodeId, status: 'working' as const }
              : agent
          ),
          errorNodes: state.errorNodes.map(node =>
            node.id === nodeId
              ? { ...node, isBeingSolved: true, solverAgent: agentId }
              : node
          )
        }));
      }
    },
    
    moveAgent: (agentId, position) => {
      set((state) => ({
        agents: state.agents.map(agent =>
          agent.id === agentId
            ? { ...agent, position, status: 'moving' as const }
            : agent
        )
      }));
    },
    
    addNRN: (amount) => {
      set((state) => ({ nrnBalance: state.nrnBalance + amount }));
    },
    
    spendNRN: (amount) => {
      const state = get();
      if (state.nrnBalance >= amount) {
        set({ nrnBalance: state.nrnBalance - amount });
        return true;
      }
      return false;
    },
    
    runEpoch: async () => {
      const state = get();
      if (state.gamePhase !== 'playing') return;
      
      console.log('Running tournament epoch');
      try {
        // Get current error node context (using first active error node)
        const activeErrorNode = state.errorNodes.find(node => node.isBeingSolved);
        if (!activeErrorNode) return;
        
        // Simulate agent responses
        const simulatedAgents = state.agents.map(agent => ({
          ...agent,
          proposeSolution: async (context: string) => {
            // Simulate agent thinking
            await new Promise(resolve => setTimeout(resolve, Math.random() * 1000));
            return {
              chainOfThought: [`Thinking about ${context}`, 'Found a solution'],
              code: `function solve(${context}) { return 42; }`,
              estimatedLatency: Math.random() * 1000
            };
          }
        }));
        
        await state.tournament.runEpoch(simulatedAgents, activeErrorNode.type);
        
        // Update skill slot owner
        const skillSlotOwner = state.tournament.getSkillSlotOwner();
        const incumbentScore = state.tournament.getIncumbentScore();
        set({ skillSlotOwner, incumbentScore });
        
      } catch (error) {
        console.error('Error running tournament epoch:', error);
      }
    },
    
    applySabotage: (targetAgentId, type, magnitude) => {
      const state = get();
      const targetAgent = state.agents.find(a => a.id === targetAgentId);
      
      if (targetAgent) {
        // Check if player has enough compute
        const cost = magnitude * 10;
        if (targetAgent.resources.compute >= cost) {
          SabotageEngine.applyEffect(type, targetAgent, magnitude);
          targetAgent.resources.compute -= cost;
          
          set((s) => ({
            agents: s.agents.map(a =>
              a.id === targetAgentId ? { ...targetAgent } : a
            )
          }));
        }
      }
    },
    
    startTraining: (agentId) => {
      set({ gamePhase: 'training' });
      console.log(`Starting training for agent ${agentId}`);
    },
    
    distillTrajectory: (agentId) => {
      console.log(`Distilling trajectory for agent ${agentId}`);
      // In a real implementation, this would process the agent's trajectory
    },
    
    hardenAgent: (agentId) => {
      const state = get();
      const agent = state.agents.find(a => a.id === agentId);
      
      if (agent) {
        state.trainingManager.harden(agent);
        set((s) => ({
          agents: s.agents.map(a =>
            a.id === agentId ? { ...agent } : a
          )
        }));
      }
    },
    
    updateVerifierWeights: (weights) => {
      const state = get();
      state.verifier.updateWeights(weights);
      console.log('Updated verifier weights:', weights);
    },
    
    addVerifierConstraint: (id, validator) => {
      const state = get();
      state.verifier.addConstraint(id, validator);
      console.log(`Added verifier constraint: ${id}`);
    },
    
    setState: (updater: Partial<KnirvanaState> | ((state: KnirvanaState) => Partial<KnirvanaState>)) => {
      set(updater);
    }
  }))
);