'use client';

import React, { useState, useEffect } from 'react';
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { 
  Cloud,
  Play,
  Square,
  RotateCcw,
  Activity,
  Database,
  Cpu,
  HardDrive,
  Globe,
  Lock,
  Terminal,
  Download,
  ExternalLink,
  CheckCircle2,
  AlertCircle,
  Loader2,
  ChevronRight,
  Copy,
  Settings,
  BarChart3,
  Clock,
  Home
} from "lucide-react";
import { useToast } from "@/hooks/use-toast";

interface CloudCortexInfoCardProps {
  onReset?: () => void;
}

interface ServiceStatus {
  status: 'running' | 'stopped' | 'starting' | 'stopping' | 'error';
  uptime: string;
  lastStarted: string;
}

interface Metrics {
  cpuUsage: number;
  memoryUsage: number;
  storageUsed: number;
  storageTotal: number;
  requestsPerMinute: number;
  activeConnections: number;
}

const CloudCortexInfoCard = ({ onReset }: CloudCortexInfoCardProps) => {
  const { toast } = useToast();
  const [serviceStatus, setServiceStatus] = useState<ServiceStatus>({
    status: 'running',
    uptime: '2d 14h 32m',
    lastStarted: new Date().toISOString()
  });
  
  const [metrics, setMetrics] = useState<Metrics>({
    cpuUsage: 34,
    memoryUsage: 42,
    storageUsed: 156,
    storageTotal: 500,
    requestsPerMinute: 142,
    activeConnections: 8
  });

  const [isLoading, setIsLoading] = useState(false);
  const [copiedEndpoint, setCopiedEndpoint] = useState<string | null>(null);

  // Simulate metrics updates
  useEffect(() => {
    const interval = setInterval(() => {
      setMetrics(prev => ({
        ...prev,
        cpuUsage: Math.max(10, Math.min(90, prev.cpuUsage + (Math.random() - 0.5) * 10)),
        memoryUsage: Math.max(20, Math.min(80, prev.memoryUsage + (Math.random() - 0.5) * 5)),
        requestsPerMinute: Math.max(50, Math.min(300, prev.requestsPerMinute + Math.floor((Math.random() - 0.5) * 20))),
      }));
    }, 3000);

    return () => clearInterval(interval);
  }, []);

  const handleServiceAction = async (action: 'start' | 'stop' | 'restart') => {
    setIsLoading(true);
    
    // Simulate API call
    await new Promise(resolve => setTimeout(resolve, 1500));
    
    if (action === 'start') {
      setServiceStatus(prev => ({ ...prev, status: 'running', lastStarted: new Date().toISOString() }));
      toast({ title: "Service Started", description: "Cloud Cortex is now running" });
    } else if (action === 'stop') {
      setServiceStatus(prev => ({ ...prev, status: 'stopped' }));
      toast({ title: "Service Stopped", description: "Cloud Cortex has been stopped" });
    } else if (action === 'restart') {
      setServiceStatus(prev => ({ ...prev, status: 'running', lastStarted: new Date().toISOString() }));
      toast({ title: "Service Restarted", description: "Cloud Cortex has been restarted" });
    }
    
    setIsLoading(false);
  };

  const copyToClipboard = (text: string, label: string) => {
    navigator.clipboard.writeText(text);
    setCopiedEndpoint(label);
    toast({ title: "Copied!", description: `${label} copied to clipboard` });
    setTimeout(() => setCopiedEndpoint(null), 2000);
  };

  const getStatusColor = (status: ServiceStatus['status']) => {
    switch (status) {
      case 'running': return 'text-green-400 bg-green-400/10 border-green-400/30';
      case 'stopped': return 'text-slate-400 bg-slate-400/10 border-slate-400/30';
      case 'starting':
      case 'stopping': return 'text-amber-400 bg-amber-400/10 border-amber-400/30';
      case 'error': return 'text-red-400 bg-red-400/10 border-red-400/30';
      default: return 'text-slate-400';
    }
  };

  const apiEndpoints = [
    { name: 'REST API', url: 'https://cortex-api.knirv.network/v1', type: 'Primary' },
    { name: 'GraphQL', url: 'https://cortex-api.knirv.network/graphql', type: 'Query' },
    { name: 'WebSocket', url: 'wss://cortex-ws.knirv.network/realtime', type: 'Realtime' },
    { name: 'Admin Panel', url: 'https://cortex-admin.knirv.network', type: 'Management' }
  ];

  return (
    <div className="min-h-screen bg-[#0a0a0c] text-slate-200 font-sans selection:bg-blue-500/30">
      {/* Background Effects */}
      <div className="fixed inset-0 overflow-hidden pointer-events-none opacity-20">
        <div className="absolute top-[-10%] left-[-10%] w-[40%] h-[40%] bg-blue-600/20 blur-[120px] rounded-full" />
        <div className="absolute bottom-[-10%] right-[-10%] w-[40%] h-[40%] bg-indigo-600/10 blur-[120px] rounded-full" />
      </div>

      {/* Header */}
      <nav className="relative z-10 p-6 flex justify-between items-center border-b border-white/5 bg-black/40 backdrop-blur-md">
        <div className="flex items-center space-x-2">
          <div className="w-6 h-6 bg-blue-600 rounded-sm transform rotate-45" />
          <span className="text-xl font-extrabold tracking-tighter uppercase">KNIRV <span className="text-blue-500 font-light italic">CORTEX</span></span>
        </div>
        <div className="flex items-center space-x-4">
          <Badge className={getStatusColor(serviceStatus.status)}>
            <span className={`flex h-2 w-2 rounded-full mr-2 ${
              serviceStatus.status === 'running' ? 'bg-green-500 animate-pulse' : 
              serviceStatus.status === 'stopped' ? 'bg-slate-500' : 'bg-amber-500'
            }`} />
            {serviceStatus.status.toUpperCase()}
          </Badge>
          {onReset && (
            <button
              onClick={onReset}
              className="flex items-center space-x-1 text-xs text-slate-500 hover:text-red-400 transition-colors ml-4"
              title="Return to home"
            >
              <Home size={14} />
              <span className="hidden sm:inline">Exit</span>
            </button>
          )}
        </div>
      </nav>

      <main className="relative z-10 max-w-7xl mx-auto px-6 py-8">
        {/* Welcome Banner */}
        <div className="mb-8 p-6 bg-gradient-to-r from-blue-600/10 to-indigo-600/10 border border-blue-500/20 rounded-2xl">
          <div className="flex items-start space-x-4">
            <div className="p-3 bg-blue-600/20 rounded-xl">
              <CheckCircle2 className="text-blue-500" size={32} />
            </div>
            <div className="flex-1">
              <h1 className="text-2xl font-bold mb-2">Your Private Cloud Cortex is Ready</h1>
              <p className="text-slate-400">
                Welcome to your personal data vault. Your Cloud Cortex is now active and ready to process your data securely.
              </p>
            </div>
          </div>
        </div>

        <div className="grid lg:grid-cols-3 gap-6">
          {/* Left Column - Controls & Status */}
          <div className="space-y-6">
            {/* Service Controls */}
            <Card className="bg-white/5 border-white/10">
              <CardHeader>
                <CardTitle className="text-white flex items-center text-lg">
                  <Settings className="mr-2 text-blue-500" size={20} />
                  Service Controls
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex items-center justify-between p-3 bg-slate-800/50 rounded-lg">
                  <div>
                    <div className="text-sm font-medium text-slate-300">Status</div>
                    <div className={`text-xs ${serviceStatus.status === 'running' ? 'text-green-400' : 'text-slate-500'}`}>
                      {serviceStatus.status === 'running' ? '● Online' : '○ Offline'}
                    </div>
                  </div>
                  <div className="flex space-x-2">
                    {serviceStatus.status === 'running' ? (
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleServiceAction('stop')}
                        disabled={isLoading}
                        className="border-red-500/30 text-red-400 hover:bg-red-500/10"
                      >
                        {isLoading ? <Loader2 className="w-4 h-4 animate-spin" /> : <Square className="w-4 h-4" />}
                        <span className="ml-2">Stop</span>
                      </Button>
                    ) : (
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleServiceAction('start')}
                        disabled={isLoading}
                        className="border-green-500/30 text-green-400 hover:bg-green-500/10"
                      >
                        {isLoading ? <Loader2 className="w-4 h-4 animate-spin" /> : <Play className="w-4 h-4" />}
                        <span className="ml-2">Start</span>
                      </Button>
                    )}
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleServiceAction('restart')}
                      disabled={isLoading || serviceStatus.status !== 'running'}
                      className="border-amber-500/30 text-amber-400 hover:bg-amber-500/10"
                    >
                      {isLoading ? <Loader2 className="w-4 h-4 animate-spin" /> : <RotateCcw className="w-4 h-4" />}
                    </Button>
                  </div>
                </div>

                <div className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <span className="text-slate-500">Uptime</span>
                    <span className="text-slate-300 font-mono">{serviceStatus.uptime}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-slate-500">Last Started</span>
                    <span className="text-slate-300 font-mono">{new Date(serviceStatus.lastStarted).toLocaleTimeString()}</span>
                  </div>
                </div>
              </CardContent>
            </Card>

            {/* API Endpoints */}
            <Card className="bg-white/5 border-white/10">
              <CardHeader>
                <CardTitle className="text-white flex items-center text-lg">
                  <Globe className="mr-2 text-blue-500" size={20} />
                  Private API Endpoints
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                {apiEndpoints.map((endpoint) => (
                  <div key={endpoint.name} className="p-3 bg-slate-800/50 rounded-lg space-y-2">
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-medium text-slate-300">{endpoint.name}</span>
                      <Badge variant="outline" className="text-xs border-blue-500/30 text-blue-400">
                        {endpoint.type}
                      </Badge>
                    </div>
                    <div className="flex items-center space-x-2">
                      <code className="flex-1 text-xs text-slate-500 font-mono truncate">
                        {endpoint.url}
                      </code>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => copyToClipboard(endpoint.url, endpoint.name)}
                        className="h-6 w-6 p-0"
                      >
                        {copiedEndpoint === endpoint.name ? (
                          <CheckCircle2 className="w-3 h-3 text-green-400" />
                        ) : (
                          <Copy className="w-3 h-3 text-slate-400" />
                        )}
                      </Button>
                    </div>
                  </div>
                ))}
              </CardContent>
            </Card>

            {/* Download Controller */}
            <Card className="bg-gradient-to-br from-blue-600/10 to-indigo-600/10 border-blue-500/20">
              <CardContent className="p-6">
                <div className="flex items-start space-x-4">
                  <div className="p-3 bg-blue-600/20 rounded-lg">
                    <Download className="text-blue-500" size={24} />
                  </div>
                  <div className="flex-1">
                    <h3 className="font-bold text-white mb-1">Download Controller App</h3>
                    <p className="text-sm text-slate-400 mb-4">
                      Install the desktop controller for enhanced management
                    </p>
                    <Button
                      variant="outline"
                      className="w-full border-blue-500/30 text-blue-400 hover:bg-blue-500/10"
                      onClick={() => window.open('https://releases.knirv.network/knirvcontroller-desktop.zip', '_blank')}
                    >
                      <Download className="w-4 h-4 mr-2" />
                      Download for Desktop
                    </Button>
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>

          {/* Middle & Right Columns - Metrics & Admin */}
          <div className="lg:col-span-2 space-y-6">
            <Tabs defaultValue="metrics" className="w-full">
              <TabsList className="grid w-full grid-cols-3 bg-white/5 border border-white/10">
                <TabsTrigger value="metrics" className="data-[state=active]:bg-blue-600/20">
                  <BarChart3 className="w-4 h-4 mr-2" />
                  Metrics
                </TabsTrigger>
                <TabsTrigger value="resources" className="data-[state=active]:bg-blue-600/20">
                  <HardDrive className="w-4 h-4 mr-2" />
                  Resources
                </TabsTrigger>
                <TabsTrigger value="admin" className="data-[state=active]:bg-blue-600/20">
                  <Terminal className="w-4 h-4 mr-2" />
                  Admin
                </TabsTrigger>
              </TabsList>

              <TabsContent value="metrics" className="space-y-4 mt-6">
                {/* Metrics Cards */}
                <div className="grid grid-cols-2 gap-4">
                  <Card className="bg-white/5 border-white/10">
                    <CardContent className="p-6">
                      <div className="flex items-center justify-between mb-4">
                        <div className="p-2 bg-blue-600/10 rounded-lg">
                          <Activity className="text-blue-500" size={20} />
                        </div>
                        <span className="text-2xl font-bold text-white">{metrics.requestsPerMinute}</span>
                      </div>
                      <div className="text-sm text-slate-500">Requests/min</div>
                      <div className="mt-2 text-xs text-green-400">+12% from last hour</div>
                    </CardContent>
                  </Card>

                  <Card className="bg-white/5 border-white/10">
                    <CardContent className="p-6">
                      <div className="flex items-center justify-between mb-4">
                        <div className="p-2 bg-green-600/10 rounded-lg">
                          <Cloud className="text-green-500" size={20} />
                        </div>
                        <span className="text-2xl font-bold text-white">{metrics.activeConnections}</span>
                      </div>
                      <div className="text-sm text-slate-500">Active Connections</div>
                      <div className="mt-2 text-xs text-slate-400">Stable</div>
                    </CardContent>
                  </Card>
                </div>

                {/* Usage Charts Placeholder */}
                <Card className="bg-white/5 border-white/10">
                  <CardHeader>
                    <CardTitle className="text-white text-lg">Resource Usage</CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-6">
                    {/* CPU Usage */}
                    <div>
                      <div className="flex justify-between text-sm mb-2">
                        <span className="text-slate-400 flex items-center">
                          <Cpu className="w-4 h-4 mr-2" />
                          CPU Usage
                        </span>
                        <span className="text-white font-mono">{Math.round(metrics.cpuUsage)}%</span>
                      </div>
                      <div className="h-2 bg-slate-700 rounded-full overflow-hidden">
                        <div 
                          className="h-full bg-blue-500 transition-all duration-500"
                          style={{ width: `${metrics.cpuUsage}%` }}
                        />
                      </div>
                    </div>

                    {/* Memory Usage */}
                    <div>
                      <div className="flex justify-between text-sm mb-2">
                        <span className="text-slate-400 flex items-center">
                          <Database className="w-4 h-4 mr-2" />
                          Memory Usage
                        </span>
                        <span className="text-white font-mono">{Math.round(metrics.memoryUsage)}%</span>
                      </div>
                      <div className="h-2 bg-slate-700 rounded-full overflow-hidden">
                        <div 
                          className="h-full bg-green-500 transition-all duration-500"
                          style={{ width: `${metrics.memoryUsage}%` }}
                        />
                      </div>
                    </div>

                    {/* Storage Usage */}
                    <div>
                      <div className="flex justify-between text-sm mb-2">
                        <span className="text-slate-400 flex items-center">
                          <HardDrive className="w-4 h-4 mr-2" />
                          Storage
                        </span>
                        <span className="text-white font-mono">{metrics.storageUsed}GB / {metrics.storageTotal}GB</span>
                      </div>
                      <div className="h-2 bg-slate-700 rounded-full overflow-hidden">
                        <div 
                          className="h-full bg-blue-500 transition-all duration-500"
                          style={{ width: `${(metrics.storageUsed / metrics.storageTotal) * 100}%` }}
                        />
                      </div>
                    </div>
                  </CardContent>
                </Card>
              </TabsContent>

              <TabsContent value="resources" className="space-y-4 mt-6">
                <Card className="bg-white/5 border-white/10">
                  <CardHeader>
                    <CardTitle className="text-white text-lg">Data Resources</CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <div className="grid grid-cols-3 gap-4">
                      <div className="p-4 bg-slate-800/50 rounded-lg text-center">
                        <div className="text-2xl font-bold text-blue-400 mb-1">1.2TB</div>
                        <div className="text-xs text-slate-500">Total Data Stored</div>
                      </div>
                      <div className="p-4 bg-slate-800/50 rounded-lg text-center">
                        <div className="text-2xl font-bold text-green-400 mb-1">847</div>
                        <div className="text-xs text-slate-500">Active Datasets</div>
                      </div>
                      <div className="p-4 bg-slate-800/50 rounded-lg text-center">
                        <div className="text-2xl font-bold text-blue-400 mb-1">12ms</div>
                        <div className="text-xs text-slate-500">Avg Response Time</div>
                      </div>
                    </div>

                    <div className="p-4 bg-slate-800/30 rounded-lg">
                      <h4 className="font-medium text-slate-300 mb-3">Resource Allocation</h4>
                      <div className="space-y-2 text-sm">
                        <div className="flex justify-between">
                          <span className="text-slate-500">Compute Units</span>
                          <span className="text-slate-300">4 vCPU / 16GB RAM</span>
                        </div>
                        <div className="flex justify-between">
                          <span className="text-slate-500">Storage Class</span>
                          <span className="text-slate-300">SSD Standard</span>
                        </div>
                        <div className="flex justify-between">
                          <span className="text-slate-500">Network Bandwidth</span>
                          <span className="text-slate-300">1 Gbps</span>
                        </div>
                        <div className="flex justify-between">
                          <span className="text-slate-500">Region</span>
                          <span className="text-slate-300">us-east-1 (N. Virginia)</span>
                        </div>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              </TabsContent>

              <TabsContent value="admin" className="space-y-4 mt-6">
                <Card className="bg-white/5 border-white/10">
                  <CardHeader>
                    <CardTitle className="text-white text-lg">Administration</CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <div className="p-4 bg-slate-800/30 rounded-lg border border-white/5">
                      <div className="flex items-center justify-between mb-4">
                        <div>
                          <h4 className="font-medium text-slate-300">Observability Dashboard</h4>
                          <p className="text-xs text-slate-500">View logs, metrics, and traces</p>
                        </div>
                        <Button
                          variant="outline"
                          onClick={() => window.open('https://cortex-admin.knirv.network', '_blank')}
                          className="border-blue-500/30 text-blue-400 hover:bg-blue-500/10"
                        >
                          <ExternalLink className="w-4 h-4 mr-2" />
                          Open Admin Panel
                        </Button>
                      </div>
                    </div>

                    <div className="grid grid-cols-2 gap-4">
                      <div className="p-4 bg-slate-800/30 rounded-lg">
                        <div className="flex items-center space-x-2 mb-2">
                          <Clock className="w-4 h-4 text-slate-500" />
                          <span className="text-sm text-slate-400">Last Backup</span>
                        </div>
                        <div className="text-lg font-mono text-slate-300">2 hours ago</div>
                      </div>
                      <div className="p-4 bg-slate-800/30 rounded-lg">
                        <div className="flex items-center space-x-2 mb-2">
                          <Lock className="w-4 h-4 text-slate-500" />
                          <span className="text-sm text-slate-400">Security Status</span>
                        </div>
                        <div className="text-lg font-medium text-green-400">Secured</div>
                      </div>
                    </div>

                    <div className="p-4 bg-amber-500/5 border border-amber-500/20 rounded-lg">
                      <div className="flex items-start space-x-3">
                        <AlertCircle className="text-amber-500 shrink-0 mt-0.5" size={16} />
                        <div>
                          <h4 className="text-sm font-medium text-amber-400 mb-1">Usage Instructions</h4>
                          <ul className="text-xs text-slate-400 space-y-1">
                            <li>• Use the API endpoints to connect your applications</li>
                            <li>• Monitor resource usage to optimize costs</li>
                            <li>• Access logs via the Admin Panel for debugging</li>
                            <li>• Regular backups are performed automatically</li>
                          </ul>
                        </div>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              </TabsContent>
            </Tabs>
          </div>
        </div>

        {/* Footer Actions */}
        <div className="mt-8 flex justify-between items-center border-t border-white/5 pt-6">
          <div className="text-xs text-slate-500">
            Cloud Cortex v2.1.4 • Instance ID: ctx-{Math.random().toString(36).substr(2, 8).toUpperCase()}
          </div>
          {onReset && (
            <Button
              variant="ghost"
              onClick={onReset}
              className="text-slate-500 hover:text-white text-sm"
            >
              Return to Home
            </Button>
          )}
        </div>
      </main>

      {/* Footer Meta */}
      <footer className="max-w-7xl mx-auto p-12 text-center">
        <div className="inline-flex items-center space-x-4 px-6 py-2 rounded-full border border-white/5 bg-white/5 text-[10px] mono text-slate-500 font-bold uppercase tracking-[0.2em]">
          <span className="flex h-2 w-2 rounded-full bg-green-500 animate-pulse" />
          <span>Nexus Network Secure</span>
          <span className="h-3 w-[1px] bg-white/10" />
          <span>Sovereign Encryption Active</span>
        </div>
      </footer>
    </div>
  );
};

export default CloudCortexInfoCard;
