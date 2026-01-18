// Chat-Brain Type Definitions

export type LLMProvider = 'gemini' | 'openai' | 'deepseek' | 'adaline';

export type MessageType = 'user' | 'bot' | 'system';

export interface ChatMessage {
  id: string;
  text: string;
  type: MessageType;
  timestamp: number;
  provider?: LLMProvider;
  metadata?: Record<string, unknown>;
}

export interface ChatResponse {
  text: string;
  provider: string;
  metadata?: Record<string, unknown>;
}

// Memory Graph Types
export type MemoryNodeType = 'entity' | 'concept' | 'event' | 'keyword' | 'topic' | 'custom' | 'skill';

export interface MemoryNode {
  id: string;
  label: string;
  type: MemoryNodeType;
  properties?: Record<string, unknown>;
  embedding?: number[];
  timestamp: number;
  userId?: string;
}

export type MemoryRelationType = 'related_to' | 'mentioned_in' | 'part_of' | 'temporal' | 'caused_by' | 'custom';

export interface MemoryEdge {
  id?: string;
  sourceId: string;
  targetId: string;
  relation: MemoryRelationType;
  weight?: number;
  properties?: Record<string, unknown>;
  timestamp?: number;
}

export interface MemoryGraph {
  nodes: MemoryNode[];
  edges: MemoryEdge[];
}

// Notes Types
export interface Note {
  id: string;
  title: string;
  content: string;
  timestamp: number;
  userId?: string;
  tags?: string[];
  metadata?: Record<string, unknown>;
}

// Conversation Types
export interface Conversation {
  id: string;
  sessionId?: string;
  messages: ChatMessage[];
  timestamp: number;
  userId?: string;
  metadata?: Record<string, unknown>;
}

// Persona Analytics Types
export interface PersonaMetrics {
  sentiment: SentimentAnalysis;
  interests: InterestProfile[];
  goals: Goal[];
  personality: PersonalityTraits;
  errors: ErrorLog[];
  tools: ToolUsage[];
}

export interface SentimentAnalysis {
  polarity: number; // -1 to 1
  emotion: 'positive' | 'negative' | 'neutral';
  confidence: number;
}

export interface InterestProfile {
  topic: string;
  score: number;
  lastUpdated: number;
  decay?: number;
}

export interface Goal {
  id: string;
  description: string;
  status: 'active' | 'completed' | 'abandoned';
  progress: number;
  timestamp: number;
}

export interface PersonalityTraits {
  openness: number; // 0-1
  conscientiousness: number;
  extraversion: number;
  agreeableness: number;
  neuroticism: number;
}

export interface ErrorLog {
  id: string;
  type: string;
  message: string;
  timestamp: number;
  resolved: boolean;
}

export interface ToolUsage {
  tool: string;
  count: number;
  lastUsed: number;
}

// KNIRVGRAPH API Types
export interface KNIRVGraphConfig {
  nodeEndpoint: string;
  chainId: string;
  apiKey?: string;
}

export interface KNIRVGraphResponse<T> {
  success: boolean;
  data?: T;
  error?: string;
  timestamp: number;
}

// Chat Session Types
export interface ChatSession {
  id: string;
  name: string;
  messages: ChatMessage[];
  createdAt: number;
  updatedAt: number;
  provider: LLMProvider;
}

// Settings Types
export interface ChatBrainSettings {
  defaultProvider: LLMProvider;
  apiKeys: {
    gemini?: string;
    openai?: string;
    deepseek?: string;
    adaline?: string;
  };
  knirvGraph: KNIRVGraphConfig;
  features: {
    memoryGraph: boolean;
    personaAnalytics: boolean;
    autoSaveNotes: boolean;
  };
  maxContextLength: number;
}
