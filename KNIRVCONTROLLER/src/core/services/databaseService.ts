import {
  createRxDatabase,
  RxDatabase,
  RxCollection,
  // RxDocument,
  addRxPlugin
} from 'rxdb';
import { RxDBDevModePlugin, disableWarnings } from 'rxdb/plugins/dev-mode';
import { getRxStorageDexie } from 'rxdb/plugins/storage-dexie';
import { getRxStorageMemory } from 'rxdb/plugins/storage-memory';
import { RxDBQueryBuilderPlugin } from 'rxdb/plugins/query-builder';
import { wrappedValidateAjvStorage } from 'rxdb/plugins/validate-ajv';

// Add RxDB plugins conditionally
if (process.env.NODE_ENV !== 'test') {
  addRxPlugin(RxDBDevModePlugin);
}
addRxPlugin(RxDBQueryBuilderPlugin);

// Disable dev-mode warnings in development
disableWarnings();

// Schema definitions
const agentSchema = {
  version: 0,
  primaryKey: 'agentId',
  type: 'object',
  properties: {
    agentId: {
      type: 'string',
      maxLength: 100
    },
    name: {
      type: 'string'
    },
    version: {
      type: 'string'
    },
    baseModelId: {
      type: 'string'
    },
    type: {
      type: 'string'
    },
    status: {
      type: 'string'
    },
    nrnCost: {
      type: 'number'
    },
    capabilities: {
      type: 'array',
      items: {
        type: 'string'
      }
    },
    metadata: {
      type: 'object',
      additionalProperties: false
    },
    wasmModule: {
      type: 'string'
    },
    loraAdapter: {
      type: 'string'
    },
    createdAt: {
      type: 'string',
      format: 'date-time'
    },
    lastActivity: {
      type: 'string',
      format: 'date-time'
    }
  },
  required: ['agentId', 'name', 'version', 'type', 'status', 'nrnCost'],
  additionalProperties: false
} as const;

const skillSchema = {
  version: 0,
  primaryKey: 'skillId',
  type: 'object',
  properties: {
    skillId: {
      type: 'string',
      maxLength: 100
    },
    name: {
      type: 'string'
    },
    description: {
      type: 'string'
    },
    loraAdapter: {
      type: 'object',
      additionalProperties: false
    },
    version: {
      type: 'number'
    },
    createdAt: {
      type: 'string',
      format: 'date-time'
    },
    updatedAt: {
      type: 'string',
      format: 'date-time'
    }
  },
  required: ['skillId', 'name'],
  additionalProperties: false
} as const;

const chatSessionSchema = {
  version: 0,
  primaryKey: 'id',
  type: 'object',
  properties: {
    id: {
      type: 'string',
      maxLength: 100
    },
    title: {
      type: 'string'
    },
    messages: {
      type: 'array',
      items: {
        type: 'object',
        additionalProperties: false
      }
    },
    createdAt: {
      type: 'string',
      format: 'date-time'
    },
    updatedAt: {
      type: 'string',
      format: 'date-time'
    }
  },
  required: ['id', 'title'],
  additionalProperties: false
} as const;

// Type definitions
type AgentDocType = {
  agentId: string;
  name: string;
  version: string;
  baseModelId?: string;
  type: string;
  status: string;
  nrnCost: number;
  capabilities?: string[];
  metadata?: Record<string, unknown>;
  wasmModule?: string;
  loraAdapter?: string;
  createdAt?: string;
  lastActivity?: string;
};

type SkillDocType = {
  skillId: string;
  name: string;
  description?: string;
  loraAdapter?: Record<string, unknown>;
  version?: number;
  createdAt?: string;
  updatedAt?: string;
};

type ChatSessionDocType = {
  id: string;
  title: string;
  messages?: Array<{
    id: string;
    content: string;
    sender: string;
    timestamp: string;
  }>;
  createdAt?: string;
  updatedAt?: string;
};

