/**
 * KNIRVANA Bridge Service
 * Bridges KNIRVCONTROLLER personal graphs with KNIRVANA collective graph mechanics
 * Provides game-like interactions and agent deployment visualization
 */

import { personalKNIRVGRAPHService, GraphNode, PersonalGraph } from './PersonalKNIRVGRAPHService';
import { rxdbService } from './RxDBService';

// Import node data types for proper typing
interface ErrorNodeData {
  errorId: string;
  errorType: string;
  description: string;
  context: Record<string, unknown>;
  timestamp: number;
}

interface SkillNodeData {
  skillId: string;
  skillName: string;
  description: string;
  category: string;
  proficiency: number;
}

interface CollectiveInsight {
  id: string;
  type: 'pattern' | 'optimization' | 'collaboration' | 'trend';
  description: string;
  confidence: number;
  impact: number;
  timestamp: number;
}

// Re-export types from KNIRVANA concepts but adapted for KNIRVCONTROLLER
export interface KnirvanaErrorNode {
  id: string;
  position: { x: number; y: number; z: number };
  type: string;
  difficulty: number;
  bounty: number;
  isBeingSolved: boolean;
  progress: number;
  solverAgent?: string;
  description: string;
  context: Record<string, unknown>;
  timestamp: number;
}

export interface KnirvanaSkillNode {
  id: string;
  position: { x: number; y: number; z: number };
  name: string;
  creator: string;
  usageCount: number;
  proficiency: number;
  category: string;
  description: string;
}

export interface KnirvanaAgent {
  id: string;
  position: { x: number; y: number; z: number };
  target: string | null;
  status: 'idle' | 'moving' | 'working' | 'upgrading';
  type: string;
  efficiency: number;
  experience: number;
  capabilities: string[];
}

interface GameState {
  gamePhase: 'menu' | 'playing' | 'paused';
  gameTime: number;
  nrnBalance: number;
  skillsLearned: number;
  errorsResolved: number;
  errorNodes: KnirvanaErrorNode[];
  skillNodes: KnirvanaSkillNode[];
  agents: KnirvanaAgent[];
  selectedErrorNode: string | null;
  selectedAgent: string | null;
}

export class KnirvanaBridgeService {
  private gameState: GameState;
  private isInitialized = false;
  private personalGraph: PersonalGraph | null = null;
  private gameLoop: NodeJS.Timeout | null = null;

  constructor() {
    this.gameState = this.initializeGameState();
  }

  private initializeGameState(): GameState {
    return {
      gamePhase: 'menu',
      gameTime: 0,
      nrnBalance: 500,
      skillsLearned: 0,
      errorsResolved: 0,
      errorNodes: [],
      skillNodes: [],
      agents: [],
      selectedErrorNode: null,
      selectedAgent: null
    };
  }

  async initialize(): Promise<void> {
    if (this.isInitialized) return;

    try {
      // Initialize RxDB
      if (!rxdbService.isDatabaseInitialized()) {
        await rxdbService.initialize();
      }

      // Load or create personal graph
      this.personalGraph = await personalKNIRVGRAPHService.loadPersonalGraph('current_user');

      // Sync graph nodes to KNIRVANA game state
      await this.syncPersonalGraphToGame();

      // Initialize basic agents
      await this.initializeAgents();

      this.isInitialized = true;
      console.log('KNIRVANA Bridge service initialized');
    } catch (error) {
      console.error('Failed to initialize KNIRVANA Bridge:', error);
    }
  }

  private async syncPersonalGraphToGame(): Promise<void> {
    if (!this.personalGraph) return;

    // Convert personal graph nodes to KNIRVANA format
    this.gameState.errorNodes = this.personalGraph.nodes
      .filter(node => node.type === 'error')
      .map(node => this.convertToKnirvanaErrorNode(node));

    this.gameState.skillNodes = this.personalGraph.nodes
      .filter(node => node.type === 'skill')
      .map(node => this.convertToKnirvanaSkillNode(node));
  }

  private convertToKnirvanaErrorNode(node: GraphNode): KnirvanaErrorNode {
    // Type guard to ensure we have error node data
    if (node.type !== 'error') {
      throw new Error(`Expected error node, got ${node.type}`);
    }

    const errorData = node.data as ErrorNodeData;
    return {
      id: node.id,
      position: node.position,
      type: errorData.errorType || 'Unknown Error',
      difficulty: 0.5, // Default difficulty
      bounty: Math.floor(Math.random() * 50) + 10, // Random bounty
      isBeingSolved: false,
      progress: 0,
      description: node.label,
      context: errorData.context || {},
      timestamp: errorData.timestamp || Date.now()
    };
  }

