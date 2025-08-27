"use client";

import { useState, useEffect, useCallback } from 'react';
import type { APIResponse } from '@/types/api';
import { apiRequest, API_BASE_URL, StandardWebSocket } from '@/lib/api';

export interface SystemHealth {
  overall_status: "healthy" | "degraded" | "critical";
  timestamp: string;
  uptime: number;
  components: Record<string, ComponentHealth>;
  component_summary?: ComponentSummary;
  active_alerts?: number;
  alerts: SystemAlert[];
  metrics: SystemMetrics;
}

export interface ComponentHealth {
  status: "healthy" | "warning" | "critical";
  message: string;
  metrics?: Record<string, any>;
}

export interface ComponentSummary {
  total_components: number;
  healthy_components: number;
  warning_components: number;
  critical_components: number;
}

export interface SystemAlert {
  id: string;
  severity: "low" | "medium" | "high" | "critical";
  component: string;
  message: string;
  timestamp: string;
  resolved: boolean;
  resolved_at?: string;
  metadata?: Record<string, any>;
}

export interface SystemMetrics {
  system_load: number;
  memory_usage: number;
  disk_usage: number;
  network_throughput: number;
  active_connections: number;
  goroutine_count: number;
  cpu_usage: number;
}

export interface DiagnosticsResult {
  id: string;
  timestamp: string;
  status: "completed" | "failed";
  summary: string;
  tests: DiagnosticTest[];
}

export interface DiagnosticTest {
  name: string;
  status: "passed" | "failed";
  duration: number;
  message: string;
  details?: Record<string, any>;
}

export interface SystemHealthAction {
  action: "run_diagnostics" | "add_alert" | "resolve_alert";
  parameters?: Record<string, any>;
}

