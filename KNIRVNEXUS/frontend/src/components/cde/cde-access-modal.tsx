'use client';

import React, { useState } from 'react';
import { X, Terminal, Play, Code, Database, Settings, Cpu, Zap, FileText, Download, Share2, Radio, Shield, BarChart3, Upload } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';

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
    '$ Welcome to KNIRV CDE Terminal',
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

  const executeWorkflow = (template: typeof workflowTemplates[0]) => {
    const newOutput = [...terminalOutput];
    newOutput.push('$ Executing workflow: ' + template.name);
    newOutput.push('$ Running commands: ' + template.commands.join(', '));
    
    template.commands.forEach((cmd, index) => {
      setTimeout(() => {
        const updatedOutput = [...newOutput];
        updatedOutput.push('$ ' + cmd);
        updatedOutput.push('✓ ' + cmd + ' completed successfully');
        if (index === template.commands.length - 1) {
          updatedOutput.push('$ Workflow "' + template.name + '" completed successfully');
          updatedOutput.push('$ ');
        }
        setTerminalOutput(updatedOutput);
      }, (index + 1) * 1000);
    });
    
    setTerminalOutput(newOutput);
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
              <Button 
                variant="outline" 
                size="sm" 
                className="text-[10px] font-black uppercase border-blue-500/30 text-blue-400 hover:bg-blue-500 hover:text-white"
                onClick={onToggleMode}
              >
                Switch to Modular
              </Button>
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
                <TabsList className="grid w-full grid-cols-3">
                  <TabsTrigger value="terminal">Terminal</TabsTrigger>
                  <TabsTrigger value="workflows">Workflows</TabsTrigger>
                  <TabsTrigger value="tools">Tools</TabsTrigger>
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

                  {/* Connections Tool */}
                  <Card className="knirv-card-gradient">
                    <CardHeader className="pb-2">
                      <CardTitle className="flex items-center space-x-2 text-sm">
                        <Radio className="w-4 h-4" />
                        <span>Connections</span>
                      </CardTitle>
                      <CardDescription className="text-xs">
                        Connected NRV nodes list
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

                  {/* DVE Solver Tool */}
                  <Card className="knirv-card-gradient">
                    <CardHeader className="pb-2">
                      <CardTitle className="flex items-center space-x-2 text-sm">
                        <Zap className="w-4 h-4" />
                        <span>DVE Solver</span>
                      </CardTitle>
                      <CardDescription className="text-xs">
                        Distributed validation engine
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      <Button 
                        variant={showDVESolver ? "default" : "outline"}
                        size="sm" 
                        className="w-full"
                        onClick={() => setShowDVESolver(!showDVESolver)}
                      >
                        <Zap className="w-3 h-3 mr-1" />
                        {showDVESolver ? 'Hide' : 'Open'}
                      </Button>
                    </CardContent>
                  </Card>

                  {/* KNIRVENGINE Tool */}
                  <Card className="knirv-card-gradient border-primary/50">
                    <CardHeader className="pb-2">
                      <CardTitle className="flex items-center space-x-2 text-sm">
                        <Cpu className="w-4 h-4" />
                        <span>KNIRVENGINE</span>
                      </CardTitle>
                      <CardDescription className="text-xs">
                        AI Engine interface
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      <Button 
                        variant="default" 
                        size="sm" 
                        className="w-full"
                        onClick={onOpenKNIRVEngine}
                      >
                        <Cpu className="w-3 h-3 mr-1" />
                        Open
                      </Button>
                    </CardContent>
                  </Card>

                  {/* Code Editor Tool */}
                  <Card className="knirv-card-gradient">
                    <CardHeader className="pb-2">
                      <CardTitle className="flex items-center space-x-2 text-sm">
                        <Code className="w-4 h-4" />
                        <span>Code Editor</span>
                      </CardTitle>
                      <CardDescription className="text-xs">
                        Web-based development IDE
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      <Button variant="outline" size="sm" className="w-full">
                        <Code className="w-3 h-3 mr-1" />
                        Open
                      </Button>
                    </CardContent>
                  </Card>
                </div>
              </TabsContent>
            </Tabs>

            {/* NESTED PANELS - Slide out from CDE modal */}
            

            {/* Console Panel - Slides out from left edge of CDE modal */}
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

            {/* Policy Panel - Slides out from left edge of CDE modal, below Console */}
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
                      <select className="w-full bg-slate-800 border border-blue-600/30 rounded px-2 py-1 text-sm text-slate-200">
                        <option>SGX (Intel)</option>
                        <option>TDX (Intel)</option>
                        <option>SEV (AMD)</option>
                      </select>
                    </div>
                    <div>
                      <label className="text-xs text-slate-400 block mb-1">Max Memory (MB)</label>
                      <input type="number" defaultValue="4096" className="w-full bg-slate-800 border border-blue-600/30 rounded px-2 py-1 text-sm text-slate-200" />
                    </div>
                    <div>
                      <label className="text-xs text-slate-400 flex items-center space-x-2 cursor-pointer">
                        <input type="checkbox" defaultChecked className="w-4 h-4" />
                        <span>Enable Attestation</span>
                      </label>
                    </div>
                    <button className="w-full bg-cyan-500 hover:bg-cyan-600 text-slate-900 font-semibold py-2 rounded text-sm transition-colors">
                      Save Policy
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

            {/* DVE Solver Modal - Replaces Connected NRVs panel on left side */}
            {showDVESolver && (
              <div className="fixed left-0 top-0 bottom-0 z-[60] pointer-events-auto transform transition-all duration-300 ease-out translate-x-0 w-80">
                <div className="h-full bg-slate-900 rounded-r-lg border-r border-blue-600/30 shadow-2xl">
                  <div className="flex items-center justify-between p-4 border-b border-blue-600/30">
                    <div className="flex items-center space-x-2">
                      <Zap className="w-4 h-4 text-cyan-400" />
                      <h3 className="font-semibold text-sm">DVE Solver</h3>
                    </div>
                    <button
                      onClick={() => setShowDVESolver(false)}
                      className="p-1 hover:bg-slate-800 rounded transition-colors">
                      <X className="w-4 h-4" />
                    </button>
                  </div>
                  <div className="p-4 space-y-4 overflow-y-auto" style={{maxHeight: 'calc(100% - 60px)'}}>
                    {!showValidationReport ? (
                      <>
                        <div>
                          <h4 className="text-xs font-semibold text-cyan-400 mb-3 flex items-center space-x-2">
                            <FileText className="w-3 h-3" />
                            <span>Select Problem:</span>
                          </h4>
                          <div className="space-y-2">
                            {problems.map((prob) => (
                              <button
                                key={prob.id}
                                onClick={() => setSelectedProblem(prob.id)}
                                className={`w-full text-left p-3 border rounded transition-all ${selectedProblem === prob.id ? 'bg-blue-600/20 border-blue-600/50' : 'bg-slate-800/50 border-blue-600/20 hover:border-blue-600/50'}`}>
                                <div className="flex items-start justify-between">
                                  <div className="flex-1">
                                    <div className="text-xs font-medium">{prob.title}</div>
                                    <div className="text-xs text-slate-400 mt-1">{prob.description}</div>
                                  </div>
                                  <span className={`text-xs px-1.5 py-0.5 rounded font-semibold whitespace-nowrap ml-2 ${prob.severity === 'critical' ? 'bg-red-500/20 text-red-400' : prob.severity === 'high' ? 'bg-yellow-500/20 text-yellow-400' : 'bg-blue-500/20 text-blue-400'}`}>
                                    {prob.severity.toUpperCase()}
                                  </span>
                                </div>
                              </button>
                            ))}
                          </div>
                        </div>
                        <button
                          onClick={runValidation}
                          disabled={!selectedProblem || isValidating}
                          className="w-full bg-cyan-500 hover:bg-cyan-600 disabled:bg-cyan-500/50 disabled:cursor-not-allowed text-slate-900 font-semibold py-2 rounded text-xs transition-colors flex items-center justify-center space-x-1">
                          {isValidating ? (
                            <>
                              <div className="w-3 h-3 border border-slate-900/30 border-t-slate-900 rounded-full animate-spin"></div>
                              <span>Running...</span>
                            </>
                          ) : (
                            <>
                              <Play className="w-3 h-3" />
                              <span>Run Validation</span>
                            </>
                          )}
                        </button>
                      </>
                    ) : (
                      <>
                        <div className="bg-blue-500/10 border border-blue-500/30 rounded p-3">
                          <div className="flex items-center space-x-2 mb-1">
                            <div className="w-2 h-2 rounded-full bg-blue-500"></div>
                            <span className="text-xs font-semibold text-blue-400">Complete</span>
                          </div>
                          <div className="text-xs text-slate-400 space-y-0.5">
                            <div>Problem: {validationResults?.problemTitle}</div>
                            <div>Tests: {validationResults?.testsPassed}/{validationResults?.testsPassed + validationResults?.testsFailed}</div>
                          </div>
                        </div>

                        <div>
                          <h5 className="text-xs font-semibold text-cyan-400 mb-1 flex items-center space-x-1">
                            <BarChart3 className="w-2.5 h-2.5" />
                            <span>Logs</span>
                          </h5>
                          <div className="bg-black rounded p-2 font-mono text-xs text-blue-400 space-y-0.5 max-h-32 overflow-y-auto">
                            {validationResults?.logs.map((log: string, idx: number) => (
                              <div key={idx}>{log}</div>
                            ))}
                          </div>
                        </div>

                        <div className="space-y-1">
                          <button
                            onClick={() => setShowValidationReport(false)}
                            className="w-full bg-slate-700 hover:bg-slate-600 text-slate-100 font-semibold py-1.5 rounded text-xs transition-colors">
                            Run Another
                          </button>
                          <button
                            onClick={submitToConsensus}
                            className="w-full bg-cyan-500 hover:bg-cyan-600 text-slate-900 font-semibold py-1.5 rounded text-xs transition-colors flex items-center justify-center space-x-1">
                            <Upload className="w-3 h-3" />
                            <span>Submit</span>
                          </button>
                        </div>
                      </>
                    )}
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
