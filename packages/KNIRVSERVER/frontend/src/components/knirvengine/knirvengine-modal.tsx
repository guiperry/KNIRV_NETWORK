'use client';

import React, { useState } from 'react';
import { X, Cpu, Zap, Brain, Settings, Play, Pause, RotateCcw, Download, Share2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Progress } from '@/components/ui/progress';

interface KNIRVEngineModalProps {
  isOpen: boolean;
  onClose: () => void;
  nodeId: string;
}

export function KNIRVEngineModal({ isOpen, onClose, nodeId }: KNIRVEngineModalProps) {
  const [engineStatus, setEngineStatus] = useState<'running' | 'stopped' | 'loading'>('running');
  const [currentTask, setCurrentTask] = useState('Processing validation request #1247');

  const engineMetrics = {
    cpuUsage: 78,
    memoryUsage: 65,
    gpuUsage: 92,
    throughput: '1.2k req/s',
    accuracy: '94.7%',
    latency: '23ms'
  };

  const activeModels = [
    { name: 'CodeT5-Large', status: 'active', accuracy: '96.2%', load: '45%' },
    { name: 'Deepseek-Coder', status: 'active', accuracy: '94.8%', load: '32%' },
    { name: 'Gemini-Pro', status: 'standby', accuracy: '97.1%', load: '0%' },
    { name: 'Custom-Validator', status: 'active', accuracy: '93.5%', load: '23%' }
  ];

  const recentTasks = [
    { id: '#1247', type: 'Code Validation', status: 'processing', progress: 67 },
    { id: '#1246', type: 'Model Training', status: 'completed', progress: 100 },
    { id: '#1245', type: 'Data Analysis', status: 'completed', progress: 100 },
    { id: '#1244', type: 'Security Audit', status: 'failed', progress: 45 },
    { id: '#1243', type: 'Performance Test', status: 'completed', progress: 100 }
  ];

  const handleEngineControl = (action: 'start' | 'stop' | 'restart') => {
    setEngineStatus('loading');
    setTimeout(() => {
      switch (action) {
        case 'start':
          setEngineStatus('running');
          break;
        case 'stop':
          setEngineStatus('stopped');
          break;
        case 'restart':
          setEngineStatus('running');
          break;
      }
    }, 2000);
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center">
      {/* Backdrop */}
      <div 
        className="absolute inset-0 bg-black/50 backdrop-blur-sm"
        onClick={onClose}
      />
      
      {/* Modal */}
      <div className="relative w-full max-w-6xl max-h-[90vh] bg-background rounded-lg shadow-2xl border overflow-hidden">
        <div className="flex flex-col h-full max-h-[90vh]">
          {/* Header */}
          <div className="flex items-center justify-between p-6 border-b bg-gradient-to-r from-primary/10 to-secondary/10">
            <div className="flex items-center space-x-4">
              <div className="w-12 h-12 bg-gradient-to-r from-primary to-secondary rounded-lg flex items-center justify-center">
                <Brain className="w-6 h-6 text-white" />
              </div>
              <div>
                <h2 className="text-2xl font-bold">KNIRVENGINE</h2>
                <p className="text-muted-foreground">
                  AI Engine Interface - Node {nodeId}
                </p>
              </div>
            </div>
            <div className="flex items-center space-x-2">
              <Badge variant={engineStatus === 'running' ? 'default' : engineStatus === 'stopped' ? 'destructive' : 'secondary'}>
                {engineStatus === 'running' ? 'Running' : engineStatus === 'stopped' ? 'Stopped' : 'Loading...'}
              </Badge>
              <Button variant="ghost" size="sm" onClick={onClose}>
                <X className="w-4 h-4" />
              </Button>
            </div>
          </div>

          {/* Content */}
          <div className="flex-1 overflow-auto p-6">
            <Tabs defaultValue="dashboard" className="space-y-6">
              <TabsList className="grid w-full grid-cols-4">
                <TabsTrigger value="dashboard">Dashboard</TabsTrigger>
                <TabsTrigger value="fabric">Fabric</TabsTrigger>
                <TabsTrigger value="tasks">Tasks</TabsTrigger>
                <TabsTrigger value="settings">Settings</TabsTrigger>
              </TabsList>

              <TabsContent value="dashboard" className="space-y-6">
                {/* Engine Controls */}
                <Card className="knirv-card-gradient">
                  <CardHeader>
                    <CardTitle className="flex items-center space-x-2">
                      <Cpu className="w-5 h-5" />
                      <span>Engine Controls</span>
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="flex items-center space-x-4">
                      <Button 
                        variant={engineStatus === 'running' ? 'secondary' : 'default'}
                        onClick={() => handleEngineControl('start')}
                        disabled={engineStatus === 'loading'}
                      >
                        <Play className="w-4 h-4 mr-2" />
                        Start
                      </Button>
                      <Button 
                        variant={engineStatus === 'stopped' ? 'secondary' : 'destructive'}
                        onClick={() => handleEngineControl('stop')}
                        disabled={engineStatus === 'loading'}
                      >
                        <Pause className="w-4 h-4 mr-2" />
                        Stop
                      </Button>
                      <Button 
                        variant="outline"
                        onClick={() => handleEngineControl('restart')}
                        disabled={engineStatus === 'loading'}
                      >
                        <RotateCcw className="w-4 h-4 mr-2" />
                        Restart
                      </Button>
                      <div className="flex-1" />
                      <Button variant="outline" size="sm">
                        <Download className="w-4 h-4 mr-2" />
                        Export Logs
                      </Button>
                      <Button variant="outline" size="sm">
                        <Share2 className="w-4 h-4 mr-2" />
                        Share Status
                      </Button>
                    </div>
                    {engineStatus === 'running' && (
                      <div className="mt-4 p-3 bg-green-50 dark:bg-green-950 rounded-lg">
                        <p className="text-sm text-green-700 dark:text-green-300">
                          <Zap className="w-4 h-4 inline mr-1" />
                          Current Task: {currentTask}
                        </p>
                      </div>
                    )}
                  </CardContent>
                </Card>

                {/* Performance Metrics */}
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                  <Card className="knirv-card-gradient">
                    <CardHeader className="pb-2">
                      <CardTitle className="text-sm">CPU Usage</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="text-2xl font-bold">{engineMetrics.cpuUsage}%</div>
                      <Progress value={engineMetrics.cpuUsage} className="mt-2" />
                    </CardContent>
                  </Card>

                  <Card className="knirv-card-gradient">
                    <CardHeader className="pb-2">
                      <CardTitle className="text-sm">Memory Usage</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="text-2xl font-bold">{engineMetrics.memoryUsage}%</div>
                      <Progress value={engineMetrics.memoryUsage} className="mt-2" />
                    </CardContent>
                  </Card>

                  <Card className="knirv-card-gradient">
                    <CardHeader className="pb-2">
                      <CardTitle className="text-sm">GPU Usage</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="text-2xl font-bold">{engineMetrics.gpuUsage}%</div>
                      <Progress value={engineMetrics.gpuUsage} className="mt-2" />
                    </CardContent>
                  </Card>

                  <Card className="knirv-card-gradient">
                    <CardHeader className="pb-2">
                      <CardTitle className="text-sm">Throughput</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="text-2xl font-bold">{engineMetrics.throughput}</div>
                      <p className="text-xs text-muted-foreground">Requests per second</p>
                    </CardContent>
                  </Card>

                  <Card className="knirv-card-gradient">
                    <CardHeader className="pb-2">
                      <CardTitle className="text-sm">Accuracy</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="text-2xl font-bold">{engineMetrics.accuracy}</div>
                      <p className="text-xs text-muted-foreground">Average fabric accuracy</p>
                    </CardContent>
                  </Card>

                  <Card className="knirv-card-gradient">
                    <CardHeader className="pb-2">
                      <CardTitle className="text-sm">Latency</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="text-2xl font-bold">{engineMetrics.latency}</div>
                      <p className="text-xs text-muted-foreground">Average response time</p>
                    </CardContent>
                  </Card>
                </div>
              </TabsContent>

              <TabsContent value="fabric" className="space-y-4">
                <div className="flex items-center justify-between">
                  <h3 className="text-lg font-semibold">Active Fabric Items</h3>
                  <div className="flex space-x-2">
                    <Button variant="outline" size="sm">
                      <Download className="w-4 h-4 mr-2" />
                      Export Fabric Data
                    </Button>
                    <Button variant="outline" size="sm">
                      <Share2 className="w-4 h-4 mr-2" />
                      Share Performance
                    </Button>
                  </div>
                </div>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  {activeModels.map((model, index) => (
                    <Card key={index} className="knirv-card-gradient">
                      <CardHeader className="pb-2">
                        <div className="flex items-center justify-between">
                          <CardTitle className="text-sm">{model.name}</CardTitle>
                          <Badge variant={model.status === 'active' ? 'default' : 'secondary'}>
                            {model.status}
                          </Badge>
                        </div>
                      </CardHeader>
                      <CardContent className="space-y-2">
                        <div className="flex justify-between text-sm">
                          <span>Accuracy:</span>
                          <span className="font-medium">{model.accuracy}</span>
                        </div>
                        <div className="flex justify-between text-sm">
                          <span>Load:</span>
                          <span className="font-medium">{model.load}</span>
                        </div>
                        <Button variant="outline" size="sm" className="w-full mt-2">
                          <Download className="w-3 h-3 mr-1" />
                          Fabric Reports
                        </Button>
                      </CardContent>
                    </Card>
                  ))}
                </div>
              </TabsContent>

              <TabsContent value="tasks" className="space-y-4">
                <div className="flex items-center justify-between">
                  <h3 className="text-lg font-semibold">Recent Tasks</h3>
                  <div className="flex space-x-2">
                    <Button variant="outline" size="sm">
                      <Download className="w-4 h-4 mr-2" />
                      Export Task Data
                    </Button>
                    <Button variant="outline" size="sm">
                      <Share2 className="w-4 h-4 mr-2" />
                      Share Results
                    </Button>
                  </div>
                </div>
                <div className="space-y-2">
                  {recentTasks.map((task, index) => (
                    <Card key={index} className="knirv-card-gradient">
                      <CardContent className="p-4">
                        <div className="flex items-center justify-between">
                          <div className="flex items-center space-x-4">
                            <div>
                              <p className="font-medium">{task.id}</p>
                              <p className="text-sm text-muted-foreground">{task.type}</p>
                            </div>
                            <Badge variant={
                              task.status === 'completed' ? 'default' : 
                              task.status === 'processing' ? 'secondary' : 
                              'destructive'
                            }>
                              {task.status}
                            </Badge>
                          </div>
                          <div className="flex items-center space-x-4">
                            <div className="w-24">
                              <Progress value={task.progress} />
                            </div>
                            <span className="text-sm font-medium">{task.progress}%</span>
                            <Button variant="outline" size="sm">
                              <Download className="w-3 h-3 mr-1" />
                              Report
                            </Button>
                          </div>
                        </div>
                      </CardContent>
                    </Card>
                  ))}
                </div>
              </TabsContent>

              <TabsContent value="settings" className="space-y-4">
                <Card className="knirv-card-gradient">
                  <CardHeader>
                    <CardTitle className="flex items-center space-x-2">
                      <Settings className="w-5 h-5" />
                      <span>Engine Configuration</span>
                    </CardTitle>
                    <CardDescription>
                      Configure KNIRVENGINE parameters and behavior
                    </CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      <div>
                        <label className="text-sm font-medium">Max Concurrent Tasks</label>
                        <input 
                          type="number" 
                          defaultValue="10" 
                          className="w-full mt-1 px-3 py-2 border rounded-md"
                        />
                      </div>
                      <div>
                        <label className="text-sm font-medium">Memory Limit (GB)</label>
                        <input 
                          type="number" 
                          defaultValue="32" 
                          className="w-full mt-1 px-3 py-2 border rounded-md"
                        />
                      </div>
                      <div>
                        <label className="text-sm font-medium">GPU Memory (GB)</label>
                        <input 
                          type="number" 
                          defaultValue="16" 
                          className="w-full mt-1 px-3 py-2 border rounded-md"
                        />
                      </div>
                      <div>
                        <label className="text-sm font-medium">Timeout (seconds)</label>
                        <input 
                          type="number" 
                          defaultValue="300" 
                          className="w-full mt-1 px-3 py-2 border rounded-md"
                        />
                      </div>
                    </div>
                    <div className="flex space-x-2">
                      <Button variant="default">Save Configuration</Button>
                      <Button variant="outline">Reset to Defaults</Button>
                      <Button variant="outline">
                        <Download className="w-4 h-4 mr-2" />
                        Export Config
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              </TabsContent>
            </Tabs>
          </div>
        </div>
      </div>
    </div>
  );
}
