import { create } from "zustand";
import { subscribeWithSelector } from "zustand/middleware";

export type GamePhase = "menu" | "playing" | "paused";

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
  position: { x: number; y: number; z: number };
  target: string | null;
  status: 'idle' | 'moving' | 'working' | 'upgrading';
  type: string;
  efficiency: number;
  experience: number;
}

interface KnirvanaState {
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

  // Selection
  selectedErrorNode: string | null;
  selectedIdeaNode: string | null;
  selectedPropertyNode: string | null;
  selectedAgent: string | null;
  
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
  const agents: Agent[] = [];
  
  for (let i = 0; i < 3; i++) {
    agents.push({
      id: `agent-${i}`,
      position: {
        x: (Math.random() - 0.5) * 10,
        y: 1,
        z: (Math.random() - 0.5) * 10
      },
      target: null,
      status: 'idle',
      type: types[Math.floor(Math.random() * types.length)],
      efficiency: 0.6 + Math.random() * 0.4,
      experience: Math.floor(Math.random() * 100)
    });
  }
  
  return agents;
};

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
    
    selectedErrorNode: null,
    selectedIdeaNode: null,
    selectedPropertyNode: null,
    selectedAgent: null,
    
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
    
    createAgent: (type) => {
      const cost = 50;
      if (get().spendNRN(cost)) {
        const newAgent: Agent = {
          id: `agent-${Date.now()}`,
          position: {
            x: (Math.random() - 0.5) * 5,
            y: 1,
            z: (Math.random() - 0.5) * 5
          },
          target: null,
          status: 'idle',
          type,
          efficiency: 0.5 + Math.random() * 0.3,
          experience: 0
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
    
    setState: (updater: any) => {
      set(updater);
    }
  }))
);