"use client";

import { useState, useEffect, useCallback } from 'react';
import type { APIResponse } from '@/types/api';
import { apiRequest, API_BASE_URL } from '@/lib/api';
import { webSocketService } from '@/lib/websocket-service';

export interface QRCode {
  id: string;
  session_id: string;
  desktop_id: string;
  user_id: string;
  device_type: string;
  capabilities: string[];
  status: "active" | "used" | "expired";
  created_at: string;
  expires_at: string;
  used_at?: string;
  last_scanned_at?: string;
  scan_count: number;
  max_scans: number;
  data: QRCodeData;
}

export interface QRCodeData {
  version: string;
  type: string;
  session_id: string;
  desktop_id: string;
  user_id: string;
  device_type: string;
  capabilities: string[];
  expires_at: number;
  timestamp: number;
  signature: string;
}

export interface PairingRequest {
  id: string;
  qr_code_id: string;
  session_id: string;
  desktop_id: string;
  user_id: string;
  mobile_device_id: string;
  status: "pending" | "confirmed" | "rejected" | "expired";
  created_at: string;
  expires_at: string;
  confirmed_at?: string;
  rejected_at?: string;
  capabilities: string[];
  device_info: DeviceInfo;
}

export interface DeviceInfo {
  device_id: string;
  device_type: string;
  platform: string;
  version: string;
  user_agent?: string;
}

export interface ControllerSession {
  id: string;
  session_id: string;
  desktop_id: string;
  user_id: string;
  mobile_device_id: string;
  status: "active" | "inactive" | "terminated" | "expired";
  created_at: string;
  last_activity: string;
  expires_at: string;
  terminated_at?: string;
  termination_reason?: string;
  capabilities: string[];
  device_info: DeviceInfo;
  connection_info: ConnectionInfo;
  session_data: Record<string, any>;
  message_count: number;
}

export interface ConnectionInfo {
  ip_address: string;
  user_agent: string;
  connection_type: string;
  encrypted: boolean;
}

export interface ControllerMessage {
  id: string;
  session_id: string;
  type: "command" | "response" | "notification" | "heartbeat";
  direction: "inbound" | "outbound";
  payload: Record<string, any>;
  timestamp: string;
  processed: boolean;
}

export interface QRCodeRequest {
  user_id: string;
  device_type: string;
  capabilities: string[];
}

export interface ScanRequest {
  qr_data: string;
  mobile_device_id: string;
}

export interface PairingConfirmation {
  confirmed: boolean;
}

