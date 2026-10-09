import { useState, useEffect, useCallback } from 'react';
import { arenaStore } from '../storage/arenaStore';

// Mock UserProfiles for now - replace with actual implementation
const UserProfiles = {
  findOne: async (_query: unknown) => null,
  updateOne: async (_query: unknown, _update: unknown, _options: unknown) => {},
  deleteOne: async (_query: unknown) => {}
};

// Mock AgentCache for now - replace with actual implementation
const AgentCache = {
  find: async (_query: unknown) => [],
  insertOne: async (_data: unknown) => {},
  deleteMany: async (_query: unknown) => {}
};

// Mock SkillCache for now - replace with actual implementation
const SkillCache = {
  find: async (_query: unknown) => [],
  insertOne: async (_data: unknown) => {},
  deleteMany: async (_query: unknown) => {}
};

export function useClientCache(userId: string) {
  const [profile, setProfile] = useState<unknown>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const refreshProfile = useCallback(async (forceRefresh = false) => {
    if (!userId) return;

    setIsLoading(true);
    setError(null);

    try {
      // 1. Try to get from local cache first
      let cachedProfile = await UserProfiles.findOne({ userId });

      // 2. Check if cache is stale (5 minutes)
      const isStale = !cachedProfile ||
        (cachedProfile && (cachedProfile as { lastFetched?: Date }).lastFetched &&
         (new Date().getTime() - (cachedProfile as { lastFetched: Date }).lastFetched.getTime()) > 5 * 60 * 1000);

      if (isStale || forceRefresh) {
        try {
          // §3.1: GET /api/users/:id never existed on KNIRVSERVER —
          // the authenticated /api/auth/me returns the bearer token's
          // own user.
          const response = await fetch('/api/auth/me', { credentials: 'include' });
          if (response.ok) {
            const serverProfile = await response.json();

            // 3. Update cache
            await UserProfiles.updateOne(
              { userId },
              { ...serverProfile, lastFetched: new Date() },
              { upsert: true }
            );
            cachedProfile = await UserProfiles.findOne({ userId });
          } else if (!cachedProfile) {
            throw new Error('Failed to fetch profile and no cache available');
          }
          // If we have cached data and server fails, use cached data
        } catch (fetchError) {
          if (!cachedProfile) {
            throw fetchError;
          }
          // Use stale cache if server is unavailable
          console.warn('Using stale cache due to server error:', fetchError);
        }
      }

      setProfile(cachedProfile);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setIsLoading(false);
    }
  }, [userId]);

  useEffect(() => {
    refreshProfile();
  }, [refreshProfile]);

  return {
    profile,
    isLoading,
    error,
    refreshProfile: () => refreshProfile(true),
    clearCache: async () => {
      await UserProfiles.deleteOne({ userId });
      setProfile(null);
    }
  };
}

