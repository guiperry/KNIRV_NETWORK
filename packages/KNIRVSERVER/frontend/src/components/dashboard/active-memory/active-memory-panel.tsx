'use client';

import React, { useEffect, useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { ScrollArea } from '@/components/ui/scroll-area';
import { AlertCircle, Clock, ShieldCheck } from 'lucide-react';
import { apiRequest, API_BASE_URL } from '@/lib/api';

interface TraceSummary {
  id: string;
  agent_id: string;
  error_id: string;
  timestamp: string;
  type: string;
}

interface TraceDetail extends TraceSummary {
  steps: string[];
  result: string;
  content: string;
}

export function ActiveMemoryPanel() {
  const [traces, setTraces] = useState<TraceSummary[]>([]);
  const [selectedTraceId, setSelectedTraceId] = useState<string | null>(null);
  const [selectedTrace, setSelectedTrace] = useState<TraceDetail | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    const loadTraces = async () => {
      try {
        const response = await apiRequest<{ traces: TraceSummary[] }>(`${API_BASE_URL}/api/cognitive/active-memory/traces`);
        if (!response.success || !response.data) {
          throw new Error(response.error || 'Failed to load reasoning traces');
        }

        if (cancelled) {
          return;
        }

        const nextTraces = response.data.traces ?? [];
        setTraces(nextTraces);

        if (nextTraces.length > 0) {
          setSelectedTraceId((current) => current ?? nextTraces[0].id);
        }
      } catch (loadError) {
        if (!cancelled) {
          setError(loadError instanceof Error ? loadError.message : 'Failed to load reasoning traces');
        }
      }
    };

    loadTraces();
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    let cancelled = false;

    if (!selectedTraceId) {
      setSelectedTrace(null);
      return;
    }

    const loadTrace = async () => {
      try {
        const response = await apiRequest<TraceDetail>(`${API_BASE_URL}/api/cognitive/active-memory/traces/${selectedTraceId}`);
        if (!response.success || !response.data) {
          throw new Error(response.error || 'Failed to load trace details');
        }

        if (!cancelled) {
          setSelectedTrace(response.data);
          setError(null);
        }
      } catch (loadError) {
        if (!cancelled) {
          setError(loadError instanceof Error ? loadError.message : 'Failed to load trace details');
        }
      }
    };

    loadTrace();
    return () => {
      cancelled = true;
    };
  }, [selectedTraceId]);

  return (
    <div className="grid grid-cols-1 gap-6 md:grid-cols-3 h-[600px]">
      <Card className="md:col-span-1 flex flex-col">
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            <Clock className="w-5 h-5 text-primary" />
            <span>Reasoning Traces</span>
          </CardTitle>
          <CardDescription>Hypergraph-backed memory vault</CardDescription>
        </CardHeader>
        <CardContent className="flex-1 overflow-hidden">
          <ScrollArea className="h-full pr-4">
            <div className="space-y-2">
              {traces.map((trace) => (
                <div
                  key={trace.id}
                  className={`p-3 rounded-lg border cursor-pointer transition-colors ${
                    selectedTraceId === trace.id ? 'bg-primary/10 border-primary' : 'hover:bg-muted'
                  }`}
                  onClick={() => setSelectedTraceId(trace.id)}
                >
                  <div className="flex items-center justify-between mb-1">
                    <span className="font-mono text-xs truncate">{trace.id}</span>
                    <Badge variant="outline" className="text-[10px]">PQC Signed</Badge>
                  </div>
                  <div className="text-sm font-medium">{trace.agent_id}</div>
                  <div className="text-[10px] text-muted-foreground">{new Date(trace.timestamp).toLocaleString()}</div>
                </div>
              ))}
              {!error && traces.length === 0 && (
                <div className="text-sm text-muted-foreground py-6 text-center">
                  No reasoning traces available yet
                </div>
              )}
            </div>
          </ScrollArea>
        </CardContent>
      </Card>

      <Card className="md:col-span-2 flex flex-col">
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>Trace Explorer</CardTitle>
            <ShieldCheck className="w-5 h-5 text-green-500" />
          </div>
        </CardHeader>
        <CardContent className="flex-1 bg-black/5 rounded-lg m-4 mt-0 p-6 font-mono text-sm overflow-auto">
          {error ? (
            <div className="h-full flex items-center justify-center text-destructive gap-2">
              <AlertCircle className="w-4 h-4" />
              <span>{error}</span>
            </div>
          ) : selectedTrace ? (
            <div className="space-y-4">
              <div className="text-primary"># Reasoning Trace: {selectedTrace.id}</div>
              <div>**Agent:** {selectedTrace.agent_id}</div>
              <div>**Error ID:** {selectedTrace.error_id}</div>
              <div className="border-l-2 border-primary/30 pl-4 space-y-2">
                {selectedTrace.steps.map((step, index) => (
                  <div key={`${selectedTrace.id}-${index}`} className={index === selectedTrace.steps.length - 1 && selectedTrace.result === 'Success' ? 'text-green-500' : 'text-muted-foreground'}>
                    {index + 1}. {step}
                  </div>
                ))}
              </div>
              {selectedTrace.result && <div className="text-green-500">Result: {selectedTrace.result}</div>}
            </div>
          ) : (
            <div className="h-full flex items-center justify-center text-muted-foreground">
              Select a trace to view reasoning details
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
