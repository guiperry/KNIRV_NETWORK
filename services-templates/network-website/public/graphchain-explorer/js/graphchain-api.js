/**
 * KNIRV GraphChain API Client
 * Handles all API communication with KNIRV GraphChain backend via KNIRVORACLE proxy
 */

class GraphChainAPI {
  constructor(baseUrl = '') {
    this.baseUrl = baseUrl;
    this.cache = new Map();
    this.cacheTimeout = 30000; // 30 seconds
    this.requestQueue = new Map();
    this.retryAttempts = 3;
    this.retryDelay = 1000; // 1 second
  }

  /**
   * Make HTTP request with caching, retry logic, and error handling
   */
  async request(endpoint, options = {}) {
    const url = `${this.baseUrl}/api/graphchain${endpoint}`;
    const cacheKey = `${options.method || 'GET'}:${url}`;
    
    // Check cache first (for GET requests)
    if ((!options.method || options.method === 'GET') && options.cache && this.cache.has(cacheKey)) {
      const cached = this.cache.get(cacheKey);
      if (Date.now() - cached.timestamp < this.cacheTimeout) {
        console.log(`Cache hit for ${endpoint}`);
        return cached.data;
      }
    }

    // Check if request is already in progress
    if (this.requestQueue.has(cacheKey)) {
      console.log(`Request already in progress for ${endpoint}, waiting...`);
      return await this.requestQueue.get(cacheKey);
    }

    // Create request promise
    const requestPromise = this._makeRequest(url, options);
    this.requestQueue.set(cacheKey, requestPromise);

    try {
      const data = await requestPromise;
      
      // Cache successful GET responses
      if ((!options.method || options.method === 'GET') && options.cache) {
        this.cache.set(cacheKey, { data, timestamp: Date.now() });
      }

      return data;
    } finally {
      // Remove from queue
      this.requestQueue.delete(cacheKey);
    }
  }

