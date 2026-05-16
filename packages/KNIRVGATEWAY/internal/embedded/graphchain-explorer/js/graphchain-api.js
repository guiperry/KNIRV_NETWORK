/**
 * KNIRV GraphChain API Client
 * Proxies through KNIRVGATEWAY:
 *   Graph ops   → /api/graph/*   (KNIRVGRAPH via graph.sock)
 *   Chain ops   → /api/chain/*   (KNIRVCHAIN via chain.sock)
 */

class GraphChainAPI {
  constructor() {
    this.cache = new Map();
    this.cacheTimeout = 30000;
    this.requestQueue = new Map();
    this.retryAttempts = 3;
    this.retryDelay = 1000;
  }

  /**
   * Make HTTP request with caching, retry logic, and error handling.
   * endpoint must be the full gateway-proxy path (e.g. /api/graph/node/abc).
   */
  async request(endpoint, options = {}) {
    const url = endpoint;
    const cacheKey = `${options.method || 'GET'}:${url}`;
    
    if ((!options.method || options.method === 'GET') && options.cache && this.cache.has(cacheKey)) {
      const cached = this.cache.get(cacheKey);
      if (Date.now() - cached.timestamp < this.cacheTimeout) {
        return cached.data;
      }
    }

    if (this.requestQueue.has(cacheKey)) {
      return await this.requestQueue.get(cacheKey);
    }

    const requestPromise = this._makeRequest(url, options);
    this.requestQueue.set(cacheKey, requestPromise);

    try {
      const data = await requestPromise;
      if ((!options.method || options.method === 'GET') && options.cache) {
        this.cache.set(cacheKey, { data, timestamp: Date.now() });
      }
      return data;
    } finally {
      this.requestQueue.delete(cacheKey);
    }
  }

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
      }
      return await response.text();
    } catch (error) {
      if (attempt < this.retryAttempts && this._shouldRetry(error)) {
        await this._delay(this.retryDelay * attempt);
        return this._makeRequest(url, options, attempt + 1);
      }
      throw new Error(`API Error after ${attempt} attempts: ${error.message}`);
    }
  }

  _shouldRetry(error) {
    return error.message.includes('fetch') || 
           error.message.includes('500') || 
           error.message.includes('502') || 
           error.message.includes('503') || 
           error.message.includes('504');
  }

  _delay(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  clearCache() { this.cache.clear(); }

  clearExpiredCache() {
    const now = Date.now();
    for (const [key, value] of this.cache.entries()) {
      if (now - value.timestamp > this.cacheTimeout) this.cache.delete(key);
    }
  }

  // ── Chain operations (KNIRVCHAIN) ──────────────────────────────────────

  async getHeight() {
    const response = await this.request('/api/graph/height', { cache: true });
    return typeof response === 'object' ? response.height : response;
  }

  async getSkills() {
    return await this.request('/api/graph/nrv/skills', { cache: true });
  }

  async getErrors() {
    return await this.request('/api/graph/nrv/errors', { cache: true });
  }

  async getVectors() {
    return await this.request('/api/graph/nrv/vectors', { cache: true });
  }

  async getSkillsForError(errorType) {
    return await this.request(`/api/graph/nrv/skills/for-error/${encodeURIComponent(errorType)}`);
  }

  async createSkill(skill) {
    return await this.request('/api/graph/nrv/skills', {
      method: 'POST',
      body: JSON.stringify(skill)
    });
  }

  async createError(error) {
    return await this.request('/api/graph/nrv/errors', {
      method: 'POST',
      body: JSON.stringify(error)
    });
  }

  // ── Graph operations (KNIRVGRAPH) ──────────────────────────────────────

  async searchNodes(query) {
    return await this.request(`/api/graph/search?q=${encodeURIComponent(query)}`);
  }

  async getNode(nodeId) {
    return await this.request(`/api/graph/node/${encodeURIComponent(nodeId)}`, { cache: true });
  }

  async getEdge(edgeId) {
    return await this.request(`/api/graph/edge/${encodeURIComponent(edgeId)}`, { cache: true });
  }

  async getGraphHeads() {
    const response = await this.request('/api/graph/graph/heads', { cache: true });
    return response.heads || response;
  }

  async getNodeNeighbors(nodeId) {
    return await this.request(`/api/graph/graph/neighbors/${encodeURIComponent(nodeId)}`, { cache: true });
  }

  async findPath(fromId, toId, maxDepth = 50) {
    const response = await this.request(
      `/api/graph/graph/path/${encodeURIComponent(fromId)}/${encodeURIComponent(toId)}?max_depth=${maxDepth}`,
      { cache: true }
    );
    return response.path || response;
  }

  // ── Composite ──────────────────────────────────────────────────────────

  async getGraphChainStats() {
    try {
      const [height, skills, errors, vectors] = await Promise.all([
        this.getHeight().catch(() => 0),
        this.getSkills().catch(() => []),
        this.getErrors().catch(() => []),
        this.getVectors().catch(() => [])
      ]);

      const skillsWithPerformance = skills.filter(skill => skill.performance);
      const avgResolutionTime = skillsWithPerformance.length > 0
        ? skillsWithPerformance.reduce((sum, skill) => sum + (skill.performance?.avg_resolution_time || 0), 0) / skillsWithPerformance.length
        : 0;

      return {
        height: typeof height === 'number' ? height : (height?.height || 0),
        totalNodes: 0,
        totalEdges: 0,
        totalSkillNodes: skills.length,
        totalErrorNodes: errors.length,
        totalVectors: vectors.length,
        avgResolutionTime
      };
    } catch (error) {
      throw new Error('Failed to fetch KNIRV GraphChain statistics');
    }
  }

  async getRecentSkills(count = 10) {
    const skills = await this.getSkills();
    return skills
      .sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())
      .slice(0, count);
  }

  async getRecentErrors(count = 10) {
    const errors = await this.getErrors();
    return errors
      .sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())
      .slice(0, count);
  }

  async getFilteredSkills(filters = {}) {
    const skills = await this.getSkills();
    return skills.filter(skill => {
      if (filters.skillType && !skill.skill_type.toLowerCase().includes(filters.skillType.toLowerCase())) return false;
      if (filters.validated !== undefined) {
        const isValidated = skill.validation?.is_validated || false;
        if (filters.validated !== isValidated) return false;
      }
      if (filters.capability && !skill.capabilities.some(cap => cap.toLowerCase().includes(filters.capability.toLowerCase()))) return false;
      return true;
    });
  }

  async getFilteredErrors(filters = {}) {
    const errors = await this.getErrors();
    return errors.filter(error => {
      if (filters.errorType && !error.error_type.toLowerCase().includes(filters.errorType.toLowerCase())) return false;
      if (filters.severity !== undefined && error.severity !== filters.severity) return false;
      if (filters.status && error.resolution_status !== filters.status) return false;
      return true;
    });
  }
}

window.graphChainAPI = new GraphChainAPI();

setInterval(() => {
  window.graphChainAPI.clearExpiredCache();
}, 5 * 60 * 1000);

if (typeof module !== 'undefined' && module.exports) {
  module.exports = GraphChainAPI;
}
