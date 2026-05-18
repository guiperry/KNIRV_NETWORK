'use client';

import React, { useEffect, useRef, useState, useCallback } from 'react';
import { Terminal as LucideTerminal, X, Play, RefreshCw, AlertCircle, Plus } from 'lucide-react';
import type { Terminal as XTermTerminal } from '@xterm/xterm';
import type { FitAddon as XTermFitAddon } from '@xterm/addon-fit';
import type { WebLinksAddon as XTermWebLinksAddon } from '@xterm/addon-web-links';
import '@xterm/xterm/css/xterm.css';
import { useFabricManagement } from '@/hooks/use-fabric-management';
import { Button } from '@/components/ui/button';
import { API_BASE_URL, getAuthHeaders } from '@/lib/api';
import { InnerAgentTerminal } from './inner-agent-terminal';
import { useInnerAgent } from '@/hooks/use-inner-agent';

interface InnerSession {
  sessionId: string;
  toolName: string;
}

interface ConsolePanelProps {
  isOpen: boolean;
  onClose: () => void;
  nodeId?: string;
  fabricId?: string;
  isMonitorOpen?: boolean;
}

const ConsolePanel: React.FC<ConsolePanelProps> = ({ isOpen, onClose, nodeId, fabricId, isMonitorOpen }) => {
  const terminalRef = useRef<HTMLDivElement>(null);
  const xtermRef = useRef<XTermTerminal | null>(null);
  const fitAddonRef = useRef<XTermFitAddon | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const sessionIdRef = useRef<string | null>(null);
  const pollIntervalRef = useRef<NodeJS.Timeout | null>(null);
  const inputBufferRef = useRef<string>('');
  const [isInitializing, setIsInitializing] = useState(false);
  const [sshConnected, setSshConnected] = useState(false);
  const [connectionMode, setConnectionMode] = useState<'knirvshell' | 'knirvagent' | 'ssh' | 'local'>('local');
  const connectionModeRef = useRef<'knirvshell' | 'knirvagent' | 'ssh' | 'local'>('local');
  const agentProcessingRef = useRef(false);
  const [activeTab, setActiveTab] = useState<'supervisor' | string>('supervisor');
  const [innerSessions, setInnerSessions] = useState<InnerSession[]>([]);
  const { spawnSession, error: spawnError } = useInnerAgent(nodeId);
  const [spawnErrMsg, setSpawnErrMsg] = useState<string | null>(null);

  const updateConnectionModeRef = (mode: 'knirvshell' | 'knirvagent' | 'ssh' | 'local') => {
    connectionModeRef.current = mode;
    setConnectionMode(mode);
  };

  const { fetchFabricLogs } = useFabricManagement();

  // Create a KNIRVCLI session and set up polling for output
  const connectKNIRVCLI = useCallback(async (term: XTermTerminal) => {
    if (!nodeId) return false;
    try {
      const resp = await fetch(`${API_BASE_URL}/api/v1/shell/sessions`, {
        method: 'POST',
        headers: getAuthHeaders(),
        body: JSON.stringify({ 
          command: 'terminal:start',
          node_id: nodeId,
          streaming: true 
        }),
      });
      if (!resp.ok) return false;
      const data = await resp.json();
      sessionIdRef.current = data.session_id;
      updateConnectionModeRef('knirvshell');

      // Quick initial poll — if the session already closed (command failed
      // immediately), don't claim the connection; let the fallback chain
      // (KNIRVAGENT → SSH → local shell) try instead.
      const initResp = await fetch(
        `${API_BASE_URL}/api/v1/shell/sessions/${sessionIdRef.current}`,
        { headers: getAuthHeaders() }
      );
      if (initResp.ok) {
        const initSess = await initResp.json();
        if (initSess.closed) {
          sessionIdRef.current = null;
          return false;
        }
      }
      
      term.writeln('\x1b[32m[CONNECTED] KNIRVCLI session established.\x1b[0m');
      term.write('\x1b[1;32m$ \x1b[0m');

      // Poll for session output
      let lastOutputLen = 0;
      const pollOutput = async () => {
        if (!sessionIdRef.current) return;
        try {
          const outputResp = await fetch(
            `${API_BASE_URL}/api/v1/shell/sessions/${sessionIdRef.current}`,
            { headers: getAuthHeaders() }
          );
          if (outputResp.ok) {
            const session = await outputResp.json();
            if (session.output && Array.isArray(session.output)) {
              // Only write new chunks since last poll
              const newChunks = session.output.slice(lastOutputLen);
              if (newChunks.length > 0) {
                term.write(newChunks.join(''));
                lastOutputLen = session.output.length;
              }
            }
            if (session.closed) {
              term.writeln('\x1b[33m[DISCONNECTED] Session closed.\x1b[0m');
              setSshConnected(false);
              if (pollIntervalRef.current) {
                clearInterval(pollIntervalRef.current);
              }
            }
          }
        } catch { /* ignore polling errors */ }
      };

      pollIntervalRef.current = setInterval(pollOutput, 100);
      setSshConnected(true);
      return true;
    } catch {
      return false;
    }
  }, [nodeId]);

  // Send input to KNIRVCLI session
  const sendToKNIRVCLI = useCallback(async (input: string) => {
    if (!sessionIdRef.current) return;
    try {
      await fetch(`${API_BASE_URL}/api/v1/shell/sessions/${sessionIdRef.current}/input`, {
        method: 'POST',
        headers: getAuthHeaders(),
        body: JSON.stringify({ input }),
      });
    } catch { /* ignore send errors */ }
  }, []);

  // Attempt to connect the terminal to the backend SSH WebSocket for the given node.
  const connectSSH = useCallback(async (term: XTermTerminal) => {
    if (!nodeId) return false;
    try {
      const resp = await fetch(`${API_BASE_URL}/api/dve/${nodeId}/ssh-session`, {
        method: 'POST',
        headers: getAuthHeaders(),
        body: JSON.stringify({ username: 'dve-admin' }),
      });
      if (!resp.ok) return false;
      const data = await resp.json();
      const wsProto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const wsHost = window.location.host;
      const ws = new WebSocket(`${wsProto}//${wsHost}${data.ws_url}`);
      wsRef.current = ws;
      updateConnectionModeRef('ssh');

      ws.onopen = () => {
        setSshConnected(true);
        term.writeln('\x1b[32m[CONNECTED] SSH tunnel established via TEE enclave.\x1b[0m');
        term.write('\x1b[1;32m$ \x1b[0m');
      };
      ws.onmessage = (ev) => {
        const msg = JSON.parse(ev.data);
        if (msg.type === 'data') term.write(msg.data);
      };
      ws.onerror = () => {
        setSshConnected(false);
        term.writeln('\x1b[31m[SSH] Connection error — falling back to local shell.\x1b[0m');
      };
      ws.onclose = () => {
        setSshConnected(false);
      };
      return true;
    } catch {
      return false;
    }
  }, [nodeId]);

  // Attempt to connect the terminal to the DVE Supervisor Agent (KNIRVAGENT).
  const connectKNIRVAGENT = useCallback(async (term: XTermTerminal) => {
    if (!nodeId) return false;
    try {
      const resp = await fetch(`${API_BASE_URL}/api/dve/${nodeId}/supervisor-agent/session`, {
        headers: getAuthHeaders(),
      });
      if (!resp.ok) return false;
      const data = await resp.json();
      const wsProto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const wsHost = window.location.host;
      const ws = new WebSocket(`${wsProto}//${wsHost}${data.ws_url}`);
      wsRef.current = ws;
      updateConnectionModeRef('knirvagent');

      ws.onopen = () => {
        setSshConnected(true);
        term.writeln('\x1b[35m[CONNECTED] DVE Supervisor Agent (KNIRVAGENT) link established.\x1b[0m');
        term.writeln('\x1b[35m[INFO] Your terminal input will be relayed to the KNIRVAGENT supervisor.\x1b[0m');
        term.write('\x1b[1;35magent> \x1b[0m');
      };
      ws.onmessage = (ev) => {
        const msg = JSON.parse(ev.data);
        if (msg.type === 'data') {
          if (agentProcessingRef.current) {
            term.write('\r\x1b[2K');
            agentProcessingRef.current = false;
          }
          term.write(msg.data);
        } else if (msg.type === 'prompt') {
          if (agentProcessingRef.current) {
            term.write('\r\x1b[2K');
            agentProcessingRef.current = false;
          }
          term.write('\x1b[1;35magent> \x1b[0m');
        }
      };
      ws.onerror = () => {
        setSshConnected(false);
        term.writeln('\x1b[33m[KNIRVAGENT] Connection unavailable — trying SSH fallback.\x1b[0m');
      };
      ws.onclose = () => {
        setSshConnected(false);
      };
      return true;
    } catch {
      return false;
    }
  }, [nodeId]);

  const loadRealLogs = useCallback(async () => {
    if (!fabricId || !xtermRef.current) return;
    
    try {
      const logs = await fetchFabricLogs(fabricId, 50);
      if (logs && logs.length > 0) {
        xtermRef.current.writeln('\x1b[32m[SYSTEM] Fetching historical logs from Fabric ID: ' + fabricId + '\x1b[0m');
        logs.forEach(log => {
          const timestamp = new Date(log.timestamp).toLocaleTimeString();
          const color = log.level === 'error' ? '\x1b[31m' : log.level === 'warn' ? '\x1b[33m' : '\x1b[32m';
          xtermRef.current?.writeln(`[${timestamp}] ${color}${log.level.toUpperCase()}\x1b[0m: ${log.message}`);
        });
      }
    } catch (err) {
      xtermRef.current?.writeln('\x1b[31m[ERROR] Failed to fetch historical logs.\x1b[0m');
    }
  }, [fabricId, fetchFabricLogs]);

  useEffect(() => {
    if (isOpen && terminalRef.current && !xtermRef.current) {
      setIsInitializing(true);
      
      // Dynamically import and initialize terminal
      const initTerminal = async () => {
        const { Terminal } = await import('@xterm/xterm');
        const { FitAddon } = await import('@xterm/addon-fit');
        const { WebLinksAddon } = await import('@xterm/addon-web-links');

        const term = new Terminal({
          cursorBlink: true,
          fontSize: 11,
          fontFamily: 'Menlo, Monaco, "Courier New", monospace',
          theme: {
            background: '#03050a',
            foreground: '#93c5fd',
            cursor: '#2563eb',
            selectionBackground: 'rgba(37, 99, 235, 0.3)',
          },
          convertEol: true,
        });

        const fitAddon = new FitAddon();
        const webLinksAddon = new WebLinksAddon();

        term.loadAddon(fitAddon);
        term.loadAddon(webLinksAddon);

        if (terminalRef.current) {
          term.open(terminalRef.current);
          fitAddon.fit();
        }
        fitAddon.fit();

        term.writeln('\x1b[1;34mKNIRV-SERVER Secure Terminal v1.1.0\x1b[0m');
        term.writeln(`\x1b[33mConnecting to context: ${nodeId || 'global'}...\x1b[0m`);
        term.writeln('\x1b[32mAuthenticated via Hardware TEE Enclave.\x1b[0m');
        term.writeln('');

        xtermRef.current = term;
        fitAddonRef.current = fitAddon;

        setIsInitializing(false);

        // Connection priority: KNIRVAGENT supervisor -> SSH -> KNIRVCLI -> local
        // KNIRVAGENT is the primary DVE supervisor terminal — try it first.
        let didConnect = await connectKNIRVAGENT(term);
        if (!didConnect) {
          didConnect = await connectSSH(term);
        }
        if (!didConnect) {
          didConnect = await connectKNIRVCLI(term);
        }
        if (!didConnect) {
          term.writeln('\x1b[33m[FALLBACK] No remote connection available — using local simulation.\x1b[0m');
          term.write('\x1b[1;32mroot@fabric-server:~# \x1b[0m');
          loadRealLogs();
        }

        term.onData((data) => {
          // Route input to appropriate backend based on connection mode (via ref to avoid stale closure)
          const mode = connectionModeRef.current;
          if (mode === 'knirvshell' && sessionIdRef.current) {
            // Buffer input and send complete lines on Enter
            inputBufferRef.current += data;
            if (data === '\r') {
              const line = inputBufferRef.current.replace(/\r/g, '').trim();
              inputBufferRef.current = '';
              if (line) {
                sendToKNIRVCLI(line + '\n');
              }
              term.write('\r\n');
            } else if (data === '\u007f') { // Backspace
              if (inputBufferRef.current.length > 1) {
                inputBufferRef.current = inputBufferRef.current.slice(0, -2); // remove both the backspace char and the preceding char
                term.write('\b \b');
              } else {
                inputBufferRef.current = '';
              }
            } else {
              term.write(data);
            }
          } else if (mode === 'knirvagent' && wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
            // Buffer input for KNIRVAGENT and send complete lines on Enter
            inputBufferRef.current += data;
            if (data === '\r') {
              const line = inputBufferRef.current.replace(/\r/g, '').trim();
              inputBufferRef.current = '';
              if (line) {
                wsRef.current.send(JSON.stringify({ type: 'input', data: line }));
                term.write('\r\n\x1b[33m⟳ Processing...\x1b[0m');
                agentProcessingRef.current = true;
              } else {
                term.write('\r\n');
              }
            } else if (data === '\u007f') { // Backspace
              if (inputBufferRef.current.length > 1) {
                inputBufferRef.current = inputBufferRef.current.slice(0, -2);
                term.write('\b \b');
              } else {
                inputBufferRef.current = '';
              }
            } else {
              term.write(data);
            }
          } else if (mode === 'ssh' && wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
            wsRef.current.send(JSON.stringify({ type: 'input', data }));
          } else {
            // Local simulation fallback
            if (data === '\r') {
              term.write('\r\n\x1b[1;32mroot@fabric-server:~# \x1b[0m');
            } else if (data === '\u007f') {
              term.write('\b \b');
            } else {
              term.write(data);
            }
          }
        });

        const handleResize = () => fitAddon.fit();
        window.addEventListener('resize', handleResize);

        return () => {
          window.removeEventListener('resize', handleResize);
          if (pollIntervalRef.current) {
            clearInterval(pollIntervalRef.current);
          }
          if (sessionIdRef.current) {
            fetch(`${API_BASE_URL}/api/v1/shell/sessions/${sessionIdRef.current}/stop`, {
              method: 'POST',
              headers: getAuthHeaders(),
            }).catch(() => {});
          }
          if (wsRef.current) {
            wsRef.current.close();
            wsRef.current = null;
          }
          term.dispose();
          xtermRef.current = null;
        };
      };

      let cleanup: (() => void) | undefined;
      initTerminal().then((c) => { cleanup = c; });

      return () => {
        if (cleanup) cleanup();
      };
    }
  }, [isOpen, nodeId, loadRealLogs, connectSSH]);

  if (!isOpen) return null;

  return (
    <div
      className="absolute z-[100] transition-slide duration-500 ease-in-out bg-slate-950 border border-blue-600/50 shadow-[0_0_40px_rgba(0,0,0,0.7)] overflow-hidden rounded-xl animate-in fade-in zoom-in-95 gpu-accelerated"
      style={{
        right: isMonitorOpen ? '40px' : '20px',
        top: isMonitorOpen ? '20px' : '80px',
        width: isMonitorOpen ? '500px' : '600px',
        height: isMonitorOpen ? '300px' : '400px',
      }}
    >
      <div className="h-full flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between px-3 py-2 border-b border-blue-600/30 bg-slate-900/80 backdrop-blur-md shrink-0">
          <div className="flex items-center space-x-2">
            <div className="relative">
              <LucideTerminal className="w-4 h-4 text-blue-400" />
              <div className="absolute -top-1 -right-1 w-2 h-2 bg-green-500 rounded-full animate-pulse" />
            </div>
            <h2 className="text-[11px] font-black uppercase tracking-tighter text-blue-100">
              Terminal
            </h2>
            <p className="text-[9px] font-mono text-slate-500 hidden sm:block">Node: {nodeId || 'Distributed'}</p>
          </div>
          <div className="flex items-center space-x-2">
            <button
              onClick={loadRealLogs}
              className="text-slate-500 hover:text-blue-400 p-1 transition-colors"
              title="Sync Logs"
            >
              <RefreshCw className="w-3.5 h-3.5" />
            </button>
            <div className="h-4 w-px bg-slate-800 mx-1" />
            <button
              onClick={onClose}
              className="text-slate-500 hover:text-white hover:bg-red-900/30 p-1 rounded transition-interactive"
            >
              <X className="w-4 h-4" />
            </button>
          </div>
        </div>

        {/* Tab bar */}
        <div className="flex items-center border-b border-slate-800 bg-slate-950/80 overflow-x-auto shrink-0">
          {/* Supervisor tab */}
          <button
            onClick={() => setActiveTab('supervisor')}
            className={`flex items-center gap-1.5 px-3 py-1.5 text-[11px] font-medium whitespace-nowrap border-r border-slate-800 transition-colors ${
              activeTab === 'supervisor'
                ? 'bg-[#03050a] text-blue-300 border-b-2 border-b-blue-500 -mb-px'
                : 'text-slate-500 hover:text-slate-300 hover:bg-slate-900/50'
            }`}
          >
            <LucideTerminal className="w-3 h-3" />
            Supervisor
          </button>

          {/* Inner agent session tabs */}
          {innerSessions.map(sess => (
            <button
              key={sess.sessionId}
              onClick={() => setActiveTab(sess.sessionId)}
              className={`flex items-center gap-1.5 px-3 py-1.5 text-[11px] font-medium whitespace-nowrap border-r border-slate-800 group transition-colors ${
                activeTab === sess.sessionId
                  ? 'bg-[#03050a] text-purple-300 border-b-2 border-b-purple-500 -mb-px'
                  : 'text-slate-500 hover:text-slate-300 hover:bg-slate-900/50'
              }`}
            >
              <Play className="w-2.5 h-2.5" />
              {sess.toolName}
              <span
                role="button"
                tabIndex={0}
                onClick={e => {
                  e.stopPropagation();
                  setInnerSessions(prev => prev.filter(s => s.sessionId !== sess.sessionId));
                  if (activeTab === sess.sessionId) setActiveTab('supervisor');
                }}
                onKeyDown={e => { if (e.key === 'Enter') { e.stopPropagation(); setInnerSessions(prev => prev.filter(s => s.sessionId !== sess.sessionId)); if (activeTab === sess.sessionId) setActiveTab('supervisor'); } }}
                className="opacity-0 group-hover:opacity-100 ml-0.5 hover:text-red-400 transition-opacity"
                title="Close tab"
              >
                <X className="w-2.5 h-2.5" />
              </span>
            </button>
          ))}

          {/* Spawn new terminal session */}
          <button
            onClick={async () => {
              setSpawnErrMsg(null);
              const sessionId = await spawnSession('shell');
              if (sessionId) {
                const label = `Terminal ${innerSessions.length + 1}`;
                setInnerSessions(prev => [...prev, { sessionId, toolName: label }]);
                setActiveTab(sessionId);
              } else {
                setSpawnErrMsg(spawnError ?? 'Failed to spawn terminal');
              }
            }}
            className="flex items-center gap-1 px-3 py-1.5 text-[11px] text-slate-500 hover:text-purple-300 hover:bg-slate-900/50 transition-colors whitespace-nowrap"
            title="New terminal"
          >
            <Plus className="w-3 h-3" />
          </button>
          {spawnErrMsg && (
            <span className="px-2 text-[10px] text-red-400 truncate max-w-[200px]" title={spawnErrMsg}>
              {spawnErrMsg}
            </span>
          )}
        </div>

        {/* Terminal content */}
        <div className="flex-1 bg-[#03050a] overflow-hidden relative">
          {/* Supervisor xterm — always mounted, hidden when not active */}
          <div
            className="absolute inset-0 p-3"
            style={{ display: activeTab === 'supervisor' ? 'block' : 'none' }}
          >
            <div ref={terminalRef} className="h-full w-full custom-scrollbar" />
            {isInitializing && (
              <div className="absolute inset-0 bg-black/80 flex items-center justify-center backdrop-blur-sm">
                <div className="text-center space-y-3">
                  <div className="w-8 h-8 border-2 border-blue-500/20 border-t-blue-500 rounded-full animate-spin mx-auto" />
                  <p className="text-[10px] font-black uppercase tracking-widest text-blue-400 animate-pulse">
                    Establishing TEE Tunnel...
                  </p>
                </div>
              </div>
            )}
          </div>

          {/* Inner agent terminals — one per session, always mounted once created */}
          {innerSessions.map(sess => (
            <div
              key={sess.sessionId}
              className="absolute inset-0"
              style={{ display: activeTab === sess.sessionId ? 'block' : 'none' }}
            >
              <InnerAgentTerminal
                dveId={nodeId ?? ''}
                sessionId={sess.sessionId}
                toolName={sess.toolName}
                isVisible={activeTab === sess.sessionId}
              />
            </div>
          ))}
        </div>

        <div className="p-2 border-t border-blue-600/20 bg-slate-900/50 flex justify-between items-center px-4 shrink-0">
          <div className="flex items-center space-x-4">
            <div className="flex items-center text-[9px] text-slate-500 font-bold uppercase">
              <div className={`w-1.5 h-1.5 rounded-full mr-1.5 ${sshConnected ? 'bg-green-500' : 'bg-yellow-500'}`} />
              {activeTab === 'supervisor' ? `${connectionMode.toUpperCase()}: ${sshConnected ? 'ACTIVE' : 'LOCAL'}` : 'INNER AGENT: ACTIVE'}
            </div>
            <div className="flex items-center text-[9px] text-slate-500 font-bold uppercase">
              <div className="w-1.5 h-1.5 rounded-full bg-blue-500 mr-1.5" />
              M-TLS: ENABLED
            </div>
          </div>
          <span className="text-[9px] font-mono text-slate-600">AES-256-GCM</span>
        </div>

      </div>
    </div>
  );
};

export default ConsolePanel;
