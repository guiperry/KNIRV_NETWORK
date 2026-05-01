import { useState, useCallback } from 'react';
import { useNavigate, useParams } from 'react-router';
import { ArrowLeft, Shield, Cpu, Zap, Activity, Wifi, WifiOff, BadgeCheck, Loader2, MapPin, Clock } from 'lucide-react';
import Layout from '@/react-app/components/Layout';
import ChatThread from '@/react-app/components/ChatThread';
import VoiceChatBar from '@/react-app/components/VoiceChatBar';
import type { AgentMessage } from '@/shared/types';

// ─── Mock DVE Data ──────────────────────────────────────────────────────────

type DVEStatus = 'online' | 'degraded' | 'offline';
type TEEType = 'sgx' | 'browser-ext' | 'tdx';

interface MockDVE {
  id: string;
  name: string;
  status: DVEStatus;
  tee_type: TEEType;
  reputation: number;
  location: string;
  capabilities: string[];
  badges: { label: string; color: string }[];
  cpu: number;
  memory: number;
  pendingTasks: number;
  completedTasks: number;
}

const mockDVEs: Record<string, MockDVE> = {
  'dve-alpha': {
    id: 'DVE-ALPHA-7X',
    name: 'DVE-Alpha',
    status: 'online',
    tee_type: 'sgx',
    reputation: 847,
    location: 'us-east-1',
    capabilities: ['ATTESTED_EXECUTION', 'SGX_2.0', 'LOW_LATENCY', 'SECURE_ENCLAVE'],
    badges: [
      { label: 'ATTESTED', color: 'green' },
      { label: 'SGX 2.0', color: 'blue' },
      { label: 'LOW LATENCY', color: 'cyan' },
    ],
    cpu: 34,
    memory: 52,
    pendingTasks: 3,
    completedTasks: 147,
  },
  'dve-beta': {
    id: 'DVE-BETA-3Y',
    name: 'DVE-Beta',
    status: 'degraded',
    tee_type: 'browser-ext',
    reputation: 203,
    location: 'local',
    capabilities: ['BROWSER_EXTENSION', 'DEV_MODE', 'LOCAL_COMPUTE'],
    badges: [
      { label: 'BROWSER', color: 'purple' },
      { label: 'DEV MODE', color: 'amber' },
    ],
    cpu: 12,
    memory: 78,
    pendingTasks: 1,
    completedTasks: 42,
  },
  'dve-gamma': {
    id: 'DVE-GAMMA-1Z',
    name: 'DVE-Gamma',
    status: 'online',
    tee_type: 'tdx',
    reputation: 512,
    location: 'eu-central',
    capabilities: ['TDX', 'CONFIDENTIAL_COMPUTE', 'VERIFIED_BOOT', 'MEMORY_ENCRYPTION'],
    badges: [
      { label: 'TDX', color: 'blue' },
      { label: 'CONFIDENTIAL', color: 'purple' },
      { label: 'VERIFIED', color: 'green' },
    ],
    cpu: 56,
    memory: 44,
    pendingTasks: 5,
    completedTasks: 89,
  },
};

const defaultDVEData: MockDVE = {
  id: 'DVE-UNKNOWN',
  name: 'DVE-Unknown',
  status: 'offline',
  tee_type: 'sgx',
  reputation: 0,
  location: 'unknown',
  capabilities: ['UNKNOWN'],
  badges: [],
  cpu: 0,
  memory: 0,
  pendingTasks: 0,
  completedTasks: 0,
};

// ─── Helpers ─────────────────────────────────────────────────────────────────

const statusDotColor: Record<DVEStatus, string> = {
  online: 'bg-green-500 shadow-[0_0_8px_rgba(34,197,94,0.6)]',
  degraded: 'bg-yellow-500 shadow-[0_0_8px_rgba(234,179,8,0.6)]',
  offline: 'bg-red-500 shadow-[0_0_8px_rgba(239,68,68,0.6)]',
};

const statusLabel: Record<DVEStatus, string> = {
  online: 'ONLINE',
  degraded: 'DEGRADED',
  offline: 'OFFLINE',
};

const statusBadgeClass: Record<DVEStatus, string> = {
  online: 'bg-green-500/20 text-green-400 border-green-500/30',
  degraded: 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30',
  offline: 'bg-red-500/20 text-red-400 border-red-500/30',
};

