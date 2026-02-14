'use client';

import React, { useState } from 'react';
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Slider } from "@/components/ui/slider";
import { 
  Shield, 
  Eye,
  Lock,
  Globe,
  Database,
  Activity,
  ChevronRight,
  ChevronLeft,
  CheckCircle2,
  AlertTriangle,
  Home
} from "lucide-react";
import { useToast } from "@/hooks/use-toast";

interface DataPreferencesFormProps {
  onSubmit: () => void;
  onBack: () => void;
  onReset?: () => void;
}

interface PrivacySettings {
  dataEncryption: boolean;
  localProcessing: boolean;
  anonymizeMetrics: boolean;
  shareErrorLogs: boolean;
  allowAnalytics: boolean;
  dataRetentionDays: number;
  autoDeleteInactive: boolean;
  thirdPartyIntegrations: boolean;
}

const DataPreferencesForm = ({ onSubmit, onBack, onReset }: DataPreferencesFormProps) => {
  const { toast } = useToast();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [settings, setSettings] = useState<PrivacySettings>({
    dataEncryption: true,
    localProcessing: true,
    anonymizeMetrics: true,
    shareErrorLogs: false,
    allowAnalytics: false,
    dataRetentionDays: 90,
    autoDeleteInactive: true,
    thirdPartyIntegrations: false
  });

  const handleToggle = (key: keyof PrivacySettings) => {
    setSettings(prev => ({
      ...prev,
      [key]: !prev[key]
    }));
  };

  const handleRetentionChange = (value: number[]) => {
    setSettings(prev => ({
      ...prev,
      dataRetentionDays: value[0]
    }));
  };

  const handleSubmit = async () => {
    setIsSubmitting(true);
    
    // Simulate API call to save preferences
    await new Promise(resolve => setTimeout(resolve, 1500));
    
    toast({
      title: "Preferences Saved",
      description: "Your privacy settings have been configured. A confirmation email has been sent.",
    });
    
    setIsSubmitting(false);
    onSubmit();
  };

  const preferenceCategories = [
    {
      title: "Data Security",
      icon: Lock,
      description: "Control how your data is protected",
      settings: [
        { key: 'dataEncryption', label: 'End-to-End Encryption', desc: 'Encrypt all data at rest and in transit' },
        { key: 'localProcessing', label: 'Local Processing First', desc: 'Process data locally before cloud sync' }
      ]
    },
    {
      title: "Privacy Controls",
      icon: Eye,
      description: "Manage data visibility and sharing",
      settings: [
        { key: 'anonymizeMetrics', label: 'Anonymize Usage Metrics', desc: 'Remove personally identifiable information' },
        { key: 'shareErrorLogs', label: 'Share Error Logs', desc: 'Help improve by sharing anonymized errors' }
      ]
    },
    {
      title: "Data Retention",
      icon: Database,
      description: "Manage how long your data is stored",
      settings: [
        { key: 'autoDeleteInactive', label: 'Auto-Delete Inactive Data', desc: 'Remove data from inactive sessions' }
      ]
    },
    {
      title: "Integrations",
      icon: Globe,
      description: "Control external service connections",
      settings: [
        { key: 'allowAnalytics', label: 'Usage Analytics', desc: 'Allow anonymous usage analytics collection' },
        { key: 'thirdPartyIntegrations', label: 'Third-Party Services', desc: 'Enable integrations with external services' }
      ]
    }
  ];

  return (
    <div className="min-h-screen bg-[#0a0a0c] text-slate-200 font-sans selection:bg-blue-500/30">
      {/* Background Effects */}
      <div className="fixed inset-0 overflow-hidden pointer-events-none opacity-20">
        <div className="absolute top-[-10%] left-[-10%] w-[40%] h-[40%] bg-blue-600/20 blur-[120px] rounded-full" />
        <div className="absolute bottom-[-10%] right-[-10%] w-[40%] h-[40%] bg-indigo-600/10 blur-[120px] rounded-full" />
      </div>

      {/* Header */}
      <nav className="relative z-10 p-6 flex justify-between items-center border-b border-white/5 bg-black/40 backdrop-blur-md">
        <div className="flex items-center space-x-2">
          <div className="w-6 h-6 bg-blue-600 rounded-sm transform rotate-45" />
          <span className="text-xl font-extrabold tracking-tighter uppercase">KNIRV <span className="text-blue-500 font-light italic">PRIVACY</span></span>
        </div>
        <div className="flex items-center space-x-4">
          <span className="text-[10px] mono text-slate-500 font-bold uppercase tracking-widest">Step 05 / 05</span>
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

      <main className="relative z-10 max-w-5xl mx-auto px-6 py-12 md:py-16">
        {/* Header Section */}
        <div className="mb-8 space-y-2">
          <h2 className="text-4xl font-extrabold tracking-tight">Private Data <span className="text-blue-500">Management Preferences.</span></h2>
          <p className="text-slate-400">Configure how your Data Fabric handles and protects your information.</p>
        </div>

        {/* Security Notice */}
        <div className="mb-8 p-4 bg-blue-500/5 border border-blue-500/20 rounded-xl flex items-start space-x-4">
          <Shield className="text-blue-500 shrink-0 mt-1" size={20} />
          <div>
            <h4 className="font-bold text-sm mb-1">Your Data, Your Rules</h4>
            <p className="text-sm text-slate-400">
              These settings determine how your Data Fabric processes and stores information. 
              You can change these preferences at any time from your settings panel.
            </p>
          </div>
        </div>

        {/* Preferences Grid */}
        <div className="grid md:grid-cols-2 gap-6 mb-8">
          {preferenceCategories.map((category, idx) => {
            const Icon = category.icon;
            return (
              <div key={idx} className="bg-white/5 border border-white/10 p-6 rounded-2xl space-y-4">
                <div className="flex items-center space-x-3 mb-4">
                  <div className="p-2 bg-blue-600/10 rounded-lg">
                    <Icon className="text-blue-500" size={20} />
                  </div>
                  <div>
                    <h3 className="font-bold">{category.title}</h3>
                    <p className="text-xs text-slate-500">{category.description}</p>
                  </div>
                </div>

                <div className="space-y-4">
                  {category.settings.map((setting) => (
                    <div key={setting.key} className="flex items-center justify-between">
                      <div className="flex-1 pr-4">
                        <Label className="text-sm font-medium text-slate-300">{setting.label}</Label>
                        <p className="text-xs text-slate-500">{setting.desc}</p>
                      </div>
                      <Switch
                        checked={settings[setting.key as keyof PrivacySettings] as boolean}
                        onCheckedChange={() => handleToggle(setting.key as keyof PrivacySettings)}
                        className="data-[state=checked]:bg-blue-600"
                      />
                    </div>
                  ))}
                </div>
              </div>
            );
          })}
        </div>

        {/* Data Retention Slider */}
        <div className="bg-white/5 border border-white/10 p-6 rounded-2xl mb-8">
          <div className="flex items-center space-x-3 mb-6">
            <div className="p-2 bg-blue-600/10 rounded-lg">
              <Activity className="text-blue-500" size={20} />
            </div>
            <div>
              <h3 className="font-bold">Data Retention Period</h3>
              <p className="text-xs text-slate-500">How long to keep inactive data before automatic deletion</p>
            </div>
          </div>

          <div className="space-y-4">
            <div className="flex justify-between text-sm">
              <span className="text-slate-500">7 days</span>
              <span className="text-blue-400 font-bold">{settings.dataRetentionDays} days</span>
              <span className="text-slate-500">365 days</span>
            </div>
            <Slider
              value={[settings.dataRetentionDays]}
              onValueChange={handleRetentionChange}
              min={7}
              max={365}
              step={1}
              className="w-full"
            />
            <p className="text-xs text-slate-500">
              Data older than {settings.dataRetentionDays} days will be automatically purged from your Cloud Cortex.
            </p>
          </div>
        </div>

        {/* Privacy Summary */}
        <div className="bg-gradient-to-r from-blue-600/10 to-indigo-600/10 border border-blue-500/20 p-6 rounded-2xl mb-8">
          <div className="flex items-start space-x-4">
            <div className="p-2 bg-blue-600/20 rounded-lg">
              <CheckCircle2 className="text-blue-500" size={24} />
            </div>
            <div className="flex-1">
              <h4 className="font-bold mb-2">Privacy Configuration Summary</h4>
              <div className="grid md:grid-cols-2 gap-2 text-sm">
                <div className="flex items-center space-x-2">
                  <div className={`w-2 h-2 rounded-full ${settings.dataEncryption ? 'bg-green-500' : 'bg-red-500'}`} />
                  <span className="text-slate-400">Encryption: {settings.dataEncryption ? 'Enabled' : 'Disabled'}</span>
                </div>
                <div className="flex items-center space-x-2">
                  <div className={`w-2 h-2 rounded-full ${settings.localProcessing ? 'bg-green-500' : 'bg-red-500'}`} />
                  <span className="text-slate-400">Local Processing: {settings.localProcessing ? 'Enabled' : 'Disabled'}</span>
                </div>
                <div className="flex items-center space-x-2">
                  <div className={`w-2 h-2 rounded-full ${settings.anonymizeMetrics ? 'bg-green-500' : 'bg-red-500'}`} />
                  <span className="text-slate-400">Anonymization: {settings.anonymizeMetrics ? 'Enabled' : 'Disabled'}</span>
                </div>
                <div className="flex items-center space-x-2">
                  <div className={`w-2 h-2 rounded-full ${settings.autoDeleteInactive ? 'bg-green-500' : 'bg-red-500'}`} />
                  <span className="text-slate-400">Auto-Delete: {settings.autoDeleteInactive ? 'Enabled' : 'Disabled'}</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Warning Notice */}
        <div className="mb-8 p-4 bg-amber-500/5 border border-amber-500/20 rounded-xl flex items-start space-x-4">
          <AlertTriangle className="text-amber-500 shrink-0 mt-1" size={20} />
          <div>
            <h4 className="font-bold text-sm mb-1 text-amber-400">Important Notice</h4>
            <p className="text-sm text-slate-400">
              Once you submit these preferences, a confirmation email will be sent to verify your identity. 
              You must click the confirmation link to activate your Data Fabric.
            </p>
          </div>
        </div>

        {/* Navigation Actions */}
        <div className="flex justify-between items-center border-t border-white/5 pt-8">
          <Button 
            variant="ghost"
            onClick={onBack}
            className="text-slate-500 hover:text-white transition-colors flex items-center space-x-2 text-sm font-bold uppercase tracking-widest"
          >
            <ChevronLeft size={16} />
            <span>Back</span>
          </Button>
          
          <Button 
            onClick={handleSubmit}
            disabled={isSubmitting}
            className="group bg-blue-600 hover:bg-blue-500 text-white px-8 py-4 rounded-xl font-bold transition-all transform active:scale-95 flex items-center space-x-3 h-auto disabled:opacity-50"
          >
            {isSubmitting ? (
              <>
                <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                <span>Saving Preferences...</span>
              </>
            ) : (
              <>
                <span>Submit Preferences</span>
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
    </div>
  );
};

export default DataPreferencesForm;
