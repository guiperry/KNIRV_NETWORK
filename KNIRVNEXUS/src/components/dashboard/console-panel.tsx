'use client';

import React from 'react';
import { AlertTriangle, X } from 'lucide-react';

interface ConsolePanelProps {
  isOpen: boolean;
  onClose: () => void;
}

const ConsolePanel: React.FC<ConsolePanelProps> = ({ isOpen, onClose }) => {
  if (!isOpen) return null;

  return (
    <div
      className="fixed top-6 z-30 transition-all duration-300 bg-gradient-to-b from-blue-950 to-slate-900 border-r-2 border-blue-600 shadow-lg overflow-hidden"
      style={{
        left: '-100px',
        width: '100%',
        maxWidth: '500px',
        height: '200px',
      }}
    >
      <div className="h-full flex flex-col p-4">
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-sm font-bold flex items-center text-blue-300">
            <AlertTriangle className="w-4 h-4 mr-2 text-red-400" />
            Console - Real-Time Failure Feed
          </h2>
          <button
            onClick={onClose}
            className="bg-slate-700 hover:bg-slate-600 p-1 rounded transition-colors"
          >
            <X className="w-4 h-4" />
          </button>
        </div>
        
        <div className="bg-slate-950 rounded border border-slate-700 p-3 h-full overflow-y-auto font-mono text-xs space-y-1">
          <div className="text-red-400">[10:45:23] FAILURE DETECTED - Shell ID: cortex-8472 - Type: Unauthorized API Call</div>
          <div className="text-yellow-400">[10:44:56] FAILURE DETECTED - Shell ID: cortex-8471 - Type: Hallucination Risk</div>
          <div className="text-red-400">[10:44:12] FAILURE DETECTED - Shell ID: cortex-8470 - Type: Data Leak Attempt</div>
          <div className="text-yellow-400">[10:43:45] FAILURE DETECTED - Shell ID: cortex-8469 - Type: Logic Error</div>
          <div className="text-green-400">[10:43:20] RESOLVED - Shell ID: cortex-8468 - Strategy: Self-Correction</div>
        </div>
      </div>
    </div>
  );
};

export default ConsolePanel;