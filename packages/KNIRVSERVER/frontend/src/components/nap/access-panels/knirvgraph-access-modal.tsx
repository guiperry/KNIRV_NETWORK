'use client';

import React, { useState, useRef, useEffect, useCallback } from 'react';
import { X, Terminal, Play, Network, Settings, BarChart3, FileText, GitBranch, Search, Cpu, Zap, Shield, Clock, ShieldCheck, Layers, BookOpen, Code2, Upload, Loader2, ExternalLink, RefreshCw, AlertTriangle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { ScrollArea } from '@/components/ui/scroll-area';
import { API_BASE_URL, getAuthHeaders } from '@/lib/api';

const GITNEXUS_PORT = process.env.NEXT_PUBLIC_GITNEXUS_PORT || '8091';
const GITNEXUS_URL = `http://localhost:${GITNEXUS_PORT}`;

const GRAPHRAG_PORT = process.env.NEXT_PUBLIC_GRAPHRAG_PORT || '8092';
const GRAPHRAG_URL = `http://localhost:${GRAPHRAG_PORT}`;

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
  const [isExecuting, setIsExecuting] = useState(false);
  const [showQuery, setShowQuery] = useState(false);
  const [showTraces, setShowTraces] = useState(false);
  const [showConsensus, setShowConsensus] = useState(false);
  const [showReasoningExplorer, setShowReasoningExplorer] = useState(false);
  const [selectedTrace, setSelectedTrace] = useState<string | null>(null);
  const terminalEndRef = useRef<HTMLDivElement>(null);

  // Knowledge Graph (GitNexus + graphrag-rs) state
  const [gitNexusMode, setGitNexusMode] = useState<'idle' | 'loading' | 'open' | 'error'>('idle');
  const [graphRagMode, setGraphRagMode] = useState<'idle' | 'loading' | 'open' | 'error'>('idle');
  const [graphRagQuery, setGraphRagQuery] = useState('');
  const [graphRagResults, setGraphRagResults] = useState<string[]>([]);
  const [isQuerying, setIsQuerying] = useState(false);
  const [ingestUrl, setIngestUrl] = useState('');
  const [isIngesting, setIsIngesting] = useState(false);
  const [ingestLog, setIngestLog] = useState<string[]>([]);

  const openGitNexus = useCallback(async () => {
    setGitNexusMode('loading');
    try {
      await fetch(`${GITNEXUS_URL}/health`, { method: 'HEAD', mode: 'no-cors', signal: AbortSignal.timeout(4000) });
      setGitNexusMode('open');
    } catch {
      setGitNexusMode('error');
    }
  }, []);

  const openGraphRag = useCallback(async () => {
    setGraphRagMode('loading');
    try {
      await fetch(`${GRAPHRAG_URL}/health`, { method: 'HEAD', mode: 'no-cors', signal: AbortSignal.timeout(4000) });
      setGraphRagMode('open');
    } catch {
      setGraphRagMode('error');
    }
  }, []);

  const handleGraphRagQuery = async () => {
    if (!graphRagQuery.trim() || isQuerying) return;
    setIsQuerying(true);
    try {
      const resp = await fetch(`${API_BASE_URL}/api/knirvgraph/graphrag/query`, {
        method: 'POST',
        headers: { ...getAuthHeaders(), 'Content-Type': 'application/json' },
        body: JSON.stringify({ query: graphRagQuery.trim(), top_k: 5 }),
      });
      const data = await resp.json();
      if (resp.ok && Array.isArray(data.results)) {
        setGraphRagResults(data.results);
      } else {
        setGraphRagResults([data.error || data.message || 'No results returned.']);
      }
    } catch {
      setGraphRagResults(['Error: Could not reach graphrag-rs backend.']);
    } finally {
      setIsQuerying(false);
    }
  };

  const handleIngestRepo = async () => {
    if (!ingestUrl.trim() || isIngesting) return;
    setIsIngesting(true);
    setIngestLog(prev => [...prev, `[${new Date().toLocaleTimeString()}] Ingesting: ${ingestUrl.trim()}`]);
    try {
      const resp = await fetch(`${API_BASE_URL}/api/knirvgraph/gitnexus/ingest`, {
        method: 'POST',
        headers: { ...getAuthHeaders(), 'Content-Type': 'application/json' },
        body: JSON.stringify({ repo_url: ingestUrl.trim() }),
      });
      const data = await resp.json();
      if (resp.ok) {
        setIngestLog(prev => [...prev, `[${new Date().toLocaleTimeString()}] ✓ ${data.message || 'Ingestion queued'}`]);
        setIngestUrl('');
      } else {
        setIngestLog(prev => [...prev, `[${new Date().toLocaleTimeString()}] ✗ ${data.error || 'Ingestion failed'}`]);
      }
    } catch {
      setIngestLog(prev => [...prev, `[${new Date().toLocaleTimeString()}] ✗ Network error — backend unreachable`]);
    } finally {
      setIsIngesting(false);
    }
  };

  useEffect(() => {
    terminalEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [terminalOutput]);

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
      workflow_id: 'graph-query',
      steps: [
        { command: 'graph-query', args: {} },
        { command: 'trace-retrieve', args: {} },
        { command: 'context-build', args: {} }
      ]
    },
    {
      id: 'verify-edge',
      name: 'Verify Edge',
      description: 'Verify graph edge through consensus mechanism',
      icon: <Shield className="w-4 h-4" />,
      workflow_id: 'verify-edge',
      steps: [
        { command: 'edge-select', args: {} },
        { command: 'consensus-request', args: {} },
        { command: 'verify-signature', args: {} }
      ]
    },
    {
      id: 'reindex-graph',
      name: 'Re-index Graph',
      description: 'Rebuild graph index for optimal query performance',
      icon: <GitBranch className="w-4 h-4" />,
      workflow_id: 'reindex-graph',
      steps: [
        { command: 'index-start', args: {} },
        { command: 'rebuild-edges', args: {} },
        { command: 'optimize-paths', args: {} }
      ]
    }
  ];

  const executeWorkflow = async (template: typeof workflowTemplates[0]) => {
    setIsExecuting(true);
    setTerminalOutput(prev => [...prev, `$ Executing workflow: ${template.name}`, '$ ']);
    
    try {
      const resp = await fetch(`${API_BASE_URL}/api/workflow/execute`, {
        method: 'POST',
        headers: { ...getAuthHeaders(), 'Content-Type': 'application/json' },
        body: JSON.stringify({
          workflow_id: template.workflow_id,
          node_id: '',
          steps: template.steps,
        }),
      });
      
      const data = await resp.json();
      
      if (resp.ok) {
        setTerminalOutput(prev => [
          ...prev,
          `Workflow initiated: ${template.name}`,
          `Execution ID: ${data.execution_id || 'N/A'}`,
          `Status: ${data.status || 'running'}`,
          '$ '
        ]);
      } else {
        setTerminalOutput(prev => [
          ...prev,
          `Error: ${data.error || 'Workflow execution failed'}`,
          '$ '
        ]);
      }
    } catch {
      setTerminalOutput(prev => [...prev, 'Error: Failed to reach workflow service', '$ ']);
    } finally {
      setIsExecuting(false);
    }
  };

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
        '  status      - Show graph status',
        '  query <id>  - Query trace by ID',
        '  edges       - List graph edges',
        '  clear       - Clear terminal',
        '  <command>   - Execute via knirvshell',
        'Workflows:   - Use the Workflows tab to execute workflows',
        '$ '
      ]);
      return;
    }

    setIsExecuting(true);
    try {
      const resp = await fetch(`${API_BASE_URL}/api/v1/cli/execute`, {
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
                <TabsList className="grid w-full grid-cols-4">
                  <TabsTrigger value="terminal">Terminal</TabsTrigger>
                  <TabsTrigger value="workflows">Workflows</TabsTrigger>
                  <TabsTrigger value="tools">Tools</TabsTrigger>
                  <TabsTrigger value="knowledge-graph" className="flex items-center gap-1">
                    <Layers className="w-3 h-3" />
                    Knowledge Graph
                  </TabsTrigger>
                </TabsList>
              </div>

              <TabsContent value="terminal" className="px-6 pb-6 space-y-4">
                <Card className="knirv-card-gradient">
                  <CardHeader>
                    <CardTitle className="flex items-center space-x-2">
                      <Terminal className="w-5 h-5" />
                      <span>KNIRVGRAPH Terminal</span>
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
                          Steps: {template.steps.map(s => s.command).join(' → ')}
                        </div>
                        <Button
                          variant="outline"
                          size="sm"
                          className="w-full"
                          disabled={isExecuting}
                          onClick={() => executeWorkflow(template)}
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

              {/* Knowledge Graph tab */}
              <TabsContent value="knowledge-graph" className="px-6 pb-6 space-y-6">
                {/* GitNexus - Codebase Ingestion */}
                <Card className="knirv-card-gradient">
                  <CardHeader className="pb-3">
                    <CardTitle className="flex items-center justify-between">
                      <div className="flex items-center space-x-2 text-sm">
                        <Code2 className="w-4 h-4 text-blue-400" />
                        <span>GitNexus — Codebase Ingestion</span>
                      </div>
                      <div className="flex items-center gap-2">
                        <a href="https://github.com/abhigyanpatwari/GitNexus" target="_blank" rel="noreferrer" className="text-[10px] text-blue-400 hover:underline flex items-center gap-1">
                          <ExternalLink className="w-3 h-3" /> GitHub
                        </a>
                        <Button size="sm" variant="outline" className="h-7 text-xs" onClick={openGitNexus}>
                          {gitNexusMode === 'loading' ? <Loader2 className="w-3 h-3 animate-spin mr-1" /> : <Layers className="w-3 h-3 mr-1" />}
                          Open UI
                        </Button>
                      </div>
                    </CardTitle>
                    <CardDescription className="text-xs">
                      Ingest Git repositories into the knowledge graph for code-aware reasoning
                    </CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-3">
                    <div className="flex gap-2">
                      <input
                        type="text"
                        value={ingestUrl}
                        onChange={e => setIngestUrl(e.target.value)}
                        onKeyDown={e => { if (e.key === 'Enter') handleIngestRepo(); }}
                        placeholder="https://github.com/org/repo"
                        className="flex-1 bg-slate-950 border border-slate-700 rounded px-3 py-1.5 text-xs font-mono text-slate-200 focus:outline-none focus:ring-1 focus:ring-blue-500"
                      />
                      <Button size="sm" variant="default" onClick={handleIngestRepo} disabled={isIngesting || !ingestUrl.trim()}>
                        {isIngesting ? <Loader2 className="w-3 h-3 animate-spin mr-1" /> : <Upload className="w-3 h-3 mr-1" />}
                        Ingest
                      </Button>
                    </div>
                    {ingestLog.length > 0 && (
                      <div className="bg-black/40 rounded p-2 font-mono text-[10px] text-green-400 max-h-24 overflow-y-auto space-y-0.5">
                        {ingestLog.map((line, i) => <div key={i}>{line}</div>)}
                      </div>
                    )}

                    {/* GitNexus embedded UI */}
                    {gitNexusMode !== 'idle' && (
                      <div className="rounded-lg border border-slate-700 overflow-hidden" style={{ height: '320px' }}>
                        {gitNexusMode === 'loading' && (
                          <div className="h-full flex flex-col items-center justify-center bg-slate-950 gap-2 text-muted-foreground text-xs">
                            <Loader2 className="w-6 h-6 animate-spin" />
                            Connecting to GitNexus at {GITNEXUS_URL}...
                          </div>
                        )}
                        {gitNexusMode === 'error' && (
                          <div className="h-full flex flex-col items-center justify-center bg-slate-950 gap-3 p-4 text-center">
                            <AlertTriangle className="w-8 h-8 text-yellow-500" />
                            <p className="text-xs text-slate-300">GitNexus not reachable at <span className="font-mono text-blue-400">{GITNEXUS_URL}</span></p>
                            <div className="flex gap-2">
                              <Button size="sm" variant="outline" onClick={openGitNexus}>
                                <RefreshCw className="w-3 h-3 mr-1" /> Retry
                              </Button>
                              <a href={GITNEXUS_URL} target="_blank" rel="noreferrer">
                                <Button size="sm" variant="default"><ExternalLink className="w-3 h-3 mr-1" /> Open Externally</Button>
                              </a>
                            </div>
                          </div>
                        )}
                        {gitNexusMode === 'open' && (
                          <iframe
                            src={GITNEXUS_URL}
                            className="w-full h-full border-0"
                            title="GitNexus — Codebase Knowledge Graph"
                            sandbox="allow-scripts allow-same-origin allow-forms allow-popups"
                            onError={() => setGitNexusMode('error')}
                          />
                        )}
                      </div>
                    )}
                  </CardContent>
                </Card>

                {/* graphrag-rs - Document Ingestion & Retrieval */}
                <Card className="knirv-card-gradient">
                  <CardHeader className="pb-3">
                    <CardTitle className="flex items-center justify-between">
                      <div className="flex items-center space-x-2 text-sm">
                        <BookOpen className="w-4 h-4 text-purple-400" />
                        <span>graphrag-rs — Document Retrieval</span>
                      </div>
                      <div className="flex items-center gap-2">
                        <a href="https://github.com/automataIA/graphrag-rs" target="_blank" rel="noreferrer" className="text-[10px] text-purple-400 hover:underline flex items-center gap-1">
                          <ExternalLink className="w-3 h-3" /> GitHub
                        </a>
                        <Button size="sm" variant="outline" className="h-7 text-xs" onClick={openGraphRag}>
                          {graphRagMode === 'loading' ? <Loader2 className="w-3 h-3 animate-spin mr-1" /> : <Layers className="w-3 h-3 mr-1" />}
                          Open UI
                        </Button>
                      </div>
                    </CardTitle>
                    <CardDescription className="text-xs">
                      Semantic graph-based document ingestion and retrieval via graphrag-rs
                    </CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-3">
                    {/* Query interface */}
                    <div className="flex gap-2">
                      <input
                        type="text"
                        value={graphRagQuery}
                        onChange={e => setGraphRagQuery(e.target.value)}
                        onKeyDown={e => { if (e.key === 'Enter') handleGraphRagQuery(); }}
                        placeholder="Ask the knowledge graph..."
                        className="flex-1 bg-slate-950 border border-slate-700 rounded px-3 py-1.5 text-xs text-slate-200 focus:outline-none focus:ring-1 focus:ring-purple-500"
                      />
                      <Button size="sm" variant="default" className="bg-purple-600 hover:bg-purple-700" onClick={handleGraphRagQuery} disabled={isQuerying || !graphRagQuery.trim()}>
                        {isQuerying ? <Loader2 className="w-3 h-3 animate-spin mr-1" /> : <Search className="w-3 h-3 mr-1" />}
                        Query
                      </Button>
                    </div>
                    {graphRagResults.length > 0 && (
                      <ScrollArea className="h-28 rounded border border-slate-700 bg-black/30 p-2">
                        <div className="space-y-1 text-xs font-mono">
                          {graphRagResults.map((r, i) => (
                            <div key={i} className="text-slate-300 leading-relaxed">{i + 1}. {r}</div>
                          ))}
                        </div>
                      </ScrollArea>
                    )}

                    {/* graphrag-rs embedded UI */}
                    {graphRagMode !== 'idle' && (
                      <div className="rounded-lg border border-slate-700 overflow-hidden" style={{ height: '280px' }}>
                        {graphRagMode === 'loading' && (
                          <div className="h-full flex flex-col items-center justify-center bg-slate-950 gap-2 text-muted-foreground text-xs">
                            <Loader2 className="w-6 h-6 animate-spin" />
                            Connecting to graphrag-rs at {GRAPHRAG_URL}...
                          </div>
                        )}
                        {graphRagMode === 'error' && (
                          <div className="h-full flex flex-col items-center justify-center bg-slate-950 gap-3 p-4 text-center">
                            <AlertTriangle className="w-8 h-8 text-yellow-500" />
                            <p className="text-xs text-slate-300">graphrag-rs not reachable at <span className="font-mono text-purple-400">{GRAPHRAG_URL}</span></p>
                            <div className="flex gap-2">
                              <Button size="sm" variant="outline" onClick={openGraphRag}>
                                <RefreshCw className="w-3 h-3 mr-1" /> Retry
                              </Button>
                              <a href={GRAPHRAG_URL} target="_blank" rel="noreferrer">
                                <Button size="sm" variant="default"><ExternalLink className="w-3 h-3 mr-1" /> Open Externally</Button>
                              </a>
                            </div>
                          </div>
                        )}
                        {graphRagMode === 'open' && (
                          <iframe
                            src={GRAPHRAG_URL}
                            className="w-full h-full border-0"
                            title="graphrag-rs — Document Knowledge Graph"
                            sandbox="allow-scripts allow-same-origin allow-forms allow-popups"
                            onError={() => setGraphRagMode('error')}
                          />
                        )}
                      </div>
                    )}
                  </CardContent>
                </Card>
              </TabsContent>
            </Tabs>

            {/* Reasoning Explorer Panel */}
            {showReasoningExplorer && (
              <div className="fixed left-0 top-0 bottom-0 z-[60] pointer-events-auto transform transition-slide duration-300 ease-out translate-x-0 w-96 gpu-accelerated">
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
