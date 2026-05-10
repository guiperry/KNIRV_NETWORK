/**
 * KNIRV GraphChain SSE Client
 * Handles Server-Sent Events for real-time KNIRV GraphChain updates
 */

class GraphChainSSEClient {
  constructor() {
    this.eventSource = null;
    this.eventListeners = new Map();
    this.reconnectAttempts = 0;
    this.maxReconnectAttempts = 5;
    this.reconnectDelay = 1000; // Start with 1 second
    this.maxReconnectDelay = 30000; // Max 30 seconds
    this.isConnected = false;
    this.isConnecting = false;
    this.lastEventTime = null;
    this.heartbeatTimeout = null;
    this.heartbeatInterval = 30000; // 30 seconds
  }

  /**
   * Connect to KNIRV GraphChain SSE endpoint
   */
  connect() {
    if (this.isConnected || this.isConnecting) {
      console.log('GraphChain SSE already connected or connecting');
      return;
    }

    this.isConnecting = true;
    console.log('Connecting to GraphChain SSE...');

    try {
      // Use the graphchain-events function
      const sseUrl = '/api/graph/events';
      
      this.eventSource = new EventSource(sseUrl);
      
      // Connection opened
      this.eventSource.onopen = (event) => {
        console.log('GraphChain SSE connection opened');
        this.isConnected = true;
        this.isConnecting = false;
        this.reconnectAttempts = 0;
        this.reconnectDelay = 1000;
        this.lastEventTime = Date.now();
        this.startHeartbeatMonitor();
        this.emit('connection:open', event);
      };

      // Handle messages
      this.eventSource.onmessage = (event) => {
        this.handleMessage(event);
      };

      // Handle errors
      this.eventSource.onerror = (event) => {
        console.error('GraphChain SSE error:', event);
        this.handleError(event);
      };

      // Handle specific event types
      this.setupEventHandlers();

    } catch (error) {
      console.error('Failed to create GraphChain SSE connection:', error);
      this.isConnecting = false;
      this.handleError(error);
    }
  }

  /**
   * Setup handlers for specific SSE event types
   */
  setupEventHandlers() {
    // Connected event
    this.eventSource.addEventListener('connected', (event) => {
      const data = JSON.parse(event.data);
      console.log('GraphChain SSE connected:', data);
      this.emit('connected', data);
    });

    // Height update event
    this.eventSource.addEventListener('height_update', (event) => {
      const data = JSON.parse(event.data);
      console.log('GraphChain height update:', data.height);
      this.lastEventTime = Date.now();
      this.emit('height_changed', data.height);
      this.emit('height_update', data);
    });

    // Skills data event
    this.eventSource.addEventListener('skills_data', (event) => {
      const data = JSON.parse(event.data);
      console.log('GraphChain skills data:', data);
      this.lastEventTime = Date.now();
      this.emit('skills_data', data);
    });

    // Errors data event
    this.eventSource.addEventListener('errors_data', (event) => {
      const data = JSON.parse(event.data);
      console.log('GraphChain errors data:', data);
      this.lastEventTime = Date.now();
      this.emit('errors_data', data);
    });

    // Stats update event
    this.eventSource.addEventListener('stats_update', (event) => {
      const data = JSON.parse(event.data);
      console.log('GraphChain stats update:', data);
      this.lastEventTime = Date.now();
      this.emit('stats_update', data);
    });

    // Skill added event
    this.eventSource.addEventListener('skill_added', (event) => {
      const data = JSON.parse(event.data);
      console.log('New skill added:', data);
      this.lastEventTime = Date.now();
      this.emit('skill_added', data.skill);
    });

    // Error created event
    this.eventSource.addEventListener('error_created', (event) => {
      const data = JSON.parse(event.data);
      console.log('New error created:', data);
      this.lastEventTime = Date.now();
      this.emit('error_created', data.error);
    });

    // Error resolved event
    this.eventSource.addEventListener('error_resolved', (event) => {
      const data = JSON.parse(event.data);
      console.log('Error resolved:', data);
      this.lastEventTime = Date.now();
      this.emit('error_resolved', data.error);
    });

    // Graph updated event
    this.eventSource.addEventListener('graph_updated', (event) => {
      const data = JSON.parse(event.data);
      console.log('Graph updated:', data);
      this.lastEventTime = Date.now();
      this.emit('graph_updated', data.graph);
    });

    // Heartbeat event
    this.eventSource.addEventListener('heartbeat', (event) => {
      const data = JSON.parse(event.data);
      this.lastEventTime = Date.now();
      this.resetHeartbeatTimeout();
      this.emit('heartbeat', data);
    });

    // Error event
    this.eventSource.addEventListener('error', (event) => {
      const data = JSON.parse(event.data);
      console.error('GraphChain SSE error event:', data);
      this.emit('sse_error', data);
    });
  }

