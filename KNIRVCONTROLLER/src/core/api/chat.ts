import { Request, Response } from 'express';
import { databaseService } from '../services/databaseService';

export interface ChatMessage {
  id: string;
  content: string;
  sender: 'user' | 'ai';
  timestamp: Date;
}

export interface ChatSession {
  _id?: string;
  title: string;
  createdAt: Date;
  updatedAt: Date;
  messages: ChatMessage[];
}

/**
 * Get all chat sessions
 */
export const getSessions = async (req: Request, res: Response) => {
  try {
    const sessions = await databaseService.listChatSessions();
    res.json({ sessions });
  } catch (error) {
    console.error('Error fetching chat sessions:', error);
    res.status(500).json({ error: 'Failed to fetch chat sessions' });
  }
};

/**
 * Get a specific chat session by ID
 */
export const getSession = async (req: Request, res: Response) => {
  try {
    const { sessionId } = req.params;
    const session = await databaseService.getChatSession(sessionId);
    
    if (!session) {
      return res.status(404).json({ error: 'Chat session not found' });
    }
    
    res.json({ session });
  } catch (error) {
    console.error('Error fetching chat session:', error);
    res.status(500).json({ error: 'Failed to fetch chat session' });
  }
};

/**
 * Create a new chat session
 */
export const createSession = async (req: Request, res: Response) => {
  try {
    const { title, messages } = req.body;
    
    const sessionData = {
      title,
      messages: messages || [],
      updatedAt: new Date().toISOString(),
    };
    
    const newSession = await databaseService.createChatSession(sessionData);
    res.status(201).json({ sessionId: newSession.id, session: newSession });
  } catch (error) {
    console.error('Error creating chat session:', error);
    res.status(500).json({ error: 'Failed to create chat session' });
  }
};

/**
 * Update a chat session
 */
export const updateSession = async (req: Request, res: Response) => {
  try {
    const { sessionId } = req.params;
    const { title, messages } = req.body;
    
    const updateData: Record<string, unknown> = {
      updatedAt: new Date(),
    };
    
    if (title !== undefined) {
      updateData.title = title;
    }
    
    if (messages !== undefined) {
      updateData.messages = messages;
    }
    
    const updatedSession = await databaseService.updateChatSession(sessionId, updateData);
    
    if (!updatedSession) {
      return res.status(404).json({ error: 'Chat session not found' });
    }
    
    res.json({ session: updatedSession });
  } catch (error) {
    console.error('Error updating chat session:', error);
    res.status(500).json({ error: 'Failed to update chat session' });
  }
};

/**
 * Delete a chat session
 */
export const deleteSession = async (req: Request, res: Response) => {
  try {
    const { sessionId } = req.params;
    
    const deletedSession = await databaseService.deleteChatSession(sessionId);
    
    if (!deletedSession) {
      return res.status(404).json({ error: 'Chat session not found' });
    }
    
    res.json({ message: 'Chat session deleted successfully' });
  } catch (error) {
    console.error('Error deleting chat session:', error);
    res.status(500).json({ error: 'Failed to delete chat session' });
  }
};

/**
 * Add a message to a chat session
 */
export const addMessage = async (req: Request, res: Response) => {
  try {
    const { sessionId } = req.params;
    const { content, sender } = req.body;
    
    if (!content || !sender) {
      return res.status(400).json({ error: 'Content and sender are required' });
    }
    
    if (!['user', 'ai'].includes(sender)) {
      return res.status(400).json({ error: 'Sender must be either "user" or "ai"' });
    }
    
    const session = await databaseService.getChatSession(sessionId);
    
    if (!session) {
      return res.status(404).json({ error: 'Chat session not found' });
    }
    
    const newMessage: ChatMessage = {
      id: `msg_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
      content,
      sender,
      timestamp: new Date(),
    };

    // Convert to database format
    const dbMessage = {
      id: newMessage.id,
      content: newMessage.content,
      sender: newMessage.sender,
      timestamp: newMessage.timestamp.toISOString(),
    };
    
    const updatedMessages = [...(session.messages || []), dbMessage];
    
    const updatedSession = await databaseService.updateChatSession(sessionId, {
      messages: updatedMessages,
      updatedAt: new Date().toISOString(),
    });
    
    res.json({ message: newMessage, session: updatedSession });
  } catch (error) {
    console.error('Error adding message to chat session:', error);
    res.status(500).json({ error: 'Failed to add message to chat session' });
  }
};

/**
 * Get messages from a chat session
 */
export const getMessages = async (req: Request, res: Response) => {
  try {
    const { sessionId } = req.params;
    const { limit, offset } = req.query;
    
    const session = await databaseService.getChatSession(sessionId);
    
    if (!session) {
      return res.status(404).json({ error: 'Chat session not found' });
    }
    
    let messages = session.messages || [];
    
    // Apply pagination if specified
    if (offset) {
      const offsetNum = parseInt(offset as string, 10);
      messages = messages.slice(offsetNum);
    }
    
    if (limit) {
      const limitNum = parseInt(limit as string, 10);
      messages = messages.slice(0, limitNum);
    }
    
    res.json({ messages, total: session.messages?.length || 0 });
  } catch (error) {
    console.error('Error fetching messages:', error);
    res.status(500).json({ error: 'Failed to fetch messages' });
  }
};

/**
 * Search chat sessions by title or content
 */
export const searchSessions = async (req: Request, res: Response) => {
  try {
    const { query } = req.query;
    
    if (!query || typeof query !== 'string') {
      return res.status(400).json({ error: 'Search query is required' });
    }
    
    // Get all sessions and filter client-side for now
    // TODO: Implement full-text search when NebulaDB plugin is added
    const allSessions = await databaseService.listChatSessions();
    
    const filteredSessions = allSessions.filter(session => {
      // Search in title
      if (session.title.toLowerCase().includes(query.toLowerCase())) {
        return true;
      }
      
      // Search in message content
      if (session.messages) {
        return session.messages.some(message => 
          message.content.toLowerCase().includes(query.toLowerCase())
        );
      }
      
      return false;
    });
    
    res.json({ sessions: filteredSessions, query });
  } catch (error) {
    console.error('Error searching chat sessions:', error);
    res.status(500).json({ error: 'Failed to search chat sessions' });
  }
};
