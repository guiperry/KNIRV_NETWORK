// React hooks for error inference system
import { useState, useEffect, useCallback } from 'react';
import { errorHandler, ErrorInfo } from '../utils/errorHandler';

// Hook for managing error inference state
export const useErrorInference = () => {
  const [errors, setErrors] = useState<ErrorInfo[]>([]);
  const [errorsWithInference, setErrorsWithInference] = useState<ErrorInfo[]>([]);
  const [errorsNeedingInference, setErrorsNeedingInference] = useState<ErrorInfo[]>([]);
  const [isInferenceEnabled, setIsInferenceEnabled] = useState(true);

  useEffect(() => {
    // Load initial errors
    updateErrorStates();

    // Subscribe to new errors
    const unsubscribeErrors = errorHandler.subscribe(() => {
      updateErrorStates();
    });

    // Subscribe to inference results
    const unsubscribeInference = errorHandler.subscribeToInference(() => {
      updateErrorStates();
    });

    return () => {
      unsubscribeErrors();
      unsubscribeInference();
    };
  }, []);

  const updateErrorStates = useCallback(() => {
    setErrors(errorHandler.getAllErrors());
    setErrorsWithInference(errorHandler.getErrorsWithInference());
    setErrorsNeedingInference(errorHandler.getErrorsNeedingInference());
  }, []);

  const requestInference = useCallback(async (errorId: string) => {
    await errorHandler.requestInference(errorId);
    updateErrorStates();
  }, [updateErrorStates]);

  const toggleInference = useCallback((enabled: boolean) => {
    errorHandler.setInferenceEnabled(enabled);
    setIsInferenceEnabled(enabled);
  }, []);

  const setAutoInferenceThreshold = useCallback((severities: ErrorInfo['severity'][]) => {
    errorHandler.setAutoInferenceThreshold(severities);
  }, []);

  const clearError = useCallback((errorId: string) => {
    errorHandler.clearError(errorId);
    updateErrorStates();
  }, [updateErrorStates]);

  const clearAllErrors = useCallback(() => {
    errorHandler.clearAllErrors();
    updateErrorStates();
  }, [updateErrorStates]);

  return {
    errors,
    errorsWithInference,
    errorsNeedingInference,
    isInferenceEnabled,
    requestInference,
    toggleInference,
    setAutoInferenceThreshold,
    clearError,
    clearAllErrors,
    criticalErrors: errors.filter(e => e.severity === 'critical'),
    highSeverityErrors: errors.filter(e => e.severity === 'high' || e.severity === 'critical'),
    recentErrors: errors.filter(e => Date.now() - e.timestamp.getTime() < 5 * 60 * 1000), // Last 5 minutes
  };
};

// Hook for tracking error statistics
export const useErrorStats = () => {
  const [stats, setStats] = useState({
    total: 0,
    critical: 0,
    high: 0,
    medium: 0,
    low: 0,
    withInference: 0,
    needingInference: 0,
    recent: 0,
  });

  useEffect(() => {
    const updateStats = () => {
      const allErrors = errorHandler.getAllErrors();
      const now = Date.now();
      const fiveMinutesAgo = now - 5 * 60 * 1000;

      setStats({
        total: allErrors.length,
        critical: allErrors.filter(e => e.severity === 'critical').length,
        high: allErrors.filter(e => e.severity === 'high').length,
        medium: allErrors.filter(e => e.severity === 'medium').length,
        low: allErrors.filter(e => e.severity === 'low').length,
        withInference: errorHandler.getErrorsWithInference().length,
        needingInference: errorHandler.getErrorsNeedingInference().length,
        recent: allErrors.filter(e => e.timestamp.getTime() > fiveMinutesAgo).length,
      });
    };

    // Initial update
    updateStats();

    // Subscribe to changes
    const unsubscribeErrors = errorHandler.subscribe(updateStats);
    const unsubscribeInference = errorHandler.subscribeToInference(updateStats);

    // Update every minute
    const interval = setInterval(updateStats, 60000);

    return () => {
      unsubscribeErrors();
      unsubscribeInference();
      clearInterval(interval);
    };
  }, []);

  return stats;
};

