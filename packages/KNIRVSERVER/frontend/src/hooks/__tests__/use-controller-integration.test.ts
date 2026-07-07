import { renderHook, act, waitFor } from '@testing-library/react';
import { useControllerIntegration } from '../use-controller-integration';
import type { QRCode, PairingRequest, ControllerSession, ControllerMessage } from '../use-controller-integration';
import type { ControllerDevice, ControllerStats } from '@/types/api';
import { apiRequest } from '@/lib/api';

// Mock the API module
jest.mock('@/lib/api', () => ({
  apiRequest: jest.fn(),
  API_BASE_URL: 'http://localhost:8082',
}));

const mockApiRequest = apiRequest as jest.MockedFunction<typeof apiRequest>;

// Mock WebSocket service
jest.mock('@/lib/websocket-service', () => ({
  webSocketService: {
    connect: jest.fn(),
    disconnect: jest.fn(),
    subscribe: jest.fn(),
    unsubscribe: jest.fn(),
    getConnectionStatus: jest.fn().mockReturnValue(true),
    on: jest.fn(),
    off: jest.fn(),
  },
}));

import { webSocketService } from '@/lib/websocket-service';
const mockWebSocketService = webSocketService as jest.Mocked<typeof webSocketService>;

const mockQRCodes: QRCode[] = [
  {
    id: 'qr-1',
    session_id: 'session-1',
    desktop_id: 'desktop-1',
    user_id: 'user-1',
    device_type: 'mobile',
    capabilities: ['wallet', 'model-control'],
    status: 'active',
    created_at: '2024-01-01T00:00:00Z',
    expires_at: '2024-01-01T01:00:00Z',
    scan_count: 0,
    max_scans: 5,
    data: {
      version: '1.0',
      type: 'pairing',
      session_id: 'session-1',
      desktop_id: 'desktop-1',
      user_id: 'user-1',
      device_type: 'mobile',
      capabilities: ['wallet', 'model-control'],
      expires_at: 1704067200,
      timestamp: 1704063600,
      signature: 'signature-1'
    }
  },
  {
    id: 'qr-2',
    session_id: 'session-2',
    desktop_id: 'desktop-2',
    user_id: 'user-2',
    device_type: 'tablet',
    capabilities: ['monitoring'],
    status: 'used',
    created_at: '2024-01-02T00:00:00Z',
    expires_at: '2024-01-02T01:00:00Z',
    used_at: '2024-01-02T00:30:00Z',
    scan_count: 1,
    max_scans: 3,
    data: {
      version: '1.0',
      type: 'pairing',
      session_id: 'session-2',
      desktop_id: 'desktop-2',
      user_id: 'user-2',
      device_type: 'tablet',
      capabilities: ['monitoring'],
      expires_at: 1704153600,
      timestamp: 1704150000,
      signature: 'signature-2'
    }
  }
];

const mockPairingRequests: PairingRequest[] = [
  {
    id: 'pairing-1',
    qr_code_id: 'qr-1',
    session_id: 'session-1',
    desktop_id: 'desktop-1',
    user_id: 'user-1',
    mobile_device_id: 'mobile-1',
    status: 'pending',
    created_at: '2024-01-01T00:00:00Z',
    expires_at: '2024-01-01T01:00:00Z',
    capabilities: ['wallet', 'model-control'],
    device_info: {
      device_id: 'mobile-1',
      device_type: 'mobile',
      platform: 'iOS',
      version: '17.0',
      user_model: 'iPhone 15'
    }
  },
  {
    id: 'pairing-2',
    qr_code_id: 'qr-2',
    session_id: 'session-2',
    desktop_id: 'desktop-2',
    user_id: 'user-2',
    mobile_device_id: 'mobile-2',
    status: 'confirmed',
    created_at: '2024-01-02T00:00:00Z',
    expires_at: '2024-01-02T01:00:00Z',
    confirmed_at: '2024-01-02T00:30:00Z',
    capabilities: ['monitoring'],
    device_info: {
      device_id: 'mobile-2',
      device_type: 'tablet',
      platform: 'iPadOS',
      version: '17.0',
      user_model: 'iPad Pro'
    }
  }
];

