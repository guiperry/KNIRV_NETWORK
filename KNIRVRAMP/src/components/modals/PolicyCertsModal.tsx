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
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Database, Shield, Lock, Globe, DollarSign, Clock, AlertTriangle, CheckCircle2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Switch } from "@/components/ui/switch";
import { Slider } from "@/components/ui/slider";

export interface PolicyCert {
  id: string;
  category: string;
  name: string;
  value: string | number | boolean;
  description: string;
  enabled: boolean;
}

interface PolicyCertsModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (certs: PolicyCert[]) => void;
  initialCerts?: PolicyCert[];
}

const defaultPolicies: PolicyCert[] = [
  {
    id: 'network-access',
    category: 'Network',
    name: 'Network Access',
    value: 'Internal Only',
    description: 'Restrict network access to internal resources only',
    enabled: true,
  },
  {
    id: 'spend-limit',
    category: 'Budget',
    name: 'Spend Limit',
    value: 50,
    description: 'Maximum spending limit per hour',
    enabled: true,
  },
  {
    id: 'execution-timeout',
    category: 'Performance',
    name: 'Execution Timeout',
    value: 30,
    description: 'Maximum execution time in minutes',
    enabled: true,
  },
  {
    id: 'max-tokens',
    category: 'Usage',
    name: 'Max Tokens Per Request',
    value: 4096,
    description: 'Limit tokens per API request',
    enabled: false,
  },
  {
    id: 'data-retention',
    category: 'Privacy',
    name: 'Data Retention',
    value: 30,
    description: 'Days to retain session data',
    enabled: true,
  },
  {
    id: 'filesystem-scope',
    category: 'Security',
    name: 'Filesystem Scope',
    value: 'Sandboxed',
    description: 'Restrict file system access scope',
    enabled: true,
  },
];

const networkOptions = ['Internal Only', 'External Limited', 'External Full', 'Blocked'];
const filesystemOptions = ['Sandboxed', 'User Home', 'Full Access', 'Read Only'];

