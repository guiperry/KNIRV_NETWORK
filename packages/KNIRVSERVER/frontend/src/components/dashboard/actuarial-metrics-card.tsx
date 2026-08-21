'use client';

import { useQuery } from '@tanstack/react-query';
import { Activity, CircleDollarSign, ShieldAlert } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';

type ActuarialMetrics = { enabled: number; paused: number; risk_classes_active: number; risk_classes_observation_only: number; reports_total: number; snapshots_finalized: number; pools_active: number; pools_capacity_restricted: number; liquid_balance: number; reserved_balance: number; settlements_pending: number; settlements_failed: number; outbox_pending: number };

async function loadMetrics(): Promise<ActuarialMetrics> {
  const response = await fetch('/api/v1/actuarial/metrics');
  if (!response.ok) throw new Error(`Actuarial metrics unavailable (${response.status})`);
  const payload = await response.json() as { metrics?: ActuarialMetrics };
  if (!payload.metrics) throw new Error('Actuarial metrics response is malformed');
  return payload.metrics;
}

export function ActuarialMetricsCard() {
  const query = useQuery({ queryKey: ['actuarial', 'metrics'], queryFn: loadMetrics, refetchInterval: 15_000, staleTime: 10_000, retry: 1 });
  const metrics = query.data;
  const available = metrics ? metrics.liquid_balance - metrics.reserved_balance : 0;

  return <Card className="aether-bevel-dark rounded-2xl"><CardHeader className="pb-2"><CardTitle className="flex items-center gap-2 text-sm text-gray-200"><CircleDollarSign className="h-4 w-4 text-cyan-400" />Actuarial Syndicate</CardTitle><CardDescription className="text-xs">Aggregate testnet pool health; no private telemetry</CardDescription></CardHeader><CardContent>{query.isLoading ? <div className="text-xs text-gray-500">Loading syndicate metrics…</div> : query.error || !metrics ? <div className="flex items-center gap-2 text-xs text-gray-500"><Activity className="h-3.5 w-3.5" />Syndicate metrics unavailable</div> : <div className="space-y-2 text-xs"><div className="flex items-center justify-between"><span className="text-gray-400">Status</span><span className={metrics.enabled && !metrics.paused ? 'text-emerald-400' : 'text-amber-400'}>{metrics.enabled ? metrics.paused ? 'Paused' : 'Observation / active' : 'Disabled'}</span></div><div className="grid grid-cols-2 gap-2 text-gray-400"><span>Active pools <b className="text-gray-200">{metrics.pools_active}</b></span><span>Restricted <b className="text-gray-200">{metrics.pools_capacity_restricted}</b></span><span>Available <b className="text-gray-200">{available.toLocaleString()}</b></span><span>Reserved <b className="text-gray-200">{metrics.reserved_balance.toLocaleString()}</b></span><span>Snapshots <b className="text-gray-200">{metrics.snapshots_finalized}</b></span><span>Reports <b className="text-gray-200">{metrics.reports_total}</b></span></div>{(metrics.settlements_failed > 0 || metrics.outbox_pending > 0) ? <div className="flex items-center gap-1 text-amber-400"><ShieldAlert className="h-3.5 w-3.5" />{metrics.settlements_failed} failed settlements · {metrics.outbox_pending} queued effects</div> : null}</div>}</CardContent></Card>;
}
