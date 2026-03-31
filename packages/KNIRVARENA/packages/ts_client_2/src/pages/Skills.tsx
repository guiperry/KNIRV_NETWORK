import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { Search, Zap, Activity, Plus, Edit2, Trash2, Trophy } from 'lucide-react';
import { SlidingPanel } from '../components/SlidingPanel';
import { NetworkStatus } from '../components/NetworkStatus';
import { AgentManager } from '../components/AgentManager';
import { CognitiveShellInterface } from '../components/CognitiveShellInterface';
import QRScanner from '../components/QRScanner';
import { Agent } from '../types/common';
import { useKnirvana } from '../components/game/stores/useKnirvana';
import type { AgentPersona } from '../services/gameLLMService';

// ── Category badge colours ────────────────────────────────────────────────

const CATEGORY_COLORS: Record<string, string> = {
  analysis:      'bg-blue-500/20 text-blue-300 border-blue-500/30',
  automation:    'bg-purple-500/20 text-purple-300 border-purple-500/30',
  computation:   'bg-green-500/20 text-green-300 border-green-500/30',
  communication: 'bg-yellow-500/20 text-yellow-300 border-yellow-500/30',
};

function categoryOf(persona: AgentPersona): string {
  const p = persona.systemPrompt.toLowerCase();
  if (p.includes('security') || p.includes('debug'))  return 'analysis';
  if (p.includes('automat') || p.includes('workflow')) return 'automation';
  if (p.includes('performance') || p.includes('optim')) return 'computation';
  return 'communication';
}

// ── Persona Card ──────────────────────────────────────────────────────────

interface PersonaCardProps {
  persona: AgentPersona;
  onEdit: (persona: AgentPersona) => void;
  onDelete: (id: string) => void;
}

function PersonaCard({ persona, onEdit, onDelete }: PersonaCardProps) {
  const cat = categoryOf(persona);
  const winPct = persona.totalEpochs > 0
    ? Math.round((persona.wins / persona.totalEpochs) * 100)
    : null;

  return (
    <div className="bg-gray-800/80 border border-gray-600/50 rounded-lg p-4 hover:border-blue-500/40 transition-colors">
      <div className="flex items-start justify-between mb-2">
        <div>
          <h3 className="text-white font-semibold">{persona.name}</h3>
          <span className={`text-xs px-2 py-0.5 rounded border ${CATEGORY_COLORS[cat]}`}>
            {cat}
          </span>
        </div>
        <div className="flex gap-2">
          <button
            onClick={() => onEdit(persona)}
            className="text-gray-400 hover:text-blue-400 transition-colors p-1"
            aria-label="Edit persona"
          >
            <Edit2 className="w-4 h-4" />
          </button>
          <button
            onClick={() => onDelete(persona.id)}
            className="text-gray-400 hover:text-red-400 transition-colors p-1"
            aria-label="Delete persona"
          >
            <Trash2 className="w-4 h-4" />
          </button>
        </div>
      </div>

      <p className="text-gray-400 text-sm mb-3 line-clamp-2">
        {persona.systemPrompt}
      </p>

      <div className="flex items-center justify-between text-xs text-gray-500">
        <div className="flex items-center gap-1">
          <Activity className="w-3 h-3" />
          <span>{persona.totalEpochs} epochs</span>
        </div>
        {winPct !== null ? (
          <div className="flex items-center gap-1 text-yellow-400">
            <Trophy className="w-3 h-3" />
            <span>{winPct}% win rate</span>
          </div>
        ) : (
          <span className="text-gray-600 italic">No data yet</span>
        )}
      </div>

      {persona.totalEpochs > 0 && (
        <div className="mt-2">
          <div className="w-full bg-gray-700 rounded-full h-1.5">
            <div
              className="bg-yellow-500 h-1.5 rounded-full transition-all"
              style={{ width: `${Math.round(persona.winRate * 100)}%` }}
            />
          </div>
        </div>
      )}
    </div>
  );
}

// ── Persona Editor Modal ──────────────────────────────────────────────────

