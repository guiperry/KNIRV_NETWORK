import { renderHook, act, waitFor } from '@testing-library/react';
import { useSystemHealth } from '../use-system-health';
import { apiRequest } from '@/lib/api';
import { webSocketService } from '@/lib/websocket-service';

// Mock dependencies
jest.mock('@/lib/api');
jest.mock('@/lib/websocket-service');

const mockApiRequest = apiRequest as jest.MockedFunction<typeof apiRequest>;
const mockWebSocketService = webSocketService as jest.Mocked<typeof webSocketService>;

// Mock console methods to avoid noise in tests
const originalConsoleError = console.error;
const originalConsoleLog = console.log;

describe('useSystemHealth', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    console.error = jest.fn();
    console.log = jest.fn();
    
    // Setup default WebSocket service mocks
    mockWebSocketService.on = jest.fn();
    mockWebSocketService.off = jest.fn();
    mockWebSocketService.subscribe = jest.fn();
    mockWebSocketService.getConnectionStatus = jest.fn().mockReturnValue(false);
  });

  afterEach(() => {
    console.error = originalConsoleError;
    console.log = originalConsoleLog;
  });

  const mockSystemHealthData = {
    overall_status: "healthy" as const,
    timestamp: "2024-01-15T10:30:00Z",
    uptime: 86400,
    components: {
      database: {
        status: "healthy" as const,
        message: "All connections active",
        metrics: { connections: 10 },
      },
      api: {
        status: "healthy" as const,
        message: "All endpoints responding",
        metrics: { response_time: 150 },
      },
    },
    component_summary: {
      total_components: 2,
      healthy_components: 2,
      warning_components: 0,
      critical_components: 0,
    },
    active_alerts: 0,
    alerts: [],
    metrics: {
      system_load: 0.5,
      memory_usage: 60.5,
      disk_usage: 45.2,
      network_throughput: 1000,
      active_connections: 50,
      goroutine_count: 100,
      cpu_usage: 25.5,
    },
  };

  const mockAlert = {
    id: "alert-1",
    severity: "medium" as const,
    component: "database",
    message: "High connection count",
    timestamp: "2024-01-15T10:30:00Z",
    resolved: false,
    metadata: { connection_count: 95 },
  };

  it('should initialize with default values', async () => {
    mockApiRequest.mockResolvedValueOnce({
      success: true,
      data: mockSystemHealthData,
      timestamp: new Date().toISOString(),
    });

    const { result } = renderHook(() => useSystemHealth());

    expect(result.current.systemHealth).toBeNull();
    expect(result.current.alerts).toEqual([]);
    expect(result.current.metrics).toBeNull();
    expect(result.current.components).toEqual({});
    expect(result.current.error).toBeNull();
    expect(result.current.isConnected).toBe(false);

    // Wait for initial fetch to complete
    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });
  });

  it('should fetch system health data on mount', async () => {
    mockApiRequest.mockResolvedValueOnce({
      success: true,
      data: mockSystemHealthData,
      timestamp: new Date().toISOString(),
    });

    const { result } = renderHook(() => useSystemHealth());

    await waitFor(() => {
      expect(result.current.systemHealth).toEqual(mockSystemHealthData);
    });

    expect(result.current.alerts).toEqual(mockSystemHealthData.alerts);
    expect(result.current.metrics).toEqual(mockSystemHealthData.metrics);
    expect(result.current.components).toEqual(mockSystemHealthData.components);
    expect(mockApiRequest).toHaveBeenCalledWith(
      expect.stringContaining('/api/system-health?detailed=true'),
      { method: 'GET' }
    );
  });

  it('should handle fetch error', async () => {
    const errorMessage = 'Network error';
    mockApiRequest.mockRejectedValueOnce(new Error(errorMessage));

    const { result } = renderHook(() => useSystemHealth());

    await waitFor(() => {
      expect(result.current.error).toBe(errorMessage);
    });

    expect(result.current.systemHealth).toBeNull();
  });

  it('should fetch alerts with filters', async () => {
    mockApiRequest
      .mockResolvedValueOnce({
        success: true,
        data: mockSystemHealthData,
        timestamp: new Date().toISOString(),
      })
      .mockResolvedValueOnce({
        success: true,
        data: [mockAlert],
        timestamp: new Date().toISOString(),
      });

    const { result } = renderHook(() => useSystemHealth());

    await waitFor(() => {
      expect(result.current.systemHealth).toEqual(mockSystemHealthData);
    });

    await act(async () => {
      await result.current.fetchAlerts(false, 'medium');
    });

    expect(result.current.alerts).toEqual([mockAlert]);
    expect(mockApiRequest).toHaveBeenCalledWith(
      expect.stringContaining('/api/system-health/alerts?resolved=false&severity=medium'),
      { method: 'GET' }
    );
  });

  it('should fetch metrics', async () => {
    const mockMetrics = {
      system_load: 0.8,
      memory_usage: 75.0,
      disk_usage: 50.0,
      network_throughput: 1500,
      active_connections: 75,
      goroutine_count: 150,
      cpu_usage: 40.0,
    };

    mockApiRequest
      .mockResolvedValueOnce({
        success: true,
        data: mockSystemHealthData,
        timestamp: new Date().toISOString(),
      })
      .mockResolvedValueOnce({
        success: true,
        data: mockMetrics,
        timestamp: new Date().toISOString(),
      });

    const { result } = renderHook(() => useSystemHealth());

    await waitFor(() => {
      expect(result.current.systemHealth).toEqual(mockSystemHealthData);
    });

    await act(async () => {
      await result.current.fetchMetrics();
    });

    expect(result.current.metrics).toEqual(mockMetrics);
    expect(mockApiRequest).toHaveBeenCalledWith(
      expect.stringContaining('/api/system-health/metrics'),
      { method: 'GET' }
    );
  });

  it('should fetch components', async () => {
    const mockComponents = {
      database: {
        status: "warning" as const,
        message: "High load detected",
        metrics: { connections: 90 },
      },
    };

    mockApiRequest
      .mockResolvedValueOnce({
        success: true,
        data: mockSystemHealthData,
        timestamp: new Date().toISOString(),
      })
      .mockResolvedValueOnce({
        success: true,
        data: mockComponents,
        timestamp: new Date().toISOString(),
      });

    const { result } = renderHook(() => useSystemHealth());

    await waitFor(() => {
      expect(result.current.systemHealth).toEqual(mockSystemHealthData);
    });

    await act(async () => {
      await result.current.fetchComponents();
    });

    expect(result.current.components).toEqual(mockComponents);
    expect(mockApiRequest).toHaveBeenCalledWith(
      expect.stringContaining('/api/system-health/components'),
      { method: 'GET' }
    );
  });

  it('should execute actions', async () => {
    mockApiRequest
      .mockResolvedValueOnce({
        success: true,
        data: mockSystemHealthData,
        timestamp: new Date().toISOString(),
      })
      .mockResolvedValueOnce({
        success: true,
        data: { message: 'Diagnostics completed' },
        timestamp: new Date().toISOString(),
      })
      .mockResolvedValueOnce({
        success: true,
        data: mockSystemHealthData,
        timestamp: new Date().toISOString(),
      });

    const { result } = renderHook(() => useSystemHealth());

    await waitFor(() => {
      expect(result.current.systemHealth).toEqual(mockSystemHealthData);
    });

    let actionResult: any;
    await act(async () => {
      actionResult = await result.current.executeAction({
        action: 'run_diagnostics',
        parameters: { component: 'database' },
      });
    });

    expect(actionResult).toEqual({ message: 'Diagnostics completed' });
    expect(mockApiRequest).toHaveBeenCalledWith(
      expect.stringContaining('/api/system-health/actions'),
      {
        method: 'POST',
        body: JSON.stringify({
          action: 'run_diagnostics',
          parameters: { component: 'database' },
        }),
      }
    );
  });

  it('should resolve alerts', async () => {
    mockApiRequest
      .mockResolvedValueOnce({
        success: true,
        data: mockSystemHealthData,
        timestamp: new Date().toISOString(),
      })
      .mockResolvedValueOnce({
        success: true,
        data: { message: 'Alert resolved' },
        timestamp: new Date().toISOString(),
      });

    const { result } = renderHook(() => useSystemHealth());

    await waitFor(() => {
      expect(result.current.systemHealth).toEqual(mockSystemHealthData);
    });

    let resolveResult: boolean;
    await act(async () => {
      resolveResult = await result.current.resolveAlert('alert-1');
    });

    expect(resolveResult!).toBe(true);
    expect(mockApiRequest).toHaveBeenCalledWith(
      expect.stringContaining('/api/system-health/alerts/alert-1/resolve'),
      {
        method: 'POST',
      }
    );
  });

  it('should run diagnostics', async () => {
    mockApiRequest
      .mockResolvedValueOnce({
        success: true,
        data: mockSystemHealthData,
        timestamp: new Date().toISOString(),
      })
      .mockResolvedValueOnce({
        success: true,
        data: {
          id: 'diag-1',
          timestamp: '2024-01-15T10:30:00Z',
          status: 'completed',
          summary: 'All tests passed',
          tests: [],
        },
        timestamp: new Date().toISOString(),
      });

    const { result } = renderHook(() => useSystemHealth());

    await waitFor(() => {
      expect(result.current.systemHealth).toEqual(mockSystemHealthData);
    });

    let diagnosticsResult: any;
    await act(async () => {
      diagnosticsResult = await result.current.runDiagnostics();
    });

    expect(diagnosticsResult).toBeDefined();
    expect(diagnosticsResult.status).toBe('completed');
  });

  it('should add alerts', async () => {
    const newAlert = { ...mockAlert, id: 'new-alert-1' };
    mockApiRequest
      .mockResolvedValueOnce({
        success: true,
        data: mockSystemHealthData,
        timestamp: new Date().toISOString(),
      })
      .mockResolvedValueOnce({
        success: true,
        data: newAlert,
        timestamp: new Date().toISOString(),
      })
      .mockResolvedValueOnce({
        success: true,
        data: mockSystemHealthData,
        timestamp: new Date().toISOString(),
      });

    const { result } = renderHook(() => useSystemHealth());

    await waitFor(() => {
      expect(result.current.systemHealth).toEqual(mockSystemHealthData);
    });

    let addResult: any;
    await act(async () => {
      addResult = await result.current.addAlert('high', 'database', 'Test alert', { test: true });
    });

    expect(addResult).toEqual(newAlert);
  });

  it('should refresh all data', async () => {
    mockApiRequest.mockResolvedValue({
      success: true,
      data: mockSystemHealthData,
      timestamp: new Date().toISOString(),
    });

    const { result } = renderHook(() => useSystemHealth());

    await waitFor(() => {
      expect(result.current.systemHealth).toEqual(mockSystemHealthData);
    });

    await act(async () => {
      await result.current.refreshAll();
    });

    // Should call fetchSystemHealth, fetchAlerts, fetchMetrics, fetchComponents
    expect(mockApiRequest).toHaveBeenCalledTimes(5); // 1 initial + 4 refresh calls
  });

  it('should setup WebSocket connection', () => {
    mockWebSocketService.getConnectionStatus.mockReturnValue(true);

    const { result } = renderHook(() => useSystemHealth());

    expect(mockWebSocketService.on).toHaveBeenCalledWith('connection', expect.any(Function));
    expect(mockWebSocketService.on).toHaveBeenCalledWith('system-health-updated', expect.any(Function));
    expect(mockWebSocketService.on).toHaveBeenCalledWith('system-alert-added', expect.any(Function));
    expect(mockWebSocketService.on).toHaveBeenCalledWith('system-notification', expect.any(Function));
    expect(result.current.isConnected).toBe(true);
  });

  it('should handle WebSocket events', async () => {
    let healthUpdateHandler: (payload: any) => void;
    let alertAddedHandler: (payload: any) => void;

    mockApiRequest.mockResolvedValueOnce({
      success: true,
      data: mockSystemHealthData,
      timestamp: new Date().toISOString(),
    });

    mockWebSocketService.on.mockImplementation((event, handler) => {
      if (event === 'system-health-updated') {
        healthUpdateHandler = handler;
      } else if (event === 'system-alert-added') {
        alertAddedHandler = handler;
      }
    });

    const { result } = renderHook(() => useSystemHealth());

    await waitFor(() => {
      expect(result.current.systemHealth).toEqual(mockSystemHealthData);
    });

    // Test health update event
    act(() => {
      healthUpdateHandler!(mockSystemHealthData);
    });
    expect(result.current.systemHealth).toEqual(mockSystemHealthData);

    // Test alert added event
    act(() => {
      alertAddedHandler!(mockAlert);
    });
    expect(result.current.alerts).toContain(mockAlert);
  });

  it('should disconnect WebSocket', () => {
    const { result } = renderHook(() => useSystemHealth());

    act(() => {
      result.current.disconnectWebSocket();
    });

    expect(result.current.isConnected).toBe(false);
  });

  it('should cleanup on unmount', () => {
    const { unmount } = renderHook(() => useSystemHealth());

    unmount();

    expect(mockWebSocketService.off).toHaveBeenCalledWith('connection', expect.any(Function));
    expect(mockWebSocketService.off).toHaveBeenCalledWith('system-health-updated', expect.any(Function));
    expect(mockWebSocketService.off).toHaveBeenCalledWith('system-alert-added', expect.any(Function));
    expect(mockWebSocketService.off).toHaveBeenCalledWith('system-notification', expect.any(Function));
  });
});
