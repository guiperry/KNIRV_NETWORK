// GraphChain API Types
export interface GraphNode {
  id: string;
  node_type: string;
  data: Record<string, unknown>;
  parents: string[];
  children: string[];
  weight: number;
  timestamp: string;
  metadata: Record<string, unknown>;
}

export interface GraphEdge {
  id: string;
  from: string;
  to: string;
  weight: number;
  edge_type: string;
  metadata: Record<string, unknown>;
  timestamp: string;
}

export interface SkillNode {
  id: string;
  skill_type: string;
  capabilities: string[];
  requirements: Record<string, unknown>;
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
  context: Record<string, unknown>;
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
  metadata: Record<string, unknown>;
}

export interface GraphChainStats {
  chainHeight: number;
  totalNodes: number;
  totalEdges: number;
  totalSkillNodes: number;
  totalErrorNodes: number;
  totalVectors: number;
  avgResolutionTime: number;
}

// Proxy through the KNIRVGATEWAY — all endpoints use relative paths
// Graph operations → /api/graph/*  (KNIRVGRAPH via graph.sock)
// Chain/NRV operations → /api/chain/*  (KNIRVCHAIN via chain.sock)

const GRAPH_PROXY = '/api/graph';
const CHAIN_PROXY = '/api/chain';

// Chain response from KNIRVCHAIN /chain endpoint
export interface ChainResponse {
  blocks: unknown[];
  transaction_pool?: unknown[];
  chain_address?: string;
  reflections?: Record<string, boolean>;
  chain_id?: string;
}

class GraphChainAPI {
  private async request<T>(endpoint: string, options?: RequestInit): Promise<T> {
    try {
      const response = await fetch(endpoint, {
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

  // ── Chain operations (KNIRVCHAIN) ──────────────────────────────────────

  async getChainHeight(): Promise<number> {
    const chain = await this.request<ChainResponse>(`${CHAIN_PROXY}/chain`);
    return chain.blocks.length;
  }

  async getNode(nodeId: string): Promise<GraphNode> {
    return await this.request<GraphNode>(`${GRAPH_PROXY}/node/${nodeId}`);
  }

  async getEdge(edgeId: string): Promise<GraphEdge> {
    return await this.request<GraphEdge>(`${GRAPH_PROXY}/edge/${edgeId}`);
  }

  async getGraphHeads(): Promise<string[]> {
    const response = await this.request<{ heads: string[] }>(`${GRAPH_PROXY}/graph/heads`);
    return response.heads;
  }

  async getNodeNeighbors(nodeId: string): Promise<string[]> {
    return await this.request<string[]>(`${GRAPH_PROXY}/graph/neighbors/${nodeId}`);
  }

  async findPath(fromId: string, toId: string, maxDepth: number = 50): Promise<string[]> {
    const response = await this.request<{ path: string[] }>(`${GRAPH_PROXY}/graph/path/${fromId}/${toId}?max_depth=${maxDepth}`);
    return response.path;
  }

  // ── NRV / Chain operations (KNIRVCHAIN) ────────────────────────────────

  async getAllSkills(): Promise<SkillNode[]> {
    return await this.request<SkillNode[]>(`${CHAIN_PROXY}/nrv/skills`);
  }

  async getAllErrors(): Promise<ErrorNode[]> {
    return await this.request<ErrorNode[]>(`${CHAIN_PROXY}/nrv/errors`);
  }

  async getAllVectors(): Promise<NRVVector[]> {
    return await this.request<NRVVector[]>(`${CHAIN_PROXY}/nrv/vectors`);
  }

  async getSkillsForError(errorType: string): Promise<SkillNode[]> {
    return await this.request<SkillNode[]>(`${CHAIN_PROXY}/nrv/skills/for-error/${errorType}`);
  }

  async createSkill(skill: Partial<SkillNode>): Promise<{ status: string; skill_id: string }> {
    return await this.request<{ status: string; skill_id: string }>(`${CHAIN_PROXY}/nrv/skills`, {
      method: 'POST',
      body: JSON.stringify(skill),
    });
  }

  async createError(error: Partial<ErrorNode>): Promise<{ status: string; error_id: string }> {
    return await this.request<{ status: string; error_id: string }>(`${CHAIN_PROXY}/nrv/errors`, {
      method: 'POST',
      body: JSON.stringify(error),
    });
  }

  // ── Composite ──────────────────────────────────────────────────────────

  async getGraphChainStats(): Promise<GraphChainStats> {
    try {
      const [chainHeight, skills, errors, vectors] = await Promise.all([
        this.getChainHeight(),
        this.getAllSkills(),
        this.getAllErrors(),
        this.getAllVectors(),
      ]);

      const skillsWithPerformance = skills.filter(skill => skill.performance);
      const avgResolutionTime = skillsWithPerformance.length > 0
        ? skillsWithPerformance.reduce((sum, skill) => sum + (skill.performance?.avg_resolution_time || 0), 0) / skillsWithPerformance.length
        : 0;

      return {
        chainHeight,
        totalNodes: 0,
        totalEdges: 0,
        totalSkillNodes: skills.length,
        totalErrorNodes: errors.length,
        totalVectors: vectors.length,
        avgResolutionTime,
      };
    } catch {
      throw new Error('Failed to fetch GraphChain stats');
    }
  }

  async getRecentSkills(count: number = 10): Promise<SkillNode[]> {
    const skills = await this.getAllSkills();
    return skills
      .sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())
      .slice(0, count);
  }

  async getRecentErrors(count: number = 10): Promise<ErrorNode[]> {
    const errors = await this.getAllErrors();
    return errors
      .sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())
      .slice(0, count);
  }
}

export const graphChainApi = new GraphChainAPI();
