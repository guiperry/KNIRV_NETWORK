// KNIRVGRAPH Service - Integration with KNIRV Network Knowledge Graph
import axios, { AxiosInstance } from 'axios';
import type {
  KNIRVGraphConfig,
  KNIRVGraphResponse,
  MemoryNode,
  MemoryEdge,
  MemoryGraph,
  Note,
  Conversation,
  ChatMessage,
} from '../types/chatBrain';

export class KNIRVGraphService {
  private client: AxiosInstance;
  private config: KNIRVGraphConfig;
  private useMockData: boolean = true; // Set to false when KNIRVGRAPH backend is ready

  constructor(config: KNIRVGraphConfig) {
    this.config = config;
    this.client = axios.create({
      baseURL: config.nodeEndpoint,
      timeout: 10000,
      headers: {
        'Content-Type': 'application/json',
        ...(config.apiKey && { Authorization: `Bearer ${config.apiKey}` }),
      },
    });
  }

  private getUserId(): string {
    return localStorage.getItem('userId') || 'anonymous';
  }

  // ==================== Memory Graph Operations ====================

  async storeMemoryNode(node: Omit<MemoryNode, 'id'>): Promise<string> {
    if (this.useMockData) {
      const id = `node_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
      const mockNode = { ...node, id };

      // Store in localStorage for now
      const existingNodes = this.getMockMemoryNodes();
      existingNodes.push(mockNode);
      localStorage.setItem('chat_brain_memory_nodes', JSON.stringify(existingNodes));

      return id;
    }

    try {
      const response = await this.client.post<KNIRVGraphResponse<{ nodeId: string }>>(
        '/api/graph/memory/node',
        {
          type: node.type,
          label: node.label,
          properties: node.properties,
          vector: node.embedding,
          userId: this.getUserId(),
          timestamp: node.timestamp,
        }
      );

      if (response.data.success && response.data.data) {
        return response.data.data.nodeId;
      }
      throw new Error(response.data.error || 'Failed to store memory node');
    } catch (error) {
      console.error('Error storing memory node:', error);
      throw error;
    }
  }

  async storeMemoryEdge(edge: Omit<MemoryEdge, 'id'>): Promise<void> {
    if (this.useMockData) {
      const id = `edge_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
      const mockEdge = { ...edge, id };

      const existingEdges = this.getMockMemoryEdges();
      existingEdges.push(mockEdge);
      localStorage.setItem('chat_brain_memory_edges', JSON.stringify(existingEdges));

      return;
    }

    try {
      const response = await this.client.post<KNIRVGraphResponse<void>>(
        '/api/graph/memory/edge',
        {
          sourceId: edge.sourceId,
          targetId: edge.targetId,
          relation: edge.relation,
          weight: edge.weight || 1.0,
          properties: edge.properties,
          timestamp: edge.timestamp || Date.now(),
        }
      );

      if (!response.data.success) {
        throw new Error(response.data.error || 'Failed to store memory edge');
      }
    } catch (error) {
      console.error('Error storing memory edge:', error);
      throw error;
    }
  }

  async getMemoryGraph(userId?: string): Promise<MemoryGraph> {
    if (this.useMockData) {
      return {
        nodes: this.getMockMemoryNodes(),
        edges: this.getMockMemoryEdges(),
      };
    }

    try {
      const response = await this.client.get<KNIRVGraphResponse<MemoryGraph>>(
        '/api/graph/memory/graph',
        { params: { userId: userId || this.getUserId() } }
      );

      if (response.data.success && response.data.data) {
        return response.data.data;
      }
      throw new Error(response.data.error || 'Failed to get memory graph');
    } catch (error) {
      console.error('Error getting memory graph:', error);
      throw error;
    }
  }

