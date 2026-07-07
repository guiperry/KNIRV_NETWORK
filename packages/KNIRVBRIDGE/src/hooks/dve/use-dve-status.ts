import { useState, useEffect, useCallback, useRef } from 'react';
import { DVE_CONSTANTS } from '@common/constants/dve.constant';
import { deriveDVEIdentity, DVEIdentity } from '@services/dve/dve-identity';
import { DVEWebSocketClient } from '@services/dve/dve-ws-client';

export interface DVEStatus {
  isActive: boolean;
  nodeID: string;
  dveURI: string;
  walletAddress: string;
  status: 'online' | 'offline' | 'connecting';
  badgeCount: number;
  totalCapabilities: number;
  tasksPending: number;
  tasksCompleted: number;
  tasksFailed: number;
  reputationScore: number;
  stakeAmount: number;
}

export interface TaskCounter {
  pending: number;
  completed: number;
  failed: number;
}

const DEFAULT_STATUS: DVEStatus = {
  isActive: false,
  nodeID: '',
  dveURI: '',
  walletAddress: '',
  status: 'offline',
  badgeCount: 0,
  totalCapabilities: 0,
  tasksPending: 0,
  tasksCompleted: 0,
  tasksFailed: 0,
  reputationScore: 0,
  stakeAmount: 0,
};

export function useDVEStatus(walletAddress: string | null): {
  status: DVEStatus;
  loading: boolean;
  error: Error | null;
  connect: () => Promise<void>;
  disconnect: () => void;
} {
  const [status, setStatus] = useState<DVEStatus>(DEFAULT_STATUS);
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<Error | null>(null);
  const identityRef = useRef<DVEIdentity | null>(null);
  const wsClientRef = useRef<DVEWebSocketClient | null>(null);

  const updateStatus = useCallback((partial: Partial<DVEStatus>) => {
    setStatus((prev) => ({ ...prev, ...partial }));
  }, []);

  const connect = useCallback(async () => {
    if (!walletAddress) {
      setError(new Error('No wallet address available'));
      return;
    }

    setLoading(true);
    setError(null);
    updateStatus({ status: 'connecting' });

    try {
      // Derive DVE identity from wallet address
      const identity = await deriveDVEIdentity(walletAddress);
      identityRef.current = identity;

      // Construct WebSocket URL from environment or defaults
      const serverURL =
        process.env.DVE_SERVER_URL || 'https://dve.knirv.network';
      const authToken =
        process.env.DVE_AUTH_TOKEN || '';

      // Initialize WebSocket client
      const wsClient = new DVEWebSocketClient(serverURL, authToken);

      // Listen for connection events
      wsClient.on('_connected', () => {
        updateStatus({
          isActive: true,
          status: 'online',
          nodeID: identity.nodeID,
          dveURI: identity.dveURI,
          walletAddress: identity.walletAddress,
        });
      });

      wsClient.on('_disconnected', () => {
        updateStatus({
          isActive: false,
          status: 'offline',
        });
      });

      wsClient.on('_error', () => {
        updateStatus({ status: 'offline' });
      });

      // Listen for task assignments
      wsClient.on('task_assigned', () => {
        setStatus((prev) => ({
          ...prev,
          tasksPending: prev.tasksPending + 1,
        }));
      });

      // Listen for heartbeat acknowledgements
      wsClient.on('heartbeat_ack', () => {
        updateStatus({ status: 'online' });
      });

      // Connect
      wsClient.connect();
      wsClientRef.current = wsClient;

      updateStatus({
        nodeID: identity.nodeID,
        dveURI: identity.dveURI,
        walletAddress: identity.walletAddress,
      });

      setLoading(false);
    } catch (err) {
      const error =
        err instanceof Error ? err : new Error('Failed to connect DVE node');
      setError(error);
      updateStatus({ status: 'offline' });
      setLoading(false);
    }
  }, [walletAddress, updateStatus]);

  const disconnect = useCallback(() => {
    if (wsClientRef.current) {
      wsClientRef.current.disconnect();
      wsClientRef.current = null;
    }
    identityRef.current = null;
    setStatus(DEFAULT_STATUS);
    setError(null);
  }, []);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (wsClientRef.current) {
        wsClientRef.current.disconnect();
        wsClientRef.current = null;
      }
    };
  }, []);

  return {
    status,
    loading,
    error,
    connect,
    disconnect,
  };
}

/**
 * Hook to track DVE task counts.
 * Can be used standalone or composed inside useDVEStatus.
 */
export function useDVETasks(walletAddress: string | null): {
  tasks: TaskCounter;
  loading: boolean;
} {
  const [tasks, setTasks] = useState<TaskCounter>({
    pending: 0,
    completed: 0,
    failed: 0,
  });
  const [loading, setLoading] = useState<boolean>(false);

  useEffect(() => {
    if (!walletAddress) {
      return;
    }

    setLoading(true);

    // Simulate fetching task data from the DVE server
    const fetchTasks = async () => {
      try {
        // In production, this would call the DVE registry API
        // For now, we load from chrome.storage.local
        const result = await new Promise<chrome.storage.StorageResult>((resolve) => {
          chrome.storage.local.get('DVE_TASKS', resolve);
        });
        const stored = result?.DVE_TASKS as TaskCounter | undefined;
        if (stored) {
          setTasks(stored);
        }
      } catch (err) {
        console.error('Failed to fetch DVE tasks:', err);
      } finally {
        setLoading(false);
      }
    };

    fetchTasks();

    // Poll for task updates periodically
    const interval = setInterval(fetchTasks, DVE_CONSTANTS.BADGE_SYNC_INTERVAL_MS);

    return () => {
      clearInterval(interval);
    };
  }, [walletAddress]);

  return {
    tasks,
    loading,
  };
}
