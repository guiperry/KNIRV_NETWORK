import { renderHook, act, waitFor } from '@testing-library/react';
import { useDVENodes } from '../use-dve-nodes';
import type { DVENode, DVENodeFilter, RegisterNodeRequest } from '@/types/api';

// Mock the API module
jest.mock('@/lib/api', () => ({
  apiRequest: jest.fn(),
  API_BASE_URL: 'http://localhost:8082',
  buildQueryString: jest.fn((params) => {
    const searchParams = new URLSearchParams();
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined && value !== null) {
        searchParams.append(key, String(value));
      }
    });
    const queryString = searchParams.toString();
    return queryString ? `?${queryString}` : '';
  }),
}));

// Mock the websocket service
jest.mock('@/lib/websocket-service', () => ({
  webSocketService: {
    connect: jest.fn(),
    disconnect: jest.fn(),
    send: jest.fn(),
    onMessage: jest.fn(),
    onOpen: jest.fn(),
    onClose: jest.fn(),
    onError: jest.fn(),
    getConnectionStatus: jest.fn(() => false),
    on: jest.fn(),
    off: jest.fn(),
    emit: jest.fn(),
    subscribe: jest.fn(),
    unsubscribe: jest.fn(),
  },
}));

import { apiRequest } from '@/lib/api';

const mockApiRequest = apiRequest as jest.MockedFunction<typeof apiRequest>;

