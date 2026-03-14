// Chat-Brain Service - Business Logic Layer
import type {
  LLMProvider,
  ChatMessage,
  ChatResponse,
  Note,
  ChatSession,
  MemoryNode,
  MemoryEdge,
} from '../types/chatBrain';
import { getLLMProviderService } from './llmProviderService';
import { getKNIRVGraphService } from './knirvGraphService';
import { cognitiveEngineService } from './CognitiveEngineService';

export class ChatBrainService {
  private llmService = getLLMProviderService();
  private graphService = getKNIRVGraphService();

  // ==================== Chat Operations ====================

  async sendMessage(
    message: string,
    provider: LLMProvider,
    history?: ChatMessage[]
  ): Promise<ChatResponse> {
    try {
      // First, process the input using cortex.wasm cognitive engine
      const cognitiveResult = await cognitiveEngineService.processInput({
        input: message,
        context: history ? { history: JSON.stringify(history) } : undefined,
        taskType: 'conversation',
        requiresSkillInvocation: true,
        metadata: { provider }
      });

      // If cortex.wasm didn't produce a response, fall back to LLM provider
      let response: ChatResponse;
      if (cognitiveResult.output && cognitiveResult.output !== 'Processing completed') {
        response = {
          text: cognitiveResult.output,
          provider: 'cortex',
          metadata: {
            confidence: cognitiveResult.confidence,
            processingTime: cognitiveResult.processingTime,
            skillsInvoked: cognitiveResult.skillsInvoked,
            adaptationTriggered: cognitiveResult.adaptationTriggered
          }
        };
      } else {
        response = await this.llmService.chat(message, provider, history);
      }

      // Store conversation in KNIRVGRAPH
      await this.storeConversationMessage(message, response.text, provider);

      // Extract and store memory nodes from the conversation
      await this.extractAndStoreMemory(message, response.text);

      // Check if we need to submit to KNIRVGRAPH based on cortex.wasm analysis
      if (cognitiveResult.skillsInvoked && cognitiveResult.skillsInvoked.length > 0) {
        await this.handleSkillInvocations(cognitiveResult.skillsInvoked, message, response.text);
      }

      return response;
    } catch (error) {
      console.error('Error in sendMessage:', error);
      throw error;
    }
  }

  private async handleSkillInvocations(skillsInvoked: string[], userMessage: string, aiResponse: string): Promise<void> {
    try {
      console.log('Handling skill invocations:', skillsInvoked);
      
      // For each skill invoked, we might want to:
      // 1. Create a skill node in KNIRVGRAPH
      // 2. Create a relation between the conversation and the skill
      // 3. Trigger further processing or validation
      
      for (const skillId of skillsInvoked) {
        await this.graphService.storeMemoryNode({
          label: skillId,
          type: 'skill',
          timestamp: Date.now(),
          properties: {
            userMessage: userMessage.substring(0, 100),
            aiResponse: aiResponse.substring(0, 100)
          }
        });
      }
    } catch (error) {
      console.error('Error handling skill invocations:', error);
    }
  }

  private async storeConversationMessage(
    userMessage: string,
    aiResponse: string,
    provider: LLMProvider
  ): Promise<void> {
    try {
      const sessionId = this.getCurrentSessionId();

      await this.graphService.storeConversation({
        sessionId,
        messages: [
          {
            id: `msg_${Date.now()}_user`,
            text: userMessage,
            type: 'user',
            timestamp: Date.now(),
          },
          {
            id: `msg_${Date.now()}_bot`,
            text: aiResponse,
            type: 'bot',
            timestamp: Date.now(),
            provider,
          },
        ],
        timestamp: Date.now(),
      });
    } catch (error) {
      console.error('Error storing conversation:', error);
      // Don't throw - conversation storage failure shouldn't break chat
    }
  }

