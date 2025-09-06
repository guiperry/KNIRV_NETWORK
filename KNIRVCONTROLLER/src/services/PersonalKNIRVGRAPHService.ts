/**
 * Personal KNIRVGRAPH Service
 * Manages individual user's graph instance with error mapping and visualization
 */

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

interface ConnectionNodeData {
  connectionType: string;
  strength: number;
}

interface AgentNodeData {
  agentId: string;
  agentType: string;
  capabilities: string[];
}

type NodeData = ErrorNodeData | SkillNodeData | ConnectionNodeData | AgentNodeData;

interface EdgeData {
  connectionType: string;
  weight: number;
  metadata?: Record<string, unknown>;
}

import { rxdbService } from './RxDBService';

export interface GraphNode {
  id: string;
  type: 'error' | 'skill' | 'connection' | 'agent';
  label: string;
  position: { x: number; y: number; z: number };
  data: NodeData;
  connections: string[]; // IDs of connected nodes
}

export interface GraphEdge {
  id: string;
  source: string;
  target: string;
  type: 'error_to_skill' | 'skill_chain' | 'agent_connection';
  weight: number;
  data: EdgeData;
}

export interface PersonalGraph {
  id: string;
  userId: string;
  nodes: GraphNode[];
  edges: GraphEdge[];
  metadata: {
    createdAt: number;
    lastModified: number;
    version: number;
    complexity: number;
  };
}

export class PersonalKNIRVGRAPHService {
  private currentGraph: PersonalGraph | null = null;
  private isInitialized = false;

  constructor() {
    this.initialize();
  }

  private async initialize(): Promise<void> {
    if (this.isInitialized) return;

    try {
      // Initialize RxDB if not already done
      if (!rxdbService.isDatabaseInitialized()) {
        await rxdbService.initialize();
      }

      this.isInitialized = true;
      console.log('Personal KNIRVGRAPH service initialized');
    } catch (error) {
      console.error('Failed to initialize Personal KNIRVGRAPH service:', error);
    }
  }

  // Create a new personal graph
  async createPersonalGraph(userId: string): Promise<PersonalGraph> {
    const graph: PersonalGraph = {
      id: `graph_${userId}_${Date.now()}`,
      userId,
      nodes: [],
      edges: [],
      metadata: {
        createdAt: Date.now(),
        lastModified: Date.now(),
        version: 1,
        complexity: 0
      }
    };

    this.currentGraph = graph;

    // Store in RxDB
    await this.saveGraphToDatabase(graph);

    return graph;
  }

  // Load user's personal graph
  async loadPersonalGraph(userId: string): Promise<PersonalGraph | null> {
    try {
      if (!this.isInitialized) await this.initialize();

      // For now, create a new graph if none exists
      // In a real implementation, this would load from the database
      const graph = await this.createPersonalGraph(userId);
      return graph;
    } catch (error) {
      console.error('Failed to load personal graph:', error);
      return null;
    }
  }

  // Add error node to graph
  async addErrorNode(errorData: {
    errorId: string;
    errorType: string;
    description: string;
    context: Record<string, unknown>;
    timestamp: number;
  }): Promise<GraphNode> {
    if (!this.currentGraph) throw new Error('No active graph');

    const node: GraphNode = {
      id: `error_${errorData.errorId}`,
      type: 'error',
      label: errorData.description,
      position: this.calculateNodePosition(),
      data: errorData,
      connections: []
    };

    this.currentGraph.nodes.push(node);

    // Attempt to find related skills automatically
    await this.findRelatedSkills(node);

    // Update graph
    await this.updateGraph();

    return node;
  }

  // Add skill node to graph
  async addSkillNode(skillData: {
    skillId: string;
    skillName: string;
    description: string;
    category: string;
    proficiency: number;
  }): Promise<GraphNode> {
    if (!this.currentGraph) throw new Error('No active graph');

    const node: GraphNode = {
      id: `skill_${skillData.skillId}`,
      type: 'skill',
      label: skillData.skillName,
      position: this.calculateNodePosition(),
      data: skillData,
      connections: []
    };

    this.currentGraph.nodes.push(node);
    await this.updateGraph();

    return node;
  }

  // Create connection between nodes
  async createConnection(sourceId: string, targetId: string, type: GraphEdge['type'], weight = 1): Promise<GraphEdge> {
    if (!this.currentGraph) throw new Error('No active graph');

    const edge: GraphEdge = {
      id: `edge_${sourceId}_${targetId}`,
      source: sourceId,
      target: targetId,
      type,
      weight,
      data: {
        connectionType: type,
        weight: weight
      }
    };

    this.currentGraph.edges.push(edge);

    // Update node connections
    const sourceNode = this.currentGraph.nodes.find(n => n.id === sourceId);
    const targetNode = this.currentGraph.nodes.find(n => n.id === targetId);

    if (sourceNode && !sourceNode.connections.includes(targetId)) {
      sourceNode.connections.push(targetId);
    }
    if (targetNode && !targetNode.connections.includes(sourceId)) {
      targetNode.connections.push(sourceId);
    }

    await this.updateGraph();

    return edge;
  }

