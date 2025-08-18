import React, { useEffect, useRef, useState } from 'react';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { WebLinksAddon } from '@xterm/addon-web-links';
import { SearchAddon } from '@xterm/addon-search';
import { Unicode11Addon } from '@xterm/addon-unicode11';
import '@xterm/xterm/css/xterm.css';
import { Button, Tooltip, Dropdown } from '../ui/components';
import { 
  X, 
  Maximize2, 
  Minimize2, 
  Copy, 
  Clipboard, 
  RotateCcw, 
  Search, 
  Clock, 
  Settings, 
  Download
} from 'lucide-react';

const TerminalUI = ({ 
  sessionId, 
  apiEndpoint = '/api/terminal', 
  onClose, 
  initialCommand,
  workingDirectory,
  environmentVariables = [],
  theme = 'dark',
  readOnly = false
}) => {
  const terminalRef = useRef(null);
  const xtermRef = useRef(null);
  const fitAddonRef = useRef(null);
  const searchAddonRef = useRef(null);
  const wsRef = useRef(null);
  const [isConnected, setIsConnected] = useState(false);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [sessionInfo, setSessionInfo] = useState(null);
  const [commandHistory, setCommandHistory] = useState([]);
  const [isSearchOpen, setIsSearchOpen] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const [showSettings, setShowSettings] = useState(false);
  const [fontSize, setFontSize] = useState(14);
  const [cursorBlink, setCursorBlink] = useState(true);

  // Initialize terminal
  useEffect(() => {
    if (!terminalRef.current) return;

    // Create terminal instance
    xtermRef.current = new Terminal({
      cursorBlink: cursorBlink,
      fontSize: fontSize,
      fontFamily: 'Menlo, Monaco, "Courier New", monospace',
      theme: getThemeColors(theme),
      scrollback: 5000,
      convertEol: true,
      cursorStyle: 'block',
      allowTransparency: true
    });

    // Add addons
    fitAddonRef.current = new FitAddon();
    searchAddonRef.current = new SearchAddon();
    const webLinksAddon = new WebLinksAddon();
    const unicode11Addon = new Unicode11Addon();

    xtermRef.current.loadAddon(fitAddonRef.current);
    xtermRef.current.loadAddon(searchAddonRef.current);
    xtermRef.current.loadAddon(webLinksAddon);
    xtermRef.current.loadAddon(unicode11Addon);

    // Open terminal
    xtermRef.current.open(terminalRef.current);
    fitAddonRef.current.fit();

    // Handle terminal input
    if (!readOnly) {
      xtermRef.current.onData(data => {
        if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
          wsRef.current.send(JSON.stringify({
            type: 'input',
            data: data
          }));
        }
      });
    }

    // Handle resize
    const handleResize = () => {
      if (fitAddonRef.current) {
        fitAddonRef.current.fit();
        
        // Send terminal size to server
        if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
          const { cols, rows } = xtermRef.current;
          wsRef.current.send(JSON.stringify({
            type: 'resize',
            cols,
            rows
          }));
        }
      }
    };

    window.addEventListener('resize', handleResize);
    
    // Initial resize
    setTimeout(handleResize, 100);

    // Connect to WebSocket
    connectWebSocket();

    return () => {
      window.removeEventListener('resize', handleResize);
      
      // Close WebSocket connection
      if (wsRef.current) {
        wsRef.current.close();
      }
      
      // Dispose terminal
      if (xtermRef.current) {
        xtermRef.current.dispose();
      }
    };
  }, [sessionId, fontSize, cursorBlink, theme]);

  // Connect to WebSocket
  const connectWebSocket = () => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}${apiEndpoint}/ws/${sessionId}`;
    
    wsRef.current = new WebSocket(wsUrl);
    
    wsRef.current.onopen = () => {
      setIsConnected(true);
      
      // If initial command is provided, send it
      if (initialCommand && xtermRef.current) {
        setTimeout(() => {
          wsRef.current.send(JSON.stringify({
            type: 'input',
            data: initialCommand + '\n'
          }));
        }, 500);
      }
    };
    
    wsRef.current.onclose = () => {
      setIsConnected(false);
      
      // Try to reconnect after a delay
      setTimeout(() => {
        if (xtermRef.current) {
          xtermRef.current.writeln('\r\n\x1b[33mConnection closed. Attempting to reconnect...\x1b[0m');
          connectWebSocket();
        }
      }, 3000);
    };
    
    wsRef.current.onerror = (error) => {
      console.error('WebSocket error:', error);
      if (xtermRef.current) {
        xtermRef.current.writeln('\r\n\x1b[31mWebSocket error. Check console for details.\x1b[0m');
      }
    };
    
    wsRef.current.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data);
        
        switch (message.type) {
          case 'stdout':
            if (xtermRef.current) {
              xtermRef.current.write(message.data);
            }
            break;
            
          case 'stderr':
            if (xtermRef.current) {
              // Write stderr in red
              xtermRef.current.write(`\x1b[31m${message.data}\x1b[0m`);
            }
            break;
            
          case 'session_info':
            setSessionInfo(message);
            break;
            
          case 'session_end':
            if (xtermRef.current) {
              xtermRef.current.writeln(`\r\n\x1b[33mSession ended with exit code ${message.exitCode}\x1b[0m`);
            }
            break;
            
          case 'history':
            setCommandHistory(message.history || []);
            break;
            
          case 'error':
            if (xtermRef.current) {
              xtermRef.current.writeln(`\r\n\x1b[31mError: ${message.error}\x1b[0m`);
            }
            break;
            
          default:
            console.log('Unknown message type:', message);
        }
      } catch (error) {
        console.error('Error parsing message:', error);
      }
    };
  };

  // Get terminal history
  const fetchHistory = () => {
    fetch(`${apiEndpoint}/${sessionId}/history`)
      .then(response => response.json())
      .then(data => {
        setCommandHistory(data.history || []);
      })
      .catch(error => {
        console.error('Error fetching history:', error);
      });
  };

  // Toggle fullscreen
  const toggleFullscreen = () => {
    setIsFullscreen(!isFullscreen);
    setTimeout(() => {
      if (fitAddonRef.current) {
        fitAddonRef.current.fit();
      }
    }, 100);
  };

  // Copy terminal selection to clipboard
  const copySelection = () => {
    if (xtermRef.current) {
      const selection = xtermRef.current.getSelection();
      if (selection) {
        navigator.clipboard.writeText(selection);
      }
    }
  };

  // Copy all terminal content to clipboard
  const copyAll = () => {
    if (xtermRef.current) {
      const lines = [];
      for (let i = 0; i < xtermRef.current.buffer.active.length; i++) {
        lines.push(xtermRef.current.buffer.active.getLine(i).translateToString());
      }
      navigator.clipboard.writeText(lines.join('\n'));
    }
  };

  // Clear terminal
  const clearTerminal = () => {
    if (xtermRef.current) {
      xtermRef.current.clear();
    }
  };

  // Search in terminal
  const searchInTerminal = () => {
    if (searchAddonRef.current && searchTerm) {
      searchAddonRef.current.findNext(searchTerm);
    }
  };

  // Download terminal content
  const downloadTerminalContent = () => {
    if (xtermRef.current) {
      const lines = [];
      for (let i = 0; i < xtermRef.current.buffer.active.length; i++) {
        lines.push(xtermRef.current.buffer.active.getLine(i).translateToString());
      }
      
      const content = lines.join('\n');
      const blob = new Blob([content], { type: 'text/plain' });
      const url = URL.createObjectURL(blob);
      
      const a = document.createElement('a');
      a.href = url;
      a.download = `terminal-${sessionId}-${new Date().toISOString()}.txt`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    }
  };

  // Get theme colors
  const getThemeColors = (themeName) => {
    switch (themeName) {
      case 'dark':
        return {
          background: '#1e1e1e',
          foreground: '#f0f0f0',
          cursor: '#f0f0f0',
          cursorAccent: '#1e1e1e',
          selection: 'rgba(255, 255, 255, 0.3)',
          black: '#000000',
          red: '#e06c75',
          green: '#98c379',
          yellow: '#e5c07b',
          blue: '#61afef',
          magenta: '#c678dd',
          cyan: '#56b6c2',
          white: '#d0d0d0',
          brightBlack: '#808080',
          brightRed: '#e06c75',
          brightGreen: '#98c379',
          brightYellow: '#e5c07b',
          brightBlue: '#61afef',
          brightMagenta: '#c678dd',
          brightCyan: '#56b6c2',
          brightWhite: '#ffffff'
        };
      case 'light':
        return {
          background: '#f0f0f0',
          foreground: '#333333',
          cursor: '#333333',
          cursorAccent: '#f0f0f0',
          selection: 'rgba(0, 0, 0, 0.3)',
          black: '#000000',
          red: '#c91b00',
          green: '#00c200',
          yellow: '#c7c400',
          blue: '#0225c7',
          magenta: '#c930c7',
          cyan: '#00c5c7',
          white: '#c7c7c7',
          brightBlack: '#676767',
          brightRed: '#ff6d67',
          brightGreen: '#5ff967',
          brightYellow: '#fefb67',
          brightBlue: '#6871ff',
          brightMagenta: '#ff76ff',
          brightCyan: '#5ffdff',
          brightWhite: '#ffffff'
        };
      default:
        return {};
    }
  };

  return (
    <div className={`terminal-container ${isFullscreen ? 'fullscreen' : ''}`} 
         style={{ 
           display: 'flex', 
           flexDirection: 'column',
           height: isFullscreen ? '100vh' : '400px',
           width: isFullscreen ? '100vw' : '100%',
           position: isFullscreen ? 'fixed' : 'relative',
           top: isFullscreen ? 0 : 'auto',
           left: isFullscreen ? 0 : 'auto',
           zIndex: isFullscreen ? 9999 : 'auto',
           backgroundColor: theme === 'dark' ? '#1e1e1e' : '#f0f0f0',
           border: `1px solid ${theme === 'dark' ? '#333' : '#ccc'}`,
           borderRadius: '4px',
           overflow: 'hidden'
         }}>
      
      {/* Terminal Header */}
      <div className="terminal-header" style={{
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        padding: '8px 12px',
        backgroundColor: theme === 'dark' ? '#252525' : '#e0e0e0',
        borderBottom: `1px solid ${theme === 'dark' ? '#333' : '#ccc'}`,
        color: theme === 'dark' ? '#f0f0f0' : '#333'
      }}>
        <div className="terminal-title">
          {sessionInfo ? (
            <span>Terminal: {sessionInfo.workingDir || workingDirectory || 'No directory'}</span>
          ) : (
            <span>Terminal {sessionId}</span>
          )}
          <span className="connection-status" style={{
            marginLeft: '10px',
            display: 'inline-block',
            width: '10px',
            height: '10px',
            borderRadius: '50%',
            backgroundColor: isConnected ? '#4caf50' : '#f44336'
          }}></span>
        </div>
        
        <div className="terminal-actions" style={{
          display: 'flex',
          gap: '8px'
        }}>
          {/* Search Button */}
          <Tooltip content="Search">
            <Button
              variant="ghost"
              size="icon"
              onClick={() => setIsSearchOpen(!isSearchOpen)}
            >
              <Search size={16} />
            </Button>
          </Tooltip>
          
          {/* History Button */}
          <Tooltip content="Command History">
            <Button
              variant="ghost"
              size="icon"
              onClick={fetchHistory}
            >
              <Clock size={16} />
            </Button>
          </Tooltip>
          
          {/* Copy Button */}
          <Tooltip content="Copy Selection">
            <Button
              variant="ghost"
              size="icon"
              onClick={copySelection}
            >
              <Copy size={16} />
            </Button>
          </Tooltip>
          
          {/* Copy All Button */}
          <Tooltip content="Copy All">
            <Button
              variant="ghost"
              size="icon"
              onClick={copyAll}
            >
              <Clipboard size={16} />
            </Button>
          </Tooltip>
          
          {/* Clear Button */}
          <Tooltip content="Clear Terminal">
            <Button
              variant="ghost"
              size="icon"
              onClick={clearTerminal}
            >
              <RotateCcw size={16} />
            </Button>
          </Tooltip>
          
          {/* Download Button */}
          <Tooltip content="Download Terminal Content">
            <Button
              variant="ghost"
              size="icon"
              onClick={downloadTerminalContent}
            >
              <Download size={16} />
            </Button>
          </Tooltip>
          
          {/* Settings Button */}
          <Tooltip content="Terminal Settings">
            <Button
              variant="ghost"
              size="icon"
              onClick={() => setShowSettings(!showSettings)}
            >
              <Settings size={16} />
            </Button>
          </Tooltip>
          
          {/* Fullscreen Button */}
          <Tooltip content={isFullscreen ? "Exit Fullscreen" : "Fullscreen"}>
            <Button
              variant="ghost"
              size="icon"
              onClick={toggleFullscreen}
            >
              {isFullscreen ? <Minimize2 size={16} /> : <Maximize2 size={16} />}
            </Button>
          </Tooltip>
          
          {/* Close Button */}
          <Tooltip content="Close Terminal">
            <Button
              variant="ghost"
              size="icon"
              onClick={onClose}
            >
              <X size={16} />
            </Button>
          </Tooltip>
        </div>
      </div>
      
      {/* Search Bar */}
      {isSearchOpen && (
        <div className="terminal-search" style={{
          display: 'flex',
          padding: '8px 12px',
          backgroundColor: theme === 'dark' ? '#252525' : '#e0e0e0',
          borderBottom: `1px solid ${theme === 'dark' ? '#333' : '#ccc'}`
        }}>
          <input
            type="text"
            placeholder="Search..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                searchInTerminal();
              }
            }}
            style={{
              flex: 1,
              padding: '4px 8px',
              backgroundColor: theme === 'dark' ? '#333' : '#fff',
              color: theme === 'dark' ? '#f0f0f0' : '#333',
              border: `1px solid ${theme === 'dark' ? '#444' : '#ccc'}`,
              borderRadius: '4px'
            }}
          />
          <Button
            variant="default"
            size="sm"
            onClick={searchInTerminal}
            style={{ marginLeft: '8px' }}
          >
            Search
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              if (searchAddonRef.current) {
                searchAddonRef.current.findPrevious(searchTerm);
              }
            }}
            style={{ marginLeft: '4px' }}
          >
            Prev
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              if (searchAddonRef.current) {
                searchAddonRef.current.findNext(searchTerm);
              }
            }}
            style={{ marginLeft: '4px' }}
          >
            Next
          </Button>
        </div>
      )}
      
      {/* Settings Panel */}
      {showSettings && (
        <div className="terminal-settings" style={{
          padding: '12px',
          backgroundColor: theme === 'dark' ? '#252525' : '#e0e0e0',
          borderBottom: `1px solid ${theme === 'dark' ? '#333' : '#ccc'}`,
          color: theme === 'dark' ? '#f0f0f0' : '#333'
        }}>
          <div style={{ marginBottom: '8px' }}>
            <label style={{ display: 'block', marginBottom: '4px' }}>Font Size</label>
            <div style={{ display: 'flex', alignItems: 'center' }}>
              <input
                type="range"
                min="10"
                max="24"
                value={fontSize}
                onChange={(e) => setFontSize(parseInt(e.target.value))}
                style={{ flex: 1 }}
              />
              <span style={{ marginLeft: '8px', minWidth: '30px' }}>{fontSize}px</span>
            </div>
          </div>
          
          <div style={{ marginBottom: '8px' }}>
            <label style={{ display: 'flex', alignItems: 'center' }}>
              <input
                type="checkbox"
                checked={cursorBlink}
                onChange={(e) => setCursorBlink(e.target.checked)}
                style={{ marginRight: '8px' }}
              />
              Cursor Blink
            </label>
          </div>
          
          <div>
            <label style={{ display: 'block', marginBottom: '4px' }}>Theme</label>
            <select
              value={theme}
              onChange={(e) => {
                // This would need to be lifted to parent component
                // For now, we'll just log it
                console.log('Theme changed to:', e.target.value);
              }}
              style={{
                padding: '4px 8px',
                backgroundColor: theme === 'dark' ? '#333' : '#fff',
                color: theme === 'dark' ? '#f0f0f0' : '#333',
                border: `1px solid ${theme === 'dark' ? '#444' : '#ccc'}`,
                borderRadius: '4px'
              }}
            >
              <option value="dark">Dark</option>
              <option value="light">Light</option>
            </select>
          </div>
        </div>
      )}
      
      {/* Command History Panel */}
      {commandHistory.length > 0 && (
        <div className="terminal-history" style={{
          maxHeight: '200px',
          overflowY: 'auto',
          backgroundColor: theme === 'dark' ? '#252525' : '#e0e0e0',
          borderBottom: `1px solid ${theme === 'dark' ? '#333' : '#ccc'}`,
          color: theme === 'dark' ? '#f0f0f0' : '#333'
        }}>
          <div style={{ padding: '8px 12px', fontWeight: 'bold' }}>Command History</div>
          <ul style={{ margin: 0, padding: 0, listStyle: 'none' }}>
            {commandHistory.map((entry, index) => (
              <li key={index} style={{
                padding: '4px 12px',
                borderTop: `1px solid ${theme === 'dark' ? '#333' : '#ccc'}`,
                cursor: 'pointer',
                display: 'flex',
                justifyContent: 'space-between'
              }}
              onClick={() => {
                if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
                  wsRef.current.send(JSON.stringify({
                    type: 'input',
                    data: entry.command + '\n'
                  }));
                }
              }}>
                <div style={{ flex: 1 }}>
                  <div style={{ fontFamily: 'monospace' }}>{entry.command}</div>
                  <div style={{ fontSize: '0.8em', color: theme === 'dark' ? '#aaa' : '#666' }}>
                    Exit Code: {entry.exitCode}, Time: {new Date(entry.timestamp).toLocaleTimeString()}
                  </div>
                </div>
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={(e) => {
                    e.stopPropagation();
                    navigator.clipboard.writeText(entry.command);
                  }}
                >
                  <Copy size={14} />
                </Button>
              </li>
            ))}
          </ul>
        </div>
      )}
      
      {/* Terminal */}
      <div 
        ref={terminalRef} 
        className="terminal" 
        style={{ 
          flex: 1,
          padding: '4px',
          backgroundColor: theme === 'dark' ? '#1e1e1e' : '#f0f0f0',
          overflow: 'hidden'
        }}
      />
      
      {/* Status Bar */}
      <div className="terminal-status" style={{
        display: 'flex',
        justifyContent: 'space-between',
        padding: '4px 12px',
        fontSize: '0.8em',
        backgroundColor: theme === 'dark' ? '#252525' : '#e0e0e0',
        borderTop: `1px solid ${theme === 'dark' ? '#333' : '#ccc'}`,
        color: theme === 'dark' ? '#aaa' : '#666'
      }}>
        <div>
          {sessionInfo ? (
            <>
              Status: {sessionInfo.status} 
              {sessionInfo.status === 'exited' && ` (Exit Code: ${sessionInfo.exitCode})`}
            </>
          ) : (
            'Connecting...'
          )}
        </div>
        <div>
          {readOnly ? 'Read Only' : 'Read/Write'}
        </div>
      </div>
    </div>
  );
};

export default TerminalUI;