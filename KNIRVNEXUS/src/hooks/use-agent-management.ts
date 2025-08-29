"use client";

import { useState, useEffect, useCallback } from 'react';
import type {
  APIResponse,
  Agent,
  AgentResourceLimits,
  AgentRuntimeInstance,
  AgentResourceUsage,
  AgentDeployment,
  AgentMetrics,
  AgentLog,
  AgentEvent,
  AgentSummary,
  AgentTemplate,
  AgentAction
} from '@/types/api';
import { apiRequest, API_BASE_URL } from '@/lib/api';
import { webSocketService } from '@/lib/websocket-service';

// Additional interfaces not in main types file
export interface AgentFilter {
  status?: string[];
  type?: string[];
  tags?: string[];
  author?: string;
  environment?: string;
  created_after?: string;
  created_before?: string;
  search?: string;
  limit?: number;
  offset?: number;
}

export const useAgentManagement = () => {
  const [agents, setAgents] = useState<Agent[]>([]);
  const [selectedAgent, setSelectedAgent] = useState<Agent | null>(null);
  const [agentMetrics, setAgentMetrics] = useState<Record<string, AgentMetrics[]>>({});
  const [agentLogs, setAgentLogs] = useState<Record<string, AgentLog[]>>({});
  const [agentEvents, setAgentEvents] = useState<AgentEvent[]>([]);
  const [templates, setTemplates] = useState<AgentTemplate[]>([]);
  const [summary, setSummary] = useState<AgentSummary | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isConnected, setIsConnected] = useState(false);

  // Fetch all agents with optional filtering
  const fetchAgents = useCallback(async (filter?: AgentFilter) => {
    setIsLoading(true);
    setError(null);
    
    try {
      let url = `${API_BASE_URL}/api/agent-management/agents`;
      
      if (filter) {
        const params = new URLSearchParams();
        if (filter.status?.length) params.append('status', filter.status.join(','));
        if (filter.type?.length) params.append('type', filter.type.join(','));
        if (filter.author) params.append('author', filter.author);
        if (filter.search) params.append('search', filter.search);
        if (filter.limit) params.append('limit', filter.limit.toString());
        if (filter.offset) params.append('offset', filter.offset.toString());
        
        if (params.toString()) {
          url += `?${params.toString()}`;
        }
      }
      
      const response: APIResponse<Agent[]> = await apiRequest(url, { method: 'GET' });
      
      if (response.success && response.data) {
        setAgents(response.data);
      } else {
        throw new Error(response.error || 'Failed to fetch agents');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to fetch agents:', err);
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Fetch a specific agent
  const fetchAgent = useCallback(async (agentId: string) => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/agent-management/agents/${agentId}`;
      const response: APIResponse<Agent> = await apiRequest(url, { method: 'GET' });
      
      if (response.success && response.data) {
        setSelectedAgent(response.data);
        return response.data;
      } else {
        throw new Error(response.error || 'Failed to fetch agent');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to fetch agent:', err);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Create a new agent
  const createAgent = useCallback(async (agent: Partial<Agent>): Promise<Agent | null> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/agent-management/agents`;
      const response: APIResponse<Agent> = await apiRequest(url, {
        method: 'POST',
        body: JSON.stringify(agent),
      });
      
      if (response.success && response.data) {
        setAgents(prevAgents => [...prevAgents, response.data!]);
        return response.data;
      } else {
        throw new Error(response.error || 'Failed to create agent');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to create agent:', err);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Update an existing agent
  const updateAgent = useCallback(async (agentId: string, updates: Partial<Agent>): Promise<boolean> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/agent-management/agents/${agentId}`;
      const response: APIResponse = await apiRequest(url, {
        method: 'PUT',
        body: JSON.stringify(updates),
      });
      
      if (response.success) {
        // Update the agent in the local state
        setAgents(prevAgents => 
          prevAgents.map(agent => 
            agent.id === agentId ? { ...agent, ...updates } : agent
          )
        );
        
        if (selectedAgent?.id === agentId) {
          setSelectedAgent(prev => prev ? { ...prev, ...updates } : null);
        }
        
        return true;
      } else {
        throw new Error(response.error || 'Failed to update agent');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to update agent:', err);
      return false;
    } finally {
      setIsLoading(false);
    }
  }, [selectedAgent]);

  // Delete an agent
  const deleteAgent = useCallback(async (agentId: string): Promise<boolean> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/agent-management/agents/${agentId}`;
      const response: APIResponse = await apiRequest(url, { method: 'DELETE' });
      
      if (response.success) {
        setAgents(prevAgents => prevAgents.filter(agent => agent.id !== agentId));
        
        if (selectedAgent?.id === agentId) {
          setSelectedAgent(null);
        }
        
        return true;
      } else {
        throw new Error(response.error || 'Failed to delete agent');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to delete agent:', err);
      return false;
    } finally {
      setIsLoading(false);
    }
  }, [selectedAgent]);

  // Execute an action on an agent
  const executeAgentAction = useCallback(async (agentId: string, action: AgentAction): Promise<boolean> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/agent-management/agents/${agentId}/actions`;
      const response: APIResponse = await apiRequest(url, {
        method: 'POST',
        body: JSON.stringify(action),
      });
      
      if (response.success) {
        // Refresh the agent data after action
        await fetchAgent(agentId);
        await fetchAgents();
        return true;
      } else {
        throw new Error(response.error || 'Failed to execute action');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to execute agent action:', err);
      return false;
    } finally {
      setIsLoading(false);
    }
  }, [fetchAgent, fetchAgents]);

  // Convenience methods for common actions
  const deployAgent = useCallback((agentId: string, parameters?: Record<string, any>) => 
    executeAgentAction(agentId, { action: 'deploy', parameters }), [executeAgentAction]);
  
  const startAgent = useCallback((agentId: string, parameters?: Record<string, any>) => 
    executeAgentAction(agentId, { action: 'start', parameters }), [executeAgentAction]);
  
  const stopAgent = useCallback((agentId: string) => 
    executeAgentAction(agentId, { action: 'stop' }), [executeAgentAction]);
  
  const restartAgent = useCallback((agentId: string, parameters?: Record<string, any>) => 
    executeAgentAction(agentId, { action: 'restart', parameters }), [executeAgentAction]);

  // Fetch agent metrics
  const fetchAgentMetrics = useCallback(async (agentId: string, limit: number = 100) => {
    try {
      const url = `${API_BASE_URL}/api/agent-management/agents/${agentId}/metrics?limit=${limit}`;
      const response: APIResponse<AgentMetrics[]> = await apiRequest(url, { method: 'GET' });
      
      if (response.success && response.data) {
        setAgentMetrics(prev => ({ ...prev, [agentId]: response.data! }));
        return response.data;
      }
    } catch (err) {
      console.error('Failed to fetch agent metrics:', err);
    }
    return [];
  }, []);

  // Fetch agent logs
  const fetchAgentLogs = useCallback(async (agentId: string, limit: number = 100) => {
    try {
      const url = `${API_BASE_URL}/api/agent-management/agents/${agentId}/logs?limit=${limit}`;
      const response: APIResponse<AgentLog[]> = await apiRequest(url, { method: 'GET' });
      
      if (response.success && response.data) {
        setAgentLogs(prev => ({ ...prev, [agentId]: response.data! }));
        return response.data;
      }
    } catch (err) {
      console.error('Failed to fetch agent logs:', err);
    }
    return [];
  }, []);

  // Fetch agent events
  const fetchAgentEvents = useCallback(async (agentId?: string, limit: number = 100) => {
    try {
      const url = agentId 
        ? `${API_BASE_URL}/api/agent-management/agents/${agentId}/events?limit=${limit}`
        : `${API_BASE_URL}/api/agent-management/events?limit=${limit}`;
      const response: APIResponse<AgentEvent[]> = await apiRequest(url, { method: 'GET' });
      
      if (response.success && response.data) {
        setAgentEvents(response.data);
        return response.data;
      }
    } catch (err) {
      console.error('Failed to fetch agent events:', err);
    }
    return [];
  }, []);

  // Fetch agent templates
  const fetchTemplates = useCallback(async () => {
    try {
      const url = `${API_BASE_URL}/api/agent-management/templates`;
      const response: APIResponse<AgentTemplate[]> = await apiRequest(url, { method: 'GET' });
      
      if (response.success && response.data) {
        setTemplates(response.data);
        return response.data;
      }
    } catch (err) {
      console.error('Failed to fetch templates:', err);
    }
    return [];
  }, []);

  // Create agent template
  const createTemplate = useCallback(async (template: Partial<AgentTemplate>): Promise<AgentTemplate | null> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/agent-management/templates`;
      const response: APIResponse<AgentTemplate> = await apiRequest(url, {
        method: 'POST',
        body: JSON.stringify(template),
      });
      
      if (response.success && response.data) {
        setTemplates(prev => [...prev, response.data!]);
        return response.data;
      } else {
        throw new Error(response.error || 'Failed to create template');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to create template:', err);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Fetch agent summary
  const fetchSummary = useCallback(async () => {
    try {
      const url = `${API_BASE_URL}/api/agent-management/summary`;
      const response: APIResponse<AgentSummary> = await apiRequest(url, { method: 'GET' });
      
      if (response.success && response.data) {
        setSummary(response.data);
        return response.data;
      }
    } catch (err) {
      console.error('Failed to fetch summary:', err);
    }
    return null;
  }, []);

  // WebSocket connection management
  const connectWebSocket = useCallback(() => {
    // Set up event handlers
    const handleConnection = (data: { connected: boolean }) => {
      setIsConnected(data.connected);
      if (data.connected) {
        console.log('Agent Management WebSocket connected');
        setError(null);
      } else {
        console.log('Agent Management WebSocket disconnected');
      }
    };

    const handleAgentUpdate = (payload: any) => {
      // Update agent in the list
      setAgents(prevAgents =>
        prevAgents.map(agent =>
          agent.id === payload.id ? { ...agent, ...payload } : agent
        )
      );

      // Update selected agent if it matches
      if (selectedAgent?.id === payload.id) {
        setSelectedAgent(prev => prev ? { ...prev, ...payload } : null);
      }
    };

    const handleAgentStatusChange = (payload: any) => {
      // Update agent status
      setAgents(prevAgents =>
        prevAgents.map(agent =>
          agent.id === payload.agent_id
            ? { ...agent, status: payload.status }
            : agent
        )
      );
    };

    const handleSystemNotification = (payload: any) => {
      console.log('Agent Management system notification:', payload);
    };

    // Register event handlers
    webSocketService.on('connection', handleConnection);
    webSocketService.on('agent-updated', handleAgentUpdate);
    webSocketService.on('agent-status-changed', handleAgentStatusChange);
    webSocketService.on('system-notification', handleSystemNotification);

    // Subscribe to events
    webSocketService.subscribe(['agent-updated', 'agent-status-changed', 'system-notification']);

    // Set initial connection status
    setIsConnected(webSocketService.getConnectionStatus());

    // Return cleanup function
    return () => {
      webSocketService.off('connection', handleConnection);
      webSocketService.off('agent-updated', handleAgentUpdate);
      webSocketService.off('agent-status-changed', handleAgentStatusChange);
      webSocketService.off('system-notification', handleSystemNotification);
    };
  }, [selectedAgent]);

  const disconnectWebSocket = useCallback(() => {
    // Individual hooks don't disconnect the shared service
    setIsConnected(false);
  }, []);

  // Refresh all data
  const refreshAll = useCallback(async () => {
    await Promise.all([
      fetchAgents(),
      fetchTemplates(),
      fetchSummary(),
      fetchAgentEvents(),
    ]);
  }, [fetchAgents, fetchTemplates, fetchSummary, fetchAgentEvents]);

  // Initial fetch and WebSocket connection on mount
  useEffect(() => {
    fetchAgents();
    fetchSummary();
    return connectWebSocket();
  }, [fetchAgents, fetchSummary, connectWebSocket]);

  return {
    agents,
    selectedAgent,
    agentMetrics,
    agentLogs,
    agentEvents,
    templates,
    summary,
    isLoading,
    error,
    isConnected,
    fetchAgents,
    fetchAgent,
    createAgent,
    updateAgent,
    deleteAgent,
    executeAgentAction,
    deployAgent,
    startAgent,
    stopAgent,
    restartAgent,
    fetchAgentMetrics,
    fetchAgentLogs,
    fetchAgentEvents,
    fetchTemplates,
    createTemplate,
    fetchSummary,
    refreshAll,
    connectWebSocket,
    disconnectWebSocket,
    setSelectedAgent,
  };
};

export default useAgentManagement;
