import { renderHook, act, waitFor } from '@testing-library/react';
import { useCognitiveEngine } from '../use-cognitive-engine';
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

describe('useCognitiveEngine', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    console.error = jest.fn();
    console.log = jest.fn();
    
    // Setup default WebSocket service mocks
    mockWebSocketService.on = jest.fn();
    mockWebSocketService.off = jest.fn();
    mockWebSocketService.subscribe = jest.fn();
    mockWebSocketService.getConnectionStatus = jest.fn().mockReturnValue(false);
    
    // Clear any existing polling intervals
    if ((window as any).__cognitiveEnginePollingInterval) {
      clearInterval((window as any).__cognitiveEnginePollingInterval);
      delete (window as any).__cognitiveEnginePollingInterval;
    }
  });

  afterEach(() => {
    console.error = originalConsoleError;
    console.log = originalConsoleLog;
    
    // Clean up any polling intervals
    if ((window as any).__cognitiveEnginePollingInterval) {
      clearInterval((window as any).__cognitiveEnginePollingInterval);
      delete (window as any).__cognitiveEnginePollingInterval;
    }
  });

  const mockCognitiveEngineData = {
    status: "active" as const,
    accuracy: 0.95,
    tasks_processed: 1000,
    adaptation_rate: 0.8,
    model_version: "v2.1.0",
    uptime: 86400,
    last_training: "2024-01-15T10:30:00Z",
    performance_metrics: {
      inference_latency: 150,
      throughput: 100,
      error_rate: 0.02,
    },
    learning_metrics: {
      training_accuracy: 0.98,
      validation_accuracy: 0.95,
      loss: 0.05,
    },
  };

  it('should initialize with default values', async () => {
    mockApiRequest.mockResolvedValueOnce({
      success: true,
      data: mockCognitiveEngineData,
      timestamp: new Date().toISOString(),
    });

    const { result } = renderHook(() => useCognitiveEngine());

    // Initially, cognitiveEngine should be null, but isLoading might be true due to immediate fetch
    expect(result.current.cognitiveEngine).toBeNull();
    expect(result.current.error).toBeNull();
    expect(result.current.isPolling).toBe(false);
    expect(result.current.isConnected).toBe(false);

    // Wait for the initial fetch to complete
    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });
  });

  it('should fetch cognitive engine data on mount', async () => {
    mockApiRequest.mockResolvedValueOnce({
      success: true,
      data: mockCognitiveEngineData,
      timestamp: new Date().toISOString(),
    });

    const { result } = renderHook(() => useCognitiveEngine());

    await waitFor(() => {
      expect(result.current.cognitiveEngine).toEqual(mockCognitiveEngineData);
    });

    expect(mockApiRequest).toHaveBeenCalledWith(
      expect.stringContaining('/api/cognitive-engine'),
      { method: 'GET' }
    );
  });

  it('should handle fetch error', async () => {
    const errorMessage = 'Network error';
    mockApiRequest.mockRejectedValueOnce(new Error(errorMessage));

    const { result } = renderHook(() => useCognitiveEngine());

    await waitFor(() => {
      expect(result.current.error).toBe(errorMessage);
    });

    expect(result.current.cognitiveEngine).toBeNull();
  });

  it('should handle API response with error', async () => {
    const errorMessage = 'API error';
    mockApiRequest.mockResolvedValueOnce({
      success: false,
      error: errorMessage,
      timestamp: new Date().toISOString(),
    });

    const { result } = renderHook(() => useCognitiveEngine());

    await waitFor(() => {
      expect(result.current.error).toBe(errorMessage);
    });
  });

  it('should perform action successfully', async () => {
    mockApiRequest
      .mockResolvedValueOnce({
        success: true,
        data: mockCognitiveEngineData,
        timestamp: new Date().toISOString(),
      })
      .mockResolvedValueOnce({
        success: true,
        data: { ...mockCognitiveEngineData, status: "learning" as const },
        timestamp: new Date().toISOString(),
      });

    const { result } = renderHook(() => useCognitiveEngine());

    await waitFor(() => {
      expect(result.current.cognitiveEngine).toEqual(mockCognitiveEngineData);
    });

    let actionResult: boolean;
    await act(async () => {
      actionResult = await result.current.performAction({ action: 'start_training' });
    });

    expect(actionResult!).toBe(true);
    expect(result.current.cognitiveEngine?.status).toBe("learning");
    expect(mockApiRequest).toHaveBeenCalledWith(
      expect.stringContaining('/api/cognitive-engine'),
      {
        method: 'POST',
        body: JSON.stringify({ action: 'start_training' }),
      }
    );
  });

  it('should handle action failure', async () => {
    mockApiRequest
      .mockResolvedValueOnce({
        success: true,
        data: mockCognitiveEngineData,
        timestamp: new Date().toISOString(),
      })
      .mockRejectedValueOnce(new Error('Action failed'));

    const { result } = renderHook(() => useCognitiveEngine());

    await waitFor(() => {
      expect(result.current.cognitiveEngine).toEqual(mockCognitiveEngineData);
    });

    let actionResult: boolean;
    await act(async () => {
      actionResult = await result.current.performAction({ action: 'invalid_action' });
    });

    expect(actionResult!).toBe(false);
    expect(result.current.error).toBe('Action failed');
  });

  it('should call convenience methods', async () => {
    mockApiRequest
      .mockResolvedValueOnce({
        success: true,
        data: mockCognitiveEngineData,
        timestamp: new Date().toISOString(),
      })
      .mockResolvedValue({
        success: true,
        data: mockCognitiveEngineData,
        timestamp: new Date().toISOString(),
      });

    const { result } = renderHook(() => useCognitiveEngine());

    await waitFor(() => {
      expect(result.current.cognitiveEngine).toEqual(mockCognitiveEngineData);
    });

    // Test convenience methods
    await act(async () => {
      await result.current.startTraining();
    });

    await act(async () => {
      await result.current.stopTraining();
    });

    await act(async () => {
      await result.current.resetMetrics();
    });

    await act(async () => {
      await result.current.clearConversationHistory();
    });

    await act(async () => {
      await result.current.updateModel('v3.0.0');
    });

    expect(mockApiRequest).toHaveBeenCalledWith(
      expect.stringContaining('/api/cognitive-engine'),
      {
        method: 'POST',
        body: JSON.stringify({ action: 'start_training' }),
      }
    );

    expect(mockApiRequest).toHaveBeenCalledWith(
      expect.stringContaining('/api/cognitive-engine'),
      {
        method: 'POST',
        body: JSON.stringify({ action: 'update_model', parameters: { model_version: 'v3.0.0' } }),
      }
    );
  });

  it('should setup WebSocket connection', () => {
    mockWebSocketService.getConnectionStatus.mockReturnValue(true);

    const { result } = renderHook(() => useCognitiveEngine());

    expect(mockWebSocketService.on).toHaveBeenCalledWith('connection', expect.any(Function));
    expect(mockWebSocketService.on).toHaveBeenCalledWith('cognitive-engine-updated', expect.any(Function));
    expect(mockWebSocketService.on).toHaveBeenCalledWith('system-notification', expect.any(Function));
    expect(mockWebSocketService.subscribe).toHaveBeenCalledWith(['cognitive-engine-updated', 'system-notification']);
    expect(result.current.isConnected).toBe(true);
  });

  it('should handle WebSocket events', () => {
    let connectionHandler: (data: { connected: boolean }) => void;
    let updateHandler: (payload: any) => void;

    mockWebSocketService.on.mockImplementation((event, handler) => {
      if (event === 'connection') {
        connectionHandler = handler;
      } else if (event === 'cognitive-engine-updated') {
        updateHandler = handler;
      }
    });

    const { result } = renderHook(() => useCognitiveEngine());

    // Test connection event
    act(() => {
      connectionHandler!({ connected: true });
    });
    expect(result.current.isConnected).toBe(true);

    // Test update event
    act(() => {
      updateHandler!(mockCognitiveEngineData);
    });
    expect(result.current.cognitiveEngine).toEqual(mockCognitiveEngineData);
  });

  it('should start and stop polling', async () => {
    jest.useFakeTimers();
    
    mockApiRequest.mockResolvedValue({
      success: true,
      data: mockCognitiveEngineData,
      timestamp: new Date().toISOString(),
    });

    const { result } = renderHook(() => useCognitiveEngine());

    await waitFor(() => {
      expect(result.current.cognitiveEngine).toEqual(mockCognitiveEngineData);
    });

    // Start polling
    act(() => {
      result.current.startPolling(1000);
    });

    expect(result.current.isPolling).toBe(true);

    // Fast-forward time to trigger polling
    act(() => {
      jest.advanceTimersByTime(1000);
    });

    // Stop polling
    act(() => {
      result.current.stopPolling();
    });

    expect(result.current.isPolling).toBe(false);

    jest.useRealTimers();
  });

  it('should not start polling if already polling', () => {
    const { result } = renderHook(() => useCognitiveEngine());

    act(() => {
      result.current.startPolling();
    });

    const firstPollingState = result.current.isPolling;

    act(() => {
      result.current.startPolling();
    });

    expect(result.current.isPolling).toBe(firstPollingState);
  });

  it('should not stop polling if not polling', () => {
    const { result } = renderHook(() => useCognitiveEngine());

    expect(result.current.isPolling).toBe(false);

    act(() => {
      result.current.stopPolling();
    });

    expect(result.current.isPolling).toBe(false);
  });

  it('should disconnect WebSocket', () => {
    const { result } = renderHook(() => useCognitiveEngine());

    act(() => {
      result.current.disconnectWebSocket();
    });

    expect(result.current.isConnected).toBe(false);
  });

  it('should cleanup on unmount', () => {
    const { unmount } = renderHook(() => useCognitiveEngine());

    unmount();

    expect(mockWebSocketService.off).toHaveBeenCalledWith('connection', expect.any(Function));
    expect(mockWebSocketService.off).toHaveBeenCalledWith('cognitive-engine-updated', expect.any(Function));
    expect(mockWebSocketService.off).toHaveBeenCalledWith('system-notification', expect.any(Function));
  });
});
