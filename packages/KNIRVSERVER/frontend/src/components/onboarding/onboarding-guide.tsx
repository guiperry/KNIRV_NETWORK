'use client';

import React, { useState } from 'react';
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { 
  Shield, 
  Key, 
  Database, 
  Terminal,
  ChevronRight,
  ChevronLeft,
  CheckCircle2,
  Smartphone,
  Download,
  Settings,
  Home,
  SlidersHorizontal,
  Loader2
} from "lucide-react";
import { APIKeysModal, type APIKeyEntry } from "./modals/APIKeysModal";
import { MCPServersModal, type MCPServerEntry } from "./modals/MCPServersModal";
import { PolicyCertsModal, type PolicyCert, type CustomRule } from "./modals/PolicyCertsModal";
import { PreferencesModal, type PrivacySettings } from "./modals/PreferencesModal";
import { DatabaseConfigModal, type DatabaseConfig } from "./modals/DatabaseConfigModal";
import { KnowledgeIngestModal } from "./modals/KnowledgeIngestModal";
import { ControllerDownloadModal } from "./modals/ControllerDownloadModal";
import QRCodeDisplay from "@/components/controller/qr-code-display";
import { useAuth } from '@/lib/auth-context';
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
    databaseConfig: DatabaseConfig;
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
  databaseConfig: DatabaseConfig;
}

type ConfigCard = {
  id: string;
  icon: React.ComponentType<{ size?: number; className?: string }>;
  label: string;
  desc: string;
  modalId: string;
};

type DownloadPlatform = 'android' | 'ios';

const governanceCards: ConfigCard[] = [
  { id: 'policy-certs', icon: Database, label: 'Policy Certs', desc: 'Kernel Guardrails & Custom Rules', modalId: 'policy-certs' },
  { id: 'preferences', icon: SlidersHorizontal, label: 'Preferences', desc: 'Data Management & Privacy Settings', modalId: 'preferences' }
];

const connectionCards: ConfigCard[] = [
  { id: 'api-keys', icon: Key, label: 'API Keys', desc: 'Secure LLM & Service Credentials', modalId: 'api-keys' },
  { id: 'mcp-servers', icon: Terminal, label: 'MCP Servers', desc: 'Model Context Protocol Integrations', modalId: 'mcp-servers' }
];

const dataCards: ConfigCard[] = [
  { id: 'knowledge-ingest', icon: Database, label: 'Knowledge Ingest', desc: 'Import repositories and documents to graph', modalId: 'knowledge-ingest' },
  { id: 'database-config', icon: Database, label: 'Database Config', desc: 'Confirm KNIRVBASE schema and external MCP mapping', modalId: 'database-config' }
];

const controllerDownloadUrls: Record<DownloadPlatform, string> = {
  android:
    process.env.NEXT_PUBLIC_KNIRVCONTROLLER_ANDROID_DOWNLOAD_URL ||
    'https://YOUR-CLOUDFLARE-ENDPOINT.example/knirvcontroller-android.apk',
  ios:
    process.env.NEXT_PUBLIC_KNIRVCONTROLLER_IOS_DOWNLOAD_URL ||
    'https://YOUR-CLOUDFLARE-ENDPOINT.example/knirvcontroller-ios.ipa',
};

const defaultDatabaseConfig: DatabaseConfig = {
  internalSchemaName: 'KNIRVBASE',
  internalSchemaVersion: 'v1',
  internalTables: [
    'users',
    'sessions',
    'memory_entries',
    'knowledge_graph_nodes',
    'knowledge_graph_edges',
    'policy_rules',
    'audit_events'
  ],
  externalMcpDatabase: {
    enabled: false,
    name: '',
    provider: '',
    selectedServerId: '',
    saveProfile: false,
    saveMemory: true,
    saveKnowledgeGraph: true,
    savePolicies: true,
    saveAuditTrail: true,
  },
  notes: 'KNIRVBASE keeps the canonical internal schema; external MCP databases receive only the selected mirrored data classes.',
};

