'use client';

import React, { useEffect, useState, useRef } from 'react';
import { Brain, Cpu, Zap, Activity, TrendingUp, Clock, AlertCircle, CheckCircle, Heart, Eye, EyeOff, Play, Square } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Progress } from '@/components/ui/progress';
import { ScrollArea } from '@/components/ui/scroll-area';
import { useCognitiveEngine } from '@/hooks/use-cognitive-engine';
import NeuralDesktopPanel from './neural-desktop-panel';

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
    startPolling,
    stopPolling,
  } = useCognitiveEngine();

  const [showTaskStatus, setShowTaskStatus] = useState(false);
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

          {/* Current Task Status */}
          {cognitiveEngine?.current_task_status && showTaskStatus && (
            <div className="mt-4 p-3 bg-green-500/10 border border-green-500/20 rounded-lg">
              <div className="flex items-center gap-2 text-green-400 text-sm">
                <Activity className="w-4 h-4" />
                <span className="font-medium">Current Task:</span>
                <span>{cognitiveEngine.current_task_status}</span>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
};

export default CognitiveEnginePanel;
