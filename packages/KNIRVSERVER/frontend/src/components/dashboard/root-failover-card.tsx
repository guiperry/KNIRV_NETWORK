"use client";

import { AlertTriangle, Crown, ShieldCheck } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { useRootFailover } from '@/hooks/use-root-failover';

export function RootFailoverCard({ onOpen }: { onOpen: () => void }) {
  const { state, isLoading, error } = useRootFailover();
  const safe = state?.rootStatus === 1;
  const phase = state?.phase ?? (error ? 'UNKNOWN' : 'CHECKING');
  return <Card className="aether-bevel-dark rounded-2xl cursor-pointer aether-bevel-dark-hover transition-interactive" onClick={onOpen}>
    <CardHeader className="pb-3"><div className="flex items-center justify-between gap-3"><CardTitle className="flex items-center gap-2 text-sm text-gray-300"><Crown className="w-4 h-4 text-violet-400" />Root Failover</CardTitle><Badge variant="outline" className={safe ? 'border-emerald-500/30 text-emerald-300' : 'border-amber-500/30 text-amber-300'}>{isLoading ? 'CHECKING' : phase}</Badge></div><CardDescription className="text-xs text-gray-500">Attested root availability and governance state</CardDescription></CardHeader>
    <CardContent className="space-y-2"><div className="flex items-center gap-2 text-xs text-gray-400">{safe ? <ShieldCheck className="w-4 h-4 text-emerald-400" /> : <AlertTriangle className="w-4 h-4 text-amber-400" />}<span>{state?.currentRootID || (error ? 'Registry unavailable' : 'No confirmed root')}</span></div><div className="text-[10px] text-gray-500">Round {state?.round ?? 0} · click for validator quorum and audit detail</div></CardContent>
  </Card>;
}
