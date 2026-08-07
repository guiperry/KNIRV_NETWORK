import { Shield, Clock, Key, CheckCircle, AlertTriangle, RefreshCw, Loader } from 'lucide-react';
import Layout from '@/react-app/components/Layout';
import { useBackend } from '@/react-app/hooks/useBackend';

export default function UDC() {
  const { udcData, isLoading, refresh } = useBackend();

  if (isLoading || !udcData) {
    return (
      <Layout>
        <div className="flex items-center justify-center min-h-[60vh]">
          <Loader className="w-8 h-8 animate-spin text-blue-500" />
        </div>
      </Layout>
    );
  }

  const daysUntilExpiry = Math.ceil((new Date(udcData.expiresAt).getTime() - new Date().getTime()) / (1000 * 60 * 60 * 24));
  const isExpiringSoon = daysUntilExpiry <= 2;

  const statusConfig = {
    valid: { icon: CheckCircle, color: 'text-green-400', bg: 'bg-green-500/20', border: 'border-green-500/30' },
    expired: { icon: AlertTriangle, color: 'text-red-400', bg: 'bg-red-500/20', border: 'border-red-500/30' },
    revoked: { icon: AlertTriangle, color: 'text-red-400', bg: 'bg-red-500/20', border: 'border-red-500/30' },
    pending: { icon: Clock, color: 'text-yellow-400', bg: 'bg-yellow-500/20', border: 'border-yellow-500/30' }
  };

  const config = statusConfig[udcData.status] || statusConfig.pending;
  const StatusIcon = config.icon;

  return (
    <Layout>
      <div className="p-4 pb-24 space-y-6">
        {/* Header - Gradient Text */}
        <div className="text-center py-4">
          <h2 className="text-2xl font-bold gradient-text mb-2">
            User Delegation Certificate
          </h2>
          <p className="text-slate-400 text-sm font-mono uppercase tracking-wider">
            Your authorized access credentials for the D-TEN network
          </p>
        </div>

        {/* Certificate Status - Glass Panel */}
        <div className="relative group">
          <div className={`absolute -inset-0.5 bg-gradient-to-r ${isExpiringSoon ? 'from-red-600/50 to-orange-600/50' : 'from-blue-600/50 to-blue-800/50'} rounded-xl blur opacity-30 group-hover:opacity-50 transition duration-300`}></div>
          
          <div className={`relative glass-panel p-6 ${isExpiringSoon ? 'border-red-500/30' : ''} glow-hover`}>
            <div className="flex items-start justify-between mb-4">
              <div className="flex items-center space-x-3">
                <div className={`w-12 h-12 ${config.bg} ${config.border} border rounded-xl flex items-center justify-center`}>
                  <StatusIcon className={`w-6 h-6 ${config.color}`} />
                </div>
                <div>
                  <h3 className="text-xl font-bold text-white">Certificate Active</h3>
                  <p className={`text-sm ${config.color} capitalize font-mono`}>{udcData.status}</p>
                </div>
              </div>
              
              {isExpiringSoon && (
                <div className="px-3 py-1 rounded-full bg-red-500/20 border border-red-500/30">
                  <span className="text-xs text-red-400 font-medium font-mono">Expires Soon</span>
                </div>
              )}
            </div>

            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <p className="text-xs text-slate-500 font-mono uppercase tracking-wider">Certificate ID</p>
                  <p className="text-sm font-mono text-white">{udcData.id}</p>
                </div>
                <div>
                  <p className="text-xs text-slate-500 font-mono uppercase tracking-wider">Expires In</p>
                  <p className={`text-sm font-semibold ${isExpiringSoon ? 'text-red-400' : 'text-green-400'} font-mono`}>
                    {daysUntilExpiry} days
                  </p>
                </div>
              </div>

              <div>
                <p className="text-xs text-slate-500 font-mono uppercase tracking-wider mb-1">Valid Until</p>
                <p className="text-sm text-white font-mono">
                  {new Date(udcData.expiresAt).toLocaleDateString()} at {new Date(udcData.expiresAt).toLocaleTimeString()}
                </p>
              </div>
            </div>
          </div>
        </div>

        {/* Permissions */}
        <div>
          <h3 className="text-lg font-semibold text-white mb-4">Granted Permissions</h3>
          <div className="space-y-3">
            {udcData.permissions.map((permission, index) => (
              <div key={index} className="flex items-center justify-between p-3 glass-panel glow-hover">
                <div className="flex items-center space-x-3">
                  <div className="w-8 h-8 bg-blue-500/20 rounded-lg flex items-center justify-center border border-blue-500/20">
                    <Key className="w-4 h-4 text-blue-400" />
                  </div>
                  <div>
                    <p className="text-sm font-medium text-white font-mono">{permission}</p>
                    <p className="text-xs text-slate-400 font-mono">Full access granted</p>
                  </div>
                </div>
                <CheckCircle className="w-5 h-5 text-green-400" />
              </div>
            ))}
          </div>
        </div>

        {/* Actions */}
        <div className="space-y-3">
          <button 
            onClick={refresh}
            className="w-full flex items-center justify-center space-x-3 py-4 bg-blue-600 hover:bg-blue-500 rounded-xl text-white font-semibold transition-all transform hover:scale-[1.02] glow-hover"
          >
            <RefreshCw className="w-5 h-5" />
            <span>Renew Certificate</span>
          </button>

          <div className="grid grid-cols-2 gap-3">
            <button className="flex items-center justify-center space-x-2 py-3 glass-panel text-slate-300 hover:text-white transition-all glow-hover">
              <Shield className="w-4 h-4" />
              <span className="text-sm font-mono">View Details</span>
            </button>
            <button className="flex items-center justify-center space-x-2 py-3 glass-panel text-slate-300 hover:text-white transition-all glow-hover">
              <Key className="w-4 h-4" />
              <span className="text-sm font-mono">Export Key</span>
            </button>
          </div>
        </div>

        {/* Certificate Chain */}
        <div>
          <h3 className="text-lg font-semibold text-white mb-4">Certificate Chain</h3>
          <div className="space-y-2">
            <div className="flex items-center space-x-3 p-3 glass-panel">
              <div className="w-2 h-2 bg-green-400 rounded-full"></div>
              <span className="text-sm text-slate-300 font-mono">KNIRV Root CA</span>
            </div>
            <div className="flex items-center space-x-3 p-3 glass-panel ml-4">
              <div className="w-2 h-2 bg-green-400 rounded-full"></div>
              <span className="text-sm text-slate-300 font-mono">D-TEN Intermediate CA</span>
            </div>
            <div className="flex items-center space-x-3 p-3 glass-panel ml-8">
              <div className="w-2 h-2 bg-green-400 rounded-full"></div>
              <span className="text-sm text-slate-300 font-mono">User Certificate</span>
            </div>
          </div>
        </div>
      </div>
    </Layout>
  );
}
