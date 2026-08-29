'use client';

import React, { useState, useCallback } from 'react';
import { useDVEManagement } from '@/hooks/use-dve-management';
import { useDVESessions } from '@/hooks/use-dve-sessions';
import type { DVECreation, DVEModelPolicy } from '@/types/api';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Switch } from '@/components/ui/switch';
import { useToast } from '@/hooks/use-toast';
import {
  Shield,
  Server,
  Activity,
  Eye,
  EyeOff,
  ChevronDown,
  ChevronRight,
  Settings,
  AlertTriangle,
  CheckCircle,
  Clock
} from 'lucide-react';

interface ExpertAdvisorPanelProps {
  className?: string;
  onDrillDownToNodes?: () => void;
}

const DEFAULT_POLICY: DVEModelPolicy = {
  mode: "observe",
  allowed_providers: [],
  allowed_models: [],
  denied_models: [],
  max_requests_per_hour: 1000,
  max_token_budget_daily: 100000,
  fail_open: true,
  updated_at: new Date().toISOString()
};

export function ExpertAdvisorPanel({ className, onDrillDownToNodes }: ExpertAdvisorPanelProps) {
  const { creations, isLoading, error, updatePolicy } = useDVEManagement();
  const { toast } = useToast();
  const [expandedCards, setExpandedCards] = useState<Set<string>>(new Set());
  const [editingPolicies, setEditingPolicies] = useState<Map<string, DVEModelPolicy>>(new Map());
  const [savingPolicies, setSavingPolicies] = useState<Set<string>>(new Set());

  const toggleCard = useCallback((creationId: string) => {
    setExpandedCards(prev => {
      const next = new Set(prev);
      if (next.has(creationId)) {
        next.delete(creationId);
      } else {
        next.add(creationId);
      }
      return next;
    });
  }, []);

  const getInitialPolicy = useCallback((creation: DVECreation): DVEModelPolicy => {
    const existing = editingPolicies.get(creation.id);
    if (existing) return existing;
    return { ...DEFAULT_POLICY, ...creation.policy };
  }, [editingPolicies]);

  const updateLocalPolicy = useCallback((creationId: string, patch: Partial<DVEModelPolicy>) => {
    setEditingPolicies(prev => {
      const creation = creations.find(item => item.id === creationId);
      const current = prev.get(creationId) || (creation ? getInitialPolicy(creation) : { ...DEFAULT_POLICY });
      const updated = { ...current, ...patch, updated_at: new Date().toISOString() };
      const next = new Map(prev);
      next.set(creationId, updated);
      return next;
    });
  }, [creations, getInitialPolicy]);

  const handleSavePolicy = useCallback(async (creation: DVECreation) => {
    const policy = editingPolicies.get(creation.id) || getInitialPolicy(creation);
    setSavingPolicies(prev => new Set(prev).add(creation.id));
    try {
      await updatePolicy(creation.id, policy);
      toast({
        title: "Policy Updated",
        description: `Advisor policy for "${creation.name}" saved successfully.`,
      });
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to update policy';
      toast({
        title: "Policy Update Failed",
        description: message,
        variant: "destructive",
      });
    } finally {
      setSavingPolicies(prev => {
        const next = new Set(prev);
        next.delete(creation.id);
        return next;
      });
    }
  }, [editingPolicies, getInitialPolicy, updatePolicy, toast]);

  const handleChipInput = useCallback((creationId: string, field: 'allowed_providers' | 'allowed_models' | 'denied_models', value: string) => {
    const items = value.split(',').map(s => s.trim()).filter(Boolean);
    updateLocalPolicy(creationId, { [field]: items });
  }, [updateLocalPolicy]);

  const getStatusBadge = (status: string) => {
    switch (status) {
      case "active":
        return <Badge className="bg-green-500"><CheckCircle className="w-3 h-3 mr-1" /> Active</Badge>;
      case "pending":
        return <Badge className="bg-yellow-500"><Clock className="w-3 h-3 mr-1" /> Pending</Badge>;
      case "suspended":
        return <Badge className="bg-red-500"><AlertTriangle className="w-3 h-3 mr-1" /> Suspended</Badge>;
      case "decommissioned":
        return <Badge variant="secondary"><EyeOff className="w-3 h-3 mr-1" /> Decommissioned</Badge>;
      default:
        return <Badge>{status}</Badge>;
    }
  };

  if (isLoading && creations.length === 0) {
    return (
      <div className={`space-y-4 ${className || ''}`}>
        <div className="text-center py-8">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-400 mx-auto"></div>
          <p className="text-gray-500 mt-2">Loading Expert Advisors...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className={`space-y-4 ${className || ''}`}>
        <Alert variant="destructive">
          <AlertTriangle className="w-4 h-4" />
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      </div>
    );
  }

  return (
    <div className={`space-y-4 ${className || ''}`}>
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-gray-200">Expert Advisors</h2>
          <p className="text-gray-500">Manage your sovereign AI advisors and their policies</p>
        </div>
        <div className="flex items-center gap-2">
          {onDrillDownToNodes && (
            <Button
              variant="outline"
              size="sm"
              className="border-gray-700 text-gray-400 hover:bg-indigo-500/10 hover:text-indigo-400 hover:border-indigo-400"
              onClick={onDrillDownToNodes}
            >
              <Server className="w-4 h-4 mr-2" />
              View DVE Nodes
            </Button>
          )}
          <Badge variant="outline" className="text-gray-400">
            {creations.length} advisor{creations.length !== 1 ? 's' : ''}
          </Badge>
        </div>
      </div>

      {creations.length === 0 ? (
        <Card className="aether-bevel-dark rounded-2xl">
          <CardContent className="py-8 text-center">
            <Shield className="w-12 h-12 text-gray-600 mx-auto mb-4" />
            <p className="text-gray-400 font-medium">No Expert Advisors yet</p>
            <p className="text-gray-500 text-sm mt-1">Create a DVE to deploy your first advisor instance</p>
          </CardContent>
        </Card>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          {creations.map((creation) => {
            const isExpanded = expandedCards.has(creation.id);
            const currentPolicy = editingPolicies.get(creation.id) || getInitialPolicy(creation);
            const isSaving = savingPolicies.has(creation.id);

            return (
              <Card key={creation.id} className="aether-bevel-dark rounded-2xl">
                <CardHeader className="pb-3">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 rounded-xl aether-bevel-dark flex items-center justify-center">
                        <Shield className="w-5 h-5 text-indigo-400" />
                      </div>
                      <div>
                        <CardTitle className="text-gray-200 text-base">{creation.name}</CardTitle>
                        <CardDescription className="text-gray-500 text-xs">
                          {creation.id.slice(0, 8)}... · Owner: {creation.owner_id.slice(0, 8)}...
                        </CardDescription>
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      {getStatusBadge(creation.status)}
                      <Button
                        variant="ghost"
                        size="sm"
                        aria-label={isExpanded ? `Collapse ${creation.name}` : `Expand ${creation.name}`}
                        className="h-8 w-8 p-0 text-gray-400 hover:text-indigo-400"
                        onClick={() => toggleCard(creation.id)}
                      >
                        {isExpanded ? <ChevronDown className="w-4 h-4" /> : <ChevronRight className="w-4 h-4" />}
                      </Button>
                    </div>
                  </div>
                </CardHeader>

                <CardContent className="space-y-3">
                  <div className="grid grid-cols-2 gap-3 text-sm">
                    <div>
                      <span className="text-gray-500">Node ID</span>
                      <p className="font-mono text-gray-300 text-xs truncate">{creation.dve_node_id}</p>
                    </div>
                    <div>
                      <span className="text-gray-500">TEE Type</span>
                      <p className="text-gray-300 text-xs">{creation.tee_type}</p>
                    </div>
                    <div>
                      <span className="text-gray-500">Stake</span>
                      <p className="text-gray-300 text-xs">{creation.stake_amount.toLocaleString()} NRN</p>
                    </div>
                    <div>
                      <span className="text-gray-500">Persistent</span>
                      <p className="text-gray-300 text-xs">{creation.persistent ? 'Yes' : 'Rental'}</p>
                    </div>
                  </div>

                  {isExpanded && (
                    <div className="space-y-4 pt-3 border-t border-gray-800">
                      {/* Sessions */}
                      <AdvisorSessions creationId={creation.id} />

                      {/* Policy Editor */}
                      <div className="space-y-3">
                        <div className="flex items-center gap-2">
                          <Settings className="w-4 h-4 text-indigo-400" />
                          <span className="text-sm font-medium text-gray-300">Advisor Policy</span>
                        </div>

                        <div className="grid grid-cols-2 gap-3">
                          <div>
                            <Label className="text-gray-400 text-xs">Mode</Label>
                            <Select
                              value={currentPolicy.mode}
                              onValueChange={(value) => updateLocalPolicy(creation.id, { mode: value })}
                            >
                              <SelectTrigger className="aether-bevel-dark-light border-gray-700 text-gray-200 mt-1">
                                <SelectValue />
                              </SelectTrigger>
                              <SelectContent>
                                <SelectItem value="observe">Observe</SelectItem>
                                <SelectItem value="advise">Advise</SelectItem>
                                <SelectItem value="restrict">Restrict</SelectItem>
                                <SelectItem value="enforce">Enforce</SelectItem>
                              </SelectContent>
                            </Select>
                          </div>

                          <div className="flex items-center justify-between">
                            <div>
                              <Label className="text-gray-400 text-xs">Fail Open</Label>
                              <p className="text-gray-500 text-[10px]">Allow on policy error</p>
                            </div>
                            <Switch
                              checked={currentPolicy.fail_open}
                              onCheckedChange={(checked) => updateLocalPolicy(creation.id, { fail_open: checked })}
                            />
                          </div>
                        </div>

                        <div>
                          <Label className="text-gray-400 text-xs">Allowed Providers (comma-separated)</Label>
                          <Input
                            value={(currentPolicy.allowed_providers || []).join(', ')}
                            onChange={(e) => handleChipInput(creation.id, 'allowed_providers', e.target.value)}
                            placeholder="e.g. openai, anthropic"
                            className="aether-bevel-dark-light border-gray-700 text-gray-200 text-xs mt-1"
                          />
                        </div>

                        <div>
                          <Label className="text-gray-400 text-xs">Allowed Models (comma-separated)</Label>
                          <Input
                            value={(currentPolicy.allowed_models || []).join(', ')}
                            onChange={(e) => handleChipInput(creation.id, 'allowed_models', e.target.value)}
                            placeholder="e.g. gpt-4, claude-3"
                            className="aether-bevel-dark-light border-gray-700 text-gray-200 text-xs mt-1"
                          />
                        </div>

                        <div>
                          <Label className="text-gray-400 text-xs">Denied Models (comma-separated)</Label>
                          <Input
                            value={(currentPolicy.denied_models || []).join(', ')}
                            onChange={(e) => handleChipInput(creation.id, 'denied_models', e.target.value)}
                            placeholder="e.g. gpt-3.5-turbo"
                            className="aether-bevel-dark-light border-gray-700 text-gray-200 text-xs mt-1"
                          />
                        </div>

                        <div className="grid grid-cols-2 gap-2">
                          <div>
                            <Label className="text-gray-400 text-xs">Max Req/hr</Label>
                            <Input
                              type="number"
                              value={currentPolicy.max_requests_per_hour}
                              onChange={(e) => updateLocalPolicy(creation.id, { max_requests_per_hour: parseInt(e.target.value) || 0 })}
                              className="aether-bevel-dark-light border-gray-700 text-gray-200 text-xs mt-1"
                            />
                          </div>
                          <div>
                            <Label className="text-gray-400 text-xs">Max Token Budget/day</Label>
                            <Input
                              type="number"
                              value={currentPolicy.max_token_budget_daily}
                              onChange={(e) => updateLocalPolicy(creation.id, { max_token_budget_daily: parseInt(e.target.value) || 0 })}
                              className="aether-bevel-dark-light border-gray-700 text-gray-200 text-xs mt-1"
                            />
                          </div>
                        </div>

                        <Button
                          size="sm"
                          className="w-full bg-indigo-600 hover:bg-indigo-500 text-white"
                          onClick={() => handleSavePolicy(creation)}
                          disabled={isSaving}
                        >
                          {isSaving ? 'Saving...' : 'Save Policy'}
                        </Button>
                      </div>
                    </div>
                  )}
                </CardContent>
              </Card>
            );
          })}
        </div>
      )}
    </div>
  );
}

