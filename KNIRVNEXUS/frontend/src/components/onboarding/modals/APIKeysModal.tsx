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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Key, Plus, Trash2, Eye, EyeOff, AlertCircle } from "lucide-react";

export interface APIKeyEntry {
  id: string;
  provider: string;
  apiKey: string;
  isEncrypted: boolean;
}

interface APIKeysModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (entries: APIKeyEntry[]) => void;
  initialEntries?: APIKeyEntry[];
}

const providers = [
  { value: 'openai', label: 'OpenAI', requiresOAuth: false },
  { value: 'openai-codex', label: 'OpenAI Codex (ChatGPT Plus/Pro subscription)', requiresOAuth: true },
  { value: 'anthropic', label: 'Anthropic', requiresOAuth: false },
  { value: 'google', label: 'Google', requiresOAuth: false },
  { value: 'vertex-ai', label: 'Vertex AI (Gemini via Vertex AI)', requiresOAuth: false },
  { value: 'mistral', label: 'Mistral', requiresOAuth: false },
  { value: 'groq', label: 'Groq', requiresOAuth: false },
  { value: 'cerebras', label: 'Cerebras', requiresOAuth: false },
  { value: 'xai', label: 'xAI', requiresOAuth: false },
  { value: 'openrouter', label: 'OpenRouter', requiresOAuth: false },
  { value: 'zai', label: 'zAI (requires ZAI_API_KEY)', requiresOAuth: false },
  { value: 'minimax', label: 'MiniMax Coding Plan (requires MINIMAX_CODE_API_KEY or MINIMAX_CODE_CN_API_KEY)', requiresOAuth: false },
  { value: 'github-copilot', label: 'GitHub Copilot', requiresOAuth: true },
  { value: 'google-gemini-cli', label: 'Google Gemini CLI', requiresOAuth: true },
  { value: 'antigravity', label: 'Antigravity', requiresOAuth: true },
  { value: 'openai-compatible', label: 'Any OpenAI-compatible API (Ollama, vLLM, LM Studio, etc.)', requiresOAuth: false },
];

