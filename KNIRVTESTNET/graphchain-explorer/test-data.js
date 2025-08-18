/**
 * Test Data for KNIRV GraphChain Explorer
 * Mock data for development and testing
 */

// Mock KNIRV GraphChain height data
const mockHeight = {
  height: 12847
};

// Mock SkillNodes data
const mockSkills = [
  {
    id: "skill_001",
    skill_type: "Natural Language Processing",
    capabilities: ["text_analysis", "sentiment_analysis", "entity_extraction"],
    timestamp: new Date(Date.now() - 1000 * 60 * 30).toISOString(), // 30 minutes ago
    validation: {
      is_validated: true,
      validation_score: 0.92
    },
    performance: {
      success_rate: 0.89,
      avg_resolution_time: 2.3,
      total_resolutions: 1247
    }
  },
  {
    id: "skill_002", 
    skill_type: "Computer Vision",
    capabilities: ["image_classification", "object_detection", "face_recognition"],
    timestamp: new Date(Date.now() - 1000 * 60 * 45).toISOString(), // 45 minutes ago
    validation: {
      is_validated: true,
      validation_score: 0.87
    },
    performance: {
      success_rate: 0.94,
      avg_resolution_time: 1.8,
      total_resolutions: 892
    }
  },
  {
    id: "skill_003",
    skill_type: "Data Analysis",
    capabilities: ["statistical_analysis", "data_visualization", "pattern_recognition"],
    timestamp: new Date(Date.now() - 1000 * 60 * 60).toISOString(), // 1 hour ago
    validation: {
      is_validated: false,
      validation_score: 0.73
    },
    performance: {
      success_rate: 0.76,
      avg_resolution_time: 4.2,
      total_resolutions: 543
    }
  },
  {
    id: "skill_004",
    skill_type: "Machine Learning",
    capabilities: ["model_training", "prediction", "feature_engineering"],
    timestamp: new Date(Date.now() - 1000 * 60 * 90).toISOString(), // 1.5 hours ago
    validation: {
      is_validated: true,
      validation_score: 0.95
    },
    performance: {
      success_rate: 0.91,
      avg_resolution_time: 3.1,
      total_resolutions: 2156
    }
  },
  {
    id: "skill_005",
    skill_type: "Audio Processing",
    capabilities: ["speech_recognition", "audio_classification", "noise_reduction"],
    timestamp: new Date(Date.now() - 1000 * 60 * 120).toISOString(), // 2 hours ago
    validation: {
      is_validated: false,
      validation_score: 0.68
    },
    performance: {
      success_rate: 0.72,
      avg_resolution_time: 2.9,
      total_resolutions: 387
    }
  }
];

// Mock ErrorNodes data
const mockErrors = [
  {
    id: "error_001",
    error_type: "Memory Allocation Error",
    description: "Insufficient memory allocated for large dataset processing",
    timestamp: new Date(Date.now() - 1000 * 60 * 15).toISOString(), // 15 minutes ago
    severity: 3, // High
    resolution_status: "pending"
  },
  {
    id: "error_002",
    error_type: "Network Timeout",
    description: "Connection timeout while fetching external API data",
    timestamp: new Date(Date.now() - 1000 * 60 * 25).toISOString(), // 25 minutes ago
    severity: 2, // Medium
    resolution_status: "resolved"
  },
  {
    id: "error_003",
    error_type: "Data Validation Error",
    description: "Invalid input format detected in training dataset",
    timestamp: new Date(Date.now() - 1000 * 60 * 40).toISOString(), // 40 minutes ago
    severity: 1, // Low
    resolution_status: "failed"
  },
  {
    id: "error_004",
    error_type: "Model Convergence Error",
    description: "Machine learning model failed to converge within iteration limit",
    timestamp: new Date(Date.now() - 1000 * 60 * 55).toISOString(), // 55 minutes ago
    severity: 4, // Critical
    resolution_status: "pending"
  }
];

// Mock vectors data
const mockVectors = [
  { id: "vector_001", dimension: 512, type: "embedding" },
  { id: "vector_002", dimension: 256, type: "feature" },
  { id: "vector_003", dimension: 1024, type: "representation" }
];

// Mock API responses
const mockApiResponses = {
  '/height': mockHeight,
  '/nrv/skills': mockSkills,
  '/nrv/errors': mockErrors,
  '/nrv/vectors': mockVectors,
  '/search': (query) => {
    const results = [];
    
    // Search in skills
    mockSkills.forEach(skill => {
      if (skill.skill_type.toLowerCase().includes(query.toLowerCase()) ||
          skill.capabilities.some(cap => cap.toLowerCase().includes(query.toLowerCase()))) {
        results.push({ type: 'skill', data: skill });
      }
    });
    
    // Search in errors
    mockErrors.forEach(error => {
      if (error.error_type.toLowerCase().includes(query.toLowerCase()) ||
          error.description.toLowerCase().includes(query.toLowerCase())) {
        results.push({ type: 'error', data: error });
      }
    });
    
    return results;
  }
};

// Mock KNIRV GraphChain API for testing
class MockGraphChainAPI {
  constructor() {
    this.baseUrl = '';
    console.log('Using Mock KNIRV GraphChain API for testing');
  }

