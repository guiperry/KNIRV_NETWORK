'use client';

import React, { useState } from 'react';
import { Settings, X } from 'lucide-react';

interface PolicyEditorProps {
  isOpen: boolean;
  onClose: () => void;
}

const PolicyEditor: React.FC<PolicyEditorProps> = ({ isOpen, onClose }) => {
  const [networkWhitelist, setNetworkWhitelist] = useState('api.company.com\ncdn.trusted.io');
  const [sensitivity, setSensitivity] = useState('Balanced');
  const [blockFileIO, setBlockFileIO] = useState(true);
  const [allowReadOnly, setAllowReadOnly] = useState(false);
  const [enableForensics, setEnableForensics] = useState(true);

  if (!isOpen) return null;

  return (
    <div
      className="fixed z-30 transition-all duration-300 bg-gradient-to-b from-blue-950 to-slate-900 border-r-2 border-blue-600 shadow-lg overflow-hidden"
      style={{
        left: '-100px',
        top: '210px',
        width: '100%',
        maxWidth: '500px',
        height: '280px',
      }}
    >
      <div className="h-full flex flex-col p-4">
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-sm font-bold flex items-center text-blue-300">
            <Settings className="w-4 h-4 mr-2" />
            Policy - Security Policy Editor
          </h2>
          <button
            onClick={onClose}
            className="bg-slate-700 hover:bg-slate-600 p-1 rounded transition-colors"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        <div className="grid grid-cols-2 gap-3 text-xs overflow-y-auto">
          <div>
            <label className="block text-xs font-semibold mb-1 text-blue-200">Network Whitelist</label>
            <textarea
              value={networkWhitelist}
              onChange={(e) => setNetworkWhitelist(e.target.value)}
              className="w-full bg-slate-950 border border-slate-700 rounded p-2 text-xs font-mono focus:outline-none focus:border-blue-500 h-20"
              placeholder="api.company.com&#10;cdn.trusted.io"
            />
          </div>
          <div>
            <label className="block text-xs font-semibold mb-1 text-blue-200">Validation Sensitivity</label>
            <select
              value={sensitivity}
              onChange={(e) => setSensitivity(e.target.value)}
              className="w-full bg-slate-950 border border-slate-700 rounded p-2 text-xs focus:outline-none focus:border-blue-500 mb-2"
            >
              <option>Balanced (Default)</option>
              <option>Paranoid (Strict)</option>
              <option>Permissive (Lenient)</option>
            </select>

            <label className="flex items-center mb-1">
              <input
                type="checkbox"
                checked={blockFileIO}
                onChange={(e) => setBlockFileIO(e.target.checked)}
                className="mr-2"
              />
              <span className="text-xs">Block file I/O</span>
            </label>
            <label className="flex items-center mb-1">
              <input
                type="checkbox"
                checked={allowReadOnly}
                onChange={(e) => setAllowReadOnly(e.target.checked)}
                className="mr-2"
              />
              <span className="text-xs">Allow read-only</span>
            </label>
            <label className="flex items-center">
              <input
                type="checkbox"
                checked={enableForensics}
                onChange={(e) => setEnableForensics(e.target.checked)}
                className="mr-2"
              />
              <span className="text-xs">Enable forensics</span>
            </label>
          </div>
        </div>

        <button className="mt-2 bg-blue-600 hover:bg-blue-700 text-white text-xs font-semibold py-1 px-3 rounded transition-all">
          Save Policy
        </button>
      </div>
    </div>
  );
};

export default PolicyEditor;