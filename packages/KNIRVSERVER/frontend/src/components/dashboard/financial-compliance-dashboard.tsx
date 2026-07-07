"use client";

import { useState, useEffect } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Progress } from '@/components/ui/progress';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Separator } from '@/components/ui/separator';
import { useToast } from '@/hooks/use-toast';
import useFinTechValidator from '@/hooks/use-fintech-validator';
import {
  Shield,
  FileText,
  Activity,
  AlertTriangle,
  CheckCircle,
  XCircle,
  Clock,
  TrendingUp,
  TrendingDown,
  DollarSign,
  Users,
  Scale,
  Brain,
  Play,
  Download,
  RefreshCw,
  Eye,
  AlertCircle,
  Search,
  Filter,
} from 'lucide-react';
import type {
  NRVTrace,
  FidelityScore,
  SemanticDistance,
  KYCBypassDetection,
  PositionLimitViolation,
  EvidencePack,
  ExecutionTrajectory,
} from '@/types/api';

interface FinancialComplianceDashboardProps {
  className?: string;
}

export function FinancialComplianceDashboard({ className }: FinancialComplianceDashboardProps) {
  const { toast } = useToast();
  const {
    status,
    isLoading,
    error,
    evidencePacks,
    nrvTraces,
    selectedTrace,
    fidelityScore,
    semanticDistance,
    kycBypassDetection,
    positionLimitViolations,
    trajectories,
    validationResult,
    fetchNRVTraces,
    fetchNRVTrace,
    fetchEvidencePacks,
    fetchTrajectories,
    scoreFidelity,
    calculateSemanticDistance,
    detectKYCBypass,
    detectPositionLimits,
    validate,
    isPluginEnabled,
  } = useFinTechValidator();

  const [selectedTraceId, setSelectedTraceId] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState('overview');

  // Plugin disabled guard — show a friendly message when the validator is off.
  if (!isPluginEnabled && !isLoading) {
    return (
      <div className={className} style={{ padding: '24px', textAlign: 'center', color: '#888' }}>
        <Shield style={{ width: 48, height: 48, margin: '0 auto 12px', opacity: 0.4 }} />
        <h3 style={{ marginBottom: '8px' }}>FinTech Compliance Validator is disabled</h3>
        <p style={{ fontSize: '14px' }}>
          Enable the <strong>FinTech Compliance Validator</strong> plugin to access compliance
          dashboards, NRV traces, and evidence packs.
        </p>
      </div>
    );
  }

  useEffect(() => {
    fetchNRVTraces();
    fetchEvidencePacks();
    fetchTrajectories();
  }, [fetchNRVTraces, fetchEvidencePacks, fetchTrajectories]);

  useEffect(() => {
    if (error) {
      toast({
        title: 'Error',
        description: error,
        variant: 'destructive',
      });
    }
  }, [error, toast]);

  const handleTraceSelect = async (traceId: string) => {
    setSelectedTraceId(traceId);
    await fetchNRVTrace(traceId);
  };

  const handleScoreFidelity = async () => {
    if (selectedTraceId) {
      await scoreFidelity(selectedTraceId);
    }
  };

  const handleSemanticDistance = async () => {
    if (selectedTraceId) {
      await calculateSemanticDistance(selectedTraceId);
    }
  };

  const handleDetectKYCBypass = async () => {
    if (selectedTraceId) {
      await detectKYCBypass(selectedTraceId);
    }
  };

  const handleDetectPositionLimits = async () => {
    if (selectedTraceId) {
      await detectPositionLimits(selectedTraceId, {});
    }
  };

  const getRiskBadge = (riskLevel?: string) => {
    switch (riskLevel) {
      case 'low':
        return <Badge className="bg-green-500"><CheckCircle className="w-3 h-3 mr-1" /> Low Risk</Badge>;
      case 'medium':
        return <Badge className="bg-yellow-500"><AlertTriangle className="w-3 h-3 mr-1" /> Medium Risk</Badge>;
      case 'high':
        return <Badge className="bg-orange-500"><AlertTriangle className="w-3 h-3 mr-1" /> High Risk</Badge>;
      case 'critical':
        return <Badge className="bg-red-500"><XCircle className="w-3 h-3 mr-1" /> Critical</Badge>;
      default:
        return <Badge variant="outline">Unknown</Badge>;
    }
  };

  const getScoreColor = (score: number) => {
    if (score >= 0.75) return 'text-green-500';
    if (score >= 0.50) return 'text-yellow-500';
    if (score >= 0.25) return 'text-orange-500';
    return 'text-red-500';
  };

  const runDemoValidation = async () => {
    const result = await validate({
      agent_id: 'demo-agent-001',
      agent_name: 'Demo Trading Bot',
      validation_type: 'trade',
      parameters: {
        action: 'buy',
        instrument: 'AAPL',
        quantity: 100,
        price: 150.00,
      },
      simple_financial_actions: [
        {
          action_type: 'BUY',
          instrument: 'AAPL',
          quantity: 100,
          price: 150.00,
          timestamp: new Date().toISOString(),
          account_id: 'demo-account-001',
        },
      ],
    });
    if (result) {
      toast({
        title: 'Validation Complete',
        description: `Validation score: ${(result.overall_score * 100).toFixed(1)}%`,
      });
    }
  };

  return (
    <div className={`space-y-6 ${className}`}>
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Shield className="h-8 w-8 text-primary" />
          <div>
            <h2 className="text-2xl font-bold">Financial Compliance Dashboard</h2>
            <p className="text-muted-foreground">
              Deterministic Validation of Financial AI Agents
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => {
            fetchNRVTraces();
            fetchEvidencePacks();
            fetchTrajectories();
          }}>
            <RefreshCw className="w-4 h-4 mr-2" />
            Refresh
          </Button>
          <Button size="sm" onClick={runDemoValidation}>
            <Play className="w-4 h-4 mr-2" />
            Run Demo Validation
          </Button>
        </div>
      </div>

      {status && (
        <Alert>
          <Shield className="h-4 w-4" />
          <AlertTitle>FinTech Validator Status</AlertTitle>
          <AlertDescription>
            Version {status.version} | {status.ontologies} ontologies loaded | {nrvTraces.length} traces captured
          </AlertDescription>
        </Alert>
      )}

      <Tabs defaultValue={activeTab} onValueChange={setActiveTab}>
        <TabsList className="grid w-full grid-cols-5">
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="traces">NRV Traces</TabsTrigger>
          <TabsTrigger value="fidelity">Fidelity Scores</TabsTrigger>
          <TabsTrigger value="evidence">Evidence Packs</TabsTrigger>
          <TabsTrigger value="trajectories">Trajectories</TabsTrigger>
        </TabsList>

        {/* Overview Tab */}
        <TabsContent value="overview" className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            <Card className="knirv-card-gradient">
              <CardHeader className="flex flex-row items-center justify-between pb-2">
                <CardTitle className="text-sm font-medium">NRV Traces</CardTitle>
                <Activity className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{nrvTraces.length}</div>
                <p className="text-xs text-muted-foreground">Reasoning traces captured</p>
              </CardContent>
            </Card>

            <Card className="knirv-card-gradient">
              <CardHeader className="flex flex-row items-center justify-between pb-2">
                <CardTitle className="text-sm font-medium">Evidence Packs</CardTitle>
                <FileText className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{evidencePacks.length}</div>
                <p className="text-xs text-muted-foreground">Compliance records</p>
              </CardContent>
            </Card>

            <Card className="knirv-card-gradient">
              <CardHeader className="flex flex-row items-center justify-between pb-2">
                <CardTitle className="text-sm font-medium">Avg Fidelity</CardTitle>
                <Brain className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                {fidelityScore ? (
                  <>
                    <div className={`text-2xl font-bold ${getScoreColor(fidelityScore.overall_score)}`}>
                      {(fidelityScore.overall_score * 100).toFixed(1)}%
                    </div>
                    <p className="text-xs text-muted-foreground">
                      {getRiskBadge(fidelityScore.risk_level)}
                    </p>
                  </>
                ) : (
                  <>
                    <div className="text-2xl font-bold">--</div>
                    <p className="text-xs text-muted-foreground">Select a trace to score</p>
                  </>
                )}
              </CardContent>
            </Card>

            <Card className="knirv-card-gradient">
              <CardHeader className="flex flex-row items-center justify-between pb-2">
                <CardTitle className="text-sm font-medium">Violations</CardTitle>
                <AlertTriangle className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="flex gap-2">
                  <div className="flex-1">
                    <div className="text-lg font-bold text-red-500">
                      {kycBypassDetection?.detected ? '1' : '0'}
                    </div>
                    <p className="text-xs text-muted-foreground">KYC Bypass</p>
                  </div>
                  <div className="flex-1">
                    <div className="text-lg font-bold text-orange-500">
                      {positionLimitViolations?.violations?.length || 0}
                    </div>
                    <p className="text-xs text-muted-foreground">Position Limits</p>
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>

          {/* Recent Activity */}
          <Card>
            <CardHeader>
              <CardTitle>Recent Validation Activity</CardTitle>
              <CardDescription>Latest financial agent validations</CardDescription>
            </CardHeader>
            <CardContent>
              {nrvTraces.length > 0 ? (
                <div className="space-y-3">
                  {nrvTraces.slice(0, 5).map((trace) => (
                    <div key={trace.id} className="flex items-center justify-between p-3 border rounded-lg">
                      <div className="flex items-center gap-3">
                        <Shield className="h-5 w-5 text-muted-foreground" />
                        <div>
                          <p className="font-medium">{trace.agent_id}</p>
                          <p className="text-sm text-muted-foreground">
                            {trace.reasoning_steps.length} reasoning steps
                          </p>
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <Clock className="h-4 w-4 text-muted-foreground" />
                        <span className="text-sm text-muted-foreground">
                          {new Date(trace.created_at).toLocaleString()}
                        </span>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="text-center py-8 text-muted-foreground">
                  No validation traces yet. Run a validation to capture traces.
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* NRV Traces Tab */}
        <TabsContent value="traces" className="space-y-4">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            <Card>
              <CardHeader>
                <CardTitle>NRV Traces</CardTitle>
                <CardDescription>Select a trace to analyze</CardDescription>
              </CardHeader>
              <CardContent>
                <ScrollArea className="h-[400px]">
                  {nrvTraces.map((trace) => (
                    <div
                      key={trace.id}
                      className={`p-3 mb-2 border rounded-lg cursor-pointer hover:bg-muted/50 transition-colors ${
                        selectedTraceId === trace.id ? 'border-primary bg-muted' : ''
                      }`}
                      onClick={() => handleTraceSelect(trace.id)}
                    >
                      <div className="flex items-center justify-between">
                        <div>
                          <p className="font-medium">{trace.agent_id}</p>
                          <p className="text-sm text-muted-foreground">
                            {trace.reasoning_steps.length} steps | {trace.financial_context?.transaction_type || 'N/A'}
                          </p>
                        </div>
                        <Clock className="h-4 w-4 text-muted-foreground" />
                      </div>
                    </div>
                  ))}
                  {nrvTraces.length === 0 && (
                    <div className="text-center py-8 text-muted-foreground">
                      No traces available
                    </div>
                  )}
                </ScrollArea>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Trace Details</CardTitle>
                <CardDescription>Selected trace reasoning analysis</CardDescription>
              </CardHeader>
              <CardContent>
                {selectedTrace ? (
                  <ScrollArea className="h-[400px]">
                    <div className="space-y-4">
                      <div className="grid grid-cols-2 gap-4">
                        <div>
                          <p className="text-sm text-muted-foreground">Agent ID</p>
                          <p className="font-medium">{selectedTrace.agent_id}</p>
                        </div>
                        <div>
                          <p className="text-sm text-muted-foreground">Session ID</p>
                          <p className="font-medium">{selectedTrace.session_id}</p>
                        </div>
                        <div>
                          <p className="text-sm text-muted-foreground">Transaction Type</p>
                          <p className="font-medium">{selectedTrace.financial_context?.transaction_type || 'N/A'}</p>
                        </div>
                        <div>
                          <p className="text-sm text-muted-foreground">Amount</p>
                          <p className="font-medium">
                            {selectedTrace.financial_context?.amount
                              ? `$${selectedTrace.financial_context.amount.toLocaleString()}`
                              : 'N/A'}
                          </p>
                        </div>
                      </div>

                      <Separator />

                      <div>
                        <p className="font-medium mb-2">Reasoning Steps</p>
                        {selectedTrace.reasoning_steps.map((step, idx) => (
                          <div key={idx} className="p-2 mb-2 border rounded text-sm">
                            <div className="flex items-center gap-2 mb-1">
                              <Badge variant="outline">Step {step.step_number}</Badge>
                              <span className="text-muted-foreground">{step.action}</span>
                            </div>
                            <p>{step.reasoning}</p>
                            {step.amount && (
                              <p className="text-xs text-muted-foreground mt-1">
                                Amount: ${step.amount.toLocaleString()} | Counterparty: {step.counterparty || 'N/A'}
                              </p>
                            )}
                          </div>
                        ))}
                      </div>
                    </div>
                  </ScrollArea>
                ) : (
                  <div className="text-center py-16 text-muted-foreground">
                    Select a trace to view details
                  </div>
                )}
              </CardContent>
            </Card>
          </div>

          {/* Analysis Actions */}
          {selectedTrace && (
            <Card>
              <CardHeader>
                <CardTitle>Trace Analysis</CardTitle>
                <CardDescription>Run analysis on selected trace</CardDescription>
              </CardHeader>
              <CardContent>
                <div className="flex flex-wrap gap-2">
                  <Button onClick={handleScoreFidelity} disabled={isLoading}>
                    <Brain className="w-4 h-4 mr-2" />
                    Score Fidelity
                  </Button>
                  <Button onClick={handleSemanticDistance} disabled={isLoading}>
                    <Scale className="w-4 h-4 mr-2" />
                    Calculate Distance
                  </Button>
                  <Button onClick={handleDetectKYCBypass} disabled={isLoading}>
                    <Users className="w-4 h-4 mr-2" />
                    Detect KYC Bypass
                  </Button>
                  <Button onClick={handleDetectPositionLimits} disabled={isLoading}>
                    <TrendingUp className="w-4 h-4 mr-2" />
                    Check Position Limits
                  </Button>
                </div>
              </CardContent>
            </Card>
          )}
        </TabsContent>

        {/* Fidelity Scores Tab */}
        <TabsContent value="fidelity" className="space-y-4">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            {/* Fidelity Score Card */}
            <Card>
              <CardHeader>
                <CardTitle>Fidelity Score</CardTitle>
                <CardDescription>Semantic alignment with regulatory outcomes</CardDescription>
              </CardHeader>
              <CardContent>
                {fidelityScore ? (
                  <div className="space-y-4">
                    <div className="text-center">
                      <div className={`text-5xl font-bold ${getScoreColor(fidelityScore.overall_score)}`}>
                        {(fidelityScore.overall_score * 100).toFixed(1)}%
                      </div>
                      {getRiskBadge(fidelityScore.risk_level)}
                    </div>

                    <Separator />

                    <div className="space-y-3">
                      <div>
                        <div className="flex justify-between text-sm mb-1">
                          <span>Intent Alignment</span>
                          <span>{(fidelityScore.component_scores.intent * 100).toFixed(1)}%</span>
                        </div>
                        <Progress value={fidelityScore.component_scores.intent * 100} />
                      </div>
                      <div>
                        <div className="flex justify-between text-sm mb-1">
                          <span>Action Compliance</span>
                          <span>{(fidelityScore.component_scores.action * 100).toFixed(1)}%</span>
                        </div>
                        <Progress value={fidelityScore.component_scores.action * 100} />
                      </div>
                      <div>
                        <div className="flex justify-between text-sm mb-1">
                          <span>Regulatory Compliance</span>
                          <span>{(fidelityScore.component_scores.compliance * 100).toFixed(1)}%</span>
                        </div>
                        <Progress value={fidelityScore.component_scores.compliance * 100} />
                      </div>
                      <div>
                        <div className="flex justify-between text-sm mb-1">
                          <span>Outcome Alignment</span>
                          <span>{(fidelityScore.component_scores.outcome * 100).toFixed(1)}%</span>
                        </div>
                        <Progress value={fidelityScore.component_scores.outcome * 100} />
                      </div>
                    </div>

                    {fidelityScore.violations.length > 0 && (
                      <>
                        <Separator />
                        <div>
                          <p className="font-medium mb-2">Violations Detected</p>
                          {fidelityScore.violations.map((violation, idx) => (
                            <Alert key={idx} variant={violation.severity === 'critical' ? 'destructive' : 'default'}>
                              <AlertCircle className="h-4 w-4" />
                              <AlertTitle>{violation.category}</AlertTitle>
                              <AlertDescription>{violation.description}</AlertDescription>
                            </Alert>
                          ))}
                        </div>
                      </>
                    )}

                    {fidelityScore.recommendations.length > 0 && (
                      <>
                        <Separator />
                        <div>
                          <p className="font-medium mb-2">Recommendations</p>
                          <ul className="list-disc list-inside text-sm space-y-1">
                            {fidelityScore.recommendations.map((rec, idx) => (
                              <li key={idx}>{rec}</li>
                            ))}
                          </ul>
                        </div>
                      </>
                    )}
                  </div>
                ) : (
                  <div className="text-center py-8 text-muted-foreground">
                    Select a trace and calculate fidelity score
                  </div>
                )}
              </CardContent>
            </Card>

            {/* Semantic Distance Card */}
            <Card>
              <CardHeader>
                <CardTitle>Semantic Distance</CardTitle>
                <CardDescription>Distance from regulatory alignment</CardDescription>
              </CardHeader>
              <CardContent>
                {semanticDistance ? (
                  <div className="space-y-4">
                    <div className="text-center">
                      <div className={`text-5xl font-bold ${semanticDistance.distance < 0.25 ? 'text-green-500' : semanticDistance.distance < 0.50 ? 'text-yellow-500' : 'text-red-500'}`}>
                        {semanticDistance.distance.toFixed(3)}
                      </div>
                      <p className="text-sm text-muted-foreground">Distance Score (0.0 = Perfect)</p>
                    </div>

                    <Separator />

                    <div className="space-y-3">
                      <div className="flex items-center justify-between">
                        <span>Intent Distance</span>
                        <Progress
                          value={semanticDistance.components.intent_distance * 100}
                          className="flex-1 ml-4"
                        />
                      </div>
                      <div className="flex items-center justify-between">
                        <span>Action Distance</span>
                        <Progress
                          value={semanticDistance.components.action_distance * 100}
                          className="flex-1 ml-4"
                        />
                      </div>
                      <div className="flex items-center justify-between">
                        <span>Compliance Distance</span>
                        <Progress
                          value={semanticDistance.components.compliance_distance * 100}
                          className="flex-1 ml-4"
                        />
                      </div>
                      <div className="flex items-center justify-between">
                        <span>Outcome Distance</span>
                        <Progress
                          value={semanticDistance.components.outcome_distance * 100}
                          className="flex-1 ml-4"
                        />
                      </div>
                    </div>

                    <Separator />

                    <div>
                      <p className="font-medium mb-2">Analysis</p>
                      <p className="text-sm text-muted-foreground">{semanticDistance.details}</p>
                    </div>
                  </div>
                ) : (
                  <div className="text-center py-8 text-muted-foreground">
                    Select a trace and calculate semantic distance
                  </div>
                )}
              </CardContent>
            </Card>
          </div>

          {/* Risk Detection Results */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            <Card>
              <CardHeader>
                <CardTitle>KYC Bypass Detection</CardTitle>
                <CardDescription>Detect attempts to bypass KYC requirements</CardDescription>
              </CardHeader>
              <CardContent>
                {kycBypassDetection ? (
                  <div className="space-y-4">
                    <div className="flex items-center gap-2">
                      {kycBypassDetection.detected ? (
                        <>
                          <XCircle className="h-8 w-8 text-red-500" />
                          <div>
                            <p className="text-lg font-bold text-red-500">KYC Bypass Detected</p>
                            <p className="text-sm text-muted-foreground">
                              Confidence: {(kycBypassDetection.confidence * 100).toFixed(1)}%
                            </p>
                          </div>
                        </>
                      ) : (
                        <>
                          <CheckCircle className="h-8 w-8 text-green-500" />
                          <div>
                            <p className="text-lg font-bold text-green-500">No KYC Bypass Detected</p>
                            <p className="text-sm text-muted-foreground">
                              Confidence: {(kycBypassDetection.confidence * 100).toFixed(1)}%
                            </p>
                          </div>
                        </>
                      )}
                    </div>

                    {kycBypassDetection.indicators.length > 0 && (
                      <>
                        <Separator />
                        <div>
                          <p className="font-medium mb-2">Indicators</p>
                          <div className="flex flex-wrap gap-2">
                            {kycBypassDetection.indicators.map((indicator, idx) => (
                              <Badge key={idx} variant="destructive">{indicator}</Badge>
                            ))}
                          </div>
                        </div>
                      </>
                    )}

                    <div>
                      <p className="font-medium mb-2">Evidence</p>
                      <p className="text-sm text-muted-foreground">{kycBypassDetection.evidence}</p>
                    </div>
                  </div>
                ) : (
                  <div className="text-center py-8 text-muted-foreground">
                    Run KYC bypass detection on a trace
                  </div>
                )}
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Position Limit Violations</CardTitle>
                <CardDescription>Monitor trading position limits</CardDescription>
              </CardHeader>
              <CardContent>
                {positionLimitViolations ? (
                  <div className="space-y-4">
                    <div className="flex items-center gap-2">
                      {positionLimitViolations.detected ? (
                        <>
                          <AlertTriangle className="h-8 w-8 text-orange-500" />
                          <div>
                            <p className="text-lg font-bold text-orange-500">Violations Detected</p>
                            <p className="text-sm text-muted-foreground">
                              {positionLimitViolations.violations.length} violation(s) found
                            </p>
                          </div>
                        </>
                      ) : (
                        <>
                          <CheckCircle className="h-8 w-8 text-green-500" />
                          <div>
                            <p className="text-lg font-bold text-green-500">All Positions Valid</p>
                            <p className="text-sm text-muted-foreground">
                              No limit violations detected
                            </p>
                          </div>
                        </>
                      )}
                    </div>

                    {positionLimitViolations.violations.length > 0 && (
                      <>
                        <Separator />
                        <div>
                          <p className="font-medium mb-2">Violations</p>
                          {positionLimitViolations.violations.map((violation, idx) => (
                            <Alert key={idx} variant={violation.severity === 'critical' ? 'destructive' : 'default'}>
                              <AlertTriangle className="h-4 w-4" />
                              <AlertTitle>{violation.instrument}</AlertTitle>
                              <AlertDescription>
                                Position: {violation.current_position} | Limit: {violation.limit} | Excess: {violation.excess}
                              </AlertDescription>
                            </Alert>
                          ))}
                        </div>
                      </>
                    )}

                    <div>
                      <p className="font-medium mb-2">Current Positions</p>
                      <div className="grid grid-cols-2 gap-2 text-sm">
                        {Object.entries(positionLimitViolations.current_positions || {}).map(([symbol, position]) => (
                          <div key={symbol} className="flex justify-between">
                            <span>{symbol}:</span>
                            <span className="font-medium">{position}</span>
                          </div>
                        ))}
                      </div>
                    </div>
                  </div>
                ) : (
                  <div className="text-center py-8 text-muted-foreground">
                    Run position limit check on a trace
                  </div>
                )}
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        {/* Evidence Packs Tab */}
        <TabsContent value="evidence" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Evidence Packs</CardTitle>
              <CardDescription>Compliance evidence and audit trails</CardDescription>
            </CardHeader>
            <CardContent>
              {evidencePacks.length > 0 ? (
                <div className="space-y-3">
                  {evidencePacks.map((pack) => (
                    <div key={pack.id} className="p-4 border rounded-lg">
                      <div className="flex items-center justify-between mb-2">
                        <div>
                          <p className="font-medium">{pack.name}</p>
                          <p className="text-sm text-muted-foreground">{pack.description}</p>
                        </div>
                        <div className="flex items-center gap-2">
                          <Button variant="outline" size="sm">
                            <Eye className="w-4 h-4 mr-1" />
                            View
                          </Button>
                          <Button variant="outline" size="sm">
                            <Download className="w-4 h-4 mr-1" />
                            Export
                          </Button>
                        </div>
                      </div>
                      <div className="flex gap-4 text-sm text-muted-foreground">
                        <span>{pack.context_records.length} context records</span>
                        <span>{pack.regulatory_checks.length} regulatory checks</span>
                        <span>{pack.certificates.length} certificates</span>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="text-center py-8 text-muted-foreground">
                  No evidence packs available yet
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Trajectories Tab */}
        <TabsContent value="trajectories" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Execution Trajectories</CardTitle>
              <CardDescription>Agent execution traces for deterministic replay</CardDescription>
            </CardHeader>
            <CardContent>
              {trajectories.length > 0 ? (
                <div className="space-y-3">
                  {trajectories.map((trajectory) => (
                    <div key={trajectory.id} className="p-4 border rounded-lg">
                      <div className="flex items-center justify-between mb-2">
                        <div>
                          <p className="font-medium">Agent: {trajectory.agent_id}</p>
                          <p className="text-sm text-muted-foreground">
                            Session: {trajectory.session_id} | {trajectory.steps.length} steps
                          </p>
                        </div>
                        <div className="flex items-center gap-2">
                          <Badge variant="outline">
                            {trajectory.duration_ms}ms
                          </Badge>
                          <Button variant="outline" size="sm">
                            <Play className="w-4 h-4 mr-1" />
                            Replay
                          </Button>
                        </div>
                      </div>
                      <div className="flex gap-4 text-sm text-muted-foreground">
                        <span>Captured: {new Date(trajectory.captured_at).toLocaleString()}</span>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="text-center py-8 text-muted-foreground">
                  No trajectories captured yet
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}

export default FinancialComplianceDashboard;