  /**
   * Internal method to make HTTP request with retry logic
   */
  async _makeRequest(url, options, attempt = 1) {
    try {
      const response = await fetch(url, {
        headers: {
          'Content-Type': 'application/json',
          ...options.headers
        },
        ...options
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      const contentType = response.headers.get('content-type');
      if (contentType && contentType.includes('application/json')) {
        return await response.json();
      } else {
        return await response.text();
      }
    } catch (error) {
      console.error(`Request failed (attempt ${attempt}):`, error);
      
      // Retry logic
      if (attempt < this.retryAttempts && this._shouldRetry(error)) {
        console.log(`Retrying request in ${this.retryDelay}ms...`);
        await this._delay(this.retryDelay * attempt);
        return this._makeRequest(url, options, attempt + 1);
      }
      
      throw new Error(`API Error after ${attempt} attempts: ${error.message}`);
    }
  }

  /**
   * Determine if request should be retried
   */
  _shouldRetry(error) {
    // Retry on network errors or 5xx server errors
    return error.message.includes('fetch') || 
           error.message.includes('500') || 
           error.message.includes('502') || 
           error.message.includes('503') || 
           error.message.includes('504');
  }

  /**
   * Delay utility for retry logic
   */
  _delay(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  /**
   * Clear cache
   */
  clearCache() {
    this.cache.clear();
  }

  /**
   * Clear expired cache entries
   */
  clearExpiredCache() {
    const now = Date.now();
    for (const [key, value] of this.cache.entries()) {
      if (now - value.timestamp > this.cacheTimeout) {
        this.cache.delete(key);
      }
    }
  }

  // KNIRV GraphChain-specific API methods

  /**
   * Get current KNIRV GraphChain height
   */
  async getHeight() {
    const response = await this.request('/height', { cache: true });
    return typeof response === 'object' ? response.height : response;
  }

  /**
   * Get all SkillNodes
   */
  async getSkills() {
    return await this.request('/nrv/skills', { cache: true });
  }

  /**
   * Get all ErrorNodes
   */
  async getErrors() {
    return await this.request('/nrv/errors', { cache: true });
  }

  /**
   * Get all NRV vectors
   */
  async getVectors() {
    return await this.request('/nrv/vectors', { cache: true });
  }

  /**
   * Get SkillNodes for a specific error type
   */
  async getSkillsForError(errorType) {
    return await this.request(`/nrv/skills/for-error/${encodeURIComponent(errorType)}`);
  }

  /**
   * Search nodes by query
   */
  async searchNodes(query) {
    return await this.request(`/search?q=${encodeURIComponent(query)}`);
  }

  /**
   * Get specific node by ID
   */
  async getNode(nodeId) {
    return await this.request(`/node/${encodeURIComponent(nodeId)}`, { cache: true });
  }

  /**
   * Get specific edge by ID
   */
  async getEdge(edgeId) {
    return await this.request(`/edge/${encodeURIComponent(edgeId)}`, { cache: true });
  }

  /**
   * Get graph heads
   */
  async getGraphHeads() {
    const response = await this.request('/graph/heads', { cache: true });
    return response.heads || response;
  }

  /**
   * Get node neighbors
   */
  async getNodeNeighbors(nodeId) {
    return await this.request(`/graph/neighbors/${encodeURIComponent(nodeId)}`, { cache: true });
  }

  /**
   * Find path between two nodes
   */
  async findPath(fromId, toId, maxDepth = 50) {
    const response = await this.request(
      `/graph/path/${encodeURIComponent(fromId)}/${encodeURIComponent(toId)}?max_depth=${maxDepth}`,
      { cache: true }
    );
    return response.path || response;
  }

  /**
   * Create new SkillNode
   */
  async createSkill(skill) {
    return await this.request('/nrv/skills', {
      method: 'POST',
      body: JSON.stringify(skill)
    });
  }

  /**
   * Create new ErrorNode
   */
  async createError(error) {
    return await this.request('/nrv/errors', {
      method: 'POST',
      body: JSON.stringify(error)
    });
  }

  // Helper methods for dashboard data

  /**
   * Get comprehensive KNIRV GraphChain statistics
   */
  async getGraphChainStats() {
    try {
      const [height, skills, errors, vectors] = await Promise.all([
        this.getHeight().catch(() => 0),
        this.getSkills().catch(() => []),
        this.getErrors().catch(() => []),
        this.getVectors().catch(() => [])
      ]);

      // Calculate average resolution time from skill performance data
      const skillsWithPerformance = skills.filter(skill => skill.performance);
      const avgResolutionTime = skillsWithPerformance.length > 0
        ? skillsWithPerformance.reduce((sum, skill) => sum + (skill.performance?.avg_resolution_time || 0), 0) / skillsWithPerformance.length
        : 0;

      return {
        height: typeof height === 'number' ? height : (height?.height || 0),
        totalNodes: 0, // Would need additional endpoint
        totalEdges: 0, // Would need additional endpoint
        totalSkillNodes: skills.length,
        totalErrorNodes: errors.length,
        totalVectors: vectors.length,
        avgResolutionTime
      };
    } catch (error) {
      console.error('Failed to fetch KNIRV GraphChain stats:', error);
      throw new Error('Failed to fetch KNIRV GraphChain statistics');
    }
  }

  /**
   * Get recent SkillNodes (most recently created)
   */
  async getRecentSkills(count = 10) {
    const skills = await this.getSkills();
    return skills
      .sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())
      .slice(0, count);
  }

  /**
   * Get recent ErrorNodes (most recently created)
   */
  async getRecentErrors(count = 10) {
    const errors = await this.getErrors();
    return errors
      .sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())
      .slice(0, count);
  }

  /**
   * Get filtered SkillNodes
   */
  async getFilteredSkills(filters = {}) {
    const skills = await this.getSkills();
    
    return skills.filter(skill => {
      if (filters.skillType && !skill.skill_type.toLowerCase().includes(filters.skillType.toLowerCase())) {
        return false;
      }
      
      if (filters.validated !== undefined) {
        const isValidated = skill.validation?.is_validated || false;
        if (filters.validated !== isValidated) {
          return false;
        }
      }
      
      if (filters.capability && !skill.capabilities.some(cap => 
        cap.toLowerCase().includes(filters.capability.toLowerCase())
      )) {
        return false;
      }
      
      return true;
    });
  }

  /**
   * Get filtered ErrorNodes
   */
  async getFilteredErrors(filters = {}) {
    const errors = await this.getErrors();
    
    return errors.filter(error => {
      if (filters.errorType && !error.error_type.toLowerCase().includes(filters.errorType.toLowerCase())) {
        return false;
      }
      
      if (filters.severity !== undefined && error.severity !== filters.severity) {
        return false;
      }
      
      if (filters.status && error.resolution_status !== filters.status) {
        return false;
      }
      
      return true;
    });
  }
}

// Create global API instance
window.graphChainAPI = new GraphChainAPI();

// Auto-clear expired cache every 5 minutes
setInterval(() => {
  window.graphChainAPI.clearExpiredCache();
}, 5 * 60 * 1000);

// Export for module usage
if (typeof module !== 'undefined' && module.exports) {
  module.exports = GraphChainAPI;
}
