'use client';

import React, { useEffect } from 'react';
import { Brain, Cpu, Zap, Activity, TrendingUp, Clock, AlertCircle, CheckCircle } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Progress } from '@/components/ui/progress';
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
    startTraining,
    stopTraining,
    resetMetrics,
    clearConversationHistory,
    startPolling,
    stopPolling,
  } = useCognitiveEngine();

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
              variant={cognitiveEngine?.status === 'learning' ? 'secondary' : 'default'}
              size="sm"
              onClick={startTraining}
              disabled={isLoading || cognitiveEngine?.status === 'learning'}
            >
              <Activity className="w-4 h-4 mr-2" />
              Start Training
            </Button>
            <Button
              variant={cognitiveEngine?.status === 'learning' ? 'destructive' : 'secondary'}
              size="sm"
              onClick={stopTraining}
              disabled={isLoading || cognitiveEngine?.status !== 'learning'}
            >
              <Clock className="w-4 h-4 mr-2" />
              Stop Training
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
          {cognitiveEngine?.last_training && (
            <div className="mt-4 text-xs text-muted-foreground">
              Last training: {new Date(cognitiveEngine.last_training).toLocaleString()}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
};

export default CognitiveEnginePanel;
