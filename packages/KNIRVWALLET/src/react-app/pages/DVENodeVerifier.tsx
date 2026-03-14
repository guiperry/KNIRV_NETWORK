import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router';
import { Shield, MapPin, Activity, Clock, Terminal, CheckCircle, Zap, Database, Cpu } from 'lucide-react';
import Layout from '@/react-app/components/Layout';

// Mock types to match the original component
interface TEEEndpoint {
  endpoint_type: 'ssh' | 'validation' | 'error-resolution';
  host: string;
  port: number;
  protocol: string;
}

interface DVENode {
  id: string;
  name: string;
  status: 'online' | 'offline' | 'maintenance' | 'error';
  tee_type: string;
  stake_amount: number;
  reputation_score: number;
  location: string;
  ip_address: string;
  public_key: string;
  capabilities: string[];
  last_heartbeat: string;
  created_at: string;
  updated_at: string;
  cpu_usage: number;
  memory_usage: number;
  network_latency: number;
}

// Simple internal components to replace shadcn
const Badge = ({ children, className }: { children: React.ReactNode, className?: string }) => (
  <span className={`inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 ${className}`}>
    {children}
  </span>
);

const Progress = ({ value, className }: { value: number, className?: string }) => (
  <div className={`relative h-2 w-full overflow-hidden rounded-full bg-slate-800 ${className}`}>
    <div
      className="h-full w-full flex-1 bg-blue-600 transition-all"
      style={{ transform: `translateX(-${100 - (value || 0)}%)` }}
    />
  </div>
);

const Button = ({ children, onClick, variant, size, className }: { children: React.ReactNode, onClick?: () => void, variant?: string, size?: string, className?: string }) => (
  <button
    onClick={onClick}
    className={`inline-flex items-center justify-center rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50 ${variant === 'outline' ? 'border border-blue-600/30 text-blue-400 hover:bg-blue-600 hover:text-white' : 'bg-blue-600 text-white hover:bg-blue-600/90'} ${size === 'sm' ? 'h-8 px-3 text-xs' : 'h-9 px-4 py-2'} ${className}`}
  >
    {children}
  </button>
);

