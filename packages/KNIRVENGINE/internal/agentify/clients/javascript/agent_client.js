// agent_client.js

/**
 * Represents a message in the conversation history.
 */
class ConversationMessage {
  /**
   * Initialize a conversation message.
   * @param {string} role - The role of the message sender (user, assistant, system)
   * @param {string} content - The content of the message
   * @param {number} [timestamp] - The timestamp of the message (defaults to current time)
   */
  constructor(role, content, timestamp = null) {
    this.role = role;
    this.content = content;
    this.timestamp = timestamp || Math.floor(Date.now() / 1000);
  }

  /**
   * Convert the message to a plain object.
   * @returns {Object} The message as a plain object
   */
  toObject() {
    return {
      role: this.role,
      content: this.content,
      timestamp: this.timestamp
    };
  }

  /**
   * Create a message from a plain object.
   * @param {Object} data - The message data
   * @returns {ConversationMessage} The message
   */
  static fromObject(data) {
    return new ConversationMessage(
      data.role,
      data.content,
      data.timestamp
    );
  }
}

/**
 * Represents a call to a tool during inference.
 */
class ToolCall {
  /**
   * Initialize a tool call.
   * @param {string} name - The name of the tool
   * @param {Object} input - The input to the tool
   * @param {*} output - The output from the tool
   * @param {number} [timestamp] - The timestamp of the tool call (defaults to current time)
   */
  constructor(name, input, output, timestamp = null) {
    this.name = name;
    this.input = input;
    this.output = output;
    this.timestamp = timestamp || Math.floor(Date.now() / 1000);
  }

  /**
   * Convert the tool call to a plain object.
   * @returns {Object} The tool call as a plain object
   */
  toObject() {
    return {
      name: this.name,
      input: this.input,
      output: this.output,
      timestamp: this.timestamp
    };
  }

  /**
   * Create a tool call from a plain object.
   * @param {Object} data - The tool call data
   * @returns {ToolCall} The tool call
   */
  static fromObject(data) {
    return new ToolCall(
      data.name,
      data.input,
      data.output,
      data.timestamp
    );
  }
}

/**
 * Represents a response from the agent.
 */
class InferenceResponse {
  /**
   * Initialize an inference response.
   * @param {string} output - The output text from the agent
   * @param {ToolCall[]} [toolCalls] - The tool calls made during inference
   * @param {string} [reasoning] - The reasoning trace (if enabled)
   * @param {Object} [metadata] - Additional metadata about the response
   */
  constructor(output, toolCalls = null, reasoning = null, metadata = null) {
    this.output = output;
    this.toolCalls = toolCalls || [];
    this.reasoning = reasoning;
    this.metadata = metadata || {};
  }

  /**
   * Create an inference response from a plain object.
   * @param {Object} data - The response data
   * @returns {InferenceResponse} The inference response
   */
  static fromObject(data) {
    const toolCalls = data.toolCalls ? data.toolCalls.map(tc => ToolCall.fromObject(tc)) : null;
    
    return new InferenceResponse(
      data.output,
      toolCalls,
      data.reasoning,
      data.metadata || {}
    );
  }
}

/**
 * Represents the capabilities of an agent.
 */
class AgentCapabilities {
  /**
   * Initialize agent capabilities.
   * @param {boolean} supportsStreaming - Whether the agent supports streaming responses
   * @param {boolean} supportsToolCalls - Whether the agent supports tool calls
   * @param {boolean} supportsReasoning - Whether the agent supports reasoning traces
   * @param {number} maxContextLength - The maximum context length supported by the agent
   * @param {string[]} supportedParameters - The supported inference parameters
   */
  constructor(supportsStreaming, supportsToolCalls, supportsReasoning, maxContextLength, supportedParameters) {
    this.supportsStreaming = supportsStreaming;
    this.supportsToolCalls = supportsToolCalls;
    this.supportsReasoning = supportsReasoning;
    this.maxContextLength = maxContextLength;
    this.supportedParameters = supportedParameters;
  }

