"use client";

import type React from 'react';
import { Activity, Cpu, Network, Shield, TerminalSquare } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { useSecuritySubsystem, getSecurityBackendLabel } from '@/hooks/use-security-subsystem';

interface KernelSecurityCardProps {
  className?: string;
}

export function KernelSecurityCard({ className }: KernelSecurityCardProps) {
  const { securitySubsystem, isLoading } = useSecuritySubsystem();
  const backendLabel = getSecurityBackendLabel(securitySubsystem);
  const isFallback = securitySubsystem.backend === 'native';
  const isActive = securitySubsystem.status !== 'unavailable';
  const badgeClass = isFallback
    ? 'bg-amber-500/10 text-amber-300 border-amber-500/30'
    : isActive
      ? 'bg-green-500/10 text-green-300 border-green-500/30'
      : 'bg-red-500/10 text-red-300 border-red-500/30';

  return (
    <Card className={`aether-bevel-dark rounded-2xl ${className ?? ''}`}>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between gap-3">
          <CardTitle className="flex items-center gap-2 text-sm text-gray-300">
            <Shield className="w-4 h-4 text-cyan-400" />
            Kernel Security
          </CardTitle>
          <Badge variant="outline" className={`${badgeClass} text-[10px] uppercase`}>
            {isLoading ? 'Checking' : backendLabel}
          </Badge>
        </div>
        <CardDescription className="text-xs text-gray-500">
          Policy enforcement, rate limiting, container isolation, and syscall tracing
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="grid grid-cols-2 gap-2 text-[11px]">
          <StatusMetric icon={<Cpu className="w-3 h-3" />} label="Programs" value={securitySubsystem.manager?.programs_attached ?? 0} />
          <StatusMetric icon={<Network className="w-3 h-3" />} label="Containers" value={securitySubsystem.isolator?.active_containers ?? 0} />
          <StatusMetric icon={<Activity className="w-3 h-3" />} label="Allowed" value={securitySubsystem.rate_limiter?.allowed_packets ?? 0} />
          <StatusMetric icon={<TerminalSquare className="w-3 h-3" />} label="Syscalls" value={securitySubsystem.tracer?.syscall_events ?? 0} />
        </div>
        <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-[10px] uppercase tracking-wide text-gray-500">
          <Capability label="Policy" enabled={securitySubsystem.capabilities?.policy_enforcer} />
          <Capability label="Limiter" enabled={securitySubsystem.capabilities?.rate_limiter} />
          <Capability label="Isolator" enabled={securitySubsystem.capabilities?.container_isolator} />
          <Capability label="Tracer" enabled={securitySubsystem.capabilities?.syscall_tracer} />
        </div>
        <div className="rounded-lg border border-slate-800 bg-black/30 p-2 text-[10px] text-slate-400">
          <div className="flex justify-between gap-3">
            <span>Guardian</span>
            <span className={securitySubsystem.guardian?.running ? 'text-green-400' : 'text-amber-400'}>
              {securitySubsystem.guardian?.running ? 'running' : 'skipped'}
            </span>
          </div>
          <div className="mt-1 line-clamp-2 text-slate-500">
            {securitySubsystem.fallback_reason || securitySubsystem.guardian?.reason || 'No fallback reason reported'}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

function StatusMetric({ icon, label, value }: { icon: React.ReactNode; label: string; value: number }) {
  return (
    <div className="rounded-lg border border-slate-800 bg-slate-950/40 p-2">
      <div className="flex items-center gap-1 text-slate-500">
        {icon}
        <span>{label}</span>
      </div>
      <div className="mt-1 font-mono text-sm font-bold text-slate-200">{value.toLocaleString()}</div>
    </div>
  );
}

function Capability({ label, enabled }: { label: string; enabled?: boolean }) {
  return (
    <div className="flex items-center justify-between">
      <span>{label}</span>
      <span className={enabled ? 'text-green-400' : 'text-red-400'}>{enabled ? 'on' : 'off'}</span>
    </div>
  );
}

export default KernelSecurityCard;
