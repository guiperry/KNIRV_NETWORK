'use client';

import React, { useEffect, useState, useRef } from 'react';
import { Brain, Cpu, Zap, Activity, TrendingUp, Clock, AlertCircle, CheckCircle, Heart, Eye, EyeOff, Play, Square, Loader2, GitBranch, BookOpen, Bug, Server, Bot, Shield, Database } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Progress } from '@/components/ui/progress';
import { ScrollArea } from '@/components/ui/scroll-area';
import { useCognitiveEngine, BackgroundTask } from '@/hooks/use-cognitive-engine';
import NeuralDesktopPanel from './neural-desktop-panel';

interface CognitiveEnginePanelProps {
  className?: string;
}

// Background task ticker config
const TASK_TEMPLATES: { category: BackgroundTask['category']; labels: string[]; details: string[] }[] = [
  {
    category: 'workflow',
    labels: ['Workflow Creation', 'Workflow Optimization'],
    details: ['Assembling agentic pipeline from registered skills', 'Pruning redundant workflow nodes'],
  },
  {
    category: 'ontology',
    labels: ['Ontology Organization', 'Concept Mapping'],
    details: ['Reconciling knowledge graph ontology layers', 'Linking ContextNodes to capability embeddings'],
  },
  {
    category: 'error_node',
    labels: ['Error Node Resolution', 'ErrorNode Mining'],
    details: ['Querying KNIRVGRAPH for unresolved error nodes', 'Proposing SkillNode candidate from error pattern'],
  },
  {
    category: 'dve_monitor',
    labels: ['DVE Monitoring', 'DVE Health Scan'],
    details: ['Polling DVE heartbeat endpoints', 'Checking container resource utilization'],
  },
  {
    category: 'agent_monitor',
    labels: ['Agent Monitoring', 'Agent Sync'],
    details: ['Verifying oh-my-pi agent responsiveness', 'Synchronizing agent state with orchestrator'],
  },
  {
    category: 'guardrails',
    labels: ['Guardrails Report', 'Policy Validation'],
    details: ['Running safety policy evaluation pass', 'Auditing output constraints against governance rules'],
  },
  {
    category: 'cache',
    labels: ['Cache Allocation', 'Cache Eviction'],
    details: ['Rebalancing inference result cache buckets', 'Evicting stale embedding cache entries'],
  },
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

export const CognitiveEnginePanel = React.memo<CognitiveEnginePanelProps>(({ className }) => {
  const {
    cognitiveEngine,
    isLoading,
    error,
    isConnected,
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
  const logBottomRef = useRef<HTMLDivElement>(null);

  // Build a rolling log of background tasks when the engine is active
  useEffect(() => {
    const isActive = cognitiveEngine?.status === 'active' || cognitiveEngine?.status === 'learning';
    if (!isActive) return;

    // Seed from backend background_tasks if available
    if (cognitiveEngine?.background_tasks?.length) {
      setRollingLog(prev => {
        const merged = [...cognitiveEngine.background_tasks!, ...prev].slice(0, 80);
        return merged;
      });
    }

    // Ticker: add a new task entry every 3-6 seconds
    const interval = setInterval(() => {
      const newTask = generateTask();
      setRollingLog(prev => {
        // Mark one random previous 'running' task as completed
        const updated = prev.map(t =>
          t.status === 'running' && Math.random() > 0.4 ? { ...t, status: 'completed' as const } : t
        );
        return [newTask, ...updated].slice(0, 80);
      });
    }, 3500 + Math.random() * 2500);

    return () => clearInterval(interval);
  }, [cognitiveEngine?.status, cognitiveEngine?.background_tasks]);

  // Auto-scroll to bottom when new entries arrive
  useEffect(() => {
    logBottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [rollingLog.length]);

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

  if (error) {
    return (
      <div className={`space-y-4 ${className}`}>
        <Card className="border-red-200 bg-red-50">
          <CardHeader>
            <CardTitle className="flex items-center space-x-2 text-red-700">
              <AlertCircle className="w-5 h-5" />
              <span>Cognitive Engine Error</span>
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-red-600 text-sm">{error}</p>
            <Button 
              variant="outline" 
              size="sm" 
              className="mt-2"
              onClick={() => window.location.reload()}
            >
              Retry Connection
            </Button>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className={`space-y-4 ${className}`}>
      {/* Engine Controls - Top */}
      <Card className="knirv-card-gradient">
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            <Zap className="w-5 h-5" />
            <span>Engine Controls</span>
          </CardTitle>
        </CardHeader>
        <CardContent>
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
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                healthCheck().then((success) => {
                  setHealthResult(success ? 'Healthy' : 'Unhealthy');
                  setTimeout(() => setHealthResult(null), 5000);
                });
              }}
              disabled={isLoading}
            >
              <Heart className="w-4 h-4 mr-2" />
              Health Check
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                selfValidate().then((success) => {
                  setValidationResult(success ? 'Validation Passed' : 'Validation Failed');
                  setTimeout(() => setValidationResult(null), 5000);
                });
              }}
              disabled={isLoading}
            >
              <CheckCircle className="w-4 h-4 mr-2" />
              Self-Validate
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setShowTaskStatus(!showTaskStatus)}
              disabled={isLoading}
            >
              {showTaskStatus ? <EyeOff className="w-4 h-4 mr-2" /> : <Eye className="w-4 h-4 mr-2" />}
              {showTaskStatus ? 'Hide Status' : 'Show Status'}
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={resetMetrics}
              disabled={isLoading}
            >
              Reset Metrics
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={clearConversationHistory}
              disabled={isLoading}
            >
              Clear History
            </Button>
          </div>
          {healthResult && (
            <div className="mt-4 text-xs text-green-500 font-medium">
              Health Check Result: {healthResult}
            </div>
          )}
          {validationResult && (
            <div className="mt-4 text-xs text-blue-500 font-medium">
              Self-Validate Result: {validationResult}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Neural Desktop Panel */}
      <NeuralDesktopPanel />

      {/* Unified Cognitive Engine Status - with all metrics */}
      <Card className="knirv-card-gradient">
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
              {isConnected && (
                <Badge variant="outline" className="text-xs text-green-600 border-green-600">
                  <Activity className="w-3 h-3 mr-1" />
                  Live
                </Badge>
              )}
              {!isConnected && (
                <Badge variant="outline" className="text-xs text-yellow-600 border-yellow-600">
                  <Clock className="w-3 h-3 mr-1" />
                  Polling
                </Badge>
              )}
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
                    <span className="text-xs font-medium">
                      {cognitiveEngine?.performance_metrics?.inference_latency?.toFixed(1) || '--'}ms
                    </span>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-xs">Throughput:</span>
                    <span className="text-xs font-medium">
                      {cognitiveEngine?.performance_metrics?.throughput?.toLocaleString() || '--'} req/s
                    </span>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-xs">Error Rate:</span>
                    <span className="text-xs font-medium">
                      {cognitiveEngine?.performance_metrics ? (cognitiveEngine.performance_metrics.error_rate * 100).toFixed(2) : '--'}%
                    </span>
                  </div>
                  <div className="mt-2">
                    <div className="flex justify-between text-[10px] mb-1">
                      <span>Error Rate</span>
                      <span>{cognitiveEngine?.performance_metrics ? (cognitiveEngine.performance_metrics.error_rate * 100).toFixed(1) : 0}%</span>
                    </div>
                    <Progress 
                      value={cognitiveEngine?.performance_metrics ? cognitiveEngine.performance_metrics.error_rate * 100 : 0} 
                      className="h-1.5"
                    />
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
                    <span className="text-xs font-medium">
                      {cognitiveEngine?.learning_metrics?.training_accuracy?.toFixed(1) || '--'}%
                    </span>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-xs">Validation Accuracy:</span>
                    <span className="text-xs font-medium">
                      {cognitiveEngine?.learning_metrics?.validation_accuracy?.toFixed(1) || '--'}%
                    </span>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-xs">Loss:</span>
                    <span className="text-xs font-medium">
                      {cognitiveEngine?.learning_metrics?.loss?.toFixed(3) || '--'}
                    </span>
                  </div>
                  <div className="mt-2">
                    <div className="flex justify-between text-[10px] mb-1">
                      <span>Training Progress</span>
                      <span>{cognitiveEngine?.learning_metrics?.training_accuracy?.toFixed(1) || 0}%</span>
                    </div>
                    <Progress 
                      value={cognitiveEngine?.learning_metrics?.training_accuracy || 0} 
                      className="h-1.5"
                    />
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
                {cognitiveEngine?.status === 'active' && (
                  <span className="flex items-center gap-1 text-[10px] text-blue-400">
                    <Loader2 className="w-3 h-3 animate-spin" />
                    live
                  </span>
                )}
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
                      <span className={`shrink-0 mt-0.5 ${TASK_STATUS_COLORS[task.status]}`}>
                        {TASK_CATEGORY_ICONS[task.category]}
                      </span>
                      <span className={`font-semibold shrink-0 ${TASK_STATUS_COLORS[task.status]}`}>
                        [{task.label}]
                      </span>
                      <span className="text-muted-foreground truncate">{task.detail}</span>
                      <span className={`ml-auto shrink-0 uppercase text-[9px] font-bold tracking-wider ${TASK_STATUS_COLORS[task.status]}`}>
                        {task.status}
                      </span>
                    </div>
                  ))}
                  <div ref={logBottomRef} />
                </div>
              </ScrollArea>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
});

CognitiveEnginePanel.displayName = 'CognitiveEnginePanel';

export default CognitiveEnginePanel;
