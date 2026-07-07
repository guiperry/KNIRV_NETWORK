import { useState, useEffect, useCallback, useRef } from 'react';
import { DVEBadge, DVEBadgeManager } from '@services/dve/dve-badge-manager';

export interface UseDVEBadgesReturn {
  badges: DVEBadge[];
  allBadges: DVEBadge[];
  loading: boolean;
  error: Error | null;
  aggregateCapabilities: string[];
  toggleBadge: (tokenID: string, active: boolean) => Promise<void>;
  refreshBadges: () => Promise<void>;
}

export function useDVEBadges(walletAddress: string | null): UseDVEBadgesReturn {
  const [badges, setBadges] = useState<DVEBadge[]>([]);
  const [allBadges, setAllBadges] = useState<DVEBadge[]>([]);
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<Error | null>(null);
  const [aggregateCapabilities, setAggregateCapabilities] = useState<string[]>([]);

  const managerRef = useRef<DVEBadgeManager | null>(null);

  // Initialize the badge manager
  useEffect(() => {
    const serverURL =
      process.env.DVE_SERVER_URL || 'https://dve.knirv.network';
    const manager = new DVEBadgeManager(serverURL);
    managerRef.current = manager;

    return () => {
      manager.destroy();
      managerRef.current = null;
    };
  }, []);

  // Refresh badges from the wallet address
  const refreshBadges = useCallback(async () => {
    if (!walletAddress || !managerRef.current) {
      return;
    }

    setLoading(true);
    setError(null);

    try {
      await managerRef.current.getBadgesFromWallet(walletAddress);
      const active = managerRef.current.getActiveBadges();
      const all = managerRef.current.getAllBadges();
      const caps = managerRef.current.computeAggregateCapabilities(all);

      setBadges(active);
      setAllBadges(all);
      setAggregateCapabilities(caps);
    } catch (err) {
      const error =
        err instanceof Error ? err : new Error('Failed to fetch DVE badges');
      setError(error);
    } finally {
      setLoading(false);
    }
  }, [walletAddress]);

  // Watch for badge changes
  useEffect(() => {
    if (!walletAddress || !managerRef.current) {
      return;
    }

    // Initial fetch
    refreshBadges();

    // Subscribe to badge changes
    const unsubscribe = managerRef.current.watchBadgeChanges(
      (updatedBadges: DVEBadge[]) => {
        const active = updatedBadges.filter((b) => b.active);
        const caps = managerRef.current
          ? managerRef.current.computeAggregateCapabilities(updatedBadges)
          : [];

        setBadges(active);
        setAllBadges(updatedBadges);
        setAggregateCapabilities(caps);
      },
    );

    return () => {
      unsubscribe();
    };
  }, [walletAddress, refreshBadges]);

  // Toggle a badge's active state
  const toggleBadge = useCallback(
    async (tokenID: string, active: boolean) => {
      if (!managerRef.current) {
        return;
      }

      try {
        await managerRef.current.toggleBadgeActive(tokenID, active);

        // Update local state immediately
        const updatedAll = allBadges.map((badge) =>
          badge.nftTokenID === tokenID ? { ...badge, active } : badge,
        );
        const updatedActive = updatedAll.filter((b) => b.active);
        const updatedCaps =
          managerRef.current.computeAggregateCapabilities(updatedAll);

        setAllBadges(updatedAll);
        setBadges(updatedActive);
        setAggregateCapabilities(updatedCaps);
      } catch (err) {
        const error =
          err instanceof Error
            ? err
            : new Error(`Failed to toggle badge ${tokenID}`);
        setError(error);
      }
    },
    [allBadges],
  );

  return {
    badges,
    allBadges,
    loading,
    error,
    aggregateCapabilities,
    toggleBadge,
    refreshBadges,
  };
}

/**
 * Compute the aggregate stake requirement from a badge list.
 * Only active badges contribute to the stake.
 */
export function useAggregateStake(badges: DVEBadge[]): number {
  return badges.reduce((total, badge) => {
    if (badge.active) {
      return total + badge.stakeRequirement;
    }
    return total;
  }, 0);
}
