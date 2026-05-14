import React from 'react';
import { Shield, Key, CheckCircle, AlertTriangle, RefreshCw, Clock } from 'lucide-react';

export const UDCModalContent: React.FC = () => {
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
    <div className="space-y-4">
      {/* Certificate Status */}
      <div className={`bg-gray-800/80 border rounded-lg p-4 ${isExpiringSoon ? 'border-red-500/30' : 'border-green-500/30'}`}>
        <div className="flex items-start justify-between mb-3">
          <div className="flex items-center space-x-3">
            <div className={`w-10 h-10 ${config.bg} ${config.border} border rounded-xl flex items-center justify-center`}>
              <StatusIcon className={`w-5 h-5 ${config.color}`} />
            </div>
            <div>
              <h3 className="text-lg font-bold text-white">Certificate Active</h3>
              <p className={`text-sm ${config.color} capitalize`}>{udc.status}</p>
            </div>
          </div>
          
          {isExpiringSoon && (
            <div className="px-2 py-1 rounded-full bg-red-500/20 border border-red-500/30">
              <span className="text-xs text-red-400 font-medium">Expires Soon</span>
            </div>
          )}
        </div>

        <div className="space-y-3">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <p className="text-xs text-gray-400">Certificate ID</p>
              <p className="text-xs font-mono text-white">{udc.id}</p>
            </div>
            <div>
              <p className="text-xs text-gray-400">Expires In</p>
              <p className={`text-xs font-semibold ${isExpiringSoon ? 'text-red-400' : 'text-green-400'}`}>
                {daysUntilExpiry} days
              </p>
            </div>
          </div>

          <div>
            <p className="text-xs text-gray-400 mb-1">Valid Until</p>
            <p className="text-xs text-white">
              {new Date(udc.expiresAt).toLocaleDateString()} at {new Date(udc.expiresAt).toLocaleTimeString()}
            </p>
          </div>
        </div>
      </div>

      {/* Permissions */}
      <div>
        <h3 className="text-base font-semibold text-white mb-3">Granted Permissions</h3>
        <div className="space-y-2">
          {udc.permissions.map((permission, index) => (
            <div key={index} className="flex items-center justify-between p-2.5 bg-gray-800/80 border border-gray-600/50 rounded-lg">
              <div className="flex items-center space-x-2">
                <div className="w-6 h-6 bg-blue-500/20 rounded-lg flex items-center justify-center border border-blue-500/20">
                  <Key className="w-3.5 h-3.5 text-blue-400" />
                </div>
                <div>
                  <p className="text-xs font-medium text-white">{permission}</p>
                  <p className="text-xs text-gray-400">Full access granted</p>
                </div>
              </div>
              <CheckCircle className="w-4 h-4 text-green-400" />
            </div>
          ))}
        </div>
      </div>

      {/* Actions */}
      <div className="space-y-2">
        <button className="w-full flex items-center justify-center space-x-2 py-3 bg-blue-600 hover:bg-blue-700 rounded-lg text-white font-semibold transition-all text-sm">
          <RefreshCw className="w-4 h-4" />
          <span>Renew Certificate</span>
        </button>

        <div className="grid grid-cols-2 gap-2">
          <button className="flex items-center justify-center space-x-2 py-2.5 bg-gray-800/80 border border-gray-600/50 rounded-lg hover:border-blue-500/50 text-gray-300 hover:text-white transition-all text-sm">
            <Shield className="w-3.5 h-3.5" />
            <span>View Details</span>
          </button>
          <button className="flex items-center justify-center space-x-2 py-2.5 bg-gray-800/80 border border-gray-600/50 rounded-lg hover:border-blue-500/50 text-gray-300 hover:text-white transition-all text-sm">
            <Key className="w-3.5 h-3.5" />
            <span>Export Key</span>
          </button>
        </div>
      </div>

      {/* Certificate Chain */}
      <div>
        <h3 className="text-base font-semibold text-white mb-3">Certificate Chain</h3>
        <div className="space-y-1.5">
          <div className="flex items-center space-x-2 p-2.5 bg-gray-800/30 border border-gray-600/30 rounded-lg">
            <div className="w-1.5 h-1.5 bg-green-400 rounded-full"></div>
            <span className="text-xs text-gray-300">KNIRV Root CA</span>
          </div>
          <div className="flex items-center space-x-2 p-2.5 bg-gray-800/30 border border-gray-600/30 rounded-lg ml-4">
            <div className="w-1.5 h-1.5 bg-green-400 rounded-full"></div>
            <span className="text-xs text-gray-300">D-TEN Intermediate CA</span>
          </div>
          <div className="flex items-center space-x-2 p-2.5 bg-gray-800/30 border border-gray-600/30 rounded-lg ml-6">
            <div className="w-1.5 h-1.5 bg-green-400 rounded-full"></div>
            <span className="text-xs text-gray-300">User Certificate</span>
          </div>
        </div>
      </div>
    </div>
  );
};