type DatabaseCollections = {
  agents: RxCollection<AgentDocType>;
  skills: RxCollection<SkillDocType>;
  chatSessions: RxCollection<ChatSessionDocType>;
};

type DatabaseType = RxDatabase<DatabaseCollections>;

// Initialize database
let database: DatabaseType | null = null;

const initDatabase = async (): Promise<DatabaseType> => {
  if (database) {
    return database;
  }

  database = await createRxDatabase<DatabaseCollections>({
    name: 'knirvcontroller',
    storage: process.env.NODE_ENV === 'test'
      ? getRxStorageMemory()
      : wrappedValidateAjvStorage({
          storage: getRxStorageDexie()
        }),
    ignoreDuplicate: true
  });

  // Add collections
  await database.addCollections({
    agents: {
      schema: agentSchema
    },
    skills: {
      schema: skillSchema
    },
    chatSessions: {
      schema: chatSessionSchema
    }
  });

  console.log('RxDB Database initialized successfully');

  // Initialize with some sample data if collections are empty
  await initializeSampleData(database);

  return database;
};

// Initialize sample data if collections are empty
async function initializeSampleData(database: DatabaseType): Promise<void> {
  try {
    // Check if collections have data
    const agentCount = await database.agents.count().exec();
    const skillCount = await database.skills.count().exec();

    if (agentCount === 0) {
      console.log('Initializing sample agent data...');
      // Add sample agents if needed
    }

    if (skillCount === 0) {
      console.log('Initializing sample skill data...');
      // Add sample skills if needed
    }
  } catch (error) {
    console.warn('Failed to initialize sample data:', error);
  }
}

// Collection wrapper to match expected API
class CollectionWrapper<T> {
  constructor(
    private getCollection: () => Promise<RxCollection<T>>,
    private primaryKey: string
  ) {}

