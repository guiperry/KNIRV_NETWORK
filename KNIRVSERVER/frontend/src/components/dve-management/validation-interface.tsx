"use client";

import React, { useState, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Textarea } from '@/components/ui/textarea';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Badge } from '@/components/ui/badge';
import { Progress } from '@/components/ui/progress';
import { AlertCircle, CheckCircle, Clock, FileText, Zap, Brain, Calculator, MessageSquare } from 'lucide-react';
import { useToast } from '@/hooks/use-toast';

interface ValidationInterfaceProps {
  sessionId: string;
  validationType: string;
  onValidationComplete?: (result: ValidationResult) => void;
  className?: string;
}

interface ValidationResult {
  id: string;
  status: 'success' | 'warning' | 'error';
  confidence: number;
  issues: ValidationIssue[];
  suggestions: string[];
  timestamp: string;
}

interface ValidationIssue {
  type: 'factual' | 'logical' | 'mathematical' | 'consistency';
  severity: 'low' | 'medium' | 'high';
  description: string;
  location?: string;
}

export const ValidationInterface: React.FC<ValidationInterfaceProps> = ({
  sessionId,
  validationType,
  onValidationComplete,
  className = '',
}) => {
  const { toast } = useToast();
  const [content, setContent] = useState('');
  const [selectedValidationType, setSelectedValidationType] = useState(validationType || 'reasoning');
  const [isValidating, setIsValidating] = useState(false);
  const [progress, setProgress] = useState(0);
  const [result, setResult] = useState<ValidationResult | null>(null);
  const [validationHistory, setValidationHistory] = useState<ValidationResult[]>([]);

  const validationTypes = [
    { value: 'reasoning', label: 'Reasoning Validation', icon: Brain, description: 'Validate logical reasoning and argumentation' },
    { value: 'factual', label: 'Factual Accuracy', icon: CheckCircle, description: 'Check factual claims and references' },
    { value: 'mathematical', label: 'Mathematical Verification', icon: Calculator, description: 'Verify calculations and mathematical proofs' },
    { value: 'consistency', label: 'Consistency Check', icon: MessageSquare, description: 'Check for internal consistency and coherence' },
  ];

  const handleValidation = async () => {
    if (!content.trim()) {
      toast({
        title: "Validation Error",
        description: "Please enter content to validate.",
        variant: "destructive",
      });
      return;
    }

    setIsValidating(true);
    setProgress(0);
    setResult(null);

    try {
      // Simulate validation progress
      const progressInterval = setInterval(() => {
        setProgress(prev => {
          if (prev >= 90) {
            clearInterval(progressInterval);
            return 90;
          }
          return prev + 10;
        });
      }, 200);

      // Simulate API call
      await new Promise(resolve => setTimeout(resolve, 2000));

      clearInterval(progressInterval);
      setProgress(100);

      // Generate mock validation result
      const mockResult: ValidationResult = {
        id: `validation_${Date.now()}`,
        status: Math.random() > 0.3 ? 'success' : Math.random() > 0.5 ? 'warning' : 'error',
        confidence: Math.floor(Math.random() * 40) + 60, // 60-100%
        issues: generateMockIssues(selectedValidationType),
        suggestions: generateMockSuggestions(selectedValidationType),
        timestamp: new Date().toISOString(),
      };

      setResult(mockResult);
      setValidationHistory(prev => [mockResult, ...prev.slice(0, 4)]); // Keep last 5

      if (onValidationComplete) {
        onValidationComplete(mockResult);
      }

      toast({
        title: "Validation Complete",
        description: `Content validated with ${mockResult.confidence}% confidence.`,
      });

    } catch (error) {
      toast({
        title: "Validation Failed",
        description: "An error occurred during validation. Please try again.",
        variant: "destructive",
      });
    } finally {
      setIsValidating(false);
      setTimeout(() => setProgress(0), 1000);
    }
  };

  const generateMockIssues = (type: string): ValidationIssue[] => {
    const issues: ValidationIssue[] = [];
    const issueCount = Math.floor(Math.random() * 3);

    for (let i = 0; i < issueCount; i++) {
      const types = ['factual', 'logical', 'mathematical', 'consistency'];
      const severities: ('low' | 'medium' | 'high')[] = ['low', 'medium', 'high'];

      issues.push({
        type: types[Math.floor(Math.random() * types.length)] as any,
        severity: severities[Math.floor(Math.random() * severities.length)],
        description: `Sample ${type} validation issue ${i + 1}`,
        location: `Line ${Math.floor(Math.random() * 10) + 1}`,
      });
    }

    return issues;
  };

  const generateMockSuggestions = (type: string): string[] => {
    const suggestions = [
      "Consider providing more context for better validation",
      "Review the logical flow of your arguments",
      "Verify all factual claims with reliable sources",
      "Check mathematical calculations for accuracy",
      "Ensure consistent terminology throughout",
    ];

    return suggestions.slice(0, Math.floor(Math.random() * 3) + 1);
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'success':
        return <CheckCircle className="w-5 h-5 text-green-500" />;
      case 'warning':
        return <AlertCircle className="w-5 h-5 text-yellow-500" />;
      case 'error':
        return <AlertCircle className="w-5 h-5 text-red-500" />;
      default:
        return <Clock className="w-5 h-5 text-gray-500" />;
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'success':
        return 'bg-green-500';
      case 'warning':
        return 'bg-yellow-500';
      case 'error':
        return 'bg-red-500';
      default:
        return 'bg-gray-500';
    }
  };

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'high':
        return 'text-red-400';
      case 'medium':
        return 'text-yellow-400';
      case 'low':
        return 'text-green-400';
      default:
        return 'text-gray-400';
    }
  };

  return (
    <div className={`space-y-6 ${className}`}>
      {/* Validation Type Selection */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            <Zap className="w-5 h-5" />
            <span>Validation Configuration</span>
          </CardTitle>
          <CardDescription>
            Select the type of validation and enter content to analyze
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <label className="text-sm font-medium text-slate-300 mb-2 block">
              Validation Type
            </label>
            <Select value={selectedValidationType} onValueChange={setSelectedValidationType}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {validationTypes.map((type) => (
                  <SelectItem key={type.value} value={type.value}>
                    <div className="flex items-center space-x-2">
                      <type.icon className="w-4 h-4" />
                      <div>
                        <div className="font-medium">{type.label}</div>
                        <div className="text-xs text-slate-400">{type.description}</div>
                      </div>
                    </div>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div>
            <label className="text-sm font-medium text-slate-300 mb-2 block">
              Content to Validate
            </label>
            <Textarea
              value={content}
              onChange={(e) => setContent(e.target.value)}
              placeholder="Enter the content you want to validate..."
              className="min-h-32"
            />
          </div>

          <Button
            onClick={handleValidation}
            disabled={isValidating || !content.trim()}
            className="w-full"
          >
            {isValidating ? (
              <>
                <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
                Validating...
              </>
            ) : (
              <>
                <Zap className="w-4 h-4 mr-2" />
                Start Validation
              </>
            )}
          </Button>

          {isValidating && (
            <div className="space-y-2">
              <div className="flex justify-between text-sm">
                <span>Validation Progress</span>
                <span>{progress}%</span>
              </div>
              <Progress value={progress} className="w-full" />
            </div>
          )}
        </CardContent>
      </Card>

      {/* Validation Result */}
      {result && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center space-x-2">
              {getStatusIcon(result.status)}
              <span>Validation Result</span>
              <Badge className={`${getStatusColor(result.status)} text-white ml-auto`}>
                {result.confidence}% Confidence
              </Badge>
            </CardTitle>
            <CardDescription>
              Validation completed at {new Date(result.timestamp).toLocaleString()}
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {/* Issues */}
            {result.issues.length > 0 && (
              <div>
                <h4 className="text-sm font-medium text-slate-300 mb-2">Issues Found</h4>
                <div className="space-y-2">
                  {result.issues.map((issue, index) => (
                    <div key={index} className="flex items-start space-x-3 p-3 bg-slate-800/50 rounded-lg">
                      <AlertCircle className={`w-4 h-4 mt-0.5 ${getSeverityColor(issue.severity)}`} />
                      <div className="flex-1">
                        <div className="flex items-center space-x-2 mb-1">
                          <Badge variant="outline" className="text-xs">
                            {issue.type}
                          </Badge>
                          <span className={`text-xs ${getSeverityColor(issue.severity)}`}>
                            {issue.severity.toUpperCase()}
                          </span>
                        </div>
                        <p className="text-sm text-slate-300">{issue.description}</p>
                        {issue.location && (
                          <p className="text-xs text-slate-500 mt-1">Location: {issue.location}</p>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Suggestions */}
            {result.suggestions.length > 0 && (
              <div>
                <h4 className="text-sm font-medium text-slate-300 mb-2">Suggestions</h4>
                <ul className="space-y-1">
                  {result.suggestions.map((suggestion, index) => (
                    <li key={index} className="flex items-start space-x-2 text-sm">
                      <CheckCircle className="w-4 h-4 text-blue-500 mt-0.5 flex-shrink-0" />
                      <span className="text-slate-300">{suggestion}</span>
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {/* Validation History */}
      {validationHistory.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center space-x-2">
              <FileText className="w-5 h-5" />
              <span>Recent Validations</span>
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {validationHistory.map((historyItem) => (
                <div key={historyItem.id} className="flex items-center justify-between p-3 bg-slate-800/30 rounded-lg">
                  <div className="flex items-center space-x-3">
                    {getStatusIcon(historyItem.status)}
                    <div>
                      <p className="text-sm font-medium text-slate-300">
                        {validationTypes.find(t => t.value === selectedValidationType)?.label}
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

export default ValidationInterface;