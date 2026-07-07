'use client';

import React, { useState, useRef, useEffect } from 'react';
import { X, Terminal, Play, Database, Settings, Shield, BarChart3, Lock, Key, FileText, RefreshCw, HardDrive, Cpu } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { API_BASE_URL, getAuthHeaders } from '@/lib/api';

interface ActiveMemoryAccessModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export function ActiveMemoryAccessModal({ isOpen, onClose }: ActiveMemoryAccessModalProps) {
  const [terminalOutput, setTerminalOutput] = useState([
    '$ Welcome to Active Memory (KNIRVBASE) Terminal',
    '$ Service: Encrypted PQC Markdown Persistence',
    '$ Type "help" for available commands',
    '$ '
  ]);
  const [currentCommand, setCurrentCommand] = useState('');
  const [isExecuting, setIsExecuting] = useState(false);
  const [showConsole, setShowConsole] = useState(false);
  const [showEncryption, setShowEncryption] = useState(false);
  const [showFabric, setShowFabric] = useState(false);
  const terminalEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    terminalEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [terminalOutput]);

  const workflowTemplates = [
    {
      id: 'init-fabric',
      name: 'Initialize Fabric Slice',
      description: 'Create new encrypted .md fabric slice for persistence',
      icon: <HardDrive className="w-4 h-4" />,
      commands: ['fabric-create', 'kyber-init', 'persistence-enable']
    },
    {
      id: 'sync-memory',
      name: 'Memory Sync',
      description: 'Synchronize encrypted memory across nodes',
      icon: <RefreshCw className="w-4 h-4" />,
      commands: ['sync-start', 'verify-encryption', 'commit-changes']
    },
    {
      id: 'backup-restore',
      name: 'Backup & Restore',
      description: 'Create encrypted backups or restore from backup',
      icon: <Database className="w-4 h-4" />,
      commands: ['backup-create', 'verify-integrity', 'restore-point']
    }
  ];

  const executeCommand = async (command: string) => {
    const trimmed = command.trim();
    if (!trimmed) return;

    setTerminalOutput(prev => [...prev, '$ ' + trimmed]);
    setCurrentCommand('');

    if (trimmed.toLowerCase() === 'clear') {
      setTerminalOutput(['$ ']);
      return;
    }

    if (trimmed.toLowerCase() === 'help') {
      setTerminalOutput(prev => [
        ...prev,
        'Available commands:',
        '  help        - Show this help message',
        '  status      - Show memory status',
        '  sync        - Synchronize memory',
        '  clear       - Clear terminal',
        '  <command>   - Execute via knirvshell',
        '$ '
      ]);
      return;
    }

    setIsExecuting(true);
    try {
      const resp = await fetch(`${API_BASE_URL}/api/v1/shell/execute`, {
        method: 'POST',
        headers: { ...getAuthHeaders(), 'Content-Type': 'application/json' },
        body: JSON.stringify({ command: trimmed }),
      });
      const data = await resp.json();
      const output: string[] = [];
      if (!resp.ok) {
        output.push(`Error: ${data.error || data.message || 'Command failed'}`);
      } else if (Array.isArray(data.output) && data.output.length > 0) {
        output.push(...data.output);
      } else if (typeof data.output === 'string' && data.output) {
        output.push(...data.output.split('\n').filter(Boolean));
      } else {
        output.push(`Status: ${data.status || 'completed'}`);
      }
      if (output.length === 0) output.push('(no output)');
      setTerminalOutput(prev => [...prev, ...output, '$ ']);
    } catch {
      setTerminalOutput(prev => [...prev, 'Error: Failed to reach backend', '$ ']);
    } finally {
      setIsExecuting(false);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex">
      <div className="flex-1 bg-black/20 backdrop-blur-sm transition-colors duration-300 z-40" onClick={onClose} />

      <div className="relative w-full max-w-4xl bg-background border-l shadow-2xl transform transition-slide duration-300 ease-in-out">
        <div className="flex flex-col h-full">
          <div className="flex items-center justify-between p-6 border-b">
            <div>
              <h2 className="text-2xl font-bold">Active Memory Access</h2>
              <p className="text-muted-foreground">
                KNIRVBASE - Encrypted PQC Markdown Persistence
              </p>
            </div>
            <div className="flex items-center space-x-2">
              <Badge variant="secondary">Online</Badge>
              <Button variant="ghost" size="sm" onClick={onClose}>
                <X className="w-4 h-4" />
              </Button>
            </div>
          </div>

          <div className="flex-1 overflow-auto">
            <Tabs defaultValue="terminal" className="h-full">
              <div className="px-6 pt-4">
                <TabsList className="grid w-full grid-cols-3">
                  <TabsTrigger value="terminal">Terminal</TabsTrigger>
                  <TabsTrigger value="workflows">Workflows</TabsTrigger>
                  <TabsTrigger value="tools">Tools</TabsTrigger>
                </TabsList>
              </div>

              <TabsContent value="terminal" className="px-6 pb-6 space-y-4">
                <Card className="knirv-card-gradient">
                  <CardHeader>
                    <CardTitle className="flex items-center space-x-2">
                      <Terminal className="w-5 h-5" />
                      <span>Active Memory Terminal</span>
                      {isExecuting && <Badge variant="secondary" className="text-xs">running...</Badge>}
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="bg-black rounded-lg p-4 font-mono text-sm">
                      <div className="text-blue-400 space-y-1 max-h-96 overflow-y-auto">
                        {terminalOutput.map((line, index) => (
                          <div key={index}>{line}</div>
                        ))}
                        <div ref={terminalEndRef} />
                      </div>
                      <div className="flex items-center mt-2">
                        <span className="text-blue-400">$ </span>
                        <input
                          type="text"
                          value={currentCommand}
                          onChange={(e) => setCurrentCommand(e.target.value)}
                          onKeyDown={(e) => {
                            if (e.key === 'Enter' && currentCommand.trim() && !isExecuting) {
                              executeCommand(currentCommand.trim());
                            }
                          }}
                          className="flex-1 bg-transparent text-blue-400 outline-none ml-2"
                          placeholder={isExecuting ? 'Executing...' : 'Enter command...'}
                          disabled={isExecuting}
                          autoFocus
                        />
                      </div>
                    </div>
                  </CardContent>
                </Card>
              </TabsContent>

              <TabsContent value="workflows" className="px-6 pb-6 space-y-4">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  {workflowTemplates.map((template) => (
                    <Card key={template.id} className="knirv-card-gradient">
                      <CardHeader className="pb-2">
                        <CardTitle className="flex items-center space-x-2 text-sm">
                          {template.icon}
                          <span>{template.name}</span>
                        </CardTitle>
                        <CardDescription className="text-xs">
                          {template.description}
                        </CardDescription>
                      </CardHeader>
                      <CardContent className="space-y-2">
                        <div className="text-xs text-muted-foreground">
                          Commands: {template.commands.join(', ')}
                        </div>
                        <Button
                          variant="outline"
                          size="sm"
                          className="w-full"
                          disabled={isExecuting}
                          onClick={() => executeCommand(template.commands[0])}
                        >
                          <Play className="w-3 h-3 mr-1" />
                          Execute
                        </Button>
                      </CardContent>
                    </Card>
                  ))}
                </div>
              </TabsContent>

              <TabsContent value="tools" className="px-6 pb-6 space-y-4">
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
                  <Card className="knirv-card-gradient">
                    <CardHeader className="pb-2">
                      <CardTitle className="flex items-center space-x-2 text-sm">
                        <Key className="w-4 h-4" />
                        <span>Encryption</span>
                      </CardTitle>
                      <CardDescription className="text-xs">
                        PQC key management
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      <Button variant={showEncryption ? "default" : "outline"} size="sm" className="w-full" onClick={() => setShowEncryption(!showEncryption)}>
                        {showEncryption ? 'Hide' : 'Configure'}
                      </Button>
                    </CardContent>
                  </Card>

                  <Card className="knirv-card-gradient">
                    <CardHeader className="pb-2">
                      <CardTitle className="flex items-center space-x-2 text-sm">
                        <Database className="w-4 h-4" />
                        <span>Fabric</span>
                      </CardTitle>
                      <CardDescription className="text-xs">
                        .md fabric slices
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      <Button variant={showFabric ? "default" : "outline"} size="sm" className="w-full" onClick={() => setShowFabric(!showFabric)}>
                        {showFabric ? 'Hide' : 'Manage'}
                      </Button>
                    </CardContent>
                  </Card>

                  <Card className="knirv-card-gradient">
                    <CardHeader className="pb-2">
                      <CardTitle className="flex items-center space-x-2 text-sm">
                        <Terminal className="w-4 h-4" />
                        <span>Console</span>
                      </CardTitle>
                      <CardDescription className="text-xs">
                        Real-time logs
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      <Button variant={showConsole ? "default" : "outline"} size="sm" className="w-full" onClick={() => setShowConsole(!showConsole)}>
                        {showConsole ? 'Hide' : 'View'}
                      </Button>
                    </CardContent>
                  </Card>

                  <Card className="knirv-card-gradient">
                    <CardHeader className="pb-2">
                      <CardTitle className="flex items-center space-x-2 text-sm">
                        <BarChart3 className="w-4 h-4" />
                        <span>Monitor</span>
                      </CardTitle>
                      <CardDescription className="text-xs">
                        Memory metrics
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      <Button variant="outline" size="sm" className="w-full">
                        Open
                      </Button>
                    </CardContent>
                  </Card>
                </div>
              </TabsContent>
            </Tabs>
          </div>
        </div>
      </div>
    </div>
  );
}
