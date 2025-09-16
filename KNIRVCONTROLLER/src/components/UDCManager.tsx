import * as React from 'react';
import { useState, useEffect, useCallback } from 'react';
import {
  Shield,
  Plus,
  RefreshCw,
  CheckCircle,
  XCircle,
  AlertTriangle,
  Clock,
  Eye,
  Trash2
} from 'lucide-react';
import { udcManagementService, UDC, UDCRequest, UDCValidationResult } from '../services/UDCManagementService';

interface UDCManagerProps {
  isOpen: boolean;
  onClose: () => void;
}

const UDCManager: React.FC<UDCManagerProps> = ({ isOpen, onClose }) => {
  const [udcs, setUdcs] = useState<UDC[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [activeTab, setActiveTab] = useState<'list' | 'create' | 'expiring'>('list');
  const [selectedUDC, setSelectedUDC] = useState<UDC | null>(null);
  const [validationResults, setValidationResults] = useState<Record<string, UDCValidationResult>>({});

  const loadUDCs = useCallback(async () => {
    setIsLoading(true);
    try {
      const allUDCs = udcManagementService.getAllUDCs();
      setUdcs(allUDCs);
      
      // Validate each UDC
      const validations: Record<string, UDCValidationResult> = {};
      for (const udc of allUDCs) {
        validations[udc.id] = await udcManagementService.validateUDC(udc.id);
      }
      setValidationResults(validations);
    } catch (error) {
      console.error('Failed to load UDCs:', error);
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    if (isOpen) {
      loadUDCs();
      // Auto-refresh every 30 seconds
      const interval = setInterval(loadUDCs, 30000);
      return () => clearInterval(interval);
    }
  }, [isOpen, loadUDCs]);

  const handleCreateUDC = async (request: UDCRequest) => {
    try {
      await udcManagementService.createUDC(request);
      await loadUDCs();
      setActiveTab('list');
    } catch (error) {
      console.error('Failed to create UDC:', error);
    }
  };

  const handleRenewUDC = async (udcId: string, days: number) => {
    try {
      await udcManagementService.renewUDC(udcId, days);
      await loadUDCs();
    } catch (error) {
      console.error('Failed to renew UDC:', error);
    }
  };

  const handleRevokeUDC = async (udcId: string, reason: string) => {
    try {
      await udcManagementService.revokeUDC(udcId, reason);
      await loadUDCs();
    } catch (error) {
      console.error('Failed to revoke UDC:', error);
    }
  };

  const getStatusIcon = (status: UDC['status']) => {
    switch (status) {
      case 'active':
        return <CheckCircle className="w-4 h-4 text-green-400" />;
      case 'expired':
        return <XCircle className="w-4 h-4 text-red-400" />;
      case 'revoked':
        return <XCircle className="w-4 h-4 text-red-400" />;
      case 'suspended':
        return <AlertTriangle className="w-4 h-4 text-yellow-400" />;
      default:
        return <Clock className="w-4 h-4 text-yellow-400" />;
    }
  };

  const getAuthorityColor = (level: UDC['authorityLevel']) => {
    switch (level) {
      case 'full':
        return 'text-red-400 bg-red-500/20 border-red-500/20';
      case 'admin':
        return 'text-orange-400 bg-orange-500/20 border-orange-500/20';
      case 'execute':
        return 'text-yellow-400 bg-yellow-500/20 border-yellow-500/20';
      case 'write':
        return 'text-blue-400 bg-blue-500/20 border-blue-500/20';
      default:
        return 'text-gray-400 bg-gray-500/20 border-gray-500/20';
    }
  };

  const getExpiringUDCs = () => {
    return udcManagementService.getExpiringUDCs(7); // 7 days ahead
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black/50 backdrop-blur-sm z-50 flex items-center justify-center p-4">
      <div className="bg-gray-900/95 border border-gray-700/50 rounded-xl w-full max-w-6xl h-[90vh] overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-gray-700/50">
          <div className="flex items-center space-x-3">
            <div className="w-10 h-10 bg-emerald-500/20 rounded-lg flex items-center justify-center border border-emerald-500/20">
              <Shield className="w-5 h-5 text-emerald-400" />
            </div>
            <div>
              <h2 className="text-xl font-bold text-white">UDC Manager</h2>
              <p className="text-sm text-gray-400">Universal Delegation Certificates</p>
            </div>
          </div>
          <div className="flex items-center space-x-2">
            <button
              onClick={loadUDCs}
              disabled={isLoading}
              aria-label="Refresh UDCs"
              className="p-2 hover:bg-gray-700/50 rounded-lg text-gray-400 hover:text-white transition-all disabled:opacity-50"
            >
              <RefreshCw className={`w-4 h-4 ${isLoading ? 'animate-spin' : ''}`} />
            </button>
            <button
              onClick={onClose}
              aria-label="Close UDC Manager"
              className="p-2 hover:bg-gray-700/50 rounded-lg text-gray-400 hover:text-white transition-all"
            >
              ×
            </button>
          </div>
        </div>

        {/* Tabs */}
        <div className="flex border-b border-gray-700/50">
          {([
            { id: 'list' as const, label: 'All UDCs', icon: Shield },
            { id: 'create' as const, label: 'Create UDC', icon: Plus },
            { id: 'expiring' as const, label: 'Expiring Soon', icon: AlertTriangle }
          ] as const).map(tab => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`flex items-center space-x-2 px-6 py-3 text-sm font-medium transition-all ${
                activeTab === tab.id
                  ? 'text-emerald-400 border-b-2 border-emerald-400 bg-emerald-500/10'
                  : 'text-gray-400 hover:text-white hover:bg-gray-700/30'
              }`}
            >
              <tab.icon className="w-4 h-4" />
              <span>{tab.label}</span>
            </button>
          ))}
        </div>

        {/* Content */}
        <div className="p-6 overflow-y-auto h-full">
          {activeTab === 'list' && (
            <div className="space-y-4">
              {udcs.length === 0 ? (
                <div className="text-center py-12">
                  <Shield className="w-12 h-12 text-gray-400 mx-auto mb-4" />
                  <p className="text-gray-400">No UDCs found</p>
                  <button
                    onClick={() => setActiveTab('create')}
                    className="mt-4 px-4 py-2 bg-emerald-500/20 border border-emerald-500/20 text-emerald-400 rounded-lg hover:bg-emerald-500/30 transition-all"
                  >
                    Create First UDC
                  </button>
                </div>
              ) : (
                udcs.map(udc => {
                  const validation = validationResults[udc.id];
                  return (
                    <div key={udc.id} className="bg-gray-800/50 border border-gray-700/50 rounded-lg p-4">
                      <div className="flex items-center justify-between mb-3">
                        <div className="flex items-center space-x-3">
                          {getStatusIcon(udc.status)}
                          <div>
                            <h3 className="font-semibold text-white">UDC-{udc.id.slice(-8)}</h3>
                            <p className="text-sm text-gray-400">Agent: {udc.agentId}</p>
                          </div>
                        </div>
                        <div className="flex items-center space-x-2">
                          <span className={`px-2 py-1 text-xs rounded border ${getAuthorityColor(udc.authorityLevel)}`}>
                            {udc.authorityLevel}
                          </span>
                          <button
                            onClick={() => setSelectedUDC(udc)}
                            aria-label={`View details for UDC ${udc.id}`}
                            className="p-1 hover:bg-gray-700/50 rounded text-gray-400 hover:text-white transition-all"
                          >
                            <Eye className="w-4 h-4" />
                          </button>
                          {udc.status === 'active' && (
                            <button
                              onClick={() => handleRenewUDC(udc.id, 30)}
                              aria-label={`Renew UDC ${udc.id}`}
                              className="p-1 hover:bg-gray-700/50 rounded text-gray-400 hover:text-green-400 transition-all"
                            >
                              <RefreshCw className="w-4 h-4" />
                            </button>
                          )}
                          <button
                            onClick={() => handleRevokeUDC(udc.id, 'Manual revocation')}
                            aria-label={`Revoke UDC ${udc.id}`}
                            className="p-1 hover:bg-gray-700/50 rounded text-gray-400 hover:text-red-400 transition-all"
                          >
                            <Trash2 className="w-4 h-4" />
                          </button>
                        </div>
                      </div>
                      
                      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm mb-3">
                        <div>
                          <span className="text-gray-400">Type:</span>
                          <span className="ml-2 text-white">{udc.type}</span>
                        </div>
                        <div>
                          <span className="text-gray-400">Issued:</span>
                          <span className="ml-2 text-white">{udc.issuedDate.toLocaleDateString()}</span>
                        </div>
                        <div>
                          <span className="text-gray-400">Expires:</span>
                          <span className="ml-2 text-white">{udc.expiresDate.toLocaleDateString()}</span>
                        </div>
                        <div>
                          <span className="text-gray-400">Usage:</span>
                          <span className="ml-2 text-white">{udc.metadata.usage.executionCount}</span>
                        </div>
                      </div>

                      {validation && (
                        <div className="flex items-center space-x-4 text-sm">
                          <div className={`flex items-center space-x-1 ${validation.isValid ? 'text-green-400' : 'text-red-400'}`}>
                            {validation.isValid ? <CheckCircle className="w-3 h-3" /> : <XCircle className="w-3 h-3" />}
                            <span>{validation.isValid ? 'Valid' : 'Invalid'}</span>
                          </div>
                          {validation.remainingTime && validation.remainingTime > 0 && (
                            <div className="text-gray-400">
                              <Clock className="w-3 h-3 inline mr-1" />
                              {Math.ceil(validation.remainingTime / (24 * 60 * 60 * 1000))} days remaining
                            </div>
                          )}
                          {validation.usageQuota && (
                            <div className="text-gray-400">
                              Quota: {validation.usageQuota.remaining}/{validation.usageQuota.total}
                            </div>
                          )}
                        </div>
                      )}
                    </div>
                  );
                })
              )}
            </div>
          )}

          {activeTab === 'create' && (
            <UDCCreateForm onSubmit={handleCreateUDC} onCancel={() => setActiveTab('list')} />
          )}

          {activeTab === 'expiring' && (
            <div className="space-y-4">
              {getExpiringUDCs().map(udc => (
                <div key={udc.id} className="bg-yellow-500/10 border border-yellow-500/20 rounded-lg p-4">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center space-x-3">
                      <AlertTriangle className="w-5 h-5 text-yellow-400" />
                      <div>
                        <h3 className="font-semibold text-white">UDC-{udc.id.slice(-8)}</h3>
                        <p className="text-sm text-gray-400">
                          Expires: {udc.expiresDate.toLocaleDateString()} 
                          ({Math.ceil((udc.expiresDate.getTime() - Date.now()) / (24 * 60 * 60 * 1000))} days)
                        </p>
                      </div>
                    </div>
                    <button
                      onClick={() => handleRenewUDC(udc.id, 30)}
                      className="px-4 py-2 bg-yellow-500/20 border border-yellow-500/20 text-yellow-400 rounded-lg hover:bg-yellow-500/30 transition-all"
                    >
                      Renew 30 Days
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* UDC Details Modal */}
        {selectedUDC && (
          <UDCDetailsModal udc={selectedUDC} onClose={() => setSelectedUDC(null)} />
        )}
      </div>
    </div>
  );
};

interface UDCCreateFormProps {
  onSubmit: (request: UDCRequest) => void;
  onCancel: () => void;
}

const UDCCreateForm: React.FC<UDCCreateFormProps> = ({ onSubmit, onCancel }) => {
  const [formData, setFormData] = useState({
    agentId: '',
    type: 'basic' as UDC['type'],
    authorityLevel: 'read' as UDC['authorityLevel'],
    validityPeriod: 30,
    scope: '',
    permissions: ['read'],
    maxExecutions: 1000,
    allowedHours: Array.from({length: 24}, (_, i) => i),
    requiresMFA: false
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    
    const request: UDCRequest = {
      agentId: formData.agentId,
      type: formData.type,
      authorityLevel: formData.authorityLevel,
      validityPeriod: formData.validityPeriod,
      scope: formData.scope,
      permissions: formData.permissions,
      constraints: {
        maxExecutions: formData.maxExecutions,
        allowedHours: formData.allowedHours
      },
      metadata: {
        description: `UDC for agent ${formData.agentId}`,
        tags: [formData.type, formData.authorityLevel]
      }
    };

    onSubmit(request);
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-6 max-w-2xl">
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label htmlFor="agent-id" className="block text-sm font-medium text-gray-300 mb-2">Agent ID</label>
          <input
            id="agent-id"
            type="text"
            value={formData.agentId}
            onChange={(e) => setFormData({ ...formData, agentId: e.target.value })}
            className="w-full px-3 py-2 bg-gray-800 border border-gray-600 rounded-lg text-white focus:border-emerald-400 focus:outline-none"
            required
          />
        </div>

        <div>
          <label htmlFor="udc-type" className="block text-sm font-medium text-gray-300 mb-2">UDC Type</label>
          <select
            id="udc-type"
            value={formData.type}
            onChange={(e) => setFormData({ ...formData, type: e.target.value as UDC['type'] })}
            className="w-full px-3 py-2 bg-gray-800 border border-gray-600 rounded-lg text-white focus:border-emerald-400 focus:outline-none"
          >
            <option value="basic">Basic</option>
            <option value="advanced">Advanced</option>
            <option value="enterprise">Enterprise</option>
            <option value="custom">Custom</option>
          </select>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label htmlFor="authority-level" className="block text-sm font-medium text-gray-300 mb-2">Authority Level</label>
          <select
            id="authority-level"
            value={formData.authorityLevel}
            onChange={(e) => setFormData({ ...formData, authorityLevel: e.target.value as UDC['authorityLevel'] })}
            className="w-full px-3 py-2 bg-gray-800 border border-gray-600 rounded-lg text-white focus:border-emerald-400 focus:outline-none"
          >
            <option value="read">Read</option>
            <option value="write">Write</option>
            <option value="execute">Execute</option>
            <option value="admin">Admin</option>
            <option value="full">Full</option>
          </select>
        </div>

        <div>
          <label htmlFor="validity-period" className="block text-sm font-medium text-gray-300 mb-2">Validity Period (days)</label>
          <input
            id="validity-period"
            type="number"
            value={formData.validityPeriod}
            onChange={(e) => setFormData({ ...formData, validityPeriod: parseInt(e.target.value) })}
            className="w-full px-3 py-2 bg-gray-800 border border-gray-600 rounded-lg text-white focus:border-emerald-400 focus:outline-none"
            min="1"
            max="365"
          />
        </div>
      </div>

      <div>
        <label htmlFor="scope-description" className="block text-sm font-medium text-gray-300 mb-2">Scope Description</label>
        <textarea
          id="scope-description"
          value={formData.scope}
          onChange={(e) => setFormData({ ...formData, scope: e.target.value })}
          className="w-full px-3 py-2 bg-gray-800 border border-gray-600 rounded-lg text-white focus:border-emerald-400 focus:outline-none"
          rows={3}
          placeholder="Describe what this UDC allows the agent to do..."
        />
      </div>

      <div>
        <label htmlFor="max-executions" className="block text-sm font-medium text-gray-300 mb-2">Max Executions</label>
        <input
          id="max-executions"
          type="number"
          value={formData.maxExecutions}
          onChange={(e) => setFormData({ ...formData, maxExecutions: parseInt(e.target.value) })}
          className="w-full px-3 py-2 bg-gray-800 border border-gray-600 rounded-lg text-white focus:border-emerald-400 focus:outline-none"
          min="1"
        />
      </div>

      <div className="flex space-x-4">
        <button
          type="submit"
          data-testid="create-udc-submit-button"
          className="px-6 py-2 bg-emerald-500 text-white rounded-lg hover:bg-emerald-600 transition-all"
        >
          Create UDC
        </button>
        <button
          type="button"
          onClick={onCancel}
          className="px-6 py-2 bg-gray-600 text-white rounded-lg hover:bg-gray-700 transition-all"
        >
          Cancel
        </button>
      </div>
    </form>
  );
};

interface UDCDetailsModalProps {
  udc: UDC;
  onClose: () => void;
}

const UDCDetailsModal: React.FC<UDCDetailsModalProps> = ({ udc, onClose }) => {
  return (
    <div className="fixed inset-0 bg-black/50 backdrop-blur-sm z-60 flex items-center justify-center p-4">
      <div className="bg-gray-900 border border-gray-700 rounded-xl max-w-2xl w-full max-h-[80vh] overflow-y-auto">
        <div className="p-6">
          <div className="flex items-center justify-between mb-6">
            <h3 className="text-xl font-bold text-white">UDC Details</h3>
            <button
              onClick={onClose}
              className="p-2 hover:bg-gray-700/50 rounded-lg text-gray-400 hover:text-white transition-all"
            >
              ×
            </button>
          </div>

          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <span className="text-gray-400">ID:</span>
                <span className="ml-2 text-white font-mono">{udc.id}</span>
              </div>
              <div>
                <span className="text-gray-400">Agent ID:</span>
                <span className="ml-2 text-white">{udc.agentId}</span>
              </div>
              <div>
                <span className="text-gray-400">Type:</span>
                <span className="ml-2 text-white">{udc.type}</span>
              </div>
              <div>
                <span className="text-gray-400">Authority:</span>
                <span className="ml-2 text-white">{udc.authorityLevel}</span>
              </div>
              <div>
                <span className="text-gray-400">Status:</span>
                <span className="ml-2 text-white">{udc.status}</span>
              </div>
              <div>
                <span className="text-gray-400">Issued:</span>
                <span className="ml-2 text-white">{udc.issuedDate.toLocaleString()}</span>
              </div>
              <div>
                <span className="text-gray-400">Expires:</span>
                <span className="ml-2 text-white">{udc.expiresDate.toLocaleString()}</span>
              </div>
              <div>
                <span className="text-gray-400">Usage Count:</span>
                <span className="ml-2 text-white">{udc.metadata.usage.executionCount}</span>
              </div>
            </div>

            <div>
              <h4 className="text-lg font-semibold text-white mb-2">Scope</h4>
              <p className="text-gray-300 bg-gray-800/50 p-3 rounded-lg">{udc.scope}</p>
            </div>

            <div>
              <h4 className="text-lg font-semibold text-white mb-2">Permissions</h4>
              <div className="flex flex-wrap gap-2">
                {udc.permissions.map((permission, index) => (
                  <span key={index} className="px-2 py-1 bg-blue-500/20 border border-blue-500/20 text-blue-400 rounded text-sm">
                    {permission}
                  </span>
                ))}
              </div>
            </div>

            <div>
              <h4 className="text-lg font-semibold text-white mb-2">Signature</h4>
              <p className="text-gray-300 bg-gray-800/50 p-3 rounded-lg font-mono text-xs break-all">{udc.signature}</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default UDCManager;
