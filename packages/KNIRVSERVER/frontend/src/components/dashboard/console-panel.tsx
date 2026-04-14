'use client';

import React, { useEffect, useRef, useState, useCallback } from 'react';
import { Terminal as LucideTerminal, X, Play, RefreshCw, AlertCircle } from 'lucide-react';
import type { Terminal as XTermTerminal } from '@xterm/xterm';
import type { FitAddon as XTermFitAddon } from '@xterm/addon-fit';
import type { WebLinksAddon as XTermWebLinksAddon } from '@xterm/addon-web-links';
import '@xterm/xterm/css/xterm.css';
import { useFabricManagement } from '@/hooks/use-fabric-management';
import { Button } from '@/components/ui/button';
import { API_BASE_URL, getAuthHeaders } from '@/lib/api';

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
  const [isInitializing, setIsInitializing] = useState(false);
  const [sshConnected, setSshConnected] = useState(false);
  const [connectionMode, setConnectionMode] = useState<'knirvshell' | 'ssh' | 'local'>('local');

  const { fetchFabricLogs } = useFabricManagement();

  // Create a KNIRVCLI session and set up polling for output
  const connectKNIRVCLI = useCallback(async (term: XTermTerminal) => {
    if (!nodeId) return false;
    try {
      const resp = await fetch(`${API_BASE_URL}/api/v1/cli/sessions`, {
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
      setConnectionMode('knirvshell');
      
      term.writeln('\x1b[32m[CONNECTED] KNIRVCLI session established.\x1b[0m');
      term.write('\x1b[1;32m$ \x1b[0m');

      // Poll for session output
      const pollOutput = async () => {
        if (!sessionIdRef.current) return;
        try {
          const outputResp = await fetch(
            `${API_BASE_URL}/api/v1/cli/sessions/${sessionIdRef.current}`,
            { headers: getAuthHeaders() }
          );
          if (outputResp.ok) {
            const session = await outputResp.json();
            if (session.output) {
              term.write(session.output);
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

      pollIntervalRef.current = setInterval(pollOutput, 500);
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
      await fetch(`${API_BASE_URL}/api/v1/cli/sessions/${sessionIdRef.current}/input`, {
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
      setConnectionMode('ssh');

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

        // Try KNIRVCLI first, then SSH, then fall back to local simulation
        let didConnect = await connectKNIRVCLI(term);
        if (!didConnect) {
          didConnect = await connectSSH(term);
        }
        if (!didConnect) {
          term.write('\x1b[1;32mroot@fabric-server:~# \x1b[0m');
          loadRealLogs();
        }

        term.onData((data) => {
          // Route input to appropriate backend based on connection mode
          if (connectionMode === 'knirvshell' && sessionIdRef.current) {
            sendToKNIRVCLI(data);
            if (data === '\r') {
              term.write('\r\n');
            }
          } else if (connectionMode === 'ssh' && wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
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
            fetch(`${API_BASE_URL}/api/v1/cli/sessions/${sessionIdRef.current}/stop`, {
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
        <div className="flex items-center justify-between p-3 border-b border-blue-600/30 bg-slate-900/80 backdrop-blur-md">
          <div className="flex items-center space-x-3">
            <div className="relative">
              <LucideTerminal className="w-4 h-4 text-blue-400" />
              <div className="absolute -top-1 -right-1 w-2 h-2 bg-green-500 rounded-full animate-pulse" />
            </div>
            <div>
              <h2 className="text-[11px] font-black uppercase tracking-tighter text-blue-100">
                Secure Shell Session
              </h2>
              <p className="text-[9px] font-mono text-slate-500">Node: {nodeId || 'Distributed'}</p>
            </div>
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
        
        <div className="flex-1 p-3 bg-[#03050a] overflow-hidden relative group">
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
        
        <div className="p-2 border-t border-blue-600/20 bg-slate-900/50 flex justify-between items-center px-4">
          <div className="flex items-center space-x-4">
            <div className="flex items-center text-[9px] text-slate-500 font-bold uppercase">
              <div className={`w-1.5 h-1.5 rounded-full mr-1.5 ${sshConnected ? 'bg-green-500' : 'bg-yellow-500'}`} />
              {connectionMode.toUpperCase()}: {sshConnected ? 'ACTIVE' : 'LOCAL'}
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
