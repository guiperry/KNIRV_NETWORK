/**
 * Personal KNIRVGRAPH Service
 * Manages individual user's graph instance with error mapping and visualization
 */

import { FactualitySlice } from '../slices/factualitySlice';
import { FeasibilityReport } from '../slices/feasibilitySlice';

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

interface CapabilityNodeData {
  capabilityId: string;
  capabilityName: string;
  description: string;
  mcpServerInfo: Record<string, unknown>;
  category: string;
  timestamp: number;
}

// Allow node data to optionally include slice objects
type NodeWithSlices = NodeData & {
  factualitySlice?: FactualitySlice;
  capabilitySlice?: FactualitySlice;
  feasibilitySlice?: FeasibilityReport;
};

interface PropertyNodeData {
  propertyId: string;
  propertyName: string;
  description: string;
  feasibilityReport: {
    exists: boolean;
    similarProjects: string[];
    feasibilityScore: number;
    marketAnalysis: Record<string, unknown>;
  };
  collaborators: string[];
  timestamp: number;
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

type NodeData = ErrorNodeData | SkillNodeData | CapabilityNodeData | PropertyNodeData | ConnectionNodeData | AgentNodeData;

interface EdgeData {
  connectionType: string;
  weight: number;
  metadata?: Record<string, unknown>;
}

import { rxdbService } from './RxDBService';

export interface GraphNode {
  id: string;
  type: 'error' | 'skill' | 'capability' | 'property' | 'connection' | 'agent';
  label: string;
  position: { x: number; y: number; z: number };
  data: NodeData | NodeWithSlices;
  connections: string[]; // IDs of connected nodes
}

export interface GraphEdge {
  id: string;
  source: string;
  target: string;
  type: 'error_to_skill' | 'context_to_capability' | 'idea_to_property' | 'skill_chain' | 'agent_connection' | 'collaboration';
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

      const db = rxdbService.getDatabase();

      // Try to find an existing graph for the user
      const existing = await db.graphs.findOne({ selector: { userId } }).exec();
      if (existing) {
        try {
          const parsedNodes = (existing.nodes || []) as GraphNode[];
          const parsedEdges = (existing.edges || []) as GraphEdge[];
          const graph: PersonalGraph = {
            id: existing.id,
            userId: existing.userId,
            nodes: parsedNodes,
            edges: parsedEdges,
                    metadata: (existing.metadata as PersonalGraph['metadata']) || {
              createdAt: Date.now(),
              lastModified: Date.now(),
              version: 1,
              complexity: 0
            }
          };

          this.currentGraph = graph;
          return graph;
        } catch (err) {
          console.error('Failed parsing existing graph, creating new one:', err);
        }
      }

