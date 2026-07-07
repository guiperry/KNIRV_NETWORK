'use client';

import { useState, useCallback } from 'react';

interface BadgeAttribute {
  key: string;
  value: string;
}

interface BadgeCreateRequest {
  name: string;
  description: string;
  image_url: string;
  attributes: BadgeAttribute[];
}

interface BadgeMintRequest {
  badge_id: string;
  recipient: string;
  metadata: Record<string, string>;
}

interface BadgeInfo {
  id: string;
  name: string;
  description: string;
  image_url: string;
  attributes: BadgeAttribute[];
  total_minted: number;
}

interface UseBadgeLabReturn {
  createBadge: (req: BadgeCreateRequest) => Promise<BadgeInfo | null>;
  mintBadge: (req: BadgeMintRequest) => Promise<string | null>;
  getBadge: (id: string) => Promise<BadgeInfo | null>;
  isLoading: boolean;
  error: string | null;
}

export function useBadgeLab(): UseBadgeLabReturn {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const createBadge = useCallback(async (req: BadgeCreateRequest): Promise<BadgeInfo | null> => {
    setIsLoading(true);
    setError(null);

    try {
      const response = await fetch('/api/knirvshell/chain/badge/create', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(req),
      });

      if (!response.ok) {
        const err = await response.json();
        throw new Error(err.message || 'Failed to create badge');
      }

      const data: BadgeInfo = await response.json();
      return data;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  const mintBadge = useCallback(async (req: BadgeMintRequest): Promise<string | null> => {
    setIsLoading(true);
    setError(null);

    try {
      const response = await fetch('/api/knirvshell/chain/badge/mint', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(req),
      });

      if (!response.ok) {
        const err = await response.json();
        throw new Error(err.message || 'Failed to mint badge');
      }

      const data = await response.json();
      return data.token_id || data.tx_hash || null;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  const getBadge = useCallback(async (id: string): Promise<BadgeInfo | null> => {
    setIsLoading(true);
    setError(null);

    try {
      const response = await fetch(`/api/knirvshell/chain/badge/${id}`, {
        method: 'GET',
        headers: { 'Content-Type': 'application/json' },
      });

      if (!response.ok) {
        const err = await response.json();
        throw new Error(err.message || 'Failed to get badge');
      }

      const data: BadgeInfo = await response.json();
      return data;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  return {
    createBadge,
    mintBadge,
    getBadge,
    isLoading,
    error,
  };
}