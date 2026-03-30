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
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import { Database, Shield, Lock, Globe, DollarSign, Clock, AlertTriangle, CheckCircle2, Cpu, Plus, Trash2, FileText, Code, Terminal } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Switch } from "@/components/ui/switch";
import { Slider } from "@/components/ui/slider";
import { ScrollArea } from "@/components/ui/scroll-area";

export interface PolicyCert {
  id: string;
  category: string;
  name: string;
  value: string | number | boolean;
  description: string;
  enabled: boolean;
}

export interface CustomRule {
  id: string;
  name: string;
  description: string;
  ruleType: 'instruction' | 'code' | 'constraint';
  priority: 'low' | 'medium' | 'high' | 'critical';
}

interface PolicyCertsModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (certs: PolicyCert[], rules: CustomRule[]) => void;
  initialCerts?: PolicyCert[];
  initialRules?: CustomRule[];
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

const ruleTypeOptions = [
  { value: 'instruction', label: 'Instruction', icon: FileText },
  { value: 'code', label: 'Code Rule', icon: Code },
  { value: 'constraint', label: 'Constraint', icon: Terminal },
];

const priorityOptions = [
  { value: 'low', label: 'Low', color: 'bg-slate-500/20 text-slate-400 border-slate-500/30' },
  { value: 'medium', label: 'Medium', color: 'bg-blue-500/20 text-blue-400 border-blue-500/30' },
  { value: 'high', label: 'High', color: 'bg-amber-500/20 text-amber-400 border-amber-500/30' },
  { value: 'critical', label: 'Critical', color: 'bg-red-500/20 text-red-400 border-red-500/30' },
];