const OnboardingGuide = ({ onComplete, onReset }: OnboardingGuideProps) => {
  const { user } = useAuth();
  const [step, setStep] = useState(1);
  const [progress, setProgress] = useState(25);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [ingestUrl, setIngestUrl] = useState('');
  const [isIngesting, setIsIngesting] = useState(false);
  const [ingestLog, setIngestLog] = useState<string[]>([]);
  const [isQrModalOpen, setIsQrModalOpen] = useState(false);
  const [downloadModalPlatform, setDownloadModalPlatform] = useState<DownloadPlatform | null>(null);
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
    },
    databaseConfig: defaultDatabaseConfig
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
          selectedInputs: prev.selectedInputs.includes('knowledge-ingest')
            ? prev.selectedInputs
            : [...prev.selectedInputs, 'knowledge-ingest'],
          connectionData: {
            ...prev.connectionData,
            ingestedRepos: [...prev.connectionData.ingestedRepos, ingestUrl.trim()]
          },
          completedConnections: prev.completedConnections.includes('knowledge-ingest')
            ? prev.completedConnections
            : [...prev.completedConnections, 'knowledge-ingest']
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

      const response = await fetch(`${API_BASE_URL}/api/guardrail/policies`, {
        method: 'POST',
        headers: { ...getAuthHeaders(), 'Content-Type': 'application/json' },
        body: JSON.stringify(policy),
      });

      if (response.ok) {
        const data = await response.json();
        if (data.policy?.id) {
          await fetch(`${API_BASE_URL}/api/guardrail/policies/${data.policy.id}/commit`, {
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
    { id: 1, title: 'Sovereignty', description: 'Secure the Vault' },
    { id: 2, title: 'Governance', description: 'Set Kernel Guardrails' },
    { id: 3, title: 'Connections', description: 'Map Fabric Inputs' },
    { id: 4, title: 'Data', description: 'Initialize Data Fabric' }
  ];

  const submitOrganizationContext = async (walletName: string) => {
    try {
      const guidelines = formData.connectionData.customRules.map(r => r.description || r.name || '').filter(Boolean);
      const statedValues = formData.selectedInputs;

      const orgPayload = {
        organization_id: walletName,
        name: walletName,
        metadata: {
          database_config: formData.databaseConfig,
        },
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
          privacySettings: formData.privacySettings,
          databaseConfig: formData.databaseConfig
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

  const openModal = (modalId: string) => {
    setActiveModal(modalId);
  };

  const closeModal = () => {
    setActiveModal(null);
  };

  const isConfigured = (configId: string) => {
    switch (configId) {
      case 'api-keys':
        return formData.connectionData.apiKeys.length > 0;
      case 'mcp-servers':
        return formData.connectionData.mcpServers.length > 0;
      case 'policy-certs':
        return formData.connectionData.policyCerts.length > 0 || formData.connectionData.customRules.length > 0;
      case 'preferences':
        return formData.completedConnections.includes('preferences');
      case 'knowledge-ingest':
        return formData.connectionData.ingestedRepos.length > 0;
      case 'database-config':
        return formData.completedConnections.includes('database-config');
      default:
        return false;
    }
  };

  const handleSaveAPIKeys = (apiKeys: APIKeyEntry[]) => {
    setFormData(prev => ({
      ...prev,
      connectionData: { ...prev.connectionData, apiKeys },
      selectedInputs: prev.selectedInputs.includes('api-keys')
        ? prev.selectedInputs
        : [...prev.selectedInputs, 'api-keys'],
      completedConnections: prev.completedConnections.includes('api-keys')
        ? prev.completedConnections
        : [...prev.completedConnections, 'api-keys']
    }));
  };

  const handleSaveMCPServers = (mcpServers: MCPServerEntry[]) => {
    setFormData(prev => ({
      ...prev,
      connectionData: { ...prev.connectionData, mcpServers },
      selectedInputs: prev.selectedInputs.includes('mcp-servers')
        ? prev.selectedInputs
        : [...prev.selectedInputs, 'mcp-servers'],
      completedConnections: prev.completedConnections.includes('mcp-servers')
        ? prev.completedConnections
        : [...prev.completedConnections, 'mcp-servers']
    }));
  };

  const handleSavePolicyCerts = (policyCerts: PolicyCert[], customRules: CustomRule[]) => {
    setFormData(prev => ({
      ...prev,
      connectionData: { ...prev.connectionData, policyCerts, customRules },
      selectedInputs: prev.selectedInputs.includes('policy-certs')
        ? prev.selectedInputs
        : [...prev.selectedInputs, 'policy-certs'],
      completedConnections: prev.completedConnections.includes('policy-certs')
        ? prev.completedConnections
        : [...prev.completedConnections, 'policy-certs']
    }));
  };

  const handleSavePreferences = (privacySettings: PrivacySettings) => {
    setFormData(prev => ({
      ...prev,
      privacySettings,
      selectedInputs: prev.selectedInputs.includes('preferences')
        ? prev.selectedInputs
        : [...prev.selectedInputs, 'preferences'],
      completedConnections: prev.completedConnections.includes('preferences')
        ? prev.completedConnections
        : [...prev.completedConnections, 'preferences']
    }));
  };

  const handleSaveDatabaseConfig = (databaseConfig: DatabaseConfig) => {
    setFormData(prev => ({
      ...prev,
      databaseConfig,
      selectedInputs: prev.selectedInputs.includes('database-config')
        ? prev.selectedInputs
        : [...prev.selectedInputs, 'database-config'],
      completedConnections: prev.completedConnections.includes('database-config')
        ? prev.completedConnections
        : [...prev.completedConnections, 'database-config']
    }));
  };

  const handleOpenQRModal = () => {
    setIsQrModalOpen(true);
  };

  const openDownloadModal = (platform: DownloadPlatform) => {
    setDownloadModalPlatform(platform);
  };

  const renderConfigCard = (item: ConfigCard) => {
    const Icon = item.icon;
    const isComplete = isConfigured(item.id);

    return (
      <div
        key={item.id}
        onClick={() => openModal(item.modalId)}
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

        {submitError && (
          <div className="mb-6 rounded-xl border border-red-500/30 bg-red-500/10 px-4 py-3 text-sm text-red-200">
            {submitError}
          </div>
        )}

        {/* Dynamic Content Area */}
        <div className="min-h-[400px]">
          {step === 1 && (
            <div className="animate-in fade-in slide-in-from-bottom-4 duration-700">
              <div className="mb-8 space-y-2">
                <h2 className="text-4xl font-extrabold tracking-tight">Secure Your <span className="text-blue-500">Sovereignty.</span></h2>
                <p className="text-slate-400">Generate the real KNIRVCONTROLLER pairing QR code and complete device trust before the rest of the fabric comes online.</p>
              </div>
              <div className="grid md:grid-cols-[1.2fr_0.8fr] gap-6">
                <div className="space-y-4">
                  <div className="p-5 bg-white/5 border border-white/10 rounded-xl">
                    <h3 className="font-bold mb-3 flex items-center text-sm">
                      <Download className="mr-2 text-blue-500" size={16} />
                      Download KNIRVCONTROLLER
                    </h3>
                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                      <button
                        onClick={() => openDownloadModal('android')}
                        className="w-full flex items-center justify-between p-3 bg-white/5 border border-white/10 rounded-lg hover:border-blue-500/50 hover:bg-white/5 transition-colors text-left"
                      >
                        <div className="flex items-center">
                          <Smartphone className="mr-3 text-blue-500" size={18} />
                          <div>
                            <div className="font-bold text-sm">Android</div>
                            <div className="text-xs text-slate-500">Scan to download the APK</div>
                          </div>
                        </div>
                        <ChevronRight size={16} className="text-slate-500" />
                      </button>
                      <button
                        onClick={() => openDownloadModal('ios')}
                        className="w-full flex items-center justify-between p-3 bg-white/5 border border-white/10 rounded-lg hover:border-blue-500/50 hover:bg-white/5 transition-colors text-left"
                      >
                        <div className="flex items-center">
                          <Smartphone className="mr-3 text-blue-500" size={18} />
                          <div>
                            <div className="font-bold text-sm">iOS</div>
                            <div className="text-xs text-slate-500">Scan to download the IPA</div>
                          </div>
                        </div>
                        <ChevronRight size={16} className="text-slate-500" />
                      </button>
                    </div>
                  </div>

                  <div className="bg-white/5 border border-white/10 p-8 rounded-2xl space-y-6">
                    <div className="p-4 bg-blue-500/5 border border-blue-500/20 rounded-lg flex items-start space-x-4">
                      <Shield className="text-blue-500 shrink-0 mt-1" size={20} />
                      <p className="text-sm text-slate-400 leading-relaxed">
                        This pairing flow uses the controller integration service to mint a signed QR payload that the KNIRVCONTROLLER can scan and confirm.
                      </p>
                    </div>
                    <div className="space-y-3">
                      <Label className="text-xs uppercase font-bold text-slate-500 tracking-widest">Controller Pairing ID</Label>
                      <Input
                        type="text"
                        value={user?.user || formData.walletName || 'knirv-controller'}
                        readOnly
                        className="w-full bg-black/40 border-white/10 rounded-xl px-4 py-4 text-lg font-bold focus:ring-2 focus:ring-blue-500 focus:outline-none transition-interactive placeholder:text-slate-700 h-auto"
                      />
                    </div>
                    <Button
                      onClick={handleOpenQRModal}
                      className="w-full bg-blue-600 hover:bg-blue-500 text-white rounded-xl py-6 font-bold"
                    >
                      <Smartphone className="mr-2" size={18} />
                      Generate KNIRVCONTROLLER QR Code
                    </Button>
                  </div>
                </div>

                <div className="space-y-4">
                  <div className="p-5 bg-white/5 border border-white/10 rounded-xl">
                    <h3 className="font-bold mb-3 text-sm">Pairing Steps</h3>
                    <div className="space-y-3">
                      {[
                        { step: 1, text: 'Download the KNIRVCONTROLLER app for Android or iOS' },
                        { step: 2, text: 'Install and open the application on your phone' },
                        { step: 3, text: 'Scan the pairing QR code from the controller modal' },
                        { step: 4, text: 'Approve the pairing request and continue onboarding' }
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

                  <div className="p-5 bg-blue-500/5 border border-blue-500/20 rounded-xl">
                    <h3 className="font-bold mb-2 text-sm text-blue-300">Session Note</h3>
                    <p className="text-sm text-slate-400 leading-relaxed">
                      The QR generator uses the current authenticated user when available. If no session is active, it falls back to the KNIRV controller identifier.
                    </p>
                  </div>
                </div>
              </div>
            </div>
          )}

          {step === 2 && (
            <div className="animate-in fade-in slide-in-from-bottom-4 duration-700">
              <div className="mb-8 space-y-2">
                <h2 className="text-4xl font-extrabold tracking-tight">Configure <span className="text-blue-500">Kernel Guardrails.</span></h2>
                <p className="text-slate-400">Policy and privacy controls now live here. The old guardrail toggles were removed in favor of the modal workflow.</p>
              </div>
              <div className="grid md:grid-cols-2 gap-4">
                {governanceCards.map(renderConfigCard)}
              </div>
            </div>
          )}

          {step === 3 && (
            <div className="animate-in fade-in slide-in-from-bottom-4 duration-700">
              <div className="mb-8 space-y-2">
                <h2 className="text-4xl font-extrabold tracking-tight">Map Your <span className="text-blue-500">Fabric Inputs.</span></h2>
                <p className="text-slate-400">Mount the tools and credentials your agents require for autonomous operation.</p>
              </div>
              <div className="grid md:grid-cols-2 gap-4">
                {connectionCards.map(renderConfigCard)}
              </div>
            </div>
          )}

          {step === 4 && (
            <div className="animate-in fade-in slide-in-from-bottom-4 duration-700">
              <div className="mb-8 space-y-2">
                <h2 className="text-4xl font-extrabold tracking-tight">Initialize Your <span className="text-blue-500">Data Fabric.</span></h2>
                <p className="text-slate-400">The Data phase now owns the fabric identity, repository ingestion, and database schema controls.</p>
              </div>
              <div className="grid gap-6">
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
                      This identifier anchors the internal KNIRVBASE record and the metadata that your selected external MCP database mirrors.
                    </p>
                  </div>
                </div>

                <div className="grid md:grid-cols-2 gap-4">
                  {dataCards.map(renderConfigCard)}
                </div>

                <div className="p-5 bg-white/5 border border-white/10 rounded-xl">
                  <h3 className="font-bold mb-3 text-sm">Data Routing Summary</h3>
                  <div className="grid md:grid-cols-2 gap-3 text-sm text-slate-400">
                    <div className="p-4 rounded-lg bg-black/30 border border-white/5">
                      <div className="text-xs uppercase text-blue-400 font-bold mb-1">Internal KNIRVBASE</div>
                      <div>{formData.databaseConfig.internalSchemaName} {formData.databaseConfig.internalSchemaVersion}</div>
                      <div className="text-xs text-slate-500 mt-1">{formData.databaseConfig.internalTables.length} tables tracked</div>
                    </div>
                    <div className="p-4 rounded-lg bg-black/30 border border-white/5">
                      <div className="text-xs uppercase text-blue-400 font-bold mb-1">External MCP Mirror</div>
                      <div>{formData.databaseConfig.externalMcpDatabase.enabled ? 'Enabled' : 'Disabled'}</div>
                      <div className="text-xs text-slate-500 mt-1">
                        {formData.connectionData.ingestedRepos.length} repositories queued for graph ingestion
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
                <span>Finalizing Setup...</span>
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
      <KnowledgeIngestModal
        isOpen={activeModal === 'knowledge-ingest'}
        onClose={closeModal}
        ingestUrl={ingestUrl}
        setIngestUrl={setIngestUrl}
        ingestLog={ingestLog}
        isIngesting={isIngesting}
        onIngest={handleIngestRepo}
        ingestedRepos={formData.connectionData.ingestedRepos}
      />
      <DatabaseConfigModal
        isOpen={activeModal === 'database-config'}
        onClose={closeModal}
        onSave={handleSaveDatabaseConfig}
        initialConfig={formData.databaseConfig}
      />
      <QRCodeDisplay
        isOpen={isQrModalOpen}
        onClose={() => setIsQrModalOpen(false)}
        userId={user?.user || formData.walletName || 'knirv-controller'}
        deviceType="desktop"
        capabilities={['remote_control', 'file_transfer', 'screen_share']}
      />
      <ControllerDownloadModal
        isOpen={downloadModalPlatform !== null}
        onClose={() => setDownloadModalPlatform(null)}
        platform={downloadModalPlatform ?? 'android'}
        downloadUrl={downloadModalPlatform ? controllerDownloadUrls[downloadModalPlatform] : controllerDownloadUrls.android}
      />
    </div>
  );
};

export default OnboardingGuide;
