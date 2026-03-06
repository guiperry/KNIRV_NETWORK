'use client';

import React, { useState } from 'react';
import { X, Terminal, Play, Network, Settings, BarChart3, FileText, GitBranch, Search, Cpu, Zap, Shield, Clock, ShieldCheck } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { ScrollArea } from '@/components/ui/scroll-area';

interface KNIRVGraphAccessModalProps {
  isOpen: boolean;
  onClose: () => void;
}

interface Trace {
  id: string;
  agent_id: string;
  error_id: string;
  timestamp: string;
  type: string;
}

export function KNIRVGraphAccessModal({ isOpen, onClose }: KNIRVGraphAccessModalProps) {
  const [terminalOutput, setTerminalOutput] = useState([
    '$ Welcome to KNIRVGRAPH Terminal',
    '$ Service: Reasoning Graph - Context Knowledge Chain',
    '$ Type "help" for available commands',
    '$ '
  ]);
  const [currentCommand, setCurrentCommand] = useState('');
  const [showQuery, setShowQuery] = useState(false);
  const [showTraces, setShowTraces] = useState(false);
  const [showConsensus, setShowConsensus] = useState(false);
  const [showReasoningExplorer, setShowReasoningExplorer] = useState(false);
  const [selectedTrace, setSelectedTrace] = useState<string | null>(null);

  const [traces] = useState<Trace[]>([
    { id: 'trace_1740000000', agent_id: 'agent-alpha', error_id: 'err_992', timestamp: '2026-02-16T14:30:00Z', type: 'TRACE' },
    { id: 'trace_1740000001', agent_id: 'agent-beta', error_id: 'err_995', timestamp: '2026-02-16T14:35:00Z', type: 'TRACE' },
    { id: 'trace_1740000002', agent_id: 'agent-gamma', error_id: 'err_1021', timestamp: '2026-02-16T14:40:00Z', type: 'TRACE' },
    { id: 'trace_1740000003', agent_id: 'agent-delta', error_id: 'err_1045', timestamp: '2026-02-16T14:45:00Z', type: 'TRACE' },
    { id: 'trace_1740000004', agent_id: 'agent-epsilon', error_id: 'err_1088', timestamp: '2026-02-16T14:50:00Z', type: 'TRACE' },
  ]);

  const workflowTemplates = [
    {
      id: 'graph-query',
      name: 'Query Reasoning Graph',
      description: 'Search and retrieve context traces from the graph',
      icon: <Search className="w-4 h-4" />,
      commands: ['graph-query', 'trace-retrieve', 'context-build']
    },
    {
      id: 'verify-edge',
      name: 'Verify Edge',
      description: 'Verify graph edge through consensus mechanism',
      icon: <Shield className="w-4 h-4" />,
      commands: ['edge-select', 'consensus-request', 'verify-signature']
    },
    {
      id: 'reindex-graph',
      name: 'Re-index Graph',
      description: 'Rebuild graph index for optimal query performance',
      icon: <GitBranch className="w-4 h-4" />,
      commands: ['index-start', 'rebuild-edges', 'optimize-paths']
    }
  ];

  const executeCommand = (command: string) => {
    const newOutput = [...terminalOutput];
    newOutput.push('$ ' + command);
    
    setTimeout(() => {
      switch (command.toLowerCase()) {
        case 'help':
          newOutput.push('Available commands:');
          newOutput.push('  help - Show this help message');
          newOutput.push('  status - Show graph status');
          newOutput.push('  query <id> - Query trace by ID');
          newOutput.push('  edges - List graph edges');
          newOutput.push('  clear - Clear terminal');
          break;
        case 'status':
          newOutput.push('Graph Status: Synchronized');
          newOutput.push('Total Traces: 142');
          newOutput.push('Total Edges: 1,847');
          newOutput.push('Consensus: Quorum reached');
          break;
        case 'edges':
          newOutput.push('Loading graph edges...');
          newOutput.push('Edge: NRV-8472 → NRV-1203 (verified)');
          newOutput.push('Edge: NRV-3321 → NRV-9902 (verified)');
          newOutput.push('Edge: NRV-5512 → NRV-7721 (pending)');
          break;
        case 'clear':
          setTerminalOutput(['$ ']);
          return;
        default:
          newOutput.push('Command not found: ' + command);
      }
      newOutput.push('$ ');
      setTerminalOutput(newOutput);
    }, 500);
    
    setTerminalOutput(newOutput);
    setCurrentCommand('');
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex">
      <div className="flex-1 bg-black/20 backdrop-blur-sm transition-all duration-300 z-40" onClick={onClose} />
      
      <div className="relative w-full max-w-4xl bg-background border-l shadow-2xl transform transition-all duration-300 ease-in-out">
        <div className="flex flex-col h-full">
          <div className="flex items-center justify-between p-6 border-b">
            <div>
              <h2 className="text-2xl font-bold">KNIRVGRAPH Access</h2>
              <p className="text-muted-foreground">
                Reasoning Graph - Context Knowledge Chain
              </p>
            </div>
            <div className="flex items-center space-x-2">
              <Badge variant="secondary">Synced</Badge>
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
                      <span>KNIRVGRAPH Terminal</span>
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="bg-black rounded-lg p-4 font-mono text-sm">
                      <div className="text-blue-400 space-y-1 max-h-96 overflow-y-auto">
                        {terminalOutput.map((line, index) => (
                          <div key={index}>{line}</div>
                        ))}
                      </div>
                      <div className="flex items-center mt-2">
                        <span className="text-blue-400">$ </span>
                        <input
                          type="text"
                          value={currentCommand}
                          onChange={(e) => setCurrentCommand(e.target.value)}
                          onKeyPress={(e) => {
                            if (e.key === 'Enter' && currentCommand.trim()) {
                              executeCommand(currentCommand.trim());
                            }
                          }}
                          className="flex-1 bg-transparent text-blue-400 outline-none ml-2"
                          placeholder="Enter command..."
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
                        <Button variant="outline" size="sm" className="w-full">
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
                        <Search className="w-4 h-4" />
                        <span>Query</span>
                      </CardTitle>
                      <CardDescription className="text-xs">
                        Graph query engine
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      <Button variant={showQuery ? "default" : "outline"} size="sm" className="w-full" onClick={() => setShowQuery(!showQuery)}>
                        {showQuery ? 'Hide' : 'Open'}
                      </Button>
                    </CardContent>
                  </Card>

                  <Card className="knirv-card-gradient">
                    <CardHeader className="pb-2">
                      <CardTitle className="flex items-center space-x-2 text-sm">
                        <GitBranch className="w-4 h-4" />
                        <span>Reasoning Explorer</span>
                      </CardTitle>
                      <CardDescription className="text-xs">
                        Encrypted trace viewer
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      <Button 
                        variant={showReasoningExplorer ? "default" : "outline"} 
                        size="sm" 
                        className="w-full"
                        onClick={() => setShowReasoningExplorer(!showReasoningExplorer)}
                      >
                        <Zap className="w-3 h-3 mr-1" />
                        {showReasoningExplorer ? 'Hide' : 'Open'}
                      </Button>
                    </CardContent>
                  </Card>

                  <Card className="knirv-card-gradient">
                    <CardHeader className="pb-2">
                      <CardTitle className="flex items-center space-x-2 text-sm">
                        <Shield className="w-4 h-4" />
                        <span>Consensus</span>
                      </CardTitle>
                      <CardDescription className="text-xs">
                        Verify edges via consensus
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      <Button variant={showConsensus ? "default" : "outline"} size="sm" className="w-full" onClick={() => setShowConsensus(!showConsensus)}>
                        {showConsensus ? 'Hide' : 'Run'}
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
                        Graph statistics
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

            {/* Reasoning Explorer Panel - Slides out from left side like DVE Solver */}
            {showReasoningExplorer && (
              <div className="fixed left-0 top-0 bottom-0 z-[60] pointer-events-auto transform transition-all duration-300 ease-out translate-x-0 w-96">
                <div className="h-full bg-slate-900 rounded-r-lg border-r border-blue-600/30 shadow-2xl">
                  <div className="flex items-center justify-between p-4 border-b border-blue-600/30">
                    <div className="flex items-center space-x-2">
                      <GitBranch className="w-4 h-4 text-cyan-400" />
                      <h3 className="font-semibold text-sm">Reasoning Explorer</h3>
                    </div>
                    <button
                      onClick={() => setShowReasoningExplorer(false)}
                      className="p-1 hover:bg-slate-800 rounded transition-colors">
                      <X className="w-4 h-4" />
                    </button>
                  </div>
                  
                  <div className="flex flex-col h-[calc(100%-56px)]">
                    {/* Traces List */}
                    <div className="p-4 border-b border-blue-600/30">
                      <h4 className="text-xs font-semibold text-cyan-400 mb-3 flex items-center space-x-2">
                        <Clock className="w-3 h-3" />
                        <span>Reasoning Traces</span>
                      </h4>
                      <ScrollArea className="h-48">
                        <div className="space-y-2">
                          {traces.map((trace) => (
                            <div
                              key={trace.id}
                              className={`p-3 rounded-lg border cursor-pointer transition-colors ${
                                selectedTrace === trace.id ? 'bg-blue-600/20 border-blue-600/50' : 'bg-slate-800/50 border-blue-600/20 hover:border-blue-600/50'
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
                    </div>
                    
                    {/* Trace Explorer */}
                    <div className="flex-1 p-4 overflow-auto">
                      <h4 className="text-xs font-semibold text-cyan-400 mb-3 flex items-center space-x-2">
                        <FileText className="w-3 h-3" />
                        <span>Trace Explorer</span>
                      </h4>
                      <div className="bg-black/40 rounded-lg p-4 font-mono text-sm h-full overflow-auto">
                        {selectedTrace ? (
                          <div className="space-y-3">
                            <div className="flex items-center justify-between">
                              <span className="text-primary"># {selectedTrace}</span>
                              <ShieldCheck className="w-4 h-4 text-green-500" />
                            </div>
                            <div className="text-xs text-slate-400">
                              {traces.find(t => t.id === selectedTrace)?.agent_id && (
                                <div>**Agent:** {traces.find(t => t.id === selectedTrace)?.agent_id}</div>
                              )}
                              {traces.find(t => t.id === selectedTrace)?.error_id && (
                                <div>**Error ID:** {traces.find(t => t.id === selectedTrace)?.error_id}</div>
                              )}
                            </div>
                            <div className="border-l-2 border-primary/30 pl-4 space-y-2 text-xs text-slate-400">
                              <div>1. Detected: API Connection Timeout</div>
                              <div>2. Searching Vault for compatible solutions...</div>
                              <div>3. Found SolutionNode: sol_network_v1</div>
                              <div>4. Verifying solution integrity via PQC signature...</div>
                              <div className="text-green-500">5. Result: Success</div>
                            </div>
                          </div>
                        ) : (
                          <div className="h-full flex items-center justify-center text-muted-foreground text-xs">
                            Select a trace to view reasoning details
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
