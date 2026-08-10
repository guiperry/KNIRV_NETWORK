'use client';

import React, { useState, useRef, useEffect } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { ExternalLink } from 'lucide-react';

export function NetworkMonitorPrometheus() {
  const [loaded, setLoaded] = useState(false);
  const [error, setError] = useState(false);

  useEffect(() => {
    const timer = setTimeout(() => setLoaded(true), 100);
    return () => clearTimeout(timer);
  }, []);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-semibold text-gray-200">Prometheus</h3>
          <p className="text-sm text-gray-500">
            Targets page and expression browser
          </p>
        </div>
        <Button
          variant="outline"
          size="sm"
          className="border-gray-700 text-gray-400 hover:bg-cyan-500/10 hover:text-cyan-400"
          onClick={() => window.open('http://localhost:9090/targets', '_blank')}
        >
          <ExternalLink className="w-4 h-4 mr-2" />
          Open in new tab
        </Button>
      </div>

      <Card className="aether-bevel-dark rounded-xl overflow-hidden">
        <CardContent className="p-0">
          {!loaded && !error && (
            <div className="flex items-center justify-center py-24 text-gray-500">
              Loading Prometheus targets...
            </div>
          )}
          {error && (
            <div className="flex flex-col items-center justify-center py-24 text-gray-500">
              <p className="mb-4">Prometheus is not reachable at localhost:9090.</p>
              <Button
                variant="outline"
                onClick={() => window.open('http://localhost:9090/targets', '_blank')}
              >
                <ExternalLink className="w-4 h-4 mr-2" />
                Open Prometheus in new tab
              </Button>
            </div>
          )}
          <iframe
            src="http://localhost:9090/targets"
            className="w-full h-[600px] border-0"
            onLoad={() => setLoaded(true)}
            onError={() => setError(true)}
            title="Prometheus Targets"
          />
        </CardContent>
      </Card>
    </div>
  );
}
