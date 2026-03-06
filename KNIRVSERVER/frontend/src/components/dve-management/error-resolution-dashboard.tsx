"use client";

import React, { useState, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Textarea } from '@/components/ui/textarea';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Badge } from '@/components/ui/badge';
import { Progress } from '@/components/ui/progress';
import { AlertCircle, Bug, Code, Database, Zap, Search, FileText, CheckCircle, XCircle, AlertTriangle } from 'lucide-react';
import { useToast } from '@/hooks/use-toast';

interface ErrorResolutionDashboardProps {
  sessionId: string;
  supportedTypes: string[];
  onResolutionComplete?: (result: ResolutionResult) => void;
  className?: string;
}

interface ResolutionResult {
  id: string;
  status: 'resolved' | 'partial' | 'failed';
  errorType: string;
  rootCause: string;
  solution: string;
  confidence: number;
  steps: ResolutionStep[];
  timestamp: string;
}

interface ResolutionStep {
  id: string;
  description: string;
  status: 'pending' | 'running' | 'completed' | 'failed';
  output?: string;
}

export const ErrorResolutionDashboard: React.FC<ErrorResolutionDashboardProps> = ({
  sessionId,
  supportedTypes,
  onResolutionComplete,
  className = '',
}) => {
  const { toast } = useToast();
  const [errorDescription, setErrorDescription] = useState('');
  const [selectedErrorType, setSelectedErrorType] = useState('');
  const [isAnalyzing, setIsAnalyzing] = useState(false);
  const [analysisProgress, setAnalysisProgress] = useState(0);
  const [result, setResult] = useState<ResolutionResult | null>(null);
  const [resolutionHistory, setResolutionHistory] = useState<ResolutionResult[]>([]);

  const errorTypes = [
    { value: 'connection_timeout', label: 'Connection Timeout', icon: AlertTriangle, color: 'text-yellow-500' },
    { value: 'validation_failed', label: 'Validation Failed', icon: XCircle, color: 'text-red-500' },
    { value: 'resource_exhausted', label: 'Resource Exhausted', icon: Database, color: 'text-orange-500' },
    { value: 'custom_error', label: 'Custom Error', icon: Code, color: 'text-blue-500' },
  ].filter(type => supportedTypes.includes(type.value));

  const handleErrorAnalysis = async () => {
    if (!errorDescription.trim()) {
      toast({
        title: "Analysis Error",
        description: "Please describe the error you encountered.",
        variant: "destructive",
      });
      return;
    }

    if (!selectedErrorType) {
      toast({
        title: "Analysis Error",
        description: "Please select an error type.",
        variant: "destructive",
      });
      return;
    }

    setIsAnalyzing(true);
    setAnalysisProgress(0);
    setResult(null);

    try {
      // Simulate analysis progress
      const progressInterval = setInterval(() => {
        setAnalysisProgress(prev => {
          if (prev >= 95) {
            clearInterval(progressInterval);
            return 95;
          }
          return prev + 5;
        });
      }, 300);

      // Simulate API call
      await new Promise(resolve => setTimeout(resolve, 3000));

      clearInterval(progressInterval);
      setAnalysisProgress(100);

      // Generate mock resolution result
      const mockResult: ResolutionResult = {
        id: `resolution_${Date.now()}`,
        status: Math.random() > 0.2 ? 'resolved' : Math.random() > 0.5 ? 'partial' : 'failed',
        errorType: selectedErrorType,
        rootCause: generateMockRootCause(selectedErrorType),
        solution: generateMockSolution(selectedErrorType),
        confidence: Math.floor(Math.random() * 30) + 70, // 70-100%
        steps: generateMockSteps(selectedErrorType),
        timestamp: new Date().toISOString(),
      };

      setResult(mockResult);
      setResolutionHistory(prev => [mockResult, ...prev.slice(0, 4)]); // Keep last 5

      if (onResolutionComplete) {
        onResolutionComplete(mockResult);
      }

      toast({
        title: "Analysis Complete",
        description: `Error analyzed with ${mockResult.confidence}% confidence.`,
      });

    } catch (error) {
      toast({
        title: "Analysis Failed",
        description: "An error occurred during analysis. Please try again.",
        variant: "destructive",
      });
    } finally {
      setIsAnalyzing(false);
      setTimeout(() => setAnalysisProgress(0), 1000);
    }
  };

  const generateMockRootCause = (errorType: string): string => {
    const causes = {
      connection_timeout: "Network connectivity issue due to firewall blocking outbound connections",
      validation_failed: "Input data format mismatch with expected schema validation rules",
      resource_exhausted: "Memory allocation exceeded container limits during peak processing",
      custom_error: "Application-specific logic error in data processing pipeline",
    };
    return causes[errorType as keyof typeof causes] || "Unknown root cause identified";
  };

  const generateMockSolution = (errorType: string): string => {
    const solutions = {
      connection_timeout: "Configure firewall rules to allow outbound connections on required ports",
      validation_failed: "Update input data format to match expected schema requirements",
      resource_exhausted: "Increase container memory limits or optimize memory usage in application",
      custom_error: "Review and fix application logic in the data processing pipeline",
    };
    return solutions[errorType as keyof typeof solutions] || "Apply recommended fixes and restart the process";
  };

  const generateMockSteps = (errorType: string): ResolutionStep[] => {
    const baseSteps = [
      { id: '1', description: 'Analyze error logs and stack traces', status: 'completed' as const },
      { id: '2', description: 'Identify root cause and contributing factors', status: 'completed' as const },
      { id: '3', description: 'Develop and validate solution approach', status: 'completed' as const },
      { id: '4', description: 'Implement fix and test in staging environment', status: 'running' as const },
      { id: '5', description: 'Deploy fix to production and monitor', status: 'pending' as const },
    ];

    return baseSteps;
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'resolved':
        return <CheckCircle className="w-5 h-5 text-green-500" />;
      case 'partial':
        return <AlertTriangle className="w-5 h-5 text-yellow-500" />;
      case 'failed':
        return <XCircle className="w-5 h-5 text-red-500" />;
      default:
        return <AlertCircle className="w-5 h-5 text-gray-500" />;
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'resolved':
        return 'bg-green-500';
      case 'partial':
        return 'bg-yellow-500';
      case 'failed':
        return 'bg-red-500';
      default:
        return 'bg-gray-500';
    }
  };

  const getStepStatusIcon = (status: string) => {
    switch (status) {
      case 'completed':
        return <CheckCircle className="w-4 h-4 text-green-500" />;
      case 'running':
        return <div className="w-4 h-4 border-2 border-blue-500 border-t-transparent rounded-full animate-spin"></div>;
      case 'failed':
        return <XCircle className="w-4 h-4 text-red-500" />;
      default:
        return <AlertCircle className="w-4 h-4 text-gray-500" />;
    }
  };

  return (
    <div className={`space-y-6 ${className}`}>
      {/* Error Analysis Configuration */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            <Search className="w-5 h-5" />
            <span>Error Analysis</span>
          </CardTitle>
          <CardDescription>
            Describe the error and select its type for automated analysis and resolution
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <label className="text-sm font-medium text-slate-300 mb-2 block">
              Error Type
            </label>
            <Select value={selectedErrorType} onValueChange={setSelectedErrorType}>
              <SelectTrigger>
                <SelectValue placeholder="Select error type" />
              </SelectTrigger>
              <SelectContent>
                {errorTypes.map((type) => (
                  <SelectItem key={type.value} value={type.value}>
                    <div className="flex items-center space-x-2">
                      <type.icon className="w-4 h-4" />
                      <span>{type.label}</span>
                    </div>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div>
            <label className="text-sm font-medium text-slate-300 mb-2 block">
              Error Description
            </label>
            <Textarea
              value={errorDescription}
              onChange={(e) => setErrorDescription(e.target.value)}
              placeholder="Describe the error you encountered, including any error messages, stack traces, or symptoms..."
              className="min-h-32"
            />
          </div>

          <Button
            onClick={handleErrorAnalysis}
            disabled={isAnalyzing || !errorDescription.trim() || !selectedErrorType}
            className="w-full"
          >
            {isAnalyzing ? (
              <>
                <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
                Analyzing...
              </>
            ) : (
              <>
                <Search className="w-4 h-4 mr-2" />
                Analyze Error
              </>
            )}
          </Button>

          {isAnalyzing && (
            <div className="space-y-2">
              <div className="flex justify-between text-sm">
                <span>Analysis Progress</span>
                <span>{analysisProgress}%</span>
              </div>
              <Progress value={analysisProgress} className="w-full" />
            </div>
          )}
        </CardContent>
      </Card>

      {/* Analysis Result */}
      {result && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center space-x-2">
              {getStatusIcon(result.status)}
              <span>Analysis Result</span>
              <Badge className={`${getStatusColor(result.status)} text-white ml-auto`}>
                {result.confidence}% Confidence
              </Badge>
            </CardTitle>
            <CardDescription>
              Error analysis completed at {new Date(result.timestamp).toLocaleString()}
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {/* Root Cause */}
            <div>
              <h4 className="text-sm font-medium text-slate-300 mb-2">Root Cause</h4>
              <p className="text-slate-300 bg-slate-800/50 p-3 rounded-lg">{result.rootCause}</p>
            </div>

            {/* Solution */}
            <div>
              <h4 className="text-sm font-medium text-slate-300 mb-2">Recommended Solution</h4>
              <p className="text-slate-300 bg-slate-800/50 p-3 rounded-lg">{result.solution}</p>
            </div>

            {/* Resolution Steps */}
            <div>
              <h4 className="text-sm font-medium text-slate-300 mb-3">Resolution Steps</h4>
              <div className="space-y-3">
                {result.steps.map((step) => (
                  <div key={step.id} className="flex items-start space-x-3 p-3 bg-slate-800/30 rounded-lg">
                    {getStepStatusIcon(step.status)}
                    <div className="flex-1">
                      <p className="text-sm text-slate-300">{step.description}</p>
                      {step.output && (
                        <code className="text-xs text-slate-400 bg-slate-900/50 p-2 rounded mt-2 block">
                          {step.output}
                        </code>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Resolution History */}
      {resolutionHistory.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center space-x-2">
              <FileText className="w-5 h-5" />
              <span>Recent Analyses</span>
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {resolutionHistory.map((historyItem) => (
                <div key={historyItem.id} className="flex items-center justify-between p-3 bg-slate-800/30 rounded-lg">
                  <div className="flex items-center space-x-3">
                    {getStatusIcon(historyItem.status)}
                    <div>
                      <p className="text-sm font-medium text-slate-300">
                        {errorTypes.find(t => t.value === historyItem.errorType)?.label}
                      </p>
                      <p className="text-xs text-slate-500">
                        {new Date(historyItem.timestamp).toLocaleString()}
                      </p>
                    </div>
                  </div>
                  <Badge variant="outline" className="text-xs">
                    {historyItem.confidence}%
                  </Badge>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
};

export default ErrorResolutionDashboard;