  private async extractAndStoreMemory(userMessage: string, aiResponse: string): Promise<void> {
    try {
      // Simple keyword extraction (can be enhanced with NLP later)
      const keywords = this.extractKeywords(userMessage + ' ' + aiResponse);

      // Store keywords as memory nodes
      for (const keyword of keywords) {
        await this.graphService.storeMemoryNode({
          label: keyword,
          type: 'keyword',
          timestamp: Date.now(),
          properties: {
            source: 'conversation',
            context: userMessage.substring(0, 100),
          },
        });
      }
    } catch (error) {
      console.error('Error extracting memory:', error);
      // Don't throw - memory extraction failure shouldn't break chat
    }
  }

  private extractKeywords(text: string): string[] {
    // Simple keyword extraction - can be enhanced with NLP
    const words = text
      .toLowerCase()
      .replace(/[^\w\s]/g, '')
      .split(/\s+/)
      .filter((word) => word.length > 4); // Filter short words

    // Remove duplicates and common words
    const commonWords = new Set([
      'about',
      'would',
      'there',
      'their',
      'which',
      'could',
      'should',
      'these',
      'those',
      'where',
      'while',
    ]);

    const uniqueKeywords = Array.from(new Set(words)).filter((word) => !commonWords.has(word));

    return uniqueKeywords.slice(0, 10); // Limit to 10 keywords
  }

  // ==================== Session Management ====================