  async request(endpoint, options = {}) {
    // Simulate network delay
    await new Promise(resolve => setTimeout(resolve, 100 + Math.random() * 200));
    
    // Parse query parameters
    const url = new URL(endpoint, 'http://localhost');
    const path = url.pathname;
    const query = url.searchParams.get('q');
    
    console.log(`Mock API request: ${path}`, query ? `query: ${query}` : '');
    
    // Return mock data
    if (path === '/search' && query) {
      return mockApiResponses[path](query);
    }
    
    if (mockApiResponses[path]) {
      return mockApiResponses[path];
    }
    
    // Default response
    return { message: `Mock endpoint ${path} not implemented` };
  }

  // Implement all the same methods as the real API
  async getHeight() {
    const response = await this.request('/height');
    return response.height;
  }

  async getSkills() {
    return await this.request('/nrv/skills');
  }

  async getErrors() {
    return await this.request('/nrv/errors');
  }

  async getVectors() {
    return await this.request('/nrv/vectors');
  }

  async getSkillsForError(errorType) {
    // Return skills that might be related to this error type
    return mockSkills.filter(skill => 
      skill.capabilities.some(cap => 
        cap.toLowerCase().includes(errorType.toLowerCase().split(' ')[0])
      )
    );
  }

  async searchNodes(query) {
    return await this.request(`/search?q=${encodeURIComponent(query)}`);
  }

  async getGraphChainStats() {
    const skills = await this.getSkills();
    const errors = await this.getErrors();
    const vectors = await this.getVectors();
    
    const skillsWithPerformance = skills.filter(skill => skill.performance);
    const avgResolutionTime = skillsWithPerformance.length > 0
      ? skillsWithPerformance.reduce((sum, skill) => sum + skill.performance.avg_resolution_time, 0) / skillsWithPerformance.length
      : 0;

    return {
      height: mockHeight.height,
      totalNodes: skills.length + errors.length,
      totalEdges: Math.floor((skills.length + errors.length) * 1.5),
      totalSkillNodes: skills.length,
      totalErrorNodes: errors.length,
      totalVectors: vectors.length,
      avgResolutionTime
    };
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

  // Mock methods for other API calls
  clearCache() {
    console.log('Mock API: Cache cleared');
  }

  clearExpiredCache() {
    console.log('Mock API: Expired cache cleared');
  }
}

// Mock SSE Client for testing
class MockGraphChainSSEClient {
  constructor() {
    this.eventListeners = new Map();
    this.isConnected = false;
    this.mockEventInterval = null;
    console.log('Using Mock GraphChain SSE Client for testing');
  }

  connect() {
    console.log('Mock SSE: Connecting...');
    this.isConnected = true;
    
    // Simulate connection
    setTimeout(() => {
      this.emit('connection:open');
      this.emit('connected', {
        type: 'connected',
        timestamp: Date.now(),
        message: 'Mock SSE connection established'
      });
      
      // Start sending mock events
      this.startMockEvents();
    }, 500);
  }

  startMockEvents() {
    // Send periodic mock events
    this.mockEventInterval = setInterval(() => {
      // Random height update
      if (Math.random() < 0.3) {
        mockHeight.height++;
        this.emit('height_changed', mockHeight.height);
      }
      
      // Random stats update
      if (Math.random() < 0.2) {
        this.emit('stats_update', {
          stats: {
            height: mockHeight.height,
            totalSkillNodes: mockSkills.length,
            totalErrorNodes: mockErrors.length,
            avgResolutionTime: 2.5
          }
        });
      }
      
      // Heartbeat
      this.emit('heartbeat', { timestamp: Date.now() });
    }, 5000);
  }

  disconnect() {
    console.log('Mock SSE: Disconnecting...');
    this.isConnected = false;
    if (this.mockEventInterval) {
      clearInterval(this.mockEventInterval);
      this.mockEventInterval = null;
    }
    this.emit('connection:close');
  }

  on(event, callback) {
    if (!this.eventListeners.has(event)) {
      this.eventListeners.set(event, []);
    }
    this.eventListeners.get(event).push(callback);
  }

  off(event, callback) {
    if (!this.eventListeners.has(event)) return;
    
    const listeners = this.eventListeners.get(event);
    const index = listeners.indexOf(callback);
    if (index > -1) {
      listeners.splice(index, 1);
    }
  }

  emit(event, data = null) {
    if (!this.eventListeners.has(event)) return;
    
    const listeners = this.eventListeners.get(event);
    listeners.forEach(callback => {
      try {
        callback(data);
      } catch (error) {
        console.error(`Error in mock SSE event listener for ${event}:`, error);
      }
    });
  }

  getStatus() {
    return {
      isConnected: this.isConnected,
      isConnecting: false,
      reconnectAttempts: 0,
      lastEventTime: Date.now()
    };
  }

  reconnect() {
    this.disconnect();
    this.connect();
  }
}

// Enable mock mode for development
if (window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1') {
  console.log('Development mode detected - using mock data');
  
  // Replace global instances with mock versions
  window.graphChainAPI = new MockGraphChainAPI();
  window.graphChainSSE = new MockGraphChainSSEClient();
  
  // Auto-connect mock SSE
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => {
      window.graphChainSSE.connect();
    });
  } else {
    window.graphChainSSE.connect();
  }
}

// Export for module usage
if (typeof module !== 'undefined' && module.exports) {
  module.exports = { MockGraphChainAPI, MockGraphChainSSEClient, mockSkills, mockErrors };
}
