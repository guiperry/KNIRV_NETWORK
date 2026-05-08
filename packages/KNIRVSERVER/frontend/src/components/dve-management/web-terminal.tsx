"use client";

import React, { useEffect, useRef, useState } from 'react';

// Import xterm types and CSS
import type { Terminal as XTermTerminal } from '@xterm/xterm';
import type { FitAddon as XTermFitAddon } from '@xterm/addon-fit';
import type { WebLinksAddon as XTermWebLinksAddon } from '@xterm/addon-web-links';
import '@xterm/xterm/css/xterm.css';

interface WebTerminalProps {
  sessionId: string;
  endpoint: string;
  wsPath?: string;
  connectedMessage?: string;
  onClose?: () => void;
  className?: string;
}

export const WebTerminal: React.FC<WebTerminalProps> = ({
  sessionId,
  endpoint,
  wsPath,
  connectedMessage = 'Connected to DVE container',
  onClose,
  className = '',
}) => {
  const terminalRef = useRef<HTMLDivElement>(null);
  const terminalInstance = useRef<XTermTerminal | null>(null);
  const fitAddon = useRef<XTermFitAddon | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [isConnecting, setIsConnecting] = useState(true);

  useEffect(() => {
    if (!terminalRef.current || !sessionId) return;

    // Dynamically import and initialize terminal
    const initTerminal = async () => {
      const { Terminal } = await import('@xterm/xterm');
      const { FitAddon } = await import('@xterm/addon-fit');
      const { WebLinksAddon } = await import('@xterm/addon-web-links');

      // Initialize terminal
      const term = new Terminal({
        cursorBlink: true,
        cursorStyle: 'block',
        fontSize: 14,
        fontFamily: 'Monaco, Menlo, "Ubuntu Mono", monospace',
        theme: {
          background: '#0f172a', // slate-900
          foreground: '#e2e8f0', // slate-200
          cursor: '#3b82f6', // blue-500
          cursorAccent: '#1e293b', // slate-800
          black: '#1e293b',
          red: '#ef4444',
          green: '#10b981',
          yellow: '#f59e0b',
          blue: '#3b82f6',
          magenta: '#8b5cf6',
          cyan: '#06b6d4',
          white: '#e2e8f0',
          brightBlack: '#475569',
          brightRed: '#f87171',
          brightGreen: '#34d399',
          brightYellow: '#fbbf24',
          brightBlue: '#60a5fa',
          brightMagenta: '#a78bfa',
          brightCyan: '#22d3ee',
          brightWhite: '#f8fafc',
        },
        allowTransparency: true,
        scrollback: 1000,
      });

      // Add addons
      const fit = new FitAddon();
      const webLinks = new WebLinksAddon();

      term.loadAddon(fit);
      term.loadAddon(webLinks);

      // Open terminal
      if (terminalRef.current) {
        term.open(terminalRef.current);
        fit.fit();
      }

      terminalInstance.current = term;
      fitAddon.current = fit;

      // Connect to WebSocket
      connectWebSocket(term);

      // Handle resize
      const handleResize = () => {
        if (fitAddon.current) {
          fitAddon.current.fit();
        }
      };

      window.addEventListener('resize', handleResize);

      // Cleanup function
      return () => {
        window.removeEventListener('resize', handleResize);
        if (wsRef.current) {
          wsRef.current.close();
        }
        if (terminalInstance.current) {
          terminalInstance.current.dispose();
        }
      };
    };

    // Execute initialization and store cleanup
    let cleanup: (() => void) | undefined;
    initTerminal().then((c) => { cleanup = c; });

    // Return cleanup function
    return () => {
      if (cleanup) cleanup();
    };
  }, [sessionId, endpoint]);

  const connectWebSocket = (term: XTermTerminal) => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const resolvedPath = wsPath ?? `/ws/ssh/${sessionId}`;
    const wsUrl = `${protocol}//${window.location.host}${resolvedPath}`;

    const ws = new WebSocket(wsUrl);

    ws.onopen = () => {
      setIsConnected(true);
      setIsConnecting(false);
      term.writeln(`\r\n\x1b[32m${connectedMessage}\x1b[0m\r\n`);
      term.writeln('Type your commands below:\r\n');
    };

    ws.onmessage = (event) => {
      const data = JSON.parse(event.data);
      if (data.type === 'output') {
        term.write(data.data);
      } else if (data.type === 'error') {
        term.write(`\x1b[31m${data.data}\x1b[0m`);
      }
    };

    ws.onclose = () => {
      setIsConnected(false);
      term.writeln('\r\n\x1b[31mConnection closed\x1b[0m\r\n');
      if (onClose) {
        onClose();
      }
    };

    ws.onerror = (error) => {
      console.error('WebSocket error:', error);
      setIsConnecting(false);
      term.writeln('\r\n\x1b[31mConnection error. Please try again.\x1b[0m\r\n');
    };

    // Handle terminal input
    term.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({
          type: 'input',
          data: data,
        }));
      }
    });

    wsRef.current = ws;
  };

  const reconnect = () => {
    setIsConnecting(true);
    if (terminalInstance.current) {
      connectWebSocket(terminalInstance.current);
    }
  };

  return (
    <div className={`relative ${className}`}>
      {/* Connection Status */}
      <div className="absolute top-2 right-2 z-10 flex items-center space-x-2">
        <div className={`w-2 h-2 rounded-full ${
          isConnected ? 'bg-green-500' :
          isConnecting ? 'bg-yellow-500 animate-pulse' :
          'bg-red-500'
        }`} />
        <span className="text-xs text-slate-400">
          {isConnected ? 'Connected' :
           isConnecting ? 'Connecting...' :
           'Disconnected'}
        </span>
        {!isConnected && !isConnecting && (
          <button
            onClick={reconnect}
            className="text-xs text-blue-400 hover:text-blue-300 underline"
          >
            Reconnect
          </button>
        )}
      </div>

      {/* Terminal */}
      <div
        ref={terminalRef}
        className="w-full h-96 bg-slate-900 rounded-lg border border-slate-700 overflow-hidden"
        style={{ minHeight: '384px' }}
      />

      {/* Instructions */}
      <div className="mt-2 text-xs text-slate-500">
        <p>• Use standard terminal commands</p>
        <p>• Ctrl+C to interrupt, Ctrl+D to exit</p>
        <p>• Connection is secured and monitored</p>
      </div>
    </div>
  );
};

export default WebTerminal;
