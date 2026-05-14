/**
 * KNIRVBASE Service
 * Manages KNIRVBASE database operations for the KNIRV Controller
 */

import { DB, Collection, Options } from '@knirvcorp/knirvbase-ts';
import { BrowserDB, Options as BrowserOptions } from '../core/storage/BrowserDB';
import { Agent } from '../types/common';

export interface KNIRVBASEConfig {
  dataDir: string;
  distributedEnabled?: boolean;
  distributedNetworkID?: string;
  distributedBootstrapPeers?: string[];
}

// Generic document type for database operations
export interface Document {
  id?: string;
  [key: string]: unknown;
}

// Skill-specific interface
export interface Skill extends Document {
  name: string;
  description?: string;
  createdAt?: Date;
  updatedAt?: Date;
}

// Chat message interface
export interface ChatMessage extends Document {
  content: string;
  sender: string;
  timestamp: number;
  userId?: string;
  sessionId?: string;
  title?: string;
}

// Chat session interface
export interface ChatSession {
  _id: string;
  title: string;
  createdAt: Date | string;
  updatedAt: Date | string;
  messages: Array<{
    id: string;
    content: string;
    sender: string;
    timestamp: Date | string;
  }>;
}

export class KNIRVBASEService {
  private db: DB | BrowserDB | null = null;
  private collections: Map<string, Collection> = new Map();
  private isBrowser: boolean = false;

  constructor() {
    // Detect if we're running in browser environment
    this.isBrowser = typeof window !== 'undefined' && typeof fetch !== 'undefined';
  }

  async initialize(config: KNIRVBASEConfig = { dataDir: './data/knirvcontroller' }): Promise<void> {
    try {
      if (this.isBrowser) {
        // Use browser-compatible database
        const browserOptions: BrowserOptions = {
          sessionId: `knirvcontroller_${Date.now()}`,
          baseUrl: window.location.origin,
          dataDir: config.dataDir,
          distributedEnabled: config.distributedEnabled || false,
          distributedNetworkID: config.distributedNetworkID,
          distributedBootstrapPeers: config.distributedBootstrapPeers
        };

        this.db = new BrowserDB(browserOptions);
        await this.db.initialize();
        console.log('Browser KNIRVBASE initialized successfully');
      } else {
        // Use Node.js filesystem database
        const options: Options = {
          dataDir: config.dataDir,
          distributedEnabled: config.distributedEnabled || false,
          distributedNetworkID: config.distributedNetworkID,
          distributedBootstrapPeers: config.distributedBootstrapPeers
        };

        this.db = new DB(options);
        await this.db.initialize();
        console.log('Node.js KNIRVBASE initialized successfully');
      }

      // Initialize collections that were previously in RxDB
      await this.initializeCollections();
      
       console.log('KNIRVBASE initialized successfully');
    } catch (error) {
      console.error('Failed to initialize KNIRVBASE:', error);
      throw error;
    }
  }

  private async initializeCollections(): Promise<void> {
    if (!this.db) throw new Error('Database not initialized');

    // API Keys collection
    const apiKeys = this.db.collection('apikeys');
    this.collections.set('apikeys', apiKeys);

    // Knowledge collection
    const knowledge = this.db.collection('knowledge');
    this.collections.set('knowledge', knowledge);

    // User Settings collection
    const userSettings = this.db.collection('usersettings');
    this.collections.set('usersettings', userSettings);

    // Training Data collection
    const trainingData = this.db.collection('trainingdata');
    this.collections.set('trainingdata', trainingData);

    // Cortex State collection
    const cortexState = this.db.collection('cortexstate');
    this.collections.set('cortexstate', cortexState);
  }

  getCollection(name: string): Collection {
    if (!this.collections.has(name)) {
      throw new Error(`Collection '${name}' not found`);
    }
    return this.collections.get(name)!;
  }

  // API Key operations
  async createApiKey(apiKey: Document): Promise<Document> {
    const collection = this.getCollection('apikeys');
    return await collection.insert(apiKey);
  }

  async findApiKey(key: string): Promise<Document | null> {
    const collection = this.getCollection('apikeys');
    return await collection.find(key);
  }

  async getAllApiKeys(): Promise<Document[]> {
    const collection = this.getCollection('apikeys');
    return await collection.findAll();
  }

  async updateApiKey(key: string, update: Document): Promise<number> {
    const collection = this.getCollection('apikeys');
    return await collection.update(key, update);
  }

  async deleteApiKey(key: string): Promise<number> {
    const collection = this.getCollection('apikeys');
    return await collection.delete(key);
  }

  // Knowledge operations
  async createKnowledgeItem(item: Document): Promise<Document> {
    const collection = this.getCollection('knowledge');
    return await collection.insert(item);
  }

