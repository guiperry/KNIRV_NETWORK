'use client';

import React, { useEffect, useState } from 'react';
import { X, Download, Play, Loader2, CheckCircle2, AlertCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { type InnerAgentTool, useInnerAgent } from '@/hooks/use-inner-agent';

interface ToolPickerModalProps {
  isOpen: boolean;
  onClose: () => void;
  dveId: string | undefined;
  onSpawned: (sessionId: string, toolName: string) => void;
}

const TOOL_DESCRIPTIONS: Record<string, string> = {
  claude: 'Anthropic Claude Code — AI coding assistant with full repo context',
  aider: 'Aider — pair programming with LLMs in your terminal',
  codex: 'OpenAI Codex CLI — AI agent for your terminal',
  'gh-copilot': 'GitHub Copilot CLI — AI coding assistant via gh extension',
};

export function ToolPickerModal({ isOpen, onClose, dveId, onSpawned }: ToolPickerModalProps) {
  const { tools, loading, error, listTools, installTool, spawnSession } = useInnerAgent(dveId);
  const [actionState, setActionState] = useState<Record<string, 'idle' | 'installing' | 'spawning' | 'done' | 'error'>>({});

  useEffect(() => {
    if (isOpen && dveId) {
      listTools();
    }
  }, [isOpen, dveId]);

  const handleInstall = async (tool: InnerAgentTool) => {
    setActionState(s => ({ ...s, [tool.name]: 'installing' }));
    const ok = await installTool(tool.name);
    setActionState(s => ({ ...s, [tool.name]: ok ? 'done' : 'error' }));
    if (ok) setTimeout(() => setActionState(s => ({ ...s, [tool.name]: 'idle' })), 1500);
  };

  const handleSpawn = async (tool: InnerAgentTool) => {
    setActionState(s => ({ ...s, [tool.name]: 'spawning' }));
    const sessionId = await spawnSession(tool.name);
    if (sessionId) {
      setActionState(s => ({ ...s, [tool.name]: 'done' }));
      onSpawned(sessionId, tool.name);
      onClose();
    } else {
      setActionState(s => ({ ...s, [tool.name]: 'error' }));
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-[100] flex items-center justify-center bg-black/60" onClick={onClose}>
      <div
        className="w-full max-w-md bg-[#0d1117] border border-purple-900/40 rounded-xl shadow-2xl"
        onClick={e => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-slate-800">
          <div>
            <h2 className="text-sm font-bold text-slate-100">Spawn Inner Agent</h2>
            <p className="text-[11px] text-slate-500 mt-0.5">Select a tool to run inside the DVE workspace</p>
          </div>
          <button onClick={onClose} className="text-slate-500 hover:text-slate-300 transition-colors">
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Body */}
        <div className="p-4 space-y-2">
          {loading && tools.length === 0 && (
            <div className="flex items-center justify-center py-8 text-slate-500">
              <Loader2 className="w-5 h-5 animate-spin mr-2" />
              <span className="text-sm">Loading tools...</span>
            </div>
          )}
          {error && (
            <div className="flex items-center gap-2 px-3 py-2 bg-red-900/20 border border-red-800/40 rounded-lg text-xs text-red-400">
              <AlertCircle className="w-4 h-4 shrink-0" />
              {error}
            </div>
          )}
          {tools.map(tool => {
            const state = actionState[tool.name] ?? 'idle';
            const isActing = state === 'installing' || state === 'spawning';
            return (
              <div
                key={tool.name}
                className="flex items-center justify-between px-4 py-3 rounded-lg bg-slate-900/50 border border-slate-800 hover:border-purple-800/50 transition-colors"
              >
                <div className="flex-1 min-w-0 mr-3">
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-mono font-semibold text-purple-300">{tool.name}</span>
                    {tool.is_installed && (
                      <span className="text-[10px] bg-green-900/30 text-green-400 border border-green-800/40 px-1.5 py-0.5 rounded">
                        installed
                      </span>
                    )}
                  </div>
                  <p className="text-[11px] text-slate-500 mt-0.5 truncate">
                    {TOOL_DESCRIPTIONS[tool.name] ?? tool.binary}
                  </p>
                </div>

                <div className="flex items-center gap-2 shrink-0">
                  {state === 'done' && <CheckCircle2 className="w-4 h-4 text-green-400" />}
                  {state === 'error' && <AlertCircle className="w-4 h-4 text-red-400" />}

                  {!tool.is_installed ? (
                    <Button
                      size="sm"
                      variant="outline"
                      disabled={isActing}
                      onClick={() => handleInstall(tool)}
                      className="h-7 px-3 text-[11px] border-amber-800/40 text-amber-400 hover:bg-amber-900/20"
                    >
                      {state === 'installing' ? (
                        <Loader2 className="w-3 h-3 animate-spin mr-1" />
                      ) : (
                        <Download className="w-3 h-3 mr-1" />
                      )}
                      Install
                    </Button>
                  ) : (
                    <Button
                      size="sm"
                      disabled={isActing}
                      onClick={() => handleSpawn(tool)}
                      className="h-7 px-3 text-[11px] bg-purple-700 hover:bg-purple-600 text-white"
                    >
                      {state === 'spawning' ? (
                        <Loader2 className="w-3 h-3 animate-spin mr-1" />
                      ) : (
                        <Play className="w-3 h-3 mr-1" />
                      )}
                      Launch
                    </Button>
                  )}
                </div>
              </div>
            );
          })}
        </div>

        <div className="px-5 py-3 border-t border-slate-800">
          <p className="text-[10px] text-slate-600">
            Inner agents run inside the DVE enclave workspace and are monitored by the KNIRVAGENT supervisor.
          </p>
        </div>
      </div>
    </div>
  );
}
