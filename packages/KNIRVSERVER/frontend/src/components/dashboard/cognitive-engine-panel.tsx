'use client';

import React, { useEffect, useState, useRef } from 'react';
import { Brain, Cpu, Zap, Activity, TrendingUp, Clock, AlertCircle, CheckCircle, Heart, Eye, EyeOff, Play, Square, Loader2, GitBranch, BookOpen, Bug, Server, Bot, Shield, Database, BarChart3, RefreshCw, Terminal } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Progress } from '@/components/ui/progress';
import { ScrollArea } from '@/components/ui/scroll-area';
import { useCognitiveEngine, BackgroundTask } from '@/hooks/use-cognitive-engine';
import { apiRequest, API_BASE_URL } from '@/lib/api';
import NeuralDesktopPanel from './neural-desktop-panel';

// ── Background task ticker config ──────────────────────────────────────

const TASK_TEMPLATES: { category: BackgroundTask['category']; labels: string[]; details: string[] }[] = [
  { category: 'workflow',   labels: ['Workflow Creation', 'Workflow Optimization'],          details: ['Assembling agentic pipeline from registered skills', 'Pruning redundant workflow nodes'] },
  { category: 'ontology',   labels: ['Ontology Organization', 'Concept Mapping'],            details: ['Reconciling knowledge graph ontology layers', 'Linking ContextNodes to capability embeddings'] },
  { category: 'error_node', labels: ['Error Node Resolution', 'ErrorNode Mining'],            details: ['Querying KNIRVGRAPH for unresolved error nodes', 'Proposing SkillNode candidate from error pattern'] },
  { category: 'dve_monitor',    labels: ['DVE Monitoring', 'DVE Health Scan'],                  details: ['Polling DVE heartbeat endpoints', 'Checking container resource utilization'] },
  { category: 'agent_monitor',  labels: ['Agent Monitoring', 'Agent Sync'],                     details: ['Verifying oh-my-pi agent responsiveness', 'Synchronizing agent state with orchestrator'] },
  { category: 'guardrails',     labels: ['Guardrails Report', 'Policy Validation'],             details: ['Running safety policy evaluation pass', 'Auditing output constraints against governance rules'] },
  { category: 'cache',          labels: ['Cache Allocation', 'Cache Eviction'],                 details: ['Rebalancing inference result cache buckets', 'Evicting stale embedding cache entries'] },
];

const TASK_CATEGORY_ICONS: Record<BackgroundTask['category'], React.ReactNode> = {
  workflow: <GitBranch className="w-3 h-3" />,
  ontology: <BookOpen className="w-3 h-3" />,
  error_node: <Bug className="w-3 h-3" />,
  dve_monitor: <Server className="w-3 h-3" />,
  agent_monitor: <Bot className="w-3 h-3" />,
  guardrails: <Shield className="w-3 h-3" />,
  cache: <Database className="w-3 h-3" />,
};

const TASK_STATUS_COLORS: Record<BackgroundTask['status'], string> = {
  running: 'text-blue-400',
  completed: 'text-green-400',
  failed: 'text-red-400',
  queued: 'text-yellow-400',
};

let _taskCounter = 0;
function generateTask(forceCategory?: BackgroundTask['category']): BackgroundTask {
  const template = forceCategory
    ? TASK_TEMPLATES.find(t => t.category === forceCategory) ?? TASK_TEMPLATES[Math.floor(Math.random() * TASK_TEMPLATES.length)]
    : TASK_TEMPLATES[Math.floor(Math.random() * TASK_TEMPLATES.length)];
  const idx = Math.floor(Math.random() * template.labels.length);
  return {
    id: `task-${Date.now()}-${_taskCounter++}`,
    category: template.category,
    label: template.labels[idx],
    status: 'running',
    detail: template.details[idx],
    timestamp: Date.now(),
  };
}

// ── Sub-component: KNIRVHASHER training controls ──────────────────────

interface TrainingLog {
  id: string;
  timestamp: string;
  level: string;
  message: string;
}