  async searchMemory(query: string, limit: number = 10): Promise<MemoryNode[]> {
    if (this.useMockData) {
      const nodes = this.getMockMemoryNodes();
      // Simple text search for mock data
      return nodes
        .filter((node) =>
          node.label.toLowerCase().includes(query.toLowerCase())
        )
        .slice(0, limit);
    }

    try {
      const response = await this.client.post<KNIRVGraphResponse<{ nodes: MemoryNode[] }>>(
        '/api/graph/memory/search',
        { query, limit, userId: this.getUserId() }
      );

      if (response.data.success && response.data.data) {
        return response.data.data.nodes;
      }
      throw new Error(response.data.error || 'Failed to search memory');
    } catch (error) {
      console.error('Error searching memory:', error);
      throw error;
    }
  }

  // ==================== Notes Operations ====================

  async storeNote(note: Omit<Note, 'id'>): Promise<string> {
    if (this.useMockData) {
      const id = `note_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
      const mockNote = { ...note, id };

      const existingNotes = this.getMockNotes();
      existingNotes.push(mockNote);
      localStorage.setItem('chat_brain_notes', JSON.stringify(existingNotes));

      return id;
    }

    try {
      const response = await this.client.post<KNIRVGraphResponse<{ noteId: string }>>(
        '/api/graph/notes',
        {
          ...note,
          userId: this.getUserId(),
        }
      );

      if (response.data.success && response.data.data) {
        return response.data.data.noteId;
      }
      throw new Error(response.data.error || 'Failed to store note');
    } catch (error) {
      console.error('Error storing note:', error);
      throw error;
    }
  }

  async getNotes(): Promise<Note[]> {
    if (this.useMockData) {
      return this.getMockNotes();
    }

    try {
      const response = await this.client.get<KNIRVGraphResponse<{ notes: Note[] }>>(
        '/api/graph/notes',
        { params: { userId: this.getUserId() } }
      );

      if (response.data.success && response.data.data) {
        return response.data.data.notes;
      }
      throw new Error(response.data.error || 'Failed to get notes');
    } catch (error) {
      console.error('Error getting notes:', error);
      throw error;
    }
  }

  async updateNote(noteId: string, updates: Partial<Note>): Promise<void> {
    if (this.useMockData) {
      const notes = this.getMockNotes();
      const index = notes.findIndex((n) => n.id === noteId);
      if (index >= 0) {
        notes[index] = { ...notes[index], ...updates };
        localStorage.setItem('chat_brain_notes', JSON.stringify(notes));
      }
      return;
    }

    try {
      const response = await this.client.put<KNIRVGraphResponse<void>>(
        `/api/graph/notes/${noteId}`,
        updates
      );

      if (!response.data.success) {
        throw new Error(response.data.error || 'Failed to update note');
      }
    } catch (error) {
      console.error('Error updating note:', error);
      throw error;
    }
  }

  async deleteNote(noteId: string): Promise<void> {
    if (this.useMockData) {
      const notes = this.getMockNotes();
      const filtered = notes.filter((n) => n.id !== noteId);
      localStorage.setItem('chat_brain_notes', JSON.stringify(filtered));
      return;
    }

    try {
      const response = await this.client.delete<KNIRVGraphResponse<void>>(
        `/api/graph/notes/${noteId}`
      );

      if (!response.data.success) {
        throw new Error(response.data.error || 'Failed to delete note');
      }
    } catch (error) {
      console.error('Error deleting note:', error);
      throw error;
    }
  }

  // ==================== Conversation Operations ====================

  async storeConversation(conversation: Omit<Conversation, 'id'>): Promise<string> {
    if (this.useMockData) {
      const id = `conv_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
      const mockConv = { ...conversation, id };

      const existingConvs = this.getMockConversations();
      existingConvs.push(mockConv);
      localStorage.setItem('chat_brain_conversations', JSON.stringify(existingConvs));

      return id;
    }

    try {
      const response = await this.client.post<KNIRVGraphResponse<{ conversationId: string }>>(
        '/api/graph/conversations',
        {
          ...conversation,
          userId: this.getUserId(),
        }
      );

      if (response.data.success && response.data.data) {
        return response.data.data.conversationId;
      }
      throw new Error(response.data.error || 'Failed to store conversation');
    } catch (error) {
      console.error('Error storing conversation:', error);
      throw error;
    }
  }

