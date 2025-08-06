// GraphChain API Types
export interface GraphNode {
  id: string;
  node_type: string;
  data: any;
  parents: string[];
  children: string[];
  weight: number;
  timestamp: string;
  metadata: Record<string, any>;
}

export interface GraphEdge {
  id: string;
  from: string;
  to: string;
  weight: number;
  edge_type: string;
  metadata: Record<string, any>;
  timestamp: string;
}

export interface SkillNode {
  id: string;
  skill_type: string;
  capabilities: string[];
  requirements: Record<string, any>;
  performance?: {
    success_rate: number;
    avg_resolution_time: number;
    total_resolutions: number;
  };
  validation?: {
    is_validated: boolean;
    validated_by: string[];
    validation_score: number;
    last_validated: string;
  };
  timestamp: string;
}

export interface ErrorNode {
  id: string;
  error_type: string;
  description: string;
  context: Record<string, any>;
  severity: number;
  timestamp: string;
  resolution_status?: 'pending' | 'resolved' | 'failed';
  resolved_by?: string[];
}

export interface NRVVector {
  id: string;
  source_peer: string;
  target_hash: string;
  coordinates: number[];
  confidence: number;
  timestamp: string;
  metadata: Record<string, any>;
}

export interface GraphChainStats {
  height: number;
  totalNodes: number;
  totalEdges: number;
  totalSkillNodes: number;
  totalErrorNodes: number;
  totalVectors: number;
  avgResolutionTime: number;
}

// API Configuration
const API_BASE_URL = 'http://localhost:8080';

class GraphChainAPI {
  private async request<T>(endpoint: string, options?: RequestInit): Promise<T> {
    try {
      const response = await fetch(`${API_BASE_URL}${endpoint}`, {
        headers: {
          'Content-Type': 'application/json',
          ...options?.headers,
        },
        ...options,
      });

      if (!response.ok) {
        throw new Error(`API Error: ${response.status} ${response.statusText}`);
      }

      return await response.json();
    } catch (error) {
      if (error instanceof Error) {
        throw error;
      }
      throw new Error('Network error');
    }
  }

  async getCurrentHeight(): Promise<number> {
    const response = await this.request<{ height: number }>('/height');
    return response.height;
  }

  async getNode(nodeId: string): Promise<GraphNode> {
    return await this.request<GraphNode>(`/node/${nodeId}`);
  }

  async getEdge(edgeId: string): Promise<GraphEdge> {
    return await this.request<GraphEdge>(`/edge/${edgeId}`);
  }

  async getGraphHeads(): Promise<string[]> {
    const response = await this.request<{ heads: string[] }>('/graph/heads');
    return response.heads;
  }

  async getNodeNeighbors(nodeId: string): Promise<string[]> {
    return await this.request<string[]>(`/graph/neighbors/${nodeId}`);
  }

  async findPath(fromId: string, toId: string, maxDepth: number = 50): Promise<string[]> {
    const response = await this.request<{ path: string[] }>(`/graph/path/${fromId}/${toId}?max_depth=${maxDepth}`);
    return response.path;
  }

  // NRV System APIs
  async getAllSkills(): Promise<SkillNode[]> {
    return await this.request<SkillNode[]>('/nrv/skills');
  }

  async getAllErrors(): Promise<ErrorNode[]> {
    return await this.request<ErrorNode[]>('/nrv/errors');
  }

  async getAllVectors(): Promise<NRVVector[]> {
    return await this.request<NRVVector[]>('/nrv/vectors');
  }

  async getSkillsForError(errorType: string): Promise<SkillNode[]> {
    return await this.request<SkillNode[]>(`/nrv/skills/for-error/${errorType}`);
  }

  async createSkill(skill: Partial<SkillNode>): Promise<{ status: string; skill_id: string }> {
    return await this.request<{ status: string; skill_id: string }>('/nrv/skills', {
      method: 'POST',
      body: JSON.stringify(skill),
    });
  }

  async createError(error: Partial<ErrorNode>): Promise<{ status: string; error_id: string }> {
    return await this.request<{ status: string; error_id: string }>('/nrv/errors', {
      method: 'POST',
      body: JSON.stringify(error),
    });
  }

  async getGraphChainStats(): Promise<GraphChainStats> {
    try {
      const [height, skills, errors, vectors] = await Promise.all([
        this.getCurrentHeight(),
        this.getAllSkills(),
        this.getAllErrors(),
        this.getAllVectors(),
      ]);

      // Calculate average resolution time from skill performance data
      const skillsWithPerformance = skills.filter(skill => skill.performance);
      const avgResolutionTime = skillsWithPerformance.length > 0
        ? skillsWithPerformance.reduce((sum, skill) => sum + (skill.performance?.avg_resolution_time || 0), 0) / skillsWithPerformance.length
        : 0;

      return {
        height,
        totalNodes: 0, // Would need additional endpoint to get total graph nodes
        totalEdges: 0, // Would need additional endpoint to get total graph edges
        totalSkillNodes: skills.length,
        totalErrorNodes: errors.length,
        totalVectors: vectors.length,
        avgResolutionTime,
      };
    } catch (error) {
      throw new Error('Failed to fetch GraphChain stats');
    }
  }

  // Helper method to get recent skills (most recently created)
  async getRecentSkills(count: number = 10): Promise<SkillNode[]> {
    const skills = await this.getAllSkills();
    return skills
      .sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())
      .slice(0, count);
  }

  // Helper method to get recent errors (most recently created)
  async getRecentErrors(count: number = 10): Promise<ErrorNode[]> {
    const errors = await this.getAllErrors();
    return errors
      .sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())
      .slice(0, count);
  }
}

export const graphChainApi = new GraphChainAPI();