'use client';

import React, { useState } from 'react';
import { X, Terminal, Play, Code, Database, Settings, Cpu, Zap, FileText, Download, Share2, Radio, Shield, BarChart3, Upload, AlertTriangle, TestTube, Network, WifiOff, Loader2, Server } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Switch } from '@/components/ui/switch';
import { Label } from '@/components/ui/label';
import { useDemoMode } from '@/contexts/demo-mode-context';
import { useDHT } from '@/contexts/dht-context';
import { useToast } from '@/hooks/use-toast';
import { API_BASE_URL, getAuthHeaders } from '@/lib/api';

interface NetworkAccessModalProps {
  isOpen: boolean;
  onClose: () => void;
  nodeId: string;
  nodeName: string;
  onOpenKNIRVEngine: () => void;
  isModular?: boolean;
  onToggleMode?: () => void;
}

export function NetworkAccessModal({
  isOpen,
  onClose,
  nodeId,
  nodeName,
  onOpenKNIRVEngine,
  isModular,
  onToggleMode
}: NetworkAccessModalProps) {
  const [terminalOutput, setTerminalOutput] = useState([
    '$ Welcome to KNIRV Network Access Panel (NAP) Terminal',
    '$ Node: ' + nodeName + ' (' + nodeId + ')',
    '$ Type "help" for available commands',
    '$ '
  ]);
  const [currentCommand, setCurrentCommand] = useState('');
  const [showConsole, setShowConsole] = useState(false);
  const [showPolicy, setShowPolicy] = useState(false);
  const [showMonitor, setShowMonitor] = useState(false);
  const [showConnections, setShowConnections] = useState(false);
  const [showDVESolver, setShowDVESolver] = useState(false);
  
  // DVE Solver State
  const [selectedProblem, setSelectedProblem] = useState<string | null>(null);
  const [isValidating, setIsValidating] = useState(false);
  const [validationResults, setValidationResults] = useState<any>(null);
  const [showValidationReport, setShowValidationReport] = useState(false);

  // Admin Tab State
  const { isDemoMode, toggleDemoMode } = useDemoMode();
  const { isDHTEnabled, setDHTEnabled } = useDHT();
  const { toast } = useToast();
  const [isDHTUpdating, setIsDHTUpdating] = useState(false);
  const [wsConnected] = useState(true); // Simulated WebSocket status
  const [policyTeeMode, setPolicyTeeMode] = useState('SGX (Intel)');
  const [policyMaxMemory, setPolicyMaxMemory] = useState(4096);
  const [policyAttestation, setPolicyAttestation] = useState(true);
  const [isSavingPolicy, setIsSavingPolicy] = useState(false);
  const [policySaveStatus, setPolicySaveStatus] = useState<'idle' | 'success' | 'error'>('idle');
  const [dveNodeCount] = useState(8); // Simulated node count
  const [activeTasks] = useState(24); // Simulated active tasks

  const workflowTemplates = [
    {
      id: 'validation-setup',
      name: 'Validation Setup',
      description: 'Initialize validation environment with TEE configuration',
      icon: <Settings className="w-4 h-4" />,
      commands: ['tee-init', 'validation-config', 'security-check']
    },
    {
      id: 'fabric-deployment',
      name: 'Fabric Deployment',
      description: 'Deploy Agentic Memory Fabric to the cognitive engine',
      icon: <Cpu className="w-4 h-4" />,
      commands: ['fabric-load', 'inference-setup', 'performance-test']
    },
    {
      id: 'data-processing',
      name: 'Data Processing',
      description: 'Set up data pipelines and processing workflows',
      icon: <Database className="w-4 h-4" />,
      commands: ['pipeline-init', 'data-validate', 'process-start']
    },
    {
      id: 'performance-monitoring',
      name: 'Performance Monitoring',
      description: 'Monitor node performance and resource utilization',
      icon: <Zap className="w-4 h-4" />,
      commands: ['monitor-start', 'metrics-collect', 'alert-setup']
    },
    {
      id: 'security-audit',
      name: 'Security Audit',
      description: 'Run comprehensive security checks and audits',
      icon: <FileText className="w-4 h-4" />,
      commands: ['security-scan', 'audit-log', 'compliance-check']
    }
  ];

  const executeCommand = (command: string) => {
    const newOutput = [...terminalOutput];
    newOutput.push('$ ' + command);
    
    // Simulate command execution
    setTimeout(() => {
      switch (command.toLowerCase()) {
        case 'help':
          newOutput.push('Available commands:');
          newOutput.push('  help - Show this help message');
          newOutput.push('  status - Show node status');
          newOutput.push('  logs - Show recent logs');
          newOutput.push('  clear - Clear terminal');
          break;
        case 'status':
          newOutput.push('Node Status: Online');
          newOutput.push('CPU Usage: 45%');
          newOutput.push('Memory Usage: 62%');
          newOutput.push('TEE Status: Active (SGX)');
          break;
        case 'logs':
          newOutput.push('[2024-12-17 10:30:15] Validation task completed successfully');
          newOutput.push('[2024-12-17 10:29:42] TEE enclave initialized');
          newOutput.push('[2024-12-17 10:28:33] Node heartbeat sent');
          break;
        case 'clear':
          setTerminalOutput(['$ ']);
          return;
        default:
          newOutput.push('Command not found: ' + command);
          newOutput.push('Type "help" for available commands');
      }
      newOutput.push('$ ');
      setTerminalOutput(newOutput);
    }, 500);
    
    setTerminalOutput(newOutput);
    setCurrentCommand('');
  };

  const executeWorkflow = async (template: typeof workflowTemplates[0]) => {
    const newOutput = [...terminalOutput, '$ Executing workflow: ' + template.name];
    setTerminalOutput(newOutput);

    const workflow = {
      workflow_id: template.id,
      node_id: nodeId,
      steps: template.commands.map((cmd, i) => ({
        step_id: i + 1,
        name: cmd,
        command: cmd,
        dependency: i > 0 ? [i] : [],
      })),
      status: 'pending',
      logs: [],
    };

    try {
      const resp = await fetch(`${API_BASE_URL}/api/workflow/execute`, {
        method: 'POST',
        headers: getAuthHeaders(),
        body: JSON.stringify(workflow),
      });
      const exec = await resp.json();
      const out = [...newOutput];
      (exec.logs ?? []).forEach((log: string) => out.push('  ' + log));
      out.push(`$ Workflow "${template.name}" ${exec.status === 'completed' ? 'completed successfully' : 'failed'}`);
      out.push('$ ');
      setTerminalOutput(out);
    } catch {
      // Fallback to local simulation if backend unavailable
      template.commands.forEach((cmd, index) => {
        setTimeout(() => {
          setTerminalOutput(prev => {
            const updated = [...prev, '$ ' + cmd, '✓ ' + cmd + ' completed'];
            if (index === template.commands.length - 1) {
              updated.push('$ Workflow "' + template.name + '" completed', '$ ');
            }
            return updated;
          });
        }, (index + 1) * 600);
      });
    }
  };

  const runValidation = () => {
    if (!selectedProblem) return;
    
    setIsValidating(true);
    setShowValidationReport(false);
    
    // Simulate validation process
    setTimeout(() => {
      const problem = problems.find(p => p.id === selectedProblem);
      setValidationResults({
        problemId: selectedProblem,
        problemTitle: problem?.title,
        status: 'completed',
        timestamp: new Date().toLocaleTimeString(),
        testsPassed: Math.floor(Math.random() * 15) + 10,
        testsFailed: Math.floor(Math.random() * 3),
        coverage: Math.floor(Math.random() * 30) + 70,
        logs: [
          '✓ Test Suite 1: Initialization checks passed',
          '✓ Test Suite 2: TEE attestation verification passed',
          '✓ Test Suite 3: Memory bounds checking passed',
          Math.random() > 0.5 ? '✓ Test Suite 4: Network resilience passed' : '✗ Test Suite 4: Network timeout detected',
          '✓ Test Suite 5: Failure recovery passed',
          '✓ All critical paths validated successfully'
        ]
      });
      setIsValidating(false);
      setShowValidationReport(true);
    }, 2500);
  };

  const submitToConsensus = () => {
    if (!validationResults) return;
    alert(`Validation results submitted to consensus for problem: ${validationResults.problemTitle}`);
    setShowDVESolver(false);
    setSelectedProblem(null);
    setValidationResults(null);
    setShowValidationReport(false);
  };

  const problems = [
    { id: 'prob-001', title: 'Fabric Inference Failure', severity: 'critical', description: 'Fabric agent fails during inference execution' },
    { id: 'prob-002', title: 'TEE Attestation Timeout', severity: 'high', description: 'Trusted Execution Environment attestation takes too long' },
    { id: 'prob-003', title: 'Memory Leak in Agent', severity: 'medium', description: 'Memory usage increases without proper cleanup' },
    { id: 'prob-004', title: 'Network Disconnection', severity: 'high', description: 'Unexpected network connectivity loss' },
    { id: 'prob-005', title: 'Consensus Divergence', severity: 'critical', description: 'Nodes disagree on validation results' },
  ];

  // Global connections data
  const globalConnections = [
    { id: 'conn-001', nodeId: 'node-001', nodeName: 'Alpha-7', status: 'active', latency: 12, location: 'US-East', type: 'DVE' },
    { id: 'conn-002', nodeId: 'node-002', nodeName: 'Beta-3', status: 'active', latency: 18, location: 'US-West', type: 'DVE' },
    { id: 'conn-003', nodeId: 'node-003', nodeName: 'Gamma-9', status: 'active', latency: 25, location: 'EU-Central', type: 'DVE' },
    { id: 'conn-004', nodeId: 'node-004', nodeName: 'Delta-2', status: 'active', latency: 31, location: 'Asia-Pacific', type: 'DVE' },
    { id: 'conn-005', nodeId: 'node-005', nodeName: 'Epsilon-5', status: 'active', latency: 15, location: 'US-East', type: 'Router' },
    { id: 'conn-006', nodeId: 'node-006', nodeName: 'Zeta-8', status: 'active', latency: 22, location: 'EU-West', type: 'Router' },
    { id: 'conn-007', nodeId: 'node-007', nodeName: 'Eta-1', status: 'active', latency: 28, location: 'Asia-East', type: 'DVE' },
    { id: 'conn-008', nodeId: 'node-008', nodeName: 'Theta-4', status: 'active', latency: 19, location: 'US-Central', type: 'DVE' },
    { id: 'conn-009', nodeId: 'node-009', nodeName: 'Iota-6', status: 'active', latency: 35, location: 'South-America', type: 'DVE' },
    { id: 'conn-010', nodeId: 'node-010', nodeName: 'Kappa-0', status: 'active', latency: 14, location: 'Canada', type: 'Oracle' },
  ];

  const handleDHTToggle = async (checked: boolean) => {
    setIsDHTUpdating(true);
    try {
      const response = await fetch('/api/system/dht', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ enabled: checked }),
      });
      if (!response.ok) throw new Error('Failed to update DHT settings');
      setDHTEnabled(checked);
      toast({
        title: "DHT Settings Updated",
        description: `DHT has been ${checked ? 'enabled' : 'disabled'} successfully.`,
      });
    } catch (error) {
      toast({
        title: "Update Failed",
        description: "Failed to update DHT settings. Please try again.",
        variant: "destructive",
      });
    } finally {
      setIsDHTUpdating(false);
    }
  };

  const handleSavePolicy = async () => {
    setIsSavingPolicy(true);
    setPolicySaveStatus('idle');
    try {
      const policy = {
        name: `nap-policy-${nodeId}-${Date.now()}`,
        type: 'tee_config',
        rules: {
          tee_mode: policyTeeMode,
          max_memory_mb: policyMaxMemory,
          enable_attestation: policyAttestation,
        },
        priority: 1,
        enabled: true,
        target_dve: nodeId,
        created_at: new Date().toISOString(),
      };
      const response = await fetch(`${API_BASE_URL}/api/icme/policy/commit`, {
        method: 'POST',
        headers: getAuthHeaders(),
        body: JSON.stringify(policy),
      });
      setPolicySaveStatus(response.ok ? 'success' : 'error');
    } catch {
      setPolicySaveStatus('error');
    } finally {
      setIsSavingPolicy(false);
      setTimeout(() => setPolicySaveStatus('idle'), 3000);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex">
      {/* Backdrop */}
      <div 
        className="flex-1 bg-black/20 backdrop-blur-sm transition-all duration-300 z-40"
        onClick={onClose}
      />
      
      {/* Modal Panel - FOUNDATION FOR NESTING */}
      <div className="relative w-full max-w-4xl bg-background border-l shadow-2xl transform transition-all duration-300 ease-in-out">
        <div className="flex flex-col h-full">
          {/* Header */}
          <div className="flex items-center justify-between p-6 border-b">
            <div>
              <h2 className="text-2xl font-bold">Network Access Panel</h2>
              <p className="text-muted-foreground">
                {nodeName} ({nodeId})
              </p>
            </div>
            <div className="flex items-center space-x-2">
              <Badge variant="secondary">Connected</Badge>
              <Button variant="ghost" size="sm" onClick={onClose}>
                <X className="w-4 h-4" />
              </Button>
            </div>
          </div>

          {/* Content */}
          <div className="flex-1 overflow-auto">
            <Tabs defaultValue="terminal" className="h-full">
              <div className="px-6 pt-4">
                <TabsList className="grid w-full grid-cols-4">
                  <TabsTrigger value="terminal">Terminal</TabsTrigger>
                  <TabsTrigger value="workflows">Workflows</TabsTrigger>
                  <TabsTrigger value="tools">Tools</TabsTrigger>
                  <TabsTrigger value="admin">Admin</TabsTrigger>
                </TabsList>
              </div>

              <TabsContent value="terminal" className="px-6 pb-6 space-y-4">
                {/* Terminal */}
                <Card className="knirv-card-gradient">
                  <CardHeader>
                    <CardTitle className="flex items-center space-x-2">
                      <Terminal className="w-5 h-5" />
                      <span>Interactive Terminal</span>
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
                        <Button 
                          variant="outline" 
                          size="sm" 
                          className="w-full"
                          onClick={() => executeWorkflow(template)}>
                          <Play className="w-3 h-3 mr-1" />
                          Execute Workflow
                        </Button>
                      </CardContent>
                    </Card>
                  ))}
                </div>
              </TabsContent>

              <TabsContent value="tools" className="px-6 pb-6 space-y-4">
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
                  {/* Console Tool */}
                  <Card className="knirv-card-gradient">
                    <CardHeader className="pb-2">
                      <CardTitle className="flex items-center space-x-2 text-sm">
                        <Terminal className="w-4 h-4" />
                        <span>Console</span>
                      </CardTitle>
                      <CardDescription className="text-xs">
                        Real-time failure feed & logs
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      <Button 
                        variant={showConsole ? "default" : "outline"}
                        size="sm" 
                        className="w-full"
                        onClick={() => setShowConsole(!showConsole)}>
                        {showConsole ? 'Hide' : 'Show'}
                      </Button>
                    </CardContent>
                  </Card>

                  {/* Policy Tool */}
                  <Card className="knirv-card-gradient">
                    <CardHeader className="pb-2">
                      <CardTitle className="flex items-center space-x-2 text-sm">
                        <Shield className="w-4 h-4" />
                        <span>Policy</span>
                      </CardTitle>
                      <CardDescription className="text-xs">
                        Security configuration editor
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      <Button 
                        variant={showPolicy ? "default" : "outline"}
                        size="sm" 
                        className="w-full"
                        onClick={() => setShowPolicy(!showPolicy)}
                      >
                        {showPolicy ? 'Hide' : 'Show'}
                      </Button>
                    </CardContent>
                  </Card>

                  {/* Connections Tool - Opens Global Connections Panel */}
                  <Card className="knirv-card-gradient">
                    <CardHeader className="pb-2">
                      <CardTitle className="flex items-center space-x-2 text-sm">
                        <Radio className="w-4 h-4" />
                        <span>Connections</span>
                      </CardTitle>
                      <CardDescription className="text-xs">
                        Global active connections list
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      <Button 
                        variant={showConnections ? "default" : "outline"}
                        size="sm" 
                        className="w-full"
                        onClick={() => setShowConnections(!showConnections)}
                      >
                        {showConnections ? 'Hide' : 'Show'}
                      </Button>
                    </CardContent>
                  </Card>

                  {/* Monitor Tool */}
                  <Card className="knirv-card-gradient">
                    <CardHeader className="pb-2">
                      <CardTitle className="flex items-center space-x-2 text-sm">
                        <BarChart3 className="w-4 h-4" />
                        <span>Monitor</span>
                      </CardTitle>
                      <CardDescription className="text-xs">
                        Resolution tracking dashboard
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      <Button 
                        variant={showMonitor ? "default" : "outline"}
                        size="sm" 
                        className="w-full"
                        onClick={() => setShowMonitor(!showMonitor)}
                      >
                        {showMonitor ? 'Hide' : 'Show'}
                      </Button>
                    </CardContent>
                  </Card>
                </div>
              </TabsContent>

              <TabsContent value="admin" className="px-6 pb-6 space-y-4">
                {/* Demo Mode Toggle */}
                <Card className="w-full">
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                      {isDemoMode ? (
                        <TestTube className="h-5 w-5 text-blue-500" />
                      ) : (
                        <Database className="h-5 w-5 text-green-500" />
                      )}
                      Demo Mode Configuration
                    </CardTitle>
                    <CardDescription>
                      Control whether the application shows demo data or connects to real backend services
                    </CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-6">
                    <div className="flex items-center justify-between">
                      <div className="space-y-0.5">
                        <Label htmlFor="demo-mode" className="text-base font-medium">
                          Demo Mode
                        </Label>
                        <div className="text-sm text-muted-foreground">
                          {isDemoMode 
                            ? "Showing mock data and simulated responses" 
                            : "Connected to live backend services and database"
                          }
                        </div>
                      </div>
                      <Switch
                        id="demo-mode"
                        checked={isDemoMode}
                        onCheckedChange={toggleDemoMode}
                      />
                    </div>

                    <div className="rounded-lg border p-4 space-y-3">
                      <div className="flex items-center gap-2 text-sm font-medium">
                        <AlertTriangle className="h-4 w-4 text-amber-500" />
                        Current Mode: {isDemoMode ? "Demo" : "Production"}
                      </div>
                      
                      {isDemoMode ? (
                        <div className="text-sm text-muted-foreground space-y-2">
                          <p><strong>Demo Mode Active:</strong></p>
                          <ul className="list-disc list-inside space-y-1 ml-4">
                            <li>Mock data for DVE nodes, validation tasks, and fabric items</li>
                            <li>Simulated real-time updates and metrics</li>
                            <li>No actual backend connections required</li>
                            <li>Safe for testing and demonstrations</li>
                          </ul>
                        </div>
                      ) : (
                        <div className="text-sm text-muted-foreground space-y-2">
                          <p><strong>Production Mode Active:</strong></p>
                          <ul className="list-disc list-inside space-y-1 ml-4">
                            <li>Live data from backend database</li>
                            <li>Real WebSocket connections for updates</li>
                            <li>Actual DVE node management and validation</li>
                            <li>Empty states shown when no data available</li>
                          </ul>
                        </div>
                      )}
                    </div>
                  </CardContent>
                </Card>

                {/* DHT Toggle */}
                <Card className="w-full">
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                      {isDHTEnabled ? (
                        <Network className="h-5 w-5 text-green-500" />
                      ) : (
                        <WifiOff className="h-5 w-5 text-red-500" />
                      )}
                      DHT Configuration
                    </CardTitle>
                    <CardDescription>
                      Control whether the Distributed Hash Table (DHT) is enabled for peer discovery and resource sharing
                    </CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-6">
                    <div className="flex items-center justify-between">
                      <div className="space-y-0.5">
                        <Label htmlFor="dht-toggle" className="text-base font-medium">
                          Enable DHT
                        </Label>
                        <div className="text-sm text-muted-foreground">
                          {isDHTEnabled
                            ? "DHT is active for peer discovery and network coordination"
                            : "DHT is disabled to reduce network noise and resource usage"
                          }
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <div className={`relative ${!isDHTEnabled ? "ring-2 ring-red-500 rounded-full" : ""}`}>
                          <Switch
                            id="dht-toggle"
                            checked={isDHTEnabled}
                            onCheckedChange={handleDHTToggle}
                            disabled={isDHTUpdating}
                          />
                          {!isDHTEnabled && (
                            <div className="absolute -top-1 -right-1 w-3 h-3 bg-red-500 rounded-full animate-pulse"></div>
                          )}
                        </div>
                        {isDHTUpdating && (
                          <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
                        )}
                      </div>
                    </div>

                    <div className="rounded-lg border p-4 space-y-3">
                      <div className="flex items-center gap-2 text-sm font-medium">
                        <AlertTriangle className="h-4 w-4 text-amber-500" />
                        Current Status: {isDHTEnabled ? "Enabled" : "Disabled"}
                      </div>

                      {isDHTEnabled ? (
                        <div className="text-sm text-muted-foreground space-y-2">
                          <p><strong>DHT Active:</strong></p>
                          <ul className="list-disc list-inside space-y-1 ml-4">
                            <li>Peer discovery and network coordination enabled</li>
                            <li>Resource announcements and lookups active</li>
                            <li>Increased network traffic and resource usage</li>
                            <li>Full P2P functionality available</li>
                          </ul>
                        </div>
                      ) : (
                        <div className="text-sm text-muted-foreground space-y-2">
                          <p><strong>DHT Disabled:</strong></p>
                          <ul className="list-disc list-inside space-y-1 ml-4">
                            <li>Reduced network noise and background activity</li>
                            <li>Limited peer discovery capabilities</li>
                            <li>Lower resource consumption</li>
                            <li>Local operations only (no network-wide coordination)</li>
                          </ul>
                        </div>
                      )}
                    </div>
                  </CardContent>
                </Card>

                {/* System Information */}
                <Card className="knirv-card-gradient">
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                      <Server className="h-5 w-5" />
                      System Information
                    </CardTitle>
                    <CardDescription>
                      Current system status and configuration
                    </CardDescription>
                  </CardHeader>
                  <CardContent>
                    <div className="grid grid-cols-2 gap-4 text-sm">
                      <div>
                        <span className="font-medium">WebSocket Status:</span>
                        <Badge variant={wsConnected ? "default" : "destructive"} className="ml-2">
                          {wsConnected ? "Connected" : "Disconnected"}
                        </Badge>
                      </div>
                      <div>
                        <span className="font-medium">Backend URL:</span>
                        <span className="ml-2 text-muted-foreground">http://localhost:8082</span>
                      </div>
                      <div>
                        <span className="font-medium">DVE Nodes:</span>
                        <span className="ml-2">{dveNodeCount} registered</span>
                      </div>
                      <div>
                        <span className="font-medium">Active Tasks:</span>
                        <span className="ml-2">{activeTasks} tasks</span>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              </TabsContent>
            </Tabs>

            {/* NESTED PANELS - Slide out from NAP modal */}
            

            {/* Console Panel - Slides out from left edge of NAP modal */}
            {showConsole && (
              <div className="absolute z-50 pointer-events-auto transform transition-all duration-300 ease-out translate-x-0" style={{top: '100px', right: '896px'}}>
                <div className="bg-slate-900 rounded-lg border border-blue-600/30 shadow-2xl w-80">
                  <div className="flex items-center justify-between p-4 border-b border-blue-600/30">
                    <div className="flex items-center space-x-2">
                      <Terminal className="w-4 h-4 text-cyan-400" />
                      <h3 className="font-semibold text-sm">Real-Time Console</h3>
                    </div>
                    <button
                      onClick={() => setShowConsole(false)}
                      className="p-1 hover:bg-slate-800 rounded transition-colors">
                      <X className="w-4 h-4" />
                    </button>
                  </div>
                  <div className="p-4 bg-black rounded-b-lg max-h-80 overflow-y-auto font-mono text-xs text-blue-400">
                    <div className="space-y-1">
                      <div>[10:30:15] Validation task completed successfully</div>
                      <div>[10:29:42] TEE enclave initialized</div>
                      <div>[10:28:33] Node heartbeat sent</div>
                      <div>[10:27:11] Failure detected in fabric inference</div>
                      <div className="text-yellow-400">[10:26:45] WARNING: High memory usage detected</div>
                      <div>[10:25:22] Recovery procedure initiated</div>
                      <div className="text-cyan-400">$ Ready for input...</div>
                    </div>
                  </div>
                </div>
              </div>
            )}

            {/* Policy Panel - Slides out from left edge of NAP modal, below Console */}
            {showPolicy && (
              <div className="absolute z-50 pointer-events-auto transform transition-all duration-300 ease-out translate-x-0" style={{top: '420px', right: '896px'}}>
                <div className="bg-slate-900 rounded-lg border border-blue-600/30 shadow-2xl w-80">
                  <div className="flex items-center justify-between p-4 border-b border-blue-600/30">
                    <div className="flex items-center space-x-2">
                      <Shield className="w-4 h-4 text-cyan-400" />
                      <h3 className="font-semibold text-sm">Security Policy</h3>
                    </div>
                    <button
                      onClick={() => setShowPolicy(false)}
                      className="p-1 hover:bg-slate-800 rounded transition-colors">
                      <X className="w-4 h-4" />
                    </button>
                  </div>
                  <div className="p-4 space-y-3 max-h-80 overflow-y-auto">
                    <div>
                      <label className="text-xs text-slate-400 block mb-1">TEE Mode</label>
                      <select
                        value={policyTeeMode}
                        onChange={(e) => setPolicyTeeMode(e.target.value)}
                        className="w-full bg-slate-800 border border-blue-600/30 rounded px-2 py-1 text-sm text-slate-200"
                      >
                        <option>SGX (Intel)</option>
                        <option>TDX (Intel)</option>
                        <option>SEV (AMD)</option>
                      </select>
                    </div>
                    <div>
                      <label className="text-xs text-slate-400 block mb-1">Max Memory (MB)</label>
                      <input
                        type="number"
                        value={policyMaxMemory}
                        onChange={(e) => setPolicyMaxMemory(Number(e.target.value))}
                        className="w-full bg-slate-800 border border-blue-600/30 rounded px-2 py-1 text-sm text-slate-200"
                      />
                    </div>
                    <div>
                      <label className="text-xs text-slate-400 flex items-center space-x-2 cursor-pointer">
                        <input
                          type="checkbox"
                          checked={policyAttestation}
                          onChange={(e) => setPolicyAttestation(e.target.checked)}
                          className="w-4 h-4"
                        />
                        <span>Enable Attestation</span>
                      </label>
                    </div>
                    {policySaveStatus === 'success' && <p className="text-xs text-green-400">Policy saved successfully</p>}
                    {policySaveStatus === 'error' && <p className="text-xs text-red-400">Failed to save policy</p>}
                    <button
                      onClick={handleSavePolicy}
                      disabled={isSavingPolicy}
                      className="w-full bg-cyan-500 hover:bg-cyan-600 disabled:opacity-50 text-slate-900 font-semibold py-2 rounded text-sm transition-colors"
                    >
                      {isSavingPolicy ? 'Saving...' : 'Save Policy'}
                    </button>
                  </div>
                </div>
              </div>
            )}

            {/* Monitor Panel - Positioned below tools section */}
            {showMonitor && (
              <div className="absolute left-6 right-6 z-30 pointer-events-auto transform transition-transform duration-300 ease-out translate-y-0" style={{top: '540px'}}>
                <div className="bg-slate-900 rounded-lg border border-blue-600/30 shadow-2xl">
                  <div className="flex items-center justify-between p-4 border-b border-blue-600/30">
                    <div className="flex items-center space-x-2">
                      <BarChart3 className="w-4 h-4 text-cyan-400" />
                      <h3 className="font-semibold text-sm">Resolution Monitor</h3>
                    </div>
                    <button 
                      onClick={() => setShowMonitor(false)}
                      className="p-1 hover:bg-slate-800 rounded transition-colors">
                      <X className="w-4 h-4" />
                    </button>
                  </div>
                  <div className="p-4 space-y-3 max-h-40 overflow-y-auto">
                    <div>
                      <div className="flex justify-between text-xs mb-1">
                        <span>CPU Usage</span>
                        <span className="text-cyan-400 font-semibold">45%</span>
                      </div>
                      <div className="w-full bg-slate-700 rounded-full h-2">
                        <div className="bg-cyan-500 h-2 rounded-full" style={{width: '45%'}}></div>
                      </div>
                    </div>
                    <div>
                      <div className="flex justify-between text-xs mb-1">
                        <span>Memory Usage</span>
                        <span className="text-cyan-400 font-semibold">62%</span>
                      </div>
                      <div className="w-full bg-slate-700 rounded-full h-2">
                        <div className="bg-cyan-500 h-2 rounded-full" style={{width: '62%'}}></div>
                      </div>
                    </div>
                    <div>
                      <div className="flex justify-between text-xs mb-1">
                        <span>Validation Tasks</span>
                        <span className="text-cyan-400 font-semibold">8/10</span>
                      </div>
                      <div className="w-full bg-slate-700 rounded-full h-2">
                        <div className="bg-blue-500 h-2 rounded-full" style={{width: '80%'}}></div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            )}

            {/* Global Connections Panel - Slides out from left side when clicking Connections */}
            {showConnections && (
              <div className="fixed left-0 top-0 bottom-0 z-[60] pointer-events-auto transform transition-all duration-300 ease-out translate-x-0 w-96">
                <div className="h-full bg-slate-900 rounded-r-lg border-r border-blue-600/30 shadow-2xl">
                  <div className="flex items-center justify-between p-4 border-b border-blue-600/30">
                    <div className="flex items-center space-x-2">
                      <Radio className="w-4 h-4 text-cyan-400" />
                      <h3 className="font-semibold text-sm">Global Active Connections</h3>
                    </div>
                    <button
                      onClick={() => setShowConnections(false)}
                      className="p-1 hover:bg-slate-800 rounded transition-colors">
                      <X className="w-4 h-4" />
                    </button>
                  </div>
                  <div className="p-4 space-y-4 overflow-y-auto" style={{maxHeight: 'calc(100% - 60px)'}}>
                    <div className="flex items-center justify-between text-xs text-slate-400">
                      <span>Total: {globalConnections.length} connections</span>
                      <Badge variant="secondary" className="bg-green-500/20 text-green-400">All Active</Badge>
                    </div>
                    <div className="space-y-2">
                      {globalConnections.map((conn) => (
                        <div
                          key={conn.id}
                          className="w-full text-left p-3 border rounded bg-slate-800/50 border-blue-600/20 hover:border-blue-600/50 transition-all cursor-pointer"
                        >
                          <div className="flex items-start justify-between">
                            <div className="flex-1">
                              <div className="text-xs font-medium text-slate-200">{conn.nodeName}</div>
                              <div className="text-xs text-slate-400 mt-0.5">{conn.nodeId}</div>
                              <div className="flex items-center gap-2 mt-1">
                                <Badge variant="outline" className="text-[10px] h-4">
                                  {conn.type}
                                </Badge>
                                <span className="text-[10px] text-slate-500">{conn.location}</span>
                              </div>
                            </div>
                            <div className="text-right">
                              <div className="flex items-center gap-1">
                                <div className="w-2 h-2 rounded-full bg-green-500"></div>
                                <span className="text-xs text-green-400">Active</span>
                              </div>
                              <div className="text-[10px] text-slate-400 mt-1">{conn.latency}ms</div>
                            </div>
                          </div>
                        </div>
                      ))}
                    </div>
                    <div className="pt-2 border-t border-blue-600/20">
                      <div className="text-xs text-slate-500 text-center">
                        Last updated: {new Date().toLocaleTimeString()}
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