  /**
   * Handle generic messages
   */
  handleMessage(event) {
    try {
      const data = JSON.parse(event.data);
      console.log('GraphChain SSE message:', data);
      this.lastEventTime = Date.now();
      this.emit('message', data);
    } catch (error) {
      console.error('Failed to parse SSE message:', error);
    }
  }

  /**
   * Handle connection errors
   */
  handleError(error) {
    console.error('GraphChain SSE error:', error);
    this.isConnected = false;
    this.isConnecting = false;
    this.stopHeartbeatMonitor();
    
    this.emit('connection:error', error);
    
    // Attempt reconnection
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      this.scheduleReconnect();
    } else {
      console.error('Max reconnection attempts reached');
      this.emit('connection:failed', { attempts: this.reconnectAttempts });
    }
  }

  /**
   * Schedule reconnection attempt
   */
  scheduleReconnect() {
    this.reconnectAttempts++;
    const delay = Math.min(this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1), this.maxReconnectDelay);
    
    console.log(`Scheduling GraphChain SSE reconnection attempt ${this.reconnectAttempts} in ${delay}ms`);
    
    setTimeout(() => {
      if (!this.isConnected && !this.isConnecting) {
        this.connect();
      }
    }, delay);
  }

  /**
   * Start heartbeat monitoring
   */
  startHeartbeatMonitor() {
    this.resetHeartbeatTimeout();
  }

  /**
   * Reset heartbeat timeout
   */
  resetHeartbeatTimeout() {
    if (this.heartbeatTimeout) {
      clearTimeout(this.heartbeatTimeout);
    }
    
    this.heartbeatTimeout = setTimeout(() => {
      console.warn('GraphChain SSE heartbeat timeout');
      this.disconnect();
      this.scheduleReconnect();
    }, this.heartbeatInterval * 2); // Allow 2x heartbeat interval before timeout
  }

  /**
   * Stop heartbeat monitoring
   */
  stopHeartbeatMonitor() {
    if (this.heartbeatTimeout) {
      clearTimeout(this.heartbeatTimeout);
      this.heartbeatTimeout = null;
    }
  }

  /**
   * Disconnect from SSE
   */
  disconnect() {
    console.log('Disconnecting from GraphChain SSE');
    
    if (this.eventSource) {
      this.eventSource.close();
      this.eventSource = null;
    }
    
    this.isConnected = false;
    this.isConnecting = false;
    this.stopHeartbeatMonitor();
    
    this.emit('connection:close');
  }

  /**
   * Add event listener
   */
  on(event, callback) {
    if (!this.eventListeners.has(event)) {
      this.eventListeners.set(event, []);
    }
    this.eventListeners.get(event).push(callback);
  }

  /**
   * Remove event listener
   */
  off(event, callback) {
    if (!this.eventListeners.has(event)) return;
    
    const listeners = this.eventListeners.get(event);
    const index = listeners.indexOf(callback);
    if (index > -1) {
      listeners.splice(index, 1);
    }
  }

  /**
   * Emit event to listeners
   */
  emit(event, data = null) {
    if (!this.eventListeners.has(event)) return;
    
    const listeners = this.eventListeners.get(event);
    listeners.forEach(callback => {
      try {
        callback(data);
      } catch (error) {
        console.error(`Error in GraphChain SSE event listener for ${event}:`, error);
      }
    });
  }

  /**
   * Get connection status
   */
  getStatus() {
    return {
      isConnected: this.isConnected,
      isConnecting: this.isConnecting,
      reconnectAttempts: this.reconnectAttempts,
      lastEventTime: this.lastEventTime
    };
  }

  /**
   * Force reconnection
   */
  reconnect() {
    this.disconnect();
    this.reconnectAttempts = 0;
    this.connect();
  }
}

// Create global SSE client instance
window.graphChainSSE = new GraphChainSSEClient();

// Auto-connect when page loads
if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', () => {
    window.graphChainSSE.connect();
  });
} else {
  window.graphChainSSE.connect();
}

// Handle page visibility changes
document.addEventListener('visibilitychange', () => {
  if (document.hidden) {
    // Page is hidden, disconnect to save resources
    window.graphChainSSE.disconnect();
  } else {
    // Page is visible, reconnect
    window.graphChainSSE.connect();
  }
});

// Handle page unload
window.addEventListener('beforeunload', () => {
  window.graphChainSSE.disconnect();
});

// Export for module usage
if (typeof module !== 'undefined' && module.exports) {
  module.exports = GraphChainSSEClient;
}
