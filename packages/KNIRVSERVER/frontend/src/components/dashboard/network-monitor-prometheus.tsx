'use client';

import React, { useState, useEffect } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { getAuthHeaders } from '@/lib/api';
import { ExternalLink } from 'lucide-react';

export function NetworkMonitorPrometheus() {
  const [metricsURL, setMetricsURL] = useState<string | null>(null);
  const loaded = metricsURL !== null;
  const [error, setError] = useState(false);

  useEffect(() => {
    const controller = new AbortController();
    let objectURL: string | undefined;
    fetch('/api/v1/monitor/metrics', { headers: getAuthHeaders(), signal: controller.signal })
      .then(async (response) => {
        if (!response.ok) throw new Error(`Metrics request failed: ${response.status}`);
        const text = await response.text();
        if (controller.signal.aborted) return;
        objectURL = URL.createObjectURL(new Blob([text], { type: 'text/plain' }));
        setMetricsURL(objectURL);
      })
      .catch(() => { if (!controller.signal.aborted) setError(true); });
    return () => {
      controller.abort();
      if (objectURL) URL.revokeObjectURL(objectURL);
    };
  }, []);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-semibold text-gray-200">Prometheus</h3>
          <p className="text-sm text-gray-500">
            KNIRVMONITOR Prometheus exposition endpoint
          </p>
        </div>
        <Button
          variant="outline"
          size="sm"
          className="border-gray-700 text-gray-400 hover:bg-cyan-500/10 hover:text-cyan-400"
          disabled={!metricsURL}
          onClick={() => metricsURL && window.open(metricsURL, '_blank', 'noopener,noreferrer')}
        >
          <ExternalLink className="w-4 h-4 mr-2" />
          Open in new tab
        </Button>
      </div>

      <Card className="aether-bevel-dark rounded-xl overflow-hidden">
        <CardContent className="p-0">
          {!loaded && !error && (
            <div className="flex items-center justify-center py-24 text-gray-500">
              Loading Prometheus metrics...
            </div>
          )}
          {error && (
            <div className="flex flex-col items-center justify-center py-24 text-gray-500">
              <p className="mb-4">Prometheus metrics are not reachable through KNIRVGATEWAY.</p>
              <Button
                variant="outline"
                disabled={!metricsURL}
          onClick={() => metricsURL && window.open(metricsURL, '_blank', 'noopener,noreferrer')}
              >
                <ExternalLink className="w-4 h-4 mr-2" />
                Open Prometheus in new tab
              </Button>
            </div>
          )}
          {metricsURL && <iframe
            src={metricsURL}
            className="w-full h-[600px] border-0"
            onError={() => setError(true)}
            title="KNIRVMONITOR Prometheus Metrics"
          />}
        </CardContent>
      </Card>
    </div>
  );
}
