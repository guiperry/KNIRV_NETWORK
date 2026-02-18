'use client';

import React, { useState, useEffect } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Play, Shield, Code, Bug, Package } from 'lucide-react';

interface Solution {
  id: string;
  error_id: string;
  language: string;
  timestamp: string;
}

export function PluginVaultPanel() {
  const [solutions, setSolutions] = useState<Solution[]>([]);
  const [selectedSolution, setSelectedSolution] = useState<string | null>(null);

  useEffect(() => {
    setSolutions([
      { id: 'sol_network_v1', error_id: 'err_992', language: 'go', timestamp: '2026-02-16T10:00:00Z' },
      { id: 'sol_db_repair', error_id: 'err_404', language: 'bash', timestamp: '2026-02-15T18:20:00Z' },
    ]);
  }, []);

  return (
    <div className="grid grid-cols-1 md:grid-cols-3 gap-6 h-[600px]">
      <Card className="md:col-span-1 flex flex-col">
        <CardHeader>
          <CardTitle className="flex items-center space-x-2">
            <Package className="w-5 h-5 text-primary" />
            <span>Solution Registry</span>
          </CardTitle>
          <CardDescription>PQC Vault (.md logic)</CardDescription>
        </CardHeader>
        <CardContent className="flex-1 overflow-hidden">
          <ScrollArea className="h-full pr-4">
            <div className="space-y-2">
              {solutions.map((sol) => (
                <div
                  key={sol.id}
                  className={`p-3 rounded-lg border cursor-pointer transition-colors ${
                    selectedSolution === sol.id ? 'bg-primary/10 border-primary' : 'hover:bg-muted'
                  }`}
                  onClick={() => setSelectedSolution(sol.id)}
                >
                  <div className="flex items-center justify-between mb-1">
                    <span className="font-bold text-xs">{sol.id}</span>
                    <Badge variant="secondary" className="text-[10px] uppercase">{sol.language}</Badge>
                  </div>
                  <div className="flex items-center text-[10px] text-muted-foreground">
                    <Bug className="w-3 h-3 mr-1" />
                    <span>Solves: {sol.error_id}</span>
                  </div>
                </div>
              ))}
            </div>
          </ScrollArea>
        </CardContent>
      </Card>

      <Card className="md:col-span-2 flex flex-col">
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle className="flex items-center space-x-2">
              <Code className="w-5 h-5" />
              <span>Logic Descriptor</span>
            </CardTitle>
            <div className="flex space-x-2">
              <Button size="sm" variant="outline" className="h-8">
                <Shield className="w-3 h-3 mr-1" />
                Audit
              </Button>
              <Button size="sm" className="h-8">
                <Play className="w-3 h-3 mr-1" />
                Invoke
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent className="flex-1 bg-black/5 rounded-lg m-4 mt-0 p-6 font-mono text-sm overflow-auto">
          {selectedSolution ? (
            <div className="space-y-4">
              <div className="text-muted-foreground">// Solution Logic for {selectedSolution}</div>
              <div className="text-blue-400">package main</div>
              <div className="text-blue-400">import "github.com/knirv/sdk"</div>
              <br />
              <div>func Resolve(ctx *sdk.Context) error {'{'}</div>
              <div className="pl-4 text-green-400">
                // Logic to reset the network interface securely
                <br />
                return sdk.ResetInterface("eth0")
              </div>
              <div>{'}'}</div>
            </div>
          ) : (
            <div className="h-full flex items-center justify-center text-muted-foreground">
              Select a solution node from the vault
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
