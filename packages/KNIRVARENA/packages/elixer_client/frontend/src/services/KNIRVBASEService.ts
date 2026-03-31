/**
 * KNIRVBASE Service
 * Manages KNIRVBASE database operations for the KNIRV Controller
 */

import { DB, Collection, Options } from '@knirvcorp/knirvbase-ts';
import { BrowserDB, Options as BrowserOptions } from '../core/storage/BrowserDB';

export interface KNIRVBASEConfig {
  dataDir: string;
  distributedEnabled?: boolean;
  distributedNetworkID?: string;
  distributedBootstrapPeers?: string[];
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
  async createApiKey(apiKey: any): Promise<any> {
    const collection = this.getCollection('apikeys');
    return await collection.insert(apiKey);
  }

  async findApiKey(key: string): Promise<any> {
    const collection = this.getCollection('apikeys');
    return await collection.find(key);
  }

  async getAllApiKeys(): Promise<any[]> {
    const collection = this.getCollection('apikeys');
    return await collection.findAll();
  }

  async updateApiKey(key: string, update: any): Promise<number> {
    const collection = this.getCollection('apikeys');
    return await collection.update(key, update);
  }

  async deleteApiKey(key: string): Promise<number> {
    const collection = this.getCollection('apikeys');
    return await collection.delete(key);
  }

  // Knowledge operations
  async createKnowledgeItem(item: any): Promise<any> {
    const collection = this.getCollection('knowledge');
    return await collection.insert(item);
  }

  async findKnowledgeItem(id: string): Promise<any> {
    const collection = this.getCollection('knowledge');
    return await collection.find(id);
  }

  async getAllKnowledgeItems(): Promise<any[]> {
    const collection = this.getCollection('knowledge');
    return await collection.findAll();
  }

  // User Settings operations
  async createUserSettings(settings: any): Promise<any> {
    const collection = this.getCollection('usersettings');
    return await collection.insert(settings);
  }

  async getUserSettings(userId: string): Promise<any> {
    const collection = this.getCollection('usersettings');
    return await collection.find(userId);
  }

  async updateUserSettings(userId: string, update: any): Promise<number> {
    const collection = this.getCollection('usersettings');
    return await collection.update(userId, update);
  }

  // Training Data operations
  async createTrainingData(data: any): Promise<any> {
    const collection = this.getCollection('trainingdata');
    return await collection.insert(data);
  }

  async getAllTrainingData(): Promise<any[]> {
    const collection = this.getCollection('trainingdata');
    return await collection.findAll();
  }

  // Cortex State operations
  async saveCortexState(state: any): Promise<any> {
    const collection = this.getCollection('cortexstate');
    return await collection.insert(state);
  }

  async getCortexState(id: string): Promise<any> {
    const collection = this.getCollection('cortexstate');
    return await collection.find(id);
  }