  private getCurrentSessionId(): string {
    let sessionId = sessionStorage.getItem('chat_brain_session_id');

    if (!sessionId) {
      sessionId = `session_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
      sessionStorage.setItem('chat_brain_session_id', sessionId);
    }

    return sessionId;
  }

  async createNewSession(name: string = 'New Chat', provider: LLMProvider = 'gemini'): Promise<ChatSession> {
    const now = Date.now();
    
    const session: Omit<ChatSession, 'id'> = {
      name,
      messages: [],
      createdAt: now,
      updatedAt: now,
      provider,
    };

    const sessionId = await this.graphService.storeChatSession(session);
    sessionStorage.setItem('chat_brain_session_id', sessionId);
    
    return {
      id: sessionId,
      ...session,
    };
  }

  async getCurrentSession(): Promise<ChatSession | null> {
    const sessionId = this.getCurrentSessionId();
    return await this.getChatSession(sessionId);
  }

  async getChatSession(sessionId: string): Promise<ChatSession | null> {
    try {
      return await this.graphService.getChatSession(sessionId);
    } catch (error) {
      console.error('Error getting chat session:', error);
      return null;
    }
  }

  async getAllChatSessions(): Promise<ChatSession[]> {
    try {
      return await this.graphService.getChatSessions();
    } catch (error) {
      console.error('Error getting all chat sessions:', error);
      return [];
    }
  }

  async updateChatSession(sessionId: string, updates: Partial<ChatSession>): Promise<void> {
    try {
      await this.graphService.updateChatSession(sessionId, updates);
    } catch (error) {
      console.error('Error updating chat session:', error);
      throw error;
    }
  }

  async deleteChatSession(sessionId: string): Promise<void> {
    try {
      await this.graphService.deleteChatSession(sessionId);
      
      // If we're deleting the current session, clear it from sessionStorage
      const currentSessionId = this.getCurrentSessionId();
      if (currentSessionId === sessionId) {
        sessionStorage.removeItem('chat_brain_session_id');
      }
    } catch (error) {
      console.error('Error deleting chat session:', error);
      throw error;
    }
  }

  async loadSession(sessionId: string): Promise<ChatMessage[]> {
    try {
      const conversations = await this.graphService.getConversations(sessionId);

      if (conversations.length > 0) {
        return conversations[0].messages;
      }

      return [];
    } catch (error) {
      console.error('Error loading session:', error);
      return [];
    }
  }

  async switchToSession(sessionId: string): Promise<void> {
    sessionStorage.setItem('chat_brain_session_id', sessionId);
    
    // Update the session's updatedAt timestamp
    try {
      await this.graphService.updateChatSession(sessionId, { updatedAt: Date.now() });
    } catch (error) {
      console.error('Error updating session timestamp:', error);
      // Don't throw - this is a non-critical operation
    }
  }

  // ==================== Notes Operations ====================

  async saveNote(title: string, content: string): Promise<string> {
    try {
      const noteId = await this.graphService.storeNote({
        title,
        content,
        timestamp: Date.now(),
        tags: this.extractTags(content),
      });

      return noteId;
    } catch (error) {
      console.error('Error saving note:', error);
      throw error;
    }
  }

  async loadNotes(): Promise<Note[]> {
    try {
      return await this.graphService.getNotes();
    } catch (error) {
      console.error('Error loading notes:', error);
      return [];
    }
  }

  async updateNote(noteId: string, title: string, content: string): Promise<void> {
    try {
      await this.graphService.updateNote(noteId, {
        title,
        content,
        timestamp: Date.now(),
        tags: this.extractTags(content),
      });
    } catch (error) {
      console.error('Error updating note:', error);
      throw error;
    }
  }

  async deleteNote(noteId: string): Promise<void> {
    try {
      await this.graphService.deleteNote(noteId);
    } catch (error) {
      console.error('Error deleting note:', error);
      throw error;
    }
  }

  private extractTags(content: string): string[] {
    // Extract hashtags from markdown content
    const hashtagRegex = /#(\w+)/g;
    const matches = content.match(hashtagRegex);

    if (!matches) return [];

    return matches.map((tag) => tag.substring(1)); // Remove # symbol
  }

  // ==================== Memory Graph Operations ====================

  async getMemoryGraph() {
    try {
      return await this.graphService.getMemoryGraph();
    } catch (error) {
      console.error('Error getting memory graph:', error);
      return { nodes: [], edges: [] };
    }
  }

  async searchMemory(query: string): Promise<MemoryNode[]> {
    try {
      return await this.graphService.searchMemory(query);
    } catch (error) {
      console.error('Error searching memory:', error);
      return [];
    }
  }

  async addMemoryNode(label: string, type: MemoryNode['type']): Promise<string> {
    try {
      return await this.graphService.storeMemoryNode({
        label,
        type,
        timestamp: Date.now(),
      });
    } catch (error) {
      console.error('Error adding memory node:', error);
      throw error;
    }
  }

  async addMemoryEdge(sourceId: string, targetId: string, relation: MemoryEdge['relation']): Promise<void> {
    try {
      await this.graphService.storeMemoryEdge({
        sourceId,
        targetId,
        relation,
        weight: 1.0,
        timestamp: Date.now(),
      });
    } catch (error) {
      console.error('Error adding memory edge:', error);
      throw error;
    }
  }

  // ==================== Provider Management ====================

  getAvailableProviders(): LLMProvider[] {
    return this.llmService.getAvailableProviders();
  }

  isProviderAvailable(provider: LLMProvider): boolean {
    return this.llmService.isProviderAvailable(provider);
  }

  getProviderName(provider: LLMProvider): string {
    return this.llmService.getProviderName(provider);
  }

  getProviderIcon(provider: LLMProvider): string {
    return this.llmService.getProviderIcon(provider);
  }

  // ==================== Utility Methods ====================

  async exportChatHistory(): Promise<string> {
    try {
      const sessionId = this.getCurrentSessionId();
      const conversations = await this.graphService.getConversations(sessionId);

      return JSON.stringify(conversations, null, 2);
    } catch (error) {
      console.error('Error exporting chat history:', error);
      throw error;
    }
  }

  async clearChatHistory(): Promise<void> {
    // Create a new session to effectively clear history
    await this.createNewSession('New Chat', 'gemini');
  }

  async clearAllData(): Promise<void> {
    // Clear mock data (when using localStorage)
    this.graphService.clearMockData();
    sessionStorage.removeItem('chat_brain_session_id');
  }
}

// Singleton instance
let chatBrainServiceInstance: ChatBrainService | null = null;

export const getChatBrainService = (): ChatBrainService => {
  if (!chatBrainServiceInstance) {
    chatBrainServiceInstance = new ChatBrainService();
  }
  return chatBrainServiceInstance;
};

export default getChatBrainService;
