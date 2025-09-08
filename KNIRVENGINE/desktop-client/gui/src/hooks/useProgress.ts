// React hooks for progress management
import { useState, useEffect, useCallback } from 'react';
import { progressManager, ProgressState, ProgressUpdate } from '../utils/progressManager';

// Hook for tracking a specific operation
export const useProgress = (operationId: string) => {
  const [state, setState] = useState<ProgressState | null>(() => 
    progressManager.getOperation(operationId)
  );

  useEffect(() => {
    // Get initial state
    const initialState = progressManager.getOperation(operationId);
    setState(initialState);

    // Subscribe to updates
    const unsubscribe = progressManager.subscribe(operationId, (newState) => {
      setState(newState);
    });

    return unsubscribe;
  }, [operationId]);

  const updateProgress = useCallback((update: Omit<ProgressUpdate, 'id'>) => {
    return progressManager.updateOperation(operationId, { ...update, id: operationId });
  }, [operationId]);

  const complete = useCallback((message?: string) => {
    return progressManager.completeOperation(operationId, message);
  }, [operationId]);

  const fail = useCallback((message?: string, details?: string) => {
    return progressManager.failOperation(operationId, message, details);
  }, [operationId]);

  const cancel = useCallback((message?: string) => {
    return progressManager.cancelOperation(operationId, message);
  }, [operationId]);

  return {
    state,
    updateProgress,
    complete,
    fail,
    cancel,
    isActive: state?.status === 'pending' || state?.status === 'running',
    isCompleted: state?.status === 'completed',
    isFailed: state?.status === 'failed',
    isCancelled: state?.status === 'cancelled',
    estimatedTimeRemaining: state ? progressManager.getEstimatedTimeRemaining(operationId) : null,
    duration: state ? progressManager.getDuration(operationId) : null,
  };
};

// Hook for tracking all operations
export const useAllProgress = () => {
  const [operations, setOperations] = useState<ProgressState[]>(() => 
    progressManager.getAllOperations()
  );

  useEffect(() => {
    // Get initial state
    setOperations(progressManager.getAllOperations());

    // Subscribe to all updates
    const unsubscribe = progressManager.subscribeGlobal(() => {
      setOperations(progressManager.getAllOperations());
    });

    return unsubscribe;
  }, []);

  return {
    operations,
    activeOperations: operations.filter(op => op.status === 'pending' || op.status === 'running'),
    completedOperations: operations.filter(op => op.status === 'completed'),
    failedOperations: operations.filter(op => op.status === 'failed'),
    cancelledOperations: operations.filter(op => op.status === 'cancelled'),
  };
};

// Hook for tracking operations by type
export const useProgressByType = (type: ProgressState['type']) => {
  const [operations, setOperations] = useState<ProgressState[]>(() => 
    progressManager.getOperationsByType(type)
  );

  useEffect(() => {
    // Get initial state
    setOperations(progressManager.getOperationsByType(type));

    // Subscribe to all updates and filter by type
    const unsubscribe = progressManager.subscribeGlobal(() => {
      setOperations(progressManager.getOperationsByType(type));
    });

    return unsubscribe;
  }, [type]);

  return {
    operations,
    activeOperations: operations.filter(op => op.status === 'pending' || op.status === 'running'),
    completedOperations: operations.filter(op => op.status === 'completed'),
    failedOperations: operations.filter(op => op.status === 'failed'),
    cancelledOperations: operations.filter(op => op.status === 'cancelled'),
  };
};

// Hook for tracking active operations count
export const useActiveProgressCount = () => {
  const [count, setCount] = useState(() => 
    progressManager.getActiveOperations().length
  );

  useEffect(() => {
    const updateCount = () => {
      setCount(progressManager.getActiveOperations().length);
    };

    // Update count on any operation change
    const unsubscribe = progressManager.subscribeGlobal(updateCount);

    return unsubscribe;
  }, []);

  return count;
};

// Hook for creating and managing a new operation
export const useCreateProgress = () => {
  const startOperation = useCallback((
    id: string,
    type: ProgressState['type'],
    message: string,
    estimatedDuration?: number
  ) => {
    return progressManager.startOperation(id, type, message, estimatedDuration);
  }, []);

  const updateOperation = useCallback((id: string, update: Omit<ProgressUpdate, 'id'>) => {
    return progressManager.updateOperation(id, { ...update, id });
  }, []);

  const completeOperation = useCallback((id: string, message?: string) => {
    return progressManager.completeOperation(id, message);
  }, []);

  const failOperation = useCallback((id: string, message?: string, details?: string) => {
    return progressManager.failOperation(id, message, details);
  }, []);

  const cancelOperation = useCallback((id: string, message?: string) => {
    return progressManager.cancelOperation(id, message);
  }, []);

  return {
    startOperation,
    updateOperation,
    completeOperation,
    failOperation,
    cancelOperation,
  };
};

// Utility hook for formatting progress information
export const useProgressFormatter = () => {
  const formatDuration = useCallback((milliseconds: number) => {
    const seconds = Math.floor(milliseconds / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);

    if (hours > 0) {
      return `${hours}h ${minutes % 60}m ${seconds % 60}s`;
    } else if (minutes > 0) {
      return `${minutes}m ${seconds % 60}s`;
    } else {
      return `${seconds}s`;
    }
  }, []);

  const formatTimeRemaining = useCallback((milliseconds: number | null) => {
    if (milliseconds === null) return 'Unknown';
    if (milliseconds <= 0) return 'Almost done';
    
    const seconds = Math.ceil(milliseconds / 1000);
    const minutes = Math.ceil(seconds / 60);

    if (minutes > 1) {
      return `~${minutes} minutes`;
    } else {
      return `~${seconds} seconds`;
    }
  }, []);

  const getStatusColor = useCallback((status: ProgressState['status']) => {
    switch (status) {
      case 'pending': return 'text-yellow-600';
      case 'running': return 'text-blue-600';
      case 'completed': return 'text-green-600';
      case 'failed': return 'text-red-600';
      case 'cancelled': return 'text-gray-600';
      default: return 'text-gray-600';
    }
  }, []);

  const getStatusIcon = useCallback((status: ProgressState['status']) => {
    switch (status) {
      case 'pending': return '⏳';
      case 'running': return '🔄';
      case 'completed': return '✅';
      case 'failed': return '❌';
      case 'cancelled': return '⏹️';
      default: return '❓';
    }
  }, []);

  return {
    formatDuration,
    formatTimeRemaining,
    getStatusColor,
    getStatusIcon,
  };
};
