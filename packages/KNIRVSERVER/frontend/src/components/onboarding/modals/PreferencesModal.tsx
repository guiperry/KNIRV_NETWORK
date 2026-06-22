'use client';

import React, { useState } from 'react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
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
  CheckCircle2,
  AlertTriangle,
  SlidersHorizontal
} from "lucide-react";

export interface PrivacySettings {
  dataEncryption: boolean;
  localProcessing: boolean;
  anonymizeMetrics: boolean;
  shareErrorLogs: boolean;
  allowAnalytics: boolean;
  dataRetentionDays: number;
  autoDeleteInactive: boolean;
  thirdPartyIntegrations: boolean;
}

interface PreferencesModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (settings: PrivacySettings) => void;
  initialSettings?: PrivacySettings;
}

const defaultSettings: PrivacySettings = {
  dataEncryption: true,
  localProcessing: true,
  anonymizeMetrics: true,
  shareErrorLogs: false,
  allowAnalytics: false,
  dataRetentionDays: 90,
  autoDeleteInactive: true,
  thirdPartyIntegrations: false
};

export function PreferencesModal({ isOpen, onClose, onSave, initialSettings }: PreferencesModalProps) {
  const [settings, setSettings] = useState<PrivacySettings>(
    initialSettings || defaultSettings
  );

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

  const handleSave = () => {
    onSave(settings);
    onClose();
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
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="w-[96vw] max-w-7xl max-h-[92vh] overflow-hidden bg-[#0a0a0c] border-white/10 text-slate-200 flex flex-col">
        <DialogHeader>
          <div className="flex items-center gap-3">
            <div className="p-2 bg-blue-600/20 rounded-lg">
              <SlidersHorizontal className="text-blue-500" size={24} />
            </div>
            <div>
              <DialogTitle className="text-xl font-bold text-white">Private Data Management Preferences</DialogTitle>
              <DialogDescription className="text-slate-400 text-sm">
                Configure how your Data Fabric handles and protects your information.
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <div className="flex-1 overflow-y-auto py-4 pr-1 custom-scrollbar space-y-6">
          {/* Security Notice */}
          <div className="p-4 bg-blue-500/5 border border-blue-500/20 rounded-xl flex items-start space-x-4">
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
          <div className="grid lg:grid-cols-2 xl:grid-cols-4 gap-4">
            {preferenceCategories.map((category, idx) => {
              const Icon = category.icon;
              return (
                <div key={idx} className="bg-white/5 border border-white/10 p-5 rounded-xl space-y-4">
                  <div className="flex items-center space-x-3 mb-3">
                    <div className="p-2 bg-blue-600/10 rounded-lg">
                      <Icon className="text-blue-500" size={18} />
                    </div>
                    <div>
                      <h3 className="font-bold text-sm">{category.title}</h3>
                      <p className="text-xs text-slate-500">{category.description}</p>
                    </div>
                  </div>

                  <div className="space-y-3">
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
          <div className="bg-white/5 border border-white/10 p-5 rounded-xl">
            <div className="flex items-center space-x-3 mb-4">
              <div className="p-2 bg-blue-600/10 rounded-lg">
                <Activity className="text-blue-500" size={18} />
              </div>
              <div>
                <h3 className="font-bold text-sm">Data Retention Period</h3>
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
          <div className="bg-gradient-to-r from-blue-600/10 to-indigo-600/10 border border-blue-500/20 p-5 rounded-xl">
            <div className="flex items-start space-x-4">
              <div className="p-2 bg-blue-600/20 rounded-lg">
                <CheckCircle2 className="text-blue-500" size={20} />
              </div>
              <div className="flex-1">
                <h4 className="font-bold mb-2 text-sm">Privacy Configuration Summary</h4>
                <div className="grid md:grid-cols-2 gap-2 text-sm">
                  <div className="flex items-center space-x-2">
                    <div className={`w-2 h-2 rounded-full ${settings.dataEncryption ? 'bg-green-500' : 'bg-red-500'}`} />
                    <span className="text-slate-400 text-xs">Encryption: {settings.dataEncryption ? 'Enabled' : 'Disabled'}</span>
                  </div>
                  <div className="flex items-center space-x-2">
                    <div className={`w-2 h-2 rounded-full ${settings.localProcessing ? 'bg-green-500' : 'bg-red-500'}`} />
                    <span className="text-slate-400 text-xs">Local Processing: {settings.localProcessing ? 'Enabled' : 'Disabled'}</span>
                  </div>
                  <div className="flex items-center space-x-2">
                    <div className={`w-2 h-2 rounded-full ${settings.anonymizeMetrics ? 'bg-green-500' : 'bg-red-500'}`} />
                    <span className="text-slate-400 text-xs">Anonymization: {settings.anonymizeMetrics ? 'Enabled' : 'Disabled'}</span>
                  </div>
                  <div className="flex items-center space-x-2">
                    <div className={`w-2 h-2 rounded-full ${settings.autoDeleteInactive ? 'bg-green-500' : 'bg-red-500'}`} />
                    <span className="text-slate-400 text-xs">Auto-Delete: {settings.autoDeleteInactive ? 'Enabled' : 'Disabled'}</span>
                  </div>
                </div>
              </div>
            </div>
          </div>

          {/* Warning Notice */}
          <div className="p-4 bg-amber-500/5 border border-amber-500/20 rounded-xl flex items-start space-x-4">
            <AlertTriangle className="text-amber-500 shrink-0 mt-1" size={18} />
            <div>
              <h4 className="font-bold text-sm mb-1 text-amber-400">Important Notice</h4>
              <p className="text-sm text-slate-400">
                Once you submit these preferences, a confirmation email will be sent to verify your identity. 
                You must click the confirmation link to activate your Data Fabric.
              </p>
            </div>
          </div>
        </div>

        <DialogFooter className="gap-3">
          <Button
            variant="ghost"
            onClick={onClose}
            className="text-slate-400 hover:text-white"
          >
            Cancel
          </Button>
          <Button
            onClick={handleSave}
            className="bg-blue-600 hover:bg-blue-500 text-white"
          >
            Save Preferences
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