  async getConversations(sessionId?: string): Promise<Conversation[]> {
    if (this.useMockData) {
      const convs = this.getMockConversations();
      if (sessionId) {
        return convs.filter((c) => c.sessionId === sessionId);
      }
      return convs;
    }

    try {
      const response = await this.client.get<KNIRVGraphResponse<{ conversations: Conversation[] }>>(
        '/api/graph/conversations',
        { params: { userId: this.getUserId(), sessionId } }
      );

      if (response.data.success && response.data.data) {
        return response.data.data.conversations;
      }
      throw new Error(response.data.error || 'Failed to get conversations');
    } catch (error) {
      console.error('Error getting conversations:', error);
      throw error;
    }
  }

  // ==================== Mock Data Helpers ====================

  private getMockMemoryNodes(): MemoryNode[] {
    const stored = localStorage.getItem('chat_brain_memory_nodes');
    if (stored) {
      return JSON.parse(stored);
    }

    // Initialize with sample data
    const sampleNodes: MemoryNode[] = [
      {
        id: 'node_1',
        label: 'AI',
        type: 'concept',
        timestamp: Date.now(),
        userId: this.getUserId(),
      },
      {
        id: 'node_2',
        label: 'Machine Learning',
        type: 'concept',
        timestamp: Date.now(),
        userId: this.getUserId(),
      },
      {
        id: 'node_3',
        label: 'KNIRV Network',
        type: 'entity',
        timestamp: Date.now(),
        userId: this.getUserId(),
      },
    ];

    localStorage.setItem('chat_brain_memory_nodes', JSON.stringify(sampleNodes));
    return sampleNodes;
  }

  private getMockMemoryEdges(): MemoryEdge[] {
    const stored = localStorage.getItem('chat_brain_memory_edges');
    if (stored) {
      return JSON.parse(stored);
    }

    // Initialize with sample data
    const sampleEdges: MemoryEdge[] = [
      {
        id: 'edge_1',
        sourceId: 'node_2',
        targetId: 'node_1',
        relation: 'part_of',
        weight: 0.9,
        timestamp: Date.now(),
      },
      {
        id: 'edge_2',
        sourceId: 'node_3',
        targetId: 'node_1',
        relation: 'related_to',
        weight: 0.8,
        timestamp: Date.now(),
      },
    ];

    localStorage.setItem('chat_brain_memory_edges', JSON.stringify(sampleEdges));
    return sampleEdges;
  }

  private getMockNotes(): Note[] {
    const stored = localStorage.getItem('chat_brain_notes');
    return stored ? JSON.parse(stored) : [];
  }

  private getMockConversations(): Conversation[] {
    const stored = localStorage.getItem('chat_brain_conversations');
    return stored ? JSON.parse(stored) : [];
  }

  // ==================== Utility Methods ====================

  setMockDataMode(enabled: boolean): void {
    this.useMockData = enabled;
  }

  isMockDataMode(): boolean {
    return this.useMockData;
  }

  clearMockData(): void {
    localStorage.removeItem('chat_brain_memory_nodes');
    localStorage.removeItem('chat_brain_memory_edges');
    localStorage.removeItem('chat_brain_notes');
    localStorage.removeItem('chat_brain_conversations');
  }
}

// Singleton instance
let knirvGraphServiceInstance: KNIRVGraphService | null = null;

export const getKNIRVGraphService = (): KNIRVGraphService => {
  if (!knirvGraphServiceInstance) {
    const config: KNIRVGraphConfig = {
      nodeEndpoint: import.meta.env.VITE_KNIRVGRAPH_ENDPOINT || 'http://localhost:26657',
      chainId: import.meta.env.VITE_KNIRVGRAPH_CHAIN_ID || 'knirvgraph-1',
      apiKey: import.meta.env.VITE_KNIRVGRAPH_API_KEY,
    };
    knirvGraphServiceInstance = new KNIRVGraphService(config);
  }
  return knirvGraphServiceInstance;
};

export default getKNIRVGraphService;
