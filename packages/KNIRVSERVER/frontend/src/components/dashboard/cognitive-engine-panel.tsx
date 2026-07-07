'use client';

import React, { useEffect, useState, useRef } from 'react';
import { Brain, Cpu, Zap, Activity, TrendingUp, Clock, AlertCircle, CheckCircle, Heart, Eye, EyeOff, Play, Square, Loader2, GitBranch, BookOpen, Bug, Server, Bot, Shield, Database, BarChart3, RefreshCw, Terminal } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Progress } from '@/components/ui/progress';
import { ScrollArea } from '@/components/ui/scroll-area';
import { useCognitiveEngine, useQualityMode, BackgroundTask } from '@/hooks/use-cognitive-engine';
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

interface TrainingRunStatus {
  run_id?: string;
  status?: string;
  message?: string;
  records_loaded?: number;
  checkpoint_tokens?: number;
  average_fitness?: number;
  current_batch?: number;
  batches_done?: number;
  total_batches?: number;
  last_error?: string;
  errors?: string[];
  updated_at?: string;
}

interface HasherStatusResponse {
  available?: boolean;
  running?: boolean;
  status?: string;
  training_active?: boolean;
  training?: TrainingRunStatus;
  data?: HasherStatusResponse;
}

const MAX_TRAINING_LOGS = 200;
const MIN_LOG_HEIGHT = 144;
const MAX_LOG_HEIGHT = 640;

const hasherTrainingViewState: {
  active: boolean;
  logs: TrainingLog[];
  nextLogId: number;
  trainingStatus: TrainingRunStatus | null;
  logHeight: number;
} = {
  active: false,
  logs: [],
  nextLogId: 0,
  trainingStatus: null,
  logHeight: MIN_LOG_HEIGHT,
};

function normalizeHasherStatus(raw: HasherStatusResponse) {
  const status = raw.data ?? raw;
  const statusText = typeof status.status === 'string' ? status.status.toLowerCase() : '';
  const trainingStatusText = typeof status.training?.status === 'string' ? status.training.status.toLowerCase() : '';
  const inferredRunning = statusText === 'running' || trainingStatusText === 'initializing' || trainingStatusText === 'running';
  const running = Boolean(status.running || status.training_active || inferredRunning);
  const available = Boolean(status.available ?? (running || statusText === 'available' || statusText === 'running'));
  const training = status.training ?? null;

  return { available, running, training };
}

function formatTrainingLogTimestamp(timestamp?: string) {
  return timestamp
    ? new Date(timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })
    : new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

