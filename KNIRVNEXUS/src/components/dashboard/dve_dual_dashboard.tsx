import React, { useState } from 'react';
import { Play, Upload, Shield, Activity, FileCode, Network, AlertTriangle, CheckCircle, XCircle, Search, Filter, Terminal, Eye, Lock, Settings, Brain } from 'lucide-react';

interface NRV {
  id: string;
  title: string;
  domain: string;
  severity: 'high' | 'medium' | 'low';
  timestamp: string;
}

interface ValidationResult {
  staticAnalysis: { pass: boolean; message: string };
  dynamicAnalysis: { pass: boolean; message: string };
  forensics: { pass: boolean; message: string };
  resolution: { pass: boolean; message: string };
}

interface CleanNode {
  id: string;
  status: 'active' | 'idle' | 'error';
  cpu: number;
  gpu: number;
  resolvedToday: number;
}

interface ResolutionLog {
  timestamp: string;
  failureId: string;
  nodeId: string;
  strategy: string;
  validation: 'PASS' | 'FAIL';
  status: string;
}

interface DVEDualDashboardProps {
  nodeId?: string;
  nodeName?: string;
}

const DVEDualDashboard: React.FC<DVEDualDashboardProps> = ({ nodeId, nodeName }) => {
  const [activeTab, setActiveTab] = useState<'solver' | 'enterprise'>('solver');
  const [selectedNRV, setSelectedNRV] = useState<NRV | null>(null);
  const [isValidating, setIsValidating] = useState(false);
  const [validationResult, setValidationResult] = useState<ValidationResult | null>(null);
  const [logStream, setLogStream] = useState<string[]>([]);
  const [nrnBalance] = useState(1500);
  const [requiredBond] = useState(100);
  const [showNRVPanel, setShowNRVPanel] = useState(false);
  const [showMonitorPanel, setShowMonitorPanel] = useState(false);
  const [showCVEPanel, setShowCVEPanel] = useState(false);
  const [monitorHeight, setMonitorHeight] = useState(33);

  // Mock data
  const mockNRVs: NRV[] = [
    { id: 'NRV-001', title: 'Hallucination in Medical Diagnosis', domain: 'Healthcare', severity: 'high', timestamp: '2025-10-21T10:30:00Z' },
    { id: 'NRV-002', title: 'Code Generation Security Flaw', domain: 'Security', severity: 'high', timestamp: '2025-10-21T09:15:00Z' },
    { id: 'NRV-003', title: 'Math Calculation Error', domain: 'Finance', severity: 'medium', timestamp: '2025-10-21T08:45:00Z' },
  ];

  const mockCleanNodes: CleanNode[] = [
    { id: 'CLEAN-01', status: 'active', cpu: 45, gpu: 67, resolvedToday: 23 },
    { id: 'CLEAN-02', status: 'active', cpu: 32, gpu: 54, resolvedToday: 18 },
    { id: 'CLEAN-03', status: 'idle', cpu: 12, gpu: 15, resolvedToday: 31 },
    { id: 'CLEAN-04', status: 'error', cpu: 89, gpu: 92, resolvedToday: 12 },
  ];

  const mockResolutionLogs: ResolutionLog[] = [
    { timestamp: '10:45:23', failureId: 'FAIL-8472', nodeId: 'CLEAN-01', strategy: 'Constraint-Based', validation: 'PASS', status: 'Response Sent' },
    { timestamp: '10:44:56', failureId: 'FAIL-8471', nodeId: 'CLEAN-02', strategy: 'Forensic-Block', validation: 'FAIL', status: 'Blocked' },
    { timestamp: '10:44:12', failureId: 'FAIL-8470', nodeId: 'CLEAN-01', strategy: 'Self-Correction', validation: 'PASS', status: 'Response Sent' },
  ];

  const handleLoadProblem = (nrv: NRV) => {
    setSelectedNRV(nrv);
    setValidationResult(null);
    setLogStream([]);
  };

  const handleRunValidation = () => {
    setIsValidating(true);
    setLogStream([]);
    
    // Simulate validation process
    const logs = [
      '[strace] Intercepting system calls...',
      '[ltrace] Monitoring library calls...',
      '[tshark] Capturing network traffic...',
      '[sandbox] Executing skill in isolated environment...',
      '[ghidra] Running static analysis...',
      '[semgrep] Scanning for vulnerabilities...',
      '[volatility] Analyzing memory snapshot...',
      '[validation] Checking resolution against failure context...',
    ];

    logs.forEach((log, i) => {
      setTimeout(() => {
        setLogStream(prev => [...prev, log]);
      }, i * 500);
    });

    setTimeout(() => {
      setValidationResult({
        staticAnalysis: { pass: true, message: 'Ghidra scan: OK. Semgrep: 0 issues found.' },
        dynamicAnalysis: { pass: true, message: 'Syscall violations: 0. Unauthorized network calls: 0.' },
        forensics: { pass: true, message: 'Memory snapshot (Volatility): OK. Filesystem integrity: OK.' },
        resolution: { pass: true, message: 'Result: SUCCESS. Skill resolved the FailureContext.' },
      });
      setIsValidating(false);
    }, 4000);
  };

  const handleSubmit = () => {
    alert('Skill submitted to KNIRVGRAPH consensus! Transaction hash: 0x7a8b9c...');
  };

  return (
    <div className="min-h-screen bg-gray-900 text-gray-100 p-6">
      {/* Tab Switcher */}
      <div className="flex gap-2 mb-6">
        <button
          onClick={() => setActiveTab('solver')}
          className={`px-6 py-3 rounded-lg font-semibold transition-all ${
            activeTab === 'solver'
              ? 'bg-blue-600 text-white shadow-lg'
              : 'bg-gray-800 text-gray-400 hover:bg-gray-700'
          }`}
        >
          <FileCode className="inline-block w-5 h-5 mr-2" />
          DVE Solver Interface
        </button>
        <button
          onClick={() => setActiveTab('enterprise')}
          className={`px-6 py-3 rounded-lg font-semibold transition-all ${
            activeTab === 'enterprise'
              ? 'bg-purple-600 text-white shadow-lg'
              : 'bg-gray-800 text-gray-400 hover:bg-gray-700'
          }`}
        >
          <Shield className="inline-block w-5 h-5 mr-2" />
          Enterprise RTR Dashboard
        </button>
      </div>

      {/* Solver Interface */}
      {activeTab === 'solver' && (
        <div className="grid grid-cols-3 gap-6">
          {/* Problem Panel */}
          <div className="col-span-1 bg-gray-800 rounded-lg p-6 border border-gray-700">
            <h2 className="text-xl font-bold mb-4 flex items-center">
              <Search className="w-5 h-5 mr-2" />
              Problem Panel
            </h2>
            
            <div className="mb-4">
              <input
                type="text"
                placeholder="Search NRVs..."
                className="w-full bg-gray-900 border border-gray-600 rounded px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
              />
            </div>

            <div className="space-y-2 max-h-96 overflow-y-auto">
              {mockNRVs.map(nrv => (
                <div
                  key={nrv.id}
                  onClick={() => handleLoadProblem(nrv)}
                  className={`p-3 rounded cursor-pointer transition-all ${
                    selectedNRV?.id === nrv.id
                      ? 'bg-blue-900 border border-blue-500'
                      : 'bg-gray-900 hover:bg-gray-700 border border-gray-600'
                  }`}
                >
                  <div className="flex items-center justify-between mb-1">
                    <span className="font-mono text-xs text-blue-400">{nrv.id}</span>
                    <span className={`text-xs px-2 py-1 rounded ${
                      nrv.severity === 'high' ? 'bg-red-900 text-red-200' :
                      nrv.severity === 'medium' ? 'bg-yellow-900 text-yellow-200' :
                      'bg-green-900 text-green-200'
                    }`}>
                      {nrv.severity}
                    </span>
                  </div>
                  <div className="text-sm font-semibold mb-1">{nrv.title}</div>
                  <div className="text-xs text-gray-400">{nrv.domain}</div>
                </div>
              ))}
            </div>

            {selectedNRV && (
              <div className="mt-4 p-3 bg-gray-900 rounded border border-gray-600">
                <div className="text-xs text-gray-400 mb-2">Failure Context</div>
                <div className="text-sm">
                  <div className="mb-2"><strong>Prompt:</strong> "Diagnose patient symptoms..."</div>
                  <div className="mb-2"><strong>Bad Response:</strong> "Patient has cancer..."</div>
                  <div className="text-red-400"><strong>Issue:</strong> Hallucination without evidence</div>
                </div>
              </div>
            )}
          </div>

          {/* Validation Panel */}
          <div className="col-span-2 space-y-6">
            <div className="bg-gray-800 rounded-lg p-6 border border-gray-700">
              <h2 className="text-xl font-bold mb-4 flex items-center">
                <Terminal className="w-5 h-5 mr-2" />
                Validation Panel
              </h2>

              <button
                onClick={handleRunValidation}
                disabled={!selectedNRV || isValidating}
                className="w-full bg-green-600 hover:bg-green-700 disabled:bg-gray-600 disabled:cursor-not-allowed text-white font-semibold py-3 rounded-lg mb-4 transition-all flex items-center justify-center"
              >
                <Play className="w-5 h-5 mr-2" />
                {isValidating ? 'Running Validation...' : 'Run Validation'}
              </button>

              {/* Log Stream */}
              <div className="bg-gray-900 rounded border border-gray-600 p-4 h-48 overflow-y-auto mb-4 font-mono text-xs">
                {logStream.length === 0 ? (
                  <div className="text-gray-500">Ready to validate. Click "Run Validation" to begin...</div>
                ) : (
                  logStream.map((log, i) => (
                    <div key={i} className="text-green-400 mb-1">{log}</div>
                  ))
                )}
              </div>

              {/* Validation Report Card */}
              {validationResult && (
                <div className="space-y-3">
                  <h3 className="font-bold text-lg mb-3">Validation Report Card</h3>
                  
                  {Object.entries(validationResult).map(([key, result]) => (
                    <div key={key} className="flex items-start p-3 bg-gray-900 rounded border border-gray-600">
                      {result.pass ? (
                        <CheckCircle className="w-5 h-5 text-green-400 mr-3 mt-0.5 flex-shrink-0" />
                      ) : (
                        <XCircle className="w-5 h-5 text-red-400 mr-3 mt-0.5 flex-shrink-0" />
                      )}
                      <div>
                        <div className="font-semibold capitalize mb-1">
                          {key.replace(/([A-Z])/g, ' $1').trim()}
                        </div>
                        <div className="text-sm text-gray-400">{result.message}</div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>

            {/* Submission Panel */}
            <div className="bg-gray-800 rounded-lg p-6 border border-gray-700">
              <h2 className="text-xl font-bold mb-4 flex items-center">
                <Upload className="w-5 h-5 mr-2" />
                Submission Panel
              </h2>

              <div className="grid grid-cols-2 gap-4 mb-4">
                <div className="bg-gray-900 rounded p-4 border border-gray-600">
                  <div className="text-sm text-gray-400 mb-1">NRN Balance</div>
                  <div className="text-2xl font-bold text-blue-400">{nrnBalance}</div>
                </div>
                <div className="bg-gray-900 rounded p-4 border border-gray-600">
                  <div className="text-sm text-gray-400 mb-1">Required Bond</div>
                  <div className="text-2xl font-bold text-yellow-400">{requiredBond}</div>
                </div>
              </div>

              <button
                onClick={handleSubmit}
                disabled={!validationResult || !validationResult.resolution.pass}
                className="w-full bg-purple-600 hover:bg-purple-700 disabled:bg-gray-600 disabled:cursor-not-allowed text-white font-semibold py-3 rounded-lg transition-all flex items-center justify-center"
              >
                <Lock className="w-5 h-5 mr-2" />
                Submit to Consensus
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Enterprise Dashboard */}
      {activeTab === 'enterprise' && (
        <div className="relative w-full h-screen bg-gray-900" style={{ paddingTop: '1.5rem' }}>
          {/* Main Content Area */}
          <div 
            className="w-full transition-all duration-300 p-4"
            style={{
              marginLeft: showNRVPanel ? '320px' : '0',
              marginRight: showCVEPanel ? '320px' : '0',
              marginBottom: showMonitorPanel ? `${monitorHeight + 2}%` : '0',
              height: showMonitorPanel ? `calc(100% - ${monitorHeight}vh)` : '100%',
              overflowY: 'auto'
            }}
          >
            {/* Content Grid */}
            <div className="space-y-6 pb-6">
              {/* Real-Time Failure Feed */}
              <div className="bg-gray-800 rounded-lg p-6 border border-gray-700 shadow-md">
                <h2 className="text-lg font-bold mb-4 flex items-center">
                  <AlertTriangle className="w-5 h-5 mr-2 text-red-400" />
                  Real-Time Failure Feed
                </h2>
                
                <div className="bg-gray-900 rounded border border-gray-600 p-4 h-40 overflow-y-auto font-mono text-xs space-y-1">
                  <div className="text-red-400">[10:45:23] FAILURE DETECTED - Shell ID: cortex-8472 - Type: Unauthorized API Call</div>
                  <div className="text-yellow-400">[10:44:56] FAILURE DETECTED - Shell ID: cortex-8471 - Type: Hallucination Risk</div>
                  <div className="text-red-400">[10:44:12] FAILURE DETECTED - Shell ID: cortex-8470 - Type: Data Leak Attempt</div>
                  <div className="text-yellow-400">[10:43:45] FAILURE DETECTED - Shell ID: cortex-8469 - Type: Logic Error</div>
                  <div className="text-green-400">[10:43:20] RESOLVED - Shell ID: cortex-8468 - Strategy: Self-Correction</div>
                </div>
              </div>

              {/* Security Policy Editor */}
              <div className="bg-gray-800 rounded-lg p-6 border border-gray-700 shadow-md">
                <h2 className="text-lg font-bold mb-4 flex items-center">
                  <Settings className="w-5 h-5 mr-2" />
                  Security Policy Editor
                </h2>
                
                <div className="grid grid-cols-2 gap-6">
                  <div>
                    <label className="block text-sm font-semibold mb-2">Network Whitelist</label>
                    <textarea
                      className="w-full bg-gray-900 border border-gray-600 rounded p-3 text-sm font-mono focus:outline-none focus:border-blue-500"
                      rows={4}
                      placeholder="api.company.com&#10;cdn.trusted.io&#10;auth.internal.net"
                      defaultValue="api.company.com&#10;cdn.trusted.io"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-semibold mb-2">Validation Sensitivity</label>
                    <select className="w-full bg-gray-900 border border-gray-600 rounded p-3 text-sm focus:outline-none focus:border-blue-500 mb-3">
                      <option>Balanced (Default)</option>
                      <option>Paranoid (Strict)</option>
                      <option>Permissive (Lenient)</option>
                    </select>
                    
                    <label className="flex items-center mb-2">
                      <input type="checkbox" className="mr-2" defaultChecked />
                      <span className="text-sm">Block all file I/O operations</span>
                    </label>
                    <label className="flex items-center mb-2">
                      <input type="checkbox" className="mr-2" />
                      <span className="text-sm">Allow read-only filesystem access</span>
                    </label>
                    <label className="flex items-center">
                      <input type="checkbox" className="mr-2" defaultChecked />
                      <span className="text-sm">Enable memory forensics</span>
                    </label>
                  </div>
                </div>
                
                <button className="mt-4 bg-blue-600 hover:bg-blue-700 text-white font-semibold py-2 px-6 rounded-lg transition-all">
                  Save Policy Changes
                </button>
              </div>
            </div>
          </div>

          {/* Top-Left: DVE Listing Card */}
          <div className="fixed top-6 left-6 z-40 transition-all duration-300" style={{ left: showNRVPanel ? 'calc(320px + 1.5rem)' : '1.5rem' }}>
            <div className="bg-gray-800 rounded-lg p-4 border border-gray-700 shadow-lg" style={{ width: '280px' }}>
              <h3 className="text-lg font-bold mb-3 flex items-center">
                <Activity className="w-4 h-4 mr-2 text-green-400" />
                CLEAN Nodes
              </h3>
              <div className="space-y-2">
                {mockCleanNodes.map(node => (
                  <div key={node.id} className="bg-gray-900 rounded p-2 border border-gray-600 text-xs">
                    <div className="flex items-center justify-between mb-1">
                      <span className="font-mono font-bold text-blue-400">{node.id}</span>
                      <span className={`w-2 h-2 rounded-full ${
                        node.status === 'active' ? 'bg-green-400' :
                        node.status === 'idle' ? 'bg-yellow-400' :
                        'bg-red-400'
                      }`} />
                    </div>
                    <div className="text-gray-400">CPU: {node.cpu}% | GPU: {node.gpu}%</div>
                  </div>
                ))}
              </div>
            </div>
          </div>

          {/* Left Panel: NRV List (Slides from left) */}
          <div 
            className="fixed left-0 top-0 h-full bg-gray-800 border-r border-gray-700 shadow-lg overflow-hidden transition-all duration-300 z-30"
            style={{ 
              width: showNRVPanel ? '320px' : '0',
              paddingTop: '1.5rem'
            }}
          >
            <div className="h-full flex flex-col p-4 overflow-y-auto">
              <button
                onClick={() => setShowNRVPanel(!showNRVPanel)}
                className="absolute right-2 top-2 bg-gray-700 hover:bg-gray-600 p-1 rounded"
              >
                <XCircle className="w-5 h-5" />
              </button>
              
              <h2 className="text-lg font-bold mb-4 flex items-center">
                <Search className="w-5 h-5 mr-2" />
                NRV List
              </h2>
              
              <input
                type="text"
                placeholder="Search NRVs..."
                className="w-full bg-gray-900 border border-gray-600 rounded px-3 py-2 text-sm mb-4 focus:outline-none focus:border-blue-500"
              />

              <div className="space-y-2 overflow-y-auto">
                {mockNRVs.map(nrv => (
                  <div
                    key={nrv.id}
                    onClick={() => handleLoadProblem(nrv)}
                    className={`p-3 rounded cursor-pointer transition-all text-xs ${
                      selectedNRV?.id === nrv.id
                        ? 'bg-blue-900 border border-blue-500'
                        : 'bg-gray-900 hover:bg-gray-700 border border-gray-600'
                    }`}
                  >
                    <div className="flex items-center justify-between mb-1">
                      <span className="font-mono text-blue-400">{nrv.id}</span>
                      <span className={`text-xs px-2 py-1 rounded ${
                        nrv.severity === 'high' ? 'bg-red-900 text-red-200' :
                        nrv.severity === 'medium' ? 'bg-yellow-900 text-yellow-200' :
                        'bg-green-900 text-green-200'
                      }`}>
                        {nrv.severity}
                      </span>
                    </div>
                    <div className="font-semibold mb-1 line-clamp-2">{nrv.title}</div>
                    <div className="text-gray-400">{nrv.domain}</div>
                  </div>
                ))}
              </div>

              {selectedNRV && (
                <div className="mt-4 p-3 bg-gray-900 rounded border border-gray-600 text-xs">
                  <div className="text-gray-400 mb-2 font-semibold">Failure Context</div>
                  <div className="space-y-1 text-xs">
                    <div><strong>Prompt:</strong> "Diagnose..."</div>
                    <div><strong>Response:</strong> "Cancer..."</div>
                    <div className="text-red-400"><strong>Issue:</strong> Hallucination</div>
                  </div>
                </div>
              )}
            </div>
          </div>

          {/* Bottom Panel: Monitor Dashboard (Slides up from bottom) */}
          <div 
            className="fixed bottom-0 left-0 right-0 bg-gray-800 border-t border-gray-700 shadow-lg overflow-hidden transition-all duration-300 z-20"
            style={{ 
              height: showMonitorPanel ? `${monitorHeight}vh` : '0'
            }}
          >
            <div className="h-full flex flex-col bg-gray-800">
              {/* Drag Handle */}
              <div
                className="h-1 bg-gray-600 hover:bg-gray-500 cursor-ns-resize transition-colors"
                onMouseDown={(e) => {
                  const startY = e.clientY;
                  const startHeight = monitorHeight;
                  
                  const handleMouseMove = (moveEvent: MouseEvent) => {
                    const deltaY = moveEvent.clientY - startY;
                    const newHeight = startHeight - (deltaY / window.innerHeight) * 100;
                    setMonitorHeight(Math.max(20, Math.min(50, newHeight)));
                  };
                  
                  const handleMouseUp = () => {
                    document.removeEventListener('mousemove', handleMouseMove);
                    document.removeEventListener('mouseup', handleMouseUp);
                  };
                  
                  document.addEventListener('mousemove', handleMouseMove);
                  document.addEventListener('mouseup', handleMouseUp);
                }}
              />

              <div className="p-4 overflow-y-auto flex-1">
                <div className="flex items-center justify-between mb-4">
                  <h2 className="text-lg font-bold flex items-center">
                    <Eye className="w-5 h-5 mr-2" />
                    Live Resolution Monitor
                  </h2>
                  <button
                    onClick={() => setShowMonitorPanel(!showMonitorPanel)}
                    className="bg-gray-700 hover:bg-gray-600 p-1 rounded"
                  >
                    <XCircle className="w-5 h-5" />
                  </button>
                </div>
                
                <div className="overflow-x-auto">
                  <table className="w-full text-xs">
                    <thead className="bg-gray-900 border-b border-gray-600 sticky top-0">
                      <tr>
                        <th className="text-left p-2">Timestamp</th>
                        <th className="text-left p-2">Failure ID</th>
                        <th className="text-left p-2">Node</th>
                        <th className="text-left p-2">Strategy</th>
                        <th className="text-left p-2">Validation</th>
                        <th className="text-left p-2">Status</th>
                      </tr>
                    </thead>
                    <tbody>
                      {mockResolutionLogs.map((log, i) => (
                        <tr key={i} className="border-b border-gray-700 hover:bg-gray-900 text-xs">
                          <td className="p-2 font-mono">{log.timestamp}</td>
                          <td className="p-2 font-mono text-blue-400">{log.failureId}</td>
                          <td className="p-2 font-mono">{log.nodeId}</td>
                          <td className="p-2">{log.strategy}</td>
                          <td className="p-2">
                            <span className={`px-2 py-1 rounded text-xs ${
                              log.validation === 'PASS' ? 'bg-green-900 text-green-200' : 'bg-red-900 text-red-200'
                            }`}>
                              {log.validation}
                            </span>
                          </td>
                          <td className="p-2">{log.status}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            </div>
          </div>

          {/* Right Panel: CVE Panel (Slides from right - placeholder for future) */}
          <div 
            className="fixed right-0 top-0 h-full bg-gray-800 border-l border-gray-700 shadow-lg overflow-hidden transition-all duration-300 z-30"
            style={{ 
              width: showCVEPanel ? '320px' : '0',
              paddingTop: '1.5rem'
            }}
          >
            <div className="h-full flex flex-col p-4 overflow-y-auto">
              <button
                onClick={() => setShowCVEPanel(!showCVEPanel)}
                className="absolute left-2 top-2 bg-gray-700 hover:bg-gray-600 p-1 rounded"
              >
                <XCircle className="w-5 h-5" />
              </button>
              
              <h2 className="text-lg font-bold mb-4 flex items-center">
                <Brain className="w-5 h-5 mr-2" />
                Cognitive Engine
              </h2>
              
              <div className="bg-gray-900 rounded border border-gray-600 p-4">
                <p className="text-sm text-gray-400">CVE Panel - Ready for integration</p>
              </div>
            </div>
          </div>

          {/* Toggle Buttons */}
          <div className="fixed bottom-6 right-6 z-50 flex gap-3">
            <button
              onClick={() => setShowNRVPanel(!showNRVPanel)}
              className={`px-4 py-2 rounded-lg font-semibold transition-all flex items-center gap-2 ${
                showNRVPanel
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-700 hover:bg-gray-600 text-gray-300'
              }`}
              title="Toggle NRV Panel"
            >
              <Search className="w-4 h-4" />
              NRVs
            </button>
            
            <button
              onClick={() => setShowMonitorPanel(!showMonitorPanel)}
              className={`px-4 py-2 rounded-lg font-semibold transition-all flex items-center gap-2 ${
                showMonitorPanel
                  ? 'bg-purple-600 text-white'
                  : 'bg-gray-700 hover:bg-gray-600 text-gray-300'
              }`}
              title="Toggle Monitor Panel"
            >
              <Eye className="w-4 h-4" />
              Monitor
            </button>
            
            <button
              onClick={() => setShowCVEPanel(!showCVEPanel)}
              className={`px-4 py-2 rounded-lg font-semibold transition-all flex items-center gap-2 ${
                showCVEPanel
                  ? 'bg-indigo-600 text-white'
                  : 'bg-gray-700 hover:bg-gray-600 text-gray-300'
              }`}
              title="Toggle CVE Panel"
            >
              <Brain className="w-4 h-4" />
              CVE
            </button>
          </div>
        </div>
      )}
    </div>
  );
};

export default DVEDualDashboard;