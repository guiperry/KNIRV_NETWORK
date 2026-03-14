'use client';

import React, { useState } from 'react';
import { Shield, X, Lock, Zap, Eye, Terminal, AlertCircle } from 'lucide-react';
import { Switch } from '@/components/ui/switch';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { API_BASE_URL, getAuthHeaders } from '@/lib/api';

interface PolicyEditorProps {
  isOpen: boolean;
  onClose: () => void;
  nodeId?: string;
  isMonitorOpen?: boolean;
}

const PolicyEditor: React.FC<PolicyEditorProps> = ({ isOpen, onClose, nodeId, isMonitorOpen }) => {
  const [networkWhitelist, setNetworkWhitelist] = useState('api.company.com\ncdn.trusted.io\ngateway.knirv.network');
  const [sensitivity, setSensitivity] = useState('Balanced');
  const [blockFileIO, setBlockFileIO] = useState(true);
  const [allowReadOnly, setAllowReadOnly] = useState(false);
  const [enableForensics, setEnableForensics] = useState(true);
  const [enforceAttestation, setEnableAttestation] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [saveStatus, setSaveStatus] = useState<'idle' | 'success' | 'error'>('idle');

  const handleCommitPolicy = async () => {
    setIsSaving(true);
    setSaveStatus('idle');
    try {
      const policy = {
        name: `guardrail-${nodeId ?? 'global'}-${Date.now()}`,
        type: 'guardrail',
        rules: {
          network_whitelist: networkWhitelist.split('\n').filter(Boolean),
          sensitivity,
          block_file_io: blockFileIO,
          allow_read_only: allowReadOnly,
          enable_forensics: enableForensics,
          enforce_attestation: enforceAttestation,
        },
        priority: 1,
        enabled: true,
        target_dve: nodeId ?? '',
        created_at: new Date().toISOString(),
      };
      const response = await fetch(`${API_BASE_URL}/api/icme/policy/commit`, {
        method: 'POST',
        headers: getAuthHeaders(),
        body: JSON.stringify(policy),
      });
      setSaveStatus(response.ok ? 'success' : 'error');
    } catch {
      setSaveStatus('error');
    } finally {
      setIsSaving(false);
      setTimeout(() => setSaveStatus('idle'), 3000);
    }
  };

  if (!isOpen) return null;

  return (
    <div
      className="absolute z-[100] transition-all duration-500 transform ease-in-out bg-slate-900 border border-blue-600/50 shadow-2xl overflow-hidden rounded-xl"
      style={{
        right: isMonitorOpen ? '40px' : '20px',
        top: isMonitorOpen ? '340px' : '450px',
        width: isMonitorOpen ? '500px' : '550px',
        height: isMonitorOpen ? '280px' : '380px',
      }}
    >
      <div className="h-full flex flex-col">
        <div className="flex items-center justify-between p-4 border-b border-blue-600/30 bg-slate-950/50">
          <div className="flex items-center space-x-2">
            <Shield className="w-5 h-5 text-amber-400" />
            <div>
              <h2 className="text-sm font-bold text-blue-300 tracking-tight">
                Fabric Guardrail Policy {nodeId && `[${nodeId}]`}
              </h2>
              <div className="text-[10px] text-slate-500 font-mono">Status: ACTIVE - Enforced via TEE</div>
            </div>
          </div>
          <button
            onClick={onClose}
            className="text-slate-500 hover:text-white hover:bg-slate-800 p-1 rounded transition-colors"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        <div className="flex-1 overflow-y-auto p-5 space-y-6 custom-scrollbar">
          <div className="grid grid-cols-2 gap-6">
            {/* Left Column: Config */}
            <div className="space-y-4">
              <div>
                <Label className="text-[11px] font-bold uppercase text-slate-400 mb-2 block">Network Egress Whitelist</Label>
                <textarea
                  value={networkWhitelist}
                  onChange={(e) => setNetworkWhitelist(e.target.value)}
                  className="w-full bg-slate-950 border border-slate-700 rounded-md p-2 text-[11px] font-mono text-blue-400 focus:ring-1 focus:ring-blue-500 focus:outline-none h-24 resize-none"
                  placeholder="domain.com..."
                />
              </div>
              
              <div>
                <Label className="text-[11px] font-bold uppercase text-slate-400 mb-2 block">Constraint Sensitivity</Label>
                <select
                  value={sensitivity}
                  onChange={(e) => setSensitivity(e.target.value)}
                  className="w-full bg-slate-950 border border-slate-700 rounded-md p-2 text-[11px] text-slate-200 focus:ring-1 focus:ring-blue-500 focus:outline-none"
                >
                  <option>Balanced (Default)</option>
                  <option>Zero-Trust (Paranoid)</option>
                  <option>Open-Research (Lenient)</option>
                </select>
              </div>
            </div>

            {/* Right Column: Toggles */}
            <div className="space-y-4">
              <Label className="text-[11px] font-bold uppercase text-slate-400 mb-2 block">Hardware Enforcement</Label>
              
              <div className="space-y-3 bg-slate-950/30 p-3 rounded-lg border border-slate-800">
                <div className="flex items-center justify-between">
                  <div className="space-y-0.5">
                    <Label className="text-xs text-slate-200 flex items-center">
                      <Lock className="w-3 h-3 mr-1.5 text-blue-500" />
                      Block Host I/O
                    </Label>
                    <p className="text-[9px] text-slate-500">Prevent raw disk access</p>
                  </div>
                  <Switch checked={blockFileIO} onCheckedChange={setBlockFileIO} />
                </div>

                <div className="flex items-center justify-between">
                  <div className="space-y-0.5">
                    <Label className="text-xs text-slate-200 flex items-center">
                      <Zap className="w-3 h-3 mr-1.5 text-yellow-500" />
                      eBPF Forensics
                    </Label>
                    <p className="text-[9px] text-slate-500">Enable deep syscall tracing</p>
                  </div>
                  <Switch checked={enableForensics} onCheckedChange={setEnableForensics} />
                </div>

                <div className="flex items-center justify-between">
                  <div className="space-y-0.5">
                    <Label className="text-xs text-slate-200 flex items-center">
                      <Terminal className="w-3 h-3 mr-1.5 text-green-500" />
                      TEE Attestation
                    </Label>
                    <p className="text-[9px] text-slate-500">Verify Enclave integrity</p>
                  </div>
                  <Switch checked={enforceAttestation} onCheckedChange={setEnableAttestation} />
                </div>
              </div>
              
              <div className="flex items-center justify-center p-2 bg-amber-500/10 border border-amber-500/20 rounded-md">
                <AlertCircle className="w-3 h-3 text-amber-500 mr-2" />
                <span className="text-[9px] text-amber-200">Policy changes require sandbox restart</span>
              </div>
            </div>
          </div>
        </div>

        <div className="p-4 border-t border-blue-600/30 bg-slate-950/50 flex items-center justify-between">
          {saveStatus === 'success' && (
            <span className="text-[10px] text-green-400 font-mono">Policy committed successfully</span>
          )}
          {saveStatus === 'error' && (
            <span className="text-[10px] text-red-400 font-mono">Failed to commit policy</span>
          )}
          {saveStatus === 'idle' && <span />}
          <Button
            size="sm"
            className="bg-blue-600 hover:bg-blue-700 text-white font-bold px-6"
            onClick={handleCommitPolicy}
            disabled={isSaving}
          >
            {isSaving ? 'Committing...' : 'Commit Policy to Blockchain'}
          </Button>
        </div>
      </div>
    </div>
  );
};

export default PolicyEditor;
