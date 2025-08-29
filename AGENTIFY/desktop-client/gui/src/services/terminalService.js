/**
 * Terminal Service
 * 
 * This service provides methods for interacting with the terminal management system API.
 */

const API_BASE_URL = '/api/terminal';

/**
 * Create a new terminal session
 * 
 * @param {Object} options - Session options
 * @param {string} options.workingDir - Working directory for the terminal
 * @param {Object} options.env - Environment variables as key-value pairs
 * @returns {Promise<Object>} - Session information
 */
export const createTerminalSession = async (options = {}) => {
  try {
    const response = await fetch(`${API_BASE_URL}/create`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        workingDir: options.workingDir || '',
        env: options.env || {},
      }),
    });

    if (!response.ok) {
      const errorData = await response.json();
      throw new Error(errorData.error || 'Failed to create terminal session');
    }

    return await response.json();
  } catch (error) {
    console.error('Error creating terminal session:', error);
    throw error;
  }
};

/**
 * Get information about a terminal session
 * 
 * @param {string} sessionId - Terminal session ID
 * @returns {Promise<Object>} - Session information
 */
export const getTerminalSession = async (sessionId) => {
  try {
    const response = await fetch(`${API_BASE_URL}/${sessionId}`);

    if (!response.ok) {
      const errorData = await response.json();
      throw new Error(errorData.error || 'Failed to get terminal session');
    }

    return await response.json();
  } catch (error) {
    console.error('Error getting terminal session:', error);
    throw error;
  }
};

/**
 * List all terminal sessions for the current user
 * 
 * @returns {Promise<Array>} - List of terminal sessions
 */
export const listTerminalSessions = async () => {
  try {
    const response = await fetch(`${API_BASE_URL}/list`);

    if (!response.ok) {
      const errorData = await response.json();
      throw new Error(errorData.error || 'Failed to list terminal sessions');
    }

    return await response.json();
  } catch (error) {
    console.error('Error listing terminal sessions:', error);
    throw error;
  }
};

/**
 * Close a terminal session
 * 
 * @param {string} sessionId - Terminal session ID
 * @returns {Promise<Object>} - Result of the operation
 */
export const closeTerminalSession = async (sessionId) => {
  try {
    const response = await fetch(`${API_BASE_URL}/${sessionId}/close`, {
      method: 'POST',
    });

    if (!response.ok) {
      const errorData = await response.json();
      throw new Error(errorData.error || 'Failed to close terminal session');
    }

    return await response.json();
  } catch (error) {
    console.error('Error closing terminal session:', error);
    throw error;
  }
};

/**
 * Get command history for a terminal session
 * 
 * @param {string} sessionId - Terminal session ID
 * @returns {Promise<Array>} - Command history
 */
export const getTerminalHistory = async (sessionId) => {
  try {
    const response = await fetch(`${API_BASE_URL}/${sessionId}/history`);

    if (!response.ok) {
      const errorData = await response.json();
      throw new Error(errorData.error || 'Failed to get terminal history');
    }

    return await response.json();
  } catch (error) {
    console.error('Error getting terminal history:', error);
    throw error;
  }
};

/**
 * Execute a command in a terminal session
 * 
 * @param {string} sessionId - Terminal session ID
 * @param {string} command - Command to execute
 * @returns {Promise<Object>} - Result of the operation
 */
export const executeTerminalCommand = async (sessionId, command) => {
  try {
    const response = await fetch(`${API_BASE_URL}/${sessionId}/execute`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        command,
      }),
    });

    if (!response.ok) {
      const errorData = await response.json();
      throw new Error(errorData.error || 'Failed to execute command');
    }

    return await response.json();
  } catch (error) {
    console.error('Error executing command:', error);
    throw error;
  }
};

/**
 * Get WebSocket URL for a terminal session
 * 
 * @param {string} sessionId - Terminal session ID
 * @returns {string} - WebSocket URL
 */
export const getTerminalWebSocketUrl = (sessionId) => {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${protocol}//${window.location.host}${API_BASE_URL}/ws/${sessionId}`;
};

/**
 * Create a WebSocket connection for a terminal session
 * 
 * @param {string} sessionId - Terminal session ID
 * @param {Object} handlers - Event handlers
 * @param {Function} handlers.onOpen - Called when connection is opened
 * @param {Function} handlers.onMessage - Called when a message is received
 * @param {Function} handlers.onClose - Called when connection is closed
 * @param {Function} handlers.onError - Called when an error occurs
 * @returns {WebSocket} - WebSocket connection
 */
export const createTerminalWebSocket = (sessionId, handlers = {}) => {
  const wsUrl = getTerminalWebSocketUrl(sessionId);
  const ws = new WebSocket(wsUrl);

  if (handlers.onOpen) {
    ws.onopen = handlers.onOpen;
  }

  if (handlers.onMessage) {
    ws.onmessage = handlers.onMessage;
  }

  if (handlers.onClose) {
    ws.onclose = handlers.onClose;
  }

  if (handlers.onError) {
    ws.onerror = handlers.onError;
  }

  return ws;
};

/**
 * Terminal service object
 */
const terminalService = {
  createTerminalSession,
  getTerminalSession,
  listTerminalSessions,
  closeTerminalSession,
  getTerminalHistory,
  executeTerminalCommand,
  getTerminalWebSocketUrl,
  createTerminalWebSocket,
};

export default terminalService;