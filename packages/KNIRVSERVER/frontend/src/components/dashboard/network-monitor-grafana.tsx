'use client';

import { useQuery } from '@tanstack/react-query';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { getAuthHeaders } from '@/lib/api';
import { ExternalLink } from 'lucide-react';

type GrafanaStatus = { configured: boolean; url: string };

async function loadGrafanaStatus(): Promise<GrafanaStatus> {
  const response = await fetch('/api/v1/monitor/grafana', { headers: getAuthHeaders() });
  if (!response.ok) throw new Error(`Grafana status request failed: ${response.status}`);
  const payload = await response.json() as { data?: GrafanaStatus };
  return payload.data ?? { configured: false, url: '' };
}

export function NetworkMonitorGrafana() {
  const query = useQuery({ queryKey: ['network-monitor', 'grafana'], queryFn: loadGrafanaStatus, staleTime: 30_000, retry: 1 });
  const grafana = query.data;

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-semibold text-gray-200">Grafana Dashboard</h3>
          <p className="text-sm text-gray-500">
            Network topology and performance metrics
          </p>
        </div>
        {grafana?.configured && <Button variant="outline" size="sm" className="border-gray-700 text-gray-400 hover:bg-cyan-500/10 hover:text-cyan-400" onClick={() => window.open(grafana.url, '_blank', 'noopener,noreferrer')}><ExternalLink className="w-4 h-4 mr-2" />Open Grafana</Button>}
      </div>

      <Card className="aether-bevel-dark rounded-xl overflow-hidden">
        <CardContent className="p-0">
          {query.isLoading && <div className="py-24 text-center text-gray-500">Checking Grafana configuration…</div>}
          {query.error && <div className="py-24 text-center text-amber-300">Grafana configuration could not be loaded.</div>}
          {grafana && !grafana.configured && <div className="py-24 px-6 text-center text-gray-500">Grafana is not configured for this KNIRVSERVER. Set <code>KNIRV_MONITOR_GRAFANA_URL</code> to the externally reachable Grafana URL, then restart KNIRVSERVER.</div>}
          {grafana?.configured && <iframe src={grafana.url} className="w-full h-[600px] border-0" title="Grafana Dashboard" />}
        </CardContent>
      </Card>
    </div>
  );
}