  async updateCortexState(id: string, update: any): Promise<number> {
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
  async listSkills(): Promise<any[]> {
    const collection = this.getCollection('knowledge');
    return await collection.findAll();
  }

  async getSkill(skillId: string): Promise<any> {
    const collection = this.getCollection('knowledge');
    return await collection.find(skillId);
  }

  async createSkill(skillData: any): Promise<any> {
    const collection = this.getCollection('knowledge');
    return await collection.insert({
      ...skillData,
      createdAt: new Date(),
      updatedAt: new Date()
    });
  }

  async updateSkill(skillId: string, updateData: any): Promise<number> {
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

  async searchSkills(term: string, limit: number): Promise<any[]> {
    const allSkills = await this.listSkills();
    return allSkills
      .filter(skill => 
        skill.name.toLowerCase().includes(term.toLowerCase()) ||
        (skill.description && skill.description.toLowerCase().includes(term.toLowerCase()))
      )
      .slice(0, limit);
  }

  async getChatHistory(userId?: string): Promise<any[]> {
    const collection = this.getCollection('trainingdata');
    const allData = await collection.findAll();
    return userId ? allData.filter(item => item.userId === userId) : allData;
  }

  async saveChatMessage(message: any): Promise<any> {
    const collection = this.getCollection('trainingdata');
    return await collection.insert({
      ...message,
      timestamp: Date.now()
    });
  }

  async getChats(): Promise<any[]> {
    return await this.getChatHistory();
  }

  async saveChat(chat: any): Promise<any> {
    return await this.saveChatMessage(chat);
  }

  // Chat session methods for API compatibility
  async listChatSessions(): Promise<any[]> {
    const collection = this.getCollection('trainingdata');
    const allData = await collection.findAll();
    const sessions = new Map();
    
    allData.forEach(item => {
      if (item.sessionId) {
        if (!sessions.has(item.sessionId)) {
          sessions.set(item.sessionId, {
            _id: item.sessionId,
            title: item.title || 'Untitled Chat',
            createdAt: new Date(item.timestamp),
            updatedAt: new Date(item.timestamp),
            messages: []
          });
        }
        sessions.get(item.sessionId).messages.push({
          id: item.id,
          content: item.content,
          sender: item.sender || 'user',
          timestamp: new Date(item.timestamp)
        });
      }
    });
    
    return Array.from(sessions.values());
  }

  async getChatSession(sessionId: string): Promise<any> {
    const collection = this.getCollection('trainingdata');
    const allData = await collection.findAll();
    const sessionData = allData.filter(item => item.sessionId === sessionId);
    
    if (sessionData.length === 0) return null;
    
    const messages = sessionData.map(item => ({
      id: item.id,
      content: item.content,
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

  async createChatSession(session: any): Promise<any> {
    const collection = this.getCollection('trainingdata');
    const newSession = {
      sessionId: session._id || `session_${Date.now()}`,
      title: session.title,
      content: '',
      sender: 'system',
      timestamp: Date.now(),
      ...session
    };
    return await collection.insert(newSession);
  }

  async updateChatSession(sessionId: string, updates: any): Promise<any> {
    const collection = this.getCollection('trainingdata');
    const allData = await collection.findAll();
    const sessionData = allData.filter(item => item.sessionId === sessionId);
    
    if (sessionData.length === 0) {
      throw new Error('Chat session not found');
    }
    
    const sessionToUpdate = sessionData[0];
    const updatedSession = { ...sessionToUpdate, ...updates, timestamp: Date.now() };
    
    // Always use ID-based update for consistency
    await collection.update(sessionToUpdate.id, updatedSession);
    
    return updatedSession;
  }

  async deleteChatSession(sessionId: string): Promise<any> {
    const collection = this.getCollection('trainingdata');
    const allData = await collection.findAll();
    const sessionData = allData.filter(item => item.sessionId === sessionId);
    
    let deletedCount = 0;
    for (const session of sessionData) {
      // Always use ID-based delete for consistency
      await collection.delete(session.id);
      deletedCount++;
    }
    
    return { deletedCount };
  }

  // Agent management methods
  async createAgent(agent: any): Promise<any> {
    const collection = this.getCollection('knowledge');
    return await collection.insert({
      ...agent,
      type: 'agent',
      timestamp: Date.now()
    });
  }

  async getAgent(agentId: string): Promise<any> {
    const collection = this.getCollection('knowledge');
    const allData = await collection.findAll();
    const agents = allData.filter(item => item.id === agentId && item.type === 'agent');
    return agents.length > 0 ? agents[0] : null;
  }

  async updateAgent(agentId: string, updates: any): Promise<any> {
    const collection = this.getCollection('knowledge');
    const allData = await collection.findAll();
    const agents = allData.filter(item => item.id === agentId && item.type === 'agent');
    
    if (agents.length === 0) {
      throw new Error('Agent not found');
    }
    
    const agentToUpdate = agents[0];
    const updatedAgent = { ...agentToUpdate, ...updates };
    
    // Always use ID-based update for consistency
    await collection.update(agentToUpdate.id, updatedAgent);
    
    return updatedAgent;
  }

  async deleteAgent(agentId: string): Promise<any> {
    const collection = this.getCollection('knowledge');
    const allData = await collection.findAll();
    const agents = allData.filter(item => item.id === agentId && item.type === 'agent');
    
    let deletedCount = 0;
    for (const agent of agents) {
      // Always use ID-based delete for consistency
      await collection.delete(agent.id);
      deletedCount++;
    }
    
    return { deletedCount };
  }

  async listAgents(): Promise<any[]> {
    const collection = this.getCollection('knowledge');
    const allData = await collection.findAll();
    return allData.filter(item => item.type === 'agent');
  }

  // User settings method
  async getAllUserSettings(): Promise<any[]> {
    const collection = this.getCollection('usersettings');
    return await collection.findAll();
  }
}

// Singleton instance
export const knirvbaseService = new KNIRVBASEService();