const teeIconMap: Record<TEEType, string> = {
  sgx: 'SGX',
  'browser-ext': 'BROWSER',
  tdx: 'TDX',
};

const badgeColorMap: Record<string, string> = {
  blue: 'bg-blue-500/15 text-blue-400 border-blue-500/25',
  green: 'bg-green-500/15 text-green-400 border-green-500/25',
  purple: 'bg-purple-500/15 text-purple-400 border-purple-500/25',
  amber: 'bg-amber-500/15 text-amber-400 border-amber-500/25',
  cyan: 'bg-cyan-500/15 text-cyan-400 border-cyan-500/25',
};

let taskCounter = 0;

function generateTaskId(): string {
  taskCounter += 1;
  const hex = Math.random().toString(16).slice(2, 8).toUpperCase();
  return `TASK-${String(taskCounter).padStart(4, '0')}-${hex}`;
}

function generateTimestamp(): string {
  return new Date().toISOString();
}

function buildAgentMessage(content: string, taskID?: string): AgentMessage {
  return {
    id: `msg-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
    role: 'agent',
    content,
    timestamp: generateTimestamp(),
    taskID,
  };
}

function buildUserMessage(content: string): AgentMessage {
  return {
    id: `msg-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
    role: 'user',
    content,
    timestamp: generateTimestamp(),
  };
}

function buildSystemMessage(content: string): AgentMessage {
  return {
    id: `msg-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
    role: 'system',
    content,
    timestamp: generateTimestamp(),
  };
}

// ─── Mock Response Generator ────────────────────────────────────────────────

function generateMockResponse(input: string, dve: MockDVE): AgentMessage[] {
  const lower = input.toLowerCase().trim();
  const agentName = `KNIRVAGENT-${dve.name}`;

  // "run skill" → queue task
  if (lower.includes('run skill') || lower.includes('run') && lower.includes('skill')) {
    const skillName = input.replace(/run skill/i, '').trim() || 'generic-skill';
    const taskId = generateTaskId();
    const eta = Math.floor(Math.random() * 30) + 5;
    return [
      buildSystemMessage(`Task queued — supervisor ${agentName} dispatched`),
      buildAgentMessage(
        `Task queued successfully.\n\n` +
        `━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n` +
        `Task ID:  ${taskId}\n` +
        `Skill:    ${skillName}\n` +
        `DVE:      ${dve.name}\n` +
        `ETA:      ~${eta}s\n` +
        `━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n` +
        `The skill "${skillName}" has been dispatched to ${dve.name} (${dve.tee_type.toUpperCase()}). ` +
        `Estimated completion in ${eta} seconds. You can check task status with "pending tasks".`,
        taskId
      ),
    ];
  }

  // "pending tasks" → list pending
  if (lower.includes('pending') || lower.includes('pending tasks')) {
    if (dve.pendingTasks === 0) {
      return [
        buildAgentMessage(
          `No pending tasks on ${dve.name}. All clear.\n\n` +
          `Completed tasks today: ${dve.completedTasks}`
        ),
      ];
    }
    const pendingList = [
      { id: generateTaskId(), skill: 'data-verify', status: 'running', ago: '2m' },
      { id: generateTaskId(), skill: 'model-inference', status: 'queued', ago: '30s' },
      { id: generateTaskId(), skill: 'attestation-check', status: 'queued', ago: '15s' },
    ].slice(0, dve.pendingTasks);
    return [
      buildAgentMessage(
        `Pending tasks on ${dve.name} (${dve.pendingTasks} total):\n\n` +
        pendingList.map((t, i) =>
          `  ${i + 1}. ${t.id}\n` +
          `     Skill:  ${t.skill}\n` +
          `     Status: ${t.status === 'running' ? '● RUNNING' : '○ QUEUED'}\n` +
          `     Age:    ${t.ago}`
        ).join('\n\n') +
        `\n\nCompleted tasks today: ${dve.completedTasks}`
      ),
    ];
  }

  // "badges" → list DVE badges
  if (lower.includes('badge') || lower === 'badges') {
    const badgeLines = dve.badges.length > 0
      ? dve.badges.map((b) => `  ✓ ${b.label}`).join('\n')
      : '  (no badges earned)';
    return [
      buildAgentMessage(
        `${dve.name} — Badges & Attestations:\n\n` +
        badgeLines +
        `\n\nTEE Type: ${dve.tee_type.toUpperCase()}\n` +
        `Reputation: ${dve.reputation}/1000`
      ),
    ];
  }

  // "status" → show CPU, memory, pending tasks
  if (lower.includes('status') || lower === 'status') {
    const cpuBar = '█'.repeat(Math.round(dve.cpu / 10)) + '░'.repeat(10 - Math.round(dve.cpu / 10));
    const memBar = '█'.repeat(Math.round(dve.memory / 10)) + '░'.repeat(10 - Math.round(dve.memory / 10));
    return [
      buildAgentMessage(
        `${dve.name} — Real-Time Status\n` +
        `━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n` +
        `Status:     ${statusLabel[dve.status]}\n` +
        `TEE:        ${dve.tee_type.toUpperCase()}\n` +
        `Location:   ${dve.location}\n` +
        `━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n` +
        `CPU:        ${dve.cpu}%  ${cpuBar}\n` +
        `Memory:     ${dve.memory}% ${memBar}\n` +
        `━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n` +
        `Pending:    ${dve.pendingTasks} tasks\n` +
        `Completed:  ${dve.completedTasks} tasks\n` +
        `Reputation: ${dve.reputation}/1000\n` +
        `━━━━━━━━━━━━━━━━━━━━━━━━━━━━`
      ),
    ];
  }

  // "submit error" → error node created
  if (lower.includes('submit error') || lower.includes('submit') && lower.includes('error')) {
    const errorId = `ERR-${Math.random().toString(16).slice(2, 10).toUpperCase()}`;
    return [
      buildSystemMessage(`Error node created — ${agentName} registered anomaly`),
      buildAgentMessage(
        `Error node created successfully.\n\n` +
        `━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n` +
        `Error ID:  ${errorId}\n` +
        `DVE:       ${dve.name}\n` +
        `Status:    OPEN / PRIORITY: MEDIUM\n` +
        `Logged to: D-TEN Oracle\n` +
        `━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n` +
        `The error has been recorded on-chain. A ${agentName} supervisor will review and ` +
        `either auto-remediate or escalate based on severity.`,
        errorId
      ),
    ];
  }

  // default → helpful response
  return [
    buildAgentMessage(
      `I'm ${agentName}, supervising ${dve.name} over ${dve.tee_type.toUpperCase()}.\n\n` +
      `Here's what I can help with:\n\n` +
      `  • "run skill <name>"    — Queue a new task/skill execution\n` +
      `  • "pending tasks"       — View queued and running tasks\n` +
      `  • "badges"              — Show DVE attestation badges\n` +
      `  • "status"              — Display system resource metrics\n` +
      `  • "submit error <desc>" — Create an error node for debugging\n\n` +
      `Current resource load: CPU ${dve.cpu}% | MEM ${dve.memory}% | ${dve.pendingTasks} pending tasks`
    ),
  ];
}