const DVENodeVerifier: React.FC = () => {
  const navigate = useNavigate();
  const [loadingEndpoints, setLoadingEndpoints] = useState(false);
  const [endpoints, setEndpoints] = useState<TEEEndpoint[]>([]);

  const node: DVENode = {
    id: 'DVE-NODE-ALPHA-7',
    name: 'DVE NODE #12345',
    status: 'online',
    tee_type: 'SGX',
    stake_amount: 50000,
    reputation_score: 98,
    location: 'US-EAST-1',
    ip_address: '10.0.4.22',
    public_key: '0x3e4...f2a',
    capabilities: ['validation', 'attestation', 'execution'],
    last_heartbeat: new Date().toISOString(),
    created_at: '2025-01-15T08:00:00Z',
    updated_at: new Date().toISOString(),
    cpu_usage: 42,
    memory_usage: 68,
    network_latency: 12,
  };

  useEffect(() => {
    setLoadingEndpoints(true);
    // Simulate loading endpoints
    setTimeout(() => {
      setEndpoints([
        { endpoint_type: 'ssh', host: '10.0.4.22', port: 2222, protocol: 'tcp' },
        { endpoint_type: 'validation', host: '10.0.4.22', port: 8080, protocol: 'http' },
        { endpoint_type: 'error-resolution', host: '10.0.4.22', port: 9090, protocol: 'grpc' }
      ]);
      setLoadingEndpoints(false);
    }, 1000);
  }, []);

  const getTEEIcon = (teeType?: string) => {
    const type = teeType?.toUpperCase() || 'SOFTWARE';
    switch (type) {
      case 'SGX': return <Shield className="w-5 h-5 text-blue-500" />;
      case 'SEV-SNP': return <Shield className="w-5 h-5 text-green-500" />;
      case 'TDX': return <Shield className="w-5 h-5 text-purple-500" />;
      default: return <Shield className="w-5 h-5 text-gray-500" />;
    }
  };

  const getStatusColor = (status?: string) => {
    const s = status?.toLowerCase() || 'offline';
    switch (s) {
      case 'online': return 'bg-green-500/20 text-green-400 border-green-500/30';
      case 'offline': return 'bg-red-500/20 text-red-400 border-red-500/30';
      case 'maintenance': return 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30';
      case 'error': return 'bg-red-600/20 text-red-500 border-red-600/30';
      default: return 'bg-gray-500/20 text-gray-400 border-gray-500/30';
    }
  };

  return (
    <Layout>
      <div className="p-4 pb-24 max-w-2xl mx-auto space-y-6 mt-8">
        {/* Main Verifier UI - Clone of Metadata Panel */}
        <div className="bg-slate-950 border border-blue-600/50 shadow-[0_0_50px_rgba(0,0,0,0.8)] overflow-hidden rounded-2xl flex flex-col">
          {/* Header */}
          <div className="bg-slate-900 border-b border-blue-600/30 p-5 flex items-center justify-between">
            <div className="flex items-center space-x-4">
              <div className="p-2 bg-blue-600/10 rounded-xl border border-blue-600/20">
                <Database className="w-5 h-5 text-blue-400" />
              </div>
              <div>
                <h2 className="text-lg font-black text-blue-100 uppercase tracking-tighter">{node.name}</h2>
                <div className="text-[9px] font-mono text-slate-500 flex items-center space-x-2">
                  <span>ID: {node.id}</span>
                  <span className="text-blue-500/50">•</span>
                  <span>DVE NODE VERIFIER</span>
                </div>
              </div>
            </div>
            <div className="p-1.5 bg-blue-500/10 rounded-lg border border-blue-500/20 animate-pulse">
              <Activity className="w-5 h-5 text-blue-400" />
            </div>
          </div>

          {/* Content */}
          <div className="p-5 space-y-5">
            {/* Status Bar */}
            <div className="flex items-center justify-between bg-slate-900/50 p-2.5 rounded-xl border border-slate-800">
              <div className="flex items-center space-x-3">
                <Badge className={`${getStatusColor(node.status)} font-black text-[9px] uppercase px-2`}>
                  {node.status}
                </Badge>
                <div className="flex items-center text-[9px] font-bold text-slate-500 uppercase tracking-widest">
                  <Clock className="w-3 h-3 mr-1" />
                  {new Date(node.last_heartbeat).toLocaleTimeString()}
                </div>
              </div>
              <div className="flex items-center text-[9px] font-bold text-blue-500 uppercase tracking-widest">
                <Activity className="w-3 h-3 mr-1" />
                SYNC: 100%
              </div>
            </div>

            {/* Performance Metrics */}
            <div className="grid grid-cols-2 gap-4">
              <div className="bg-slate-900/50 border border-slate-800 rounded-xl p-3 space-y-2">
                <div className="flex justify-between items-center">
                  <div className="flex items-center text-[9px] font-black text-slate-400 uppercase">
                    <Cpu className="w-3 h-3 mr-1.5 text-blue-500" />
                    Compute
                  </div>
                  <span className="text-[10px] font-mono text-blue-400 font-bold">{node.cpu_usage}%</span>
                </div>
                <Progress value={node.cpu_usage} className="h-1 bg-slate-800" />
              </div>

              <div className="bg-slate-900/50 border border-slate-800 rounded-xl p-3 space-y-2">
                <div className="flex justify-between items-center">
                  <div className="flex items-center text-[9px] font-black text-slate-400 uppercase">
                    <Zap className="w-3 h-3 mr-1.5 text-purple-500" />
                    Memory
                  </div>
                  <span className="text-[10px] font-mono text-purple-400 font-bold">{node.memory_usage}%</span>
                </div>
                <Progress value={node.memory_usage} className="h-1 bg-slate-800" />
              </div>
            </div>

            {/* Node Metadata Grid */}
            <div className="grid grid-cols-2 gap-3">
              <div className="bg-slate-950 border border-slate-800 rounded-xl p-3 flex items-center justify-between">
                <div className="space-y-0.5">
                  <p className="text-[8px] font-black text-slate-500 uppercase tracking-widest">Hardware TEE</p>
                  <div className="flex items-center space-x-1.5">
                    {getTEEIcon(node.tee_type)}
                    <span className="text-[10px] font-bold text-slate-200">{node.tee_type}</span>
                  </div>
                </div>
              </div>

              <div className="bg-slate-950 border border-slate-800 rounded-xl p-3 flex items-center justify-between">
                <div className="space-y-0.5">
                  <p className="text-[8px] font-black text-slate-500 uppercase tracking-widest">Geographic Loc</p>
                  <div className="flex items-center space-x-1.5">
                    <MapPin className="w-3.5 h-3.5 text-red-500" />
                    <span className="text-[10px] font-bold text-slate-200">{node.location}</span>
                  </div>
                </div>
              </div>

              <div className="bg-slate-950 border border-slate-800 rounded-xl p-3 flex items-center justify-between">
                <div className="space-y-0.5">
                  <p className="text-[8px] font-black text-slate-500 uppercase tracking-widest">Economic Stake</p>
                  <span className="text-[10px] font-bold text-green-500">{node.stake_amount.toLocaleString()} NRN</span>
                </div>
              </div>

              <div className="bg-slate-950 border border-slate-800 rounded-xl p-3 flex items-center justify-between">
                <div className="space-y-0.5">
                  <p className="text-[8px] font-black text-slate-500 uppercase tracking-widest">Trust Score</p>
                  <span className={`text-[10px] font-bold ${node.reputation_score > 80 ? 'text-blue-400' : 'text-yellow-500'}`}>
                    {node.reputation_score}/100 VERIFIED
                  </span>
                </div>
              </div>
            </div>

            {/* Endpoints */}
            <div className="space-y-3">
              <h3 className="text-[9px] font-black text-blue-500 uppercase tracking-[0.2em]">Hardware Endpoints</h3>
              
              {loadingEndpoints ? (
                <div className="flex items-center justify-center py-4">
                  <Activity className="w-4 h-4 animate-spin text-blue-500" />
                </div>
              ) : (
                <div className="space-y-2">
                  {endpoints.map((endpoint, index) => (
                    <div key={index} className="flex items-center justify-between p-3 bg-slate-900 hover:bg-slate-800 transition-colors border border-slate-800 rounded-xl group">
                      <div className="flex items-center space-x-3">
                        <div className="p-1.5 bg-slate-950 rounded-lg border border-slate-800 group-hover:border-blue-500/30 transition-colors">
                          {endpoint.endpoint_type === 'ssh' && <Terminal className="w-3.5 h-3.5 text-green-400" />}
                          {endpoint.endpoint_type === 'validation' && <CheckCircle className="w-3.5 h-3.5 text-blue-400" />}
                          {endpoint.endpoint_type === 'error-resolution' && <Zap className="w-3.5 h-3.5 text-orange-400" />}
                        </div>
                        <div>
                          <p className="text-[10px] font-black text-slate-200 uppercase tracking-tight">
                            {endpoint.endpoint_type.replace('-', ' ')} ACCESS
                          </p>
                          <p className="text-[9px] font-mono text-slate-500">
                            {endpoint.host}:{endpoint.port} • {endpoint.protocol.toUpperCase()}
                          </p>
                        </div>
                      </div>
                      <Button
                        size="sm"
                        variant="outline"
                        className="h-7 text-[9px] font-black uppercase border-blue-600/30 text-blue-400 hover:bg-blue-600 hover:text-white px-3"
                      >
                        Initialize
                      </Button>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>

          {/* Footer */}
          <div className="bg-slate-900 border-t border-blue-600/30 p-3 flex justify-between items-center">
            <span className="text-[8px] font-mono text-slate-500 ml-2">VERSION 1.2.0-STABLE</span>
            <Button
              className="text-white text-[9px] font-bold uppercase tracking-widest h-8 bg-blue-600 px-6"
              onClick={() => navigate('/workflows')}
            >
              View Activity
            </Button>
          </div>
        </div>
      </div>
    </Layout>
  );
};

export default DVENodeVerifier;
