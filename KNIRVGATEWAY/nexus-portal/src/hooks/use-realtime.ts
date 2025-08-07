import { useState, useEffect, useRef, useCallback } from 'react';
import { RealtimeService, RealtimeMessage, RealtimeOptions } from '../lib/realtime-service';

export interface UseRealtimeReturn {
  connected: boolean;
  connectionType: 'sse' | 'websocket' | 'none';
  messages: RealtimeMessage[];
  lastMessage: RealtimeMessage | null;
  error: Error | null;
  send: (message: any) => void;
  reconnect: () => void;
  clearMessages: () => void;
}

export function useRealtime(options: RealtimeOptions): UseRealtimeReturn {
  const [connected, setConnected] = useState(false);
  const [messages, setMessages] = useState<RealtimeMessage[]>([]);
  const [lastMessage, setLastMessage] = useState<RealtimeMessage | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const serviceRef = useRef<RealtimeService | null>(null);

  // Initialize service
  useEffect(() => {
    const service = new RealtimeService(options);
    serviceRef.current = service;

    // Set up event handlers
    const unsubscribeConnection = service.onConnection((isConnected) => {
      setConnected(isConnected);
      if (isConnected) {
        setError(null);
      }
    });

    const unsubscribeMessage = service.onMessage((message) => {
      setMessages(prev => [...prev.slice(-99), message]); // Keep last 100 messages
      setLastMessage(message);
    });

    const unsubscribeError = service.onError((err) => {
      setError(err);
    });

    // Connect
    service.connect();

    // Cleanup
    return () => {
      unsubscribeConnection();
      unsubscribeMessage();
      unsubscribeError();
      service.disconnect();
    };
  }, [options.channel, options.token]);

  const send = useCallback((message: any) => {
    serviceRef.current?.send(message);
  }, []);

  const reconnect = useCallback(() => {
    serviceRef.current?.connect();
  }, []);

  const clearMessages = useCallback(() => {
    setMessages([]);
    setLastMessage(null);
  }, []);

  return {
    connected,
    connectionType: serviceRef.current?.connectionType || 'none',
    messages,
    lastMessage,
    error,
    send,
    reconnect,
    clearMessages
  };
}

// Specialized hooks for different NEXUS channels
export function useNexusDVE(token?: string) {
  return useRealtime({
    channel: 'dve',
    token
  });
}

export function useNexusValidation(token?: string) {
  return useRealtime({
    channel: 'validation',
    token
  });
}

export function useNexusSystem(token?: string) {
  return useRealtime({
    channel: 'system',
    token
  });
}

// Hook for filtering messages by type
export function useRealtimeMessages(
  options: RealtimeOptions,
  messageTypes?: string[]
): UseRealtimeReturn & { filteredMessages: RealtimeMessage[] } {
  const realtime = useRealtime(options);
  const [filteredMessages, setFilteredMessages] = useState<RealtimeMessage[]>([]);

  useEffect(() => {
    if (!messageTypes || messageTypes.length === 0) {
      setFilteredMessages(realtime.messages);
    } else {
      setFilteredMessages(
        realtime.messages.filter(msg => messageTypes.includes(msg.type))
      );
    }
  }, [realtime.messages, messageTypes]);

  return {
    ...realtime,
    filteredMessages
  };
}

// Hook for subscribing to specific message types
export function useRealtimeSubscription(
  options: RealtimeOptions,
  messageType: string,
  handler: (message: RealtimeMessage) => void
) {
  const { lastMessage } = useRealtime(options);

  useEffect(() => {
    if (lastMessage && lastMessage.type === messageType) {
      handler(lastMessage);
    }
  }, [lastMessage, messageType, handler]);
}