  async insertOne(doc: Partial<T> & Record<string, unknown>) {
    const collection = await this.getCollection();

    // Create a mutable copy with timestamps
    const docWithTimestamps = {
      ...doc,
      createdAt: (doc as { createdAt?: string }).createdAt || new Date().toISOString(),
      updatedAt: new Date().toISOString()
    };

    // Generate ID if not present
    if (!(docWithTimestamps as Record<string, unknown>)[this.primaryKey]) {
      (docWithTimestamps as Record<string, unknown>)[this.primaryKey] = `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
    }

    const result = await collection.insert(docWithTimestamps as T);
    return result.toJSON();
  }

  async findOne(query: Record<string, unknown>) {
    const collection = await this.getCollection();
    const result = await collection.findOne(query).exec();
    return result ? result.toJSON() : null;
  }

  async find(query: Record<string, unknown> = {}) {
    const collection = await this.getCollection();
    const results = await collection.find(query).exec();
    return results.map(doc => doc.toJSON());
  }

  async updateOne(query: Record<string, unknown>, update: Record<string, unknown>) {
    const collection = await this.getCollection();
    const doc = await collection.findOne(query).exec();
    if (doc) {
      const updateSource = update.$set || update;
      const updateData = typeof updateSource === 'object' && updateSource !== null
        ? { ...updateSource as Record<string, unknown> }
        : {} as Record<string, unknown>;
      updateData.updatedAt = new Date().toISOString();
      await doc.patch(updateData as Partial<T>);
      return doc.toJSON();
    }
    return null;
  }

  async deleteOne(query: Record<string, unknown>) {
    const collection = await this.getCollection();
    const doc = await collection.findOne(query).exec();
    if (doc) {
      await doc.remove();
      return { deletedCount: 1 };
    }
    return { deletedCount: 0 };
  }

  async deleteMany(query: Record<string, unknown>) {
    const collection = await this.getCollection();
    const docs = await collection.find(query).exec();
    let deletedCount = 0;
    for (const doc of docs) {
      await doc.remove();
      deletedCount++;
    }
    return { deletedCount };
  }
}

// Export collection wrappers
export const Agents = new CollectionWrapper<AgentDocType>(
  async () => (await initDatabase()).agents,
  'agentId'
);

export const Skills = new CollectionWrapper<SkillDocType>(
  async () => (await initDatabase()).skills,
  'skillId'
);

export const ChatSessions = new CollectionWrapper<ChatSessionDocType>(
  async () => (await initDatabase()).chatSessions,
  'id'
);

class DatabaseService {
  private static instance: DatabaseService;

  private constructor() {
    console.log("Database service initialized.");
  }

  public static getInstance(): DatabaseService {
    if (!DatabaseService.instance) {
      DatabaseService.instance = new DatabaseService();
    }
    return DatabaseService.instance;
  }

  // Expose collections for use in other services
  public agents = Agents;
  public skills = Skills;
  public chatSessions = ChatSessions;

  // Helper methods for common operations
  public async createAgent(agentData: Partial<AgentDocType>) {
    return await this.agents.insertOne(agentData);
  }

  public async getAgent(agentId: string) {
    return await this.agents.findOne({ agentId });
  }

  public async updateAgent(agentId: string, updateData: Partial<AgentDocType>) {
    return await this.agents.updateOne({ agentId }, updateData);
  }

  public async deleteAgent(agentId: string) {
    return await this.agents.deleteOne({ agentId });
  }

  public async listAgents() {
    return await this.agents.find({});
  }

  public async createSkill(skillData: Partial<SkillDocType>) {
    return await this.skills.insertOne(skillData);
  }

  public async getSkill(skillId: string) {
    return await this.skills.findOne({ skillId });
  }

  public async updateSkill(skillId: string, updateData: Partial<SkillDocType>) {
    return await this.skills.updateOne({ skillId }, updateData);
  }

  public async deleteSkill(skillId: string) {
    return await this.skills.deleteOne({ skillId });
  }

  public async listSkills() {
    return await this.skills.find({});
  }

  public async createChatSession(sessionData: Partial<ChatSessionDocType>) {
    return await this.chatSessions.insertOne(sessionData);
  }

  public async getChatSession(sessionId: string) {
    return await this.chatSessions.findOne({ _id: sessionId });
  }

  public async updateChatSession(sessionId: string, updateData: Partial<ChatSessionDocType>) {
    return await this.chatSessions.updateOne({ _id: sessionId }, updateData);
  }

  public async deleteChatSession(sessionId: string) {
    return await this.chatSessions.deleteOne({ _id: sessionId });
  }

  public async listChatSessions() {
    const sessions = await this.chatSessions.find({});
    // Sort by updatedAt descending (most recent first)
    return sessions.sort((a, b) => {
      const aDate = new Date(a.updatedAt || a.createdAt || new Date().toISOString());
      const bDate = new Date(b.updatedAt || b.createdAt || new Date().toISOString());
      return bDate.getTime() - aDate.getTime();
    });
  }

  // Search methods
  public async searchSkills(term: string, limit: number = 10) {
    try {
      const db = await initDatabase();
      const searchTerm = term.toLowerCase();

      // Use RxDB's query capabilities for better performance
      const results = await db.skills
        .find({
          $or: [
            { name: { $regex: new RegExp(searchTerm, 'i') } },
            { description: { $regex: new RegExp(searchTerm, 'i') } }
          ]
        } as Record<string, unknown>)
        .limit(limit)
        .exec();

      return results.map(doc => doc.toJSON());
    } catch (error) {
      console.error('Search error, falling back to simple search:', error);

      // Fallback to simple search if regex fails
      const allSkills = await this.skills.find({});
      const searchTerm = term.toLowerCase();

      const matchingSkills = allSkills.filter(skill =>
        skill.name.toLowerCase().includes(searchTerm) ||
        (skill.description && skill.description.toLowerCase().includes(searchTerm))
      );

      return matchingSkills.slice(0, limit);
    }
  }

  // Initialize database method for external use
  public async initializeDatabase() {
    return await initDatabase();
  }
}

export const databaseService = DatabaseService.getInstance();