// Hook for error notification management
export const useErrorNotifications = () => {
  const [notifications, setNotifications] = useState<ErrorInfo[]>([]);
  const [hasUnread, setHasUnread] = useState(false);

  useEffect(() => {
    // Subscribe to new errors for notifications
    const unsubscribe = errorHandler.subscribe((error) => {
      // Only notify for high severity errors
      if (error.severity === 'high' || error.severity === 'critical') {
        setNotifications(prev => [error, ...prev.slice(0, 4)]); // Keep last 5
        setHasUnread(true);
      }
    });

    return unsubscribe;
  }, []);

  const markAsRead = useCallback(() => {
    setHasUnread(false);
  }, []);

  const dismissNotification = useCallback((errorId: string) => {
    setNotifications(prev => prev.filter(n => n.id !== errorId));
  }, []);

  const clearNotifications = useCallback(() => {
    setNotifications([]);
    setHasUnread(false);
  }, []);

  return {
    notifications,
    hasUnread,
    markAsRead,
    dismissNotification,
    clearNotifications,
  };
};

// Hook for error recovery actions
export const useErrorRecovery = () => {
  const [recoveryInProgress, setRecoveryInProgress] = useState<Set<string>>(new Set());

  const attemptRecovery = useCallback(async (errorId: string) => {
    if (recoveryInProgress.has(errorId)) {
      return false;
    }

    setRecoveryInProgress(prev => new Set(prev).add(errorId));

    try {
      const success = await errorHandler.recover(errorId);
      return success;
    } catch (error) {
      console.error('Recovery attempt failed:', error);
      return false;
    } finally {
      setRecoveryInProgress(prev => {
        const newSet = new Set(prev);
        newSet.delete(errorId);
        return newSet;
      });
    }
  }, [recoveryInProgress]);

  const isRecovering = useCallback((errorId: string) => {
    return recoveryInProgress.has(errorId);
  }, [recoveryInProgress]);

  return {
    attemptRecovery,
    isRecovering,
  };
};

// Hook for error analysis chat
export const useErrorChat = (errorId: string) => {
  const [messages, setMessages] = useState<Array<{
    type: 'user' | 'assistant' | 'system';
    content: string;
    timestamp: Date;
    metadata?: any;
  }>>([]);
  const [isLoading, setIsLoading] = useState(false);

  const sendMessage = useCallback(async (content: string) => {
    if (!content.trim()) return;

    // Add user message
    const userMessage = {
      type: 'user' as const,
      content,
      timestamp: new Date(),
    };
    setMessages(prev => [...prev, userMessage]);

    setIsLoading(true);

    try {
      // Get the correct API base URL for Electron vs web
      const apiBaseUrl = typeof window !== 'undefined' && (window as any).electronAPI?.isElectron
        ? 'http://localhost:8081'
        : '';

      // Call chat API for follow-up questions
      const response = await fetch(`${apiBaseUrl}/api/v1/inference/chat-error`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          error_id: errorId,
          message: content,
          conversation_history: messages,
        }),
      });

      if (response.ok) {
        const result = await response.json();
        const assistantMessage = {
          type: 'assistant' as const,
          content: result.response || 'I apologize, but I could not process your request.',
          timestamp: new Date(),
          metadata: result.metadata,
        };
        setMessages(prev => [...prev, assistantMessage]);
      } else {
        throw new Error('Chat API failed');
      }
    } catch (error) {
      console.error('Chat error:', error);
      const errorMessage = {
        type: 'system' as const,
        content: 'Sorry, I encountered an error while processing your message. Please try again.',
        timestamp: new Date(),
      };
      setMessages(prev => [...prev, errorMessage]);
    } finally {
      setIsLoading(false);
    }
  }, [errorId, messages]);

  const clearChat = useCallback(() => {
    setMessages([]);
  }, []);

  return {
    messages,
    isLoading,
    sendMessage,
    clearChat,
  };
};
