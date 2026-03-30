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
import { Cpu, Plus, Trash2, FileText, AlertCircle, CheckCircle2, Code, Terminal } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";

export interface CustomRule {
  id: string;
  name: string;
  description: string;
  ruleType: 'instruction' | 'code' | 'constraint';
  priority: 'low' | 'medium' | 'high' | 'critical';
}

interface CustomRulesModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (rules: CustomRule[]) => void;
  initialRules?: CustomRule[];
}

const ruleTypeOptions = [
  { value: 'instruction', label: 'Instruction', icon: FileText, desc: 'General behavioral instructions' },
  { value: 'code', label: 'Code Rule', icon: Code, desc: 'Code-specific guidelines' },
  { value: 'constraint', label: 'Constraint', icon: Terminal, desc: 'Hard limits and restrictions' },
];

const priorityOptions = [
  { value: 'low', label: 'Low', color: 'bg-slate-500/20 text-slate-400 border-slate-500/30' },
  { value: 'medium', label: 'Medium', color: 'bg-blue-500/20 text-blue-400 border-blue-500/30' },
  { value: 'high', label: 'High', color: 'bg-amber-500/20 text-amber-400 border-amber-500/30' },
  { value: 'critical', label: 'Critical', color: 'bg-red-500/20 text-red-400 border-red-500/30' },
];