export function PolicyCertsModal({ isOpen, onClose, onSave, initialCerts = [] }: PolicyCertsModalProps) {
  const [policies, setPolicies] = useState<PolicyCert[]>(
    initialCerts.length > 0 ? initialCerts : defaultPolicies
  );

  const togglePolicy = (id: string) => {
    setPolicies(policies.map(p => 
      p.id === id ? { ...p, enabled: !p.enabled } : p
    ));
  };

  const updatePolicyValue = (id: string, value: string | number | boolean) => {
    setPolicies(policies.map(p => 
      p.id === id ? { ...p, value } : p
    ));
  };

  const handleSave = () => {
    onSave(policies);
    onClose();
  };

  const enabledCount = policies.filter(p => p.enabled).length;

  const getIcon = (category: string) => {
    switch (category) {
      case 'Network': return <Globe size={18} />;
      case 'Budget': return <DollarSign size={18} />;
      case 'Performance': return <Clock size={18} />;
      case 'Security': return <Shield size={18} />;
      case 'Privacy': return <Lock size={18} />;
      default: return <CheckCircle2 size={18} />;
    }
  };

  const renderPolicyControl = (policy: PolicyCert) => {
    if (policy.id === 'network-access') {
      return (
        <select
          value={policy.value as string}
          onChange={(e) => updatePolicyValue(policy.id, e.target.value)}
          disabled={!policy.enabled}
          className="bg-black/40 border border-white/10 rounded-lg px-3 py-2 text-sm text-white focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
        >
          {networkOptions.map(opt => (
            <option key={opt} value={opt}>{opt}</option>
          ))}
        </select>
      );
    }

    if (policy.id === 'filesystem-scope') {
      return (
        <select
          value={policy.value as string}
          onChange={(e) => updatePolicyValue(policy.id, e.target.value)}
          disabled={!policy.enabled}
          className="bg-black/40 border border-white/10 rounded-lg px-3 py-2 text-sm text-white focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
        >
          {filesystemOptions.map(opt => (
            <option key={opt} value={opt}>{opt}</option>
          ))}
        </select>
      );
    }

    if (policy.id === 'spend-limit') {
      return (
        <div className="flex items-center gap-2">
          <span className="text-slate-400">$</span>
          <Input
            type="number"
            value={typeof policy.value === 'number' ? policy.value : 0}
            onChange={(e) => updatePolicyValue(policy.id, parseFloat(e.target.value) || 0)}
            disabled={!policy.enabled}
            className="w-24 bg-black/40 border-white/10 text-white text-center"
          />
          <span className="text-slate-400 text-sm">/hr</span>
        </div>
      );
    }

    if (policy.id === 'execution-timeout') {
      return (
        <div className="flex items-center gap-2">
          <Input
            type="number"
            value={typeof policy.value === 'number' ? policy.value : 0}
            onChange={(e) => updatePolicyValue(policy.id, parseInt(e.target.value) || 0)}
            disabled={!policy.enabled}
            className="w-24 bg-black/40 border-white/10 text-white text-center"
          />
          <span className="text-slate-400 text-sm">min</span>
        </div>
      );
    }

    if (policy.id === 'data-retention') {
      return (
        <div className="flex items-center gap-2">
          <Input
            type="number"
            value={typeof policy.value === 'number' ? policy.value : 0}
            onChange={(e) => updatePolicyValue(policy.id, parseInt(e.target.value) || 0)}
            disabled={!policy.enabled}
            className="w-24 bg-black/40 border-white/10 text-white text-center"
          />
          <span className="text-slate-400 text-sm">days</span>
        </div>
      );
    }

    if (policy.id === 'max-tokens') {
      return (
        <div className="w-48">
          <Slider
            value={[policy.value as number]}
            onValueChange={([value]) => updatePolicyValue(policy.id, value)}
            max={8192}
            min={256}
            step={256}
            disabled={!policy.enabled}
            className="w-full"
          />
          <div className="text-center text-sm text-slate-400 mt-1">
            {policy.value.toLocaleString()} tokens
          </div>
        </div>
      );
    }

    return null;
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-w-2xl bg-[#0a0a0c] border-white/10 text-slate-200 max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <div className="flex items-center gap-3">
            <div className="p-2 bg-blue-600/20 rounded-lg">
              <Database className="text-blue-500" size={24} />
            </div>
            <div>
              <DialogTitle className="text-xl font-bold text-white">Policy Certs</DialogTitle>
              <DialogDescription className="text-slate-400 text-sm">
                Define the &quot;Guardrails.&quot; Set thresholds and policies that translate intent into kernel-level protections.
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <div className="space-y-6 py-4">
          {/* Info Banner */}
          <div className="p-4 bg-blue-500/5 border border-blue-500/20 rounded-xl">
            <div className="flex items-start gap-3">
              <Shield className="text-blue-500 shrink-0 mt-0.5" size={20} />
              <p className="text-sm text-slate-300 leading-relaxed">
                Instead of complex code, configure simple thresholds. The Nexus translates these 
                policies into kernel-level guardrails that protect your fabric environment.
              </p>
            </div>
          </div>

          {/* Active Policies Summary */}
          <div className="flex items-center justify-between">
            <span className="text-xs uppercase font-bold text-slate-500 tracking-wider">
              Active Guardrails
            </span>
            <Badge 
              variant={enabledCount > 0 ? "default" : "outline"}
              className={enabledCount > 0 ? "bg-green-500/20 text-green-400 border-green-500/30" : "text-slate-400 border-white/10"}
            >
              {enabledCount} enabled
            </Badge>
          </div>

          {/* Policy List */}
          <div className="space-y-3">
            {policies.map((policy) => (
              <div
                key={policy.id}
                className={`p-4 rounded-xl border transition-all ${
                  policy.enabled 
                    ? 'bg-white/5 border-blue-500/30' 
                    : 'bg-white/[0.02] border-white/10'
                }`}
              >
                <div className="flex items-start justify-between gap-4">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      <div className={`${policy.enabled ? 'text-blue-500' : 'text-slate-600'}`}>
                        {getIcon(policy.category)}
                      </div>
                      <h4 className={`font-bold ${policy.enabled ? 'text-white' : 'text-slate-500'}`}>
                        {policy.name}
                      </h4>
                      <Badge 
                        variant="outline" 
                        className={`text-[10px] ${policy.enabled ? 'border-blue-500/30 text-blue-400' : 'border-white/10 text-slate-600'}`}
                      >
                        {policy.category}
                      </Badge>
                    </div>
                    <p className={`text-sm ${policy.enabled ? 'text-slate-400' : 'text-slate-600'}`}>
                      {policy.description}
                    </p>
                  </div>
                  
                  <div className="flex items-center gap-4">
                    {renderPolicyControl(policy)}
                    <Switch
                      checked={policy.enabled}
                      onCheckedChange={() => togglePolicy(policy.id)}
                      className="data-[state=checked]:bg-blue-600"
                    />
                  </div>
                </div>
              </div>
            ))}
          </div>

          {/* Warning for no policies */}
          {enabledCount === 0 && (
            <div className="p-4 bg-amber-500/10 border border-amber-500/20 rounded-xl flex items-center gap-3">
              <AlertTriangle className="text-amber-500 shrink-0" size={20} />
              <p className="text-sm text-amber-400">
                Warning: No guardrails are enabled. Your fabric environment will operate without restrictions.
              </p>
            </div>
          )}

          {/* Security Note */}
          <div className="p-4 bg-green-500/5 border border-green-500/20 rounded-xl">
            <div className="flex items-start gap-3">
              <Lock className="text-green-500 shrink-0 mt-0.5" size={18} />
              <p className="text-xs text-slate-500">
                All policies are enforced at the kernel level. Changes take effect immediately 
                and are cryptographically signed for integrity verification.
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
            Save Policy Certs
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
