'use client';

import React, { useState } from 'react';
import { Play, Upload, Lock, Terminal, Search, FileCode, CheckCircle, XCircle, X, ShieldCheck, Cpu, Database, Activity } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Progress } from '@/components/ui/progress';
import { Badge } from '@/components/ui/badge';

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
  const [progress, setProgress] = useState(0);
  const [nrnBalance] = useState(1500);
  const [requiredBond] = useState(100);

  const mockNRVs: NRV[] = [
    { id: 'NRV-001', title: 'Fabric Reasoning Hallucination', domain: 'Medical', severity: 'high', timestamp: '2025-10-21T10:30:00Z' },
    { id: 'NRV-002', title: 'Kernel Privilege Escalation', domain: 'Cybersec', severity: 'high', timestamp: '2025-10-21T09:15:00Z' },
    { id: 'NRV-003', title: 'Neural Weights Divergence', domain: 'Validation', severity: 'medium', timestamp: '2025-10-21T08:45:00Z' },
  ];

  const handleLoadProblem = (nrv: NRV) => {
    setSelectedNRV(nrv);
    setValidationResult(null);
    setLogStream([]);
    setProgress(0);
  };

  const handleRunValidation = () => {
    setIsValidating(true);
    setLogStream([]);
    setProgress(0);
    
    const logs = [
      '[strace] Intercepting system calls from fabric agent...',
      '[ltrace] Monitoring dynamic library linkages...',
      '[tshark] Capturing network traffic egress...',
      '[sandbox] Initializing hardware-isolated TEE enclave...',
      '[ghidra] Running static binary analysis on skill weights...',
      '[semgrep] Scanning for pattern-based logic vulnerabilities...',
      '[volatility] Analyzing memory snapshot for state injection...',
      '[validation] Running high-stakes reasoning consistency check...',
      '[consensus] Verifying proof-of-validation across peers...',
    ];

    logs.forEach((log, i) => {
      setTimeout(() => {
        setLogStream(prev => [...prev, log]);
        setProgress(((i + 1) / logs.length) * 100);
      }, i * 600);
    });

    setTimeout(() => {
      setValidationResult({
        staticAnalysis: { pass: true, message: 'Ghidra scan: OK. Weights integrity verified via BLAKE3.' },
        dynamicAnalysis: { pass: true, message: 'Syscall violations: 0. TEE Attestation: VERIFIED.' },
        forensics: { pass: true, message: 'Memory snapshot: OK. No latent state corruption detected.' },
        resolution: { pass: true, message: 'Result: SUCCESS. Fabric agent successfully resolved the FailureContext.' },
      });
      setIsValidating(false);
    }, 6000);
  };

  const handleSubmit = () => {
    alert('Fabric validation proof submitted to consensus! Transaction: 0x7a8b9c...');
    onClose();
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-[100] flex items-center justify-center p-4 bg-black/70 backdrop-blur-md">
      <div className="bg-slate-950 border border-blue-600/50 shadow-[0_0_50px_rgba(30,64,175,0.3)] rounded-2xl max-w-6xl w-full max-h-[90vh] flex flex-col overflow-hidden">
        
        {/* Header */}
        <div className="bg-slate-900 border-b border-blue-600/30 p-6 flex items-center justify-between">
          <div className="flex items-center space-x-4">
            <div className="p-2 bg-blue-600/20 rounded-lg">
              <ShieldCheck className="w-6 h-6 text-blue-400" />
            </div>
            <div>
              <h2 className="text-xl font-black text-blue-100 uppercase tracking-tighter">DVE Solver Interface</h2>
              <div className="flex items-center space-x-2 text-[10px] font-mono text-slate-500">
                <span>Fabric Stack: Agentic Memory</span>
                <span className="text-blue-500/50">•</span>
                <span className="text-green-500/80">Protocol: V-SYNC 2.0</span>
              </div>
            </div>
          </div>
          <button
            onClick={onClose}
            className="text-slate-500 hover:text-white hover:bg-slate-800 p-2 rounded-lg transition-interactive"
          >
            <X className="w-6 h-6" />
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-6 flex flex-col space-y-6 custom-scrollbar">
          <div className="grid grid-cols-12 gap-6 h-full">
            
            {/* Left Column: Problem Selection */}
            <div className="col-span-12 lg:col-span-4 flex flex-col space-y-4">
              <div className="bg-slate-900/50 border border-slate-800 rounded-xl p-5 flex flex-col h-full">
                <div className="flex items-center justify-between mb-4">
                  <h3 className="text-xs font-bold uppercase tracking-widest text-slate-400 flex items-center">
                    <Database className="w-3 h-3 mr-2 text-blue-500" />
                    Failure Contexts
                  </h3>
                  <Badge variant="outline" className="text-[10px] font-mono border-blue-500/30 text-blue-400">
                    LIVE FEED
                  </Badge>
                </div>
                
                <div className="relative mb-4">
                  <Search className="w-4 h-4 absolute left-3 top-2.5 text-slate-600" />
                  <input
                    type="text"
                    placeholder="Filter NRVs..."
                    className="w-full bg-slate-950 border border-slate-800 rounded-lg pl-10 pr-4 py-2 text-xs text-slate-300 focus:outline-none focus:border-blue-500 transition-colors"
                  />
                </div>

                <div className="flex-1 space-y-2 overflow-y-auto pr-2 custom-scrollbar">
                  {mockNRVs.map(nrv => (
                    <div
                      key={nrv.id}
                      onClick={() => handleLoadProblem(nrv)}
                      className={`p-3 rounded-lg cursor-pointer transition-interactive border ${
                        selectedNRV?.id === nrv.id
                          ? 'bg-blue-600/10 border-blue-500 shadow-[0_0_15px_rgba(59,130,246,0.1)]'
                          : 'bg-slate-950/50 hover:bg-slate-900 border-slate-800 hover:border-slate-700'
                      }`}
                    >
                      <div className="flex items-center justify-between mb-1.5">
                        <span className="font-mono text-[9px] text-blue-400 font-bold">{nrv.id}</span>
                        <span className={`text-[8px] font-black uppercase px-1.5 py-0.5 rounded ${
                          nrv.severity === 'high' ? 'bg-red-500/20 text-red-400 border border-red-500/30' :
                          nrv.severity === 'medium' ? 'bg-yellow-500/20 text-yellow-400 border border-yellow-500/30' :
                          'bg-green-500/20 text-green-400 border border-green-500/30'
                        }`}>
                          {nrv.severity}
                        </span>
                      </div>
                      <div className="text-xs font-bold text-slate-200 mb-1">{nrv.title}</div>
                      <div className="text-[10px] text-slate-500 italic">{nrv.domain} Fabric</div>
                    </div>
                  ))}
                </div>

                {selectedNRV && (
                  <div className="mt-4 p-4 bg-slate-950 rounded-lg border border-blue-600/20 space-y-3">
                    <div className="text-[10px] font-black uppercase text-blue-500 tracking-wider">Context Parameters</div>
                    <div className="space-y-2 text-[11px] font-mono">
                      <div className="flex justify-between">
                        <span className="text-slate-500">Constraint:</span>
                        <span className="text-slate-300">Determinism</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-slate-500">TEE Mode:</span>
                        <span className="text-slate-300">SGX-Enforced</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-slate-500">Sovereignty:</span>
                        <span className="text-slate-300">Level 4</span>
                      </div>
                    </div>
                  </div>
                )}
              </div>
            </div>

            {/* Right Column: Execution & Submission */}
            <div className="col-span-12 lg:col-span-8 flex flex-col space-y-6">
              
              {/* Validation Sandbox */}
              <div className="bg-slate-900/50 border border-slate-800 rounded-xl p-6 flex flex-col">
                <div className="flex items-center justify-between mb-6">
                  <h3 className="text-xs font-bold uppercase tracking-widest text-slate-400 flex items-center">
                    <Terminal className="w-3 h-3 mr-2 text-green-500" />
                    Validation Sandbox
                  </h3>
                  {isValidating && (
                    <div className="flex items-center space-x-2">
                      <Activity className="w-3 h-3 text-blue-500 animate-pulse" />
                      <span className="text-[10px] font-mono text-blue-400">ANALYSIS IN PROGRESS...</span>
                    </div>
                  )}
                </div>

                <div className="mb-6">
                  <Button
                    onClick={handleRunValidation}
                    disabled={!selectedNRV || isValidating}
                    className={`w-full py-6 rounded-xl font-black uppercase tracking-widest transition-interactive ${
                      isValidating 
                        ? 'bg-slate-800 text-slate-500 cursor-not-allowed' 
                        : 'bg-green-600 hover:bg-green-500 text-white shadow-[0_0_20px_rgba(22,163,74,0.2)]'
                    }`}
                  >
                    {isValidating ? (
                      <div className="flex items-center space-x-3">
                        <div className="w-4 h-4 border-2 border-slate-500 border-t-blue-500 rounded-full animate-spin" />
                        <span>Processing Validation Logic</span>
                      </div>
                    ) : (
                      <div className="flex items-center space-x-2">
                        <Play className="w-4 h-4 fill-current" />
                        <span>Initialize Solver Logic</span>
                      </div>
                    )}
                  </Button>
                  {isValidating && <Progress value={progress} className="h-1 mt-4 bg-slate-800" />}
                </div>

                {/* Log Stream Terminal */}
                <div className="bg-black rounded-lg border border-slate-800 p-4 h-48 overflow-y-auto font-mono text-[11px] custom-scrollbar">
                  {logStream.length === 0 ? (
                    <div className="text-slate-700 italic">Sandbox ready. Select a context and initialize...</div>
                  ) : (
                    logStream.map((log, i) => (
                      <div key={i} className={`${log.startsWith('[v') ? 'text-blue-400' : 'text-green-500/80'} mb-1.5`}>
                        <span className="text-slate-600 mr-2">[{new Date().toLocaleTimeString()}]</span>
                        {log}
                      </div>
                    ))
                  )}
                </div>

                {/* Validation Results */}
                {validationResult && (
                  <div className="mt-6 grid grid-cols-2 gap-4 animate-in fade-in slide-in-from-bottom-2">
                    {Object.entries(validationResult).map(([key, result]) => (
                      <div key={key} className="flex items-start p-3 bg-slate-950/50 rounded-lg border border-slate-800">
                        {result.pass ? (
                          <CheckCircle className="w-4 h-4 text-green-500 mr-3 mt-0.5 flex-shrink-0" />
                        ) : (
                          <XCircle className="w-4 h-4 text-red-500 mr-3 mt-0.5 flex-shrink-0" />
                        )}
                        <div>
                          <div className="text-[10px] font-black uppercase text-slate-200 tracking-tighter">
                            {key.replace(/([A-Z])/g, ' $1').trim()}
                          </div>
                          <div className="text-[10px] text-slate-500 leading-tight mt-0.5">{result.message}</div>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              {/* Submission & Economics */}
              <div className="bg-slate-900/50 border border-slate-800 rounded-xl p-6">
                <div className="flex items-center justify-between mb-6">
                  <h3 className="text-xs font-bold uppercase tracking-widest text-slate-400 flex items-center">
                    <Upload className="w-3 h-3 mr-2 text-purple-500" />
                    Economic Settlement
                  </h3>
                </div>

                <div className="grid grid-cols-2 gap-4 mb-6">
                  <div className="bg-slate-950 p-4 rounded-xl border border-slate-800">
                    <div className="text-[10px] font-bold text-slate-500 uppercase mb-1">NRN Portfolio</div>
                    <div className="text-2xl font-black text-blue-400">{nrnBalance.toLocaleString()} <span className="text-xs font-normal text-slate-600">NRN</span></div>
                  </div>
                  <div className="bg-slate-950 p-4 rounded-xl border border-slate-800">
                    <div className="text-[10px] font-bold text-slate-500 uppercase mb-1">Required Staked Bond</div>
                    <div className="text-2xl font-black text-amber-500">{requiredBond} <span className="text-xs font-normal text-slate-600">NRN</span></div>
                  </div>
                </div>

                <Button
                  onClick={handleSubmit}
                  disabled={!validationResult || !validationResult.resolution.pass}
                  className={`w-full py-6 rounded-xl font-black uppercase tracking-widest transition-interactive ${
                    !validationResult || !validationResult.resolution.pass
                      ? 'bg-slate-800 text-slate-600'
                      : 'bg-purple-600 hover:bg-purple-500 text-white shadow-[0_0_30px_rgba(147,51,234,0.3)]'
                  }`}
                >
                  <Lock className="w-4 h-4 mr-2" />
                  Commit Proof to Global Consensus
                </Button>
              </div>
            </div>
          </div>
        </div>

        {/* Footer */}
        <div className="bg-slate-900 border-t border-blue-600/30 p-4 flex justify-between items-center px-8">
          <div className="flex items-center space-x-2">
            <div className="w-2 h-2 rounded-full bg-green-500 animate-pulse" />
            <span className="text-[10px] font-mono text-slate-500 uppercase">System Status: Nominal</span>
          </div>
          <Button
            variant="ghost"
            onClick={onClose}
            className="text-slate-400 hover:text-white"
          >
            Close Interface
          </Button>
        </div>
      </div>
    </div>
  );
};

export default DVESolverModal;