function AdvisorSessions({ creationId }: { creationId: string }) {
  const { sessions, loading, error } = useDVESessions(creationId);
  const [showSessions, setShowSessions] = useState(false);

  if (!showSessions) {
    return (
      <Button
        variant="ghost"
        size="sm"
        className="text-xs text-gray-400 hover:text-indigo-400 p-0 h-auto"
        onClick={() => setShowSessions(true)}
      >
        <Activity className="w-3 h-3 mr-1" />
        View Sessions ({sessions.length})
      </Button>
    );
  }

  if (loading) {
    return (
      <div className="flex items-center gap-2 text-xs text-gray-500">
        <div className="animate-spin rounded-full h-3 w-3 border-b border-indigo-400"></div>
        Loading sessions...
      </div>
    );
  }

  if (error) {
    return (
      <div className="text-xs text-red-400">Failed to load sessions: {error}</div>
    );
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <span className="text-xs font-medium text-gray-400">Sessions ({sessions.length})</span>
        <Button
          variant="ghost"
          size="sm"
          className="text-xs text-gray-400 hover:text-indigo-400 p-0 h-auto"
          onClick={() => setShowSessions(false)}
        >
          Hide
        </Button>
      </div>
      {sessions.length === 0 ? (
        <p className="text-xs text-gray-500">No active sessions</p>
      ) : (
        <div className="space-y-1 max-h-40 overflow-y-auto">
          {sessions.map((session) => (
            <div key={session.id} className="flex items-center justify-between p-2 rounded-lg bg-black/30 text-xs">
              <div className="flex items-center gap-2">
                <div className="w-2 h-2 rounded-full bg-green-500" />
                <span className="font-mono text-gray-300">{session.id.slice(0, 12)}...</span>
              </div>
              <Badge variant="outline" className="text-[10px]">
                {session.status}
              </Badge>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export default ExpertAdvisorPanel;