  /**
   * Create agent capabilities from a plain object.
   * @param {Object} data - The capabilities data
   * @returns {AgentCapabilities} The agent capabilities
   */
  static fromObject(data) {
    return new AgentCapabilities(
      data.supportsStreaming,
      data.supportsToolCalls,
      data.supportsReasoning,
      data.maxContextLength,
      data.supportedParameters
    );
  }
}

/**
 * Represents a terminal session with an agent.
 */
class TerminalSession {
  /**
   * Initialize a terminal session.
   * @param {string} terminalId - The ID of the terminal session
   * @param {string} sessionId - The agent session ID
   * @param {number} rows - The number of rows in the terminal
   * @param {number} cols - The number of columns in the terminal
   * @param {AgentClient} client - The agent client instance
   */
  constructor(terminalId, sessionId, rows, cols, client) {
    this.terminalId = terminalId;
    this.sessionId = sessionId;
    this.rows = rows;
    this.cols = cols;
    this.client = client;
    this.ws = null;
    this.outputCallback = null;
    this.isConnected = false;
  }

  /**
   * Write data to the terminal.
   * @param {string} data - The data to write
   * @returns {Promise<Object>} The response from the server
   */
  async write(data) {
    return this.client.writeToTerminal(this.sessionId, this.terminalId, data);
  }

  /**
   * Read data from the terminal.
   * @returns {Promise<string>} The terminal output
   */
  async read() {
    return this.client.readFromTerminal(this.sessionId, this.terminalId);
  }

  /**
   * Resize the terminal.
   * @param {number} rows - The new number of rows
   * @param {number} cols - The new number of columns
   * @returns {Promise<Object>} The response from the server
   */
  async resize(rows, cols) {
    this.rows = rows;
    this.cols = cols;
    return this.client.resizeTerminal(this.sessionId, this.terminalId, rows, cols);
  }

  /**
   * Connect to the terminal via WebSocket for real-time communication.
   * @param {Function} [outputCallback] - Callback function to handle terminal output
   */
  connectWebSocket(outputCallback = null) {
    this.outputCallback = outputCallback;
    const wsUrl = `${this.client.baseUrl.replace('http', 'ws')}/v1/terminal/ws?sessionId=${this.sessionId}&terminalId=${this.terminalId}`;

    this.ws = new WebSocket(wsUrl, [], {
      headers: {
        'Authorization': `Bearer ${this.client.apiKey}`
      }
    });

    this.ws.onopen = () => {
      this.isConnected = true;
    };

    this.ws.onmessage = (event) => {
      if (this.outputCallback) {
        this.outputCallback(event.data);
      }
    };

    this.ws.onclose = () => {
      this.isConnected = false;
    };

    this.ws.onerror = (error) => {
      console.error('WebSocket error:', error);
      this.isConnected = false;
    };
  }

  /**
   * Send data via WebSocket.
   * @param {string} data - The data to send
   */
  sendWebSocket(data) {
    if (this.ws && this.isConnected) {
      this.ws.send(data);
    }
  }

  /**
   * Close the terminal session.
   * @returns {Promise<Object>} The response from the server
   */
  async close() {
    if (this.ws) {
      this.ws.close();
    }
    return this.client.closeTerminal(this.sessionId, this.terminalId);
  }
}

/**
 * Client for interacting with the Agent Inferencer API.
 */
class AgentClient {
  /**
   * Initialize the agent client.
   * @param {string} baseUrl - The base URL of the Agent Inferencer API
   * @param {string} apiKey - The API key for authentication
   */
  constructor(baseUrl, apiKey) {
    this.baseUrl = baseUrl.endsWith('/') ? baseUrl.slice(0, -1) : baseUrl;
    this.apiKey = apiKey;
    this.headers = {
      'Authorization': `Bearer ${apiKey}`,
      'Content-Type': 'application/json'
    };
  }

