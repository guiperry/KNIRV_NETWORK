/**
 * KNIRV Gateway SSE Client
 * Replaces WebSocket functionality with Server-Sent Events
 * Provides real-time updates for health monitoring and metrics
 */

class KNIRVGatewayClient {
  constructor(baseUrl = '') {
    this.baseUrl = baseUrl;
    this.eventSources = new Map();
    this.listeners = new Map();
    this.reconnectAttempts = 0;
    this.maxReconnectAttempts = 5;
    this.reconnectDelay = 1000; // Start with 1 second
  }

  /**
   * Connect to health monitoring SSE stream
   */
  connectHealthMonitor() {
    const url = `${this.baseUrl}/health-monitor/events`;
    return this.createEventSource('health', url, {
      onMessage: (event) => this.handleHealthUpdate(event),
      onError: (error) => this.handleHealthError(error)
    });
  }

  /**
   * Connect to gateway events SSE stream
   */
  connectGatewayEvents() {
    const url = `${this.baseUrl}/gateway/events`;
    return this.createEventSource('gateway', url, {
      onMessage: (event) => this.handleGatewayEvent(event),
      onError: (error) => this.handleGatewayError(error)
    });
  }

  /**
   * Create and manage an EventSource connection
   */
  createEventSource(name, url, options = {}) {
    // Close existing connection if any
    if (this.eventSources.has(name)) {
      this.eventSources.get(name).close();
    }

    const eventSource = new EventSource(url);
    this.eventSources.set(name, eventSource);

    eventSource.onopen = () => {
      console.log(`SSE connection opened: ${name}`);
      this.reconnectAttempts = 0;
      this.reconnectDelay = 1000;
      this.emit('connected', { source: name });
    };

    eventSource.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        if (options.onMessage) {
          options.onMessage(data);
        }
        this.emit('message', { source: name, data });
      } catch (error) {
        console.error(`Error parsing SSE message from ${name}:`, error);
      }
    };

    eventSource.onerror = (error) => {
      console.error(`SSE error on ${name}:`, error);
      if (options.onError) {
        options.onError(error);
      }
      this.handleReconnect(name, url, options);
    };

    // Handle specific event types
    eventSource.addEventListener('health_change', (event) => {
      const data = JSON.parse(event.data);
      this.emit('health_change', data);
    });

    eventSource.addEventListener('metrics', (event) => {
      const data = JSON.parse(event.data);
      this.emit('metrics', data);
    });

    eventSource.addEventListener('ping', (event) => {
      const data = JSON.parse(event.data);
      this.emit('ping', data);
    });

    return eventSource;
  }

  /**
   * Handle reconnection logic
   */
  handleReconnect(name, url, options) {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error(`Max reconnection attempts reached for ${name}`);
      this.emit('max_reconnect_attempts', { source: name });
      return;
    }

    this.reconnectAttempts++;
    this.reconnectDelay = Math.min(this.reconnectDelay * 2, 30000); // Max 30 seconds

    console.log(`Attempting to reconnect ${name} in ${this.reconnectDelay}ms (attempt ${this.reconnectAttempts})`);

    setTimeout(() => {
      this.createEventSource(name, url, options);
    }, this.reconnectDelay);
  }

  /**
   * Handle health update events
   */
  handleHealthUpdate(data) {
    console.log('Health update received:', data);
    
    if (data.type === 'health_update') {
      this.updateHealthDisplay(data.services);
    } else if (data.type === 'health_change') {
      this.updateServiceHealth(data.service, data.healthy);
    }
  }

  /**
   * Handle gateway events
   */
  handleGatewayEvent(data) {
    console.log('Gateway event received:', data);
    
    switch (data.type) {
      case 'connected':
        this.emit('gateway_connected', data);
        break;
      case 'metrics':
        this.updateMetricsDisplay(data.data);
        break;
      default:
        console.log('Unknown gateway event:', data);
    }
  }

  /**
   * Update health display in UI
   */
  updateHealthDisplay(services) {
    Object.entries(services).forEach(([serviceName, serviceData]) => {
      const element = document.getElementById(`health-${serviceName}`);
      if (element) {
        element.className = serviceData.healthy ? 'service-healthy' : 'service-unhealthy';
        element.textContent = serviceData.healthy ? 'Healthy' : 'Unhealthy';
        
        // Update response time if available
        const responseTimeElement = document.getElementById(`response-time-${serviceName}`);
        if (responseTimeElement && serviceData.responseTime) {
          responseTimeElement.textContent = `${serviceData.responseTime}ms`;
        }
      }
    });
  }

  /**
   * Update individual service health
   */
  updateServiceHealth(serviceName, isHealthy) {
    const element = document.getElementById(`health-${serviceName}`);
    if (element) {
      element.className = isHealthy ? 'service-healthy' : 'service-unhealthy';
      element.textContent = isHealthy ? 'Healthy' : 'Unhealthy';
    }

    // Show notification
    this.showNotification(`Service ${serviceName} is now ${isHealthy ? 'healthy' : 'unhealthy'}`);
  }

  /**
   * Update metrics display
   */
  updateMetricsDisplay(metrics) {
    const metricsElement = document.getElementById('gateway-metrics');
    if (metricsElement) {
      metricsElement.innerHTML = `
        <div>Total Requests: ${metrics.totalRequests || 0}</div>
        <div>Services: ${metrics.services || 0}</div>
        <div>Last Update: ${new Date().toLocaleTimeString()}</div>
      `;
    }
  }

  /**
   * Show notification to user
   */
  showNotification(message) {
    // Simple notification - could be enhanced with a proper notification system
    console.log('Notification:', message);
    
    // If you have a notification area in your UI
    const notificationArea = document.getElementById('notifications');
    if (notificationArea) {
      const notification = document.createElement('div');
      notification.className = 'notification';
      notification.textContent = message;
      notificationArea.appendChild(notification);
      
      // Remove after 5 seconds
      setTimeout(() => {
        notification.remove();
      }, 5000);
    }
  }

  /**
   * Event listener management
   */
  on(event, callback) {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, []);
    }
    this.listeners.get(event).push(callback);
  }

  emit(event, data) {
    if (this.listeners.has(event)) {
      this.listeners.get(event).forEach(callback => callback(data));
    }
  }

  /**
   * Make HTTP requests to gateway
   */
  async request(path, options = {}) {
    const url = `${this.baseUrl}${path}`;
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

    return response.json();
  }

  /**
   * Get current health status
   */
  async getHealthStatus() {
    return this.request('/health-monitor/status');
  }

  /**
   * Get gateway metrics
   */
  async getGatewayMetrics() {
    return this.request('/gateway/metrics');
  }

  /**
   * Authenticate with the gateway
   */
  async login(username, password) {
    return this.request('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password })
    });
  }

  /**
   * Close all connections
   */
  disconnect() {
    this.eventSources.forEach((eventSource, name) => {
      console.log(`Closing SSE connection: ${name}`);
      eventSource.close();
    });
    this.eventSources.clear();
  }

  /**
   * Handle errors
   */
  handleHealthError(error) {
    console.error('Health monitor error:', error);
    this.emit('health_error', error);
  }

  handleGatewayError(error) {
    console.error('Gateway error:', error);
    this.emit('gateway_error', error);
  }
}

// Export for use in other scripts
window.KNIRVGatewayClient = KNIRVGatewayClient;

// Example usage:
/*
const gateway = new KNIRVGatewayClient();

// Connect to health monitoring
gateway.connectHealthMonitor();

// Listen for events
gateway.on('health_change', (data) => {
  console.log('Service health changed:', data);
});

gateway.on('connected', (data) => {
  console.log('Connected to:', data.source);
});

// Get initial status
gateway.getHealthStatus().then(status => {
  console.log('Initial health status:', status);
});
*/
