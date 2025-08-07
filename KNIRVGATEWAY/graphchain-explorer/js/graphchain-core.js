/**
 * KNIRV GraphChain Explorer - Core JavaScript Architecture
 * Vanilla JavaScript implementation with ES6 modules
 */

// Core Application Class
class GraphChainExplorer {
  constructor() {
    this.state = new Map();
    this.eventListeners = new Map();
    this.components = new Map();
    this.router = new SimpleRouter();
    this.initialized = false;
  }

  // Initialize the application
  async init() {
    if (this.initialized) return;
    
    console.log('Initializing KNIRV GraphChain Explorer...');
    
    try {
      // Initialize router
      this.router.init();
      
      // Initialize global event listeners
      this.initEventListeners();
      
      // Initialize components
      await this.initComponents();
      
      // Mark as initialized
      this.initialized = true;
      
      console.log('KNIRV GraphChain Explorer initialized successfully');
      
      // Emit initialization complete event
      this.emit('app:initialized');
      
    } catch (error) {
      console.error('Failed to initialize KNIRV GraphChain Explorer:', error);
      this.handleError(error);
    }
  }

  // Initialize global event listeners
  initEventListeners() {
    // Global search functionality
    const globalSearch = document.getElementById('global-search');
    const searchButton = document.getElementById('search-button');
    
    if (globalSearch) {
      globalSearch.addEventListener('keypress', (e) => {
        if (e.key === 'Enter') {
          this.handleGlobalSearch(globalSearch.value);
        }
      });
    }
    
    if (searchButton) {
      searchButton.addEventListener('click', () => {
        if (globalSearch) {
          this.handleGlobalSearch(globalSearch.value);
        }
      });
    }

    // Mobile menu toggle
    const menuToggle = document.querySelector('[data-menu-toggle]');
    if (menuToggle) {
      menuToggle.addEventListener('click', (e) => {
        e.preventDefault();
        this.toggleMobileMenu();
      });
    }

    // Window resize handler
    window.addEventListener('resize', this.debounce(() => {
      this.emit('window:resize', {
        width: window.innerWidth,
        height: window.innerHeight
      });
    }, 250));

    // Visibility change handler
    document.addEventListener('visibilitychange', () => {
      this.emit('page:visibility', !document.hidden);
    });
  }

  // Initialize components
  async initComponents() {
    // This will be called by page-specific controllers
    console.log('Core components initialized');
  }

  // Handle global search
  handleGlobalSearch(query) {
    if (!query.trim()) return;
    
    console.log('Global search:', query);
    this.router.navigate(`/search?q=${encodeURIComponent(query)}`);
  }

  // Toggle mobile menu
  toggleMobileMenu() {
    const menu = document.getElementById('header-menu');
    const toggle = document.querySelector('[data-menu-toggle]');
    
    if (menu && toggle) {
      const isOpen = menu.classList.contains('menu-open');
      
      if (isOpen) {
        menu.classList.remove('menu-open');
        toggle.classList.remove('navbar-active');
      } else {
        menu.classList.add('menu-open');
        toggle.classList.add('navbar-active');
      }
    }
  }

  // State management
  setState(key, value) {
    const oldValue = this.state.get(key);
    this.state.set(key, value);
    
    // Emit state change event
    this.emit('state:change', { key, value, oldValue });
    this.emit(`state:${key}`, { value, oldValue });
  }

  getState(key) {
    return this.state.get(key);
  }

  // Event system
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
        console.error(`Error in event listener for ${event}:`, error);
      }
    });
  }

  // Component registration
  registerComponent(name, component) {
    this.components.set(name, component);
  }

  getComponent(name) {
    return this.components.get(name);
  }

  // Error handling
  handleError(error) {
    console.error('KNIRV GraphChain Explorer Error:', error);
    
    // Show error state
    this.showErrorState(error.message || 'An unexpected error occurred');
    
    // Emit error event
    this.emit('app:error', error);
  }

  // Show error state
  showErrorState(message) {
    const errorState = document.getElementById('error-state');
    const errorMessage = document.getElementById('error-message');
    const loadingState = document.getElementById('loading-state');
    const mainContent = document.getElementById('main-content');
    
    if (errorState) {
      if (errorMessage) {
        errorMessage.textContent = message;
      }
      errorState.style.display = 'block';
    }
    
    if (loadingState) {
      loadingState.style.display = 'none';
    }
    
    if (mainContent) {
      mainContent.style.display = 'none';
    }
  }

  // Show loading state
  showLoadingState() {
    const errorState = document.getElementById('error-state');
    const loadingState = document.getElementById('loading-state');
    const mainContent = document.getElementById('main-content');
    
    if (loadingState) {
      loadingState.style.display = 'block';
    }
    
    if (errorState) {
      errorState.style.display = 'none';
    }
    
    if (mainContent) {
      mainContent.style.display = 'none';
    }
  }

  // Show main content
  showMainContent() {
    const errorState = document.getElementById('error-state');
    const loadingState = document.getElementById('loading-state');
    const mainContent = document.getElementById('main-content');
    
    if (mainContent) {
      mainContent.style.display = 'block';
    }
    
    if (loadingState) {
      loadingState.style.display = 'none';
    }
    
    if (errorState) {
      errorState.style.display = 'none';
    }
  }

  // Utility methods
  debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
      const later = () => {
        clearTimeout(timeout);
        func(...args);
      };
      clearTimeout(timeout);
      timeout = setTimeout(later, wait);
    };
  }

  throttle(func, limit) {
    let inThrottle;
    return function() {
      const args = arguments;
      const context = this;
      if (!inThrottle) {
        func.apply(context, args);
        inThrottle = true;
        setTimeout(() => inThrottle = false, limit);
      }
    };
  }

  formatTime(timestamp) {
    return new Date(timestamp).toLocaleString();
  }

  formatNumber(num) {
    return num.toLocaleString();
  }

  formatBytes(bytes, decimals = 2) {
    if (bytes === 0) return '0 Bytes';
    
    const k = 1024;
    const dm = decimals < 0 ? 0 : decimals;
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
    
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    
    return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
  }
}

// Simple Router Class
class SimpleRouter {
  constructor() {
    this.routes = new Map();
    this.currentRoute = null;
  }

  init() {
    // Handle initial route
    this.handleRoute();
    
    // Listen for popstate events
    window.addEventListener('popstate', () => {
      this.handleRoute();
    });
  }

  addRoute(path, handler) {
    this.routes.set(path, handler);
  }

  navigate(path) {
    window.history.pushState({}, '', path);
    this.handleRoute();
  }

  handleRoute() {
    const path = window.location.pathname;
    const search = window.location.search;
    
    this.currentRoute = { path, search };
    
    // Emit route change event
    if (window.graphChainApp) {
      window.graphChainApp.emit('route:change', this.currentRoute);
    }
  }

  getCurrentRoute() {
    return this.currentRoute;
  }

  getQueryParams() {
    const params = new URLSearchParams(window.location.search);
    const result = {};
    for (const [key, value] of params) {
      result[key] = value;
    }
    return result;
  }
}

// Global application instance
window.graphChainApp = new GraphChainExplorer();

// Initialize when DOM is ready
if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', () => {
    window.graphChainApp.init();
  });
} else {
  window.graphChainApp.init();
}

// Export for module usage
if (typeof module !== 'undefined' && module.exports) {
  module.exports = { GraphChainExplorer, SimpleRouter };
}