export function HasherTrainingControls() {
  const [hasherStatus, setHasherStatus] = useState<'loading' | 'available' | 'unavailable'>('loading');
  const [trainingActive, setTrainingActiveState] = useState(hasherTrainingViewState.active);
  const [trainingLogs, setTrainingLogsState] = useState<TrainingLog[]>(hasherTrainingViewState.logs);
  const [trainingStatus, setTrainingStatusState] = useState<TrainingRunStatus | null>(hasherTrainingViewState.trainingStatus);
  const [logHeight, setLogHeightState] = useState(hasherTrainingViewState.logHeight);
  const sseRef = useRef<EventSource | null>(null);
  const consoleRef = useRef<HTMLDivElement>(null);
  const logIdRef = useRef(hasherTrainingViewState.nextLogId);
  const lastStartTimeRef = useRef<number>(0);

  const setTrainingActive = (active: boolean) => {
    hasherTrainingViewState.active = active;
    setTrainingActiveState(active);
  };

  const setTrainingStatus = (status: TrainingRunStatus | null) => {
    hasherTrainingViewState.trainingStatus = status;
    setTrainingStatusState(status);
  };

  const setLogHeight = (height: number) => {
    const clamped = Math.max(MIN_LOG_HEIGHT, Math.min(MAX_LOG_HEIGHT, Math.round(height)));
    hasherTrainingViewState.logHeight = clamped;
    setLogHeightState(clamped);
  };

  const setTrainingLogs = (updater: TrainingLog[] | ((previous: TrainingLog[]) => TrainingLog[])) => {
    setTrainingLogsState(previous => {
      const next = typeof updater === 'function' ? updater(previous) : updater;
      const capped = next.slice(-MAX_TRAINING_LOGS);
      hasherTrainingViewState.logs = capped;
      hasherTrainingViewState.nextLogId = logIdRef.current;
      return capped;
    });
  };

  const mergeTrainingLogEntries = (entries: { timestamp?: string; level?: string; message?: string }[]) => {
    if (entries.length === 0) {
      return;
    }

    setTrainingLogs(previous => {
      const seen = new Set(previous.map(log => `${log.timestamp}|${log.level}|${log.message}`));
      const merged = [...previous];
      for (const entry of entries) {
        const timestamp = formatTrainingLogTimestamp(entry.timestamp);
        const level = entry.level || 'info';
        const message = entry.message || '';
        const key = `${timestamp}|${level}|${message}`;
        if (!seen.has(key)) {
          seen.add(key);
          merged.push({
            id: `log-${logIdRef.current++}`,
            timestamp,
            level,
            message,
          });
        }
      }
      return merged.slice(-MAX_TRAINING_LOGS);
    });
  };

  const fetchTrainingLogHistory = async () => {
    try {
      const resp = await fetch(`${API_BASE_URL}/api/logs/history?module=knirvhasher&limit=100`);
      if (!resp.ok) {
        return;
      }
      const data = await resp.json();
      const logs: { timestamp?: string; level?: string; message?: string }[] = (data.logs || []).slice().reverse();
      mergeTrainingLogEntries(logs);
    } catch {
      // history polling is best-effort; live SSE or the next poll may recover
    }
  };

  const fetchHasherStatus = async () => {
    try {
      const resp = await fetch(`${API_BASE_URL}/api/v1/hasher/status`);
      if (resp.ok) {
        const data: HasherStatusResponse = await resp.json();
        const status = normalizeHasherStatus(data);
        setHasherStatus(status.available ? 'available' : 'unavailable');

        const recentlyStarted = Date.now() - lastStartTimeRef.current < 20000;
        if (status.running) {
          setTrainingActive(true);
          lastStartTimeRef.current = 0;
        } else if (!recentlyStarted) {
          setTrainingActive(false);
        }

        setTrainingStatus(status.training);
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

  const logStreamingEnabled = trainingActive || hasherStatus === 'available';

  // Connect to KNIRVHASHER SSE log stream when training starts
  useEffect(() => {
    if (!logStreamingEnabled) {
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

    // 1. Pre-fill from history to get recent knirvhasher logs
    fetchTrainingLogHistory();
    const historyInterval = setInterval(fetchTrainingLogHistory, 2000);

    // 2. Open SSE stream for live updates
    const es = new EventSource(`${API_BASE_URL}/api/logs/module/knirvhasher`);
    sseRef.current = es;

    es.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        mergeTrainingLogEntries([data]);
      } catch {
        // ignore malformed events
      }
    };

    es.onerror = () => {
      // SSE will auto-reconnect
    };

    return () => {
      clearInterval(historyInterval);
      es.close();
      sseRef.current = null;
    };
  }, [logStreamingEnabled]);

  // Auto-scroll console to bottom when new logs arrive
  useEffect(() => {
    if (consoleRef.current && trainingLogs.length > 0) {
      consoleRef.current.scrollTop = consoleRef.current.scrollHeight;
    }
  }, [trainingLogs]);

  const clearTrainingLogs = () => {
    setTrainingLogs([]);
  };

  const handleResizeStart = (event: React.MouseEvent<HTMLButtonElement>) => {
    event.preventDefault();
    const startY = event.clientY;
    const startHeight = logHeight;

    const handleMove = (moveEvent: MouseEvent) => {
      setLogHeight(startHeight + moveEvent.clientY - startY);
    };

    const handleUp = () => {
      document.removeEventListener('mousemove', handleMove);
      document.removeEventListener('mouseup', handleUp);
    };

    document.addEventListener('mousemove', handleMove);
    document.addEventListener('mouseup', handleUp);
  };

  const handleStartTraining = async () => {
    lastStartTimeRef.current = Date.now();
    try {
      const resp = await fetch(`${API_BASE_URL}/api/v1/hasher/training/start`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
      });
      if (!resp.ok) {
        setTrainingActive(false);
        lastStartTimeRef.current = 0;
      }
    } catch {
      setTrainingActive(false);
      lastStartTimeRef.current = 0;
    }
  };

  const handleStopTraining = async () => {
    lastStartTimeRef.current = 0;
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
  const latestPipelineErrors = trainingStatus?.errors?.length
    ? trainingStatus.errors
    : trainingStatus?.last_error
      ? [trainingStatus.last_error]
      : [];
  const batchLabel = trainingStatus?.total_batches
    ? `${trainingStatus.current_batch || 0}/${trainingStatus.total_batches}`
    : trainingStatus?.current_batch
      ? `${trainingStatus.current_batch}`
      : '--';
  const batchesDoneLabel = trainingStatus?.total_batches
    ? `${trainingStatus.batches_done || 0}/${trainingStatus.total_batches}`
    : trainingStatus?.batches_done !== undefined
      ? `${trainingStatus.batches_done}`
      : '--';

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
      <div className="flex flex-col gap-3 lg:flex-row">
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
          <div className="rounded-md">
            <div
              ref={consoleRef}
              style={{ height: logHeight }}
              className="overflow-y-auto rounded-t-md border border-border/20 border-b-0 bg-black/50 p-2 font-mono text-[11px] leading-relaxed"
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
            <button
              type="button"
              aria-label="Resize training log"
              onMouseDown={handleResizeStart}
              className="flex h-4 w-full cursor-ns-resize items-center justify-center rounded-b-md border border-border/20 bg-black/40 hover:bg-slate-900/70"
            >
              <span className="h-0.5 w-12 rounded bg-slate-600" />
            </button>
          </div>
        </div>

        {/* Right: Training Controls */}
        <div className="flex shrink-0 flex-row flex-wrap gap-2 justify-start lg:flex-col lg:pt-6">
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

      {(trainingActive || trainingStatus) && (
        <div className="mt-2 rounded-md border border-cyan-900/30 bg-cyan-950/10 p-2">
          <div className="flex items-center gap-2 text-xs text-cyan-400">
            {trainingActive && <Loader2 className="w-3 h-3 animate-spin" />}
            <span className="font-medium">Training pipeline {trainingActive ? 'active' : 'status'}</span>
            {trainingStatus?.status && (
              <Badge variant="outline" className="ml-auto h-5 border-cyan-900/40 text-[10px] text-cyan-300">
                {trainingStatus.status}
              </Badge>
            )}
          </div>
          <div className="mt-2 grid grid-cols-2 gap-2 text-[11px] md:grid-cols-4">
            <div>
              <div className="text-slate-500">Current Batch</div>
              <div className="font-mono text-slate-200">{batchLabel}</div>
            </div>
            <div>
              <div className="text-slate-500">Batches Done</div>
              <div className="font-mono text-slate-200">{batchesDoneLabel}</div>
            </div>
            <div>
              <div className="text-slate-500">Records</div>
              <div className="font-mono text-slate-200">{trainingStatus?.records_loaded?.toLocaleString() ?? '--'}</div>
            </div>
            <div>
              <div className="text-slate-500">Pipeline Errors</div>
              <div className={latestPipelineErrors.length > 0 ? 'font-mono text-red-300' : 'font-mono text-slate-200'}>
                {latestPipelineErrors.length}
              </div>
            </div>
          </div>
          {trainingStatus?.message && (
            <div className="mt-2 truncate font-mono text-[11px] text-slate-400">{trainingStatus.message}</div>
          )}
          {latestPipelineErrors.length > 0 && (
            <div className="mt-2 space-y-1">
              {latestPipelineErrors.slice(-3).map((error, index) => (
                <div key={`${error}-${index}`} className="break-words font-mono text-[11px] text-red-300">
                  {error}
                </div>
              ))}
            </div>
          )}
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
    isActionPending,
  } = useCognitiveEngine();

  const [showTaskStatus, setShowTaskStatus] = useState(true);
  const [healthResult, setHealthResult] = useState<string | null>(null);
  const [validationResult, setValidationResult] = useState<string | null>(null);
  const [rollingLog, setRollingLog] = useState<BackgroundTask[]>([]);
  const { qualityMode, setQualityMode } = useQualityMode();

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
          {/* Quality Mode Selection */}
          <div className="flex items-center gap-4 mb-4">
            <span className="text-xs text-[#8080c0] uppercase tracking-wider">Quality:</span>
            <div className="flex gap-1 bg-[#0d1117] rounded-lg p-0.5 border border-[#1e2235]">
              <button
                onClick={() => setQualityMode('standard')}
                className={`px-3 py-1.5 text-xs rounded-md transition-all ${
                  qualityMode === 'standard'
                    ? 'bg-[#7c4dff] text-white shadow-sm'
                    : 'text-[#8080c0] hover:text-white'
                }`}
              >
                Standard
              </button>
              <button
                onClick={() => setQualityMode('deep')}
                className={`px-3 py-1.5 text-xs rounded-md transition-all ${
                  qualityMode === 'deep'
                    ? 'bg-[#7c4dff] text-white shadow-sm'
                    : 'text-[#8080c0] hover:text-white'
                }`}
              >
                Deep MoA
              </button>
            </div>
          </div>
          {/* Cognitive Engine Buttons */}
          <div className="flex flex-wrap gap-2">
            <Button
              variant={cognitiveEngine?.status === 'active' ? 'secondary' : 'default'}
              size="sm"
              onClick={startEngine}
              disabled={isLoading || isActionPending || cognitiveEngine?.status === 'active'}
            >
              <Play className="w-4 h-4 mr-2" />
              Start Engine
            </Button>
            <Button
              variant={cognitiveEngine?.status === 'active' ? 'destructive' : 'secondary'}
              size="sm"
              onClick={stopEngine}
              disabled={isLoading || isActionPending || cognitiveEngine?.status !== 'active'}
            >
              <Square className="w-4 h-4 mr-2" />
              Stop Engine
            </Button>
            <Button variant="outline" size="sm" onClick={() => { healthCheck().then((success) => { setHealthResult(success ? 'Healthy' : 'Unhealthy'); setTimeout(() => setHealthResult(null), 5000); }); }} disabled={isLoading || isActionPending}>
              <Heart className="w-4 h-4 mr-2" />
              Health Check
            </Button>
            <Button variant="outline" size="sm" onClick={() => { selfValidate().then((success) => { setValidationResult(success ? 'Validation Passed' : 'Validation Failed'); setTimeout(() => setValidationResult(null), 5000); }); }} disabled={isLoading || isActionPending}>
              <CheckCircle className="w-4 h-4 mr-2" />
              Self-Validate
            </Button>
            <Button variant="outline" size="sm" onClick={() => setShowTaskStatus(!showTaskStatus)} disabled={isActionPending}>
              {showTaskStatus ? <EyeOff className="w-4 h-4 mr-2" /> : <Eye className="w-4 h-4 mr-2" />}
              {showTaskStatus ? 'Hide Status' : 'Show Status'}
            </Button>
            <Button variant="outline" size="sm" onClick={resetMetrics} disabled={isLoading || isActionPending}>
              Reset Metrics
            </Button>
            <Button variant="outline" size="sm" onClick={clearConversationHistory} disabled={isLoading || isActionPending}>
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
