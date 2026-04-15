'use client';

import React from 'react';
import { MemoryStick, Network, Activity, RefreshCw, Server, Clock, Gauge } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { useTelemetry } from '@/hooks/use-telemetry';

interface SystemTelemetryCardProps {
  className?: string;
}

export function SystemTelemetryCard({ className }: SystemTelemetryCardProps) {
  const { telemetry, refetch } = useTelemetry();

  const formatPressure = (value: number): string => {
    if (value === 0) return 'Normal';
    if (value < 30) return 'Low';
    if (value < 70) return 'Medium';
    return 'High';
  };

  const pressureColor = (value: number): string => {
    if (value === 0) return 'text-green-400';
    if (value < 30) return 'text-yellow-400';
    if (value < 70) return 'text-orange-400';
    return 'text-red-400';
  };

  return (
    <Card className={className}>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Server className="w-5 h-5 text-cyan-400" />
            <CardTitle>System Telemetry</CardTitle>
          </div>
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            <RefreshCw className="w-4 h-4 mr-1" />
            Refresh
          </Button>
        </div>
        <CardDescription>eBPF-based system resource monitoring</CardDescription>
      </CardHeader>
      <CardContent>
        {telemetry.isLoading && (
          <div className="flex items-center justify-center py-8">
            <RefreshCw className="w-6 h-6 animate-spin text-muted-foreground" />
          </div>
        )}

        {telemetry.error && (
          <div className="flex items-center justify-center py-8 text-red-400">
            Failed to load telemetry
          </div>
        )}

        {telemetry.data && (
          <div className="space-y-4">
            {/* Timestamp */}
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              <Clock className="w-3 h-3" />
              Last updated: {new Date(telemetry.data.timestamp).toLocaleTimeString()}
            </div>

            {/* CPU Usage */}
            <div className="flex items-center justify-between p-3 rounded-lg bg-card/50 border border-border">
              <div className="flex items-center gap-3">
                <Gauge className="w-5 h-5 text-blue-400" />
                <div>
                  <div className="text-sm font-medium">CPU Usage</div>
                  <div className="text-xs text-muted-foreground">Current host utilization</div>
                </div>
              </div>
              <div className="text-right">
                <div className="text-lg font-bold">{telemetry.data.cpu_usage.toFixed(1)}%</div>
              </div>
            </div>

            {/* Memory */}
            <div className="flex items-center justify-between p-3 rounded-lg bg-card/50 border border-border">
              <div className="flex items-center gap-3">
                <MemoryStick className="w-5 h-5 text-purple-400" />
                <div>
                  <div className="text-sm font-medium">Memory</div>
                  <div className="text-xs text-muted-foreground">Total allocated</div>
                </div>
              </div>
              <div className="text-right">
                <div className="text-lg font-bold">{telemetry.data.memory_usage.toFixed(1)}%</div>
                <div className="text-xs text-muted-foreground">System memory in use</div>
              </div>
            </div>

            {/* Network */}
            <div className="grid grid-cols-2 gap-3">
              <div className="flex items-center justify-between p-3 rounded-lg bg-card/50 border border-border">
                <div className="flex items-center gap-2">
                  <Network className="w-4 h-4 text-green-400" />
                  <div>
                    <div className="text-xs text-muted-foreground">Throughput</div>
                    <div className="text-sm font-medium">{telemetry.data.network_throughput.toFixed(1)}</div>
                  </div>
                </div>
              </div>
              <div className="flex items-center justify-between p-3 rounded-lg bg-card/50 border border-border">
                <div className="flex items-center gap-2">
                  <Network className="w-4 h-4 text-yellow-400" />
                  <div>
                    <div className="text-xs text-muted-foreground">Connections</div>
                    <div className="text-sm font-medium">{telemetry.data.active_connections}</div>
                  </div>
                </div>
              </div>
            </div>

            {/* System Metrics */}
            <div className="grid grid-cols-2 gap-3">
              <div className="p-3 rounded-lg bg-card/50 border border-border">
                <div className="flex items-center gap-2 mb-1">
                  <Activity className="w-4 h-4 text-orange-400" />
                  <span className="text-xs text-muted-foreground">CPU Pressure</span>
                </div>
                <div className="text-lg font-bold">{telemetry.data.cpu_pressure.toFixed(1)}%</div>
              </div>
              <div className="p-3 rounded-lg bg-card/50 border border-border">
                <div className="flex items-center gap-2 mb-1">
                  <Activity className="w-4 h-4 text-red-400" />
                  <span className="text-xs text-muted-foreground">Memory Pressure</span>
                </div>
                <div className="text-lg font-bold">{telemetry.data.memory_pressure.toFixed(1)}%</div>
              </div>
            </div>

            {/* Pressure */}
            <div className="grid grid-cols-2 gap-3">
              <div className="p-3 rounded-lg bg-card/50 border border-border">
                <div className="flex items-center justify-between mb-1">
                  <span className="text-xs text-muted-foreground">CPU Pressure</span>
                  <span className={`text-sm font-medium ${pressureColor(telemetry.data.cpu_pressure)}`}>
                    {formatPressure(telemetry.data.cpu_pressure)}
                  </span>
                </div>
                <div className="w-full h-2 bg-muted rounded-full overflow-hidden">
                  <div 
                    className="h-full bg-gradient-to-r from-green-400 to-red-400 transition-all"
                    style={{ width: `${Math.min(telemetry.data.cpu_pressure, 100)}%` }}
                  />
                </div>
              </div>
              <div className="p-3 rounded-lg bg-card/50 border border-border">
                <div className="flex items-center justify-between mb-1">
                  <span className="text-xs text-muted-foreground">Memory Pressure</span>
                  <span className={`text-sm font-medium ${pressureColor(telemetry.data.memory_pressure)}`}>
                    {formatPressure(telemetry.data.memory_pressure)}
                  </span>
                </div>
                <div className="w-full h-2 bg-muted rounded-full overflow-hidden">
                  <div 
                    className="h-full bg-gradient-to-r from-green-400 to-red-400 transition-all"
                    style={{ width: `${Math.min(telemetry.data.memory_pressure, 100)}%` }}
                  />
                </div>
              </div>
            </div>

            {/* Runtime */}
            <div className="flex items-center justify-between text-xs text-muted-foreground pt-2 border-t">
              <span>Goroutines: {telemetry.data.goroutines}</span>
              <span>Source: System Health</span>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

export default SystemTelemetryCard;