  async findKnowledgeItem(id: string): Promise<Document | null> {
    const collection = this.getCollection('knowledge');
    return await collection.find(id);
  }

  async getAllKnowledgeItems(): Promise<Document[]> {
    const collection = this.getCollection('knowledge');
    return await collection.findAll();
  }

  // User Settings operations
  async createUserSettings(settings: Document): Promise<Document> {
    const collection = this.getCollection('usersettings');
    return await collection.insert(settings);
  }

  async getUserSettings(userId: string): Promise<Document | null> {
    const collection = this.getCollection('usersettings');
    return await collection.find(userId);
  }

  async updateUserSettings(userId: string, update: Document): Promise<number> {
    const collection = this.getCollection('usersettings');
    return await collection.update(userId, update);
  }

  // Training Data operations
  async createTrainingData(data: Document): Promise<Document> {
    const collection = this.getCollection('trainingdata');
    return await collection.insert(data);
  }

  async getAllTrainingData(): Promise<Document[]> {
    const collection = this.getCollection('trainingdata');
    return await collection.findAll();
  }

  // Cortex State operations
  async saveCortexState(state: Document): Promise<Document> {
    const collection = this.getCollection('cortexstate');
    return await collection.insert(state);
  }

  async getCortexState(id: string): Promise<Document | null> {
    const collection = this.getCollection('cortexstate');
    return await collection.find(id);
  }

  async updateCortexState(id: string, update: Document): Promise<number> {
    const collection = this.getCollection('cortexstate');
    return await collection.update(id, update);
  }

  async shutdown(): Promise<void> {
    if (this.db) {
      await this.db.shutdown();
      this.db = null;
      this.collections.clear();
    }
  }

  isInitialized(): boolean {
    return this.db !== null;
  }

  getDatabase(): DB | BrowserDB {
    if (!this.db) {
      throw new Error('Database not initialized');
    }
    return this.db;
  }

  // Additional methods for compatibility with existing services
  async listSkills(): Promise<Document[]> {
    const collection = this.getCollection('knowledge');
    return await collection.findAll();
  }

  async getSkill(skillId: string): Promise<Document | null> {
    const collection = this.getCollection('knowledge');
    return await collection.find(skillId);
  }

  async createSkill(skillData: Document): Promise<Document> {
    const collection = this.getCollection('knowledge');
    return await collection.insert({
      ...skillData,
      createdAt: new Date(),
      updatedAt: new Date()
    });
  }

  async updateSkill(skillId: string, updateData: Document): Promise<number> {
    const collection = this.getCollection('knowledge');
    return await collection.update(skillId, {
      ...updateData,
      updatedAt: new Date()
    });
  }

  async deleteSkill(skillId: string): Promise<number> {
    const collection = this.getCollection('knowledge');
    return await collection.delete(skillId);
  }

  async searchSkills(term: string, limit: number): Promise<Skill[]> {
    const allSkills = await this.listSkills();
    return allSkills
      .filter(skill => {
        const s = skill as Skill;
        return (
          s.name?.toLowerCase().includes(term.toLowerCase()) ||
          (s.description && s.description.toLowerCase().includes(term.toLowerCase()))
        );
      })
      .slice(0, limit) as Skill[];
  }

  async getChatHistory(userId?: string): Promise<ChatMessage[]> {
    const collection = this.getCollection('trainingdata');
    const allData = await collection.findAll();
    const messages = allData.map(item => item as ChatMessage);
    return userId ? messages.filter((item: ChatMessage) => item.userId === userId) : messages;
  }

  async saveChatMessage(message: ChatMessage): Promise<ChatMessage> {
    const collection = this.getCollection('trainingdata');
    return await collection.insert({
      ...message,
      timestamp: Date.now()
    }) as ChatMessage;
  }

  async getChats(): Promise<ChatMessage[]> {
    return await this.getChatHistory();
  }

  async saveChat(chat: ChatMessage): Promise<ChatMessage> {
    return await this.saveChatMessage(chat);
  }

  // Chat session methods for API compatibility
  async listChatSessions(): Promise<ChatSession[]> {
    const collection = this.getCollection('trainingdata');
    const allData = await collection.findAll() as ChatMessage[];
    const sessions = new Map<string, ChatSession>();
    
    allData.forEach((item: ChatMessage) => {
      if (item.sessionId) {
        if (!sessions.has(item.sessionId)) {
          sessions.set(item.sessionId, {
            _id: item.sessionId,
            title: item.title || 'Untitled Chat',
            createdAt: new Date(item.timestamp || Date.now()),
            updatedAt: new Date(item.timestamp || Date.now()),
            messages: []
          });
        }
        const session = sessions.get(item.sessionId)!;
        session.messages.push({
          id: item.id || '',
          content: item.content || '',
          sender: item.sender || 'user',
          timestamp: new Date(item.timestamp || Date.now())
        });
      }
    });
    
    return Array.from(sessions.values());
  }