const mockSessions: ControllerSession[] = [
  {
    id: 'session-1',
    session_id: 'session-1',
    desktop_id: 'desktop-1',
    mobile_device_id: 'mobile-1',
    user_id: 'user-1',
    status: 'active',
    capabilities: ['wallet', 'model-control'],
    created_at: '2024-01-01T00:00:00Z',
    last_activity: '2024-01-01T00:30:00Z',
    expires_at: '2024-01-01T02:00:00Z',
    device_info: {
      device_id: 'mobile-1',
      device_type: 'mobile',
      platform: 'iOS',
      version: '17.0',
      user_model: 'iPhone 15'
    },
    connection_info: {
      ip_address: '192.168.1.1',
      user_model: 'desktop',
      connection_type: 'wifi',
      encrypted: true
    },
    session_data: {
      last_command: 'wallet.getBalance'
    },
    message_count: 5
  },
  {
    id: 'session-2',
    session_id: 'session-2',
    desktop_id: 'desktop-2',
    user_id: 'user-2',
    mobile_device_id: 'mobile-2',
    status: 'inactive',
    capabilities: ['monitoring'],
    created_at: '2024-01-02T00:00:00Z',
    last_activity: '2024-01-02T01:00:00Z',
    expires_at: '2024-01-02T02:00:00Z',
    device_info: {
      device_id: 'mobile-2',
      device_type: 'tablet',
      platform: 'iPadOS',
      version: '17.0',
      user_model: 'iPad Pro'
    },
    connection_info: {
      ip_address: '192.168.1.2',
      user_model: 'desktop',
      connection_type: 'cellular',
      encrypted: true
    },
    session_data: {
      last_report: 'system_status'
    },
    message_count: 10
  }
];

const mockDevices: ControllerDevice[] = [
  {
    id: 'mobile-1',
    user_id: 'user-1',
    device_type: 'mobile',
    device_name: 'iPhone 15',
    os_version: 'iOS 17.0',
    app_version: '1.0.0',
    capabilities: ['wallet', 'model-control'],
    status: 'online',
    last_seen: '2024-01-01T00:30:00Z',
    registered_at: '2024-01-01T00:00:00Z'
  },
  {
    id: 'mobile-2',
    user_id: 'user-2',
    device_type: 'tablet',
    device_name: 'iPad Pro',
    os_version: 'iPadOS 17.0',
    app_version: '1.0.0',
    capabilities: ['monitoring'],
    status: 'offline',
    last_seen: '2024-01-02T01:00:00Z',
    registered_at: '2024-01-02T00:00:00Z'
  }
];

const mockStats: ControllerStats = {
  total_qr_codes: 25,
  active_qr_codes: 5,
  total_pairing_requests: 15,
  pending_pairing_requests: 3,
  active_sessions: 8,
  total_devices: 12,
  online_devices: 7
};

