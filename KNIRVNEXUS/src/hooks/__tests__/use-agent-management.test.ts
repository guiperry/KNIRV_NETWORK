import { renderHook, act, waitFor } from '@testing-library/react';
import { useAgentManagement } from '../use-agent-management';
import type { Agent, AgentSummary, AgentMetrics, AgentLog, AgentEvent, AgentTemplate } from '@/types/api';
import { apiRequest } from '@/lib/api';

// Mock the API module
jest.mock('@/lib/api', () => ({
  apiRequest: jest.fn(),
  API_BASE_URL: 'http://localhost:8080',
}));

const mockApiRequest = apiRequest as jest.MockedFunction<typeof apiRequest>;

// Mock WebSocket service
jest.mock('@/lib/websocket-service', () => ({
  webSocketService: {
    connect: jest.fn(),
    disconnect: jest.fn(),
    subscribe: jest.fn(),
    unsubscribe: jest.fn(),
    isConnected: jest.fn().mockReturnValue(true),
    on: jest.fn(),
    off: jest.fn(),
    getConnectionStatus: jest.fn().mockReturnValue(true),
  },
}));

import { webSocketService } from '@/lib/websocket-service';
const mockWebSocketService = webSocketService as jest.Mocked<typeof webSocketService>;

const mockAgents: Agent[] = [
  {
    id: 'agent-1',
    name: 'Test Agent 1',
    description: 'A test agent',
    version: '1.0.0',
    author: 'test-author',
    type: 'WASM',
    status: 'running',
    file_path: '/agents/agent-1.wasm',
    file_size: 1024000,
    file_hash: 'abc123',
    capabilities: ['compute', 'storage'],
    dependencies: [],
    configuration: {},
    metadata: {},
    tags: ['test', 'demo'],
    uploaded_at: '2024-01-01T00:00:00Z',
    last_modified: '2024-01-01T00:00:00Z',
    uploaded_by: 'test-user'
  },
  {
    id: 'agent-2',
    name: 'Test Agent 2',
    description: 'Another test agent',
    version: '2.0.0',
    author: 'test-author-2',
    type: 'LoRA',
    status: 'stopped',
    file_path: '/agents/agent-2.wasm',
    file_size: 2048000,
    file_hash: 'def456',
    capabilities: ['ai', 'nlp'],
    dependencies: [],
    configuration: {},
    metadata: {},
    tags: ['test', 'experimental'],
    uploaded_at: '2024-01-02T00:00:00Z',
    last_modified: '2024-01-02T00:00:00Z',
    uploaded_by: 'test-user-2'
  }
];

const mockSummary: AgentSummary = {
  total_agents: 10,
  running_agents: 3,
  deployed_agents: 2,
  stopped_agents: 4,
  error_agents: 1,
  uploaded_agents: 0
};

const mockMetrics: AgentMetrics = {
  agent_id: 'agent-1',
  cpu_usage: 45.2,
  memory_usage: 256,
  disk_usage: 512,
  network_in: 1024000,
  network_out: 2048000,
  requests_per_minute: 100,
  errors_per_minute: 2,
  uptime_seconds: 3600,
  last_updated: '2024-01-01T00:00:00Z'
};

const mockLogs: AgentLog[] = [
  {
    id: 'log-1',
    agent_id: 'agent-1',
    level: 'info',
    message: 'Agent started successfully',
    timestamp: '2024-01-01T00:00:00Z',
    source: 'runtime'
  },
  {
    id: 'log-2',
    agent_id: 'agent-1',
    level: 'error',
    message: 'Connection timeout',
    timestamp: '2024-01-01T00:01:00Z',
    source: 'network'
  }
];

const mockEvents: AgentEvent[] = [
  {
    id: 'event-1',
    agent_id: 'agent-1',
    type: 'deployment',
    status: 'success',
    message: 'Agent deployed successfully',
    timestamp: '2024-01-01T00:00:00Z',
    metadata: { deployment_id: 'deploy-1' }
  },
  {
    id: 'event-2',
    agent_id: 'agent-1',
    type: 'execution',
    status: 'completed',
    message: 'Task completed',
    timestamp: '2024-01-01T00:02:00Z',
    metadata: { task_id: 'task-1', duration: 120 }
  }
];

const mockTemplates: AgentTemplate[] = [
  {
    id: 'template-1',
    name: 'Basic WASM Template',
    description: 'A basic WASM agent template',
    type: 'WASM',
    version: '1.0.0',
    author: 'KNIRV Team',
    config: {
      memory_limit: 512,
      cpu_limit: 1.0,
      timeout: 30000
    },
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z'
  }
];