export function PolicyCertsModal({ isOpen, onClose, onSave, initialCerts = [], initialRules = [] }: PolicyCertsModalProps) {
  const [policies, setPolicies] = useState<PolicyCert[]>(
    initialCerts.length > 0 ? initialCerts : defaultPolicies
  );
  const [rules, setRules] = useState<CustomRule[]>(initialRules);
  const [showAddRuleForm, setShowAddRuleForm] = useState(false);
  const [newRule, setNewRule] = useState<Partial<CustomRule>>({
    name: '',
    description: '',
    ruleType: 'instruction',
    priority: 'medium',
  });
  const [errors, setErrors] = useState<Record<string, string>>({});

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

  const addRule = () => {
    const validationErrors: Record<string, string> = {};
    
    if (!newRule.name?.trim()) {
      validationErrors.name = 'Rule name is required';
    }
    if (!newRule.description?.trim()) {
      validationErrors.description = 'Rule description is required';
    }

    if (Object.keys(validationErrors).length > 0) {
      setErrors(validationErrors);
      return;
    }

    const rule: CustomRule = {
      id: crypto.randomUUID(),
      name: newRule.name!,
      description: newRule.description!,
      ruleType: newRule.ruleType || 'instruction',
      priority: newRule.priority || 'medium',
    };

    setRules([...rules, rule]);
    setNewRule({
      name: '',
      description: '',
      ruleType: 'instruction',
      priority: 'medium',
    });
    setErrors({});
    setShowAddRuleForm(false);
  };

  const removeRule = (id: string) => {
    setRules(rules.filter(r => r.id !== id));
  };

  const handleSave = () => {
    onSave(policies, rules);
    onClose();
  };

  const getRuleTypeIcon = (type: string) => {
    const option = ruleTypeOptions.find(o => o.value === type);
    const Icon = option?.icon || FileText;
    return <Icon size={16} />;
  };

  const getPriorityStyle = (priority: string) => {
    return priorityOptions.find(o => o.value === priority)?.color || '';
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
                className={`p-4 rounded-xl border transition-interactive ${
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

          {/* Custom Rules Section */}
          <div className="border-t border-white/10 pt-6 mt-6">
            <div className="flex items-center gap-3 mb-4">
              <div className="p-2 bg-purple-600/20 rounded-lg">
                <Cpu className="text-purple-500" size={20} />
              </div>
              <div>
                <h3 className="font-bold text-white">Custom Rules</h3>
                <p className="text-xs text-slate-400">Define behavioral rules and instructions</p>
              </div>
              <Badge variant="outline" className="ml-auto text-slate-400 border-white/10">
                {rules.length} rule{rules.length !== 1 ? 's' : ''}
              </Badge>
            </div>

            <ScrollArea className="h-[200px] border border-white/10 rounded-xl mb-4">
              <div className="p-4 space-y-3">
                {rules.length === 0 ? (
                  <div className="flex flex-col items-center justify-center h-32 text-slate-500">
                    <p className="text-sm">No custom rules configured</p>
                    <p className="text-xs text-slate-600 mt-1">Add rules to define specific behaviors</p>
                  </div>
                ) : (
                  rules.map((rule, index) => (
                    <div
                      key={rule.id}
                      className="p-3 bg-white/5 border border-white/10 rounded-lg group hover:border-white/20 transition-interactive"
                    >
                      <div className="flex items-start justify-between gap-4">
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 mb-1">
                            <span className="text-slate-600 text-xs font-mono">
                              {String(index + 1).padStart(2, '0')}
                            </span>
                            <h4 className="font-bold text-white truncate">{rule.name}</h4>
                            <Badge 
                              variant="outline" 
                              className={`text-[10px] ${getPriorityStyle(rule.priority)}`}
                            >
                              {rule.priority}
                            </Badge>
                          </div>
                          <p className="text-sm text-slate-400 line-clamp-2">{rule.description}</p>
                          <div className="flex items-center gap-2 mt-1">
                            <div className="flex items-center gap-1.5 text-xs text-slate-500">
                              {getRuleTypeIcon(rule.ruleType)}
                              <span className="capitalize">{rule.ruleType}</span>
                            </div>
                          </div>
                        </div>
                        <button
                          onClick={() => removeRule(rule.id)}
                          className="opacity-0 group-hover:opacity-100 text-red-400 hover:text-red-300 transition-opacity p-1"
                        >
                          <Trash2 size={16} />
                        </button>
                      </div>
                    </div>
                  ))
                )}
              </div>
            </ScrollArea>

            {showAddRuleForm ? (
              <div className="p-4 bg-white/5 border border-purple-500/30 rounded-xl space-y-4">
                <div className="space-y-2">
                  <Label className="text-xs uppercase font-bold text-slate-500 tracking-wider">
                    Rule Name *
                  </Label>
                  <Input
                    value={newRule.name}
                    onChange={(e) => setNewRule({ ...newRule, name: e.target.value })}
                    placeholder="e.g. Code Style Enforcement"
                    className="bg-black/40 border-white/10 text-white"
                  />
                  {errors.name && (
                    <p className="text-xs text-red-400">{errors.name}</p>
                  )}
                </div>
                <div className="space-y-2">
                  <Label className="text-xs uppercase font-bold text-slate-500 tracking-wider">
                    Description *
                  </Label>
                  <Textarea
                    value={newRule.description}
                    onChange={(e) => setNewRule({ ...newRule, description: e.target.value })}
                    placeholder="Describe what this rule enforces..."
                    className="bg-black/40 border-white/10 text-white min-h-[60px] resize-none"
                  />
                  {errors.description && (
                    <p className="text-xs text-red-400">{errors.description}</p>
                  )}
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label className="text-xs uppercase font-bold text-slate-500 tracking-wider">
                      Rule Type
                    </Label>
                    <select
                      value={newRule.ruleType}
                      onChange={(e) => setNewRule({ ...newRule, ruleType: e.target.value as CustomRule['ruleType'] })}
                      className="w-full bg-black/40 border border-white/10 rounded-lg px-3 py-2 text-sm text-white"
                    >
                      {ruleTypeOptions.map(opt => (
                        <option key={opt.value} value={opt.value}>{opt.label}</option>
                      ))}
                    </select>
                  </div>
                  <div className="space-y-2">
                    <Label className="text-xs uppercase font-bold text-slate-500 tracking-wider">
                      Priority
                    </Label>
                    <select
                      value={newRule.priority}
                      onChange={(e) => setNewRule({ ...newRule, priority: e.target.value as CustomRule['priority'] })}
                      className="w-full bg-black/40 border border-white/10 rounded-lg px-3 py-2 text-sm text-white"
                    >
                      {priorityOptions.map(opt => (
                        <option key={opt.value} value={opt.value}>{opt.label}</option>
                      ))}
                    </select>
                  </div>
                </div>
                <div className="flex gap-2">
                  <Button
                    variant="outline"
                    onClick={() => {
                      setShowAddRuleForm(false);
                      setErrors({});
                    }}
                    className="flex-1 border-white/20 text-slate-400"
                  >
                    Cancel
                  </Button>
                  <Button
                    onClick={addRule}
                    className="flex-1 bg-purple-600 hover:bg-purple-500 text-white"
                  >
                    Add Rule
                  </Button>
                </div>
              </div>
            ) : (
              <Button
                variant="outline"
                onClick={() => setShowAddRuleForm(true)}
                className="w-full border-dashed border-white/20 text-slate-400 hover:text-white hover:border-purple-500/50 hover:bg-purple-500/10"
              >
                <Plus size={18} className="mr-2" />
                Add Custom Rule
              </Button>
            )}
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
