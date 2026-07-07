"use client";

import { Activity, AlertTriangle, Cpu, Network } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { useNetworkMonitor } from '@/hooks/use-network-monitor';

interface NetworkMonitorCardProps {
  className?: string;
}

export function NetworkMonitorCard({ className }: NetworkMonitorCardProps) {
  const { status, metrics, isLoading, error } = useNetworkMonitor();
  const health = status?.overall_status ?? (error ? 'unavailable' : 'checking');
  const badgeClass = health === 'healthy'
    ? 'bg-green-500/10 text-green-300 border-green-500/30'
    : health === 'checking'
      ? 'bg-blue-500/10 text-blue-300 border-blue-500/30'
      : 'bg-amber-500/10 text-amber-300 border-amber-500/30';

  return (
    <Card className={`aether-bevel-dark rounded-2xl ${className ?? ''}`}>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between gap-3">
          <CardTitle className="flex items-center gap-2 text-sm text-gray-300">
            <Network className="w-4 h-4 text-blue-400" />
            Network Monitor
          </CardTitle>
          <Badge variant="outline" className={`${badgeClass} text-[10px] uppercase`}>
            {isLoading ? 'Checking' : health}
          </Badge>
        </div>
        <CardDescription className="text-xs text-gray-500">
          Gateway-proxied monitor status, services, and infrastructure metrics
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="grid grid-cols-3 gap-2 text-[11px]">
          <Metric label="Up" value={status?.services_up ?? 0} tone="text-green-400" />
          <Metric label="Down" value={status?.services_down ?? 0} tone="text-red-400" />
          <Metric label="Total" value={status?.services_total ?? 0} tone="text-slate-200" />
        </div>
        <div className="grid grid-cols-2 gap-2 text-[11px]">
          <div className="rounded-lg border border-slate-800 bg-slate-950/40 p-2">
            <div className="flex items-center gap-1 text-slate-500">
              <Cpu className="w-3 h-3" />
              <span>CPU</span>
            </div>
            <div className="mt-1 font-mono text-sm font-bold text-slate-200">
              {(metrics?.cpu?.usage_percent ?? 0).toFixed(1)}%
            </div>
          </div>
          <div className="rounded-lg border border-slate-800 bg-slate-950/40 p-2">
            <div className="flex items-center gap-1 text-slate-500">
              <Activity className="w-3 h-3" />
              <span>Memory</span>
            </div>
            <div className="mt-1 font-mono text-sm font-bold text-slate-200">
              {(metrics?.memory?.usage_percent ?? 0).toFixed(1)}%
            </div>
          </div>
        </div>
        {error && (
          <div className="flex items-center gap-2 rounded-lg border border-amber-500/20 bg-amber-500/10 p-2 text-[10px] text-amber-300">
            <AlertTriangle className="w-3 h-3" />
            Network monitor proxy unavailable
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function Metric({ label, value, tone }: { label: string; value: number; tone: string }) {
  return (
    <div className="rounded-lg border border-slate-800 bg-slate-950/40 p-2">
      <div className="text-slate-500">{label}</div>
      <div className={`mt-1 font-mono text-sm font-bold ${tone}`}>{value.toLocaleString()}</div>
    </div>
  );
}

export default NetworkMonitorCard;