// Hook for caching agents
export function useAgentCache() {
  const [agents, setAgents] = useState<unknown[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const refreshAgents = useCallback(async (forceRefresh = false) => {
    setIsLoading(true);
    setError(null);

    try {
      let cachedAgents = await AgentCache.find({});

      // Check if any cached agent is stale (10 minutes for agents)
      const now = new Date().getTime();
      const isStale = cachedAgents.length === 0 ||
        cachedAgents.some((agent: { lastFetched?: Date }) => agent.lastFetched && (now - agent.lastFetched.getTime()) > 10 * 60 * 1000);

      if (isStale || forceRefresh) {
        try {
          // §3.1: GET /api/agents never existed on KNIRVSERVER. Agents
          // arrive from the paired KNIRVCLI via the arena relay
          // (§2.4, arena.status over the CLI tunnel); until the relay
          // lands, the browser-side agent store (arenaStore) is the
          // source and no server fetch is made.
          const serverAgents = await arenaStore.listAgents();
          if (serverAgents.length > 0) {
            // Clear old cache and update with fresh data
            await AgentCache.deleteMany({});

            for (const agent of serverAgents) {
              await AgentCache.insertOne({
                agentId: (agent as { agentId?: string }).agentId || (agent as { id?: string }).id,
                name: (agent as { name?: string }).name,
                type: (agent as { type?: string }).type,
                status: (agent as { status?: string }).status,
                metadata: (agent as { metadata?: unknown }).metadata,
                lastFetched: new Date()
              });
            }

            cachedAgents = await AgentCache.find({});
          } else if (cachedAgents.length === 0) {
            throw new Error('Failed to fetch agents and no cache available');
          }
        } catch (fetchError) {
          if (cachedAgents.length === 0) {
            throw fetchError;
          }
          console.warn('Using stale agent cache due to server error:', fetchError);
        }
      }

      setAgents(cachedAgents);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    refreshAgents();
  }, [refreshAgents]);

  return {
    agents,
    isLoading,
    error,
    refreshAgents: () => refreshAgents(true),
    clearCache: async () => {
      await AgentCache.deleteMany({});
      setAgents([]);
    }
  };
}

// Hook for caching skills
export function useSkillCache() {
  const [skills, setSkills] = useState<unknown[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const refreshSkills = useCallback(async (forceRefresh = false) => {
    setIsLoading(true);
    setError(null);

    try {
      let cachedSkills = await SkillCache.find({});

      // Check if any cached skill is stale (15 minutes for skills)
      const now = new Date().getTime();
      const isStale = cachedSkills.length === 0 ||
        cachedSkills.some((skill: { lastFetched?: Date }) => skill.lastFetched && (now - skill.lastFetched.getTime()) > 15 * 60 * 1000);

      if (isStale || forceRefresh) {
        try {
          // §3.1: GET /api/skills never existed on KNIRVSERVER. Badge
          // credentials cover owned skills: /api/badge-credentials
          // lists the caller's issued badges (skills they hold).
          const response = await fetch('/api/badge-credentials', { credentials: 'include' });
          if (response.ok) {
            const body = await response.json();
            const credentials = Array.isArray(body)
              ? body
              : ((body as { badges?: unknown[]; credentials?: unknown[] }).badges
                  ?? (body as { credentials?: unknown[] }).credentials
                  ?? []);
            const serverSkills = credentials.map((badge: Record<string, unknown>) => ({
              skillId: String(badge.id ?? badge.badgeId ?? badge.credentialId ?? ''),
              name: String(badge.name ?? badge.badgeName ?? badge.skill ?? ''),
              description: String(badge.description ?? ''),
              version: String(badge.version ?? '1')
            }));

            // Clear old cache and update with fresh data
            await SkillCache.deleteMany({});

            for (const skill of serverSkills) {
              await SkillCache.insertOne({
                skillId: skill.skillId,
                name: skill.name,
                description: skill.description,
                version: skill.version,
                lastFetched: new Date()
              });
            }

            cachedSkills = await SkillCache.find({});
          } else if (cachedSkills.length === 0) {
            throw new Error('Failed to fetch skills and no cache available');
          }
        } catch (fetchError) {
          if (cachedSkills.length === 0) {
            throw fetchError;
          }
          console.warn('Using stale skill cache due to server error:', fetchError);
        }
      }

      setSkills(cachedSkills);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setIsLoading(false);
    }
  }, []);

  const searchSkills = useCallback(async (term: string) => {
    const allSkills = await SkillCache.find({});
    return allSkills.filter((skill: { name?: string; description?: string }) =>
      skill.name?.toLowerCase().includes(term.toLowerCase()) ||
      skill.description?.toLowerCase().includes(term.toLowerCase())
    );
  }, []);

  useEffect(() => {
    refreshSkills();
  }, [refreshSkills]);

  return {
    skills,
    isLoading,
    error,
    refreshSkills: () => refreshSkills(true),
    searchSkills,
    clearCache: async () => {
      await SkillCache.deleteMany({});
      setSkills([]);
    }
  };
}
