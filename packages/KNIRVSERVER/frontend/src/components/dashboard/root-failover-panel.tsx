"use client";

import { Clock, Crown, RefreshCw, ShieldAlert, Users } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { useRootFailover } from '@/hooks/use-root-failover';

function Tile({ label, value }: { label: string; value: string | number }) { return <Card className="aether-bevel-dark rounded-xl"><CardHeader className="pb-2"><CardTitle className="text-xs text-gray-400">{label}</CardTitle></CardHeader><CardContent><div className="break-all font-mono text-sm text-gray-200">{value}</div></CardContent></Card>; }
export function RootFailoverPanel() {
  const { state, isLoading, error, refetch } = useRootFailover();
  if (isLoading) return <div className="py-12 text-center text-gray-500">Loading attested failover state…</div>;
  if (error || !state) return <div className="rounded-lg border border-amber-500/20 bg-amber-500/10 p-4 text-amber-300">Unable to load root failover state: {error ?? 'unknown error'}</div>;
  const consensus = state.consensus;
  return <div className="space-y-4"><div className="flex items-start justify-between gap-4"><div><h3 className="flex items-center gap-2 text-lg font-semibold text-gray-200"><Crown className="w-5 h-5 text-violet-400" />Root Failover</h3><p className="text-sm text-gray-500">Registry-attested health and governance progression. Signing and credential actions remain server-governed.</p></div><Button size="sm" variant="outline" className="border-gray-700" onClick={() => refetch()}><RefreshCw className="w-4 h-4 mr-1" />Refresh</Button></div>
    <div className="flex gap-2"><Badge variant="outline" className={state.rootStatus === 1 ? 'border-emerald-500/30 text-emerald-300' : 'border-red-500/30 text-red-300'}>{state.rootStatus === 1 ? 'ROOT AVAILABLE' : 'ROOT ATTENTION REQUIRED'}</Badge><Badge variant="outline" className="border-violet-500/30 text-violet-300">{state.phase}</Badge></div>
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4"><Tile label="Current root" value={state.currentRootID || 'Unassigned'} /><Tile label="Previous root" value={state.previousRootID || '—'} /><Tile label="Governance round" value={state.round} /><Tile label="Phase started" value={state.since ? new Date(state.since).toLocaleString() : '—'} /></div>
    <Card className="aether-bevel-dark rounded-xl"><CardHeader><CardTitle className="flex gap-2 text-sm text-gray-200"><Users className="w-4 h-4 text-cyan-400" />Validator quorum</CardTitle><CardDescription>Only fresh, registry-verified validator attestations count toward a failover decision.</CardDescription></CardHeader><CardContent className="grid grid-cols-2 md:grid-cols-5 gap-3 text-center"><Stat label="Active" value={consensus?.activeValidators.length ?? 0} /><Stat label="Unhealthy" value={consensus?.unhealthy.length ?? 0} /><Stat label="Healthy" value={consensus?.healthy.length ?? 0} /><Stat label="Missing" value={consensus?.missing.length ?? 0} /><Stat label="Votes" value={consensus?.votes ?? 0} /></CardContent></Card>
    <Card className="aether-bevel-dark rounded-xl"><CardHeader><CardTitle className="flex gap-2 text-sm text-gray-200"><ShieldAlert className="w-4 h-4 text-amber-400" />Governance levers</CardTitle><CardDescription>These actions are deliberately unavailable from the browser until authenticated server-side governance endpoints are deployed.</CardDescription></CardHeader><CardContent className="flex flex-wrap gap-2"><Button disabled>Begin vote</Button><Button disabled variant="outline">Submit liveness proof</Button><Button disabled variant="outline">Request reclaim</Button><Button disabled variant="outline">Rotate credentials</Button></CardContent></Card>
  </div>;
}
function Stat({ label, value }: { label: string; value: number }) { return <div className="rounded-lg border border-slate-800 bg-slate-950/40 p-3"><div className="text-xl font-bold text-gray-200">{value}</div><div className="text-xs text-gray-500">{label}</div></div>; }
