// GraphChain API Service
// Provides access to GraphChain data including SkillNodes, ErrorNodes, and GraphNodes
//
// Proxied through KNIRVGATEWAY:
//   Graph ops   → /api/graph/*  (KNIRVGRAPH via graph.sock)
//   Chain ops   → /api/chain/*  (KNIRVCHAIN via chain.sock)
//   NRV ops     → /api/graph/*  (KNIRVGRAPH via graph.sock)

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
 * @property {number} height
 * @property {number} totalNodes
 * @property {number} totalEdges
 * @property {number} totalSkillNodes
 * @property {number} totalErrorNodes
 * @property {number} totalVectors
 * @property {number} avgResolutionTime
 */

// ===== API CONFIGURATION =====
// All endpoints use relative paths proxied through the KNIRVGATEWAY

const GRAPH_PROXY = '/api/graph';
const API_TIMEOUT = 3000; // ms
const CACHE_TTL = 30000;  // ms (30s)

// ===== GraphChain API CLASS =====

class GraphChainAPI {
  constructor() {
    /** @type {Map<string, {data: any, timestamp: number}>} */
    this.cache = new Map();
    this.cacheTimeout = CACHE_TTL;
    /** @type {Map<string, Promise<any>>} */
    this.requestQueue = new Map();
    this.retryAttempts = 1;
    this.retryDelay = 500;
  }

  /**
   * Make HTTP request with caching, dedup, retry logic, and error handling.
   * @private
   * @template T
   * @param {string} endpoint - Full gateway-proxy path (e.g. /api/graph/node/abc)
   * @param {RequestInit & {cache?: boolean}} [options]
   * @returns {Promise<T>}
   */
  async request(endpoint, options = {}) {
    const cacheKey = `${options.method || 'GET'}:${endpoint}`;

    // Return cached result if still valid
    if ((!options.method || options.method === 'GET') && options.cache && this.cache.has(cacheKey)) {
      const cached = this.cache.get(cacheKey);
      if (Date.now() - cached.timestamp < this.cacheTimeout) {
        return cached.data;
      }
    }

    // Dedup in-flight requests
    if (this.requestQueue.has(cacheKey)) {
      return await this.requestQueue.get(cacheKey);
    }

    const requestPromise = this._makeRequest(endpoint, options);
    this.requestQueue.set(cacheKey, requestPromise);

    try {
      const data = await requestPromise;
      // Cache successful GET responses
      if ((!options.method || options.method === 'GET') && options.cache) {
        this.cache.set(cacheKey, { data, timestamp: Date.now() });
      }
      return data;
    } finally {
      this.requestQueue.delete(cacheKey);
    }
  }

