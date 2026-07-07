import { useState, useEffect, useCallback, useRef } from 'react';
import { Brain, Activity, Wifi, WifiOff, Server } from 'lucide-react';
import Layout from '@/react-app/components/Layout';
import ChatThread from '@/react-app/components/ChatThread';
import VoiceChatBar from '@/react-app/components/VoiceChatBar';
import type { AgentMessage } from '@/shared/types';

/* ─── Mock Fleet Data ─── */
const MOCK_FLEET_DATA = [
  { id: 'dve-001', name: 'Hermes-Alpha', status: 'online', tasks: 4, uptime: '14d 6h' },
  { id: 'dve-002', name: 'Athena-Beta', status: 'online', tasks: 2, uptime: '8d 2h' },
  { id: 'dve-003', name: 'Apollo-Gamma', status: 'online', tasks: 7, uptime: '22d 11h' },
  { id: 'dve-004', name: 'Artemis-Delta', status: 'online', tasks: 1, uptime: '3d 19h' },
  { id: 'dve-005', name: 'Ares-Epsilon', status: 'online', tasks: 5, uptime: '11d 5h' },
  { id: 'dve-006', name: 'Zeus-Zeta', status: 'online', tasks: 3, uptime: '6d 14h' },
  { id: 'dve-007', name: 'Hera-Eta', status: 'online', tasks: 0, uptime: '1d 22h' },
  { id: 'dve-008', name: 'Poseidon-Theta', status: 'online', tasks: 6, uptime: '18d 3h' },
  { id: 'dve-009', name: 'Hades-Iota', status: 'offline', tasks: 0, uptime: '0d 0h' },
  { id: 'dve-010', name: 'Demeter-Kappa', status: 'offline', tasks: 0, uptime: '0d 0h' },
  { id: 'dve-011', name: 'Hestia-Lambda', status: 'offline', tasks: 0, uptime: '0d 0h' },
  { id: 'dve-012', name: 'Dionysus-Mu', status: 'offline', tasks: 0, uptime: '0d 0h' },
];

const MOCK_HEALTH_STATUS = {
  cpu: '23%',
  memory: '41%',
  latency: '12ms',
  oracleSync: 'synced',
  uptime: '47d 8h 12m',
  dveConnected: 8,
};

/* ─── Helpers ─── */

let msgCounter = 0;

function createMessage(
  role: AgentMessage['role'],
  content: string,
  extras?: Partial<AgentMessage>,
): AgentMessage {
  msgCounter += 1;
  return {
    id: `ce-msg-${Date.now()}-${msgCounter}`,
    role,
    content,
    timestamp: new Date().toISOString(),
    ...extras,
  };
}

function formatFleetResponse(): string {
  const online = MOCK_FLEET_DATA.filter((d) => d.status === 'online');
  const offline = MOCK_FLEET_DATA.filter((d) => d.status === 'offline');

  const header = '=== DVE FLEET STATUS ===\n\n';
  const onlineSection =
    `Online (${online.length}):\n` +
    online
      .map((d) => `  ${d.id} | ${d.name.padEnd(16)} | tasks: ${d.tasks} | uptime: ${d.uptime}`)
      .join('\n');
  const offlineSection = `\n\nOffline (${offline.length}):\n${offline.map((d) => `  ${d.id} | ${d.name}`).join('\n')}`;

  return header + onlineSection + offlineSection;
}

function formatHealthResponse(): string {
  const h = MOCK_HEALTH_STATUS;
  return (
    '=== NETWORK HEALTH ===\n\n' +
    `  CPU Usage:        ${h.cpu}\n` +
    `  Memory Usage:     ${h.memory}\n` +
    `  Avg Latency:      ${h.latency}\n` +
    `  Oracle Sync:      ${h.oracleSync}\n` +
    `  Uptime:           ${h.uptime}\n` +
    `  DVE Nodes Online: ${h.dveConnected}/12\n\n` +
    'All systems nominal. No anomalies detected.'
  );
}

function formatValidationResponse(badgeName: string): string {
  return (
    `✓ Validation task queued for badge "${badgeName}".\n\n` +
    `Running static analysis...\n` +
    `Checking dependency integrity...\n` +
    `Simulating execution environment...\n\n` +
    `Estimated completion: 30-45 seconds. You will be notified when results are ready.`
  );
}

function formatPolicyAttachResponse(policyId: string, dveName: string): string {
  return (
    `Policy ${policyId} is being attached to DVE "${dveName}".\n\n` +
    `→ Verifying policy signature...\n` +
    `→ Validating DVE capabilities...\n` +
    `→ Applying policy constraints...\n\n` +
    `Attachment successful. Policy is now active on ${dveName}.`
  );
}

