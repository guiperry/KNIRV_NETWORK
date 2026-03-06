'use client';

import React, { useEffect, useState, useRef } from 'react';
import { Brain, Cpu, Zap, Activity, TrendingUp, Clock, AlertCircle, CheckCircle, GitBranch, Network, Server, Users, FileText, Database, ChevronRight, Heart, MessageSquare, Eye, EyeOff, Play, Square, X } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Progress } from '@/components/ui/progress';
import { ScrollArea } from '@/components/ui/scroll-area';
import { useCognitiveEngine } from '@/hooks/use-cognitive-engine';

interface CognitiveEnginePanelProps {
  className?: string;
}

export const CognitiveEnginePanel: React.FC<CognitiveEnginePanelProps> = ({ className }) => {
  const {
    cognitiveEngine,
    isLoading,
    error,
    isPolling,
    isConnected,
    startEngine,
    stopEngine,
    resetMetrics,
    clearConversationHistory,
    healthCheck,
    selfValidate,
    makeRequest,
    startPolling,
    stopPolling,
  } = useCognitiveEngine();

  const [showTaskStatus, setShowTaskStatus] = useState(false);
  const [showChat, setShowChat] = useState(false);
  const [chatMessage, setChatMessage] = useState('');
  const [healthResult, setHealthResult] = useState<string | null>(null);
  const [validationResult, setValidationResult] = useState<string | null>(null);

  // Start polling as fallback if WebSocket is not connected
  useEffect(() => {
    if (!isConnected) {
      startPolling(10000); // Poll every 10 seconds as fallback
    } else {
      stopPolling(); // Stop polling when WebSocket is connected
    }
    return () => stopPolling();
  }, [isConnected, startPolling, stopPolling]);

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active': return 'bg-green-500';
      case 'learning': return 'bg-blue-500';
      case 'idle': return 'bg-yellow-500';
      case 'error': return 'bg-red-500';
      default: return 'bg-gray-500';
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'active': return <CheckCircle className="w-4 h-4" />;
      case 'learning': return <Activity className="w-4 h-4" />;
      case 'idle': return <Clock className="w-4 h-4" />;
      case 'error': return <AlertCircle className="w-4 h-4" />;
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
      {/* Status Overview */}
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
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="text-center">
              <div className="text-2xl font-bold text-primary">
                {cognitiveEngine?.accuracy.toFixed(1) || '--'}%
              </div>
              <div className="text-xs text-muted-foreground">Accuracy</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-primary">
                {cognitiveEngine?.tasks_processed.toLocaleString() || '--'}
              </div>
              <div className="text-xs text-muted-foreground">Tasks Processed</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-primary">
                {cognitiveEngine ? (cognitiveEngine.adaptation_rate * 100).toFixed(0) : '--'}%
              </div>
              <div className="text-xs text-muted-foreground">Adaptation Rate</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-primary">
                {cognitiveEngine ? formatUptime(cognitiveEngine.uptime) : '--'}
              </div>
              <div className="text-xs text-muted-foreground">Uptime</div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Rolling Status Box - Cognitive Engine Activity */}
      <Card className="knirv-card-gradient border-blue-500/30">
        <CardHeader className="pb-2">
          <CardTitle className="flex items-center justify-between text-sm">
            <div className="flex items-center space-x-2">
              <Activity className="w-4 h-4 text-blue-500 animate-pulse" />
              <span>Current Processing</span>
            </div>
            <Badge variant="outline" className="text-xs bg-blue-500/10 text-blue-400 border-blue-500/30">
              Live
            </Badge>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <RollingStatusBox />
        </CardContent>
      </Card>

      {/* Performance and Learning Metrics */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Card className="knirv-card-gradient">
          <CardHeader>
            <CardTitle className="flex items-center space-x-2">
              <Cpu className="w-5 h-5" />
              <span>Performance Metrics</span>
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              <div className="flex justify-between items-center">
                <span className="text-sm">Inference Latency:</span>
                <span className="text-sm font-medium">
                  {cognitiveEngine?.performance_metrics?.inference_latency?.toFixed(1) || '--'}ms
                </span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-sm">Throughput:</span>
                <span className="text-sm font-medium">
                  {cognitiveEngine?.performance_metrics?.throughput?.toLocaleString() || '--'} req/s
                </span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-sm">Error Rate:</span>
                <span className="text-sm font-medium">
                  {cognitiveEngine?.performance_metrics ? (cognitiveEngine.performance_metrics.error_rate * 100).toFixed(2) : '--'}%
                </span>
              </div>
              <div className="mt-4">
                <div className="flex justify-between text-xs mb-1">
                  <span>Error Rate</span>
                  <span>{cognitiveEngine?.performance_metrics ? (cognitiveEngine.performance_metrics.error_rate * 100).toFixed(1) : 0}%</span>
                </div>
                <Progress 
                  value={cognitiveEngine?.performance_metrics ? cognitiveEngine.performance_metrics.error_rate * 100 : 0} 
                  className="h-2"
                />
              </div>
            </div>
          </CardContent>
        </Card>

        <Card className="knirv-card-gradient">
          <CardHeader>
            <CardTitle className="flex items-center space-x-2">
              <TrendingUp className="w-5 h-5" />
              <span>Learning Metrics</span>
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              <div className="flex justify-between items-center">
                <span className="text-sm">Training Accuracy:</span>
                <span className="text-sm font-medium">
                  {cognitiveEngine?.learning_metrics?.training_accuracy?.toFixed(1) || '--'}%
                </span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-sm">Validation Accuracy:</span>
                <span className="text-sm font-medium">
                  {cognitiveEngine?.learning_metrics?.validation_accuracy?.toFixed(1) || '--'}%
                </span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-sm">Loss:</span>
                <span className="text-sm font-medium">
                  {cognitiveEngine?.learning_metrics?.loss?.toFixed(3) || '--'}
                </span>
              </div>
              <div className="mt-4">
                <div className="flex justify-between text-xs mb-1">
                  <span>Training Progress</span>
                  <span>{cognitiveEngine?.learning_metrics?.training_accuracy?.toFixed(1) || 0}%</span>
                </div>
                <Progress 
                  value={cognitiveEngine?.learning_metrics?.training_accuracy || 0} 
                  className="h-2"
                />
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Control Actions */}
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
              variant={cognitiveEngine?.status === 'running' ? 'secondary' : 'default'}
              size="sm"
              onClick={startEngine}
              disabled={isLoading || cognitiveEngine?.status === 'running'}
            >
              <Play className="w-4 h-4 mr-2" />
              Start Engine
            </Button>
            <Button
              variant={cognitiveEngine?.status === 'running' ? 'destructive' : 'secondary'}
              size="sm"
              onClick={stopEngine}
              disabled={isLoading || cognitiveEngine?.status !== 'running'}
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
              onClick={() => setShowChat(!showChat)}
              disabled={isLoading}
            >
              <MessageSquare className="w-4 h-4 mr-2" />
              Make Request
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
          {cognitiveEngine?.current_task_status && showTaskStatus && (
            <div className="mt-4 text-xs text-muted-foreground">
              Current Task: {cognitiveEngine.current_task_status}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Chat Request Panel */}
      {showChat && (
        <Card className="knirv-card-gradient border-blue-500/30">
          <CardHeader className="pb-2">
            <CardTitle className="flex items-center justify-between text-sm">
              <div className="flex items-center space-x-2">
                <MessageSquare className="w-4 h-4 text-blue-500" />
                <span>Make Request (Chat)</span>
              </div>
              <Button variant="ghost" size="sm" onClick={() => setShowChat(false)}>
                <X className="w-4 h-4" />
              </Button>
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              <textarea
                className="w-full bg-slate-800 border border-blue-600/30 rounded p-2 text-sm text-slate-200 resize-none"
                rows={3}
                placeholder="Enter your request message..."
                value={chatMessage}
                onChange={(e) => setChatMessage(e.target.value)}
              />
              <Button
                variant="default"
                size="sm"
                className="w-full"
                onClick={() => {
                  if (chatMessage.trim()) {
                    makeRequest(chatMessage);
                    setChatMessage('');
                    setShowChat(false);
                  }
                }}
                disabled={!chatMessage.trim() || isLoading}
              >
                <MessageSquare className="w-4 h-4 mr-2" />
                Send Request
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Current Task Status Panel */}
      {showTaskStatus && cognitiveEngine?.current_task_status && (
        <Card className="knirv-card-gradient border-green-500/30">
          <CardHeader className="pb-2">
            <CardTitle className="flex items-center justify-between text-sm">
              <div className="flex items-center space-x-2">
                <Activity className="w-4 h-4 text-green-500" />
                <span>Current Task Status</span>
              </div>
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-sm text-slate-200">
              {cognitiveEngine.current_task_status}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
};

// Rolling Status Box Component - Shows current cognitive engine activities
const RollingStatusBox: React.FC = () => {
  const [activities, setActivities] = useState<Array<{
    id: string;
    icon: React.ReactNode;
    title: string;
    description: string;
    status: 'active' | 'pending' | 'completed';
    timestamp: Date;
  }>>([]);
  
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    // Simulated cognitive engine activities
    const initialActivities = [
      {
        id: '1',
        icon: <GitBranch className="w-3 h-3" />,
        title: 'Creating Workflow',
        description: 'Analyzing recent activity patterns to generate new workflow',
        status: 'active' as const,
        timestamp: new Date(),
      },
      {
        id: '2',
        icon: <Network className="w-3 h-3" />,
        title: 'Organizing Ontology',
        description: 'Resolving pending error nodes in the knowledge graph',
        status: 'pending' as const,
        timestamp: new Date(Date.now() - 5000),
      },
      {
        id: '3',
        icon: <Server className="w-3 h-3" />,
        title: 'DVE Monitoring',
        description: 'Monitoring new connection activity in DVE instances',
        status: 'pending' as const,
        timestamp: new Date(Date.now() - 10000),
      },
      {
        id: '4',
        icon: <Users className="w-3 h-3" />,
        title: 'Agent Activity',
        description: 'Tracking active agent tasks across all DVE nodes',
        status: 'pending' as const,
        timestamp: new Date(Date.now() - 15000),
      },
      {
        id: '5',
        icon: <FileText className="w-3 h-3" />,
        title: 'Guardrails Report',
        description: 'Generating security guardrails compliance report',
        status: 'pending' as const,
        timestamp: new Date(Date.now() - 20000),
      },
      {
        id: '6',
        icon: <Database className="w-3 h-3" />,
        title: 'Cache Allocation',
        description: 'Allocating memory resources to active DVE containers',
        status: 'pending' as const,
        timestamp: new Date(Date.now() - 25000),
      },
    ];
    
    setActivities(initialActivities);

    // Rotate activities periodically
    const interval = setInterval(() => {
      setActivities(prev => {
        if (prev.length === 0) return prev;
        const updated = [...prev];
        const first = updated.shift();
        if (first) {
          updated.push({
            ...first,
            status: 'completed',
            timestamp: new Date(),
          });
        }
        // Mark next pending as active
        const nextPending = updated.find(a => a.status === 'pending');
        if (nextPending) {
          nextPending.status = 'active';
        }
        return updated;
      });
    }, 5000);

    return () => clearInterval(interval);
  }, []);

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active': return 'text-blue-400';
      case 'completed': return 'text-green-400';
      default: return 'text-slate-400';
    }
  };

  const getIconColor = (status: string) => {
    switch (status) {
      case 'active': return 'bg-blue-500/20 text-blue-400';
      case 'completed': return 'bg-green-500/20 text-green-400';
      default: return 'bg-slate-500/20 text-slate-400';
    }
  };

  return (
    <div ref={scrollRef} className="space-y-2 max-h-[200px] overflow-y-auto">
      {activities.map((activity) => (
        <div 
          key={activity.id}
          className={`flex items-start gap-3 p-2 rounded-lg transition-all ${
            activity.status === 'active' ? 'bg-blue-500/10 border border-blue-500/20' : 'bg-slate-800/30'
          }`}
        >
          <div className={`p-1.5 rounded-md ${getIconColor(activity.status)} ${activity.status === 'active' ? 'animate-pulse' : ''}`}>
            {activity.icon}
          </div>
          <div className="flex-1 min-w-0">
            <div className="flex items-center justify-between">
              <span className={`text-xs font-medium ${getStatusColor(activity.status)}`}>
                {activity.title}
              </span>
              {activity.status === 'active' && (
                <Badge variant="outline" className="text-[8px] h-4 bg-blue-500/20 text-blue-400 border-blue-500/30">
                  ACTIVE
                </Badge>
              )}
              {activity.status === 'completed' && (
                <CheckCircle className="w-3 h-3 text-green-400" />
              )}
            </div>
            <p className="text-[10px] text-slate-400 truncate">
              {activity.description}
            </p>
          </div>
          <ChevronRight className={`w-3 h-3 ${getStatusColor(activity.status)} opacity-50`} />
        </div>
      ))}
      <div className="text-[10px] text-slate-500 text-center pt-2">
        Processing queue: {activities.filter(a => a.status === 'active').length} active • {activities.filter(a => a.status === 'pending').length} pending
      </div>
    </div>
  );
};

export default CognitiveEnginePanel;
