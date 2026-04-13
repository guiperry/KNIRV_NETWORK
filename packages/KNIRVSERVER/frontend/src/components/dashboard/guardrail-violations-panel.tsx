'use client';

import React, { useState } from 'react';
import { AlertTriangle, CheckCircle, Clock, Shield, RefreshCw, XCircle } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { ScrollArea } from '@/components/ui/scroll-area';
import { useGuardrail, GuardrailViolation, GuardrailStatistics } from '@/hooks/use-guardrail';

const severityColors: Record<string, string> = {
  low: 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30',
  medium: 'bg-orange-500/20 text-orange-400 border-orange-500/30',
  high: 'bg-red-500/20 text-red-400 border-red-500/30',
  critical: 'bg-purple-500/20 text-purple-400 border-purple-500/30',
};

interface GuardrailViolationsPanelProps {
  className?: string;
  nodeId?: string;
}

export function GuardrailViolationsPanel({ className, nodeId }: GuardrailViolationsPanelProps) {
  const [selectedViolation, setSelectedViolation] = useState<GuardrailViolation | null>(null);
  
  const { 
    violations, 
    statistics, 
    resolveViolation, 
    refetchViolations, 
    refetchStatistics 
  } = useGuardrail();

  const handleResolve = async (violationId: string) => {
    try {
      await resolveViolation(violationId);
      setSelectedViolation(null);
    } catch (error) {
      console.error('Failed to resolve violation:', error);
    }
  };

  const handleRefresh = () => {
    refetchViolations();
    refetchStatistics();
  };

  const formatTimestamp = (timestamp: string) => {
    const date = new Date(timestamp);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMins / 60);
    const diffDays = Math.floor(diffHours / 24);

    if (diffMins < 1) return 'Just now';
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    return `${diffDays}d ago`;
  };

  return (
    <Card className={className}>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Shield className="w-5 h-5 text-red-400" />
            <CardTitle>Guardrail Violations</CardTitle>
          </div>
          <Button variant="outline" size="sm" onClick={handleRefresh}>
            <RefreshCw className="w-4 h-4 mr-1" />
            Refresh
          </Button>
        </div>
        {statistics.data && (
          <CardDescription className="flex items-center gap-4 mt-2">
            <span className="flex items-center gap-1">
              <AlertTriangle className="w-3 h-3 text-red-400" />
              {statistics.data.active_violations} active
            </span>
            <span className="flex items-center gap-1">
              <CheckCircle className="w-3 h-3 text-green-400" />
              {statistics.data.resolved_violations} resolved
            </span>
          </CardDescription>
        )}
      </CardHeader>
      <CardContent>
        {violations.isLoading && (
          <div className="flex items-center justify-center py-8">
            <RefreshCw className="w-6 h-6 animate-spin text-muted-foreground" />
          </div>
        )}

        {violations.error && (
          <div className="flex items-center justify-center py-8 text-red-400">
            <XCircle className="w-5 h-5 mr-2" />
            Failed to load violations
          </div>
        )}

        {violations.data && violations.data.length === 0 && (
          <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
            <CheckCircle className="w-10 h-10 text-green-400 mb-2" />
            <p>No violations detected</p>
          </div>
        )}

        {violations.data && violations.data.length > 0 && (
          <ScrollArea className="h-[300px]">
            <div className="space-y-2">
              {violations.data.map((violation) => (
                <div
                  key={violation.id}
                  className={`p-3 rounded-lg border transition-colors cursor-pointer ${
                    selectedViolation?.id === violation.id
                      ? 'bg-red-500/10 border-red-500/30'
                      : 'bg-card hover:bg-card/80 border-border'
                  }`}
                  onClick={() => setSelectedViolation(violation)}
                >
                  <div className="flex items-start justify-between gap-2">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-1">
                        <Badge className={severityColors[violation.severity]}>
                          {violation.severity}
                        </Badge>
                        <span className="text-xs text-muted-foreground">
                          {violation.violation_type}
                        </span>
                      </div>
                      <p className="text-sm font-medium truncate">{violation.details}</p>
                      <div className="flex items-center gap-2 mt-1 text-xs text-muted-foreground">
                        <span>Node: {violation.node_id}</span>
                        <span>•</span>
                        <span className="flex items-center gap-1">
                          <Clock className="w-3 h-3" />
                          {formatTimestamp(violation.timestamp)}
                        </span>
                      </div>
                    </div>
                    {selectedViolation?.id === violation.id && !violation.resolved && (
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={(e) => {
                          e.stopPropagation();
                          handleResolve(violation.id);
                        }}
                      >
                        Resolve
                      </Button>
                    )}
                  </div>
                  {violation.resolved && (
                    <div className="flex items-center gap-1 mt-2 text-xs text-green-400">
                      <CheckCircle className="w-3 h-3" />
                      Resolved {violation.resolved_at && formatTimestamp(violation.resolved_at)}
                    </div>
                  )}
                </div>
              ))}
            </div>
          </ScrollArea>
        )}
      </CardContent>
    </Card>
  );
}

interface GuardrailStatisticsCardProps {
  className?: string;
}

export function GuardrailStatisticsCard({ className }: GuardrailStatisticsCardProps) {
  const { statistics, refetchStatistics } = useGuardrail();

  return (
    <Card className={className}>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Shield className="w-5 h-5 text-blue-400" />
            <CardTitle>Guardrail Statistics</CardTitle>
          </div>
          <Button variant="ghost" size="sm" onClick={() => refetchStatistics()}>
            <RefreshCw className="w-4 h-4" />
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        {statistics.isLoading && (
          <div className="flex items-center justify-center py-4">
            <RefreshCw className="w-5 h-5 animate-spin text-muted-foreground" />
          </div>
        )}

        {statistics.data && (
          <div className="grid grid-cols-3 gap-4">
            <div className="text-center p-3 rounded-lg bg-card/50">
              <div className="text-2xl font-bold">{statistics.data.total_violations}</div>
              <div className="text-xs text-muted-foreground">Total</div>
            </div>
            <div className="text-center p-3 rounded-lg bg-red-500/10">
              <div className="text-2xl font-bold text-red-400">{statistics.data.active_violations}</div>
              <div className="text-xs text-muted-foreground">Active</div>
            </div>
            <div className="text-center p-3 rounded-lg bg-green-500/10">
              <div className="text-2xl font-bold text-green-400">{statistics.data.resolved_violations}</div>
              <div className="text-xs text-muted-foreground">Resolved</div>
            </div>
          </div>
        )}

        {statistics.data?.policies && statistics.data.policies.length > 0 && (
          <div className="mt-4">
            <h4 className="text-sm font-medium mb-2">Policy Status</h4>
            <div className="space-y-2">
              {statistics.data.policies.slice(0, 5).map((policy) => (
                <div key={policy.id} className="flex items-center justify-between text-sm">
                  <div className="flex items-center gap-2">
                    <span className="truncate max-w-[150px]">{policy.name}</span>
                    <Badge variant={policy.enabled ? 'default' : 'secondary'}>
                      {policy.enabled ? 'Active' : 'Disabled'}
                    </Badge>
                  </div>
                  <span className="text-muted-foreground">
                    {policy.violations_count} violations
                  </span>
                </div>
              ))}
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

export default GuardrailViolationsPanel;
