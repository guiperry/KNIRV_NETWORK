"use client";

import React, { useState, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Textarea } from '@/components/ui/textarea';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Badge } from '@/components/ui/badge';
import { Progress } from '@/components/ui/progress';
import { AlertCircle, CheckCircle, Clock, FileText, Zap, Brain, Calculator, MessageSquare, ShieldCheck, ShieldAlert, Award } from 'lucide-react';
import { useToast } from '@/hooks/use-toast';
import { apiRequest } from '@/lib/api';
import type { CertificateOfCorrectness, FactualityCheckRequest, FactualityCheckResponse, NRVTrailSummary } from '@/types/api';

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
  proof?: string;
  degraded?: boolean;
  nrvTrail?: NRVTrailSummary;
  certificate?: CertificateOfCorrectness;
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

    // UI-only progress feedback while the real request is in flight.
    const progressInterval = setInterval(() => {
      setProgress(prev => (prev >= 90 ? 90 : prev + 10));
    }, 200);

    try {
      const request: FactualityCheckRequest = {
        prompt: content,
        response: content,
        agent_id: 'validation-interface',
        dve_id: sessionId,
        ontology_domains: [selectedValidationType],
        objective_name: selectedValidationType,
        preference_weights: {},
      };

      const response = await apiRequest<FactualityCheckResponse>('/api/validation/factuality', {
        method: 'POST',
        body: JSON.stringify(request),
      });

      if (!response.success || !response.data) {
        throw new Error(response.error || 'Validation failed');
      }

      const factuality = response.data;
      setProgress(100);

      const realResult = mapFactualityResult(factuality);

      setResult(realResult);
      setValidationHistory(prev => [realResult, ...prev.slice(0, 4)]); // Keep last 5

      if (onValidationComplete) {
        onValidationComplete(realResult);
      }

      toast({
        title: "Validation Complete",
        description: `Content validated with ${realResult.confidence}% confidence${realResult.proof ? ' · proof attached' : ''}${realResult.certificate ? ' · certified' : ''}.`,
      });

    } catch (error) {
      toast({
        title: "Validation Failed",
        description: error instanceof Error && error.message ? error.message : "An error occurred during validation. Please try again.",
        variant: "destructive",
      });
    } finally {
      clearInterval(progressInterval);
      setIsValidating(false);
      setTimeout(() => setProgress(0), 1000);
    }
  };

  const mapFactualityResult = (resp: FactualityCheckResponse): ValidationResult => {
    const issues: ValidationIssue[] = [];

    if (resp.degraded) {
      issues.push({
        type: 'factual',
        severity: 'medium',
        description: 'Validator is operating in degraded mode; the result was produced by fallback logic.',
      });
    }

    if (!resp.is_accurate) {
      issues.push({
        type: 'factual',
        severity: 'high',
        description: resp.explanation || 'The response could not be confirmed as factually accurate.',
      });
    } else if (resp.citations.length > 0) {
      issues.push({
        type: 'factual',
        severity: 'low',
        description: `Cross-checked against ${resp.citations.length} evidence chunk${resp.citations.length === 1 ? '' : 's'}.`,
      });
    }

    const suggestions: string[] = [];
    if (resp.explanation && resp.explanation !== 'All validation checks passed') {
      suggestions.push(resp.explanation);
    }
    if (resp.domain_scores) {
      const highestDomain = Object.entries(resp.domain_scores).sort((a, b) => b[1] - a[1])[0];
      if (highestDomain) {
        suggestions.push(`Highest-confidence domain: ${highestDomain[0]} (${Math.round(highestDomain[1] * 100)}%)`);
      }
    }

    const status: ValidationResult['status'] = resp.degraded ? 'warning' : resp.is_accurate ? 'success' : 'error';

    return {
      id: `validation_${Date.now()}`,
      status,
      confidence: Math.round(resp.confidence * 100),
      issues,
      suggestions,
      timestamp: new Date().toISOString(),
      proof: resp.proof,
      degraded: resp.degraded,
      nrvTrail: resp.nrv_trail,
      certificate: resp.certificate_of_correctness,
    };
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

  const getCertBadge = (status: string) => {
    switch (status) {
      case 'COMPLIANT':
        return { className: 'bg-green-600 text-white', label: 'Compliant', color: 'text-green-400', bg: 'bg-green-900/10 border-green-700/40' };
      case 'PROVISIONAL':
        return { className: 'bg-yellow-600 text-white', label: 'Provisional', color: 'text-yellow-400', bg: 'bg-yellow-900/10 border-yellow-700/40' };
      default:
        return { className: 'bg-red-600 text-white', label: 'Non-Compliant', color: 'text-red-400', bg: 'bg-red-900/10 border-red-700/40' };
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
            {/* Degraded mode warning */}
            {result.degraded && (
              <div className="flex items-start space-x-3 p-3 bg-amber-900/20 border border-amber-700/50 rounded-lg text-amber-200 text-sm">
                <ShieldAlert className="w-5 h-5 mt-0.5 text-amber-400 flex-shrink-0" />
                <div>
                  <p className="font-medium text-amber-300">Validator Degraded</p>
                  <p className="mt-1">
                    The factuality endpoint returned a degraded result. The response was produced by fallback logic and
                    should be treated with caution.
                  </p>
                </div>
              </div>
            )}

            {/* Proof + verified badge */}
            {result.proof ? (
              <div className="flex items-start space-x-3 p-3 bg-green-900/10 border border-green-700/40 rounded-lg">
                <ShieldCheck className="w-5 h-5 mt-0.5 text-green-500 flex-shrink-0" />
                <div className="min-w-0 flex-1">
                  <div className="flex items-center space-x-2 mb-1">
                    <Badge className="bg-green-600 text-white">Verified</Badge>
                    <span className="text-xs text-slate-400">1 NRN consumed · proof sealed</span>
                  </div>
                  <code className="block bg-slate-800/70 px-3 py-2 rounded text-xs text-green-300 break-all font-mono leading-relaxed mt-1">
                    {result.proof}
                  </code>
                </div>
              </div>
            ) : (
              <div className="flex items-center space-x-2 p-3 bg-slate-800/30 rounded-lg text-sm text-slate-400">
                <Clock className="w-4 h-4" />
                <span>No cryptographic proof attached — check may not have been billed.</span>
              </div>
            )}

            {/* Certificate of Correctness (FINTECH-3) */}
            {result.certificate && (
              (() => {
                const cert = result.certificate;
                const certBadge = getCertBadge(cert.status);
                return (
                  <div className={`flex items-start space-x-3 p-3 rounded-lg border ${certBadge.bg}`}>
                    <Award className={`w-5 h-5 mt-0.5 flex-shrink-0 ${certBadge.color}`} />
                    <div className="min-w-0 flex-1 space-y-1">
                      <div className="flex items-center space-x-2">
                        <Badge className={certBadge.className}>Certificate of Correctness</Badge>
                        {cert.signed ? (
                          <span className="text-xs text-green-400">PQC-signed</span>
                        ) : (
                          <span className="text-xs text-slate-400">unsigned</span>
                        )}
                      </div>
                      <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-xs text-slate-300">
                        <span>Status: <span className={certBadge.color}>{certBadge.label}</span></span>
                        <span>Level: {cert.compliance_level}</span>
                        <span>Score: {Math.round(cert.overall_score)}</span>
                        <span>Issuer: {cert.issuer_node_id}</span>
                      </div>
                      <div className="text-xs text-slate-500">
                        {cert.id}
                        {result.nrvTrail && (
                          <span className="ml-2 text-slate-400">· trace {result.nrvTrail.trace_id} · {result.nrvTrail.step_count} steps</span>
                        )}
                      </div>
                    </div>
                  </div>
                );
              })()
            )}

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
                <h4 className="text-sm font-medium text-slate-300 mb-2">Details</h4>
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