import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate, useParams } from 'react-router';
import { ArrowLeft, BadgeCheck, Shield, Wifi, WifiOff } from 'lucide-react';
import Layout from '@/react-app/components/Layout';
import ChatThread from '@/react-app/components/ChatThread';
import VoiceChatBar from '@/react-app/components/VoiceChatBar';
import {
  getJoinedControllerSession,
  workerWebSocketURL,
  type JoinedControllerSession,
} from '@/react-app/platform/controllerSession';
import type { AgentMessage } from '@/shared/types';

type DVEStatus = 'online' | 'degraded' | 'offline';
type TEEType = 'sgx' | 'browser-ext' | 'tdx';

interface DVEView {
  id: string;
  name: string;
  status: DVEStatus;
  tee_type: TEEType;
  reputation: number;
  capabilities: string[];
  badges: { label: string; color: string }[];
  pendingTasks: number;
  completedTasks: number;
}

const knownDVEs: Record<string, DVEView> = {
  'dve-alpha': {
    id: 'DVE-ALPHA-7X', name: 'DVE-Alpha', status: 'online', tee_type: 'sgx', reputation: 847,
    capabilities: ['ATTESTED_EXECUTION', 'SGX_2.0', 'LOW_LATENCY', 'SECURE_ENCLAVE'],
    badges: [
      { label: 'ATTESTED', color: 'green' },
      { label: 'SGX 2.0', color: 'blue' },
      { label: 'LOW LATENCY', color: 'cyan' },
    ],
    pendingTasks: 3, completedTasks: 147,
  },
  'dve-beta': {
    id: 'DVE-BETA-3Y', name: 'DVE-Beta', status: 'degraded', tee_type: 'browser-ext', reputation: 203,
    capabilities: ['BROWSER_EXTENSION', 'DEV_MODE', 'LOCAL_COMPUTE'],
    badges: [
      { label: 'BROWSER', color: 'purple' },
      { label: 'DEV MODE', color: 'amber' },
    ],
    pendingTasks: 1, completedTasks: 42,
  },
  'dve-gamma': {
    id: 'DVE-GAMMA-1Z', name: 'DVE-Gamma', status: 'online', tee_type: 'tdx', reputation: 512,
    capabilities: ['TDX', 'CONFIDENTIAL_COMPUTE', 'VERIFIED_BOOT', 'MEMORY_ENCRYPTION'],
    badges: [
      { label: 'TDX', color: 'blue' },
      { label: 'CONFIDENTIAL', color: 'purple' },
      { label: 'VERIFIED', color: 'green' },
    ],
    pendingTasks: 5, completedTasks: 89,
  },
};

const unknownDVE: DVEView = {
  id: 'DVE-UNKNOWN', name: 'DVE-Unknown', status: 'offline', tee_type: 'sgx', reputation: 0,
  capabilities: ['UNKNOWN'], badges: [], pendingTasks: 0, completedTasks: 0,
};

const statusDotColor: Record<DVEStatus, string> = {
  online: 'bg-green-500 shadow-[0_0_8px_rgba(34,197,94,0.6)]',
  degraded: 'bg-yellow-500 shadow-[0_0_8px_rgba(234,179,8,0.6)]',
  offline: 'bg-red-500 shadow-[0_0_8px_rgba(239,68,68,0.6)]',
};

const statusLabel: Record<DVEStatus, string> = {
  online: 'ONLINE', degraded: 'DEGRADED', offline: 'OFFLINE',
};

const statusBadgeClass: Record<DVEStatus, string> = {
  online: 'bg-green-500/20 text-green-400 border-green-500/30',
  degraded: 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30',
  offline: 'bg-red-500/20 text-red-400 border-red-500/30',
};

