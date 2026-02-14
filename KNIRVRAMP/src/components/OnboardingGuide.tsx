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
  Cpu, 
  Terminal,
  ChevronRight,
  ChevronLeft,
  CheckCircle2,
  Plus,
  Lock,
  Smartphone,
  Download
} from "lucide-react";

interface OnboardingGuideProps {
  onComplete: () => void;
}

interface FormData {
  walletName: string;
  selectedInputs: string[];
  guardrails: {
    networkDrift: boolean;
    filesystemAccess: boolean;
    computeCostCap: boolean;
  };
}

const fabricInputs = [
  { id: 'api-keys', icon: Key, label: 'API Keys', desc: 'Secure LLM & Service Credentials' },
  { id: 'mcp-servers', icon: Terminal, label: 'MCP Servers', desc: 'Model Context Protocol Integrations' },
  { id: 'custom-certs', icon: Database, label: 'Custom Certs', desc: 'Private TLS/Identity Certificates' },
  { id: 'asic-nodes', icon: Cpu, label: 'ASIC Nodes', desc: 'Custom Hardware Compute IDs' }
];

const guardrailPolicies = [
  { id: 'network-drift', label: 'Outbound Network Drift', value: 'High Accuracy (Strict)', defaultEnabled: true },
  { id: 'filesystem-access', label: 'Filesystem Access Depth', value: 'Restricted to /mnt/nexus', defaultEnabled: true },
  { id: 'compute-cost', label: 'Compute Cost Cap', value: '$100.00 / Session', defaultEnabled: true }
];

