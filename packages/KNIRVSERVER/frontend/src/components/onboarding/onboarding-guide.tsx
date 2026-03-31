'use client';

import React, { useState } from 'react';
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import { 
  Shield, 
  Key, 
  Database, 
  Terminal,
  ChevronRight,
  ChevronLeft,
  CheckCircle2,
  Lock,
  Smartphone,
  Download,
  Settings,
  Home,
  SlidersHorizontal,
  Cpu,
  Mail,
  Loader2
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { APIKeysModal, type APIKeyEntry } from "./modals/APIKeysModal";
import { MCPServersModal, type MCPServerEntry } from "./modals/MCPServersModal";
import { PolicyCertsModal, type PolicyCert, type CustomRule } from "./modals/PolicyCertsModal";
import { PreferencesModal, type PrivacySettings } from "./modals/PreferencesModal";
import { API_BASE_URL, getAuthHeaders } from '@/lib/api';

interface OnboardingGuideProps {
  onComplete: (config: {
    walletName: string;
    fabricInputs: string[];
    guardrails: {
      networkDrift: boolean;
      filesystemAccess: boolean;
      computeCostCap: boolean;
    };
    connectionData: {
      apiKeys: APIKeyEntry[];
      mcpServers: MCPServerEntry[];
      policyCerts: PolicyCert[];
      customRules: CustomRule[];
    };
    completedConnections: string[];
    privacySettings: PrivacySettings;
  }) => void;
  onReset?: () => void;
}

interface FormData {
  walletName: string;
  selectedInputs: string[];
  guardrails: {
    networkDrift: boolean;
    filesystemAccess: boolean;
    computeCostCap: boolean;
  };
  connectionData: {
    apiKeys: APIKeyEntry[];
    mcpServers: MCPServerEntry[];
    policyCerts: PolicyCert[];
    customRules: CustomRule[];
    ingestedRepos: string[];
  };
  completedConnections: string[];
  privacySettings: PrivacySettings;
}

const fabricInputs = [
  { id: 'api-keys', icon: Key, label: 'API Keys', desc: 'Secure LLM & Service Credentials' },
  { id: 'mcp-servers', icon: Terminal, label: 'MCP Servers', desc: 'Model Context Protocol Integrations' },
  { id: 'policy-certs', icon: Database, label: 'Policy Certs', desc: 'Kernel Guardrails & Custom Rules' },
  { id: 'preferences', icon: SlidersHorizontal, label: 'Preferences', desc: 'Data Management & Privacy Settings' },
  { id: 'repo-ingest', icon: Database, label: 'Knowledge Ingest', desc: 'Import repositories & documents to graph' }
];

const guardrailPolicies = [
  { id: 'network-drift', label: 'Outbound Network Drift', value: 'High Accuracy (Strict)', defaultEnabled: true },
  { id: 'filesystem-access', label: 'Filesystem Access Depth', value: 'Restricted to /mnt/server', defaultEnabled: true },
  { id: 'compute-cost', label: 'Compute Cost Cap', value: '$100.00 / Session', defaultEnabled: true }
];

const OnboardingGuide = ({ onComplete, onReset }: OnboardingGuideProps) => {
  const [step, setStep] = useState(1);
  const [progress, setProgress] = useState(25);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [ingestUrl, setIngestUrl] = useState('');
  const [isIngesting, setIsIngesting] = useState(false);
  const [ingestLog, setIngestLog] = useState<string[]>([]);
  const [formData, setFormData] = useState<FormData>({
    walletName: '',
    selectedInputs: [],
    guardrails: {
      networkDrift: true,
      filesystemAccess: true,
      computeCostCap: true
    },
    connectionData: {
      apiKeys: [],
      mcpServers: [],
      policyCerts: [],
      customRules: [],
      ingestedRepos: []
    },
    completedConnections: [],
    privacySettings: {
      dataEncryption: true,
      localProcessing: true,
      anonymizeMetrics: true,
      shareErrorLogs: false,
      allowAnalytics: false,
      dataRetentionDays: 90,
      autoDeleteInactive: true,
      thirdPartyIntegrations: false
    }
  });

  const handleIngestRepo = async () => {
    if (!ingestUrl.trim() || isIngesting) return;
    setIsIngesting(true);
    setIngestLog(prev => [...prev, `[${new Date().toLocaleTimeString()}] Ingesting: ${ingestUrl.trim()}`]);
    try {
      const resp = await fetch(`${API_BASE_URL}/api/knirvgraph/gitnexus/ingest`, {
        method: 'POST',
        headers: { ...getAuthHeaders(), 'Content-Type': 'application/json' },
        body: JSON.stringify({ repo_url: ingestUrl.trim() }),
      });
      const data = await resp.json();
      if (resp.ok) {
        setIngestLog(prev => [...prev, `[${new Date().toLocaleTimeString()}] ✓ ${data.message || 'Ingestion queued'}`]);
        setFormData(prev => ({
          ...prev,
          connectionData: {
            ...prev.connectionData,
            ingestedRepos: [...prev.connectionData.ingestedRepos, ingestUrl.trim()]
          },
          completedConnections: prev.completedConnections.includes('repo-ingest')
            ? prev.completedConnections
            : [...prev.completedConnections, 'repo-ingest']
        }));
        setIngestUrl('');
      } else {
        setIngestLog(prev => [...prev, `[${new Date().toLocaleTimeString()}] ✗ ${data.error || 'Ingestion failed'}`]);
      }
    } catch {
      setIngestLog(prev => [...prev, `[${new Date().toLocaleTimeString()}] ✗ Network error — backend unreachable`]);
    } finally {
      setIsIngesting(false);
    }
  };

  const submitGuardrailsToBackend = async (walletName: string, certs: PolicyCert[], rules: CustomRule[]) => {
    try {
      const policy = {
        name: `onboarding-${walletName}-${Date.now()}`,
        rules: certs.map(cert => ({
          id: cert.id,
          description: cert.description,
          dveId: walletName,
          metric: cert.category.toLowerCase().replace(' ', '_'),
          operator: 'eq',
          threshold: String(cert.value),
          severity: cert.enabled ? 'high' : 'low',
          remediationAction: 'notify',
          enabled: cert.enabled,
          triggerCount: 0,
        })),
        priority: 1,
        enabled: true,
        targetDVE: walletName,
        createdAt: new Date().toISOString(),
      };

      const response = await fetch(`${API_BASE_URL}/api/guardrails/policies`, {
        method: 'POST',
        headers: { ...getAuthHeaders(), 'Content-Type': 'application/json' },
        body: JSON.stringify(policy),
      });

      if (response.ok) {
        const data = await response.json();
        if (data.policy?.id) {
          await fetch(`${API_BASE_URL}/api/guardrails/policies/${data.policy.id}/commit`, {
            method: 'POST',
            headers: getAuthHeaders(),
          });
        }
      }
    } catch (error) {
      console.error('Failed to submit guardrails:', error);
    }
  };

  // Modal states
  const [activeModal, setActiveModal] = useState<string | null>(null);

  const steps = [
    { id: 1, title: 'Identity', description: 'Initialize Data Fabric' },
    { id: 2, title: 'Connections', description: 'Map Fabric Inputs' },
    { id: 3, title: 'Governance', description: 'Set Kernel Guardrails' },
    { id: 4, title: 'Sovereignty', description: 'Secure the Vault' }
  ];

  const submitOrganizationContext = async (walletName: string) => {
    try {
      const guidelines = formData.connectionData.customRules.map(r => r.description || r.name || '').filter(Boolean);
      const statedValues = formData.selectedInputs;

      const orgPayload = {
        organization_id: walletName,
        name: walletName,
        value_system: {
          guidelines,
          stated_values: statedValues,
          mission_statement: `Data Fabric for ${walletName}`,
          risk_appetite: {
            level: formData.guardrails.computeCostCap ? 'medium' : 'high',
            tolerance_score: formData.guardrails.networkDrift ? 0.6 : 0.9,
            max_loss_percent: 10.0,
            recovery_time: '24h',
          },
        },
        ontology: {
          policies: formData.connectionData.policyCerts.map(c => c.description || c.name || '').filter(Boolean),
          regulations: [],
          trade_secrets: [],
          rules: formData.connectionData.customRules.map(r => r.description || '').filter(Boolean),
        },
      };

      await fetch(`${API_BASE_URL}/api/onboarding/organizations`, {
        method: 'POST',
        headers: { ...getAuthHeaders(), 'Content-Type': 'application/json' },
        body: JSON.stringify(orgPayload),
      });
    } catch (error) {
      console.error('Failed to submit organization context:', error);
    }
  };

  const handleNext = async () => {
    if (step < 4) {
      setStep(step + 1);
      setProgress((step + 1) * 25);
    } else {
      setIsSubmitting(true);
      setSubmitError(null);

      try {
        const walletName = formData.walletName || 'DEFAULT-WALLET';

        await Promise.all([
          submitGuardrailsToBackend(
            walletName,
            formData.connectionData.policyCerts,
            formData.connectionData.customRules
          ),
          submitOrganizationContext(walletName),
        ]);

        onComplete({
          walletName,
          fabricInputs: formData.selectedInputs,
          guardrails: formData.guardrails,
          connectionData: formData.connectionData,
          completedConnections: formData.completedConnections,
          privacySettings: formData.privacySettings
        });
      } catch (error) {
        setSubmitError('Failed to complete onboarding. Please try again.');
      } finally {
        setIsSubmitting(false);
      }
    }
  };

  const handleBack = () => {
    if (step > 1) {
      setStep(step - 1);
      setProgress((step - 1) * 25);
    }
  };

  const toggleInput = (inputId: string) => {
    setFormData(prev => ({
      ...prev,
      selectedInputs: prev.selectedInputs.includes(inputId)
        ? prev.selectedInputs.filter(id => id !== inputId)
        : [...prev.selectedInputs, inputId]
    }));
  };

  const openModal = (modalId: string) => {
    setActiveModal(modalId);
  };

  const closeModal = () => {
    setActiveModal(null);
  };

  const isConnectionComplete = (connectionId: string) => {
    switch (connectionId) {
      case 'api-keys':
        return formData.connectionData.apiKeys.length > 0;
      case 'mcp-servers':
        return formData.connectionData.mcpServers.length > 0;
      case 'policy-certs':
        return formData.connectionData.policyCerts.length > 0 || formData.connectionData.customRules.length > 0;
      case 'preferences':
        return formData.completedConnections.includes('preferences');
      case 'repo-ingest':
        return formData.connectionData.ingestedRepos.length > 0;
      default:
        return false;
    }
  };

  const handleSaveAPIKeys = (apiKeys: APIKeyEntry[]) => {
    setFormData(prev => ({
      ...prev,
      connectionData: { ...prev.connectionData, apiKeys },
      completedConnections: prev.completedConnections.includes('api-keys')
        ? prev.completedConnections
        : [...prev.completedConnections, 'api-keys']
    }));
  };

  const handleSaveMCPServers = (mcpServers: MCPServerEntry[]) => {
    setFormData(prev => ({
      ...prev,
      connectionData: { ...prev.connectionData, mcpServers },
      completedConnections: prev.completedConnections.includes('mcp-servers')
        ? prev.completedConnections
        : [...prev.completedConnections, 'mcp-servers']
    }));
  };

  const handleSavePolicyCerts = (policyCerts: PolicyCert[], customRules: CustomRule[]) => {
    setFormData(prev => ({
      ...prev,
      connectionData: { ...prev.connectionData, policyCerts, customRules },
      completedConnections: prev.completedConnections.includes('policy-certs')
        ? prev.completedConnections
        : [...prev.completedConnections, 'policy-certs']
    }));
  };

  const handleSavePreferences = (privacySettings: PrivacySettings) => {
    setFormData(prev => ({
      ...prev,
      privacySettings,
      completedConnections: prev.completedConnections.includes('preferences')
        ? prev.completedConnections
        : [...prev.completedConnections, 'preferences']
    }));
  };

  const toggleGuardrail = (guardrailId: string) => {
    setFormData(prev => ({
      ...prev,
      guardrails: {
        ...prev.guardrails,
        [guardrailId]: !prev.guardrails[guardrailId as keyof typeof prev.guardrails]
      }
    }));
  };

  return (
    <div className="bg-[#0a0a0c] text-slate-200 font-sans selection:bg-blue-500/30 relative">
      {/* Background Sentinel Mesh - restricted to container */}
      <div className="absolute inset-0 overflow-hidden pointer-events-none opacity-20">
        <div className="absolute top-[-10%] left-[-10%] w-[40%] h-[40%] bg-blue-600/20 blur-[120px] rounded-full" />
        <div className="absolute bottom-[-10%] right-[-10%] w-[40%] h-[40%] bg-indigo-600/10 blur-[120px] rounded-full" />
      </div>

      {/* Header */}
      <nav className="relative z-10 p-6 flex justify-between items-center border-b border-white/5 bg-black/40 backdrop-blur-md">
        <div className="flex items-center space-x-2">
          <div className="w-6 h-6 bg-blue-600 rounded-sm transform rotate-45" />
          <span className="text-xl font-extrabold tracking-tighter uppercase">KNIRV <span className="text-blue-500 font-light italic">ONBOARDING</span></span>
        </div>
        <div className="flex items-center space-x-4">
          <div className="h-2 w-32 bg-white/5 rounded-full overflow-hidden">
            <div 
              className="h-full bg-blue-600 transition-all duration-700 ease-out" 
              style={{ width: `${progress}%` }}
            />
          </div>
          <span className="text-[10px] mono text-slate-500 font-bold uppercase tracking-widest">Step 0{step} / 04</span>
          {onReset && (
            <button
              onClick={onReset}
              className="flex items-center space-x-1 text-xs text-slate-500 hover:text-red-400 transition-colors ml-4"
              title="Exit onboarding and return to home"
            >
              <Home size={14} />
              <span className="hidden sm:inline">Exit</span>
            </button>
          )}
        </div>
      </nav>

      <main className="relative z-10 max-w-4xl mx-auto px-6 py-12 md:py-16">
        
        {/* Step Progress Display */}
        <div className="flex justify-between mb-12 relative">
          <div className="absolute top-1/2 left-0 w-full h-[1px] bg-white/5 -z-10" />
          {steps.map((s) => (
            <div key={s.id} className="flex flex-col items-center group">
              <div className={`w-8 h-8 rounded-full flex items-center justify-center border transition-interactive duration-500 ${
                step >= s.id ? 'bg-blue-600 border-blue-400 text-white shadow-[0_0_15px_rgba(59,130,246,0.5)]' : 'bg-black border-white/10 text-slate-500'
              }`}>
                {step > s.id ? <CheckCircle2 size={16} /> : <span className="text-xs font-bold">{s.id}</span>}
              </div>
              <span className={`mt-3 text-[10px] uppercase tracking-widest font-bold transition-colors ${
                step === s.id ? 'text-blue-400' : 'text-slate-600'
              }`}>{s.title}</span>
            </div>
          ))}
        </div>

        {/* Dynamic Content Area */}
        <div className="min-h-[400px]">
          {step === 1 && (
            <div className="animate-in fade-in slide-in-from-bottom-4 duration-700">
              <div className="mb-8 space-y-2">
                <h2 className="text-4xl font-extrabold tracking-tight">Initialize Your <span className="text-blue-500">Data Fabric.</span></h2>
                <p className="text-slate-400">The Data Fabric is a secure metadata container that defines your agent&apos;s private data layer for short and long term memory, we call it the fabric.</p>
              </div>
              <div className="bg-white/5 border border-white/10 p-8 rounded-2xl space-y-6">
                <div className="space-y-2">
                  <Label className="text-xs uppercase font-bold text-slate-500 tracking-widest">Fabric Identifier</Label>
                  <Input 
                    type="text" 
                    value={formData.walletName}
                    onChange={(e) => setFormData(prev => ({ ...prev, walletName: e.target.value }))}
                    placeholder="e.g. ALPHA-STRATEGIC-FABRIC"
                    className="w-full bg-black/40 border-white/10 rounded-xl px-4 py-4 text-xl font-bold focus:ring-2 focus:ring-blue-500 focus:outline-none transition-interactive placeholder:text-slate-700 h-auto"
                  />
                </div>
                <div className="p-4 bg-blue-500/5 border border-blue-500/20 rounded-lg flex items-start space-x-4">
                  <Shield className="text-blue-500 shrink-0 mt-1" size={20} />
                  <p className="text-sm text-slate-400 leading-relaxed">
                    This identifier will be used to anchor your data container's <span className="text-blue-400 font-bold">Kernel Guardrails</span>. Once initialized, the container geometry is encrypted and stored in the Nexus as a living data fabric.
                  </p>
                </div>
              </div>
            </div>
          )}

          {step === 2 && (
            <div className="animate-in fade-in slide-in-from-bottom-4 duration-700">
              <div className="mb-8 space-y-2">
                <h2 className="text-4xl font-extrabold tracking-tight">Map Your <span className="text-blue-500">Fabric Inputs.</span></h2>
                <p className="text-slate-400">Mount the tools and credentials your agents require for autonomous operation.</p>
              </div>
              <div className="grid md:grid-cols-2 gap-4">
                {fabricInputs.map((item) => {
                  const Icon = item.icon;
                  const isComplete = isConnectionComplete(item.id);
                  return (
                    <div 
                      key={item.id} 
                      onClick={() => openModal(item.id)}
                      className={`group cursor-pointer p-6 rounded-2xl transition-interactive ${
                        isComplete 
                          ? 'bg-green-500/10 border border-green-500' 
                          : 'bg-white/5 border border-white/10 hover:border-blue-500/50 hover:bg-white/10'
                      }`}
                    >
                      <div className="flex justify-between items-start mb-4">
                        <div className={`p-3 rounded-lg transition-colors ${
                          isComplete ? 'bg-green-600/20' : 'bg-blue-600/10 group-hover:bg-blue-600/20'
                        }`}>
                          <Icon className={isComplete ? 'text-green-500' : 'text-blue-500'} size={24} />
                        </div>
                        <div className={`w-5 h-5 border rounded-full flex items-center justify-center transition-interactive ${
                          isComplete 
                            ? 'border-green-500 bg-green-500' 
                            : 'border-white/20 group-hover:border-blue-500'
                        }`}>
                          {isComplete ? (
                            <CheckCircle2 size={14} className="text-white" />
                          ) : (
                            <Settings size={14} className="text-blue-500" />
                          )}
                        </div>
                      </div>
                      <h4 className="font-bold mb-1">{item.label}</h4>
                      <p className="text-xs text-slate-500">{item.desc}</p>
                      {isComplete && (
                        <p className="text-xs text-green-400 mt-2 font-medium">Configured</p>
                      )}
                    </div>
                  );
                })}
              </div>
              
              {/* Knowledge Repository Ingestion */}
              <div className="mt-6 p-4 bg-white/5 border border-white/10 rounded-xl">
                <h3 className="text-sm font-bold text-white mb-3 flex items-center gap-2">
                  <Database className="w-4 h-4 text-blue-500" />
                  Knowledge Repository Ingestion
                </h3>
                <p className="text-xs text-slate-400 mb-4">
                  Import Git repositories or documents into your knowledge graph for context-aware reasoning.
                </p>
                <div className="flex gap-2">
                  <Input 
                    type="text"
                    value={ingestUrl}
                    onChange={(e) => setIngestUrl(e.target.value)}
                    onKeyDown={(e) => { if (e.key === 'Enter') handleIngestRepo(); }}
                    placeholder="https://github.com/org/repo"
                    className="flex-1 bg-black/40 border-white/10 text-white text-sm font-mono"
                  />
                  <Button 
                    onClick={handleIngestRepo} 
                    disabled={isIngesting || !ingestUrl.trim()}
                    className="bg-blue-600 hover:bg-blue-500"
                  >
                    {isIngesting ? <Loader2 className="w-4 h-4 animate-spin" /> : 'Ingest'}
                  </Button>
                </div>
                {ingestLog.length > 0 && (
                  <div className="mt-3 bg-black/40 rounded p-2 font-mono text-[10px] text-green-400 max-h-20 overflow-y-auto">
                    {ingestLog.map((line, i) => <div key={i}>{line}</div>)}
                  </div>
                )}
                {formData.connectionData.ingestedRepos.length > 0 && (
                  <div className="mt-3 flex flex-wrap gap-2">
                    {formData.connectionData.ingestedRepos.map((repo, i) => (
                      <Badge key={i} variant="outline" className="border-blue-500/30 text-blue-400 text-xs">
                        {repo.split('/').slice(-2).join('/')}
                      </Badge>
                    ))}
                  </div>
                )}
              </div>
              
              {/* Completion Progress */}
              <div className="mt-6 p-4 bg-white/5 border border-white/10 rounded-xl">
                <div className="flex items-center justify-between mb-2">
                  <span className="text-xs uppercase font-bold text-slate-500 tracking-wider">Configuration Progress</span>
                  <span className="text-sm text-slate-400">
                    {formData.completedConnections.length} of {fabricInputs.length} completed
                  </span>
                </div>
                <div className="h-2 bg-white/5 rounded-full overflow-hidden">
                  <div 
                    className="h-full bg-blue-600 transition-all duration-500"
                    style={{ width: `${(formData.completedConnections.length / fabricInputs.length) * 100}%` }}
                  />
                </div>
              </div>
            </div>
          )}

          {step === 3 && (
            <div className="animate-in fade-in slide-in-from-bottom-4 duration-700">
              <div className="mb-8 space-y-2">
                <h2 className="text-4xl font-extrabold tracking-tight">Configure <span className="text-blue-500">Kernel Guardrails.</span></h2>
                <p className="text-slate-400">Set the automated kill-switches that govern agent behavior at the system level.</p>
              </div>
              <div className="space-y-4">
                {guardrailPolicies.map((policy, idx) => {
                  const isEnabled = formData.guardrails[policy.id as keyof typeof formData.guardrails] ?? policy.defaultEnabled;
                  return (
                    <div key={policy.id} className="flex items-center justify-between p-6 bg-white/5 border border-white/10 rounded-2xl">
                      <div className="space-y-1">
                        <span className="text-[10px] uppercase font-bold text-blue-500 tracking-tighter">Policy 0{idx + 1}</span>
                        <h4 className="font-bold text-lg">{policy.label}</h4>
                        <p className="text-sm text-slate-500 italic">{policy.value}</p>
                      </div>
                      <div className="flex items-center space-x-4">
                        <div className="h-12 w-[1px] bg-white/10 mx-4" />
                        <button
                          onClick={() => toggleGuardrail(policy.id)}
                          className={`text-xs mono font-bold px-3 py-1 rounded border transition-interactive ${
                            isEnabled
                              ? 'text-green-500 bg-green-500/10 border-green-500/20'
                              : 'text-slate-500 bg-slate-500/10 border-slate-500/20'
                          }`}
                        >
                          {isEnabled ? 'ENABLED' : 'DISABLED'}
                        </button>
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          )}

          {step === 4 && (
            <div className="animate-in fade-in slide-in-from-bottom-4 duration-700">
              {/* Success Message */}
              <div className="text-center mb-8">
                <div className="inline-flex items-center justify-center p-3 bg-green-500/10 rounded-full mb-4">
                  <CheckCircle2 className="text-green-500" size={32} />
                </div>
                <h2 className="text-3xl md:text-4xl font-extrabold tracking-tight mb-3">
                  Welcome to Your <span className="text-blue-500">Data Fabric.</span>
                </h2>
                <p className="text-slate-400 max-w-2xl mx-auto">
                  Your private cloud cortex is ready. Download the mobile app to complete your setup and secure your vault.
                </p>
              </div>

              <div className="grid md:grid-cols-2 gap-8 items-start">
                {/* Left Column - Instructions */}
                <div className="space-y-4">
                  {/* Email Confirmation Notice */}
                  <div className="p-5 bg-green-500/5 border border-green-500/20 rounded-xl">
                    <div className="flex items-start space-x-3">
                      <Mail className="text-green-500 shrink-0 mt-1" size={20} />
                      <div>
                        <h3 className="font-bold mb-1 text-green-400">Configuration Complete</h3>
                        <p className="text-sm text-slate-400">
                          Your Data Fabric has been configured. Download the mobile wallet 
                          to complete your setup and manage your vault on the go.
                        </p>
                      </div>
                    </div>
                  </div>

                  {/* Download Options */}
                  <div className="p-5 bg-white/5 border border-white/10 rounded-xl">
                    <h3 className="font-bold mb-3 flex items-center text-sm">
                      <Download className="mr-2 text-blue-500" size={16} />
                      Download Options
                    </h3>
                    
                    <div className="space-y-2">
                      <button
                        onClick={() => window.open('https://beta-controller.knirv.com/', '_blank')}
                        className="w-full flex items-center justify-between p-3 bg-white/5 border border-white/10 rounded-lg hover:border-blue-500/50 hover:bg-white/5 transition-colors text-left"
                      >
                        <div className="flex items-center">
                          <Smartphone className="mr-3 text-blue-500" size={18} />
                          <div>
                            <div className="font-bold text-sm">Open Live PWA</div>
                            <div className="text-xs text-slate-500">Install directly on your device</div>
                          </div>
                        </div>
                        <ChevronRight size={16} className="text-slate-500" />
                      </button>

                      <div className="grid grid-cols-2 gap-2">
                        <button
                          onClick={() => window.open('https://releases.knirv.network/knirvcontroller-ios-pwa.zip', '_blank')}
                          className="p-3 bg-white/5 border border-white/10 rounded-lg hover:border-blue-500/50 hover:bg-white/5 transition-colors text-center text-sm"
                        >
                          <Download size={14} className="inline mr-1" />
                          iOS ZIP
                        </button>
                        <button
                          onClick={() => window.open('https://releases.knirv.network/knirvcontroller-android-pwa.zip', '_blank')}
                          className="p-3 bg-white/5 border border-white/10 rounded-lg hover:border-blue-500/50 hover:bg-white/5 transition-colors text-center text-sm"
                        >
                          <Download size={14} className="inline mr-1" />
                          Android
                        </button>
                      </div>
                    </div>
                  </div>

                  {/* Setup Steps */}
                  <div className="p-5 bg-white/5 border border-white/10 rounded-xl">
                    <h3 className="font-bold mb-3 text-sm">Setup Steps</h3>
                    <div className="space-y-3">
                      {[
                        { step: 1, text: 'Download the KNIRV Mobile Wallet app' },
                        { step: 2, text: 'Install and open the application' },
                        { step: 3, text: 'Scan the QR code to pair your device' },
                        { step: 4, text: 'Complete biometric authorization' }
                      ].map((item) => (
                        <div key={item.step} className="flex items-center space-x-3">
                          <div className="w-6 h-6 rounded-full bg-blue-600 flex items-center justify-center text-xs font-bold shrink-0">
                            {item.step}
                          </div>
                          <span className="text-sm text-slate-300">{item.text}</span>
                        </div>
                      ))}
                    </div>
                  </div>
                </div>

                {/* Right Column - QR Code */}
                <div className="flex flex-col items-center">
                  <div className="bg-white/5 border border-white/10 p-6 rounded-xl w-full">
                    <div className="flex items-center justify-between mb-4">
                      <h3 className="font-bold flex items-center text-sm">
                        <Shield className="mr-2 text-blue-500" size={16} />
                        Secure Download
                      </h3>
                      <span className="text-xs bg-blue-500/20 text-blue-400 px-2 py-1 rounded border border-blue-500/30">
                        Valid for 15 min
                      </span>
                    </div>

                    {/* QR Code */}
                    <div className="flex flex-col items-center space-y-3 mb-4">
                      <div className="p-3 bg-white rounded-lg shadow-[0_0_30px_rgba(59,130,246,0.2)]">
                        <div className="w-40 h-40 bg-slate-100 flex items-center justify-center">
                          <span className="text-xs text-slate-400 text-center px-4">
                            QR Code will be generated
                          </span>
                        </div>
                      </div>
                      <p className="text-xs text-slate-400 text-center">
                        Scan with your mobile device to download
                      </p>
                    </div>

                    {/* Connection Info */}
                    <div className="pt-4 border-t border-white/10">
                      <div className="text-xs text-slate-500 space-y-1 font-mono text-center">
                        <div>Session: <span className="text-slate-400">FABRIC-{Math.random().toString(36).substr(2, 6).toUpperCase()}</span></div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Navigation Actions */}
        <div className="mt-16 flex justify-between items-center border-t border-white/5 pt-8">
          <Button 
            variant="ghost"
            onClick={handleBack}
            disabled={step === 1}
            className={`text-slate-500 hover:text-white transition-colors flex items-center space-x-2 text-sm font-bold uppercase tracking-widest ${step === 1 ? 'opacity-0 pointer-events-none' : ''}`}
          >
            <ChevronLeft size={16} />
            <span>Back</span>
          </Button>
          
          <Button 
            onClick={handleNext}
            disabled={isSubmitting}
            className="group bg-blue-600 hover:bg-blue-500 text-white px-8 py-4 rounded-xl font-bold transition-interactive transform active:scale-95 flex items-center space-x-3 h-auto disabled:opacity-50"
          >
            {isSubmitting ? (
              <>
                <Loader2 size={18} className="animate-spin" />
                <span>Setting up Guardrails...</span>
              </>
            ) : (
              <>
                <span>{step === 4 ? 'Complete Setup' : 'Continue Configuration'}</span>
                <ChevronRight size={18} className="group-hover:translate-x-1 transition-transform" />
              </>
            )}
          </Button>
        </div>

      </main>

      {/* Footer Meta */}
      <footer className="max-w-7xl mx-auto p-12 text-center">
        <div className="inline-flex items-center space-x-4 px-6 py-2 rounded-full border border-white/5 bg-white/5 text-[10px] mono text-slate-500 font-bold uppercase tracking-[0.2em]">
          <span className="flex h-2 w-2 rounded-full bg-green-500 animate-pulse" />
          <span>Nexus Network Secure</span>
          <span className="h-3 w-[1px] bg-white/10" />
          <span>Sovereign Encryption Active</span>
        </div>
      </footer>

      {/* Connection Modals */}
      <APIKeysModal
        isOpen={activeModal === 'api-keys'}
        onClose={closeModal}
        onSave={handleSaveAPIKeys}
        initialEntries={formData.connectionData.apiKeys}
      />
      <MCPServersModal
        isOpen={activeModal === 'mcp-servers'}
        onClose={closeModal}
        onSave={handleSaveMCPServers}
        initialServers={formData.connectionData.mcpServers}
      />
      <PolicyCertsModal
        isOpen={activeModal === 'policy-certs'}
        onClose={closeModal}
        onSave={handleSavePolicyCerts}
        initialCerts={formData.connectionData.policyCerts}
        initialRules={formData.connectionData.customRules}
      />
      <PreferencesModal
        isOpen={activeModal === 'preferences'}
        onClose={closeModal}
        onSave={handleSavePreferences}
        initialSettings={formData.privacySettings}
      />
    </div>
  );
};

export default OnboardingGuide;
