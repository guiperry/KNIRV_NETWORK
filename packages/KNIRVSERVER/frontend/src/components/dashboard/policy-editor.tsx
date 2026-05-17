'use client';

import React, { useState, useEffect } from 'react';
import { Shield, X, Lock, Zap, Eye, Terminal, AlertCircle, RefreshCw, Save, Loader2, Award } from 'lucide-react';
import { Switch } from '@/components/ui/switch';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { API_BASE_URL, getAuthHeaders } from '@/lib/api';
import { useBadgeTemplates, type BadgeTemplate } from '@/hooks/use-badge-templates';

interface PolicyRule {
  id: string;
  description: string;
  dveId: string;
  metric: string;
  operator: string;
  threshold: string;
  severity: string;
  remediationAction: string;
  enabled: boolean;
  triggerCount: number;
}

interface Policy {
  id: string;
  name: string;
  rules: PolicyRule[];
  priority: number;
  enabled: boolean;
  targetDVE: string;
  createdAt: string;
  committedAt?: string;
  txHash?: string;
}

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
  const [isCommitting, setIsCommitting] = useState(false);
  const [saveStatus, setSaveStatus] = useState<'idle' | 'saving' | 'success' | 'error'>('idle');
  const [commitStatus, setCommitStatus] = useState<'idle' | 'committing' | 'success' | 'error'>('idle');
  const [policies, setPolicies] = useState<Policy[]>([]);
  const [selectedPolicy, setSelectedPolicy] = useState<Policy | null>(null);
  const [txHash, setTxHash] = useState<string | null>(null);

  // Badge template integration
  const { templates, listTemplates, mintFromTemplate, isLoading: templatesLoading } = useBadgeTemplates();
  const [attachedBadgeIds, setAttachedBadgeIds] = useState<string[]>([]);
  const [attachingId, setAttachingId] = useState<string | null>(null);

  useEffect(() => {
    if (isOpen) {
      loadPolicies();
      listTemplates(true);
      loadAttachedBadges();
    }
  }, [isOpen, nodeId]);

  const loadAttachedBadges = async () => {
    if (!nodeId) return;
    try {
      const resp = await fetch(`${API_BASE_URL}/api/badge/templates/dve/${nodeId}/badges`, {
        headers: getAuthHeaders(),
      });
      if (resp.ok) {
        const data = await resp.json();
        setAttachedBadgeIds(data.badge_ids || data.attached_badges || []);
      }
    } catch {
      // Best-effort
    }
  };

  const loadPolicies = async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/guardrail/policies?node_id=${nodeId ?? ''}`, {
        headers: getAuthHeaders(),
      });
      if (response.ok) {
        const data = await response.json();
        setPolicies(data.policies || []);
        if (data.policies?.length > 0) {
          setSelectedPolicy(data.policies[0]);
        }
      }
    } catch (error) {
      console.error('Failed to load policies:', error);
    }
  };

  const handleSavePolicy = async () => {
    setIsSaving(true);
    setSaveStatus('saving');
    try {
      const policyRule: PolicyRule = {
        id: `rule-${Date.now()}`,
        description: `Guardrail policy for ${nodeId ?? 'global'}`,
        dveId: nodeId ?? '',
        metric: 'network_egress',
        operator: 'in',
        threshold: networkWhitelist.split('\n').filter(Boolean).join(','),
        severity: 'high',
        remediationAction: 'block',
        enabled: true,
        triggerCount: 0,
      };

      const policy: Partial<Policy> = {
        name: `guardrail-${nodeId ?? 'global'}-${Date.now()}`,
        rules: [policyRule],
        priority: 1,
        enabled: true,
        targetDVE: nodeId ?? '',
        createdAt: new Date().toISOString(),
      };

      const response = await fetch(`${API_BASE_URL}/api/guardrail/policies`, {
        method: 'POST',
        headers: { ...getAuthHeaders(), 'Content-Type': 'application/json' },
        body: JSON.stringify(policy),
      });

      if (response.ok) {
        const data = await response.json();
        setSelectedPolicy(data.policy);
        setSaveStatus('success');

        // Attach the new policy to the DVE node it was created for
        if (nodeId && data.policy?.id) {
          await fetch(`${API_BASE_URL}/api/dve-nodes/nodes/${nodeId}/policies`, {
            method: 'POST',
            headers: { ...getAuthHeaders(), 'Content-Type': 'application/json' },
            body: JSON.stringify({ policy_id: data.policy.id }),
          });
        }

        await loadPolicies();
      } else {
        setSaveStatus('error');
      }
    } catch {
      setSaveStatus('error');
    } finally {
      setIsSaving(false);
      setTimeout(() => setSaveStatus('idle'), 3000);
    }
  };

  const handleCommitPolicy = async () => {
    if (!selectedPolicy?.id) {
      await handleSavePolicy();
    }

    setIsCommitting(true);
    setCommitStatus('committing');
    try {
      const policyId = selectedPolicy?.id || policies[0]?.id;
      if (!policyId) {
        setCommitStatus('error');
        return;
      }

      const response = await fetch(`${API_BASE_URL}/api/guardrail/policies/${policyId}/commit`, {
        method: 'POST',
        headers: getAuthHeaders(),
      });

      if (response.ok) {
        const data = await response.json();
        setTxHash(data.tx_hash);
        setCommitStatus('success');
        await loadPolicies();
      } else {
        setCommitStatus('error');
      }
    } catch {
      setCommitStatus('error');
    } finally {
      setIsCommitting(false);
      setTimeout(() => setCommitStatus('idle'), 5000);
    }
  };

  const handleAttachBadge = async (templateId: string) => {
    if (!nodeId) return;
    setAttachingId(templateId);
    try {
      const result = await mintFromTemplate(templateId, nodeId);
      if (result) {
        await loadAttachedBadges();
      }
    } finally {
      setAttachingId(null);
    }
  };

  const handleDetachBadge = async (badgeId: string) => {
    if (!nodeId) return;
    try {
      const resp = await fetch(`${API_BASE_URL}/api/badge/templates/dve/${nodeId}/badges/${badgeId}`, {
        method: 'DELETE',
        headers: getAuthHeaders(),
      });
      if (resp.ok) {
        await loadAttachedBadges();
      }
    } catch {
      // Best-effort
    }
  };

  if (!isOpen) return null;

  return (
    <div
      className="absolute z-[100] transition-slide duration-500 ease-in-out bg-slate-900 border border-blue-600/50 shadow-2xl overflow-hidden rounded-xl gpu-accelerated"
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

          {/* Badge Template Attachments Section */}
          <div className="border-t border-blue-600/20 pt-4 mt-2">
            <div className="flex items-center justify-between mb-3">
              <Label className="text-[11px] font-bold uppercase text-amber-400 flex items-center">
                <Award className="w-3.5 h-3.5 mr-1.5" />
                Attached Badge Templates
              </Label>
              {templatesLoading && <Loader2 className="w-3 h-3 animate-spin text-amber-400" />}
            </div>

            {/* Currently attached badges */}
            {attachedBadgeIds.length > 0 && (
              <div className="mb-3 space-y-1.5">
                <p className="text-[9px] text-slate-500 font-mono mb-1">Attached to this DVE:</p>
                {attachedBadgeIds.map((bid) => (
                  <div key={bid} className="flex items-center justify-between bg-green-900/10 border border-green-500/20 rounded-md px-3 py-1.5">
                    <span className="text-[10px] font-mono text-green-400 truncate">{bid}</span>
                    <button
                      onClick={() => handleDetachBadge(bid)}
                      className="text-[9px] text-red-400 hover:text-red-300 font-bold uppercase tracking-wider ml-2 shrink-0"
                    >
                      Detach
                    </button>
                  </div>
                ))}
              </div>
            )}

            {/* Available templates to attach */}
            {nodeId && templates.length > 0 && (
              <div className="space-y-1.5 max-h-32 overflow-y-auto">
                <p className="text-[9px] text-slate-500 font-mono mb-1">Available templates:</p>
                {templates.map((tmpl) => (
                  <div key={tmpl.id} className="flex items-center justify-between bg-slate-950/40 border border-slate-800 rounded-md px-3 py-1.5 hover:border-amber-500/30 transition-colors">
                    <div className="flex items-center gap-2 min-w-0">
                      <span className="text-[10px] font-bold text-slate-200 truncate">{tmpl.name}</span>
                      <Badge variant="outline" className="text-[7px] text-amber-400 border-amber-400/20 px-1 py-0">
                        {tmpl.value_signals?.length || 0}v
                      </Badge>
                    </div>
                    <button
                      onClick={() => handleAttachBadge(tmpl.id)}
                      disabled={attachingId === tmpl.id}
                      className="text-[9px] px-2 py-1 rounded bg-amber-600/20 hover:bg-amber-600/30 text-amber-400 font-bold uppercase tracking-wider transition-colors shrink-0 ml-2"
                    >
                      {attachingId === tmpl.id ? <Loader2 className="w-2.5 h-2.5 animate-spin" /> : 'Attach'}
                    </button>
                  </div>
                ))}
              </div>
            )}

            {!nodeId && (
              <p className="text-[9px] text-slate-600 italic">Select a DVE node to attach badge templates</p>
            )}
          </div>
        </div>

        <div className="p-4 border-t border-blue-600/30 bg-slate-950/50 flex items-center justify-between">
          {saveStatus === 'success' && (
            <span className="text-[10px] text-green-400 font-mono">Policy saved successfully</span>
          )}
          {saveStatus === 'error' && (
            <span className="text-[10px] text-red-400 font-mono">Failed to save policy</span>
          )}
          {saveStatus === 'saving' && (
            <span className="text-[10px] text-blue-400 font-mono flex items-center">
              <Loader2 className="w-3 h-3 mr-1 animate-spin" /> Saving...
            </span>
          )}
          {commitStatus === 'success' && txHash && (
            <span className="text-[10px] text-green-400 font-mono">Committed! TX: {txHash.slice(0, 12)}...</span>
          )}
          {commitStatus === 'error' && (
            <span className="text-[10px] text-red-400 font-mono">Failed to commit to blockchain</span>
          )}
          {commitStatus === 'committing' && (
            <span className="text-[10px] text-blue-400 font-mono flex items-center">
              <Loader2 className="w-3 h-3 mr-1 animate-spin" /> Committing to blockchain...
            </span>
          )}
          {saveStatus === 'idle' && commitStatus === 'idle' && (
            <span className="text-[10px] text-slate-500 font-mono">
              {policies.length > 0 ? `${policies.length} policies loaded` : 'No policies loaded'}
            </span>
          )}
          <div className="flex gap-2">
            <Button
              size="sm"
              variant="outline"
              className="border-slate-600 hover:border-blue-500"
              onClick={handleSavePolicy}
              disabled={isSaving || isCommitting}
            >
              {isSaving ? <Loader2 className="w-3 h-3 animate-spin mr-1" /> : <Save className="w-3 h-3 mr-1" />}
              Save Policy
            </Button>
            <Button
              size="sm"
              className="bg-blue-600 hover:bg-blue-700 text-white font-bold px-6"
              onClick={handleCommitPolicy}
              disabled={isSaving || isCommitting}
            >
              {isCommitting ? <Loader2 className="w-3 h-3 animate-spin mr-1" /> : <Shield className="w-3 h-3 mr-1" />}
              Commit to Blockchain
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
};

export default PolicyEditor;