export function CustomRulesModal({ isOpen, onClose, onSave, initialRules = [] }: CustomRulesModalProps) {
  const [rules, setRules] = useState<CustomRule[]>(initialRules);
  const [showAddForm, setShowAddForm] = useState(false);
  const [newRule, setNewRule] = useState<Partial<CustomRule>>({
    name: '',
    description: '',
    ruleType: 'instruction',
    priority: 'medium',
  });
  const [errors, setErrors] = useState<Record<string, string>>({});

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
    setShowAddForm(false);
  };

  const removeRule = (id: string) => {
    setRules(rules.filter(r => r.id !== id));
  };

  const handleSave = () => {
    onSave(rules);
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

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-w-3xl bg-[#0a0a0c] border-white/10 text-slate-200 max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <div className="flex items-center gap-3">
            <div className="p-2 bg-blue-600/20 rounded-lg">
              <Cpu className="text-blue-500" size={24} />
            </div>
            <div>
              <DialogTitle className="text-xl font-bold text-white">Custom Rules</DialogTitle>
              <DialogDescription className="text-slate-400 text-sm">
                Define custom behavioral rules, instructions, and constraints for your fabric environment.
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <div className="space-y-6 py-4">
          {/* Info Banner */}
          <div className="p-4 bg-blue-500/5 border border-blue-500/20 rounded-xl">
            <div className="flex items-start gap-3">
              <Terminal className="text-blue-500 shrink-0 mt-0.5" size={20} />
              <p className="text-sm text-slate-300 leading-relaxed">
                Custom rules allow you to define specific behaviors, coding standards, or constraints 
                that govern how agents operate within your fabric environment.
              </p>
            </div>
          </div>

          {/* Rules List */}
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <Label className="text-xs uppercase font-bold text-slate-500 tracking-wider">
                Configured Rules
              </Label>
              <Badge variant="outline" className="text-slate-400 border-white/10">
                {rules.length} rule{rules.length !== 1 ? 's' : ''}
              </Badge>
            </div>

            <ScrollArea className="h-[250px] border border-white/10 rounded-xl">
              <div className="p-4 space-y-3">
                {rules.length === 0 ? (
                  <div className="flex flex-col items-center justify-center h-32 text-slate-500">
                    <AlertCircle className="mb-2" size={24} />
                    <p className="text-sm">No custom rules configured yet</p>
                    <p className="text-xs text-slate-600 mt-1">Add your first rule below</p>
                  </div>
                ) : (
                  rules.map((rule, index) => (
                    <div
                      key={rule.id}
                      className="p-4 bg-white/5 border border-white/10 rounded-xl group hover:border-white/20 transition-interactive"
                    >
                      <div className="flex items-start justify-between gap-4">
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 mb-2">
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
                          <p className="text-sm text-slate-400 mb-2 line-clamp-2">{rule.description}</p>
                          <div className="flex items-center gap-2">
                            <div className="flex items-center gap-1.5 text-xs text-slate-500">
                              {getRuleTypeIcon(rule.ruleType)}
                              <span className="capitalize">{rule.ruleType}</span>
                            </div>
                          </div>
                        </div>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => removeRule(rule.id)}
                          className="opacity-0 group-hover:opacity-100 text-red-400 hover:text-red-300 hover:bg-red-500/10 transition-opacity"
                        >
                          <Trash2 size={16} />
                        </Button>
                      </div>
                    </div>
                  ))
                )}
              </div>
            </ScrollArea>
          </div>

          {/* Add Rule Form */}
          {showAddForm ? (
            <div className="p-4 bg-white/5 border border-blue-500/30 rounded-xl space-y-4">
              <h4 className="font-bold text-white flex items-center gap-2">
                <Plus size={18} className="text-blue-500" />
                Add New Rule
              </h4>

              <div className="space-y-4">
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
                    <p className="text-xs text-red-400 flex items-center gap-1">
                      <AlertCircle size={12} />
                      {errors.name}
                    </p>
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
                    className="bg-black/40 border-white/10 text-white min-h-[80px] resize-none"
                  />
                  {errors.description && (
                    <p className="text-xs text-red-400 flex items-center gap-1">
                      <AlertCircle size={12} />
                      {errors.description}
                    </p>
                  )}
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label className="text-xs uppercase font-bold text-slate-500 tracking-wider">
                      Rule Type
                    </Label>
                    <div className="grid grid-cols-1 gap-2">
                      {ruleTypeOptions.map((option) => {
                        const Icon = option.icon;
                        return (
                          <button
                            key={option.value}
                            onClick={() => setNewRule({ ...newRule, ruleType: option.value as CustomRule['ruleType'] })}
                            className={`p-3 rounded-lg border text-left transition-interactive ${
                              newRule.ruleType === option.value
                                ? 'bg-blue-500/10 border-blue-500'
                                : 'bg-black/40 border-white/10 hover:border-white/20'
                            }`}
                          >
                            <div className="flex items-center gap-2">
                              <Icon size={16} className={newRule.ruleType === option.value ? 'text-blue-500' : 'text-slate-500'} />
                              <span className={`text-sm font-medium ${newRule.ruleType === option.value ? 'text-white' : 'text-slate-400'}`}>
                                {option.label}
                              </span>
                            </div>
                            <p className="text-xs text-slate-500 mt-1">{option.desc}</p>
                          </button>
                        );
                      })}
                    </div>
                  </div>

                  <div className="space-y-2">
                    <Label className="text-xs uppercase font-bold text-slate-500 tracking-wider">
                      Priority
                    </Label>
                    <div className="grid grid-cols-1 gap-2">
                      {priorityOptions.map((option) => (
                        <button
                          key={option.value}
                          onClick={() => setNewRule({ ...newRule, priority: option.value as CustomRule['priority'] })}
className={`p-3 rounded-lg border text-left transition-interactive ${
                            priority === option.value
                              ? 'bg-blue-500/10 border-blue-500'
                              : 'bg-black/40 border-white/10 hover:border-white/20'
                          }`}
                        >
                          <div className="flex items-center justify-between">
                            <span className={`text-sm font-medium ${newRule.priority === option.value ? 'text-white' : 'text-slate-400'}`}>
                              {option.label}
                            </span>
                            <div className={`w-2 h-2 rounded-full ${option.color.split(' ')[0].replace('/20', '')}`} />
                          </div>
                        </button>
                      ))}
                    </div>
                  </div>
                </div>
              </div>

              <div className="flex gap-2 pt-2">
                <Button
                  variant="outline"
                  onClick={() => {
                    setShowAddForm(false);
                    setErrors({});
                  }}
                  className="flex-1 border-white/20 text-slate-400"
                >
                  Cancel
                </Button>
                <Button
                  onClick={addRule}
                  className="flex-1 bg-blue-600 hover:bg-blue-500 text-white"
                >
                  Add Rule
                </Button>
              </div>
            </div>
          ) : (
            <Button
              variant="outline"
              onClick={() => setShowAddForm(true)}
              className="w-full border-dashed border-white/20 text-slate-400 hover:text-white hover:border-blue-500/50 hover:bg-blue-500/10"
            >
              <Plus size={18} className="mr-2" />
              Add Custom Rule
            </Button>
          )}

          {/* Summary */}
          {rules.length > 0 && (
            <div className="p-3 bg-green-500/10 border border-green-500/20 rounded-lg">
              <div className="flex items-center gap-2">
                <CheckCircle2 className="text-green-500" size={18} />
                <p className="text-sm text-green-400">
                  {rules.length} custom rule{rules.length !== 1 ? 's' : ''} configured
                </p>
              </div>
            </div>
          )}
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
            Save Custom Rules
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
