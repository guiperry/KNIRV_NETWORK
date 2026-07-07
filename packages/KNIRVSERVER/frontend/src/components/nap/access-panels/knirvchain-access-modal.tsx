'use client';

import React, { useState, useRef, useEffect } from 'react';
import { X, Terminal, Play, Lock, Settings, BarChart3, FileText, Coins, Zap, Shield, GitCommit, Package, Code, Bug } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { ScrollArea } from '@/components/ui/scroll-area';
import { API_BASE_URL, getAuthHeaders } from '@/lib/api';

interface KNIRVChainAccessModalProps {
  isOpen: boolean;
  onClose: () => void;
}

interface Solution {
  id: string;
  error_id: string;
  language: string;
  timestamp: string;
}

export function KNIRVChainAccessModal({ isOpen, onClose }: KNIRVChainAccessModalProps) {
  const [terminalOutput, setTerminalOutput] = useState([
    '$ Welcome to KNIRVCHAIN Terminal',
    '$ Service: Solution Vault - Verifiable Executable Logic',
    '$ Type "help" for available commands',
    '$ '
  ]);
  const [currentCommand, setCurrentCommand] = useState('');
  const [isExecuting, setIsExecuting] = useState(false);
  const [showMint, setShowMint] = useState(false);
  const [showNodes, setShowNodes] = useState(false);
  const [showRewards, setShowRewards] = useState(false);
  const [showSolutionVault, setShowSolutionVault] = useState(false);
  const [selectedSolution, setSelectedSolution] = useState<string | null>(null);
  const terminalEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    terminalEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [terminalOutput]);

  const [solutions] = useState<Solution[]>([
    { id: 'sol_network_v1', error_id: 'err_992', language: 'go', timestamp: '2026-02-16T10:00:00Z' },
    { id: 'sol_db_repair', error_id: 'err_404', language: 'bash', timestamp: '2026-02-15T18:20:00Z' },
    { id: 'sol_api_timeout', error_id: 'err_503', language: 'go', timestamp: '2026-02-14T12:00:00Z' },
    { id: 'sol_mem_leak', error_id: 'err_421', language: 'rust', timestamp: '2026-02-13T09:30:00Z' },
    { id: 'sol_auth_fail', error_id: 'err_401', language: 'go', timestamp: '2026-02-12T16:45:00Z' },
  ]);

  const workflowTemplates = [
    {
      id: 'mint-node',
      name: 'Mint New Node',
      description: 'Create and mint new capability node to blockchain',
      icon: <Coins className="w-4 h-4" />,
      commands: ['node-create', 'sign-proof', 'submit-transaction']
    },
    {
      id: 'validate-skill',
      name: 'Validate Skill',
      description: 'Run proof-of-skill validation for node',
      icon: <Shield className="w-4 h-4" />,
      commands: ['skill-verify', 'attestation-check', 'consensus-vote']
    },
    {
      id: 'view-blocks',
      name: 'View Blocks',
      description: 'Browse recent blocks and transactions',
      icon: <GitCommit className="w-4 h-4" />,
      commands: ['blocks-recent', 'tx-list', 'verify-chain']
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
        '  status      - Show chain status',
        '  blocks      - View recent blocks',
        '  nodes       - List capability nodes',
        '  clear       - Clear terminal',
        '  <command>   - Execute via knirvshell chain',
        '$ '
      ]);
      return;
    }

    setIsExecuting(true);
    try {
      const resp = await fetch(`${API_BASE_URL}/api/v1/shell/chain/execute`, {
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
              <h2 className="text-2xl font-bold">KNIRVCHAIN Access</h2>
              <p className="text-muted-foreground">
                Solution Vault - Verifiable Executable Logic
              </p>
            </div>
            <div className="flex items-center space-x-2">
              <Badge variant="secondary">Minting</Badge>
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
                      <span>KNIRVCHAIN Terminal</span>
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
                        <Package className="w-4 h-4" />
                        <span>Solution Vault</span>
                      </CardTitle>
                      <CardDescription className="text-xs">
                        PQC vault .md logic
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      <Button
                        variant={showSolutionVault ? "default" : "outline"}
                        size="sm"
                        className="w-full"
                        onClick={() => setShowSolutionVault(!showSolutionVault)}
                      >
                        <Zap className="w-3 h-3 mr-1" />
                        {showSolutionVault ? 'Hide' : 'Open'}
                      </Button>
                    </CardContent>
                  </Card>

                  <Card className="knirv-card-gradient">
                    <CardHeader className="pb-2">
                      <CardTitle className="flex items-center space-x-2 text-sm">
                        <Coins className="w-4 h-4" />
                        <span>Mint</span>
                      </CardTitle>
                      <CardDescription className="text-xs">
                        Mint new nodes
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      <Button variant={showMint ? "default" : "outline"} size="sm" className="w-full" onClick={() => setShowMint(!showMint)}>
                        {showMint ? 'Hide' : 'Open'}
                      </Button>
                    </CardContent>
                  </Card>

                  <Card className="knirv-card-gradient">
                    <CardHeader className="pb-2">
                      <CardTitle className="flex items-center space-x-2 text-sm">
                        <Lock className="w-4 h-4" />
                        <span>Nodes</span>
                      </CardTitle>
                      <CardDescription className="text-xs">
                        View capability nodes
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      <Button variant={showNodes ? "default" : "outline"} size="sm" className="w-full" onClick={() => setShowNodes(!showNodes)}>
                        {showNodes ? 'Hide' : 'View'}
                      </Button>
                    </CardContent>
                  </Card>

                  <Card className="knirv-card-gradient">
                    <CardHeader className="pb-2">
                      <CardTitle className="flex items-center space-x-2 text-sm">
                        <Zap className="w-4 h-4" />
                        <span>Rewards</span>
                      </CardTitle>
                      <CardDescription className="text-xs">
                        Staking & rewards
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      <Button variant={showRewards ? "default" : "outline"} size="sm" className="w-full" onClick={() => setShowRewards(!showRewards)}>
                        {showRewards ? 'Hide' : 'View'}
                      </Button>
                    </CardContent>
                  </Card>

                  <Card className="knirv-card-gradient">
                    <CardHeader className="pb-2">
                      <CardTitle className="flex items-center space-x-2 text-sm">
                        <BarChart3 className="w-4 h-4" />
                        <span>Analytics</span>
                      </CardTitle>
                      <CardDescription className="text-xs">
                        Chain statistics
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

            {/* Solution Vault Panel */}
            {showSolutionVault && (
              <div className="fixed left-0 top-0 bottom-0 z-[60] pointer-events-auto transform transition-slide duration-300 ease-out translate-x-0 w-96 gpu-accelerated">
                <div className="h-full bg-slate-900 rounded-r-lg border-r border-blue-600/30 shadow-2xl">
                  <div className="flex items-center justify-between p-4 border-b border-blue-600/30">
                    <div className="flex items-center space-x-2">
                      <Package className="w-4 h-4 text-cyan-400" />
                      <h3 className="font-semibold text-sm">Solution Vault</h3>
                    </div>
                    <button
                      onClick={() => setShowSolutionVault(false)}
                      className="p-1 hover:bg-slate-800 rounded transition-colors">
                      <X className="w-4 h-4" />
                    </button>
                  </div>

                  <div className="flex flex-col h-[calc(100%-56px)]">
                    <div className="p-4 border-b border-blue-600/30">
                      <h4 className="text-xs font-semibold text-cyan-400 mb-3 flex items-center space-x-2">
                        <Package className="w-3 h-3" />
                        <span>Solution Registry</span>
                      </h4>
                      <ScrollArea className="h-48">
                        <div className="space-y-2">
                          {solutions.map((sol) => (
                            <div
                              key={sol.id}
                              className={`p-3 rounded-lg border cursor-pointer transition-colors ${
                                selectedSolution === sol.id ? 'bg-blue-600/20 border-blue-600/50' : 'bg-slate-800/50 border-blue-600/20 hover:border-blue-600/50'
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
                    </div>

                    <div className="flex-1 p-4 overflow-auto">
                      <div className="flex items-center justify-between mb-3">
                        <h4 className="text-xs font-semibold text-cyan-400 flex items-center space-x-2">
                          <Code className="w-3 h-3" />
                          <span>Logic Descriptor</span>
                        </h4>
                        <div className="flex space-x-2">
                          <Button size="sm" variant="outline" className="h-7">
                            <Shield className="w-3 h-3 mr-1" />
                            Audit
                          </Button>
                          <Button size="sm" className="h-7">
                            <Play className="w-3 h-3 mr-1" />
                            Invoke
                          </Button>
                        </div>
                      </div>
                      <div className="bg-black/40 rounded-lg p-4 font-mono text-sm h-full overflow-auto">
                        {selectedSolution ? (
                          <div className="space-y-3">
                            <div className="text-blue-400">package main</div>
                            <div className="text-blue-400">import "github.com/knirv/sdk"</div>
                            <br />
                            <div>func Resolve(ctx *sdk.Context) error {'{'}</div>
                            <div className="pl-4 text-green-400">
                              return sdk.ResolveError("{solutions.find(s => s.id === selectedSolution)?.error_id}")
                            </div>
                            <div>{'}'}</div>
                          </div>
                        ) : (
                          <div className="h-full flex items-center justify-center text-muted-foreground text-xs">
                            Select a solution node from the vault
                          </div>
                        )}
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
