import React, { useState, useEffect } from 'react';
import {
  Shield,
  CheckCircle,
  AlertTriangle,
  Clock,
  Download,
  Filter,
  Search,
  RefreshCw,
  Lock,
  Key,
  FileText,
  Eye,
  ExternalLink
} from 'lucide-react';

const TEEAttestationLogs = () => {
  const [attestations, setAttestations] = useState([]);
  const [securityLogs, setSecurityLogs] = useState([]);
  const [loading, setLoading] = useState(true);
  const [selectedAttestation, setSelectedAttestation] = useState(null);
  const [filterStatus, setFilterStatus] = useState('all');
  const [searchTerm, setSearchTerm] = useState('');

  // Mock data for TEE attestations
  useEffect(() => {
    const mockAttestations = [
      {
        id: 'att-001',
        taskId: 'task-7804',
        teeType: 'Intel SGX',
        status: 'verified',
        timestamp: new Date(Date.now() - 2 * 60 * 1000),
        measurementHash: '0x4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b',
        attestationReport: 'SGX_ATTESTATION_REPORT_V2',
        verificationStatus: 'PASSED',
        securityLevel: 'HIGH'
      },
      {
        id: 'att-002',
        taskId: 'task-3749',
        teeType: 'AMD SEV-SNP',
        status: 'verified',
        timestamp: new Date(Date.now() - 5 * 60 * 1000),
        measurementHash: '0x1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2b',
        attestationReport: 'SEV_SNP_ATTESTATION_REPORT',
        verificationStatus: 'PASSED',
        securityLevel: 'HIGH'
      },
      {
        id: 'att-003',
        taskId: 'task-182',
        teeType: 'Intel TDX',
        status: 'pending',
        timestamp: new Date(Date.now() - 10 * 60 * 1000),
        measurementHash: '0x9f8e7d6c5b4a3928374658291038475647382910384756473829103847564738',
        attestationReport: 'TDX_ATTESTATION_REPORT',
        verificationStatus: 'IN_PROGRESS',
        securityLevel: 'MEDIUM'
      }
    ];

    const mockSecurityLogs = [
      {
        id: 'log-001',
        type: 'ATTESTATION_SUCCESS',
        message: 'TEE attestation verified successfully for task-7804',
        timestamp: new Date(Date.now() - 2 * 60 * 1000),
        severity: 'INFO',
        taskId: 'task-7804'
      },
      {
        id: 'log-002',
        type: 'SECURITY_ALERT',
        message: 'Anomalous memory access pattern detected in TEE',
        timestamp: new Date(Date.now() - 15 * 60 * 1000),
        severity: 'WARNING',
        taskId: 'task-456'
      },
      {
        id: 'log-003',
        type: 'ENCRYPTION_KEY_ROTATION',
        message: 'TEE encryption keys rotated successfully',
        timestamp: new Date(Date.now() - 30 * 60 * 1000),
        severity: 'INFO',
        taskId: null
      }
    ];

    setAttestations(mockAttestations);
    setSecurityLogs(mockSecurityLogs);
    setLoading(false);
  }, []);

  const getStatusIcon = (status) => {
    switch (status) {
      case 'verified':
        return <CheckCircle className="w-5 h-5 text-green-400" />;
      case 'pending':
        return <Clock className="w-5 h-5 text-yellow-400" />;
      case 'failed':
        return <AlertTriangle className="w-5 h-5 text-red-400" />;
      default:
        return <Clock className="w-5 h-5 text-gray-400" />;
    }
  };

  const getSeverityColor = (severity) => {
    switch (severity) {
      case 'INFO':
        return 'text-blue-400';
      case 'WARNING':
        return 'text-yellow-400';
      case 'ERROR':
        return 'text-red-400';
      default:
        return 'text-gray-400';
    }
  };

  const formatTimestamp = (timestamp) => {
    return timestamp.toLocaleString();
  };

  const filteredAttestations = attestations.filter(att => {
    const matchesFilter = filterStatus === 'all' || att.status === filterStatus;
    const matchesSearch = att.taskId.toLowerCase().includes(searchTerm.toLowerCase()) ||
                         att.teeType.toLowerCase().includes(searchTerm.toLowerCase());
    return matchesFilter && matchesSearch;
  });

  return (
    <div className="min-h-screen bg-knirv-gradient p-6">
      <div className="max-w-7xl mx-auto">
        {/* Header */}
        <div className="flex items-center justify-between mb-8">
          <div className="flex items-center space-x-3">
            <Shield className="w-8 h-8 text-knirv-primary" />
            <div>
              <h1 className="text-3xl font-bold text-white">TEE Attestation & Security Logs</h1>
              <p className="text-slate-300">Transparent security attestations and comprehensive audit logs</p>
            </div>
          </div>
          <button
            onClick={() => setLoading(true)}
            className="flex items-center space-x-2 px-4 py-2 bg-knirv-primary text-white rounded-lg hover:bg-knirv-secondary transition-colors"
          >
            <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
            <span>Refresh</span>
          </button>
        </div>

        {/* Filters */}
        <div className="bg-slate-800 rounded-lg p-4 mb-6 border border-slate-700">
          <div className="flex items-center space-x-4">
            <div className="flex items-center space-x-2">
              <Filter className="w-4 h-4 text-slate-400" />
              <select
                value={filterStatus}
                onChange={(e) => setFilterStatus(e.target.value)}
                className="bg-slate-700 text-white rounded-lg px-3 py-2 border border-slate-600"
              >
                <option value="all">All Status</option>
                <option value="verified">Verified</option>
                <option value="pending">Pending</option>
                <option value="failed">Failed</option>
              </select>
            </div>
            <div className="flex items-center space-x-2 flex-1">
              <Search className="w-4 h-4 text-slate-400" />
              <input
                type="text"
                placeholder="Search by task ID or TEE type..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="bg-slate-700 text-white rounded-lg px-3 py-2 border border-slate-600 flex-1"
              />
            </div>
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* TEE Attestations */}
          <div className="bg-slate-800 rounded-lg border border-slate-700">
            <div className="p-6 border-b border-slate-700">
              <div className="flex items-center space-x-3">
                <Lock className="w-6 h-6 text-knirv-primary" />
                <h2 className="text-xl font-semibold text-white">TEE Attestations</h2>
              </div>
            </div>
            <div className="p-6">
              {loading ? (
                <div className="text-center py-8">
                  <RefreshCw className="w-6 h-6 text-knirv-primary animate-spin mx-auto mb-2" />
                  <p className="text-slate-400">Loading attestations...</p>
                </div>
              ) : (
                <div className="space-y-4">
                  {filteredAttestations.map((attestation) => (
                    <div
                      key={attestation.id}
                      className="bg-slate-700 rounded-lg p-4 border border-slate-600 hover:border-knirv-primary transition-colors cursor-pointer"
                      onClick={() => setSelectedAttestation(attestation)}
                    >
                      <div className="flex items-center justify-between mb-2">
                        <div className="flex items-center space-x-2">
                          {getStatusIcon(attestation.status)}
                          <span className="font-medium text-white">{attestation.taskId}</span>
                        </div>
                        <span className="text-xs text-slate-400">{formatTimestamp(attestation.timestamp)}</span>
                      </div>
                      <div className="text-sm text-slate-300">
                        <p>TEE Type: {attestation.teeType}</p>
                        <p>Security Level: {attestation.securityLevel}</p>
                        <p className="truncate">Hash: {attestation.measurementHash}</p>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>

          {/* Security Logs */}
          <div className="bg-slate-800 rounded-lg border border-slate-700">
            <div className="p-6 border-b border-slate-700">
              <div className="flex items-center space-x-3">
                <FileText className="w-6 h-6 text-knirv-primary" />
                <h2 className="text-xl font-semibold text-white">Security Event Logs</h2>
              </div>
            </div>
            <div className="p-6">
              <div className="space-y-4 max-h-96 overflow-y-auto">
                {securityLogs.map((log) => (
                  <div
                    key={log.id}
                    className="bg-slate-700 rounded-lg p-4 border border-slate-600"
                  >
                    <div className="flex items-center justify-between mb-2">
                      <span className={`text-sm font-medium ${getSeverityColor(log.severity)}`}>
                        {log.type}
                      </span>
                      <span className="text-xs text-slate-400">{formatTimestamp(log.timestamp)}</span>
                    </div>
                    <p className="text-sm text-slate-300">{log.message}</p>
                    {log.taskId && (
                      <p className="text-xs text-slate-400 mt-1">Task: {log.taskId}</p>
                    )}
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>

        {/* Attestation Detail Modal */}
        {selectedAttestation && (
          <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
            <div className="bg-slate-800 rounded-lg p-6 max-w-2xl w-full mx-4 border border-slate-700">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-xl font-semibold text-white">Attestation Details</h3>
                <button
                  onClick={() => setSelectedAttestation(null)}
                  className="text-slate-400 hover:text-white"
                >
                  ×
                </button>
              </div>
              <div className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="text-sm text-slate-400">Task ID</label>
                    <p className="text-white">{selectedAttestation.taskId}</p>
                  </div>
                  <div>
                    <label className="text-sm text-slate-400">TEE Type</label>
                    <p className="text-white">{selectedAttestation.teeType}</p>
                  </div>
                  <div>
                    <label className="text-sm text-slate-400">Status</label>
                    <div className="flex items-center space-x-2">
                      {getStatusIcon(selectedAttestation.status)}
                      <span className="text-white capitalize">{selectedAttestation.status}</span>
                    </div>
                  </div>
                  <div>
                    <label className="text-sm text-slate-400">Security Level</label>
                    <p className="text-white">{selectedAttestation.securityLevel}</p>
                  </div>
                </div>
                <div>
                  <label className="text-sm text-slate-400">Measurement Hash</label>
                  <p className="text-white font-mono text-sm break-all">{selectedAttestation.measurementHash}</p>
                </div>
                <div>
                  <label className="text-sm text-slate-400">Attestation Report Type</label>
                  <p className="text-white">{selectedAttestation.attestationReport}</p>
                </div>
                <div className="flex space-x-3">
                  <button className="flex items-center space-x-2 px-4 py-2 bg-knirv-primary text-white rounded-lg hover:bg-knirv-secondary transition-colors">
                    <Download className="w-4 h-4" />
                    <span>Download Report</span>
                  </button>
                  <button className="flex items-center space-x-2 px-4 py-2 bg-slate-700 text-white rounded-lg hover:bg-slate-600 transition-colors">
                    <ExternalLink className="w-4 h-4" />
                    <span>Verify Externally</span>
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default TEEAttestationLogs;
