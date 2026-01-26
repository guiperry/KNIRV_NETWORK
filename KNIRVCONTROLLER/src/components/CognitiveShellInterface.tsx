import React from 'react';


import { useState, useEffect, useRef, useCallback } from 'react';
import { Brain, Activity, Zap, Settings, Trash2 } from 'lucide-react';
import { cognitiveEngineService, CognitiveProcessingRequest } from '../services/CognitiveEngineService';
import { CognitiveEngine, CognitiveConfig, CognitiveState } from '../sensory-shell/CognitiveEngine';
import { HRMBridge } from '../sensory-shell/HRMBridge';
import { WASMOrchestrator } from '../sensory-shell/WASMOrchestrator';
import { ChatBrainProvider } from '../contexts/ChatBrainContext';
import { ChatInterface } from './chat-brain/ChatInterface';
import { MemoryGraphView } from './chat-brain/MemoryGraphView';
import { NotesPanel } from './chat-brain/NotesPanel';
import { LLMSelector } from './chat-brain/LLMSelector';
import { useChatBrain } from '../contexts/ChatBrainContext';

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
  const [engineState, setEngineState] = useState<CognitiveState | null>(null);
  const [wasmOrchestrator, setWasmOrchestrator] = useState<WASMOrchestrator | null>(null);
  const [isRunning, setIsRunning] = useState(false);
  const [metrics, setMetrics] = useState<Record<string, unknown> | null>(null);
  const [learningMode, setLearningMode] = useState(false);
  const [showSettings, setShowSettings] = useState(false);
  const [showMemoryGraph, setShowMemoryGraph] = useState(false);
  const [showNotes, setShowNotes] = useState(false);

  const [currentInput, setCurrentInput] = useState('');
  // Real-time conversation state
  const [conversationMessages, setConversationMessages] = useState<ConversationMessage[]>([]);
  const { clearChat } = useChatBrain();

  // Load engine metrics
  const loadEngineMetrics = async () => {
    try {
      if (cognitiveEngine) {
        const engineMetrics = await cognitiveEngine.getMetrics();
        setMetrics(engineMetrics as Record<string, unknown>);

        // Initialize WASM orchestrator if not already done
        if (!wasmOrchestrator && cognitiveEngine.isWASMAgentReady()) {
          const orchestrator = new WASMOrchestrator();
          setWasmOrchestrator(orchestrator);
        }
      }
    } catch (error) {
      console.error('Failed to load engine metrics:', error);
    }
  };
  const [isProcessing, setIsProcessing] = useState(false);
  const [hrmBridge, setHrmBridge] = useState<HRMBridge | null>(null);
  const [config, setConfig] = useState<CognitiveConfig>({
    maxContextSize: 100,
    learningRate: 0.01,
    adaptationThreshold: 0.3,
    skillTimeout: 30000,
    voiceEnabled: true,
    visualEnabled: true,
    loraEnabled: true,
    enhancedLoraEnabled: false,
    hrmEnabled: true,
    wasmAgentsEnabled: true,
    typeScriptCompilerEnabled: false,
    adaptiveLearningEnabled: true,
    walletIntegrationEnabled: true,
    chainIntegrationEnabled: true,
    ecosystemCommunicationEnabled: true,
    errorContextEnabled: false,
  });

  const engineRef = useRef<CognitiveEngine | null>(null);

  // Track cleanup function to ensure proper disposal
  const cleanupRef = useRef<(() => void) | null>(null);

  // Load engine status from service
  const loadEngineStatus = useCallback(async () => {
    try {
      const running = cognitiveEngineService.isEngineRunning();
      setIsRunning(running);
    } catch (error) {
      console.error('Failed to load engine status:', error);
    }
  }, []);

  const initializeHRMBridge = async () => {
    try {
      const bridge = new HRMBridge({
        l_module_count: 8,
        h_module_count: 4,
        enable_adaptation: true,
        processing_timeout: 5000
      });

      await bridge.initialize();
      setHrmBridge(bridge);
      console.log('HRM Bridge initialized successfully:', bridge);
    } catch (error) {
      console.error('Failed to initialize HRM Bridge:', error);
    }
  };

  const initializeWASMOrchestrator = async () => {
    try {
      const orchestrator = new WASMOrchestrator();
      await orchestrator.initialize();
      setWasmOrchestrator(orchestrator);
      console.log('WASM Orchestrator initialized successfully:', orchestrator);
    } catch (error) {
      console.error('Failed to initialize WASM Orchestrator:', error);
    }
  };


  const updateMetrics = useCallback(() => {
    if (cognitiveEngine) {
      const newMetrics = cognitiveEngine.getMetrics();
      setMetrics(newMetrics as Record<string, unknown>);
    }
  }, [cognitiveEngine]);

  const initializeCognitiveEngine = useCallback(async () => {
    try {
      const engine = new CognitiveEngine(config);
      engineRef.current = engine;
      setCognitiveEngine(engine);

      // Get initial state
      if (typeof engine.getState === 'function') {
        const initialState = engine.getState();
        setEngineState(initialState);
        if (onStateChange) {
          onStateChange(initialState);
        }
      }

      // Connect HRM Bridge if available
      if (hrmBridge && engine.isHRMLoRABridgeReady()) {
        console.log('HRM Bridge connected to Cognitive Engine');
      }

      // Set up event listeners
      engine.on('engineStarted', () => {
        setIsRunning(true);
        console.log('Cognitive Engine started');
      });

      engine.on('engineStopped', () => {
        setIsRunning(false);
        console.log('Cognitive Engine stopped');
      });

      engine.on('inputProcessed', (data) => {
        console.log('Input processed:', data);
        updateMetrics();
      });

      engine.on('skillInvoked', (data: unknown) => {
        console.log('Skill invoked:', data);
        if (onSkillInvoked) {
          const skillData = data as { skillId: string; result: unknown };
          onSkillInvoked(skillData.skillId, skillData.result);
        }
      });

      engine.on('adaptationTriggered', (data) => {
        console.log('Adaptation triggered:', data);
        if (onAdaptationTriggered) {
          onAdaptationTriggered(data);
        }
      });

      engine.on('learningModeStarted', () => {
        setLearningMode(true);
      });

      engine.on('cognitiveEvent', (event) => {
        console.log('Cognitive event:', event);
      });

      // Add event listeners expected by tests
      engine.on('stateChanged', (state: unknown) => {
        console.log('State changed:', state);
        const cognitiveState = state as CognitiveState;
        setEngineState(cognitiveState);
        if (onStateChange) {
          onStateChange(cognitiveState);
        }
      });

      engine.on('skillActivated', (data: unknown) => {
        console.log('Skill activated:', data);
        if (onSkillInvoked) {
          const skillData = data as { skillId: string; result: unknown };
          onSkillInvoked(skillData.skillId, skillData.result);
        }
      });

      engine.on('learningEvent', (event) => {
        console.log('Learning event:', event);
        updateMetrics();
      });

      // Update state periodically - but only in non-test environment
      let stateInterval: NodeJS.Timeout | null = null;

      if (process.env.NODE_ENV !== 'test') {
        stateInterval = setInterval(() => {
          if (engine && engineRef.current === engine) {
            const state = engine.getState();
            setEngineState(state);
            if (onStateChange) {
              onStateChange(state);
            }
          }
        }, 1000) as unknown as NodeJS.Timeout;
      }

      // Store cleanup function
      const cleanup = () => {
        if (stateInterval) {
          clearInterval(stateInterval);
          stateInterval = null;
        }
      };

      cleanupRef.current = cleanup;

      return cleanup;

    } catch (error) {
      console.error('Failed to initialize Cognitive Engine:', error);
      return () => {}; // Return empty cleanup function on error
    }
  }, [config, onStateChange, onSkillInvoked, onAdaptationTriggered, updateMetrics, hrmBridge]);

  useEffect(() => {
    initializeCognitiveEngine();
    initializeHRMBridge();
    initializeWASMOrchestrator();
    loadEngineStatus();

    // Auto-start cognitive engine after initialization
    const autoStartEngine = async () => {
      try {
        if (cognitiveEngine && !isRunning) {
          await cognitiveEngine.start();
          console.log('Cognitive Engine auto-started successfully');
          loadEngineMetrics();
        }
      } catch (error) {
        console.error('Failed to auto-start Cognitive Engine:', error);
      }
    };

    // Small delay to ensure initialization is complete
    const autoStartTimer = setTimeout(autoStartEngine, 1000);

    // Refresh metrics every 5 seconds
    const metricsInterval = setInterval(updateMetrics, 5000);

    return () => {
      clearTimeout(autoStartTimer);
      clearInterval(metricsInterval);
      if (engineRef.current) {
        // Force stop engine immediately
        const cleanup = async () => {
          try {
            console.log('Starting CognitiveEngine cleanup...');

            // Stop engine first if stop method exists
            if (typeof engineRef.current?.stop === 'function') {
              await engineRef.current.stop();
            }

            // Remove event listeners if removeAllListeners method exists
            if (typeof engineRef.current?.removeAllListeners === 'function') {
              engineRef.current.removeAllListeners();
            }

            // Dispose resources if dispose method exists
            if (typeof engineRef.current?.dispose === 'function') {
              await engineRef.current.dispose();
            }

            // Clear the reference
            engineRef.current = null;
            setCognitiveEngine(null);
            console.log('CognitiveEngine cleanup completed');
          } catch (error) {
            console.error('Error during cleanup:', error);
            // Force clear even if cleanup fails
            engineRef.current = null;
            setCognitiveEngine(null);
          }
        };

        // Clear any existing cleanup function
        if (cleanupRef.current) {
          cleanupRef.current();
          cleanupRef.current = null;
        }

        // In test environment, force immediate synchronous cleanup
        if (process.env.NODE_ENV === 'test') {
          // Force synchronous cleanup for tests
          try {
            if (typeof engineRef.current?.stop === 'function') {
              // Don't await in test environment to prevent hanging
              engineRef.current.stop().catch(() => {});
            }
            if (typeof engineRef.current?.removeAllListeners === 'function') {
              engineRef.current.removeAllListeners();
            }
            if (typeof engineRef.current?.dispose === 'function') {
              engineRef.current.dispose();
            }
            engineRef.current = null;
            setCognitiveEngine(null);
          } catch (error) {
            console.error('Error during test cleanup:', error);
            engineRef.current = null;
            setCognitiveEngine(null);
          }
        } else {
          cleanup().catch((error) => {
            console.error('Async cleanup failed:', error);
            // Force clear if async cleanup fails
            engineRef.current = null;
            setCognitiveEngine(null);
          });
        }
      }
    };
  }, [initializeCognitiveEngine, loadEngineStatus, updateMetrics]);


  // Process conversation message using service
  const handleSendMessage = async () => {
    if (!currentInput.trim() || isProcessing) return;

    const userMessage: ConversationMessage = {
      id: `msg_${Date.now()}`,
      type: 'user',
      content: currentInput.trim(),
      timestamp: new Date()
    };

    setConversationMessages(prev => [...prev, userMessage]);
    setCurrentInput('');
    setIsProcessing(true);

    try {
      const processingRequest: CognitiveProcessingRequest = {
        input: userMessage.content,
        context: {},
        taskType: 'conversation',
        requiresSkillInvocation: false
      };

      const result = await cognitiveEngineService.processInput(processingRequest);

      const assistantMessage: ConversationMessage = {
        id: `msg_${Date.now() + 1}`,
        type: 'assistant',
        content: result.output,
        timestamp: new Date(),
        processingTime: result.processingTime,
        skillsInvoked: result.skillsInvoked
      };

      setConversationMessages(prev => [...prev, assistantMessage]);

      // Trigger callbacks
      if (result.skillsInvoked.length > 0) {
        result.skillsInvoked.forEach(skillId => {
          onSkillInvoked?.(skillId, result.output);
        });
      }

      if (result.adaptationTriggered) {
        onAdaptationTriggered?.(result.contextUpdates);
      }

      // Update conversation callback
      const updatedMessages = [...conversationMessages, userMessage, assistantMessage];
      onConversationUpdate?.(updatedMessages);

    } catch (error) {
      console.error('Failed to process message:', error);

      const errorMessage: ConversationMessage = {
        id: `msg_${Date.now() + 1}`,
        type: 'system',
        content: `Error: ${error instanceof Error ? error.message : 'Processing failed'}`,
        timestamp: new Date()
      };

      setConversationMessages(prev => [...prev, errorMessage]);
    } finally {
      setIsProcessing(false);
    }
  };

  const handleStart = async () => {
    if (cognitiveEngine && !isRunning) {
      try {
        await cognitiveEngine.start();
        loadEngineMetrics(); // Call loadEngineMetrics on start
      } catch (error) {
        console.error('Failed to start Cognitive Engine:', error);
      }
    }
  };

  const handleStop = async () => {
    if (cognitiveEngine && isRunning) {
      try {
        await cognitiveEngine.stop();
        loadEngineMetrics(); // Call loadEngineMetrics on stop
      } catch (error) {
        console.error('Failed to stop Cognitive Engine:', error);
      }
    }
  };

  const handleUpdateConfig = useCallback((newConfig: Partial<CognitiveConfig>) => {
    setConfig(prevConfig => ({ ...prevConfig, ...newConfig }));
  }, []);

  const handleClearChat = () => {
    if (confirm('Are you sure you want to clear chat history?')) {
      clearChat();
    }
  };

  const getStatusColor = () => {
    if (!isRunning) return 'text-gray-400';
    if (learningMode) return 'text-blue-400';
    return 'text-green-400';
  };

  const getStatusText = () => {
    if (!isRunning) return 'Offline';
    if (learningMode) return 'Learning';
    return 'Active';
  };

  // Re-fetch state on every render (for test rerenders)
  useEffect(() => {
    if (cognitiveEngine && typeof cognitiveEngine.getState === 'function') {
      const currentState = cognitiveEngine.getState();
      if (onStateChange) {
        onStateChange(currentState);
      }
    }
  }, [cognitiveEngine, onStateChange]);

  return (
    <div data-testid="cognitive-shell-interface" className="bg-gray-800/90 backdrop-blur-sm rounded-lg border border-gray-700/50 h-full overflow-hidden flex flex-col">
        {/* Engine Status Header - always visible */}
        <div className="flex-shrink-0 border-b border-gray-700/50">
          <div className="flex items-center justify-between p-3">
            <div className="flex items-center space-x-2">
              <div className="w-6 h-6 bg-gradient-to-r from-purple-500 to-pink-500 rounded-lg flex items-center justify-center">
                <Brain className="w-4 h-4 text-white" />
              </div>
              <div className="flex items-center space-x-2">
                <span className="text-sm font-semibold text-white">Engine Status:</span>
                <span className={`text-xs ${getStatusColor()} px-2 py-1 rounded bg-gray-700/50`}>
                  {getStatusText()}
                </span>
                <span className="text-xs text-gray-400">
                  Confidence: {Math.round(((metrics?.confidenceLevel as number) || 0.95) * 100)}%
                </span>
              </div>
            </div>
            
            <div className="flex items-center space-x-2">
              <div className="text-xs text-gray-400">
                Adaptation: {Math.round(((metrics?.adaptationLevel as number) || 0.75) * 100)}%
              </div>
              
              {isRunning ? (
                <button
                  onClick={handleStop}
                  className="px-2 py-1 bg-red-500\\/20 text-red-400 rounded border border-red-500\\/30 hover:bg-red-500\\/30 transition-colors text-xs"
                >
                  Stop
                </button>
              ) : (
                <button
                  onClick={handleStart}
                  className="px-2 py-1 bg-green-500\\/20 text-green-400 rounded border border-green-500\\/30 hover:bg-green-500\\/30 transition-colors text-xs"
                >
                  Start
                </button>
              )}
            </div>
          </div>

          {/* ChatBrain Controls Row */}
          <div className="p-3 bg-gray-800/50">
            <div className="flex items-center justify-between">
              <div className="flex items-center space-x-4">
                <LLMSelector />
                <span className="text-sm text-gray-400">Chat-Brain</span>
              </div>
              
              <div className="flex items-center space-x-2">
                <button
                  onClick={() => setShowMemoryGraph(!showMemoryGraph)}
                  className={`px-3 py-1 rounded-lg text-xs font-medium transition-colors flex items-center space-x-1 ${
                    showMemoryGraph
                      ? 'bg-blue-600 text-white'
                      : 'bg-gray-700 hover:bg-gray-600 text-gray-300'
                  }`}
                >
                  <Activity size={14} />
                  <span>Memory</span>
                </button>

                <button
                  onClick={() => setShowNotes(!showNotes)}
                  className={`px-3 py-1 rounded-lg text-xs font-medium transition-colors flex items-center space-x-1 ${
                    showNotes
                      ? 'bg-blue-600 text-white'
                      : 'bg-gray-700 hover:bg-gray-600 text-gray-300'
                  }`}
                >
                  <Settings size={14} />
                  <span>Notes</span>
                </button>

                <button
                  onClick={handleClearChat}
                  className="px-3 py-1 bg-gray-700 hover:bg-gray-600 text-gray-300 rounded-lg text-xs font-medium transition-colors flex items-center space-x-1"
                >
                  <Trash2 size={14} />
                  <span>Clear</span>
                </button>
              </div>
            </div>
          </div>
        </div>

        {/* Chat Interface - takes remaining space */}
        <div className="flex-1 overflow-hidden">
          <ChatInterface />
        </div>

        {/* Side Panels */}
        <div className="flex space-x-4">
          {showMemoryGraph && <MemoryGraphView onClose={() => setShowMemoryGraph(false)} />}
          {showNotes && <NotesPanel onClose={() => setShowNotes(false)} />}
        </div>
    </div>
  );
};

export default CognitiveShellInterface;