  async getChatSession(sessionId: string): Promise<ChatSession | null> {
    const collection = this.getCollection('trainingdata');
    const allData = await collection.findAll() as ChatMessage[];
    const sessionData = allData.filter((item: ChatMessage) => item.sessionId === sessionId);
    
    if (sessionData.length === 0) return null;
    
    const messages = sessionData.map((item: ChatMessage) => ({
      id: item.id || '',
      content: item.content || '',
      sender: item.sender || 'user',
      timestamp: new Date(item.timestamp)
    }));
    
    const firstMessage = sessionData[0];
    return {
      _id: sessionId,
      title: firstMessage.title || 'Untitled Chat',
      createdAt: new Date(firstMessage.timestamp),
      updatedAt: new Date(firstMessage.timestamp),
      messages
    };
  }

  async createChatSession(session: Partial<ChatSession>): Promise<Document> {
    const collection = this.getCollection('trainingdata');
    const newSession = {
      sessionId: session._id || `session_${Date.now()}`,
      title: session.title || 'Untitled Chat',
      content: '',
      sender: 'system',
      timestamp: Date.now(),
      ...session
    };
    return await collection.insert(newSession);
  }

  async updateChatSession(sessionId: string, updates: Partial<ChatSession>): Promise<Document> {
    const collection = this.getCollection('trainingdata');
    const allData = await collection.findAll() as ChatMessage[];
    const sessionData = allData.filter((item: ChatMessage) => item.sessionId === sessionId);
    
    if (sessionData.length === 0) {
      throw new Error('Chat session not found');
    }
    
    const sessionToUpdate = sessionData[0];
    const updatedSession = { ...sessionToUpdate, ...updates, timestamp: Date.now() };
    
    // Always use ID-based update for consistency
    await collection.update(sessionToUpdate.id || '', updatedSession);
    
    return updatedSession;
  }

  async deleteChatSession(sessionId: string): Promise<{ deletedCount: number }> {
    const collection = this.getCollection('trainingdata');
    const allData = await collection.findAll() as ChatMessage[];
    const sessionData = allData.filter((item: ChatMessage) => item.sessionId === sessionId);
    
    let deletedCount = 0;
    for (const session of sessionData) {
      // Always use ID-based delete for consistency
      await collection.delete(session.id || '');
      deletedCount++;
    }
    
    return { deletedCount };
  }

  // Agent management methods
  async createAgent(agent: Partial<Agent>): Promise<Document> {
    const collection = this.getCollection('knowledge');
    return await collection.insert({
      ...agent,
      type: 'agent',
      timestamp: Date.now()
    });
  }

  async getAgent(agentId: string): Promise<Agent | null> {
    const collection = this.getCollection('knowledge');
    const allData = await collection.findAll() as Document[];
    const agents = allData.filter((item: Document) => item.id === agentId && item.type === 'agent');
    return agents.length > 0 ? agents[0] as unknown as Agent : null;
  }

  async updateAgent(agentId: string, updates: Partial<Agent>): Promise<Agent> {
    const collection = this.getCollection('knowledge');
    const allData = await collection.findAll() as Document[];
    const agents = allData.filter((item: Document) => item.id === agentId && item.type === 'agent');
    
    if (agents.length === 0) {
      throw new Error('Agent not found');
    }
    
    const agentToUpdate = agents[0];
    const updatedAgent = { ...agentToUpdate, ...updates };
    
    // Always use ID-based update for consistency
    await collection.update(agentToUpdate.id || '', updatedAgent);
    
    return updatedAgent as unknown as Agent;
  }

  async deleteAgent(agentId: string): Promise<{ deletedCount: number }> {
    const collection = this.getCollection('knowledge');
    const allData = await collection.findAll() as Document[];
    const agents = allData.filter((item: Document) => item.id === agentId && item.type === 'agent');
    
    let deletedCount = 0;
    for (const agent of agents) {
      // Always use ID-based delete for consistency
      await collection.delete(agent.id || '');
      deletedCount++;
    }
    
    return { deletedCount };
  }

  async listAgents(): Promise<Agent[]> {
    const collection = this.getCollection('knowledge');
    const allData = await collection.findAll() as Document[];
    return allData.filter((item: Document) => item.type === 'agent') as unknown as Agent[];
  }

  // User settings method
  async getAllUserSettings(): Promise<Document[]> {
    const collection = this.getCollection('usersettings');
    return await collection.findAll();
  }
}

// Singleton instance
export const knirvbaseService = new KNIRVBASEService();