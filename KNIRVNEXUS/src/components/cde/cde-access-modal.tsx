'use client';

import React, { useState } from 'react';
import { X, Terminal, Play, Code, Database, Settings, Cpu, Zap, FileText, Download, Share2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';

interface CDEAccessModalProps {
  isOpen: boolean;
  onClose: () => void;
  nodeId: string;
  nodeName: string;
  onOpenKNIRVEngine: () => void;
}

export function CDEAccessModal({ isOpen, onClose, nodeId, nodeName, onOpenKNIRVEngine }: CDEAccessModalProps) {
  const [terminalOutput, setTerminalOutput] = useState([
    '$ Welcome to KNIRV CDE Terminal',
    '$ Node: ' + nodeName + ' (' + nodeId + ')',
    '$ Type "help" for available commands',
    '$ '
  ]);
  const [currentCommand, setCurrentCommand] = useState('');

  const workflowTemplates = [
    {
      id: 'validation-setup',
      name: 'Validation Setup',
      description: 'Initialize validation environment with TEE configuration',
      icon: <Settings className="w-4 h-4" />,
      commands: ['tee-init', 'validation-config', 'security-check']
    },
    {
      id: 'model-deployment',
      name: 'Model Deployment',
      description: 'Deploy AI models to the cognitive engine',
      icon: <Cpu className="w-4 h-4" />,
      commands: ['model-load', 'inference-setup', 'performance-test']
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

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex">
      {/* Backdrop */}
      <div 
        className="flex-1 bg-black/20 backdrop-blur-sm transition-all duration-300"
        onClick={onClose}
      />
      
      {/* Modal Panel */}
      <div className="w-full max-w-4xl bg-background border-l shadow-2xl transform transition-all duration-300 ease-in-out">
        <div className="flex flex-col h-full">
          {/* Header */}
          <div className="flex items-center justify-between p-6 border-b">
            <div>
              <h2 className="text-2xl font-bold">Cloud Development Environment</h2>
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
                      <div className="text-green-400 space-y-1 max-h-96 overflow-y-auto">
                        {terminalOutput.map((line, index) => (
                          <div key={index}>{line}</div>
                        ))}
                      </div>
                      <div className="flex items-center mt-2">
                        <span className="text-green-400">$ </span>
                        <input
                          type="text"
                          value={currentCommand}
                          onChange={(e) => setCurrentCommand(e.target.value)}
                          onKeyPress={(e) => {
                            if (e.key === 'Enter' && currentCommand.trim()) {
                              executeCommand(currentCommand.trim());
                            }
                          }}
                          className="flex-1 bg-transparent text-green-400 outline-none ml-2"
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
                          onClick={() => executeWorkflow(template)}
                        >
                          <Play className="w-3 h-3 mr-1" />
                          Execute Workflow
                        </Button>
                      </CardContent>
                    </Card>
                  ))}
                </div>
              </TabsContent>

              <TabsContent value="tools" className="px-6 pb-6 space-y-4">
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                  {/* KNIRVENGINE Tool */}
                  <Card className="knirv-card-gradient border-primary/50">
                    <CardHeader className="pb-2">
                      <CardTitle className="flex items-center space-x-2 text-sm">
                        <Cpu className="w-4 h-4" />
                        <span>KNIRVENGINE</span>
                      </CardTitle>
                      <CardDescription className="text-xs">
                        Access the KNIRV AI Engine interface
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      <Button 
                        variant="default" 
                        size="sm" 
                        className="w-full"
                        onClick={onOpenKNIRVEngine}
                      >
                        <Zap className="w-3 h-3 mr-1" />
                        Open KNIRVENGINE
                      </Button>
                    </CardContent>
                  </Card>

                  {/* Other Tools */}
                  <Card className="knirv-card-gradient">
                    <CardHeader className="pb-2">
                      <CardTitle className="flex items-center space-x-2 text-sm">
                        <Code className="w-4 h-4" />
                        <span>Code Editor</span>
                      </CardTitle>
                      <CardDescription className="text-xs">
                        Web-based code editor with syntax highlighting
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      <Button variant="outline" size="sm" className="w-full">
                        <Code className="w-3 h-3 mr-1" />
                        Open Editor
                      </Button>
                    </CardContent>
                  </Card>

                  <Card className="knirv-card-gradient">
                    <CardHeader className="pb-2">
                      <CardTitle className="flex items-center space-x-2 text-sm">
                        <Database className="w-4 h-4" />
                        <span>Database Console</span>
                      </CardTitle>
                      <CardDescription className="text-xs">
                        Query and manage node databases
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      <Button variant="outline" size="sm" className="w-full">
                        <Database className="w-3 h-3 mr-1" />
                        Open Console
                      </Button>
                    </CardContent>
                  </Card>

                  <Card className="knirv-card-gradient">
                    <CardHeader className="pb-2">
                      <CardTitle className="flex items-center space-x-2 text-sm">
                        <FileText className="w-4 h-4" />
                        <span>Log Viewer</span>
                      </CardTitle>
                      <CardDescription className="text-xs">
                        Real-time log monitoring and analysis
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      <Button variant="outline" size="sm" className="w-full">
                        <FileText className="w-3 h-3 mr-1" />
                        View Logs
                      </Button>
                    </CardContent>
                  </Card>

                  <Card className="knirv-card-gradient">
                    <CardHeader className="pb-2">
                      <CardTitle className="flex items-center space-x-2 text-sm">
                        <Download className="w-4 h-4" />
                        <span>Reports</span>
                      </CardTitle>
                      <CardDescription className="text-xs">
                        Generate and download node reports
                      </CardDescription>
                    </CardHeader>
                    <CardContent className="space-y-1">
                      <Button variant="outline" size="sm" className="w-full">
                        <Download className="w-3 h-3 mr-1" />
                        Download
                      </Button>
                      <Button variant="outline" size="sm" className="w-full">
                        <Share2 className="w-3 h-3 mr-1" />
                        Share
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
