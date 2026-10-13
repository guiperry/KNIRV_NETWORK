"use client";

import { Cpu } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { useModelsStatus } from '@/hooks/use-models-status';
import { roles, type RoleState } from '@/lib/models-status';

const STATE_CLASS: Record<RoleState, string> = {
  ready: 'border-emerald-500/30 text-emerald-300',
  idle: 'border-gray-500/30 text-gray-300',
  'not ready': 'border-amber-500/30 text-amber-300',
  unavailable: 'border-red-500/30 text-red-300',
};

export function ModelsStatusCard({ className }: { className?: string }) {
  const { data, error, isLoading } = useModelsStatus();
  return <Card className={className ?? 'aether-bevel-dark rounded-2xl'} data-testid="models-status-card">
    <CardHeader className="pb-3">
      <CardTitle className="flex items-center gap-2 text-sm text-gray-300"><Cpu className="w-4 h-4 text-violet-400" />Models</CardTitle>
      <CardDescription className="text-xs text-gray-500">Which model does what, and how recent requests were routed</CardDescription>
    </CardHeader>
    <CardContent className="space-y-3">
      {isLoading && <div className="text-xs text-gray-500">Reading model status…</div>}
      {error && !data && <div className="text-xs text-red-300" role="alert">{error instanceof Error ? error.message : 'Model status unavailable'}</div>}
      {data && <>
        <div className="space-y-1.5">
          {roles(data).map((r) => <div key={r.role} className="flex items-start justify-between gap-3" data-testid={`models-role-${r.role}`}>
            <div className="min-w-0">
              <div className="text-xs text-gray-300">{r.label}</div>
              <div className="text-[10px] text-gray-500 truncate">{r.model}{r.version ? ` · ${r.version}` : ''}{r.detail ? ` · ${r.detail}` : ''}</div>
            </div>
            <Badge variant="outline" className={STATE_CLASS[r.state]}>{r.state}</Badge>
          </div>)}
        </div>
        <div>
          <div className="text-[10px] uppercase tracking-wide text-gray-500 mb-1">Recent routing</div>
          {data.routing.length === 0 && <div className="text-[10px] text-gray-600">None since the server started</div>}
          {data.routing.slice(0, 5).map((ev) => <div key={ev.id} className="text-[10px] text-gray-400" data-testid="models-routing-event">
            {ev.kind} {ev.error ? <span className="text-red-300">failed: {ev.error}</span> : <>→ {ev.chosen}{ev.adapter ? ` +${ev.adapter}` : ''}</>}
            {ev.fallbacks && ev.fallbacks.length > 0 && <span className="text-amber-300"> after failing: {ev.fallbacks.join(', ')}</span>}
            {ev.degraded && <span className="text-red-300"> DEGRADED: skill did not run</span>}
          </div>)}
        </div>
      </>}
    </CardContent>
  </Card>;
}