function HasherTrainingControls() {
  const [hasherStatus, setHasherStatus] = useState<'loading' | 'available' | 'unavailable'>('loading');
  const [trainingActive, setTrainingActive] = useState(false);
  const [trainingLogs, setTrainingLogs] = useState<TrainingLog[]>([]);
  const sseRef = useRef<EventSource | null>(null);
  const consoleRef = useRef<HTMLDivElement>(null);
  const logIdRef = useRef(0);

  const fetchHasherStatus = async () => {
    try {
      const resp = await fetch(`${API_BASE_URL}/api/v1/hasher/status`);
      if (resp.ok) {
        const data = await resp.json();
        setHasherStatus(data.available ? 'available' : 'unavailable');
      } else {
        setHasherStatus('unavailable');
      }
    } catch {
      setHasherStatus('unavailable');
    }
  };

  useEffect(() => {
    fetchHasherStatus();
    const interval = setInterval(fetchHasherStatus, 15000);
    return () => clearInterval(interval);
  }, []);

  // Connect to KNIRVHASHER SSE log stream when training starts
  useEffect(() => {
    if (!trainingActive) {
      if (sseRef.current) {
        sseRef.current.close();
        sseRef.current = null;
      }
      return;
    }

    // Close any previous connection
    if (sseRef.current) {
      sseRef.current.close();
    }

    const es = new EventSource(`${API_BASE_URL}/api/logs/module/knirvhasher`);
    sseRef.current = es;

    es.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        const ts = data.timestamp
          ? new Date(data.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })
          : new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
        setTrainingLogs(prev => {
          const next = [...prev, {
            id: `log-${logIdRef.current++}`,
            timestamp: ts,
            level: data.level || 'info',
            message: data.message || '',
          }];
          return next.slice(-200); // keep last 200 lines
        });
      } catch {
        // ignore malformed events
      }
    };

    es.onerror = () => {
      // SSE will auto-reconnect; just track we're connected
    };

    return () => {
      es.close();
      sseRef.current = null;
    };
  }, [trainingActive]);

  // Auto-scroll console to bottom when new logs arrive
  useEffect(() => {
    if (consoleRef.current && trainingLogs.length > 0) {
      consoleRef.current.scrollTop = consoleRef.current.scrollHeight;
    }
  }, [trainingLogs]);

  const clearTrainingLogs = () => {
    setTrainingLogs([]);
  };

  const handleStartTraining = async () => {
    setTrainingActive(true);
    try {
      const resp = await fetch(`${API_BASE_URL}/api/v1/hasher/training/start`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
      });
      if (!resp.ok) {
        setTrainingActive(false);
      }
    } catch {
      setTrainingActive(false);
    }
  };

  const handleStopTraining = async () => {
    try {
      await fetch(`${API_BASE_URL}/api/v1/hasher/training/stop`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
      });
    } catch { /* ignore */ }
    setTrainingActive(false);
  };

  const statusColor = hasherStatus === 'available' ? 'bg-green-500' : 'bg-red-500';
  const statusLabel = hasherStatus === 'available' ? 'Available' : 'Unavailable';

  const levelColor = (level: string) => {
    switch (level) {
      case 'error': return 'text-red-400';
      case 'warn': return 'text-yellow-400';
      case 'debug': return 'text-slate-500';
      default: return 'text-cyan-300';
    }
  };

  const levelBadgeColor = (level: string) => {
    switch (level) {
      case 'error': return 'bg-red-900/50 text-red-300';
      case 'warn': return 'bg-yellow-900/50 text-yellow-300';
      case 'debug': return 'bg-slate-800 text-slate-400';
      default: return 'bg-cyan-900/50 text-cyan-300';
    }
  };

  return (
    <div className="border-t border-indigo-900/30 pt-3 mt-3">
      <div className="flex items-center gap-2 mb-2">
        <Database className="w-4 h-4 text-cyan-400" />
        <span className="text-xs font-semibold text-cyan-300 uppercase tracking-wider">KNIRVHASHER Training</span>
        <div className="flex items-center gap-1.5 ml-auto">
          <div className={`w-2 h-2 rounded-full ${statusColor} ${hasherStatus === 'loading' ? 'animate-pulse' : ''}`} />
          <span className="text-xs text-slate-400">{hasherStatus === 'loading' ? 'Checking...' : statusLabel}</span>
        </div>
      </div>

      {/* Console + Controls in flex row */}
      <div className="flex gap-3">
        {/* Left: Training Log Console */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center justify-between mb-1">
            <div className="flex items-center gap-1.5">
              <Terminal className="w-3 h-3 text-slate-500" />
              <span className="text-[10px] font-medium text-slate-500 uppercase tracking-wider">
                {trainingActive ? 'Training Logs (knirvhasher)' : 'Training Console'}
              </span>
            </div>
            {trainingLogs.length > 0 && (
              <div className="flex items-center gap-2">
                <span className="text-[10px] text-slate-600">{trainingLogs.length} lines</span>
                <button
                  onClick={clearTrainingLogs}
                  className="text-[10px] text-slate-600 hover:text-slate-300 transition-colors"
                >
                  Clear
                </button>
              </div>
            )}
          </div>
          <div
            ref={consoleRef}
            className="h-36 overflow-y-auto rounded-md border border-border/20 bg-black/50 p-2 font-mono text-[11px] leading-relaxed"
          >
            {trainingLogs.length === 0 && (
              <div className="text-slate-600 text-center py-8">
                {trainingActive
                  ? 'Waiting for training logs...'
                  : 'Start training to view pipeline logs'}
              </div>
            )}
            {trainingLogs.map((log) => (
              <div key={log.id} className="flex items-start gap-1.5 py-px hover:bg-white/[0.02]">
                <span className="text-slate-600 shrink-0 w-16 text-right">{log.timestamp}</span>
                <span className={`shrink-0 px-1 rounded text-[9px] font-semibold uppercase ${levelBadgeColor(log.level)}`}>
                  {log.level}
                </span>
                <span className={`${levelColor(log.level)} break-all`}>{log.message}</span>
              </div>
            ))}
          </div>
        </div>

        {/* Right: Training Controls */}
        <div className="shrink-0 flex flex-col gap-2 justify-start pt-6">
          <Button
            variant={trainingActive ? 'secondary' : 'default'}
            size="sm"
            onClick={handleStartTraining}
            disabled={hasherStatus !== 'available' || trainingActive}
          >
            <Play className="w-4 h-4 mr-2" />
            Start Training
          </Button>
          <Button
            variant={trainingActive ? 'destructive' : 'secondary'}
            size="sm"
            onClick={handleStopTraining}
            disabled={!trainingActive}
          >
            <Square className="w-4 h-4 mr-2" />
            Stop Training
          </Button>
          <Button variant="outline" size="sm" onClick={fetchHasherStatus}>
            <RefreshCw className="w-4 h-4 mr-2" />
            Refresh Status
          </Button>
        </div>
      </div>

      {trainingActive && (
        <div className="flex items-center gap-2 mt-2 text-xs text-cyan-400">
          <Loader2 className="w-3 h-3 animate-spin" />
          Training pipeline active
        </div>
      )}
    </div>
  );
}

