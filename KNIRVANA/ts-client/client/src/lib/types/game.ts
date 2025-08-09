export interface Vector3 {
  x: number;
  y: number;
  z: number;
}

export interface GameNode {
  id: string;
  position: Vector3;
  type: string;
}

export interface ErrorNodeData extends GameNode {
  difficulty: number;
  bounty: number;
  isBeingSolved: boolean;
  progress: number;
  solverAgent?: string;
  description?: string;
}

export interface SkillNodeData extends GameNode {
  name: string;
  creator: string;
  usageCount: number;
  category: string;
  value: number;
}

export interface AgentData {
  id: string;
  position: Vector3;
  target: string | null;
  status: 'idle' | 'moving' | 'working' | 'upgrading';
  type: 'Analyzer' | 'Optimizer' | 'Synthesizer' | 'Debugger';
  efficiency: number;
  experience: number;
  skills: string[];
  level: number;
}

export interface NetworkStats {
  totalNodes: number;
  activeAgents: number;
  solutionsFound: number;
  networkHealth: number;
  intelligenceLevel: number;
}

export interface GameConfig {
  cameraSpeed: number;
  maxAgents: number;
  nrnStartBalance: number;
  agentCreationCost: number;
  deploymentCost: number;
}
