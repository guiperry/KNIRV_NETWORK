'use client';

import React from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { useSystemHealth } from '@/hooks/use-system-health';
import { useNetworkMonitor } from '@/hooks/use-network-monitor';
import { Activity, Server, Wifi, AlertTriangle } from 'lucide-react';

export function NetworkMonitorHealth() {
  const { systemHealth, isLoading: healthLoading, error: healthError } = useSystemHealth();
  const { status, isLoading: statusLoading } = useNetworkMonitor();

  const isLoading = healthLoading || statusLoading;
  const error = healthError;

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12 text-gray-500">
        <Activity className="w-5 h-5 mr-2 animate-spin" />
        Loading health data...
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-lg border border-amber-500/20 bg-amber-500/10 p-4 text-amber-300">
        Failed to load health data: {error}
      </div>
    );
  }

  const overallStatus = systemHealth?.overall_status ?? status?.overall_status ?? 'unknown';
  const statusBadgeClass = overallStatus === 'healthy'
    ? 'bg-green-500/10 text-green-300 border-green-500/30'
    : overallStatus === 'degraded'
      ? 'bg-amber-500/10 text-amber-300 border-amber-500/30'
      : overallStatus === 'critical'
        ? 'bg-red-500/10 text-red-300 border-red-500/30'
        : 'bg-gray-500/10 text-gray-300 border-gray-500/30';

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-semibold text-gray-200">System Health</h3>
          <p className="text-sm text-gray-500">
            Host metrics and service status overview
          </p>
        </div>
        <Badge variant="outline" className={`${statusBadgeClass} text-xs uppercase`}>
          {overallStatus}
        </Badge>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <Card className="aether-bevel-dark rounded-xl">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm text-gray-300 flex items-center gap-2">
              <Activity className="w-4 h-4 text-blue-400" />
              CPU Usage
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-gray-200">
              {(systemHealth?.metrics?.cpu_usage ?? 0).toFixed(1)}%
            </div>
            <p className="text-xs text-gray-500">Host CPU utilization</p>
          </CardContent>
        </Card>

        <Card className="aether-bevel-dark rounded-xl">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm text-gray-300 flex items-center gap-2">
              <Activity className="w-4 h-4 text-green-400" />
              Memory Usage
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-gray-200">
              {(systemHealth?.metrics?.memory_usage ?? 0).toFixed(1)}%
            </div>
            <p className="text-xs text-gray-500">Host memory utilization</p>
          </CardContent>
        </Card>

        <Card className="aether-bevel-dark rounded-xl">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm text-gray-300 flex items-center gap-2">
              <Server className="w-4 h-4 text-amber-400" />
              Disk Usage
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-gray-200">
              {(systemHealth?.metrics?.disk_usage ?? 0).toFixed(1)}%
            </div>
            <p className="text-xs text-gray-500">Host disk utilization</p>
          </CardContent>
        </Card>

        <Card className="aether-bevel-dark rounded-xl">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm text-gray-300 flex items-center gap-2">
              <Wifi className="w-4 h-4 text-cyan-400" />
              Services
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <span className="text-2xl font-bold text-green-400">
                {status?.services_up ?? 0}
              </span>
              <span className="text-sm text-gray-500">/</span>
              <span className="text-2xl font-bold text-gray-200">
                {status?.services_total ?? 0}
              </span>
            </div>
            <p className="text-xs text-gray-500">Services up / total</p>
          </CardContent>
        </Card>
      </div>

      {systemHealth?.alerts && systemHealth.alerts.length > 0 && (
        <Card className="aether-bevel-dark rounded-xl">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm text-gray-300 flex items-center gap-2">
              <AlertTriangle className="w-4 h-4 text-red-400" />
              Active Alerts ({systemHealth.alerts.filter((a) => !a.resolved).length})
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              {systemHealth.alerts.filter((a) => !a.resolved).slice(0, 5).map((alert) => (
                <div
                  key={alert.id}
                  className="flex items-center justify-between rounded-lg border border-slate-800 bg-slate-950/40 p-2 text-xs"
                >
                  <div className="flex items-center gap-2">
                    <Badge
                      variant="outline"
                      className={
                        alert.severity === 'critical'
                          ? 'border-red-500/30 text-red-300'
                          : alert.severity === 'high'
                            ? 'border-amber-500/30 text-amber-300'
                            : 'border-blue-500/30 text-blue-300'
                      }
                    >
                      {alert.severity}
                    </Badge>
                    <span className="text-gray-300">{alert.message}</span>
                  </div>
                  <span className="text-gray-500">{alert.component}</span>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
