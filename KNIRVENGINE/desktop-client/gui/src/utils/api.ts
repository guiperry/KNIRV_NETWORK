// API utility functions for the KNIRVENGINE

// API configuration - detect if running in Electron
const getApiBaseUrl = () => {
  // Check if running in Electron (file:// protocol)
  if (window.location.protocol === 'file:') {
    return 'http://localhost:8081'; // Backend server port in Electron
  }
  return ''; // Use relative URLs for web version
};

// Base API URL - adjust based on environment
const API_BASE_URL = `${getApiBaseUrl()}/api/v1`;

// Helper function to get auth headers
const getAuthHeaders = (): HeadersInit => {
  const token = localStorage.getItem('token');
  return {
    'Content-Type': 'application/json',
    ...(token && { 'Authorization': `Bearer ${token}` })
  };
};

// Helper function to handle API responses with automatic token refresh
const handleApiResponse = async (response: Response): Promise<any> => {
  if (response.status === 401) {
    // Token might be expired, try to refresh
    const refreshResult = await refreshAuthToken();
    if (!refreshResult) {
      // Refresh failed, redirect to login
      window.location.href = '/login';
      throw new Error('Authentication required');
    }
    // If refresh succeeded, the original request should be retried by the caller
    throw new Error('TOKEN_REFRESH_NEEDED');
  }

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.message || `HTTP ${response.status}: ${response.statusText}`);
  }

  return response.json();
};

// Function to refresh authentication token
const refreshAuthToken = async (): Promise<boolean> => {
  try {
    const token = localStorage.getItem('token');
    if (!token) return false;

    const response = await fetch(`${API_BASE_URL}/auth/refresh`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      return false;
    }

    const data = await response.json();
    localStorage.setItem('token', data.token);
    if (data.user) {
      localStorage.setItem('user', JSON.stringify(data.user));
    }

    return true;
  } catch (error) {
    console.error('Token refresh failed:', error);
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    return false;
  }
};

// Enhanced fetch function with automatic retry on token refresh and connection errors
const apiRequest = async (url: string, options: RequestInit = {}, retryCount = 0): Promise<any> => {
  const maxRetries = 3;
  const retryDelay = 1000; // 1 second

  const requestOptions = {
    ...options,
    headers: {
      ...getAuthHeaders(),
      ...options.headers,
    },
  };

  try {
    const response = await fetch(url, requestOptions);
    return await handleApiResponse(response);
  } catch (error) {
    // Handle token refresh
    if (error.message === 'TOKEN_REFRESH_NEEDED') {
      const retryOptions = {
        ...options,
        headers: {
          ...getAuthHeaders(),
          ...options.headers,
        },
      };
      const retryResponse = await fetch(url, retryOptions);
      return await handleApiResponse(retryResponse);
    }

    // Handle connection errors with retry
    if (error.name === 'TypeError' && error.message.includes('fetch') && retryCount < maxRetries) {
      console.warn(`API request failed, retrying... (${retryCount + 1}/${maxRetries})`, error);
      await new Promise(resolve => setTimeout(resolve, retryDelay * (retryCount + 1)));
      return apiRequest(url, options, retryCount + 1);
    }

    throw error;
  }
};

// Agent interfaces based on the backend model in api/agent_service.go
export interface Agent {
  id: string;
  owner_id: number;
  name: string;
  type: string;
  config: string;
  created_at?: string;
  updated_at?: string;
  
  // Frontend-specific fields (not in backend model)
  collection?: string;
  image_url?: string;
  status?: string;
  capabilities?: string[];
  target_types?: string[];
  apiKeys?: {
    openai?: string;
    claude?: string;
    gemini?: string;
    deepseek?: string;
    cerebras?: string;
  };
}

export interface AgentCreationRequest {
  name: string;
  type: string;
  config: string;

  // Frontend-specific fields to be stored in config
  collection?: string;
  image_url?: string;
  capabilities?: string[];
  target_types?: string[];
  api_keys?: {
    openai?: string;
    claude?: string;
    gemini?: string;
    deepseek?: string;
    cerebras?: string;
  };
}

// Agent Builder interfaces
export interface AgentTemplate {
  id: string;
  name: string;
  description: string;
  type: string;
  config_schema: Record<string, any>;
  preview?: string;
}

export interface AgentBuildConfig {
  template_id: string;
  agent_name: string;
  agent_description: string;
  capabilities: string[];
  memory_limit?: number;
  cpu_cores?: number;
  timeout_sec?: number;
  network_access?: boolean;
  file_system_access?: boolean;
  tool_imports?: string;
  custom_tools?: any[];
  sub_agents?: string[];
  extra_params?: Record<string, any>;
}

export interface AgentBuildStatus {
  agent_id: string;
  status: 'pending' | 'building' | 'success' | 'error';
  progress: number;
  message?: string;
  plugin_path?: string;
  error_details?: string;
  started_at?: string;
  completed_at?: string;
}

export interface CompiledPlugin {
  id: string;
  agent_id: string;
  filename: string;
  path: string;
  size: number;
  created_at: string;
  version: string;
}

export interface SubAgent {
  id: string;
  parent_id: string;
  name: string;
  template: 'python' | 'java';
  status: 'starting' | 'running' | 'stopped' | 'error';
  config: Record<string, any>;
  created_at: string;
  terminal_session?: string;
}

