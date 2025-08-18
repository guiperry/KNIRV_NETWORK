import { Shield, Clock, Key, CheckCircle, AlertTriangle, RefreshCw } from 'lucide-react';
import Layout from '@/react-app/components/Layout';

export default function UDC() {
  const udc = {
    id: 'UDC-7A8B9C2D',
    status: 'valid' as const,
    issuedAt: '2024-08-01T10:30:00Z',
    expiresAt: '2024-08-08T10:30:00Z',
    permissions: [
      'agent.deploy',
      'skill.activate',
      'nrn.transfer',
      'dten.access',
      'wallet.connect'
    ]
  };

  const daysUntilExpiry = Math.ceil((new Date(udc.expiresAt).getTime() - new Date().getTime()) / (1000 * 60 * 60 * 24));
  const isExpiringSoon = daysUntilExpiry <= 2;

  const statusConfig = {
    valid: { icon: CheckCircle, color: 'text-green-400', bg: 'bg-green-500/20', border: 'border-green-500/30' },
    expired: { icon: AlertTriangle, color: 'text-red-400', bg: 'bg-red-500/20', border: 'border-red-500/30' },
    revoked: { icon: AlertTriangle, color: 'text-red-400', bg: 'bg-red-500/20', border: 'border-red-500/30' },
    pending: { icon: Clock, color: 'text-yellow-400', bg: 'bg-yellow-500/20', border: 'border-yellow-500/30' }
  };

  const config = statusConfig[udc.status];
  const StatusIcon = config.icon;

  return (
    <Layout>
      <div className="p-4 pb-24 space-y-6">
        {/* Header */}
        <div className="text-center py-4">
          <h2 className="text-2xl font-bold bg-gradient-to-r from-purple-400 to-cyan-400 bg-clip-text text-transparent mb-2">
            User Delegation Certificate
          </h2>
          <p className="text-slate-400 text-sm">
            Your authorized access credentials for the D-TEN network
          </p>
        </div>

        {/* Certificate Status */}
        <div className="relative group">
          <div className={`absolute -inset-0.5 bg-gradient-to-r ${isExpiringSoon ? 'from-red-600/50 to-orange-600/50' : 'from-green-600/50 to-cyan-600/50'} rounded-xl blur opacity-30 group-hover:opacity-50 transition duration-300`}></div>
          
          <div className={`relative bg-slate-800/90 backdrop-blur-xl rounded-xl p-6 border ${isExpiringSoon ? 'border-red-500/30' : 'border-green-500/30'}`}>
            <div className="flex items-start justify-between mb-4">
              <div className="flex items-center space-x-3">
                <div className={`w-12 h-12 ${config.bg} ${config.border} border rounded-xl flex items-center justify-center`}>
                  <StatusIcon className={`w-6 h-6 ${config.color}`} />
                </div>
                <div>
                  <h3 className="text-xl font-bold text-white">Certificate Active</h3>
                  <p className={`text-sm ${config.color} capitalize`}>{udc.status}</p>
                </div>
              </div>
              
              {isExpiringSoon && (
                <div className="px-3 py-1 rounded-full bg-red-500/20 border border-red-500/30">
                  <span className="text-xs text-red-400 font-medium">Expires Soon</span>
                </div>
              )}
            </div>

            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <p className="text-sm text-slate-400">Certificate ID</p>
                  <p className="text-sm font-mono text-white">{udc.id}</p>
                </div>
                <div>
                  <p className="text-sm text-slate-400">Expires In</p>
                  <p className={`text-sm font-semibold ${isExpiringSoon ? 'text-red-400' : 'text-green-400'}`}>
                    {daysUntilExpiry} days
                  </p>
                </div>
              </div>

              <div>
                <p className="text-sm text-slate-400 mb-2">Valid Until</p>
                <p className="text-sm text-white">
                  {new Date(udc.expiresAt).toLocaleDateString()} at {new Date(udc.expiresAt).toLocaleTimeString()}
                </p>
              </div>
            </div>
          </div>
        </div>

        {/* Permissions */}
        <div>
          <h3 className="text-lg font-semibold text-white mb-4">Granted Permissions</h3>
          <div className="space-y-3">
            {udc.permissions.map((permission, index) => (
              <div key={index} className="flex items-center justify-between p-3 bg-slate-800/60 backdrop-blur-xl rounded-xl border border-slate-700/50">
                <div className="flex items-center space-x-3">
                  <div className="w-8 h-8 bg-gradient-to-br from-purple-500/20 to-cyan-500/20 rounded-lg flex items-center justify-center border border-purple-500/20">
                    <Key className="w-4 h-4 text-purple-400" />
                  </div>
                  <div>
                    <p className="text-sm font-medium text-white">{permission}</p>
                    <p className="text-xs text-slate-400">Full access granted</p>
                  </div>
                </div>
                <CheckCircle className="w-5 h-5 text-green-400" />
              </div>
            ))}
          </div>
        </div>

        {/* Actions */}
        <div className="space-y-3">
          <button className="w-full flex items-center justify-center space-x-3 py-4 bg-gradient-to-r from-purple-600 to-cyan-600 hover:from-purple-700 hover:to-cyan-700 rounded-xl text-white font-semibold transition-all transform hover:scale-[1.02]">
            <RefreshCw className="w-5 h-5" />
            <span>Renew Certificate</span>
          </button>

          <div className="grid grid-cols-2 gap-3">
            <button className="flex items-center justify-center space-x-2 py-3 bg-slate-800/60 backdrop-blur-xl rounded-xl border border-slate-700/50 hover:border-purple-500/50 text-slate-300 hover:text-white transition-all">
              <Shield className="w-4 h-4" />
              <span className="text-sm">View Details</span>
            </button>
            <button className="flex items-center justify-center space-x-2 py-3 bg-slate-800/60 backdrop-blur-xl rounded-xl border border-slate-700/50 hover:border-purple-500/50 text-slate-300 hover:text-white transition-all">
              <Key className="w-4 h-4" />
              <span className="text-sm">Export Key</span>
            </button>
          </div>
        </div>

        {/* Certificate Chain */}
        <div>
          <h3 className="text-lg font-semibold text-white mb-4">Certificate Chain</h3>
          <div className="space-y-2">
            <div className="flex items-center space-x-3 p-3 bg-slate-800/30 backdrop-blur-xl rounded-lg border border-slate-700/30">
              <div className="w-2 h-2 bg-green-400 rounded-full"></div>
              <span className="text-sm text-slate-300">KNIRV Root CA</span>
            </div>
            <div className="flex items-center space-x-3 p-3 bg-slate-800/30 backdrop-blur-xl rounded-lg border border-slate-700/30 ml-4">
              <div className="w-2 h-2 bg-green-400 rounded-full"></div>
              <span className="text-sm text-slate-300">D-TEN Intermediate CA</span>
            </div>
            <div className="flex items-center space-x-3 p-3 bg-slate-800/30 backdrop-blur-xl rounded-lg border border-slate-700/30 ml-8">
              <div className="w-2 h-2 bg-green-400 rounded-full"></div>
              <span className="text-sm text-slate-300">User Certificate</span>
            </div>
          </div>
        </div>
      </div>
    </Layout>
  );
}