  private convertToKnirvanaSkillNode(node: GraphNode): KnirvanaSkillNode {
    // Type guard to ensure we have skill node data
    if (node.type !== 'skill') {
      throw new Error(`Expected skill node, got ${node.type}`);
    }

    const skillData = node.data as SkillNodeData;
    return {
      id: node.id,
      position: node.position,
      name: skillData.skillName || node.label,
      creator: (skillData as any).creator || 'System',
      usageCount: (skillData as any).usageCount || 0,
      proficiency: skillData.proficiency || 0.5,
      category: skillData.category || 'general',
      description: skillData.description || node.label
    };
  }

  private async initializeAgents(): Promise<void> {
    // Create initial agents based on available skills
    const analyzerAgent: KnirvanaAgent = {
      id: `agent_analyzer_${Date.now()}`,
      position: { x: 0, y: 1, z: 0 },
      target: null,
      status: 'idle',
      type: 'Analyzer',
      efficiency: 0.7,
      experience: 0,
      capabilities: ['error_analysis', 'pattern_recognition']
    };

    const optimizerAgent: KnirvanaAgent = {
      id: `agent_optimizer_${Date.now()}`,
      position: { x: 5, y: 1, z: 5 },
      target: null,
      status: 'idle',
      type: 'Optimizer',
      efficiency: 0.6,
      experience: 0,
      capabilities: ['code_optimization', 'performance_tuning']
    };

    this.gameState.agents = [analyzerAgent, optimizerAgent];
  }

  // Game Control Methods
  startGame(): void {
    console.log('Starting KNIRVANA game session');
    this.gameState.gamePhase = 'playing';
    this.startGameLoop();
  }

  pauseGame(): void {
    console.log('Pausing KNIRVANA game session');
    this.gameState.gamePhase = 'paused';
    this.stopGameLoop();
  }

  private gameLoopCallback = (): void => {
    if (this.gameState.gamePhase === 'playing') {
      this.updateGameTime(0.016); // ~60 FPS
      this.gameLoop = setTimeout(this.gameLoopCallback, 16) as unknown as NodeJS.Timeout;
    }
  };

  private startGameLoop(): void {
    if (this.gameLoop) clearTimeout(this.gameLoop);
    this.gameLoopCallback();
  }

  private stopGameLoop(): void {
    if (this.gameLoop) {
      clearTimeout(this.gameLoop);
      this.gameLoop = null;
    }
  }

  private updateGameTime(delta: number): void {
    if (this.gameState.gamePhase !== 'playing') return;

    this.gameState.gameTime += delta;

    // Update error solving progress
    this.gameState.errorNodes = this.gameState.errorNodes.map(error => {
      if (error.isBeingSolved && error.progress < 1) {
        const agent = this.gameState.agents.find(a => a.id === error.solverAgent);
        const progressRate = agent ? agent.efficiency * 0.1 : 0.05;
        const newProgress = Math.min(1, error.progress + progressRate * delta);

        // Check if completed
        if (newProgress >= 1 && error.progress < 1) {
          console.log(`Error ${error.id} resolved! Awarding ${error.bounty} NRN`);
          this.awardNRN(error.bounty);
          this.gameState.errorsResolved++;
        }

        return { ...error, progress: newProgress };
      }
      return error;
    });
  }

  // Node Selection Methods
  selectErrorNode(id: string): void {
    console.log(`Selected error node: ${id}`);
    this.gameState.selectedErrorNode = id;
  }

  selectAgent(id: string): void {
    console.log(`Selected agent: ${id}`);
    this.gameState.selectedAgent = id;
  }

  // Agent Deployment Methods
  async deployAgent(agentId: string, nodeId: string): Promise<boolean> {
    const deploymentCost = 10;
    if (!this.spendNRN(deploymentCost)) {
      console.log('Insufficient NRN for agent deployment');
      return false;
    }

    // Update agent status
    const agent = this.gameState.agents.find(a => a.id === agentId);
    if (!agent) return false;

    agent.target = nodeId;
    agent.status = 'working';

    // Update error node
    const errorNode = this.gameState.errorNodes.find(n => n.id === nodeId);
    if (errorNode) {
      errorNode.isBeingSolved = true;
      errorNode.solverAgent = agentId;

      // Sync back to personal graph
      await this.syncErrorNodeToPersonalGraph(errorNode);
    }

    console.log(`Deployed agent ${agentId} to solve error ${nodeId}`);
    return true;
  }

