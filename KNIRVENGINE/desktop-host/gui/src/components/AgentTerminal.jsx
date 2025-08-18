import React, { useEffect, useRef, useState } from 'react';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { WebLinksAddon } from '@xterm/addon-web-links';
import { SearchAddon } from '@xterm/addon-search';
import { Unicode11Addon } from '@xterm/addon-unicode11';
import '@xterm/xterm/css/xterm.css';
import agentTerminalService from '../services/agentTerminalService';

// Add custom styles to improve terminal selection
const terminalStyles = `
  .xterm-selection {
    opacity: 0.5;
    background-color: #3a3a3a !important;
  }
  .xterm-text-layer {
    user-select: text !important;
    -webkit-user-select: text !important;
  }
  .xterm {
    user-select: text !important;
    -webkit-user-select: text !important;
  }
`;
import {
  Minimize2,
  Maximize2,
  X,
  Terminal as TerminalIcon,
  AlertCircle,
  Copy,
  Check,
  FileText,
  RotateCcw,
  Search
} from 'lucide-react';

const AgentTerminal = ({ 
  agent, 
  sessionId, 
  isMinimized = false, 
  onMinimize, 
  onMaximize, 
  onClose 
}) => {
  const terminalRef = useRef(null);
  const [terminal, setTerminal] = useState(null);
  const [terminalId, setTerminalId] = useState(null);
  const [isConnected, setIsConnected] = useState(false);
  const [error, setError] = useState(null);
  const [selectedText, setSelectedText] = useState('');
  const [copied, setCopied] = useState(false);
  const [showLogs, setShowLogs] = useState(false);
  const [logs, setLogs] = useState([]);
  const [loadingLogs, setLoadingLogs] = useState(false);
  const wsRef = useRef(null);
  const fitAddonRef = useRef(null);
  
  // Function to copy terminal content to clipboard
  const copyTerminalContent = () => {
    if (terminal) {
      const content = terminal.getSelection();
      if (content) {
        navigator.clipboard.writeText(content)
          .then(() => {
            setCopied(true);
            setTimeout(() => setCopied(false), 2000);
          })
          .catch(err => console.error('Failed to copy text: ', err));
      } else {
        // If no selection, copy all visible content
        const buffer = terminal.buffer.active;
        const lines = [];
        for (let i = 0; i < buffer.length; i++) {
          lines.push(buffer.getLine(i).translateToString());
        }
        const allContent = lines.join('\n');
        navigator.clipboard.writeText(allContent)
          .then(() => {
            setCopied(true);
            setTimeout(() => setCopied(false), 2000);
          })
          .catch(err => console.error('Failed to copy text: ', err));
      }
    }
  };

  // Function to fetch comprehensive logs
  const fetchLogs = async () => {
    if (!terminalId || !sessionId || terminalId === 'offline-mode') return;

    setLoadingLogs(true);
    try {
      console.log(`Fetching logs for terminal: ${terminalId}, session: ${sessionId}`);
      
      const result = await agentTerminalService.getAgentTerminalLogs({
        sessionId: sessionId,
        terminalId: terminalId
      });

      if (result.success) {
        setLogs(result.logs || []);
        setShowLogs(true);
      } else {
        console.error('Failed to fetch logs:', result.error);
      }
    } catch (error) {
      console.error('Error fetching logs:', error);
    } finally {
      setLoadingLogs(false);
    }
  };

  useEffect(() => {
    if (!agent || !sessionId || isMinimized) return;

    // Initialize terminal
    const term = new Terminal({
      rows: 12,
      cols: 37, // Changed from 60 to 37 to prevent text bleeding
      theme: {
        background: '#1e1e1e',
        foreground: '#f0f0f0',
        cursor: '#f0f0f0',
        selection: '#3a3a3a',
        black: '#000000',
        red: '#ff5555',
        green: '#50fa7b',
        yellow: '#f1fa8c',
        blue: '#bd93f9',
        magenta: '#ff79c6',
        cyan: '#8be9fd',
        white: '#bfbfbf',
        brightBlack: '#4d4d4d',
        brightRed: '#ff6e67',
        brightGreen: '#5af78e',
        brightYellow: '#f4f99d',
        brightBlue: '#caa9fa',
        brightMagenta: '#ff92d0',
        brightCyan: '#9aedfe',
        brightWhite: '#e6e6e6'
      },
      fontFamily: 'Monaco, Menlo, "Ubuntu Mono", monospace',
      fontSize: 12,
      disableStdin: false,
      allowTransparency: true,
      copyOnSelect: true, // Enable copy on select
      rightClickSelectsWord: true, // Select word on right click
      selectionStyle: 'line', // Make selection more visible
      rendererType: 'dom', // Use DOM renderer for better text selection
      convertEol: true, // Convert line endings to make selection easier
      lineHeight: 1.2,
      cursorBlink: true,
      scrollback: 1000,
    });

    // Add addons
    const fitAddon = new FitAddon();
    const webLinksAddon = new WebLinksAddon();
    
    term.loadAddon(fitAddon);
    term.loadAddon(webLinksAddon);
    
    fitAddonRef.current = fitAddon;

    // Open terminal in the DOM
    if (terminalRef.current) {
      term.open(terminalRef.current);
      fitAddon.fit();
      setTerminal(term);

      // Enable text selection in the terminal
      const enableTextSelection = () => {
        // Find the terminal elements and make them selectable
        const termElement = terminalRef.current;
        if (termElement) {
          const textLayers = termElement.querySelectorAll('.xterm-text-layer');
          const viewports = termElement.querySelectorAll('.xterm-viewport');
          const screens = termElement.querySelectorAll('.xterm-screen');
          
          // Apply styles to make text selectable
          [textLayers, viewports, screens].forEach(elements => {
            elements.forEach(el => {
              el.style.userSelect = 'text';
              el.style.webkitUserSelect = 'text';
              el.style.MozUserSelect = 'text';
              el.style.msUserSelect = 'text';
            });
          });
        }
      };

      // Apply selection styles after a short delay to ensure terminal is fully rendered
      setTimeout(enableTextSelection, 100);

      // Initialize terminal session
      initTerminal(term, fitAddon);
    }

    // Cleanup
    return () => {
      if (wsRef.current) {
        wsRef.current.close();
      }
      if (terminalId) {
        closeTerminal();
      }
      if (term) {
        term.dispose();
      }
    };
  }, [agent, sessionId, isMinimized]);

  const initTerminal = async (term, fitAddon) => {
    try {
      console.log(`Initializing terminal for agent: ${agent?.name}, sessionId: ${sessionId}`);
      console.log('Agent object:', agent);

      // Extract agent ID from the agent object
      const agentId = agent?.id || agent?.name?.toLowerCase().replace(/[^a-z0-9]/g, '') || 'default_agent';
      
      // Use a consistent version format (1.0 instead of 1.0.0)
      const version = "1.0";
      
      console.log(`Activating agent: ${agentId} with version: ${version} for session: ${sessionId}`);
      
      // First, activate the agent before creating terminal
      const activateData = await agentTerminalService.activateAgent({
        agentId: agentId,
        version: version,
        sessionId: sessionId,
        config: {
          agent_type: agent?.build_target || 'plugin',
          capabilities: agent?.capabilities || ['test', 'Web Analysis'],
          collection: agent?.collection || 'Test_Collection'
        }
      });
      
      console.log('Agent activated successfully:', activateData);

      // Now create terminal session
      console.log(`Creating terminal for session: ${sessionId} with rows: ${term.rows}, cols: ${term.cols}`);
      const terminalData = await agentTerminalService.createAgentTerminal({
        sessionId: sessionId,
        agentId: agentId,
        rows: term.rows,
        cols: term.cols
      });

      console.log('Terminal created successfully:', terminalData);
      setTerminalId(terminalData.terminalId);

      // Connect WebSocket
      connectWebSocket(terminalData.terminalId, term);

    } catch (err) {
      console.error('Failed to initialize terminal:', err);
      setError(err.message);

      // Show fallback terminal with basic agent info
      term.write('\r\n\x1b[31mFailed to connect to agent backend\x1b[0m\r\n');
      term.write('\x1b[33mShowing agent information in offline mode:\x1b[0m\r\n\r\n');
      term.write(`\x1b[36mAgent Name: ${agent?.name || 'Unknown'}\x1b[0m\r\n`);
      term.write(`\x1b[36mStatus: ${agent?.status || 'Unknown'}\x1b[0m\r\n`);
      term.write(`\x1b[36mCollection: ${agent?.collection || 'Unknown'}\x1b[0m\r\n`);
      if (agent?.capabilities && agent.capabilities.length > 0) {
        term.write(`\x1b[36mCapabilities: ${agent.capabilities.join(', ')}\x1b[0m\r\n`);
      }
      term.write('\r\n\x1b[33mTo enable full terminal functionality, please start the backend server.\x1b[0m\r\n');

      // Set a fake terminal ID to prevent further connection attempts
      setTerminalId('offline-mode');
      setIsConnected(false);
    }
  };

  const connectWebSocket = (termId, term) => {
    console.log(`Connecting to WebSocket for terminal: ${termId}, session: ${sessionId}`);
    
    const ws = agentTerminalService.createAgentTerminalWebSocket({
      sessionId: sessionId,
      terminalId: termId,
      handlers: {
        onOpen: () => {
          setIsConnected(true);
          setError(null);
          term.write('\r\n\x1b[32mTerminal connected\x1b[0m\r\n');
          term.write(`\x1b[36mAgent: ${agent?.name || 'Unknown'}\x1b[0m\r\n`);
          term.write(`\x1b[36mSession: ${sessionId}\x1b[0m\r\n`);
          term.write('\x1b[33mWaiting for agent activity...\x1b[0m\r\n');
        },
        onMessage: (event) => {
          term.write(event.data);
        },
        onClose: (event) => {
          setIsConnected(false);
          if (event.wasClean) {
            term.write('\r\n\x1b[33mTerminal disconnected\x1b[0m\r\n');
          } else {
            term.write('\r\n\x1b[31mTerminal connection lost\x1b[0m\r\n');
            term.write('\x1b[31mTrying to reconnect...\x1b[0m\r\n');
            // Try to reconnect after a delay
            setTimeout(() => {
              if (terminalId) {
                connectWebSocket(terminalId, term);
              }
            }, 3000);
          }
        },
        onError: (error) => {
          console.error('WebSocket error:', error);
          setError('WebSocket connection failed');
          term.write('\r\n\x1b[31mConnection error: Agent backend may not be running\x1b[0m\r\n');
          term.write('\x1b[31mPlease ensure the backend server is started\x1b[0m\r\n');
        }
      }
    });
    
    wsRef.current = ws;

    // Handle terminal input
    term.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(data);
      }
    });

    // Handle terminal resize
    term.onResize(({ rows, cols }) => {
      resizeTerminal(termId, rows, cols);
    });
  };

  const resizeTerminal = async (termId, rows, cols) => {
    if (!termId || termId === 'offline-mode') return;
    
    try {
      console.log(`Resizing terminal: ${termId} to rows: ${rows}, cols: ${cols}`);
      
      await agentTerminalService.resizeAgentTerminal({
        sessionId: sessionId,
        terminalId: termId,
        rows: rows,
        cols: cols
      });
    } catch (err) {
      console.error('Failed to resize terminal:', err);
    }
  };

  const closeTerminal = async () => {
    if (!terminalId || terminalId === 'offline-mode') return;

    try {
      console.log(`Closing terminal: ${terminalId} for session: ${sessionId}`);
      
      await agentTerminalService.closeAgentTerminal({
        sessionId: sessionId,
        terminalId: terminalId
      });
    } catch (err) {
      console.error('Failed to close terminal:', err);
    }
  };

  const handleResize = () => {
    if (fitAddonRef.current && !isMinimized) {
      setTimeout(() => {
        fitAddonRef.current.fit();
      }, 100);
    }
  };

  useEffect(() => {
    handleResize();
  }, [isMinimized]);

  if (isMinimized) {
    return (
      <div className="flex items-center space-x-2 bg-slate-800/50 rounded-lg px-3 py-2 border border-slate-700/50 AgentTerminal" data-terminal="true">
        <TerminalIcon className="w-4 h-4 text-green-400" />
        <span className="text-sm text-slate-300">Terminal</span>
        <button
          onClick={onMaximize}
          className="p-1 text-slate-400 hover:text-white transition-colors"
        >
          <Maximize2 className="w-3 h-3" />
        </button>
        <button
          onClick={onClose}
          className="p-1 text-slate-400 hover:text-red-400 transition-colors"
        >
          <X className="w-3 h-3" />
        </button>
      </div>
    );
  }

  return (
    <div className="bg-slate-900/90 rounded-lg border border-slate-700/50 overflow-hidden AgentTerminal" data-terminal="true">
      {/* Add custom styles for terminal selection */}
      <style>{terminalStyles}</style>
      {/* Terminal Header */}
      <div className="flex items-center justify-between px-3 py-2 bg-slate-800/50 border-b border-slate-700/50">
        <div className="flex items-center space-x-2">
          <TerminalIcon className="w-4 h-4 text-green-400" />
          <span className="text-sm font-medium text-white">Agent Terminal</span>
          {isConnected ? (
            <div className="w-2 h-2 bg-green-400 rounded-full"></div>
          ) : (
            <div className="w-2 h-2 bg-red-400 rounded-full"></div>
          )}
        </div>
        
        <div className="flex items-center space-x-1">
          {error && (
            <AlertCircle className="w-4 h-4 text-red-400" title={error} />
          )}
          {/* Copy Button */}
          <button
            onClick={copyTerminalContent}
            className="p-1 text-slate-400 hover:text-green-400 transition-colors"
            title="Copy terminal content"
          >
            {copied ? (
              <Check className="w-4 h-4 text-green-400" />
            ) : (
              <Copy className="w-4 h-4" />
            )}
          </button>
          {/* Logs Button */}
          <button
            onClick={fetchLogs}
            disabled={loadingLogs || !terminalId}
            className="p-1 text-slate-400 hover:text-blue-400 transition-colors disabled:opacity-50"
            title="View comprehensive logs"
          >
            <FileText className="w-4 h-4" />
          </button>
          <button
            onClick={onMinimize}
            className="p-1 text-slate-400 hover:text-white transition-colors"
          >
            <Minimize2 className="w-4 h-4" />
          </button>
          <button
            onClick={onClose}
            className="p-1 text-slate-400 hover:text-red-400 transition-colors"
          >
            <X className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* Terminal Content */}
      <div className="relative">
        <div 
          ref={terminalRef} 
          className="h-48 p-2 select-text"
          style={{ 
            minHeight: '192px',
            paddingRight: '20px', /* Ensure text doesn't bleed behind scrollbar */
            userSelect: 'text',
            WebkitUserSelect: 'text',
            MozUserSelect: 'text',
            msUserSelect: 'text'
          }}
        />
      </div>

      {/* Logs Modal */}
      {showLogs && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-slate-800 rounded-lg p-6 max-w-4xl max-h-[80vh] overflow-hidden flex flex-col">
            <div className="flex justify-between items-center mb-4">
              <h3 className="text-lg font-semibold text-white">Comprehensive Agent Logs</h3>
              <button
                onClick={() => setShowLogs(false)}
                className="text-slate-400 hover:text-white"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="flex-1 overflow-auto bg-slate-900 rounded p-4 font-mono text-sm">
              {logs.length === 0 ? (
                <div className="text-slate-400 text-center py-8">No logs available</div>
              ) : (
                logs.map((log, index) => (
                  <div key={index} className="mb-2 border-b border-slate-700 pb-2">
                    <div className="flex items-center space-x-2 text-xs text-slate-400 mb-1">
                      <span>{new Date(log.timestamp).toLocaleString()}</span>
                      <span className={`px-2 py-1 rounded ${
                        log.type === 'stdout' ? 'bg-green-900 text-green-300' :
                        log.type === 'stderr' ? 'bg-red-900 text-red-300' :
                        'bg-blue-900 text-blue-300'
                      }`}>
                        {log.type}
                      </span>
                      <span className="text-slate-500">{log.source}</span>
                      {log.process_id && <span className="text-slate-500">PID: {log.process_id}</span>}
                    </div>
                    <pre className="text-slate-200 whitespace-pre-wrap break-words">
                      {new TextDecoder().decode(new Uint8Array(log.data))}
                    </pre>
                  </div>
                ))
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default AgentTerminal;