// Agent API functions
export const fetchAgents = async (ownerId: number): Promise<Agent[]> => {
  try {
    // Include owner_id as a query parameter as required by the backend
    const data = await apiRequest(`${API_BASE_URL}/agents?owner_id=${ownerId}`);
    
    // Process each agent to extract frontend-specific fields from config
    return data.agents.map((agent: Agent) => {
      let configData: Record<string, any> = {};
      try {
        if (agent.config) {
          configData = JSON.parse(agent.config);
        }
      } catch (e) {
        console.warn(`Failed to parse config for agent ${agent.id}:`, e);
      }
      
      return {
        ...agent,
        collection: configData && 'collection' in configData ? configData.collection : '',
        image_url: configData && 'image_url' in configData ? configData.image_url : '',
        capabilities: configData && 'capabilities' in configData ? configData.capabilities : [],
        target_types: configData && 'target_types' in configData ? configData.target_types : [],
        apiKeys: configData && 'api_keys' in configData ? configData.api_keys : {},
        status: configData && 'status' in configData ? configData.status : 'idle'
      };
    });
  } catch (error) {
    console.error('Error fetching agents:', error);
    throw error;
  }
};

export const fetchAgentById = async (id: string): Promise<Agent> => {
  try {
    const response = await fetch(`${API_BASE_URL}/agents/${id}`);
    
    if (!response.ok) {
      throw new Error(`Failed to fetch agent: ${response.statusText}`);
    }
    
    const data = await response.json();
    const agent = data.agent;
    
    // Extract frontend-specific fields from config
    let configData: Record<string, any> = {};
    try {
      if (agent.config) {
        configData = JSON.parse(agent.config);
      }
    } catch (e) {
      console.warn(`Failed to parse config for agent ${agent.id}:`, e);
    }
    
    return {
      ...agent,
      collection: configData && 'collection' in configData ? configData.collection : '',
      image_url: configData && 'image_url' in configData ? configData.image_url : '',
      capabilities: configData && 'capabilities' in configData ? configData.capabilities : [],
      target_types: configData && 'target_types' in configData ? configData.target_types : [],
      apiKeys: configData && 'api_keys' in configData ? configData.api_keys : {},
      status: configData && 'status' in configData ? configData.status : 'idle'
    };
  } catch (error) {
    console.error(`Error fetching agent ${id}:`, error);
    throw error;
  }
};

export const createAgent = async (agentData: AgentCreationRequest): Promise<Agent> => {
  try {
    // Prepare the agent data according to the backend model
    const configObj = {
      collection: agentData.collection || '',
      image_url: agentData.image_url || '',
      capabilities: agentData.capabilities || [],
      target_types: agentData.target_types || [],
      api_keys: agentData.api_keys || {},
      status: 'idle'
    };

    // Create the API request payload
    const apiPayload = {
      name: agentData.name,
      type: agentData.type || 'standard',
      config: agentData.config || JSON.stringify(configObj)
    };

    const data = await apiRequest(`${API_BASE_URL}/agents`, {
      method: 'POST',
      body: JSON.stringify(apiPayload),
    });
    
    // Parse the config string to extract frontend-specific fields
    let configData: Record<string, any> = {};
    try {
      if (data.agent.config) {
        configData = JSON.parse(data.agent.config);
      }
    } catch (e) {
      console.warn('Failed to parse agent config:', e);
    }
    
    // Combine backend data with frontend-specific fields from config
    return {
      ...data.agent,
      collection: configData && 'collection' in configData ? configData.collection : '',
      image_url: configData && 'image_url' in configData ? configData.image_url : '',
      capabilities: configData && 'capabilities' in configData ? configData.capabilities : [],
      target_types: configData && 'target_types' in configData ? configData.target_types : [],
      apiKeys: configData && 'api_keys' in configData ? configData.api_keys : {},
      status: configData && 'status' in configData ? configData.status : 'idle'
    };
  } catch (error) {
    console.error('Error creating agent:', error);
    throw error;
  }
};

