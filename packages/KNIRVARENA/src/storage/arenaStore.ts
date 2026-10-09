/**
 * Browser-local persistence for arena-owned data (§3.3 / D2).
 *
 * Server-authoritative data (error nodes, error-node tests, .nrv
 * datasets, badges) lives in KNIRVSERVER; this store holds only what
 * the browser owns: the personal graph, trained cortex models, agent
 * configs and local preferences. localStorage keeps these across
 * reloads (the arena already persists UI state this way); the Dexie /
 * IndexedDB cache for server-owned data lands with the arenaStore
 * hardening pass.
 */

const PREFIX = 'knirv.arena.';

const read = <T>(key: string): T | null => {
  try {
    const raw = localStorage.getItem(PREFIX + key);
    return raw ? (JSON.parse(raw) as T) : null;
  } catch {
    return null;
  }
};

const write = (key: string, value: unknown): void => {
  try {
    localStorage.setItem(PREFIX + key, JSON.stringify(value));
  } catch {
    // Quota or serialization failure: the session copy stays in memory.
  }
};

const remove = (key: string): void => {
  try {
    localStorage.removeItem(PREFIX + key);
  } catch {
    /* storage unavailable */
  }
};

const listCollection = <T>(key: string): T[] => read<T[]>(key) ?? [];

type AgentRecord = Record<string, unknown> & { agentId?: string };

const agentIdOf = (record: AgentRecord): string =>
  String(record.agentId ?? Math.random().toString(36));

let initialized = false;

export const arenaStore = {
  initialize(): Promise<void> {
    initialized = true;
    return Promise.resolve();
  },

  isInitialized(): boolean {
    return initialized;
  },

  // Personal graphs, keyed by user id.
  getPersonalGraph(userId: string): Promise<Record<string, unknown> | null> {
    return Promise.resolve(read<Record<string, unknown>>(`personalGraph:${userId}`));
  },

  savePersonalGraph(graph: Record<string, unknown>): Promise<void> {
    const userId = typeof graph.userId === 'string' ? graph.userId : 'current_user';
    write(`personalGraph:${userId}`, graph);
    return Promise.resolve();
  },

  // Trained cortex models.
  getAllCortexModels(): Promise<Record<string, unknown>[]> {
    return Promise.resolve(listCollection<Record<string, unknown>>('cortexModels'));
  },

  saveCortexModel(model: Record<string, unknown>): Promise<void> {
    const models = listCollection<Record<string, unknown>>('cortexModels');
    const id = String(model.id ?? model.modelId ?? '');
    const next = id
      ? models.filter(m => String(m.id ?? m.modelId ?? '') !== id)
      : models;
    next.push(model);
    write('cortexModels', next);
    return Promise.resolve();
  },

  // Agent configs.
  createAgent(agentData: AgentRecord): Promise<AgentRecord> {
    const agent: AgentRecord = {
      ...agentData,
      agentId: agentIdOf(agentData)
    };
    const agents = listCollection<AgentRecord>('agents');
    agents.push(agent);
    write('agents', agents);
    return Promise.resolve(agent);
  },

  getAgent(agentId: string): Promise<AgentRecord | null> {
    const agent = listCollection<AgentRecord>('agents').find(a => a.agentId === agentId) ?? null;
    return Promise.resolve(agent);
  },

  updateAgent(agentId: string, updateData: Record<string, unknown>): Promise<AgentRecord | null> {
    const agents = listCollection<AgentRecord>('agents');
    const index = agents.findIndex(a => a.agentId === agentId);
    if (index === -1) return Promise.resolve(null);
    const updated = { ...agents[index], ...updateData, agentId };
    agents[index] = updated;
    write('agents', agents);
    return Promise.resolve(updated);
  },

  listAgents(): Promise<AgentRecord[]> {
    return Promise.resolve(listCollection<AgentRecord>('agents'));
  },

  getAllAgents(): Promise<AgentRecord[]> {
    return Promise.resolve(listCollection<AgentRecord>('agents'));
  },

  deleteAgent(agentId: string): Promise<boolean> {
    const agents = listCollection<AgentRecord>('agents');
    const next = agents.filter(a => a.agentId !== agentId);
    const deleted = next.length !== agents.length;
    write('agents', next);
    return Promise.resolve(deleted);
  },

  // Local preferences (never server-owned).
  getPref<T>(key: string, fallback: T): T {
    const stored = read<unknown>(`pref:${key}`);
    return stored === null || stored === undefined ? fallback : (stored as T);
  },

  setPref(key: string, value: unknown): void {
    write(`pref:${key}`, value);
  },

  removePref(key: string): void {
    remove(`pref:${key}`);
  }
};

export default arenaStore;