  /**
   * Perform the actual fetch with timeout and retry logic.
   * @private
   * @param {string} url
   * @param {RequestInit & {cache?: boolean}} options
   * @param {number} [attempt=1]
   * @returns {Promise<any>}
   */
  async _makeRequest(url, options, attempt = 1) {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), API_TIMEOUT);

    try {
      const response = await fetch(url, {
        headers: {
          'Content-Type': 'application/json',
          ...options.headers,
        },
        signal: controller.signal,
        ...options,
      });

      clearTimeout(timeoutId);

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      const contentType = response.headers.get('content-type');
      if (contentType && contentType.includes('application/json')) {
        return await response.json();
      }
      return await response.text();
    } catch (error) {
      clearTimeout(timeoutId);
      if (error.name === 'AbortError') {
        throw new Error(`Request timed out after ${API_TIMEOUT}ms: ${url}`);
      }
      if (attempt < this.retryAttempts && this._shouldRetry(error)) {
        await this._delay(this.retryDelay * attempt);
        return this._makeRequest(url, options, attempt + 1);
      }
      throw new Error(`API Error after ${attempt} attempt(s): ${error.message}`);
    }
  }

  /**
   * Determine if a request should be retried.
   * @private
   * @param {Error} error
   * @returns {boolean}
   */
  _shouldRetry(error) {
    return error.message.includes('fetch') ||
           error.message.includes('500') ||
           error.message.includes('502') ||
           error.message.includes('503') ||
           error.message.includes('504');
  }

  /**
   * Promise-based delay.
   * @private
   * @param {number} ms
   * @returns {Promise<void>}
   */
  _delay(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  /** Clear the entire cache. */
  clearCache() {
    this.cache.clear();
  }

  /** Remove expired entries from the cache. */
  clearExpiredCache() {
    const now = Date.now();
    for (const [key, value] of this.cache.entries()) {
      if (now - value.timestamp > this.cacheTimeout) {
        this.cache.delete(key);
      }
    }
  }

  // ===== BASIC GRAPH OPERATIONS =====

  /**
   * Get current GraphChain density (alias for getHeight for backward compat)
   * @returns {Promise<number>}
   */
  async getCurrentDensity() {
    const height = await this.getHeight();
    return height;
  }

  /**
   * Get current GraphChain block height
   * @returns {Promise<number>}
   */
  async getHeight() {
    const response = await this.request('/api/graph/height', { cache: true });
    return typeof response === 'object' ? (response.height || 0) : response;
  }

  /**
   * Get a specific node by ID.
   * @param {string} nodeId
   * @returns {Promise<GraphNode>}
   */
  async getNode(nodeId) {
    return await this.request(`${GRAPH_PROXY}/node/${encodeURIComponent(nodeId)}`, { cache: true });
  }

  /**
   * Get a specific edge by ID.
   * @param {string} edgeId
   * @returns {Promise<GraphEdge>}
   */
  async getEdge(edgeId) {
    return await this.request(`${GRAPH_PROXY}/edge/${encodeURIComponent(edgeId)}`, { cache: true });
  }

  /**
   * Get graph heads (nodes with no parents).
   * @returns {Promise<string[]>}
   */
  async getGraphHeads() {
    const response = await this.request(`${GRAPH_PROXY}/graph/heads`, { cache: true });
    return response.heads || response;
  }

  /**
   * Get neighbors of a node.
   * @param {string} nodeId
   * @returns {Promise<string[]>}
   */
  async getNodeNeighbors(nodeId) {
    return await this.request(`${GRAPH_PROXY}/graph/neighbors/${encodeURIComponent(nodeId)}`, { cache: true });
  }

  /**
   * Find path between two nodes.
   * @param {string} fromId
   * @param {string} toId
   * @param {number} [maxDepth=50]
   * @returns {Promise<string[]>}
   */
  async findPath(fromId, toId, maxDepth = 50) {
    const response = await this.request(
      `${GRAPH_PROXY}/graph/path/${encodeURIComponent(fromId)}/${encodeURIComponent(toId)}?max_depth=${maxDepth}`,
      { cache: true }
    );
    return response.path || response;
  }

  /**
   * Search nodes by query string.
   * @param {string} query
   * @returns {Promise<Array>}
   */
  async searchNodes(query) {
    return await this.request(`${GRAPH_PROXY}/search?q=${encodeURIComponent(query)}`);
  }

  // ===== NRV SYSTEM OPERATIONS =====

  /**
   * Get all SkillNodes.
   * @returns {Promise<SkillNode[]>}
   */
  async getAllSkills() {
    return await this.request(`${GRAPH_PROXY}/nrv/skills`, { cache: true });
  }

  /**
   * Get all ErrorNodes.
   * @returns {Promise<ErrorNode[]>}
   */
  async getAllErrors() {
    return await this.request(`${GRAPH_PROXY}/nrv/errors`, { cache: true });
  }

  /**
   * Get all NRV Vectors.
   * @returns {Promise<NRVVector[]>}
   */
  async getAllVectors() {
    return await this.request(`${GRAPH_PROXY}/nrv/vectors`, { cache: true });
  }

  /**
   * Get skills for a specific error type.
   * @param {string} errorType
   * @returns {Promise<SkillNode[]>}
   */
  async getSkillsForError(errorType) {
    return await this.request(`${GRAPH_PROXY}/nrv/skills/for-error/${encodeURIComponent(errorType)}`);
  }

  /**
   * Create a new SkillNode.
   * @param {Partial<SkillNode>} skill
   * @returns {Promise<{status: string, skill_id: string}>}
   */
  async createSkill(skill) {
    return await this.request(`${GRAPH_PROXY}/nrv/skills`, {
      method: 'POST',
      body: JSON.stringify(skill),
    });
  }

  /**
   * Create a new ErrorNode.
   * @param {Partial<ErrorNode>} error
   * @returns {Promise<{status: string, error_id: string}>}
   */
  async createError(error) {
    return await this.request(`${GRAPH_PROXY}/nrv/errors`, {
      method: 'POST',
      body: JSON.stringify(error),
    });
  }

  // ===== STATISTICS & AGGREGATIONS =====

  /**
   * Get comprehensive GraphChain statistics.
   * @returns {Promise<GraphChainStats>}
   */
  async getGraphChainStats() {
    try {
      const [height, skills, errors, vectors] = await Promise.all([
        this.getHeight().catch(() => 0),
        this.getAllSkills().catch(() => []),
        this.getAllErrors().catch(() => []),
        this.getAllVectors().catch(() => []),
      ]);

      // Calculate average resolution time from skill performance data
      const skillsWithPerformance = skills.filter(skill => skill.performance);
      const avgResolutionTime = skillsWithPerformance.length > 0
        ? skillsWithPerformance.reduce((sum, skill) => sum + (skill.performance?.avg_resolution_time || 0), 0) / skillsWithPerformance.length
        : 0;

      return {
        height: typeof height === 'number' ? height : (height?.height || 0),
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
   * Get recent SkillNodes (most recently created).
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
   * Get recent ErrorNodes (most recently created).
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
   * Check if the GraphChain API is reachable.
   * @returns {Promise<boolean>}
   */
  async healthCheck() {
    try {
      await this.getHeight();
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
