'use client';

import React, { useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { useSystemHealth } from '@/hooks/use-system-health';
import { AlertTriangle, CheckCircle2, XCircle } from 'lucide-react';

type SeverityFilter = 'all' | 'critical' | 'high' | 'medium' | 'low';

export function NetworkMonitorAlerts() {
  const { alerts, isLoading, executeAction } = useSystemHealth();
  const [severityFilter, setSeverityFilter] = useState<SeverityFilter>('all');

  const filteredAlerts = alerts?.filter((alert) => {
    if (severityFilter === 'all') return true;
    return alert.severity === severityFilter;
  }) ?? [];

  const unresolvedAlerts = filteredAlerts.filter((a) => !a.resolved);
  const resolvedAlerts = filteredAlerts.filter((a) => a.resolved);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12 text-gray-500">
        Loading alerts...
      </div>
    );
  }

  const handleResolve = async (alertId: string) => {
    await executeAction({ action: 'resolve_alert', parameters: { alert_id: alertId } });
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-semibold text-gray-200">Alerts</h3>
          <p className="text-sm text-gray-500">
            Unified alert feed from all KNIRV services
          </p>
        </div>
        <div className="flex gap-2">
          {(['all', 'critical', 'high', 'medium', 'low'] as SeverityFilter[]).map((severity) => (
            <Button
              key={severity}
              size="sm"
              variant={severityFilter === severity ? 'default' : 'outline'}
              className={
                severityFilter === severity
                  ? 'bg-cyan-500/20 text-cyan-400 border-cyan-500/30'
                  : 'border-gray-700 text-gray-400 hover:bg-gray-800'
              }
              onClick={() => setSeverityFilter(severity)}
            >
              {severity === 'all' ? `All (${alerts?.length ?? 0})` : severity}
            </Button>
          ))}
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4">
        <Card className="aether-bevel-dark rounded-xl">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm text-gray-300 flex items-center gap-2">
              <AlertTriangle className="w-4 h-4 text-red-400" />
              Unresolved ({unresolvedAlerts.length})
            </CardTitle>
          </CardHeader>
          <CardContent>
            {unresolvedAlerts.length === 0 ? (
              <p className="text-sm text-gray-500">No unresolved alerts.</p>
            ) : (
              <div className="space-y-2">
                {unresolvedAlerts.map((alert) => (
                  <div
                    key={alert.id}
                    className="flex items-center justify-between rounded-lg border border-slate-800 bg-slate-950/40 p-3"
                  >
                    <div className="flex items-center gap-3">
                      <Badge
                        variant="outline"
                        className={
                          alert.severity === 'critical'
                            ? 'border-red-500/30 text-red-300'
                            : alert.severity === 'high'
                              ? 'border-amber-500/30 text-amber-300'
                              : alert.severity === 'medium'
                                ? 'border-blue-500/30 text-blue-300'
                                : 'border-gray-500/30 text-gray-300'
                        }
                      >
                        {alert.severity}
                      </Badge>
                      <div>
                        <div className="text-sm text-gray-200">{alert.message}</div>
                        <div className="text-xs text-gray-500">
                          {alert.component} | {new Date(alert.timestamp).toLocaleString()}
                        </div>
                      </div>
                    </div>
                    <Button
                      size="sm"
                      variant="outline"
                      className="border-green-700/50 text-green-400 hover:bg-green-600/20"
                      onClick={() => handleResolve(alert.id)}
                    >
                      <CheckCircle2 className="w-4 h-4 mr-1" />
                      Resolve
                    </Button>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        {resolvedAlerts.length > 0 && (
          <Card className="aether-bevel-dark rounded-xl">
            <CardHeader className="pb-2">
              <CardTitle className="text-sm text-gray-300 flex items-center gap-2">
                <CheckCircle2 className="w-4 h-4 text-green-400" />
                Recently Resolved ({resolvedAlerts.length})
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-2">
                {resolvedAlerts.slice(0, 10).map((alert) => (
                  <div
                    key={alert.id}
                    className="flex items-center justify-between rounded-lg border border-slate-800 bg-slate-950/40 p-3 opacity-60"
                  >
                    <div className="flex items-center gap-3">
                      <XCircle className="w-4 h-4 text-gray-500" />
                      <div>
                        <div className="text-sm text-gray-400 line-through">{alert.message}</div>
                        <div className="text-xs text-gray-600">
                          {alert.component} | Resolved: {alert.resolved_at ? new Date(alert.resolved_at).toLocaleString() : 'unknown'}
                        </div>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  );
}