export const updateAgent = async (id: string, agentData: Partial<Agent>): Promise<Agent> => {
  try {
    // First, get the current agent to merge with updates
    const currentAgent = await fetchAgentById(id);
    
    // Parse the current config
    interface AgentConfig {
      collection?: string;
      image_url?: string;
      capabilities?: string[];
      target_types?: string[];
      api_keys?: {
        openai?: string;
        claude?: string;
        gemini?: string;
        deepseek?: string;
        cerebras?: string;
      };
      status?: string;
    }
    
    let configObj: AgentConfig = {};
    try {
      configObj = JSON.parse(currentAgent.config);
    } catch (e) {
      console.warn('Failed to parse current agent config:', e);
    }
    
    // Update the config object with new frontend-specific fields
    if (agentData.collection !== undefined) configObj.collection = agentData.collection;
    if (agentData.image_url !== undefined) configObj.image_url = agentData.image_url;
    if (agentData.capabilities !== undefined) configObj.capabilities = agentData.capabilities;
    if (agentData.target_types !== undefined) configObj.target_types = agentData.target_types;
    if (agentData.apiKeys !== undefined) configObj.api_keys = agentData.apiKeys;
    if (agentData.status !== undefined) configObj.status = agentData.status;
    
    // Create the API request payload
    const apiPayload: Partial<Agent> = {
      name: agentData.name !== undefined ? agentData.name : currentAgent.name,
      type: agentData.type !== undefined ? agentData.type : currentAgent.type,
      config: JSON.stringify(configObj)
    };
    
    const response = await fetch(`${API_BASE_URL}/agents/${id}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(apiPayload),
    });
    
    if (!response.ok) {
      throw new Error(`Failed to update agent: ${response.statusText}`);
    }
    
    const data = await response.json();
    
    // Parse the updated config string to extract frontend-specific fields
    let updatedConfigData: Record<string, any> = {};
    try {
      if (data.agent.config) {
        updatedConfigData = JSON.parse(data.agent.config);
      }
    } catch (e) {
      console.warn('Failed to parse updated agent config:', e);
    }
    
    // Combine backend data with frontend-specific fields from config
    return {
      ...data.agent,
      collection: updatedConfigData && 'collection' in updatedConfigData ? updatedConfigData.collection : '',
      image_url: updatedConfigData && 'image_url' in updatedConfigData ? updatedConfigData.image_url : '',
      capabilities: updatedConfigData && 'capabilities' in updatedConfigData ? updatedConfigData.capabilities : [],
      target_types: updatedConfigData && 'target_types' in updatedConfigData ? updatedConfigData.target_types : [],
      apiKeys: updatedConfigData && 'api_keys' in updatedConfigData ? updatedConfigData.api_keys : {},
      status: updatedConfigData && 'status' in updatedConfigData ? updatedConfigData.status : 'idle'
    };
  } catch (error) {
    console.error(`Error updating agent ${id}:`, error);
    throw error;
  }
};

export const deleteAgent = async (id: string): Promise<void> => {
  try {
    const response = await fetch(`${API_BASE_URL}/agents/${id}`, {
      method: 'DELETE',
    });

    if (!response.ok) {
      throw new Error(`Failed to delete agent: ${response.statusText}`);
    }
  } catch (error) {
    console.error(`Error deleting agent ${id}:`, error);
    throw error;
  }
};

export const deployAgent = async (agentId: string, targetId: string): Promise<{ success: boolean }> => {
  try {
    const response = await fetch(`${API_BASE_URL}/agents/${agentId}/deploy`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ target_id: targetId }),
    });

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }

    return await response.json();
  } catch (error) {
    console.error('Error deploying agent:', error);
    throw error;
  }
};

export const uploadAgentFile = async (file: File): Promise<{ success: boolean; fileId?: string; message?: string }> => {
  try {
    const formData = new FormData();
    formData.append('file', file);

    const response = await fetch(`${API_BASE_URL}/agents/upload`, {
      method: 'POST',
      headers: {
        // Don't set Content-Type for FormData, let browser set it with boundary
        ...(localStorage.getItem('token') && { 'Authorization': `Bearer ${localStorage.getItem('token')}` })
      },
      body: formData,
    });

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }

    return await response.json();
  } catch (error) {
    console.error('Error uploading agent file:', error);
    throw error;
  }
};

export const stopAgent = async (agentId: string): Promise<{ success: boolean }> => {
  try {
    const response = await fetch(`${API_BASE_URL}/agents/${agentId}/stop`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }

    return await response.json();
  } catch (error) {
    console.error('Error stopping agent:', error);
    throw error;
  }
};

// Workflow interfaces
export interface WorkflowRequest {
  agent_id: string;
  target_id: string;
  capability_id: string;
  input: Record<string, any>;
}

export interface Workflow {
  id: string;
  status: string;
  start_time: string;
  end_time?: string;
  agent_id: string;
  target_id: string;
  capability_id: string;
  input: Record<string, any>;
  output?: Record<string, any>;
  error?: string;
  owner_id: number;
}

// Target System API functions
export const fetchTargets = async (): Promise<any[]> => {
  try {
    const response = await fetch(`${API_BASE_URL}/targets`);

    if (!response.ok) {
      throw new Error(`Failed to fetch targets: ${response.statusText}`);
    }

    const data = await response.json();
    return data.targets;
  } catch (error) {
    console.error('Error fetching targets:', error);
    throw error;
  }
};

// Workflow API functions
export const startWorkflow = async (workflowData: WorkflowRequest): Promise<Workflow> => {
  try {
    const response = await fetch(`${API_BASE_URL}/workflows`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(workflowData),
    });
    
    if (!response.ok) {
      throw new Error(`Failed to start workflow: ${response.statusText}`);
    }
    
    const data = await response.json();
    return data.workflow;
  } catch (error) {
    console.error('Error starting workflow:', error);
    throw error;
  }
};

export const fetchWorkflow = async (id: string): Promise<Workflow> => {
  try {
    const response = await fetch(`${API_BASE_URL}/workflows/${id}`);
    
    if (!response.ok) {
      throw new Error(`Failed to fetch workflow: ${response.statusText}`);
    }
    
    const data = await response.json();
    return data.workflow;
  } catch (error) {
    console.error(`Error fetching workflow ${id}:`, error);
    throw error;
  }
};

export const fetchWorkflows = async (): Promise<Workflow[]> => {
  try {
    const response = await fetch(`${API_BASE_URL}/workflows`);
    
    if (!response.ok) {
      throw new Error(`Failed to fetch workflows: ${response.statusText}`);
    }
    
    const data = await response.json();
    return data.workflows;
  } catch (error) {
    console.error('Error fetching workflows:', error);
    throw error;
  }
};

export const cancelWorkflow = async (id: string): Promise<void> => {
  try {
    const response = await fetch(`${API_BASE_URL}/workflows/${id}/cancel`, {
      method: 'POST',
    });
    
    if (!response.ok) {
      throw new Error(`Failed to cancel workflow: ${response.statusText}`);
    }
  } catch (error) {
    console.error(`Error canceling workflow ${id}:`, error);
    throw error;
  }
};

// ADK Agent interfaces
export interface ADKAgentConfig {
  agent_id?: string;
  agent_type: 'llm' | 'sequential' | 'parallel' | 'loop';
  name: string;
  model?: string;
  instruction?: string;
  description?: string;
  use_search?: boolean;
  use_code_execution?: boolean;
  use_vertex_search?: boolean;
  vertex_datastore_id?: string;
  custom_tools?: ADKTool[];
  sub_agents?: string[];
  max_iterations?: number;
  extra_params?: Record<string, any>;
}

export interface ADKTool {
  name: string;
  description: string;
  endpoint: string;
  parameters: Record<string, any>;
}

// ADK Agent API functions
export const createADKAgent = async (config: ADKAgentConfig): Promise<string> => {
  try {
    const response = await fetch(`${API_BASE_URL}/adk/agents`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(config),
    });
    
    if (!response.ok) {
      throw new Error(`Failed to create ADK agent: ${response.statusText}`);
    }
    
    const data = await response.json();
    return data.agent_id;
  } catch (error) {
    console.error('Error creating ADK agent:', error);
    throw error;
  }
};

export const runADKAgent = async (
  agentId: string, 
  input: string, 
  sessionId: string = 'default'
): Promise<string> => {
  try {
    const response = await fetch(`${API_BASE_URL}/adk/agents/${agentId}/run`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        input,
        session_id: sessionId,
      }),
    });
    
    if (!response.ok) {
      throw new Error(`Failed to run ADK agent: ${response.statusText}`);
    }
    
    const data = await response.json();
    return data.response;
  } catch (error) {
    console.error(`Error running ADK agent ${agentId}:`, error);
    throw error;
  }
};

export const getADKAgent = async (agentId: string): Promise<ADKAgentConfig> => {
  try {
    const response = await fetch(`${API_BASE_URL}/adk/agents/${agentId}`);
    
    if (!response.ok) {
      throw new Error(`Failed to get ADK agent: ${response.statusText}`);
    }
    
    const data = await response.json();
    return data.agent;
  } catch (error) {
    console.error(`Error getting ADK agent ${agentId}:`, error);
    throw error;
  }
};

export const listADKAgents = async (): Promise<string[]> => {
  try {
    const response = await fetch(`${API_BASE_URL}/adk/agents`);
    
    if (!response.ok) {
      throw new Error(`Failed to list ADK agents: ${response.statusText}`);
    }
    
    const data = await response.json();
    return data.agents;
  } catch (error) {
    console.error('Error listing ADK agents:', error);
    throw error;
  }
};

export const deleteADKAgent = async (agentId: string): Promise<void> => {
  try {
    const response = await fetch(`${API_BASE_URL}/adk/agents/${agentId}`, {
      method: 'DELETE',
    });

    if (!response.ok) {
      throw new Error(`Failed to delete ADK agent: ${response.statusText}`);
    }
  } catch (error) {
    console.error(`Error deleting ADK agent ${agentId}:`, error);
    throw error;
  }
};

// Plugin discovery and import interfaces
export interface PluginInfo {
  filePath: string;
  fileName: string;
  agentId?: string;
  version?: string;
  isRegistered: boolean;
  size: number;
  modTime: string;
  metadata?: Record<string, any>;
  error?: string;
}

export interface ImportPluginRequest {
  filePath: string;
  agentId: string;
  version: string;
}

// Plugin discovery and import API functions
export const discoverAllPlugins = async (): Promise<PluginInfo[]> => {
  try {
    const response = await fetch(`${API_BASE_URL}/plugins/discover`);

    if (!response.ok) {
      throw new Error(`Failed to discover plugins: ${response.statusText}`);
    }

    const data = await response.json();
    return data.plugins;
  } catch (error) {
    console.error('Error discovering plugins:', error);
    throw error;
  }
};

export const importPlugin = async (request: ImportPluginRequest): Promise<void> => {
  try {
    const response = await fetch(`${API_BASE_URL}/plugins/import`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(request),
    });

    if (!response.ok) {
      throw new Error(`Failed to import plugin: ${response.statusText}`);
    }
  } catch (error) {
    console.error('Error importing plugin:', error);
    throw error;
  }
};

// ADK Agent-to-Agent (A2A) Communication Interfaces
export interface DiscoveredAgent {
  id: string;
  name: string;
  description: string;
  capabilities: string[];
  endpoint: string;
}

export interface AgentMessage {
  sender_id: string;
  recipient_id: string;
  content: string;
  timestamp: string;
  metadata?: Record<string, any>;
}

export interface AgentTask {
  id: string;
  name: string;
  description: string;
  status: 'pending' | 'in_progress' | 'completed' | 'failed';
  created_at: string;
  updated_at: string;
  input?: Record<string, any>;
  output?: Record<string, any>;
}

export interface AgentCapabilities {
  agent_id: string;
  capabilities: string[];
  updated_at: string;
}

// ADK Agent-to-Agent (A2A) Communication API functions
export const discoverAgents = async (): Promise<DiscoveredAgent[]> => {
  try {
    const response = await fetch(`${API_BASE_URL}/adk/agents`);

    if (!response.ok) {
      throw new Error(`Failed to discover agents: ${response.statusText}`);
    }

    const data = await response.json();
    return data.agents;
  } catch (error) {
    console.error('Error discovering agents:', error);
    throw error;
  }
};

/**
 * Fetch available agents for selection (both plugins and WASM)
 * This returns a list of agent IDs that can be used for agent creation
 */
export const fetchAvailableAgents = async (): Promise<string[]> => {
  try {
    const response = await fetch(`${API_BASE_URL}/adk/agents`);

    if (!response.ok) {
      throw new Error(`Failed to fetch available agents: ${response.statusText}`);
    }

    const data = await response.json();
    return data.agents || [];
  } catch (error) {
    console.error('Error fetching available agents:', error);
    throw error;
  }
};

/**
 * Fetch all agents (database + discovered) for the main agent listing
 * This combines user-created agents with discovered plugin/WASM agents
 */
export const fetchAllAgents = async (): Promise<any[]> => {
  try {
    const response = await fetch(`${API_BASE_URL}/agents/all`);

    if (!response.ok) {
      throw new Error(`Failed to fetch all agents: ${response.statusText}`);
    }

    const data = await response.json();
    return data.agents || [];
  } catch (error) {
    console.error('Error fetching all agents:', error);
    throw error;
  }
};

export const sendAgentMessage = async (
  recipientId: string,
  content: string,
  metadata?: Record<string, any>
): Promise<AgentMessage> => {
  try {
    const response = await fetch(`${API_BASE_URL}/agents/message`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        recipient_id: recipientId,
        content,
        metadata
      }),
    });
    
    if (!response.ok) {
      throw new Error(`Failed to send agent message: ${response.statusText}`);
    }
    
    const data = await response.json();
    return data.message;
  } catch (error) {
    console.error('Error sending agent message:', error);
    throw error;
  }
};

export const getAgentTask = async (taskId: string): Promise<AgentTask> => {
  try {
    const response = await fetch(`${API_BASE_URL}/agents/tasks/${taskId}`);
    
    if (!response.ok) {
      throw new Error(`Failed to get agent task: ${response.statusText}`);
    }
    
    const data = await response.json();
    return data.task;
  } catch (error) {
    console.error(`Error getting agent task ${taskId}:`, error);
    throw error;
  }
};

export const updateAgentCapabilities = async (
  agentId: string,
  capabilities: string[]
): Promise<AgentCapabilities> => {
  try {
    const response = await fetch(`${API_BASE_URL}/agents/capabilities`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        agent_id: agentId,
        capabilities
      }),
    });

    if (!response.ok) {
      throw new Error(`Failed to update agent capabilities: ${response.statusText}`);
    }

    const data = await response.json();
    return data.capabilities;
  } catch (error) {
    console.error(`Error updating capabilities for agent ${agentId}:`, error);
    throw error;
  }
};

// MCP Server interfaces
export interface MCPServer {
  id: string;
  name: string;
  description: string;
  type: 'typescript' | 'python';
  category: string;
  repository: string;
  install_command: string;
  run_command: string;
  status: 'available' | 'installed' | 'running' | 'stopped';
  version: string;
  dependencies?: string[];
  configuration?: Record<string, any>;
  tags?: string[];
  rating: number;
  downloads: number;
  last_updated: string;
  owner_id?: string;
  created_at: string;
  updated_at: string;
}

export interface MCPServerInstallStatus {
  server_id: string;
  status: 'pending' | 'installing' | 'completed' | 'failed';
  progress: number;
  message: string;
  error?: string;
  started_at: string;
  completed_at?: string;
  log_output: string[];
}

export interface MCPServerConfig {
  server_id: string;
  name: string;
  description: string;
  environment: Record<string, string>;
  arguments: string[];
  working_dir: string;
  auto_start: boolean;
  restart_on_fail: boolean;
  max_restarts: number;
  health_check: {
    enabled: boolean;
    interval: number;
    timeout: number;
  };
  resources: {
    max_memory: number;
    max_cpu: number;
  };
  security: {
    sandboxed: boolean;
    allowed_paths: string[];
    allowed_hosts: string[];
    allowed_ports: number[];
    read_only_mode: boolean;
  };
  created_at: string;
  updated_at: string;
}

export interface MCPServerMetrics {
  server_id: string;
  status: string;
  uptime: number;
  restart_count: number;
  last_restart: string;
  memory_usage: number;
  cpu_usage: number;
  request_count: number;
  error_count: number;
  last_health_check: string;
  health_status: 'healthy' | 'unhealthy' | 'unknown';
  log_level: string;
  created_at: string;
  updated_at: string;
}

export interface MCPLogEntry {
  id: string;
  server_id: string;
  level: 'debug' | 'info' | 'warn' | 'error';
  message: string;
  timestamp: string;
  source: 'server' | 'system' | 'user';
  metadata?: Record<string, any>;
}

export interface MCPAlert {
  id: string;
  server_id: string;
  type: 'error' | 'warning' | 'info';
  title: string;
  description: string;
  severity: 'low' | 'medium' | 'high' | 'critical';
  status: 'active' | 'resolved' | 'acknowledged';
  created_at: string;
  resolved_at?: string;
  metadata?: Record<string, any>;
}

// Analytics and Performance API functions
export interface AgentActivity {
  id: number;
  type: string;
  target: string;
  capability: string;
  status: string;
  timestamp: string;
  duration: string;
  result: string;
}

export interface PerformanceData {
  successRate: Array<{ date: string; rate: number }>;
  responseTime: Array<{ date: string; time: number }>;
  usage: Array<{ date: string; count: number }>;
}

export const fetchAgentActivity = async (agentId: string): Promise<AgentActivity[]> => {
  try {
    const response = await fetch(`${API_BASE_URL}/analytics/agent/${agentId}/activity`);

    if (!response.ok) {
      throw new Error(`Failed to fetch agent activity: ${response.statusText}`);
    }

    const data = await response.json();
    return data.activities || [];
  } catch (error) {
    console.error('Error fetching agent activity:', error);
    // Return empty array instead of mock data for production
    return [];
  }
};

export const fetchAgentPerformance = async (agentId: string): Promise<PerformanceData> => {
  try {
    const response = await fetch(`${API_BASE_URL}/analytics/agent/${agentId}/performance`);

    if (!response.ok) {
      throw new Error(`Failed to fetch agent performance: ${response.statusText}`);
    }

    const data = await response.json();
    return data.performance || {
      successRate: [],
      responseTime: [],
      usage: []
    };
  } catch (error) {
    console.error('Error fetching agent performance:', error);
    // Return mock data as fallback for now
    return {
      successRate: [
        { date: '2024-01-01', rate: 92.5 },
        { date: '2024-01-02', rate: 94.2 },
        { date: '2024-01-03', rate: 91.8 },
        { date: '2024-01-04', rate: 95.3 },
        { date: '2024-01-05', rate: 97.1 },
        { date: '2024-01-06', rate: 98.4 },
        { date: '2024-01-07', rate: 98.7 }
      ],
      responseTime: [
        { date: '2024-01-01', time: 4.8 },
        { date: '2024-01-02', time: 4.5 },
        { date: '2024-01-03', time: 5.2 },
        { date: '2024-01-04', time: 4.1 },
        { date: '2024-01-05', time: 3.7 },
        { date: '2024-01-06', time: 3.4 },
        { date: '2024-01-07', time: 3.2 }
      ],
      usage: [
        { date: '2024-01-01', count: 23 },
        { date: '2024-01-02', count: 31 },
        { date: '2024-01-03', count: 28 },
        { date: '2024-01-04', count: 42 },
        { date: '2024-01-05', count: 56 },
        { date: '2024-01-06', count: 67 },
        { date: '2024-01-07', count: 78 }
      ]
    };
  }
};

// MCP Server API functions
export const fetchMCPServers = async (params?: {
  category?: string;
  type?: string;
  status?: string;
  search?: string;
  limit?: number;
  offset?: number;
}): Promise<{ servers: MCPServer[]; count: number; last_sync: string }> => {
  try {
    const queryParams = new URLSearchParams();
    if (params?.category) queryParams.append('category', params.category);
    if (params?.type) queryParams.append('type', params.type);
    if (params?.status) queryParams.append('status', params.status);
    if (params?.search) queryParams.append('search', params.search);
    if (params?.limit) queryParams.append('limit', params.limit.toString());
    if (params?.offset) queryParams.append('offset', params.offset.toString());

    const url = `${API_BASE_URL}/mcp/servers${queryParams.toString() ? '?' + queryParams.toString() : ''}`;
    const response = await fetch(url);

    if (!response.ok) {
      throw new Error(`Failed to fetch MCP servers: ${response.statusText}`);
    }

    return await response.json();
  } catch (error) {
    console.error('Error fetching MCP servers:', error);
    throw error;
  }
};

export const fetchMCPServer = async (serverId: string): Promise<MCPServer> => {
  try {
    const response = await fetch(`${API_BASE_URL}/mcp/servers/${serverId}`);

    if (!response.ok) {
      throw new Error(`Failed to fetch MCP server: ${response.statusText}`);
    }

    const data = await response.json();
    return data.server;
  } catch (error) {
    console.error(`Error fetching MCP server ${serverId}:`, error);
    throw error;
  }
};

export const installMCPServer = async (serverId: string): Promise<{ message: string }> => {
  try {
    const response = await fetch(`${API_BASE_URL}/mcp/servers/${serverId}/install`, {
      method: 'POST',
    });

    if (!response.ok) {
      throw new Error(`Failed to install MCP server: ${response.statusText}`);
    }

    return await response.json();
  } catch (error) {
    console.error(`Error installing MCP server ${serverId}:`, error);
    throw error;
  }
};

export const uninstallMCPServer = async (serverId: string): Promise<{ message: string }> => {
  try {
    const response = await fetch(`${API_BASE_URL}/mcp/servers/${serverId}/uninstall`, {
      method: 'DELETE',
    });

    if (!response.ok) {
      throw new Error(`Failed to uninstall MCP server: ${response.statusText}`);
    }

    return await response.json();
  } catch (error) {
    console.error(`Error uninstalling MCP server ${serverId}:`, error);
    throw error;
  }
};

export const getMCPServerInstallStatus = async (serverId: string): Promise<{ status: MCPServerInstallStatus }> => {
  try {
    const response = await fetch(`${API_BASE_URL}/mcp/servers/${serverId}/status`);

    if (!response.ok) {
      throw new Error(`Failed to get MCP server status: ${response.statusText}`);
    }

    return await response.json();
  } catch (error) {
    console.error(`Error getting MCP server status ${serverId}:`, error);
    throw error;
  }
};

export const startMCPServer = async (serverId: string): Promise<{ message: string }> => {
  try {
    const response = await fetch(`${API_BASE_URL}/mcp/servers/${serverId}/start`, {
      method: 'POST',
    });

    if (!response.ok) {
      throw new Error(`Failed to start MCP server: ${response.statusText}`);
    }

    return await response.json();
  } catch (error) {
    console.error(`Error starting MCP server ${serverId}:`, error);
    throw error;
  }
};

export const stopMCPServer = async (serverId: string): Promise<{ message: string }> => {
  try {
    const response = await fetch(`${API_BASE_URL}/mcp/servers/${serverId}/stop`, {
      method: 'POST',
    });

    if (!response.ok) {
      throw new Error(`Failed to stop MCP server: ${response.statusText}`);
    }

    return await response.json();
  } catch (error) {
    console.error(`Error stopping MCP server ${serverId}:`, error);
    throw error;
  }
};

export const restartMCPServer = async (serverId: string): Promise<{ message: string }> => {
  try {
    const response = await fetch(`${API_BASE_URL}/mcp/servers/${serverId}/restart`, {
      method: 'POST',
    });

    if (!response.ok) {
      throw new Error(`Failed to restart MCP server: ${response.statusText}`);
    }

    return await response.json();
  } catch (error) {
    console.error(`Error restarting MCP server ${serverId}:`, error);
    throw error;
  }
};

export const getMCPServerConfig = async (serverId: string): Promise<{ config: MCPServerConfig }> => {
  try {
    const response = await fetch(`${API_BASE_URL}/mcp/servers/${serverId}/config`);

    if (!response.ok) {
      throw new Error(`Failed to get MCP server config: ${response.statusText}`);
    }

    return await response.json();
  } catch (error) {
    console.error(`Error getting MCP server config ${serverId}:`, error);
    throw error;
  }
};

export const updateMCPServerConfig = async (serverId: string, config: Partial<MCPServerConfig>): Promise<{ config: MCPServerConfig }> => {
  try {
    const response = await fetch(`${API_BASE_URL}/mcp/servers/${serverId}/config`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(config),
    });

    if (!response.ok) {
      throw new Error(`Failed to update MCP server config: ${response.statusText}`);
    }

    return await response.json();
  } catch (error) {
    console.error(`Error updating MCP server config ${serverId}:`, error);
    throw error;
  }
};

export const syncMCPServers = async (): Promise<{ count: number; last_sync: string }> => {
  try {
    const response = await fetch(`${API_BASE_URL}/mcp/servers/sync`, {
      method: 'POST',
    });

    if (!response.ok) {
      throw new Error(`Failed to sync MCP servers: ${response.statusText}`);
    }

    return await response.json();
  } catch (error) {
    console.error('Error syncing MCP servers:', error);
    throw error;
  }
};

export const getMCPServerMetrics = async (serverId?: string): Promise<{ metrics: Record<string, MCPServerMetrics>; count: number }> => {
  try {
    const url = `${API_BASE_URL}/mcp/metrics${serverId ? `?server_id=${serverId}` : ''}`;
    const response = await fetch(url);

    if (!response.ok) {
      throw new Error(`Failed to get MCP server metrics: ${response.statusText}`);
    }

    return await response.json();
  } catch (error) {
    console.error('Error getting MCP server metrics:', error);
    throw error;
  }
};

export const getMCPServerLogs = async (params?: {
  server_id?: string;
  level?: string;
  limit?: number;
}): Promise<{ logs: MCPLogEntry[]; count: number }> => {
  try {
    const queryParams = new URLSearchParams();
    if (params?.server_id) queryParams.append('server_id', params.server_id);
    if (params?.level) queryParams.append('level', params.level);
    if (params?.limit) queryParams.append('limit', params.limit.toString());

    const url = `${API_BASE_URL}/mcp/logs${queryParams.toString() ? '?' + queryParams.toString() : ''}`;
    const response = await fetch(url);

    if (!response.ok) {
      throw new Error(`Failed to get MCP server logs: ${response.statusText}`);
    }

    return await response.json();
  } catch (error) {
    console.error('Error getting MCP server logs:', error);
    throw error;
  }
};

export const getMCPServerAlerts = async (params?: {
  server_id?: string;
  status?: string;
}): Promise<{ alerts: MCPAlert[]; count: number }> => {
  try {
    const queryParams = new URLSearchParams();
    if (params?.server_id) queryParams.append('server_id', params.server_id);
    if (params?.status) queryParams.append('status', params.status);

    const url = `${API_BASE_URL}/mcp/alerts${queryParams.toString() ? '?' + queryParams.toString() : ''}`;
    const response = await fetch(url);

    if (!response.ok) {
      throw new Error(`Failed to get MCP server alerts: ${response.statusText}`);
    }

    return await response.json();
  } catch (error) {
    console.error('Error getting MCP server alerts:', error);
    throw error;
  }
};

export const resolveMCPAlert = async (alertId: string): Promise<{ message: string }> => {
  try {
    const response = await fetch(`${API_BASE_URL}/mcp/alerts/${alertId}/resolve`, {
      method: 'POST',
    });

    if (!response.ok) {
      throw new Error(`Failed to resolve MCP alert: ${response.statusText}`);
    }

    return await response.json();
  } catch (error) {
    console.error(`Error resolving MCP alert ${alertId}:`, error);
    throw error;
  }
};

// ============================================================================
// Agent Builder API Functions
// ============================================================================

/**
 * Build an agent plugin from a template
 */
export const buildAgentPlugin = async (agentId: string, templateId: string, config: AgentBuildConfig): Promise<{ message: string; build_id: string }> => {
  try {
    const response = await fetch(`${API_BASE_URL}/agents/${agentId}/build`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        template_id: templateId,
        config: config,
      }),
    });

    if (!response.ok) {
      throw new Error(`Failed to build agent plugin: ${response.statusText}`);
    }

    return await response.json();
  } catch (error) {
    console.error(`Error building agent plugin for ${agentId}:`, error);
    throw error;
  }
};

/**
 * Get the build status of an agent plugin
 */
export const getAgentBuildStatus = async (agentId: string): Promise<AgentBuildStatus> => {
  try {
    const response = await fetch(`${API_BASE_URL}/agents/${agentId}/build`);

    if (!response.ok) {
      throw new Error(`Failed to get agent build status: ${response.statusText}`);
    }

    const data = await response.json();
    return data.build_status;
  } catch (error) {
    console.error(`Error getting build status for agent ${agentId}:`, error);
    throw error;
  }
};

/**
 * Rebuild an existing agent plugin
 */
export const rebuildAgentPlugin = async (agentId: string): Promise<{ message: string; build_id: string }> => {
  try {
    const response = await fetch(`${API_BASE_URL}/agents/${agentId}/rebuild`, {
      method: 'POST',
    });

    if (!response.ok) {
      throw new Error(`Failed to rebuild agent plugin: ${response.statusText}`);
    }

    return await response.json();
  } catch (error) {
    console.error(`Error rebuilding agent plugin for ${agentId}:`, error);
    throw error;
  }
};

/**
 * Fetch available agent templates
 */
export const fetchAgentTemplates = async (): Promise<AgentTemplate[]> => {
  try {
    const response = await fetch(`${API_BASE_URL}/templates`);

    if (!response.ok) {
      throw new Error(`Failed to fetch agent templates: ${response.statusText}`);
    }

    const data = await response.json();
    return data.templates;
  } catch (error) {
    console.error('Error fetching agent templates:', error);
    throw error;
  }
};

/**
 * Fetch compiled agent plugins
 */
export const fetchCompiledPlugins = async (): Promise<CompiledPlugin[]> => {
  try {
    const response = await fetch(`${API_BASE_URL}/plugins`);

    if (!response.ok) {
      throw new Error(`Failed to fetch compiled plugins: ${response.statusText}`);
    }

    const data = await response.json();
    return data.plugins;
  } catch (error) {
    console.error('Error fetching compiled plugins:', error);
    throw error;
  }
};

/**
 * Delete a compiled plugin
 */
export const deleteAgentPlugin = async (pluginId: string): Promise<{ message: string }> => {
  try {
    const response = await fetch(`${API_BASE_URL}/plugins/${pluginId}`, {
      method: 'DELETE',
    });

    if (!response.ok) {
      throw new Error(`Failed to delete agent plugin: ${response.statusText}`);
    }

    return await response.json();
  } catch (error) {
    console.error(`Error deleting agent plugin ${pluginId}:`, error);
    throw error;
  }
};

// ============================================================================
// Sub-Agent API Functions
// ============================================================================

/**
 * Spawn a new sub-agent
 */
export const spawnSubAgent = async (parentId: string, template: 'python' | 'java', config: Record<string, any>): Promise<SubAgent> => {
  try {
    const response = await fetch(`${API_BASE_URL}/agents/${parentId}/sub-agents`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        template: template,
        config: config,
      }),
    });

    if (!response.ok) {
      throw new Error(`Failed to spawn sub-agent: ${response.statusText}`);
    }

    const data = await response.json();
    return data.sub_agent;
  } catch (error) {
    console.error(`Error spawning sub-agent for parent ${parentId}:`, error);
    throw error;
  }
};

/**
 * Get all sub-agents for a parent agent
 */
export const getSubAgents = async (parentId: string): Promise<SubAgent[]> => {
  try {
    const response = await fetch(`${API_BASE_URL}/agents/${parentId}/sub-agents`);

    if (!response.ok) {
      throw new Error(`Failed to get sub-agents: ${response.statusText}`);
    }

    const data = await response.json();
    return data.sub_agents;
  } catch (error) {
    console.error(`Error getting sub-agents for parent ${parentId}:`, error);
    throw error;
  }
};

/**
 * Terminate a sub-agent
 */
export const terminateSubAgent = async (parentId: string, subAgentId: string): Promise<{ message: string }> => {
  try {
    const response = await fetch(`${API_BASE_URL}/agents/${parentId}/sub-agents/${subAgentId}`, {
      method: 'DELETE',
    });

    if (!response.ok) {
      throw new Error(`Failed to terminate sub-agent: ${response.statusText}`);
    }

    return await response.json();
  } catch (error) {
    console.error(`Error terminating sub-agent ${subAgentId}:`, error);
    throw error;
  }
};

/**
 * Get sub-agent terminal session
 */
export const getSubAgentTerminal = async (parentId: string, subAgentId: string): Promise<{ terminal_session: string }> => {
  try {
    const response = await fetch(`${API_BASE_URL}/agents/${parentId}/sub-agents/${subAgentId}/terminal`);

    if (!response.ok) {
      throw new Error(`Failed to get sub-agent terminal: ${response.statusText}`);
    }

    return await response.json();
  } catch (error) {
    console.error(`Error getting terminal for sub-agent ${subAgentId}:`, error);
    throw error;
  }
};

/**
 * Send command to sub-agent terminal
 */
export const sendSubAgentCommand = async (parentId: string, subAgentId: string, command: string): Promise<{ message: string }> => {
  try {
    const response = await fetch(`${API_BASE_URL}/agents/${parentId}/sub-agents/${subAgentId}/command`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        command: command,
      }),
    });

    if (!response.ok) {
      throw new Error(`Failed to send command to sub-agent: ${response.statusText}`);
    }

    return await response.json();
  } catch (error) {
    console.error(`Error sending command to sub-agent ${subAgentId}:`, error);
    throw error;
  }
};

/**
 * Get sub-agent logs
 */
export const getSubAgentLogs = async (parentId: string, subAgentId: string, limit?: number): Promise<{ logs: string[] }> => {
  try {
    const url = `${API_BASE_URL}/agents/${parentId}/sub-agents/${subAgentId}/logs${limit ? `?limit=${limit}` : ''}`;
    const response = await fetch(url);

    if (!response.ok) {
      throw new Error(`Failed to get sub-agent logs: ${response.statusText}`);
    }

    return await response.json();
  } catch (error) {
    console.error(`Error getting logs for sub-agent ${subAgentId}:`, error);
    throw error;
  }
};

/**
 * Toggle demo data on/off
 */
export const toggleDemoData = async (): Promise<{ status: string; enabled: boolean; message: string; has_backup: boolean; errors?: string[] }> => {
  try {
    const response = await fetch(`${API_BASE_URL}/debug/toggle-demo-data`, {
      method: 'POST',
      headers: getAuthHeaders(),
    });

    if (!response.ok) {
      throw new Error(`Failed to toggle demo data: ${response.statusText}`);
    }

    return await response.json();
  } catch (error) {
    console.error('Error toggling demo data:', error);
    throw error;
  }
};

/**
 * Get demo data status
 */
export const getDemoDataStatus = async (): Promise<{ enabled: boolean; has_backup: boolean; backup_timestamp?: string }> => {
  try {
    const response = await fetch(`${API_BASE_URL}/debug/demo-data-status`, {
      method: 'GET',
      headers: getAuthHeaders(),
    });

    if (!response.ok) {
      throw new Error(`Failed to get demo data status: ${response.statusText}`);
    }

    return await response.json();
  } catch (error) {
    console.error('Error getting demo data status:', error);
    throw error;
  }
};