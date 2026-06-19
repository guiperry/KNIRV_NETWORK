export interface DVEBadge {
  nftTokenID: string;
  collectionPath: string;
  capabilities: string[];
  supportedTags: string[];
  attachedPolicies: string[];
  stakeRequirement: number;
  trustTier: 'standard' | 'verified' | 'root';
  active: boolean;
}

/**
 * Manages DVE badges associated with the current wallet.
 * Badges are NFTs that grant capabilities and determine trust tier.
 */
export class DVEBadgeManager {
  private badges: Map<string, DVEBadge> = new Map();
  private watchCallbacks: Set<(badges: DVEBadge[]) => void> = new Set();
  private walletAddress: string = '';
  private refreshInterval: ReturnType<typeof setInterval> | null = null;
  private apiBaseURL: string;

  constructor(apiBaseURL: string = '') {
    this.apiBaseURL = apiBaseURL.replace(/\/+$/, '');
  }

  /**
   * Fetch all DVE badges associated with a wallet address.
   * Calls a badge-indexer API endpoint.
   */
  async getBadgesFromWallet(walletAddress: string): Promise<DVEBadge[]> {
    this.walletAddress = walletAddress;

    if (!walletAddress) {
      return [];
    }

    try {
      let badges: DVEBadge[];

      if (this.apiBaseURL) {
        badges = await this.fetchBadgesFromAPI(walletAddress);
      } else {
        badges = await this.fetchBadgesFromChain(walletAddress);
      }

      // Merge fetched badges with the stored map, preserving active state
      for (const badge of badges) {
        const existing = this.badges.get(badge.nftTokenID);
        if (existing) {
          badge.active = existing.active;
        }
        this.badges.set(badge.nftTokenID, badge);
      }

      // Remove badges no longer held by the wallet
      const fetchedIDs = new Set(badges.map((b) => b.nftTokenID));
      for (const tokenID of this.badges.keys()) {
        if (!fetchedIDs.has(tokenID)) {
          this.badges.delete(tokenID);
        }
      }

      return this.getActiveBadges();
    } catch (err) {
      console.error('Failed to fetch DVE badges from wallet:', err);
      return [];
    }
  }

  /**
   * Compute aggregate capabilities from a set of badges.
   * Deduplicates and flattens the capabilities arrays.
   */
  computeAggregateCapabilities(badges: DVEBadge[]): string[] {
    const capabilitySet = new Set<string>();

    for (const badge of badges) {
      if (badge.active) {
        for (const cap of badge.capabilities) {
          capabilitySet.add(cap);
        }
      }
    }

    return Array.from(capabilitySet).sort();
  }

  /**
   * Compute the total stake requirement from all active badges.
   */
  computeAggregateStake(badges: DVEBadge[]): number {
    let total = 0;

    for (const badge of badges) {
      if (badge.active) {
        total += badge.stakeRequirement;
      }
    }

    return total;
  }

  /**
   * Watch for badge changes by polling the badge source at a fixed interval.
   * Returns an unsubscribe function.
   */
  watchBadgeChanges(callback: (badges: DVEBadge[]) => void): () => void {
    this.watchCallbacks.add(callback);

    // Start polling if not already running
    if (this.refreshInterval === null && this.walletAddress) {
      this.startPolling();
    }

    return () => {
      this.watchCallbacks.delete(callback);
      if (this.watchCallbacks.size === 0) {
        this.stopPolling();
      }
    };
  }

  /**
   * Toggle a badge's active state.
   */
  async toggleBadgeActive(tokenID: string, active: boolean): Promise<void> {
    const badge = this.badges.get(tokenID);
    if (!badge) {
      throw new Error(`Badge ${tokenID} not found`);
    }

    badge.active = active;
    this.badges.set(tokenID, badge);

    // Notify watchers
    this.notifyWatchers();
  }

  /**
   * Get all badges that are currently active.
   */
  getActiveBadges(): DVEBadge[] {
    return Array.from(this.badges.values()).filter((b) => b.active);
  }

  /**
   * Get all badges (active and inactive).
   */
  getAllBadges(): DVEBadge[] {
    return Array.from(this.badges.values());
  }

  /**
   * Destroy the manager — stop polling and clear state.
   */
  destroy(): void {
    this.stopPolling();
    this.badges.clear();
    this.watchCallbacks.clear();
    this.walletAddress = '';
  }

  private async fetchBadgesFromAPI(walletAddress: string): Promise<DVEBadge[]> {
    const response = await fetch(
      `${this.apiBaseURL}/api/v1/dve/badges/${encodeURIComponent(walletAddress)}`,
    );

    if (!response.ok) {
      throw new Error(`Badge API returned ${response.status}`);
    }

    const data: DVEBadge[] = await response.json();
    return data;
  }

  private async fetchBadgesFromChain(_walletAddress: string): Promise<DVEBadge[]> {
    // Fallback: query local badge storage or chain RPC
    // In the browser extension context, we read from chrome.storage.local
    return new Promise((resolve) => {
      chrome.storage.local.get('DVE_BADGES', (result) => {
        const stored: DVEBadge[] = result?.DVE_BADGES || [];
        resolve(stored);
      });
    });
  }

  private startPolling(): void {
    this.stopPolling();
    this.refreshInterval = setInterval(async () => {
      if (!this.walletAddress) {
        return;
      }

      try {
        await this.getBadgesFromWallet(this.walletAddress);
        this.notifyWatchers();
      } catch (err) {
        console.error('DVE badge polling error:', err);
      }
    }, 300_000); // 5 minutes
  }

  private stopPolling(): void {
    if (this.refreshInterval) {
      clearInterval(this.refreshInterval);
      this.refreshInterval = null;
    }
  }

  private notifyWatchers(): void {
    const allBadges = Array.from(this.badges.values());
    for (const callback of this.watchCallbacks) {
      try {
        callback(allBadges);
      } catch (err) {
        console.error('DVE badge watcher callback error:', err);
      }
    }
  }
}