function formatDefaultResponse(input: string): string {
  const responses = [
    `Acknowledged. Processing your request: "${input}". The Cognitive Engine will analyze this through the SEAL loop and respond shortly.`,
    `Cognitive Engine received: "${input}". Oracle is consulting available data streams for an optimal response path.`,
    `Input registered by the Cognitive Engine. Running pattern analysis on "${input}" through the DVE fleet.`,
    `The Cognitive Engine is processing your command. Current network load is nominal — expected response within standard latency bounds.`,
  ];
  return responses[Math.floor(Math.random() * responses.length)];
}

/* ─── Component ─── */

export default function CognitiveEngineChat() {
  const [ceActive, setCeActive] = useState<boolean | null>(null); // null = loading
  const [messages, setMessages] = useState<AgentMessage[]>([]);
  const [isStreaming, setIsStreaming] = useState(false);
  const [voiceStatus, _setVoiceStatus] = useState<'idle' | 'listening' | 'processing' | 'speaking' | 'error'>('idle');
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  /* ─── Health check on mount ─── */
  useEffect(() => {
    const t = setTimeout(() => {
      setCeActive(true);
      setMessages([
        createMessage('system', 'Cognitive Engine active. Oracle loaded. DVE fleet: 12 nodes, 8 online.'),
      ]);
    }, 800);
    return () => clearTimeout(t);
  }, []);

  /* Cleanup any pending response timer */
  useEffect(() => {
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
    };
  }, []);

  /* ─── Simulated agent response ─── */
  const simulateAgentResponse = useCallback((userInput: string) => {
    setIsStreaming(true);

    const lower = userInput.toLowerCase();

    let responseContent: string;

    if (lower.includes('fleet') || lower.includes('dve status') || lower.includes('fleet status')) {
      responseContent = formatFleetResponse();
    } else if (lower.includes('health') || lower.includes('network health')) {
      responseContent = formatHealthResponse();
    } else if (lower.includes('validation') || lower.includes('validate')) {
      const match = lower.match(/badge\s+["']?([a-zA-Z0-9_-]+)["']?/);
      const badgeName = match ? match[1] : 'unknown-badge';
      responseContent = formatValidationResponse(badgeName);
    } else if (lower.includes('attach policy') || lower.includes('policy attach')) {
      const policyMatch = lower.match(/policy\s+["']?([a-zA-Z0-9_-]+)["']?/);
      const dveMatch = lower.match(/dve\s+["']?([a-zA-Z0-9_-]+)["']?/);
      const policyId = policyMatch ? policyMatch[1] : 'POL-0000';
      const dveName = dveMatch ? dveMatch[1] : 'unknown-dve';
      responseContent = formatPolicyAttachResponse(policyId, dveName);
    } else {
      responseContent = formatDefaultResponse(userInput);
    }

    timerRef.current = setTimeout(() => {
      setMessages((prev) => [
        ...prev,
        createMessage('agent', responseContent, { dveID: 'ce-knirv-1' }),
      ]);
      setIsStreaming(false);
    }, 2000);
  }, []);

  /* ─── Handle send (text or voice) ─── */
  const handleSendMessage = useCallback(
    (text: string) => {
      if (!text.trim() || isStreaming) return;

      const userMsg = createMessage('user', text.trim());
      setMessages((prev) => [...prev, userMsg]);
      simulateAgentResponse(text.trim());
    },
    [isStreaming, simulateAgentResponse],
  );

  /* ─── Copy handler ─── */
  const handleCopyMessage = useCallback(async (_id: string, content: string) => {
    try {
      await navigator.clipboard.writeText(content);
    } catch {
      // clipboard not available
    }
  }, []);

  /* ─── Reconnect (retry health) ─── */
  const handleRetry = useCallback(() => {
    setCeActive(null);
    setMessages([]);
    setTimeout(() => {
      setCeActive(true);
      setMessages([
        createMessage('system', 'Cognitive Engine active. Oracle loaded. DVE fleet: 12 nodes, 8 online.'),
      ]);
    }, 800);
  }, []);

  /* ─── Render: Loading ─── */
  if (ceActive === null) {
    return (
      <Layout>
        <div className="p-4 pb-24 space-y-6">
          <div className="flex flex-col items-center justify-center min-h-[60vh] space-y-4">
            <div className="relative">
              <Brain className="w-12 h-12 text-blue-400 animate-pulse" />
              <span className="absolute -bottom-1 -right-1 flex h-4 w-4">
                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-blue-400 opacity-75" />
                <span className="relative inline-flex rounded-full h-4 w-4 bg-blue-500" />
              </span>
            </div>
            <p className="text-slate-400 text-sm font-mono animate-pulse">
              Connecting to Cognitive Engine...
            </p>
          </div>
        </div>
      </Layout>
    );
  }

  /* ─── Render: CE Not Active ─── */
  if (!ceActive) {
    return (
      <Layout>
        <div className="p-4 pb-24 space-y-6">
          {/* Header */}
          <div className="text-center py-4">
            <h2 className="text-2xl font-bold gradient-text mb-2">◈ COGNITIVE ENGINE</h2>
            <p className="text-slate-400 text-sm font-mono uppercase tracking-wider">Root Node Required</p>
          </div>

          {/* Not Active Notice */}
          <div className="relative group">
            <div className="absolute -inset-0.5 bg-gradient-to-r from-amber-600/40 to-amber-800/40 rounded-xl blur opacity-40 group-hover:opacity-60 transition duration-300" />
            <div className="relative glass-panel p-6 text-center space-y-4 glow-hover">
              <div className="flex justify-center">
                <div className="w-16 h-16 rounded-full bg-amber-500/20 border border-amber-500/30 flex items-center justify-center">
                  <WifiOff className="w-8 h-8 text-amber-400" />
                </div>
              </div>
              <div>
                <h3 className="text-lg font-semibold text-white mb-1">Cognitive Engine Offline</h3>
                <p className="text-sm text-slate-400">
                  The Cognitive Engine requires a connection to a root node instance to operate.
                  Please connect to a root node to enable cognitive capabilities.
                </p>
              </div>
              <button
                onClick={handleRetry}
                className="px-6 py-2.5 bg-blue-600 hover:bg-blue-500 rounded-lg text-white font-medium transition-all transform hover:scale-105 glow-hover text-sm"
              >
                Retry Connection
              </button>
            </div>
          </div>

          {/* Instructions */}
          <div className="glass-panel p-4 space-y-3 glow-hover">
            <h4 className="text-sm font-semibold text-white uppercase tracking-wider flex items-center gap-2">
              <Server className="w-4 h-4 text-blue-400" />
              How to connect
            </h4>
            <ul className="text-xs text-slate-400 space-y-2 font-mono">
              <li className="flex items-start gap-2">
                <span className="text-blue-400 mt-0.5">1.</span>
                <span>Ensure your KNIRV root node (knirv-1) is running and accessible on the D-TEN network.</span>
              </li>
              <li className="flex items-start gap-2">
                <span className="text-blue-400 mt-0.5">2.</span>
                <span>Verify the root node endpoint is configured in your environment settings.</span>
              </li>
              <li className="flex items-start gap-2">
                <span className="text-blue-400 mt-0.5">3.</span>
                <span>Firewall rules allow connections on port 8443 (CE API) and port 443 (D-TEN mesh).</span>
              </li>
              <li className="flex items-start gap-2">
                <span className="text-blue-400 mt-0.5">4.</span>
                <span>Click "Retry Connection" above or restart the controller to re-establish the link.</span>
              </li>
            </ul>
          </div>
        </div>
      </Layout>
    );
  }

  /* ─── Render: CE Active ─── */
  return (
    <Layout>
      <div className="flex flex-col h-[calc(100vh-8rem)]">
        {/* Header */}
        <div className="p-4 pb-2 space-y-2">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Brain className="w-5 h-5 text-blue-400" />
              <h2 className="text-xl font-bold gradient-text">◈ COGNITIVE ENGINE</h2>
            </div>
            <div className="flex items-center gap-2 px-2.5 py-1 rounded-full bg-blue-500/15 border border-blue-500/25">
              <div className="w-2 h-2 bg-blue-400 rounded-full animate-pulse" />
              <span className="text-xs text-blue-400 font-mono font-medium">Root: knirv-1</span>
            </div>
          </div>

          {/* Health status bar */}
          <div className="flex items-center gap-3 text-xs text-slate-500 font-mono">
            <span className="flex items-center gap-1">
              <Activity className="w-3 h-3 text-green-400" />
              CPU 23%
            </span>
            <span className="text-slate-700">|</span>
            <span className="flex items-center gap-1">
              <Wifi className="w-3 h-3 text-blue-400" />
              8/12 online
            </span>
            <span className="text-slate-700">|</span>
            <span className="flex items-center gap-1 text-green-400">
              <span className="w-1.5 h-1.5 rounded-full bg-green-400" />
              Oracle synced
            </span>
          </div>
        </div>

        {/* Chat Messages */}
        <div className="flex-1 overflow-y-auto scroll-smooth">
          <ChatThread
            messages={messages}
            isStreaming={isStreaming}
            onCopyMessage={handleCopyMessage}
          />
        </div>

        {/* Voice-powered input bar */}
        <VoiceChatBar
          onSendMessage={handleSendMessage}
          disabled={isStreaming}
          placeholder="Type a command or use voice..."
          voiceStatus={voiceStatus}
        />
      </div>
    </Layout>
  );
}