export const useControllerIntegration = () => {
  const [qrCode, setQRCode] = useState<QRCode | null>(null);
  const [pairingRequest, setPairingRequest] = useState<PairingRequest | null>(null);
  const [activeSessions, setActiveSessions] = useState<ControllerSession[]>([]);
  const [selectedSession, setSelectedSession] = useState<ControllerSession | null>(null);
  const [messages, setMessages] = useState<ControllerMessage[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isConnected, setIsConnected] = useState(false);

  // Generate QR code for controller pairing
  const generateQRCode = useCallback(async (request: QRCodeRequest): Promise<QRCode | null> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/controller-integration/qr-code`;
      const response: APIResponse<QRCode> = await apiRequest(url, {
        method: 'POST',
        body: JSON.stringify(request),
      });
      
      if (response.success && response.data) {
        setQRCode(response.data);
        return response.data;
      } else {
        throw new Error(response.error || 'Failed to generate QR code');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to generate QR code:', err);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Scan QR code (mobile app)
  const scanQRCode = useCallback(async (request: ScanRequest): Promise<PairingRequest | null> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/controller-integration/qr-code/scan`;
      const response: APIResponse<PairingRequest> = await apiRequest(url, {
        method: 'POST',
        body: JSON.stringify(request),
      });
      
      if (response.success && response.data) {
        setPairingRequest(response.data);
        return response.data;
      } else {
        throw new Error(response.error || 'Failed to scan QR code');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to scan QR code:', err);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Confirm pairing request
  const confirmPairing = useCallback(async (pairingRequestId: string, confirmed: boolean): Promise<ControllerSession | null> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/controller-integration/pairing/${pairingRequestId}/confirm`;
      const response: APIResponse<ControllerSession> = await apiRequest(url, {
        method: 'POST',
        body: JSON.stringify({ confirmed }),
      });
      
      if (response.success) {
        if (response.data) {
          // Session created
          setActiveSessions(prev => [...prev, response.data!]);
          setSelectedSession(response.data);
          return response.data;
        } else {
          // Pairing rejected
          setPairingRequest(null);
          return null;
        }
      } else {
        throw new Error(response.error || 'Failed to confirm pairing');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to confirm pairing:', err);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Get session details
  const getSession = useCallback(async (sessionId: string): Promise<ControllerSession | null> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/controller-integration/sessions/${sessionId}`;
      const response: APIResponse<ControllerSession> = await apiRequest(url, { method: 'GET' });
      
      if (response.success && response.data) {
        setSelectedSession(response.data);
        return response.data;
      } else {
        throw new Error(response.error || 'Failed to get session');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to get session:', err);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Get user sessions
  const getUserSessions = useCallback(async (userId: string): Promise<ControllerSession[]> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/controller-integration/users/${userId}/sessions`;
      const response: APIResponse<ControllerSession[]> = await apiRequest(url, { method: 'GET' });
      
      if (response.success && response.data) {
        setActiveSessions(response.data);
        return response.data;
      } else {
        throw new Error(response.error || 'Failed to get user sessions');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to get user sessions:', err);
      return [];
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Send message through session
  const sendMessage = useCallback(async (sessionId: string, message: Partial<ControllerMessage>): Promise<boolean> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/controller-integration/sessions/${sessionId}/messages`;
      const response: APIResponse<ControllerMessage> = await apiRequest(url, {
        method: 'POST',
        body: JSON.stringify(message),
      });
      
      if (response.success && response.data) {
        setMessages(prev => [...prev, response.data!]);
        return true;
      } else {
        throw new Error(response.error || 'Failed to send message');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to send message:', err);
      return false;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Terminate session
  const terminateSession = useCallback(async (sessionId: string, reason?: string): Promise<boolean> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/controller-integration/sessions/${sessionId}`;
      const response: APIResponse = await apiRequest(url, {
        method: 'DELETE',
        body: reason ? JSON.stringify({ reason }) : undefined,
      });
      
      if (response.success) {
        setActiveSessions(prev => prev.filter(session => session.id !== sessionId));
        if (selectedSession?.id === sessionId) {
          setSelectedSession(null);
        }
        return true;
      } else {
        throw new Error(response.error || 'Failed to terminate session');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to terminate session:', err);
      return false;
    } finally {
      setIsLoading(false);
    }
  }, [selectedSession]);

  // Convenience methods for common message types
  const sendCommand = useCallback((sessionId: string, command: string, parameters?: Record<string, any>) => {
    return sendMessage(sessionId, {
      type: 'command',
      direction: 'outbound',
      payload: { command, parameters },
    });
  }, [sendMessage]);

  const sendNotification = useCallback((sessionId: string, notification: string, data?: Record<string, any>) => {
    return sendMessage(sessionId, {
      type: 'notification',
      direction: 'outbound',
      payload: { notification, data },
    });
  }, [sendMessage]);

  const sendHeartbeat = useCallback((sessionId: string) => {
    return sendMessage(sessionId, {
      type: 'heartbeat',
      direction: 'outbound',
      payload: { timestamp: new Date().toISOString() },
    });
  }, [sendMessage]);

  // WebSocket connection management
  const connectWebSocket = useCallback(() => {
    // Set up event handlers
    const handleConnection = (data: { connected: boolean }) => {
      setIsConnected(data.connected);
      if (data.connected) {
        console.log('Controller Integration WebSocket connected');
        setError(null);
      } else {
        console.log('Controller Integration WebSocket disconnected');
      }
    };

    const handleControllerSessionUpdate = (payload: any) => {
      // Update session in the list
      setActiveSessions(prevSessions =>
        prevSessions.map(session =>
          session.id === payload.id ? { ...session, ...payload } : session
        )
      );

      // Update selected session if it matches
      if (selectedSession?.id === payload.id) {
        setSelectedSession(prev => prev ? { ...prev, ...payload } : null);
      }
    };

    const handleControllerMessageReceived = (payload: any) => {
      // Add new message
      setMessages(prev => [...prev, payload]);
    };

    const handlePairingRequestCreated = (payload: any) => {
      // Update pairing request
      setPairingRequest(payload);
    };

    const handleSystemNotification = (payload: any) => {
      console.log('Controller Integration system notification:', payload);
    };

    // Register event handlers
    webSocketService.on('connection', handleConnection);
    webSocketService.on('controller-session-updated', handleControllerSessionUpdate);
    webSocketService.on('controller-message-received', handleControllerMessageReceived);
    webSocketService.on('pairing-request-created', handlePairingRequestCreated);
    webSocketService.on('system-notification', handleSystemNotification);

    // Subscribe to events
    webSocketService.subscribe(['controller-session-updated', 'controller-message-received', 'pairing-request-created', 'system-notification']);

    // Set initial connection status
    setIsConnected(webSocketService.getConnectionStatus());

    // Return cleanup function
    return () => {
      webSocketService.off('connection', handleConnection);
      webSocketService.off('controller-session-updated', handleControllerSessionUpdate);
      webSocketService.off('controller-message-received', handleControllerMessageReceived);
      webSocketService.off('pairing-request-created', handlePairingRequestCreated);
      webSocketService.off('system-notification', handleSystemNotification);
    };
  }, [selectedSession]);

  const disconnectWebSocket = useCallback(() => {
    // Individual hooks don't disconnect the shared service
    setIsConnected(false);
  }, []);

  // Clear current QR code
  const clearQRCode = useCallback(() => {
    setQRCode(null);
  }, []);

  // Clear pairing request
  const clearPairingRequest = useCallback(() => {
    setPairingRequest(null);
  }, []);

  // Refresh all data
  const refreshAll = useCallback(async (userId?: string) => {
    if (userId) {
      await getUserSessions(userId);
    }
  }, [getUserSessions]);

  // Initial WebSocket connection on mount
  useEffect(() => {
    return connectWebSocket();
  }, [connectWebSocket]);

  return {
    qrCode,
    pairingRequest,
    activeSessions,
    selectedSession,
    messages,
    isLoading,
    error,
    isConnected,
    generateQRCode,
    scanQRCode,
    confirmPairing,
    getSession,
    getUserSessions,
    sendMessage,
    terminateSession,
    sendCommand,
    sendNotification,
    sendHeartbeat,
    clearQRCode,
    clearPairingRequest,
    refreshAll,
    connectWebSocket,
    disconnectWebSocket,
    setSelectedSession,
  };
};

export default useControllerIntegration;