describe('useAgentManagement Hook', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockApiRequest.mockClear();
    mockWebSocketService.connect.mockClear();
    mockWebSocketService.disconnect.mockClear();
    mockWebSocketService.subscribe.mockClear();
    mockWebSocketService.unsubscribe.mockClear();
    mockWebSocketService.isConnected.mockReturnValue(true);

    // Set up default mock responses for initial API calls that happen on mount
    mockApiRequest.mockImplementation((url) => {
      if (url.includes('/summary')) {
        return Promise.resolve({ success: true, data: null });
      }
      if (url.includes('/agents') && !url.includes('/actions') && !url.includes('/metrics') && !url.includes('/logs') && !url.includes('/events')) {
        return Promise.resolve({ success: true, data: [] });
      }
      if (url.includes('/templates')) {
        return Promise.resolve({ success: true, data: [] });
      }
      return Promise.resolve({ success: true, data: [] });
    });
  });

  it('initializes with default state', async () => {
    // Set up specific mock responses for initial API calls
    mockApiRequest.mockImplementation((url) => {
      if (url.includes('/summary')) {
        return Promise.resolve({ success: true, data: null });
      }
      return Promise.resolve({ success: true, data: [] });
    });

    const { result } = renderHook(() => useAgentManagement());

    // Wait for initial loading to complete
    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.agents).toEqual([]);
    expect(result.current.selectedAgent).toBeNull();
    expect(result.current.agentMetrics).toEqual({});
    expect(result.current.agentLogs).toEqual({});
    expect(result.current.agentEvents).toEqual([]);
    expect(result.current.templates).toEqual([]);
    expect(result.current.summary).toBeNull();
    expect(result.current.error).toBeNull();
    expect(result.current.isConnected).toBe(true);
  });

  it('fetches agents successfully', async () => {
    // Override the default mock for this specific test
    mockApiRequest.mockImplementation((url) => {
      if (url.includes('/summary')) {
        return Promise.resolve({ success: true, data: null });
      }
      if (url.includes('/agents') && !url.includes('/actions') && !url.includes('/metrics') && !url.includes('/logs') && !url.includes('/events')) {
        return Promise.resolve({ success: true, data: mockAgents });
      }
      if (url.includes('/templates')) {
        return Promise.resolve({ success: true, data: [] });
      }
      return Promise.resolve({ success: true, data: [] });
    });

    const { result } = renderHook(() => useAgentManagement());

    // Wait for initial loading to complete
    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8080/api/agent-management/agents', { method: 'GET' });
    expect(result.current.agents).toEqual(mockAgents);
    expect(result.current.error).toBeNull();
  });

  it('handles fetch agents error', async () => {
    const errorMessage = 'Failed to fetch agents';
    mockApiRequest.mockRejectedValueOnce(new Error(errorMessage));

    const { result } = renderHook(() => useAgentManagement());

    await act(async () => {
      await result.current.fetchAgents();
    });

    expect(result.current.agents).toEqual([]);
    expect(result.current.error).toBe(errorMessage);
  });

  it('fetches single agent successfully', async () => {
    mockApiRequest.mockResolvedValueOnce({
      success: true,
      data: mockAgents[0]
    });

    const { result } = renderHook(() => useAgentManagement());

    await act(async () => {
      await result.current.fetchAgent('agent-1');
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8080/api/agent-management/agents/agent-1', { method: 'GET' });
    expect(result.current.selectedAgent).toEqual(mockAgents[0]);
  });

  it('creates agent successfully', async () => {
    const { result } = renderHook(() => useAgentManagement());

    // Wait for initial loading to complete so state is properly initialized
    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    const newAgent = { ...mockAgents[0], id: 'agent-3', name: 'New Agent' };
    mockApiRequest.mockResolvedValueOnce({
      success: true,
      data: newAgent
    });

    const agentData = {
      name: 'New Agent',
      description: 'A new test agent',
      type: 'WASM',
      file: new File(['test'], 'test.wasm'),
      config: {}
    };

    let createdAgent: Agent | null = null;
    await act(async () => {
      createdAgent = await result.current.createAgent(agentData);
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8080/api/agent-management/agents', {
      method: 'POST',
      body: JSON.stringify(agentData)
    });
    expect(createdAgent).toEqual(newAgent);
  });

  it('updates agent successfully', async () => {
    const updatedAgent = { ...mockAgents[0], name: 'Updated Agent' };
    mockApiRequest.mockResolvedValueOnce({
      success: true,
      data: updatedAgent
    });

    const { result } = renderHook(() => useAgentManagement());

    const updateData = { name: 'Updated Agent' };

    let updated: boolean = false;
    await act(async () => {
      updated = await result.current.updateAgent('agent-1', updateData);
    });

    expect(mockApiRequest).toHaveBeenCalledWith('/api/agents/agent-1', {
      method: 'PUT',
      body: JSON.stringify(updateData)
    });
    expect(updated).toBe(true);
  });

  it('deletes agent successfully', async () => {
    mockApiRequest.mockResolvedValueOnce({
      success: true
    });

    const { result } = renderHook(() => useAgentManagement());

    let deleted: boolean = false;
    await act(async () => {
      deleted = await result.current.deleteAgent('agent-1');
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8080/api/agent-management/agents/agent-1', {
      method: 'DELETE'
    });
    expect(deleted).toBe(true);
  });

  it('executes agent action successfully', async () => {
    mockApiRequest.mockResolvedValueOnce({
      success: true,
      data: { action_id: 'action-1', status: 'completed' }
    });

    const { result } = renderHook(() => useAgentManagement());

    const actionData = {
      action: 'start' as const,
      parameters: { timeout: 30000 }
    };

    let executed: boolean = false;
    await act(async () => {
      executed = await result.current.executeAgentAction('agent-1', actionData);
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8080/api/agent-management/agents/agent-1/actions', {
      method: 'POST',
      body: JSON.stringify(actionData)
    });
    expect(executed).toBe(true);
  });

  it('deploys agent successfully', async () => {
    mockApiRequest.mockResolvedValueOnce({
      success: true,
      data: { deployment_id: 'deploy-1' }
    });

    const { result } = renderHook(() => useAgentManagement());

    const deploymentConfig = {
      environment: 'production',
      resources: { memory: 512, cpu: 1.0 }
    };

    let deployed: boolean = false;
    await act(async () => {
      deployed = await result.current.deployAgent('agent-1', deploymentConfig);
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8080/api/agent-management/agents/agent-1/actions', {
      method: 'POST',
      body: JSON.stringify({ action: 'deploy', parameters: deploymentConfig })
    });
    expect(deployed).toBe(true);
  });

  it('starts agent successfully', async () => {
    mockApiRequest.mockResolvedValueOnce({
      success: true
    });

    const { result } = renderHook(() => useAgentManagement());

    let started: boolean = false;
    await act(async () => {
      started = await result.current.startAgent('agent-1');
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8080/api/agent-management/agents/agent-1/actions', {
      method: 'POST',
      body: JSON.stringify({ action: 'start' })
    });
    expect(started).toBe(true);
  });

  it('stops agent successfully', async () => {
    mockApiRequest.mockResolvedValueOnce({
      success: true
    });

    const { result } = renderHook(() => useAgentManagement());

    let stopped: boolean = false;
    await act(async () => {
      stopped = await result.current.stopAgent('agent-1');
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8080/api/agent-management/agents/agent-1/actions', {
      method: 'POST',
      body: JSON.stringify({ action: 'stop' })
    });
    expect(stopped).toBe(true);
  });

  it('restarts agent successfully', async () => {
    mockApiRequest.mockResolvedValueOnce({
      success: true
    });

    const { result } = renderHook(() => useAgentManagement());

    let restarted: boolean = false;
    await act(async () => {
      restarted = await result.current.restartAgent('agent-1');
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8080/api/agent-management/agents/agent-1/actions', {
      method: 'POST',
      body: JSON.stringify({ action: 'restart' })
    });
    expect(restarted).toBe(true);
  });

  it('fetches agent metrics successfully', async () => {
    mockApiRequest.mockResolvedValueOnce({
      success: true,
      data: mockMetrics
    });

    const { result } = renderHook(() => useAgentManagement());

    await act(async () => {
      await result.current.fetchAgentMetrics('agent-1');
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8080/api/agent-management/agents/agent-1/metrics?limit=100', { method: 'GET' });
    expect(result.current.agentMetrics).toEqual(mockMetrics);
  });

  it('fetches agent logs successfully', async () => {
    mockApiRequest.mockResolvedValueOnce({
      success: true,
      data: mockLogs
    });

    const { result } = renderHook(() => useAgentManagement());

    await act(async () => {
      await result.current.fetchAgentLogs('agent-1');
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8080/api/agent-management/agents/agent-1/logs?limit=100', { method: 'GET' });
    expect(result.current.agentLogs).toEqual(mockLogs);
  });

  it('fetches agent events successfully', async () => {
    mockApiRequest.mockResolvedValueOnce({
      success: true,
      data: mockEvents
    });

    const { result } = renderHook(() => useAgentManagement());

    await act(async () => {
      await result.current.fetchAgentEvents('agent-1');
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8080/api/agent-management/agents/agent-1/events?limit=100', { method: 'GET' });
    expect(result.current.agentEvents).toEqual(mockEvents);
  });

  it('fetches templates successfully', async () => {
    mockApiRequest.mockResolvedValueOnce({
      success: true,
      data: mockTemplates
    });

    const { result } = renderHook(() => useAgentManagement());

    await act(async () => {
      await result.current.fetchTemplates();
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8080/api/agent-management/templates', { method: 'GET' });
    expect(result.current.templates).toEqual(mockTemplates);
  });

  it('creates template successfully', async () => {
    const newTemplate = { ...mockTemplates[0], id: 'template-2', name: 'New Template' };
    mockApiRequest.mockResolvedValueOnce({
      success: true,
      data: newTemplate
    });

    const { result } = renderHook(() => useAgentManagement());

    const templateData = {
      name: 'New Template',
      description: 'A new template',
      type: 'WASM',
      config: {}
    };

    let created: boolean = false;
    await act(async () => {
      created = await result.current.createTemplate(templateData);
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8080/api/agent-management/templates', {
      method: 'POST',
      body: JSON.stringify(templateData)
    });
    expect(created).toBe(true);
  });

  it('fetches summary successfully', async () => {
    mockApiRequest.mockResolvedValueOnce({
      success: true,
      data: mockSummary
    });

    const { result } = renderHook(() => useAgentManagement());

    await act(async () => {
      await result.current.fetchSummary();
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8080/api/agent-management/summary', { method: 'GET' });
    expect(result.current.summary).toEqual(mockSummary);
  });

  it('refreshes all data successfully', async () => {
    mockApiRequest
      .mockResolvedValueOnce({ success: true, data: mockAgents })
      .mockResolvedValueOnce({ success: true, data: mockSummary });

    const { result } = renderHook(() => useAgentManagement());

    await act(async () => {
      await result.current.refreshAll();
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8080/api/agent-management/agents', { method: 'GET' });
    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8080/api/agent-management/summary', { method: 'GET' });
    expect(result.current.agents).toEqual(mockAgents);
    expect(result.current.summary).toEqual(mockSummary);
  });

  it('connects to WebSocket successfully', async () => {
    mockWebSocketService.connect.mockResolvedValueOnce(undefined);
    mockWebSocketService.isConnected.mockReturnValue(true);

    const { result } = renderHook(() => useAgentManagement());

    await act(async () => {
      await result.current.connectWebSocket();
    });

    expect(mockWebSocketService.connect).toHaveBeenCalled();
    expect(mockWebSocketService.subscribe).toHaveBeenCalledWith('agents', expect.any(Function));
    expect(result.current.isConnected).toBe(true);
  });

  it('disconnects from WebSocket successfully', () => {
    const { result } = renderHook(() => useAgentManagement());

    act(() => {
      result.current.disconnectWebSocket();
    });

    expect(mockWebSocketService.unsubscribe).toHaveBeenCalledWith('agents');
    expect(mockWebSocketService.disconnect).toHaveBeenCalled();
  });

  it('sets selected agent', () => {
    const { result } = renderHook(() => useAgentManagement());

    act(() => {
      result.current.setSelectedAgent(mockAgents[0]);
    });

    expect(result.current.selectedAgent).toEqual(mockAgents[0]);
  });

  it('handles loading states correctly', async () => {
    let resolvePromise: (value: any) => void;
    const promise = new Promise((resolve) => {
      resolvePromise = resolve;
    });
    mockApiRequest.mockReturnValueOnce(promise);

    const { result } = renderHook(() => useAgentManagement());

    // Start async operation
    act(() => {
      result.current.fetchAgents();
    });

    // Should be loading
    expect(result.current.isLoading).toBe(true);

    // Resolve the promise
    await act(async () => {
      resolvePromise!({ success: true, data: mockAgents });
      await promise;
    });

    // Should no longer be loading
    expect(result.current.isLoading).toBe(false);
  });
});
