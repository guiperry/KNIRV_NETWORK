'use client';

import React, { useEffect, useMemo, useState } from 'react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Database, Shield, Server, CheckCircle2 } from "lucide-react";

export interface DatabaseConfig {
  internalSchemaName: string;
  internalSchemaVersion: string;
  internalTables: string[];
  externalMcpDatabase: {
    enabled: boolean;
    name: string;
    provider: string;
    selectedServerId: string;
    saveProfile: boolean;
    saveMemory: boolean;
    saveKnowledgeGraph: boolean;
    savePolicies: boolean;
    saveAuditTrail: boolean;
  };
  notes: string;
}

interface DatabaseConfigModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (config: DatabaseConfig) => void;
  initialConfig?: DatabaseConfig;
}

const defaultTables = [
  'users',
  'sessions',
  'memory_entries',
  'knowledge_graph_nodes',
  'knowledge_graph_edges',
  'policy_rules',
  'audit_events',
];

export function DatabaseConfigModal({ isOpen, onClose, onSave, initialConfig }: DatabaseConfigModalProps) {
  const startingConfig = useMemo<DatabaseConfig>(() => (
    initialConfig || {
      internalSchemaName: 'KNIRVBASE',
      internalSchemaVersion: 'v1',
      internalTables: defaultTables,
      externalMcpDatabase: {
        enabled: false,
        name: '',
        provider: '',
        selectedServerId: '',
        saveProfile: false,
        saveMemory: true,
        saveKnowledgeGraph: true,
        savePolicies: true,
        saveAuditTrail: true,
      },
      notes: '',
    }
  ), [initialConfig]);

  const [config, setConfig] = useState<DatabaseConfig>(startingConfig);

  useEffect(() => {
    if (isOpen) {
      setConfig(startingConfig);
    }
  }, [isOpen, startingConfig]);

  const toggleInternalTable = (table: string) => {
    setConfig(prev => ({
      ...prev,
      internalTables: prev.internalTables.includes(table)
        ? prev.internalTables.filter(item => item !== table)
        : [...prev.internalTables, table],
    }));
  };

  const toggleExternalFlag = (key: keyof DatabaseConfig['externalMcpDatabase']) => {
    if (typeof config.externalMcpDatabase[key] !== 'boolean') {
      return;
    }
    setConfig(prev => ({
      ...prev,
      externalMcpDatabase: {
        ...prev.externalMcpDatabase,
        [key]: !prev.externalMcpDatabase[key],
      },
    }));
  };

  const handleSave = () => {
    onSave(config);
    onClose();
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="w-[96vw] max-w-7xl max-h-[92vh] overflow-hidden bg-[#0a0a0c] border-white/10 text-slate-200 flex flex-col">
        <DialogHeader>
          <div className="flex items-center gap-3">
            <div className="p-2 bg-blue-600/20 rounded-lg">
              <Database className="text-blue-500" size={24} />
            </div>
            <div>
              <DialogTitle className="text-xl font-bold text-white">Database Config</DialogTitle>
              <DialogDescription className="text-slate-400 text-sm">
                Confirm the KNIRVBASE schema and choose what mirrors into any external MCP-selected database.
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <div className="flex-1 overflow-y-auto py-4 pr-1 custom-scrollbar space-y-6">
          <div className="p-4 bg-blue-500/5 border border-blue-500/20 rounded-xl flex items-start gap-3">
            <Shield className="text-blue-500 shrink-0 mt-1" size={18} />
            <p className="text-sm text-slate-400 leading-relaxed">
              Internal KNIRVBASE remains canonical. External database settings only mirror the data classes you explicitly enable.
            </p>
          </div>

          <div className="grid md:grid-cols-2 gap-4">
            <div className="p-5 bg-white/5 border border-white/10 rounded-xl space-y-4">
              <div className="flex items-center gap-2">
                <Database className="text-blue-500" size={18} />
                <h3 className="font-bold text-sm">Internal KNIRVBASE</h3>
              </div>

              <div className="space-y-2">
                <Label className="text-xs uppercase font-bold text-slate-500 tracking-wider">Schema Name</Label>
                <Input
                  value={config.internalSchemaName}
                  onChange={(e) => setConfig(prev => ({ ...prev, internalSchemaName: e.target.value }))}
                  className="bg-black/40 border-white/10 text-white"
                />
              </div>

              <div className="space-y-2">
                <Label className="text-xs uppercase font-bold text-slate-500 tracking-wider">Schema Version</Label>
                <Input
                  value={config.internalSchemaVersion}
                  onChange={(e) => setConfig(prev => ({ ...prev, internalSchemaVersion: e.target.value }))}
                  className="bg-black/40 border-white/10 text-white"
                />
              </div>

              <div className="space-y-3">
                <Label className="text-xs uppercase font-bold text-slate-500 tracking-wider">Tables</Label>
                <ScrollArea className="h-44 rounded-lg border border-white/10 bg-black/20 p-3">
                  <div className="space-y-3">
                    {defaultTables.map((table) => (
                      <div key={table} className="flex items-center justify-between gap-3">
                        <div>
                          <div className="text-sm text-slate-200">{table}</div>
                          <div className="text-xs text-slate-500">Persisted in the internal KNIRVBASE store</div>
                        </div>
                        <Switch
                          checked={config.internalTables.includes(table)}
                          onCheckedChange={() => toggleInternalTable(table)}
                        />
                      </div>
                    ))}
                  </div>
                </ScrollArea>
              </div>
            </div>

            <div className="p-5 bg-white/5 border border-white/10 rounded-xl space-y-4">
              <div className="flex items-center gap-2">
                <Server className="text-blue-500" size={18} />
                <h3 className="font-bold text-sm">External MCP Database</h3>
              </div>

              <div className="flex items-center justify-between rounded-lg border border-white/10 bg-black/20 p-3">
                <div>
                  <div className="text-sm text-slate-200">Enable external mirror</div>
                  <div className="text-xs text-slate-500">Select which data classes should be persisted externally</div>
                </div>
                <Switch
                  checked={config.externalMcpDatabase.enabled}
                  onCheckedChange={() => setConfig(prev => ({
                    ...prev,
                    externalMcpDatabase: {
                      ...prev.externalMcpDatabase,
                      enabled: !prev.externalMcpDatabase.enabled,
                    },
                  }))}
                />
              </div>

              <div className="space-y-2">
                <Label className="text-xs uppercase font-bold text-slate-500 tracking-wider">Database Name</Label>
                <Input
                  value={config.externalMcpDatabase.name}
                  onChange={(e) => setConfig(prev => ({
                    ...prev,
                    externalMcpDatabase: { ...prev.externalMcpDatabase, name: e.target.value },
                  }))}
                  placeholder="Optional external DB alias"
                  className="bg-black/40 border-white/10 text-white"
                />
              </div>

              <div className="space-y-2">
                <Label className="text-xs uppercase font-bold text-slate-500 tracking-wider">Provider</Label>
                <Input
                  value={config.externalMcpDatabase.provider}
                  onChange={(e) => setConfig(prev => ({
                    ...prev,
                    externalMcpDatabase: { ...prev.externalMcpDatabase, provider: e.target.value },
                  }))}
                  placeholder="MCP server or database provider"
                  className="bg-black/40 border-white/10 text-white"
                />
              </div>

              <div className="space-y-2">
                <Label className="text-xs uppercase font-bold text-slate-500 tracking-wider">Selected MCP Server ID</Label>
                <Input
                  value={config.externalMcpDatabase.selectedServerId}
                  onChange={(e) => setConfig(prev => ({
                    ...prev,
                    externalMcpDatabase: { ...prev.externalMcpDatabase, selectedServerId: e.target.value },
                  }))}
                  placeholder="server-id"
                  className="bg-black/40 border-white/10 text-white"
                />
              </div>

              <div className="space-y-3">
                {[
                  ['saveProfile', 'Profile metadata'],
                  ['saveMemory', 'Memory entries'],
                  ['saveKnowledgeGraph', 'Knowledge graph'],
                  ['savePolicies', 'Policy records'],
                  ['saveAuditTrail', 'Audit trail'],
                ].map(([key, label]) => (
                  <div key={key} className="flex items-center justify-between rounded-lg border border-white/10 bg-black/20 p-3">
                    <div>
                      <div className="text-sm text-slate-200">{label}</div>
                      <div className="text-xs text-slate-500">Control what syncs to the selected external store</div>
                    </div>
                    <Switch
                      checked={config.externalMcpDatabase[key as keyof DatabaseConfig['externalMcpDatabase']] as boolean}
                      onCheckedChange={() => toggleExternalFlag(key as keyof DatabaseConfig['externalMcpDatabase'])}
                    />
                  </div>
                ))}
              </div>
            </div>
          </div>

          <div className="space-y-2">
            <Label className="text-xs uppercase font-bold text-slate-500 tracking-wider">Notes</Label>
            <Textarea
              value={config.notes}
              onChange={(e) => setConfig(prev => ({ ...prev, notes: e.target.value }))}
              className="min-h-28 bg-black/40 border-white/10 text-white"
              placeholder="Optional schema notes or migration details"
            />
          </div>

          <div className="flex items-center justify-end gap-3">
            <Button variant="ghost" onClick={onClose}>Cancel</Button>
            <Button onClick={handleSave} className="bg-blue-600 hover:bg-blue-500 text-white">
              <CheckCircle2 className="mr-2" size={16} />
              Save Database Config
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

export default DatabaseConfigModal;