const badgeColorMap: Record<string, string> = {
  blue: 'bg-blue-500/15 text-blue-400 border-blue-500/25',
  green: 'bg-green-500/15 text-green-400 border-green-500/25',
  purple: 'bg-purple-500/15 text-purple-400 border-purple-500/25',
  amber: 'bg-amber-500/15 text-amber-400 border-amber-500/25',
  cyan: 'bg-cyan-500/15 text-cyan-400 border-cyan-500/25',
};

let messageCounter = 0;

function createMessage(role: AgentMessage['role'], content: string, id?: string): AgentMessage {
  messageCounter += 1;
  return {
    id: id || `msg-${Date.now()}-${messageCounter}`,
    role,
    content,
    timestamp: new Date().toISOString(),
  };
}

function agentResponseContent(value: unknown): string {
  if (!value || typeof value !== 'object') {
    return '';
  }
  const response = value as Record<string, unknown>;
  for (const key of ['output', 'response', 'message', 'content']) {
    if (typeof response[key] === 'string' && response[key]) {
      return response[key];
    }
  }
  if (response.data && typeof response.data === 'object') {
    return agentResponseContent(response.data);
  }
  return '';
}

interface ChatFrame {
  id: string;
  type: string;
  sender: string;
  content: string;
  timestamp: string;
}