describe('useDVENodes Hook', () => {
  const mockNode: DVENode = {
    id: 'node-1',
    name: 'Test Node',
    status: 'online',
    capabilities: ['compute', 'storage'],
    stake_amount: 1000,
    location: 'US-East',
    last_heartbeat: new Date().toISOString(),
    tee_type: 'sgx',
    reputation_score: 95,
    ip_address: '192.168.1.1',
    public_key: 'test-public-key',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    cpu_usage: 50.5,
    memory_usage: 75.2,
    network_latency: 10,
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('should initialize with default state', () => {
    const { result } = renderHook(() => useDVENodes());

    expect(result.current.nodes).toEqual([]);
    expect(result.current.isLoading).toBe(true); // Initially loading due to useEffect
    expect(result.current.error).toBe(null);
    expect(result.current.isConnected).toBe(false);
  });

  it('should fetch nodes successfully', async () => {
    const mockResponse = {
      success: true,
      data: [mockNode],
      timestamp: new Date().toISOString(),
    };

    mockApiRequest.mockResolvedValue(mockResponse);

    const { result } = renderHook(() => useDVENodes());

    await act(async () => {
      await result.current.fetchNodes();
    });

    expect(result.current.nodes).toEqual([mockNode]);
    expect(result.current.isLoading).toBe(false);
    expect(result.current.error).toBe(null);
    expect(mockApiRequest).toHaveBeenCalledWith(
      'http://localhost:8082/api/dve-nodes',
      { method: 'GET' }
    );
  });

  it('should fetch nodes with filter', async () => {
    const mockResponse = {
      success: true,
      data: [mockNode],
      timestamp: new Date().toISOString(),
    };

    const filter: DVENodeFilter = {
      status: 'online',
      min_stake: 500,
    };

    mockApiRequest.mockResolvedValue(mockResponse);

    const { result } = renderHook(() => useDVENodes());

    await act(async () => {
      await result.current.fetchNodes(filter);
    });

    expect(mockApiRequest).toHaveBeenCalledWith(
      'http://localhost:8082/api/dve-nodes?status=online&min_stake=500',
      { method: 'GET' }
    );
  });

  it('should handle fetch nodes error', async () => {
    const errorMessage = 'Failed to fetch nodes';
    mockApiRequest.mockRejectedValue(new Error(errorMessage));

    const { result } = renderHook(() => useDVENodes());

    await act(async () => {
      await result.current.fetchNodes();
    });

    expect(result.current.nodes).toEqual([]);
    expect(result.current.isLoading).toBe(false);
    expect(result.current.error).toBe(errorMessage);
  });

  it('should get specific node by ID', async () => {
    const mockResponse = {
      success: true,
      data: mockNode,
      timestamp: new Date().toISOString(),
    };

    mockApiRequest.mockResolvedValue(mockResponse);

    const { result } = renderHook(() => useDVENodes());

    let nodeResult: DVENode | null = null;

    await act(async () => {
      nodeResult = await result.current.getNode('node-1');
    });

    expect(nodeResult).toEqual(mockNode);
    expect(mockApiRequest).toHaveBeenCalledWith(
      'http://localhost:8082/api/dve-nodes/node-1',
      { method: 'GET' }
    );
  });

  it('should handle get node error', async () => {
    mockApiRequest.mockRejectedValue(new Error('Node not found'));

    const { result } = renderHook(() => useDVENodes());

    let nodeResult: DVENode | null = null;

    await act(async () => {
      nodeResult = await result.current.getNode('invalid-id');
    });

    expect(nodeResult).toBe(null);
    expect(result.current.error).toBe('Node not found');
  });

  it('should register new node', async () => {
    const registerRequest: RegisterNodeRequest = {
      name: 'New Node',
      capabilities: ['compute'],
      stake_amount: 1000,
      location: 'US-West',
      tee_type: 'software', // Added tee_type
    };

    const mockResponse = {
      success: true,
      data: {
        ...mockNode,
        name: 'New Node',
        capabilities: ['compute'],
        stake_amount: 1000,
        location: 'US-West',
        lastSeen: '2025-09-16T17:42:02.578Z',
        performance: {
          cpu: 80,
          memory: 60,
          storage: 40,
          network: 90,
        },
        earnings: {
          total: 500,
          thisMonth: 50,
          lastMonth: 45,
        }
      },
      timestamp: new Date().toISOString(),
    };

    mockApiRequest.mockResolvedValue(mockResponse);

    const { result } = renderHook(() => useDVENodes());

    let registeredNode: DVENode | null = null;

    await act(async () => {
      registeredNode = await result.current.registerNode(registerRequest);
    });

    expect(registeredNode).toEqual(mockResponse.data);
    expect(mockApiRequest).toHaveBeenCalledWith(
      'http://localhost:8082/api/dve-nodes',
      {
        method: 'POST',
        body: JSON.stringify(registerRequest),
      }
    );
  });

  it('should handle register node error', async () => {
    const registerRequest: RegisterNodeRequest = {
      name: 'New Node',
      capabilities: ['compute'],
      stake_amount: 1000,
      location: 'US-West',
      tee_type: 'software', // Added tee_type
    };

    mockApiRequest.mockRejectedValue(new Error('Registration failed'));

    const { result } = renderHook(() => useDVENodes());

    let registeredNode: DVENode | null = null;

    await act(async () => {
      registeredNode = await result.current.registerNode(registerRequest);
    });

    expect(registeredNode).toBe(null);
    expect(result.current.error).toBe('Registration failed');
  });

  it('should update node status', async () => {
    const mockResponse = {
      success: true,
      data: { ...mockNode, status: 'inactive' as const },
      timestamp: new Date().toISOString(),
    };

    mockApiRequest.mockResolvedValue(mockResponse);

    const { result } = renderHook(() => useDVENodes());

    let success = false;

    await act(async () => {
      success = await result.current.updateNodeStatus('node-1', 'inactive');
    });

    expect(success).toBe(true);
    expect(mockApiRequest).toHaveBeenCalledWith(
      'http://localhost:8082/api/dve-nodes/node-1/status',
      {
        method: 'PUT',
        body: JSON.stringify({ status: 'inactive' }),
      }
    );
  });

  it('should handle update node status error', async () => {
    mockApiRequest.mockRejectedValue(new Error('Update failed'));

    const { result } = renderHook(() => useDVENodes());

    let success = false;

    await act(async () => {
      success = await result.current.updateNodeStatus('node-1', 'inactive');
    });

    expect(success).toBe(false);
    expect(result.current.error).toBe('Update failed');
  });

  it('should delete node', async () => {
    const mockResponse = {
      success: true,
      data: null,
      timestamp: new Date().toISOString(),
    };

    mockApiRequest.mockResolvedValue(mockResponse);

    const { result } = renderHook(() => useDVENodes());

    let success = false;

    await act(async () => {
      success = await result.current.deleteNode('node-1');
    });

    expect(success).toBe(true);
    expect(mockApiRequest).toHaveBeenCalledWith(
      'http://localhost:8082/api/dve-nodes/node-1',
      { method: 'DELETE' }
    );
  });

  it('should handle delete node error', async () => {
    mockApiRequest.mockRejectedValue(new Error('Delete failed'));

    const { result } = renderHook(() => useDVENodes());

    let success = false;

    await act(async () => {
      success = await result.current.deleteNode('node-1');
    });

    expect(success).toBe(false);
    expect(result.current.error).toBe('Delete failed');
  });

  it('should refresh nodes data', async () => {
    const mockResponse = {
      success: true,
      data: [mockNode],
      timestamp: new Date().toISOString(),
    };

    mockApiRequest.mockResolvedValue(mockResponse);

    const { result } = renderHook(() => useDVENodes());

    await act(async () => {
      await result.current.refreshNodes();
    });

    expect(result.current.nodes).toEqual([mockNode]);
    expect(mockApiRequest).toHaveBeenCalledWith(
      'http://localhost:8082/api/dve-nodes',
      { method: 'GET' }
    );
  });
});
