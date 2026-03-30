'use client';

import React, { useState, useEffect } from 'react';
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
import { Terminal, Plus, Trash2, Server, ExternalLink, Sparkles, AlertCircle } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";

export interface MCPServerEntry {
  id: string;
  name: string;
  description: string;
  url?: string;
  category?: string;
}

interface MCPServersModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (servers: MCPServerEntry[]) => void;
  initialServers?: MCPServerEntry[];
}

// MCP Server from registry
interface RegistryServer {
  id: string;
  name: string;
  description: string;
  category?: string;
  url?: string;
}

export function MCPServersModal({ isOpen, onClose, onSave, initialServers = [] }: MCPServersModalProps) {
  const [servers, setServers] = useState<MCPServerEntry[]>(initialServers);
  const [registryServers, setRegistryServers] = useState<RegistryServer[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [showPaidUpgrade, setShowPaidUpgrade] = useState(false);
  const [selectedRegistryServers, setSelectedRegistryServers] = useState<Set<string>>(new Set());
  const [customServer, setCustomServer] = useState({ name: '', description: '', url: '' });
  const [showCustomForm, setShowCustomForm] = useState(false);

  // Fetch MCP servers from registry
  useEffect(() => {
    if (isOpen) {
      fetchMCPServers();
    }
  }, [isOpen]);

  const fetchMCPServers = async () => {
    setIsLoading(true);
    try {
      // Fetch from MCP registry API
      const response = await fetch('https://registry.modelcontextprotocol.io/servers');
      if (response.ok) {
        const data = await response.json();
        // Map the registry data to our format
        const mappedServers: RegistryServer[] = data.servers?.map((server: any) => ({
          id: server.id || server.name,
          name: server.name,
          description: server.description || 'No description available',
          category: server.category || 'General',
          url: server.url || server.repository
        })) || [];
        setRegistryServers(mappedServers);
      } else {
        // Fallback to sample data if API fails
        setRegistryServers([
          { id: 'filesystem', name: 'Filesystem', description: 'Access and manage files on the local file system', category: 'Utility' },
          { id: 'github', name: 'GitHub', description: 'Repository management, code search, and file operations', category: 'Development' },
          { id: 'postgres', name: 'PostgreSQL', description: 'Read-only database access with schema inspection', category: 'Database' },
          { id: 'slack', name: 'Slack', description: 'Channel management and messaging capabilities', category: 'Communication' },
          { id: 'puppeteer', name: 'Puppeteer', description: 'Browser automation and web scraping', category: 'Automation' },
          { id: 'aws-kb-retrieval', name: 'AWS KB Retrieval', description: 'Retrieval from AWS Knowledge Base', category: 'Cloud' },
          { id: 'brave-search', name: 'Brave Search', description: 'Web search capabilities via Brave Search API', category: 'Search' },
        ]);
      }
    } catch (error) {
      console.error('Failed to fetch MCP servers:', error);
      // Fallback data
      setRegistryServers([
        { id: 'filesystem', name: 'Filesystem', description: 'Access and manage files on the local file system', category: 'Utility' },
        { id: 'github', name: 'GitHub', description: 'Repository management, code search, and file operations', category: 'Development' },
        { id: 'postgres', name: 'PostgreSQL', description: 'Read-only database access with schema inspection', category: 'Database' },
        { id: 'slack', name: 'Slack', description: 'Channel management and messaging capabilities', category: 'Communication' },
      ]);
    } finally {
      setIsLoading(false);
    }
  };

  const toggleRegistryServer = (serverId: string) => {
    setSelectedRegistryServers(prev => {
      const newSet = new Set(prev);
      if (newSet.has(serverId)) {
        newSet.delete(serverId);
      } else {
        newSet.add(serverId);
      }
      return newSet;
    });
  };

  const addCustomServer = () => {
    if (customServer.name && customServer.description) {
      const newServer: MCPServerEntry = {
        id: crypto.randomUUID(),
        name: customServer.name,
        description: customServer.description,
        url: customServer.url,
        category: 'Custom'
      };
      setServers([...servers, newServer]);
      setCustomServer({ name: '', description: '', url: '' });
      setShowCustomForm(false);
    }
  };

  const removeServer = (id: string) => {
    setServers(servers.filter(s => s.id !== id));
  };

  const handleSave = () => {
    // Add selected registry servers to the list
    const selectedServers = registryServers
      .filter(s => selectedRegistryServers.has(s.id))
      .map(s => ({
        id: s.id,
        name: s.name,
        description: s.description,
        url: s.url,
        category: s.category
      }));
    
    const allServers = [...servers, ...selectedServers];
    onSave(allServers);
    onClose();
  };

  const totalSelected = servers.length + selectedRegistryServers.size;

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-w-3xl bg-[#0a0a0c] border-white/10 text-slate-200 max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <div className="flex items-center gap-3">
            <div className="p-2 bg-blue-600/20 rounded-lg">
              <Terminal className="text-blue-500" size={24} />
            </div>
            <div>
              <DialogTitle className="text-xl font-bold text-white">MCP Servers</DialogTitle>
              <DialogDescription className="text-slate-400 text-sm">
                Model Context Protocol servers provide capabilities to the cognitive shell of your fabric environment.
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <div className="space-y-6 py-4">
          {/* Info Banner */}
          <div className="p-4 bg-blue-500/5 border border-blue-500/20 rounded-xl">
            <p className="text-sm text-slate-300 leading-relaxed">
              MCP servers extend your agent&apos;s capabilities by providing tools for file system access, 
              API integrations, database connections, and more. Select from the registry below or add custom servers.
            </p>
          </div>

          {/* Paid Plan Upgrade Banner */}
          <div className="p-4 bg-gradient-to-r from-amber-500/10 to-orange-500/10 border border-amber-500/20 rounded-xl">
            <div className="flex items-start gap-3">
              <Sparkles className="text-amber-500 shrink-0 mt-0.5" size={20} />
              <div className="flex-1">
                <h4 className="font-bold text-amber-400 mb-1">MCP Serving Service</h4>
                <p className="text-sm text-slate-400 mb-3">
                  Host and serve your own MCP servers with our managed infrastructure. 
                  Available with paid plan upgrade.
                </p>
                <Button 
                  size="sm" 
                  variant="outline" 
                  className="border-amber-500/30 text-amber-400 hover:bg-amber-500/10"
                  onClick={() => setShowPaidUpgrade(true)}
                >
                  View Plans
                </Button>
              </div>
            </div>
          </div>

          {/* Registry Servers */}
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <Label className="text-xs uppercase font-bold text-slate-500 tracking-wider">
                Available Servers from Registry
              </Label>
              <Badge variant="outline" className="text-slate-400 border-white/10">
                {selectedRegistryServers.size} selected
              </Badge>
            </div>
            
            <ScrollArea className="h-[200px] border border-white/10 rounded-xl">
              <div className="p-4 space-y-2">
                {isLoading ? (
                  <div className="flex items-center justify-center h-32 text-slate-500">
                    <Server className="animate-pulse mr-2" size={20} />
                    Loading registry...
                  </div>
                ) : registryServers.length === 0 ? (
                  <div className="flex items-center justify-center h-32 text-slate-500">
                    <AlertCircle className="mr-2" size={20} />
                    No servers available
                  </div>
                ) : (
                  registryServers.map((server) => (
                    <div
                      key={server.id}
                      onClick={() => toggleRegistryServer(server.id)}
                      className={`p-3 rounded-lg border cursor-pointer transition-interactive ${
                        selectedRegistryServers.has(server.id)
                          ? 'bg-blue-500/10 border-blue-500'
                          : 'bg-white/5 border-white/10 hover:border-white/20'
                      }`}
                    >
                      <div className="flex items-start justify-between">
                        <div className="flex-1">
                          <div className="flex items-center gap-2 mb-1">
                            <h4 className="font-bold text-white">{server.name}</h4>
                            {server.category && (
                              <Badge variant="outline" className="text-[10px] border-white/10 text-slate-400">
                                {server.category}
                              </Badge>
                            )}
                          </div>
                          <p className="text-sm text-slate-400">{server.description}</p>
                        </div>
                        <div className={`w-5 h-5 rounded-full border-2 flex items-center justify-center ${
                          selectedRegistryServers.has(server.id)
                            ? 'bg-blue-500 border-blue-500'
                            : 'border-white/30'
                        }`}>
                          {selectedRegistryServers.has(server.id) && (
                            <svg className="w-3 h-3 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M5 13l4 4L19 7" />
                            </svg>
                          )}
                        </div>
                      </div>
                    </div>
                  ))
                )}
              </div>
            </ScrollArea>
            
            <a 
              href="https://registry.modelcontextprotocol.io/docs#/operations/list-servers-v0.1" 
              target="_blank" 
              rel="noopener noreferrer"
              className="text-xs text-blue-400 hover:text-blue-300 flex items-center gap-1"
            >
              View full registry documentation
              <ExternalLink size={12} />
            </a>
          </div>

          {/* Custom Servers Section */}
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <Label className="text-xs uppercase font-bold text-slate-500 tracking-wider">
                Custom Servers
              </Label>
              <Badge variant="outline" className="text-slate-400 border-white/10">
                {servers.length} added
              </Badge>
            </div>

            {servers.length > 0 && (
              <div className="space-y-2">
                {servers.map((server) => (
                  <div
                    key={server.id}
                    className="p-3 bg-white/5 border border-white/10 rounded-lg flex items-start justify-between"
                  >
                    <div>
                      <div className="flex items-center gap-2">
                        <h4 className="font-bold text-white">{server.name}</h4>
                        <Badge variant="outline" className="text-[10px] border-white/10 text-slate-400">
                          Custom
                        </Badge>
                      </div>
                      <p className="text-sm text-slate-400">{server.description}</p>
                      {server.url && (
                        <p className="text-xs text-slate-500 mt-1 font-mono">{server.url}</p>
                      )}
                    </div>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => removeServer(server.id)}
                      className="text-red-400 hover:text-red-300 hover:bg-red-500/10"
                    >
                      <Trash2 size={16} />
                    </Button>
                  </div>
                ))}
              </div>
            )}

            {/* Add Custom Server Form */}
            {showCustomForm ? (
              <div className="p-4 bg-white/5 border border-white/10 rounded-xl space-y-4">
                <div className="space-y-2">
                  <Label className="text-xs uppercase font-bold text-slate-500 tracking-wider">
                    Server Name
                  </Label>
                  <Input
                    value={customServer.name}
                    onChange={(e) => setCustomServer({ ...customServer, name: e.target.value })}
                    placeholder="e.g. My Custom Server"
                    className="bg-black/40 border-white/10 text-white"
                  />
                </div>
                <div className="space-y-2">
                  <Label className="text-xs uppercase font-bold text-slate-500 tracking-wider">
                    Description
                  </Label>
                  <Input
                    value={customServer.description}
                    onChange={(e) => setCustomServer({ ...customServer, description: e.target.value })}
                    placeholder="What does this server do?"
                    className="bg-black/40 border-white/10 text-white"
                  />
                </div>
                <div className="space-y-2">
                  <Label className="text-xs uppercase font-bold text-slate-500 tracking-wider">
                    URL (Optional)
                  </Label>
                  <Input
                    value={customServer.url}
                    onChange={(e) => setCustomServer({ ...customServer, url: e.target.value })}
                    placeholder="https://..."
                    className="bg-black/40 border-white/10 text-white font-mono"
                  />
                </div>
                <div className="flex gap-2">
                  <Button
                    variant="outline"
                    onClick={() => setShowCustomForm(false)}
                    className="flex-1 border-white/20 text-slate-400"
                  >
                    Cancel
                  </Button>
                  <Button
                    onClick={addCustomServer}
                    disabled={!customServer.name || !customServer.description}
                    className="flex-1 bg-blue-600 hover:bg-blue-500 text-white"
                  >
                    Add Server
                  </Button>
                </div>
              </div>
            ) : (
              <Button
                variant="outline"
                onClick={() => setShowCustomForm(true)}
                className="w-full border-dashed border-white/20 text-slate-400 hover:text-white hover:border-blue-500/50 hover:bg-blue-500/10"
              >
                <Plus size={18} className="mr-2" />
                Add Custom Server
              </Button>
            )}
          </div>

          {totalSelected > 0 && (
            <div className="p-3 bg-green-500/10 border border-green-500/20 rounded-lg">
              <p className="text-sm text-green-400 text-center">
                {totalSelected} server{totalSelected !== 1 ? 's' : ''} will be configured
              </p>
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
            disabled={totalSelected === 0}
            className="bg-blue-600 hover:bg-blue-500 text-white"
          >
            Save MCP Servers
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