export default function AgentChat() {
  const { dveId } = useParams<{ dveId: string }>();
  const navigate = useNavigate();
  const dveKey = dveId?.toLowerCase() || '';
  const baseDVE = knownDVEs[dveKey] || unknownDVE;
  const agentName = `KNIRVAGENT-${baseDVE.name}`;
  const initialGreeting = useMemo(() => createMessage(
    'agent',
    `I'm ${agentName}, supervising ${baseDVE.name}.\n\n` +
      `━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n` +
      `Current tasks: ${baseDVE.pendingTasks} pending, ${baseDVE.completedTasks} completed\n` +
      `━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n` +
      `Capabilities:\n${baseDVE.capabilities.map((capability) => `  • ${capability.replace(/_/g, ' ')}`).join('\n')}` +
      '\n\nHow can I assist you today?',
  ), [agentName, baseDVE]);

  const [messages, setMessages] = useState<AgentMessage[]>([initialGreeting]);
  const [isStreaming, setIsStreaming] = useState(false);
  const [connectionStatus, setConnectionStatus] = useState<DVEStatus>(baseDVE.status);
  const [joinedSession, setJoinedSession] = useState<JoinedControllerSession | null>(null);
  const socketRef = useRef<WebSocket | null>(null);
  const pendingEchoesRef = useRef<string[]>([]);

  useEffect(() => {
    let disposed = false;
    let socket: WebSocket | null = null;

    const connect = async () => {
      const joined = await getJoinedControllerSession();
      if (disposed || !joined || joined.session_id !== dveId) {
        return;
      }
      setJoinedSession(joined);
      setConnectionStatus('degraded');
      setMessages([createMessage(
        'system',
        `Joining supervised DVE session ${joined.session_id}. Mobile input is ${joined.trust_level.replace(/_/g, ' ')}.`,
      )]);

      try {
        const response = await fetch(
          `/worker/dve/sessions/${encodeURIComponent(joined.session_id)}/chat-socket`,
          {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              access_token: joined.access_token,
              device_id: joined.device_id,
              role: 'mobile',
            }),
          },
        );
        const result = await response.json() as { ws_url?: string; error?: string };
        if (!response.ok || !result.ws_url) {
          throw new Error(result.error || 'Unable to create the DVE session socket.');
        }
        socket = new WebSocket(workerWebSocketURL(joined.session_id, result.ws_url));
        socketRef.current = socket;
        socket.onopen = () => {
          if (!disposed) {
            setConnectionStatus('online');
            setMessages((previous) => [
              ...previous,
              createMessage('system', 'Connected to the shared KNIRVAGENT session thread.'),
            ]);
          }
        };
        socket.onmessage = (event) => {
          let frame: ChatFrame;
          try {
            frame = JSON.parse(String(event.data)) as ChatFrame;
          } catch {
            return;
          }
          if (frame.type !== 'message' || !frame.content) {
            return;
          }
          const isOwnEcho = frame.sender === `mobile:${joined.device_id}`;
          if (isOwnEcho) {
            const echoIndex = pendingEchoesRef.current.indexOf(frame.content);
            if (echoIndex >= 0) {
              pendingEchoesRef.current.splice(echoIndex, 1);
              return;
            }
          }
          const role: AgentMessage['role'] =
            frame.sender === 'agent' ? 'agent' : frame.sender.startsWith('mobile:') || frame.sender === 'user' ? 'user' : 'system';
          setMessages((previous) => [
            ...previous,
            {
              ...createMessage(role, frame.content, frame.id),
              timestamp: frame.timestamp || new Date().toISOString(),
            },
          ]);
          if (frame.sender === 'agent') {
            setIsStreaming(false);
          }
        };
        socket.onclose = () => {
          socketRef.current = null;
          if (!disposed) {
            setConnectionStatus('offline');
            setIsStreaming(false);
            setMessages((previous) => [
              ...previous,
              createMessage('system', 'The shared DVE session connection closed. Reopen the chat to reconnect.'),
            ]);
          }
        };
        socket.onerror = () => {
          if (!disposed) {
            setConnectionStatus('offline');
          }
        };
      } catch (error) {
        if (!disposed) {
          setConnectionStatus('offline');
          setMessages((previous) => [
            ...previous,
            createMessage('system', error instanceof Error ? error.message : 'Unable to join the DVE session.'),
          ]);
        }
      }
    };
    void connect();
    return () => {
      disposed = true;
      socket?.close();
      if (socketRef.current === socket) {
        socketRef.current = null;
      }
    };
  }, [dveId]);

  const handleSendMessage = useCallback((text: string) => {
    const content = text.trim();
    if (!content || isStreaming) {
      return;
    }
    setMessages((previous) => [...previous, createMessage('user', content)]);
    setIsStreaming(true);

    const socket = socketRef.current;
    if (joinedSession && socket?.readyState === WebSocket.OPEN) {
      pendingEchoesRef.current.push(content);
      socket.send(JSON.stringify({ type: 'message', content, metadata: { origin: 'knirvcontroller' } }));
      return;
    }

    const sendSingleAgentCall = async () => {
      try {
        const authToken = localStorage.getItem('knirv_auth_token');
        const headers: Record<string, string> = { 'Content-Type': 'application/json' };
        if (authToken) {
          headers.Authorization = `Bearer ${authToken}`;
        }
        const response = await fetch(`/worker/dve/${encodeURIComponent(dveId || baseDVE.id)}/agent`, {
          method: 'POST',
          headers,
          body: JSON.stringify({ command: content, message: content }),
        });
        const result = await response.json() as unknown;
        if (!response.ok) {
          const errorMessage = agentResponseContent(result);
          throw new Error(errorMessage || `KNIRVAGENT request failed with HTTP ${response.status}.`);
        }
        const reply = agentResponseContent(result);
        if (!reply) {
          throw new Error('KNIRVAGENT returned an empty response.');
        }
        setMessages((previous) => [...previous, createMessage('agent', reply)]);
      } catch (error) {
        setMessages((previous) => [
          ...previous,
          createMessage('system', error instanceof Error ? error.message : 'Unable to reach KNIRVAGENT.'),
        ]);
      } finally {
        setIsStreaming(false);
      }
    };
    void sendSingleAgentCall();
  }, [baseDVE.id, dveId, isStreaming, joinedSession]);

  const handleCopyMessage = useCallback((_id: string, content: string) => {
    navigator.clipboard.writeText(content).catch(() => undefined);
  }, []);

  const displayDVE: DVEView = joinedSession ? {
    ...baseDVE,
    id: joinedSession.environment_id,
    name: `Session ${joinedSession.session_id.slice(0, 12)}`,
    status: connectionStatus,
    capabilities: ['SHARED_SESSION_CHAT', 'VAULT_SIGNED_PAIRING', 'HASH_CHAINED_EVIDENCE'],
  } : { ...baseDVE, status: connectionStatus };
  const displayAgentName = `KNIRVAGENT-${displayDVE.name}`;

  return (
    <Layout>
      <div className="flex flex-col h-[calc(100vh-8rem)] max-w-3xl mx-auto">
        <div className="glass-panel mx-4 mt-4 mb-2 p-3 glow-hover relative">
          <button
            onClick={() => navigate('/dves')}
            className="absolute left-3 top-1/2 -translate-y-1/2 flex items-center justify-center w-8 h-8 rounded-lg text-slate-400 hover:text-white hover:bg-white/10 transition-all"
            title="Back to DVEs"
          >
            <ArrowLeft className="w-4 h-4" />
          </button>

          <div className="flex items-center justify-center space-x-3">
            <div className={`flex-shrink-0 w-9 h-9 rounded-xl flex items-center justify-center border ${
              displayDVE.status === 'online' ? 'bg-green-500/10 border-green-500/20' :
                displayDVE.status === 'degraded' ? 'bg-yellow-500/10 border-yellow-500/20' :
                  'bg-red-500/10 border-red-500/20'
            }`}>
              {displayDVE.status === 'offline' ? (
                <WifiOff className="w-4 h-4 text-red-400" />
              ) : (
                <Wifi className={`w-4 h-4 ${displayDVE.status === 'online' ? 'text-green-400' : 'text-yellow-400'}`} />
              )}
            </div>

            <div className="text-center">
              <div className="flex items-center justify-center space-x-2">
                <h2 className="text-base font-bold text-white">{displayDVE.name}</h2>
                <span className={`inline-block w-2 h-2 rounded-full ${statusDotColor[displayDVE.status]} animate-pulse`} />
              </div>
              <div className="flex items-center justify-center space-x-2 mt-0.5">
                <span className={`inline-flex items-center rounded-full border px-1.5 py-0.5 text-[8px] font-black uppercase tracking-wider ${statusBadgeClass[displayDVE.status]}`}>
                  {statusLabel[displayDVE.status]}
                </span>
                <span className="text-[9px] font-mono text-slate-500">{displayDVE.tee_type.toUpperCase()}</span>
                <span className="text-slate-600">•</span>
                <span className="text-[9px] font-mono text-blue-400">Rep: {displayDVE.reputation}</span>
              </div>
            </div>
          </div>

          <div className="absolute right-3 top-1/2 -translate-y-1/2">
            <div className="flex items-center space-x-1.5 px-2 py-1 rounded-lg bg-purple-500/15 border border-purple-500/25">
              <Shield className="w-3 h-3 text-purple-400" />
              <span className="text-[8px] font-black uppercase tracking-widest text-purple-400">
                {displayAgentName}
              </span>
            </div>
          </div>
        </div>

        <div className="flex flex-wrap items-center justify-center gap-1.5 px-4 mb-2">
          {displayDVE.badges.map((badge) => (
            <span
              key={badge.label}
              className={`inline-flex items-center rounded-md border px-1.5 py-0.5 text-[8px] font-black uppercase tracking-wider ${badgeColorMap[badge.color] || 'bg-slate-500/15 text-slate-400 border-slate-500/25'}`}
            >
              <BadgeCheck className="w-2.5 h-2.5 mr-0.5" />
              {badge.label}
            </span>
          ))}
        </div>

        <div className="flex-1 overflow-y-auto scroll-smooth">
          <ChatThread messages={messages} isStreaming={isStreaming} onCopyMessage={handleCopyMessage} />
        </div>

        <div className="px-4 pb-4">
          <VoiceChatBar
            onSendMessage={handleSendMessage}
            disabled={isStreaming || (joinedSession !== null && connectionStatus !== 'online')}
            placeholder={`Ask ${displayAgentName} something...`}
          />
        </div>
      </div>
    </Layout>
  );
}