  async createAgent(type: string): Promise<boolean> {
    const agentCost = 50;
    if (!this.spendNRN(agentCost)) {
      console.log('Insufficient NRN to create agent');
      return false;
    }

    const newAgent: KnirvanaAgent = {
      id: `agent_${type.toLowerCase()}_${Date.now()}`,
      position: {
        x: (Math.random() - 0.5) * 10,
        y: 1,
        z: (Math.random() - 0.5) * 10
      },
      target: null,
      status: 'idle',
      type,
      efficiency: 0.5 + Math.random() * 0.3,
      experience: 0,
      capabilities: this.getCapabilitiesForAgentType(type)
    };

    this.gameState.agents.push(newAgent);

    // Sync to personal graph
    await this.syncAgentToPersonalGraph(newAgent);

    console.log(`Created new ${type} agent`);
    return true;
  }

  private getCapabilitiesForAgentType(type: string): string[] {
    switch (type.toLowerCase()) {
      case 'analyzer':
        return ['error_analysis', 'pattern_recognition', 'diagnostic_tools'];
      case 'optimizer':
        return ['code_optimization', 'performance_tuning', 'refactoring'];
      case 'debugger':
        return ['breakpoints', 'step_through', 'variable_inspection'];
      default:
        return ['general_problem_solving'];
    }
  }

  // NRN Management
  private awardNRN(amount: number): void {
    this.gameState.nrnBalance += amount;
    console.log(`Awarded ${amount} NRN. New balance: ${this.gameState.nrnBalance}`);
  }

  private spendNRN(amount: number): boolean {
    if (this.gameState.nrnBalance >= amount) {
      this.gameState.nrnBalance -= amount;
      console.log(`Spent ${amount} NRN. New balance: ${this.gameState.nrnBalance}`);
      return true;
    }
    return false;
  }

  // Sync Methods to Personal Graph
  private async syncErrorNodeToPersonalGraph(knirvanaNode: KnirvanaErrorNode): Promise<void> {
    if (!this.personalGraph) return;

    const existingNode = this.personalGraph.nodes.find(n => n.id === knirvanaNode.id);
    if (existingNode && 'errorType' in existingNode.data) {
      // Update the error node data with the current game state
      const updatedData = {
        ...existingNode.data,
        progress: knirvanaNode.progress,
        isBeingSolved: knirvanaNode.isBeingSolved,
        solverAgent: knirvanaNode.solverAgent
      };
      existingNode.data = updatedData;
    }
  }

  private async syncAgentToPersonalGraph(agent: KnirvanaAgent): Promise<void> {
    await personalKNIRVGRAPHService.addSkillNode({
      skillId: agent.id,
      skillName: `${agent.type} Agent`,
      description: `AI Agent with capabilities: ${agent.capabilities.join(', ')}`,
      category: 'agent',
      proficiency: agent.efficiency
    });
  }

  // Collective Graph Integration
  async mergeToCollectiveNetwork(_collectiveGraph: Record<string, unknown>): Promise<void> {
    console.log('Merging personal graph to collective KNIRVANA network');

    // This would connect to KNIRVANA collective graph API
    // For now, simulate the merge
    if (this.personalGraph) {
      // Add collective insights as new nodes
      const collectiveInsights = await this.generateCollectiveInsights();

      for (const insight of collectiveInsights) {
        await personalKNIRVGRAPHService.addSkillNode({
          skillId: `collective_${insight.id}`,
          skillName: (insight as any).name || `Insight ${insight.id}`,
          description: insight.description,
          category: 'collective',
          proficiency: 0.8
        });
      }

      await this.syncPersonalGraphToGame();
    }
  }

  private async generateCollectiveInsights(): Promise<CollectiveInsight[]> {
    // Simulate collective insights from KNIRVANA network
    return [
      {
        id: 'advanced_error_patterns',
        type: 'pattern',
        description: 'Recognize complex error patterns learned from collective experiences',
        confidence: 0.85,
        impact: 0.9,
        timestamp: Date.now()
      },
      {
        id: 'predictive_debugging',
        type: 'optimization',
        description: 'Anticipate errors before they occur based on collective data',
        confidence: 0.78,
        impact: 0.85,
        timestamp: Date.now()
      }
    ];
  }

  // Getters
  getGameState(): GameState {
    return { ...this.gameState };
  }

  getCurrentGraph(): PersonalGraph | null {
    return this.personalGraph;
  }

  // Cleanup
  cleanup(): void {
    this.stopGameLoop();
    this.gameState = this.initializeGameState();
  }
}

// Export singleton instance
export const knirvanaBridgeService = new KnirvanaBridgeService();