// ─── AgentChat Page ──────────────────────────────────────────────────────────

export default function AgentChat() {
  const { dveId } = useParams<{ dveId: string }>();
  const navigate = useNavigate();

  const dveKey = dveId?.toLowerCase() || '';
  const dve = mockDVEs[dveKey] || defaultDVEData;

  const agentName = `KNIRVAGENT-${dve.name}`;

  // Build initial greeting based on DVE data
  const initialGreeting = buildAgentMessage(
    `I'm ${agentName}, supervising ${dve.name}.\n\n` +
    `━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n` +
    `Current tasks: ${dve.pendingTasks} pending, ${dve.completedTasks} completed\n` +
    `━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n` +
    `Capabilities:\n` +
    dve.capabilities.map((c) => `  • ${c.replace(/_/g, ' ')}`).join('\n') +
    `\n\nHow can I assist you today?`
  );

  const [messages, setMessages] = useState<AgentMessage[]>([initialGreeting]);
  const [isStreaming, setIsStreaming] = useState(false);

  const handleSendMessage = useCallback((text: string) => {
    if (!text.trim() || isStreaming) return;

    const userMsg = buildUserMessage(text);
    setMessages((prev) => [...prev, userMsg]);

    // Simulate streaming delay
    setIsStreaming(true);
    setTimeout(() => {
      const responses = generateMockResponse(text, dve);
      setMessages((prev) => [...prev, ...responses]);
      setIsStreaming(false);
    }, 600 + Math.random() * 400);
  }, [dve, isStreaming]);

  const handleCopyMessage = useCallback((_id: string, content: string) => {
    navigator.clipboard.writeText(content).catch(() => {
      // Clipboard not available in all contexts
    });
  }, []);

  const handleBack = () => {
    navigate('/dves');
  };

  return (
    <Layout>
      <div className="flex flex-col h-[calc(100vh-8rem)] max-w-3xl mx-auto">
        {/* ── Header ────────────────────────────────────────────────── */}
        <div className="glass-panel mx-4 mt-4 mb-2 p-3 glow-hover relative">
          {/* Back Button */}
          <button
            onClick={handleBack}
            className="absolute left-3 top-1/2 -translate-y-1/2 flex items-center justify-center w-8 h-8 rounded-lg text-slate-400 hover:text-white hover:bg-white/10 transition-all"
            title="Back to DVEs"
          >
            <ArrowLeft className="w-4 h-4" />
          </button>

          <div className="flex items-center justify-center space-x-3">
            {/* DVE Icon */}
            <div className={`flex-shrink-0 w-9 h-9 rounded-xl flex items-center justify-center border ${
              dve.status === 'online' ? 'bg-green-500/10 border-green-500/20' :
              dve.status === 'degraded' ? 'bg-yellow-500/10 border-yellow-500/20' :
              'bg-red-500/10 border-red-500/20'
            }`}>
              {dve.status === 'online' ? (
                <Wifi className="w-4 h-4 text-green-400" />
              ) : dve.status === 'degraded' ? (
                <Wifi className="w-4 h-4 text-yellow-400" />
              ) : (
                <WifiOff className="w-4 h-4 text-red-400" />
              )}
            </div>

            {/* DVE Name + Status */}
            <div className="text-center">
              <div className="flex items-center justify-center space-x-2">
                <h2 className="text-base font-bold text-white">{dve.name}</h2>
                <span className={`inline-block w-2 h-2 rounded-full ${statusDotColor[dve.status]} animate-pulse`} />
              </div>
              <div className="flex items-center justify-center space-x-2 mt-0.5">
                <span className={`inline-flex items-center rounded-full border px-1.5 py-0.5 text-[8px] font-black uppercase tracking-wider ${statusBadgeClass[dve.status]}`}>
                  {statusLabel[dve.status]}
                </span>
                <span className="text-[9px] font-mono text-slate-500">{dve.tee_type.toUpperCase()}</span>
                <span className="text-slate-600">•</span>
                <span className="text-[9px] font-mono text-blue-400">Rep: {dve.reputation}</span>
              </div>
            </div>
          </div>

          {/* Agent Badge */}
          <div className="absolute right-3 top-1/2 -translate-y-1/2">
            <div className="flex items-center space-x-1.5 px-2 py-1 rounded-lg bg-purple-500/15 border border-purple-500/25">
              <Shield className="w-3 h-3 text-purple-400" />
              <span className="text-[8px] font-black uppercase tracking-widest text-purple-400">
                {agentName}
              </span>
            </div>
          </div>
        </div>

        {/* ── Badges Row ─────────────────────────────────────────────── */}
        <div className="flex flex-wrap items-center justify-center gap-1.5 px-4 mb-2">
          {dve.badges.map((badge, i) => (
            <span
              key={i}
              className={`inline-flex items-center rounded-md border px-1.5 py-0.5 text-[8px] font-black uppercase tracking-wider ${badgeColorMap[badge.color] || 'bg-slate-500/15 text-slate-400 border-slate-500/25'}`}
            >
              <BadgeCheck className="w-2.5 h-2.5 mr-0.5" />
              {badge.label}
            </span>
          ))}
        </div>

        {/* ── Chat Area ─────────────────────────────────────────────── */}
        <div className="flex-1 overflow-y-auto scroll-smooth">
          <ChatThread
            messages={messages}
            isStreaming={isStreaming}
            onCopyMessage={handleCopyMessage}
          />
        </div>

        {/* ── Voice Chat Bar ─────────────────────────────────────────── */}
        <div className="px-4 pb-4">
          <VoiceChatBar
            onSendMessage={handleSendMessage}
            disabled={isStreaming}
            placeholder={`Ask ${agentName} something...`}
          />
        </div>
      </div>
    </Layout>
  );
}
