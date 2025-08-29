/**
 * Agent Terminal Service
 * 
 * This service provides methods for interacting with agent-specific terminal sessions.
 * It connects each agent plugin to its corresponding mini-terminal on the frontend.
 */

// Base API endpoints
const TERMINAL_API_BASE = '/api/v1/terminal';
const AGENT_API_BASE = '/api/v1/adk/agents';

/**
 * Get the base URL based on environment
 * @returns {string} The base URL
 */
const getBaseUrl = () => {
  return window.location.protocol === 'file:' ? 'http://localhost:8081' : '';
};

/**
 * Activate an agent for a specific session
 * 
 * @param {Object} options - Activation options
 * @param {string} options.agentId - Agent ID
 * @param {string} options.version - Agent version
 * @param {string} options.sessionId - Session ID
 * @param {Object} options.config - Agent configuration
 * @returns {Promise<Object>} - Activation result
 */
export const activateAgent = async (options) => {
  try {
    const baseUrl = getBaseUrl();
    const response = await fetch(`${baseUrl}${AGENT_API_BASE}/activate`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer test-api-key' // TODO: Use proper auth
      },
      body: JSON.stringify({
        agentId: options.agentId,
        version: options.version || '1.0',
        sessionId: options.sessionId,
        config: options.config || {}
      })
    });

    if (!response.ok) {
      const errorText = await response.text();
      throw new Error(`Failed to activate agent: ${response.statusText} - ${errorText}`);
    }

    return await response.json();
  } catch (error) {
    console.error('Error activating agent:', error);
    throw error;
  }
};

/**
 * Create a terminal session for an agent
 * 
 * @param {Object} options - Terminal options
 * @param {string} options.sessionId - Session ID
 * @param {string} options.agentId - Agent ID
 * @param {number} options.rows - Terminal rows
 * @param {number} options.cols - Terminal columns
 * @returns {Promise<Object>} - Terminal session information
 */
export const createAgentTerminal = async (options) => {
  try {
    const baseUrl = getBaseUrl();
    const response = await fetch(`${baseUrl}${TERMINAL_API_BASE}/create`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer test-api-key' // TODO: Use proper auth
      },
      body: JSON.stringify({
        sessionId: options.sessionId,
        agentId: options.agentId,
        rows: options.rows || 24,
        cols: options.cols || 80
      })
    });

    if (!response.ok) {
      const errorText = await response.text();
      throw new Error(`Failed to create terminal: ${response.statusText} - ${errorText}`);
    }

    return await response.json();
  } catch (error) {
    console.error('Error creating agent terminal:', error);
    throw error;
  }
};

/**
 * Resize a terminal session
 * 
 * @param {Object} options - Resize options
 * @param {string} options.sessionId - Session ID
 * @param {string} options.terminalId - Terminal ID
 * @param {number} options.rows - New rows
 * @param {number} options.cols - New columns
 * @returns {Promise<Object>} - Resize result
 */
export const resizeAgentTerminal = async (options) => {
  try {
    const baseUrl = getBaseUrl();
    const response = await fetch(`${baseUrl}${TERMINAL_API_BASE}/resize`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer test-api-key' // TODO: Use proper auth
      },
      body: JSON.stringify({
        sessionId: options.sessionId,
        terminalId: options.terminalId,
        rows: options.rows,
        cols: options.cols
      })
    });

    if (!response.ok) {
      const errorText = await response.text();
      throw new Error(`Failed to resize terminal: ${response.statusText} - ${errorText}`);
    }

    return await response.json();
  } catch (error) {
    console.error('Error resizing agent terminal:', error);
    throw error;
  }
};

/**
 * Close a terminal session
 * 
 * @param {Object} options - Close options
 * @param {string} options.sessionId - Session ID
 * @param {string} options.terminalId - Terminal ID
 * @returns {Promise<Object>} - Close result
 */
export const closeAgentTerminal = async (options) => {
  try {
    const baseUrl = getBaseUrl();
    const response = await fetch(`${baseUrl}${TERMINAL_API_BASE}/close`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer test-api-key' // TODO: Use proper auth
      },
      body: JSON.stringify({
        sessionId: options.sessionId,
        terminalId: options.terminalId
      })
    });

    if (!response.ok) {
      const errorText = await response.text();
      throw new Error(`Failed to close terminal: ${response.statusText} - ${errorText}`);
    }

    return await response.json();
  } catch (error) {
    console.error('Error closing agent terminal:', error);
    throw error;
  }
};

/**
 * Get terminal logs
 * 
 * @param {Object} options - Log options
 * @param {string} options.sessionId - Session ID
 * @param {string} options.terminalId - Terminal ID
 * @param {number} options.limit - Maximum number of logs to return
 * @returns {Promise<Object>} - Terminal logs
 */
export const getAgentTerminalLogs = async (options) => {
  try {
    const baseUrl = getBaseUrl();
    const url = new URL(`${baseUrl}${TERMINAL_API_BASE}/logs`);
    url.searchParams.append('sessionId', options.sessionId);
    url.searchParams.append('terminalId', options.terminalId);
    
    if (options.limit) {
      url.searchParams.append('limit', options.limit);
    }

    const response = await fetch(url.toString(), {
      headers: {
        'Authorization': 'Bearer test-api-key' // TODO: Use proper auth
      }
    });

    if (!response.ok) {
      const errorText = await response.text();
      throw new Error(`Failed to get terminal logs: ${response.statusText} - ${errorText}`);
    }

    return await response.json();
  } catch (error) {
    console.error('Error getting agent terminal logs:', error);
    throw error;
  }
};

/**
 * Get WebSocket URL for a terminal session
 * 
 * @param {Object} options - WebSocket options
 * @param {string} options.sessionId - Session ID
 * @param {string} options.terminalId - Terminal ID
 * @returns {string} - WebSocket URL
 */
export const getAgentTerminalWebSocketUrl = (options) => {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const host = window.location.protocol === 'file:' ? 'localhost:8081' : window.location.host;
  return `${protocol}//${host}${TERMINAL_API_BASE}/ws?sessionId=${options.sessionId}&terminalId=${options.terminalId}`;
};

/**
 * Create a WebSocket connection for a terminal session
 * 
 * @param {Object} options - WebSocket options
 * @param {string} options.sessionId - Session ID
 * @param {string} options.terminalId - Terminal ID
 * @param {Object} options.handlers - Event handlers
 * @returns {WebSocket} - WebSocket connection
 */
export const createAgentTerminalWebSocket = (options) => {
  const wsUrl = getAgentTerminalWebSocketUrl(options);
  const ws = new WebSocket(wsUrl);

  if (options.handlers?.onOpen) {
    ws.onopen = options.handlers.onOpen;
  }

  if (options.handlers?.onMessage) {
    ws.onmessage = options.handlers.onMessage;
  }

  if (options.handlers?.onClose) {
    ws.onclose = options.handlers.onClose;
  }

  if (options.handlers?.onError) {
    ws.onerror = options.handlers.onError;
  }

  return ws;
};

/**
 * Agent terminal service object
 */
const agentTerminalService = {
  activateAgent,
  createAgentTerminal,
  resizeAgentTerminal,
  closeAgentTerminal,
  getAgentTerminalLogs,
  getAgentTerminalWebSocketUrl,
  createAgentTerminalWebSocket
};

export default agentTerminalService;