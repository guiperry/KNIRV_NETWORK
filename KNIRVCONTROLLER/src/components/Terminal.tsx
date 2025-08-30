import React, { useState, useRef, useEffect } from 'react';
import { terminalCommandService, CommandHistory } from '../services/TerminalCommandService';

interface TerminalEntry {
  input: string;
  output: string;
  timestamp: Date;
}

interface TerminalProps {
  onCommand?: (command: string) => void; // Made optional for backward compatibility
  history?: TerminalEntry[]; // Made optional since we'll use service history
  prompt?: string;
  className?: string;
}

export const Terminal: React.FC<TerminalProps> = ({
  onCommand,
  history: externalHistory,
  prompt = '$ ',
  className = '',
}) => {
  const [currentInput, setCurrentInput] = useState('');
  const [commandHistory, setCommandHistory] = useState<CommandHistory[]>([]);
  const [isExecuting, setIsExecuting] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const historyRef = useRef<HTMLDivElement>(null);

  // Load command history from service
  useEffect(() => {
    const loadHistory = () => {
      const serviceHistory = terminalCommandService.getCommandHistory();
      setCommandHistory(serviceHistory);
    };

    loadHistory();

    // Refresh history every 5 seconds
    const interval = setInterval(loadHistory, 5000);
    return () => clearInterval(interval);
  }, []);

  // Auto-scroll to bottom when new history entries are added
  useEffect(() => {
    if (historyRef.current) {
      historyRef.current.scrollTop = historyRef.current.scrollHeight;
    }
  }, [commandHistory, externalHistory]);

  const handleKeyDown = async (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      if (currentInput.trim() && !isExecuting) {
        await executeCommand(currentInput.trim());
        setCurrentInput('');
      }
    }
  };

  const executeCommand = async (command: string) => {
    setIsExecuting(true);

    try {
      // Execute command using the service
      await terminalCommandService.executeCommand(command);

      // Update local history
      const updatedHistory = terminalCommandService.getCommandHistory();
      setCommandHistory(updatedHistory);

      // Call external callback if provided (for backward compatibility)
      if (onCommand) {
        onCommand(command);
      }
    } catch (error) {
      console.error('Command execution failed:', error);
    } finally {
      setIsExecuting(false);
    }
  };

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setCurrentInput(e.target.value);
  };

  // Combine service history with external history for display
  const displayHistory = [
    ...(externalHistory || []),
    ...commandHistory.map(cmd => ({
      input: cmd.command,
      output: cmd.result.success ? cmd.result.output : (cmd.result.error || 'Command failed'),
      timestamp: cmd.timestamp
    }))
  ].sort((a, b) => a.timestamp.getTime() - b.timestamp.getTime());

  return (
    <div className={`terminal bg-black text-green-400 font-mono text-sm ${className}`}>
      <div
        ref={historyRef}
        className="terminal-history h-64 overflow-y-auto p-2 border border-gray-600"
        data-testid="terminal-history"
      >
        {displayHistory.map((entry, index) => (
          <div key={index} className="mb-1" data-testid={`history-${index}`}>
            <div className="text-gray-300">
              {prompt}{entry.input}
            </div>
            <div className={`ml-2 ${entry.output.startsWith('Error') || entry.output.includes('failed') ? 'text-red-400' : 'text-green-400'}`}>
              {entry.output}
            </div>
            <div className="text-xs text-gray-500 ml-2">
              {entry.timestamp.toLocaleTimeString()}
            </div>
          </div>
        ))}
        {isExecuting && (
          <div className="mb-1">
            <div className="text-gray-300">
              {prompt}{currentInput}
            </div>
            <div className="text-yellow-400 ml-2">
              Executing...
            </div>
          </div>
        )}
      </div>
      <div className="terminal-input-line flex items-center p-2 border-t border-gray-600">
        <span className="text-gray-300 mr-2">{prompt}</span>
        <input
          ref={inputRef}
          type="text"
          value={currentInput}
          onChange={handleInputChange}
          onKeyDown={handleKeyDown}
          disabled={isExecuting}
          className={`flex-1 bg-transparent outline-none ${
            isExecuting ? 'text-gray-500 cursor-not-allowed' : 'text-green-400'
          }`}
          placeholder={isExecuting ? "Executing command..." : "Enter command..."}
          data-testid="terminal-input"
          autoFocus
        />
      </div>
    </div>
  );
};

export default Terminal;
