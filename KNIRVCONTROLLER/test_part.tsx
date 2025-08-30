import React, { useState, useEffect, useRef, useCallback } from 'react';
import { Brain, Activity, Zap, Eye, Mic, Settings, BarChart3, Cpu, MessageSquare, Send } from 'lucide-react';
import { cognitiveEngineService, CognitiveProcessingRequest, SkillExecutionRequest, CognitiveMetrics } from '../services/CognitiveEngineService';
import { CognitiveEngine, CognitiveConfig, CognitiveState } from '../sensory-shell/CognitiveEngine';
import { HRMBridge } from '../sensory-shell/HRMBridge';
import { WASMOrchestrator } from '../sensory-shell/WASMOrchestrator';

interface ConversationMessage {
  id: string;
  type: 'user' | 'assistant' | 'system';
  content: string;
  timestamp: Date;
  processingTime?: number;
  hrmResponse?: unknown;
  skillsInvoked?: string[];
}

interface CognitiveShellInterfaceProps {
  onStateChange?: (state: CognitiveState) => void;
  onSkillInvoked?: (skillId: string, result: unknown) => void;
  onAdaptationTriggered?: (adaptation: unknown) => void;
  onConversationUpdate?: (messages: ConversationMessage[]) => void;
}

export const CognitiveShellInterface: React.FC<CognitiveShellInterfaceProps> = ({
  onStateChange,
  onSkillInvoked,
  onAdaptationTriggered,
  onConversationUpdate,
}) => {
  const [cognitiveEngine, setCognitiveEngine] = useState<CognitiveEngine | null>(null);
  const [isEngineRunning, setIsEngineRunning] = useState(false);
  const [engineMetrics, setEngineMetrics] = useState<CognitiveMetrics | null>(null);
  const [engineState, setEngineState] = useState<CognitiveState | null>(null);
  const [isRunning, setIsRunning] = useState(false);
  const [metrics, setMetrics] = useState<Record<string, unknown> | null>(null);
  const [learningMode, setLearningMode] = useState(false);
  const [showSettings, setShowSettings] = useState(false);
  const [commandHistory, setCommandHistory] = useState<Array<{input: string, output: string}>>([]);

  // Real-time conversation state
  const [conversationMessages, setConversationMessages] = useState<ConversationMessage[]>([]);
  const [currentInput, setCurrentInput] = useState('');
  const [isProcessing, setIsProcessing] = useState(false);
  const [hrmBridge, setHrmBridge] = useState<HRMBridge | null>(null);
  const [wasmOrchestrator, setWasmOrchestrator] = useState<WASMOrchestrator | null>(null);
  const [showConversation, setShowConversation] = useState(true);
  const [config, setConfig] = useState<CognitiveConfig>({
    maxContextSize: 100,
    learningRate: 0.01,
};