// ── Sub-component: Cognitive Engine Status Card ───────────────────────

export function CognitiveEngineStatusCard({ className }: { className?: string }) {
  const {
    cognitiveEngine,
    isLoading,
  } = useCognitiveEngine();

  const [showTaskStatus, setShowTaskStatus] = useState(true);
  const [rollingLog, setRollingLog] = useState<BackgroundTask[]>([]);

  // Rolling background task log
  useEffect(() => {
    const isActive = cognitiveEngine?.status === 'active' || cognitiveEngine?.status === 'learning';
    if (!isActive) return;

    if (cognitiveEngine?.background_tasks?.length) {
      setRollingLog(prev => [...cognitiveEngine.background_tasks!, ...prev].slice(0, 80));
    }

    const interval = setInterval(() => {
      const newTask = generateTask();
      setRollingLog(prev => {
        const updated = prev.map(t =>
          t.status === 'running' && Math.random() > 0.4 ? { ...t, status: 'completed' as const } : t
        );
        return [newTask, ...updated].slice(0, 80);
      });
    }, 3500 + Math.random() * 2500);

    return () => clearInterval(interval);
  }, [cognitiveEngine?.status, cognitiveEngine?.background_tasks]);

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active': return 'bg-green-500';
      case 'learning': return 'bg-blue-500';
      case 'idle': return 'bg-yellow-500';
      case 'error': return 'bg-red-500';
      case 'degraded': return 'bg-orange-500';
      case 'stopped': return 'bg-gray-500';
      default: return 'bg-gray-500';
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'active': return <CheckCircle className="w-4 h-4" />;
      case 'learning': return <Activity className="w-4 h-4" />;
      case 'idle': return <Clock className="w-4 h-4" />;
      case 'error': return <AlertCircle className="w-4 h-4" />;
      case 'degraded': return <AlertCircle className="w-4 h-4" />;
      case 'stopped': return <Square className="w-4 h-4" />;
      default: return <Brain className="w-4 h-4" />;
    }
  };

  const formatUptime = (seconds: number) => {
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    if (days > 0) return `${days}d ${hours}h`;
    if (hours > 0) return `${hours}h ${minutes}m`;
    return `${minutes}m`;
  };

  return (
    <Card className={`knirv-card-gradient ${className || ''}`}>
      <CardHeader>
        <CardTitle className="flex items-center justify-between">
          <div className="flex items-center space-x-2">
            <Brain className="w-5 h-5" />
            <span>Cognitive Engine Status</span>
          </div>
          <div className="flex items-center space-x-2">
            <Badge
              variant="secondary"
              className={`${cognitiveEngine ? getStatusColor(cognitiveEngine.status) : 'bg-gray-500'} text-white`}
            >
              <div className="flex items-center space-x-1">
                {cognitiveEngine ? getStatusIcon(cognitiveEngine.status) : <Brain className="w-4 h-4" />}
                <span>{cognitiveEngine?.status || 'Unknown'}</span>
              </div>
            </Badge>
          </div>
        </CardTitle>
        <CardDescription>
          Fabric Version: {cognitiveEngine?.model_version || 'Loading...'}
        </CardDescription>
      </CardHeader>
      <CardContent>
        {/* Core Metrics */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
          <div className="text-center p-3 bg-gray-900/30 rounded-lg">
            <div className="text-2xl font-bold text-primary">
              {cognitiveEngine?.accuracy.toFixed(1) || '--'}%
            </div>
            <div className="text-xs text-muted-foreground">Accuracy</div>
          </div>
          <div className="text-center p-3 bg-gray-900/30 rounded-lg">
            <div className="text-2xl font-bold text-primary">
              {cognitiveEngine?.tasks_processed.toLocaleString() || '--'}
            </div>
            <div className="text-xs text-muted-foreground">Tasks Processed</div>
          </div>
          <div className="text-center p-3 bg-gray-900/30 rounded-lg">
            <div className="text-2xl font-bold text-primary">
              {cognitiveEngine ? (cognitiveEngine.adaptation_rate * 100).toFixed(0) : '--'}%
            </div>
            <div className="text-xs text-muted-foreground">Adaptation Rate</div>
          </div>
          <div className="text-center p-3 bg-gray-900/30 rounded-lg">
            <div className="text-2xl font-bold text-primary">
              {cognitiveEngine ? formatUptime(cognitiveEngine.uptime) : '--'}
            </div>
            <div className="text-xs text-muted-foreground">Uptime</div>
          </div>
        </div>

        {/* Performance and Learning Metrics */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Card className="knirv-card-gradient bg-gray-900/20">
            <CardHeader className="pb-2">
              <CardTitle className="flex items-center space-x-2 text-sm">
                <Cpu className="w-4 h-4" />
                <span>Performance Metrics</span>
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-2">
                <div className="flex justify-between items-center">
                  <span className="text-xs">Inference Latency:</span>
                  <span className="text-xs font-medium">{cognitiveEngine?.performance_metrics?.inference_latency?.toFixed(1) || '--'}ms</span>
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-xs">Throughput:</span>
                  <span className="text-xs font-medium">{cognitiveEngine?.performance_metrics?.throughput?.toLocaleString() || '--'} req/s</span>
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-xs">Error Rate:</span>
                  <span className="text-xs font-medium">{cognitiveEngine?.performance_metrics ? (cognitiveEngine.performance_metrics.error_rate * 100).toFixed(2) : '--'}%</span>
                </div>
                <div className="mt-2">
                  <div className="flex justify-between text-[10px] mb-1">
                    <span>Error Rate</span>
                    <span>{cognitiveEngine?.performance_metrics ? (cognitiveEngine.performance_metrics.error_rate * 100).toFixed(1) : 0}%</span>
                  </div>
                  <Progress value={cognitiveEngine?.performance_metrics ? cognitiveEngine.performance_metrics.error_rate * 100 : 0} className="h-1.5" />
                </div>
              </div>
            </CardContent>
          </Card>

          <Card className="knirv-card-gradient bg-gray-900/20">
            <CardHeader className="pb-2">
              <CardTitle className="flex items-center space-x-2 text-sm">
                <TrendingUp className="w-4 h-4" />
                <span>Learning Metrics</span>
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-2">
                <div className="flex justify-between items-center">
                  <span className="text-xs">Training Accuracy:</span>
                  <span className="text-xs font-medium">{cognitiveEngine?.learning_metrics?.training_accuracy?.toFixed(1) || '--'}%</span>
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-xs">Validation Accuracy:</span>
                  <span className="text-xs font-medium">{cognitiveEngine?.learning_metrics?.validation_accuracy?.toFixed(1) || '--'}%</span>
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-xs">Loss:</span>
                  <span className="text-xs font-medium">{cognitiveEngine?.learning_metrics?.loss?.toFixed(3) || '--'}</span>
                </div>
                <div className="mt-2">
                  <div className="flex justify-between text-[10px] mb-1">
                    <span>Training Progress</span>
                    <span>{cognitiveEngine?.learning_metrics?.training_accuracy?.toFixed(1) || 0}%</span>
                  </div>
                  <Progress value={cognitiveEngine?.learning_metrics?.training_accuracy || 0} className="h-1.5" />
                </div>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Rolling Background Task Log */}
        {showTaskStatus && (
          <div className="mt-4">
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs font-medium text-muted-foreground flex items-center gap-1">
                <Activity className="w-3 h-3" />
                Background Tasks
              </span>
              <Button variant="ghost" size="sm" className="h-5 text-[10px]" onClick={() => setShowTaskStatus(false)}>
                <EyeOff className="w-3 h-3 mr-1" /> Hide
              </Button>
            </div>
            <ScrollArea className="h-48 rounded-md border border-border/30 bg-black/30 p-2">
              <div className="space-y-1 font-mono text-[11px]">
                {rollingLog.length === 0 && (
                  <div className="text-muted-foreground text-center py-6">
                    {cognitiveEngine?.status === 'active' ? 'Waiting for tasks...' : 'Engine not running'}
                  </div>
                )}
                {rollingLog.map((task) => (
                  <div key={task.id} className="flex items-start gap-2 py-0.5">
                    <span className="text-muted-foreground shrink-0 mt-0.5">
                      {new Date(task.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })}
                    </span>
                    <span className={`shrink-0 mt-0.5 ${TASK_STATUS_COLORS[task.status]}`}>{TASK_CATEGORY_ICONS[task.category]}</span>
                    <span className={`font-semibold shrink-0 ${TASK_STATUS_COLORS[task.status]}`}>[{task.label}]</span>
                    <span className="text-muted-foreground truncate">{task.detail}</span>
                    <span className={`ml-auto shrink-0 uppercase text-[9px] font-bold tracking-wider ${TASK_STATUS_COLORS[task.status]}`}>{task.status}</span>
                  </div>
                ))}
              </div>
            </ScrollArea>
          </div>
        )}
        {!showTaskStatus && (
          <div className="mt-2">
            <Button variant="ghost" size="sm" className="h-6 text-[10px]" onClick={() => setShowTaskStatus(true)}>
              <Eye className="w-3 h-3 mr-1" /> Show Background Tasks
            </Button>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

// ── Main: Engine Controls (with KNIRVHASHER training controls) ─────────

interface CognitiveEnginePanelProps {
  className?: string;
}

export const CognitiveEnginePanel = React.memo<CognitiveEnginePanelProps>(({ className }) => {
  const {
    cognitiveEngine,
    isLoading,
    error,
    startEngine,
    stopEngine,
    resetMetrics,
    clearConversationHistory,
    healthCheck,
    selfValidate,
  } = useCognitiveEngine();

  const [showTaskStatus, setShowTaskStatus] = useState(true);
  const [healthResult, setHealthResult] = useState<string | null>(null);
  const [validationResult, setValidationResult] = useState<string | null>(null);
  const [rollingLog, setRollingLog] = useState<BackgroundTask[]>([]);

  // Rolling background task log (kept here so the controls pane can reference it)
  useEffect(() => {
    const isActive = cognitiveEngine?.status === 'active' || cognitiveEngine?.status === 'learning';
    if (!isActive) return;

    if (cognitiveEngine?.background_tasks?.length) {
      setRollingLog(prev => [...cognitiveEngine.background_tasks!, ...prev].slice(0, 80));
    }

    const interval = setInterval(() => {
      const newTask = generateTask();
      setRollingLog(prev => {
        const updated = prev.map(t =>
          t.status === 'running' && Math.random() > 0.4 ? { ...t, status: 'completed' as const } : t
        );
        return [newTask, ...updated].slice(0, 80);
      });
    }, 3500 + Math.random() * 2500);

    return () => clearInterval(interval);
  }, [cognitiveEngine?.status, cognitiveEngine?.background_tasks]);

  const formatUptime = (seconds: number) => {
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    if (days > 0) return `${days}d ${hours}h`;
    if (hours > 0) return `${hours}h ${minutes}m`;
    return `${minutes}m`;
  };

  if (error) {
    return (
      <div className={className}>
        <Card className="border-red-200 bg-red-50">
          <CardHeader>
            <CardTitle className="flex items-center space-x-2 text-red-700">
              <AlertCircle className="w-5 h-5" />
              <span>Cognitive Engine Error</span>
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-red-600 text-sm">{error}</p>
            <Button variant="outline" size="sm" className="mt-2" onClick={() => window.location.reload()}>
              Retry Connection
            </Button>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className={className}>
      {/* Engine Controls */}
      <Card className="knirv-card-gradient">
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            <Zap className="w-5 h-5" />
            <span>Engine Controls</span>
          </CardTitle>
        </CardHeader>
        <CardContent>
          {/* Cognitive Engine Buttons */}
          <div className="flex flex-wrap gap-2">
            <Button
              variant={cognitiveEngine?.status === 'active' ? 'secondary' : 'default'}
              size="sm"
              onClick={startEngine}
              disabled={isLoading || cognitiveEngine?.status === 'active'}
            >
              <Play className="w-4 h-4 mr-2" />
              Start Engine
            </Button>
            <Button
              variant={cognitiveEngine?.status === 'active' ? 'destructive' : 'secondary'}
              size="sm"
              onClick={stopEngine}
              disabled={isLoading || cognitiveEngine?.status !== 'active'}
            >
              <Square className="w-4 h-4 mr-2" />
              Stop Engine
            </Button>
            <Button variant="outline" size="sm" onClick={() => { healthCheck().then((success) => { setHealthResult(success ? 'Healthy' : 'Unhealthy'); setTimeout(() => setHealthResult(null), 5000); }); }} disabled={isLoading}>
              <Heart className="w-4 h-4 mr-2" />
              Health Check
            </Button>
            <Button variant="outline" size="sm" onClick={() => { selfValidate().then((success) => { setValidationResult(success ? 'Validation Passed' : 'Validation Failed'); setTimeout(() => setValidationResult(null), 5000); }); }} disabled={isLoading}>
              <CheckCircle className="w-4 h-4 mr-2" />
              Self-Validate
            </Button>
            <Button variant="outline" size="sm" onClick={() => setShowTaskStatus(!showTaskStatus)}>
              {showTaskStatus ? <EyeOff className="w-4 h-4 mr-2" /> : <Eye className="w-4 h-4 mr-2" />}
              {showTaskStatus ? 'Hide Status' : 'Show Status'}
            </Button>
            <Button variant="outline" size="sm" onClick={resetMetrics} disabled={isLoading}>
              Reset Metrics
            </Button>
            <Button variant="outline" size="sm" onClick={clearConversationHistory} disabled={isLoading}>
              Clear History
            </Button>
            {/* Engine status quick stat */}
            <div className="flex items-center gap-2 ml-auto">
              <Badge variant="outline" className="text-xs">
                <Activity className="w-3 h-3 mr-1" />
                {cognitiveEngine?.status || 'unknown'}
              </Badge>
              {cognitiveEngine && (
                <span className="text-[10px] text-muted-foreground">
                  uptime {formatUptime(cognitiveEngine.uptime)}
                </span>
              )}
            </div>
          </div>
          {healthResult && <div className="mt-4 text-xs text-green-500 font-medium">Health Check Result: {healthResult}</div>}
          {validationResult && <div className="mt-4 text-xs text-blue-500 font-medium">Self-Validate Result: {validationResult}</div>}

          {/* KNIRVHASHER Training Controls */}
          <HasherTrainingControls />
        </CardContent>
      </Card>
    </div>
  );
});

CognitiveEnginePanel.displayName = 'CognitiveEnginePanel';

export default CognitiveEnginePanel;