  // Find skills related to an error
  private async findRelatedSkills(errorNode: GraphNode): Promise<void> {
    // Simple pattern matching for demo
    const errorText = (errorNode.data as ErrorNodeData).description.toLowerCase();
    const skillMappings = [
      { pattern: /type.*error/i, skill: 'TypeScript' },
      { pattern: /import.*error/i, skill: 'Module Management' },
      { pattern: /network.*error/i, skill: 'Network Programming' },
      { pattern: /async.*error/i, skill: 'Asynchronous Programming' }
    ];

    for (const mapping of skillMappings) {
      if (mapping.pattern.test(errorText)) {
        // Create skill node if it doesn't exist
        const existingSkill = this.currentGraph?.nodes.find(
          n => n.type === 'skill' && 'skillName' in n.data && n.data.skillName === mapping.skill
        );

        if (!existingSkill) {
          await this.addSkillNode({
            skillId: `skill_${mapping.skill.toLowerCase().replace(/\s+/g, '_')}`,
            skillName: mapping.skill,
            description: `Skill related to ${mapping.skill}`,
            category: 'programming',
            proficiency: 0.5
          });
        }

        // Create connection
        const skillNode = this.currentGraph?.nodes.find(
          n => n.type === 'skill' && 'skillName' in n.data && (n.data as SkillNodeData).skillName === mapping.skill
        );

        if (skillNode) {
          await this.createConnection(errorNode.id, skillNode.id, 'error_to_skill');
        }
      }
    }
  }

  // Calculate node position (simple algorithm)
  private calculateNodePosition(): { x: number; y: number; z: number } {
    if (!this.currentGraph) return { x: 0, y: 0, z: 0 };

    const nodeCount = this.currentGraph.nodes.length;
    const angle = (nodeCount * 137.5) * (Math.PI / 180); // Golden angle
    const radius = Math.sqrt(nodeCount) * 50;

    return {
      x: Math.cos(angle) * radius,
      y: Math.sin(angle) * radius,
      z: nodeCount * 10
    };
  }

  // Update graph and save to database
  private async updateGraph(): Promise<void> {
    if (!this.currentGraph) return;

    this.currentGraph.metadata.lastModified = Date.now();
    this.currentGraph.metadata.complexity = this.currentGraph.nodes.length + this.currentGraph.edges.length;

    await this.saveGraphToDatabase(this.currentGraph);
  }

  // Save graph to RxDB
  private async saveGraphToDatabase(graph: PersonalGraph): Promise<void> {
    try {
      const db = rxdbService.getDatabase();

      // Store as settings or create a graph document
      await db.settings.insert({
        id: `graph_${graph.id}`,
        type: 'settings',
        walletId: 'personal_graph',
        autoSync: false,
        biometricEnabled: false,
        notificationsEnabled: false,
        defaultNetwork: 'testnet',
        preferredCurrency: 'NRN',
        theme: 'dark',
        createdAt: graph.metadata.createdAt,
        updatedAt: graph.metadata.lastModified
      });

      console.log('Graph saved to database');
    } catch (error) {
      console.error('Failed to save graph to database:', error);
    }
  }

  // Get current graph
  getCurrentGraph(): PersonalGraph | null {
    return this.currentGraph;
  }

  // Export graph data for visualization
  exportGraphData(): { nodes: GraphNode[]; edges: GraphEdge[] } | null {
    if (!this.currentGraph) return null;

    return {
      nodes: this.currentGraph.nodes,
      edges: this.currentGraph.edges
    };
  }

  // Reset graph
  async resetGraph(): Promise<void> {
    if (this.currentGraph) {
      this.currentGraph.nodes = [];
      this.currentGraph.edges = [];
      await this.updateGraph();
    }
  }

  // Get graph statistics
  getGraphStats(): {
    nodeCount: number;
    edgeCount: number;
    complexity: number;
    nodeTypes: Record<string, number>;
  } | null {
    if (!this.currentGraph) return null;

    const nodeTypes: Record<string, number> = {};
    this.currentGraph.nodes.forEach(node => {
      nodeTypes[node.type] = (nodeTypes[node.type] || 0) + 1;
    });

    return {
      nodeCount: this.currentGraph.nodes.length,
      edgeCount: this.currentGraph.edges.length,
      complexity: this.currentGraph.metadata.complexity,
      nodeTypes
    };
  }
}

// Export singleton instance
export const personalKNIRVGRAPHService = new PersonalKNIRVGRAPHService();
