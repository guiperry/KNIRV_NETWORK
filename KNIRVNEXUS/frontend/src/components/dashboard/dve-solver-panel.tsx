'use client';

import React, { useState, useEffect } from 'react';
import { Play, Upload, Lock, Terminal, Search, FileCode, CheckCircle, XCircle, X, ShieldCheck, Cpu, Database, Activity, AlertTriangle, Clock, RefreshCw, Send, Eye, EyeOff } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Progress } from '@/components/ui/progress';
import { Badge } from '@/components/ui/badge';
import { useToast } from '@/hooks/use-toast';
import { useDemoMode } from '@/contexts/demo-mode-context';

interface DVETask {
  id: string;
  name: string;
  type: string;
  status: 'completed' | 'failed' | 'running' | 'pending';
  timestamp: string;
  duration?: number;
  workerId?: string;
  errorDetails?: string;
  trace?: string[];
  documentsTouched?: string[];
  userMetadata?: Record<string, string>;
  policiesEnforced?: string[];
}

interface DVESolverPanelProps {
  isOpen: boolean;
  onClose: () => void;
  isMonitorOpen?: boolean;
  initialTask?: DVETask | null;
  onTaskSelect?: (task: DVETask) => void;
}

const DVESolverPanel: React.FC<DVESolverPanelProps> = ({ isOpen, onClose, isMonitorOpen, initialTask, onTaskSelect }) => {
  const [tasks, setTasks] = useState<DVETask[]>([]);
  const [selectedTask, setSelectedTask] = useState<DVETask | null>(null);
  const [isReplaying, setIsReplaying] = useState(false);
  const [replayStep, setReplayStep] = useState(0);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [showTrace, setShowTrace] = useState(true);
  const { isDemoMode } = useDemoMode();
  const { toast } = useToast();

  useEffect(() => {
    if (isOpen) {
      loadTasks();
    }
  }, [isOpen, isDemoMode]);

  useEffect(() => {
    if (initialTask) {
      setSelectedTask(initialTask);
    }
  }, [initialTask]);

  const loadTasks = async () => {
    if (isDemoMode) {
      setTasks(getDemoTasks());
    } else {
      try {
        const response = await fetch('/api/dve/tasks');
        if (response.ok) {
          const data = await response.json();
          setTasks(data.tasks || []);
        } else {
          setTasks([]);
        }
      } catch (error) {
        console.error('Failed to load tasks:', error);
        setTasks([]);
      }
    }
  };

  const getDemoTasks = (): DVETask[] => [
    { id: 'TASK-001', name: 'Data Validation Flow', type: 'validation', status: 'completed', timestamp: '2025-10-21T10:45:00Z', duration: 2340, workerId: 'WORKER-001', trace: ['[10:45:00] Task initialized', '[10:45:01] Loading validation rules', '[10:45:03] Running constraint checks', '[10:45:10] Validating data integrity', '[10:45:12] Applying policy rules', '[10:45:13] Task completed successfully'], policiesEnforced: ['data_integrity', 'privacy'] },
    { id: 'TASK-002', name: 'Image Processing Pipeline', type: 'computation', status: 'failed', timestamp: '2025-10-21T10:44:00Z', duration: 890, workerId: 'WORKER-006', errorDetails: 'Memory allocation failed: out of memory', trace: ['[10:44:00] Task initialized', '[10:44:01] Loading image processor', '[10:44:05] Allocating memory buffer', '[ERROR] Failed to allocate 2GB buffer', '[ERROR] Out of memory exception thrown', '[FATAL] Task terminated abnormally'], documentsTouched: ['image_001.jpg', 'config.json'], userMetadata: { user_id: 'user_4821', session: 'sess_001', request_id: 'req_abc123' }, policiesEnforced: ['resource_limits', 'memory_policy'] },
    { id: 'TASK-003', name: 'API Request Handler', type: 'workflow', status: 'completed', timestamp: '2025-10-21T10:43:00Z', duration: 156, workerId: 'WORKER-004', trace: ['[10:43:00] Task initialized', '[10:43:01] Parsing request', '[10:43:02] Routing to handler', '[10:43:03] Executing business logic', '[10:43:04] Formatting response', '[10:43:05] Task completed'], policiesEnforced: ['rate_limit', 'auth', 'input_validation'] },
    { id: 'TASK-004', name: 'Model Inference Task', type: 'inference', status: 'running', timestamp: '2025-10-21T10:42:30Z', workerId: 'WORKER-001', trace: ['[10:42:30] Task initialized', '[10:42:31] Loading model weights', '[10:42:35] Tokenizing input', '[10:42:40] Running inference...', '[IN PROGRESS] Generating tokens'], policiesEnforced: ['model_safety', 'output_filtering'] },
    { id: 'TASK-005', name: 'Database Sync', type: 'sync', status: 'completed', timestamp: '2025-10-21T10:41:00Z', duration: 450, workerId: 'WORKER-008', trace: ['[10:41:00] Task initialized', '[10:41:01] Connecting to database', '[10:41:03] Syncing schema', '[10:41:10] Verifying consistency', '[10:41:12] Sync complete'], policiesEnforced: ['data_consistency'] },
    { id: 'TASK-006', name: 'User Authentication', type: 'auth', status: 'failed', timestamp: '2025-10-21T10:40:00Z', duration: 45, workerId: 'WORKER-007', errorDetails: 'Invalid token signature: cryptographic verification failed', trace: ['[10:40:00] Task initialized', '[10:40:01] Extracting token', '[10:40:02] Validating token structure', '[10:40:03] Verifying signature', '[ERROR] Signature mismatch detected', '[FATAL] Authentication failed'], userMetadata: { attempt: '3', ip: '192.168.1.1', user_agent: 'Mozilla/5.0' }, policiesEnforced: ['authentication', 'token_validation'] },
    { id: 'TASK-007', name: 'Cache Invalidation', type: 'maintenance', status: 'completed', timestamp: '2025-10-21T10:39:00Z', duration: 89, workerId: 'WORKER-004', trace: ['[10:39:00] Task initialized', '[10:39:01] Scanning cache keys', '[10:39:03] Identifying stale entries', '[10:39:05] Invalidating 47 keys', '[10:39:06] Task completed'], policiesEnforced: [] },
    { id: 'TASK-008', name: 'Report Generation', type: 'computation', status: 'pending', timestamp: '2025-10-21T10:38:00Z', workerId: 'WORKER-002', trace: ['[PENDING] Awaiting execution', '[10:38:00] Task initialized', '[10:38:01] Gathering report data...'], policiesEnforced: ['data_access'] },
  ];

  const handleTaskSelect = (task: DVETask) => {
    setSelectedTask(task);
    setReplayStep(0);
    setIsReplaying(false);
    if (onTaskSelect) {
      onTaskSelect(task);
    }
  };

  const handleReplay = () => {
    if (!selectedTask?.trace) return;
    
    setIsReplaying(true);
    setReplayStep(0);
    
    const interval = setInterval(() => {
      setReplayStep(prev => {
        if (prev >= selectedTask.trace!.length - 1) {
          clearInterval(interval);
          setIsReplaying(false);
          return prev;
        }
        return prev + 1;
      });
    }, 800);
  };

  const handleStopReplay = () => {
    setIsReplaying(false);
    setReplayStep(0);
  };

  const handleSubmitToResolution = async () => {
    if (!selectedTask || selectedTask.status !== 'failed') return;
    
    setIsSubmitting(true);
    
    const errorNode = {
      taskId: selectedTask.id,
      taskName: selectedTask.name,
      taskType: selectedTask.type,
      errorDetails: selectedTask.errorDetails,
      timestamp: selectedTask.timestamp,
      duration: selectedTask.duration,
      workerId: selectedTask.workerId,
      trace: selectedTask.trace,
      documentsTouched: selectedTask.documentsTouched,
      userMetadata: selectedTask.userMetadata,
      policiesEnforced: selectedTask.policiesEnforced,
      submittedAt: new Date().toISOString(),
    };
    
    try {
      if (isDemoMode) {
        await new Promise(resolve => setTimeout(resolve, 1500));
        toast({
          title: "Error Node Created",
          description: `Task ${selectedTask.id} bundled and sent to internal resolution queue.`,
        });
      } else {
        const response = await fetch('/api/resolution-queue', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(errorNode),
        });
        
        if (!response.ok) throw new Error('Failed to submit');
        
        toast({
          title: "Error Node Created",
          description: `Task ${selectedTask.id} bundled and sent to internal resolution queue.`,
        });
      }
      
      const response = await fetch('/api/knirvgraph/error-node', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          ...errorNode,
          resolutionStatus: 'pending_internal',
        }),
      });
      
    } catch (error) {
      console.error('Failed to submit to resolution:', error);
      toast({
        title: "Submission Failed",
        description: "Failed to submit error node. Please try again.",
        variant: "destructive",
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'completed': return <CheckCircle className="w-4 h-4 text-green-400" />;
      case 'failed': return <XCircle className="w-4 h-4 text-red-400" />;
      case 'running': return <Activity className="w-4 h-4 text-blue-400 animate-pulse" />;
      case 'pending': return <Clock className="w-4 h-4 text-slate-400" />;
      default: return null;
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'completed': return 'bg-green-500/20 text-green-400 border-green-500/30';
      case 'failed': return 'bg-red-500/20 text-red-400 border-red-500/30';
      case 'running': return 'bg-blue-500/20 text-blue-400 border-blue-500/30';
      case 'pending': return 'bg-slate-500/20 text-slate-400 border-slate-500/30';
      default: return 'bg-slate-500/20 text-slate-400 border-slate-500/30';
    }
  };

  const formatDuration = (ms?: number) => {
    if (!ms) return '-';
    if (ms < 1000) return `${ms}ms`;
    if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
    return `${Math.floor(ms / 60000)}m ${((ms % 60000) / 1000).toFixed(0)}s`;
  };

  const formatTime = (timestamp: string) => {
    const date = new Date(timestamp);
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
  };

  const failedTasks = tasks.filter(t => t.status === 'failed');

  if (!isOpen) return null;

  return (
    <div
      className="absolute z-[100] transition-all duration-500 transform ease-in-out bg-slate-950 border border-blue-600/50 shadow-[0_0_60px_rgba(0,0,0,0.9)] overflow-hidden rounded-2xl flex flex-col"
      style={{
        left: '50%',
        top: isMonitorOpen ? '42%' : '50%',
        transform: 'translate(-50%, -50%)',
        width: '950px',
        maxHeight: isMonitorOpen ? '65vh' : '80vh',
      }}
    >
      <div className="bg-slate-900 border-b border-blue-600/30 p-4 flex items-center justify-between">
        <div className="flex items-center space-x-4">
          <div className="p-2 bg-blue-600/20 rounded-lg">
            <ShieldCheck className="w-5 h-5 text-blue-400" />
          </div>
          <div>
            <h2 className="text-lg font-black text-blue-100 uppercase tracking-tighter">Task Tracer & Replay Simulator</h2>
            <div className="flex items-center space-x-2 text-[10px] font-mono text-slate-500">
              <span>Task History & Resolution</span>
              <span className="text-blue-500/50">•</span>
              <span className="text-green-500/80">{failedTasks.length} Failed Tasks</span>
            </div>
          </div>
        </div>
        <button
          onClick={onClose}
          className="text-slate-500 hover:text-white hover:bg-slate-800 p-2 rounded-lg transition-all"
        >
          <X className="w-5 h-5" />
        </button>
      </div>

      <div className="flex-1 overflow-hidden flex custom-scrollbar">
        <div className="w-72 border-r border-slate-800 flex flex-col">
          <div className="p-3 border-b border-slate-800">
            <div className="relative">
              <Search className="w-4 h-4 absolute left-3 top-2.5 text-slate-600" />
              <input
                type="text"
                placeholder="Filter tasks..."
                className="w-full bg-slate-950 border border-slate-800 rounded-lg pl-10 pr-4 py-2 text-xs text-slate-300 focus:outline-none focus:border-blue-500 transition-colors"
              />
            </div>
          </div>
          
          <div className="flex-1 overflow-y-auto p-2 space-y-1">
            <div className="text-[9px] font-bold text-slate-500 uppercase px-2 mb-2">All Tasks ({tasks.length})</div>
            {tasks.map(task => (
              <div
                key={task.id}
                onClick={() => handleTaskSelect(task)}
                className={`p-2 rounded-lg cursor-pointer transition-all border ${
                  selectedTask?.id === task.id
                    ? 'bg-blue-600/20 border-blue-500'
                    : 'bg-slate-900/50 border-slate-800 hover:border-slate-700'
                }`}
              >
                <div className="flex items-center justify-between mb-1">
                  <div className="flex items-center space-x-2">
                    {getStatusIcon(task.status)}
                    <span className="font-mono text-[9px] text-blue-400 font-bold">{task.id}</span>
                  </div>
                  <span className={`text-[8px] font-black uppercase px-1.5 py-0.5 rounded border ${getStatusColor(task.status)}`}>
                    {task.status}
                  </span>
                </div>
                <div className="text-[10px] text-slate-200 line-clamp-1">{task.name}</div>
                <div className="text-[9px] text-slate-500 mt-1 flex justify-between">
                  <span className="capitalize">{task.type}</span>
                  <span>{formatDuration(task.duration)}</span>
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="flex-1 flex flex-col overflow-hidden">
          {selectedTask ? (
            <>
              <div className="p-4 border-b border-slate-800">
                <div className="flex items-center justify-between mb-3">
                  <div>
                    <div className="flex items-center space-x-2 mb-1">
                      {getStatusIcon(selectedTask.status)}
                      <span className="text-sm font-bold text-slate-200">{selectedTask.name}</span>
                    </div>
                    <div className="text-[10px] text-slate-500">
                      {selectedTask.id} • {selectedTask.type} • {formatTime(selectedTask.timestamp)} • {formatDuration(selectedTask.duration)}
                    </div>
                  </div>
                  <div className="flex space-x-2">
                    {selectedTask.status === 'failed' && (
                      <Button
                        onClick={handleSubmitToResolution}
                        disabled={isSubmitting}
                        size="sm"
                        className="bg-red-600 hover:bg-red-500 text-white text-[10px] font-black uppercase"
                      >
                        {isSubmitting ? (
                          <RefreshCw className="w-3 h-3 mr-1 animate-spin" />
                        ) : (
                          <Send className="w-3 h-3 mr-1" />
                        )}
                        Submit to Resolution
                      </Button>
                    )}
                    {selectedTask.trace && (
                      <Button
                        onClick={isReplaying ? handleStopReplay : handleReplay}
                        size="sm"
                        variant="outline"
                        className="text-[10px] font-black uppercase border-blue-500/30 text-blue-400"
                      >
                        {isReplaying ? (
                          <>
                            <EyeOff className="w-3 h-3 mr-1" />
                            Stop Replay
                          </>
                        ) : (
                          <>
                            <Play className="w-3 h-3 mr-1" />
                            Replay Trace
                          </>
                        )}
                      </Button>
                    )}
                  </div>
                </div>
                
                {selectedTask.status === 'failed' && selectedTask.errorDetails && (
                  <div className="bg-red-950/30 border border-red-500/30 rounded-lg p-3 mb-3">
                    <div className="flex items-center text-red-400 text-[10px] font-bold uppercase mb-1">
                      <AlertTriangle className="w-3 h-3 mr-1" />
                      Error Details
                    </div>
                    <div className="text-[11px] text-red-300 font-mono">{selectedTask.errorDetails}</div>
                  </div>
                )}
              </div>

              <div className="flex-1 overflow-y-auto p-4 custom-scrollbar">
                {selectedTask.trace && (
                  <div className="mb-4">
                    <div className="flex items-center justify-between mb-2">
                      <h3 className="text-[10px] font-bold uppercase tracking-widest text-slate-400 flex items-center">
                        <Terminal className="w-3 h-3 mr-2 text-green-500" />
                        Process Trace
                      </h3>
                      <Badge variant="outline" className="text-[9px]">
                        {selectedTask.trace.length} steps
                      </Badge>
                    </div>
                    <div className="bg-black rounded-lg border border-slate-800 p-3 h-48 overflow-y-auto font-mono text-[11px] custom-scrollbar">
                      {selectedTask.trace.map((line, i) => {
                        const isError = line.startsWith('[ERROR]') || line.startsWith('[FATAL]');
                        const isHighlighted = isReplaying && i <= replayStep;
                        const isCurrent = isReplaying && i === replayStep;
                        
                        return (
                          <div 
                            key={i} 
                            className={`mb-1 transition-all ${
                              isError 
                                ? 'text-red-400' 
                                : isCurrent 
                                  ? 'text-blue-400 bg-blue-900/30 px-1 rounded' 
                                  : isHighlighted 
                                    ? 'text-green-400' 
                                    : 'text-slate-500'
                            }`}
                          >
                            {line}
                          </div>
                        );
                      })}
                    </div>
                    {isReplaying && (
                      <Progress 
                        value={(replayStep / (selectedTask.trace.length - 1)) * 100} 
                        className="h-1 mt-2 bg-slate-800" 
                      />
                    )}
                  </div>
                )}

                <div className="grid grid-cols-2 gap-4">
                  {selectedTask.documentsTouched && selectedTask.documentsTouched.length > 0 && (
                    <div>
                      <h3 className="text-[10px] font-bold uppercase tracking-widest text-slate-400 mb-2 flex items-center">
                        <FileCode className="w-3 h-3 mr-2 text-purple-500" />
                        Documents Touched
                      </h3>
                      <div className="bg-slate-900/50 border border-slate-800 rounded-lg p-2 space-y-1">
                        {selectedTask.documentsTouched.map((doc, i) => (
                          <div key={i} className="text-[10px] text-slate-300 font-mono">{doc}</div>
                        ))}
                      </div>
                    </div>
                  )}

                  {selectedTask.userMetadata && Object.keys(selectedTask.userMetadata).length > 0 && (
                    <div>
                      <h3 className="text-[10px] font-bold uppercase tracking-widest text-slate-400 mb-2 flex items-center">
                        <Database className="w-3 h-3 mr-2 text-amber-500" />
                        User Metadata
                      </h3>
                      <div className="bg-slate-900/50 border border-slate-800 rounded-lg p-2 space-y-1">
                        {Object.entries(selectedTask.userMetadata).map(([key, value]) => (
                          <div key={key} className="text-[10px] flex justify-between">
                            <span className="text-slate-500 font-bold uppercase">{key}:</span>
                            <span className="text-slate-300 font-mono">{value}</span>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}

                  {selectedTask.policiesEnforced && selectedTask.policiesEnforced.length > 0 && (
                    <div className="col-span-2">
                      <h3 className="text-[10px] font-bold uppercase tracking-widest text-slate-400 mb-2 flex items-center">
                        <ShieldCheck className="w-3 h-3 mr-2 text-blue-500" />
                        Policies Enforced
                      </h3>
                      <div className="flex flex-wrap gap-2">
                        {selectedTask.policiesEnforced.map((policy, i) => (
                          <Badge key={i} variant="outline" className="text-[9px] bg-blue-950/30 text-blue-400 border-blue-500/30">
                            {policy}
                          </Badge>
                        ))}
                      </div>
                    </div>
                  )}

                  {selectedTask.workerId && (
                    <div>
                      <h3 className="text-[10px] font-bold uppercase tracking-widest text-slate-400 mb-2 flex items-center">
                        <Cpu className="w-3 h-3 mr-2 text-green-500" />
                        Worker
                      </h3>
                      <div className="bg-slate-900/50 border border-slate-800 rounded-lg p-2">
                        <div className="text-[10px] text-slate-300 font-mono">{selectedTask.workerId}</div>
                      </div>
                    </div>
                  )}
                </div>
              </div>
            </>
          ) : (
            <div className="flex-1 flex items-center justify-center text-slate-500">
              <div className="text-center">
                <Activity className="w-12 h-12 text-slate-700 mx-auto mb-3" />
                <p className="text-sm">Select a task to view details</p>
              </div>
            </div>
          )}
        </div>
      </div>

      <div className="bg-slate-900 border-t border-blue-600/30 p-3 flex justify-between items-center px-4">
        <div className="flex items-center space-x-2">
          <div className="w-2 h-2 rounded-full bg-green-500 animate-pulse" />
          <span className="text-[10px] font-mono text-slate-500 uppercase">
            {isDemoMode ? 'Demo Mode' : 'Live Data'}
          </span>
        </div>
        <Button
          variant="ghost"
          onClick={onClose}
          className="text-slate-400 hover:text-white text-[10px]"
        >
          Close Interface
        </Button>
      </div>
    </div>
  );
};

export default DVESolverPanel;
