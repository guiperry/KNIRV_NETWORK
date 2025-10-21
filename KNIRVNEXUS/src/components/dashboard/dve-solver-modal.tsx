'use client';

import React, { useState } from 'react';
import { Play, Upload, Lock, Terminal, Search, FileCode, CheckCircle, XCircle, X } from 'lucide-react';

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

interface DVESolverModalProps {
  isOpen: boolean;
  onClose: () => void;
}

const DVESolverModal: React.FC<DVESolverModalProps> = ({ isOpen, onClose }) => {
  const [selectedNRV, setSelectedNRV] = useState<NRV | null>(null);
  const [isValidating, setIsValidating] = useState(false);
  const [validationResult, setValidationResult] = useState<ValidationResult | null>(null);
  const [logStream, setLogStream] = useState<string[]>([]);
  const [nrnBalance] = useState(1500);
  const [requiredBond] = useState(100);

  const mockNRVs: NRV[] = [
    { id: 'NRV-001', title: 'Hallucination in Medical Diagnosis', domain: 'Healthcare', severity: 'high', timestamp: '2025-10-21T10:30:00Z' },
    { id: 'NRV-002', title: 'Code Generation Security Flaw', domain: 'Security', severity: 'high', timestamp: '2025-10-21T09:15:00Z' },
    { id: 'NRV-003', title: 'Math Calculation Error', domain: 'Finance', severity: 'medium', timestamp: '2025-10-21T08:45:00Z' },
  ];

  const handleLoadProblem = (nrv: NRV) => {
    setSelectedNRV(nrv);
    setValidationResult(null);
    setLogStream([]);
  };

  const handleRunValidation = () => {
    setIsValidating(true);
    setLogStream([]);
    
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

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 bg-black/50 backdrop-blur-sm flex items-center justify-center p-4">
      <div className="bg-gradient-to-br from-slate-900 via-blue-950 to-slate-950 rounded-lg border-2 border-blue-600/50 shadow-2xl max-w-6xl w-full max-h-[90vh] overflow-y-auto">
        {/* Header */}
        <div className="sticky top-0 bg-gradient-to-r from-slate-900 to-blue-950 border-b border-blue-600/30 p-6 flex items-center justify-between">
          <div className="flex items-center space-x-3">
            <FileCode className="w-6 h-6 text-blue-400" />
            <div>
              <h2 className="text-2xl font-bold text-blue-300">DVE Solver Interface</h2>
              <p className="text-xs text-slate-400 mt-1">Distributed Validation Engine</p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="bg-slate-700 hover:bg-slate-600 p-2 rounded-lg transition-colors"
          >
            <X className="w-6 h-6" />
          </button>
        </div>

        {/* Content */}
        <div className="p-6">
          <div className="grid grid-cols-3 gap-6">
            {/* Problem Panel */}
            <div className="col-span-1 bg-slate-800/50 border border-blue-600/30 rounded-lg p-6">
              <h3 className="text-lg font-bold mb-4 flex items-center text-blue-300">
                <Search className="w-5 h-5 mr-2" />
                Problem Panel
              </h3>
              
              <div className="mb-4">
                <input
                  type="text"
                  placeholder="Search NRVs..."
                  className="w-full bg-slate-950 border border-slate-700 rounded px-3 py-2 text-sm focus:outline-none focus:border-blue-500 text-slate-100"
                />
              </div>

              <div className="space-y-2 max-h-96 overflow-y-auto">
                {mockNRVs.map(nrv => (
                  <div
                    key={nrv.id}
                    onClick={() => handleLoadProblem(nrv)}
                    className={`p-3 rounded cursor-pointer transition-all border-2 ${
                      selectedNRV?.id === nrv.id
                        ? 'bg-blue-900/60 border-blue-500 shadow-lg shadow-blue-500/30'
                        : 'bg-slate-900/40 hover:bg-slate-800/60 border-slate-700'
                    }`}
                  >
                    <div className="flex items-center justify-between mb-1">
                      <span className="font-mono text-xs text-blue-400">{nrv.id}</span>
                      <span className={`text-xs px-2 py-1 rounded font-semibold ${
                        nrv.severity === 'high' ? 'bg-red-900/60 text-red-200' :
                        nrv.severity === 'medium' ? 'bg-yellow-900/60 text-yellow-200' :
                        'bg-green-900/60 text-green-200'
                      }`}>
                        {nrv.severity}
                      </span>
                    </div>
                    <div className="text-sm font-semibold mb-1 text-slate-100">{nrv.title}</div>
                    <div className="text-xs text-slate-500">{nrv.domain}</div>
                  </div>
                ))}
              </div>

              {selectedNRV && (
                <div className="mt-4 p-3 bg-slate-900/60 rounded border-2 border-blue-600/30">
                  <div className="text-xs text-slate-400 mb-2">Failure Context</div>
                  <div className="text-sm space-y-2">
                    <div><strong className="text-blue-300">Prompt:</strong> <span className="text-slate-300">"Diagnose patient symptoms..."</span></div>
                    <div><strong className="text-blue-300">Bad Response:</strong> <span className="text-slate-300">"Patient has cancer..."</span></div>
                    <div className="text-red-400"><strong>Issue:</strong> Hallucination without evidence</div>
                  </div>
                </div>
              )}
            </div>

            {/* Validation Panel */}
            <div className="col-span-2 space-y-6">
              <div className="bg-slate-800/50 border border-blue-600/30 rounded-lg p-6">
                <h3 className="text-lg font-bold mb-4 flex items-center text-blue-300">
                  <Terminal className="w-5 h-5 mr-2" />
                  Validation Panel
                </h3>

                <button
                  onClick={handleRunValidation}
                  disabled={!selectedNRV || isValidating}
                  className="w-full bg-green-600 hover:bg-green-700 disabled:bg-slate-600 disabled:cursor-not-allowed text-white font-semibold py-3 rounded-lg mb-4 transition-all flex items-center justify-center"
                >
                  <Play className="w-5 h-5 mr-2" />
                  {isValidating ? 'Running Validation...' : 'Run Validation'}
                </button>

                {/* Log Stream */}
                <div className="bg-slate-950 rounded border border-slate-700 p-4 h-48 overflow-y-auto mb-4 font-mono text-xs">
                  {logStream.length === 0 ? (
                    <div className="text-slate-500">Ready to validate. Click "Run Validation" to begin...</div>
                  ) : (
                    logStream.map((log, i) => (
                      <div key={i} className="text-green-400 mb-1">{log}</div>
                    ))
                  )}
                </div>

                {/* Validation Report Card */}
                {validationResult && (
                  <div className="space-y-3">
                    <h4 className="font-bold text-base text-blue-300">Validation Report Card</h4>
                    
                    {Object.entries(validationResult).map(([key, result]) => (
                      <div key={key} className="flex items-start p-3 bg-slate-900/60 rounded border border-slate-700">
                        {result.pass ? (
                          <CheckCircle className="w-5 h-5 text-green-400 mr-3 mt-0.5 flex-shrink-0" />
                        ) : (
                          <XCircle className="w-5 h-5 text-red-400 mr-3 mt-0.5 flex-shrink-0" />
                        )}
                        <div>
                          <div className="font-semibold capitalize text-slate-200">
                            {key.replace(/([A-Z])/g, ' $1').trim()}
                          </div>
                          <div className="text-sm text-slate-400">{result.message}</div>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              {/* Submission Panel */}
              <div className="bg-slate-800/50 border border-blue-600/30 rounded-lg p-6">
                <h3 className="text-lg font-bold mb-4 flex items-center text-blue-300">
                  <Upload className="w-5 h-5 mr-2" />
                  Submission Panel
                </h3>

                <div className="grid grid-cols-2 gap-4 mb-4">
                  <div className="bg-slate-900/60 rounded p-4 border border-slate-700">
                    <div className="text-sm text-slate-400 mb-1">NRN Balance</div>
                    <div className="text-2xl font-bold text-blue-400">{nrnBalance}</div>
                  </div>
                  <div className="bg-slate-900/60 rounded p-4 border border-slate-700">
                    <div className="text-sm text-slate-400 mb-1">Required Bond</div>
                    <div className="text-2xl font-bold text-yellow-400">{requiredBond}</div>
                  </div>
                </div>

                <button
                  onClick={handleSubmit}
                  disabled={!validationResult || !validationResult.resolution.pass}
                  className="w-full bg-purple-600 hover:bg-purple-700 disabled:bg-slate-600 disabled:cursor-not-allowed text-white font-semibold py-3 rounded-lg transition-all flex items-center justify-center"
                >
                  <Lock className="w-5 h-5 mr-2" />
                  Submit to Consensus
                </button>
              </div>
            </div>
          </div>
        </div>

        {/* Footer */}
        <div className="sticky bottom-0 bg-gradient-to-r from-slate-900 to-blue-950 border-t border-blue-600/30 p-4 flex justify-end">
          <button
            onClick={onClose}
            className="px-6 py-2 bg-slate-700 hover:bg-slate-600 text-slate-100 font-medium rounded-lg transition-colors"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
};

export default DVESolverModal;