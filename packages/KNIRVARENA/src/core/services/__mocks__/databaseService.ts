/**
 * Mock Database Service for Testing
 * Provides in-memory mock implementation of RxDB functionality
 */

export interface MockDocument {
  [key: string]: unknown;
  agentId?: string;
  id?: string;
  _id?: string;
}

export interface MockCollection {
  name: string;
  data: Map<string, MockDocument>;
  
  insert(doc: MockDocument): Promise<MockDocument>;
  findOne(selector?: Record<string, unknown>): { exec(): Promise<MockDocument | null> };
  find(selector?: Record<string, unknown>): { exec(): Promise<MockDocument[]> };
  upsert(doc: MockDocument): Promise<MockDocument>;
  remove(): Promise<void>;
  bulkInsert(docs: MockDocument[]): Promise<MockDocument[]>;
}

export interface MockDatabase {
  collections: Record<string, MockCollection>;
  destroy(): Promise<void>;
}

class MockRxCollection implements MockCollection {
  public data = new Map<string, MockDocument>();
  
  constructor(public name: string) {}
  
  async insert(doc: MockDocument): Promise<MockDocument> {
    const id = doc.agentId || doc.id || doc._id || Math.random().toString(36);
    const docWithId = { ...doc, _id: id };
    this.data.set(id as string, docWithId);
    return docWithId;
  }
  
  findOne(selector?: Record<string, unknown>) {
    return {
      exec: async (): Promise<MockDocument | null> => {
        if (!selector) {
          const firstEntry = this.data.values().next();
          return firstEntry.done ? null : firstEntry.value;
        }
        
        for (const doc of this.data.values()) {
          let matches = true;
          for (const [key, value] of Object.entries(selector)) {
            if (doc[key] !== value) {
              matches = false;
              break;
            }
          }
          if (matches) return doc;
        }
        return null;
      }
    };
  }
  
  find(selector?: Record<string, unknown>) {
    return {
      exec: async (): Promise<MockDocument[]> => {
        if (!selector) {
          return Array.from(this.data.values());
        }
        
        const results: MockDocument[] = [];
        for (const doc of this.data.values()) {
          let matches = true;
          for (const [key, value] of Object.entries(selector)) {
            if (doc[key] !== value) {
              matches = false;
              break;
            }
          }
          if (matches) results.push(doc);
        }
        return results;
      }
    };
  }
  
  async upsert(doc: MockDocument): Promise<MockDocument> {
    const id = doc.agentId || doc.id || doc._id || Math.random().toString(36);
    const docWithId = { ...doc, _id: id };
    this.data.set(id as string, docWithId);
    return docWithId;
  }
  
  async remove(): Promise<void> {
    this.data.clear();
  }
  
  async bulkInsert(docs: MockDocument[]): Promise<MockDocument[]> {
    const results: MockDocument[] = [];
    for (const doc of docs) {
      const result = await this.insert(doc);
      results.push(result);
    }
    return results;
  }
}

class MockRxDatabase implements MockDatabase {
  public collections: Record<string, MockCollection> = {};
  
  constructor() {
    // Initialize collections
    this.collections.agents = new MockRxCollection('agents');
    this.collections.personal_graphs = new MockRxCollection('personal_graphs');
    this.collections.error_clusters = new MockRxCollection('error_clusters');
    this.collections.skills = new MockRxCollection('skills');
    this.collections.wallet_data = new MockRxCollection('wallet_data');
  }
  
  async destroy(): Promise<void> {
    for (const collection of Object.values(this.collections)) {
      await collection.remove();
    }
    this.collections = {};
  }
}

// Mock database instance
let mockDatabase: MockRxDatabase | null = null;

export const createDatabase = async (): Promise<MockDatabase> => {
  if (!mockDatabase) {
    mockDatabase = new MockRxDatabase();
  }
  return mockDatabase;
};

export const getDatabase = (): MockDatabase | null => {
  return mockDatabase;
};

export const destroyDatabase = async (): Promise<void> => {
  if (mockDatabase) {
    await mockDatabase.destroy();
    mockDatabase = null;
  }
};

// Mock database service methods
export const createAgent = async (agentData: MockDocument): Promise<MockDocument> => {
  const database = await createDatabase();
  return await database.collections.agents.insert(agentData);
};

export const updateAgent = async (agentId: string, updateData: Partial<MockDocument>): Promise<MockDocument | null> => {
  const database = await createDatabase();
  const existingAgent = await database.collections.agents.findOne({ agentId }).exec();
  if (existingAgent) {
    const updatedAgent = { ...existingAgent, ...updateData };
    return await database.collections.agents.upsert(updatedAgent);
  }
  return null;
};

export const getAgent = async (agentId: string): Promise<MockDocument | null> => {
  const database = await createDatabase();
  return await database.collections.agents.findOne({ agentId }).exec();
};

export const getAllAgents = async (): Promise<MockDocument[]> => {
  const database = await createDatabase();
  return await database.collections.agents.find().exec();
};

// Mock database service instance
export const databaseService = {
  createAgent,
  updateAgent,
  getAgent,
  getAllAgents,
  createDatabase,
  getDatabase,
  destroyDatabase
};

// Export the mock database service
export default databaseService;
