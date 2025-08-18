// API service for making HTTP requests
// This service provides a simple interface for making API calls

// API configuration - detect if running in Electron
const getApiBaseUrl = () => {
  // Check if running in Electron (file:// protocol)
  if (window.location.protocol === 'file:') {
    return 'http://localhost:8081'; // Backend server port in Electron
  }
  return ''; // Use relative URLs for web version
};

// Base API URL
const API_BASE_URL = getApiBaseUrl();

// Helper function to get auth headers
const getAuthHeaders = () => {
  const token = localStorage.getItem('token');
  return {
    'Content-Type': 'application/json',
    ...(token && { 'Authorization': `Bearer ${token}` })
  };
};

// Enhanced fetch function with error handling
const apiRequest = async (url, options = {}) => {
  const requestOptions = {
    ...options,
    headers: {
      ...getAuthHeaders(),
      ...options.headers,
    },
  };

  try {
    const response = await fetch(url, requestOptions);
    
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('API request failed:', error);
    throw error;
  }
};

// API service object
export const api = {
  // GET request
  get: async (endpoint) => {
    const url = `${API_BASE_URL}${endpoint}`;
    return apiRequest(url, { method: 'GET' });
  },

  // POST request
  post: async (endpoint, data) => {
    const url = `${API_BASE_URL}${endpoint}`;
    return apiRequest(url, {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  // PUT request
  put: async (endpoint, data) => {
    const url = `${API_BASE_URL}${endpoint}`;
    return apiRequest(url, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  },

  // DELETE request
  delete: async (endpoint) => {
    const url = `${API_BASE_URL}${endpoint}`;
    return apiRequest(url, { method: 'DELETE' });
  },

  // PATCH request
  patch: async (endpoint, data) => {
    const url = `${API_BASE_URL}${endpoint}`;
    return apiRequest(url, {
      method: 'PATCH',
      body: JSON.stringify(data),
    });
  }
};

// Default export
export default api;