export function APIKeysModal({ isOpen, onClose, onSave, initialEntries = [] }: APIKeysModalProps) {
  const [entries, setEntries] = useState<APIKeyEntry[]>(initialEntries.length > 0 ? initialEntries : [
    { id: crypto.randomUUID(), provider: '', apiKey: '', isEncrypted: false }
  ]);
  const [visibleKeys, setVisibleKeys] = useState<Set<string>>(new Set());
  const [errors, setErrors] = useState<Record<string, string>>({});

  const addEntry = () => {
    setEntries([...entries, { 
      id: crypto.randomUUID(), 
      provider: '', 
      apiKey: '', 
      isEncrypted: false 
    }]);
  };

  const removeEntry = (id: string) => {
    if (entries.length > 1) {
      setEntries(entries.filter(e => e.id !== id));
    }
  };

  const updateEntry = (id: string, field: keyof APIKeyEntry, value: string | boolean) => {
    setEntries(entries.map(entry => 
      entry.id === id ? { ...entry, [field]: value } : entry
    ));
    // Clear error when user types
    if (errors[id]) {
      setErrors(prev => { const newErrors = { ...prev }; delete newErrors[id]; return newErrors; });
    }
  };

  const toggleKeyVisibility = (id: string) => {
    setVisibleKeys(prev => {
      const newSet = new Set(prev);
      if (newSet.has(id)) {
        newSet.delete(id);
      } else {
        newSet.add(id);
      }
      return newSet;
    });
  };

  const validateEntries = (): boolean => {
    const newErrors: Record<string, string> = {};
    let isValid = true;

    entries.forEach(entry => {
      if (!entry.provider) {
        newErrors[entry.id] = 'Please select a provider';
        isValid = false;
      } else if (!entry.apiKey.trim()) {
        newErrors[entry.id] = 'Please enter an API key';
        isValid = false;
      }
    });

    setErrors(newErrors);
    return isValid;
  };

  const handleSave = () => {
    if (validateEntries()) {
      // Mark all entries as encrypted before saving
      const encryptedEntries = entries.map(entry => ({
        ...entry,
        isEncrypted: true
      }));
      onSave(encryptedEntries);
      onClose();
    }
  };

  const getProviderLabel = (value: string) => {
    return providers.find(p => p.value === value)?.label || value;
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-w-2xl bg-[#0a0a0c] border-white/10 text-slate-200 max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <div className="flex items-center gap-3">
            <div className="p-2 bg-blue-600/20 rounded-lg">
              <Key className="text-blue-500" size={24} />
            </div>
            <div>
              <DialogTitle className="text-xl font-bold text-white">API Keys</DialogTitle>
              <DialogDescription className="text-slate-400 text-sm">
                Configure secure LLM and service credentials for your fabric environment.
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <div className="space-y-6 py-4">
          {entries.map((entry, index) => (
            <div 
              key={entry.id} 
              className="p-4 bg-white/5 border border-white/10 rounded-xl space-y-4"
            >
              <div className="flex items-center justify-between">
                <span className="text-xs uppercase font-bold text-blue-500 tracking-wider">
                  Provider {index + 1}
                </span>
                {entries.length > 1 && (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => removeEntry(entry.id)}
                    className="text-red-400 hover:text-red-300 hover:bg-red-500/10"
                  >
                    <Trash2 size={16} />
                  </Button>
                )}
              </div>

              <div className="space-y-4">
                <div className="space-y-2">
                  <Label className="text-xs uppercase font-bold text-slate-500 tracking-wider">
                    Provider
                  </Label>
                  <Select
                    value={entry.provider}
                    onValueChange={(value) => updateEntry(entry.id, 'provider', value)}
                  >
                    <SelectTrigger className="bg-black/40 border-white/10 text-white h-11">
                      <SelectValue placeholder="Select a provider..." />
                    </SelectTrigger>
                    <SelectContent className="bg-[#1a1a1c] border-white/10 max-h-[300px]">
                      {providers.map((provider) => (
                        <SelectItem 
                          key={provider.value} 
                          value={provider.value}
                          className="text-slate-200 focus:bg-blue-500/20 focus:text-white"
                        >
                          <div className="flex items-center gap-2">
                            <span>{provider.label}</span>
                            {provider.requiresOAuth && (
                              <span className="text-[10px] bg-amber-500/20 text-amber-400 px-1.5 py-0.5 rounded">
                                OAuth
                              </span>
                            )}
                          </div>
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>

                {entry.provider && (
                  <div className="space-y-2">
                    <Label className="text-xs uppercase font-bold text-slate-500 tracking-wider">
                      API Key
                    </Label>
                    <div className="relative">
                      <Input
                        type={visibleKeys.has(entry.id) ? 'text' : 'password'}
                        value={entry.apiKey}
                        onChange={(e) => updateEntry(entry.id, 'apiKey', e.target.value)}
                        placeholder="Enter your API key..."
                        className="bg-black/40 border-white/10 text-white pr-20 h-11 font-mono"
                      />
                      <button
                        type="button"
                        onClick={() => toggleKeyVisibility(entry.id)}
                        className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-500 hover:text-slate-300 transition-colors"
                      >
                        {visibleKeys.has(entry.id) ? <EyeOff size={18} /> : <Eye size={18} />}
                      </button>
                    </div>
                    <p className="text-[10px] text-slate-500">
                      Your API key will be encrypted and stored securely.
                    </p>
                  </div>
                )}

                {errors[entry.id] && (
                  <div className="flex items-center gap-2 text-red-400 text-sm">
                    <AlertCircle size={16} />
                    <span>{errors[entry.id]}</span>
                  </div>
                )}

                {entry.provider && providers.find(p => p.value === entry.provider)?.requiresOAuth && (
                  <div className="p-3 bg-amber-500/10 border border-amber-500/20 rounded-lg">
                    <p className="text-xs text-amber-400">
                      This provider requires OAuth authentication. You will be prompted to authenticate after saving.
                    </p>
                  </div>
                )}
              </div>
            </div>
          ))}

          <Button
            variant="outline"
            onClick={addEntry}
            className="w-full border-dashed border-white/20 text-slate-400 hover:text-white hover:border-blue-500/50 hover:bg-blue-500/10"
          >
            <Plus size={18} className="mr-2" />
            Add Another Provider
          </Button>
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
            Save API Keys
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