  /**
   * Make a request to the API.
   * @param {string} method - The HTTP method
   * @param {string} path - The API path
   * @param {Object} [data] - The request data
   * @returns {Promise<Object>} The response data
   * @private
   */
  async _request(method, path, data = null) {
    const url = `${this.baseUrl}${path}`;
    const options = {
      method,
      headers: this.headers,
      body: data ? JSON.stringify(data) : undefined
    };

    const response = await fetch(url, options);
    
    if (!response.ok) {
      const errorText = await response.text();
      throw new Error(`API request failed: ${response.status} ${response.statusText} - ${errorText}`);
    }
    
    return response.json();
  }

  /**
   * List available agents.
   * @returns {Promise<string[]>} A list of available agent IDs
   */
  async listAgents() {
    const response = await this._request('GET', '/v1/agents');
    return response.agents;
  }

  /**
   * Activate an agent for a session.
   * @param {string} agentId - The ID of the agent to activate
   * @param {string} version - The version of the agent to activate
   * @param {string} sessionId - The session ID to use
   * @param {Object} [config] - Additional configuration for the agent
   * @returns {Promise<Object>} The activation response
   */
  async activateAgent(agentId, version, sessionId, config = null) {
    const data = {
      agentId,
      version,
      sessionId
    };
    
    if (config) {
      data.config = config;
    }
    
    return this._request('POST', '/v1/agents/activate', data);
  }

  /**
   * Deactivate an agent for a session.
   * @param {string} sessionId - The session ID to deactivate
   * @returns {Promise<Object>} The deactivation response
   */
  async deactivateAgent(sessionId) {
    const data = {
      sessionId
    };
    
    return this._request('POST', '/v1/agents/deactivate', data);
  }

  /**
   * Process an inference request.
   * @param {string} sessionId - The session ID to use
   * @param {string} inputText - The input text for the agent
   * @param {ConversationMessage[]} [history] - The conversation history
   * @param {Object} [parameters] - Additional parameters for the inference
   * @returns {Promise<InferenceResponse>} The inference response
   */
  async processInference(sessionId, inputText, history = null, parameters = null) {
    const data = {
      sessionId,
      input: inputText
    };
    
    if (history) {
      data.history = history.map(msg => msg.toObject());
    }
    
    if (parameters) {
      data.parameters = parameters;
    }
    
    const response = await this._request('POST', '/v1/inference', data);
    return InferenceResponse.fromObject(response);
  }

  /**
   * Get the schema of an agent.
   * @param {string} sessionId - The session ID to use
   * @returns {Promise<Object>} The agent schema
   */
  async getAgentSchema(sessionId) {
    return this._request('GET', `/v1/schema?sessionId=${sessionId}`);
  }

  /**
   * Get the capabilities of an agent.
   * @param {string} sessionId - The session ID to use
   * @returns {Promise<AgentCapabilities>} The agent capabilities
   */
  async getAgentCapabilities(sessionId) {
    const response = await this._request('GET', `/v1/capabilities?sessionId=${sessionId}`);
    return AgentCapabilities.fromObject(response);
  }

  /**
   * Get a value from the agent's memory.
   * @param {string} sessionId - The session ID to use
   * @param {string} key - The key to get
   * @returns {Promise<*>} The memory value
   */
  async getMemory(sessionId, key) {
    const response = await this._request('GET', `/v1/memory?sessionId=${sessionId}&key=${key}`);
    return response.value;
  }

  /**
   * Set a value in the agent's memory.
   * @param {string} sessionId - The session ID to use
   * @param {string} key - The key to set
   * @param {*} value - The value to set
   * @returns {Promise<Object>} The response
   */
  async setMemory(sessionId, key, value) {
    const data = {
      value
    };
    
    return this._request('POST', `/v1/memory?sessionId=${sessionId}&key=${key}`, data);
  }

  /**
   * Get information about the TEE for an agent.
   * @param {string} sessionId - The session ID to use
   * @returns {Promise<Object>} The TEE information
   */
  async getTEEInfo(sessionId) {
    return this._request('GET', `/v1/tee?sessionId=${sessionId}`);
  }

  // Terminal management methods

