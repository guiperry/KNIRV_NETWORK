// Chat-Brain Context - Global State Management
import React, { createContext, useState, useContext, useCallback, ReactNode } from 'react';
import type {
  ChatMessage,
  LLMProvider,
  MemoryNode,
  MemoryEdge,
  Note,
} from '../types/chatBrain';
import { getChatBrainService } from '../services/chatBrainService';

interface ChatBrainContextType {
  // Chat state
  messages: ChatMessage[];
  addMessage: (message: ChatMessage) => void;
  clearChat: () => void;
  sendMessage: (text: string) => Promise<void>;

  // Memory graph state
  memoryNodes: MemoryNode[];
  memoryEdges: MemoryEdge[];
  refreshMemoryGraph: () => Promise<void>;

  // Notes state
  notes: Note[];
  currentNote: Note | null;
  setCurrentNote: (note: Note | null) => void;
  saveNote: (title: string, content: string) => Promise<void>;
  loadNotes: () => Promise<void>;
  deleteNote: (noteId: string) => Promise<void>;

  // LLM provider state
  selectedProvider: LLMProvider;
  setSelectedProvider: (provider: LLMProvider) => void;
  availableProviders: LLMProvider[];

  // UI state
  isLoading: boolean;
  error: string | null;
  clearError: () => void;
}

const ChatBrainContext = createContext<ChatBrainContextType | undefined>(undefined);

export function ChatBrainProvider({ children }: { children: ReactNode }) {
  const chatBrainService = getChatBrainService();

  // Chat state
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Memory graph state
  const [memoryNodes, setMemoryNodes] = useState<MemoryNode[]>([]);
  const [memoryEdges, setMemoryEdges] = useState<MemoryEdge[]>([]);

  // Notes state
  const [notes, setNotes] = useState<Note[]>([]);
  const [currentNote, setCurrentNote] = useState<Note | null>(null);

  // LLM provider state
  const [selectedProvider, setSelectedProvider] = useState<LLMProvider>('gemini');
  const [availableProviders] = useState<LLMProvider[]>(
    chatBrainService.getAvailableProviders()
  );

  // ==================== Chat Operations ====================

  const addMessage = useCallback((message: ChatMessage) => {
    setMessages((prev) => [...prev, message]);
  }, []);

  const clearChat = useCallback(() => {
    setMessages([]);
    chatBrainService.clearChatHistory();
  }, [chatBrainService]);

  const sendMessage = useCallback(
    async (text: string) => {
      if (!text.trim()) return;

      setIsLoading(true);
      setError(null);

      try {
        // Add user message
        const userMessage: ChatMessage = {
          id: `msg_${Date.now()}_user`,
          text,
          type: 'user',
          timestamp: Date.now(),
        };
        addMessage(userMessage);

        // Get AI response
        const response = await chatBrainService.sendMessage(text, selectedProvider, messages);

        // Add AI message
        const aiMessage: ChatMessage = {
          id: `msg_${Date.now()}_bot`,
          text: response.text,
          type: 'bot',
          timestamp: Date.now(),
          provider: selectedProvider,
          metadata: response.metadata,
        };
        addMessage(aiMessage);
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
        setError(errorMessage);

        // Add error message to chat
        const errorMsg: ChatMessage = {
          id: `msg_${Date.now()}_error`,
          text: `Error: ${errorMessage}`,
          type: 'system',
          timestamp: Date.now(),
        };
        addMessage(errorMsg);
      } finally {
        setIsLoading(false);
      }
    },
    [selectedProvider, messages, addMessage, chatBrainService]
  );

  // ==================== Memory Graph Operations ====================

  const refreshMemoryGraph = useCallback(async () => {
    try {
      const graph = await chatBrainService.getMemoryGraph();
      setMemoryNodes(graph.nodes);
      setMemoryEdges(graph.edges);
    } catch (err) {
      console.error('Error refreshing memory graph:', err);
      setError(err instanceof Error ? err.message : 'Failed to refresh memory graph');
    }
  }, [chatBrainService]);

  // ==================== Notes Operations ====================

  const saveNote = useCallback(
    async (title: string, content: string) => {
      try {
        setIsLoading(true);
        setError(null);

        if (currentNote) {
          // Update existing note
          await chatBrainService.updateNote(currentNote.id, title, content);
        } else {
          // Create new note
          await chatBrainService.saveNote(title, content);
        }

        // Reload notes
        await loadNotes();
        setCurrentNote(null);
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : 'Failed to save note';
        setError(errorMessage);
        throw err;
      } finally {
        setIsLoading(false);
      }
    },
    [currentNote, chatBrainService]
  );

  const loadNotes = useCallback(async () => {
    try {
      const loadedNotes = await chatBrainService.loadNotes();
      setNotes(loadedNotes);
    } catch (err) {
      console.error('Error loading notes:', err);
      setError(err instanceof Error ? err.message : 'Failed to load notes');
    }
  }, [chatBrainService]);

  const deleteNote = useCallback(
    async (noteId: string) => {
      try {
        setIsLoading(true);
        setError(null);

        await chatBrainService.deleteNote(noteId);
        await loadNotes();

        if (currentNote?.id === noteId) {
          setCurrentNote(null);
        }
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : 'Failed to delete note';
        setError(errorMessage);
        throw err;
      } finally {
        setIsLoading(false);
      }
    },
    [currentNote, chatBrainService, loadNotes]
  );

  // ==================== Error Handling ====================

  const clearError = useCallback(() => {
    setError(null);
  }, []);

  // ==================== Context Value ====================

  const value: ChatBrainContextType = {
    // Chat
    messages,
    addMessage,
    clearChat,
    sendMessage,

    // Memory graph
    memoryNodes,
    memoryEdges,
    refreshMemoryGraph,

    // Notes
    notes,
    currentNote,
    setCurrentNote,
    saveNote,
    loadNotes,
    deleteNote,

    // LLM provider
    selectedProvider,
    setSelectedProvider,
    availableProviders,

    // UI state
    isLoading,
    error,
    clearError,
  };

  return <ChatBrainContext.Provider value={value}>{children}</ChatBrainContext.Provider>;
}

// ==================== Custom Hook ====================

export const useChatBrain = (): ChatBrainContextType => {
  const context = useContext(ChatBrainContext);

  if (!context) {
    throw new Error('useChatBrain must be used within a ChatBrainProvider');
  }

  return context;
};
