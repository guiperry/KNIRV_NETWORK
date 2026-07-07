import { useState, useEffect } from 'react';

export interface NexusEvent {
  timestamp: number;
  agent_id: string;
  intent: string;
  observed_action: string;
  verified: boolean;
}

export function useNexusEvents() {
  const [events, setEvents] = useState<NexusEvent[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchEvents = async () => {
    try {
      const response = await fetch('/api/nexus/events');
      if (!response.ok) {
        throw new Error('Failed to fetch server events');
      }
      const data = await response.json();
      setEvents(data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchEvents();
    const interval = setInterval(fetchEvents, 5000);
    return () => clearInterval(interval);
  }, []);

  return { events, isLoading, error, refresh: fetchEvents };
}
