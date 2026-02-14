'use client';

import React, { useState, useEffect } from 'react';
import { Activity, X, TrendingUp, Cpu, HardDrive, Network } from 'lucide-react';
import { 
  LineChart, 
  Line, 
  XAxis, 
  YAxis, 
  CartesianGrid, 
  Tooltip, 
  ResponsiveContainer,
  AreaChart,
  Area
} from 'recharts';

interface MonitorPanelProps {
  isOpen: boolean;
  onClose: () => void;
  nodeId?: string;
  isSidebarOpen?: boolean;
  onOpenSolver?: () => void;
}

interface ResolutionLog {
  timestamp: string;
  failureId: string;
  nodeId: string;
  strategy: string;
  validation: 'PASS' | 'FAIL';
  status: string;
}

interface MetricData {
  time: string;
  cpu: number;
  memory: number;
  network: number;
}

const MonitorPanel: React.FC<MonitorPanelProps> = ({ isOpen, onClose, nodeId, isSidebarOpen, onOpenSolver }) => {
  const [metrics, setMetrics] = useState<MetricData[]>([]);
  
  const mockResolutionLogs: ResolutionLog[] = [
    { timestamp: '10:45:23', failureId: 'FAIL-8472', nodeId: nodeId || 'CLEAN-01', strategy: 'Constraint-Based', validation: 'PASS', status: 'Response Sent' },
    { timestamp: '10:44:56', failureId: 'FAIL-8471', nodeId: nodeId || 'CLEAN-02', strategy: 'Forensic-Block', validation: 'FAIL', status: 'Blocked' },
    { timestamp: '10:44:12', failureId: 'FAIL-8470', nodeId: nodeId || 'CLEAN-01', strategy: 'Self-Correction', validation: 'PASS', status: 'Response Sent' },
  ];

  // Simulate real-time metrics
  useEffect(() => {
    if (isOpen) {
      const initialData = Array.from({ length: 20 }, (_, i) => ({
        time: `${10}:${45 + i}`,
        cpu: 40 + Math.random() * 20,
        memory: 60 + Math.random() * 10,
        network: 20 + Math.random() * 40,
      }));
      setMetrics(initialData);

      const interval = setInterval(() => {
        setMetrics(prev => {
          const newData = [...prev.slice(1)];
          const lastTime = prev[prev.length - 1].time;
          const [h, m] = lastTime.split(':').map(Number);
          const nextM = (m + 1) % 60;
          const nextH = nextM === 0 ? (h + 1) % 24 : h;
          
          newData.push({
            time: `${nextH}:${nextM < 10 ? '0' + nextM : nextM}`,
            cpu: 40 + Math.random() * 20,
            memory: 60 + Math.random() * 10,
            network: 20 + Math.random() * 40,
          });
          return newData;
        });
      }, 3000);

      return () => clearInterval(interval);
    }
  }, [isOpen]);

  if (!isOpen) return null;

  return (
    <div
      className="absolute bottom-0 right-0 z-[100] transition-all duration-500 transform ease-in-out bg-gradient-to-t from-slate-950 via-blue-950/90 to-slate-900 border-t border-blue-600/50 shadow-[0_-10px_40px_rgba(0,0,0,0.5)] overflow-hidden"
      style={{
        height: '35vh',
        left: isSidebarOpen ? '300px' : '0',
      }}
    >
      <div className="h-full flex flex-col p-4">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center space-x-4">
            <h2 className="text-sm font-bold flex items-center text-blue-300">
              <Activity className="w-4 h-4 mr-2 text-green-400 animate-pulse" />
              Fabric Monitoring - Resolution Tracking {nodeId && `[${nodeId}]`}
            </h2>
            <div className="flex space-x-4">
              <div className="flex items-center text-[10px] text-slate-400">
                <div className="w-2 h-2 rounded-full bg-blue-500 mr-1" /> CPU
              </div>
              <div className="flex items-center text-[10px] text-slate-400">
                <div className="w-2 h-2 rounded-full bg-purple-500 mr-1" /> MEM
              </div>
              <div className="flex items-center text-[10px] text-slate-400">
                <div className="w-2 h-2 rounded-full bg-green-500 mr-1" /> NET
              </div>
            </div>
          </div>
          <button
            onClick={onClose}
            className="text-slate-400 hover:text-white hover:bg-slate-700 p-1 rounded transition-colors"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        <div className="flex-1 grid grid-cols-1 lg:grid-cols-3 gap-6 overflow-hidden">
          {/* Metrics Chart */}
          <div className="lg:col-span-1 h-full bg-slate-950/50 rounded-lg border border-blue-600/20 p-2">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={metrics}>
                <defs>
                  <linearGradient id="colorCpu" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.3}/>
                    <stop offset="95%" stopColor="#3b82f6" stopOpacity={0}/>
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="#1e293b" vertical={false} />
                <XAxis 
                  dataKey="time" 
                  stroke="#475569" 
                  fontSize={10} 
                  tickLine={false}
                  axisLine={false}
                />
                <YAxis 
                  stroke="#475569" 
                  fontSize={10} 
                  tickLine={false}
                  axisLine={false}
                  domain={[0, 100]}
                />
                <Tooltip 
                  contentStyle={{ backgroundColor: '#0f172a', border: '1px solid #1e40af', fontSize: '10px' }}
                  itemStyle={{ padding: '0' }}
                />
                <Area type="monotone" dataKey="cpu" stroke="#3b82f6" fillOpacity={1} fill="url(#colorCpu)" isAnimationActive={false} />
                <Area type="monotone" dataKey="memory" stroke="#a855f7" fillOpacity={0} isAnimationActive={false} />
                <Area type="monotone" dataKey="network" stroke="#22c55e" fillOpacity={0} isAnimationActive={false} />
              </AreaChart>
            </ResponsiveContainer>
          </div>

          {/* Resolution Table */}
          <div className="lg:col-span-2 overflow-y-auto pr-2 custom-scrollbar">
            <table className="w-full text-[11px] font-mono border-separate border-spacing-y-1">
              <thead className="sticky top-0 bg-slate-900/95 backdrop-blur-sm z-10">
                <tr>
                  <th className="text-left p-2 text-blue-400 font-semibold uppercase tracking-wider">Timestamp</th>
                  <th className="text-left p-2 text-blue-400 font-semibold uppercase tracking-wider">Failure ID</th>
                  <th className="text-left p-2 text-blue-400 font-semibold uppercase tracking-wider">Strategy</th>
                  <th className="text-left p-2 text-blue-400 font-semibold uppercase tracking-wider">Validation</th>
                  <th className="text-left p-2 text-blue-400 font-semibold uppercase tracking-wider">Status</th>
                </tr>
              </thead>
              <tbody>
                {mockResolutionLogs.map((log, idx) => (
                  <tr 
                    key={idx} 
                    className="bg-slate-800/30 hover:bg-blue-900/20 transition-all rounded-md cursor-pointer group"
                    onClick={onOpenSolver}
                  >
                    <td className="p-2 text-slate-400 border-l-2 border-transparent group-hover:border-blue-500">{log.timestamp}</td>
                    <td className="p-2 text-blue-300 font-bold">{log.failureId}</td>
                    <td className="p-2 text-slate-300">{log.strategy}</td>
                    <td className="p-2">
                      <span className={`px-2 py-0.5 rounded-full text-[10px] font-bold ${
                        log.validation === 'PASS' ? 'bg-green-500/20 text-green-400 border border-green-500/30' : 'bg-red-500/20 text-red-400 border border-red-500/30'
                      }`}>
                        {log.validation}
                      </span>
                    </td>
                    <td className="p-2 text-slate-400 italic">{log.status}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  );
};

export default MonitorPanel;
