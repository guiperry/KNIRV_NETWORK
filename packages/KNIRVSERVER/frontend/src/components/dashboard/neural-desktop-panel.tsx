'use client';

import React, { useState, useEffect, useRef, useCallback } from 'react';
import {
  Terminal, Database, Shield, RefreshCw, ShieldAlert,
  CheckCircle, AlertCircle, Search, Compass, Loader2, Send, ChevronRight,
  Mic, MicOff, GitBranch, Network, Server, Users, Activity
} from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { webSocketService } from '@/lib/websocket-service';

interface Thought {
  id: string;
  timestamp: number;
  content: string;
  type: 'plan' | 'observation' | 'conclusion' | 'error' | 'instruction' | 'fact';
}

enum AgentStatus {
  IDLE = 'IDLE',
  THINKING = 'THINKING',
  EXECUTING = 'EXECUTING',
  LEARNING = 'LEARNING',
  ERROR = 'ERROR',
  LIVE = 'LIVE',
  EDITING = 'EDITING'
}

const STORAGE_KEYS = {
  THOUGHTS: 'aether_agent_thoughts',
};

const generateId = (content: string): string => {
  let hash = 0;
  for (let i = 0; i < content.length; i++) {
    const char = content.charCodeAt(i);
    hash = ((hash << 5) - hash) + char;
    hash = hash & hash;
  }
  return Math.abs(hash).toString(36) + Date.now().toString(36);
};

const saveThoughts = (thoughts: Thought[]) => {
  const uniqueThoughts = Array.from(new Map(thoughts.map(t => [t.content, t])).values());
  localStorage.setItem(STORAGE_KEYS.THOUGHTS, JSON.stringify(uniqueThoughts.slice(0, 100)));
};

const loadThoughts = (): Thought[] => {
  const data = localStorage.getItem(STORAGE_KEYS.THOUGHTS);
  return data ? JSON.parse(data) : [];
};

interface NeuralDesktopPanelProps {
  className?: string;
}

