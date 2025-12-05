import { renderHook, act, waitFor } from '@testing-library/react';
import { useModelManagement } from '../use-model-management';
import type { Model, ModelSummary, ModelMetrics, ModelLog, ModelEvent, ModelTemplate, APIResponse, ModelResourceLimits } from '@/types/api';
import { apiRequest } from '@/lib/api';

// Mock the API module
jest.mock('@/lib/api', () => ({
  apiRequest: jest.fn(),
  API_BASE_URL: 'http://localhost:8082',
}));

const mockApiRequest = apiRequest as jest.MockedFunction<typeof apiRequest>;

// Helper function to create mock API responses with timestamp
const createMockResponse = <T>(data: T, success: boolean = true, error?: string): APIResponse<T> => ({
  success,
  data,
  timestamp: new Date().toISOString(),
  ...(error && { error })
});

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
const mockWebSocketService = webSocketService as any;

const mockModels: Model[] = [
  {
    id: 'model-1',
    name: 'Test Model 1',
    description: 'A test model',
    version: '1.0.0',
    author: 'test-author',
    type: 'WASM',
    status: 'running',
    file_path: '/models/model-1.wasm',
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
    id: 'model-2',
    name: 'Test Model 2',
    description: 'Another test model',
    version: '2.0.0',
    author: 'test-author-2',
    type: 'LoRA',
    status: 'stopped',
    file_path: '/models/model-2.wasm',
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

const mockSummary: ModelSummary = {
  total_models: 10,
  running_models: 3,
  deployed_models: 2,
  stopped_models: 4,
  error_models: 1,
  uploaded_models: 0
};

const mockMetrics: ModelMetrics = {
  model_id: 'model-1',
  timestamp: '2024-01-01T00:00:00Z',
  cpu_usage_percent: 45.2,
  memory_usage_mb: 256,
  network_in_mbps: 1024,
  network_out_mbps: 2048,
  disk_read_mbps: 512,
  disk_write_mbps: 256,
  requests_per_second: 100,
  errors_per_second: 2,
  response_time_ms: 150,
};

const mockLogs: ModelLog[] = [
  {
    id: 'log-1',
    model_id: 'model-1',
    level: 'info',
    message: 'Model started successfully',
    timestamp: '2024-01-01T00:00:00Z',
    source: 'runtime',
    metadata: {}
  },
  {
    id: 'log-2',
    model_id: 'model-1',
    level: 'error',
    message: 'Connection timeout',
    timestamp: '2024-01-01T00:01:00Z',
    source: 'network',
    metadata: {}
  }
];

const mockEvents: ModelEvent[] = [
  {
    id: 'event-1',
    model_id: 'model-1',
    type: 'deployment',
    description: 'Model deployed successfully',
    timestamp: '2024-01-01T00:00:00Z',
    metadata: { deployment_id: 'deploy-1' }
  },
  {
    id: 'event-2',
    model_id: 'model-1',
    type: 'execution',
    description: 'Task completed',
    timestamp: '2024-01-01T00:02:00Z',
    metadata: { task_id: 'task-1', duration: 120 }
  }
];

const mockResourceLimits: ModelResourceLimits = {
  max_memory_mb: 512,
  max_cpu_percent: 100,
  max_execution_time_seconds: 300,
  max_concurrency: 10,
  max_disk_mb: 1024,
  network_access: true,
  file_system_access: false,
};

const mockTemplates: ModelTemplate[] = [
  {
    id: 'template-1',
    name: 'Basic WASM Template',
    description: 'A basic WASM model template',
    type: 'WASM',
    default_configuration: {
      memory_limit: 512,
      cpu_limit: 1.0,
      timeout: 30000
    },
    default_resource_limits: mockResourceLimits,
    created_at: '2024-01-01T00:00:00Z',
    created_by: 'KNIRV Team'
  }
];

describe('useModelManagement Hook', () => {
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
        return Promise.resolve(createMockResponse(null));
      }
      if (url.includes('/models') && !url.includes('/actions') && !url.includes('/metrics') && !url.includes('/logs') && !url.includes('/events')) {
        return Promise.resolve(createMockResponse([]));
      }
      if (url.includes('/templates')) {
        return Promise.resolve(createMockResponse([]));
      }
      return Promise.resolve(createMockResponse([]));
    });
  });

  it('initializes with default state', async () => {
    // Set up specific mock responses for initial API calls
    mockApiRequest.mockImplementation((url) => {
      if (url.includes('/summary')) {
        return Promise.resolve(createMockResponse(null));
      }
      return Promise.resolve(createMockResponse([]));
    });

    const { result } = renderHook(() => useModelManagement());

    // Wait for initial loading to complete
    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.models).toEqual([]);
    expect(result.current.selectedModel).toBeNull();
    expect(result.current.modelMetrics).toEqual({});
    expect(result.current.modelLogs).toEqual({});
    expect(result.current.modelEvents).toEqual([]);
    expect(result.current.templates).toEqual([]);
    expect(result.current.summary).toBeNull();
    expect(result.current.error).toBeNull();
    expect(result.current.isConnected).toBe(true);
  });

  it('fetches models successfully', async () => {
    // Clear the default mock and set up specific mock before renderHook
    mockApiRequest.mockClear();
    mockApiRequest.mockImplementation((url) => {
      if (url.includes('/summary')) {
        return Promise.resolve(createMockResponse(null));
      }
      if (url.includes('/models') && !url.includes('/actions') && !url.includes('/metrics') && !url.includes('/logs') && !url.includes('/events')) {
        return Promise.resolve(createMockResponse(mockModels));
      }
      if (url.includes('/templates')) {
        return Promise.resolve(createMockResponse([]));
      }
      return Promise.resolve(createMockResponse([]));
    });

    const { result } = renderHook(() => useModelManagement());

    // Wait for initial loading to complete
    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8082/api/model-management/objects', { method: 'GET' });
    expect(result.current.models).toEqual(mockModels);
    expect(result.current.error).toBeNull();
  });

  it('handles fetch models error', async () => {
    const errorMessage = 'Failed to fetch models';
    mockApiRequest.mockRejectedValueOnce(new Error(errorMessage));

    const { result } = renderHook(() => useModelManagement());

    await act(async () => {
      await result.current.fetchModels();
    });

    expect(result.current.models).toEqual([]);
    expect(result.current.error).toBe(errorMessage);
  });

  it('fetches single model successfully', async () => {
    mockApiRequest.mockResolvedValueOnce(createMockResponse(mockModels[0]));

    const { result } = renderHook(() => useModelManagement());

    await act(async () => {
      await result.current.fetchModel('model-1');
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8082/api/model-management/objects/model-1', { method: 'GET' });
    expect(result.current.selectedModel).toEqual(mockModels[0]);
  });

  it('creates model successfully', async () => {
    const { result } = renderHook(() => useModelManagement());

    // Wait for initial loading to complete so state is properly initialized
    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    const newModel = { ...mockModels[0], id: 'model-3', name: 'New Model' };
    mockApiRequest.mockResolvedValueOnce(createMockResponse(newModel));

    const modelData = {
      name: 'New Model',
      description: 'A new test model',
      type: 'WASM' as const,
      file: new File(['test'], 'test.wasm'),
      config: {}
    };

    let createdModel: Model | null = null;
    await act(async () => {
      createdModel = await result.current.createModel(modelData);
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8082/api/model-management/objects', {
      method: 'POST',
      body: JSON.stringify(modelData)
    });
    expect(createdModel).toEqual(newModel);
  });

  it('updates model successfully', async () => {
    const updatedModel = { ...mockModels[0], name: 'Updated Model' };
    mockApiRequest.mockResolvedValueOnce(createMockResponse(updatedModel));

    const { result } = renderHook(() => useModelManagement());

    const updateData = { name: 'Updated Model' };

    let updated: boolean = false;
    await act(async () => {
      updated = await result.current.updateModel('model-1', updateData);
    });

    expect(mockApiRequest).toHaveBeenCalledWith('/api/models/model-1', {
      method: 'PUT',
      body: JSON.stringify(updateData)
    });
    expect(updated).toBe(true);
  });

  it('deletes model successfully', async () => {
    mockApiRequest.mockResolvedValueOnce(createMockResponse(null));

    const { result } = renderHook(() => useModelManagement());

    let deleted: boolean = false;
    await act(async () => {
      deleted = await result.current.deleteModel('model-1');
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8082/api/model-management/objects/model-1', {
      method: 'DELETE'
    });
    expect(deleted).toBe(true);
  });

  it('executes model action successfully', async () => {
    mockApiRequest.mockResolvedValueOnce(createMockResponse({ action_id: 'action-1', status: 'completed' }));

    const { result } = renderHook(() => useModelManagement());

    const actionData = {
      action: 'start' as const,
      parameters: { timeout: 30000 }
    };

    let executed: boolean = false;
    await act(async () => {
      executed = await result.current.executeModelAction('model-1', actionData);
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8082/api/model-management/objects/model-1/actions', {
      method: 'POST',
      body: JSON.stringify(actionData)
    });
    expect(executed).toBe(true);
  });

  it('deploys model successfully', async () => {
    mockApiRequest.mockResolvedValueOnce(createMockResponse({ deployment_id: 'deploy-1' }));

    const { result } = renderHook(() => useModelManagement());

    const deploymentConfig = {
      environment: 'production',
      resources: { memory: 512, cpu: 1.0 }
    };

    let deployed: boolean = false;
    await act(async () => {
      deployed = await result.current.deployModel('model-1', deploymentConfig);
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8082/api/model-management/objects/model-1/actions', {
      method: 'POST',
      body: JSON.stringify({ action: 'deploy', parameters: deploymentConfig })
    });
    expect(deployed).toBe(true);
  });

  it('starts model successfully', async () => {
    mockApiRequest.mockResolvedValueOnce(createMockResponse(null));

    const { result } = renderHook(() => useModelManagement());

    let started: boolean = false;
    await act(async () => {
      started = await result.current.startModel('model-1');
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8082/api/model-management/objects/model-1/actions', {
      method: 'POST',
      body: JSON.stringify({ action: 'start' })
    });
    expect(started).toBe(true);
  });

  it('stops model successfully', async () => {
    mockApiRequest.mockResolvedValueOnce(createMockResponse(null));

    const { result } = renderHook(() => useModelManagement());

    let stopped: boolean = false;
    await act(async () => {
      stopped = await result.current.stopModel('model-1');
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8082/api/model-management/objects/model-1/actions', {
      method: 'POST',
      body: JSON.stringify({ action: 'stop' })
    });
    expect(stopped).toBe(true);
  });

  it('restarts model successfully', async () => {
    mockApiRequest.mockResolvedValueOnce(createMockResponse(null));

    const { result } = renderHook(() => useModelManagement());

    let restarted: boolean = false;
    await act(async () => {
      restarted = await result.current.restartModel('model-1');
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8082/api/model-management/objects/model-1/actions', {
      method: 'POST',
      body: JSON.stringify({ action: 'restart' })
    });
    expect(restarted).toBe(true);
  });

  it('fetches model metrics successfully', async () => {
    mockApiRequest.mockResolvedValueOnce(createMockResponse(mockMetrics));

    const { result } = renderHook(() => useModelManagement());

    await act(async () => {
      await result.current.fetchModelMetrics('model-1');
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8082/api/model-management/objects/model-1/metrics?limit=100', { method: 'GET' });
    expect(result.current.modelMetrics).toEqual(mockMetrics);
  });

  it('fetches model logs successfully', async () => {
    mockApiRequest.mockResolvedValueOnce(createMockResponse(mockLogs));

    const { result } = renderHook(() => useModelManagement());

    await act(async () => {
      await result.current.fetchModelLogs('model-1');
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8082/api/model-management/objects/model-1/logs?limit=100', { method: 'GET' });
    expect(result.current.modelLogs).toEqual(mockLogs);
  });

  it('fetches model events successfully', async () => {
    mockApiRequest.mockResolvedValueOnce(createMockResponse(mockEvents));

    const { result } = renderHook(() => useModelManagement());

    await act(async () => {
      await result.current.fetchModelEvents('model-1');
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8082/api/model-management/objects/model-1/events?limit=100', { method: 'GET' });
    expect(result.current.modelEvents).toEqual(mockEvents);
  });

  it('fetches templates successfully', async () => {
    mockApiRequest.mockResolvedValueOnce(createMockResponse(mockTemplates));

    const { result } = renderHook(() => useModelManagement());

    await act(async () => {
      await result.current.fetchTemplates();
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8082/api/model-management/templates', { method: 'GET' });
    expect(result.current.templates).toEqual(mockTemplates);
  });

  it('creates template successfully', async () => {
    const newTemplate = { ...mockTemplates[0], id: 'template-2', name: 'New Template' };
    mockApiRequest.mockResolvedValueOnce(createMockResponse(newTemplate));

    const { result } = renderHook(() => useModelManagement());

    const templateData = {
      name: 'New Template',
      description: 'A new template',
      type: 'WASM' as const,
      config: {}
    };

    let created: ModelTemplate | null = null;
    await act(async () => {
      created = await result.current.createTemplate(templateData);
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8082/api/model-management/templates', {
      method: 'POST',
      body: JSON.stringify(templateData)
    });
    expect(created).toEqual(newTemplate);
  });

  it('fetches summary successfully', async () => {
    mockApiRequest.mockResolvedValueOnce(createMockResponse(mockSummary));

    const { result } = renderHook(() => useModelManagement());

    await act(async () => {
      await result.current.fetchSummary();
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8082/api/model-management/summary', { method: 'GET' });
    expect(result.current.summary).toEqual(mockSummary);
  });

  it('refreshes all data successfully', async () => {
    mockApiRequest
      .mockResolvedValueOnce(createMockResponse(mockModels))
      .mockResolvedValueOnce(createMockResponse(mockSummary));

    const { result } = renderHook(() => useModelManagement());

    await act(async () => {
      await result.current.refreshAll();
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8082/api/model-management/objects', { method: 'GET' });
    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8082/api/model-management/summary', { method: 'GET' });
    expect(result.current.models).toEqual(mockModels);
    expect(result.current.summary).toEqual(mockSummary);
  });

  it('connects to WebSocket successfully', async () => {
    mockWebSocketService.connect.mockResolvedValueOnce(undefined);
    mockWebSocketService.isConnected.mockReturnValue(true);

    const { result } = renderHook(() => useModelManagement());

    await act(async () => {
      await result.current.connectWebSocket();
    });

    expect(mockWebSocketService.connect).toHaveBeenCalled();
    expect(mockWebSocketService.subscribe).toHaveBeenCalledWith('models', expect.any(Function));
    expect(result.current.isConnected).toBe(true);
  });

  it('disconnects from WebSocket successfully', () => {
    const { result } = renderHook(() => useModelManagement());

    act(() => {
      result.current.disconnectWebSocket();
    });

    expect(mockWebSocketService.unsubscribe).toHaveBeenCalledWith('models');
    expect(mockWebSocketService.disconnect).toHaveBeenCalled();
  });

  it('sets selected model', () => {
    const { result } = renderHook(() => useModelManagement());

    act(() => {
      result.current.setSelectedModel(mockModels[0]);
    });

    expect(result.current.selectedModel).toEqual(mockModels[0]);
  });

  it('handles loading states correctly', async () => {
    let resolvePromise: (value: APIResponse<Model[]>) => void;
    const promise = new Promise<APIResponse<Model[]>>((resolve) => {
      resolvePromise = resolve;
    });
    mockApiRequest.mockReturnValueOnce(promise);

    const { result } = renderHook(() => useModelManagement());

    // Start async operation
    act(() => {
      result.current.fetchModels();
    });

    // Should be loading
    expect(result.current.isLoading).toBe(true);

    // Resolve the promise
    await act(async () => {
      resolvePromise!(createMockResponse(mockModels));
      await promise;
    });

    // Should no longer be loading
    expect(result.current.isLoading).toBe(false);
  });
});