export const useSystemHealth = () => {
  const [systemHealth, setSystemHealth] = useState<SystemHealth | null>(null);
  const [alerts, setAlerts] = useState<SystemAlert[]>([]);
  const [metrics, setMetrics] = useState<SystemMetrics | null>(null);
  const [components, setComponents] = useState<Record<string, ComponentHealth>>({});
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [socket, setSocket] = useState<StandardWebSocket | null>(null);

  // Fetch system health status
  const fetchSystemHealth = useCallback(async (detailed: boolean = true) => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/system-health?detailed=${detailed}`;
      const response: APIResponse<SystemHealth> = await apiRequest(url, { method: 'GET' });
      
      if (response.success && response.data) {
        setSystemHealth(response.data);
        setAlerts(response.data.alerts || []);
        setMetrics(response.data.metrics);
        setComponents(response.data.components || {});
      } else {
        throw new Error(response.error || 'Failed to fetch system health');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to fetch system health:', err);
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Fetch system alerts
  const fetchAlerts = useCallback(async (resolved?: boolean, severity?: string) => {
    setIsLoading(true);
    setError(null);
    
    try {
      let url = `${API_BASE_URL}/api/system-health/alerts`;
      const params = new URLSearchParams();
      
      if (resolved !== undefined) {
        params.append('resolved', resolved.toString());
      }
      if (severity) {
        params.append('severity', severity);
      }
      
      if (params.toString()) {
        url += `?${params.toString()}`;
      }
      
      const response: APIResponse<SystemAlert[]> = await apiRequest(url, { method: 'GET' });
      
      if (response.success && response.data) {
        setAlerts(response.data);
      } else {
        throw new Error(response.error || 'Failed to fetch alerts');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to fetch alerts:', err);
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Fetch system metrics
  const fetchMetrics = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/system-health/metrics`;
      const response: APIResponse<SystemMetrics> = await apiRequest(url, { method: 'GET' });
      
      if (response.success && response.data) {
        setMetrics(response.data);
      } else {
        throw new Error(response.error || 'Failed to fetch metrics');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to fetch metrics:', err);
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Fetch system components
  const fetchComponents = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/system-health/components`;
      const response: APIResponse<Record<string, ComponentHealth>> = await apiRequest(url, { method: 'GET' });
      
      if (response.success && response.data) {
        setComponents(response.data);
      } else {
        throw new Error(response.error || 'Failed to fetch components');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to fetch components:', err);
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Execute system health action
  const executeAction = useCallback(async (action: SystemHealthAction): Promise<any> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/system-health/actions`;
      const response: APIResponse<any> = await apiRequest(url, {
        method: 'POST',
        body: JSON.stringify(action),
      });
      
      if (response.success) {
        // Refresh data after action
        await fetchSystemHealth();
        return response.data;
      } else {
        throw new Error(response.error || 'Failed to execute action');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to execute system health action:', err);
      throw err;
    } finally {
      setIsLoading(false);
    }
  }, [fetchSystemHealth]);

  // Resolve a specific alert
  const resolveAlert = useCallback(async (alertId: string): Promise<boolean> => {
    setIsLoading(true);
    setError(null);
    
    try {
      const url = `${API_BASE_URL}/api/system-health/alerts/${alertId}/resolve`;
      const response: APIResponse = await apiRequest(url, { method: 'POST' });
      
      if (response.success) {
        // Remove the alert from the active alerts list
        setAlerts(prevAlerts => prevAlerts.map(alert => 
          alert.id === alertId ? { ...alert, resolved: true, resolved_at: new Date().toISOString() } : alert
        ));
        return true;
      } else {
        throw new Error(response.error || 'Failed to resolve alert');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(errorMessage);
      console.error('Failed to resolve alert:', err);
      return false;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Convenience methods for common actions
  const runDiagnostics = useCallback(async (): Promise<DiagnosticsResult | null> => {
    try {
      const result = await executeAction({ action: 'run_diagnostics' });
      return result as DiagnosticsResult;
    } catch (err) {
      return null;
    }
  }, [executeAction]);

  const addAlert = useCallback(async (severity: string, component: string, message: string, metadata?: Record<string, any>): Promise<SystemAlert | null> => {
    try {
      const result = await executeAction({ 
        action: 'add_alert', 
        parameters: { severity, component, message, metadata } 
      });
      return result as SystemAlert;
    } catch (err) {
      return null;
    }
  }, [executeAction]);

  // WebSocket connection management
  const connectWebSocket = useCallback(() => {
    if (socket?.isConnected()) return;

    const ws = new StandardWebSocket();
    
    ws.onOpen = () => {
      console.log('System Health WebSocket connected');
      setIsConnected(true);
      setError(null);
      
      // Subscribe to system health updates
      ws.subscribe(['system-health-updated', 'system-alert-added', 'system-notification']);
    };

    ws.onMessage = (message) => {
      if (message.event === 'system-health-updated' && message.payload) {
        // Update system health data
        setSystemHealth(prevHealth => {
          if (!prevHealth) return prevHealth;
          return {
            ...prevHealth,
            overall_status: message.payload.overall_status || prevHealth.overall_status,
            metrics: message.payload.metrics || prevHealth.metrics,
            components: message.payload.components || prevHealth.components,
          };
        });
      } else if (message.event === 'system-alert-added' && message.payload) {
        // Add new alert
        setAlerts(prevAlerts => [message.payload, ...prevAlerts]);
      } else if (message.event === 'connected') {
        console.log('System Health WebSocket welcome:', message.payload);
      }
    };

    ws.onClose = () => {
      console.log('System Health WebSocket disconnected');
      setIsConnected(false);
    };

    ws.onError = (error) => {
      console.error('System Health WebSocket error:', error);
      setError('WebSocket connection failed');
    };

    setSocket(ws);
  }, [socket]);

  const disconnectWebSocket = useCallback(() => {
    if (socket) {
      socket.close();
      setSocket(null);
      setIsConnected(false);
    }
  }, [socket]);

  // Refresh all data
  const refreshAll = useCallback(async () => {
    await Promise.all([
      fetchSystemHealth(),
      fetchAlerts(),
      fetchMetrics(),
      fetchComponents(),
    ]);
  }, [fetchSystemHealth, fetchAlerts, fetchMetrics, fetchComponents]);

  // Initial fetch and WebSocket connection on mount
  useEffect(() => {
    fetchSystemHealth();
    connectWebSocket();
  }, [fetchSystemHealth, connectWebSocket]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      disconnectWebSocket();
    };
  }, [disconnectWebSocket]);

  return {
    systemHealth,
    alerts,
    metrics,
    components,
    isLoading,
    error,
    isConnected,
    fetchSystemHealth,
    fetchAlerts,
    fetchMetrics,
    fetchComponents,
    executeAction,
    resolveAlert,
    runDiagnostics,
    addAlert,
    refreshAll,
    connectWebSocket,
    disconnectWebSocket,
  };
};

export default useSystemHealth;
