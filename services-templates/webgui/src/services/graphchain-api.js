// GraphChain API Service
// Provides access to GraphChain data including SkillNodes, ErrorNodes, and GraphNodes

// ===== TYPE DEFINITIONS (JSDoc) =====

/**
 * @typedef {Object} GraphNode
 * @property {string} id
 * @property {string} node_type
 * @property {Object} data
 * @property {string[]} parents
 * @property {string[]} children
 * @property {number} weight
 * @property {string} timestamp
 * @property {Object} metadata
 */

/**
 * @typedef {Object} GraphEdge
 * @property {string} id
 * @property {string} from
 * @property {string} to
 * @property {number} weight
 * @property {string} edge_type
 * @property {Object} metadata
 * @property {string} timestamp
 */

/**
 * @typedef {Object} SkillNodePerformance
 * @property {number} success_rate
 * @property {number} avg_resolution_time
 * @property {number} total_resolutions
 */

/**
 * @typedef {Object} SkillNodeValidation
 * @property {boolean} is_validated
 * @property {string[]} validated_by
 * @property {number} validation_score
 * @property {string} last_validated
 */

/**
 * @typedef {Object} SkillNode
 * @property {string} id
 * @property {string} skill_type
 * @property {string[]} capabilities
 * @property {Object} requirements
 * @property {SkillNodePerformance} [performance]
 * @property {SkillNodeValidation} [validation]
 * @property {string} timestamp
 */

/**
 * @typedef {('pending'|'resolved'|'failed')} ResolutionStatus
 */

/**
 * @typedef {Object} ErrorNode
 * @property {string} id
 * @property {string} error_type
 * @property {string} description
 * @property {Object} context
 * @property {number} severity
 * @property {string} timestamp
 * @property {ResolutionStatus} [resolution_status]
 * @property {string[]} [resolved_by]
 */

/**
 * @typedef {Object} NRVVector
 * @property {string} id
 * @property {string} source_peer
 * @property {string} target_hash
 * @property {number[]} coordinates
 * @property {number} confidence
 * @property {string} timestamp
 * @property {Object} metadata
 */

/**
 * @typedef {Object} GraphChainStats
 * @property {number} density
 * @property {number} totalNodes
 * @property {number} totalEdges
 * @property {number} totalSkillNodes
 * @property {number} totalErrorNodes
 * @property {number} totalVectors
 * @property {number} avgResolutionTime
 */

// ===== API CONFIGURATION =====

const API_BASE_URL = process.env.NEXT_PUBLIC_KNIRVCHAIN_API_URL || 'http://localhost:8080';

// ===== GraphChain API CLASS =====

class GraphChainAPI {
  /**
   * Make an API request
   * @private
   * @template T
   * @param {string} endpoint
   * @param {RequestInit} [options]
   * @returns {Promise<T>}
   */
  async request(endpoint, options = {}) {
    try {
      const response = await fetch(`${API_BASE_URL}${endpoint}`, {
        headers: {
          'Content-Type': 'application/json',
          ...options.headers,
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

  // ===== BASIC GRAPH OPERATIONS =====

  /**
   * Get current GraphChain density
   * @returns {Promise<number>}
   */
  async getCurrentDensity() {
    const response = await this.request('/density');
    return response.density;
  }

  /**
   * Get a specific node by ID
   * @param {string} nodeId
   * @returns {Promise<GraphNode>}
   */
  async getNode(nodeId) {
    return await this.request(`/node/${nodeId}`);
  }

  /**
   * Get a specific edge by ID
   * @param {string} edgeId
   * @returns {Promise<GraphEdge>}
   */
  async getEdge(edgeId) {
    return await this.request(`/edge/${edgeId}`);
  }

  /**
   * Get graph heads (nodes with no parents)
   * @returns {Promise<string[]>}
   */
  async getGraphHeads() {
    const response = await this.request('/graph/heads');
    return response.heads;
  }

  /**
   * Get neighbors of a node
   * @param {string} nodeId
   * @returns {Promise<string[]>}
   */
  async getNodeNeighbors(nodeId) {
    return await this.request(`/graph/neighbors/${nodeId}`);
  }

  /**
   * Find path between two nodes
   * @param {string} fromId
   * @param {string} toId
   * @param {number} [maxDepth=50]
   * @returns {Promise<string[]>}
   */
  async findPath(fromId, toId, maxDepth = 50) {
    const response = await this.request(`/graph/path/${fromId}/${toId}?max_depth=${maxDepth}`);
    return response.path;
  }

  // ===== NRV SYSTEM OPERATIONS =====

  /**
   * Get all SkillNodes
   * @returns {Promise<SkillNode[]>}
   */
  async getAllSkills() {
    return await this.request('/nrv/skills');
  }

  /**
   * Get all ErrorNodes
   * @returns {Promise<ErrorNode[]>}
   */
  async getAllErrors() {
    return await this.request('/nrv/errors');
  }

  /**
   * Get all NRV Vectors
   * @returns {Promise<NRVVector[]>}
   */
  async getAllVectors() {
    return await this.request('/nrv/vectors');
  }

  /**
   * Get skills for a specific error type
   * @param {string} errorType
   * @returns {Promise<SkillNode[]>}
   */
  async getSkillsForError(errorType) {
    return await this.request(`/nrv/skills/for-error/${errorType}`);
  }

  /**
   * Create a new SkillNode
   * @param {Partial<SkillNode>} skill
   * @returns {Promise<{status: string, skill_id: string}>}
   */
  async createSkill(skill) {
    return await this.request('/nrv/skills', {
      method: 'POST',
      body: JSON.stringify(skill),
    });
  }

  /**
   * Create a new ErrorNode
   * @param {Partial<ErrorNode>} error
   * @returns {Promise<{status: string, error_id: string}>}
   */
  async createError(error) {
    return await this.request('/nrv/errors', {
      method: 'POST',
      body: JSON.stringify(error),
    });
  }

  // ===== STATISTICS & AGGREGATIONS =====

  /**
   * Get comprehensive GraphChain statistics
   * @returns {Promise<GraphChainStats>}
   */
  async getGraphChainStats() {
    try {
      const [density, skills, errors, vectors] = await Promise.all([
        this.getCurrentDensity(),
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
        density,
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

  /**
   * Get recent SkillNodes (most recently created)
   * @param {number} [count=10]
   * @returns {Promise<SkillNode[]>}
   */
  async getRecentSkills(count = 10) {
    const skills = await this.getAllSkills();
    return skills
      .sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())
      .slice(0, count);
  }

  /**
   * Get recent ErrorNodes (most recently created)
   * @param {number} [count=10]
   * @returns {Promise<ErrorNode[]>}
   */
  async getRecentErrors(count = 10) {
    const errors = await this.getAllErrors();
    return errors
      .sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())
      .slice(0, count);
  }

  // ===== UTILITY METHODS =====

  /**
   * Check if the GraphChain API is reachable
   * @returns {Promise<boolean>}
   */
  async healthCheck() {
    try {
      await this.getCurrentDensity();
      return true;
    } catch (error) {
      return false;
    }
  }
}

// Export a singleton instance
export const graphChainApi = new GraphChainAPI();

// Also export the class for testing purposes
export { GraphChainAPI };