const OnboardingGuide = ({ onComplete }: OnboardingGuideProps) => {
  const [step, setStep] = useState(1);
  const [progress, setProgress] = useState(25);
  const [formData, setFormData] = useState<FormData>({
    walletName: '',
    selectedInputs: [],
    guardrails: {
      networkDrift: true,
      filesystemAccess: true,
      computeCostCap: true
    }
  });

  const steps = [
    { id: 1, title: 'Identity', description: 'Initialize Data Fabric' },
    { id: 2, title: 'Connections', description: 'Map Fabric Inputs' },
    { id: 3, title: 'Governance', description: 'Set Kernel Guardrails' },
    { id: 4, title: 'Sovereignty', description: 'Secure the Vault' }
  ];

  const handleNext = () => {
    if (step < 4) {
      setStep(step + 1);
      setProgress((step + 1) * 25);
    } else {
      onComplete();
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
    <div className="min-h-screen bg-[#0a0a0c] text-slate-200 font-sans selection:bg-blue-500/30">
      {/* Background Sentinel Mesh */}
      <div className="fixed inset-0 overflow-hidden pointer-events-none opacity-20">
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
        </div>
      </nav>

      <main className="relative z-10 max-w-4xl mx-auto px-6 py-12 md:py-16">
        
        {/* Step Progress Display */}
        <div className="flex justify-between mb-12 relative">
          <div className="absolute top-1/2 left-0 w-full h-[1px] bg-white/5 -z-10" />
          {steps.map((s) => (
            <div key={s.id} className="flex flex-col items-center group">
              <div className={`w-8 h-8 rounded-full flex items-center justify-center border transition-all duration-500 ${
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
                    className="w-full bg-black/40 border-white/10 rounded-xl px-4 py-4 text-xl font-bold focus:ring-2 focus:ring-blue-500 focus:outline-none transition-all placeholder:text-slate-700 h-auto"
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
                  const isSelected = formData.selectedInputs.includes(item.id);
                  return (
                    <div 
                      key={item.id} 
                      onClick={() => toggleInput(item.id)}
                      className={`group cursor-pointer p-6 rounded-2xl transition-all ${
                        isSelected 
                          ? 'bg-blue-500/10 border border-blue-500' 
                          : 'bg-white/5 border border-white/10 hover:border-blue-500/50 hover:bg-white/10'
                      }`}
                    >
                      <div className="flex justify-between items-start mb-4">
                        <div className={`p-3 rounded-lg transition-colors ${
                          isSelected ? 'bg-blue-600/20' : 'bg-blue-600/10 group-hover:bg-blue-600/20'
                        }`}>
                          <Icon className="text-blue-500" size={24} />
                        </div>
                        <div className={`w-5 h-5 border rounded-full flex items-center justify-center transition-all ${
                          isSelected 
                            ? 'border-blue-500 bg-blue-500' 
                            : 'border-white/20 group-hover:border-blue-500'
                        }`}>
                          {isSelected ? (
                            <CheckCircle2 size={14} className="text-white" />
                          ) : (
                            <Plus size={14} className="text-blue-500" />
                          )}
                        </div>
                      </div>
                      <h4 className="font-bold mb-1">{item.label}</h4>
                      <p className="text-xs text-slate-500">{item.desc}</p>
                    </div>
                  );
                })}
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
                          className={`text-xs mono font-bold px-3 py-1 rounded border transition-all ${
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
            <div className="animate-in fade-in slide-in-from-bottom-4 duration-700 text-center">
              <div className="mb-12 space-y-4">
                <div className="inline-block p-4 bg-blue-600/20 rounded-full mb-4 animate-pulse">
                  <Lock className="text-blue-500" size={48} />
                </div>
                <h2 className="text-4xl font-extrabold tracking-tight">Secure the <span className="text-blue-500">Vault.</span></h2>
                <p className="text-slate-400 max-w-lg mx-auto">
                  To finalize sovereignty, your Data Fabric must be anchored to your physical device with a Data Wallet. KNIRV does not store your private keys.
                </p>
              </div>

              <div className="grid md:grid-cols-2 gap-12 items-center">
                <div className="space-y-6 text-left">
                  <div className="p-6 bg-white/5 border border-white/10 rounded-2xl space-y-4">
                    <div className="flex items-center space-x-3 text-blue-400">
                      <Smartphone size={20} />
                      <h4 className="font-bold uppercase text-sm tracking-widest">Mobile Validation Required</h4>
                    </div>
                    <p className="text-sm text-slate-400">
                      1. Download the KNIRV Mobile Wallet<br />
                      2. Scan the secure handshake QR code<br />
                      3. Biometrically authorize your Nexus Vault
                    </p>
                    <div className="flex space-x-3 pt-4">
                      <button className="flex-1 bg-white/10 hover:bg-white/20 px-4 py-3 rounded-lg flex items-center justify-center space-x-2 transition-colors">
                        <Download size={16} />
                        <span className="text-xs font-bold">iOS</span>
                      </button>
                      <button className="flex-1 bg-white/10 hover:bg-white/20 px-4 py-3 rounded-lg flex items-center justify-center space-x-2 transition-colors">
                        <Download size={16} />
                        <span className="text-xs font-bold">Android</span>
                      </button>
                    </div>
                  </div>
                </div>

                <div className="flex flex-col items-center space-y-4">
                  <div className="p-4 bg-white rounded-xl shadow-[0_0_50px_rgba(59,130,246,0.2)]">
                    <div className="w-48 h-48 bg-slate-100 flex flex-col items-center justify-center border-2 border-dashed border-slate-300 rounded-lg">
                      <Shield className="text-slate-300 mb-2" size={32} />
                      <span className="text-[10px] text-slate-400 uppercase font-bold tracking-widest">Scan to Secure</span>
                    </div>
                  </div>
                  <span className="mono text-[10px] text-slate-600 font-bold">VAULT_HANDSHAKE_ID: NEX-882-991</span>
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
            className="group bg-blue-600 hover:bg-blue-500 text-white px-8 py-4 rounded-xl font-bold transition-all transform active:scale-95 flex items-center space-x-3 h-auto"
          >
            <span>{step === 4 ? 'Complete Setup' : 'Continue Configuration'}</span>
            <ChevronRight size={18} className="group-hover:translate-x-1 transition-transform" />
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
    </div>
  );
};

export default OnboardingGuide;
