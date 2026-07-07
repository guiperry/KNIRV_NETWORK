// Global type declarations for KNIRV-CONTROLLER

// Web Speech API types
interface SpeechRecognition extends EventTarget {
  continuous: boolean;
  interimResults: boolean;
  lang: string;
  start(): void;
  stop(): void;
  onstart: ((this: SpeechRecognition, ev: Event) => void) | null;
  onresult: ((this: SpeechRecognition, ev: SpeechRecognitionEvent) => void) | null;
  onerror: ((this: SpeechRecognition, ev: SpeechRecognitionErrorEvent) => void) | null;
  onend: ((this: SpeechRecognition, ev: Event) => void) | null;
}

interface SpeechRecognitionEvent extends Event {
  results: SpeechRecognitionResultList;
}

interface SpeechRecognitionErrorEvent extends Event {
  error: string;
}

interface SpeechRecognitionResultList {
  length: number;
  item(index: number): SpeechRecognitionResult;
  [index: number]: SpeechRecognitionResult;
}

interface SpeechRecognitionResult {
  length: number;
  item(index: number): SpeechRecognitionAlternative;
  [index: number]: SpeechRecognitionAlternative;
  isFinal: boolean;
}

interface SpeechRecognitionAlternative {
  transcript: string;
  confidence: number;
}

declare const SpeechRecognition: {
  prototype: SpeechRecognition;
  new(): SpeechRecognition;
};

declare const webkitSpeechRecognition: {
  prototype: SpeechRecognition;
  new(): SpeechRecognition;
};

// Extend Window interface
interface Window {
  SpeechRecognition?: typeof SpeechRecognition;
  webkitSpeechRecognition?: typeof webkitSpeechRecognition;
}

// Voice Integration types
export interface VoiceStatus {
  isActive: boolean;
  isListening: boolean;
  isProcessing: boolean;
  isSpeaking: boolean;
  hasError: boolean;
}

export interface EdgeColoringState {
  color: string;
  intensity: number;
  isAnimating: boolean;
}

// Cognitive Shell types
export interface CognitiveState {
  mode: 'basic' | 'advanced';
  isLearning: boolean;
  adaptationLevel: number;
  badgesActive: string[];
}

// Workflow types
export interface WorkflowMetrics {
  performance: number;
  tasksCompleted: number;
  errorRate: number;
  lastActive: Date;
}

// NRV (Network Resource Virtualization) types
export interface NRVData {
  id: string;
  type: 'badge' | 'workflow' | 'system';
  status: 'active' | 'idle' | 'error';
  metrics: WorkflowMetrics;
}
