'use client';

import React, { useState, useRef, useEffect } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { ExternalLink } from 'lucide-react';

export function NetworkMonitorGrafana() {
  const [loaded, setLoaded] = useState(false);
  const [error, setError] = useState(false);
  const iframeRef = useRef<HTMLIFrameElement>(null);

  useEffect(() => {
    const timer = setTimeout(() => setLoaded(true), 100);
    return () => clearTimeout(timer);
  }, []);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-semibold text-gray-200">Grafana Dashboard</h3>
          <p className="text-sm text-gray-500">
            Network topology and performance metrics
          </p>
        </div>
        <Button
          variant="outline"
          size="sm"
          className="border-gray-700 text-gray-400 hover:bg-cyan-500/10 hover:text-cyan-400"
          onClick={() => window.open('http://localhost:3333/d/knirv-network', '_blank')}
        >
          <ExternalLink className="w-4 h-4 mr-2" />
          Open in new tab
        </Button>
      </div>

      <Card className="aether-bevel-dark rounded-xl overflow-hidden">
        <CardContent className="p-0">
          {!loaded && !error && (
            <div className="flex items-center justify-center py-24 text-gray-500">
              Loading Grafana dashboard...
            </div>
          )}
          {error && (
            <div className="flex flex-col items-center justify-center py-24 text-gray-500">
              <p className="mb-4">Grafana is not reachable at localhost:3333.</p>
              <Button
                variant="outline"
                onClick={() => window.open('http://localhost:3333/d/knirv-network', '_blank')}
              >
                <ExternalLink className="w-4 h-4 mr-2" />
                Open Grafana in new tab
              </Button>
            </div>
          )}
          <iframe
            ref={iframeRef}
            src="http://localhost:3333/d/knirv-network"
            className="w-full h-[600px] border-0"
            onLoad={() => setLoaded(true)}
            onError={() => setError(true)}
            title="Grafana Dashboard"
          />
        </CardContent>
      </Card>
    </div>
  );
}