export const NeuralDesktopPanel: React.FC<NeuralDesktopPanelProps> = ({ className }) => {
  const [status, setStatus] = useState<AgentStatus>(AgentStatus.IDLE);
  const [thoughts, setThoughts] = useState<Thought[]>([]);
  const [input, setInput] = useState('');
  const [isListening, setIsListening] = useState(false);
  const [isKeyActive, setIsKeyActive] = useState(true);
  const [recognitionError, setRecognitionError] = useState<string | null>(null);
  const [wsConnected, setWsConnected] = useState(false);

  const recognitionRef = useRef<any>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const loadedThoughts = loadThoughts();
    if (loadedThoughts.length > 0) {
      setThoughts(loadedThoughts);
    } else {
      // Add demo thoughts for visual feedback when no data exists
      const demoThoughts: Thought[] = [
        {
          id: 'demo-1',
          timestamp: Date.now(),
          content: 'NEURAL_DESKTOP_INITIALIZED: System ready for autonomous heuristic reasoning',
          type: 'fact'
        },
        {
          id: 'demo-2',
          timestamp: Date.now() - 1000,
          content: 'COGNITIVE_ENGINE_STANDBY: Waiting for mission parameters',
          type: 'observation'
        }
      ];
      setThoughts(demoThoughts);
    }

    const SpeechRecognition = (window as any).SpeechRecognition || (window as any).webkitSpeechRecognition;
    if (SpeechRecognition) {
      recognitionRef.current = new SpeechRecognition();
      recognitionRef.current.continuous = true;
      recognitionRef.current.interimResults = true;
      recognitionRef.current.lang = 'en-US';

      recognitionRef.current.onresult = (event: any) => {
        let interimTranscript = '';
        let finalTranscript = '';

        for (let i = event.resultIndex; i < event.results.length; ++i) {
          if (event.results[i].isFinal) {
            finalTranscript += event.results[i][0].transcript;
          } else {
            interimTranscript += event.results[i][0].transcript;
          }
        }

        if (finalTranscript) {
          setInput(prev => (prev + ' ' + finalTranscript).trim());
        }
      };

      recognitionRef.current.onerror = (event: any) => {
        setRecognitionError(event.error);
        setIsListening(false);
      };

      recognitionRef.current.onend = () => {
        setIsListening(false);
      };
    }
  }, []);

  useEffect(() => {
    saveThoughts(thoughts);
  }, [thoughts]);

  useEffect(() => {
    if (containerRef.current) {
      containerRef.current.scrollTop = 0;
    }
  }, [thoughts]);

  // Must be declared before the useEffect below that uses it as a dependency.
  const addThought = useCallback((thought: Thought) => {
    setThoughts(prev => {
      if (prev.some(t => t.content === thought.content)) {
        return prev;
      }
      return [thought, ...prev].slice(0, 100);
    });
  }, []);

  // Subscribe to WebSocket neural_task_response events (Gap 9 / Gap 12)
  useEffect(() => {
    setWsConnected(webSocketService.getConnectionStatus());

    const handleConnection = (data: { connected: boolean }) => {
      setWsConnected(data.connected);
    };

    const handleNeuralResult = (payload: any) => {
      const thoughtType: Thought['type'] = payload.type === 'conclusion' ? 'conclusion' :
        payload.type === 'error' ? 'error' : 'conclusion';

      if (payload.error) {
        addThought({
          id: generateId(`ERROR: ${payload.error}`),
          timestamp: Date.now(),
          content: `ERROR: Inference engine returned: ${payload.error}`,
          type: 'error',
        });
      } else {
        addThought({
          id: generateId(`RESULT: ${payload.result}`),
          timestamp: Date.now(),
          content: `INFERENCE RESULT: ${payload.result}`,
          type: thoughtType,
        });
      }
      setStatus(AgentStatus.IDLE);
    };

    webSocketService.on('connection', handleConnection);
    webSocketService.on('neural_task_result', handleNeuralResult);
    webSocketService.on('neural_task_error', handleNeuralResult);
    webSocketService.subscribe(['neural_task_result', 'neural_task_error']);

    return () => {
      webSocketService.off('connection', handleConnection);
      webSocketService.off('neural_task_result', handleNeuralResult);
      webSocketService.off('neural_task_error', handleNeuralResult);
    };
  }, [addThought]);

  const toggleListening = () => {
    if (!recognitionRef.current) {
      alert("Speech recognition is not supported in this browser. Please use Chrome or Edge.");
      return;
    }

    if (isListening) {
      recognitionRef.current.stop();
      setIsListening(false);
    } else {
      setRecognitionError(null);
      recognitionRef.current.start();
      setIsListening(true);
    }
  };

  const handleRunTask = useCallback(async (goal: string) => {
    if (!goal || status !== AgentStatus.IDLE) return;

    setStatus(AgentStatus.THINKING);

    addThought({
      id: generateId(`SYNAPSE_INIT: ${goal}`),
      timestamp: Date.now(),
      content: `SYNAPSE_INIT: Orchestrating autonomous workflow for goal: "${goal}"`,
      type: 'plan',
    });

    // Route through the backend Inference Engine via WebSocket
    if (webSocketService.getConnectionStatus()) {
      addThought({
        id: generateId(`ROUTING: ${goal}`),
        timestamp: Date.now(),
        content: `Dispatching to KNIRV Inference Engine: "${goal}"`,
        type: 'observation',
      });
      setStatus(AgentStatus.EXECUTING);
      // Response arrives as neural_task_result / neural_task_error event
      webSocketService.send({ type: 'neural_task', payload: { goal } });
    } else {
      addThought({
        id: generateId(`OFFLINE: ${goal}`),
        timestamp: Date.now(),
        content: `Inference engine offline — WebSocket not connected. Please wait for reconnection.`,
        type: 'error',
      });
      setStatus(AgentStatus.IDLE);
    }
  }, [status, addThought]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (isListening) {
      recognitionRef.current?.stop();
      setIsListening(false);
    }
    if (input.trim() && status === AgentStatus.IDLE) {
      handleRunTask(input);
      setInput('');
    }
  };

  const getTypeIcon = (type: Thought['type']) => {
    switch (type) {
      case 'plan': return <Compass size={14} className="text-blue-400" />;
      case 'observation': return <Search size={14} className="text-amber-400" />;
      case 'conclusion': return <CheckCircle size={14} className="text-emerald-400" />;
      case 'error': return <AlertCircle size={14} className="text-rose-400" />;
    }
  };

  const getTypeStyles = (type: Thought['type']) => {
    switch (type) {
      case 'plan': return 'border-blue-500/20 bg-blue-500/5';
      case 'observation': return 'border-amber-500/20 bg-amber-500/5';
      case 'conclusion': return 'border-emerald-500/20 bg-emerald-500/5';
      case 'error': return 'border-rose-500/20 bg-rose-500/5';
    }
  };

  const isBusy = status !== AgentStatus.IDLE;

  return (
    <div className={`space-y-4 ${className}`}>
      <Card className="knirv-card-gradient border-indigo-500/30">
        <CardHeader className="pb-2">
          <CardTitle className="flex items-center justify-between text-sm">
            <div className="flex items-center space-x-2">
              <Terminal className="w-4 h-4 text-indigo-400" />
              <span>Neural Desktop</span>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-[10px] font-mono text-gray-600">ID: PERSISTENT_LOG</span>
            </div>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 lg:grid-cols-7 gap-4">
            {/* Neural Stream - Left Side */}
            <div className="lg:col-span-4 space-y-4">
              {/* Thought Process */}
              <div className="rounded-2xl border border-gray-800 bg-gray-950/40 overflow-hidden">
                <div className="px-4 py-3 border-b border-gray-800 flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <div className="p-1 bg-indigo-500/10 rounded-lg text-indigo-400">
                      <Terminal size={14} />
                    </div>
                    <span className="text-xs font-bold text-gray-300 uppercase tracking-widest">Neural Stream</span>
                  </div>
                </div>
                <div 
                  ref={containerRef}
                  className="h-[280px] overflow-y-auto p-4 space-y-3"
                >
                  {thoughts.length === 0 && (
                    <div className="h-full flex flex-col items-center justify-center text-gray-700 space-y-3">
                      <div className="relative">
                        <div className="absolute inset-0 bg-indigo-500/20 blur-2xl rounded-full" />
                        <Loader2 className="animate-spin text-indigo-500 relative" size={28} />
                      </div>
                      <div className="text-center">
                        <p className="text-[10px] font-bold uppercase tracking-[0.3em] opacity-40">Aether Standby</p>
                        <p className="text-[9px] opacity-30 mt-1">Awaiting Heuristic Trigger</p>
                      </div>
                    </div>
                  )}
                  
                  {thoughts.map((thought, idx) => (
                    <div 
                      key={thought.id}
                      className={`group relative p-3 rounded-xl border transition-all ${getTypeStyles(thought.type)}`}
                    >
                      <div className="flex items-start gap-3">
                        <div className="mt-0.5 p-1 rounded-lg bg-gray-950/50 border border-white/5">
                          {getTypeIcon(thought.type)}
                        </div>
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center justify-between mb-1">
                            <div className="flex items-center gap-2">
                              <span className="text-[9px] uppercase font-black tracking-widest text-white/40">
                                Phase::{thought.type}
                              </span>
                              {idx === 0 && <span className="flex h-1.5 w-1.5 rounded-full bg-indigo-500 animate-pulse" />}
                            </div>
                            <span className="text-[9px] font-mono opacity-30">
                              {new Date(thought.timestamp).toLocaleTimeString([], { hour12: false, fractionalSecondDigits: 3 } as any)}
                            </span>
                          </div>
                          <p className="text-xs text-gray-300 leading-relaxed whitespace-pre-wrap">
                            {thought.content}
                          </p>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              {/* Agent Control - Input */}
              <div className="rounded-2xl border border-gray-800 bg-gray-950/40 p-4">
                {!isKeyActive ? (
                  <div className="flex flex-col items-center justify-center gap-3 py-4">
                    <div className="flex items-center gap-2 text-rose-400 font-black uppercase text-xs tracking-widest">
                      <ShieldAlert size={14} /> Link Offline
                    </div>
                    <button 
                      className="px-6 py-2 bg-indigo-600 hover:bg-indigo-500 text-white rounded-lg font-black uppercase text-[10px] tracking-[0.2em] shadow-lg transition-all"
                    >
                      Initialize Secure API Link
                    </button>
                    <p className="text-[9px] text-gray-600">A valid paid-tier Google Cloud project key is required for autonomous heuristic reasoning.</p>
                  </div>
                ) : (
                  <div>
                    <div className="flex items-center justify-between mb-3">
                      <h3 className="text-[10px] font-black uppercase tracking-[0.2em] text-gray-500 flex items-center gap-2">
                        <div className={`w-1.5 h-1.5 rounded-full transition-all ${isListening ? 'bg-rose-500 shadow-[0_0_12px_rgba(244,63,94,1)] animate-pulse' : 'bg-indigo-500 shadow-[0_0_8px_rgba(99,102,241,1)]'}`} />
                        Mission Parameters
                      </h3>
                      <div className="text-[9px] font-mono text-gray-600 flex items-center gap-1 italic">
                        {isListening ? 'VOICE_LINK_ACTIVE' : 'READY_FOR_SYNTHESIS'} <ChevronRight size={8} />
                      </div>
                    </div>

                    <form onSubmit={handleSubmit} className="relative">
                      <input
                        type="text"
                        value={input}
                        onChange={(e) => setInput(e.target.value)}
                        placeholder={isBusy ? "KIMI_REASONING_IN_PROGRESS..." : isListening ? "Listening to dictation..." : "Define autonomous objective..."}
                        disabled={isBusy}
                        className={`w-full bg-gray-900 border border-gray-800 rounded-xl py-3 pl-10 pr-24 text-xs font-medium focus:outline-none transition-all placeholder:text-gray-700 placeholder:font-mono text-gray-200 ${
                          isListening 
                            ? 'border-rose-500/40 ring-1 ring-rose-500/10' 
                            : 'focus:border-indigo-500/50 focus:ring-1 focus:ring-indigo-500/10'
                        }`}
                      />
                      
                      {recognitionError && (
                        <div className="mt-2 text-[10px] text-rose-400 font-mono bg-rose-500/10 px-2 py-1 rounded-lg">
                          ERROR: {recognitionError.toUpperCase()}
                        </div>
                      )}

                      <button
                        type="button"
                        onClick={toggleListening}
                        disabled={isBusy}
                        className={`absolute left-2 top-1/2 -translate-y-1/2 w-7 h-7 flex items-center justify-center rounded-lg transition-all ${
                          isListening 
                          ? 'bg-rose-600 text-white' 
                          : 'bg-gray-800 text-gray-500 hover:text-indigo-400'
                        } ${isBusy ? 'opacity-20 cursor-not-allowed' : ''}`}
                      >
                        {isListening ? <MicOff size={14} /> : <Mic size={14} />}
                      </button>

                      <button
                        type="submit"
                        disabled={isBusy || !input.trim()}
                        className={`absolute right-1.5 top-1/2 -translate-y-1/2 w-8 h-8 flex items-center justify-center rounded-lg transition-all ${
                          isBusy 
                          ? 'bg-gray-800 text-gray-600 cursor-not-allowed' 
                          : input.trim() 
                            ? 'bg-indigo-600 text-white hover:bg-indigo-500' 
                            : 'bg-gray-800 text-gray-600'
                        }`}
                      >
                        {isBusy ? (
                          <div className="flex gap-0.5 items-center">
                            <span className="w-0.5 h-1 bg-indigo-400 rounded-full animate-bounce [animation-delay:-0.3s]" />
                            <span className="w-0.5 h-1 bg-indigo-400 rounded-full animate-bounce [animation-delay:-0.15s]" />
                            <span className="w-0.5 h-1 bg-indigo-400 rounded-full animate-bounce" />
                          </div>
                        ) : (
                          <Send size={14} className={input.trim() ? 'animate-in fade-in zoom-in duration-300' : ''} />
                        )}
                      </button>
                    </form>
                    
                    <div className="mt-3 flex flex-wrap gap-1.5">
                      {['Market Intel', 'Deep Research', 'Code Synthesis', 'Logic Audit'].map((tag) => (
                        <button
                          key={tag}
                          disabled={isBusy}
                          onClick={() => setInput(prev => `${prev} ${tag}`.trim())}
                          className="text-[8px] font-black uppercase tracking-widest text-gray-500 hover:text-indigo-300 transition-all border border-gray-800/50 hover:border-indigo-500/30 px-2 py-1 rounded bg-gray-900/30 hover:bg-indigo-500/5"
                        >
                          {tag}
                        </button>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            </div>

            {/* Current Processing - Right Side */}
            <div className="lg:col-span-3 space-y-4">
              {/* Current Processing */}
              <div className="rounded-2xl border border-gray-800 bg-gray-950/40 overflow-hidden">
                <div className="px-4 py-3 border-b border-gray-800 flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <div className="p-1 bg-blue-500/10 rounded-lg text-blue-400 animate-pulse">
                      <Activity size={14} />
                    </div>
                    <span className="text-xs font-bold text-gray-300 uppercase tracking-widest">Current Processing</span>
                  </div>
                  <Badge variant="outline" className="text-[8px] bg-blue-500/10 text-blue-400 border-blue-500/30">
                    Live
                  </Badge>
                </div>
                <div className="h-[200px] overflow-y-auto p-3 space-y-2">
                  <CurrentProcessingBox />
                </div>
              </div>

              {/* Session Indices */}
              <div className="rounded-2xl border border-gray-800 bg-gray-950/40 p-4">
                <h3 className="text-[9px] font-black text-gray-400 uppercase tracking-widest mb-3 flex items-center gap-2">
                  <Shield size={10} className="text-indigo-400" />
                  Session Indices
                </h3>
                <div className="grid grid-cols-2 gap-2">
                  <div className="bg-gray-900/40 p-3 rounded-xl border border-gray-800/50">
                    <p className="text-[8px] text-gray-600 uppercase font-black tracking-widest mb-1">Neural Traces</p>
                    <p className="text-lg font-mono font-bold text-indigo-400 tracking-tighter">{thoughts.length}</p>
                  </div>
                  <div className="bg-gray-900/40 p-3 rounded-xl border border-gray-800/50">
                    <p className="text-[8px] text-gray-600 uppercase font-black tracking-widest mb-1">Key Status</p>
                    <p className={`text-lg font-mono font-bold tracking-tighter ${isKeyActive ? 'text-emerald-400' : 'text-rose-500'}`}>
                      {isKeyActive ? 'ACTIVE' : 'OFFLINE'}
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
};

interface ProcessingActivity {
  id: string;
  timestamp: string;
  type: string;
  title: string;
  description: string;
  status: 'active' | 'completed';
}

const activityIcon = (type: string): React.ReactNode => {
  switch (type) {
    case 'learning_cycle':   return <GitBranch className="w-3 h-3" />;
    case 'metrics_collection': return <Database className="w-3 h-3" />;
    case 'pattern_analysis': return <Network className="w-3 h-3" />;
    case 'guardrail_check':  return <Shield className="w-3 h-3" />;
    case 'increase_priority': return <Activity className="w-3 h-3" />;
    case 'optimize_resources': return <Server className="w-3 h-3" />;
    case 'redistribute_load': return <Users className="w-3 h-3" />;
    default:                 return <RefreshCw className="w-3 h-3" />;
  }
};

const CurrentProcessingBox: React.FC = () => {
  const [activities, setActivities] = useState<ProcessingActivity[]>([]);
  const [isConnected, setIsConnected] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleConnection = (data: { connected: boolean }) => {
      setIsConnected(data.connected);
    };

    const handleActivity = (payload: any) => {
      if (Array.isArray(payload?.activities)) {
        setActivities(payload.activities);
      }
    };

    webSocketService.on('connection', handleConnection);
    webSocketService.on('cognitive-engine-activity', handleActivity);
    setIsConnected(webSocketService.getConnectionStatus());
    webSocketService.subscribe(['connection', 'cognitive-engine-activity']);

    // Fallback: show demo activities if no real data arrives after 3 seconds
    const fallbackTimeout = setTimeout(() => {
      if (activities.length === 0 && isConnected) {
        setActivities([
          {
            id: 'demo-learning',
            timestamp: new Date().toISOString(),
            type: 'learning_cycle',
            title: 'Learning State',
            description: 'Tasks: 0 | Success Rate: 0% | Learning: 0% | Confidence: 0%',
            status: 'active'
          },
          {
            id: 'demo-telemetry',
            timestamp: new Date().toISOString(),
            type: 'metrics_collection',
            title: 'System Telemetry',
            description: 'CPU: 0% | Memory: 0% | Goroutines: 0 | Heap: 0MB',
            status: 'active'
          }
        ]);
      }
    }, 3000);

    return () => {
      webSocketService.off('connection', handleConnection);
      webSocketService.off('cognitive-engine-activity', handleActivity);
      clearTimeout(fallbackTimeout);
    };
  }, [isConnected]);

  const active = activities.filter(a => a.status === 'active');
  const completed = activities.filter(a => a.status === 'completed');

  if (activities.length === 0) {
    return (
      <div className="h-full flex flex-col items-center justify-center text-gray-700 space-y-2">
        <Loader2 className="animate-spin text-blue-500/40" size={20} />
        <p className="text-[9px] uppercase tracking-widest opacity-40">Awaiting engine activity</p>
      </div>
    );
  }

  return (
    <div ref={scrollRef} className="space-y-2">
      {activities.map((activity) => (
        <div
          key={activity.id}
          className={`flex items-start gap-2 p-2 rounded-lg transition-all ${
            activity.status === 'active' ? 'bg-blue-500/10 border border-blue-500/20' : 'bg-gray-900/30'
          }`}
        >
          <div className={`p-1 rounded-md shrink-0 ${
            activity.status === 'active'
              ? 'bg-blue-500/20 text-blue-400 animate-pulse'
              : 'bg-green-500/20 text-green-400'
          }`}>
            {activityIcon(activity.type)}
          </div>
          <div className="flex-1 min-w-0">
            <div className="flex items-center justify-between">
              <span className={`text-[10px] font-medium ${activity.status === 'active' ? 'text-blue-400' : 'text-green-400'}`}>
                {activity.title}
              </span>
              {activity.status === 'active'
                ? <Badge variant="outline" className="text-[8px] h-4 bg-blue-500/20 text-blue-400 border-blue-500/30">ACTIVE</Badge>
                : <CheckCircle className="w-3 h-3 text-green-400" />
              }
            </div>
            <p className="text-[9px] text-gray-500 truncate">{activity.description}</p>
          </div>
          <ChevronRight className={`w-3 h-3 ${activity.status === 'active' ? 'text-blue-400' : 'text-green-400'} opacity-50 shrink-0`} />
        </div>
      ))}
      <div className="text-[9px] text-gray-600 text-center pt-2">
        {active.length} active • {completed.length} completed
      </div>
    </div>
  );
};

export default NeuralDesktopPanel;