interface PersonaEditorProps {
  persona: Partial<AgentPersona> | null;
  onSave: (persona: AgentPersona) => void;
  onClose: () => void;
}

function PersonaEditor({ persona, onSave, onClose }: PersonaEditorProps) {
  const [name, setName] = useState(persona?.name ?? '');
  const [systemPrompt, setSystemPrompt] = useState(persona?.systemPrompt ?? '');

  const handleSave = () => {
    if (!name.trim() || !systemPrompt.trim()) return;
    onSave({
      id: persona?.id ?? `persona-${Date.now()}`,
      name: name.trim(),
      systemPrompt: systemPrompt.trim(),
      winRate: persona?.winRate ?? 0,
      totalEpochs: persona?.totalEpochs ?? 0,
      wins: persona?.wins ?? 0,
    });
  };

  return (
    <div className="fixed inset-0 bg-black/70 z-[100] flex items-center justify-center p-4">
      <div className="bg-gray-900 border border-gray-600 rounded-xl w-full max-w-lg p-6 space-y-4">
        <h2 className="text-white font-bold text-lg">
          {persona?.id ? 'Edit Skill Persona' : 'New Skill Persona'}
        </h2>

        <div>
          <label className="text-gray-400 text-sm block mb-1">Name</label>
          <input
            value={name}
            onChange={e => setName(e.target.value)}
            placeholder="e.g. Security Analyst"
            className="w-full bg-gray-800 border border-gray-600 rounded-lg px-3 py-2 text-white placeholder-gray-500 focus:border-blue-500 outline-none"
          />
        </div>

        <div>
          <label className="text-gray-400 text-sm block mb-1">
            System Prompt{' '}
            <span className="text-gray-600">(defines this agent's expertise and strategy)</span>
          </label>
          <textarea
            value={systemPrompt}
            onChange={e => setSystemPrompt(e.target.value)}
            rows={6}
            placeholder="You are an expert in ... When given a bug, your approach is ..."
            className="w-full bg-gray-800 border border-gray-600 rounded-lg px-3 py-2 text-white placeholder-gray-500 focus:border-blue-500 outline-none resize-none"
          />
        </div>

        <div className="flex gap-3 justify-end">
          <button
            onClick={onClose}
            className="px-4 py-2 text-gray-400 hover:text-white transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={handleSave}
            disabled={!name.trim() || !systemPrompt.trim()}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-500 disabled:bg-gray-700 disabled:text-gray-500 text-white rounded-lg transition-colors"
          >
            Save Persona
          </button>
        </div>
      </div>
    </div>
  );
}

// ── Main Skills Page ──────────────────────────────────────────────────────

export default function Skills() {
  const navigate = useNavigate();

  // Pull persona state from game store (shared with tournament)
  const personas = useKnirvana(s => s.personas);
  const updatePersona = useKnirvana(s => s.updatePersona);
  const addPersona = useKnirvana(s => s.addPersona);
  const removePersona = useKnirvana(s => s.removePersona);
  const usingMockLLM = useKnirvana(s => s.usingMockLLM);

  const [menuOpen, setMenuOpen] = useState(false);
  const [activePanels, setActivePanels] = useState<string[]>([]);
  const [searchTerm, setSearchTerm] = useState('');
  const [editingPersona, setEditingPersona] = useState<Partial<AgentPersona> | null>(null);
  const [showEditor, setShowEditor] = useState(false);

  const [networkConnections] = useState<{
    [key: string]: 'connected' | 'disconnected' | 'connecting';
  }>({
    knirvChain: 'connected',
    knirvGraph: 'connected',
    knirvNexus: 'connecting',
    knirvGateway: 'disconnected',
  });

  const [availableAgents] = useState<Agent[]>([
    {
      agentId: 'agent-1',
      name: 'CodeT5-Alpha',
      version: '1.0.0',
      type: 'wasm' as const,
      status: 'Available' as const,
      nrnCost: 85,
      capabilities: ['code-generation', 'optimization'],
      metadata: {
        name: 'CodeT5-Alpha',
        version: '1.0.0',
        description: 'Code generation agent',
        author: 'KNIRV Team',
        capabilities: ['code-generation', 'optimization'],
        requirements: { memory: 256, cpu: 2, storage: 50 },
        permissions: ['code-execution', 'file-access'],
      },
      createdAt: new Date().toISOString(),
    },
  ]);

  const [nrnBalance] = useState(1250);

  const filteredPersonas = personas.filter(p =>
    p.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    p.systemPrompt.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const totalWins = personas.reduce((s, p) => s + p.wins, 0);
  const totalEpochs = personas.reduce((s, p) => s + p.totalEpochs, 0);
  const topPersona = personas.reduce<AgentPersona | null>((best, p) =>
    (best === null || p.winRate > best.winRate) && p.totalEpochs > 0 ? p : best, null
  );

  // ── Handlers ────────────────────────────────────────────────────────────

  const handleSavePersona = (persona: AgentPersona) => {
    if (personas.find(p => p.id === persona.id)) {
      updatePersona(persona.id, persona);
    } else {
      addPersona(persona);
    }
    setShowEditor(false);
    setEditingPersona(null);
  };

  const handleEditPersona = (persona: AgentPersona) => {
    setEditingPersona(persona);
    setShowEditor(true);
  };

  const handleDeletePersona = (id: string) => {
    if (personas.length <= 1) {
      alert('You must keep at least one skill persona.');
      return;
    }
    removePersona(id);
  };

  const closePanel = (panelId: string) => setActivePanels(prev => prev.filter(id => id !== panelId));

  const togglePanel = (id: string) =>
    setActivePanels(prev =>
      prev.includes(id) ? prev.filter(x => x !== id) : [...prev, id]
    );

  // ── Burger menu ──────────────────────────────────────────────────────────

  const BurgerMenu = ({ isOpen, onToggle, children }: {
    isOpen: boolean; onToggle: () => void; children: React.ReactNode;
  }) => (
    <div className="relative">
      <button
        onClick={onToggle}
        className="bg-gray-800/80 hover:bg-gray-700/80 text-white p-3 rounded-lg shadow-lg transition-all duration-200 border border-gray-600/50 backdrop-blur-sm"
        aria-label="Navigation menu"
      >
        <div className="w-5 h-5 flex flex-col justify-center items-center gap-1">
          <div className={`w-5 h-0.5 bg-white transition-all ${isOpen ? 'rotate-45 translate-y-1.5' : ''}`} />
          <div className={`w-5 h-0.5 bg-white transition-all ${isOpen ? 'opacity-0' : ''}`} />
          <div className={`w-5 h-0.5 bg-white transition-all ${isOpen ? '-rotate-45 -translate-y-1.5' : ''}`} />
        </div>
      </button>
      {isOpen && (
        <div className="absolute top-full right-0 mt-2 w-56 bg-gray-800/95 backdrop-blur-xl rounded-lg shadow-xl border border-gray-600/50 py-2 z-50">
          {children}
        </div>
      )}
    </div>
  );

  const MenuItem = ({ onClick, icon, children }: {
    onClick: () => void; icon: string; children: React.ReactNode;
  }) => (
    <button
      onClick={onClick}
      className="w-full flex items-center gap-3 px-4 py-3 text-left hover:bg-gray-700/50 transition-colors text-white"
    >
      <span>{icon}</span>
      <span className="font-medium">{children}</span>
    </button>
  );

  return (
    <div className="min-h-screen bg-gray-900 text-white relative overflow-hidden">
      {/* Burger Menu */}
      <div className="absolute top-4 right-4 z-50">
        <BurgerMenu isOpen={menuOpen} onToggle={() => setMenuOpen(!menuOpen)}>
          <MenuItem onClick={() => { navigate('/manager/udc'); setMenuOpen(false); }} icon="🔐">UDC</MenuItem>
          <MenuItem onClick={() => { navigate('/'); setMenuOpen(false); }} icon="🎮">Arena</MenuItem>
          <MenuItem onClick={() => { togglePanel('qr-scanner'); setMenuOpen(false); }} icon="📱">QR Scanner</MenuItem>
          <MenuItem onClick={() => { togglePanel('cognitive-shell'); setMenuOpen(false); }} icon="🧠">Cognitive Shell</MenuItem>
          <MenuItem onClick={() => { togglePanel('network-status'); setMenuOpen(false); }} icon="🌐">Network Status</MenuItem>
          <MenuItem onClick={() => { togglePanel('agent-management'); setMenuOpen(false); }} icon="🤖">Agent Management</MenuItem>
        </BurgerMenu>
      </div>

      <div className="max-w-4xl mx-auto p-4 pb-24 overflow-y-auto h-screen">
        <div className="space-y-6">
          {/* Header */}
          <div className="text-center py-4">
            <h1 className="text-2xl font-bold bg-gradient-to-r from-blue-400 to-cyan-400 bg-clip-text text-transparent mb-1">
              Skill Personas
            </h1>
            <p className="text-gray-400 text-sm">
              Each persona is an LLM system prompt that defines how your agents approach challenges.
              Win rates are tracked across tournament epochs.
            </p>
          </div>

          {/* Mock LLM warning */}
          {usingMockLLM && (
            <div className="bg-yellow-900/30 border border-yellow-500/40 rounded-lg p-4 text-sm">
              <span className="text-yellow-400 font-semibold">Mock Mode Active</span>
              <span className="text-yellow-200/80 ml-2">
                No LLM API keys detected. Tournament runs with deterministic mock scores.
                Set <code className="bg-black/30 px-1 rounded">VITE_OPENAI_API_KEY</code>,{' '}
                <code className="bg-black/30 px-1 rounded">VITE_GOOGLE_API_KEY</code>, or{' '}
                <code className="bg-black/30 px-1 rounded">VITE_DEEPSEEK_API_KEY</code> for real gameplay.
              </span>
            </div>
          )}

          {/* Stats */}
          <div className="grid grid-cols-3 gap-4">
            <div className="text-center p-4 bg-gray-800/80 border border-gray-600/50 rounded-lg">
              <div className="text-2xl font-bold text-white">{personas.length}</div>
              <div className="text-xs text-gray-400">Personas</div>
            </div>
            <div className="text-center p-4 bg-gray-800/80 border border-gray-600/50 rounded-lg">
              <div className="text-2xl font-bold text-yellow-400">{totalWins}</div>
              <div className="text-xs text-gray-400">Total Wins</div>
            </div>
            <div className="text-center p-4 bg-gray-800/80 border border-gray-600/50 rounded-lg">
              <div className="text-2xl font-bold text-cyan-400">{totalEpochs}</div>
              <div className="text-xs text-gray-400">Epochs Run</div>
            </div>
          </div>

          {/* Top performer */}
          {topPersona && (
            <div className="bg-yellow-900/20 border border-yellow-500/30 rounded-lg p-4 flex items-center gap-4">
              <Trophy className="w-8 h-8 text-yellow-400 flex-shrink-0" />
              <div>
                <div className="text-yellow-300 font-semibold">Top Performer: {topPersona.name}</div>
                <div className="text-gray-400 text-sm">
                  {topPersona.wins} wins / {topPersona.totalEpochs} epochs
                  ({Math.round(topPersona.winRate * 100)}% win rate)
                </div>
              </div>
            </div>
          )}

          {/* Search + Add */}
          <div className="flex gap-3">
            <div className="flex-1 relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
              <input
                type="text"
                value={searchTerm}
                onChange={e => setSearchTerm(e.target.value)}
                placeholder="Search personas..."
                className="w-full pl-10 pr-4 py-3 bg-gray-800/80 border border-gray-600/50 rounded-lg focus:border-blue-500/50 focus:outline-none text-white placeholder-gray-400"
              />
            </div>
            <button
              onClick={() => { setEditingPersona(null); setShowEditor(true); }}
              className="flex items-center gap-2 px-4 py-3 bg-blue-600 hover:bg-blue-500 rounded-lg text-white font-medium transition-colors"
            >
              <Plus className="w-4 h-4" />
              New
            </button>
          </div>

          {/* LoRA Status Bar (informational) */}
          <div className="flex items-center justify-between p-3 bg-gray-800/50 rounded-lg border border-gray-600/30">
            <div className="flex items-center gap-3">
              <Activity className="w-4 h-4 text-purple-400" />
              <span className="text-sm text-white">Tournament Engine</span>
            </div>
            <div className="flex items-center gap-4 text-xs text-gray-400">
              <span>Personas: {personas.length}</span>
              <span>LLM: {usingMockLLM ? 'Mock' : 'Live'}</span>
              <span>Avg Win Rate: {personas.length > 0
                ? `${Math.round(personas.reduce((s, p) => s + p.winRate, 0) / personas.length * 100)}%`
                : 'N/A'}</span>
            </div>
          </div>

          {/* Personas Grid */}
          {filteredPersonas.length === 0 ? (
            <div className="text-center py-8">
              <Zap className="w-12 h-12 mx-auto mb-4 text-gray-500 opacity-50" />
              <p className="text-gray-400">
                {searchTerm ? 'No personas match your search.' : 'No personas yet. Create one to start.'}
              </p>
            </div>
          ) : (
            <div className="grid gap-4 sm:grid-cols-2">
              {filteredPersonas.map(persona => (
                <PersonaCard
                  key={persona.id}
                  persona={persona}
                  onEdit={handleEditPersona}
                  onDelete={handleDeletePersona}
                />
              ))}
            </div>
          )}

          {/* Help section */}
          <div className="bg-gray-800/80 border border-gray-600/50 rounded-lg p-6">
            <h3 className="text-lg font-semibold text-white mb-2">How Skill Personas Work</h3>
            <ul className="text-sm text-gray-400 space-y-1 list-disc list-inside">
              <li>Each persona defines an agent's strategy via a custom LLM system prompt.</li>
              <li>During tournament epochs, each agent proposes a solution using their persona.</li>
              <li>An LLM judge scores all proposals — the highest score wins the skill slot.</li>
              <li>Win rates update automatically after every epoch, so you can optimize over time.</li>
            </ul>
          </div>
        </div>
      </div>

      {/* Persona Editor Modal */}
      {showEditor && (
        <PersonaEditor
          persona={editingPersona}
          onSave={handleSavePersona}
          onClose={() => { setShowEditor(false); setEditingPersona(null); }}
        />
      )}

      {/* Sliding Panels */}
      <SlidingPanel id="qr-scanner" isOpen={activePanels.includes('qr-scanner')} onClose={() => closePanel('qr-scanner')} title="QR Scanner" side="right">
        <QRScanner onScan={result => console.log('QR Result:', result)} onClose={() => closePanel('qr-scanner')} isOpen={activePanels.includes('qr-scanner')} />
      </SlidingPanel>

      <SlidingPanel id="network-status" isOpen={activePanels.includes('network-status')} onClose={() => closePanel('network-status')} title="Network Status" side="right">
        <NetworkStatus connections={networkConnections} />
      </SlidingPanel>

      <SlidingPanel id="agent-management" isOpen={activePanels.includes('agent-management')} onClose={() => closePanel('agent-management')} title="Manage Agent" side="left">
        <AgentManager agents={availableAgents} nrvs={[]} selectedNRV={null} onAgentAssignment={() => {}} nrnBalance={nrnBalance} />
      </SlidingPanel>

      <SlidingPanel id="cognitive-shell" isOpen={activePanels.includes('cognitive-shell')} onClose={() => closePanel('cognitive-shell')} title="Cognitive Shell" side="right">
        <CognitiveShellInterface onStateChange={() => {}} onSkillInvoked={() => {}} onAdaptationTriggered={() => {}} />
      </SlidingPanel>
    </div>
  );
}
