'use client';

import React from 'react';
import { Activity, X, TrendingUp } from 'lucide-react';

interface MonitorPanelProps {
  isOpen: boolean;
  onClose: () => void;
}

interface ResolutionLog {
  timestamp: string;
  failureId: string;
  nodeId: string;
  strategy: string;
  validation: 'PASS' | 'FAIL';
  status: string;
}

const MonitorPanel: React.FC<MonitorPanelProps> = ({ isOpen, onClose }) => {
  const mockResolutionLogs: ResolutionLog[] = [
    { timestamp: '10:45:23', failureId: 'FAIL-8472', nodeId: 'CLEAN-01', strategy: 'Constraint-Based', validation: 'PASS', status: 'Response Sent' },
    { timestamp: '10:44:56', failureId: 'FAIL-8471', nodeId: 'CLEAN-02', strategy: 'Forensic-Block', validation: 'FAIL', status: 'Blocked' },
    { timestamp: '10:44:12', failureId: 'FAIL-8470', nodeId: 'CLEAN-01', strategy: 'Self-Correction', validation: 'PASS', status: 'Response Sent' },
  ];

  if (!isOpen) return null;

  return (
    <div
      className="fixed bottom-0 left-0 right-0 z-30 transition-all duration-300 bg-gradient-to-t from-blue-950 to-slate-900 border-t-2 border-blue-600 shadow-lg overflow-hidden"
      style={{
        height: '33vh',
      }}
    >
      <div className="h-full flex flex-col p-4">
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-sm font-bold flex items-center text-blue-300">
            <Activity className="w-4 h-4 mr-2 text-green-400" />
            Monitor - Resolution Tracking
          </h2>
          <button
            onClick={onClose}
            className="bg-slate-700 hover:bg-slate-600 p-1 rounded transition-colors"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        <div className="overflow-y-auto">
          <table className="w-full text-xs font-mono">
            <thead className="sticky top-0 bg-slate-800 border-b border-slate-700">
              <tr>
                <th className="text-left p-2 text-blue-300">Timestamp</th>
                <th className="text-left p-2 text-blue-300">Failure ID</th>
                <th className="text-left p-2 text-blue-300">Node</th>
                <th className="text-left p-2 text-blue-300">Strategy</th>
                <th className="text-left p-2 text-blue-300">Validation</th>
                <th className="text-left p-2 text-blue-300">Status</th>
              </tr>
            </thead>
            <tbody>
              {mockResolutionLogs.map((log, idx) => (
                <tr key={idx} className="border-b border-slate-700 hover:bg-slate-800 transition-colors">
                  <td className="p-2 text-gray-300">{log.timestamp}</td>
                  <td className="p-2 text-blue-400">{log.failureId}</td>
                  <td className="p-2 text-purple-400">{log.nodeId}</td>
                  <td className="p-2 text-gray-300">{log.strategy}</td>
                  <td className={`p-2 ${log.validation === 'PASS' ? 'text-green-400' : 'text-red-400'}`}>
                    {log.validation}
                  </td>
                  <td className="p-2 text-gray-300">{log.status}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};

export default MonitorPanel;