  /**
   * Create a new terminal session.
   * @param {string} sessionId - The session ID to use
   * @param {number} [rows=24] - The number of rows in the terminal
   * @param {number} [cols=80] - The number of columns in the terminal
   * @returns {Promise<TerminalSession>} A TerminalSession object
   */
  async createTerminal(sessionId, rows = 24, cols = 80) {
    const data = {
      sessionId,
      rows,
      cols
    };

    const response = await this._request('POST', '/v1/terminal/create', data);

    return new TerminalSession(
      response.terminalId,
      sessionId,
      rows,
      cols,
      this
    );
  }

  /**
   * Resize a terminal session.
   * @param {string} sessionId - The session ID to use
   * @param {string} terminalId - The terminal ID to resize
   * @param {number} rows - The new number of rows
   * @param {number} cols - The new number of columns
   * @returns {Promise<Object>} The response from the server
   */
  async resizeTerminal(sessionId, terminalId, rows, cols) {
    const data = {
      sessionId,
      terminalId,
      rows,
      cols
    };

    return this._request('POST', '/v1/terminal/resize', data);
  }

  /**
   * Write data to a terminal session.
   * @param {string} sessionId - The session ID to use
   * @param {string} terminalId - The terminal ID to write to
   * @param {string} data - The data to write
   * @returns {Promise<Object>} The response from the server
   */
  async writeToTerminal(sessionId, terminalId, data) {
    const requestData = {
      sessionId,
      terminalId,
      data
    };

    return this._request('POST', '/v1/terminal/write', requestData);
  }

  /**
   * Read data from a terminal session.
   * @param {string} sessionId - The session ID to use
   * @param {string} terminalId - The terminal ID to read from
   * @returns {Promise<string>} The terminal output
   */
  async readFromTerminal(sessionId, terminalId) {
    const response = await this._request('GET', `/v1/terminal/read?sessionId=${sessionId}&terminalId=${terminalId}`);
    return response.data;
  }

  /**
   * Close a terminal session.
   * @param {string} sessionId - The session ID to use
   * @param {string} terminalId - The terminal ID to close
   * @returns {Promise<Object>} The response from the server
   */
  async closeTerminal(sessionId, terminalId) {
    const data = {
      sessionId,
      terminalId
    };

    return this._request('POST', '/v1/terminal/close', data);
  }
}

// Export for Node.js
if (typeof module !== 'undefined' && module.exports) {
  module.exports = {
    AgentClient,
    ConversationMessage,
    ToolCall,
    InferenceResponse,
    AgentCapabilities,
    TerminalSession
  };
}

// Example usage (for browsers)
// const client = new AgentClient('http://localhost:8080', 'test-api-key');
// 
// async function example() {
//   // List available agents
//   const agents = await client.listAgents();
//   console.log(`Available agents: ${agents}`);
//   
//   // Create a session ID
//   const sessionId = `session-${Date.now()}`;
//   
//   // Activate an agent
//   await client.activateAgent('example', '1.0', sessionId);
//   
//   try {
//     // Get agent capabilities
//     const capabilities = await client.getAgentCapabilities(sessionId);
//     console.log('Agent capabilities:', capabilities);
//     
//     // Process an inference request
//     const response = await client.processInference(sessionId, 'Hello, world!');
//     console.log(`Response: ${response.output}`);
//     
//     // Set a memory value
//     await client.setMemory(sessionId, 'greeting', 'Hello, world!');
//     
//     // Get a memory value
//     const greeting = await client.getMemory(sessionId, 'greeting');
//     console.log(`Greeting: ${greeting}`);
//
//     // Create a terminal session
//     const terminal = await client.createTerminal(sessionId, 24, 80);
//     console.log(`Created terminal: ${terminal.terminalId}`);
//
//     // Write to the terminal
//     await terminal.write("echo 'Hello from terminal!'\n");
//
//     // Wait a bit for the command to execute
//     await new Promise(resolve => setTimeout(resolve, 500));
//
//     // Read from the terminal
//     const output = await terminal.read();
//     console.log(`Terminal output: ${output}`);
//
//     // Close the terminal
//     await terminal.close();
//     console.log('Terminal closed');
//
//   } finally {
//     // Deactivate the agent
//     await client.deactivateAgent(sessionId);
//   }
// }
// 
// example().catch(console.error);