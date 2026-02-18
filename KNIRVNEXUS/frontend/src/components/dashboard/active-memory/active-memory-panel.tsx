'use client';

import React, { useState, useEffect } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { ScrollArea } from '@/components/ui/scroll-area';
import { FileText, Search, Clock, ShieldCheck, Activity } from 'lucide-react';

interface Trace {
  id: string;
  agent_id: string;
  error_id: string;
  timestamp: string;
  type: string;
}

export function ActiveMemoryPanel() {
  const [traces, setTraces] = useState<Trace[]>([]);
  const [selectedTrace, setSelectedTrace] = useState<string | null>(null);

  // Mock data for prototype
  useEffect(() => {
    setTraces([
      { id: 'trace_1740000000', agent_id: 'agent-alpha', error_id: 'err_992', timestamp: '2026-02-16T14:30:00Z', type: 'TRACE' },
      { id: 'trace_1740000001', agent_id: 'agent-beta', error_id: 'err_995', timestamp: '2026-02-16T14:35:00Z', type: 'TRACE' },
    ]);
  }, []);

  return (
    <div className="grid grid-cols-1 md:grid-cols-3 gap-6 h-[600px]">
      <Card className="md:col-span-1 flex flex-col">
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            <Clock className="w-5 h-5 text-primary" />
            <span>Reasoning Traces</span>
          </CardTitle>
          <CardDescription>Encrypted Markdown Fabric</CardDescription>
        </CardHeader>
        <CardContent className="flex-1 overflow-hidden">
          <ScrollArea className="h-full pr-4">
            <div className="space-y-2">
              {traces.map((trace) => (
                <div
                  key={trace.id}
                  className={`p-3 rounded-lg border cursor-pointer transition-colors ${
                    selectedTrace === trace.id ? 'bg-primary/10 border-primary' : 'hover:bg-muted'
                  }`}
                  onClick={() => setSelectedTrace(trace.id)}
                >
                  <div className="flex items-center justify-between mb-1">
                    <span className="font-mono text-xs truncate">{trace.id}</span>
                    <Badge variant="outline" className="text-[10px]">PQC Signed</Badge>
                  </div>
                  <div className="text-sm font-medium">{trace.agent_id}</div>
                  <div className="text-[10px] text-muted-foreground">{new Date(trace.timestamp).toLocaleString()}</div>
                </div>
              ))}
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
          {selectedTrace ? (
            <div className="space-y-4">
              <div className="text-primary"># Reasoning Trace: {selectedTrace}</div>
              <div>**Agent:** agent-alpha</div>
              <div>**Error ID:** err_992</div>
              <div className="border-l-2 border-primary/30 pl-4 space-y-2">
                <div className="text-muted-foreground">1. Detected: API Connection Timeout</div>
                <div className="text-muted-foreground">2. Searching Vault for compatible solutions...</div>
                <div className="text-muted-foreground">3. Found SolutionNode: sol_network_v1</div>
                <div className="text-muted-foreground">4. Verifying solution integrity via PQC signature...</div>
                <div className="text-green-500">5. Result: Success</div>
              </div>
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
