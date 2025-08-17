import React, { useState, useRef, useEffect } from 'react';

interface TerminalEntry {
  input: string;
  output: string;
  timestamp: Date;
}

interface TerminalProps {
  onCommand: (command: string) => void;
  history: TerminalEntry[];
  prompt?: string;
  className?: string;
}

export const Terminal: React.FC<TerminalProps> = ({
  onCommand,
  history,
  prompt = '$ ',
  className = '',
}) => {
  const [currentInput, setCurrentInput] = useState('');
  const inputRef = useRef<HTMLInputElement>(null);
  const historyRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    // Auto-scroll to bottom when new history entries are added
    if (historyRef.current) {
      historyRef.current.scrollTop = historyRef.current.scrollHeight;
    }
  }, [history]);

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      if (currentInput.trim()) {
        onCommand(currentInput.trim());
        setCurrentInput('');
      }
    }
  };

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setCurrentInput(e.target.value);
  };

  return (
    <div className={`terminal bg-black text-green-400 font-mono text-sm ${className}`}>
      <div 
        ref={historyRef}
        className="terminal-history h-64 overflow-y-auto p-2 border border-gray-600"
        data-testid="terminal-history"
      >
        {history.map((entry, index) => (
          <div key={index} className="mb-1" data-testid={`history-${index}`}>
            <div className="text-gray-300">
              {prompt}{entry.input}
            </div>
            <div className="text-green-400 ml-2">
              {entry.output}
            </div>
          </div>
        ))}
      </div>
      <div className="terminal-input-line flex items-center p-2 border-t border-gray-600">
        <span className="text-gray-300 mr-2">{prompt}</span>
        <input
          ref={inputRef}
          type="text"
          value={currentInput}
          onChange={handleInputChange}
          onKeyDown={handleKeyDown}
          className="flex-1 bg-transparent text-green-400 outline-none"
          placeholder="Enter command..."
          data-testid="terminal-input"
          autoFocus
        />
      </div>
    </div>
  );
};

export default Terminal;