describe('useControllerIntegration Hook', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockApiRequest.mockClear();
    mockWebSocketService.on.mockClear();
    mockWebSocketService.off.mockClear();
    mockWebSocketService.subscribe.mockClear();
    mockWebSocketService.unsubscribe.mockClear();
    mockWebSocketService.getConnectionStatus.mockClear().mockReturnValue(true);
    mockWebSocketService.disconnect.mockClear();
  });

  it('initializes with default state', () => {
    const { result } = renderHook(() => useControllerIntegration());

    expect(result.current.qrCode).toBeNull();
    expect(result.current.pairingRequest).toBeNull();
    expect(result.current.activeSessions).toEqual([]);
    expect(result.current.selectedSession).toBeNull();
    expect(result.current.messages).toEqual([]);
    expect(result.current.isLoading).toBe(false);
    expect(result.current.error).toBeNull();
    expect(result.current.isConnected).toBe(true);
  });

  it('generates QR code successfully', async () => {
    const mockQRCode = mockQRCodes[0];
    mockApiRequest.mockResolvedValueOnce({
      success: true,
      data: mockQRCode,
      timestamp: new Date().toISOString()
    });

    const { result } = renderHook(() => useControllerIntegration());

    const qrRequest = {
      desktop_id: 'desktop-1',
      user_id: 'user-1',
      device_type: 'mobile',
      capabilities: ['wallet', 'model-control'],
      expires_in: 3600,
      max_scans: 5
    };

    let generatedQRCode: any = null;
    await act(async () => {
      generatedQRCode = await result.current.generateQRCode(qrRequest);
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8082/api/controller-integration/qr-code', {
      method: 'POST',
      body: JSON.stringify(qrRequest)
    });
    expect(generatedQRCode).toEqual(mockQRCode);
    expect(result.current.qrCode).toEqual(mockQRCode);
    expect(result.current.error).toBeNull();
  });

  it('handles generate QR code error', async () => {
    const errorMessage = 'Network error';
    mockApiRequest.mockRejectedValueOnce(new Error(errorMessage));

    const { result } = renderHook(() => useControllerIntegration());

    const qrRequest = {
      desktop_id: 'desktop-1',
      user_id: 'user-1',
      device_type: 'mobile',
      capabilities: ['wallet'],
      expires_in: 3600,
      max_scans: 5
    };

    let generatedQRCode: any = null;
    await act(async () => {
      generatedQRCode = await result.current.generateQRCode(qrRequest);
    });

    expect(generatedQRCode).toBeNull();
    expect(result.current.qrCode).toBeNull();
    expect(result.current.error).toBe(errorMessage);
  });



  it('scans QR code successfully', async () => {
    const mockPairingRequest = mockPairingRequests[0];
    mockApiRequest.mockResolvedValueOnce({
      success: true,
      data: mockPairingRequest,
      timestamp: new Date().toISOString()
    });

    const { result } = renderHook(() => useControllerIntegration());

    const scanRequest = {
      qr_data: 'qr-data-string',
      mobile_device_id: 'mobile-1'
    };

    let pairingRequest: any = null;
    await act(async () => {
      pairingRequest = await result.current.scanQRCode(scanRequest);
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8082/api/controller-integration/qr-code/scan', {
      method: 'POST',
      body: JSON.stringify(scanRequest)
    });
    expect(pairingRequest).toEqual(mockPairingRequest);
    expect(result.current.pairingRequest).toEqual(mockPairingRequest);
  });

  it('confirms pairing request successfully', async () => {
    const mockSession = mockSessions[0];
    mockApiRequest.mockResolvedValueOnce({
      success: true,
      data: mockSession,
      timestamp: new Date().toISOString()
    });

    const { result } = renderHook(() => useControllerIntegration());

    let session: any = null;
    await act(async () => {
      session = await result.current.confirmPairing('pairing-1', true);
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8082/api/controller-integration/pairing/pairing-1/confirm', {
      method: 'POST',
      body: JSON.stringify({ confirmed: true })
    });
    expect(session).toEqual(mockSession);
    expect(result.current.activeSessions).toContain(mockSession);
    expect(result.current.selectedSession).toEqual(mockSession);
  });

  it('rejects pairing request successfully', async () => {
    mockApiRequest.mockResolvedValueOnce({
      success: true,
      data: null,
      timestamp: new Date().toISOString()
    });

    const { result } = renderHook(() => useControllerIntegration());

    let session: any = null;
    await act(async () => {
      session = await result.current.confirmPairing('pairing-1', false);
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8082/api/controller-integration/pairing/pairing-1/confirm', {
      method: 'POST',
      body: JSON.stringify({ confirmed: false })
    });
    expect(session).toBeNull();
    expect(result.current.pairingRequest).toBeNull();
  });

  it('gets user sessions successfully', async () => {
    mockApiRequest.mockResolvedValueOnce({
      success: true,
      data: mockSessions,
      timestamp: new Date().toISOString()
    });

    const { result } = renderHook(() => useControllerIntegration());

    await act(async () => {
      await result.current.getUserSessions('user-1');
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8082/api/controller-integration/users/user-1/sessions', {
      method: 'GET'
    });
    expect(result.current.activeSessions).toEqual(mockSessions);
  });

  it('terminates session successfully', async () => {
    mockApiRequest.mockResolvedValueOnce({
      success: true,
      timestamp: new Date().toISOString()
    });

    const { result } = renderHook(() => useControllerIntegration());

    let terminated: boolean = false;
    await act(async () => {
      terminated = await result.current.terminateSession('session-1');
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8082/api/controller-integration/sessions/session-1', {
      method: 'DELETE'
    });
    expect(terminated).toBe(true);
  });

  it('sends message successfully', async () => {
    const mockMessage = {
      id: 'msg-1',
      session_id: 'session-1',
      type: 'notification',
      content: 'Hello from desktop',
      timestamp: new Date().toISOString(),
      direction: 'outbound'
    };
    mockApiRequest.mockResolvedValueOnce({
      success: true,
      data: mockMessage,
      timestamp: new Date().toISOString()
    });

    const { result } = renderHook(() => useControllerIntegration());

    const messageData: Partial<ControllerMessage> = {
      type: 'notification',
      payload: {
        content: 'Hello from desktop',
      },
      timestamp: new Date().toISOString()
    };

    let sent: boolean = false;
    await act(async () => {
      sent = await result.current.sendMessage('session-1', messageData);
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8082/api/controller-integration/sessions/session-1/messages', {
      method: 'POST',
      body: JSON.stringify(messageData)
    });
    expect(sent).toBe(true);
  });

  it('sends command successfully', async () => {
    const mockCommandMessage = {
      id: 'cmd-1',
      session_id: 'session-1',
      type: 'command',
      direction: 'outbound',
      payload: {
        command: 'wallet.getBalance',
        parameters: { address: '0x123' }
      },
      timestamp: new Date().toISOString()
    };
    mockApiRequest.mockResolvedValueOnce({
      success: true,
      data: mockCommandMessage,
      timestamp: new Date().toISOString()
    });

    const { result } = renderHook(() => useControllerIntegration());

    let sent: boolean = false;
    await act(async () => {
      sent = await result.current.sendCommand('session-1', 'wallet.getBalance', { address: '0x123' });
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8082/api/controller-integration/sessions/session-1/messages', {
      method: 'POST',
      body: JSON.stringify({
        type: 'command',
        direction: 'outbound',
        payload: {
          command: 'wallet.getBalance',
          parameters: { address: '0x123' }
        }
      })
    });
    expect(sent).toBe(true);
  });

  it('refreshes user sessions successfully', async () => {
    mockApiRequest.mockResolvedValueOnce({
      success: true,
      data: mockSessions,
      timestamp: new Date().toISOString()
    });

    const { result } = renderHook(() => useControllerIntegration());

    await act(async () => {
      await result.current.refreshAll('user-1');
    });

    expect(mockApiRequest).toHaveBeenCalledWith('http://localhost:8082/api/controller-integration/users/user-1/sessions', {
      method: 'GET'
    });
    expect(result.current.activeSessions).toEqual(mockSessions);
  });

  it('connects to WebSocket successfully', async () => {
    const { result } = renderHook(() => useControllerIntegration());

    await act(async () => {
      result.current.connectWebSocket();
    });

    // The hook sets up event listeners and connection status
    expect(result.current.isConnected).toBe(true);
  });

  it('disconnects from WebSocket successfully', () => {
    const { result } = renderHook(() => useControllerIntegration());

    act(() => {
      result.current.disconnectWebSocket();
    });

    expect(result.current.isConnected).toBe(false);
  });

  it('clears QR code successfully', () => {
    const { result } = renderHook(() => useControllerIntegration());

    // Clear it
    act(() => {
      result.current.clearQRCode();
    });

    expect(result.current.qrCode).toBeNull();
  });

  it('clears pairing request successfully', () => {
    const { result } = renderHook(() => useControllerIntegration());

    // Clear it
    act(() => {
      result.current.clearPairingRequest();
    });

    expect(result.current.pairingRequest).toBeNull();
  });

  it('sets selected session successfully', () => {
    const { result } = renderHook(() => useControllerIntegration());

    const session = mockSessions[0];

    act(() => {
      result.current.setSelectedSession(session);
    });

    expect(result.current.selectedSession).toEqual(session);
  });


});
