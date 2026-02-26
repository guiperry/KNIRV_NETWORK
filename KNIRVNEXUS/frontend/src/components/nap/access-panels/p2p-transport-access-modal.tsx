'use client';

import React, { useState } from 'react';
import { X, Terminal, Play, Globe, Settings, BarChart3, FileText, Radio, Shield, Wifi, Users } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';

interface P2PTransportAccessModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export function P2PTransportAccessModal({ isOpen, onClose }: P2PTransportAccessModalProps) {
  const [terminalOutput, setTerminalOutput] = useState([
    '$ Welcome to P2P Transport Terminal',
    '$ Service: Secure NAT Traversal & Peer Connectivity',
    '$ Type "help" for available commands',
    '$ '
  ]);
  const [currentCommand, setCurrentCommand] = useState('');
  const [showPeers, setShowPeers] = useState(false);
  const [showRelay, setShowRelay] = useState(false);
  const [showNAT, setShowNAT] = useState(false);

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

  const executeCommand = (command: string) => {
    const newOutput = [...terminalOutput];
    newOutput.push('$ ' + command);
    
    setTimeout(() => {
      switch (command.toLowerCase()) {
        case 'help':
          newOutput.push('Available commands:');
          newOutput.push('  help - Show this help message');
          newOutput.push('  status - Show transport status');
          newOutput.push('  peers - List connected peers');
          newOutput.push('  relay - Show relay status');
          newOutput.push('  clear - Clear terminal');
          break;
        case 'status':
          newOutput.push('Transport Status: Active');
          newOutput.push('TURN Relay: Active (BK-4)');
          newOutput.push('NAT Type: Full Cone');
          newOutput.push('Connected Peers: 24');
          break;
        case 'peers':
          newOutput.push('Connected Peers: 24');
          newOutput.push('  Peer-001: 192.168.1.101:45678');
          newOutput.push('  Peer-002: 10.0.0.52:39281');
          newOutput.push('  Peer-003: 172.16.0.88:54321');
          break;
        case 'relay':
          newOutput.push('TURN Relay: Active');
          newOutput.push('  Relay ID: BK-4');
          newOutput.push('  Public IP: 203.0.113.45');
          newOutput.push('  Port Range: 49152-49172');
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
              <h2 className="text-2xl font-bold">P2P Transport Access</h2>
              <p className="text-muted-foreground">
                Secure NAT Traversal & Peer Connectivity
              </p>
            </div>
            <div className="flex items-center space-x-2">
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
                        <Users className="w-4 h-4" />
                        <span>Peers</span>
                      </CardTitle>
                      <CardDescription className="text-xs">
                        Connected peer list
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      <Button variant={showPeers ? "default" : "outline"} size="sm" className="w-full" onClick={() => setShowPeers(!showPeers)}>
                        {showPeers ? 'Hide' : 'View'}
                      </Button>
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
