'use client';

import React, { useState, useEffect, useRef, useCallback } from 'react';
import { X, Terminal, Play, Globe, Settings, BarChart3, FileText, Radio, Shield, Wifi, Users, ArrowLeft, ExternalLink, Loader2, AlertCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { API_BASE_URL, getAuthHeaders } from '@/lib/api';

interface P2PPeer {
  id: string;
  address: string;
  role: string;
  status: string;
  latency_ms: number;
}

interface P2PTransportAccessModalProps {
  isOpen: boolean;
  onClose: () => void;
}

// The gateway WebGUI is served via the wrapper's /gateway proxy route
// (registered in main.go as registerGatewayPrefix("/gateway", "/explorer")).
// This avoids scanning a range of TCP ports for the gateway binary.
const GATEWAY_PROXY_PATH = '/gateway';
let cachedGatewayPort: string | null = null;

async function checkHealth(port: string): Promise<boolean> {
  try {
    const resp = await fetch(`http://localhost:${port}/health`, {
      method: 'HEAD',
      signal: AbortSignal.timeout(2000),
    });
    return resp.ok || resp.type === 'opaque';
  } catch {
    return false;
  }
}

async function scanForGateway(): Promise<string> {
  if (cachedGatewayPort) return cachedGatewayPort;

  // Try NEXT_PUBLIC_GATEWAY_PORT first if set
  const envPort = process.env.NEXT_PUBLIC_GATEWAY_PORT;
  if (envPort) {
    const ok = await checkHealth(envPort);
    if (ok) {
      cachedGatewayPort = envPort;
      return cachedGatewayPort;
    }
  }

  // Check the current page's origin (port 8090 — the wrapper) first.
  // The wrapper always serves a /health endpoint.
  const currentPort = String(window.location.port || (window.location.protocol === 'https:' ? '443' : '80'));
  if (await checkHealth(currentPort)) {
    cachedGatewayPort = currentPort;
    return cachedGatewayPort;
  }

  // Fallback: known gateway config ports
  const knownPorts = ['8081', '8080', '8090'];
  for (const port of knownPorts) {
    if (await checkHealth(port)) {
      cachedGatewayPort = port;
      return cachedGatewayPort;
    }
  }

  // Fallback to default
  return '8090';
}

function buildWebGuiUrl(port: string, path?: string): string {
  // Use the wrapper's /gateway proxy route which forwards to the gateway's
  // explorer interface (registered in main.go line 927).
  if (port === String(window.location.port || '80') || port === '8090') {
    return path
      ? `http://localhost:${port}/gateway/${path.replace(/^\//, '')}`
      : `http://localhost:${port}/gateway`;
  }
  return path
    ? `http://localhost:${port}${path}`
    : `http://localhost:${port}/dashboard`;
}

export function P2PTransportAccessModal({ isOpen, onClose }: P2PTransportAccessModalProps) {
  const [terminalOutput, setTerminalOutput] = useState([
    '$ Welcome to P2P Transport Terminal',
    '$ Service: Secure NAT Traversal & Peer Connectivity',
    '$ Type "help" for available commands',
    '$ '
  ]);
  const [currentCommand, setCurrentCommand] = useState('');
  const [isExecuting, setIsExecuting] = useState(false);
  const [showPeers, setShowPeers] = useState(false);
  const [showRelay, setShowRelay] = useState(false);
  const [showNAT, setShowNAT] = useState(false);
  const [peers, setPeers] = useState<P2PPeer[]>([]);
  const [peersLoading, setPeersLoading] = useState(false);
  const [showWebGui, setShowWebGui] = useState(false);
  const [webGuiLoading, setWebGuiLoading] = useState(false);
  const [webGuiError, setWebGuiError] = useState(false);
  const [webGuiUrl, setWebGuiUrl] = useState<string>('');
  const terminalEndRef = useRef<HTMLDivElement>(null);

  // Initialize WebGUI URL when modal opens
  useEffect(() => {
    if (isOpen && !webGuiUrl) {
      scanForGateway().then(port => {
        setWebGuiUrl(buildWebGuiUrl(port));
      });
    }
  }, [isOpen]);

  // Pre-flight check: ping the gateway before opening the iframe so the user
  // sees an actionable error message instead of a blank black screen.
  const openWebGui = useCallback(async () => {
    setWebGuiError(false);
    setWebGuiLoading(true);
    setShowWebGui(true);
    try {
      const port = await scanForGateway();
      const baseUrl = `http://localhost:${port}`;
      const resp = await fetch(`${baseUrl}/health`, { method: 'HEAD', mode: 'no-cors', signal: AbortSignal.timeout(4000) });
      // no-cors fetch succeeds even for opaque responses — means server is reachable
      void resp;
      setWebGuiLoading(false);
    } catch {
      setWebGuiLoading(false);
      setWebGuiError(true);
    }
  }, []);

  useEffect(() => {
    if (!isOpen) return;
    setPeersLoading(true);
    fetch(`${API_BASE_URL}/api/p2p/peers`, { headers: getAuthHeaders() })
      .then(r => r.json())
      .then(data => setPeers(data.peers ?? []))
      .catch(() => setPeers([]))
      .finally(() => setPeersLoading(false));
  }, [isOpen]);

  useEffect(() => {
    terminalEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [terminalOutput]);

  const workflowTemplates = [
    {
      id: 'connect-peer',
      name: 'Connect Peer',
      description: 'Establish secure P2P connection with new peer',
      icon: <Users className="w-4 h-4" />,
      commands: ['peer-discover', 'hole-punch', 'secure-channel']
    },
    {
      id: 'relay-config',
      name: 'Configure Relay',
      description: 'Configure TURN/STUN relay settings',
      icon: <Radio className="w-4 h-4" />,
      commands: ['relay-select', 'nat-detect', 'allocate-port']
    },
    {
      id: 'diagnose',
      name: 'Network Diagnosis',
      description: 'Run network diagnostics and connectivity tests',
      icon: <Wifi className="w-4 h-4" />,
      commands: ['nat-type', 'connectivity-test', 'bandwidth-check']
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
        '  status      - Show transport status',
        '  peers       - List connected peers',
        '  relay       - Show relay status',
        '  clear       - Clear terminal',
        '  <command>   - Execute via knirvshell p2p',
        '$ '
      ]);
      return;
    }

    if (trimmed.toLowerCase() === 'peers') {
      setTerminalOutput(prev => [
        ...prev,
        `Connected Peers: ${peers.length}`,
        ...(peers.length > 0
          ? peers.map(p => `  ${p.id.slice(0, 12)}: ${p.address} [${p.role || 'peer'}, ${p.latency_ms}ms]`)
          : ['  No connected peers found']),
        '$ '
      ]);
      return;
    }

    // Execute via backend knirvshell p2p API
    setIsExecuting(true);
    try {
      const resp = await fetch(`${API_BASE_URL}/api/v1/cli/p2p/execute`, {
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

  // WebGUI view — overlays the entire modal panel
  if (showWebGui) {
    return (
      <div className="fixed inset-0 z-50 flex flex-col bg-[#0a0f1e]">
        {/* Header bar */}
        <div className="flex items-center justify-between px-4 py-2 border-b border-indigo-900/40 bg-[#0a0f1e]/95 backdrop-blur-sm shrink-0">
          <div className="flex items-center space-x-3">
            <Button variant="ghost" size="sm" onClick={() => { setShowWebGui(false); setWebGuiError(false); }}>
              <ArrowLeft className="w-4 h-4 mr-1" />
              Back
            </Button>
            <span className="text-sm font-medium text-indigo-300">KNIRV Network WebGUI</span>
            <Badge variant="outline" className="text-xs text-indigo-400 border-indigo-700">
              {webGuiUrl}
            </Badge>
          </div>
          <div className="flex items-center space-x-1">
            <a href={webGuiUrl} target="_blank" rel="noopener noreferrer">
              <Button variant="ghost" size="sm" title="Open in browser">
                <ExternalLink className="w-4 h-4" />
              </Button>
            </a>
            <Button variant="ghost" size="sm" onClick={onClose} title="Close">
              <X className="w-4 h-4" />
            </Button>
          </div>
        </div>

        {/* Content area */}
        <div className="flex-1 relative overflow-hidden">
          {webGuiLoading && (
            <div className="absolute inset-0 flex flex-col items-center justify-center gap-3 bg-[#0a0f1e] z-10">
              <Loader2 className="w-8 h-8 animate-spin text-indigo-400" />
              <span className="text-sm text-indigo-300">Connecting to gateway...</span>
              <span className="text-xs text-muted-foreground">{webGuiUrl}</span>
            </div>
          )}

          {!webGuiLoading && webGuiError && (
            <div className="absolute inset-0 flex flex-col items-center justify-center gap-4 bg-[#0a0f1e] z-10">
              <AlertCircle className="w-10 h-10 text-amber-400" />
              <div className="text-center space-y-1">
                <p className="text-sm font-medium text-amber-300">Gateway not reachable</p>
                <p className="text-xs text-muted-foreground">
                  No response at <span className="font-mono text-indigo-400">{webGuiUrl}</span>
                </p>
                <p className="text-xs text-muted-foreground">
                  Ensure the KNIRVGATEWAY service is running and <code className="text-indigo-400">gateway.enabled = true</code> in config.
                </p>
              </div>
              <div className="flex gap-2">
                <Button size="sm" variant="outline" onClick={openWebGui}>
                  Retry
                </Button>
                <a href={webGuiUrl} target="_blank" rel="noopener noreferrer">
                  <Button size="sm" variant="ghost">
                    <ExternalLink className="w-3 h-3 mr-1" />
                    Open in browser
                  </Button>
                </a>
              </div>
            </div>
          )}

          {!webGuiError && (
            <iframe
              src={webGuiLoading ? undefined : webGuiUrl}
              className="w-full h-full border-0"
              title="KNIRV Network WebGUI"
              onLoad={() => setWebGuiLoading(false)}
              onError={() => { setWebGuiLoading(false); setWebGuiError(true); }}
            />
          )}
        </div>
      </div>
    );
  }

  return (
    <div className="fixed inset-0 z-50 flex">
      <div className="flex-1 bg-black/20 backdrop-blur-sm transition-colors duration-300 z-40" onClick={onClose} />

      <div className="relative w-full max-w-4xl bg-background border-l shadow-2xl transform transition-slide duration-300 ease-in-out">
        <div className="flex flex-col h-full">
          <div className="flex items-center justify-between p-6 border-b">
            <div>
              <h2 className="text-2xl font-bold">KNIRVGATEWAY Access</h2>
              <p className="text-muted-foreground">
                Secure NAT Traversal & Peer Connectivity
              </p>
            </div>
            <div className="flex items-center space-x-2">
              <Button variant="outline" size="sm" onClick={openWebGui}>
                <Globe className="w-4 h-4 mr-1" />
                WebGUI
              </Button>
              <Badge variant="secondary">Active</Badge>
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
                      <span>P2P Transport Terminal</span>
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
                        <Users className="w-4 h-4" />
                        <span>Peers {!peersLoading && <span className="text-muted-foreground">({peers.length})</span>}</span>
                      </CardTitle>
                      <CardDescription className="text-xs">
                        {peersLoading ? 'Loading...' : 'Connected peer list'}
                      </CardDescription>
                    </CardHeader>
                    <CardContent className="space-y-1">
                      <Button variant={showPeers ? "default" : "outline"} size="sm" className="w-full" onClick={() => setShowPeers(!showPeers)}>
                        {showPeers ? 'Hide' : 'View'}
                      </Button>
                      {showPeers && (
                        <div className="text-xs font-mono space-y-1 mt-2 max-h-32 overflow-y-auto">
                          {peers.length === 0 ? (
                            <div className="text-muted-foreground">No peers connected</div>
                          ) : peers.map(p => (
                            <div key={p.id} className="flex justify-between">
                              <span className="text-blue-400 truncate">{p.id.slice(0, 12)}</span>
                              <span className="text-muted-foreground ml-2">{p.latency_ms}ms</span>
                            </div>
                          ))}
                        </div>
                      )}
                    </CardContent>
                  </Card>

                  <Card className="knirv-card-gradient">
                    <CardHeader className="pb-2">
                      <CardTitle className="flex items-center space-x-2 text-sm">
                        <Radio className="w-4 h-4" />
                        <span>Relay</span>
                      </CardTitle>
                      <CardDescription className="text-xs">
                        TURN relay status
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      <Button variant={showRelay ? "default" : "outline"} size="sm" className="w-full" onClick={() => setShowRelay(!showRelay)}>
                        {showRelay ? 'Hide' : 'View'}
                      </Button>
                    </CardContent>
                  </Card>

                  <Card className="knirv-card-gradient">
                    <CardHeader className="pb-2">
                      <CardTitle className="flex items-center space-x-2 text-sm">
                        <Globe className="w-4 h-4" />
                        <span>NAT</span>
                      </CardTitle>
                      <CardDescription className="text-xs">
                        NAT traversal config
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      <Button variant={showNAT ? "default" : "outline"} size="sm" className="w-full" onClick={() => setShowNAT(!showNAT)}>
                        {showNAT ? 'Hide' : 'Configure'}
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
                        Network metrics
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      <Button variant="outline" size="sm" className="w-full" onClick={openWebGui}>
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