      // Create a new graph if none exists
      const graph = await this.createPersonalGraph(userId);
      return graph;
    } catch (error) {
      console.error('Failed to load personal graph:', error);
      return null;
    }
  }

  // Add error node to graph - Competitive process for SkillNode creation
  async addErrorNode(errorData: {
    errorId: string;
    errorType: string;
    description: string;
    context: Record<string, unknown>;
    timestamp: number;
  factualitySlice?: FactualitySlice;
  }): Promise<GraphNode> {
    if (!this.currentGraph) throw new Error('No active graph');

    const node: GraphNode = {
      id: `error_${errorData.errorId}`,
      type: 'error',
      label: errorData.description,
      position: this.calculateNodePosition(),
  data: { ...(errorData as ErrorNodeData), factualitySlice: errorData.factualitySlice } as NodeWithSlices,
      connections: []
    };

    this.currentGraph.nodes.push(node);

    // Attempt to find related skills automatically
    await this.findRelatedSkills(node);

    // Update graph
    await this.updateGraph();

    return node;
  }

  // Add context node to graph - Creates CapabilityNodes from MCP server information
  async addContextNode(contextData: {
    contextId: string;
    contextName: string;
    description: string;
    mcpServerInfo: Record<string, unknown>;
    category: string;
    timestamp: number;
  capabilitySlice?: FactualitySlice;
  }): Promise<GraphNode> {
    if (!this.currentGraph) throw new Error('No active graph');

    const capabilityData: CapabilityNodeData = {
      capabilityId: contextData.contextId,
      capabilityName: contextData.contextName,
      description: contextData.description,
      mcpServerInfo: contextData.mcpServerInfo,
      category: contextData.category,
      timestamp: contextData.timestamp
    };

    const node: GraphNode = {
      id: `capability_${contextData.contextId}`,
      type: 'capability',
      label: contextData.contextName,
      position: this.calculateNodePosition(),
  data: { ...capabilityData, capabilitySlice: contextData.capabilitySlice } as NodeWithSlices,
      connections: []
    };

    this.currentGraph.nodes.push(node);

    // Find related capabilities and create connections
    await this.findRelatedCapabilities(node);

    // Update graph
    await this.updateGraph();

    return node;
  }

  // Add idea node to graph - Collaborative process for PropertyNode creation
  async addIdeaNode(ideaData: {
    ideaId: string;
    ideaName: string;
    description: string;
    timestamp: number;
  feasibilitySlice?: FeasibilityReport;
  }): Promise<GraphNode> {
    if (!this.currentGraph) throw new Error('No active graph');

    // Generate feasibility report
    const feasibilityReport = await this.generateFeasibilityReport(ideaData);

    const propertyData: PropertyNodeData = {
      propertyId: ideaData.ideaId,
      propertyName: ideaData.ideaName,
      description: ideaData.description,
      feasibilityReport,
      collaborators: [this.currentGraph.userId], // Start with current user
      timestamp: ideaData.timestamp
    };

    const node: GraphNode = {
      id: `property_${ideaData.ideaId}`,
      type: 'property',
      label: ideaData.ideaName,
      position: this.calculateNodePosition(),
      data: { ...propertyData, feasibilitySlice: ideaData.feasibilitySlice } as NodeWithSlices,
      connections: []
    };

    this.currentGraph.nodes.push(node);

    // Find potential collaborators and similar ideas
    await this.findCollaborationOpportunities(node);

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

  // Helper method to find related capabilities for context nodes
  private async findRelatedCapabilities(capabilityNode: GraphNode): Promise<void> {
    const existingCapabilities = this.currentGraph?.nodes.filter(n => n.type === 'capability') || [];

    for (const existingCapability of existingCapabilities) {
      if ('category' in existingCapability.data && 'category' in capabilityNode.data) {
        const similarity = this.calculateSimilarity(
          existingCapability.data.category as string,
          capabilityNode.data.category as string
        );
        if (similarity > 0.6) {
          await this.createConnection(capabilityNode.id, existingCapability.id, 'context_to_capability');
        }
      }
    }
  }

  // Helper method to find collaboration opportunities for idea nodes
  private async findCollaborationOpportunities(propertyNode: GraphNode): Promise<void> {
    const existingProperties = this.currentGraph?.nodes.filter(n => n.type === 'property') || [];

    for (const existingProperty of existingProperties) {
      if ('description' in existingProperty.data && 'description' in propertyNode.data) {
        const similarity = this.calculateSimilarity(
          existingProperty.data.description as string,
          propertyNode.data.description as string
        );
        if (similarity > 0.4) {
          await this.createConnection(propertyNode.id, existingProperty.id, 'collaboration');
        }
      }
    }
  }

  // Generate feasibility report for ideas
  private async generateFeasibilityReport(ideaData: {
    ideaId: string;
    ideaName: string;
    description: string;
  }): Promise<PropertyNodeData['feasibilityReport']> {
    // In a real implementation, this would query external APIs, databases, etc.
    // For now, return a mock feasibility report
    return {
      exists: Math.random() > 0.7, // 30% chance idea already exists
      similarProjects: [
        `Similar project 1 for ${ideaData.ideaName}`,
        `Related concept: ${ideaData.ideaName} variant`
      ],
      feasibilityScore: Math.random() * 100, // Random score 0-100
      marketAnalysis: {
        marketSize: Math.floor(Math.random() * 1000000),
        competition: Math.floor(Math.random() * 50),
        trendScore: Math.random() * 10
      }
    };
  }

  // Calculate similarity between two strings (simple implementation)
  private calculateSimilarity(str1: string, str2: string): number {
    const words1 = str1.toLowerCase().split(/\s+/);
    const words2 = str2.toLowerCase().split(/\s+/);
    const intersection = words1.filter(word => words2.includes(word));
    const union = [...new Set([...words1, ...words2])];
    return intersection.length / union.length;
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

      // Upsert graph into graphs collection
      await db.graphs.upsert({
        id: graph.id,
        type: 'graph',
        userId: graph.userId,
        nodes: graph.nodes,
        edges: graph.edges,
        metadata: graph.metadata,
        timestamp: Date.now()
      });

      console.log('Graph saved to database (graphs collection)');
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
