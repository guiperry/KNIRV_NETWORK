import * as React from 'react';
import { Check, AlertTriangle, Loader } from 'lucide-react';

interface NetworkStatusProps {
  connections: {
    [key: string]: 'connected' | 'disconnected' | 'connecting';
  };
}

export const NetworkStatus: React.FC<NetworkStatusProps> = ({ connections }) => {
  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'connected': return <Check className="w-4 h-4 text-green-400" />;
      case 'disconnected': return <AlertTriangle className="w-4 h-4 text-red-400" />;
      case 'connecting': return <Loader className="w-4 h-4 text-yellow-400 animate-spin" />;
      default: return <AlertTriangle className="w-4 h-4 text-gray-400" />;
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'connected': return 'text-green-400';
      case 'disconnected': return 'text-red-400';
      case 'connecting': return 'text-yellow-400';
      default: return 'text-gray-400';
    }
  };

  const serviceNames = {
    knirvChain: 'KNIRV-CHAIN',
    knirvGraph: 'KNIRV-GRAPH',
    knirvWallet: 'KNIRV-WALLET',
    knirvRouters: 'KNIRV-ROUTERS',
    knirvana: 'KNIRVANA',
    knirvNexus: 'KNIRV-NEXUS'
  };

  return (
    <div className="space-y-3" data-testid="network-status">
      <div className="text-sm text-gray-400 mb-4">
        Network connections status for all KNIRV services
      </div>
      
      {Object.entries(connections).map(([service, status]) => {
        const statusStr = String(status);
        return (
          <div
            key={service}
            className="flex items-center justify-between p-3 bg-gray-800/50 rounded-lg border border-gray-700/50"
          >
            <div className="flex items-center space-x-3">
              {getStatusIcon(statusStr)}
              <span className="text-white font-medium">
                {serviceNames[service as keyof typeof serviceNames]}
              </span>
            </div>
            <span className={`text-sm font-medium ${getStatusColor(statusStr)}`}>
              {statusStr.charAt(0).toUpperCase() + statusStr.slice(1)}
            </span>
          </div>
        );
      })}
      
      <div className="mt-6 p-3 bg-blue-500/10 rounded-lg border border-blue-500/20">
        <div className="flex items-center space-x-2 mb-2">
          <div className="w-2 h-2 bg-blue-400 rounded-full"></div>
          <span className="text-sm font-medium text-blue-400">zkTLS Active</span>
        </div>
        <p className="text-xs text-gray-400">
          All connections secured with zero-knowledge transport layer security
        </p>
      </div>
    </div>
  );
};