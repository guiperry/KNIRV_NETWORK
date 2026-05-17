'use client';

import React, { useEffect, useRef, useCallback } from 'react';
import type { Terminal as XTermTerminal } from '@xterm/xterm';
import type { FitAddon as XTermFitAddon } from '@xterm/addon-fit';
import '@xterm/xterm/css/xterm.css';

interface InnerAgentTerminalProps {
  dveId: string;
  sessionId: string;
  toolName: string;
  isVisible: boolean;
}

// Resolve the WebSocket base URL from the current page origin.
function wsBaseUrl(): string {
  if (typeof window === 'undefined') return 'ws://localhost';
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${proto}//${window.location.host}`;
}

export function InnerAgentTerminal({ dveId, sessionId, toolName, isVisible }: InnerAgentTerminalProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const termRef = useRef<XTermTerminal | null>(null);
  const fitRef = useRef<XTermFitAddon | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const mountedRef = useRef(false);

  const sendInput = useCallback((data: string) => {
    const ws = wsRef.current;
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: 'input', data }));
    }
  }, []);

  const sendResize = useCallback((cols: number, rows: number) => {
    const ws = wsRef.current;
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: 'resize', cols, rows }));
    }
  }, []);

  useEffect(() => {
    if (!containerRef.current || mountedRef.current) return;
    mountedRef.current = true;

    let term: XTermTerminal;
    let fitAddon: XTermFitAddon;

    const init = async () => {
      const { Terminal } = await import('@xterm/xterm');
      const { FitAddon } = await import('@xterm/addon-fit');
      const { WebLinksAddon } = await import('@xterm/addon-web-links');

      term = new Terminal({
        cursorBlink: true,
        fontSize: 12,
        fontFamily: 'Menlo, Monaco, "Courier New", monospace',
        theme: {
          background: '#03050a',
          foreground: '#d4d4d4',
          cursor: '#a855f7',
          selectionBackground: 'rgba(168,85,247,0.25)',
        },
        // Raw mode — inner agents handle echo themselves
        convertEol: false,
      });

      fitAddon = new FitAddon();
      term.loadAddon(fitAddon);
      term.loadAddon(new WebLinksAddon());

      if (containerRef.current) {
        term.open(containerRef.current);
        fitAddon.fit();
      }

      termRef.current = term;
      fitRef.current = fitAddon;

      term.writeln(`\x1b[35m[KNIRV]\x1b[0m Connecting to ${toolName} session ${sessionId.slice(0, 8)}…`);

      // Open WebSocket to /ws/{dveId}/inner/stream/{sessionId}
      const ws = new WebSocket(`${wsBaseUrl()}/ws/${dveId}/inner/stream/${sessionId}`);
      wsRef.current = ws;

      ws.onopen = () => {
        term.writeln(`\x1b[32m[CONNECTED]\x1b[0m ${toolName} PTY stream active`);
        const { cols, rows } = term;
        sendResize(cols, rows);
      };

      ws.onmessage = (ev) => {
        try {
          const msg = JSON.parse(ev.data) as { type: string; data?: string };
          if (msg.type === 'data' && msg.data) {
            term.write(msg.data);
          } else if (msg.type === 'stream_end') {
            term.writeln('\r\n\x1b[33m[SESSION ENDED]\x1b[0m');
          } else if (msg.type === 'error') {
            term.writeln(`\r\n\x1b[31m[ERROR]\x1b[0m ${msg.data}`);
          }
        } catch {
          // raw bytes fallback
          term.write(ev.data);
        }
      };

      ws.onclose = () => {
        term.writeln('\r\n\x1b[33m[DISCONNECTED]\x1b[0m WebSocket closed.');
      };

      ws.onerror = () => {
        term.writeln('\r\n\x1b[31m[ERROR]\x1b[0m WebSocket error — check DVE agent status.');
      };

      // Send keystrokes to PTY
      term.onData(sendInput);

      // Resize on terminal dimension change
      term.onResize(({ cols, rows }) => sendResize(cols, rows));
    };

    init();

    return () => {
      wsRef.current?.close();
      termRef.current?.dispose();
      mountedRef.current = false;
    };
  }, [dveId, sessionId, toolName, sendInput, sendResize]);

  // Re-fit when tab becomes visible
  useEffect(() => {
    if (isVisible && fitRef.current) {
      setTimeout(() => fitRef.current?.fit(), 50);
    }
  }, [isVisible]);

  return (
    <div
      ref={containerRef}
      className="w-full h-full"
      style={{ display: isVisible ? 'block' : 'none' }}
    />
  );
}
